package notifications

import (
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

const NotificationType refs.IdentifiableType = "notification"
const ErrorNotificationType refs.IdentifiableType = "error-notification"
