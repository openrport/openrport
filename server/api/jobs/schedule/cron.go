package schedule

import (
	"context"
	"sync"

	cron "github.com/robfig/cron/v3"
)

type CronImplementation struct {
	mtx        sync.Mutex
	cronParser cron.ScheduleParser
	cron       *cron.Cron
	mapping    map[string]cron.EntryID
}

func newCron() *CronImplementation {
	c := &CronImplementation{
		cronParser: cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor),
		cron:       cron.New(cron.WithChain(cron.Recover(cron.DefaultLogger))),
		mapping:    make(map[string]cron.EntryID),
	}
	c.cron.Start()
	return c
}

func (c *CronImplementation) Validate(schedule string) error {
	_, err := c.cronParser.Parse(schedule)
	return err
}

func (c *CronImplementation) Add(id string, schedule string, f func(context.Context, string)) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	sch, err := c.cronParser.Parse(schedule)
	if err != nil {
		return err
	}

	entryID := c.cron.Schedule(sch, cron.FuncJob(func() {
		f(context.Background(), id)
	}))

	c.mapping[id] = entryID

	return nil
}

func (c *CronImplementation) Remove(id string) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	entryID := c.mapping[id]
	c.cron.Remove(entryID)
	delete(c.mapping, id)
}
