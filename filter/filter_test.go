//go:build test
// +build test

package filter

import (
	"testing"

	"github.com/Hsn723/certspotter-client/api"
	"github.com/stretchr/testify/assert"
)

const (
	testFilterBin = "/tmp/ct-monitor/work/testfilter"
)

func TestApplyFilters(t *testing.T) {
	t.Parallel()
	cases := []struct {
		title     string
		filter    string
		issuances []api.Issuance
		expected  []api.Issuance
		isErr     bool
	}{
		{
			title:     "Empty",
			filter:    testFilterBin,
			issuances: []api.Issuance{},
			expected:  []api.Issuance{},
		},
		{
			title:  "Single",
			filter: testFilterBin,
			issuances: []api.Issuance{
				{
					ID:           1,
					TBSSHA256:    "",
					Domains:      []string{"dummy"},
					PubKeySHA256: "",
					Issuer:       api.Issuer{},
					NotBefore:    "",
					NotAfter:     "",
					Cert:         api.Certificate{},
				},
			},
			expected: []api.Issuance{
				{
					ID:           1,
					TBSSHA256:    "",
					Domains:      []string{"dummy"},
					PubKeySHA256: "",
					Issuer:       api.Issuer{},
					NotBefore:    "",
					NotAfter:     "",
					Cert:         api.Certificate{},
				},
			},
		},
		{
			title:  "Multi",
			filter: testFilterBin,
			issuances: []api.Issuance{
				{
					ID:           1,
					TBSSHA256:    "",
					Domains:      []string{"dummy"},
					PubKeySHA256: "",
					Issuer:       api.Issuer{},
					NotBefore:    "",
					NotAfter:     "",
					Cert:         api.Certificate{},
				},
				{
					ID:           2,
					TBSSHA256:    "",
					Domains:      []string{"test"},
					PubKeySHA256: "",
					Issuer:       api.Issuer{},
					NotBefore:    "",
					NotAfter:     "",
					Cert:         api.Certificate{},
				},
			},
			expected: []api.Issuance{
				{
					ID:           1,
					TBSSHA256:    "",
					Domains:      []string{"dummy"},
					PubKeySHA256: "",
					Issuer:       api.Issuer{},
					NotBefore:    "",
					NotAfter:     "",
					Cert:         api.Certificate{},
				},
			},
		},
		{
			title:  "NonExisting",
			filter: "/dev/null",
			isErr:  true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()
			actual, err := ApplyFilters([]string{tc.filter}, tc.issuances)
			if tc.isErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.ElementsMatch(t, tc.expected, actual)
		})
	}
}
