package chserver

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"golang.org/x/crypto/ssh"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

func GetSessionID(sshConn ssh.ConnMetadata) string {
	return fmt.Sprintf("%x", sshConn.SessionID())
}

// ClientSession represents active client connection
type ClientSession struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Version string    `json:"version"`
	Address string    `json:"address"`
	Tunnels []*Tunnel `json:"tunnels"`

	Connection ssh.Conn        `json:"-"`
	Context    context.Context `json:"-"`
	User       *chshare.User   `json:"-"`
	Logger     *chshare.Logger `json:"-"`

	tunnelIDAutoIncrement int64
	lock                  sync.Mutex
}

func (c *ClientSession) Lock() {
	c.lock.Lock()
}

func (c *ClientSession) Unlock() {
	c.lock.Unlock()
}

func (c *ClientSession) HasRemote(r *chshare.Remote) bool {
	for _, curr := range c.Tunnels {
		if curr.Equals(r) {
			return true
		}
	}
	return false
}

func (c *ClientSession) StartRemoteTunnel(r *chshare.Remote) (string, error) {
	tunnelID := strconv.FormatInt(c.generateNewTunnelID(), 10)
	t := NewTunnel(c.Logger, c.Connection, tunnelID, r)
	err := t.Start(c.Context)
	if err != nil {
		return "", err
	}
	c.Tunnels = append(c.Tunnels, t)
	return tunnelID, nil
}

func (c *ClientSession) TerminateTunnel(t *Tunnel) {
	c.Logger.Infof("Terminating tunnel %s...", t.ID)
	t.Terminate()
	c.removeTunnel(t)
}

func (c *ClientSession) FindTunnel(id string) *Tunnel {
	for _, curr := range c.Tunnels {
		if curr.ID == id {
			return curr
		}
	}
	return nil
}

func (c *ClientSession) generateNewTunnelID() int64 {
	return atomic.AddInt64(&c.tunnelIDAutoIncrement, 1)
}

func (c *ClientSession) removeTunnel(t *Tunnel) {
	result := make([]*Tunnel, 0)
	for _, curr := range c.Tunnels {
		if curr.ID != t.ID {
			result = append(result, curr)
		}
	}
	c.Tunnels = result
}
