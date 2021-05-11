package message

import (
	"fmt"
	"net"
	"net/smtp"
)

type SMTPService struct {
	Auth     smtp.Auth
	Hostport string
	From     string
}

func NewSMTPService(hostPort, username, password, fromEmail string) (*SMTPService, error) {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return nil, err
	}

	return &SMTPService{
		Hostport: hostPort,
		Auth:     smtp.PlainAuth("", username, password, host),
		From:     fromEmail,
	}, nil
}

func (s *SMTPService) Send(title, msg, receiver string) error {
	emailTemplate := "To: %s\r\n" +
		"Subject: %s\r\n" +
		"\r\n" +
		"Token: %s\r\n"
	email := []byte(fmt.Sprintf(emailTemplate, receiver, title, msg))
	err := smtp.SendMail(s.Hostport, s.Auth, s.From, []string{receiver}, email)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}
	return nil
}
