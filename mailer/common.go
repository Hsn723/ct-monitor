package mailer

// Mailer is a generic interface for mail senders
type Mailer interface {
	Init(from, to string) error
	Send(subject, body string) error
}
