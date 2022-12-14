package clients

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/hashicorp/go-version"
	"github.com/jmoiron/sqlx"

	"github.com/cloudradar-monitoring/rport/caddy"
	"github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/cgroups"
	"github.com/cloudradar-monitoring/rport/server/clients/clienttunnel"
	"github.com/cloudradar-monitoring/rport/server/ports"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/query"
)

type ClientService interface {
	Count() (int, error)
	CountActive() (int, error)
	CountDisconnected() (int, error)
	GetByID(id string) (*Client, error)
	GetActiveByID(id string) (*Client, error)
	GetActiveByGroups(groups []*cgroups.ClientGroup) []*Client
	GetClientsByTag(tags []string, operator string, allowDisconnected bool) (clients []*Client, err error)
	GetAllByClientID(clientID string) []*Client
	GetAll() ([]*Client, error)
	GetUserClients(groups []*cgroups.ClientGroup, user User) ([]*Client, error)
	GetFilteredUserClients(user User, filterOptions []query.FilterOption, groups []*cgroups.ClientGroup) ([]*CalculatedClient, error)

	PopulateGroupsWithUserClients(groups []*cgroups.ClientGroup, user User)

	StartClient(
		ctx context.Context, clientAuthID, clientID string, sshConn ssh.Conn, authMultiuseCreds bool,
		req *chshare.ConnectionRequest, clog *logger.Logger,
	) (*Client, error)
	Terminate(client *Client) error
	ForceDelete(client *Client) error
	DeleteOffline(clientID string) error

	SetACL(clientID string, allowedUserGroups []string) error
	CheckClientAccess(clientID string, user User, groups []*cgroups.ClientGroup) error
	CheckClientsAccess(clients []*Client, user User, groups []*cgroups.ClientGroup) error

	SetUpdatesStatus(clientID string, updatesStatus *models.UpdatesStatus) error
	SetLastHeartbeat(clientID string, heartbeat time.Time) error

	GetRepo() *ClientRepository

	SetCaddyAPI(capi caddy.API)
	StartClientTunnels(client *Client, remotes []*models.Remote) ([]*clienttunnel.Tunnel, error)
	StartTunnel(c *Client, r *models.Remote, acl *clienttunnel.TunnelACL) (*clienttunnel.Tunnel, error)
	FindTunnel(c *Client, id string) *clienttunnel.Tunnel
	FindTunnelByRemote(c *Client, r *models.Remote) *clienttunnel.Tunnel
	TerminateTunnel(c *Client, t *clienttunnel.Tunnel, force bool) error
}

type ClientServiceProvider struct {
	repo              *ClientRepository
	portDistributor   *ports.PortDistributor
	tunnelProxyConfig *clienttunnel.InternalTunnelProxyConfig
	caddyAPI          caddy.API
	logger            *logger.Logger

	mu sync.Mutex
}

var OptionsSupportedFilters = map[string]bool{
	"id":                       true,
	"name":                     true,
	"os":                       true,
	"os_arch":                  true,
	"os_family":                true,
	"os_kernel":                true,
	"os_full_name":             true,
	"os_version":               true,
	"os_virtualization_system": true,
	"os_virtualization_role":   true,
	"cpu_family":               true,
	"cpu_model":                true,
	"cpu_model_name":           true,
	"cpu_vendor":               true,
	"num_cpus":                 true,
	"timezone":                 true,
	"hostname":                 true,
	"ipv4":                     true,
	"ipv6":                     true,
	"tags":                     true,
	"version":                  true,
	"address":                  true,
	"client_auth_id":           true,
	"allowed_user_groups":      true,
	"groups":                   true,
	"connection_state":         true,
}

var OptionsSupportedSorts = map[string]bool{
	"id":       true,
	"name":     true,
	"os":       true,
	"hostname": true,
	"version":  true,
}

var OptionsSupportedFields = map[string]map[string]bool{
	"clients": {
		"id":                       true,
		"name":                     true,
		"os":                       true,
		"os_arch":                  true,
		"os_family":                true,
		"os_kernel":                true,
		"hostname":                 true,
		"ipv4":                     true,
		"ipv6":                     true,
		"tags":                     true,
		"version":                  true,
		"address":                  true,
		"tunnels":                  true,
		"disconnected_at":          true,
		"last_heartbeat_at":        true,
		"connection_state":         true,
		"client_auth_id":           true,
		"os_full_name":             true,
		"os_version":               true,
		"os_virtualization_system": true,
		"os_virtualization_role":   true,
		"cpu_family":               true,
		"cpu_model":                true,
		"cpu_model_name":           true,
		"cpu_vendor":               true,
		"timezone":                 true,
		"num_cpus":                 true,
		"mem_total":                true,
		"allowed_user_groups":      true,
		"updates_status":           true,
		"client_configuration":     true,
		"groups":                   true,
	},
}

var OptionsListDefaultFields = map[string][]string{
	"fields[clients]": {
		"id",
		"name",
		"hostname",
	},
}

// NewClientService returns a new instance of client service.
func NewClientService(
	tunnelProxyConfig *clienttunnel.InternalTunnelProxyConfig,
	portDistributor *ports.PortDistributor,
	repo *ClientRepository,
	logger *logger.Logger,
) *ClientServiceProvider {
	csp := &ClientServiceProvider{
		tunnelProxyConfig: tunnelProxyConfig,
		portDistributor:   portDistributor,
		repo:              repo,
		logger:            logger.Fork("client-service"),
	}

	return csp
}

func InitClientService(
	ctx context.Context,
	tunnelProxyConfig *clienttunnel.InternalTunnelProxyConfig,
	portDistributor *ports.PortDistributor,
	db *sqlx.DB,
	keepDisconnectedClients *time.Duration,
	logger *logger.Logger,
) (*ClientServiceProvider, error) {
	repo, err := InitClientRepository(ctx, db, keepDisconnectedClients, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to init Client Repository: %v", err)
	}

	return NewClientService(tunnelProxyConfig, portDistributor, repo, logger), nil
}

func (s *ClientServiceProvider) Count() (int, error) {
	return s.repo.Count()
}

func (s *ClientServiceProvider) CountActive() (int, error) {
	return s.repo.CountActive()
}

func (s *ClientServiceProvider) CountDisconnected() (int, error) {
	return s.repo.CountDisconnected()
}

func (s *ClientServiceProvider) GetByID(id string) (*Client, error) {
	return s.repo.GetByID(id)
}

func (s *ClientServiceProvider) GetActiveByID(id string) (*Client, error) {
	return s.repo.GetActiveByID(id)
}

func (s *ClientServiceProvider) GetActiveByGroups(groups []*cgroups.ClientGroup) []*Client {
	if len(groups) == 0 {
		return nil
	}

	var res []*Client
	for _, cur := range s.repo.GetAllActive() {
		if cur.BelongsToOneOf(groups) {
			res = append(res, cur)
		}
	}
	return res
}

func (s *ClientServiceProvider) GetClientsByTag(tags []string, operator string, allowDisconnected bool) (clients []*Client, err error) {
	return s.repo.GetClientsByTag(tags, operator, allowDisconnected)
}

func (s *ClientServiceProvider) PopulateGroupsWithUserClients(groups []*cgroups.ClientGroup, user User) {
	all, _ := s.repo.GetUserClients(user, groups)
	for _, curClient := range all {
		for _, curGroup := range groups {
			if curClient.BelongsTo(curGroup) {
				curGroup.ClientIDs = append(curGroup.ClientIDs, curClient.ID)
			}
		}
	}
	for _, curGroup := range groups {
		sort.Strings(curGroup.ClientIDs)
	}
}

func (s *ClientServiceProvider) GetAllByClientID(clientID string) []*Client {
	return s.repo.GetAllByClientAuthID(clientID)
}

func (s *ClientServiceProvider) GetAll() ([]*Client, error) {
	return s.repo.GetAll()
}

func (s *ClientServiceProvider) GetUserClients(groups []*cgroups.ClientGroup, user User) ([]*Client, error) {
	return s.repo.GetUserClients(user, groups)
}

func (s *ClientServiceProvider) GetFilteredUserClients(user User, filterOptions []query.FilterOption, groups []*cgroups.ClientGroup) ([]*CalculatedClient, error) {
	return s.repo.GetFilteredUserClients(user, filterOptions, groups)
}

func (s *ClientServiceProvider) StartClient(
	ctx context.Context, clientAuthID, clientID string, sshConn ssh.Conn, authMultiuseCreds bool,
	req *chshare.ConnectionRequest, clog *logger.Logger,
) (*Client, error) {
	clog.Debugf("starting client session: %s", clientID)

	s.mu.Lock()
	defer s.mu.Unlock()

	// if client id is in use, deny connection
	client, err := s.repo.GetByID(clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client by id %q", clientID)
	}

	if client != nil {
		var sessionReUsed = false
		if req.SessionID != "" && req.SessionID == client.SessionID {
			// Stored previous session id and the session id of the connection attempt are equal
			sessionReUsed = true
			clog.Debugf("resuming existing session %s for client %s [%s]", req.SessionID, client.Name, clientID)
		}
		if client.DisconnectedAt == nil && !sessionReUsed {
			return nil, fmt.Errorf("client is already connected: %s [%s]", client.Name, clientID)
		}

		oldTunnels := GetTunnelsToReestablish(getRemotes(client.Tunnels), req.Remotes)
		clientVersion, err := version.NewVersion(req.Version)
		if err != nil {
			return nil, fmt.Errorf("failed to determine client version: %v", err)
		}
		requiredVersion, _ := version.NewVersion("0.6.4")
		if clientVersion.GreaterThanOrEqual(requiredVersion) {
			oldTunnels, err = ExcludeNotAllowedTunnels(clog, oldTunnels, sshConn)
			if err != nil {
				return nil, fmt.Errorf("failed to filter tunnels: %v", err)
			}
		} else {
			clog.Infof("client %s (%s) version %s does not support 'tunnel_allowed' policies. Consider upgrading.", client.ID, client.Name, client.Version)
		}

		clog.Infof("tunnels to create %d: %v", len(req.Remotes), req.Remotes)
		if len(oldTunnels) > 0 {
			clog.Infof("old tunnels to re-establish %d: %v", len(oldTunnels), oldTunnels)
			req.Remotes = append(req.Remotes, oldTunnels...)
		}
	}

	// check if client auth ID is already used by another client
	if !authMultiuseCreds && s.isClientAuthIDInUse(clientAuthID, clientID) {
		return nil, fmt.Errorf("client auth ID is already in use: %q", clientAuthID)
	}

	clientAddr := sshConn.RemoteAddr().String()
	clientHost, _, err := net.SplitHostPort(clientAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get host for address %q: %v", clientAddr, err)
	}

	if client == nil {
		client = &Client{
			ID: clientID,
		}
	}
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
	client.Version = req.Version
	client.ClientConfiguration = req.ClientConfiguration
	client.Address = clientHost
	client.Tunnels = make([]*clienttunnel.Tunnel, 0)
	client.DisconnectedAt = nil
	client.ClientAuthID = clientAuthID
	client.Connection = sshConn
	client.Context = ctx
	client.Logger = clog

	client.SetConnected()

	_, err = s.startClientTunnels(client, req.Remotes)
	if err != nil {
		return nil, err
	}

	err = s.repo.Save(client)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func getRemotes(tunnels []*clienttunnel.Tunnel) []*models.Remote {
	r := make([]*models.Remote, 0, len(tunnels))
	for _, t := range tunnels {
		r = append(r, &t.Remote)
	}
	return r
}

// GetTunnelsToReestablish returns old tunnels that should be re-establish taking into account new tunnels.
func GetTunnelsToReestablish(old, new []*models.Remote) []*models.Remote {
	if len(new) > len(old) {
		return nil
	}

	// check if old tunnels contain all new tunnels
	// NOTE: old tunnels contain random port if local was not specified
	oldMarked := make([]bool, len(old))

	// at first check new with local specified. It's done at first to cover a case when a new tunnel was specified
	// with a port that is among random ports in old tunnels.
loop1:
	for _, curNew := range new {
		if curNew.IsLocalSpecified() {
			for i, curOld := range old {
				if !oldMarked[i] && curNew.String() == curOld.String() {
					oldMarked[i] = true
					continue loop1
				}
			}
			return nil
		}
	}

	// then check without local
loop2:
	for _, curNew := range new {
		if !curNew.IsLocalSpecified() {
			for i, curOld := range old {
				if !oldMarked[i] && curOld.LocalPortRandom && curNew.Remote() == curOld.Remote() && curNew.EqualACL(curOld.ACL) {
					oldMarked[i] = true
					continue loop2
				}
			}
			return nil
		}
	}

	// add tunnels that left among old
	var res []*models.Remote
	for i, marked := range oldMarked {
		if !marked {
			r := *old[i]
			// if it was random then set up zero values
			if r.LocalPortRandom {
				r.LocalHost = ""
				r.LocalPort = ""
			}
			res = append(res, &r)
		}
	}

	return res
}

// StartClientTunnels returns a new tunnel for each requested remote or nil if error occurred
func (s *ClientServiceProvider) StartClientTunnels(client *Client, remotes []*models.Remote) ([]*clienttunnel.Tunnel, error) {
	s.logger.Debugf("starting client tunnels: %s", client.ID)

	s.mu.Lock()
	defer s.mu.Unlock()
	newTunnels, err := s.startClientTunnels(client, remotes)
	if err != nil {
		return nil, err
	}

	err = s.repo.Save(client)
	if err != nil {
		return nil, err
	}

	return newTunnels, err
}

func (s *ClientServiceProvider) startClientTunnels(client *Client, remotes []*models.Remote) ([]*clienttunnel.Tunnel, error) {
	err := s.portDistributor.Refresh()
	if err != nil {
		return nil, err
	}

	tunnels := make([]*clienttunnel.Tunnel, 0, len(remotes))
	for _, remote := range remotes {
		if !remote.IsLocalSpecified() {
			s.logger.Debugf("no local specified")
			port, err := s.portDistributor.GetRandomPort(remote.Protocol)
			if err != nil {
				return nil, err
			}
			remote.LocalPort = strconv.Itoa(port)
			remote.LocalHost = models.ZeroHost
			remote.LocalPortRandom = true
			s.logger.Debugf("using random port %s", remote.LocalPort)
		} else {
			s.logger.Debugf("checking local port %s", remote.LocalPort)
			if err := s.checkLocalPort(remote.Protocol, remote.LocalPort); err != nil {
				return nil, err
			}
		}

		s.logger.Debugf("initiating tunnel %+v", remote)

		var acl *clienttunnel.TunnelACL
		if remote.ACL != nil {
			var err error
			acl, err = clienttunnel.ParseTunnelACL(*remote.ACL)
			if err != nil {
				return nil, err
			}
		}

		t, err := s.StartTunnel(client, remote, acl)
		if err != nil {
			return nil, errors.APIError{
				HTTPStatus: http.StatusConflict,
				Err:        fmt.Errorf("unable to start tunnel: %s", err),
			}
		}
		tunnels = append(tunnels, t)
	}

	return tunnels, nil
}

func (s *ClientServiceProvider) checkLocalPort(protocol, port string) error {
	localPort, err := strconv.Atoi(port)
	if err != nil {
		return errors.NewAPIError(http.StatusBadRequest, "", fmt.Sprintf("Invalid local port: %s.", port), err)
	}

	if !s.portDistributor.IsPortAllowed(localPort) {
		return errors.NewAPIError(http.StatusBadRequest, "", fmt.Sprintf("Local port %d is not among allowed ports.", localPort), nil)
	}

	if s.portDistributor.IsPortBusy(protocol, localPort) {
		return errors.NewAPIError(http.StatusConflict, "", fmt.Sprintf("Local port %d already in use.", localPort), nil)
	}

	return nil
}

func (s *ClientServiceProvider) Terminate(client *Client) error {
	s.logger.Infof("terminating client: %s", client.ID)

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.repo.KeepDisconnectedClients != nil && *s.repo.KeepDisconnectedClients == 0 {
		return s.repo.Delete(client)
	}

	now := time.Now()
	client.SetDisconnected(&now)

	// Do not save if client doesn't exist in repo - it was force deleted
	existing, err := s.repo.GetByID(client.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return nil
	}
	return s.repo.Save(client)
}

// ForceDelete deletes client from repo regardless off KeepDisconnectedClients setting,
// if client is active it will be closed
func (s *ClientServiceProvider) ForceDelete(client *Client) error {
	s.logger.Debugf("force deleting client: %s", client.ID)

	s.mu.Lock()
	defer s.mu.Unlock()
	if client.DisconnectedAt == nil {
		if err := client.Close(); err != nil {
			return err
		}
	}
	return s.repo.Delete(client)
}

func (s *ClientServiceProvider) DeleteOffline(clientID string) error {
	s.logger.Debugf("deleting offline client: %s", clientID)

	existing, err := s.getExistingByID(clientID)
	if err != nil {
		return err
	}

	if existing.DisconnectedAt == nil {
		return errors.APIError{
			Message:    "Client is active, should be disconnected",
			HTTPStatus: http.StatusBadRequest,
		}
	}

	return s.repo.Delete(existing)
}

// isClientAuthIDInUse returns true when the client with different id exists for the client auth
func (s *ClientServiceProvider) isClientAuthIDInUse(clientAuthID, clientID string) bool {
	for _, s := range s.repo.GetAllByClientAuthID(clientAuthID) {
		if s.ID != clientID {
			return true
		}
	}
	return false
}

func (s *ClientServiceProvider) SetACL(clientID string, allowedUserGroups []string) error {
	existing, err := s.getExistingByID(clientID)
	if err != nil {
		return err
	}

	existing.AllowedUserGroups = allowedUserGroups

	return s.repo.Save(existing)
}

func (s *ClientServiceProvider) SetUpdatesStatus(clientID string, updatesStatus *models.UpdatesStatus) error {
	existing, err := s.getExistingByID(clientID)
	if err != nil {
		return err
	}

	existing.UpdatesStatus = updatesStatus

	return s.repo.Save(existing)
}

func (s *ClientServiceProvider) SetLastHeartbeat(clientID string, heartbeat time.Time) error {
	existing, err := s.getExistingByID(clientID)
	if err != nil {
		return err
	}
	existing.LastHeartbeatAt = &heartbeat
	return nil
}

// CheckClientAccess returns nil if a given user has an access to a given client.
// Otherwise, APIError with 403 is returned.
func (s *ClientServiceProvider) CheckClientAccess(clientID string, user User, groups []*cgroups.ClientGroup) error {
	existing, err := s.getExistingByID(clientID)
	if err != nil {
		return err
	}

	return s.CheckClientsAccess([]*Client{existing}, user, groups)
}

// CheckClientsAccess returns nil if a given user has an access to all of the given
// Otherwise, APIError with 403 is returned.
func (s *ClientServiceProvider) CheckClientsAccess(clients []*Client, user User, clientGroups []*cgroups.ClientGroup) error {
	if user.IsAdmin() {
		return nil
	}

	var clientsWithNoAccess []string
	userGroups := user.GetGroups()
	for _, curClient := range clients {
		if curClient.HasAccessViaUserGroups(userGroups) || curClient.UserGroupHasAccessViaClientGroup(userGroups, clientGroups) {
			continue
		}
		clientsWithNoAccess = append(clientsWithNoAccess, curClient.ID)
	}

	if len(clientsWithNoAccess) > 0 {
		return errors.APIError{
			Message:    fmt.Sprintf("Access denied to client(s) with ID(s): %v", strings.Join(clientsWithNoAccess, ", ")),
			HTTPStatus: http.StatusForbidden,
		}
	}

	return nil
}

// getExistingByID returns non-nil client by id. If not found or failed to get a client - an error is returned.
func (s *ClientServiceProvider) getExistingByID(clientID string) (*Client, error) {
	if clientID == "" {
		return nil, errors.APIError{
			Message:    "Client id is empty",
			HTTPStatus: http.StatusBadRequest,
		}
	}

	existing, err := s.repo.GetByID(clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to find a client with id=%q: %w", clientID, err)
	}

	if existing == nil {
		return nil, errors.APIError{
			Message:    fmt.Sprintf("Client with id=%q not found.", clientID),
			HTTPStatus: http.StatusNotFound,
		}
	}

	return existing, nil
}

func (s *ClientServiceProvider) GetRepo() *ClientRepository {
	return s.repo
}

func ExcludeNotAllowedTunnels(clog *logger.Logger, tunnels []*models.Remote, conn ssh.Conn) ([]*models.Remote, error) {
	filtered := make([]*models.Remote, 0, len(tunnels))
	for _, t := range tunnels {
		allowed, err := clienttunnel.IsAllowed(t.Remote(), conn)
		if err != nil {
			if strings.Contains(err.Error(), "unknown request") {
				return tunnels, nil
			}
			return nil, err
		}
		if !allowed {
			clog.Infof("Tunnel %q is no longer allowed by client config, removing.", t)
			continue
		}
		filtered = append(filtered, t)
	}
	return filtered, nil
}

func (s *ClientServiceProvider) FindTunnelByRemote(c *Client, r *models.Remote) *clienttunnel.Tunnel {
	for _, curr := range c.Tunnels {
		if curr.Equals(r) {
			return curr
		}
	}
	return nil
}

func (s *ClientServiceProvider) FindTunnel(c *Client, id string) *clienttunnel.Tunnel {
	for _, curr := range c.Tunnels {
		if curr.ID == id {
			return curr
		}
	}
	return nil
}

func (s *ClientServiceProvider) SetCaddyAPI(capi caddy.API) {
	s.caddyAPI = capi
}

func (s *ClientServiceProvider) StartTunnel(
	c *Client,
	r *models.Remote,
	acl *clienttunnel.TunnelACL) (t *clienttunnel.Tunnel, err error) {
	t = s.FindTunnelByRemote(c, r)
	// tunnel exists
	if t != nil {
		return t, nil
	}

	s.logger.Debugf("starting tunnel: %s", r)

	ctx := c.Context
	if r.AutoClose > 0 {
		// no need to cancel the ctx since it will be canceled by parent ctx or after given timeout
		ctx, _ = context.WithTimeout(ctx, r.AutoClose) // nolint: govet
	}

	startTunnelProxy := s.tunnelProxyConfig.Enabled && r.HTTPProxy
	if startTunnelProxy {
		t, err = s.startTunnelWithProxy(ctx, c, r, acl)
		if err != nil {
			return nil, err
		}
		if r.UseDownstreamSubdomainProxy {
			err = s.startCaddyDownstreamProxy(ctx, c, r, t)
			if err != nil {
				tunnelStopErr := t.InternalTunnelProxy.Stop(c.Context)
				if tunnelStopErr != nil {
					c.Logger.Infof("unable to stop internal tunnel proxy after failing to create caddy downstream proxy: %s", tunnelStopErr)
				}
				return nil, err
			}
			t.CaddyDownstreamProxyExists = true
		}
	} else {
		t, err = s.startRegularTunnel(ctx, c, r, acl)
		if err != nil {
			return nil, err
		}
	}

	// in case tunnel auto-closed due to auto close - run background task to remove the tunnel from the list
	// TODO: consider to create a separate background task to terminate all inactive tunnels based on some deadline/lastActivity time
	if t.AutoClose > 0 {
		go s.cleanupOnAutoCloseDeadlineExceeded(ctx, t, c)
	}

	if t.IdleTimeoutMinutes > 0 {
		go s.terminateTunnelOnIdleTimeout(ctx, t, c)
	}

	c.Tunnels = append(c.Tunnels, t)
	return t, nil
}

func (s *ClientServiceProvider) startCaddyDownstreamProxy(
	ctx context.Context,
	c *Client,
	r *models.Remote,
	t *clienttunnel.Tunnel,
) (err error) {
	c.Logger.Infof("starting downstream caddy proxy at %s", r.DownstreamProxyURL())
	c.Logger.Infof("tunnel = %#v", t)
	c.Logger.Infof("remote = %#v", r)

	nrr := &caddy.NewRouteRequest{
		RouteID:                   r.DownstreamSubdomain,
		TargetTunnelHost:          t.LocalHost,
		TargetTunnelPort:          t.LocalPort,
		DownstreamProxySubdomain:  r.DownstreamSubdomain,
		DownstreamProxyBaseDomain: r.DownstreamBasedomain,
	}

	c.Logger.Debugf("requesting new caddy route = %+v", nrr)

	res, err := s.caddyAPI.AddRoute(ctx, nrr)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create downstream caddy proxy: status_code: %d", res.StatusCode)
	}

	c.Logger.Infof("started downstream caddy proxy at %s to %s:%s", r.DownstreamProxyURL(), t.LocalHost, t.LocalPort)
	return nil
}

func (s *ClientServiceProvider) startRegularTunnel(ctx context.Context, c *Client, r *models.Remote, acl *clienttunnel.TunnelACL) (*clienttunnel.Tunnel, error) {
	tunnelID := c.NewTunnelID()

	t, err := clienttunnel.NewTunnel(c.Logger, c.Connection, tunnelID, *r, acl)
	if err != nil {
		return nil, err
	}

	err = t.Start(ctx)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (s *ClientServiceProvider) startTunnelWithProxy(
	ctx context.Context,
	c *Client,
	r *models.Remote,
	acl *clienttunnel.TunnelACL,
) (*clienttunnel.Tunnel, error) {
	proxyHost := ""
	proxyPort := ""
	var proxyACL *clienttunnel.TunnelACL

	// assuming that we still want to log activity in the client log
	c.Logger.Debugf("client %s will use tunnel proxy", c.ID)

	// get values for tunnel proxy local host addr from original remote
	proxyHost = r.LocalHost
	proxyPort = r.LocalPort
	proxyACL = acl

	// reconfigure tunnel local host/addr to use 127.0.0.1 with a random port and make new acl
	r.LocalHost = "127.0.0.1"
	port, err := s.portDistributor.GetRandomPort(r.Protocol)
	if err != nil {
		return nil, err
	}

	r.LocalPort = strconv.Itoa(port)
	acl, _ = clienttunnel.ParseTunnelACL("127.0.0.1") // access to tunnel is only allowed from localhost

	tunnelID := c.NewTunnelID()

	// original tunnel will use the reconfigured original remote
	t, err := clienttunnel.NewTunnel(c.Logger, c.Connection, tunnelID, *r, acl)
	if err != nil {
		return nil, err
	}

	// start the original tunnel before the proxy tunnel
	err = t.Start(ctx)
	if err != nil {
		return nil, err
	}

	// create new proxy tunnel listening at the original tunnel local host addr
	tProxy := clienttunnel.NewInternalTunnelProxy(t, c.Logger, s.tunnelProxyConfig, proxyHost, proxyPort, proxyACL)
	c.Logger.Debugf("client %s starting tunnel proxy", c.ID)
	if err := tProxy.Start(ctx); err != nil {
		c.Logger.Debugf("tunnel proxy could not be started, tunnel must be terminated: %v", err)
		if tErr := t.Terminate(true); tErr != nil {
			return nil, tErr
		}
		return nil, fmt.Errorf("tunnel started and terminated because of tunnel proxy start error")
	}

	t.InternalTunnelProxy = tProxy

	// reconfigure original tunnel remote host addr to be the new proxy tunnel
	t.Remote.LocalHost = t.InternalTunnelProxy.Host
	t.Remote.LocalPort = t.InternalTunnelProxy.Port

	c.Logger.Debugf("client %s started tunnel with proxy: %#v", c.ID, t)
	c.Logger.Debugf("internal tunnel proxy: %#v", t.InternalTunnelProxy)

	return t, nil
}

func (s *ClientServiceProvider) cleanupOnAutoCloseDeadlineExceeded(ctx context.Context, t *clienttunnel.Tunnel, c *Client) {
	<-ctx.Done()
	// DeadlineExceeded err is expected when tunnel AutoClose period is reached, otherwise skip cleanup
	if ctx.Err() == context.DeadlineExceeded {
		s.cleanupAfterAutoClose(c, t)
	}
}

func (s *ClientServiceProvider) terminateTunnelOnIdleTimeout(ctx context.Context, t *clienttunnel.Tunnel, c *Client) {
	idleTimeout := time.Duration(t.IdleTimeoutMinutes) * time.Minute
	timer := time.NewTimer(idleTimeout)
	for {
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
			sinceLastActive := time.Since(t.LastActive())
			if sinceLastActive > idleTimeout {
				c.Logger.Infof("Terminating... inactivity period is reached: %d minute(s)", t.IdleTimeoutMinutes)
				_ = t.Terminate(true)
				s.cleanupAfterAutoClose(c, t)
				return
			}
			timer.Reset(idleTimeout - sinceLastActive)
		}
	}
}

func (s *ClientServiceProvider) cleanupAfterAutoClose(c *Client, t *clienttunnel.Tunnel) {
	c.Lock()
	defer c.Unlock()

	c.Logger.Infof("Auto closing tunnel %s ...", t.ID)

	//stop tunnel proxy
	if t.InternalTunnelProxy != nil {
		if err := t.InternalTunnelProxy.Stop(c.Context); err != nil {
			c.Logger.Errorf("error while stopping tunnel proxy: %v", err)
		}
		if t.CaddyDownstreamProxyExists {
			// err is logged in the remove fn
			_ = s.removeCaddyDownstreamProxy(c, t)
		}
	}

	c.RemoveTunnelByID(t.ID)

	err := s.repo.Save(c)
	if err != nil {
		c.Logger.Errorf("unable to save client after auto close cleanup: %v", err)
	}

	c.Logger.Debugf("auto closed tunnel with id=%s removed", t.ID)
}

func (s *ClientServiceProvider) TerminateTunnel(c *Client, t *clienttunnel.Tunnel, force bool) error {
	c.Logger.Infof("Terminating tunnel %s (force: %v) ...", t.ID, force)

	err := t.Terminate(force)
	if err != nil {
		return err
	}

	if t.InternalTunnelProxy != nil {
		if err := t.InternalTunnelProxy.Stop(c.Context); err != nil {
			c.Logger.Errorf("error while stopping tunnel proxy: %v", err)
		}
		if t.CaddyDownstreamProxyExists {
			// err is logged in the remove fn
			_ = s.removeCaddyDownstreamProxy(c, t)
		}
		if err != nil {
			return err
		}
	}

	c.RemoveTunnelByID(t.ID)

	err = s.repo.Save(c)
	if err != nil {
		c.Logger.Errorf("unable to save client after auto close cleanup: %v", err)
	}

	c.Logger.Debugf("terminated tunnel with id=%s removed", t.ID)
	return nil
}

func (s *ClientServiceProvider) removeCaddyDownstreamProxy(c *Client, t *clienttunnel.Tunnel) (err error) {
	c.Logger.Infof("removing downstream caddy proxy at %s", t.Remote.DownstreamProxyURL())

	res, err := s.caddyAPI.DeleteRoute(c.Context, t.Remote.DownstreamSubdomain)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete downstream caddy proxy: status_code: %d", res.StatusCode)
	}

	c.Logger.Infof("removed downstream caddy proxy at %s", t.Remote.DownstreamProxyURL())
	return nil
}
