package caddy_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/server/caddy"
	"github.com/cloudradar-monitoring/rport/server/chconfig"
)

func caddyAvailable(t *testing.T, cfg *caddy.Config) (available bool) {
	t.Helper()

	_, err := os.Stat(cfg.ExecPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		require.NoError(t, err)
	}
	return true
}

func TestShouldGetCaddyServerVersion(t *testing.T) {
	cfg := &caddy.Config{
		ExecPath: "/usr/bin/caddy",
	}

	if !caddyAvailable(t, cfg) {
		t.Skip()
	}

	version, err := caddy.GetExecVersion(cfg)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, version, 2)
}

func TestShouldStartCaddyServer(t *testing.T) {
	cfg := &caddy.Config{
		ExecPath:         "/usr/bin/caddy",
		DataDir:          "/tmp",
		BaseConfFilename: "caddy-base.conf",
		HostAddress:      "0.0.0.0:8443",
		BaseDomain:       "tunnels.rport.test",
		CertFile:         "../../testdata/certs/tunnels.rport.test.crt",
		KeyFile:          "../../testdata/certs/tunnels.rport.test.key",
		APIHostname:      "api.rport.test",
		APIPort:          "8443",
		APICertFile:      "../../testdata/certs/api.rport.test.crt",
		APIKeyFile:       "../../testdata/certs/api.rport.test.key",
	}

	chCfg := &chconfig.Config{
		API: chconfig.APIConfig{
			Address: "0.0.0.0:3000",
		},
	}

	if !caddyAvailable(t, cfg) {
		t.Skip()
	}

	ctx, cancel := context.WithCancel(context.Background())

	_, err := chCfg.WriteCaddyBaseConfig(cfg)
	require.NoError(t, err)

	caddyServer := caddy.NewCaddyServer(cfg, testLog)

	err = caddyServer.Start(ctx)
	require.NoError(t, err)

	time.AfterFunc(500*time.Millisecond, func() {
		cancel()
	})

	err = caddyServer.Wait()
	assert.EqualError(t, err, "signal: killed")
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
