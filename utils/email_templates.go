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

func GetAppURL() string {
	url := os.Getenv("APP_URL")
	if url == "" {
		return "https://archeris.net"
	}
	return url
}

func getAppURL() string {
	return GetAppURL()
}

// BuildCleanCardEmail generates a lightweight, minimal, high-deliverability email template.
func BuildCleanCardEmail(title, contentHTML string) string {
	return buildCleanCardEmail(title, contentHTML)
}

// buildCleanCardEmail generates a lightweight, minimal, high-deliverability email template.
// Clean modern design without heavy headers or complex elements.
func buildCleanCardEmail(title, contentHTML string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
</head>
<body style="margin:0;padding:0;background-color:#f9fafb;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:#1f2937;line-height:1.6;-webkit-text-size-adjust:none;">
<table role="presentation" width="100%%" border="0" cellspacing="0" cellpadding="0" style="background-color:#f9fafb;padding:32px 16px;">
<tr>
<td align="center">
    <table role="presentation" width="480" border="0" cellspacing="0" cellpadding="0" style="max-width:480px;width:100%%;background-color:#ffffff;border:1px solid #e5e7eb;border-radius:12px;padding:36px 32px;box-sizing:border-box;">
        
        <!-- Brand Header -->
        <tr>
            <td style="padding-bottom:24px;">
                <span style="font-size:20px;font-weight:800;color:#111827;letter-spacing:-0.5px;">Archeris</span>
            </td>
        </tr>

        <!-- Main Body -->
        <tr>
            <td>
                <h1 style="margin:0 0 16px;font-size:18px;font-weight:700;color:#111827;line-height:1.4;">%s</h1>
                %s
            </td>
        </tr>

        <!-- Footer -->
        <tr>
            <td style="border-top:1px solid #f3f4f6;padding-top:24px;margin-top:32px;text-align:center;">
                <p style="margin:0 0 4px;font-size:12px;color:#9ca3af;line-height:1.5;">
                    This is an automated email from <strong>Archeris</strong>. Please do not reply.
                </p>
                <p style="margin:0;font-size:11px;color:#9ca3af;">
                    &copy; 2026 Archeris. All rights reserved.
                </p>
            </td>
        </tr>

    </table>
</td>
</tr>
</table>
</body>
</html>`, title, title, contentHTML)
}

// buildOTPBox generates a clean, high-contrast OTP code display.
func buildOTPBox(otp string, expiresMinutes int) string {
	return fmt.Sprintf(`
		<div style="background-color:#f3f4f6;border:1px solid #e5e7eb;border-radius:8px;padding:20px;text-align:center;margin:24px 0;">
			<div style="font-size:11px;font-weight:700;color:#6b7280;letter-spacing:1px;text-transform:uppercase;margin-bottom:6px;">Verification Code</div>
			<div style="font-family:'Courier New',Consolas,monospace;font-size:34px;font-weight:900;color:#111827;letter-spacing:6px;line-height:1;">%s</div>
		</div>
		<p style="margin:0 0 16px;font-size:13px;color:#6b7280;line-height:1.5;">
			This code will expire in <strong>%d minutes</strong>. For your security, do not share this code with anyone.
		</p>
	`, otp, expiresMinutes)
}

// SendRegisterOTPEmail sends an OTP verification email for new account registrations.
func SendRegisterOTPEmail(to, fullName, otp string, expiresMinutes int) error {
	subject := fmt.Sprintf("Archeris verification code: %s", otp)
	greeting := "Hi"
	if strings.TrimSpace(fullName) != "" {
		greeting = fmt.Sprintf("Hi %s", fullName)
	}

	contentHTML := fmt.Sprintf(`
		<p style="margin:0 0 12px;font-size:14px;color:#374151;">
			%s, thank you for joining <strong>Archeris</strong>.
		</p>
		<p style="margin:0 0 16px;font-size:14px;color:#374151;">
			Use the verification code below to activate your account:
		</p>
		%s
		<p style="margin:0;font-size:12px;color:#9ca3af;">
			If you did not sign up for Archeris, you can safely ignore this email.
		</p>
	`, greeting, buildOTPBox(otp, expiresMinutes))

	body := buildCleanCardEmail("Verify Your Email Address", contentHTML)
	return SendEmail(to, subject, body)
}

// SendPasswordResetOTPEmail sends an OTP email specifically for password resets.
func SendPasswordResetOTPEmail(to, fullName, otp string, expiresMinutes int) error {
	subject := fmt.Sprintf("Archeris password reset code: %s", otp)
	greeting := "Hi"
	if strings.TrimSpace(fullName) != "" {
		greeting = fmt.Sprintf("Hi %s", fullName)
	}

	contentHTML := fmt.Sprintf(`
		<p style="margin:0 0 12px;font-size:14px;color:#374151;">
			%s, we received a request to reset the password for your Archeris account.
		</p>
		<p style="margin:0 0 16px;font-size:14px;color:#374151;">
			Use the verification code below to proceed:
		</p>
		%s
		<p style="margin:0;font-size:12px;color:#9ca3af;">
			If you didn't request a password reset, you can safely ignore this email. Your password will remain unchanged.
		</p>
	`, greeting, buildOTPBox(otp, expiresMinutes))

	body := buildCleanCardEmail("Reset Your Password", contentHTML)
	return SendEmail(to, subject, body)
}

// SendEmailChangeOTPEmail sends an OTP email to verify a new email address.
func SendEmailChangeOTPEmail(to, newEmail, otp string, expiresMinutes int) error {
	subject := fmt.Sprintf("Archeris email change verification code: %s", otp)

	contentHTML := fmt.Sprintf(`
		<p style="margin:0 0 12px;font-size:14px;color:#374151;">
			We received a request to change your Archeris account email to <strong>%s</strong>.
		</p>
		<p style="margin:0 0 16px;font-size:14px;color:#374151;">
			Use the verification code below to confirm this change:
		</p>
		%s
		<p style="margin:0;font-size:12px;color:#9ca3af;">
			If you did not request this change, please ignore this email.
		</p>
	`, newEmail, buildOTPBox(otp, expiresMinutes))

	body := buildCleanCardEmail("Confirm Email Change", contentHTML)
	return SendEmail(to, subject, body)
}

// SendEventResetOTPEmail sends an authorization OTP code to reset tournament data.
func SendEventResetOTPEmail(to, organizerName, otp string, expiresMinutes int) error {
	subject := fmt.Sprintf("Archeris tournament reset authorization: %s", otp)

	contentHTML := fmt.Sprintf(`
		<p style="margin:0 0 12px;font-size:14px;color:#374151;">
			Hi %s, we received an authorization request to reset tournament data on your account.
		</p>
		<p style="margin:0 0 16px;font-size:14px;color:#374151;">
			Use the verification code below to confirm this action:
		</p>
		%s
		<p style="margin:0;font-size:12px;color:#ef4444;font-weight:600;">
			Warning: Resetting tournament data is permanent and cannot be undone.
		</p>
	`, organizerName, buildOTPBox(otp, expiresMinutes))

	body := buildCleanCardEmail("Confirm Tournament Reset", contentHTML)
	return SendEmail(to, subject, body)
}

// SendOTPEmail is a backward-compatible wrapper defaulting to generic verification.
func SendOTPEmail(to, fullName, otp string, expiresMinutes int) error {
	return SendRegisterOTPEmail(to, fullName, otp, expiresMinutes)
}

// SendArcherWelcomeEmail sends account credentials to a newly created archer.
func SendArcherWelcomeEmail(to, fullName, email, plainPassword string) error {
	subject := "Welcome to Archeris"
	loginURL := getAppURL() + "/auth/login"

	contentHTML := fmt.Sprintf(`
		<p style="margin:0 0 12px;font-size:14px;color:#374151;">
			Hi <strong>%s</strong>, your athlete account at <strong>Archeris</strong> has been created.
		</p>
		<div style="background-color:#f9fafb;border:1px solid #e5e7eb;border-radius:8px;padding:16px 20px;margin:20px 0;">
			<div style="font-size:11px;font-weight:700;color:#6b7280;text-transform:uppercase;margin-bottom:8px;">Account Credentials</div>
			<div style="font-size:13px;color:#374151;margin-bottom:4px;"><strong>Email:</strong> %s</div>
			<div style="font-size:13px;color:#374151;"><strong>Temporary Password:</strong> %s</div>
		</div>
		<div style="text-align:center;margin:24px 0;">
			<a href="%s" style="display:inline-block;background-color:#111827;color:#ffffff;text-decoration:none;padding:12px 28px;border-radius:8px;font-weight:700;font-size:13px;">
				Log In to Your Account
			</a>
		</div>
		<p style="margin:0;font-size:12px;color:#9ca3af;">
			For security reasons, please change your password after logging in.
		</p>
	`, fullName, email, plainPassword, loginURL)

	body := buildCleanCardEmail("Your Account is Ready", contentHTML)
	return SendEmail(to, subject, body)
}

// SendPaymentApprovedEmail sends a confirmation when payment is confirmed.
func SendPaymentApprovedEmail(to, archerName, eventName string, amount float64, categories []string) error {
	subject := fmt.Sprintf("Payment Confirmed - %s", eventName)
	overviewURL := getAppURL() + "/dashboard/archer/tournaments"

	categoriesListHTML := ""
	for _, cat := range categories {
		categoriesListHTML += fmt.Sprintf(`<li style="margin-bottom:4px;">%s</li>`, cat)
	}

	contentHTML := fmt.Sprintf(`
		<p style="margin:0 0 12px;font-size:14px;color:#374151;">
			Hi <strong>%s</strong>, your registration payment for <strong>%s</strong> has been verified and confirmed.
		</p>
		<div style="background-color:#f0fdf4;border:1px solid #bbf7d0;border-radius:8px;padding:16px 20px;margin:20px 0;">
			<div style="font-size:11px;font-weight:700;color:#16a34a;text-transform:uppercase;margin-bottom:4px;">Status: Paid</div>
			<div style="font-size:22px;font-weight:800;color:#111827;">Rp %s</div>
		</div>
		<div style="font-size:13px;color:#374151;margin-bottom:20px;">
			<strong>Registered Categories:</strong>
			<ul style="margin:8px 0 0;padding-left:20px;">%s</ul>
		</div>
		<div style="text-align:center;margin:24px 0;">
			<a href="%s" style="display:inline-block;background-color:#111827;color:#ffffff;text-decoration:none;padding:12px 28px;border-radius:8px;font-weight:700;font-size:13px;">
				View Schedule & Status
			</a>
		</div>
	`, archerName, eventName, formatIDR(amount), categoriesListHTML, overviewURL)

	body := buildCleanCardEmail("Registration Confirmed", contentHTML)
	return SendEmail(to, subject, body)
}

// SendPaymentCreatedEmail sends an invoice notice with a payment link.
func SendPaymentCreatedEmail(to, archerName, eventName, paymentURL string, amount float64) error {
	subject := fmt.Sprintf("Registration Invoice - %s", eventName)

	contentHTML := fmt.Sprintf(`
		<p style="margin:0 0 12px;font-size:14px;color:#374151;">
			Hi <strong>%s</strong>, your registration for <strong>%s</strong> has been recorded.
		</p>
		<div style="background-color:#fffbeb;border:1px solid #fde68a;border-radius:8px;padding:16px 20px;margin:20px 0;">
			<div style="font-size:11px;font-weight:700;color:#d97706;text-transform:uppercase;margin-bottom:4px;">Total Amount</div>
			<div style="font-size:22px;font-weight:800;color:#111827;">Rp %s</div>
		</div>
		<div style="text-align:center;margin:24px 0;">
			<a href="%s" style="display:inline-block;background-color:#111827;color:#ffffff;text-decoration:none;padding:12px 28px;border-radius:8px;font-weight:700;font-size:13px;">
				Pay Now
			</a>
		</div>
		<p style="margin:0;font-size:12px;color:#9ca3af;">
			Please ignore this email if you have already completed the payment.
		</p>
	`, archerName, eventName, formatIDR(amount), paymentURL)

	body := buildCleanCardEmail("Payment Invoice", contentHTML)
	return SendEmail(to, subject, body)
}
