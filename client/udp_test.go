package chclient

import (
	"io"
	"log"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/realvnc-labs/rport/share/comm"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/test"
)

func TestUDPHandler(t *testing.T) {
	mockServer := newMockUDPServer(t)
	logger := logger.NewLogger("udp-handler-test", logger.LogOutput{File: os.Stdout}, logger.LogLevelDebug)
	serverChannel, clientChannel := test.NewMockChannel()
	channel := comm.NewUDPChannel(clientChannel)
	handler := newUDPHandler(logger, mockServer.LocalAddr().String())
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err := handler.Handle(serverChannel)
		require.NoError(t, err)
		wg.Done()
	}()
	addr1, err := net.ResolveUDPAddr("udp", "127.0.0.1:12345")
	require.NoError(t, err)
	addr2, err := net.ResolveUDPAddr("udp", "127.0.0.1:23456")
	require.NoError(t, err)

	// Check responses come back to correct address
	err = channel.Encode(addr1, []byte("123"))
	require.NoError(t, err)
	time.Sleep(20 * time.Millisecond)
	err = channel.Encode(addr2, []byte("456"))
	require.NoError(t, err)

	addr, data, err := channel.Decode()
	require.NoError(t, err)
	assert.Equal(t, addr1, addr)
	assert.Equal(t, []byte("123"), data)

	addr, data, err = channel.Decode()
	require.NoError(t, err)
	assert.Equal(t, addr2, addr)
	assert.Equal(t, []byte("456"), data)

	addr, data, err = channel.Decode()
	require.NoError(t, err)
	assert.Equal(t, addr1, addr)
	assert.Equal(t, []byte("123"), data)

	addr, data, err = channel.Decode()
	require.NoError(t, err)
	assert.Equal(t, addr2, addr)
	assert.Equal(t, []byte("456"), data)

	clientChannel.Close()
	serverChannel.Close()

	wg.Wait()
}

type mockUDPServer struct {
	*net.UDPConn
}

func newMockUDPServer(t *testing.T) *mockUDPServer {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)
	conn, err := net.ListenUDP("udp", addr)
	require.NoError(t, err)
	s := &mockUDPServer{
		UDPConn: conn,
	}
	go func() {
		err := s.handle()
		require.NoError(t, err)
	}()
	return s
}

func (s *mockUDPServer) handle() error {
	for {
		buffer := make([]byte, 1024)

		n, addr, err := s.ReadFromUDP(buffer)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		_, err = s.WriteToUDP(buffer[:n], addr)
		if err != nil {
			return err
		}

		time.AfterFunc(40*time.Millisecond, func() {
			_, err := s.WriteToUDP(buffer[:n], addr)
			if err != nil {
				log.Printf("Error in second udp write: %v", err)
			}
		})
	}
}
