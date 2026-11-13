//go:build test
// +build test

package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Hsn723/certspotter-client/api"
	"github.com/Hsn723/ct-monitor/config"
	"github.com/Hsn723/ct-monitor/mailer"
	smtpmock "github.com/mocktools/go-smtp-mock/v2"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestGetDomainConfigName(t *testing.T) {
	t.Parallel()
	expect := "example-com"
	actual := getDomainConfigName("example.com")
	assert.Equal(t, expect, actual)
}

func TestGetTemplatedMailContent(t *testing.T) {
	t.Parallel()
	cases := []struct {
		title  string
		tmpl   string
		vars   mailTemplateVars
		expect string
		isErr  bool
	}{
		{
			title: "Success",
			tmpl:  "{{.Domain}}の証明書発行",
			vars: mailTemplateVars{
				Domain: "example.com",
			},
			expect: "example.comの証明書発行",
		},
		{
			title: "InvalidField",
			tmpl:  "{{.Hoge}}の証明書発行",
			vars: mailTemplateVars{
				Domain: "example.com",
			},
			isErr: true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Helper()
			actual, err := getTemplatedMailContent(tc.tmpl, tc.vars)
			if tc.isErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expect, actual)
		})
	}
}

func TestSendMail(t *testing.T) {
	t.Parallel()
	server := smtpmock.New(smtpmock.ConfigurationAttr{})
	err := server.Start()
	assert.NoError(t, err)
	mailer := mailer.SMTPMailer{
		From:   "root@example.com",
		To:     "admin@example.net",
		Server: "127.0.0.1",
		Port:   server.PortNumber(),
	}
	tmplVars := mailTemplateVars{
		Domain: "example.com",
		Issuances: []api.Issuance{
			{
				TBSSHA256: "db7c55f74732269c45fda91264003b2a25adc7ff2df687252f60772850449926",
				Domains: []string{
					"certs.sandbox.sslmate.com",
					"certs.sslmate.com",
					"sandbox.sslmate.com",
					"sslmate.com",
					"www.sslmate.com",
				},
				PubKeySHA256: "1b1cebcd061ba39746a477db7b90d6871d648bd293ef50e053a6c54c5c3ac112",
				Issuer: api.Issuer{
					Name:         "C=GB, ST=Greater Manchester, L=Salford, O=Sectigo Limited, CN=Sectigo RSA Domain Validation Secure Server CA",
					PubKeySHA256: "e1ae9c3de848ece1ba72e0d991ae4d0d9ec547c6bad1dddab9d6beb0a7e0e0d8",
					FriendlyName: "Sectigo",
					Website:      "https://sectigo.com/",
					CAADomains: []string{
						"sectigo.com",
						"comodo.com",
						"comodoca.com",
						"usertrust.com",
						"trust-provider.com",
					},
					Operator: api.Operator{
						Name:    "Sectigo",
						Website: "https://sectigo.com/",
					},
				},
				NotBefore:        "2022-10-22T00:00:00Z",
				NotAfter:         "2023-11-21T23:59:59Z",
				ProblemReporting: "To revoke one or more certificates issued by Sectigo for which you (i) are the Subscriber or (ii) control the domain or (iii) have in your possession the private key, you may use our automated Revocation Portal here:\u000A  ?? https://secure.sectigo.com/products/RevocationPortal\u000A\u000ATo programatically revoke one or more certificates issued by Sectigo for which you have in your possession the private key, you may use the ACME revokeCert method at this endpoint:\u000A  ?? ACME Directory: https://acme.sectigo.com/v2/keyCompromise\u000A  ?? revokeCert API: https://acme.sectigo.com/v2/keyCompromise/revokeCert\u000A\u000ATo report any other abuse, fraudulent, or malicious use of Certificates issued by Sectigo, please send email to:\u000A  ?? For Code Signing Certificates: signedmalwarealert[at]sectigo[dot]com\u000A  ?? For Other Certificates (SSL/TLS, S/MIME, etc): sslabuse[at]sectigo[dot]com",

				CertSHA256: "20cbc0d1e87ed1d71d3b84533667ef60f22fffee634108711376dec87a38d4e2",
			},
		},
	}
	mt := config.MailTemplate{
		Subject: config.DefaultSubjectTemplate,
		Body:    config.DefaultBodyTemplate,
	}
	err = sendMail(mailer, tmplVars, mt)
	assert.NoError(t, err)
	var messageData string
	assert.Eventually(t, func() bool {
		for _, message := range server.Messages() {
			if !message.Msg() {
				continue
			}
			messageData = message.MsgRequest()
			return true
		}
		return false
	}, 10*time.Second, 1*time.Second)
	expects := []string{
		"Certificate Transparency Notification for example.com",
		"ct-monitor has observed the issuance of the following certificate for the example.com domain:",
		"Issuer Friendly Name: Sectigo",
		"TBS SHA256: db7c55f74732269c45fda91264003b2a25adc7ff2df687252f60772850449926",
	}
	for _, expect := range expects {
		assert.Contains(t, messageData, expect)
	}
}

// TestRunRoot must not run in parallel since runRoot relies on global state
// (configFile, the global viper instance and the position viper instance).
func TestRunRoot(t *testing.T) {
	const (
		okDomain   = "ok.example.com"
		failDomain = "fail.example.com"
		okID       = 200
		failID     = 100
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("domain") {
		case okDomain:
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, `[{"id":"%d","dns_names":["%s"]}]`, okID, okDomain)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	cases := []struct {
		title   string
		domains []string
		isErr   bool
		expect  map[string]uint64
	}{
		{
			title:   "Success",
			domains: []string{okDomain},
			expect: map[string]uint64{
				okDomain: okID,
			},
		},
		{
			title:   "PartialFailure",
			domains: []string{failDomain, okDomain},
			isErr:   true,
			expect: map[string]uint64{
				okDomain:   okID,
				failDomain: failID,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			// initPosition starts a config watcher goroutine on the position instance,
			// so use a fresh instance for each case to avoid racing with the previous watcher.
			position = viper.New()
			dir := t.TempDir()
			positionFile := filepath.Join(dir, "positions.toml")
			initial := fmt.Sprintf("%s = %d\n", getDomainConfigName(failDomain), failID)
			assert.NoError(t, os.WriteFile(positionFile, []byte(initial), 0644))

			conf := fmt.Sprintf(`certspotter_endpoint = "%s"

[position_config]
filename = "%s"
`, server.URL, positionFile)
			for _, d := range tc.domains {
				conf += fmt.Sprintf("\n[[domain]]\nname = \"%s\"\n", d)
			}
			configFile = filepath.Join(dir, "config.toml")
			assert.NoError(t, os.WriteFile(configFile, []byte(conf), 0644))

			err := runRoot(nil, nil)
			if tc.isErr {
				assert.Error(t, err)
				assert.ErrorContains(t, err, failDomain)
			} else {
				assert.NoError(t, err)
			}

			// Read the written position file with a fresh viper instance.
			written := viper.New()
			written.SetConfigFile(positionFile)
			assert.NoError(t, written.ReadInConfig())
			for domain, id := range tc.expect {
				assert.Equal(t, id, written.GetUint64(getDomainConfigName(domain)), domain)
			}
		})
	}
}
