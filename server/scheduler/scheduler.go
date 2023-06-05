package scheduler

import (
	"context"
	"time"

	"github.com/realvnc-labs/rport/share/logger"
)

type Task interface {
	Run(ctx context.Context) error
}

// Run runs the given task periodically with a given interval between executions.
func Run(ctx context.Context, log *logger.Logger, task Task, interval time.Duration) {
	log.Debugf("started")
	tick := time.NewTicker(interval)
	for {
		select {
		case <-tick.C:
			log.Debugf("running")
			if err := task.Run(ctx); err != nil {
				log.Errorf("finished with an error: %v.", err)
			}
			log.Debugf("finished")
		case <-ctx.Done():
			tick.Stop()
			log.Debugf("context canceled")
			log.Debugf("stopped")
			return
		}
	}
}
