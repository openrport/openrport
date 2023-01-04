package scheduler

import (
	"context"
	"time"

	"github.com/cloudradar-monitoring/rport/share/logger"
)

type Task interface {
	Run(ctx context.Context) error
}

// Run runs the given task periodically with a given interval between executions.
func Run(ctx context.Context, log *logger.Logger, task Task, interval time.Duration) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if err := task.Run(ctx); err != nil {
				log.Errorf("Task %T finished with an error: %v.", task, err)
			}
		case <-ctx.Done():
			log.Debugf("%T: context canceled", task)
			return
		}
	}
}
