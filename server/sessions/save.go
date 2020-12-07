package sessions

import (
	"context"
	"fmt"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type SaveTask struct {
	log *chshare.Logger
	csr *ClientSessionRepository
	csp ClientSessionProvider
}

// NewSaveTask returns a task to save Client Session Repository to an internal storage.
func NewSaveTask(log *chshare.Logger, csr *ClientSessionRepository, csp ClientSessionProvider) *SaveTask {
	return &SaveTask{
		log: log,
		csr: csr,
		csp: csp,
	}
}

func (t *SaveTask) Run(ctx context.Context) error {
	sessions, err := t.csr.GetAll()
	if err != nil {
		return fmt.Errorf("failed to get client sessions from CSR: %v", err)
	}
	t.log.Debugf("Got %d client sessions from CSR. Writing...", len(sessions))

	for _, cur := range sessions {
		err := t.csp.Save(ctx, cur)
		if err != nil {
			t.log.Errorf("failed to save session: %v", err)
		}
	}
	return nil
}
