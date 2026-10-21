package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"meteor/internal/config"
)

const resendEmailsURL = "https://api.resend.com/emails"

// ResendMailer sends verification and password-reset links through Resend's email API.
type ResendMailer struct {
	apiKey string
	from   string
	client *http.Client
}

// NewResendMailer creates an email adapter from the Resend settings in application config.
func NewResendMailer(cfg config.Config) *ResendMailer {
	return &ResendMailer{
		apiKey: cfg.ResendAPIKey,
		from:   cfg.ResendFrom,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// SendVerification delivers a one-time email verification link.
func (m *ResendMailer) SendVerification(ctx context.Context, recipient, link string) error {
	return m.send(ctx, recipient, "Verify your Meteor account", "Use this one-time link to verify your Meteor account. It expires in 24 hours.", link)
}

// SendPasswordReset delivers a one-time password recovery link.
func (m *ResendMailer) SendPasswordReset(ctx context.Context, recipient, link string) error {
	return m.send(ctx, recipient, "Reset your Meteor password", "Use this one-time link to reset your Meteor password. It expires in 1 hour. If you did not request this, you can ignore this message.", link)
}

// send posts a plain-text transactional email to Resend and reports delivery failures.
func (m *ResendMailer) send(ctx context.Context, recipient, subject, body, link string) error {
	if strings.TrimSpace(m.apiKey) == "" || strings.TrimSpace(m.from) == "" {
		return errors.New("Resend email delivery is not configured")
	}
	to, err := mail.ParseAddress(recipient)
	if err != nil || to.Address != recipient {
		return errors.New("email recipient address is invalid")
	}
	if _, err := mail.ParseAddress(m.from); err != nil {
		return errors.New("Resend sender address is invalid")
	}

	payload := struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		Text    string   `json:"text"`
	}{
		From: m.from, To: []string{to.Address}, Subject: subject,
		Text: body + "\n\n" + link + "\n",
	}
	bodyJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode Resend email: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, resendEmailsURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return fmt.Errorf("create Resend request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+m.apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := m.client.Do(request)
	if err != nil {
		return fmt.Errorf("send email through Resend: %w", err)
	}
	defer response.Body.Close()
	if _, err := io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10)); err != nil {
		return fmt.Errorf("read Resend response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("Resend rejected email with HTTP status %d", response.StatusCode)
	}
	return nil
}
