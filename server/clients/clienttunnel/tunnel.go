package clienttunnel

import (
	"context"

	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
)

type TunnelProtocol interface {
	Start(ctx context.Context) (autoCloseChan chan bool, err error)
	Terminate(force bool) error
}

// TODO(m-terel): Refactor to use separate models for representation and business logic.
// Tunnel represents active remote proxy connection
type Tunnel struct {
	ID string `json:"id"`
	models.Remote

	TunnelProtocol `json:"-"`
	Proxy          *TunnelProxy `json:"-"`
}

func NewTunnel(logger *logger.Logger, ssh ssh.Conn, id string, remote models.Remote, acl *TunnelACL) *Tunnel {
	logger = logger.Fork("tunnel#%s:%s", id, remote)

	var tunnelProtocol TunnelProtocol
	switch remote.Protocol {
	case models.ProtocolUDP:
		tunnelProtocol = newTunnelUDP(logger, ssh, remote, acl)
	default:
		tunnelProtocol = newTunnelTCP(logger, ssh, remote, acl)
	}

	return &Tunnel{
		Remote:         remote,
		ID:             id,
		TunnelProtocol: tunnelProtocol,
	}
}
