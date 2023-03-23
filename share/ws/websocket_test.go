package ws_test

import (
	"os"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/ws"
)

func TestAutoClose(t *testing.T) {
	log := logger.NewLogger("websocket-test", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)
	mockConn := &connMock{}

	ws := ws.NewConcurrentWebSocket(mockConn, log)
	ws.SetWritesBeforeClose(2)

	err := ws.WriteMessage(websocket.TextMessage, []byte("test"))
	require.NoError(t, err)

	assert.False(t, mockConn.Closed)

	err = ws.WriteNonFinalJSON(1)
	require.NoError(t, err)

	assert.False(t, mockConn.Closed)

	err = ws.WriteJSON(1)
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
