package sessions

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/cgroups"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

// now is used to stub time.Now in tests
var now = time.Now

func GetSessionID(sshConn ssh.ConnMetadata) string {
	return fmt.Sprintf("%x", sshConn.SessionID())
}

// ClientSession represents client connection
type ClientSession struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	OS       string    `json:"os"`
	OSArch   string    `json:"os_arch"`
	OSFamily string    `json:"os_family"`
	OSKernel string    `json:"os_kernel"`
	Hostname string    `json:"hostname"`
	IPv4     []string  `json:"ipv4"`
	IPv6     []string  `json:"ipv6"`
	Tags     []string  `json:"tags"`
	Version  string    `json:"version"`
	Address  string    `json:"address"`
	Tunnels  []*Tunnel `json:"tunnels"`
	// Disconnected is a time when a client session was disconnected. If nil - it's connected.
	Disconnected *time.Time `json:"disconnected,omitempty"`
	ClientAuthID string     `json:"client_auth_id"`

	Connection ssh.Conn        `json:"-"`
	Context    context.Context `json:"-"`
	Logger     *chshare.Logger `json:"-"`

	tunnelIDAutoIncrement int64
	lock                  sync.Mutex
}

// Obsolete returns true if a given client session was disconnected longer than a given duration.
// If a given duration is nil - returns false.
func (c *ClientSession) Obsolete(duration *time.Duration) bool {
	return duration != nil && c.Disconnected != nil &&
		c.Disconnected.Add(*duration).Before(now())
}

func (c *ClientSession) Lock() {
	c.lock.Lock()
}

func (c *ClientSession) Unlock() {
	c.lock.Unlock()
}

func (c *ClientSession) FindTunnelByRemote(r *chshare.Remote) *Tunnel {
	for _, curr := range c.Tunnels {
		if curr.Equals(r) {
			return curr
		}
	}
	return nil
}

func (c *ClientSession) StartTunnel(r *chshare.Remote, acl *TunnelACL) (*Tunnel, error) {
	t := c.FindTunnelByRemote(r)
	if t != nil {
		return t, nil
	}

	tunnelID := strconv.FormatInt(c.generateNewTunnelID(), 10)
	t = NewTunnel(c.Logger, c.Connection, tunnelID, r, acl)
	err := t.Start(c.Context)
	if err != nil {
		return nil, err
	}
	c.Tunnels = append(c.Tunnels, t)
	return t, nil
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

func (c *ClientSession) Banner() string {
	banner := c.ID
	if c.Name != "" {
		banner += " (" + c.Name + ")"
	}
	if len(c.Tags) != 0 {
		for _, t := range c.Tags {
			banner += " #" + t
		}
	}
	return banner
}

func (c *ClientSession) Close() error {
	// The tunnels are closed automatically when ssh connection is closed.
	return c.Connection.Close()
}

func (c *ClientSession) BelongsToOneOf(groups []*cgroups.ClientGroup) bool {
	for _, cur := range groups {
		if c.belongsTo(cur) {
			return true
		}
	}
	return false
}

func (c *ClientSession) belongsTo(group *cgroups.ClientGroup) bool {
	p := group.Params
	if !p.ClientID.MatchesOneOf(c.ID) {
		return false
	}
	if !p.Name.MatchesOneOf(c.Name) {
		return false
	}
	if !p.OS.MatchesOneOf(c.OS) {
		return false
	}
	if !p.OSArch.MatchesOneOf(c.OSArch) {
		return false
	}
	if !p.OSFamily.MatchesOneOf(c.OSFamily) {
		return false
	}
	if !p.OSKernel.MatchesOneOf(c.OSKernel) {
		return false
	}
	if !p.Hostname.MatchesOneOf(c.Hostname) {
		return false
	}
	if !p.IPv4.MatchesOneOf(c.IPv4...) {
		return false
	}
	if !p.IPv6.MatchesOneOf(c.IPv6...) {
		return false
	}
	if !p.Tag.MatchesOneOf(c.Tags...) {
		return false
	}
	if !p.Version.MatchesOneOf(c.Version) {
		return false
	}
	if !p.Address.MatchesOneOf(c.Address) {
		return false
	}
	if !p.ClientAuthID.MatchesOneOf(c.ClientAuthID) {
		return false
	}
	return true
}
