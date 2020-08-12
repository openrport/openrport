package chserver

import (
	"context"
	"strconv"

	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/ports"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

type SessionService struct {
	repo            *ClientSessionRepository
	portDistributor *ports.PortDistributor
}

func NewSessionService(portDistributor *ports.PortDistributor) *SessionService {
	return &SessionService{
		repo:            NewSessionRepository(),
		portDistributor: portDistributor,
	}
}

func (s *SessionService) Count() (int, error) {
	return s.repo.Count()
}

func (s *SessionService) FindOne(id string) (*ClientSession, error) {
	return s.repo.FindOne(id)
}

func (s *SessionService) GetAll() ([]*ClientSession, error) {
	return s.repo.GetAll()
}

func (s *SessionService) StartClientSession(
	ctx context.Context, sid string, sshConn ssh.Conn,
	cfg *chshare.Config, user *chshare.User, clog *chshare.Logger,
) (*ClientSession, error) {
	session := &ClientSession{
		ID:         sid,
		Name:       cfg.Name,
		Tags:       cfg.Tags,
		OS:         cfg.OS,
		Hostname:   cfg.Hostname,
		Version:    cfg.Version,
		IPv4:       cfg.IPv4,
		IPv6:       cfg.IPv6,
		Address:    sshConn.RemoteAddr().String(),
		Tunnels:    make([]*Tunnel, 0),
		Connection: sshConn,
		Context:    ctx,
		User:       user,
		Logger:     clog,
	}

	_, err := s.StartSessionTunnels(session, cfg.Remotes)

	err = s.repo.Save(session)
	if err != nil {
		return nil, err
	}
	return session, nil
}

// StartSessionTunnels returns a new tunnel for each requested remote or nil if error occurred
func (s *SessionService) StartSessionTunnels(session *ClientSession, remotes []*chshare.Remote) ([]*Tunnel, error) {
	err := s.portDistributor.Refresh()
	if err != nil {
		return nil, err
	}

	tunnels := make([]*Tunnel, 0, len(remotes))
	for _, remote := range remotes {
		if !remote.IsLocalSpecified() {
			port, err := s.portDistributor.GetRandomPort()
			if err != nil {
				return nil, err
			}
			remote.LocalPort = strconv.Itoa(port)
			remote.LocalHost = "0.0.0.0"
		}

		t, err := session.StartTunnel(remote)
		if err != nil {
			return nil, err
		}
		tunnels = append(tunnels, t)
	}
	return tunnels, nil
}

func (s *SessionService) Terminate(session *ClientSession) error {
	return s.repo.Delete(session)
}
