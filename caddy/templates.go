package caddy

import (
	"bytes"
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

const NewRouteRequestTemplate = `
{
	"@id": "{{.RouteID}}",
	"handle": [
			{
					"handler": "subroute",
					"routes": [
							{
									"handle": [
											{
													"body": "Access denied",
													"close": true,
													"handler": "static_response",
													"status_code": 403
											}
									]
							},
							{
									"handle": [
											{
													"handler": "reverse_proxy",
													"headers": {
															"request": {
																	"set": {
																			"X-Forwarded-Host": [
																					"{http.request.host}"
																			]
																	}
															}
													},
													"transport": {
															"protocol": "http",
															"tls": {
																	"insecure_skip_verify": true
															}
													},
													"upstreams": [
															{
																	"dial": "{{.TargetTunnelHost}}:{{.TargetTunnelPort}}"
															}
													]
											}
									]
							}
					]
			}
	],
	"match": [
			{
					"host": [
							"{{.UpstreamProxySubdomain}}.{{.UpstreamProxyBaseDomain}}"
					]
			}
	],
	"terminal": true
}`

const allTemplate = `
	{{ template "GS" .GlobalSettings }}

	{{ template "DVH" .DefaultVirtualHost }}

	{{ template "ARP" .APIReverseProxySettings }}

	{{ range $erp := .ReverseProxies }}
		{{ template "ERP" $erp }}
	{{ end }}
`

const globalSettingsTemplate = `
{{ define "GS"}}
	{
		grace_period 1s
		auto_https off

		log {
			level {{.LogLevel}}
		}

		admin unix/{{.AdminSocket}} {
			origins localhost
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

const apiReverseProxySettingsTemplate = `
{{ define "ARP"}}
https://{{.ProxyDomain}}:{{.ProxyPort}} {
	tls {{.CertsFile}} {{.KeyFile}} {
		protocols tls1.3
	}
	reverse_proxy {{.APIScheme}}://{{.APITargetHost}}:{{.APITargetPort}}
	log {
		output file {{.ProxyLogFile}}
	}
}
{{ end }}
`
