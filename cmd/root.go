package cmd

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Hsn723/certspotter-client/api"
	"github.com/Hsn723/ct-monitor/config"
	"github.com/Hsn723/ct-monitor/filter"
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

type mailTemplateVars struct {
	Domain    string
	Issuances []api.Issuance
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

func getTemplatedMailContent(templateString string, vars mailTemplateVars) (string, error) {
	tmpl, err := template.New("template").Parse(templateString)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, vars)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func sendMail(mailSender mailer.Mailer, tplVars mailTemplateVars, mt config.MailTemplate) error {
	subject, err := getTemplatedMailContent(mt.Subject, tplVars)
	if err != nil {
		return err
	}
	body, err := getTemplatedMailContent(mt.Body, tplVars)
	if err != nil {
		return err
	}
	_ = log.Info("sending report", map[string]interface{}{
		"domain": tplVars.Domain,
	})
	return mailSender.Send(subject, body)
}

func checkIssuances(dc config.DomainConfig, c api.CertspotterClient, mailSender mailer.Mailer, fc config.FilterConfig, mt config.MailTemplate) error {
	key := getDomainConfigName(dc.Name)
	lastIssuance := position.GetUint64(key)
	issuances, err := c.GetIssuances(dc.Name, dc.MatchWildcards, dc.IncludeSubdomains, lastIssuance)
	if err != nil {
		return err
	}
	if len(issuances) == 0 {
		_ = log.Info("no new issuances observed", map[string]interface{}{
			"domain": dc.Name,
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
	issuances, err = filter.ApplyFilters(fc.Filters, issuances)
	if err != nil {
		_ = log.Info("errors encountered running filters", map[string]interface{}{
			"error":   err.Error(),
			"domain":  dc.Name,
			"filters": fc.Filters,
		})
	}
	if len(issuances) == 0 {
		position.Set(key, lastIssuance)
		return nil
	}
	tplVars := mailTemplateVars{
		Domain:    dc.Name,
		Issuances: issuances,
	}
	if err := sendMail(mailSender, tplVars, mt); err != nil {
		return err
	}
	position.Set(key, lastIssuance)
	_ = log.Info("done checking", map[string]interface{}{
		"domain": dc.Name,
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

func getMailSenderForDomain(conf *config.Config, dc config.DomainConfig, defaultMailSender mailer.Mailer) mailer.Mailer {
	if dc.Mailer == "" {
		return defaultMailSender
	}
	domainMailer := conf.GetMailer(dc.Mailer)
	if err := domainMailer.Init(); err != nil {
		_ = log.Error("could not initialize domain mailer, using default", map[string]interface{}{
			"error":  err.Error(),
			"domain": dc.Name,
		})
		return defaultMailSender
	}
	return domainMailer
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
		domainMailer := getMailSenderForDomain(conf, domain, defaultMailSender)
		if err := checkIssuances(domain, csp, domainMailer, conf.FilterConfig, conf.MailTemplate); err != nil {
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
