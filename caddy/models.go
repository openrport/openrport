package caddy

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
	CertsFile          string
	KeyFile            string
	UseAPIProxy        bool
	ProxyDomain        string
	ProxyPort          string
	APIDomain          string
	APIScheme          string
	APITargetHost      string
	APITargetPort      string
	AllowInsecureCerts bool
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
