# ct-monitor
[![GoDoc](https://godoc.org/github.com/Hsn723/ct-monitor?status.svg)](https://godoc.org/github.com/Hsn723/ct-monitor) [![Go Report Card](https://goreportcard.com/badge/github.com/Hsn723/ct-monitor)](https://goreportcard.com/report/github.com/Hsn723/ct-monitor) ![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/Hsn723/ct-monitor?label=latest%20version)


Queries [Cert Spotter](https://sslmate.com/certspotter/) for new certificate issuances. When new certificate issuances are found, an email report is sent. Currently supported email providers: [SendGrid](https://sendgrid.com/), [Amazon SES](https://aws.amazon.com/ses/), SMTP.

## Usage
```sh
Usage:
  ct-monitor [flags]

Flags:
  -c, --config string     path to configuration file (default "/etc/ct-monitor/config.toml")
  -h, --help              help for ct-monitor
```

## Plugins
Custom plugins can be specified to filter issuances or perform any extra work with the issuances detected. For instance, you may want to get certificate issuances for `example.com` including wildcard and subdomains, but ignore issuances for the `dev.example.com` subdomain only. Better yet, you can use plugins to implement your own mailer or send notifications to Slack instead of using the built-in mailer.

A plugin simply needs to implement the `IssuanceFilter` interface via `net/rpc`.

For instance, this plugin simply prints out the number of issuances and otherwise does not modify the slice of Issuance objects.

```go
package main

import (
	"github.com/Hsn723/certspotter-client/api"
	"github.com/Hsn723/ct-monitor/filter"
	"github.com/cybozu-go/log"
	"github.com/hashicorp/go-plugin"
)

type sampleFilter struct{}

func (sampleFilter) Filter(is []api.Issuance) ([]api.Issuance, error) {
	_ = log.Info("running sample filter", map[string]interface{}{
		"issuances": len(is),
	})
	return is, nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: filter.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			filter.PluginKey: &filter.IssuanceFilterPlugin{Impl: &sampleFilter{}},
		},
	})
}
```

For more detailed examples, refer to the documentation of [HashiCorp's go-plugin](https://github.com/hashicorp/go-plugin).

## Example config
```toml
[alert_config]
    mailer_config = "sendgrid"

[sendgrid]
    from = "me@example.com"
    to = "me@example.com"
    apiKey = "your-api-key"

[position_config]
    filename = "/var/log/ct-monitor/positions.toml"
```

For more details, check the documentation.
