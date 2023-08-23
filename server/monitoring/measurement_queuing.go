package monitoring

import (
	"context"
	"sync/atomic"

	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/models"
)

type saver interface {
	SaveMeasurement(ctx context.Context, measurement *models.Measurement) error
}

type MeasurementSaver interface {
	Notify(models.Measurement)
	Close() error
}

type queue struct {
	saver    saver
	queue    chan models.Measurement
	closed   atomic.Bool
	ctx      context.Context
	cancelFn context.CancelFunc
	done     chan struct{}
	logger   *logger.Logger
}

func (q *queue) Close() error {
	q.closed.Store(true)
	q.cancelFn()
	close(q.queue)
	<-q.done
	return nil
}

func (q *queue) Notify(measurement models.Measurement) {
	if q.closed.Load() {
		return
	}
	q.queue <- measurement
}

func (q *queue) process() {
	for m := range q.queue {
		if !q.closed.Load() {
			err := q.saver.SaveMeasurement(q.ctx, &m)
			if err != nil {
				q.logger.Errorf("Failed to save measurement for client %s: %s", m.ClientID, err)
				continue
			}
		}
	}
	close(q.done)
}

func NewMeasurementQueuing(logger *logger.Logger, saver saver, queueSize int) MeasurementSaver {
	ctx, cfn := context.WithCancel(context.Background())
	q := queue{
		saver:    saver,
		queue:    make(chan models.Measurement, queueSize),
		ctx:      ctx,
		cancelFn: cfn,
		done:     make(chan struct{}),
		logger:   logger,
	}
	go q.process()
	return &q
}
