package notifications

import (
	"context"

	"github.com/realvnc-labs/rport/share/refs"
)

type ProcessingState string

const ProcessingStateQueued ProcessingState = "queued"
const ProcessingStateRunning ProcessingState = "running"
const ProcessingStateDone ProcessingState = "done"
const ProcessingStateError ProcessingState = "error"

type Dispatcher interface {
	Dispatch(ctx context.Context, origin refs.Origin, notification NotificationData) (refs.Identifiable, error)
}

type store interface {
	Create(ctx context.Context, details NotificationDetails) error
}

type dispatcher struct {
	store store
}

func (f dispatcher) Dispatch(ctx context.Context, origin refs.Origin, notification NotificationData) (refs.Identifiable, error) {

	details := NotificationDetails{
		Data:   notification,
		State:  ProcessingStateQueued,
		Origin: origin,
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
	Origin refs.Origin
	Data   NotificationData
	State  ProcessingState
	ID     refs.Identifiable
	Out    string
	Target Target
}
