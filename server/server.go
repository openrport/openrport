package chserver

import (
	"context"
	"errors"
	"fmt"
	"path"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"

	"github.com/cloudradar-monitoring/rport/server/api/jobs"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/server/ports"
	"github.com/cloudradar-monitoring/rport/server/scheduler"
	"github.com/cloudradar-monitoring/rport/server/sessions"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/files"
)

// Server represents a rport service
type Server struct {
	*chshare.Logger
	clientListener *ClientListener
	apiListener    *APIListener
	config         *Config
	sessionService *SessionService
	clientCache    *clients.ClientCache
	jobProvider    JobProvider
}

// NewServer creates and returns a new rport server
func NewServer(config *Config, filesAPI files.FileAPI) (*Server, error) {
	s := &Server{
		Logger: chshare.NewLogger("server", config.Logging.LogOutput, config.Logging.LogLevel),
		config: config,
	}

	privateKey, err := initPrivateKey(config.Server.KeySeed)
	if err != nil {
		return nil, err
	}
	fingerprint := chshare.FingerprintKey(privateKey.PublicKey())
	s.Infof("Fingerprint %s", fingerprint)

	s.Infof("data directory path: %q", config.Server.DataDir)
	if config.Server.DataDir != "" {
		// create --data-dir path if not exist
		if makedirErr := filesAPI.MakeDirAll(config.Server.DataDir); makedirErr != nil {
			s.Errorf("Failed to create data dir %q: %v", config.Server.DataDir, makedirErr)
		} else {
			jobsDir := getJobsDirectory(config.Server.DataDir)
			if err := filesAPI.MakeDirAll(jobsDir); err != nil {
				s.Errorf("Failed to create jobs dir %q: %s", jobsDir, err)
			} else {
				s.jobProvider = jobs.NewFileProvider(filesAPI, jobsDir)
			}
		}
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

	s.sessionService = NewSessionService(
		ports.NewPortDistributor(config.ExcludedPorts()),
		repo,
	)

	clientProvider, err := getClientProvider(s.Logger, config)
	if err != nil {
		return nil, err
	}
	s.clientCache, err = clients.NewClientCache(clientProvider)
	if err != nil {
		return nil, err
	}

	s.clientListener, err = NewClientListener(s, privateKey)
	if err != nil {
		return nil, err
	}

	s.apiListener, err = NewAPIListener(s, fingerprint)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func getJobsDirectory(datDir string) string {
	if datDir == "" {
		return ""
	}
	return path.Join(datDir, "jobs")
}

func getClientProvider(log *chshare.Logger, config *Config) (clients.Provider, error) {
	if config.Server.AuthFile != "" && config.Server.Auth != "" {
		return nil, errors.New("'auth_file' and 'auth' are both set: expected only one of them ")
	}

	if config.Server.AuthFile != "" {
		return clients.NewFileClients(log, config.Server.AuthFile), nil
	}

	if config.Server.Auth != "" {
		return clients.NewSingleClient(log, config.Server.Auth), nil
	}

	return nil, errors.New("client authentication must to be enabled: set either 'auth' or 'auth_file'")
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
			lockableClients = s.clientListener.clientCache
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
	err := s.clientListener.Start(s.config.Server.ListenAddress)
	if err != nil {
		return err
	}

	if s.config.API.Address != "" {
		err = s.apiListener.Start(s.config.API.Address)
	}
	return err
}

func (s *Server) Wait() error {
	wg := &errgroup.Group{}
	wg.Go(s.clientListener.Wait)
	wg.Go(s.apiListener.Wait)
	return wg.Wait()
}

func (s *Server) Close() error {
	wg := &errgroup.Group{}
	wg.Go(s.clientListener.Close)
	wg.Go(s.apiListener.Close)
	return wg.Wait()
}
