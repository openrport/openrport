package chserver

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/server/ports"
	"github.com/cloudradar-monitoring/rport/server/scheduler"
	"github.com/cloudradar-monitoring/rport/server/sessions"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

// Server represents a rport service
type Server struct {
	*chshare.Logger
	listenAddr     string
	apiAddr        string
	clientListener *ClientListener
	apiListener    *APIListener
	config         *Config
}

// NewServer creates and returns a new rport server
func NewServer(config *Config, repo *sessions.ClientSessionRepository) (*Server, error) {
	s := &Server{
		Logger:     chshare.NewLogger("server", config.LogOutput, config.LogLevel),
		listenAddr: config.ListenAddress,
		apiAddr:    config.API.Address,
		config:     config,
	}

	privateKey, err := initPrivateKey(config.KeySeed)
	if err != nil {
		return nil, err
	}
	fingerprint := chshare.FingerprintKey(privateKey.PublicKey())
	s.Infof("Fingerprint %s", fingerprint)

	sessionService := NewSessionService(
		ports.NewPortDistributor(config.ExcludedPorts()),
		repo,
	)

	clientProvider, err := getClientProvider(s.Logger, config)
	if err != nil {
		return nil, err
	}

	var rClients *clients.ClientCache
	if clientProvider != nil {
		s.Infof("Client authentication enabled.")

		all, InErr := clientProvider.GetAll()
		if InErr != nil {
			return nil, InErr
		}
		rClients = clients.NewClientCache(all)
	}

	s.clientListener, err = NewClientListener(config, sessionService, rClients, privateKey)
	if err != nil {
		return nil, err
	}

	s.apiListener, err = NewAPIListener(config, sessionService, rClients, clientProvider, config.AuthWrite, fingerprint)
	if err != nil {
		return nil, err
	}

	return s, nil
}

type ClientProvider interface {
	GetAll() ([]*clients.Client, error)
}

func getClientProvider(log *chshare.Logger, config *Config) (ClientProvider, error) {
	if config.AuthFile != "" && config.Auth != "" {
		return nil, errors.New("'auth_file' and 'auth' are both set: expected only one of them ")
	}

	if config.AuthFile != "" {
		return clients.NewFileClients(log, config.AuthFile), nil
	}

	if config.Auth != "" {
		return clients.NewSingleClient(log, config.Auth), nil
	}

	return nil, nil
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

	if s.config.AuthFile != "" && s.config.AuthWrite {
		ctx := context.Background()
		// TODO(m-terel): add graceful shutdown when a global shutdown mechanism will be ready and working
		go scheduler.Run(ctx, s.Logger, clients.NewSaveToFileTask(s.Logger, s.apiListener.clientCache, s.config.AuthFile), s.config.SaveClientsAuth)
		s.Infof("Task to save rport clients auth credentials to disk will run with interval %v", s.config.SaveClientsAuth)
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
