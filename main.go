package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"sync"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/fatih/color"
)

const toolVersion = "0.1.1"

// CertInfo holds the result for a single domain check
type CertInfo struct {
	XMLName     xml.Name  `xml:"CertificateInfo" json:"-"` // For XML marshalling
	URL         string    `json:"url" xml:"URL"`
	Domain      string    `json:"domain" xml:"Domain"`
	Issuer      string    `json:"issuer" xml:"Issuer"`
	Subject     string    `json:"subject" xml:"Subject"`
	NotBefore   time.Time `json:"notBefore" xml:"NotBefore"`
	NotAfter    time.Time `json:"notAfter" xml:"NotAfter"`
	DaysLeft    int       `json:"daysLeft" xml:"DaysLeft"`
	Status      string    `json:"status" xml:"Status"` // OK, Warning, Alert, Expired, Error
	Error       string    `json:"error,omitempty" xml:"Error,omitempty"`
	CheckedTime time.Time `json:"checkedTime" xml:"CheckedTime"`
}

// Results holds multiple CertInfo structs, useful for structured output
type Results struct {
	XMLName xml.Name   `xml:"SSLCheckResults" json:"-"` // For XML marshalling
	Certs   []CertInfo `xml:"CertificateInfo" json:"certificates"`
}

// Configuration flags
var (
	targetURL            = flag.String("url", "", "Single URL/domain to check (e.g., google.com or https://google.com:443)")
	inputFile            = flag.String("file", "", "Path to a file containing URLs/domains (one per line)")
	outputFormat         = flag.String("output", "terminal", "Output format: terminal, nocolor, json, csv, html, xml")
	concurrency          = flag.Int("concurrency", 10, "Number of concurrent checks")
	throttle             = flag.Int("throttle", 0, "Max requests per second (0 for no throttle)")
	warningDays          = flag.Int("warning", 30, "Days left threshold for Warning status")
	alertDays            = flag.Int("alert", 14, "Days left threshold for Alert status")
	verbose              = flag.Bool("verbose", false, "Enable verbose logging")
	debug                = flag.Bool("debug", false, "Enable debug logging")
	showVersion          = flag.Bool("version", false, "Show version and exit")
	noColor              = flag.Bool("no-color", false, "Disable color output in terminal mode")
	noBanner             = flag.Bool("no-banner", false, "Disable banner output")
	timeout              = flag.Duration("timeout", 10*time.Second, "Connection timeout per domain")
	notifySlackURL       = flag.String("slack-webhook", "", "Slack webhook URL for notifications")
	notifyTelegramToken  = flag.String("telegram-token", "", "Telegram Bot Token")
	notifyTelegramChatID = flag.String("telegram-chat-id", "", "Telegram Chat ID")
	notifyEmailTo        = flag.String("email-to", "", "Comma-separated list of email recipients")
	notifyEmailFrom      = flag.String("email-from", "", "Sender email address")
	notifyEmailServer    = flag.String("email-server", "", "SMTP server (host:port)")
	notifyEmailUser      = flag.String("email-user", "", "SMTP username")
	notifyEmailPass      = flag.String("email-pass", "", "SMTP password")
	notifyDiscordURL     = flag.String("discord-webhook", "", "Discord webhook URL")
	notifyOn             = flag.String("notify-on", "alert", "When to notify: alert, warning, expired, all")
)

var (
	okColor      = color.New(color.FgGreen).SprintFunc()
	warningColor = color.New(color.FgYellow).SprintFunc()
	alertColor   = color.New(color.FgRed).SprintFunc()
	expiredColor = color.New(color.FgHiRed, color.Bold).SprintFunc()
	errorColor   = color.New(color.FgMagenta).SprintFunc()
	headerColor  = color.New(color.FgCyan, color.Bold).SprintFunc()
	faintColor   = color.New(color.Faint).SprintFunc()
)

func disableColors() {
	okColor = fmt.Sprint
	warningColor = fmt.Sprint
	alertColor = fmt.Sprint
	expiredColor = fmt.Sprint
	errorColor = fmt.Sprint
	headerColor = fmt.Sprint
	faintColor = fmt.Sprint
}

// Print the banner
func printBanner() {
	banner := `
                        _               _                 _                     _       
  __ _  ___         ___| |__   ___  ___| | __     ___ ___| |       ___ ___ _ __| |_ ___ 
 / _' |/ _ \ _____ / __| '_ \ / _ \/ __| |/ /____/ __/ __| |_____ / __/ _ \ '__| __/ __|
| (_| | (_) |_____| (__| | | |  __/ (__|   <_____\__ \__ \ |_____| (_|  __/ |  | |_\__ \
 \__, |\___/       \___|_| |_|\___|\___|_|\_\    |___/___/_|      \___\___|_|   \__|___/
 |___/                                                                                  
`
	fmt.Println(headerColor(banner))
	color.New(color.FgYellow).Printf("                          	Version: %s\n", toolVersion)
	color.New(color.FgRed).Println("                   Made by Abhinandan-Khurana (@l0u51f3r007)")
	fmt.Print("----------------------------------------------------------------------------------------\n\n")
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("go-check-ssl-certs version %s\n", toolVersion)
		os.Exit(0)
	}

	if !*noBanner {
		printBanner()
	}
	if *noColor || *outputFormat != "terminal" {
		disableColors()
	}

	urlsToCheck, err := getURLs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting URLs: %v\n", err)
		os.Exit(1)
	}

	if len(urlsToCheck) == 0 {
		fmt.Fprintln(os.Stderr, "No URLs provided via -url, -file, or stdin.")
		os.Exit(1)
	}

	results := processURLs(urlsToCheck)

	switch strings.ToLower(*outputFormat) {
	case "terminal", "nocolor":
		printTerminal(results)
	case "json":
		printJSON(results)
	case "csv":
		printCSV(results)
	case "html":
		printHTML(results)
	case "xml":
		printXML(results)
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown output format '%s'\n", *outputFormat)
		os.Exit(1)
	}

	sendNotifications(results)
}

// getURLs determines the list of URLs to check based on flags or stdin
func getURLs() ([]string, error) {
	if *targetURL != "" {
		return []string{*targetURL}, nil
	}

	if *inputFile != "" {
		file, err := os.Open(*inputFile)
		if err != nil {
			return nil, fmt.Errorf("opening file '%s': %w", *inputFile, err)
		}
		defer file.Close()
		return readURLs(file)
	}

	// Check if data is being piped via stdin
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Data is being piped
		return readURLs(os.Stdin)
	}

	return []string{}, nil // No input specified
}

// readURLs reads URLs line by line from an io.Reader
func readURLs(reader io.Reader) ([]string, error) {
	var urls []string
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") { // Ignore empty lines and comments
			urls = append(urls, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	return urls, nil
}

// processURLs checks certificates for multiple URLs concurrently
func processURLs(urls []string) []CertInfo {
	results := make([]CertInfo, 0, len(urls))
	var wg sync.WaitGroup
	urlChan := make(chan string, len(urls))
	resultChan := make(chan CertInfo, len(urls))

	// Limit concurrency
	semaphore := make(chan struct{}, *concurrency)

	// Start workers
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range urlChan {
				semaphore <- struct{}{}
				resultChan <- checkCertificate(u)
				<-semaphore
			}
		}()
	}

	for _, u := range urls {
		urlChan <- u
	}
	close(urlChan)

	wg.Wait()
	close(resultChan)

	// Collect results
	for res := range resultChan {
		results = append(results, res)
	}

	return results
}

// checkCertificate performs the TLS connection and extracts certificate info
func checkCertificate(host string) CertInfo {
	now := time.Now()
	info := CertInfo{URL: host, CheckedTime: now}

	// Ensure host includes port 443 if not specified
	domain := host
	port := "443"
	if strings.Contains(host, ":") {
		hostPart, portPart, err := net.SplitHostPort(host)
		if err == nil {
			domain = hostPart
			port = portPart
		} else {
			// Could be an IPv6 address without port, try adding default port
			host = net.JoinHostPort(host, port)
			domain = host // Use the full host:port if splitting failed initially
		}
	} else {
		host = net.JoinHostPort(host, port)
	}
	info.Domain = domain // Store the extracted/original domain

	// Attempt TLS connection
	dialer := &net.Dialer{
		Timeout: *timeout,
	}
	conn, err := tls.DialWithDialer(dialer, "tcp", host, &tls.Config{
		InsecureSkipVerify: true,   // We need to check the cert ourselves, including expired/invalid ones
		ServerName:         domain, // Set SNI
	})
	if err != nil {
		info.Error = fmt.Sprintf("Connection failed: %v", err)
		info.Status = "Error"
		return info
	}
	defer conn.Close()

	// Check connection state
	connState := conn.ConnectionState()
	if len(connState.PeerCertificates) == 0 {
		info.Error = "No peer certificates received"
		info.Status = "Error"
		return info
	}

	// Get the leaf certificate (the server's certificate)
	leafCert := connState.PeerCertificates[0]

	info.Issuer = leafCert.Issuer.String()
	info.Subject = leafCert.Subject.String()
	info.NotBefore = leafCert.NotBefore
	info.NotAfter = leafCert.NotAfter

	// Calculate days left
	if now.After(leafCert.NotAfter) {
		info.DaysLeft = 0
		info.Status = "Expired"
	} else {
		durationUntilExpiry := leafCert.NotAfter.Sub(now)
		info.DaysLeft = int(durationUntilExpiry.Hours() / 24)

		// Determine status based on thresholds
		if info.DaysLeft <= *alertDays {
			info.Status = "Alert"
		} else if info.DaysLeft <= *warningDays {
			info.Status = "Warning"
		} else {
			info.Status = "OK"
		}
	}

	return info
}

// --- Output Functions ---

func printTerminal(results []CertInfo) {
	if len(results) == 0 {
		fmt.Println("No results to display.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, headerColor("DOMAIN\tISSUER\tSUBJECT\tEXPIRES ON\tDAYS LEFT\tSTATUS\tCHECKED AT\tERROR"))
	fmt.Fprintln(w, headerColor("------\t------\t-------\t----------\t---------\t------\t----------\t-----"))

	for _, r := range results {
		statusStr := r.Status
		switch r.Status {
		case "OK":
			statusStr = okColor(r.Status)
		case "Warning":
			statusStr = warningColor(r.Status)
		case "Alert":
			statusStr = alertColor(r.Status)
		case "Expired":
			statusStr = expiredColor(r.Status)
		case "Error":
			statusStr = errorColor(r.Status)
		}

		expiresStr := r.NotAfter.Format("Jan 02 2006 15:04 MST")
		if r.NotAfter.IsZero() {
			expiresStr = "N/A"
		}

		checkedStr := r.CheckedTime.Format("2006-01-02 15:04:05")

		issuerStr := formatDN(r.Issuer)
		subjectStr := formatDN(r.Subject)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
			r.Domain,
			faintColor(issuerStr),
			faintColor(subjectStr),
			expiresStr,
			r.DaysLeft,
			statusStr,
			faintColor(checkedStr),
			errorColor(r.Error),
		)
	}
	w.Flush()
}

// formatDN simplifies the Distinguished Name string for better readability
func formatDN(dn string) string {
	parts := strings.Split(dn, ",")
	commonName := ""
	org := ""
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "CN=") {
			commonName = strings.TrimPrefix(part, "CN=")
		} else if strings.HasPrefix(part, "O=") {
			org = strings.TrimPrefix(part, "O=")
		}
	}
	if commonName != "" && org != "" {
		return fmt.Sprintf("%s (%s)", commonName, org)
	} else if commonName != "" {
		return commonName
	} else if org != "" {
		return org
	}
	// Fallback to a shorter version if CN/O aren't found easily
	if len(parts) > 0 {
		return parts[0]
	}
	return dn // Return original if all else fails
}

func printJSON(results []CertInfo) {
	outputData := Results{Certs: results}
	jsonData, err := json.MarshalIndent(outputData, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshalling JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonData))
}

func printCSV(results []CertInfo) {
	// Print header
	fmt.Println("URL,Domain,Issuer,Subject,NotBefore,NotAfter,DaysLeft,Status,Error,CheckedTime")
	// Print data rows
	for _, r := range results {
		// Basic CSV quoting for fields that might contain commas
		fmt.Printf("\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",%d,\"%s\",\"%s\",\"%s\"\n",
			r.URL,
			r.Domain,
			strings.ReplaceAll(r.Issuer, "\"", "\"\""),
			strings.ReplaceAll(r.Subject, "\"", "\"\""),
			r.NotBefore.Format(time.RFC3339),
			r.NotAfter.Format(time.RFC3339),
			r.DaysLeft,
			r.Status,
			strings.ReplaceAll(r.Error, "\"", "\"\""),
			r.CheckedTime.Format(time.RFC3339),
		)
	}
}

func printXML(results []CertInfo) {
	outputData := Results{Certs: results}
	xmlData, err := xml.MarshalIndent(outputData, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshalling XML: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(xmlData))
}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SSL Certificate Check Results</title>
    <style>
        body { font-family: sans-serif; line-height: 1.6; padding: 20px; }
        table { border-collapse: collapse; width: 100%; margin-top: 20px; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        tr:nth-child(even) { background-color: #f9f9f9; }
        .status-OK { color: green; }
        .status-Warning { color: orange; }
        .status-Alert { color: red; }
        .status-Expired { color: red; font-weight: bold; }
        .status-Error { color: magenta; }
		.faint { color: #777; font-size: 0.9em;}
    </style>
</head>
<body>
    <h1>SSL Certificate Check Results</h1>
    <p>Checked on: {{ .Timestamp }}</p>
    <table>
        <thead>
            <tr>
                <th>Domain</th>
                <th>Issuer</th>
                <th>Subject</th>
                <th>Expires On</th>
                <th>Days Left</th>
                <th>Status</th>
                <th>Checked At</th>
                <th>Error</th>
            </tr>
        </thead>
        <tbody>
            {{ range .Results }}
            <tr>
                <td>{{ .Domain }}</td>
                <td class="faint">{{ .Issuer | formatDN }}</td>
                <td class="faint">{{ .Subject | formatDN }}</td>
                <td>{{ if .NotAfter.IsZero }}N/A{{ else }}{{ .NotAfter.Format "Jan 02 2006 15:04 MST" }}{{ end }}</td>
                <td>{{ .DaysLeft }}</td>
                <td class="status-{{ .Status }}">{{ .Status }}</td>
                <td class="faint">{{ .CheckedTime.Format "2006-01-02 15:04:05" }}</td>
                <td class="status-Error">{{ .Error }}</td>
            </tr>
            {{ end }}
        </tbody>
    </table>
</body>
</html>
`

func printHTML(results []CertInfo) {
	funcMap := template.FuncMap{
		"formatDN": formatDN, // Make Go function available in template
	}

	tmpl, err := template.New("html").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing HTML template: %v\n", err)
		os.Exit(1)
	}

	data := struct {
		Timestamp string
		Results   []CertInfo
	}{
		Timestamp: time.Now().Format(time.RFC1123),
		Results:   results,
	}

	err = tmpl.Execute(os.Stdout, data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing HTML template: %v\n", err)
		os.Exit(1)
	}
}

// --- Notification Functions ---

// sendNotifications sends notifications based on the configured channels and the notification trigger level
func sendNotifications(results []CertInfo) {
	if *verbose {
		fmt.Println("Checking for certificates to notify about...")
	}

	// Filter results based on notifyOn setting
	var notifyResults []CertInfo
	for _, cert := range results {
		shouldNotify := false
		switch strings.ToLower(*notifyOn) {
		case "all":
			shouldNotify = true
		case "alert":
			shouldNotify = cert.Status == "Alert" || cert.Status == "Expired" || cert.Status == "Error"
		case "warning":
			shouldNotify = cert.Status == "Warning" || cert.Status == "Alert" || cert.Status == "Expired" || cert.Status == "Error"
		case "expired":
			shouldNotify = cert.Status == "Expired" || cert.Status == "Error"
		case "error":
			shouldNotify = cert.Status == "Error"
		}

		if shouldNotify {
			notifyResults = append(notifyResults, cert)
		}
	}

	// If no certificates require notification, return
	if len(notifyResults) == 0 {
		if *verbose {
			fmt.Println("No certificates require notification based on current settings.")
		}
		return
	}

	// Create notification message
	message := formatNotificationMessage(notifyResults)

	// Send notifications through configured channels
	var wg sync.WaitGroup

	// Slack notification
	if *notifySlackURL != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := sendSlackNotification(*notifySlackURL, message)
			if err != nil && *verbose {
				fmt.Printf("Slack notification error: %v\n", err)
			} else if *verbose {
				fmt.Println("Slack notification sent successfully")
			}
		}()
	}

	// Telegram notification
	if *notifyTelegramToken != "" && *notifyTelegramChatID != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := sendTelegramNotification(*notifyTelegramToken, *notifyTelegramChatID, message)
			if err != nil && *verbose {
				fmt.Printf("Telegram notification error: %v\n", err)
			} else if *verbose {
				fmt.Println("Telegram notification sent successfully")
			}
		}()
	}

	// Email notification
	if *notifyEmailTo != "" && *notifyEmailFrom != "" && *notifyEmailServer != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := sendEmailNotification(*notifyEmailServer, *notifyEmailUser, *notifyEmailPass, *notifyEmailFrom, *notifyEmailTo, message)
			if err != nil && *verbose {
				fmt.Printf("Email notification error: %v\n", err)
			} else if *verbose {
				fmt.Println("Email notification sent successfully")
			}
		}()
	}

	// Discord notification
	if *notifyDiscordURL != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := sendDiscordNotification(*notifyDiscordURL, message)
			if err != nil && *verbose {
				fmt.Printf("Discord notification error: %v\n", err)
			} else if *verbose {
				fmt.Println("Discord notification sent successfully")
			}
		}()
	}

	// Wait for all notifications to complete
	wg.Wait()
}

// formatNotificationMessage creates a formatted message for notifications
func formatNotificationMessage(results []CertInfo) string {
	var b bytes.Buffer
	b.WriteString("SSL Certificate Alert\n\n")
	b.WriteString(fmt.Sprintf("The following %d certificates require attention:\n\n", len(results)))

	for i, cert := range results {
		b.WriteString(fmt.Sprintf("%d. Domain: %s\n", i+1, cert.Domain))
		b.WriteString(fmt.Sprintf("   Status: %s\n", cert.Status))

		if cert.Status != "Error" {
			b.WriteString(fmt.Sprintf("   Expires: %s\n", cert.NotAfter.Format("Jan 02 2006 15:04 MST")))
			b.WriteString(fmt.Sprintf("   Days left: %d\n", cert.DaysLeft))
		}

		if cert.Error != "" {
			b.WriteString(fmt.Sprintf("   Error: %s\n", cert.Error))
		}

		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("Checked at: %s\n", time.Now().Format(time.RFC1123)))
	return b.String()
}

// sendSlackNotification sends a message to a Slack webhook
func sendSlackNotification(webhookURL, message string) error {
	// Format message for Slack
	payload := map[string]interface{}{
		"text": "```" + message + "```", // Monospace formatting
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Send HTTP request
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack API error: %s - %s", resp.Status, string(body))
	}

	return nil
}

// sendTelegramNotification sends a message via Telegram Bot API
func sendTelegramNotification(token, chatID, message string) error {
	// Format message for Telegram (URL encode)
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	payload := map[string]string{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "Markdown",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Send HTTP request
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error: %s - %s", resp.Status, string(body))
	}

	return nil
}

// sendEmailNotification sends an email via SMTP
func sendEmailNotification(server, username, password, from, to, message string) error {
	// Create email headers and body
	subject := "SSL Certificate Alert"
	recipients := strings.Split(to, ",")

	// Construct email
	header := make(map[string]string)
	header["From"] = from
	header["To"] = to
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/plain; charset=utf-8"

	var msg bytes.Buffer
	for k, v := range header {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(message)

	// Check if server contains port, if not, add default SMTP port
	if !strings.Contains(server, ":") {
		server = server + ":25" // Default SMTP port
	}

	// Authentication and sending
	var auth smtp.Auth
	if username != "" && password != "" {
		// Split host from port
		host, _, err := net.SplitHostPort(server)
		if err != nil {
			return fmt.Errorf("invalid server address: %v", err)
		}
		auth = smtp.PlainAuth("", username, password, host)
	}

	err := smtp.SendMail(server, auth, from, recipients, msg.Bytes())
	if err != nil {
		return fmt.Errorf("sending email: %v", err)
	}

	return nil
}

// sendDiscordNotification sends a message to a Discord webhook
func sendDiscordNotification(webhookURL, message string) error {
	// Format message for Discord
	// Break message into chunks if needed (Discord has a 2000 char limit)
	const maxLength = 1900 // Leave some buffer

	var chunks []string
	for len(message) > 0 {
		chunkSize := len(message)
		if chunkSize > maxLength {
			chunkSize = maxLength
			// Try to break at a newline
			lastNewline := strings.LastIndex(message[:chunkSize], "\n")
			if lastNewline > maxLength/2 {
				chunkSize = lastNewline + 1
			}
		}

		chunks = append(chunks, message[:chunkSize])
		message = message[chunkSize:]
	}

	// Send each chunk as a separate message
	for _, chunk := range chunks {
		payload := map[string]interface{}{
			"content": "```" + chunk + "```", // Monospace formatting
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		// Send HTTP request
		resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("discord API error: %s - %s", resp.Status, string(body))
		}
	}

	return nil
}
