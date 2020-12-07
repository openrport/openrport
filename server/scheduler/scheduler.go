package scheduler

import (
	"context"
	"time"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type Task interface {
	Run(ctx context.Context) error
}

// Run runs the given task periodically with a given interval between executions.
func Run(ctx context.Context, log *chshare.Logger, task Task, interval time.Duration) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if err := task.Run(ctx); err != nil {
				log.Errorf("Task %T finished with an error: %v.", task, err)
			}
		case <-ctx.Done():
			return
		}
	}
}
