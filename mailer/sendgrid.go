package mailer

import (
	"os"

	"github.com/cybozu-go/log"
	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

const (
	sendgridTokenEnv = "SENDGRID_TOKEN"
)

// SendgridMailer represents a mail sender for sendgrid.
type SendgridMailer struct {
	From   string `mapstructure:"from"`
	To     string `mapstructure:"to"`
	APIKey string `mapstructure:"token"`
	Client *sendgrid.Client
	Logger *log.Logger
}

func (s *SendgridMailer) init() {
	if s.APIKey == "" {
		s.APIKey = os.Getenv(sendgridTokenEnv)
	}
	s.Client = sendgrid.NewSendClient(s.APIKey)
}

// Init implements the Mailer's Init interface.
func (s SendgridMailer) Init(from, to string) error {
	if s.From == "" {
		return ErrMissingSender
	}
	if s.To == "" {
		return ErrMissingRecipient
	}
	if s.Client != nil {
		return nil
	}
	s.init()
	return nil
}

// Send implements the Mailer's Send interface.
func (s SendgridMailer) Send(subject, body string) error {
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
