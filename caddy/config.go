package caddy

import (
	"bytes"
	"errors"
	"net"
	"text/template"

	"github.com/aidarkhanov/nanoid/v2"
	"github.com/cloudradar-monitoring/rport/share/models"
)

const (
	baseConfFilename    = "caddy-base.conf"
	adminDomainSockName = "caddyadmin.sock"
)

var (
	ErrCaddyExecPathMissing                = errors.New("caddy executable path missing")
	ErrCaddyTunnelsHostAddressMissing      = errors.New("caddy tunnels address missing")
	ErrCaddyTunnelsBaseDomainMissing       = errors.New("caddy tunnels subdomain prefix missing")
	ErrCaddyTunnelsWildcardCertFileMissing = errors.New("caddy tunnels wildcard domains cert file missing")
	ErrCaddyTunnelsWildcardKeyFileMissing  = errors.New("caddy tunnels wildcard domains key file missing")
)

func (c *Config) ParseAndValidate(serverDataDir string) error {
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

	c.DataDir = serverDataDir
	c.BaseConfFilename = baseConfFilename

	c.Enabled = true
	return nil
}

func (c *Config) GetBaseConfText(bc *BaseConfig) (text []byte, err error) {
	tmpl := template.New("ALL")

	tmpl, err = tmpl.Parse(globalSettingsTemplate)
	if err != nil {
		return nil, err
	}

	tmpl, err = tmpl.Parse(defaultVirtualHost)
	if err != nil {
		return nil, err
	}

	tmpl, err = tmpl.Parse(apiReverseProxySettingsTemplate)
	if err != nil {
		return nil, err
	}

	tmpl, err = tmpl.Parse(allTemplate)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, bc)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (c *Config) MakeBaseConfFilename() (filename string) {
	filename = c.DataDir + "/" + c.BaseConfFilename
	return filename
}

func (c *Config) MakeBaseConfig(
	APICertFile string,
	APIKeyFile string,
	APIAddress string,
	APIHostNamePort string,
) (bc *BaseConfig, err error) {
	adminSocket := c.DataDir + "/" + adminDomainSockName

	gs := &GlobalSettings{
		LogLevel:    "ERROR",
		AdminSocket: adminSocket,
	}

	host, port, err := net.SplitHostPort(c.HostAddress)
	if err != nil {
		return nil, err
	}

	dvh := &DefaultVirtualHost{
		ListenAddress: host,
		ListenPort:    port,
		CertsFile:     c.CertFile,
		KeyFile:       c.KeyFile,
	}

	APIHost, APIPort, err := net.SplitHostPort(APIHostNamePort)
	if err != nil {
		return nil, err
	}

	APITargetHost, APITargetPort, err := net.SplitHostPort(APIAddress)
	if err != nil {
		return nil, err
	}

	arp := &APIReverseProxySettings{
		CertsFile:     APICertFile,
		KeyFile:       APIKeyFile,
		ProxyDomain:   APIHost,
		ProxyPort:     APIPort,
		APIScheme:     "https",
		APITargetHost: APITargetHost,
		APITargetPort: APITargetPort,
		ProxyLogFile:  "proxy_log_file",
	}

	bc = &BaseConfig{
		GlobalSettings:          gs,
		DefaultVirtualHost:      dvh,
		APIReverseProxySettings: arp,
	}

	return bc, nil
}

func (c *Config) GetLocalHostURL(subdomain string, remote *models.Remote) (hostURL string) {
	hostURL = subdomain + "." + c.BaseDomain
	return hostURL
}

func (c *Config) GetPort() (port string, err error) {
	_, port, err = net.SplitHostPort(c.HostAddress)
	return port, err
}

var DefaultAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func (c *Config) GetRandomSubdomain() (subdomain string, err error) {
	subdomain, err = nanoid.GenerateString(DefaultAlphabet, 21)
	if err != nil {
		return "", err
	}

	return subdomain, nil
}
