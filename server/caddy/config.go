package caddy

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"text/template"

	"github.com/realvnc-labs/rport/share/files"

	"github.com/aidarkhanov/nanoid/v2"
)

const (
	baseConfFilename    = "caddy-base.conf"
	adminDomainSockName = "caddy-admin.sock"
)

var (
	ErrCaddyExecPathMissing                     = errors.New("caddy executable path missing")
	ErrCaddyExecNotFound                        = errors.New("caddy executable not found")
	ErrCaddyFailedCheckingExecPath              = errors.New("failed checking caddy exec path")
	ErrCaddyUnableToGetCaddyServerVersion       = errors.New("failed getting caddy server version")
	ErrCaddyServerExecutableTooOld              = errors.New("caddy server version too old. please use v2 or later")
	ErrCaddyTunnelsHostAddressMissing           = errors.New("caddy tunnels address missing")
	ErrCaddyTunnelsBaseDomainMissing            = errors.New("caddy tunnels subdomain prefix missing")
	ErrCaddyTunnelsWildcardCertFileMissing      = errors.New("caddy tunnels wildcard domains cert file missing")
	ErrCaddyTunnelsWildcardKeyFileMissing       = errors.New("caddy tunnels wildcard domains key file missing")
	ErrCaddyUnknownLogLevel                     = errors.New("rport log level not a known caddy log level")
	ErrCaddyMissingAPIPort                      = errors.New("when api_hostname specified then api_port must also be set")
	ErrCaddyMissingAPIHostname                  = errors.New("when api_port specified then api_hostname must also be set")
	ErrUnableToGetAddressAndPortFromHostAddress = errors.New("unable to get ip address and port from caddy address. please make sure both are set")
	ErrUnableToCheckIfCertFileExists            = errors.New("unable to check if caddy cert file exists")
	ErrCaddyCertFileNotFound                    = errors.New("caddy cert file not found")
	ErrUnableToCheckIfKeyFileExists             = errors.New("unable to check if caddy key file exists")
	ErrCaddyKeyFileNotFound                     = errors.New("caddy key file not found")
	ErrUnableToCheckIfAPICertFileExists         = errors.New("unable to check if caddy api cert file exists")
	ErrCaddyAPICertFileNotFound                 = errors.New("caddy api cert file not found")
	ErrUnableToCheckIfAPIKeyFileExists          = errors.New("unable to check if caddy api cert file exists")
	ErrCaddyAPIKeyFileNotFound                  = errors.New("caddy api key file not found")
	ErrCaddyUnknownTlsMin                       = errors.New("tls_min not a known tls protocol version")
)

type Config struct {
	ExecPath    string `mapstructure:"caddy"`
	HostAddress string `mapstructure:"address"`
	BaseDomain  string `mapstructure:"subdomain_prefix"`
	CertFile    string `mapstructure:"cert_file"`
	KeyFile     string `mapstructure:"key_file"`
	APIHostname string `mapstructure:"api_hostname"`
	APIPort     string `mapstructure:"api_port"`
	APICertFile string `mapstructure:"api_cert_file"`
	APIKeyFile  string `mapstructure:"api_key_file"`
	TlsMin      string `mapstructure:"tls_min"`

	LogLevel         string `mapstructure:"-"` // taken from the rport server log level
	DataDir          string `mapstructure:"-"` // taken from the rport server datadir
	BaseConfFilename string `mapstructure:"-"`
	Enabled          bool   `mapstructure:"-"`

	SubDomainGenerator SubdomainGenerator
}

var caddyLogLevels = []string{"debug", "info", "warn", "error", "panic", "fatal"}

func existingCaddyLogLevel(loglevel string) (found bool) {
	for _, level := range caddyLogLevels {
		if strings.EqualFold(loglevel, level) {
			return true
		}
	}
	return false
}

func (c *Config) ParseAndValidate(serverDataDir string, serverLogLevel string, filesAPI files.FileAPI) error {
	// first check if not configured at all
	if c.ExecPath == "" && c.HostAddress == "" && c.BaseDomain == "" && c.CertFile == "" && c.KeyFile == "" {
		return nil
	}

	if c.ExecPath == "" {
		return ErrCaddyExecPathMissing
	}

	exists, err := filesAPI.Exist(c.ExecPath)
	if err != nil {
		return ErrCaddyFailedCheckingExecPath
	}
	if !exists {
		return ErrCaddyExecNotFound
	}

	version, err := GetExecVersion(c)
	if err != nil {
		return ErrCaddyUnableToGetCaddyServerVersion
	}
	if version < 2 {
		return ErrCaddyServerExecutableTooOld
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

	_, _, err = net.SplitHostPort(c.HostAddress)
	if err != nil {
		return ErrUnableToGetAddressAndPortFromHostAddress
	}

	exists, err = filesAPI.Exist(c.CertFile)
	if err != nil {
		return ErrUnableToCheckIfCertFileExists
	}
	if !exists {
		return ErrCaddyCertFileNotFound
	}

	exists, err = filesAPI.Exist(c.KeyFile)
	if err != nil {
		return ErrUnableToCheckIfKeyFileExists
	}
	if !exists {
		return ErrCaddyKeyFileNotFound
	}

	_, err = tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return fmt.Errorf("invalid 'cert_file', 'key_file': %v", err)
	}

	if c.APIHostname != "" && c.APIPort == "" {
		return ErrCaddyMissingAPIPort
	}
	if c.APIPort != "" && c.APIHostname == "" {
		return ErrCaddyMissingAPIHostname
	}

	if c.APICertFile != "" {
		exists, err = filesAPI.Exist(c.APICertFile)
		if err != nil {
			return ErrUnableToCheckIfAPICertFileExists
		}
		if !exists {
			return ErrCaddyAPICertFileNotFound
		}

		_, err = tls.LoadX509KeyPair(c.APICertFile, c.APIKeyFile)
		if err != nil {
			return fmt.Errorf("invalid 'cert_file', 'key_file': %v", err)
		}

	}

	if c.APIKeyFile != "" {
		exists, err = filesAPI.Exist(c.APIKeyFile)
		if err != nil {
			return ErrUnableToCheckIfAPIKeyFileExists
		}
		if !exists {
			return ErrCaddyAPIKeyFileNotFound
		}
	}

	if c.TlsMin != "" && c.TlsMin != "1.2" && c.TlsMin != "1.3" {
		return ErrCaddyUnknownTlsMin
	}

	c.LogLevel = serverLogLevel
	if c.LogLevel != "" && !existingCaddyLogLevel(c.LogLevel) {
		return ErrCaddyUnknownLogLevel
	}

	c.DataDir = serverDataDir
	c.BaseConfFilename = baseConfFilename
	c.SubDomainGenerator = c
	c.Enabled = true

	return nil
}

func (c *Config) APIReverseProxyEnabled() (enabled bool) {
	return c.APIHostname != "" && c.APIPort != ""
}

func (c *Config) GetBaseConf(bc *BaseConfig) (text []byte, err error) {
	tmpl := template.New("ALL")

	tmpl, err = tmpl.Parse(globalSettingsTemplate)
	if err != nil {
		return nil, err
	}

	tmpl, err = tmpl.Parse(defaultVirtualHost)
	if err != nil {
		return nil, err
	}

	if bc.IncludeAPIProxy {
		tmpl, err = tmpl.Parse(apiReverseProxySettingsTemplate)
		if err != nil {
			return nil, err
		}

		tmpl, err = tmpl.Parse(combinedTemplatesWithAPIProxy)
		if err != nil {
			return nil, err
		}
	} else {
		tmpl, err = tmpl.Parse(combinedTemplates)
		if err != nil {
			return nil, err
		}
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, bc)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (c *Config) MakeBaseConfig(targetAPIPort string) (bc *BaseConfig, err error) {
	APICertFile := c.CertFile
	if c.APICertFile != "" {
		APICertFile = c.APICertFile
	}
	APIKeyFile := c.CertFile
	if c.APIKeyFile != "" {
		APIKeyFile = c.APIKeyFile
	}

	adminSocket := c.DataDir + "/" + adminDomainSockName

	// make sure we always start with a new sock file, as leaving the old file around seems to cause
	// startup issues otherwise.
	err = os.Remove(adminSocket)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	logLevel := c.LogLevel
	if logLevel == "" {
		logLevel = "info"
	}

	tlsMin := c.TlsMin
	if tlsMin == "" {
		tlsMin = "tls1.3"
	}

	gs := &GlobalSettings{
		LogLevel:    logLevel,
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
		TlsMin:        tlsMin,
	}

	bc = &BaseConfig{
		GlobalSettings:     gs,
		DefaultVirtualHost: dvh,
	}

	if c.APIReverseProxyEnabled() {
		arp := &APIReverseProxySettings{
			CertsFile:     APICertFile,
			KeyFile:       APIKeyFile,
			ProxyDomain:   c.APIHostname,
			ProxyPort:     port, // Start the API reverse proxy on the same port as the tunnel reverse proxy
			APIScheme:     "http",
			APITargetHost: "127.0.0.1",
			APITargetPort: targetAPIPort,
			TlsMin:        tlsMin,
		}

		bc.APIReverseProxySettings = arp
		bc.IncludeAPIProxy = true
	}

	return bc, nil
}

func (c *Config) MakeBaseConfFilename() (filename string) {
	filename = c.DataDir + "/" + c.BaseConfFilename
	return filename
}

const DefaultAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"

type SubdomainGenerator interface {
	GetRandomSubdomain() (subdomain string, err error)
}

func (c *Config) GetRandomSubdomain() (subdomain string, err error) {
	subdomain, err = nanoid.GenerateString(DefaultAlphabet, 21)
	if err != nil {
		return "", err
	}

	return subdomain, nil
}
