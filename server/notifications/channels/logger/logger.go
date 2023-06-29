package logger

import (
	"github.com/realvnc-labs/rport/server/notifications"
	"github.com/realvnc-labs/rport/share/logger"
)

type logConsumer struct {
	logger *logger.Logger
	target notifications.Target
}

//nolint:revive
func NewLogConsumer(logger *logger.Logger, target notifications.Target) *logConsumer {
	return &logConsumer{logger: logger, target: target}
}

func (l logConsumer) Process(details notifications.NotificationDetails) error {
	l.logger.Logf(l.logger.Level, "received notification: %v", details)
	return nil
}

func (l logConsumer) Target() notifications.Target {
	return l.target
}
