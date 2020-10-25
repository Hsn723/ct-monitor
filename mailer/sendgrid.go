package mailer

import (
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"log"
)

type SendgridMailer struct {
	From   string
	To     string
	APIKey string
	Client *sendgrid.Client
}

func (s *SendgridMailer) Init(from, to string) error {
	s.From = from
	s.To = to
	if s.Client != nil {
		return nil
	}
	s.Client = sendgrid.NewSendClient(s.APIKey)
	return nil
}

func (s *SendgridMailer) Send(subject, body string) error {
	fromEmail := mail.NewEmail(s.From, s.From)
	toEmail := mail.NewEmail(s.To, s.To)
	message := mail.NewSingleEmail(fromEmail, subject, toEmail, body, "")
	res, err := s.Client.Send(message)
	log.Print(res)
	return err
}
