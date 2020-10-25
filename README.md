# ct-monitor
[![CircleCI](https://circleci.com/gh/Hsn723/ct-monitor.svg?style=svg)](https://circleci.com/gh/Hsn723/ct-monitor) [![Docker Repository on Quay](https://quay.io/repository/hsn723/ct-monitor/status "Docker Repository on Quay")](https://quay.io/repository/hsn723/ct-monitor) [![GoDoc](https://godoc.org/github.com/Hsn723/ct-monitor?status.svg)](https://godoc.org/github.com/Hsn723/ct-monitor) [![Go Report Card](https://goreportcard.com/badge/github.com/Hsn723/ct-monitor)](https://goreportcard.com/report/github.com/Hsn723/ct-monitor) ![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/Hsn723/ct-monitor?label=latest%20version)


Queries [Cert Spotter](https://sslmate.com/certspotter/) for new certificate issuances. When new certificate issuances are found, an email report is sent. Currently supported email providers: [SendGrid](https://sendgrid.com/), [Amazon SES](https://aws.amazon.com/ses/), SMTP.

## Usage
```sh
Usage:
  ct-monitor [flags]

Flags:
  -c, --config string     path to configuration file (default "/etc/ct-monitor/config.toml")
  -d, --domains strings   domains to query certspotter for issuances
  -e, --endpoint string   API endpoint (default "https://api.certspotter.com/v1/issuances")
  -h, --help              help for ct-monitor
  -p, --position string   path to position file (default "/var/log/ct-monitor/positions.toml")
  -s, --subdomains        include subdomains (default true)
  -t, --token string      API token
  -w, --wildcard          match wildcards (default true)
```

## Example config
```toml
[alert_config]
    from = me@example.com
    recipient = me@example.com
    mailer_config = sendgrid

[sendgrid]
    apiKey = "your-api-key"

[position_config]
    filename = /var/log/ct-monitor/positions.toml
```
