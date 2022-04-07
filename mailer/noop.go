package mailer

import "github.com/cybozu-go/log"

// NoOpMailer is a dummy mailer that doesn't send mail.
type NoOpMailer struct{}

// Init implements the Mailer's Init interface.
func (m NoOpMailer) Init() error {
	return nil
}

// Send implements the Mailer's Send interface.
func (m NoOpMailer) Send(_, _ string) error {
	_ = log.Info("no-op mailer, no email reports will be sent", nil)
	return nil
}
