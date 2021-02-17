package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/Hsn723/ct-monitor/api"
	"github.com/Hsn723/ct-monitor/mailer"
	"github.com/cybozu-go/log"
	"github.com/mitchellh/mapstructure"
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

	// CurrentVersion stores the current version number.
	CurrentVersion string
)

func getPositionFilePath() string {
	positionFile := config.GetString(positionFileConfigKey)
	if positionFile == "" {
		return defaultPositionFile
	}
	return positionFile
}

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
	_ = log.Info(fmt.Sprintf("ct-monitor version %s", CurrentVersion), nil)
	config.SetConfigFile(configFile)
	if err := createFile(configFile); err != nil {
		_ = log.Critical(err.Error(), nil)
	}
	if err := config.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			_ = log.Critical(err.Error(), nil)
		}
	}
	_ = log.Info("using config file", map[string]interface{}{
		"path": config.ConfigFileUsed(),
	})
	config.WatchConfig()

	initPosition()
	initMailer()
}

func initPosition() {
	positionFile := getPositionFilePath()
	position.SetConfigFile(positionFile)
	if err := createFile(positionFile); err != nil {
		_ = log.Critical(err.Error(), nil)
	}
	if err := position.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			_ = log.Critical(err.Error(), nil)
		}
	}
	_ = log.Info("using position file", map[string]interface{}{
		"path": position.ConfigFileUsed(),
	})
	position.WatchConfig()
}

func initMailer() {
	from := config.GetString(mailFromConfigKey)
	to := config.GetString(mailToConfigKey)
	if to == "" {
		_ = log.Warn("alert mail recipient missing", nil)
		return
	}
	if from == "" {
		_ = log.Warn("alert mail from address missing", nil)
		return
	}
	mailerName := config.GetString(mailerConfigKey)
	switch mailerName {
	case "smtp":
		mailSender = &mailer.SMTPMailer{}
	case "sendgrid":
		mailSender = &mailer.SendgridMailer{
			APIKey: config.GetString(sendgridTokenConfigKey),
			Logger: log.DefaultLogger(),
		}
	case "amazonses":
		mailSender = &mailer.AmazonSESMailer{
			Logger: log.DefaultLogger(),
		}
	default:
		_ = log.Warn("no mailer configured, email report will not be sent", nil)
		return
	}

	senderConf := config.GetStringMap(mailerName)
	if err := mapstructure.Decode(senderConf, &mailSender); err != nil {
		_ = log.Critical(err.Error(), nil)
	}

	if err := mailSender.Init(from, to); err != nil {
		_ = log.Critical(err.Error(), nil)
	}
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
	_ = log.Info("sending report", map[string]interface{}{
		"domain": domain,
	})
	return mailSender.Send(subject, body)
}

func checkIssuances(domain string, wildcards, subdomains bool, c api.CertspotterClient) error {
	key := getDomainConfigName(domain)
	lastIssuance := position.GetUint64(key)
	issuances, err := c.GetIssuances(domain, wildcards, subdomains, lastIssuance)
	if err != nil {
		return err
	}
	if len(issuances) == 0 {
		_ = log.Info("no new issuances observed", map[string]interface{}{
			"domain": domain,
		})
		return nil
	}
	lastIssuance = issuances[len(issuances)-1].ID
	for _, issuance := range issuances {
		_ = log.Info("observed issuance", map[string]interface{}{
			"id":     issuance.ID,
			"names":  issuance.Domains,
			"sha256": issuance.Cert.SHA256,
		})
	}
	if err := sendMail(domain, issuances); err != nil {
		return err
	}
	position.Set(key, lastIssuance)
	_ = log.Info("done checking", map[string]interface{}{
		"domain": domain,
	})
	return nil
}

func atomicWritePosition() error {
	positionFile := getPositionFilePath()
	tmpFile, err := ioutil.TempFile(filepath.Dir(positionFile), "position.*.toml")
	if err != nil {
		return err
	}
	if err := position.WriteConfigAs(tmpFile.Name()); err != nil {
		return err
	}
	fi, err := tmpFile.Stat()
	if err != nil {
		return err
	}
	if fi.Size() == 0 {
		return nil
	}
	return os.Rename(tmpFile.Name(), positionFile)
}

func runRoot(cmd *cobra.Command, args []string) error {
	csp := api.CertspotterClient{
		Endpoint: endpoint,
		Token:    config.GetString(certspotterTokenConfigKey),
	}
	for _, domain := range config.GetStringSlice(domainsConfigKey) {
		if err := checkIssuances(domain, matchWildcards, includeSubdomains, csp); err != nil {
			_ = log.Error(err.Error(), map[string]interface{}{
				"domain": domain,
			})
		}
	}

	return atomicWritePosition()
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_ = log.Critical(err.Error(), nil)
	}
}
