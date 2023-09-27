// Package contains everything related to Client Repository.
package clients

import (
	"context"
	"fmt"

	"github.com/openrport/openrport/share/logger"
)

type CleanupTask struct {
	log *logger.Logger
	cr  *ClientRepository
}

// NewCleanupTask returns a task to cleanup Client Repository from obsolete clients.
func NewCleanupTask(log *logger.Logger, cr *ClientRepository) *CleanupTask {
	return &CleanupTask{
		log: log,
		cr:  cr,
	}
}

func (t *CleanupTask) Run(ctx context.Context) error {
	deleted, err := t.cr.DeleteObsolete()
	if err != nil {
		return fmt.Errorf("failed to delete obsolete clients: %v", err)
	}

	if len(deleted) > 0 {
		t.log.Debugf("Deleted %d obsolete client(s).", len(deleted))
	}

	return nil
}
