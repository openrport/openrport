package caddy

import (
	"bytes"
	_ "embed"
	"text/template"
)

func ExecuteTemplate(templateName string, t string, params any) (applied []byte, err error) {
	tmpl := template.New(templateName)
	tmpl, err = tmpl.Parse(t)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, params)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

//go:embed new_route_request_template.json
var NewRouteRequestTemplate string

const combinedTemplates = `
{{- template "GS" .GlobalSettings }}
{{- template "ARP" .APIReverseProxySettings }}
{{- template "DVH" .DefaultVirtualHost -}}`

const globalSettingsTemplate = `
{{- define "GS"}}
{
	grace_period 1s
	auto_https off

	log {
		level {{.LogLevel}}
	}

	admin unix/{{.AdminSocket}} {
		origins unix
	}
}
{{ end }}
`

const defaultVirtualHost = `
{{ define "DVH"}}
https://{{.ListenAddress}}:{{.ListenPort}} {
	tls {{.CertsFile}} {{.KeyFile}} {
		protocols tls1.3
	}
	respond "not found" 404
}
{{ end }}
`

// TODO: (rs): add test for when not using API proxy

const apiReverseProxySettingsTemplate = `
{{ define "ARP"}}
{{- if .UseAPIProxy }}
https://{{.ProxyDomain}}:{{.ProxyPort}} {
	tls {{.CertsFile}} {{.KeyFile}} {
		protocols tls1.3
	}
	reverse_proxy {
		to {{.APIScheme}}://{{.APITargetHost}}:{{.APITargetPort}}
			transport http {
				tls
				tls_insecure_skip_verify
			}
	}
	log {
		output discard
	}
}
{{- end }}
{{ end }}
`
