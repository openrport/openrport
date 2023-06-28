package rmailer

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"

	"github.com/realvnc-labs/rport/server/notifications"
)

type consumer struct {
	mailer Mailer
}

//nolint:revive
func NewConsumer(mailer Mailer) *consumer {
	return &consumer{mailer: mailer}
}

func (c consumer) Process(details notifications.NotificationDetails) error {
	content := details.Data.Content
	if ContentType(details.Data.ContentType) == ContentTypeTextHTML {
		var err error
		content, err = wrapWithTemplate(details.Data.Content)
		if err != nil {
			return fmt.Errorf("failed preparing notification to dispatch: %v", err)
		}
	}
	return c.mailer.Send(details.Data.Recipients, details.Data.Subject, ContentType(details.Data.ContentType), content)
}

func (c consumer) Target() notifications.Target {
	return notifications.TargetMail
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
