package caddy

type BaseConfig struct {
	IncludeAPIProxy         bool
	GlobalSettings          *GlobalSettings
	DefaultVirtualHost      *DefaultVirtualHost
	APIReverseProxySettings *APIReverseProxySettings
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
	TLSMin        string
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
	TLSMin        string
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
