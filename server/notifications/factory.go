package notifications

import (
	"context"

	"github.com/realvnc-labs/rport/share/refs"
)

type Factory interface {
	Dispatch(ctx context.Context, origin refs.Origin, notification NotificationData) (refs.Identifiable, error)
}

type store interface {
	Save(ctx context.Context, details NotificationDetails) error
}

type factory struct {
	store store
}

func (f factory) Dispatch(ctx context.Context, origin refs.Origin, notification NotificationData) (refs.Identifiable, error) {
	return nil, f.store.Save(ctx, NotificationDetails{})
}

func NewFactory(repository store) factory {
	return factory{
		store: repository,
	}
}

type NotificationDetails struct {
	Origin string
}
