package chserver

import (
	"context"
	"errors"
	"fmt"
	"path"
	"runtime"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"

	// sql drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"

	"github.com/patrickmn/go-cache"

	"github.com/cloudradar-monitoring/rport/db/migration/client_groups"
	clientsmigration "github.com/cloudradar-monitoring/rport/db/migration/clients"
	jobsmigration "github.com/cloudradar-monitoring/rport/db/migration/jobs"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	rportplus "github.com/cloudradar-monitoring/rport/plus"
	"github.com/cloudradar-monitoring/rport/server/api/jobs"
	"github.com/cloudradar-monitoring/rport/server/api/jobs/schedule"
	"github.com/cloudradar-monitoring/rport/server/api/session"
	"github.com/cloudradar-monitoring/rport/server/auditlog"
	"github.com/cloudradar-monitoring/rport/server/caddy"
	"github.com/cloudradar-monitoring/rport/server/cgroups"
	"github.com/cloudradar-monitoring/rport/server/chconfig"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/server/clientsauth"
	"github.com/cloudradar-monitoring/rport/server/monitoring"
	"github.com/cloudradar-monitoring/rport/server/ports"
	"github.com/cloudradar-monitoring/rport/server/scheduler"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/capabilities"
	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/ws"
)

const (
	cleanupMeasurementsInterval = time.Minute * 2
	cleanupAPISessionsInterval  = time.Hour
	cleanupJobsInterval         = time.Hour
	LogNumGoRoutinesInterval    = time.Minute * 2
)

// Server represents a rport service
type Server struct {
	*logger.Logger
	clientListener      *ClientListener
	apiListener         *APIListener
	config              *chconfig.Config
	clientService       clients.ClientService
	clientDB            *sqlx.DB
	clientAuthProvider  clientsauth.Provider
	jobProvider         JobProvider
	clientGroupProvider cgroups.ClientGroupProvider
	monitoringService   monitoring.Service
	authDB              *sqlx.DB
	uiJobWebSockets     ws.WebSocketCache // used to push job result to UI
	uploadWebSockets    sync.Map
	jobsDoneChannel     jobResultChanMap // used for sequential command execution to know when command is finished
	auditLog            *auditlog.AuditLog
	capabilities        *models.Capabilities
	scheduleManager     *schedule.Manager
	filesAPI            files.FileAPI
	plusManager         rportplus.Manager
	caddyServer         *caddy.Server
}

type ServerOpts struct {
	FilesAPI    files.FileAPI
	PlusManager rportplus.Manager
}

// NewServer creates and returns a new rport server
func NewServer(ctx context.Context, config *chconfig.Config, opts *ServerOpts) (*Server, error) {
	s := &Server{
		Logger:           logger.NewLogger("server", config.Logging.LogOutput, config.Logging.LogLevel),
		config:           config,
		uiJobWebSockets:  ws.NewWebSocketCache(),
		uploadWebSockets: sync.Map{},
		jobsDoneChannel: jobResultChanMap{
			m: make(map[string]chan *models.Job),
		},
	}

	filesAPI := opts.FilesAPI
	s.plusManager = opts.PlusManager

	if s.config.PlusEnabled() {
		licCap := s.plusManager.GetLicenseCapabilityEx()
		if licCap == nil {
			return nil, errors.New("failed to get license info capability from rport-plus")
		}

		licCap.SetLicenseInfoAvailableNotifier(s.HandlePlusLicenseInfoAvailable)
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

	jobsDB, err := sqlite.New(
		path.Join(config.Server.DataDir, "jobs.db"),
		jobsmigration.AssetNames(),
		jobsmigration.Asset,
		config.Server.GetSQLiteDataSourceOptions(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create jobs DB instance: %v", err)
	}

	s.jobProvider = jobs.NewSqliteProvider(jobsDB, s.Logger)

	groupsDB, err := sqlite.New(
		path.Join(config.Server.DataDir, "client_groups.db"),
		client_groups.AssetNames(),
		client_groups.Asset,
		config.Server.GetSQLiteDataSourceOptions(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create client_groups DB instance: %v", err)
	}

	s.clientGroupProvider, err = cgroups.NewSqliteProvider(groupsDB)
	if err != nil {
		return nil, err
	}

	// create monitoringProvider and monitoringService
	monitoringProvider, err := monitoring.NewSqliteProvider(
		path.Join(config.Server.DataDir, "monitoring.db"),
		config.Server.GetSQLiteDataSourceOptions(),
		s.Logger,
	)
	if err != nil {
		return nil, err
	}
	s.monitoringService = monitoring.NewService(monitoringProvider)

	s.clientDB, err = sqlite.New(
		path.Join(config.Server.DataDir, "clients.db"),
		clientsmigration.AssetNames(),
		clientsmigration.Asset,
		config.Server.GetSQLiteDataSourceOptions(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create clients DB instance: %v", err)
	}

	// keepDisconnectedClients is nil when cleanup of clients is disabled (keep clients forever)
	var keepDisconnectedClients *time.Duration
	if config.Server.PurgeDisconnectedClients {
		keepDisconnectedClients = &config.Server.KeepDisconnectedClients
	}

	s.clientService, err = clients.InitClientService(
		ctx,
		&s.config.Server.InternalTunnelProxyConfig,
		ports.NewPortDistributor(config.AllowedPorts()),
		s.clientDB,
		keepDisconnectedClients,
		s.Logger,
	)
	if err != nil {
		return nil, err
	}
	s.clientService.SetPlusManager(s.plusManager)

	s.auditLog, err = auditlog.New(
		logger.NewLogger("auditlog", config.Logging.LogOutput, config.Logging.LogLevel),
		s.clientService,
		s.config.Server.DataDir,
		s.config.API.AuditLog,
		s.config.Server.GetSQLiteDataSourceOptions(),
	)
	if err != nil {
		return nil, err
	}

	if config.Database.Driver != "" {
		s.authDB, err = sqlx.Connect(config.Database.Driver, config.Database.Dsn)
		if err != nil {
			return nil, err
		}
		s.Infof("DB: successfully connected to %s", config.Database.DsnForLogs())
	}

	s.clientAuthProvider, err = getClientProvider(config, s.authDB)
	if err != nil {
		return nil, err
	}

	s.clientListener, err = NewClientListener(s, privateKey)
	if err != nil {
		return nil, err
	}

	s.filesAPI = filesAPI

	s.apiListener, err = NewAPIListener(s, fingerprint)
	if err != nil {
		return nil, err
	}

	s.capabilities = capabilities.NewServerCapabilities()

	s.scheduleManager, err = schedule.New(ctx, s.Logger, jobsDB, s.apiListener, config.Server.RunRemoteCmdTimeoutSec)
	if err != nil {
		return nil, err
	}

	if s.config.CaddyEnabled() {
		cfg := s.config
		caddyLog := logger.NewLogger("caddy", cfg.Logging.LogOutput, cfg.Logging.LogLevel)

		baseConfig, err := cfg.WriteCaddyBaseConfig(&cfg.Caddy)
		if err != nil {
			return nil, err
		}

		caddy.HostDomainSocket = baseConfig.GlobalSettings.AdminSocket

		s.caddyServer = caddy.NewCaddyServer(&cfg.Caddy, caddyLog)
		s.clientService.SetCaddyAPI(s.caddyServer)
	}

	return s, nil
}

func (s *Server) HandlePlusLicenseInfoAvailable() {
	s.Logger.Debugf("received license info from rport-plus")

	s.plusManager.SetPlusLicenseInfoAvailable(true)

	if s.clientListener != nil && s.clientListener.clientService != nil {
		s.clientListener.clientService.UpdateClientStatus()
	}
}

func getClientProvider(config *chconfig.Config, db *sqlx.DB) (clientsauth.Provider, error) {
	if config.Server.AuthTable != "" {
		return clientsauth.NewDatabaseProvider(db, config.Server.AuthTable), nil
	}

	if config.Server.AuthFile != "" {
		return clientsauth.NewFileProvider(config.Server.AuthFile, cache.New(60*time.Minute, 15*time.Minute)), nil
	}

	if config.Server.Auth != "" {
		return clientsauth.NewSingleProvider(config.Server.AuthID, config.Server.AuthPassword), nil
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
func (s *Server) Run(ctx context.Context) error {
	if err := s.Start(ctx); err != nil {
		return err
	}

	// TODO(m-terel): add graceful shutdown of background task
	if s.config.Server.PurgeDisconnectedClients {
		s.Infof("Period to keep disconnected clients is set to %v", s.config.Server.KeepDisconnectedClients)
		go scheduler.Run(ctx, s.Logger, clients.NewCleanupTask(s.Logger, s.clientListener.clientService.GetRepo()), s.config.Server.PurgeDisconnectedClientsInterval)
		s.Infof("Task to purge disconnected clients will run with interval %v", s.config.Server.PurgeDisconnectedClientsInterval)
	} else {
		s.Debugf("Task to purge disconnected clients disabled")
	}

	//Run a task to Check the client connections status by sending and receiving pings
	go scheduler.Run(ctx, s.Logger, NewClientsStatusCheckTask(
		s.Logger,
		s.clientListener.clientService.GetRepo(),
		s.config.Server.CheckClientsConnectionInterval,
		s.config.Server.CheckClientsConnectionTimeout,
	), s.config.Server.CheckClientsConnectionInterval)
	s.Infof("Task to check the clients connection status will run with interval %v", s.config.Server.CheckClientsConnectionInterval)

	cleaningPeriod := time.Hour * 24 * time.Duration(s.config.Monitoring.DataStorageDays)
	go scheduler.Run(ctx, s.Logger, monitoring.NewCleanupTask(s.Logger, s.monitoringService, cleaningPeriod), cleanupMeasurementsInterval)
	s.Infof("Task to cleanup measurements will run with interval %v", cleanupMeasurementsInterval)

	go scheduler.Run(ctx, s.Logger, session.NewCleanupTask(s.apiListener.apiSessions), cleanupAPISessionsInterval)
	s.Infof("Task to cleanup expired api sessions will run with interval %v", cleanupAPISessionsInterval)

	go scheduler.Run(ctx, s.Logger, jobs.NewCleanupTask(s.jobProvider, s.config.Server.JobsMaxResults), cleanupJobsInterval)
	s.Infof("Task to cleanup jobs will run with interval %v", cleanupJobsInterval)

	// Only on debug mode, log the number of running go routines
	if s.config.Logging.LogLevel == logger.LogLevelDebug {
		go func() {
			for {
				s.Logger.Debugf("Number of running go routines: %d", runtime.NumGoroutine())
				time.Sleep(LogNumGoRoutinesInterval)
			}
		}()
	}

	err := s.Wait()

	// allow time for go-routines (and the caddy server) to process their cancellations
	time.Sleep(250 * time.Millisecond)

	s.Close()

	return err
}

// Start is responsible for kicking off the http server
func (s *Server) Start(ctx context.Context) error {
	s.Logger.Infof("will start server on %s", s.config.Server.ListenAddress)
	err := s.clientListener.Start(ctx, s.config.Server.ListenAddress)
	if err != nil {
		return err
	}

	if s.config.API.Address != "" {
		err = s.apiListener.Start(ctx, s.config.API.Address)
	}

	if s.config.CaddyEnabled() {
		err = s.caddyServer.Start(ctx)
	}

	return err
}

func (s *Server) Wait() error {
	wg := &errgroup.Group{}
	wg.Go(s.clientListener.Wait)
	wg.Go(s.apiListener.Wait)
	// if caddy configured then also setup a dependencies on the caddy server running
	if s.config.CaddyEnabled() {
		wg.Go(s.caddyServer.Wait)
	}

	return wg.Wait()
}

func (s *Server) Close() error {
	s.Logger.Debugf("closing server")
	wg := &errgroup.Group{}

	wg.Go(s.clientListener.Close)
	wg.Go(s.apiListener.Close)
	if s.config.CaddyEnabled() {
		wg.Go(s.caddyServer.Close)
	}

	if s.authDB != nil {
		wg.Go(s.authDB.Close)
	}
	wg.Go(s.clientDB.Close)
	wg.Go(s.jobProvider.Close)
	wg.Go(s.clientGroupProvider.Close)
	wg.Go(s.uiJobWebSockets.CloseConnections)
	if s.auditLog != nil {
		wg.Go(s.auditLog.Close)
	}

	s.uploadWebSockets.Range(func(key, value interface{}) bool {
		if wsConn, ok := value.(*ws.ConcurrentWebSocket); ok {
			wg.Go(wsConn.Close)
		}
		return true
	})

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

// TODO: only used for testing purposes. we should review the approach.
func (m *jobResultChanMap) GetAllKeys() (jobIDs []string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	jobIDs = make([]string, 0, len(m.m))
	for k := range m.m {
		jobIDs = append(jobIDs, k)
	}
	return jobIDs
}
