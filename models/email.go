package models

import (
	"fmt"

	"github.com/go-mail/mail/v2"
)

const (
	DefaultSender = "support@galaria.com"
)

type Email struct {
	From      string
	To        string
	Subject   string
	Plaintext string
	HTML      string
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
}

func NewEmailService(cfg SMTPConfig) *EmailService {
	es := EmailService{
		dialer: mail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password),
	}

	return &es
}

type EmailService struct {
	DefaultSender string
	dialer        *mail.Dialer
}

func (es *EmailService) Send(email Email) error {
	msg := mail.NewMessage()
	es.setFrom(msg, email)
	msg.SetHeader("To", email.To)
	msg.SetHeader("Subject", email.Subject)
	msg.SetBody("text/plain", email.Plaintext)
	msg.AddAlternative("text/html", email.HTML)

	err := es.dialer.DialAndSend(msg)
	if err != nil {
		return fmt.Errorf("sending email: %w", err)
	}

	return nil
}

func (es *EmailService) setFrom(msg *mail.Message, email Email) {
	var from string

	switch {
	case email.From != "":
		from = email.From

	case es.DefaultSender != "":
		from = es.DefaultSender

	default:
		from = DefaultSender
	}

	msg.SetHeader("From", from)
}
