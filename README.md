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
