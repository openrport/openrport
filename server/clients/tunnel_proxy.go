package clients

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type TunnelProxyConfig struct {
	ProxyRequired bool
	cert          string
	key           string
	Host          string
	Port          string
	ProxyACL      *TunnelACL
}

func NewTunnelProxyNotRequiredConfig() *TunnelProxyConfig {
	return &TunnelProxyConfig{}
}

func NewTunnelProxyRequiredConfig(certFile string, keyFile string) *TunnelProxyConfig {
	return &TunnelProxyConfig{
		ProxyRequired: true,
		cert:          certFile,
		key:           keyFile,
	}
}

func (tpc *TunnelProxyConfig) Addr() string {
	return tpc.Host + ":" + tpc.Port
}

type TunnelProxy struct {
	Tunnel *Tunnel
	*chshare.Logger
	config      *TunnelProxyConfig
	proxyServer *http.Server
}

func NewTunnelProxy(tunnel *Tunnel, logger *chshare.Logger, config *TunnelProxyConfig) *TunnelProxy {
	return &TunnelProxy{
		Tunnel: tunnel,
		Logger: logger.Fork("tunnel-proxy:%s", config.Addr()),
		config: config,
	}
}

func (tp *TunnelProxy) tunnelProxyHandlerFunc(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if tp.config.ProxyACL != nil {
			clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
			if err == nil {
				ipv4 := net.ParseIP(clientIP)
				if ipv4 != nil {
					tcpIP := &net.TCPAddr{IP: ipv4}
					if tp.config.ProxyACL.CheckAccess(tcpIP) {
						goto forward
					}
				}
			}

			tp.Logger.Debugf("Access rejected. Remote addr: %s", clientIP)
			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			w.WriteHeader(http.StatusForbidden)
			return
		}

	forward:
		w.Header().Set("X-RPort-Tunnel-Proxy", tp.config.Addr())
		p.ServeHTTP(w, r)
	}
}

func (tp *TunnelProxy) Start(ctx context.Context) error {
	forwardTo := *tp.Tunnel.Remote.Scheme + "://" + tp.Tunnel.Remote.LocalHost + ":" + tp.Tunnel.Remote.LocalPort
	forwardURL, err := url.Parse(forwardTo)
	if err != nil {
		return err
	}

	tp.Logger.Debugf("create https reverse proxy with ssl offloading forwarding to %s", forwardURL)
	proxy := httputil.NewSingleHostReverseProxy(forwardURL)
	sslOffloadingTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
			MinVersion:         tls.VersionTLS12,
		},
	}
	proxy.Transport = sslOffloadingTransport
	proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, err error) {
		tp.Logger.Debugf("Error during proxy request %v", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", tp.tunnelProxyHandlerFunc(proxy))

	tp.proxyServer = &http.Server{
		Addr:    tp.config.Addr(),
		Handler: mux,
	}

	go func() {
		err = tp.proxyServer.ListenAndServeTLS(tp.config.cert, tp.config.key)
		if err != nil && err == http.ErrServerClosed {
			tp.Logger.Debugf("tunnel proxy closed")
			return
		}
		if err != nil {
			tp.Logger.Debugf("tunnel proxy ended with %v", err)
		}
	}()

	tp.Logger.Debugf("tunnel proxy started")
	return nil
}

func (tp *TunnelProxy) Stop(ctx context.Context) error {
	ctxShutDown, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := tp.proxyServer.Shutdown(ctxShutDown); err != nil {
		tp.Logger.Debugf("tunnel proxy shutdown failed:%+s", err)
	}

	return nil
}
