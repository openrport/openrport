// Package contains everything related to Client Session Repository (CSR).
package sessions

import (
	"fmt"

	"github.com/cloudradar-monitoring/rport/server/clients"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

type CleanupTask struct {
	log             *chshare.Logger
	csr             *ClientSessionRepository
	lockableClients *clients.ClientCache
}

// NewCleanupTask returns a task to cleanup Client Session Repository from obsolete client sessions.
func NewCleanupTask(log *chshare.Logger, csr *ClientSessionRepository, lockableClients *clients.ClientCache) *CleanupTask {
	return &CleanupTask{
		log:             log,
		csr:             csr,
		lockableClients: lockableClients,
	}
}

func (t *CleanupTask) Run() error {
	deleted, err := t.csr.DeleteObsolete()
	if err != nil {
		return fmt.Errorf("failed to delete obsolete client sessions from CSR: %v", err)
	}

	if len(deleted) > 0 {
		t.log.Debugf("Deleted %d obsolete client session(s) from CSR.", len(deleted))
	}

	// unlock clients that were locked by disconnected sessions
	if t.lockableClients != nil {
		sessionClientPairs := make(map[string]string)
		for _, s := range deleted {
			if s.ClientID != nil {
				sessionClientPairs[s.ID] = *s.ClientID
			}
		}

		if len(sessionClientPairs) > 0 {
			unlockedClientIDs := t.lockableClients.UnlockClientsLockedBySIDs(sessionClientPairs)
			if len(unlockedClientIDs) > 0 {
				t.log.Debugf("Unlocked %d client credentials: %v.", len(unlockedClientIDs), unlockedClientIDs)
			}
		}
	}

	return nil
}
