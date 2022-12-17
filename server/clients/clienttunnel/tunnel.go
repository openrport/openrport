package clienttunnel

import (
	"context"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
)

type TunnelProtocol interface {
	Start(ctx context.Context) error
	Terminate(force bool) error
	LastActive() time.Time
}

type MultiProtocolTunnel struct {
	Protocols []TunnelProtocol
}

func (mt *MultiProtocolTunnel) Start(ctx context.Context) error {
	for _, tp := range mt.Protocols {
		err := tp.Start(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (mt *MultiProtocolTunnel) Terminate(force bool) error {
	var result error
	for _, tp := range mt.Protocols {
		err := tp.Terminate(force)
		if err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result
}

func (mt *MultiProtocolTunnel) LastActive() time.Time {
	var result time.Time
	for _, tp := range mt.Protocols {
		v := tp.LastActive()
		if v.After(result) {
			result = v
		}
	}
	return result
}

// TODO(m-terel): Refactor to use separate models for representation and business logic.
// Tunnel represents active remote proxy connection
type Tunnel struct {
	ID string `json:"id"`

	models.Remote

	TunnelProtocol      `json:"-"`
	InternalTunnelProxy *InternalTunnelProxy `json:"-"`
	CreatedAt           time.Time            `json:"created_at"`
}

func NewTunnel(logger *logger.Logger, ssh ssh.Conn, id string, remote models.Remote, acl *TunnelACL) (*Tunnel, error) {
	logger = logger.Fork("tunnel#%s:%s", id, remote)
	logger.Debugf("new tunnel with remote = %#v", remote)

	var tunnelProtocol TunnelProtocol
	switch remote.Protocol {
	case models.ProtocolUDP:
		tunnelProtocol = newTunnelUDP(logger, ssh, remote, acl)
	case models.ProtocolTCP:
		tunnelProtocol = newTunnelTCP(logger, ssh, remote, acl)
	case models.ProtocolTCPUDP:
		tunnelProtocol = &MultiProtocolTunnel{
			Protocols: []TunnelProtocol{
				newTunnelTCP(logger, ssh, remote, acl),
				newTunnelUDP(logger, ssh, remote, acl),
			},
		}
	default:
		return nil, errors.Errorf("unsupported protocol %q", remote.Protocol)
	}

	return &Tunnel{
		Remote:         remote,
		ID:             id,
		TunnelProtocol: tunnelProtocol,
		CreatedAt:      time.Now(),
	}, nil
}
