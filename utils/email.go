package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/smtp"
	"os"
)

// SendEmail sends an HTML email via SMTP
func SendEmail(to, subject, body string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	smtpFrom := os.Getenv("SMTP_FROM")

	if smtpHost == "" || smtpPort == "" {
		// Mock for development if not set
		if os.Getenv("ENV") != "production" {
			fmt.Printf("\n--- MOCK EMAIL BEGIN ---\nTo: %s\nSubject: %s\nBody: %s\n--- MOCK EMAIL END ---\n\n", to, subject, body)
			return nil
		}
		return fmt.Errorf("SMTP configuration not found")
	}

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	header := make(map[string]string)
	header["From"] = smtpFrom
	header["To"] = to
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/html; charset=\"utf-8\""

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, smtpFrom, []string{to}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}

// GenerateOTP generates a random 6-digit OTP
func GenerateOTP() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("%06d", n.Int64()+100000)
}
