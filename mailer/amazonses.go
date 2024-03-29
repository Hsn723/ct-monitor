package mailer

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sesv2"
	"github.com/cybozu-go/log"
)

// AmazonSESMailer represents a mail sender for Amazon SES.
type AmazonSESMailer struct {
	From    string `mapstructure:"from"`
	To      string `mapstructure:"to"`
	Region  string `mapstructure:"region"`
	Session *sesv2.SESV2
	Logger  *log.Logger
}

func (s *AmazonSESMailer) init() error {
	creds := credentials.NewEnvCredentials()

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(s.Region),
		Credentials: creds,
	})
	if err != nil {
		return err
	}
	s.Session = sesv2.New(sess)
	return nil
}

// Init implements the Mailer's Init interface.
func (s AmazonSESMailer) Init() error {
	if s.From == "" {
		return ErrMissingSender
	}
	if s.To == "" {
		return ErrMissingRecipient
	}
	if s.Session != nil {
		return nil
	}
	return s.init()
}

// Send implements the Mailer's Send interface.
func (s AmazonSESMailer) Send(subject, body string) error {
	charset := "UTF-8"
	email := &sesv2.SendEmailInput{
		Destination: &sesv2.Destination{
			ToAddresses: []*string{
				aws.String(s.To),
			},
		},
		Content: &sesv2.EmailContent{
			Simple: &sesv2.Message{
				Body: &sesv2.Body{
					Text: &sesv2.Content{
						Charset: aws.String(charset),
						Data:    aws.String(body),
					},
				},
				Subject: &sesv2.Content{
					Charset: aws.String(charset),
					Data:    aws.String(subject),
				},
			},
		},
		FromEmailAddress: aws.String(s.From),
	}
	res, err := s.Session.SendEmail(email)
	_ = log.Info("SES email sent", map[string]interface{}{
		"message_id": *res.MessageId,
	})
	return err
}
