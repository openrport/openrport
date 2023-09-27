package monitoring

import (
	"context"
	"fmt"
	"time"

	"github.com/openrport/openrport/share/logger"
)

type CleanupTask struct {
	log      *logger.Logger
	service  Service
	duration time.Duration
}

// NewCleanupTask returns a task to cleanup monitoring data after configured period
func NewCleanupTask(log *logger.Logger, service Service, duration time.Duration) *CleanupTask {
	return &CleanupTask{
		log:      log,
		service:  service,
		duration: duration,
	}
}

func (t *CleanupTask) Run(ctx context.Context) error {
	deletedRecords, err := t.service.DeleteMeasurementsOlderThan(ctx, t.duration)
	if err != nil {
		return fmt.Errorf("failed to cleanup measurements: %v", err)
	}
	t.log.Debugf("monitoring.CleanupTask: %d measurement records deleted", deletedRecords)
	return nil
}
