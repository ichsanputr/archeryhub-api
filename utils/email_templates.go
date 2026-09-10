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

// SendOTPEmail mengirim email OTP reset password dengan design system Archeris
// (navy #0F172A + neon yellow #D9FF00, matching DESIGN.md & app_colors.dart).
// expiresMinutes = berapa menit OTP berlaku (ditampilkan di email).
func SendOTPEmail(to, fullName, otp string, expiresMinutes int) error {
	subject := "Kode OTP Reset Password - Archeris"

	html := `<!DOCTYPE html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"><title>OTP Archeris</title></head>
<body style="margin:0;padding:0;background:#F9FAFB;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;color:#1E293B;">
<table role="presentation" cellspacing="0" cellpadding="0" border="0" width="100%%" style="background:#F9FAFB;"><tr><td align="center" style="padding:32px 12px;">
<table role="presentation" cellspacing="0" cellpadding="0" border="0" width="560" style="max-width:560px;width:100%%;background:#fff;border-radius:20px;overflow:hidden;box-shadow:0 8px 30px rgba(15,23,42,0.08);">

<!-- HERO -->
<tr><td style="background:linear-gradient(135deg,#0F172A 0%%,#111827 60%%,#052e16 120%%);padding:36px 32px 28px;">
<div style="font-family:'Lexend',sans-serif;font-size:28px;font-weight:900;color:#fff;letter-spacing:-0.5px;">Archeris<span style="color:#D9FF00;">.id</span></div>
<div style="height:14px;line-height:14px;font-size:14px;">&nbsp;</div>
<div style="font-family:'Lexend',sans-serif;font-size:20px;font-weight:800;color:#E2E8F0;line-height:1.35;">Atur Ulang Akses Akunmu</div>
<div style="height:8px;line-height:8px;font-size:8px;">&nbsp;</div>
<div style="font-size:14px;color:#94A3B8;line-height:1.6;">Kami menjaga akunmu tetap aman dengan verifikasi OTP sekali pakai.</div>
</td></tr>

<!-- CONTENT -->
<tr><td style="padding:32px 32px 8px;">
<div style="display:inline-block;background:#F0FDF4;border:1px solid #D9F99D;border-radius:999px;padding:6px 14px;font-size:11px;font-weight:800;color:#4D7C0F;letter-spacing:0.8px;text-transform:uppercase;">Reset Password</div>
<div style="font-family:'Lexend',sans-serif;font-size:24px;font-weight:900;color:#0F172A;line-height:1.25;margin-top:18px;">Hai, %s!</div>
<div style="font-size:15px;color:#1E293B;line-height:1.7;margin-top:12px;">Kami menerima permintaan untuk mereset password akun kamu. Masukkan kode OTP berikut ke halaman reset password Archeris.</div>

<!-- OTP CARD -->
<table role="presentation" cellspacing="0" cellpadding="0" border="0" width="100%%" style="margin-top:24px;margin-bottom:24px;background:linear-gradient(180deg,#F7FEE7 0%%,#ECFCCB 100%%);border:1px solid #D9F99D;border-radius:18px;">
<tr><td align="center" style="padding:26px 20px;">
<div style="font-size:11px;font-weight:800;color:#4D7C0F;letter-spacing:2.4px;text-transform:uppercase;margin-bottom:12px;">KODE OTP KAMU</div>
<div style="font-family:'Lexend','Courier New',monospace;font-size:42px;font-weight:900;color:#0F172A;letter-spacing:10px;line-height:1;">%s</div>
</td></tr></table>

<div style="font-size:13px;color:#6B7280;line-height:1.7;margin-bottom:24px;"><strong style="color:#0F172A;">Penting:</strong> kode ini berlaku selama <strong style="color:#0F172A;">%d menit</strong> dan hanya bisa dipakai satu kali. Jangan bagikan kode ini kepada siapapun.</div>
</td></tr>

<!-- DISCLAIMER -->
<tr><td style="padding:0 32px 28px;">
<table role="presentation" cellspacing="0" cellpadding="0" border="0" width="100%%" style="background:#F8FAFC;border-left:3px solid #D9FF00;border-radius:6px;">
<tr><td style="padding:14px 16px;font-size:12.5px;color:#475569;line-height:1.6;">Tidak merasa meminta reset password? Abaikan email ini dan pastikan password akunmu tetap aman.</td></tr>
</table></td></tr>

<!-- FOOTER -->
<tr><td style="padding:20px 32px 28px;border-top:1px solid #E2E8F0;">
<div style="text-align:center;font-size:12px;color:#94A3B8;line-height:1.7;margin-bottom:8px;">Email ini dikirim otomatis oleh sistem Archeris untuk keamanan akunmu.</div>
<div style="text-align:center;font-size:11px;font-weight:600;color:#CBD5E1;">&copy; 2026 Archeris &bull; Platform Panahan No. 1 Indonesia</div>
</td></tr>

</table></td></tr></table></body></html>`

	body := fmt.Sprintf(html, fullName, otp, expiresMinutes)
	return SendEmail(to, subject, body)
}

