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

	"github.com/realvnc-labs/rport/server/clients"
	"github.com/realvnc-labs/rport/server/clients/clientdata"
	"github.com/realvnc-labs/rport/share/comm"
	"github.com/realvnc-labs/rport/share/logger"
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

	c1 := clientdata.Client{}
	c1.SetID("1")
	c1.SetClientAuthID("1")
	c1.SetConnection(connSuccess)
	c1.Logger = myTestLog

	c2 := clientdata.Client{}
	c2.SetID("2")
	c2.SetClientAuthID("2")
	c2.SetConnection(connSuccess)
	c2.SetLastHeartbeatAt(&now)
	c2.Logger = myTestLog

	c3 := clientdata.Client{}
	c3.SetID("3")
	c3.SetClientAuthID("3")
	c3.SetConnection(connFailure)
	c3.Logger = myTestLog

	c4 := clientdata.Client{}
	c4.SetID("4")
	c4.SetClientAuthID("4")
	c4.SetConnection(connTimeout)
	c4.Logger = myTestLog

	cr := clients.NewClientRepository([]*clientdata.Client{&c1, &c2, &c3, &c4}, nil, myTestLog)
	task := NewClientsStatusCheckTask(myTestLog, cr, 120*time.Second, timeout)

	// Check the last heartbeat of c1 has changed due to the ping sent
	err = task.Run(context.Background())
	assert.NoError(t, err)
	tcl1, err := cr.GetByID("1")
	assert.NoError(t, err)
	assert.IsType(t, &time.Time{}, tcl1.GetLastHeartbeatAt())
	t.Logf("tcl1: LastHeartbeatAt: %s", tcl1.GetLastHeartbeatAt())

	// Check the last heartbeat of c2 has not changed because the task must skip this client
	tcl2, err := cr.GetByID("2")
	assert.NoError(t, err)
	assert.Equal(t, &now, tcl2.GetLastHeartbeatAt(), "LastHeartbeatAt of tcl2 must not change")
	t.Logf("tcl2: LastHeartbeatAt: %s", tcl2.GetLastHeartbeatAt())

	// Check the status of c3 changed to disconnected
	tcl3, err := cr.GetByID("3")
	assert.NoError(t, err)
	assert.NotNil(t, tcl3.GetDisconnectedAt())
	assert.Equal(t, "disconnected", string(tcl3.CalculateConnectionState()))
	t.Logf("tcl3: GetDisconnectedAt(): %s", tcl3.GetDisconnectedAt())

	// Check the status of c4 changed to disconnected caused by a timeout
	tcl4, err := cr.GetByID("4")
	assert.NoError(t, err)
	assert.NotNil(t, tcl4.GetDisconnectedAt())
	assert.Equal(t, "disconnected", string(tcl4.CalculateConnectionState()))
	t.Logf("tcl4: DisconnectedAt: %s", tcl4.GetDisconnectedAt())
	log, err := os.ReadFile(logfile)
	assert.NoError(t, err, "error reading log file")
	assert.Contains(t, string(log), fmt.Sprintf("ping to  [4] failed: conn.SendRequest(ping), timeout %s exceeded", timeout))
}
