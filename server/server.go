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
	"go.etcd.io/bbolt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"

	// sql drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"

	"github.com/patrickmn/go-cache"

	"github.com/realvnc-labs/rport/db/migration/client_groups"
	clientsmigration "github.com/realvnc-labs/rport/db/migration/clients"
	jobsmigration "github.com/realvnc-labs/rport/db/migration/jobs"
	"github.com/realvnc-labs/rport/db/sqlite"
	rportplus "github.com/realvnc-labs/rport/plus"
	alertingcap "github.com/realvnc-labs/rport/plus/capabilities/alerting"
	"github.com/realvnc-labs/rport/server/acme"
	"github.com/realvnc-labs/rport/server/api/jobs"
	"github.com/realvnc-labs/rport/server/api/jobs/schedule"
	"github.com/realvnc-labs/rport/server/api/session"
	"github.com/realvnc-labs/rport/server/auditlog"
	"github.com/realvnc-labs/rport/server/caddy"
	"github.com/realvnc-labs/rport/server/cgroups"
	"github.com/realvnc-labs/rport/server/chconfig"
	"github.com/realvnc-labs/rport/server/clients"
	"github.com/realvnc-labs/rport/server/clientsauth"
	"github.com/realvnc-labs/rport/server/monitoring"
	"github.com/realvnc-labs/rport/server/notifications"
	"github.com/realvnc-labs/rport/server/ports"
	"github.com/realvnc-labs/rport/server/scheduler"
	chshare "github.com/realvnc-labs/rport/share"
	"github.com/realvnc-labs/rport/share/capabilities"
	"github.com/realvnc-labs/rport/share/files"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/models"
	"github.com/realvnc-labs/rport/share/ws"
)

const (
	cleanupMeasurementsInterval = time.Minute * 2
	cleanupAPISessionsInterval  = time.Hour
	cleanupJobsInterval         = time.Hour
	LogNumGoRoutinesInterval    = time.Minute * 2

	DefaultMaxClientDBConnections = 50
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
	acme                *acme.Acme
	alertingService     alertingcap.Service
}

type ServerOpts struct {
	FilesAPI    files.FileAPI
	PlusManager rportplus.Manager
}

// NewServer creates and returns a new rport server
func NewServer(ctx context.Context, config *chconfig.Config, opts *ServerOpts) (*Server, error) {
	var err error

	s := &Server{
		Logger:           logger.NewLogger("server", config.Logging.LogOutput, config.Logging.LogLevel),
		config:           config,
		uiJobWebSockets:  ws.NewWebSocketCache(),
		uploadWebSockets: sync.Map{},
		jobsDoneChannel: jobResultChanMap{
			m: make(map[string]chan *models.Job),
		},
	}

	s.acme = acme.New(s.Logger.Fork("acme"), config.Server.DataDir, config.Server.AcmeHTTPPort)
	if config.Server.InternalTunnelProxyConfig.EnableAcme {
		s.acme.AddHost(config.Server.InternalTunnelProxyConfig.Host)
	}

	filesAPI := opts.FilesAPI
	s.plusManager = opts.PlusManager

	if rportplus.IsPlusEnabled(config.PlusConfig) {
		licCap := s.plusManager.GetLicenseCapabilityEx()
		if licCap == nil {
			return nil, errors.New("failed to get license info capability from rport-plus")
		}

		licCap.SetLicenseInfoAvailableNotifier(s.HandlePlusLicenseInfoAvailable)

		alertingCap := s.plusManager.GetAlertingCapabilityEx()
		if alertingCap != nil {
			s.alertingService, err = s.StartPlusAlertingService(alertingCap, config.Server.DataDir)
			if err != nil {
				return nil, err
			}
			s.Infof("Alerting capability enabled")
		} else {
			s.Infof("Alerting capability not available")
		}
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

	// even if monitoring disabled, always create the monitoring service to support queries of past data etc
	s.monitoringService = monitoring.NewService(monitoringProvider)

	sourceOptions := config.Server.GetSQLiteDataSourceOptions()

	// particularly the client.db needs performant db access, so allow multi-threaded access
	// and use the RetryWhenBusy fn to ensure writes succeed if we get a busy error due to
	// concurrent thread access.
	sourceOptions.MaxOpenConnections = DefaultMaxClientDBConnections

	s.clientDB, err = sqlite.New(
		path.Join(config.Server.DataDir, "clients.db"),
		clientsmigration.AssetNames(),
		clientsmigration.Asset,
		sourceOptions,
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
		s.acme,
	)
	if err != nil {
		return nil, err
	}

	if rportplus.IsPlusEnabled(config.PlusConfig) {
		licCapEx := s.plusManager.GetLicenseCapabilityEx()
		s.clientService.SetPlusLicenseInfoCap(licCapEx)
		s.clientService.SetPlusAlertingServiceCap(s.alertingService)
	}

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

	s.capabilities = capabilities.NewServerCapabilities(&config.Monitoring)

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

	if s.alertingService != nil {
		dispatcher := notifications.NewDispatcher(s.apiListener.notificationsStorage)
		// TODO: (rs): add the scripts dir from the notification config here
		s.alertingService.Run(ctx, ".", dispatcher)
	}
	return s, nil
}

func (s *Server) HandlePlusLicenseInfoAvailable() {
	s.Logger.Debugf("received license info from rport-plus")

	if s.clientListener != nil && s.clientListener.server.clientService != nil {
		s.clientListener.server.clientService.UpdateClientStatus()
	}
}

func (s *Server) StartPlusAlertingService(alertingCap alertingcap.CapabilityEx,
	dataDir string) (as alertingcap.Service, err error) {
	opts := bbolt.DefaultOptions
	bdb, err := bbolt.Open(dataDir+"/alerts.boltdb", 0600, opts)
	if err != nil {
		return nil, err
	}

	err = alertingCap.Init(bdb)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize the alerting service: %w", err)
	}

	as = alertingCap.GetService()

	err = as.LoadDefaultRuleSet()
	if err != nil {
		s.Infof("failed to load latest ruleset: %v", err)
	}

	return as, nil
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
	// generate private key (optionally using seed)
	key, err := chshare.GenerateKey(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key seed: %s", err)
	}
	// convert into ssh.PrivateKey
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

	s.acme.Start()

	// TODO(m-terel): add graceful shutdown of background task
	if s.config.Server.PurgeDisconnectedClients {
		s.Infof("Period to keep disconnected clients is set to %v", s.config.Server.KeepDisconnectedClients)
		go scheduler.Run(ctx, s.Logger, clients.NewCleanupTask(s.Logger, s.clientListener.server.clientService.GetRepo()), s.config.Server.PurgeDisconnectedClientsInterval)
		s.Infof("Task to purge disconnected clients will run with interval %v", s.config.Server.PurgeDisconnectedClientsInterval)
	} else {
		s.Debugf("Task to purge disconnected clients disabled")
	}

	// Run a task to Check the client connections status by sending and receiving pings
	clientsStatusCheckTask := NewClientsStatusCheckTask(
		s.Logger,
		s.clientListener.server.clientService.GetRepo(),
		s.config.Server.CheckClientsConnectionInterval,
		s.config.Server.CheckClientsConnectionTimeout,
	)
	go scheduler.Run(ctx, s.Logger.Fork(fmt.Sprintf("task %T", clientsStatusCheckTask)), clientsStatusCheckTask, s.config.Server.CheckClientsConnectionInterval)
	s.Infof("Task to check the clients connection status will run with interval %v", s.config.Server.CheckClientsConnectionInterval)

	if s.config.Monitoring.Enabled {
		var cleaningPeriod time.Duration
		if s.config.Monitoring.DataStorageDays > 0 {
			s.Infof("Period to keep measurements will be %d day(s)", s.config.Monitoring.DataStorageDays)
			cleaningPeriod = time.Hour * 24 * time.Duration(s.config.Monitoring.DataStorageDays)
		} else {
			s.Infof("Period to keep measurements will be %s", s.config.Monitoring.DataStorageDuration)
			cleaningPeriod = s.config.Monitoring.GetDataStorageDuration()
		}

		monitoringCleanupTask := monitoring.NewCleanupTask(s.Logger, s.monitoringService, cleaningPeriod)
		go scheduler.Run(ctx, s.Logger.Fork(fmt.Sprintf("task %T", monitoringCleanupTask)), monitoringCleanupTask, cleanupMeasurementsInterval)
		s.Infof("Task to cleanup measurements will run with interval %v", cleanupMeasurementsInterval)
	} else {
		s.Infof("Measurement disabled")
	}

	sessionsCleanupTask := session.NewCleanupTask(s.apiListener.apiSessions)
	go scheduler.Run(ctx, s.Logger.Fork(fmt.Sprintf("task %T", sessionsCleanupTask)), sessionsCleanupTask, cleanupAPISessionsInterval)
	s.Infof("Task to cleanup expired api sessions will run with interval %v", cleanupAPISessionsInterval)

	jobsCleanupTask := jobs.NewCleanupTask(s.jobProvider, s.config.Server.JobsMaxResults)
	go scheduler.Run(ctx, s.Logger.Fork(fmt.Sprintf("task %T", jobsCleanupTask)), jobsCleanupTask, cleanupJobsInterval)
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

	// a little more time for everything to settle on shutdown
	time.Sleep(250 * time.Millisecond)

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

	// TODO: (rs):  should we be shutting down the other plugin capabilities here?
	if s.alertingService != nil {
		wg.Go(s.alertingService.Stop)
	}

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
