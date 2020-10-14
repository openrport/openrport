package test

import (
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

	inputRequestName string
	inputWantReply   bool
	inputPayload     []byte
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
