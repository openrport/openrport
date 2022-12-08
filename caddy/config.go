package caddy

import (
	"errors"
	"net"

	"github.com/cloudradar-monitoring/rport/share/models"
)

var (
	ErrCaddyExecPathMissing                = errors.New("caddy executable path missing")
	ErrCaddyTunnelsHostAddressMissing      = errors.New("caddy tunnels address missing")
	ErrCaddyTunnelsBaseDomainMissing       = errors.New("caddy tunnels subdomain prefix missing")
	ErrCaddyTunnelsWildcardCertFileMissing = errors.New("caddy tunnels wildcard domains cert file missing")
	ErrCaddyTunnelsWildcardKeyFileMissing  = errors.New("caddy tunnels wildcard domains key file missing")
)

func (c *Config) ParseAndValidate() error {
	// first check if not configured at all
	if c.ExecPath == "" && c.HostAddress == "" && c.BaseDomain == "" && c.CertFile == "" && c.KeyFile == "" {
		return nil
	}

	if c.ExecPath == "" {
		return ErrCaddyExecPathMissing
	}
	if c.HostAddress == "" {
		return ErrCaddyTunnelsHostAddressMissing
	}
	if c.BaseDomain == "" {
		return ErrCaddyTunnelsBaseDomainMissing
	}
	if c.CertFile == "" {
		return ErrCaddyTunnelsWildcardCertFileMissing
	}
	if c.KeyFile == "" {
		return ErrCaddyTunnelsWildcardKeyFileMissing
	}

	// TODO: (rs): check executable exists at path
	// TODO: (rs): check executable if greater than v2
	// TODO: (rs): validate the cert and key files

	c.Enabled = true
	return nil
}

var (
	nextSubdomain = 0
	subdomains    = []string{"1", "2", "3", "4"}
)

func (c *Config) GetLocalHostURL(subdomain string, remote *models.Remote) (hostURL string) {
	hostURL = subdomain + "." + c.BaseDomain
	return hostURL
}

func (c *Config) GetPort() (port string, err error) {
	_, port, err = net.SplitHostPort(c.HostAddress)
	return port, err
}

func (c *Config) ResetRandomDomains() {
	nextSubdomain = 0
}

func (c *Config) GetRandomSubdomain() (subdomain string) {
	subdomain = subdomains[nextSubdomain]
	nextSubdomain++
	return subdomain
}
