// Package contains everything related to Client Session Repository (CSR).
package csr

import (
	"fmt"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type CleanupTask struct {
	log *chshare.Logger
	csr *ClientSessionRepository
}

// NewCleanupTask returns a task to cleanup Client Session Repository from obsolete client sessions.
func NewCleanupTask(log *chshare.Logger, csr *ClientSessionRepository) *CleanupTask {
	return &CleanupTask{
		log: log,
		csr: csr,
	}
}

func (t *CleanupTask) Run() error {
	deleted, err := t.csr.DeleteObsolete()
	if err != nil {
		return fmt.Errorf("failed to delete obsolete client sessions from CSR: %v", err)
	}

	t.log.Infof("Deleted %d obsolete client session(s) from CSR.", len(deleted))
	return nil
}
