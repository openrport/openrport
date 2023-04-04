package acme

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"golang.org/x/crypto/acme/autocert"

	"github.com/realvnc-labs/rport/share/logger"
)

const httpChallengeServerReadHeaderTimeout = 3 * time.Second

type Acme struct {
	*logger.Logger
	manager  *autocert.Manager
	hosts    map[string]bool
	httpPort int
}

func New(l *logger.Logger, dataDir string, httpPort int) *Acme {
	a := &Acme{
		Logger:   l,
		hosts:    make(map[string]bool),
		httpPort: httpPort,
	}
	a.manager = &autocert.Manager{
		Cache:      autocert.DirCache(filepath.Join(dataDir, "acme")),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: a.hostPolicy,
	}
	return a
}

func (a *Acme) Start() {
	if a.httpPort > 0 {
		go a.listenHTTP()
	}
}

func (a *Acme) listenHTTP() {
	addr := fmt.Sprintf(":%d", a.httpPort)
	a.Infof("listening on %s", addr)

	server := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: httpChallengeServerReadHeaderTimeout,
		Handler:           a.manager.HTTPHandler(nil),
	}
	err := server.ListenAndServe()
	if err != nil {
		a.Errorf("failed to listen on acme http port: %v", err)
	}
}

func (a *Acme) hostPolicy(ctx context.Context, host string) error {
	if !a.hosts[host] {
		return fmt.Errorf("host %q not configured for acme", host)
	}
	return nil
}

// AddHost adds host(s) to allowed acme hosts, if host is url, then host part is extracted
func (a *Acme) AddHost(hosts ...string) {
	for _, host := range hosts {
		u, err := url.Parse(host)
		if err == nil && u.Host != "" {
			host = u.Host
		}
		h, _, err := net.SplitHostPort(host)
		if err == nil {
			host = h
		}
		a.Infof("enabled for %q", host)
		a.hosts[host] = true
	}
}

func (a *Acme) ApplyTLSConfig(cfg *tls.Config) *tls.Config {
	acmeConfig := a.manager.TLSConfig()
	cfg.GetCertificate = acmeConfig.GetCertificate
	cfg.NextProtos = acmeConfig.NextProtos
	return cfg
}
