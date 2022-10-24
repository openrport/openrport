package test

import (
	"net"
	"sync"

	"golang.org/x/crypto/ssh"
)

type ConnMock struct {
	ssh.Conn
	mu          sync.Mutex
	DoneChannel chan bool

	ReturnOk              bool
	ReturnResponsePayload []byte
	ReturnErr             error
	ReturnRemoteAddr      net.Addr

	inputRequestName string
	inputWantReply   bool
	inputPayload     []byte

	ChannelMocks map[string]*ChannelMock
}

func NewConnMock() *ConnMock {
	return &ConnMock{}
}

func (c *ConnMock) SendRequest(name string, wantReply bool, payload []byte) (bool, []byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inputRequestName = name
	c.inputWantReply = wantReply
	c.inputPayload = payload
	if c.DoneChannel != nil {
		c.DoneChannel <- true
	}
	return c.ReturnOk, c.ReturnResponsePayload, c.ReturnErr
}

func (c *ConnMock) InputSendRequest() (name string, wantReply bool, payload []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.inputRequestName, c.inputWantReply, c.inputPayload
}

func (c *ConnMock) OpenChannel(name string, data []byte) (ssh.Channel, <-chan *ssh.Request, error) {
	ch := &ChannelMock{}
	if c.ChannelMocks == nil {
		c.ChannelMocks = make(map[string]*ChannelMock)
	}
	c.ChannelMocks[name] = ch
	return ch, nil, nil
}

func (c *ConnMock) RemoteAddr() net.Addr {
	return c.ReturnRemoteAddr
}

type ChannelMock struct {
	ssh.Channel

	Writes []string
}

func (c *ChannelMock) Write(d []byte) (int, error) {
	c.Writes = append(c.Writes, string(d))
	return len(d), nil
}

func (c *ChannelMock) Close() error { return nil }
