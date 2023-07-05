package sqlite

import (
	"context"
	"time"

	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/random"
)

type Closeable interface {
	Close() error
}

type cleaner struct {
	closer  chan struct{}
	logger  *logger.Logger
	keepFor time.Duration
	repo    repository
}

func (c cleaner) Close() error {
	close(c.closer)
	return nil
}

func StartCleaner(logger *logger.Logger, r repository, keepFor time.Duration, checkEvery time.Duration) Closeable {
	c := cleaner{
		closer:  make(chan struct{}),
		logger:  logger,
		keepFor: keepFor,
		repo:    r,
	}
	jam := random.AlphaNum(5)
	logger.Infof("started notifications cleaner id:", jam)
	go func() {
		c.cleanOld()
		for {
			select {
			case <-time.After(checkEvery):
				logger.Infof("cleaning notifications id:", jam)
				c.cleanOld()
			case <-c.closer:
				logger.Infof("closed notifications cleaner id:", jam)
				return
			}
		}
	}()

	return c
}

func (c cleaner) cleanOld() {
	before := time.Now().Add(-c.keepFor).UTC()
	c.logger.Infof("cleaning ", before.Format("2006-01-02 15:04:05"))
	ctx := context.Background()
	_, err := c.repo.db.ExecContext(
		ctx,
		"DELETE FROM `notifications_log` WHERE timestamp <= ?",
		before.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		c.logger.Errorf("cleaning notifications failed: %v", err)
	}
}

// time.Now().UTC().Add(-time.Second)
