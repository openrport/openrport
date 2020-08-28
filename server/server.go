package chserver

import (
	"fmt"

	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/ports"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

// Server represents a rport service
type Server struct {
	*chshare.Logger
	listenAddr     string
	apiAddr        string
	clientListener *ClientListener
	apiListener    *APIListener
}

// NewServer creates and returns a new rport server
func NewServer(config *Config) (*Server, error) {
	s := &Server{
		Logger:     chshare.NewLogger("server", config.LogOutput, config.LogLevel),
		listenAddr: config.ListenAddress,
		apiAddr:    config.API.Address,
	}

	privateKey, err := initPrivateKey(config.KeySeed)
	if err != nil {
		return nil, err
	}
	fingerprint := chshare.FingerprintKey(privateKey.PublicKey())
	s.Infof("Fingerprint %s", fingerprint)

	sessionService := NewSessionService(
		ports.NewPortDistributor(config.GetExcludedPorts()),
	)

	s.clientListener, err = NewClientListener(config, sessionService, privateKey)
	if err != nil {
		return nil, err
	}

	s.apiListener, err = NewAPIListener(config, sessionService, fingerprint)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func initPrivateKey(seed string) (ssh.Signer, error) {
	//generate private key (optionally using seed)
	key, _ := chshare.GenerateKey(seed)
	//convert into ssh.PrivateKey
	private, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key: %s", err)
	}
	return private, nil
}

// Run is responsible for starting the rport service
func (s *Server) Run() error {
	if err := s.Start(); err != nil {
		return err
	}

	return s.Wait()
}

// Start is responsible for kicking off the http server
func (s *Server) Start() error {
	err := s.clientListener.Start(s.listenAddr)
	if err != nil {
		return err
	}

	if s.apiAddr != "" {
		err = s.apiListener.Start(s.apiAddr)
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
