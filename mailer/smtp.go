package mailer

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

// SMTPMailer represents a mail sender for plain SMTP.
type SMTPMailer struct {
	From              string `mapstructure:"from"`
	To                string `mapstructure:"to"`
	Server            string `mapstructure:"server"`
	Port              int    `mapstructure:"port"`
	Username          string `mapstructure:"username"`
	Password          string `mapstructure:"password"`
	CaCert            string `mapstructure:"ca_cert_file"`
	RequireEncryption bool   `mapstructure:"require_encryption"`
}

// Init implements the Mailer's Init interface.
func (s SMTPMailer) Init() error {
	if s.From == "" {
		return ErrMissingSender
	}
	if s.To == "" {
		return ErrMissingRecipient
	}
	return nil
}

// Send implements the Mailer's Send interface.
func (s SMTPMailer) Send(subject, body string) error {
	addr := net.JoinHostPort(s.Server, strconv.Itoa(s.Port))
	msg := strings.NewReader(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s\r\n", s.To, subject, body))
	tos := []string{s.To}

	rootCAs, _ := LoadCACert(s.CaCert)
	tlsConfig := &tls.Config{
		ServerName: s.Server,
		RootCAs:    rootCAs,
	}

	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()

	if err := c.StartTLS(tlsConfig); err != nil {
		if s.RequireEncryption {
			return err
		}
	}
	if s.Username != "" && s.Password != "" {
		auth := sasl.NewPlainClient("", s.Username, s.Password)
		if err := c.Auth(auth); err != nil {
			return err
		}
	}

	return c.SendMail(s.From, tos, msg)
}
