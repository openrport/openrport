package chserver

import (
	"fmt"
	"net/http"
	"os"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/jpillora/requestlog"
	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/server/ports"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

// Config is the configuration for the rport service
type Config struct {
	URL           string
	KeySeed       string
	AuthFile      string
	Auth          string
	Proxy         string
	APIAuth       string
	APIJWTSecret  string
	DocRoot       string
	LogOutput     *os.File
	LogLevel      chshare.LogLevel
	ExcludedPorts mapset.Set
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
func NewServer(config *Config) (*Server, error) {
	s := &Server{
		Logger: chshare.NewLogger("server", config.LogOutput, config.LogLevel),
	}

	privateKey, err := initPrivateKey(config.KeySeed)
	if err != nil {
		return nil, err
	}
	fingerprint := chshare.FingerprintKey(privateKey.PublicKey())
	s.Infof("Fingerprint %s", fingerprint)

	sessionService := NewSessionService(
		ports.NewPortDistributor(config.ExcludedPorts),
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
