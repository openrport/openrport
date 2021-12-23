package clients

import (
	"context"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/test"
)

func TestTunnelUDP(t *testing.T) {
	remote := models.Remote{}
	logger := logger.NewLogger("udp-handler-test", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)
	tunnel := newTunnelUDP(logger, nil, remote, nil)
	serverChannel, clientChannel := test.NewMockChannel()
	channel := comm.NewUDPChannel(clientChannel)
	_, err := tunnel.start(context.Background(), serverChannel)
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
}
