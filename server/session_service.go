package chserver

import (
	"context"

	"golang.org/x/crypto/ssh"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type SessionService struct {
	repo *ClientSessionRepository
}

func NewSessionService() *SessionService {
	return &SessionService{
		repo: NewSessionRepository(),
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

	for _, r := range cfg.Remotes {
		_, err := session.StartRemoteTunnel(r)
		if err != nil {
			return nil, err
		}
	}

	err := s.repo.Save(session)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (s *SessionService) Terminate(session *ClientSession) error {
	return s.repo.Delete(session)
}
