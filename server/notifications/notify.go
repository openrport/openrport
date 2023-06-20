package notifications

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"

	"github.com/realvnc-labs/rport/server/notifications/channels/rmailer"
	"github.com/realvnc-labs/rport/share/refs"
)

// ContentType represents a content type for the Msg
type ContentType string

const (
	ContentTypeTextPlain ContentType = "text/plain"
	ContentTypeTextHTML  ContentType = "text/html"
	ContentTypeTextJSON  ContentType = "text/json"
)

type NotificationData struct {
	Target      string
	Recipients  []string
	Subject     string
	Content     string
	ContentType ContentType
}

type Notifier interface {
	Dispatch(origin refs.Origin, notification NotificationData) (refs.Identifiable, error)
}

type ScriptRunner interface {
	Run(script string, recipients []string, subject string, body string) error
}

type broker struct {
	mailer       rmailer.Mailer
	scriptRunner ScriptRunner
}

type NotificationID string

const NotificationType refs.IdentifiableType = "notification"

func (nid NotificationID) Type() refs.IdentifiableType {
	return NotificationType
}

var identifiebleFactory = refs.MustIdentifiableFactory(NotificationType)

func (b broker) Dispatch(origin refs.Origin, notification NotificationData) (refs.Identifiable, error) {
	if notification.Target == "smtp" {
		content := notification.Content
		if notification.ContentType == ContentTypeTextHTML {
			var err error
			content, err = wrapWithTemplate(content)
			if err != nil {
				return identifiebleFactory(""), fmt.Errorf("failed preparing notification to dispatch: %v", err)
			}
		}
		return identifiebleFactory(""), b.mailer.Send(notification.Recipients, notification.Subject, rmailer.ContentType(notification.ContentType), content)
	}
	return identifiebleFactory(""), b.scriptRunner.Run(notification.Target, notification.Recipients, notification.Subject, notification.Content)
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

	return buf.String(), err
}

func NewNotifier(mailer rmailer.Mailer, runner ScriptRunner) Notifier {
	return broker{mailer: mailer, scriptRunner: runner}
}
