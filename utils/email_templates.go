package utils

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strings"
)

func formatIDR(amount float64) string {
	s := fmt.Sprintf("%.0f", amount)
	var result []string
	for i := len(s) - 1; i >= 0; i-- {
		result = append([]string{string(s[i])}, result...)
		if (len(s)-i)%3 == 0 && i != 0 {
			result = append([]string{"."}, result...)
		}
	}
	return strings.Join(result, "")
}

func getAppURL() string {
	url := os.Getenv("APP_URL")
	if url == "" {
		return "http://localhost:3003"
	}
	return url
}

const baseEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #ddd; border-radius: 8px; }
        .header { background-color: #0a1628; color: #ffd700; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .footer { text-align: center; margin-top: 20px; font-size: 12px; color: #777; }
        .btn { display: inline-block; padding: 10px 20px; background-color: #ffd700; color: #0a1628; text-decoration: none; border-radius: 5px; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>ArcheryHub</h2>
        </div>
        <div class="content">
            {{.Body}}
        </div>
        <div class="footer">
            <p>&copy; ArcheryHub. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`

func renderTemplate(bodyHTML string, data interface{}) (string, error) {
	tmpl, err := template.New("email").Parse(baseEmailTemplate)
	if err != nil {
		return "", err
	}
	
	bodyTmpl, err := template.New("body").Parse(bodyHTML)
	if err != nil {
		return "", err
	}
	var bodyBuf bytes.Buffer
	if err := bodyTmpl.Execute(&bodyBuf, data); err != nil {
		return "", err
	}
	
	var finalBuf bytes.Buffer
	if err := tmpl.Execute(&finalBuf, map[string]interface{}{"Body": template.HTML(bodyBuf.String())}); err != nil {
		return "", err
	}
	
	return finalBuf.String(), nil
}

func SendArcherWelcomeEmail(to, fullName, email, plainPassword string) error {
	subject := "Selamat Datang di ArcheryHub!"
	
	bodyHTML := `
	<p>Halo {{.FullName}},</p>
	<p>Selamat datang di ArcheryHub! Akun Anda telah berhasil dibuat.</p>
	<p>Berikut adalah informasi login Anda:</p>
	<ul>
		<li><strong>Email:</strong> {{.Email}}</li>
		<li><strong>Password:</strong> {{.Password}}</li>
	</ul>
	<p>Silakan login dan lengkapi profil Anda.</p>
	<p><a href="{{.LoginURL}}" class="btn">Login Sekarang</a></p>
	`
	
	data := map[string]interface{}{
		"FullName": fullName,
		"Email": email,
		"Password": plainPassword,
		"LoginURL": getAppURL() + "/login",
	}
	
	html, err := renderTemplate(bodyHTML, data)
	if err != nil {
		return err
	}
	
	return SendEmail(to, subject, html)
}

func SendPaymentApprovedEmail(to, archerName, eventName string, amount float64, categories []string) error {
	subject := "Pembayaran Berhasil Dikonfirmasi - ArcheryHub"
	
	bodyHTML := `
	<p>Halo {{.ArcherName}},</p>
	<p>Pembayaran Anda untuk event <strong>{{.EventName}}</strong> sebesar Rp {{.Amount}} telah berhasil dikonfirmasi.</p>
	<p>Kategori yang didaftarkan:</p>
	<ul>
		{{range .Categories}}
		<li>{{.}}</li>
		{{end}}
	</ul>
	<p>Terima kasih telah berpartisipasi!</p>
	`
	
	data := map[string]interface{}{
		"ArcherName": archerName,
		"EventName": eventName,
		"Amount": formatIDR(amount),
		"Categories": categories,
	}
	
	html, err := renderTemplate(bodyHTML, data)
	if err != nil {
		return err
	}
	
	return SendEmail(to, subject, html)
}

func SendPaymentCreatedEmail(to, archerName, eventName, paymentURL string, amount float64) error {
	subject := "Selesaikan Pembayaran Anda - ArcheryHub"
	
	bodyHTML := `
	<p>Halo {{.ArcherName}},</p>
	<p>Pendaftaran Anda untuk event <strong>{{.EventName}}</strong> hampir selesai.</p>
	<p>Silakan selesaikan pembayaran sebesar Rp {{.Amount}} melalui tautan berikut:</p>
	<p><a href="{{.PaymentURL}}" class="btn">Bayar Sekarang</a></p>
	<p>Abaikan email ini jika Anda sudah melakukan pembayaran.</p>
	`
	
	data := map[string]interface{}{
		"ArcherName": archerName,
		"EventName": eventName,
		"Amount": formatIDR(amount),
		"PaymentURL": paymentURL,
	}
	
	html, err := renderTemplate(bodyHTML, data)
	if err != nil {
		return err
	}
	
	return SendEmail(to, subject, html)
}
