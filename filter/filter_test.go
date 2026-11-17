//go:build test
// +build test

package filter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Hsn723/certspotter-client/api"
	"github.com/stretchr/testify/assert"
)

const (
	testFilterBin = "/tmp/ct-monitor/work/testfilter"
	luaDropDev = `function filter(is)
  local res = {}
  for _, i in ipairs(is) do
    local keep = true
    for _, d in ipairs(i.dns_names) do
      if d == "dev.example.org" then keep = false end
    end
    if keep then table.insert(res, i) end
  end
  return res
end`
	luaPassthrough = `function filter(is) return is end`
)

var (
	dummyIssuanceProd = api.Issuance{
		ID:           1,
		TBSSHA256:    "",
		Domains:      []string{"example.com", "www.example.com"},
		PubKeySHA256: "",
		Issuer:       api.Issuer{},
		NotBefore:    "",
		NotAfter:     "",
		Cert:         api.Certificate{},
	}
	dummyIssuanceDev = api.Issuance{
		ID:           2,
		TBSSHA256:    "",
		Domains:      []string{"dev.example.org"},
		PubKeySHA256: "",
		Issuer:       api.Issuer{},
		NotBefore:    "",
		NotAfter:     "",
		Cert:         api.Certificate{},
	}
)

func TestApplyFilters(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lua := func(name, content string) string {
		path := filepath.Join(dir, name+".lua")
		assert.NoError(t, os.WriteFile(path, []byte(content), 0644))
		return path
	}
	cases := []struct {
		title     string
		filters   []string
		issuances []api.Issuance
		expected  []api.Issuance
		isErr     bool
	}{
		{
			title:     "Empty",
			filters:   []string{testFilterBin},
			issuances: []api.Issuance{},
			expected:  []api.Issuance{},
		},
		{
			title:  "Single",
			filters: []string{testFilterBin},
			issuances: []api.Issuance{
				dummyIssuanceProd,
			},
			expected: []api.Issuance{
				dummyIssuanceProd,
			},
		},
		{
			title:  "Multi",
			filters: []string{testFilterBin},
			issuances: []api.Issuance{
				dummyIssuanceProd,
				dummyIssuanceDev,
			},
			expected: []api.Issuance{
				dummyIssuanceProd,
			},
		},
		{
			title:  "NonExisting",
			filters: []string{"/dev/null"},
			isErr:  true,
		},
		{
			title: "LuaPassthrough",
			filters: []string{
				lua("passthrough", luaPassthrough),
			},
			issuances: []api.Issuance{
				dummyIssuanceProd,
				dummyIssuanceDev,
			},
			expected: []api.Issuance{
				dummyIssuanceProd,
				dummyIssuanceDev,
			},
		},
		{
			title: "LuaEmpty",
			filters: []string{
				lua("empty", luaPassthrough),
			},
			issuances: []api.Issuance{},
			expected:  []api.Issuance{},
		},
		{
			title: "LuaDropDev",
			filters: []string{
				lua("dropdev", luaDropDev),
			},
			issuances: []api.Issuance{
				dummyIssuanceProd,
				dummyIssuanceDev,
			},
			expected: []api.Issuance{
				dummyIssuanceProd,
			},
		},
		{
			title: "LuaNoFilterFunction",
			filters: []string{
				lua("nofilter", `function notfilter(is) return is end`),
			},
			issuances: []api.Issuance{
				dummyIssuanceProd,
				dummyIssuanceDev,
			},
			expected: []api.Issuance{
				dummyIssuanceProd,
				dummyIssuanceDev,
			},
			isErr: true,
		},
		{
			title: "LuaSyntaxError",
			filters: []string{
				lua("syntaxerror", `function filter(is`),
			},
			issuances: []api.Issuance{
				dummyIssuanceProd,
			},
			expected: []api.Issuance{
				dummyIssuanceProd,
			},
			isErr: true,
		},
		{
			title: "LuaRuntimeError",
			filters: []string{
				lua("runtime", `function filter(is) error("boom") end`),
			},
			issuances: []api.Issuance{
				dummyIssuanceProd,
			},
			expected: []api.Issuance{
				dummyIssuanceProd,
			},
			isErr: true,
		},
		{
			title: "LuaReturnNonTable",
			filters: []string{
				lua("nil", `function filter(is) return nil end`),
			},
			issuances: []api.Issuance{
				dummyIssuanceProd,
			},
			expected: []api.Issuance{
				dummyIssuanceProd,
			},
			isErr: true,
		},
		{
			title: "LuaSingleIssuanceInsteadOfList",
			filters: []string{
				lua("single", `function filter(is) return is[1] end`),
			},
			issuances: []api.Issuance{
				dummyIssuanceProd,
			},
			expected: []api.Issuance{
				dummyIssuanceProd,
			},
			isErr: true,
		},
		{
			title: "LuaCycle",
			filters: []string{
				lua("cycle", `function filter(is) local t = {} t[1] = t return t end`),
			},
			issuances: []api.Issuance{
				dummyIssuanceProd,
			},
			expected: []api.Issuance{
				dummyIssuanceProd,
			},
			isErr: true,
		},
		{
			title: "LuaNonExisting",
			filters:   []string{"/nonexistent/filter.lua"},
			issuances: []api.Issuance{
				dummyIssuanceProd,
			},
			expected: []api.Issuance{
				dummyIssuanceProd,
			},
			isErr: true,
		},
		{
			title: "MixFilters",
			filters: []string{
				testFilterBin,
				lua("mixrpclua", luaDropDev),
			},
			issuances: []api.Issuance{
				dummyIssuanceDev,
				dummyIssuanceProd,
			},
			expected: []api.Issuance{},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()
			actual, err := ApplyFilters(tc.filters, tc.issuances)
			if tc.isErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.ElementsMatch(t, tc.expected, actual)
		})
	}
}
