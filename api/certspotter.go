package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	defaultParams = "?expand=dns_names&expand=issuer&expand=cert"
)

// HTTPClient is an interface implementing for the http.Client Do function.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// CertspotterClient is a client for interacting with the certspotter API.
type CertspotterClient struct {
	Endpoint string
	Token    string
	Client   HTTPClient
}

// Issuance represents an issuance object for the certspotter API.
type Issuance struct {
	ID           uint64      `json:"id,string"`
	TBSSHA256    string      `json:"tbs_sha256"`
	Domains      []string    `json:"dns_names"`
	PubKeySHA256 string      `json:"pubkey_sha256"`
	Issuer       Issuer      `json:"issuer"`
	NotBefore    string      `json:"not_before"`
	NotAfter     string      `json:"not_after"`
	Cert         Certificate `json:"cert"`
}

// Issuer represents an issuer object for the certspotter API.
type Issuer struct {
	Name         string `json:"name"`
	PubKeySHA256 string `json:"pubkey_sha256"`
}

// Certificate represents a certificate object for the certspotter API.
type Certificate struct {
	Type   string `json:"type"`
	SHA256 string `json:"sha256"`
	Data   string `json:"data"`
}

func (c *CertspotterClient) getQueryParams(domain string, matchWildcards, includeSubdomains bool, position uint64) string {
	q := fmt.Sprintf("%s&domain=%s", defaultParams, domain)
	if position > 0 {
		q = fmt.Sprintf("%s&after=%d", q, position)
	}
	if matchWildcards {
		q += "&match_wildcards=true"
	}
	if includeSubdomains {
		q += "&include_subdomains=true"
	}
	return q
}

func (c *CertspotterClient) get(queryParams string) ([]byte, error) {
	if c.Client == nil {
		c.Client = http.DefaultClient
	}
	req, err := http.NewRequest(http.MethodGet, c.Endpoint+queryParams, nil)
	if err != nil {
		return nil, err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	res, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusTooManyRequests {
		retryAfter := res.Header.Get("Retry-After")
		return nil, fmt.Errorf("hit Cert Spotter's API limit. Retry-After: %v", retryAfter)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("undocumented status code returned by the Cert Spotter API: %d", res.StatusCode)
	}
	return ioutil.ReadAll(res.Body)
}

// GetIssuances queries the certspotter API for new issuances.
func (c *CertspotterClient) GetIssuances(domain string, matchWildcards, includeSubdomains bool, position uint64) ([]Issuance, error) {
	queryParams := c.getQueryParams(domain, matchWildcards, includeSubdomains, position)

	body, err := c.get(queryParams)
	if err != nil {
		return nil, err
	}
	var issuances []Issuance
	if err := json.Unmarshal(body, &issuances); err != nil {
		return nil, err
	}
	return issuances, nil
}
