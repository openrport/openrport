package clienttunnel

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/realvnc-labs/rport/share/comm"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/models"
	"github.com/realvnc-labs/rport/share/test"
)

func TestTunnelUDP(t *testing.T) {
	udpReadTimeout = time.Millisecond
	remote := models.Remote{}
	logger := logger.NewLogger("udp-handler-test", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)
	tunnel := newTunnelUDP(logger, nil, remote, nil)
	serverChannel, clientChannel := test.NewMockChannel()
	channel := comm.NewUDPChannel(clientChannel)
	err := tunnel.start(context.Background(), serverChannel)
	require.NoError(t, err)
	conn, err := net.Dial("udp", tunnel.conn.LocalAddr().String())
	require.NoError(t, err)

	// udp send
	_, err = conn.Write([]byte("abc"))
	require.NoError(t, err)

	addr, data, err := channel.Decode()
	require.NoError(t, err)
	assert.Equal(t, []byte("abc"), data)
	assert.Equal(t, conn.LocalAddr(), addr)

	// udp receive
	err = channel.Encode(conn.LocalAddr().(*net.UDPAddr), []byte("123"))
	require.NoError(t, err)

	buffer := make([]byte, 128)
	n, err := conn.Read(buffer)
	require.NoError(t, err)
	assert.Equal(t, []byte("123"), buffer[:n])

	err = tunnel.Terminate(false)
	require.NoError(t, err)

	assert.WithinDuration(t, time.Now(), tunnel.LastActive(), 10*time.Millisecond)
}

func TestTunnelUDPWithACL(t *testing.T) {
	udpReadTimeout = time.Millisecond
	remote := models.Remote{}
	logger := logger.NewLogger("udp-handler-test", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)
	acl, err := ParseTunnelACL("127.0.0.2")
	require.NoError(t, err)
	tunnel := newTunnelUDP(logger, nil, remote, acl)
	serverChannel, clientChannel := test.NewMockChannel()
	channel := comm.NewUDPChannel(clientChannel)
	local1, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)
	local2, err := net.ResolveUDPAddr("udp", "127.0.0.2:0")
	require.NoError(t, err)
	tunnel.idleTimeout = time.Millisecond * 10
	err = tunnel.start(context.Background(), serverChannel)
	require.NoError(t, err)

	// send from local1 - not allowed
	conn, err := net.DialUDP("udp", local1, tunnel.conn.LocalAddr().(*net.UDPAddr))
	require.NoError(t, err)
	_, err = conn.Write([]byte("abc"))
	require.NoError(t, err)

	// send from local2 - allowed
	conn, err = net.DialUDP("udp", local2, tunnel.conn.LocalAddr().(*net.UDPAddr))
	require.NoError(t, err)
	_, err = conn.Write([]byte("def"))
	require.NoError(t, err)

	addr, data, err := channel.Decode()
	require.NoError(t, err)
	assert.Equal(t, []byte("def"), data)
	assert.Equal(t, conn.LocalAddr(), addr)

	// update ACL
	acl2, err := ParseTunnelACL("127.0.0.1")
	require.NoError(t, err)
	tunnel.SetACL(acl2)

	// send from local2 - not allowed
	conn, err = net.DialUDP("udp", local2, tunnel.conn.LocalAddr().(*net.UDPAddr))
	require.NoError(t, err)
	_, err = conn.Write([]byte("abc"))
	require.NoError(t, err)

	// send from local1 - allowed
	conn, err = net.DialUDP("udp", local1, tunnel.conn.LocalAddr().(*net.UDPAddr))
	require.NoError(t, err)
	_, err = conn.Write([]byte("def"))
	require.NoError(t, err)

	addr, data, err = channel.Decode()
	require.NoError(t, err)
	assert.Equal(t, []byte("def"), data)
	assert.Equal(t, conn.LocalAddr(), addr)
}
