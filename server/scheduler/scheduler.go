package scheduler

import (
	"context"
	"time"

	"github.com/openrport/openrport/share/logger"
)

type Task interface {
	Run(ctx context.Context) error
}

// Run runs the given task periodically with a given interval between executions.
func Run(ctx context.Context, log *logger.Logger, task Task, interval time.Duration) {
	log.Debugf("task running")
	tick := time.NewTicker(interval)
	for {
		select {
		case <-tick.C:
			log.Debugf("task started")
			if err := task.Run(ctx); err != nil {
				log.Errorf("finished with an error: %v.", err)
			}
			log.Debugf("task finished")
		case <-ctx.Done():
			tick.Stop()
			log.Debugf("context canceled")
			log.Debugf("task stopped")
			return
		}
	}
}
