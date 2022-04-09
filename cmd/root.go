package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/Hsn723/certspotter-client/api"
	"github.com/Hsn723/ct-monitor/config"
	"github.com/Hsn723/ct-monitor/mailer"
	"github.com/cybozu-go/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	rootCmd = &cobra.Command{
		Use:   "ct-monitor",
		Short: "ct-monitor queries the certspotter API for new certificate issuances",
		RunE:  runRoot,
	}
	position = viper.New()

	configFile string

	version string
	commit  string
	date    string
	builtBy string
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

func initPosition(pc config.PositionConfig) {
	position.SetConfigFile(pc.Filename)
	if err := createFile(pc.Filename); err != nil {
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

func init() {
	rootCmd.Flags().StringVarP(&configFile, "config", "c", config.DefaultConfigFile, "path to configuration file")
}

func getDomainConfigName(domain string) string {
	return strings.ReplaceAll(domain, ".", "-")
}

func sendMail(mailSender mailer.Mailer, domain string, issuances []api.Issuance) error {
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

func checkIssuances(domain string, wildcards, subdomains bool, c api.CertspotterClient, mailSender mailer.Mailer) error {
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
	if err := sendMail(mailSender, domain, issuances); err != nil {
		return err
	}
	position.Set(key, lastIssuance)
	_ = log.Info("done checking", map[string]interface{}{
		"domain": domain,
	})
	return nil
}

func atomicWritePosition(pc config.PositionConfig) error {
	tmpFile, err := ioutil.TempFile(filepath.Dir(pc.Filename), "position.*.toml")
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
	return os.Rename(tmpFile.Name(), pc.Filename)
}

func runRoot(cmd *cobra.Command, args []string) error {
	_ = log.Info("ct-monitor", map[string]interface{}{
		"version":  version,
		"commit":   commit,
		"date":     date,
		"built_by": builtBy,
	})
	conf, err := config.Load(configFile)
	if err != nil {
		return err
	}
	_ = log.Info("loaded configuration", map[string]interface{}{
		"config": configFile,
	})
	initPosition(conf.PositionConfig)
	defaultMailSender := conf.GetMailer(conf.AlertConfig.Mailer)
	if err := defaultMailSender.Init(); err != nil {
		return err
	}
	csp := api.CertspotterClient{
		Endpoint: conf.Endpoint,
		Token:    conf.Token,
	}
	for _, domain := range conf.Domains {
		domainMailer := defaultMailSender
		if domain.Mailer != "" {
			domainMailer = conf.GetMailer(domain.Mailer)
		}
		if err := checkIssuances(domain.Name, domain.MatchWildcards, domain.IncludeSubdomains, csp, domainMailer); err != nil {
			_ = log.Error(err.Error(), map[string]interface{}{
				"domain": domain.Name,
			})
		}
	}

	return atomicWritePosition(conf.PositionConfig)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_ = log.Critical(err.Error(), nil)
	}
}
