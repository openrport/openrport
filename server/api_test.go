package chserver

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/db/sqlite"
	"github.com/cloudradar-monitoring/rport/server/api/session"
	chshare "github.com/cloudradar-monitoring/rport/share/logger"
)

var testLog = chshare.NewLogger("api-listener-test", chshare.LogOutput{File: os.Stdout}, chshare.LogLevelDebug)
var hour = time.Hour

var DataSourceOptions = sqlite.DataSourceOptions{WALEnabled: false}

type mockConnection struct {
	ssh.Conn
	closed bool
}

func (m *mockConnection) Close() error {
	m.closed = true
	return nil
}

func newEmptyAPISessionCache(t *testing.T) *session.Cache {
	p, err := session.NewSqliteProvider(":memory:", DataSourceOptions)
	require.NoError(t, err)
	c, err := session.NewCache(context.Background(), p, defaultTokenLifetime, cleanupAPISessionsInterval)
	require.NoError(t, err)
	return c
}
