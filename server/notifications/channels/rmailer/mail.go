package rmailer

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/wneessen/go-mail"

	"github.com/realvnc-labs/rport/server/chconfig"
	"github.com/realvnc-labs/rport/share/logger"
)

type Mailer interface {
	Send(ctx context.Context, to []string, subject string, contentType ContentType, body string) error
}

const MaxHangingMailSends = 20

// ContentType represents a content type for the Msg
type ContentType string

func (t ContentType) Valid() error {
	switch t {
	case ContentTypeTextHTML:
		return nil
	case ContentTypeTextPlain:
		return nil
	default:
		return fmt.Errorf("badd content type: %v", t)
	}

}

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
	config    Config
	doomQueue chan struct{}

	l *logger.Logger
}

func (rm rMailer) Send(ctx context.Context, to []string, subject string, contentType ContentType, body string) error {

	if err := contentType.Valid(); err != nil {
		return fmt.Errorf("invalid content type: %v", err)
	}

	mailerOut := rm.enqueueSend(ctx, to, subject, contentType, body)

	if len(rm.doomQueue) >= MaxHangingMailSends {
		return fmt.Errorf("smtp server non-responsive")
	}

	select {
	case <-ctx.Done():
		select {
		case <-time.After(time.Millisecond):
			return fmt.Errorf("timeout sending mail")
		case err := <-mailerOut:
			return err
		}

	case err := <-mailerOut:
		return err
	}

}

func (rm rMailer) send(ctx context.Context, to []string, subject string, contentType ContentType, body string) error {
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

	rm.l.Debugf("dialing and sending mail message")
	if err := client.DialAndSendWithContext(ctx, m); err != nil {
		return fmt.Errorf("failed to send mail: %s", err)
	}

	rm.l.Debugf("sent smtp message: %v", m)
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

func (rm rMailer) enqueueSend(ctx context.Context, to []string, subject string, contentType ContentType, body string) chan error {
	done := make(chan error, 1)
	go func() {
		rm.doomQueue <- struct{}{}
		done <- rm.send(ctx, to, subject, contentType, body)
		close(done)
		<-rm.doomQueue
	}()

	return done
}

type AuthUserPass struct {
	User string
	Pass string
}

func NewRMailer(config Config, l *logger.Logger) Mailer {
	return rMailer{
		config:    config,
		doomQueue: make(chan struct{}, MaxHangingMailSends),
		l:         l,
	}
}

type Config struct {
	Host         string
	Port         int
	Domain       string
	From         string
	TLS          bool
	AuthType     AuthType
	AuthUserPass AuthUserPass
	NoNoop       bool
}

func ConfigFromSMTPConfig(config chconfig.SMTPConfig) (Config, error) {
	u, err := url.Parse(config.Server)
	if err != nil {
		return Config{}, fmt.Errorf("can't parse host from SMTP config: %v", err)
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
		port = -1
	} else {
		port, err = strconv.Atoi(sPort)
		if err != nil {
			return Config{}, fmt.Errorf("can't parse port number: %v", err)
		}
	}

	emailSplit := strings.Split(config.SenderEmail, "@")
	if len(emailSplit) != 2 {
		return Config{}, fmt.Errorf("can't parse email from SMTP config")
	}

	return Config{
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
