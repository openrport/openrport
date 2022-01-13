package clients

import (
	"net/http"
)

type TunnelConnector interface {
	Start()
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	Stop()
}

func NewTunnelConnector(tp *TunnelProxy) TunnelConnector {
	switch *tp.Tunnel.Scheme {
	case "http":
		return NewTunnelConnectorHTTP(tp)
	case "vnc":
		return NewTunnelConnectorVNC(tp)
	default:
		return nil
	}
}
