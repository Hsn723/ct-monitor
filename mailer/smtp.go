package mailer

import (
	"fmt"
	"net/smtp"
)

type SMTPMailer struct {
	From     string
	To       string
	Server   string
	Port     int
	Username string
	Password string
	Auth     *smtp.Auth
}

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

func (s *SMTPMailer) Send(subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.Server, s.Port)
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s\r\n", s.To, subject, body))
	return smtp.SendMail(addr, *s.Auth, s.From, []string{s.To}, msg)
}
