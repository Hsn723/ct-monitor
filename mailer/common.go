package mailer

type Mailer interface {
	Init(from, to string) error
	Send(subject, body string) error
}
