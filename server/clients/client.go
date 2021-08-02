package clients

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/cloudradar-monitoring/rport/server/cgroups"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/collections"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/random"
)

// now is used to stub time.Now in tests
var now = time.Now

type ConnectionState string

const (
	Connected    ConnectionState = "connected"
	Disconnected ConnectionState = "disconnected"
)

// Client represents client connection
type Client struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	OS                     string    `json:"os"`
	OSArch                 string    `json:"os_arch"`
	OSFamily               string    `json:"os_family"`
	OSKernel               string    `json:"os_kernel"`
	OSFullName             string    `json:"os_full_name"`
	OSVersion              string    `json:"os_version"`
	OSVirtualizationSystem string    `json:"os_virtualization_system"`
	OSVirtualizationRole   string    `json:"os_virtualization_role"`
	CPUFamily              string    `json:"cpu_family"`
	CPUModel               string    `json:"cpu_model"`
	CPUModelName           string    `json:"cpu_model_name"`
	CPUVendor              string    `json:"cpu_vendor"`
	NumCPUs                int       `json:"num_cpus"`
	MemoryTotal            uint64    `json:"mem_total"`
	Timezone               string    `json:"timezone"`
	Hostname               string    `json:"hostname"`
	IPv4                   []string  `json:"ipv4"`
	IPv6                   []string  `json:"ipv6"`
	Tags                   []string  `json:"tags"`
	Version                string    `json:"version"`
	Address                string    `json:"address"`
	Tunnels                []*Tunnel `json:"tunnels"`
	// DisconnectedAt is a time when a client was disconnected. If nil - it's connected.
	DisconnectedAt    *time.Time            `json:"disconnected_at"`
	ClientAuthID      string                `json:"client_auth_id"`
	AllowedUserGroups []string              `json:"allowed_user_groups"`
	UpdatesStatus     *models.UpdatesStatus `json:"updates_status"`

	Connection ssh.Conn        `json:"-"`
	Context    context.Context `json:"-"`
	Logger     *chshare.Logger `json:"-"`

	tunnelIDAutoIncrement int64
	lock                  sync.Mutex
}

// Obsolete returns true if a given client was disconnected longer than a given duration.
// If a given duration is nil - returns false.
func (c *Client) Obsolete(duration *time.Duration) bool {
	return duration != nil && c.DisconnectedAt != nil &&
		c.DisconnectedAt.Add(*duration).Before(now())
}

func (c *Client) Lock() {
	c.lock.Lock()
}

func (c *Client) Unlock() {
	c.lock.Unlock()
}

func (c *Client) FindTunnelByRemote(r *chshare.Remote) *Tunnel {
	for _, curr := range c.Tunnels {
		if curr.Equals(r) {
			return curr
		}
	}
	return nil
}

func (c *Client) StartTunnel(r *chshare.Remote, acl *TunnelACL) (*Tunnel, error) {
	t := c.FindTunnelByRemote(r)
	if t != nil {
		return t, nil
	}

	tunnelID := strconv.FormatInt(c.generateNewTunnelID(), 10)
	t = NewTunnel(c.Logger, c.Connection, tunnelID, r, acl)
	autoCloseChan, err := t.Start(c.Context)
	if err != nil {
		return nil, err
	}

	// in case tunnel auto-closed due to inactivity - run background task to remove the tunnel from the list
	// TODO: in case tunnel would be extended to have active/inactive status this wouldn't be needed
	if autoCloseChan != nil {
		go func() {
			select {
			case <-c.Context.Done():
				return
			case <-autoCloseChan:
				c.Lock()
				defer c.Unlock()
				c.removeTunnelByID(t.ID)
				c.Logger.Debugf("tunnel with id=%s removed", t.ID)
			}
		}()
	}

	c.Tunnels = append(c.Tunnels, t)
	return t, nil
}

func (c *Client) TerminateTunnel(t *Tunnel, force bool) error {
	c.Logger.Infof("Terminating tunnel %s (force: %v) ...", t.ID, force)
	err := t.Terminate(force)
	if err != nil {
		return err
	}
	c.removeTunnelByID(t.ID)
	return nil
}

func (c *Client) FindTunnel(id string) *Tunnel {
	for _, curr := range c.Tunnels {
		if curr.ID == id {
			return curr
		}
	}
	return nil
}

func (c *Client) generateNewTunnelID() int64 {
	return atomic.AddInt64(&c.tunnelIDAutoIncrement, 1)
}

func (c *Client) removeTunnelByID(tunnelID string) {
	result := make([]*Tunnel, 0)
	for _, curr := range c.Tunnels {
		if curr.ID != tunnelID {
			result = append(result, curr)
		}
	}
	c.Tunnels = result
}

func (c *Client) Banner() string {
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

func (c *Client) Close() error {
	// The tunnels are closed automatically when ssh connection is closed.
	return c.Connection.Close()
}

func (c *Client) BelongsToOneOf(groups []*cgroups.ClientGroup) bool {
	for _, cur := range groups {
		if c.BelongsTo(cur) {
			return true
		}
	}
	return false
}

func (c *Client) BelongsTo(group *cgroups.ClientGroup) bool {
	p := group.Params
	if p.HasNoParams() {
		return false
	}
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

func (c *Client) ConnectionState() ConnectionState {
	if c.DisconnectedAt == nil {
		return Connected
	}
	return Disconnected
}

// HasAccess returns true if at least one of given user groups has access to a current client.
func (c *Client) HasAccess(userGroups []string) bool {
	allowedGroups := collections.ConvertToStringBoolMap(c.AllowedUserGroups)
	for _, curUserGroup := range userGroups {
		if curUserGroup == users.Administrators || allowedGroups.Has(curUserGroup) {
			return true
		}
	}

	return false
}

// NewClientID generates a new client ID.
func NewClientID() string {
	return random.UUID4()
}
