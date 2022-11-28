package clienttunnel

import "errors"

var (
	ErrTunnelSubdomainAddressMissing  = errors.New("tunnel subdomain address missing")
	ErrTunnelSubdomainMissing         = errors.New("tunnel subdomain missing")
	ErrTunnelSubdomainCertFileMissing = errors.New("tunnel subdomain cert file missing")
	ErrTunnelSubdomainKeyFileMissing  = errors.New("tunnel subdomain key file missing")
)

type TunnelSubdomainConfig struct {
	Address   string `mapstructure:"tunnel_subdomain_address"`
	Subdomain string `mapstructure:"tunnel_subdomain"`
	CertFile  string `mapstructure:"tunnel_subdomain_cert_file"`
	KeyFile   string `mapstructure:"tunnel_subdomain_key_file"`
	Enabled   bool
}

func (c *TunnelSubdomainConfig) ParseAndValidate() error {
	// first check if not configured
	if c.Address == "" && c.Subdomain == "" && c.CertFile == "" && c.KeyFile == "" {
		return nil
	}
	if c.Address == "" {
		return ErrTunnelSubdomainAddressMissing
	}
	if c.Subdomain == "" {
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
