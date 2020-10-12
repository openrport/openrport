package chserver

import (
	"context"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/server/ports"
	"github.com/cloudradar-monitoring/rport/server/sessions"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

type SessionService struct {
	repo            *sessions.ClientSessionRepository
	portDistributor *ports.PortDistributor
}

// NewSessionService returns a new instance of client session service.
func NewSessionService(
	portDistributor *ports.PortDistributor,
	repo *sessions.ClientSessionRepository,
) *SessionService {
	return &SessionService{
		portDistributor: portDistributor,
		repo:            repo,
	}
}

func (s *SessionService) Count() (int, error) {
	return s.repo.Count()
}

func (s *SessionService) GetByID(id string) (*sessions.ClientSession, error) {
	return s.repo.GetByID(id)
}

func (s *SessionService) GetActiveByID(id string) (*sessions.ClientSession, error) {
	return s.repo.GetActiveByID(id)
}

// TODO(m-terel): make it consistent with others whether to return an error. No need for now return an err
func (s *SessionService) GetAllByClientID(clientID string) []*sessions.ClientSession {
	return s.repo.GetAllByClientID(clientID)
}

func (s *SessionService) GetAll() ([]*sessions.ClientSession, error) {
	return s.repo.GetAll()
}

func (s *SessionService) StartClientSession(
	ctx context.Context, sid string, sshConn ssh.Conn,
	req *chshare.ConnectionRequest, client *clients.Client, clog *chshare.Logger,
) (*sessions.ClientSession, error) {
	session := &sessions.ClientSession{
		ID:         sid,
		Name:       req.Name,
		Tags:       req.Tags,
		OS:         req.OS,
		Hostname:   req.Hostname,
		Version:    req.Version,
		IPv4:       req.IPv4,
		IPv6:       req.IPv6,
		Address:    sshConn.RemoteAddr().String(),
		Tunnels:    make([]*sessions.Tunnel, 0),
		Connection: sshConn,
		Context:    ctx,
		Logger:     clog,
	}

	if client != nil {
		session.ClientID = &client.ID
	}

	_, err := s.StartSessionTunnels(session, req.Remotes)
	if err != nil {
		return nil, err
	}

	err = s.repo.Save(session)
	if err != nil {
		return nil, err
	}
	return session, nil
}

// StartSessionTunnels returns a new tunnel for each requested remote or nil if error occurred
func (s *SessionService) StartSessionTunnels(session *sessions.ClientSession, remotes []*chshare.Remote) ([]*sessions.Tunnel, error) {
	err := s.portDistributor.Refresh()
	if err != nil {
		return nil, err
	}

	tunnels := make([]*sessions.Tunnel, 0, len(remotes))
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

		var acl *sessions.TunnelACL
		if remote.ACL != nil {
			var err error
			acl, err = sessions.ParseTunnelACL(*remote.ACL)
			if err != nil {
				return nil, err
			}
		}

		t, err := session.StartTunnel(remote, acl)
		if err != nil {
			return nil, err
		}
		tunnels = append(tunnels, t)
	}
	return tunnels, nil
}

func (s *SessionService) Terminate(session *sessions.ClientSession) error {
	if s.repo.KeepLostClients == nil {
		return s.repo.Delete(session)
	}

	now := time.Now()
	session.Disconnected = &now

	// Do not save if session doesn't exist in repo - it was force deleted
	existing, err := s.repo.GetByID(session.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return nil
	}
	return s.repo.Save(session)
}

// ForceDelete deletes session from repo regardless off KeepLostClients setting,
// if session is active it will be closed
func (s *SessionService) ForceDelete(session *sessions.ClientSession) error {
	if session.Disconnected == nil {
		if err := session.Close(); err != nil {
			return err
		}
	}
	return s.repo.Delete(session)
}
