package caddy

import (
	"bytes"
	"errors"
	"net"
	"strings"
	"text/template"

	"github.com/cloudradar-monitoring/rport/share/files"

	"github.com/aidarkhanov/nanoid/v2"
)

const (
	baseConfFilename    = "caddy-base.conf"
	adminDomainSockName = "caddy-admin.sock"
)

var (
	ErrCaddyExecPathMissing                = errors.New("caddy executable path missing")
	ErrCaddyExecNotFound                   = errors.New("caddy executable not found")
	ErrCaddyFailedCheckingExecPath         = errors.New("failed checking caddy exec path")
	ErrCaddyUnableToGetCaddyServerVersion  = errors.New("failed getting caddy server version")
	ErrCaddyServerExecutableTooOld         = errors.New("caddy server version too old. please use v2 or later")
	ErrCaddyTunnelsHostAddressMissing      = errors.New("caddy tunnels address missing")
	ErrCaddyTunnelsBaseDomainMissing       = errors.New("caddy tunnels subdomain prefix missing")
	ErrCaddyTunnelsWildcardCertFileMissing = errors.New("caddy tunnels wildcard domains cert file missing")
	ErrCaddyTunnelsWildcardKeyFileMissing  = errors.New("caddy tunnels wildcard domains key file missing")
	ErrCaddyUnknownLogLevel                = errors.New("unknown caddy log level")
)

type Config struct {
	ExecPath         string `mapstructure:"caddy"`
	BaseConfFilename string `mapstructure:"-"`
	HostAddress      string `mapstructure:"address"`
	BaseDomain       string `mapstructure:"subdomain_prefix"`
	CertFile         string `mapstructure:"cert_file"`
	KeyFile          string `mapstructure:"key_file"`
	LogLevel         string `mapstructure:"-"`
	DataDir          string `mapstructure:"-"`
	Enabled          bool   `mapstructure:"-"`

	SubDomainGenerator SubdomainGenerator
}

var caddyLogLevels = []string{"Debug", "Info", "Warn", "Error", "Panic", "Fatal"}

func existingCaddyLogLevel(loglevel string) (found bool) {
	for _, level := range caddyLogLevels {
		if strings.EqualFold(loglevel, level) {
			return true
		}
	}
	return false
}

func (c *Config) ParseAndValidate(serverDataDir string, serverLogLevel string, filesAPI *files.FileSystem) error {
	// first check if not configured at all
	if c.ExecPath == "" && c.HostAddress == "" && c.BaseDomain == "" && c.CertFile == "" && c.KeyFile == "" {
		return nil
	}

	if c.ExecPath == "" {
		return ErrCaddyExecPathMissing
	}

	exists, err := ExecExists(c.ExecPath, filesAPI)
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

	tmpl, err = tmpl.Parse(apiReverseProxySettingsTemplate)
	if err != nil {
		return nil, err
	}

	tmpl, err = tmpl.Parse(combinedTemplates)
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

	logLevel := c.LogLevel
	if logLevel == "" {
		logLevel = "info"
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
	}

	APIHost := ""
	APIPort := ""
	UseAPIProxy := false
	if APIHostNamePort != "" {
		APIHost, APIPort, err = net.SplitHostPort(APIHostNamePort)
		if err != nil {
			return nil, err
		}
		UseAPIProxy = true
	}

	APITargetScheme := "http"
	if APICertFile != "" && APIKeyFile != "" {
		APITargetScheme = "https"
	}

	APITargetHost, APITargetPort, err := net.SplitHostPort(APIAddress)
	if err != nil {
		return nil, err
	}

	arp := &APIReverseProxySettings{
		CertsFile:     APICertFile,
		KeyFile:       APIKeyFile,
		UseAPIProxy:   UseAPIProxy,
		ProxyDomain:   APIHost,
		ProxyPort:     APIPort,
		APIScheme:     APITargetScheme,
		APITargetHost: APITargetHost,
		APITargetPort: APITargetPort,
		// TODO: (rs): decide what to do about this log file
		ProxyLogFile: "caddy_log_file",
		// TODO: (rs): don't allow insecure certs by default
		AllowInsecureCerts: true,
	}

	bc = &BaseConfig{
		GlobalSettings:          gs,
		APIReverseProxySettings: arp,
		DefaultVirtualHost:      dvh,
	}

	return bc, nil
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
