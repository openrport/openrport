package caddy_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/caddy"
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
		BaseDomain:       "tunnels.rpdev.lan",
		CertFile:         "../testdata/certs/tunnels.rpdev.lan.crt",
		KeyFile:          "../testdata/certs/tunnels.rpdev.lan.key",
	}

	chCfg := &chconfig.Config{
		API: chconfig.APIConfig{
			Address:            "0.0.0.0:3000",
			DomainBasedAddress: "",
			CertFile:           cfg.CertFile,
			KeyFile:            cfg.KeyFile,
		},
	}

	if !caddyAvailable(t, cfg) {
		t.Skip()
	}

	ctx, cancel := context.WithCancel(context.Background())

	_, err := chCfg.WriteCaddyBaseConfig(cfg)
	require.NoError(t, err)

	errCh := make(chan error, 1)
	caddyServer := caddy.NewCaddyServer(cfg, testLog, errCh)

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
	}

	bc, err := cfg.MakeBaseConfig("api_cert_file", "api_key_file", "127.0.0.0:3000", "api.rpdev:443")
	require.NoError(t, err)

	bcBytes, err := cfg.GetBaseConf(bc)
	require.NoError(t, err)

	text := string(bcBytes)

	assert.Contains(t, text, "admin unix/./caddy-admin.sock")
	assert.Contains(t, text, "https://0.0.0.0:443")
	assert.Contains(t, text, "tls proxy_cert_file proxy_key_file {")
	assert.Contains(t, text, "https://api.rpdev:443")
	assert.Contains(t, text, "tls api_cert_file api_key_file")
	assert.Contains(t, text, "to https://127.0.0.0:3000")
}
