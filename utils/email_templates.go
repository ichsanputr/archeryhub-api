package utils

import (
	"fmt"
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
		return "https://dev.archeris.net"
	}
	return url
}

// buildCleanCardEmail generates a lightweight, high text-to-HTML ratio email template.
// Designed specifically to maximize deliverability and achieve near-zero SpamAssassin penalties.
func buildCleanCardEmail(badgeText, title, contentHTML string) string {
	badgeHTML := ""
	if badgeText != "" {
		badgeHTML = fmt.Sprintf(`<div style="display:inline-block;background-color:#f1f5f9;color:#334155;border:1px solid #cbd5e1;padding:4px 12px;border-radius:20px;font-size:11px;font-weight:700;letter-spacing:0.5px;text-transform:uppercase;margin-bottom:16px;">%s</div>`, badgeText)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
</head>
<body style="margin:0;padding:0;background-color:#f8fafc;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:#1e293b;line-height:1.6;-webkit-text-size-adjust:none;">
<table role="presentation" width="100%%" border="0" cellspacing="0" cellpadding="0" style="background-color:#f8fafc;padding:30px 15px;">
<tr>
<td align="center">
    <table role="presentation" width="560" border="0" cellspacing="0" cellpadding="0" style="max-width:560px;width:100%%;background-color:#ffffff;border:1px solid #e2e8f0;border-radius:16px;overflow:hidden;">
        
        <!-- Header -->
        <tr>
            <td style="background-color:#0f172a;padding:24px 30px;border-bottom:3px solid #d9ff00;">
                <table role="presentation" width="100%%" border="0" cellspacing="0" cellpadding="0">
                    <tr>
                        <td>
                            <div style="font-size:22px;font-weight:900;color:#ffffff;letter-spacing:-0.5px;">
                                Archeris<span style="color:#d9ff00;">.net</span>
                            </div>
                        </td>
                        <td align="right">
                            <span style="font-size:11px;color:#94a3b8;font-weight:600;">Platform Panahan</span>
                        </td>
                    </tr>
                </table>
            </td>
        </tr>

        <!-- Main Body -->
        <tr>
            <td style="padding:32px 30px 24px;">
                %s
                <h1 style="margin:0 0 16px;font-size:18px;font-weight:800;color:#0f172a;line-height:1.4;">%s</h1>
                %s
            </td>
        </tr>

        <!-- Footer -->
        <tr>
            <td style="background-color:#f8fafc;padding:20px 30px;border-top:1px solid #e2e8f0;text-align:center;">
                <p style="margin:0 0 4px;font-size:12px;color:#64748b;">
                    Email dikirim otomatis oleh <strong>Archeris.net</strong>. Mohon tidak membalas email ini.
                </p>
                <p style="margin:0;font-size:11px;color:#94a3b8;">
                    &copy; 2026 Archeris.net &bull; Platform Panahan Indonesia
                </p>
            </td>
        </tr>

    </table>
</td>
</tr>
</table>
</body>
</html>`, title, badgeHTML, title, contentHTML)
}

// buildOTPBox generates a clean, high-contrast OTP code display.
func buildOTPBox(otp string, expiresMinutes int) string {
	return fmt.Sprintf(`
		<div style="background-color:#f8fafc;border:1px solid #e2e8f0;border-radius:12px;padding:20px;text-align:center;margin:24px 0;">
			<div style="font-size:11px;font-weight:700;color:#64748b;letter-spacing:1.5px;text-transform:uppercase;margin-bottom:8px;">Kode Verifikasi</div>
			<div style="font-family:'Courier New',Consolas,monospace;font-size:36px;font-weight:900;color:#0f172a;letter-spacing:8px;line-height:1;margin-left:8px;">%s</div>
		</div>
		<p style="margin:0 0 16px;font-size:13px;color:#64748b;line-height:1.6;">
			Kode berlaku selama <strong>%d menit</strong>. Jangan berikan kode ini kepada siapapun demi keamanan akun Anda.
		</p>
	`, otp, expiresMinutes)
}

// SendRegisterOTPEmail sends an OTP verification email for new account registrations.
func SendRegisterOTPEmail(to, fullName, otp string, expiresMinutes int) error {
	subject := fmt.Sprintf("Kode verifikasi akun Archeris: %s", otp)
	greeting := "Halo"
	if strings.TrimSpace(fullName) != "" {
		greeting = fmt.Sprintf("Halo %s", fullName)
	}

	contentHTML := fmt.Sprintf(`
		<p style="margin:0 0 12px;font-size:14px;color:#334155;">
			%s, terima kasih telah mendaftar di <strong>Archeris.net</strong>.
		</p>
		<p style="margin:0 0 16px;font-size:14px;color:#334155;">
			Gunakan kode verifikasi berikut untuk mengaktifkan akun Anda:
		</p>
		%s
		<p style="margin:0;font-size:12px;color:#94a3b8;">
			Jika Anda tidak merasa mendaftar di Archeris, silakan abaikan email ini.
		</p>
	`, greeting, buildOTPBox(otp, expiresMinutes))

	body := buildCleanCardEmail("Verifikasi Akun Baru", "Verifikasi Alamat Email Anda", contentHTML)
	return SendEmail(to, subject, body)
}

// SendPasswordResetOTPEmail sends an OTP email specifically for password resets.
func SendPasswordResetOTPEmail(to, fullName, otp string, expiresMinutes int) error {
	subject := fmt.Sprintf("Kode reset password Archeris: %s", otp)
	greeting := "Halo"
	if strings.TrimSpace(fullName) != "" {
		greeting = fmt.Sprintf("Halo %s", fullName)
	}

	contentHTML := fmt.Sprintf(`
		<p style="margin:0 0 12px;font-size:14px;color:#334155;">
			%s, kami menerima permintaan untuk mereset kata sandi akun Archeris Anda.
		</p>
		<p style="margin:0 0 16px;font-size:14px;color:#334155;">
			Gunakan kode OTP berikut untuk melanjutkan proses reset kata sandi:
		</p>
		%s
		<p style="margin:0;font-size:12px;color:#94a3b8;">
			Jika Anda tidak melakukan permintaan ini, abaikan email ini dan kata sandi akun Anda tetap aman.
		</p>
	`, greeting, buildOTPBox(otp, expiresMinutes))

	body := buildCleanCardEmail("Reset Password", "Atur Ulang Kata Sandi Akun", contentHTML)
	return SendEmail(to, subject, body)
}

// SendEmailChangeOTPEmail sends an OTP email to verify a new email address.
func SendEmailChangeOTPEmail(to, newEmail, otp string, expiresMinutes int) error {
	subject := fmt.Sprintf("Kode verifikasi pergantian email Archeris: %s", otp)

	contentHTML := fmt.Sprintf(`
		<p style="margin:0 0 12px;font-size:14px;color:#334155;">
			Halo, kami menerima permintaan untuk mengubah alamat email akun Archeris Anda ke <strong>%s</strong>.
		</p>
		<p style="margin:0 0 16px;font-size:14px;color:#334155;">
			Gunakan kode verifikasi berikut untuk mengonfirmasi perubahan alamat email:
		</p>
		%s
		<p style="margin:0;font-size:12px;color:#94a3b8;">
			Jika Anda tidak merasa mengajukan perubahan ini, silakan abaikan email ini.
		</p>
	`, newEmail, buildOTPBox(otp, expiresMinutes))

	body := buildCleanCardEmail("Perubahan Email", "Verifikasi Alamat Email Baru", contentHTML)
	return SendEmail(to, subject, body)
}

// SendEventResetOTPEmail sends an authorization OTP code to reset tournament data.
func SendEventResetOTPEmail(to, organizerName, otp string, expiresMinutes int) error {
	subject := fmt.Sprintf("Kode konfirmasi reset data turnamen Archeris: %s", otp)

	contentHTML := fmt.Sprintf(`
		<p style="margin:0 0 12px;font-size:14px;color:#334155;">
			Halo %s, kami menerima permintaan otorisasi untuk mereset data turnamen pada akun Anda.
		</p>
		<p style="margin:0 0 16px;font-size:14px;color:#334155;">
			Gunakan kode verifikasi berikut untuk mengonfirmasi tindakan:
		</p>
		%s
		<p style="margin:0;font-size:12px;color:#dc2626;font-weight:bold;">
			Peringatan: Reset data bersifat permanen dan tidak dapat dibatalkan.
		</p>
	`, organizerName, buildOTPBox(otp, expiresMinutes))

	body := buildCleanCardEmail("Peringatan Keamanan", "Konfirmasi Reset Data Turnamen", contentHTML)
	return SendEmail(to, subject, body)
}

// SendOTPEmail is a backward-compatible wrapper defaulting to generic verification.
func SendOTPEmail(to, fullName, otp string, expiresMinutes int) error {
	return SendRegisterOTPEmail(to, fullName, otp, expiresMinutes)
}

// SendArcherWelcomeEmail sends account credentials to a newly created archer.
func SendArcherWelcomeEmail(to, fullName, email, plainPassword string) error {
	subject := "Selamat Datang di Archeris!"
	loginURL := getAppURL() + "/auth/login"

	contentHTML := fmt.Sprintf(`
		<p style="margin:0 0 12px;font-size:14px;color:#334155;">
			Halo <strong>%s</strong>, akun atlet Anda di <strong>Archeris.net</strong> telah berhasil dibuat.
		</p>
		<div style="background-color:#f8fafc;border:1px solid #e2e8f0;border-radius:12px;padding:16px 20px;margin:20px 0;">
			<div style="font-size:11px;font-weight:700;color:#64748b;text-transform:uppercase;margin-bottom:8px;">Informasi Akun Login</div>
			<div style="font-size:13px;color:#334155;margin-bottom:4px;"><strong>Email:</strong> %s</div>
			<div style="font-size:13px;color:#334155;"><strong>Password:</strong> %s</div>
		</div>
		<div style="text-align:center;margin:24px 0;">
			<a href="%s" style="display:inline-block;background-color:#0f172a;color:#d9ff00;text-decoration:none;padding:12px 28px;border-radius:10px;font-weight:800;font-size:13px;">
				Masuk ke Akun
			</a>
		</div>
		<p style="margin:0;font-size:12px;color:#94a3b8;">
			Demi keamanan akun Anda, segera ubah password default setelah berhasil masuk.
		</p>
	`, fullName, email, plainPassword, loginURL)

	body := buildCleanCardEmail("Selamat Datang", "Akun Archeris Anda Telah Aktif", contentHTML)
	return SendEmail(to, subject, body)
}

// SendPaymentApprovedEmail sends a confirmation when payment is confirmed.
func SendPaymentApprovedEmail(to, archerName, eventName string, amount float64, categories []string) error {
	subject := fmt.Sprintf("Pembayaran Terkonfirmasi - %s", eventName)
	overviewURL := getAppURL() + "/dashboard/archer/tournaments"

	categoriesListHTML := ""
	for _, cat := range categories {
		categoriesListHTML += fmt.Sprintf(`<li style="margin-bottom:4px;">%s</li>`, cat)
	}

	contentHTML := fmt.Sprintf(`
		<p style="margin:0 0 12px;font-size:14px;color:#334155;">
			Halo <strong>%s</strong>, pembayaran pendaftaran Anda untuk event <strong>%s</strong> telah berhasil diverifikasi oleh panitia.
		</p>
		<div style="background-color:#f0fdf4;border:1px solid #bbf7d0;border-radius:12px;padding:16px 20px;margin:20px 0;">
			<div style="font-size:11px;font-weight:700;color:#15803d;text-transform:uppercase;margin-bottom:4px;">Status: Lunas</div>
			<div style="font-size:22px;font-weight:900;color:#0f172a;">Rp %s</div>
		</div>
		<div style="font-size:13px;color:#334155;margin-bottom:20px;">
			<strong>Kategori Terdaftar:</strong>
			<ul style="margin:8px 0 0;padding-left:20px;">%s</ul>
		</div>
		<div style="text-align:center;margin:24px 0;">
			<a href="%s" style="display:inline-block;background-color:#0f172a;color:#d9ff00;text-decoration:none;padding:12px 28px;border-radius:10px;font-weight:800;font-size:13px;">
				Lihat Jadwal & Status
			</a>
		</div>
	`, archerName, eventName, formatIDR(amount), categoriesListHTML, overviewURL)

	body := buildCleanCardEmail("Pembayaran Lunas", "Pendaftaran Event Terkonfirmasi", contentHTML)
	return SendEmail(to, subject, body)
}

// SendPaymentCreatedEmail sends an invoice notice with a payment link.
func SendPaymentCreatedEmail(to, archerName, eventName, paymentURL string, amount float64) error {
	subject := fmt.Sprintf("Tagihan Pendaftaran - %s", eventName)

	contentHTML := fmt.Sprintf(`
		<p style="margin:0 0 12px;font-size:14px;color:#334155;">
			Halo <strong>%s</strong>, pendaftaran Anda untuk event <strong>%s</strong> telah tercatat.
		</p>
		<div style="background-color:#fffbeb;border:1px solid #fde68a;border-radius:12px;padding:16px 20px;margin:20px 0;">
			<div style="font-size:11px;font-weight:700;color:#92400e;text-transform:uppercase;margin-bottom:4px;">Total Tagihan</div>
			<div style="font-size:22px;font-weight:900;color:#0f172a;">Rp %s</div>
		</div>
		<div style="text-align:center;margin:24px 0;">
			<a href="%s" style="display:inline-block;background-color:#0f172a;color:#d9ff00;text-decoration:none;padding:12px 28px;border-radius:10px;font-weight:800;font-size:13px;">
				Bayar Sekarang
			</a>
		</div>
		<p style="margin:0;font-size:12px;color:#94a3b8;">
			Abaikan email ini jika Anda sudah menyelesaikan pembayaran sebelumnya.
		</p>
	`, archerName, eventName, formatIDR(amount), paymentURL)

	body := buildCleanCardEmail("Tagihan Pendaftaran", "Selesaikan Pembayaran Pendaftaran", contentHTML)
	return SendEmail(to, subject, body)
}
