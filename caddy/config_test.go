package caddy_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudradar-monitoring/rport/caddy"
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
				CertFile:    "/var/lib/rport/wildcard.crt",
				KeyFile:     "/var/lib/rport/wildcard.key",
			},
			ExpectedErrorStr: "",
		},
		{
			Name: "error if exec path missing",
			CaddyConfig: caddy.Config{
				// ExecPath: "/usr/bin/caddy",
				HostAddress: "0.0.0.0:443",
				BaseDomain:  "tunnels.rport.example.com",
				CertFile:    "/var/lib/rport/wildcard.crt",
				KeyFile:     "/var/lib/rport/wildcard.key",
			},
			ExpectedErrorStr: caddy.ErrCaddyExecPathMissing.Error(),
		},
		{
			Name: "error if address missing",
			CaddyConfig: caddy.Config{
				ExecPath: "/usr/bin/caddy",
				// HostAddress:   "0.0.0.0:443",
				BaseDomain: "tunnels.rport.example.com",
				CertFile:   "/var/lib/rport/wildcard.crt",
				KeyFile:    "/var/lib/rport/wildcard.key",
			},
			ExpectedErrorStr: caddy.ErrCaddyTunnelsHostAddressMissing.Error(),
		},
		{
			Name: "error if subdomain missing",
			CaddyConfig: caddy.Config{
				ExecPath:    "/usr/bin/caddy",
				HostAddress: "0.0.0.0:443",
				// BaseDomain: "tunnels.rport.example.com",
				CertFile: "/var/lib/rport/wildcard.crt",
				KeyFile:  "/var/lib/rport/wildcard.key",
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
				CertFile:    "/var/lib/rport/wildcard.crt",
				KeyFile:     "/var/lib/rport/wildcard.key",
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
				CertFile:    "/var/lib/rport/wildcard.crt",
				KeyFile:     "/var/lib/rport/wildcard.key",
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
