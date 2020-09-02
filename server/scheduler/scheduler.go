package scheduler

import (
	"context"
	"time"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type Task interface {
	Run() error
}

// Run runs the given task periodically with a given interval between executions.
func Run(ctx context.Context, log *chshare.Logger, task Task, interval time.Duration) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			start := time.Now()
			log.Debugf("Start to run %T.", task)
			if err := task.Run(); err != nil {
				log.Errorf("Task %T finished with an error: %v.", task, err)
			}
			log.Debugf("Finished to run %T in %v.", task, time.Since(start))
		case <-ctx.Done():
			return
		}
	}
}
