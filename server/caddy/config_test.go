package caddy_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/server/caddy"
	"github.com/cloudradar-monitoring/rport/share/files"
)

type mockFileSystem struct {
	*files.FileSystem

	ShouldNotExist bool
	CheckedPath    string
}

func (m *mockFileSystem) Exist(path string) (bool, error) {
	m.CheckedPath = path
	if m.ShouldNotExist {
		return false, nil
	}
	return true, nil
}

func TestShouldParseAndValidateCaddyIntegration(t *testing.T) {
	// used when check cert paths
	filesAPI := &mockFileSystem{}

	cfg := &caddy.Config{
		ExecPath: "/usr/bin/caddy",
	}

	// the config checks currently include physical checks for the caddy
	// version. skip the config tests if caddy not installed.
	if !caddyAvailable(t, cfg) {
		t.Skip()
	}

	cases := []struct {
		Name             string
		CaddyConfig      caddy.Config
		ExpectedErrorStr string
		NotConfigured    bool
	}{
		{
			Name: "no error if not configured",
			CaddyConfig: caddy.Config{
				ExecPath:    "",
				HostAddress: "",
				BaseDomain:  "",
				CertFile:    "",
				KeyFile:     "",
			},
			ExpectedErrorStr: "",
			NotConfigured:    true,
		},
		{
			Name: "no error if mandatory values configured",
			CaddyConfig: caddy.Config{
				ExecPath:    "/usr/bin/caddy",
				HostAddress: "0.0.0.0:443",
				BaseDomain:  "tunnels.rport.example.com",
				CertFile:    "../../testdata/certs/tunnels.rport.test.crt",
				KeyFile:     "../../testdata/certs/tunnels.rport.test.key",
			},
			ExpectedErrorStr: "",
		},
		{
			Name: "error if exec path missing",
			CaddyConfig: caddy.Config{
				// ExecPath: "/usr/bin/caddy",
				HostAddress: "0.0.0.0:443",
				BaseDomain:  "tunnels.rport.example.com",
				CertFile:    "../../testdata/certs/tunnels.rport.test.crt",
				KeyFile:     "../../testdata/certs/tunnels.rport.test.key",
			},
			ExpectedErrorStr: caddy.ErrCaddyExecPathMissing.Error(),
		},
		{
			Name: "error if address missing",
			CaddyConfig: caddy.Config{
				ExecPath: "/usr/bin/caddy",
				// HostAddress:   "0.0.0.0:443",
				BaseDomain: "tunnels.rport.example.com",
				CertFile:   "../../testdata/certs/tunnels.rport.test.crt",
				KeyFile:    "../../testdata/certs/tunnels.rport.test.key",
			},
			ExpectedErrorStr: caddy.ErrCaddyTunnelsHostAddressMissing.Error(),
		},
		{
			Name: "error if basedomain missing",
			CaddyConfig: caddy.Config{
				ExecPath:    "/usr/bin/caddy",
				HostAddress: "0.0.0.0:443",
				// BaseDomain: "tunnels.rport.example.com",
				CertFile: "../../testdata/certs/tunnels.rport.test.crt",
				KeyFile:  "../../testdata/certs/tunnels.rport.test.key",
			},
			ExpectedErrorStr: caddy.ErrCaddyTunnelsBaseDomainMissing.Error(),
		},
		{
			Name: "error if cert file missing",
			CaddyConfig: caddy.Config{
				ExecPath:    "/usr/bin/caddy",
				HostAddress: "0.0.0.0:443",
				BaseDomain:  "tunnels.rport.example.com",
				// CertFile: "/var/lib/rport/wildcard.crt",
				KeyFile: "/var/lib/rport/wildcard.key",
			},
			ExpectedErrorStr: caddy.ErrCaddyTunnelsWildcardCertFileMissing.Error(),
		},
		{
			Name: "error if key file missing",
			CaddyConfig: caddy.Config{
				ExecPath:    "/usr/bin/caddy",
				HostAddress: "0.0.0.0:443",
				BaseDomain:  "tunnels.rport.example.com",
				CertFile:    "/var/lib/rport/wildcard.crt",
				// KeyFile:  "/var/lib/rport/wildcard.key",
			},
			ExpectedErrorStr: caddy.ErrCaddyTunnelsWildcardKeyFileMissing.Error(),
		},
		{
			Name: "error if api_hostname set but not api_port",
			CaddyConfig: caddy.Config{
				ExecPath:    "/usr/bin/caddy",
				HostAddress: "0.0.0.0:443",
				BaseDomain:  "tunnels.rport.example.com",
				CertFile:    "../../testdata/certs/tunnels.rport.test.crt",
				KeyFile:     "../../testdata/certs/tunnels.rport.test.key",
				APIHostname: "api.rport.test",
				// APIPort: "443",
			},
			ExpectedErrorStr: caddy.ErrCaddyMissingAPIPort.Error(),
		},
		{
			Name: "error if api_port set but not api_hostname",
			CaddyConfig: caddy.Config{
				ExecPath:    "/usr/bin/caddy",
				HostAddress: "0.0.0.0:443",
				BaseDomain:  "tunnels.rport.example.com",
				CertFile:    "../../testdata/certs/tunnels.rport.test.crt",
				KeyFile:     "../../testdata/certs/tunnels.rport.test.key",
				// APIHostname: "api.rport.test",
				APIPort: "443",
			},
			ExpectedErrorStr: caddy.ErrCaddyMissingAPIHostname.Error(),
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			err := tc.CaddyConfig.ParseAndValidate("datadir", "info", filesAPI)
			if tc.ExpectedErrorStr == "" {
				if tc.NotConfigured {
					assert.NoError(t, err)
				} else {
					assert.NoError(t, err)
					assert.True(t, tc.CaddyConfig.Enabled)
				}
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.ExpectedErrorStr)
			}
		})
	}
}

func TestShouldGenerateBaseConf(t *testing.T) {
	cfg := &caddy.Config{
		ExecPath:    "/usr/bin/caddy",
		DataDir:     ".",
		HostAddress: "0.0.0.0:443",
		BaseDomain:  "tunnels.rpdev",
		CertFile:    "proxy_cert_file",
		KeyFile:     "proxy_key_file",
		APICertFile: "api_cert_file",
		APIKeyFile:  "api_key_file",
		APIHostname: "api_hostname",
		APIPort:     "api_port",
	}

	bc, err := cfg.MakeBaseConfig("target_api_port")
	require.NoError(t, err)

	bcBytes, err := cfg.GetBaseConf(bc)
	require.NoError(t, err)

	text := string(bcBytes)

	assert.Contains(t, text, "admin unix/./caddy-admin.sock")
	assert.Contains(t, text, "https://0.0.0.0:443")
	assert.Contains(t, text, "tls proxy_cert_file proxy_key_file {")
	assert.Contains(t, text, "https://api_hostname:api_port")
	assert.Contains(t, text, "tls api_cert_file api_key_file")
	assert.Contains(t, text, "to http://127.0.0.1:target_api_port")
}
