package ws_test

import (
	"os"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openrport/openrport/share/logger"
	"github.com/openrport/openrport/share/ws"
)

func TestAutoClose(t *testing.T) {
	log := logger.NewLogger("websocket-test", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)
	mockConn := &connMock{}

	socket := ws.NewConcurrentWebSocket(mockConn, log)
	socket.SetWritesBeforeClose(2)

	err := socket.WriteMessage(websocket.TextMessage, []byte("test"))
	require.NoError(t, err)

	assert.False(t, mockConn.Closed)

	err = socket.WriteNonFinalJSON(1)
	require.NoError(t, err)

	assert.False(t, mockConn.Closed)

	err = socket.WriteJSON(1)
	require.NoError(t, err)

	assert.True(t, mockConn.Closed)
}

type connMock struct {
	ws.Conn

	Closed bool
}

func (c *connMock) Close() error {
	c.Closed = true
	return nil
}

func (c *connMock) WriteMessage(messageType int, data []byte) error {
	return nil
}

func (c *connMock) WriteJSON(jsonOutboundMsg interface{}) error {
	return nil
}
