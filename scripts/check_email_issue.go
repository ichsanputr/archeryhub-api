package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"crypto/tls"
	"net"
	"time"
	"io"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// ===== 1. Check DB for the specific emails =====
	fmt.Println("========================================")
	fmt.Println("STEP 1: Check Database")
	fmt.Println("========================================")
	
	db, err := sql.Open("mysql", "ichsan:12345@tcp(151.243.222.93:30036)/archeris?parseTime=true")
	if err != nil {
		fmt.Println("DB Error:", err)
		return
	}
	defer db.Close()

	emails := []string{"anggerraka2017@gmail.com", "stewie4king@gmail.com"}
	for _, email := range emails {
		fmt.Printf("\n--- %s ---\n", email)
		for _, table := range []string{"archers", "organizers", "sellers"} {
			var count int
			db.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE email = ?", email).Scan(&count)
			if count > 0 {
				fmt.Printf("  ✅ Found in %s\n", table)
			} else {
				fmt.Printf("  ❌ Not in %s\n", table)
			}
		}
	}

	// ===== 2. Start API and test forgot-password =====
	fmt.Println("\n========================================")
	fmt.Println("STEP 2: Test API Forgot Password")
	fmt.Println("========================================")

	// Test with both emails
	for _, email := range emails {
		fmt.Printf("\n--- Testing: %s ---\n", email)
		
		resp, err := http.Post(
			"http://localhost:8001/mobile/auth/forgot-password",
			"application/json",
			strings.NewReader(fmt.Sprintf(`{"email":"%s"}`, email)),
		)
		if err != nil {
			fmt.Printf("  ❌ API Error: %v\n", err)
			fmt.Println("  → API might not be running on port 8001")
			continue
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("  HTTP Status: %d\n", resp.StatusCode)
		fmt.Printf("  Response: %s\n", string(body))
		
		if resp.StatusCode == 200 {
			fmt.Println("  ✅ API returned 200 (email processed)")
		} else if resp.StatusCode == 500 {
			fmt.Println("  ❌ API returned 500 (server error - possibly email send failed)")
		}
	}

	// ===== 3. Test SMTP directly =====
	fmt.Println("\n========================================")
	fmt.Println("STEP 3: Test SMTP Direct")
	fmt.Println("========================================")

	smtpHost := "mail.karyasija.id"
	smtpPort := "587"
	smtpUser := "admin@archeris.net"
	smtpPass := "lh4Wsh8u92D4"
	addr := smtpHost + ":" + smtpPort

	for _, toEmail := range emails {
		fmt.Printf("\n--- Sending to: %s ---\n", toEmail)
		
		err := testSendEmail(addr, smtpHost, smtpUser, smtpPass, toEmail)
		if err != nil {
			fmt.Printf("  ❌ SMTP Error: %v\n", err)
		} else {
			fmt.Printf("  ✅ SMTP Success!\n")
		}
	}
}

func testSendEmail(addr, host, user, pass, to string) error {
	// Connect with timeout
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
		return fmt.Errorf("hello: %w", err)
	}

	// STARTTLS
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{ServerName: host}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}

	// LOGIN auth
	auth := &loginAuth{username: user, password: pass}
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	if err := client.Mail(user); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: SMTP Test - Archeris OTP\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"utf-8\"\r\n\r\n<h1>Test Email</h1><p>This is a test OTP email from Archeris.</p>", user, to)
	if _, err := wc.Write([]byte(msg)); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	if err := wc.Close(); err != nil {
		return fmt.Errorf("close writer: %w", err)
	}

	return client.Quit()
}

// loginAuth implements smtp.Auth for SMTP LOGIN mechanism
type loginAuth struct {
	username, password string
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch strings.ToUpper(string(fromServer)) {
		case "USERNAME:", "USER:", "EMAIL:":
			return []byte(a.username), nil
		case "PASSWORD:", "PASS:":
			return []byte(a.password), nil
		default:
			return []byte(a.password), nil
		}
	}
	return nil, nil
}

