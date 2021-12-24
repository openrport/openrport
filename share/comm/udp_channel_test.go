package comm

import (
	"bytes"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUDPChannel(t *testing.T) {
	buffer := &bytes.Buffer{}
	channel := NewUDPChannel(buffer)
	data := []byte("test123")
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:12345")
	require.NoError(t, err)

	err = channel.Encode(addr, data)
	require.NoError(t, err)

	receivedAddr, receivedData, err := channel.Decode()
	require.NoError(t, err)

	assert.Equal(t, addr, receivedAddr)
	assert.Equal(t, data, receivedData)
}
