package rmailer

import (
	"fmt"
	"github.com/realvnc-labs/rport/server/chconfig"
	"github.com/wneessen/go-mail"
	"net/url"
	"strconv"
	"strings"
)

type Mailer interface {
	Send(to []string, subject string, contentType ContentType, body string) error
}

// ContentType represents a content type for the Msg
type ContentType string

// List of common content types
const (
	ContentTypeTextPlain ContentType = "text/plain"
	ContentTypeTextHTML  ContentType = "text/html"
)

type AuthType string

// List of common content types
const (
	AuthTypeNone     AuthType = "none"
	AuthTypeUserPass AuthType = "user-pass"
)

type rMailer struct {
	config RMailerConfig
}

func (rm rMailer) Send(to []string, subject string, contentType ContentType, body string) error {
	m := mail.NewMsg()
	if err := m.From(rm.config.From); err != nil {
		return fmt.Errorf("failed to set From address: %s", err)
	}
	if err := m.To(to...); err != nil {
		return fmt.Errorf("failed to set To address: %s", err)
	}

	m.Subject(subject)
	m.SetBodyString(mail.ContentType(contentType), body)

	client, err := rm.buildClient()

	if err != nil {
		return fmt.Errorf("failed to create mail client: %s", err)
	}

	if err := client.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send mail: %s", err)
	}

	return nil
}

func (rm rMailer) buildClient() (*mail.Client, error) {

	options := []mail.Option{
		mail.WithHELO(rm.config.Domain),
	}

	if rm.config.TLS {
		options = append(options, mail.WithTLSPolicy(mail.TLSMandatory))
	} else {
		options = append(options, mail.WithTLSPolicy(mail.NoTLS))
	}

	if rm.config.NoNoop {
		options = append(options, mail.WithoutNoop())
	}

	if rm.config.Port > 0 { // if we have Port, don't let library guess but enforce Port
		options = append(options, mail.WithPort(rm.config.Port))
	}

	client, err := mail.NewClient(
		rm.config.Host,
		options...,
	)

	return client, err
}

type AuthUserPass struct {
	User string
	Pass string
}

// NewMailerFromSMTPConfig NewMailer gives you something that is thread safe and can send mail
func NewMailerFromSMTPConfig(smtpConfig chconfig.SMTPConfig) (Mailer, error) {
	config, err := RMailerConfigFromSMTPConfig(smtpConfig)
	if err != nil {
		return nil, fmt.Errorf("can't convert SMTPConfig to RMailerConfig: %v", err)
	}

	return NewRMailer(config), nil
}

func NewRMailer(config RMailerConfig) Mailer {
	return rMailer{
		config: config,
	}
}

type RMailerConfig struct {
	Host         string
	Port         int
	Domain       string
	From         string
	TLS          bool
	AuthType     AuthType
	AuthUserPass AuthUserPass
	NoNoop       bool
}

func RMailerConfigFromSMTPConfig(config chconfig.SMTPConfig) (RMailerConfig, error) {
	u, err := url.Parse(config.Server)
	if err != nil {
		return RMailerConfig{}, fmt.Errorf("can't parse host from SMTP config: %v", err)
	}
	sPort := u.Port()

	var host string
	if u.Hostname() == "" {
		parts := strings.Split(config.Server, ":")
		host = parts[0]

		if len(parts) == 2 {
			sPort = parts[1]
		}
	} else {
		host = u.Hostname()
	}

	var port int
	if sPort == "" {
		port = -1 // note to self to set lib to Port guessing
	} else {
		port, err = strconv.Atoi(sPort)
		if err != nil {
			return RMailerConfig{}, fmt.Errorf("can't parse port number: %v", err)
		}
	}

	emailSplit := strings.Split(config.SenderEmail, "@")
	if len(emailSplit) != 2 {
		return RMailerConfig{}, fmt.Errorf("can't parse email from SMTP config")
	}

	return RMailerConfig{
		Host:     host,
		Port:     port,
		Domain:   emailSplit[1],
		From:     config.SenderEmail,
		TLS:      config.Secure,
		AuthType: AuthTypeUserPass,
		AuthUserPass: AuthUserPass{
			User: config.AuthUsername,
			Pass: config.AuthPassword,
		},
	}, nil
}
