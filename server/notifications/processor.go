package notifications

import (
	"context"
	"sync"
	"time"

	"github.com/realvnc-labs/rport/share/logger"
)

type Processor interface {
	Close() error
}

type Consumer interface {
	Process(ctx context.Context, details NotificationDetails) error
	Target() Target
}

const MaxProcessingTime = time.Second * 10

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
	SetDone(ctx context.Context, details NotificationDetails) error
	SetError(ctx context.Context, details NotificationDetails, out string) error
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
		case notification, ok := <-updates:
			if !ok {
				break root
			}
			ctx, cancelFn := context.WithTimeout(context.Background(), MaxProcessingTime)
			p.logger.Infof("notification %v(%v)  started processing", notification.Target, notification.ID)
			err := consumer.Process(ctx, notification)
			cancelFn()

			if err == nil {
				p.logger.Infof("notification %v(%v) done", notification.Target, notification.ID)
				err = p.store.SetDone(context.Background(), notification)
			} else {
				p.logger.Infof("notification %v(%v) error", notification.Target, notification.ID)
				err = p.store.SetError(context.Background(), notification, err.Error())
			}
			if err != nil {
				p.logger.Errorf("failed updating state: %v", err)
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
