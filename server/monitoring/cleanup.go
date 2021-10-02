package monitoring

import (
	"context"
	"fmt"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

type CleanupTask struct {
	log     *chshare.Logger
	service Service
	days    int64
}

// NewCleanupTask returns a task to cleanup monitoring data after configured period
func NewCleanupTask(log *chshare.Logger, service Service, days int64) *CleanupTask {
	return &CleanupTask{
		log:     log,
		service: service,
		days:    days,
	}
}

func (t *CleanupTask) Run(ctx context.Context) error {
	err := t.service.Cleanup(ctx, 30)
	if err != nil {
		return fmt.Errorf("failed to cleanup measurements: %v", err)
	}

	return nil
}
