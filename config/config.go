package config

import (
	"reflect"
	"strings"

	"github.com/Hsn723/ct-monitor/mailer"
	"github.com/spf13/viper"
)

const (
	DefaultConfigFile          = "/etc/ct-monitor/config.toml"
	defaultPositionFile        = "/var/log/ct-monitor/positions.toml"
	defaultMailer              = NoOpMailer
	defaultCertspotterEndpoint = "https://api.certspotter.com/v1/issuances"
	certspotterTokenEnv        = "CERTSPOTTER_TOKEN"
	DefaultSubjectTemplate     = "Certificate Transparency Notification for {{.Domain}}"
	DefaultBodyTemplate        = `ct-monitor has observed the issuance of the following certificate{{ if gt (len .Issuances) 1}}s{{end}} for the {{.Domain}} domain:
{{range .Issuances}}
Issuer Friendly Name: {{.Issuer.FriendlyName}}
Issuer Distinguished Name: {{.Issuer.Name}}
DNS Names: {{.Domains}}
Validity: {{.NotBefore}} - {{.NotAfter}}
SHA256: {{.CertSHA256}}
TBS SHA256: {{.TBSSHA256}}

{{.ProblemReporting}}
{{end}}`
)

// Config contains the configuration for ct-monitor.
type Config struct {
	// Domains is a list of domain configurations.
	Domains []DomainConfig `mapstructure:"domain"`
	// Endpoint is the CertSpotter API endpoint.
	// This defaults to "https://api.certspotter.com/v1/issuances".
	Endpoint string `mapstructure:"certspotter_endpoint"`
	// Token is the token used for interacting with the CertSpotter API.
	// This can also be provided via the CERTSPOTTER_TOKEN environment variable.
	Token string `mapstructure:"certspotter_token"`
	// AlertConfig represents the configuration for alert mails.
	AlertConfig AlertConfig `mapstructure:"alert_config"`
	// PositionConfig represents the configuration for recording log position.
	PositionConfig PositionConfig `mapstructure:"position_config"`
	// AmazonSES represents the mailer configuration for using Amazon Simple Email Service.
	AmazonSES mailer.AmazonSESMailer `mapstructure:"amazonses"`
	// Sendgrid represents the mailer configuration for using Sendgrid.
	Sendgrid mailer.SendgridMailer `mapstructure:"sendgrid"`
	// SMTP represents the mailer configuration for using plain SMTP.
	SMTP mailer.SMTPMailer `mapstructure:"smtp"`
	// FilterConfig represent filter plugin configuration.
	FilterConfig FilterConfig `mapstructure:"filter_config"`
	// MailTemplate represents template strings for emails being sent out.
	MailTemplate MailTemplate `mapstructure:"mail_template"`
}

// DomainConfig contains domain configurations.
type DomainConfig struct {
	// Name is the FQDN of the domain to query for.
	Name string `mapstructure:"name"`
	// MatchWildcards should be set to true to include wildcards.
	MatchWildcards bool `mapstructure:"match_wildcards"`
	// IncludeSubdomains should be set to true if subdomains should also be scanned.
	IncludeSubdomains bool `mapstructure:"include_subdomains"`
	// Mailer is the name of the mail provider to use for this domain.
	// If not provided, the global configuration in alert_config is used.
	Mailer Mailer `mapstructure:"mailer_config"`
}

// AlertConfig contains alert configuration.
type AlertConfig struct {
	// Mailer is the name of the mail provider to use.
	// If the provider doesn't exist or isn't configured,
	// the no-op mailer will be used.
	Mailer Mailer `mapstructure:"mailer_config"`
}

// PositionConfig represents a position file config.
type PositionConfig struct {
	Filename string `mapstructure:"filename"`
}

type FilterConfig struct {
	Filters []string `mapstructure:"filters"`
}

// MailTemplate represents template strings for emails being sent out.
// The following variables are made available for templating.
//     Domain: the configured domain name which was queried.
//     Issuances: the Issuance object returned by the certspotter API.
type MailTemplate struct {
	Subject string `mapstructure:"subject"`
	Body    string `mapstructure:"body"`
}

// Mailer represents a mailer name.
type Mailer string

const (
	AmazonSESMailer Mailer = "amazonses"
	SendgridMailer  Mailer = "sendgrid"
	SMTPMailer      Mailer = "smtp"
	NoOpMailer      Mailer = "none"
)

// Load loads the configuration from file.
func Load(confFile string) (conf *Config, err error) {
	viper.SetConfigFile(confFile)
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	conf = &Config{
		Endpoint: defaultCertspotterEndpoint,
		AlertConfig: AlertConfig{
			Mailer: defaultMailer,
		},
		PositionConfig: PositionConfig{
			Filename: defaultPositionFile,
		},
		MailTemplate: MailTemplate{
			Subject: DefaultSubjectTemplate,
			Body:    DefaultBodyTemplate,
		},
	}
	if err := viper.Unmarshal(&conf); err != nil {
		return nil, err
	}
	if conf.Token == "" {
		conf.Token = viper.GetString(certspotterTokenEnv)
	}
	return conf, nil
}

// GetMailer retrieves a Mailer instance from the configuration.
func (c *Config) GetMailer(name Mailer) (m mailer.Mailer) {
	defer func() {
		if err := recover(); err != nil {
			m = mailer.NoOpMailer{}
			return
		}
	}()
	r := reflect.ValueOf(c)
	f := reflect.Indirect(r).FieldByNameFunc(func(s string) bool {
		return strings.ToLower(s) == string(name)
	})
	m = f.Interface().(mailer.Mailer)
	return m
}
