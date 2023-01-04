package clienttunnel

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gorilla/mux"
)

// TunnelProxyConnectorHTTP uses the standard ReverseProxy from package httputil to connect to HTTP/HTTPS server on tunnel endpoint
type TunnelProxyConnectorHTTP struct {
	tunnelProxy  *InternalTunnelProxy
	reverseProxy *httputil.ReverseProxy
}

func NewTunnelConnectorHTTP(tp *InternalTunnelProxy) *TunnelProxyConnectorHTTP {
	return &TunnelProxyConnectorHTTP{tunnelProxy: tp}
}

func (tc *TunnelProxyConnectorHTTP) InitRouter(router *mux.Router) *mux.Router {
	router.PathPrefix("/").HandlerFunc(tc.serveHTTP)

	if tc.tunnelProxy.Tunnel.Remote.HostHeader != "" {
		tc.tunnelProxy.Logger.Debugf("using host header %s", tc.tunnelProxy.Tunnel.HostHeader)
		router.Use(tc.addHostHeader)
	}

	tc.createReverseProxy()

	return router
}

func (tc *TunnelProxyConnectorHTTP) createReverseProxy() {
	tunnelURL := url.URL{
		Scheme: *tc.tunnelProxy.Tunnel.Remote.Scheme,
		Host:   tc.tunnelProxy.TunnelAddr(),
	}

	tc.tunnelProxy.Logger.Infof("create https reverse proxy with ssl offloading forwarding to %s", tunnelURL.String())
	tc.reverseProxy = httputil.NewSingleHostReverseProxy(&tunnelURL)
	sslOffloadingTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
		},
	}
	tc.reverseProxy.Transport = sslOffloadingTransport
	tc.reverseProxy.ErrorHandler = tc.tunnelProxy.handleProxyError
}

func (tc *TunnelProxyConnectorHTTP) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if tc.tunnelProxy.Tunnel.Remote.AuthUser != "" && tc.tunnelProxy.Tunnel.Remote.AuthPassword != "" {
		user, password, ok := r.BasicAuth()
		if !ok || user != tc.tunnelProxy.Tunnel.Remote.AuthUser || password != tc.tunnelProxy.Tunnel.Remote.AuthPassword {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}
	tc.reverseProxy.ServeHTTP(w, r)
}

func (tc *TunnelProxyConnectorHTTP) addHostHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("Host", tc.tunnelProxy.Tunnel.Remote.HostHeader)
		r.Host = tc.tunnelProxy.Tunnel.Remote.HostHeader

		next.ServeHTTP(w, r)
	})
}
