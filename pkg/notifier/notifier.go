package notifier

import (
	"fmt"
	"log/slog"
	"net/smtp"
)

type Email struct {
	smtpHost       string
	smtpPort       string
	senderEmail    string
	senderPassword string
}

func NewEmail(
	smtpHost string,
	smtpPort string,
	senderEmail string,
	senderPassword string,
) *Email {
	return &Email{
		smtpHost:       smtpHost,
		smtpPort:       smtpPort,
		senderEmail:    senderEmail,
		senderPassword: senderPassword,
	}
}

func (e *Email) Send(addr string, msg string) error {
	message := []byte("To: " + addr + "\r\n" +
		"Subject: Warning: Suspicious Activity Detected\r\n" +
		"\r\n" +
		"Hello,\r\n\r\n" + msg + "\r\n")

	auth := smtp.PlainAuth("", e.senderEmail, e.senderPassword, e.smtpHost)

	err := smtp.SendMail(e.smtpHost+":"+e.smtpPort, auth, e.senderEmail, []string{addr}, message)
	if err != nil {
		return fmt.Errorf("Email.Send: %w", err)
	}
	slog.Info("Email sent", "email", addr, "text", msg)
	return nil
}
