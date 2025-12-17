package auth

import (
	"os"

	"github.com/StellaShiina/ktauth/internal/crypto"
	"github.com/resend/resend-go/v2"
)

var resend_token = os.Getenv("RESEND_API_KEY")

// Return code string, err error.
func Resend(emails ...string) (string, error) {
	verficationCode, err := crypto.GenerateCode(6)
	if err != nil {
		return "", err
	}

	client := resend.NewClient(resend_token)

	params := &resend.SendEmailRequest{
		From:    "ktauth.noreply@ktauth.vvan.win",
		To:      emails,
		Subject: "ktauth",
		Html:    "<p>Your verification code(TTL:15min): <strong>" + verficationCode + "</strong></p>",
	}

	_, err = client.Emails.Send(params)

	if err != nil {
		return "", err
	}
	return verficationCode, nil
}
