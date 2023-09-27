package clienttunnel_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openrport/openrport/server/clients/clienttunnel"
)

func TestInternalTunnelProxyConfigParseAndValidate(t *testing.T) {
	testCases := []struct {
		Name            string
		Config          clienttunnel.InternalTunnelProxyConfig
		ExpectedError   string
		ExpectedEnabled bool
	}{
		{
			Name: "not enabled",
			Config: clienttunnel.InternalTunnelProxyConfig{
				Host: "example.com",
			},
			ExpectedEnabled: false,
		},
		{
			Name: "enabled with acme",
			Config: clienttunnel.InternalTunnelProxyConfig{
				Host:       "example.com",
				EnableAcme: true,
			},
			ExpectedEnabled: true,
		},
		{
			Name: "enabled with certs",
			Config: clienttunnel.InternalTunnelProxyConfig{
				Host:     "example.com",
				CertFile: "../../../testdata/certs/tunnels.rport.test.crt",
				KeyFile:  "../../../testdata/certs/tunnels.rport.test.key",
			},
			ExpectedEnabled: true,
		},
		{
			Name: "cert file only",
			Config: clienttunnel.InternalTunnelProxyConfig{
				Host:     "example.com",
				CertFile: "../../../testdata/certs/tunnels.rport.test.crt",
			},
			ExpectedError: "when 'tunnel_proxy_cert_file' is set, 'tunnel_proxy_key_file' must be set as well",
		},
		{
			Name: "key file only",
			Config: clienttunnel.InternalTunnelProxyConfig{
				Host:    "example.com",
				KeyFile: "../../../testdata/certs/tunnels.rport.test.key",
			},
			ExpectedError: "when 'tunnel_proxy_key_file' is set, 'tunnel_proxy_cert_file' must be set as well",
		},
		{
			Name: "acme and certs",
			Config: clienttunnel.InternalTunnelProxyConfig{
				Host:       "example.com",
				EnableAcme: true,
				CertFile:   "../../../testdata/certs/tunnels.rport.test.crt",
				KeyFile:    "../../../testdata/certs/tunnels.rport.test.key",
			},
			ExpectedError: "tunnel_proxy_cert_file, tunnel_proxy_key_file and tunnel_enable_acme cannot be used together",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			err := tc.Config.ParseAndValidate()
			if tc.ExpectedError != "" {
				assert.EqualError(t, err, tc.ExpectedError)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.ExpectedEnabled, tc.Config.Enabled)
		})

	}
}
