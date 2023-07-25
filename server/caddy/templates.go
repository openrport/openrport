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
{{- template "DVH" .DefaultVirtualHost -}}`

const combinedTemplatesWithAPIProxy = `
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
		protocols {{.TLSMin}}
	}
	respond "not found" 404
}
{{ end }}
`

const apiReverseProxySettingsTemplate = `
{{ define "ARP"}}
https://{{.ProxyDomain}}:{{.ProxyPort}} {
	tls {{.CertsFile}} {{.KeyFile}} {
		protocols {{.TLSMin}}
	}
	reverse_proxy {
		to {{.APIScheme}}://{{.APITargetHost}}:{{.APITargetPort}}
	}
	log {
		output discard
	}
}
{{ end }}
`
