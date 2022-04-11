package mailer

import (
	"crypto/x509"
	"fmt"
	"os"
)

// Mailer is a generic interface for mail senders.
type Mailer interface {
	Init() error
	Send(subject, body string) error
}

var (
	ErrMissingSender    = fmt.Errorf("sender address missing")
	ErrMissingRecipient = fmt.Errorf("recipient address missing")
)

func LoadCACert(path string) (*x509.CertPool, error) {
	var pool *x509.CertPool
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pool = x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(data); !ok {
		return nil, fmt.Errorf("error loading CA certificates")
	}
	return pool, nil
}
