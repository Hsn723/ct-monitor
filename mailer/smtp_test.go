//go:build test
// +build test

package mailer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
				To:   "to@localhost",
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
