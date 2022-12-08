package caddy

type NewRouteRequest struct {
	RouteID                 string
	TargetTunnelHost        string
	TargetTunnelPort        string
	UpstreamProxySubdomain  string
	UpstreamProxyBaseDomain string
}

type BaseConfig struct {
	GlobalSettings          *GlobalSettings
	DefaultVirtualHost      *DefaultVirtualHost
	APIReverseProxySettings *APIReverseProxySettings
	ReverseProxies          []ExternalReverseProxy
}

type GlobalSettings struct {
	LogLevel    string
	AdminSocket string
}

type DefaultVirtualHost struct {
	ListenAddress string
	ListenPort    string
	CertsFile     string
	KeyFile       string
}

type APIReverseProxySettings struct {
	CertsFile     string
	KeyFile       string
	ProxyDomain   string
	ProxyPort     string
	APIDomain     string
	APIScheme     string
	APITargetHost string
	APITargetPort string
	ProxyLogFile  string
}

type ExternalReverseProxy struct {
	CertsFile        string
	KeyFile          string
	BaseDomain       string
	Subdomain        string
	AllowedIPAddress string
	TunnelScheme     string
	TunnelIPAddress  string
	TunnelPort       string
}

type Config struct {
	ExecPath         string `mapstructure:"caddy"`
	BaseConfFilename string `mapstructure:"-"`
	HostAddress      string `mapstructure:"address"`
	BaseDomain       string `mapstructure:"subdomain_prefix"`
	CertFile         string `mapstructure:"cert_file"`
	KeyFile          string `mapstructure:"key_file"`
	DataDir          string `mapstructure:"-"`
	Enabled          bool   `mapstructure:"-"`
}
