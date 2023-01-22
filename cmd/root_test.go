//go:build test
// +build test

package cmd

import (
	"testing"

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
