package message

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"

	"github.com/jordan-wright/email"
)

type SMTPService struct {
	Auth      smtp.Auth
	HostPort  string
	From      string
	TLSConfig *tls.Config
}

func NewSMTPService(hostPort, username, password, fromEmail string, withTLS bool) (*SMTPService, error) {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return nil, err
	}

	s := &SMTPService{
		HostPort: hostPort,
		From:     fromEmail,
	}
	if username != "" || password != "" {
		s.Auth = smtp.PlainAuth("", username, password, host)
	}
	if withTLS {
		s.TLSConfig = &tls.Config{
			ServerName: host,
			MinVersion: tls.VersionTLS12,
		}
	}
	return s, nil
}

func (s *SMTPService) Send(title, msg, receiver string) error {
	e := &email.Email{
		From:    s.From,
		To:      []string{receiver},
		Subject: title,
		Text:    []byte(msg),
	}

	if s.TLSConfig != nil {
		err := e.SendWithTLS(s.HostPort, s.Auth, s.TLSConfig)
		if err != nil {
			return fmt.Errorf("failed to send email using TLS: %v", err)
		}
	} else {
		err := e.Send(s.HostPort, s.Auth)
		if err != nil {
			return fmt.Errorf("failed to send email: %v", err)
		}
	}

	return nil
}

func (s *SMTPService) DeliveryMethod() string {
	return "email"
}
