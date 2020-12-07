// Package contains everything related to Client Session Repository (CSR).
package sessions

import (
	"context"
	"fmt"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type CleanupTask struct {
	log *chshare.Logger
	csr *ClientSessionRepository
	csp ClientSessionProvider
}

// NewCleanupTask returns a task to cleanup Client Session Repository from obsolete client sessions.
func NewCleanupTask(log *chshare.Logger, csr *ClientSessionRepository, csp ClientSessionProvider) *CleanupTask {
	return &CleanupTask{
		log: log,
		csr: csr,
		csp: csp,
	}
}

func (t *CleanupTask) Run(ctx context.Context) error {
	deleted, err := t.csr.DeleteObsolete()
	if err != nil {
		return fmt.Errorf("failed to delete obsolete client sessions from CSR: %v", err)
	}

	if len(deleted) > 0 {
		t.log.Debugf("Deleted %d obsolete client session(s) from CSR.", len(deleted))
	}

	return t.csp.DeleteObsolete(ctx)
}
