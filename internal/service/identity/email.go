package identity

import (
	"context"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/StellaShiina/ktauth/internal/crypto"
	"github.com/StellaShiina/ktauth/internal/repository"
	"github.com/google/uuid"
)

// EmailService sends and validates short-lived registration email codes.
type EmailService struct {
	registerRepo  *repository.RegisterRepo
	countdownRepo *repository.CountDownRepo
	host          string
	port          string
	username      string
	password      string
	from          string
}

func NewEmailService(registerRepo *repository.RegisterRepo, countdownRepo *repository.CountDownRepo, host, port, username, password, from string) *EmailService {
	return &EmailService{registerRepo, countdownRepo, host, port, username, password, from}
}

func (s *EmailService) SendCode(ctx context.Context, email string) error {
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return fmt.Errorf("invalid email address")
	}
	if s.host == "" || s.port == "" || s.from == "" {
		return fmt.Errorf("SMTP is not configured")
	}
	fromAddress, err := mail.ParseAddress(s.from)
	if err != nil {
		return fmt.Errorf("invalid SMTP from address")
	}

	inCooldown, err := s.countdownRepo.CD(ctx, email)
	if err != nil {
		return err
	}
	if inCooldown {
		return fmt.Errorf("verification code recently sent")
	}

	code, err := crypto.GenerateCode(6)
	if err != nil {
		return err
	}
	body := fmt.Sprintf(`<!doctype html>
<html lang="en">
<body style="margin:0;padding:0;background:#f3f5f7;font-family:Arial,Helvetica,sans-serif;color:#20252b;">
  <div style="display:none;max-height:0;overflow:hidden;opacity:0;">Use %s to complete your KTAUTH registration.</div>
  <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="background:#f3f5f7;padding:32px 16px;">
    <tr><td align="center">
      <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="max-width:560px;background:#ffffff;border:1px solid #dfe3e8;">
        <tr><td style="padding:24px 32px;background:#17202a;color:#ffffff;font-size:20px;font-weight:bold;">KTAUTH</td></tr>
        <tr><td style="padding:36px 32px 20px;font-size:24px;font-weight:bold;line-height:1.3;">Verify your email address</td></tr>
        <tr><td style="padding:0 32px 24px;font-size:15px;line-height:1.6;color:#525b65;">Enter this verification code to complete your registration:</td></tr>
        <tr><td align="center" style="padding:0 32px 28px;">
          <div style="display:inline-block;padding:16px 24px;border:1px solid #b8c2cc;background:#f7f9fa;font-family:Consolas,'Courier New',monospace;font-size:32px;font-weight:bold;letter-spacing:8px;color:#17202a;">%s</div>
        </td></tr>
        <tr><td style="padding:0 32px 36px;font-size:13px;line-height:1.6;color:#69737d;">This code expires in 15 minutes and can only be used once. If you did not request it, you can safely ignore this email.</td></tr>
        <tr><td style="padding:18px 32px;border-top:1px solid #e5e8eb;font-size:12px;color:#8a939c;">KTAUTH account security</td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`, code, code)
	message := []byte("From: " + fromAddress.String() + "\r\n" +
		"To: " + address.String() + "\r\n" +
		"Date: " + time.Now().Format(time.RFC1123Z) + "\r\n" +
		"Message-ID: <" + uuid.NewString() + "@" + s.host + ">\r\n" +
		"Subject: KTAUTH verification code\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Transfer-Encoding: 8bit\r\n\r\n" + body + "\r\n")

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}
	if err := smtp.SendMail(s.host+":"+s.port, auth, fromAddress.Address, []string{address.Address}, message); err != nil {
		return err
	}
	if err := s.registerRepo.Set(ctx, email, code); err != nil {
		return err
	}
	return s.countdownRepo.Set(ctx, email)
}

func (s *EmailService) VerifyCode(ctx context.Context, email, code string) (bool, error) {
	if strings.TrimSpace(email) == "" || strings.TrimSpace(code) == "" {
		return false, fmt.Errorf("email and code are required")
	}
	return s.registerRepo.Validate(ctx, email, code)
}
