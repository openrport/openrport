package notifications

import (
	"context"
	"sync"

	"github.com/realvnc-labs/rport/share/logger"
)

type Processor interface {
	Close() error
}

type Consumer interface {
	Process(details NotificationDetails) error
	Target() Target
}

type Target string

const TargetMail Target = "smtp"
const TargetScript Target = "script"

var AllTargets = []Target{TargetMail, TargetScript}

func (t Target) Valid() bool {
	for _, target := range AllTargets {
		if t == target {
			return true
		}
	}
	return false
}

type Store interface {
	Create(ctx context.Context, details NotificationDetails) error
	SetDispatching(ctx context.Context, nid string) error
	SetDone(ctx context.Context, nid string) error
	SetError(ctx context.Context, nid string, error string) error
	NotificationStream(target Target) chan NotificationDetails
	Close() error
}

type processor struct {
	store       Store
	done        context.CancelFunc
	waitForDead context.Context
	timeToDie   context.Context
	killMe      context.CancelFunc
	consumers   []Consumer
	logger      *logger.Logger
}

func (p *processor) start() {
	w := sync.WaitGroup{}
	w.Add(len(p.consumers))
	for _, c := range p.consumers {
		go func(consumer Consumer) {
			p.startConsumer(consumer)
			w.Done()
		}(c)
	}
	w.Wait()
	p.done()
}

func (p *processor) startConsumer(consumer Consumer) {
	updates := p.store.NotificationStream(consumer.Target())
root:
	for {
		select {
		case <-p.timeToDie.Done():
			break root
		case notification := <-updates:
			// TODO: (rs): what about the primary rport ctx, used for shutdown?
			err := p.store.SetDispatching(context.Background(), notification.ID.ID())
			if err != nil {
				p.logger.Errorf("failed updating state: %v", err)
				continue root
			}

			err = consumer.Process(notification)
			if err != nil {
				p.logger.Errorf("failed processing notification: %v", err)
				err = p.store.SetError(context.Background(), notification.ID.ID(), err.Error())
				if err != nil {
					p.logger.Errorf("failed updating state: %v", err)
				}
				continue root
			}

			err = p.store.SetDone(context.Background(), notification.ID.ID())
			if err != nil {
				p.logger.Errorf("failed updating state: %v", err)
				continue root
			}
		}
	}
}

func (p *processor) Close() error {
	p.killMe()
	<-p.waitForDead.Done()
	return nil
}

func NewProcessor(logger *logger.Logger, store Store, consumers ...Consumer) Processor {
	ctx := context.Background()
	waitForDead, cancel := context.WithCancel(ctx)
	timeToDie, killMe := context.WithCancel(ctx)

	p := &processor{
		logger:      logger,
		consumers:   consumers,
		store:       store,
		waitForDead: waitForDead,
		done:        cancel,
		timeToDie:   timeToDie,
		killMe:      killMe,
	}
	go p.start()
	return p
}
