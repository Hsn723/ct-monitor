//go:build test
// +build test

package mailer

import (
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/stretchr/testify/assert"
)

type mockBackend struct {
	t        *testing.T
	username string
	password string
	from     string
	to       string
}

type mockSession struct {
	backend *mockBackend
}

func (b *mockBackend) NewSession(_ smtp.ConnectionState, _ string) (smtp.Session, error) {
	return &mockSession{backend: b}, nil
}

func (s *mockSession) AuthPlain(username, password string) error {
	assert.Equal(s.backend.t, s.backend.username, username)
	assert.Equal(s.backend.t, s.backend.password, password)
	return nil
}

func (s *mockSession) Mail(from string, _ *smtp.MailOptions) error {
	assert.Equal(s.backend.t, s.backend.from, from)
	return nil
}

func (s *mockSession) Rcpt(to string) error {
	assert.Equal(s.backend.t, s.backend.to, to)
	return nil
}

func (s *mockSession) Data(r io.Reader) error {
	assert.NotEmpty(s.backend.t, r)
	return nil
}

func (s *mockSession) Reset() {}

func (s *mockSession) Logout() error {
	return nil
}

func TestSMTPInit(t *testing.T) {
	t.Parallel()
	cases := []struct {
		title    string
		mailer   SMTPMailer
		expected error
	}{
		{
			title: "Ok",
			mailer: SMTPMailer{
				From: "from@localhost",
				To:   "to@localhost",
			},
		},
		{
			title: "MissingSender",
			mailer: SMTPMailer{
				To: "to@localhost",
			},
			expected: ErrMissingSender,
		},
		{
			title: "MissingRecipient",
			mailer: SMTPMailer{
				From: "from@localhost",
			},
			expected: ErrMissingRecipient,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()
			actual := tc.mailer.Init()
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func testSMTPSend(t *testing.T, mailer SMTPMailer, backend *mockBackend, authDisabled, allowInsecureAuth, withServerTLS, withCAFile, isErrorExpected bool) {
	t.Helper()
	rand.Seed(time.Now().UnixNano())
	port := rand.Intn(50000) + 10000
	addr := fmt.Sprintf(":%d", port)

	s := smtp.NewServer(backend)
	s.Domain = "localhost"
	s.Addr = addr
	s.AuthDisabled = authDisabled
	s.AllowInsecureAuth = allowInsecureAuth
	caCert, key, err := createTestCA()
	assert.NoError(t, err)
	if withServerTLS {
		serverCert, serverKey, err := createServerCert(caCert, key)
		assert.NoError(t, err)
		cert, err := tls.LoadX509KeyPair(serverCert.Name(), serverKey.Name())
		assert.NoError(t, err)
		pool, err := LoadCACert(caCert.Name())
		assert.NoError(t, err)
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      pool,
		}
		s.TLSConfig = tlsConfig
	}

	if withCAFile {
		mailer.CaCert = caCert.Name()
	}

	mailer.Server = "localhost"
	mailer.Port = port

	go func() {
		assert.NoError(t, s.ListenAndServe())
	}()

	err = mailer.Send("hello", "world")
	if isErrorExpected {
		assert.Error(t, err)
	} else {
		assert.NoError(t, err)
	}
}

func TestSMTPSend(t *testing.T) {
	t.Parallel()
	cases := []struct {
		title             string
		mailer            SMTPMailer
		backend           *mockBackend
		authDisabled      bool
		allowInsecureAuth bool
		withServerTLS     bool
		withCAFile        bool
		isErrorExpected   bool
	}{
		{
			title: "Unauthenticated",
			mailer: SMTPMailer{
				From: "from@localhost",
				To:   "to@localhost",
			},
			backend: &mockBackend{
				t:    t,
				from: "from@localhost",
				to:   "to@localhost",
			},
		},
		{
			title: "UnsupportedAuth",
			mailer: SMTPMailer{
				From:     "from@localhost",
				To:       "to@localhost",
				Username: "hoge",
				Password: "hige",
			},
			backend: &mockBackend{
				t:    t,
				from: "from@localhost",
				to:   "to@localhost",
			},
			authDisabled:    true,
			isErrorExpected: true,
		},
		{
			title: "NoTLSSupport",
			mailer: SMTPMailer{
				From:              "from@localhost",
				To:                "to@localhost",
				RequireEncryption: true,
			},
			backend: &mockBackend{
				t:    t,
				from: "from@localhost",
				to:   "to@localhost",
			},
			isErrorExpected: true,
		},
		{
			title: "InsecureAuthWithRequiredTLS",
			mailer: SMTPMailer{
				From:     "from@localhost",
				To:       "to@localhost",
				Username: "hoge",
				Password: "hige",
			},
			backend: &mockBackend{
				t:    t,
				from: "from@localhost",
				to:   "to@localhost",
			},
			isErrorExpected: true,
		},
		{
			title: "AuthWithRequiredTLS",
			mailer: SMTPMailer{
				From:     "from@localhost",
				To:       "to@localhost",
				Username: "hoge",
				Password: "hige",
			},
			backend: &mockBackend{
				t:        t,
				from:     "from@localhost",
				to:       "to@localhost",
				username: "hoge",
				password: "hige",
			},
			withServerTLS: true,
			withCAFile:    true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()
			testSMTPSend(t, tc.mailer, tc.backend, tc.authDisabled, tc.allowInsecureAuth, tc.withServerTLS, tc.withCAFile, tc.isErrorExpected)
		})
	}
}
