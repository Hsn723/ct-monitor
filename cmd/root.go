package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Hsn723/certspotter-client/api"
	"github.com/Hsn723/ct-monitor/config"
	"github.com/Hsn723/ct-monitor/filter"
	"github.com/Hsn723/ct-monitor/mailer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	rootCmd = &cobra.Command{
		Use:           "ct-monitor",
		Short:         "ct-monitor queries the certspotter API for new certificate issuances",
		RunE:          runRoot,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	position = viper.New()

	errCheckIssuances = errors.New("some issuances could not be retrieved")

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

func initPosition(pc config.PositionConfig) error {
	position.SetConfigFile(pc.Filename)
	if err := createFile(pc.Filename); err != nil {
		return fmt.Errorf("failed to create position file: %w", err)
	}
	if err := position.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read position file: %w", err)
		}
	}
	slog.Info("using position file", "path", position.ConfigFileUsed())
	position.WatchConfig()
	return nil
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
	slog.Info("sending report", "domain", tplVars.Domain)
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
		slog.Info("no new issuances observed", "domain", dc.Name)
		return nil
	}
	lastIssuance = issuances[len(issuances)-1].ID
	for _, issuance := range issuances {
		slog.Info("observed issuance", "id", issuance.ID, "names", issuance.Domains, "sha256", issuance.Cert.SHA256)
	}
	issuances, err = filter.ApplyFilters(fc.Filters, issuances)
	if err != nil {
		slog.Info("errors encountered running filters", "error", err.Error(), "domain", dc.Name, "filters", fc.Filters)
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
	slog.Info("done checking", "domain", dc.Name)
	return nil
}

func atomicWritePosition(pc config.PositionConfig) error {
	tmpFile, err := os.CreateTemp(filepath.Dir(pc.Filename), "position.*.toml")
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
		slog.Error("could not initialize domain mailer, using default", "error", err.Error(), "domain", dc.Name)
		return defaultMailSender
	}
	return domainMailer
}

func runRoot(_ *cobra.Command, _ []string) error {
	slog.Info("ct-monitor", "version", version, "commit", commit, "date", date, "built_by", builtBy)
	conf, err := config.Load(configFile)
	if err != nil {
		return err
	}
	slog.Info("loaded configuration", "config", configFile)
	if err := initPosition(conf.PositionConfig); err != nil {
		return err
	}
	defaultMailSender := conf.GetMailer(conf.AlertConfig.Mailer)
	if err := defaultMailSender.Init(); err != nil {
		return err
	}
	csp := api.CertspotterClient{
		Endpoint: conf.Endpoint,
		Token:    conf.Token,
	}
	var checkErr error
	for _, domain := range conf.Domains {
		domainMailer := getMailSenderForDomain(conf, domain, defaultMailSender)
		if err := checkIssuances(domain, csp, domainMailer, conf.FilterConfig, conf.MailTemplate); err != nil {
			slog.Error("could not check issuances", "error", err.Error(), "domain", domain.Name)
			checkErr = errCheckIssuances
		}
	}

	return errors.Join(checkErr, atomicWritePosition(conf.PositionConfig))
}

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))
	slog.SetDefault(logger)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		slog.Error("ct-monitor encountered an error", "error", err.Error())
		os.Exit(1)
	}
}
