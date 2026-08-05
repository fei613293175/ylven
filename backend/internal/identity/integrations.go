package identity

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type TurnstileVerifier interface {
	Verify(context.Context, string, string, string) error
}

type CloudflareTurnstileVerifier struct {
	Secret           string
	ExpectedHostname string
	Endpoint         string
	Client           *http.Client
}

type turnstileResponse struct {
	Success    bool     `json:"success"`
	Hostname   string   `json:"hostname"`
	Action     string   `json:"action"`
	ErrorCodes []string `json:"error-codes"`
}

func (v CloudflareTurnstileVerifier) Verify(ctx context.Context, token, remoteIP, action string) error {
	if strings.TrimSpace(v.Secret) == "" {
		return errors.New("turnstile_unconfigured")
	}
	endpoint := v.Endpoint
	if endpoint == "" {
		endpoint = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	}
	values := url.Values{"secret": {v.Secret}, "response": {token}}
	if remoteIP != "" {
		values.Set("remoteip", remoteIP)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return fmt.Errorf("turnstile_request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := v.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("turnstile_unavailable: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return fmt.Errorf("turnstile_http_%d", response.StatusCode)
	}
	var result turnstileResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return fmt.Errorf("turnstile_response: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("turnstile_failed: %s", strings.Join(result.ErrorCodes, ","))
	}
	if v.ExpectedHostname != "" && !strings.EqualFold(result.Hostname, v.ExpectedHostname) {
		return errors.New("turnstile_hostname_mismatch")
	}
	if action != "" && result.Action != "" && result.Action != action {
		return errors.New("turnstile_action_mismatch")
	}
	return nil
}

type MockTurnstileVerifier struct{ Token string }

func (v MockTurnstileVerifier) Verify(_ context.Context, token, _ string, _ string) error {
	if v.Token == "" || token != v.Token {
		return errors.New("turnstile_failed")
	}
	return nil
}

type OTPMailer interface {
	SendOTP(context.Context, string, string, string, string) error
}

type SMTPMailer struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
}

func (m SMTPMailer) SendOTP(ctx context.Context, recipient, subject, body, _ string) error {
	if m.Host == "" || m.Port == 0 || m.From == "" {
		return errors.New("smtp_unconfigured")
	}
	for _, value := range []string{recipient, subject, m.From, m.FromName} {
		if strings.ContainsAny(value, "\r\n") {
			return errors.New("smtp_header_invalid")
		}
	}
	address := net.JoinHostPort(m.Host, strconv.Itoa(m.Port))
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("smtp_connect: %w", err)
	}
	defer connection.Close()
	client, err := smtp.NewClient(connection, m.Host)
	if err != nil {
		return fmt.Errorf("smtp_client: %w", err)
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: m.Host}); err != nil {
			return fmt.Errorf("smtp_starttls: %w", err)
		}
	}
	if m.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", m.Username, m.Password, m.Host)); err != nil {
			return fmt.Errorf("smtp_auth: %w", err)
		}
	}
	if err := client.Mail(m.From); err != nil {
		return fmt.Errorf("smtp_from: %w", err)
	}
	if err := client.Rcpt(recipient); err != nil {
		return fmt.Errorf("smtp_recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp_data: %w", err)
	}
	from := m.From
	if m.FromName != "" {
		from = fmt.Sprintf("%s <%s>", m.FromName, m.From)
	}
	message := strings.Join([]string{
		"From: " + from,
		"To: " + recipient,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
	}, "\r\n")
	if _, err := io.WriteString(writer, message); err != nil {
		_ = writer.Close()
		return fmt.Errorf("smtp_write: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp_close: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp_quit: %w", err)
	}
	return nil
}

func envOrFile(name string) (string, error) {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value, nil
	}
	path := strings.TrimSpace(os.Getenv(name + "_FILE"))
	if path == "" {
		return "", nil
	}
	value, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s_FILE: %w", name, err)
	}
	return strings.TrimSpace(string(value)), nil
}
