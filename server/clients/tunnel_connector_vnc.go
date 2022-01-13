package clients

import (
	"net/http"
)

//TunnelConnectorVNC is a kind of 'websockify' vnc to tcp proxy to be used by a novnc instance to connect to a vnc tunnel
type TunnelConnectorVNC struct {
	tunnelProxy *TunnelProxy
}

func NewTunnelConnectorVNC(tp *TunnelProxy) *TunnelConnectorVNC {
	return &TunnelConnectorVNC{tunnelProxy: tp}
}

func (tc *TunnelConnectorVNC) Start() {
}

func (tc *TunnelConnectorVNC) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}

func (tc *TunnelConnectorVNC) Stop() {
}
