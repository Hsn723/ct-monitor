//go:build test
// +build test

package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"text/template"
	"time"

	"github.com/Hsn723/certspotter-client/api"
	"github.com/Hsn723/ct-monitor/config"
	"github.com/Hsn723/ct-monitor/mailer"
	smtpmock "github.com/mocktools/go-smtp-mock/v2"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

const testConfigTemplate = `certspotter_endpoint = "{{ .Endpoint }}"

[position_config]
filename = "{{ .PositionFile }}"
{{ range .Domains }}
[[domain]]
name = "{{ . }}"
{{ end }}`

const (
	testDomain      = "example.com"
	testOtherDomain = "example.net"
)

func TestGetDomainConfigName(t *testing.T) {
	t.Parallel()
	expect := "example-com"
	actual := getDomainConfigName(testDomain)
	assert.Equal(t, expect, actual)
}

func TestGetTemplatedMailContent(t *testing.T) {
	t.Parallel()
	cases := []struct {
		title   string
		tmpl    string
		vars    mailTemplateVars
		expect  string
		errFunc assert.ErrorAssertionFunc
	}{
		{
			title: "Success",
			tmpl:  "{{.Domain}}の証明書発行",
			vars: mailTemplateVars{
				Domain: testDomain,
			},
			expect:  "example.comの証明書発行",
			errFunc: assert.NoError,
		},
		{
			title: "InvalidField",
			tmpl:  "{{.Hoge}}の証明書発行",
			vars: mailTemplateVars{
				Domain: testDomain,
			},
			errFunc: assert.Error,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Helper()
			actual, err := getTemplatedMailContent(tc.tmpl, tc.vars)
			tc.errFunc(t, err)
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

type testConfig struct {
	Endpoint     string
	PositionFile string
	Domains      []string
}

func newTestServer(t *testing.T, issuances map[string][]uint64) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ids, ok := issuances[r.URL.Query().Get("domain")]
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		res := []api.Issuance{}
		for _, id := range ids {
			res = append(res, api.Issuance{ID: id})
		}
		assert.NoError(t, json.NewEncoder(w).Encode(res))
	}))
	t.Cleanup(server.Close)
	return server
}

func writeTestConfig(t *testing.T, path string, c testConfig) {
	t.Helper()
	tmpl := template.Must(template.New("config").Parse(testConfigTemplate))
	f, err := fs.Create(path)
	assert.NoError(t, err)
	defer f.Close()
	assert.NoError(t, tmpl.Execute(f, c))
}

func readTestPositions(t *testing.T, path string) map[string]uint64 {
	t.Helper()
	data, err := afero.ReadFile(fs, path)
	assert.NoError(t, err)
	positions := map[string]uint64{}
	assert.NoError(t, toml.Unmarshal(data, &positions))
	return positions
}

func TestRunRoot(t *testing.T) {
	assertSpecificErrs := func(errs []error, notErrs []error) func(t assert.TestingT, err error, msgAndArgs ...interface{}) bool {
		return func(t assert.TestingT, err error, msgAndArgs ...interface{}) bool {
			for _, e := range errs {
				if !assert.ErrorIs(t, err, e, msgAndArgs...) {
					return false
				}
			}
			for _, e := range notErrs {
				if !assert.NotErrorIs(t, err, e, msgAndArgs...) {
					return false
				}
			}
			return true
		}
	}
	cases := []struct {
		title            string
		domains          []string
		issuances        map[string][]uint64
		existingPosition []byte
		readOnlyDir      bool
		expect           map[string]uint64
		assertErr        assert.ErrorAssertionFunc
	}{
		{
			title:   "Success",
			domains: []string{testOtherDomain, testDomain},
			issuances: map[string][]uint64{
				testDomain:      {100, 200},
				testOtherDomain: {300},
			},
			expect:    map[string]uint64{"example-com": 200, "example-net": 300},
			assertErr: assert.NoError,
		},
		{
			title:     "PartialFailure",
			domains:   []string{testOtherDomain, testDomain},
			issuances: map[string][]uint64{testDomain: {100, 200}},
			expect:    map[string]uint64{"example-com": 200},
			assertErr: assertSpecificErrs([]error{errCheckIssuances}, []error{os.ErrPermission}),
		},
		{
			title:     "Failure",
			domains:   []string{testOtherDomain, testDomain},
			assertErr: assertSpecificErrs([]error{errCheckIssuances}, []error{os.ErrPermission}),
		},
		{
			title:            "PositionWriteFailure",
			domains:          []string{testOtherDomain, testDomain},
			issuances:        map[string][]uint64{testDomain: {100, 200}, testOtherDomain: {300}},
			existingPosition: []byte{},
			readOnlyDir:      true,
			assertErr:        assertSpecificErrs([]error{os.ErrPermission}, []error{errCheckIssuances}),
		},
		{
			title:            "CheckAndPositionWriteFailure",
			domains:          []string{testOtherDomain, testDomain},
			issuances:        map[string][]uint64{testDomain: {100, 200}},
			existingPosition: []byte{},
			readOnlyDir:      true,
			assertErr:        assertSpecificErrs([]error{errCheckIssuances, os.ErrPermission}, nil),
		},
		{
			title:            "PositionInitFailure",
			domains:          []string{testDomain},
			issuances:        map[string][]uint64{testDomain: {100, 200}},
			existingPosition: []byte("invalid"),
			assertErr:        assert.Error,
		},
	}
	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			position = viper.New()
			fs = afero.NewMemMapFs()
			server := newTestServer(t, tc.issuances)
			dir := t.TempDir()
			positionDir := filepath.Join(dir, "positions")
			assert.NoError(t, os.Mkdir(positionDir, 0755))
			positionFile := filepath.Join(positionDir, "positions.toml")
			if tc.existingPosition != nil {
				err := afero.WriteFile(fs, positionFile, tc.existingPosition, 0644)
				assert.NoError(t, err)
			}
			configFile = filepath.Join(dir, "config.toml")
			writeTestConfig(t, configFile, testConfig{
				Endpoint:     server.URL,
				PositionFile: positionFile,
				Domains:      tc.domains,
			})
			if tc.readOnlyDir {
				fs = afero.NewReadOnlyFs(fs)
			}
			position.SetFs(fs)
			tc.assertErr(t, runRoot(nil, nil))
			if tc.expect != nil {
				assert.Equal(t, tc.expect, readTestPositions(t, positionFile))
			}
		})
	}
}
