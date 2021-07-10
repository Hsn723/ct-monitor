package mailer

import (
	"github.com/cybozu-go/log"
	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// SendgridMailer represents a mail sender for sendgrid.
type SendgridMailer struct {
	From   string
	To     string
	APIKey string
	Client *sendgrid.Client
	Logger *log.Logger
}

// Init implements the Mailer's Init interface.
func (s *SendgridMailer) Init(from, to string) error {
	s.From = from
	s.To = to
	if s.Client != nil {
		return nil
	}
	s.Client = sendgrid.NewSendClient(s.APIKey)
	return nil
}

// Send implements the Mailer's Send interface.
func (s *SendgridMailer) Send(subject, body string) error {
	fromEmail := mail.NewEmail(s.From, s.From)
	toEmail := mail.NewEmail(s.To, s.To)
	message := mail.NewSingleEmail(fromEmail, subject, toEmail, body, "")
	res, err := s.Client.Send(message)
	_ = log.Info("sendgrid response", map[string]interface{}{
		"status_code": res.StatusCode,
		"body":        res.Body,
		"headers":     res.Headers,
	})
	return err
}
