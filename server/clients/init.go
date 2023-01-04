package clients

import (
	"context"
	"fmt"

	"github.com/cloudradar-monitoring/rport/share/logger"
)

// LoadInitialClients returns an initial Client Repository state populated with clients from the internal storage.
func LoadInitialClients(ctx context.Context, p ClientStore, logger *logger.Logger) ([]*Client, error) {
	if logger != nil {
		logger.Debugf("loading existing clients")
	}
	all, err := p.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get clients: %v", err)
	}

	if logger != nil {
		logger.Debugf("loaded %d clients", len(all))
	}

	// mark previously connected clients as disconnected with current time
	now := now()
	for _, cur := range all {
		if cur.DisconnectedAt == nil {
			cur.SetDisconnected(&now)
			err := p.Save(ctx, cur)
			if err != nil {
				return nil, fmt.Errorf("failed to save client: %v", err)
			}
		}
	}

	return all, nil
}
