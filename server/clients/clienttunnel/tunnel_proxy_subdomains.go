package clienttunnel

import (
	"errors"
	"net"

	"github.com/cloudradar-monitoring/rport/share/models"
)

var (
	ErrTunnelSubdomainAddressMissing  = errors.New("tunnel subdomain address missing")
	ErrTunnelSubdomainMissing         = errors.New("tunnel subdomain missing")
	ErrTunnelSubdomainCertFileMissing = errors.New("tunnel subdomain cert file missing")
	ErrTunnelSubdomainKeyFileMissing  = errors.New("tunnel subdomain key file missing")
)

type SubdomainConfig struct {
	Address       string `mapstructure:"tunnel_subdomain_address"`
	BaseSubdomain string `mapstructure:"tunnel_subdomain"`
	CertFile      string `mapstructure:"tunnel_subdomain_cert_file"`
	KeyFile       string `mapstructure:"tunnel_subdomain_key_file"`
	Enabled       bool
}

func (c *SubdomainConfig) ParseAndValidate() error {
	// first check if not configured
	if c.Address == "" && c.BaseSubdomain == "" && c.CertFile == "" && c.KeyFile == "" {
		return nil
	}
	if c.Address == "" {
		return ErrTunnelSubdomainAddressMissing
	}
	if c.BaseSubdomain == "" {
		return ErrTunnelSubdomainMissing
	}
	if c.CertFile == "" {
		return ErrTunnelSubdomainCertFileMissing
	}
	if c.KeyFile == "" {
		return ErrTunnelSubdomainKeyFileMissing
	}

	// TODO: (rs): validate the cert and key files

	c.Enabled = true
	return nil
}

var (
	nextSubdomain = 0
	subdomains    = []string{"1", "2", "3", "4"}
)

func (c *SubdomainConfig) GetLocalHostURL(subdomain string, remote *models.Remote) (hostURL string) {
	hostURL = subdomain + "." + c.BaseSubdomain
	return hostURL
}

func (c *SubdomainConfig) GetPort() (port string, err error) {
	_, port, err = net.SplitHostPort(c.Address)
	return port, err
}

func (c *SubdomainConfig) ResetRandomDomains() {
	nextSubdomain = 0
}

func (c *SubdomainConfig) GetRandomSubdomain() (subdomain string) {
	subdomain = subdomains[nextSubdomain]
	nextSubdomain++
	return subdomain
}
