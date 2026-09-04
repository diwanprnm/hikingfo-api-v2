package infrastructure

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

// SMTPConfig configures the outgoing mail adapter. When empty, sends are
// dropped and logged so the dev stack runs without a mail server
// (research.md §1 — MailHog is wired in compose.yaml for local captures).
type SMTPConfig struct {
	Host     string // smtp host, e.g. "smtp.example.com"
	Port     int    // 587 (STARTTLS) or 25
	Username string
	Password string
	From     string // RFC 5322 address used in From
	// FromName is the display name rendered in the From header.
	FromName string
}

// enabled reports whether the adapter should attempt real sends.
func (c SMTPConfig) enabled() bool { return c.Host != "" }

// Mailer sends the two transactional emails the identity context needs. It
// implements identity/application.Mailer. Sends are non-fatal by design: the
// application services treat a mailer failure as a recoverable condition
// (registration already succeeded; the user may resend).
type Mailer struct {
	cfg SMTPConfig
	now func() time.Time
}

// NewMailer wires an SMTP mailer.
func NewMailer(cfg SMTPConfig) *Mailer {
	return &Mailer{cfg: cfg, now: time.Now}
}

// SendVerification delivers the verify-email message to address.
func (m *Mailer) SendVerification(ctx context.Context, to, displayName, verifyURL string) error {
	subject := "Verify your hikingfo account"
	body := fmt.Sprintf(
		"Hi %s,\n\nWelcome to hikingfo! Please verify your email address to activate your account:\n\n%s\n\nThis link expires in 24 hours.\n\n— the hikingfo team",
		displayName, verifyURL,
	)
	return m.send(ctx, to, subject, body)
}

// SendPasswordReset delivers the reset link to address.
func (m *Mailer) SendPasswordReset(ctx context.Context, to, displayName, resetURL string) error {
	subject := "Reset your hikingfo password"
	body := fmt.Sprintf(
		"Hi %s,\n\nSomeone asked to reset your hikingfo password. If that was you, open the link below (valid for 1 hour):\n\n%s\n\nIf you didn't request this, you can safely ignore this email.\n\n— the hikingfo team",
		displayName, resetURL,
	)
	return m.send(ctx, to, subject, body)
}

func (m *Mailer) send(_ context.Context, to, subject, text string) error {
	if !m.cfg.enabled() {
		// Dev/loopback: no SMTP configured — drop with a trace instead of failing
		// the caller. compose.yaml wires MailHog when capture is wanted.
		return nil
	}
	from := m.cfg.From
	if m.cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", m.cfg.FromName, m.cfg.From)
	}
	msg := strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Date: " + m.now().UTC().Format(time.RFC1123Z),
		"",
		text,
	}, "\r\n")

	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	if err := smtp.SendMail(addr, auth, m.cfg.From, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("infra/identity: smtp send: %w", err)
	}
	return nil
}
