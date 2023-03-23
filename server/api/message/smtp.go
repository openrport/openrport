package message

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"

	"github.com/jordan-wright/email"

	email2 "github.com/realvnc-labs/rport/share/email"
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

func (s *SMTPService) Send(ctx context.Context, data Data) error {
	message := fmt.Sprintf(`You have requested a token for the login to your RPort server.
The token is: %s

The token has been requested from %s
with user agent %s.
Token is valid for %.0f seconds.`, data.Token, data.RemoteAddress, data.UserAgent, data.TTL.Seconds())
	e := &email.Email{
		From:    s.From,
		To:      []string{data.SendTo},
		Subject: data.Title,
		Text:    []byte(message),
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

func (s *SMTPService) ValidateReceiver(ctx context.Context, email string) error {
	return email2.Validate(email)
}
