package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Hsn723/ct-monitor/api"
	"github.com/Hsn723/ct-monitor/mailer"
	"github.com/Hsn723/ct-monitor/pusher"
	"github.com/mitchellh/mapstructure"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	positionFileConfigKey     = "position_config.filename"
	mailFromConfigKey         = "alert_config.from"
	mailToConfigKey           = "alert_config.recipient"
	mailerConfigKey           = "alert_config.mailer_config"
	domainsConfigKey          = "domains"
	endpointConfigKey         = "endpoint"
	certspotterTokenConfigKey = "certspotter_token"
	sendgridTokenConfigKey    = "sendgrid.apiKey"
	pushgatewayEndpointKey    = "push_config.endpoint"
	pushgatewayPortKey        = "push_config.port"

	defaultEndpoint     = "https://api.certspotter.com/v1/issuances"
	defaultConfigFile   = "/etc/ct-monitor/config.toml"
	defaultPositionFile = "/var/log/ct-monitor/positions.toml"
	tokenEnv            = "CERTSPOTTER_TOKEN"
	sendgridTokenEnv    = "SENDGRID_TOKEN"
)

var (
	rootCmd = &cobra.Command{
		Use:   "ct-monitor",
		Short: "ct-monitor queries the certspotter API for new certificate issuances",
		RunE:  runRoot,
	}
	position = viper.New()
	config   = viper.New()

	configFile        string
	endpoint          string
	matchWildcards    bool
	includeSubdomains bool
	mailSender        mailer.Mailer
	metricPusher      *push.Pusher

	// CurrentVersion stores the current version number
	CurrentVersion string
)

func createFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		file.Close()
	}
	return nil
}

func initConfig() {
	log.Printf("ct-monitor version %s", CurrentVersion)
	config.SetConfigFile(configFile)
	if err := createFile(configFile); err != nil {
		log.Fatal(err)
	}
	if err := config.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatal(err)
		}
	}
	log.Print("using config file:", config.ConfigFileUsed())
	config.WatchConfig()

	initPosition()
	initMailer()
	initPusher()
}

func initPosition() {
	positionFile := config.GetString(positionFileConfigKey)
	if positionFile == "" {
		positionFile = defaultPositionFile
	}
	position.SetConfigFile(positionFile)
	if err := createFile(positionFile); err != nil {
		log.Fatal(err)
	}
	if err := position.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatal(err)
		}
	}
	log.Print("using position file:", position.ConfigFileUsed())
	position.WatchConfig()
}

func initMailer() {
	from := config.GetString(mailFromConfigKey)
	to := config.GetString(mailToConfigKey)
	if to == "" {
		log.Print("alert mail recipient missing")
		return
	}
	if from == "" {
		log.Println("alert mail from address missing")
		return
	}
	mailerName := config.GetString(mailerConfigKey)
	switch mailerName {
	case "smtp":
		mailSender = &mailer.SMTPMailer{}
	case "sendgrid":
		mailSender = &mailer.SendgridMailer{
			APIKey: viper.GetString(sendgridTokenConfigKey),
		}
	case "amazonses":
		mailSender = &mailer.AmazonSESMailer{}
	default:
		log.Println("no mailer configured, email report will not be sent")
		return
	}

	senderConf := config.GetStringMap(mailerName)
	if err := mapstructure.Decode(senderConf, &mailSender); err != nil {
		log.Fatal(err)
	}

	if err := mailSender.Init(from, to); err != nil {
		log.Fatal(err)
	}
}

func initPusher() {
	pgEndpoint := config.GetString(pushgatewayEndpointKey)
	pgPort := config.GetInt(pushgatewayPortKey)
	if pgEndpoint == "" || pgPort <= 0 {
		log.Println("no configuration for pushgateway")
		return
	}
	metricPusher = pusher.GetPusher(fmt.Sprintf("%s:%d", pgEndpoint, pgPort))
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.Flags().StringVarP(&configFile, "config", "c", defaultConfigFile, "path to configuration file")
	rootCmd.Flags().StringVarP(&endpoint, "endpoint", "e", defaultEndpoint, "API endpoint")

	rootCmd.Flags().StringP("position", "p", defaultPositionFile, "path to position file")
	rootCmd.Flags().StringP("token", "t", "", "API token")
	rootCmd.Flags().StringSliceP("domains", "d", []string{}, "domains to query certspotter for issuances")

	rootCmd.Flags().BoolVarP(&matchWildcards, "wildcard", "w", true, "match wildcards")
	rootCmd.Flags().BoolVarP(&includeSubdomains, "subdomains", "s", true, "include subdomains")

	_ = config.BindEnv(certspotterTokenConfigKey, tokenEnv)
	_ = config.BindEnv(sendgridTokenConfigKey, sendgridTokenEnv)

	_ = config.BindPFlag(certspotterTokenConfigKey, rootCmd.Flags().Lookup("token"))
	_ = config.BindPFlag(domainsConfigKey, rootCmd.Flags().Lookup(domainsConfigKey))
	_ = config.BindPFlag(positionFileConfigKey, rootCmd.Flags().Lookup("position"))
	_ = config.BindPFlag(endpointConfigKey, rootCmd.Flags().Lookup("endpoint"))
}

func getDomainConfigName(domain string) string {
	return strings.ReplaceAll(domain, ".", "-")
}

func sendMail(domain string, issuances []api.Issuance) error {
	if mailSender == nil {
		return nil
	}
	subject := fmt.Sprintf("Certificate Transparency Notification for %s", domain)
	body := fmt.Sprintf("ct-monitor has observed the issuance of the following certificate(s) for the %s domain:\n", domain)
	for _, i := range issuances {
		body += fmt.Sprintf("\nIssuer: %s\nDNS Names: %v\nValidity: %s - %s\nSHA256: %s\nIssuance type: %s\n", i.Issuer.Name, i.Domains, i.NotBefore, i.NotAfter, i.Cert.SHA256, i.Cert.Type)
	}
	log.Printf("sending report for %s", domain)
	return mailSender.Send(subject, body)
}

func pushMetrics(domain string, issuances []api.Issuance) error {
	if metricPusher == nil {
		return nil
	}
	for _, i := range issuances {
		pusher.IssuancesObserved.With(prometheus.Labels{
			"id":         strconv.FormatUint(i.ID, 10),
			"domain":     domain,
			"dns_names":  strings.Join(i.Domains, ","),
			"issuer":     i.Issuer.Name,
			"not_before": i.NotBefore,
		}).Set(1)
	}

	return metricPusher.Add()
}

func checkIssuances(domain string, wildcards, subdomains bool, c api.CertspotterClient) error {
	key := getDomainConfigName(domain)
	lastIssuance := position.GetUint64(key)
	issuances, err := c.GetIssuances(domain, wildcards, subdomains, lastIssuance)
	if err != nil {
		return err
	}
	if len(issuances) == 0 {
		log.Printf("no new issuances observed for %s", domain)
		return nil
	}
	lastIssuance = issuances[len(issuances)-1].ID
	for _, issuance := range issuances {
		log.Printf("observed issuance %d for %v with SHA256: %s", issuance.ID, issuance.Domains, issuance.Cert.SHA256)
	}
	if err := sendMail(domain, issuances); err != nil {
		return err
	}
	if err := pushMetrics(domain, issuances); err != nil {
		return err
	}
	position.Set(key, lastIssuance)
	log.Printf("done checking %s", domain)
	return nil
}

func runRoot(cmd *cobra.Command, args []string) error {
	csp := api.CertspotterClient{
		Endpoint: endpoint,
		Token:    viper.GetString(certspotterTokenConfigKey),
	}
	for _, domain := range config.GetStringSlice(domainsConfigKey) {
		if err := checkIssuances(domain, matchWildcards, includeSubdomains, csp); err != nil {
			log.Print(err)
		}
	}

	return position.WriteConfig()
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
