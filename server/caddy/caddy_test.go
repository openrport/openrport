package caddy_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/realvnc-labs/rport/server/caddy"
	"github.com/realvnc-labs/rport/server/chconfig"
	"github.com/realvnc-labs/rport/share/logger"
)

var testLog = logger.NewLogger("caddy", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

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
	assert.EqualError(t, err, "context canceled")
}
