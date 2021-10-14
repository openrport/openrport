package chserver

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/cloudradar-monitoring/rport/server/monitoring"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"

	// sql drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"

	"github.com/cloudradar-monitoring/rport/server/api/jobs"
	"github.com/cloudradar-monitoring/rport/server/cgroups"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/server/clientsauth"
	"github.com/cloudradar-monitoring/rport/server/ports"
	"github.com/cloudradar-monitoring/rport/server/scheduler"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/ws"
)

// Server represents a rport service
type Server struct {
	*chshare.Logger
	clientListener      *ClientListener
	apiListener         *APIListener
	config              *Config
	clientService       *ClientService
	clientProvider      clients.ClientProvider
	clientAuthProvider  clientsauth.Provider
	jobProvider         JobProvider
	clientGroupProvider cgroups.ClientGroupProvider
	monitoringService   monitoring.Service
	db                  *sqlx.DB
	uiJobWebSockets     ws.WebSocketCache // used to push job result to UI
	jobsDoneChannel     jobResultChanMap  // used for sequential command execution to know when command is finished
}

// NewServer creates and returns a new rport server
func NewServer(config *Config, filesAPI files.FileAPI) (*Server, error) {
	ctx := context.Background()
	s := &Server{
		Logger:          chshare.NewLogger("server", config.Logging.LogOutput, config.Logging.LogLevel),
		config:          config,
		uiJobWebSockets: ws.NewWebSocketCache(),
		jobsDoneChannel: jobResultChanMap{
			m: make(map[string]chan *models.Job),
		},
	}

	privateKey, err := initPrivateKey(config.Server.KeySeed)
	if err != nil {
		return nil, err
	}
	fingerprint := chshare.FingerprintKey(privateKey.PublicKey())
	s.Infof("Fingerprint %s", fingerprint)

	s.Infof("data directory path: %q", config.Server.DataDir)
	if config.Server.DataDir == "" {
		return nil, errors.New("data directory cannot be empty")
	}

	// create --data-dir path if not exist
	if makedirErr := filesAPI.MakeDirAll(config.Server.DataDir); makedirErr != nil {
		return nil, fmt.Errorf("failed to create data dir %q: %v", config.Server.DataDir, makedirErr)
	}

	// store fingerprint in file
	fingerprintFile := path.Join(config.Server.DataDir, "rportd-fingerprint.txt")
	if err := filesAPI.Write(fingerprintFile, fingerprint); err != nil {
		// juts log it and proceed
		s.Errorf("Failed to store fingerprint %q in file %q: %v", fingerprint, fingerprintFile, err)
	}

	s.jobProvider, err = jobs.NewSqliteProvider(path.Join(config.Server.DataDir, "jobs.db"), s.Logger)
	if err != nil {
		return nil, err
	}

	s.clientGroupProvider, err = cgroups.NewSqliteProvider(path.Join(config.Server.DataDir, "client_groups.db"))
	if err != nil {
		return nil, err
	}

	// create monitoringProvider and monitoringService
	monitoringProvider, err := monitoring.NewSqliteProvider(path.Join(config.Server.DataDir, "monitoring.db?_journal_mode=WAL"), s.Logger)
	if err != nil {
		return nil, err
	}
	s.monitoringService = monitoring.NewService(monitoringProvider)

	s.clientProvider, err = clients.NewSqliteProvider(
		path.Join(config.Server.DataDir, "clients.db"),
		config.Server.KeepLostClients,
	)
	if err != nil {
		return nil, err
	}

	var keepLostClients *time.Duration
	if config.Server.KeepLostClients > 0 {
		keepLostClients = &config.Server.KeepLostClients
	}

	s.clientService, err = InitClientService(
		ctx,
		ports.NewPortDistributor(config.AllowedPorts()),
		s.clientProvider,
		keepLostClients,
		s.Logger,
	)
	if err != nil {
		return nil, err
	}

	if config.Database.driver != "" {
		s.db, err = sqlx.Connect(config.Database.driver, config.Database.dsn)
		if err != nil {
			return nil, err
		}
		s.Infof("DB: successfully connected to %s", config.Database.dsnForLogs())
	}

	s.clientAuthProvider, err = getClientProvider(config, s.db)
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

func getClientProvider(config *Config, db *sqlx.DB) (clientsauth.Provider, error) {
	if config.Server.AuthTable != "" {
		dbProvider := clientsauth.NewDatabaseProvider(db, config.Server.AuthTable)
		cachedProvider, err := clientsauth.NewCachedProvider(dbProvider)
		if err != nil {
			return nil, err
		}
		return cachedProvider, nil
	}

	if config.Server.AuthFile != "" {
		fileProvider := clientsauth.NewFileProvider(config.Server.AuthFile)
		cachedProvider, err := clientsauth.NewCachedProvider(fileProvider)
		if err != nil {
			return nil, err
		}
		return cachedProvider, nil
	}

	if config.Server.Auth != "" {
		return clientsauth.NewSingleProvider(config.Server.authID, config.Server.authPassword), nil
	}

	return nil, errors.New("client authentication must to be enabled: set either 'auth' or 'auth_file'")
}

func initPrivateKey(seed string) (ssh.Signer, error) {
	//generate private key (optionally using seed)
	key, err := chshare.GenerateKey(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key seed: %s", err)
	}
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

	s.Infof("Variable to keep lost clients is set to %v", s.config.Server.KeepLostClients)

	// TODO(m-terel): add graceful shutdown of background task
	go scheduler.Run(ctx, s.Logger, clients.NewCleanupTask(s.Logger, s.clientListener.clientService.repo), s.config.Server.CleanupClients)
	s.Infof("Task to cleanup obsolete clients will run with interval %v", s.config.Server.CleanupClients)

	// TODO(gkahrer): do we need a config parameter here ?
	cleanupMeasurementsInterval := time.Minute * 2
	go scheduler.Run(ctx, s.Logger, monitoring.NewCleanupTask(s.Logger, s.monitoringService, s.config.Monitoring.DataStorageDays), cleanupMeasurementsInterval)
	s.Infof("Task to cleanup measurements will run with interval %v", cleanupMeasurementsInterval)

	return s.Wait()
}

// Start is responsible for kicking off the http server
func (s *Server) Start() error {
	s.Logger.Infof("will start server on %s", s.config.Server.ListenAddress)
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
	if s.db != nil {
		wg.Go(s.db.Close)
	}
	wg.Go(s.clientProvider.Close)
	wg.Go(s.jobProvider.Close)
	wg.Go(s.clientGroupProvider.Close)
	wg.Go(s.uiJobWebSockets.CloseConnections)
	return wg.Wait()
}

// jobResultChanMap is thread safe map with [jobID, chan *models.Job] pairs.
type jobResultChanMap struct {
	m  map[string]chan *models.Job
	mu sync.RWMutex
}

func (m *jobResultChanMap) Set(jobID string, done chan *models.Job) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[jobID] = done
}

func (m *jobResultChanMap) Del(jobID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.m, jobID)
}

func (m *jobResultChanMap) Get(jobID string) chan *models.Job {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.m[jobID]
}
