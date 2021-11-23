package session

import (
	"context"
)

type CleanupTask struct {
	c *Cache
}

func NewCleanupTask(c *Cache) *CleanupTask {
	return &CleanupTask{
		c: c,
	}
}

func (t *CleanupTask) Run(ctx context.Context) error {
	return t.c.DeleteExpired(ctx)
}
