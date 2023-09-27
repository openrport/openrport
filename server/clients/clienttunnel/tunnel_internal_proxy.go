package clienttunnel

import (
	"context"
	"crypto/tls"
	"embed"
	_ "embed" //to embed CSS
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"github.com/openrport/openrport/server/acme"
	chshare "github.com/openrport/openrport/share"
	"github.com/openrport/openrport/share/logger"
	"github.com/openrport/openrport/share/security"
)

//go:embed css/tunnel-proxy.css
var tunnelProxyCSS embed.FS

// semantic.css was deminified via js-beautify
//
//go:embed css/semantic.css
var semanticCSS embed.FS

type InternalTunnelProxyConfig struct {
	Host         string   `mapstructure:"tunnel_host"`
	CertFile     string   `mapstructure:"tunnel_proxy_cert_file"`
	KeyFile      string   `mapstructure:"tunnel_proxy_key_file"`
	EnableAcme   bool     `mapstructure:"tunnel_enable_acme"`
	NovncRoot    string   `mapstructure:"novnc_root"`
	TLSMin       string   `mapstructure:"tls_min"`
	GuacdAddress string   `mapstructure:"guacd_address"`
	CORS         []string `mapstructure:"tunnel_cors"`
	Enabled      bool
}

func (c *InternalTunnelProxyConfig) ParseAndValidate() error {
	if c.CertFile == "" && c.KeyFile == "" && !c.EnableAcme {
		c.Enabled = false
		return nil
	}
	if c.CertFile != "" && c.KeyFile == "" {
		return errors.New("when 'tunnel_proxy_cert_file' is set, 'tunnel_proxy_key_file' must be set as well")
	}
	if c.KeyFile != "" && c.CertFile == "" {
		return errors.New("when 'tunnel_proxy_key_file' is set, 'tunnel_proxy_cert_file' must be set as well")
	}
	if c.EnableAcme && c.CertFile != "" {
		return errors.New("tunnel_proxy_cert_file, tunnel_proxy_key_file and tunnel_enable_acme cannot be used together")
	}
	if c.CertFile != "" && c.KeyFile != "" {
		_, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err != nil {
			return fmt.Errorf("invalid 'tunnel_proxy_cert_file', 'tunnel_proxy_key_file': %v", err)
		}
	}
	if err := c.validateGuacd(c.GuacdAddress); err != nil {
		return fmt.Errorf("try guacd connection: %v", err)
	}
	if err := c.validateHost(); err != nil {
		return err
	}
	if c.TLSMin != "" && c.TLSMin != "1.2" && c.TLSMin != "1.3" {
		return errors.New("TLS must be either 1.2 or 1.3")
	}
	c.Enabled = true

	return nil
}

func (c *InternalTunnelProxyConfig) validateGuacd(addr string) error {
	if addr == "" {
		return nil
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return err
	}
	conn.Close()

	return nil
}

func (c *InternalTunnelProxyConfig) validateHost() error {
	if c.Host != "" && !govalidator.IsDNSName(c.Host) && !govalidator.IsIP(c.Host) {
		return fmt.Errorf("invalid tunnel_host '%s': use IP address or FQDN", c.Host)
	}
	return nil
}

type InternalTunnelProxy struct {
	Tunnel               *Tunnel
	Logger               *logger.Logger
	Config               *InternalTunnelProxyConfig
	Host                 string
	Port                 string
	TunnelHost           string
	TunnelPort           string
	acl                  atomic.Pointer[TunnelACL]
	proxyServer          *http.Server
	tunnelProxyConnector TunnelProxyConnector
	acme                 *acme.Acme
}

func NewInternalTunnelProxy(tunnel *Tunnel, logger *logger.Logger, config *InternalTunnelProxyConfig, host string, port string, acl *TunnelACL, acme *acme.Acme) *InternalTunnelProxy {
	tp := &InternalTunnelProxy{
		Tunnel:     tunnel,
		Config:     config,
		Host:       host,
		Port:       port,
		TunnelHost: tunnel.Remote.LocalHost,
		TunnelPort: tunnel.Remote.LocalPort,
		acme:       acme,
	}
	tp.SetACL(acl)
	tp.Logger = logger.Fork("tunnel-proxy:%s", tp.Addr())
	tp.tunnelProxyConnector = NewTunnelProxyConnector(tp)
	return tp
}

func (tp *InternalTunnelProxy) Start(ctx context.Context) error {
	router := mux.NewRouter()
	router.Use(tp.handleACL)

	router.Handle("/css/tunnel-proxy.css", http.FileServer(http.FS(tunnelProxyCSS)))
	router.Handle("/css/semantic.css", http.FileServer(http.FS(semanticCSS)))

	router = tp.tunnelProxyConnector.InitRouter(router)

	if len(tp.Config.CORS) > 0 {
		router.Use(cors.New(cors.Options{
			AllowedOrigins:   tp.Config.CORS,
			AllowCredentials: true,
			AllowedMethods: []string{
				http.MethodHead,
				http.MethodGet,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
			},
			AllowedHeaders: []string{"*"},
		}).Handler)
	}

	tp.proxyServer = &http.Server{
		Addr:              tp.Addr(),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go tp.listen()

	tp.Logger.Infof("tunnel proxy started")
	return nil
}

func (tp *InternalTunnelProxy) listen() {
	tp.Logger.Debugf("listener starting")

	// this tlsmin is the InternalTunnelProxyConfig config in the server section
	tp.proxyServer.TLSConfig = security.TLSConfig(tp.Config.TLSMin)
	if tp.Config.EnableAcme {
		tp.proxyServer.TLSConfig = tp.acme.ApplyTLSConfig(tp.proxyServer.TLSConfig)
	}
	err := tp.proxyServer.ListenAndServeTLS(tp.Config.CertFile, tp.Config.KeyFile)
	if err != nil && err == http.ErrServerClosed {
		tp.Logger.Infof("tunnel proxy closed")
		return
	}
	if err != nil {
		tp.Logger.Debugf("tunnel proxy ended with %v", err)
	}
}

func (tp *InternalTunnelProxy) Stop(ctx context.Context) error {
	ctxShutDown, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := tp.proxyServer.Shutdown(ctxShutDown); err != nil {
		tp.Logger.Infof("tunnel proxy shutdown failed:%+s", err)
	}

	return nil
}

func (tp *InternalTunnelProxy) Addr() string {
	return net.JoinHostPort(tp.Host, tp.Port)
}

func (tp *InternalTunnelProxy) TunnelAddr() string {
	return net.JoinHostPort(tp.TunnelHost, tp.TunnelPort)
}

// handleACL middleware to handle ACL
func (tp *InternalTunnelProxy) handleACL(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acl := tp.acl.Load()
		if acl == nil {
			next.ServeHTTP(w, r)
			return
		}
		clientIP := chshare.RemoteIP(r)
		ipv4 := net.ParseIP(clientIP)
		if ipv4 == nil {
			tp.Logger.Infof("Proxy Access rejected. Cannot parse ip: %s", clientIP)
		}
		if ipv4 != nil {
			tcpIP := &net.TCPAddr{IP: ipv4}
			if acl.CheckAccess(tcpIP.IP) {
				next.ServeHTTP(w, r)
				return
			}

			tp.Logger.Infof("Proxy Access rejected. Remote addr: %s", clientIP)
		}
		tp.sendHTML(w, http.StatusForbidden, "Access rejected by ACL")
	})
}

func (tp *InternalTunnelProxy) serveTemplate(w http.ResponseWriter, r *http.Request, templateContent string, templateData map[string]interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	tmpl, err := template.New("").Parse(templateContent)
	if err == nil {
		err = tmpl.Execute(w, templateData)
	}
	if err != nil {
		tp.Logger.Errorf("Error while serving template for request %s: %v", r.RequestURI, err)
	}
}

func (tp *InternalTunnelProxy) handleProxyError(w http.ResponseWriter, r *http.Request, err error) {
	tp.Logger.Errorf("Error during proxy request %v", err)
	tp.sendHTML(w, http.StatusInternalServerError, err.Error())
}

func (tp *InternalTunnelProxy) sendHTML(w http.ResponseWriter, statusCode int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(statusCode)
	m := fmt.Sprintf("[%d] Rport tunnel proxy: %s", statusCode, msg)
	_, _ = w.Write([]byte(m))
}

// noCache middleware to disable caching
func (tp *InternalTunnelProxy) noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
}

func (tp *InternalTunnelProxy) SetACL(acl *TunnelACL) {
	tp.acl.Store(acl)
}
