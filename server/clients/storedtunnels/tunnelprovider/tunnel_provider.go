package tunnelprovider

import (
	"context"
	"fmt"

	"github.com/realvnc-labs/rport/server/clients/storedtunnels"
	"github.com/realvnc-labs/rport/share/query"
)

type TunnelKV interface {
	GetAll(context.Context) ([]storedtunnels.StoredTunnel, error)
	Save(context.Context, string, storedtunnels.StoredTunnel) error
	Delete(context.Context, string) error
	Filter(ctx context.Context, options query.ListOptions) ([]storedtunnels.StoredTunnel, error)
}

type TunnelProvider struct {
	kv TunnelKV
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
	newOps := *options
	newOps.Filters = append([]query.FilterOption{{
		Column:                []string{"ClientID"},
		Operator:              query.FilterOperatorTypeEQ,
		Values:                []string{clientID},
		ValuesLogicalOperator: query.FilterLogicalOperatorTypeAND,
	}}, newOps.Filters...)

	tunnels, err := t.kv.Filter(ctx, newOps)
	if err != nil {
		return nil, err
	}

	tmp := make([]*storedtunnels.StoredTunnel, len(tunnels))
	for k, v := range tunnels {
		tmp[k] = &v
	}

	return tmp, err
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
