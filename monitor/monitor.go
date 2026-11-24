//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package monitor

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
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

var (
	ErrCheckIssuances = errors.New("some issuances could not be retrieved")
)

type IMonitor interface {
	Run() error
}

type Monitor struct {
	config   *config.Config
	fs       afero.Fs
	position *viper.Viper
}

type mailTemplateVars struct {
	Domain    string
	Issuances []api.Issuance
}

func NewMonitor(cfg *config.Config, fs afero.Fs) *Monitor {
	pos := viper.New()
	pos.SetFs(fs)
	return &Monitor{
		config:   cfg,
		fs:       fs,
		position: pos,
	}
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

func (m *Monitor) createFile(path string) error {
	if _, err := m.fs.Stat(path); os.IsNotExist(err) {
		if err := m.fs.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		file, err := m.fs.Create(path)
		if err != nil {
			return err
		}
		file.Close()
	}
	return nil
}

func (m *Monitor) initPosition() error {
	filename := m.config.PositionConfig.Filename
	m.position.SetConfigFile(filename)
	if err := m.createFile(filename); err != nil {
		return fmt.Errorf("failed to create position file: %w", err)
	}
	if err := m.position.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read position file: %w", err)
		}
	}
	slog.Info("using position file", "path", m.position.ConfigFileUsed())
	m.position.WatchConfig()
	return nil
}

func (m *Monitor) checkIssuances(dc config.DomainConfig, c api.CertspotterClient, mailSender mailer.Mailer) error {
	key := getDomainConfigName(dc.Name)
	lastIssuance := m.position.GetUint64(key)
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
	issuances, err = filter.ApplyFilters(m.config.FilterConfig.Filters, issuances)
	if err != nil {
		slog.Info("errors encountered running filters", "error", err.Error(), "domain", dc.Name, "filters", m.config.FilterConfig.Filters)
	}
	if len(issuances) == 0 {
		m.position.Set(key, lastIssuance)
		return nil
	}
	tplVars := mailTemplateVars{
		Domain:    dc.Name,
		Issuances: issuances,
	}
	if err := sendMail(mailSender, tplVars, m.config.MailTemplate); err != nil {
		return err
	}
	m.position.Set(key, lastIssuance)
	slog.Info("done checking", "domain", dc.Name)
	return nil
}

func (m *Monitor) atomicWritePosition() error {
	filename := m.config.PositionConfig.Filename
	tmpFile, err := afero.TempFile(m.fs, filepath.Dir(filename), "position.*.toml")
	if err != nil {
		return err
	}
	if err := m.position.WriteConfigAs(tmpFile.Name()); err != nil {
		return err
	}
	fi, err := tmpFile.Stat()
	if err != nil {
		return err
	}
	if fi.Size() == 0 {
		return nil
	}
	return m.fs.Rename(tmpFile.Name(), filename)
}

func (m *Monitor) getMailSenderForDomain(dc config.DomainConfig, defaultMailSender mailer.Mailer) mailer.Mailer {
	if dc.Mailer == "" {
		return defaultMailSender
	}
	domainMailer := m.config.GetMailer(dc.Mailer)
	if err := domainMailer.Init(); err != nil {
		slog.Error("could not initialize domain mailer, using default", "error", err.Error(), "domain", dc.Name)
		return defaultMailSender
	}
	return domainMailer
}

func (m *Monitor) Run() error {
	if err := m.initPosition(); err != nil {
		return err
	}
	defaultMailSender := m.config.GetMailer(m.config.AlertConfig.Mailer)
	if err := defaultMailSender.Init(); err != nil {
		return err
	}
	csp := api.CertspotterClient{
		Endpoint: m.config.Endpoint,
		Token:    m.config.Token,
	}
	var checkErr error
	for _, dc := range m.config.Domains {
		mailSender := m.getMailSenderForDomain(dc, defaultMailSender)
		if err := m.checkIssuances(dc, csp, mailSender); err != nil {
			slog.Error("could not check issuances", "error", err.Error(), "domain", dc.Name)
			checkErr = ErrCheckIssuances
		}
	}
	return errors.Join(checkErr, m.atomicWritePosition())
}
