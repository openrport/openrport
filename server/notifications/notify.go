package notifications

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"

	"github.com/realvnc-labs/rport/server/notifications/channels/rmailer"
)

// ContentType represents a content type for the Msg
type ContentType string

const (
	ContentTypeTextPlain ContentType = "text/plain"
	ContentTypeTextHTML  ContentType = "text/html"
	ContentTypeTextJSON  ContentType = "text/json"
)

type Notification struct {
	Target      string
	Recipients  []string
	Subject     string
	Content     string
	ContentType ContentType
}

type Notifier interface {
	Dispatch(notification Notification) error
}

type ScriptRunner interface {
	Run(script string, recipients []string, subject string, body string) error
}

type broker struct {
	mailer       rmailer.Mailer
	scriptRunner ScriptRunner
}

func (b broker) Dispatch(notification Notification) error {
	if notification.Target == "smtp" {
		content := notification.Content
		if notification.ContentType == ContentTypeTextHTML {
			var err error
			content, err = wrapWithTemplate(content)
			if err != nil {
				return fmt.Errorf("failed preparing notification to dispatch: %v", err)
			}
		}
		return b.mailer.Send(notification.Recipients, notification.Subject, rmailer.ContentType(notification.ContentType), content)
	}
	return b.scriptRunner.Run(notification.Target, notification.Recipients, notification.Subject, notification.Content)
}

//go:embed mailTemplate.tmpl
var mailTemplate string

func wrapWithTemplate(content string) (string, error) {
	tmpl, err := template.New("mail").Parse(mailTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, struct {
		Body string
	}{Body: content})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func NewNotifier(mailer rmailer.Mailer, runner ScriptRunner) Notifier {
	return broker{mailer: mailer, scriptRunner: runner}
}
