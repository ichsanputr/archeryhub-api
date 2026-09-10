package main

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := "ichsan:12345@tcp(151.243.222.93:30036)/archeris?parseTime=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println("DB open error:", err)
		os.Exit(1)
	}
	defer db.Close()

	// 1. Query OTP terbaru
	fmt.Println("=== 1. Query OTP dari DB ===")
	rows, err := db.Query(`
		SELECT uuid, email, user_id, user_type, otp_code, is_used, expires_at, created_at 
		FROM password_resets 
		WHERE email = 'seller@panahan.com' 
		ORDER BY created_at DESC LIMIT 3
	`)
	if err != nil {
		fmt.Println("Query error:", err)
		os.Exit(1)
	}
	for rows.Next() {
		var uid, email, userID, uType, otp string
		var isUsed bool
		var expiresAt, createdAt time.Time
		rows.Scan(&uid, &email, &userID, &uType, &otp, &isUsed, &expiresAt, &createdAt)
		fmt.Printf("  UUID=%s Email=%s Type=%s OTP=%s Used=%v Expires=%v Created=%v\n",
			uid[:8], email, uType, otp, isUsed,
			expiresAt.Format("15:04:05"), createdAt.Format("15:04:05"))
	}

	// Check stewie4king@gmail.com OTP
	fmt.Println("\n=== Check stewie4king@gmail.com ===")
	var stewieOTP string
	err = db.QueryRow(`SELECT otp_code FROM password_resets 
		WHERE email='stewie4king@gmail.com' AND is_used=0 AND expires_at>NOW()
		ORDER BY created_at DESC LIMIT 1`).Scan(&stewieOTP)
	if err != nil {
		fmt.Println("  No active OTP:", err)
	} else {
		fmt.Printf("  >>> Active OTP for stewie4king: %s\n", stewieOTP)
	}

	rows.Close()

	var latestOTP string
	err = db.QueryRow(`SELECT otp_code FROM password_resets 
		WHERE email='seller@panahan.com' AND is_used=0 AND expires_at>NOW()
		ORDER BY created_at DESC LIMIT 1`).Scan(&latestOTP)
	if err != nil {
		fmt.Println("  No active OTP:", err)
	} else {
		fmt.Printf("  >>> Active OTP: %s\n", latestOTP)
	}

	// 2. Test SMTP
	fmt.Println("\n=== 2. Test SMTP ===")
	host := "mail.karyasija.id"
	addr := host + ":587"
	smtpUser := "admin@archeris.net"
	smtpPass := "lh4Wsh8u92D4"
	smtpFrom := "Admin Archeris <admin@archeris.net>"
	to := "seller@panahan.com"

	auth := smtp.PlainAuth("", smtpUser, smtpPass, host)
	body := "<h3>Test Archeris</h3><p>OTP: 123456</p>"

	headers := make(map[string]string)
	headers["From"] = smtpFrom
	headers["To"] = to
	headers["Subject"] = "Test SMTP - Archeris"
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"utf-8\""
	msg := ""
	for k, v := range headers {
		msg += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	msg += "\r\n" + body

	start := time.Now()
	err = smtp.SendMail(addr, auth, smtpFrom, []string{to}, []byte(msg))
	elapsed := time.Since(start)
	if err != nil {
		fmt.Printf("  SendMail FAILED (%v): %v\n", elapsed, err)
		fmt.Println("  Trying manual STARTTLS...")
		testSTARTTLS(host, smtpUser, smtpPass, smtpFrom, to, body)
	} else {
		fmt.Printf("  SendMail SUCCESS (%v)\n", elapsed)
	}
}

// loginAuth implements LOGIN authentication mechanism
type loginAuth struct {
	username, password string
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	prompt := strings.ToLower(string(fromServer))
	switch {
	case strings.Contains(prompt, "username"):
		return []byte(a.username), nil
	case strings.Contains(prompt, "password"):
		return []byte(a.password), nil
	default:
		return nil, fmt.Errorf("unexpected LOGIN prompt: %s", string(fromServer))
	}
}

func testSTARTTLS(host, user, pass, from, to, body string) {
	conn, err := net.DialTimeout("tcp", host+":587", 10*time.Second)
	if err != nil {
		fmt.Println("  Dial:", err)
		return
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		fmt.Println("  Client:", err)
		return
	}
	defer client.Close()
	client.Hello("localhost")
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsCfg := &tls.Config{ServerName: host, InsecureSkipVerify: true}
		if err := client.StartTLS(tlsCfg); err != nil {
			fmt.Println("  STARTTLS:", err)
			return
		}
		fmt.Println("  STARTTLS OK")
	}

	// Try LOGIN auth instead of PLAIN
	auth := &loginAuth{username: user, password: pass}
	if err := client.Auth(auth); err != nil {
		fmt.Println("  LOGIN Auth error:", err)
		return
	}
	fmt.Println("  LOGIN Auth OK!")

	// Extract bare email from "Display Name <email>" format for SMTP envelope
	bareFrom := from
	if idx := strings.Index(from, "<"); idx >= 0 {
		end := strings.Index(from, ">")
		if end > idx {
			bareFrom = from[idx+1 : end]
		}
	}
	fmt.Println("  MAIL FROM:", bareFrom)
	if err := client.Mail(bareFrom); err != nil {
		fmt.Println("  MAIL FROM error:", err)
		return
	}
	if err := client.Rcpt(to); err != nil {
		fmt.Println("  RCPT TO:", err)
		return
	}
	w, err := client.Data()
	if err != nil {
		fmt.Println("  Data:", err)
		return
	}
	hdr := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: Test STARTTLS + LOGIN Auth\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"utf-8\"\r\n\r\n", from, to)
	w.Write([]byte(hdr + body))
	w.Close()
	fmt.Println("  >>> Email SENT successfully via STARTTLS + LOGIN!")
	client.Quit()
}