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
	"github.com/cloudradar-monitoring/rport/server/clients/clienttunnel"
	"github.com/cloudradar-monitoring/rport/share/clientconfig"
	"github.com/cloudradar-monitoring/rport/share/logger"
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
	// Declare 64-bit integer before 32-bit for alignment when compiling Go on 32-bit ARM platforms
	tunnelIDAutoIncrement int64

	ID                     string                 `json:"id"`
	SessionID              string                 `json:"session_id"`
	Name                   string                 `json:"name"`
	OS                     string                 `json:"os"`
	OSArch                 string                 `json:"os_arch"`
	OSFamily               string                 `json:"os_family"`
	OSKernel               string                 `json:"os_kernel"`
	OSFullName             string                 `json:"os_full_name"`
	OSVersion              string                 `json:"os_version"`
	OSVirtualizationSystem string                 `json:"os_virtualization_system"`
	OSVirtualizationRole   string                 `json:"os_virtualization_role"`
	CPUFamily              string                 `json:"cpu_family"`
	CPUModel               string                 `json:"cpu_model"`
	CPUModelName           string                 `json:"cpu_model_name"`
	CPUVendor              string                 `json:"cpu_vendor"`
	NumCPUs                int                    `json:"num_cpus"`
	MemoryTotal            uint64                 `json:"mem_total"`
	Timezone               string                 `json:"timezone"`
	Hostname               string                 `json:"hostname"`
	IPv4                   []string               `json:"ipv4"`
	IPv6                   []string               `json:"ipv6"`
	Tags                   []string               `json:"tags"`
	Version                string                 `json:"version"`
	Address                string                 `json:"address"`
	Tunnels                []*clienttunnel.Tunnel `json:"tunnels"`
	// DisconnectedAt is a time when a client was disconnected. If nil - it's connected.
	DisconnectedAt      *time.Time            `json:"disconnected_at"`
	LastHeartbeatAt     *time.Time            `json:"last_heartbeat_at"`
	ClientAuthID        string                `json:"client_auth_id"`
	AllowedUserGroups   []string              `json:"allowed_user_groups"`
	UpdatesStatus       *models.UpdatesStatus `json:"updates_status"`
	ClientConfiguration *clientconfig.Config  `json:"client_configuration"`

	Connection   ssh.Conn        `json:"-"`
	Logger       *logger.Logger  `json:"-"`
	Context      context.Context `json:"-"`
	Paused       bool            `json:"-"`
	PausedReason string          `json:"-"`
	lock         sync.Mutex
}

// CalculatedClient contains additional fields and is calculated on each request
type CalculatedClient struct {
	*Client
	Groups          []string        `json:"groups"`
	ConnectionState ConnectionState `json:"connection_state"`
}

func (c *Client) IsPaused() (paused bool) {
	paused = c.Paused
	return paused
}

const PausedDueToMaxClientsExceeded = "unlicensed"

func (c *Client) SetPaused(paused bool, reason string) {
	c.Paused = paused
	c.PausedReason = reason
	if paused {
		c.Logger.Errorf("client %s is paused (reason = %s)", c.ID, c.PausedReason)
	} else {
		c.Logger.Errorf("client %s is running", c.ID)
	}
}

func (c *Client) SetConnected() {
	c.Logger.Debugf("%s: set to connected at %s", c.ID, time.Now())
	c.DisconnectedAt = nil
}

func (c *Client) SetDisconnected(at *time.Time) {
	if c.Logger != nil {
		c.Logger.Debugf("%s: set to disconnected at %s", c.ID, at)
	}
	c.DisconnectedAt = at
}

func (c *Client) SetDisconnectedNow() {
	now := time.Now()
	c.DisconnectedAt = &now
}

func (c *Client) SetHeartbeatNow() {
	now := time.Now()
	c.DisconnectedAt = nil
	c.LastHeartbeatAt = &now
}

func (c *Client) SetHeartbeat(at *time.Time) {
	c.DisconnectedAt = nil
	c.LastHeartbeatAt = at
}

func (c *Client) ToCalculated(allGroups []*cgroups.ClientGroup) *CalculatedClient {
	clientGroups := []string{}
	for _, group := range allGroups {
		if c.BelongsTo(group) {
			clientGroups = append(clientGroups, group.ID)
		}
	}
	return &CalculatedClient{
		Client:          c,
		Groups:          clientGroups,
		ConnectionState: c.CalculateConnectionState(),
	}
}

// Obsolete returns true if a given client was disconnected longer than a given duration.
// If a given duration is nil - returns false (never obsolete).
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

func (c *Client) NewTunnelID() (tunnelID string) {
	tunnelID = strconv.FormatInt(c.generateNewTunnelID(), 10)
	return tunnelID
}

func (c *Client) generateNewTunnelID() int64 {
	return atomic.AddInt64(&c.tunnelIDAutoIncrement, 1)
}

func (c *Client) RemoveTunnelByID(tunnelID string) {
	result := make([]*clienttunnel.Tunnel, 0)
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

func (c *Client) CalculateConnectionState() ConnectionState {
	if c.DisconnectedAt == nil {
		return Connected
	}
	return Disconnected
}

// HasAccessViaUserGroups returns true if at least one of given user groups has access to a current client.
func (c *Client) HasAccessViaUserGroups(userGroups []string) bool {
	for _, curUserGroup := range userGroups {
		if curUserGroup == users.Administrators {
			return true
		}
		for _, allowedGroup := range c.AllowedUserGroups {
			if allowedGroup == curUserGroup {
				return true
			}
		}
	}

	return false
}

// UserGroupHasAccessViaClientGroup returns true if the user is member of a user group that has access to a client
// group the current client is member of
func (c *Client) UserGroupHasAccessViaClientGroup(userGroups []string, allClientGroups []*cgroups.ClientGroup) bool {
	for _, clientGroup := range allClientGroups {
		if c.BelongsTo(clientGroup) && clientGroup.OneOfUserGroupsIsAllowed(userGroups) {
			return true
		}
	}
	return false
}

// NewClientID generates a new client ID.
func NewClientID() (string, error) {
	return random.UUID4()
}
