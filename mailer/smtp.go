package mailer

import (
	"fmt"
	"strings"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

// SMTPMailer represents a mail sender for plain SMTP.
type SMTPMailer struct {
	From     string `mapstructure:"from"`
	To       string `mapstructure:"to"`
	Server   string `mapstructure:"server"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
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
	addr := fmt.Sprintf("%s:%d", s.Server, s.Port)
	msg := strings.NewReader(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s\r\n", s.To, subject, body))
	tos := []string{s.To}
	if s.Username != "" && s.Password != "" {
		auth := sasl.NewPlainClient("", s.Username, s.Password)
		return smtp.SendMail(addr, auth, s.From, tos, msg)
	}
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()
	return c.SendMail(s.From, tos, msg)
}
