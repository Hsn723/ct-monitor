// +build test

package config

import (
	"reflect"
	"testing"

	"github.com/Hsn723/ct-monitor/mailer"
	"github.com/stretchr/testify/assert"
)

func testLoad(t *testing.T, confFile string, expectedConfig *Config, isErrorExpected bool) {
	t.Helper()
	actualConfig, err := Load(confFile)
	if isErrorExpected {
		assert.Error(t, err)
	} else {
		assert.NoError(t, err)
	}
	if !reflect.DeepEqual(actualConfig, expectedConfig) {
		t.Errorf("expected %v, got %v", expectedConfig, actualConfig)
	}
}

func TestLoad(t *testing.T) {
	cases := []struct {
		title           string
		file            string
		expected        *Config
		isErrorExpected bool
	}{
		{
			title: "AllFields",
			file:  "t/full.toml",
			expected: &Config{
				Domains: []DomainConfig{
					{
						Name:           "example.com",
						MatchWildcards: true,
					},
					{
						Name:              "example.jp",
						IncludeSubdomains: true,
					},
				},
				Endpoint:       "dummy.endpoint",
				Token:          "dummy",
				PositionConfig: PositionConfig{Filename: "positions.toml"},
				AlertConfig:    AlertConfig{Mailer: SendgridMailer},
				SMTP: mailer.SMTPMailer{
					From:   "from@example.com",
					To:     "to@example.com",
					Server: "localhost",
					Port:   25,
				},
				Sendgrid: mailer.SendgridMailer{
					From:   "from@example.com",
					To:     "to@example.com",
					APIKey: "hoge",
				},
			},
		},
		{
			title: "Defaults",
			file:  "t/defaults.toml",
			expected: &Config{
				Domains: []DomainConfig{
					{
						Name:           "example.com",
						MatchWildcards: true,
					},
					{
						Name:              "example.jp",
						IncludeSubdomains: true,
					},
				},
				Endpoint:       defaultCertspotterEndpoint,
				Token:          "",
				PositionConfig: PositionConfig{Filename: defaultPositionFile},
				AlertConfig:    AlertConfig{Mailer: NoOpMailer},
				SMTP: mailer.SMTPMailer{
					From:   "from@example.com",
					To:     "to@example.com",
					Server: "localhost",
					Port:   25,
				},
				Sendgrid: mailer.SendgridMailer{
					From:   "from@example.com",
					To:     "to@example.com",
					APIKey: "hoge",
				},
			},
		},
		{
			title:           "NoFile",
			file:            "t/dummy.toml",
			expected:        nil,
			isErrorExpected: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			testLoad(t, tc.file, tc.expected, tc.isErrorExpected)
		})
	}
}

func TestLoadEnv(t *testing.T) {
	t.Setenv(certspotterTokenEnv, "dummy-from-env")
	expected := &Config{
		Domains: []DomainConfig{
			{
				Name:           "example.com",
				MatchWildcards: true,
			},
			{
				Name:              "example.jp",
				IncludeSubdomains: true,
			},
		},
		Endpoint:       defaultCertspotterEndpoint,
		Token:          "dummy-from-env",
		PositionConfig: PositionConfig{Filename: defaultPositionFile},
		AlertConfig:    AlertConfig{Mailer: NoOpMailer},
		SMTP: mailer.SMTPMailer{
			From:   "from@example.com",
			To:     "to@example.com",
			Server: "localhost",
			Port:   25,
		},
		Sendgrid: mailer.SendgridMailer{
			From:   "from@example.com",
			To:     "to@example.com",
			APIKey: "hoge",
		},
	}
	testLoad(t, "t/defaults.toml", expected, false)
}

func testGetMailer(t *testing.T, conf Config, expected mailer.Mailer) {
	t.Helper()
	actual := conf.GetMailer(conf.AlertConfig.Mailer)
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

func TestGetMailer(t *testing.T) {
	t.Parallel()
	cases := []struct {
		title    string
		conf     Config
		expected mailer.Mailer
	}{
		{
			title: "SMTP",
			conf: Config{
				AlertConfig: AlertConfig{Mailer: SMTPMailer},
				SMTP:        mailer.SMTPMailer{},
			},
			expected: mailer.SMTPMailer{},
		},
		{
			title: "NoOp",
			conf: Config{
				AlertConfig: AlertConfig{Mailer: NoOpMailer},
				SMTP:        mailer.SMTPMailer{},
			},
			expected: mailer.NoOpMailer{},
		},
		{
			title: "Mispell",
			conf: Config{
				AlertConfig: AlertConfig{Mailer: "hoge"},
				SMTP:        mailer.SMTPMailer{},
			},
			expected: mailer.NoOpMailer{},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()
			testGetMailer(t, tc.conf, tc.expected)
		})
	}
}
