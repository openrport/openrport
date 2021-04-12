package chserver

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/cgroups"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/server/ports"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

type ClientService struct {
	repo            *clients.ClientRepository
	portDistributor *ports.PortDistributor

	mu sync.Mutex
}

// NewClientService returns a new instance of client service.
func NewClientService(
	portDistributor *ports.PortDistributor,
	repo *clients.ClientRepository,
) *ClientService {
	return &ClientService{
		portDistributor: portDistributor,
		repo:            repo,
	}
}

func (s *ClientService) Count() (int, error) {
	return s.repo.Count()
}

func (s *ClientService) CountActive() (int, error) {
	return s.repo.CountActive()
}

func (s *ClientService) CountDisconnected() (int, error) {
	return s.repo.CountDisconnected()
}

func (s *ClientService) GetByID(id string) (*clients.Client, error) {
	return s.repo.GetByID(id)
}

func (s *ClientService) GetActiveByID(id string) (*clients.Client, error) {
	return s.repo.GetActiveByID(id)
}

func (s *ClientService) GetActiveByGroups(groups []*cgroups.ClientGroup) []*clients.Client {
	if len(groups) == 0 {
		return nil
	}

	var res []*clients.Client
	for _, cur := range s.repo.GetAllActive() {
		if cur.BelongsToOneOf(groups) {
			res = append(res, cur)
		}
	}
	return res
}

func (s *ClientService) PopulateGroupsWithClients(groups []*cgroups.ClientGroup) {
	all, _ := s.repo.GetAll()
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

// TODO(m-terel): make it consistent with others whether to return an error. No need for now return an err
func (s *ClientService) GetAllByClientID(clientID string) []*clients.Client {
	return s.repo.GetAllByClientAuthID(clientID)
}

func (s *ClientService) GetAll() ([]*clients.Client, error) {
	return s.repo.GetAll()
}

func (s *ClientService) StartClient(
	ctx context.Context, clientAuthID, clientID string, sshConn ssh.Conn, authMultiuseCreds bool,
	req *chshare.ConnectionRequest, clog *chshare.Logger,
) (*clients.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// if client id is in use, deny connection
	oldClient, err := s.repo.GetByID(clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client by id %q", clientID)
	}
	if oldClient != nil {
		if oldClient.DisconnectedAt == nil {
			return nil, fmt.Errorf("client id %q is already in use", clientID)
		}

		oldTunnels := GetTunnelsToReestablish(getRemotes(oldClient.Tunnels), req.Remotes)
		clog.Infof("Tunnels to create %d: %v", len(req.Remotes), req.Remotes)
		if len(oldTunnels) > 0 {
			clog.Infof("Old tunnels to re-establish %d: %v", len(oldTunnels), oldTunnels)
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

	client := &clients.Client{
		ID:           clientID,
		ClientAuthID: clientAuthID,
		Name:         req.Name,
		Tags:         req.Tags,
		OS:           req.OS,
		OSArch:       req.OSArch,
		OSFamily:     req.OSFamily,
		OSKernel:     req.OSKernel,
		Hostname:     req.Hostname,
		Version:      req.Version,
		IPv4:         req.IPv4,
		IPv6:         req.IPv6,
		Address:      clientHost,
		Tunnels:      make([]*clients.Tunnel, 0),
		Connection:   sshConn,
		Context:      ctx,
		Logger:       clog,
	}

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

// StartClientTunnels returns a new tunnel for each requested remote or nil if error occurred
func (s *ClientService) StartClientTunnels(client *clients.Client, remotes []*chshare.Remote) ([]*clients.Tunnel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.startClientTunnels(client, remotes)
}

func (s *ClientService) startClientTunnels(client *clients.Client, remotes []*chshare.Remote) ([]*clients.Tunnel, error) {
	err := s.portDistributor.Refresh()
	if err != nil {
		return nil, err
	}

	tunnels := make([]*clients.Tunnel, 0, len(remotes))
	for _, remote := range remotes {
		if !remote.IsLocalSpecified() {
			port, err := s.portDistributor.GetRandomPort()
			if err != nil {
				return nil, err
			}
			remote.LocalPort = strconv.Itoa(port)
			remote.LocalHost = "0.0.0.0"
			remote.LocalPortRandom = true
		}

		var acl *clients.TunnelACL
		if remote.ACL != nil {
			var err error
			acl, err = clients.ParseTunnelACL(*remote.ACL)
			if err != nil {
				return nil, err
			}
		}

		t, err := client.StartTunnel(remote, acl)
		if err != nil {
			return nil, err
		}
		tunnels = append(tunnels, t)
	}
	return tunnels, nil
}

func (s *ClientService) Terminate(client *clients.Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.repo.KeepLostClients == nil {
		return s.repo.Delete(client)
	}

	now := time.Now()
	client.DisconnectedAt = &now

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

// ForceDelete deletes client from repo regardless off KeepLostClients setting,
// if client is active it will be closed
func (s *ClientService) ForceDelete(client *clients.Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if client.DisconnectedAt == nil {
		if err := client.Close(); err != nil {
			return err
		}
	}
	return s.repo.Delete(client)
}

// isClientAuthIDInUse returns true when the client with different id exists for the client auth
func (s *ClientService) isClientAuthIDInUse(clientAuthID, clientID string) bool {
	for _, s := range s.repo.GetAllByClientAuthID(clientAuthID) {
		if s.ID != clientID {
			return true
		}
	}
	return false
}
