package clients

import (
	"github.com/gorilla/mux"
)

//TunnelProxyConnector connects the tunnel proxy http server with the tunnel behind
type TunnelProxyConnector interface {
	InitRouter(router *mux.Router) *mux.Router
	DisableHTTP2() bool
}

func NewTunnelProxyConnector(tp *TunnelProxy) TunnelProxyConnector {
	switch *tp.Tunnel.Remote.Scheme {
	case "http", "https":
		return NewTunnelConnectorHTTP(tp)
	case "vnc":
		return NewTunnelConnectorVNC(tp)
	}

	return nil
}
