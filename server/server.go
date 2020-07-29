package chserver

import (
	chshare "github.com/cloudradar-monitoring/rport/share"
)

// Config is the configuration for the rport service
type Config struct {
	KeySeed      string
	AuthFile     string
	Auth         string
	Proxy        string
	Verbose      bool
	APIAuth      string
	APIJWTSecret string
	DocRoot      string
}

// Server represents a rport service
type Server struct {
	clientListener *ClientListener
	apiListener    *APIListener
}

// NewServer creates and returns a new rport server
func NewServer(config *Config) (*Server, error) {
	s := &Server{}

	var err error
	sessionRepo := NewSessionRepository()

	s.clientListener, err = NewClientListener(config, sessionRepo)
	if err != nil {
		return nil, err
	}

	s.apiListener, err = NewAPIListener(config, sessionRepo)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// Run is responsible for starting the rport service
func (s *Server) Run(listenAddr string, apiAddr string) error {
	if err := s.Start(listenAddr, apiAddr); err != nil {
		return err
	}

	return s.Wait()
}

// Start is responsible for kicking off the http server
func (s *Server) Start(listenAddr string, apiAddr string) error {
	err := s.clientListener.Start(listenAddr)
	if err != nil {
		return err
	}

	if apiAddr != "" {
		err = s.apiListener.Start(apiAddr)
	}
	return err
}

func (s *Server) Wait() error {
	return chshare.SyncCall(
		s.clientListener.Wait,
		s.apiListener.Wait,
	)
}

func (s *Server) Close() error {
	return chshare.SyncCall(
		s.clientListener.Close,
		s.apiListener.Close,
	)
}
