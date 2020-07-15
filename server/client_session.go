package chserver

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"golang.org/x/crypto/ssh"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

func GetSessionID(sshConn ssh.ConnMetadata) string {
	return fmt.Sprintf("%x", sshConn.SessionID())
}

type ClientSession struct {
	ID      string            `json:"id"`
	Version string            `json:"version"`
	Address string            `json:"address"`
	Remotes []*chshare.Remote `json:"remotes"`

	Connection ssh.Conn        `json:"-"`
	Context    context.Context `json:"-"`
	User       *chshare.User   `json:"-"`
	Logger     *chshare.Logger `json:"-"`

	tunnelIDAutoIncrement int32
	lock                  sync.Mutex
}

func (c *ClientSession) Lock() {
	c.lock.Lock()
}

func (c *ClientSession) Unlock() {
	c.lock.Unlock()
}

func (c *ClientSession) HasRemote(r *chshare.Remote) bool {
	for _, curr := range c.Remotes {
		if curr.Equals(r) {
			return true
		}
	}
	return false
}

func (c *ClientSession) StartRemoteTunnel(r *chshare.Remote) error {
	proxy := chshare.NewTCPProxy(c.Logger, func() ssh.Conn { return c.Connection }, c.generateNewTunnelID(), r)
	err := proxy.Start(c.Context)
	if err != nil {
		return err
	}
	c.Remotes = append(c.Remotes, r)
	return nil
}

func (c *ClientSession) generateNewTunnelID() int32 {
	return atomic.AddInt32(&c.tunnelIDAutoIncrement, 1)
}
