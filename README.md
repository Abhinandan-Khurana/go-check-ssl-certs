<p align="center">
  <img src="go-check-ssl-certs.png" width="300" height="300">
</p>

<p align="center">
  <a href="https://goreportcard.com/report/github.com/Abhinandan-Khurana/go-check-ssl-certs"><img src="https://goreportcard.com/badge/github.com/Abhinandan-Khurana/go-check-ssl-certs" alt="Go Report Card"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="https://golang.org/doc/devel/release.html"><img src="https://img.shields.io/badge/Go-1.23+-00ADD8.svg" alt="Go Version"></a>
<img src="https://img.shields.io/badge/version-v0.1.0-blue.svg" alt="Version">
</p>

A powerful, flexible SSL certificate monitoring tool that checks the validity and expiration dates of SSL certificates for multiple domains. Built with Go for speed and efficiency.

## Features

- **Fast, concurrent certificate checking** - Process hundreds of domains in seconds
- **Multiple input methods** - Check individual domains, read from files, or pipe from other commands
- **Multiple output formats** - Terminal with color-coding, JSON, CSV, HTML, and XML output
- **Expiration thresholds** - Configurable warning and alert thresholds
- **Multiple notification channels**:
  - Slack
  - Telegram
  - Email
  - Discord
- **Cross-platform support** - Runs on macOS, Linux, and Windows (both amd64 and arm64)

## Installation

### Pre-built binaries

Download the latest release for your platform from the [releases page](https://github.com/Abhinandan-Khurana/go-check-ssl-certs/releases).

### Direct installation

```bash
go install -v github.com/Abhinandan-Khurana/go-check-ssl-certs@latest
```

### Build from source

```bash
# Clone the repository
git clone https://github.com/Abhinandan-Khurana/go-check-ssl-certs.git
cd go-check-ssl-certs

# Build
go build -o go-check-ssl-certs

# Or use the Makefile to build for all platforms
make
```

## Usage

```
go-check-ssl-certs [options]
```

### Basic Examples

Check a single domain:

```bash
go-check-ssl-certs -url example.com
```

Check multiple domains from a file:

```bash
go-check-ssl-certs -file domains.txt
```

Where `domains.txt` contains one domain per line:

```
example.com
github.com
google.com
# Lines starting with # are ignored
```

Pipe domains from another command:

```bash
cat domains.txt | go-check-ssl-certs
```

### Output Formats

Terminal output with colors (default):

```bash
go-check-ssl-certs -file domains.txt
```

![Terminal Output Example](./terminal%20output.png)

JSON output:

```bash
go-check-ssl-certs -file domains.txt -output json
```

```json
{
  "certificates": [
    {
      "url": "example.com",
      "domain": "example.com",
      "issuer": "CN=DigiCert TLS RSA SHA256 2020 CA1",
      "subject": "CN=example.com",
      "notBefore": "2023-01-15T00:00:00Z",
      "notAfter": "2025-01-14T23:59:59Z",
      "daysLeft": 254,
      "status": "OK",
      "checkedTime": "2025-05-05T10:15:30Z"
    },
    {
      "url": "expired-example.com",
      "domain": "expired-example.com",
      "issuer": "CN=Let's Encrypt Authority X3",
      "subject": "CN=expired-example.com",
      "notBefore": "2024-02-01T00:00:00Z",
      "notAfter": "2025-01-31T23:59:59Z",
      "daysLeft": 0,
      "status": "Expired",
      "checkedTime": "2025-05-05T10:15:30Z"
    }
  ]
}
```

CSV output:

```bash
go-check-ssl-certs -file domains.txt -output csv
```

HTML output:

```bash
go-check-ssl-certs -file domains.txt -output html > certificates.html
```

XML output:

```bash
go-check-ssl-certs -file domains.txt -output xml
```

### Notification Options

Send alerts to Slack:

```bash
go-check-ssl-certs -file domains.txt -slack-webhook https://hooks.slack.com/services/XXX/YYY/ZZZ
```

Send alerts to Telegram:

```bash
go-check-ssl-certs -file domains.txt -telegram-token YOUR_BOT_TOKEN -telegram-chat-id YOUR_CHAT_ID
```

Send alerts via email:

```bash
go-check-ssl-certs -file domains.txt -email-from sender@example.com -email-to recipient@example.com -email-server smtp.example.com:587 -email-user username -email-pass password
```

Send alerts to Discord:

```bash
go-check-ssl-certs -file domains.txt -discord-webhook https://discord.com/api/webhooks/XXX/YYY
```

Notification trigger level:

```bash
go-check-ssl-certs -file domains.txt -slack-webhook https://hooks.slack.com/services/XXX/YYY/ZZZ -notify-on warning
```

Available notification trigger levels: `alert` (default), `warning`, `expired`, `error`, `all`

## Advanced Options

Configure warning and alert thresholds:

```bash
go-check-ssl-certs -file domains.txt -warning 45 -alert 14
```

Increase concurrency for faster checks:

```bash
go-check-ssl-certs -file domains.txt -concurrency 50
```

Set connection timeout:

```bash
go-check-ssl-certs -file domains.txt -timeout 5s
```

Hide the startup banner for cleaner piped output:

```bash
go-check-ssl-certs -file domains.txt -no-banner
```

## Full Command Reference

```
Usage of go-check-ssl-certs:
  -alert int
        Days left threshold for Alert status (default 14)
  -concurrency int
        Number of concurrent checks (default 10)
  -debug
        Enable debug logging
  -discord-webhook string
        Discord webhook URL
  -email-from string
        Sender email address
  -email-pass string
        SMTP password
  -email-server string
        SMTP server (host:port)
  -email-to string
        Comma-separated list of email recipients
  -email-user string
        SMTP username
  -file string
        Path to a file containing URLs/domains (one per line)
  -no-banner
        Disable banner output
  -no-color
        Disable color output in terminal mode
  -notify-on string
        When to notify: alert, warning, expired, all (default "alert")
  -output string
        Output format: terminal, nocolor, json, csv, html, xml (default "terminal")
  -slack-webhook string
        Slack webhook URL for notifications
  -telegram-chat-id string
        Telegram Chat ID
  -telegram-token string
        Telegram Bot Token
  -throttle int
        Max requests per second (0 for no throttle)
  -timeout duration
        Connection timeout per domain (default 10s)
  -url string
        Single URL/domain to check (e.g., google.com or https://google.com:443)
  -verbose
        Enable verbose logging
  -version
        Show version and exit
  -warning int
        Days left threshold for Warning status (default 30)
```

## Examples of Tool Output

### Terminal Output

- **Green**: Certificates that are valid and far from expiration
- **Yellow**: Certificates that will expire within the warning threshold
- **Red**: Certificates that will expire within the alert threshold
- **Bold Red**: Certificates that have already expired
- **Purple**: Domains with connection errors

### Email Notification

```
Subject: SSL Certificate Alert

SSL Certificate Alert

The following 3 certificates require attention:

1. Domain: expired-example.com
   Status: Expired
   Expires: Jan 31 2025 23:59 UTC
   Days left: 0

2. Domain: example-alert.com
   Status: Alert
   Expires: May 18 2025 12:30 UTC
   Days left: 13

3. Domain: example-error.com
   Status: Error
   Error: Connection failed: dial tcp: lookup example-error.com: no such host

Checked at: Mon, 05 May 2025 10:15:30 UTC
```

## Integrating with CI/CD Pipelines

### GitHub Actions

Create a workflow file `.github/workflows/ssl-check.yml`:

```yaml
name: SSL Certificate Check

on:
  schedule:
    - cron: '0 0 * * *'  # Run daily at midnight UTC
  workflow_dispatch:      # Allow manual triggering

jobs:
  check-certificates:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v3
        
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
          
      - name: Download go-check-ssl-certs
        run: |
          curl -L https://github.com/Abhinandan-Khurana/go-check-ssl-certs/releases/download/v0.1.0/go-check-ssl-certs-linux-amd64.tar.gz -o ssl-checker.tar.gz
          tar -xzf ssl-checker.tar.gz
          chmod +x go-check-ssl-certs
          
      - name: Check SSL certificates
        run: |
          go-check-ssl-certs -file domains.txt -output json > ssl-results.json
          
      - name: Send notifications
        run: |
          go-check-ssl-certs -file domains.txt -slack-webhook ${{ secrets.SLACK_WEBHOOK_URL }} -notify-on warning
        
      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: ssl-check-results
          path: ssl-results.json
```

Be sure to:
1. Add your domains to `domains.txt` in your repository
2. Set up a `SLACK_WEBHOOK_URL` in your GitHub repository secrets

### GitLab CI

Create a `.gitlab-ci.yml` file:

```yaml
stages:
  - check

check-ssl-certificates:
  stage: check
  image: golang:1.21-alpine
  script:
    - apk add --no-cache curl tar
    - curl -L https://github.com/Abhinandan-Khurana/go-check-ssl-certs/releases/download/v0.1.0/go-check-ssl-certs-linux-amd64.tar.gz -o ssl-checker.tar.gz
    - tar -xzf ssl-checker.tar.gz
    - chmod +x go-check-ssl-certs
    - go-check-ssl-certs -file domains.txt -output json > ssl-results.json
    - go-check-ssl-certs -file domains.txt -slack-webhook ${SLACK_WEBHOOK_URL} -notify-on warning
  artifacts:
    paths:
      - ssl-results.json
    expire_in: 1 week
  only:
    - schedules  # Run based on GitLab scheduled pipelines
```

### Jenkins Pipeline

Create a `Jenkinsfile`:

```groovy
pipeline {
    agent any
    
    triggers {
        cron('0 0 * * *')  // Run daily at midnight
    }
    
    stages {
        stage('Check SSL Certificates') {
            steps {
                sh '''
                curl -L https://github.com/Abhinandan-Khurana/go-check-ssl-certs/releases/download/v0.1.0/go-check-ssl-certs-linux-amd64.tar.gz -o ssl-checker.tar.gz
                tar -xzf ssl-checker.tar.gz
                chmod +x go-check-ssl-certs
                go-check-ssl-certs -file domains.txt -output json > ssl-results.json
                go-check-ssl-certs -file domains.txt -slack-webhook ${SLACK_WEBHOOK_URL} -notify-on warning
                '''
                
                archiveArtifacts artifacts: 'ssl-results.json', fingerprint: true
            }
        }
    }
}
```

## Best Practices

1. **Schedule regular checks** - Set up your CI/CD pipeline to run daily or weekly checks
2. **Use different thresholds** - Set `-warning` to 30 days and `-alert` to 14 days for good advance notice
3. **Log historical data** - Save outputs to track certificate changes over time
4. **Use multiple notification channels** - Combine Slack and email for redundancy

## Contributing

Contributions, issues, and feature requests are welcome! Feel free to check the [issues page](https://github.com/Abhinandan-Khurana/go-check-ssl-certs/issues).

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request