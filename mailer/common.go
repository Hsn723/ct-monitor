package mailer

import "fmt"

// Mailer is a generic interface for mail senders.
type Mailer interface {
	Init() error
	Send(subject, body string) error
}

var (
	ErrMissingSender    = fmt.Errorf("sender address missing")
	ErrMissingRecipient = fmt.Errorf("recipient address missing")
)
