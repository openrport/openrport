package inmemory

import (
	"context"

	"github.com/realvnc-labs/rport/server/notifications"
	"github.com/realvnc-labs/rport/share/refs"
)

type Notification struct {
}

type NotificationID refs.Identifiable

type NotificationSummary struct {
	State notifications.ProcessingState
}

type NotificationRepository interface {
	Save(ctx context.Context, notification Notification) error
	List(ctx context.Context) ([]NotificationSummary, error) // TODO: need to add query params
	Details(ctx context.Context, notificationID NotificationID) (Notification, error)
}
