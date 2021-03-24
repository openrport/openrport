package clients

import (
	"context"
	"fmt"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type SaveTask struct {
	log *chshare.Logger
	cr  *ClientRepository
	cp  ClientProvider
}

// NewSaveTask returns a task to save Client Repository to an internal storage.
func NewSaveTask(log *chshare.Logger, cr *ClientRepository, cp ClientProvider) *SaveTask {
	return &SaveTask{
		log: log,
		cr:  cr,
		cp:  cp,
	}
}

func (t *SaveTask) Run(ctx context.Context) error {
	clients, err := t.cr.GetAll()
	if err != nil {
		return fmt.Errorf("failed to get clients from Repository: %v", err)
	}

	for _, cur := range clients {
		err := t.cp.Save(ctx, cur)
		if err != nil {
			t.log.Errorf("failed to save client: %v", err)
		}
	}
	return nil
}
