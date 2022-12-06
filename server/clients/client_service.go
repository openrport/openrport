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
	PopulateGroupsWithUserClients(groups []*cgroups.ClientGroup, user User)
	GetAllByClientID(clientID string) []*Client
	GetAll() ([]*Client, error)
	GetUserClients(groups []*cgroups.ClientGroup, user User) ([]*Client, error)
	GetFilteredUserClients(user User, filterOptions []query.FilterOption, groups []*cgroups.ClientGroup) ([]*CalculatedClient, error)
	StartClient(
		ctx context.Context, clientAuthID, clientID string, sshConn ssh.Conn, authMultiuseCreds bool,
		req *chshare.ConnectionRequest, clog *logger.Logger,
	) (*Client, error)
	StartClientTunnels(client *Client, remotes []*models.Remote) ([]*clienttunnel.Tunnel, error)
	Terminate(client *Client) error
	ForceDelete(client *Client) error
	DeleteOffline(clientID string) error
	SetACL(clientID string, allowedUserGroups []string) error
	SetUpdatesStatus(clientID string, updatesStatus *models.UpdatesStatus) error
	SetLastHeartbeat(clientID string, heartbeat time.Time) error
	CheckClientAccess(clientID string, user User, groups []*cgroups.ClientGroup) error
	CheckClientsAccess(clients []*Client, user User, groups []*cgroups.ClientGroup) error
	GetRepo() *ClientRepository
}

type ClientServiceProvider struct {
	repo              *ClientRepository
	portDistributor   *ports.PortDistributor
	tunnelProxyConfig *clienttunnel.TunnelProxyConfig
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

// New returns a new instance of client service.
func NewClientService(
	tunnelProxyConfig *clienttunnel.TunnelProxyConfig,
	portDistributor *ports.PortDistributor,
	repo *ClientRepository,
	logger *logger.Logger,
) *ClientServiceProvider {
	return &ClientServiceProvider{
		tunnelProxyConfig: tunnelProxyConfig,
		portDistributor:   portDistributor,
		repo:              repo,
		logger:            logger.Fork("client-service"),
	}
}

func InitClientService(
	ctx context.Context,
	tunnelProxyConfig *clienttunnel.TunnelProxyConfig,
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
			port, err := s.portDistributor.GetRandomPort(remote.Protocol)
			if err != nil {
				return nil, err
			}
			remote.LocalPort = strconv.Itoa(port)
			remote.LocalHost = models.ZeroHost
			remote.LocalPortRandom = true
		} else {
			if err := s.checkLocalPort(remote.Protocol, remote.LocalPort); err != nil {
				return nil, err
			}
		}

		var acl *clienttunnel.TunnelACL
		if remote.ACL != nil {
			var err error
			acl, err = clienttunnel.ParseTunnelACL(*remote.ACL)
			if err != nil {
				return nil, err
			}
		}

		s.logger.Debugf("starting tunnnel: %s", remote)
		t, err := client.StartTunnel(remote, acl, s.tunnelProxyConfig, s.portDistributor)
		if err != nil {
			return nil, errors.APIError{
				HTTPStatus: http.StatusConflict,
				Err:        fmt.Errorf("can't create tunnel: %s", err),
			}
		}
		tunnels = append(tunnels, t)
	}

	return tunnels, nil
}

func (s *ClientServiceProvider) checkLocalPort(protocol, port string) error {
	localPort, err := strconv.Atoi(port)
	if err != nil {
		return errors.APIError{
			HTTPStatus: http.StatusBadRequest,
			Message:    fmt.Sprintf("Invalid local port: %s.", port),
			Err:        err,
		}
	}

	if !s.portDistributor.IsPortAllowed(localPort) {
		return errors.APIError{
			HTTPStatus: http.StatusBadRequest,
			Message:    fmt.Sprintf("Local port %d is not among allowed ports.", localPort),
		}
	}

	if s.portDistributor.IsPortBusy(protocol, localPort) {
		return errors.APIError{
			HTTPStatus: http.StatusConflict,
			Message:    fmt.Sprintf("Local port %d already in use.", localPort),
		}
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
