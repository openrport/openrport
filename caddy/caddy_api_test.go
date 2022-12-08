package caddy_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/caddy"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

var testLog = logger.NewLogger("caddy", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)

func TestShouldAddRouteToCaddyServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer withWait(cancel)

	setupNewCaddyServer(ctx, t)

	nrr := makeTestNewRouteRequest()

	res, err := caddy.AddRoute(ctx, nrr)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestShouldDeleteRouteFromCaddyServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer withWait(cancel)

	setupNewCaddyServer(ctx, t)

	nrr := makeTestNewRouteRequest()

	res, err := caddy.AddRoute(ctx, nrr)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	res, err = caddy.DeleteRoute(ctx, nrr.RouteID)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestShouldErrorWhenCaddyNotAvailable(t *testing.T) {
	ctx := context.Background()

	nrr := makeTestNewRouteRequest()

	_, err := caddy.AddRoute(ctx, nrr)
	assert.ErrorContains(t, err, "unable to send request")
}

func makeTestNewRouteRequest() (nrr *caddy.NewRouteRequest) {
	nrr = &caddy.NewRouteRequest{
		RouteID:                 "1111",
		TargetTunnelHost:        "127.0.0.1",
		TargetTunnelPort:        "5555",
		UpstreamProxySubdomain:  "1111",
		UpstreamProxyBaseDomain: "tunnels.rpdev",
	}

	return nrr
}

func setupNewCaddyServer(ctx context.Context, t *testing.T) {
	t.Helper()

	cfg := &caddy.Config{
		ExecPath: "/usr/bin/caddy",
		DataDir:  ".",
	}

	if !caddyAvailable(t, cfg) {
		t.Skip("caddy server not available")
	}

	caddyServer := caddy.NewCaddyServer(cfg, testLog, nil)
	go caddyServer.Start(ctx)

	// allow time for the server start to settle
	time.Sleep(100 * time.Millisecond)
}

func withWait(cancel context.CancelFunc) {
	cancel()
	// give the server time to receive the cancel
	time.Sleep(10 * time.Millisecond)
}
