package notifications

import (
	"context"

	"github.com/openrport/openrport/share/refs"
)

type ProcessingState string

const ProcessingStateQueued ProcessingState = "queued"
const ProcessingStateDispatching ProcessingState = "dispatching"
const ProcessingStateDone ProcessingState = "done"
const ProcessingStateError ProcessingState = "error"

type Dispatcher interface {
	Dispatch(ctx context.Context, refID refs.Identifiable, notification NotificationData) (refs.Identifiable, error)
}

type store interface {
	Create(ctx context.Context, details NotificationDetails) error
}

type dispatcher struct {
	store store
}

func (f dispatcher) Dispatch(ctx context.Context, refID refs.Identifiable, notification NotificationData) (refs.Identifiable, error) {

	if err := notification.ContentType.Valid(); err != nil {
		return nil, err
	}

	details := NotificationDetails{
		Data:   notification,
		State:  ProcessingStateQueued,
		RefID:  refID,
		ID:     refs.GenerateIdentifiable(NotificationType),
		Target: FigureOutTarget(notification.Target),
	}

	return details.ID, f.store.Create(ctx, details)
}

func FigureOutTarget(target string) Target {
	switch target {
	case "smtp":
		return TargetMail
	default:
		return TargetScript
	}
}

//nolint:revive
func NewDispatcher(repository store) dispatcher {
	return dispatcher{
		store: repository,
	}
}

type NotificationDetails struct {
	RefID  refs.Identifiable
	Data   NotificationData
	State  ProcessingState
	ID     refs.Identifiable
	Out    string
	Target Target
	Err    string
}
