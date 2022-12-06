package clienttunnel

import (
	"github.com/gorilla/mux"
)

// TunnelProxyConnector connects the tunnel proxy http server with the tunnel behind
type TunnelProxyConnector interface {
	InitRouter(router *mux.Router) *mux.Router
}

func NewTunnelProxyConnector(tp *InternalTunnelProxy) TunnelProxyConnector {
	switch *tp.Tunnel.Remote.Scheme {
	case "http", "https":
		return NewTunnelConnectorHTTP(tp)
	case "vnc":
		return NewTunnelConnectorVNC(tp)
	case "rdp":
		return NewTunnelConnectorRDP(tp)
	}

	return nil
}
