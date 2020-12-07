package sessions

import (
	"context"
	"fmt"
)

// GetInitState returns an initial Client Session Repository state populated with sessions from the internal storage.
func GetInitState(ctx context.Context, p ClientSessionProvider) ([]*ClientSession, error) {
	all, err := p.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %v", err)
	}

	// mark previously connected client sessions as disconnected with current time
	now := now()
	for _, cur := range all {
		if cur.Disconnected == nil {
			cur.Disconnected = &now
			err := p.Save(ctx, cur)
			if err != nil {
				return nil, fmt.Errorf("failed to save session: %v", err)
			}
		}
	}

	return all, nil
}
