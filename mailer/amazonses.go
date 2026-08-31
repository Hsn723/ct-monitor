package mailer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/cybozu-go/log"
)

// AmazonSESMailer represents a mail sender for Amazon SES.
type AmazonSESMailer struct {
	From    string `mapstructure:"from"`
	To      string `mapstructure:"to"`
	Region  string `mapstructure:"region"`
	Session *sesv2.Client
	Logger  *log.Logger
}

func (s *AmazonSESMailer) init() error {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(s.Region),
	)
	if err != nil {
		return err
	}
	s.Session = sesv2.NewFromConfig(cfg)
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
		Destination: &types.Destination{
			ToAddresses: []string{
				s.To,
			},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Body: &types.Body{
					Text: &types.Content{
						Charset: aws.String(charset),
						Data:    aws.String(body),
					},
				},
				Subject: &types.Content{
					Charset: aws.String(charset),
					Data:    aws.String(subject),
				},
			},
		},
		FromEmailAddress: aws.String(s.From),
	}
	res, err := s.Session.SendEmail(context.Background(), email)
	if err != nil {
		return err
	}
	_ = log.Info("SES email sent", map[string]interface{}{
		"message_id": *res.MessageId,
	})
	return nil
}
