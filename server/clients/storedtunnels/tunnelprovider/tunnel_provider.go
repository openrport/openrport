package tunnelprovider

import (
	"context"
	"fmt"

	"github.com/realvnc-labs/rport/server/clients/storedtunnels"
	"github.com/realvnc-labs/rport/share/dynops"
	"github.com/realvnc-labs/rport/share/dynops/dyncopy"
	"github.com/realvnc-labs/rport/share/dynops/filterer"
	"github.com/realvnc-labs/rport/share/query"
	"github.com/realvnc-labs/rport/share/simpleops"
)

type TunnelKV interface {
	GetAll(context.Context) ([]storedtunnels.StoredTunnel, error)
	Save(context.Context, string, storedtunnels.StoredTunnel) error
	Delete(context.Context, string) error
	Filter(ctx context.Context, sieve func(tunnel storedtunnels.StoredTunnel) bool) ([]storedtunnels.StoredTunnel, error)
}

type TunnelProvider struct {
	kv                   TunnelKV
	sortTranslationTable map[string]dyncopy.Field
}

func NewTunnelProvider(kv TunnelKV) *TunnelProvider {
	return &TunnelProvider{
		kv:                   kv,
		sortTranslationTable: dyncopy.BuildTranslationTable(storedtunnels.StoredTunnel{})}
}

func (t TunnelProvider) Delete(ctx context.Context, clientID string, tunnelID string) error {
	return t.kv.Delete(ctx, t.genKey(clientID, tunnelID))
}

func (t TunnelProvider) Insert(ctx context.Context, tunnel *storedtunnels.StoredTunnel) error {
	return t.Update(ctx, tunnel)
}

func (t TunnelProvider) Update(ctx context.Context, tunnel *storedtunnels.StoredTunnel) error {
	return t.kv.Save(ctx, t.genKey(tunnel.ClientID, tunnel.ID), *tunnel)
}

func (t TunnelProvider) List(ctx context.Context, clientID string, options *query.ListOptions) ([]*storedtunnels.StoredTunnel, error) {
	if options == nil {
		return nil, fmt.Errorf("options for filtering and pagination is nil")
	}

	filter, err := filterer.CompileFromQueryListOptions[storedtunnels.StoredTunnel](options.Filters)
	if err != nil {
		return nil, err
	}

	tunnels, err := t.kv.Filter(ctx, func(tunnel storedtunnels.StoredTunnel) bool {
		return tunnel.ClientID == clientID && filter.Run(tunnel)
	})
	if err != nil {
		return nil, err
	}

	tunnels, err = dynops.FastSorter1(t.sortTranslationTable, tunnels, options.Sorts)
	if err != nil {
		return nil, err
	}

	tunnels = dynops.Paginator(tunnels, options.Pagination)

	return simpleops.ToPointerSlice(tunnels), nil
}

func (t TunnelProvider) Count(ctx context.Context, clientID string, options *query.ListOptions) (int, error) {
	list, err := t.List(ctx, clientID, options)
	if err != nil {
		return 0, err
	}

	return len(list), err
}

func (t TunnelProvider) genKey(clientID string, tunnelID string) string {
	return fmt.Sprintf("clientid_%v_tunnelid_%v", clientID, tunnelID)
}
