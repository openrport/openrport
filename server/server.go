package chserver

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/jpillora/requestlog"
	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/ports"
	"github.com/cloudradar-monitoring/rport/server/sessions"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

const (
	MinKeepLostClients = time.Second
	MaxKeepLostClients = 7 * 24 * time.Hour
)

// Config is the configuration for the rport service
type Config struct {
	URL             string
	KeySeed         string
	AuthFile        string
	Auth            string
	Proxy           string
	APIAuth         string
	APIJWTSecret    string
	DocRoot         string
	LogOutput       *os.File
	LogLevel        chshare.LogLevel
	ExcludedPorts   mapset.Set
	DataDir         string
	CSRFileName     string
	KeepLostClients time.Duration
	SaveClients     time.Duration
	CleanupClients  time.Duration
}

func (c *Config) CSRFilePath() string {
	return c.DataDir + DirectorySeperator + c.CSRFileName
}

func (c *Config) InitRequestLogOptions() *requestlog.Options {
	o := requestlog.DefaultOptions
	o.Writer = c.LogOutput
	o.Filter = func(r *http.Request, code int, duration time.Duration, size int64) bool {
		return c.LogLevel == chshare.LogLevelInfo || c.LogLevel == chshare.LogLevelDebug
	}
	return &o
}

// Server represents a rport service
type Server struct {
	*chshare.Logger
	clientListener *ClientListener
	apiListener    *APIListener
}

// NewServer creates and returns a new rport server
func NewServer(config *Config, repo *sessions.ClientSessionRepository) (*Server, error) {
	s := &Server{
		Logger: chshare.NewLogger("server", config.LogOutput, config.LogLevel),
	}

	if config.DataDir == "" {
		return nil, errors.New("'data directory path' cannot be empty")
	}
	s.Infof("data directory path: %q", config.DataDir)

	if config.CSRFileName == "" {
		return nil, errors.New("'csr filename' cannot be empty")
	}

	s.Infof("csr file path: %q", config.CSRFilePath())

	if config.KeepLostClients != 0 && (config.KeepLostClients.Nanoseconds() < MinKeepLostClients.Nanoseconds() ||
		config.KeepLostClients.Nanoseconds() > MaxKeepLostClients.Nanoseconds()) {
		return nil, fmt.Errorf("expected 'Keep Lost Clients' can be in range [%v, %v], actual: %v", MinKeepLostClients, MaxKeepLostClients, config.KeepLostClients)
	}

	privateKey, err := initPrivateKey(config.KeySeed)
	if err != nil {
		return nil, err
	}
	fingerprint := chshare.FingerprintKey(privateKey.PublicKey())
	s.Infof("Fingerprint %s", fingerprint)

	sessionService := NewSessionService(
		ports.NewPortDistributor(config.ExcludedPorts),
		repo,
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
