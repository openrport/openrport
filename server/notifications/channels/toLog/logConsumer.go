package toLog

import (
	"context"

	"github.com/openrport/openrport/server/notifications"
	"github.com/openrport/openrport/share/logger"
)

type logConsumer struct {
	logger *logger.Logger
	target notifications.Target
}

//nolint:revive
func NewLogConsumer(logger *logger.Logger, target notifications.Target) *logConsumer {
	return &logConsumer{logger: logger, target: target}
}

func (l logConsumer) Process(ctx context.Context, details notifications.NotificationDetails) (string, error) {
	l.logger.Logf(l.logger.Level, "received notification: %v", details)
	return "", nil
}

func (l logConsumer) Target() notifications.Target {
	return l.target
}
