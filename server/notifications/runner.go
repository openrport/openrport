package notifications

import (
	"context"
	"sync"
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
	UpdatesFor(target Target) chan NotificationDetails
	Save(ctx context.Context, details NotificationDetails) error
}

type processor struct {
	store       Store
	done        context.CancelFunc
	waitForDead context.Context
	timeToDie   context.Context
	killMe      context.CancelFunc
	consumers   []Consumer
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
	updates := p.store.UpdatesFor(consumer.Target())
root:
	for {
		select {
		case <-p.timeToDie.Done():
			break root
		case notification := <-updates:
			notification.State = ProcessingStateRunning
			// TODO: should rport crush when save errors?
			p.store.Save(context.Background(), notification)
			err := consumer.Process(notification)
			if err == nil {
				notification.State = ProcessingStateDone
			} else {
				notification.State = ProcessingStateError
				notification.Out = err.Error()
			}
			p.store.Save(context.Background(), notification)
		}
	}
}

func (p *processor) Close() error {
	p.killMe()
	<-p.waitForDead.Done()
	return nil
}

func NewProcessor(store Store, consumers ...Consumer) Processor {
	ctx := context.Background()
	waitForDead, cancel := context.WithCancel(ctx)
	timeToDie, killMe := context.WithCancel(ctx)

	p := &processor{
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
