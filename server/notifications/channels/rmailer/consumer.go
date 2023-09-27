package rmailer

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"text/template"

	"github.com/openrport/openrport/server/notifications"
	"github.com/openrport/openrport/share/logger"
)

type consumer struct {
	mailer Mailer

	l *logger.Logger
}

//nolint:revive
func NewConsumer(mailer Mailer, l *logger.Logger) *consumer {
	return &consumer{mailer: mailer, l: l}
}

func (c consumer) Process(ctx context.Context, details notifications.NotificationDetails) (string, error) {
	content := details.Data.Content
	if ContentType(details.Data.ContentType) == ContentTypeTextHTML {
		var err error
		content, err = WrapWithTemplate(details.Data.Content)
		if err != nil {
			return "", fmt.Errorf("failed preparing notification to dispatch: %v", err)
		}
	}
	err := c.mailer.Send(ctx, details.Data.Recipients, details.Data.Subject, ContentType(details.Data.ContentType), content)
	if err != nil {
		c.l.Errorf("unable to send smtp message: %s, %v", details.RefID, err)
		return "", err
	}

	c.l.Debugf("sent message: %s", details.RefID)
	return "", nil
}

func (c consumer) Target() notifications.Target {
	return notifications.TargetMail
}

//go:embed mailTemplate.tmpl
var mailTemplate string

func WrapWithTemplate(content string) (string, error) {
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
