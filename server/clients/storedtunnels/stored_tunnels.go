package storedtunnels

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/openrport/openrport/server/api"
	"github.com/openrport/openrport/share/query"
	"github.com/openrport/openrport/share/random"
)

var (
	supportedFilters = map[string]bool{
		"name":        true,
		"scheme":      true,
		"remote_ip":   true,
		"remote_port": true,
	}
	supportedSorts = map[string]bool{
		"created_at":  true,
		"name":        true,
		"scheme":      true,
		"remote_ip":   true,
		"remote_port": true,
	}
)

type Provider interface {
	Delete(context.Context, string, string) error
	Insert(context.Context, *StoredTunnel) error
	Update(context.Context, *StoredTunnel) error
	List(context.Context, string, *query.ListOptions) ([]*StoredTunnel, error)
	Count(context.Context, string, *query.ListOptions) (int, error)
}

type Manager struct {
	provider Provider
}

func New(db *sqlx.DB) *Manager {
	return &Manager{
		provider: newSQLiteProvider(db),
	}
}

func (m *Manager) List(ctx context.Context, options *query.ListOptions, clientID string) (*api.SuccessPayload, error) {

	err := query.ValidateListOptions(options, supportedSorts, supportedFilters, nil, &query.PaginationConfig{
		DefaultLimit: 10,
		MaxLimit:     100,
	})
	if err != nil {
		return nil, err
	}

	entries, err := m.provider.List(ctx, clientID, options)
	if err != nil {
		return nil, err
	}

	count, err := m.provider.Count(ctx, clientID, options)
	if err != nil {
		return nil, err
	}

	return &api.SuccessPayload{
		Data: entries,
		Meta: api.NewMeta(count),
	}, nil
}

func (m *Manager) Create(ctx context.Context, clientID string, t *StoredTunnel) (*StoredTunnel, error) {
	id, err := random.UUID4()
	if err != nil {
		return nil, err
	}
	t.ID = id
	t.CreatedAt = time.Now()
	t.ClientID = clientID

	err = m.provider.Insert(ctx, t)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (m *Manager) Update(ctx context.Context, clientID string, t *StoredTunnel) (*StoredTunnel, error) {
	t.ClientID = clientID

	err := m.provider.Update(ctx, t)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (m *Manager) Delete(ctx context.Context, clientID, id string) error {
	return m.provider.Delete(ctx, clientID, id)
}
