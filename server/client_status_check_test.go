package chserver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

type mockSSHConn struct {
	ssh.Conn
	shallFail    bool
	shallTimeout bool
}

func (m mockSSHConn) SendRequest(name string, wantReply bool, payload []byte) (bool, []byte, error) {
	if name != comm.RequestTypePing {
		return false, []byte(""), errors.New("bad request: only ping simulated")
	}
	if m.shallFail {
		return false, []byte("null"), errors.New("EOF")
	}
	if m.shallTimeout {
		time.Sleep(2 * time.Millisecond)
	}
	return true, []byte(""), nil
}

func (m mockSSHConn) Close() error {
	return nil
}

func TestClientsStatusDeterminationTask(t *testing.T) {
	logfile := t.TempDir() + "/test.log"
	l, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0444)
	assert.NoError(t, err, "error creating log file")
	defer l.Close()
	var (
		myTestLog = logger.NewLogger(
			"server",
			logger.LogOutput{File: l},
			logger.LogLevelDebug,
		)
		connSuccess ssh.Conn = mockSSHConn{
			shallFail:    false,
			shallTimeout: false,
		}
		connFailure ssh.Conn = mockSSHConn{
			shallFail:    true,
			shallTimeout: false,
		}
		connTimeout ssh.Conn = mockSSHConn{
			shallFail:    false,
			shallTimeout: true,
		}
		timeout = 1 * time.Millisecond
	)
	now := time.Now()
	cr := clients.NewClientRepository([]*clients.Client{
		{
			ID:           "1",
			ClientAuthID: "1",
			Connection:   connSuccess,
		},
		{
			ID:              "2",
			ClientAuthID:    "2",
			Connection:      connSuccess,
			LastHeartbeatAt: &now,
		},
		{
			ID:           "3",
			ClientAuthID: "3",
			Connection:   connFailure,
		},
		{
			ID:           "4",
			ClientAuthID: "4",
			Connection:   connTimeout,
		},
	}, nil, myTestLog)
	task := NewClientsStatusCheckTask(myTestLog, cr, 120*time.Second, timeout)

	// Check the last heartbeat of c1 has changed due to the ping sent
	err = task.Run(context.Background())
	assert.NoError(t, err)
	c1, err := cr.GetByID("1")
	assert.NoError(t, err)
	assert.IsType(t, &time.Time{}, c1.LastHeartbeatAt)
	t.Logf("c1: LastHeartbeatAt: %s", c1.LastHeartbeatAt)

	// Check the last heartbeat of c2 has not changed because the task must skip this client
	c2, err := cr.GetByID("2")
	assert.NoError(t, err)
	assert.Equal(t, &now, c2.LastHeartbeatAt, "LastHeartbeatAt of c2 must not change")
	t.Logf("c2: LastHeartbeatAt: %s", c2.LastHeartbeatAt)

	// Check the status of c3 changed to disconnected
	c3, err := cr.GetByID("3")
	assert.NoError(t, err)
	assert.NotNil(t, c3.DisconnectedAt)
	assert.Equal(t, "disconnected", string(c3.ConnectionState()))
	t.Logf("c3: DisconnectedAt: %s", c3.DisconnectedAt)

	// Check the status of c4 changed to disconnected caused by a timeout
	c4, err := cr.GetByID("4")
	assert.NoError(t, err)
	assert.NotNil(t, c4.DisconnectedAt)
	assert.Equal(t, "disconnected", string(c4.ConnectionState()))
	t.Logf("c4: DisconnectedAt: %s", c4.DisconnectedAt)
	log, err := os.ReadFile(logfile)
	assert.NoError(t, err, "error reading log file")
	assert.Contains(t, string(log), fmt.Sprintf("ping to  [4] failed: timeout %s exceeded", timeout))
}
