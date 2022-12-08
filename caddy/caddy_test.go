package caddy_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/caddy"
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

	ctx := context.Background()

	version, err := caddy.GetServerVersion(ctx, cfg)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, version, 2)
}

func TestShouldStartCaddyServer(t *testing.T) {
	cfg := &caddy.Config{
		ExecPath: "/usr/bin/caddy",
		DataDir:  ".",
	}

	if !caddyAvailable(t, cfg) {
		t.Skip()
	}

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	caddyServer := caddy.NewCaddyServer(cfg, testLog, errCh)

	go caddyServer.Start(ctx)

	time.AfterFunc(500*time.Millisecond, func() {
		cancel()
	})

	err := <-errCh
	assert.EqualError(t, err, "signal: killed")
}

func TestShouldStopCaddyServerWhenCtxCancelled(t *testing.T) {
	t.Skip()
}
