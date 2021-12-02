package monitoring

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudradar-monitoring/rport/share/logger"
)

type CleanupTask struct {
	log     *logger.Logger
	service Service
	days    time.Duration
}

// NewCleanupTask returns a task to cleanup monitoring data after configured period
func NewCleanupTask(log *logger.Logger, service Service, days time.Duration) *CleanupTask {
	return &CleanupTask{
		log:     log,
		service: service,
		days:    days,
	}
}

func (t *CleanupTask) Run(ctx context.Context) error {
	deletedRecords, err := t.service.DeleteMeasurementsOlderThan(ctx, t.days)
	if err != nil {
		return fmt.Errorf("failed to cleanup measurements: %v", err)
	}
	t.log.Debugf("monitoring.CleanupTask: %d measurement records deleted", deletedRecords)
	return nil
}
