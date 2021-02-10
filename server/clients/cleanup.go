// Package contains everything related to Client Repository.
package clients

import (
	"context"
	"fmt"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type CleanupTask struct {
	log *chshare.Logger
	cr  *ClientRepository
	cp  ClientProvider
}

// NewCleanupTask returns a task to cleanup Client Repository from obsolete clients.
func NewCleanupTask(log *chshare.Logger, cr *ClientRepository, cp ClientProvider) *CleanupTask {
	return &CleanupTask{
		log: log,
		cr:  cr,
		cp:  cp,
	}
}

func (t *CleanupTask) Run(ctx context.Context) error {
	deleted, err := t.cr.DeleteObsolete()
	if err != nil {
		return fmt.Errorf("failed to delete obsolete clients from Repository: %v", err)
	}

	if len(deleted) > 0 {
		t.log.Debugf("Deleted %d obsolete client(s) from Repository.", len(deleted))
	}

	return t.cp.DeleteObsolete(ctx)
}
