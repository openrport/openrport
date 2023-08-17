package notifications

import (
	"fmt"

	"github.com/realvnc-labs/rport/share/refs"
)

// ContentType represents a content type for the Msg
type ContentType string

func (t ContentType) Valid() error {
	switch t {
	case ContentTypeTextHTML:
		return nil
	case ContentTypeTextPlain:
		return nil
	case ContentTypeTextJSON:
		return nil
	default:
		return fmt.Errorf("bad content type: %v", t)
	}

}

const (
	ContentTypeTextPlain ContentType = "text/plain"
	ContentTypeTextHTML  ContentType = "text/html"
	ContentTypeTextJSON  ContentType = "text/json"
)

type NotificationData struct {
	Target      string      `json:"target"`
	Recipients  []string    `json:"recipients"`
	Subject     string      `json:"subject"`
	Content     string      `json:"content"`
	ContentType ContentType `json:"content_type"`
}

const NotificationType refs.IdentifiableType = "notification"
