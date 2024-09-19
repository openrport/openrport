package clientdata

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/openrport/openrport/server/api/users"
	"github.com/openrport/openrport/server/cgroups"
	"github.com/openrport/openrport/server/clients/clienttunnel"
	chshare "github.com/openrport/openrport/share"
	"github.com/openrport/openrport/share/clientconfig"
	"github.com/openrport/openrport/share/logger"
	"github.com/openrport/openrport/share/models"
	"github.com/openrport/openrport/share/random"
)

func CopyAttrsToClient(attributes models.Attributes, client *Client) {
	client.Labels = attributes.Labels
	client.Tags = attributes.Tags
}

func CopyClientsToAttrs(client Client, attributes *models.Attributes) { //nolint:govet
	attributes.Labels = client.Labels
	attributes.Tags = client.Tags
}

// Now is used to stub time.Now in tests
var Now = time.Now

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
	Labels                 map[string]string      `json:"labels"`
	Version                string                 `json:"version"`
	Address                string                 `json:"address"`
	Tunnels                []*clienttunnel.Tunnel `json:"tunnels"`

	// DisconnectedAt is a time when a client was disconnected. If nil - it's connected.
	DisconnectedAt      *time.Time            `json:"disconnected_at"`
	LastHeartbeatAt     *time.Time            `json:"last_heartbeat_at"`
	ClientAuthID        string                `json:"client_auth_id"`
	AllowedUserGroups   []string              `json:"allowed_user_groups"`
	UpdatesStatus       *models.UpdatesStatus `json:"updates_status"`
	Inventory           *models.Inventory     `json:"inventory"`
	IPAddresses         *models.IPAddresses   `json:"ext_ip_addresses"`
	ClientConfiguration *clientconfig.Config  `json:"client_configuration"`

	Connection   ssh.Conn        `json:"-"`
	Context      context.Context `json:"-"`
	Paused       bool            `json:"-"`
	PausedReason string          `json:"-"`

	Logger *logger.Logger `json:"-"`

	flock sync.RWMutex
}

// CalculatedClient contains additional fields and is calculated on each request
type CalculatedClient struct {
	*Client
	Groups          []string        `json:"groups"`
	ConnectionState ConnectionState `json:"connection_state"`
}

func NewCalculatedClient(c *Client, groups []string, connectionState ConnectionState) (cc *CalculatedClient) {
	cc = &CalculatedClient{}
	cc.Client = c
	cc.Groups = groups
	cc.ConnectionState = connectionState
	return cc
}

func (cc *CalculatedClient) GetConnectionState() (cs ConnectionState) {
	return cc.ConnectionState
}

func (c *Client) GetLock() (mu *sync.RWMutex) {
	return &c.flock
}

func (c *Client) GetID() (id string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.ID
}

func (c *Client) GetName() (name string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.Name
}

func (c *Client) GetSessionID() (sessionID string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.SessionID
}

func (c *Client) GetOS() (os string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.OS
}

func (c *Client) GetHostname() (hostname string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.Hostname
}

func (c *Client) GetTags() (tags []string) {
	c.flock.RLock()
	defer c.flock.RUnlock()

	if c.Tags == nil {
		return nil
	}

	// make sure not to return reference to underlying array
	tags = make([]string, len(c.Tags))
	copy(tags, c.Tags)
	return tags
}

func (c *Client) GetAllowedUserGroups() (groups []string) {
	c.flock.RLock()
	defer c.flock.RUnlock()

	if c.AllowedUserGroups == nil {
		return nil
	}

	// make sure not to return reference to underlying array
	groups = make([]string, len(c.AllowedUserGroups))
	copy(groups, c.AllowedUserGroups)
	return groups
}

func (c *Client) GetVersion() (version string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.Version
}

// TODO: (rs): these extra getters probably aren't required. talk to KK about options.

func (c *Client) GetAddress() (address string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.Address
}

func (c *Client) GetMemoryTotal() (mem uint64) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.MemoryTotal
}

func (c *Client) GetNumCPUs() (num int) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.NumCPUs
}

func (c *Client) GetOSArch() (arch string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.OSArch
}

func (c *Client) GetOSFamily() (fam string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.OSFamily
}

func (c *Client) GetOSFullName() (name string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.OSFullName
}

func (c *Client) GetOSKernel() (kernel string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.OSKernel
}

func (c *Client) GetOSVersion() (ver string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.OSVersion
}

func (c *Client) GetOSVirtualizationRole() (role string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.OSVirtualizationRole
}

func (c *Client) GetOSVirtualizationSystem() (sys string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.OSVirtualizationSystem
}

func (c *Client) GetTimezone() (tz string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.Timezone
}

func (c *Client) GetLabels() (labels map[string]string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	labels = make(map[string]string, len(c.Labels))
	for k, v := range c.Labels {
		labels[k] = v
	}
	return labels
}

func (c *Client) GetIPv4() (ipv4 []string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	ipv4 = make([]string, 0, len(c.IPv4))
	copy(ipv4, c.IPv4)
	return ipv4
}

func (c *Client) GetIPv6() (ipv6 []string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	ipv6 = make([]string, 0, len(c.IPv6))
	copy(ipv6, c.IPv6)
	return ipv6
}

func (c *Client) GetUpdatesStatus() (status models.UpdatesStatus) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	status = *c.UpdatesStatus
	return status
}

func (c *Client) GetInventory() (inventory models.Inventory) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	inventory = *c.Inventory
	return inventory
}

func (c *Client) GetDisconnectedAt() (at *time.Time) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.DisconnectedAt
}

func (c *Client) GetDisconnectedAtValue() (at time.Time) {
	c.flock.RLock()
	if c.DisconnectedAt != nil {
		at = *c.DisconnectedAt
	}
	c.flock.RUnlock()
	return at
}

func (c *Client) HasLastHeartbeatAt() (has bool) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.LastHeartbeatAt != nil
}

func (c *Client) GetLastHeartbeatAt() (at *time.Time) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.LastHeartbeatAt
}

func (c *Client) GetLastHeartbeatAtValue() (at time.Time) {
	c.flock.RLock()
	if c.LastHeartbeatAt != nil {
		at = *c.LastHeartbeatAt
	}
	c.flock.RUnlock()
	return at
}

func (c *Client) GetConnection() (conn ssh.Conn) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.Connection
}

func (c *Client) GetPausedReason() (reason string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.PausedReason
}

func (c *Client) GetContext() (ctx context.Context) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.Context
}

func (c *Client) GetClientAuthID() (authID string) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.ClientAuthID
}

func (c *Client) GetTunnels() (tunnels []*clienttunnel.Tunnel) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.Tunnels
}

func (c *Client) Log() (l *logger.Logger) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.Logger
}

func (c *Client) IsPaused() (paused bool) {
	c.flock.RLock()
	defer c.flock.RUnlock()
	return c.Paused
}

func (c *Client) GetMonitoringConfig() (monitoringConfig *clientconfig.MonitoringConfig) {
	c.flock.RLock()
	defer c.flock.RUnlock()

	if c.ClientConfiguration == nil {
		return nil
	}

	return &c.ClientConfiguration.Monitoring
}

func (c *Client) GetFileReceptionConfig() (fileReceptionConfig *clientconfig.FileReceptionConfig) {
	c.flock.RLock()
	defer c.flock.RUnlock()

	if c.ClientConfiguration == nil {
		return nil
	}

	return &c.ClientConfiguration.FileReceptionConfig
}

// test only
func (c *Client) SetID(id string) {
	c.flock.Lock()
	c.ID = id
	c.flock.Unlock()
}

// test only
func (c *Client) SetAddress(address string) {
	c.flock.Lock()
	c.Address = address
	c.flock.Unlock()
}

// test only
func (c *Client) SetHostname(hostname string) {
	c.flock.Lock()
	c.Hostname = hostname
	c.flock.Unlock()
}

// test only
func (c *Client) SetClientAuthID(authID string) {
	c.flock.Lock()
	c.ClientAuthID = authID
	c.flock.Unlock()
}

// test only
func (c *Client) SetTags(tags []string) {
	c.flock.Lock()
	defer c.flock.Unlock()

	if c.Tags == nil {
		return
	}

	// make sure not to just copy the tag reference
	c.Tags = make([]string, len(c.Tags))
	copy(c.Tags, tags)
}

// test only
func (c *Client) SetConnection(conn ssh.Conn) {
	c.flock.Lock()
	c.Connection = conn
	c.flock.Unlock()
}

func (c *Client) SetTunnels(tunnels []*clienttunnel.Tunnel) {
	c.flock.Lock()
	c.Tunnels = tunnels
	c.flock.Unlock()
}

func (c *Client) SetAllowedUserGroups(groups []string) {
	c.flock.Lock()
	defer c.flock.Unlock()

	// make sure not to just copy the tag reference
	c.AllowedUserGroups = make([]string, len(groups))
	copy(c.AllowedUserGroups, groups)
}

func (c *Client) SetUpdatesStatus(status *models.UpdatesStatus) {
	c.flock.Lock()
	c.UpdatesStatus = status
	c.flock.Unlock()
}

func (c *Client) SetInventory(inventory *models.Inventory) {
	c.flock.Lock()
	c.Inventory = inventory
	c.flock.Unlock()
}

func (c *Client) SetIPAddresses(IPAddresses *models.IPAddresses) {
	c.flock.Lock()
	c.IPAddresses = IPAddresses
	c.flock.Unlock()
	c.Log().Debugf("IP addresses updated for '%s'", c.Name)
}

func (c *Client) SetDisconnectedAt(at *time.Time) {
	// TODO: (rs): do we want this log? very noisy when starting a server with many clients.
	// if at != nil {
	// 	c.Log().Debugf("%s: set to disconnected at %s", c.GetID(), at)
	// }
	c.flock.Lock()
	c.DisconnectedAt = at
	c.flock.Unlock()
}

func (c *Client) SetLastHeartbeatAt(at *time.Time) {
	c.SetDisconnectedAt(nil)
	c.flock.Lock()
	c.LastHeartbeatAt = at
	c.flock.Unlock()
}

const PausedDueToMaxClientsExceeded = "unlicensed"

func (c *Client) SetPaused(paused bool, reason string) {
	c.flock.Lock()
	c.Paused = paused
	c.PausedReason = reason
	c.flock.Unlock()
	if paused {
		c.Log().Infof("client %s is paused (reason = %s)", c.GetID(), reason)
	}
}

func (c *Client) IsConnected() bool {
	return c.GetDisconnectedAt() == nil
}

func (c *Client) SetConnected() {
	c.Log().Debugf("%s: set to connected at %s", c.GetID(), time.Now())
	c.SetDisconnectedAt(nil)
}

func (c *Client) SetDisconnectedNow() {
	now := time.Now()
	c.SetDisconnectedAt(&now)
}

func (c *Client) SetHeartbeatNow() {
	now := time.Now()
	c.SetLastHeartbeatAt(&now)
	c.SetDisconnectedAt(nil)
}

func (c *Client) ToCalculated(allGroups []*cgroups.ClientGroup) *CalculatedClient {
	clientGroups := []string{}
	for _, group := range allGroups {
		if c.BelongsTo(group) {
			clientGroups = append(clientGroups, group.ID)
		}
	}

	return NewCalculatedClient(c, clientGroups, c.CalculateConnectionState())
}

// Obsolete returns true if a given client was disconnected longer than a given duration.
// If a given duration is nil - returns false (never obsolete).
func (c *Client) Obsolete(duration *time.Duration) bool {
	disconnectedAt := c.GetDisconnectedAt()
	return duration != nil && !c.IsConnected() && disconnectedAt.Add(*duration).Before(Now())
}

func (c *Client) NewTunnelID() (tunnelID string) {
	tunnelID = strconv.FormatInt(c.generateNewTunnelID(), 10)
	return tunnelID
}

func (c *Client) generateNewTunnelID() int64 {
	return atomic.AddInt64(&c.tunnelIDAutoIncrement, 1)
}

func (c *Client) RemoveTunnelByID(tunnelID string) {
	updatedTunnelList := make([]*clienttunnel.Tunnel, 0)
	// TODO: (rs): not thread-safe
	for _, tunnel := range c.GetTunnels() {
		if tunnel.ID != tunnelID {
			updatedTunnelList = append(updatedTunnelList, tunnel)
		}
	}
	c.SetTunnels(updatedTunnelList)
}

func (c *Client) Banner() string {
	clientID := c.GetID()
	clientName := c.GetName()
	tags := c.GetTags()

	banner := clientID
	if clientName != "" {
		banner += " (" + clientName + ")"
	}
	if len(tags) != 0 {
		for _, t := range tags {
			banner += " #" + t
		}
	}

	return banner
}

func (c *Client) Close() error {
	// The tunnels are closed automatically when ssh connection is closed.
	return c.GetConnection().Close()
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

	c.flock.RLock()
	defer c.flock.RUnlock()

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

	if !cgroups.MatchesRawTags(p.Tag, c.Tags) {
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

	if !p.ConnectionState.MatchesOneOf(string(c.CalculateConnectionState())) {
		return false
	}

	return true
}

func (c *Client) CalculateConnectionState() ConnectionState {
	if c.IsConnected() {
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
		for _, allowedGroup := range c.GetAllowedUserGroups() {
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

func (c *Client) GetAttributes() models.Attributes {
	attr := models.Attributes{}
	c.flock.RLock()
	CopyClientsToAttrs(*c, &attr) //nolint:govet
	c.flock.RUnlock()
	return attr
}

func (c *Client) SetAttributes(attributes models.Attributes) {
	c.flock.Lock()
	CopyAttrsToClient(attributes, c)
	c.flock.Unlock()
}

// NewClientID generates a new client ID.
func NewClientID() (string, error) {
	return random.UUID4()
}

func NewClientFromConnRequest(ctx context.Context, existingClient *Client, clientAuthID string, clientID string, req *chshare.ConnectionRequest, clientHost string, sshConn ssh.Conn, clog *logger.Logger) (client *Client) {
	if existingClient == nil {
		client = &Client{
			ID: clientID,
		}
	} else {
		client = existingClient
	}

	client.flock.Lock()
	client.Name = req.Name
	client.SessionID = req.SessionID
	client.OS = req.OS
	client.OSArch = req.OSArch
	client.OSFamily = req.OSFamily
	client.OSKernel = req.OSKernel
	client.OSFullName = req.OSFullName
	client.OSVersion = req.OSVersion
	client.OSVirtualizationSystem = req.OSVirtualizationSystem
	client.OSVirtualizationRole = req.OSVirtualizationRole
	client.Hostname = req.Hostname
	client.CPUFamily = req.CPUFamily
	client.CPUModel = req.CPUModel
	client.CPUModelName = req.CPUModelName
	client.CPUVendor = req.CPUVendor
	client.NumCPUs = req.NumCPUs
	client.MemoryTotal = req.MemoryTotal
	client.Timezone = req.Timezone
	client.IPv4 = req.IPv4
	client.IPv6 = req.IPv6
	client.Tags = req.Tags
	client.Labels = req.Labels
	client.Version = req.Version
	client.ClientConfiguration = req.ClientConfiguration
	client.Address = clientHost
	client.Tunnels = make([]*clienttunnel.Tunnel, 0)
	client.DisconnectedAt = nil
	client.ClientAuthID = clientAuthID
	client.Connection = sshConn
	client.Context = ctx
	client.Logger = clog
	client.flock.Unlock()

	return client
}
