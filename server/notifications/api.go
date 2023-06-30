package notifications

import (
	"github.com/realvnc-labs/rport/share/refs"
)

type NotificationID refs.Identifiable

type NotificationSummary struct {
	State          ProcessingState `db:"state"`
	NotificationID string          `db:"notification_id"`
	Transport      string          `db:"transport"`
}
