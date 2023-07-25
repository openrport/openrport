package caddy

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldMakeNewRouteRequestJSON(t *testing.T) {
	tmpl := template.New("NRR")
	tmpl, err := tmpl.Parse(NewRouteRequestTemplate)
	require.NoError(t, err)

	nrr := &NewRouteRequest{
		RouteID:                   "new_route_id",
		TargetTunnelHost:          "target_tunnel_host",
		TargetTunnelPort:          "target_tunnel_port",
		DownstreamProxySubdomain:  "downstream_proxy_subdomain",
		DownstreamProxyBaseDomain: "downstream_proxy_basedomain",
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, nrr)
	require.NoError(t, err)

	templateText := b.String()

	assert.Contains(t, templateText, `"@id": "new_route_id"`)
	assert.Contains(t, templateText, `"handler": "reverse_proxy"`)
	assert.Contains(t, templateText, `"dial": "target_tunnel_host:target_tunnel_port"`)
	assert.Contains(t, templateText, `"downstream_proxy_subdomain.downstream_proxy_basedomain"`)
}

func TestShouldParseTemplates(t *testing.T) {
	cases := []struct {
		name     string
		template string
	}{
		{
			name:     "global settings",
			template: globalSettingsTemplate,
		},
		{
			name:     "default virtual host",
			template: defaultVirtualHost,
		},
		{
			name:     "api reverse proxy settings",
			template: apiReverseProxySettingsTemplate,
		},
		{
			name:     "combined without api reverse proxy",
			template: combinedTemplates,
		},
		{
			name:     "combined with api reverse proxy",
			template: combinedTemplatesWithAPIProxy,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := template.New(tc.name)
			_, err := tmpl.Parse(tc.template)
			assert.NoError(t, err)
		})
	}
}

func TestShouldMakeGlobalSettingsText(t *testing.T) {
	tmpl := template.New("GS")
	tmpl, err := tmpl.Parse(globalSettingsTemplate)
	require.NoError(t, err)

	gs := &GlobalSettings{
		LogLevel:    "ERROR",
		AdminSocket: "/tmp/caddy-admin.sock",
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, gs)
	require.NoError(t, err)

	templateText := b.String()

	assert.Contains(t, templateText, "level ERROR")
	assert.Contains(t, templateText, "admin unix//tmp/caddy-admin.sock")
}

func TestShouldMakeDefaultVirtualHostText(t *testing.T) {
	tmpl := template.New("DVH")
	tmpl, err := tmpl.Parse(defaultVirtualHost)
	require.NoError(t, err)

	dvh := &DefaultVirtualHost{
		ListenAddress: "listen_address",
		ListenPort:    "listen_port",
		CertsFile:     "certs_file",
		KeyFile:       "key_file",
		TLSMin:        "tls_min",
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, dvh)
	require.NoError(t, err)

	templateText := b.String()

	assert.Contains(t, templateText, "https://listen_address:listen_port")
	assert.Contains(t, templateText, "tls certs_file key_file")
}

func TestShouldMakeAPIReverseProxySettingsText(t *testing.T) {
	tmpl := template.New("ARP")
	tmpl, err := tmpl.Parse(apiReverseProxySettingsTemplate)
	require.NoError(t, err)

	arp := &APIReverseProxySettings{
		CertsFile:     "certs_file",
		KeyFile:       "key_file",
		ProxyDomain:   "proxy_domain",
		ProxyPort:     "proxy_port",
		APIDomain:     "api_domain",
		APIScheme:     "api_scheme",
		APITargetHost: "api_ip_address",
		APITargetPort: "api_port",
		TLSMin:        "tls_min",
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, arp)
	require.NoError(t, err)

	templateText := b.String()

	assert.Contains(t, templateText, "https://proxy_domain:proxy_port")
	assert.Contains(t, templateText, "tls certs_file key_file")
	assert.Contains(t, templateText, "to api_scheme://api_ip_address:api_port")
	assert.Contains(t, templateText, "output discard")
}

func TestShouldMakeAllWithAPIReverseProxy(t *testing.T) {
	tmpl := template.New("ALL")

	tmpl, err := tmpl.Parse(globalSettingsTemplate)
	require.NoError(t, err)

	tmpl, err = tmpl.Parse(defaultVirtualHost)
	require.NoError(t, err)

	tmpl, err = tmpl.Parse(apiReverseProxySettingsTemplate)
	require.NoError(t, err)

	// combined template with api reverse proxy
	tmpl, err = tmpl.Parse(combinedTemplatesWithAPIProxy)
	require.NoError(t, err)

	gs := &GlobalSettings{
		LogLevel:    "ERROR",
		AdminSocket: "/tmp/caddy-admin.sock",
	}

	dvh := &DefaultVirtualHost{
		ListenAddress: "listen_address",
		ListenPort:    "listen_port",
		CertsFile:     "certs_file",
		KeyFile:       "key_file",
		TLSMin:        "tls_min",
	}

	arp := &APIReverseProxySettings{
		CertsFile:     "certs_file",
		KeyFile:       "key_file",
		ProxyDomain:   "proxy_domain",
		ProxyPort:     "proxy_port",
		APIDomain:     "api_domain",
		APIScheme:     "api_scheme",
		APITargetHost: "api_ip_address",
		APITargetPort: "api_port",
		TLSMin:        "tls_min",
	}

	c := BaseConfig{
		GlobalSettings:          gs,
		APIReverseProxySettings: arp,
		DefaultVirtualHost:      dvh,
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, c)
	require.NoError(t, err)

	templateText := b.String()

	assert.Contains(t, templateText, "admin unix//tmp/caddy-admin.sock")
	assert.Contains(t, templateText, "https://listen_address:listen_port")
	assert.Contains(t, templateText, "https://proxy_domain:proxy_port")
}

func TestShouldMakeAllWithoutAPIReverseProxy(t *testing.T) {
	tmpl := template.New("ALL")

	tmpl, err := tmpl.Parse(globalSettingsTemplate)
	require.NoError(t, err)

	tmpl, err = tmpl.Parse(defaultVirtualHost)
	require.NoError(t, err)

	tmpl, err = tmpl.Parse(apiReverseProxySettingsTemplate)
	require.NoError(t, err)

	// combined template without api reverse proxy
	tmpl, err = tmpl.Parse(combinedTemplates)
	require.NoError(t, err)

	gs := &GlobalSettings{
		LogLevel:    "ERROR",
		AdminSocket: "/tmp/caddy-admin.sock",
	}

	dvh := &DefaultVirtualHost{
		ListenAddress: "listen_address",
		ListenPort:    "listen_port",
		CertsFile:     "certs_file",
		KeyFile:       "key_file",
		TLSMin:        "tls_min",
	}

	arp := &APIReverseProxySettings{
		CertsFile:     "certs_file",
		KeyFile:       "key_file",
		ProxyDomain:   "proxy_domain",
		ProxyPort:     "proxy_port",
		APIDomain:     "api_domain",
		APIScheme:     "api_scheme",
		APITargetHost: "api_ip_address",
		APITargetPort: "api_port",
		TLSMin:        "tls_min",
	}

	// include everything but we shouldn't see the api  reverse proxy settings in the text later
	c := BaseConfig{
		GlobalSettings:          gs,
		APIReverseProxySettings: arp,
		DefaultVirtualHost:      dvh,
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, c)
	require.NoError(t, err)

	templateText := b.String()

	assert.Contains(t, templateText, "admin unix//tmp/caddy-admin.sock")
	assert.Contains(t, templateText, "https://listen_address:listen_port")
	assert.NotContains(t, templateText, "https://proxy_domain:proxy_port")
}
