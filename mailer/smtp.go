package mailer

import (
	"fmt"
	"net/smtp"
)

// SMTPMailer represents a mail sender for plain SMTP
type SMTPMailer struct {
	From     string
	To       string
	Server   string
	Port     int
	Username string
	Password string
	Auth     *smtp.Auth
}

// Init implements the Mailer's Init interface
func (s *SMTPMailer) Init(from, to string) error {
	s.From = from
	s.To = to
	if s.Auth != nil {
		return nil
	}
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Server)
	s.Auth = &auth
	return nil
}

// Send implements the Mailer's Send interface
func (s *SMTPMailer) Send(subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.Server, s.Port)
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s\r\n", s.To, subject, body))
	return smtp.SendMail(addr, *s.Auth, s.From, []string{s.To}, msg)
}
