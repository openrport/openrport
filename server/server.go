package chserver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

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
func NewServer(config *Config) (*Server, error) {
	s := &Server{
		Logger:     chshare.NewLogger("server", config.Logging.LogOutput, config.Logging.LogLevel),
		listenAddr: config.Server.ListenAddress,
		apiAddr:    config.API.Address,
		config:     config,
	}

	privateKey, err := initPrivateKey(config.Server.KeySeed)
	if err != nil {
		return nil, err
	}
	fingerprint := chshare.FingerprintKey(privateKey.PublicKey())
	s.Infof("Fingerprint %s", fingerprint)

	s.Infof("data directory path: %q", config.Server.DataDir)
	// create --data-dir path if not exist
	if makedirErr := os.MkdirAll(config.Server.DataDir, os.ModePerm); makedirErr != nil {
		s.Errorf("Failed to create data dir %q: %v", config.Server.DataDir, makedirErr)
	}

	var keepLostClients *time.Duration
	if config.Server.KeepLostClients > 0 {
		keepLostClients = &config.Server.KeepLostClients
	}
	// TODO(m-terel): add a check whether a file exists
	initSessions, err := sessions.GetInitStateFromFile(config.CSRFilePath(), keepLostClients)
	if err != nil {
		if len(initSessions) == 0 {
			s.Errorf("Failed to get init CSR state from file %q: %v", config.CSRFilePath(), err)
		} else {
			s.Infof("Partial failure. Successfully read %d sessions from file %q. Error: %v", len(initSessions), config.CSRFilePath(), err)
		}
		// proceed further
	}
	repo := sessions.NewSessionRepository(initSessions, keepLostClients)

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

	s.apiListener, err = NewAPIListener(config, sessionService, rClients, clientProvider, config.Server.AuthWrite, fingerprint)
	if err != nil {
		return nil, err
	}

	return s, nil
}

type ClientProvider interface {
	GetAll() ([]*clients.Client, error)
}

func getClientProvider(log *chshare.Logger, config *Config) (ClientProvider, error) {
	if config.Server.AuthFile != "" && config.Server.Auth != "" {
		return nil, errors.New("'auth_file' and 'auth' are both set: expected only one of them ")
	}

	if config.Server.AuthFile != "" {
		return clients.NewFileClients(log, config.Server.AuthFile), nil
	}

	if config.Server.Auth != "" {
		return clients.NewSingleClient(log, config.Server.Auth), nil
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
	ctx := context.Background()

	if err := s.Start(); err != nil {
		return err
	}

	if s.config.Server.KeepLostClients > 0 {
		repo := s.clientListener.sessionService.repo
		s.Infof("Variable to keep lost clients is set. Enables keeping disconnected clients for period: %v", s.config.Server.KeepLostClients)
		s.Infof("csr file path: %q", s.config.CSRFilePath())

		var lockableClients *clients.ClientCache
		if !s.config.Server.AuthMultiuseCreds {
			lockableClients = s.clientListener.allClients
		}
		go scheduler.Run(ctx, s.Logger, sessions.NewCleanupTask(s.Logger, repo, lockableClients), s.config.Server.CleanupClients)
		s.Infof("Task to cleanup obsolete clients will run with interval %v", s.config.Server.CleanupClients)
		// TODO(m-terel): add graceful shutdown of background task
		go scheduler.Run(ctx, s.Logger, sessions.NewSaveToFileTask(s.Logger, repo, s.config.CSRFilePath()), s.config.Server.SaveClients)
		s.Infof("Task to save clients to disk will run with interval %v", s.config.Server.SaveClients)
	}

	if s.config.Server.AuthFile != "" && s.config.Server.AuthWrite {
		ctx := context.Background()
		// TODO(m-terel): add graceful shutdown when a global shutdown mechanism will be ready and working
		go scheduler.Run(ctx, s.Logger, clients.NewSaveToFileTask(s.Logger, s.apiListener.clientCache, s.config.Server.AuthFile), s.config.Server.SaveClientsAuth)
		s.Infof("Task to save rport clients auth credentials to disk will run with interval %v", s.config.Server.SaveClientsAuth)
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
