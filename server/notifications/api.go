package notifications

import (
	"context"

	"github.com/realvnc-labs/rport/share/query"
	"github.com/realvnc-labs/rport/share/refs"
)

type NotificationID refs.Identifiable

type NotificationSummary struct {
	State          ProcessingState `db:"state"`
	NotificationID string          `db:"notification_id"`
}

type REST interface {
	List(ctx context.Context, options *query.ListOptions) (NotificationSummary, error)
	Details(ctx context.Context, nid string) (NotificationDetails, bool, error)
}
