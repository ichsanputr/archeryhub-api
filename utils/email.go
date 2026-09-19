package utils

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"math/big"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// loginAuth implements the SMTP LOGIN authentication mechanism.
// Some mail servers (cPanel, older Postfix/Exim) do not support PLAIN auth
// and require LOGIN instead. Go's standard library only ships PlainAuth
// and CRAMMD5Auth, so we implement LOGIN here.
type loginAuth struct {
	username, password string
}

func (a *loginAuth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	prompt := strings.ToLower(string(fromServer))
	switch {
	case strings.Contains(prompt, "username") || strings.Contains(prompt, "user"):
		return []byte(a.username), nil
	case strings.Contains(prompt, "password"):
		return []byte(a.password), nil
	default:
		return nil, fmt.Errorf("unexpected LOGIN challenge: %s", string(fromServer))
	}
}

// extractBareEmail extracts the bare email address from a "Display Name <email>" string.
// e.g. "Admin Archeris <admin@archeris.net>" → "admin@archeris.net"
func extractBareEmail(from string) string {
	if idx := strings.Index(from, "<"); idx >= 0 {
		if end := strings.Index(from, ">"); end > idx {
			return from[idx+1 : end]
		}
	}
	return from
}

// SendEmail sends an HTML email via SMTP with STARTTLS + LOGIN auth.
// Falls back to plain smtp.SendMail (PLAIN auth) if LOGIN fails, so that
// servers supporting PLAIN still work.
// Implements retry logic with exponential backoff for transient failures.
func SendEmail(to, subject, body string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	smtpFrom := os.Getenv("SMTP_FROM")
	if smtpFrom == "" {
		smtpFrom = "Archeris <admin@archeris.net>"
	}

	if smtpHost == "" || smtpPort == "" {
		// Mock for development if not set
		if os.Getenv("ENV") != "production" {
			fmt.Printf("\n--- MOCK EMAIL BEGIN ---\nTo: %s\nSubject: %s\nBody: %s\n--- MOCK EMAIL END ---\n\n", to, subject, body)
			return nil
		}
		return fmt.Errorf("SMTP configuration not found")
	}

	// Build RFC-5322 compliant email headers for high deliverability & spam filter compliance
	msgID := fmt.Sprintf("<%d.%s@archeris.net>", time.Now().UnixNano(), GenerateOTP())
	header := make(map[string]string)
	header["From"] = smtpFrom
	header["To"] = to
	header["Reply-To"] = "Archeris Support <admin@archeris.net>"
	header["Subject"] = subject
	header["Date"] = time.Now().Format(time.RFC1123Z)
	header["Message-ID"] = msgID
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/html; charset=\"UTF-8\""
	header["Content-Transfer-Encoding"] = "8bit"
	header["X-Mailer"] = "Archeris Mailer"
	header["Auto-Submitted"] = "auto-generated"

	var message strings.Builder
	for k, v := range header {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")
	message.WriteString(body)

	bareFrom := extractBareEmail(smtpFrom)
	addr := smtpHost + ":" + smtpPort

	// Retry logic: up to 3 attempts with exponential backoff
	maxAttempts := 3
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.WithFields(log.Fields{
			"to":      to,
			"subject": subject,
			"attempt": attempt,
			"max":     maxAttempts,
		}).Info("Attempting to send email")

		// Primary path: manual STARTTLS + LOGIN auth
		err := sendWithSTARTTLSAndLogin(addr, smtpHost, smtpUser, smtpPass, bareFrom, to, message.String())
		if err == nil {
			log.WithFields(log.Fields{
				"to":      to,
				"subject": subject,
				"attempt": attempt,
			}).Info("Email sent successfully")
			return nil
		}

		lastErr = err
		log.WithError(err).WithFields(log.Fields{
			"to":      to,
			"subject": subject,
			"attempt": attempt,
			"max":     maxAttempts,
		}).Warn("SMTP STARTTLS+LOGIN failed")

		// Fallback: standard smtp.SendMail with PlainAuth (for servers that support PLAIN)
		log.WithField("attempt", attempt).Info("Falling back to PLAIN auth")
		auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
		if err2 := smtp.SendMail(addr, auth, bareFrom, []string{to}, []byte(message.String())); err2 != nil {
			log.WithError(err2).WithFields(log.Fields{
				"to":      to,
				"subject": subject,
				"attempt": attempt,
			}).Warn("SMTP PLAIN auth also failed")
			lastErr = fmt.Errorf("LOGIN: %v, PLAIN: %v", err, err2)
		} else {
			log.WithFields(log.Fields{
				"to":      to,
				"subject": subject,
				"attempt": attempt,
			}).Info("Email sent successfully via PLAIN auth")
			return nil
		}

		// If not the last attempt, wait with exponential backoff
		if attempt < maxAttempts {
			backoff := time.Duration(attempt*attempt) * time.Second
			log.WithFields(log.Fields{
				"attempt": attempt,
				"backoff": backoff,
			}).Info("Retrying after backoff")
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("failed to send email after %d attempts: %w", maxAttempts, lastErr)
}

// sendWithSTARTTLSAndLogin connects to the SMTP server, upgrades to TLS via
// STARTTLS, authenticates with LOGIN, and sends the email.
func sendWithSTARTTLSAndLogin(addr, host, user, pass, from, to, msg string) error {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("ehlo: %w", err)
	}

	// Upgrade to TLS if supported
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: false,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			// Retry with InsecureSkipVerify for self-signed certs
			log.WithError(err).Warn("STARTTLS strict TLS failed, retrying with InsecureSkipVerify")
			// Need a fresh connection
			conn.Close()
			conn2, err2 := net.DialTimeout("tcp", addr, 10*time.Second)
			if err2 != nil {
				return fmt.Errorf("dial retry: %w", err2)
			}
			defer conn2.Close()
			client2, err2 := smtp.NewClient(conn2, host)
			if err2 != nil {
				return fmt.Errorf("smtp client retry: %w", err2)
			}
			defer client2.Close()
			client2.Hello("localhost")
			tlsConfig.InsecureSkipVerify = true
			if err2 := client2.StartTLS(tlsConfig); err2 != nil {
				return fmt.Errorf("starttls retry: %w", err2)
			}
			client = client2
		}
	}

	// Authenticate using LOGIN mechanism
	auth := &loginAuth{username: user, password: pass}
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("auth LOGIN: %w", err)
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}

	client.Quit()
	return nil
}

// GenerateOTP generates a random 6-digit OTP
func GenerateOTP() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("%06d", n.Int64()+100000)
}
