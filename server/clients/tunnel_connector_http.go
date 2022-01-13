package clients

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

//TunnelConnectorHTTP uses the standard ReverseProxy from package httputil to connect to HTTP server on tunnel endpoint
type TunnelConnectorHTTP struct {
	tunnelProxy  *TunnelProxy
	reverseProxy *httputil.ReverseProxy
}

func NewTunnelConnectorHTTP(tp *TunnelProxy) *TunnelConnectorHTTP {
	return &TunnelConnectorHTTP{tunnelProxy: tp}
}

func (tc *TunnelConnectorHTTP) Start() {
	tunnelURL := url.URL{
		Scheme: *tc.tunnelProxy.Tunnel.Remote.Scheme,
		Host:   net.JoinHostPort(tc.tunnelProxy.Tunnel.Remote.LocalHost, tc.tunnelProxy.Tunnel.Remote.LocalPort),
	}

	tc.tunnelProxy.Logger.Debugf("create https reverse proxy with ssl offloading forwarding to %s", tunnelURL.String())
	tc.reverseProxy = httputil.NewSingleHostReverseProxy(&tunnelURL)
	sslOffloadingTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
		},
	}
	tc.reverseProxy.Transport = sslOffloadingTransport
	tc.reverseProxy.ErrorHandler = tc.tunnelProxy.handleProxyError
}

func (tc *TunnelConnectorHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if tc.tunnelProxy.Tunnel.Remote.HostHeader != "" {
		r.Header.Set("Host", tc.tunnelProxy.Tunnel.Remote.HostHeader)
		r.Host = tc.tunnelProxy.Tunnel.Remote.HostHeader
	}
	tc.reverseProxy.ServeHTTP(w, r)
}

func (tc *TunnelConnectorHTTP) Stop() {
}
