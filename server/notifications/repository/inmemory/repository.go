package inmemory

import (
	"context"

	"github.com/realvnc-labs/rport/server/notifications"
)

type Notification struct {
}

type NotificationRepository interface {
	Save(ctx context.Context, notification Notification) error
	List(ctx context.Context) ([]notifications.NotificationSummary, error) // TODO: need to add query params
	Details(ctx context.Context, notificationID notifications.NotificationID) (Notification, error)
}
