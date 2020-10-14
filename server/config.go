package chserver

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/jpillora/requestlog"

	"github.com/cloudradar-monitoring/rport/server/ports"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

type APIConfig struct {
	Address   string `mapstructure:"address"`
	Auth      string `mapstructure:"auth"`
	AuthFile  string `mapstructure:"auth_file"`
	JWTSecret string `mapstructure:"jwt_secret"`
	DocRoot   string `mapstructure:"doc_root"`
	CertFile  string `mapstructure:"cert_file"`
	KeyFile   string `mapstructure:"key_file"`
}

const (
	DefaultCSRFileName = "csr.json"

	MinKeepLostClients = time.Second
	MaxKeepLostClients = 7 * 24 * time.Hour
)

type LogConfig struct {
	LogOutput chshare.LogOutput `mapstructure:"log_file"`
	LogLevel  chshare.LogLevel  `mapstructure:"log_level"`
}

type ServerConfig struct {
	ListenAddress              string        `mapstructure:"address"`
	URL                        string        `mapstructure:"url"`
	KeySeed                    string        `mapstructure:"key_seed"`
	AuthFile                   string        `mapstructure:"auth_file"`
	Auth                       string        `mapstructure:"auth"`
	Proxy                      string        `mapstructure:"proxy"`
	ExcludedPortsRaw           []string      `mapstructure:"excluded_ports"`
	DataDir                    string        `mapstructure:"data_dir"`
	KeepLostClients            time.Duration `mapstructure:"keep_lost_clients"`
	SaveClients                time.Duration `mapstructure:"save_clients_interval"`
	CleanupClients             time.Duration `mapstructure:"cleanup_clients_interval"`
	MaxRequestBytes            int64         `mapstructure:"max_request_bytes"`
	CheckPortTimeout           time.Duration `mapstructure:"check_port_timeout"`
	RunRemoteCmdTimeout        time.Duration `mapstructure:"run_remote_cmd_timeout"`
	AuthWrite                  bool          `mapstructure:"auth_write"`
	AuthMultiuseCreds          bool          `mapstructure:"auth_multiuse_creds"`
	EquateAuthusernameClientid bool          `mapstructure:"equate_authusername_clientid"`
	SaveClientsAuth            time.Duration `mapstructure:"save_clients_auth_interval"`
	AllowRoot                  bool          `mapstructure:"allow_root"`

	excludedPorts mapset.Set
}

type Config struct {
	Server  ServerConfig `mapstructure:"server"`
	Logging LogConfig    `mapstructure:"logging"`
	API     APIConfig    `mapstructure:"api"`
}

func (c *Config) CSRFilePath() string {
	return c.Server.DataDir + string(os.PathSeparator) + DefaultCSRFileName
}

func (c *Config) InitRequestLogOptions() *requestlog.Options {
	o := requestlog.DefaultOptions
	o.Writer = c.Logging.LogOutput.File
	o.Filter = func(r *http.Request, code int, duration time.Duration, size int64) bool {
		return c.Logging.LogLevel == chshare.LogLevelInfo || c.Logging.LogLevel == chshare.LogLevelDebug
	}
	return &o
}

func (c *Config) ExcludedPorts() mapset.Set {
	return c.Server.excludedPorts
}

func (c *Config) ParseAndValidate() error {
	if c.Server.URL == "" {
		c.Server.URL = "http://" + c.Server.ListenAddress
	}
	u, err := url.Parse(c.Server.URL)
	if err != nil {
		return fmt.Errorf("invalid connection url %s. %s", u, err)
	}
	if u.Host == "" {
		return fmt.Errorf("invalid connection url %s. must be absolute url", u)
	}

	if c.API.DocRoot != "" && c.API.Address == "" {
		return errors.New("to use document root you need to specify API address")
	}

	excludedPorts, err := ports.TryParsePortRanges(c.Server.ExcludedPortsRaw)
	if err != nil {
		return fmt.Errorf("can't parse excluded ports: %s", err)
	}
	c.Server.excludedPorts = excludedPorts

	if c.API.JWTSecret == "" {
		c.API.JWTSecret, err = generateJWTSecret()
		if err != nil {
			return err
		}
	}

	if c.Server.DataDir == "" {
		return errors.New("'data directory path' cannot be empty")
	}

	if c.Server.KeepLostClients != 0 && (c.Server.KeepLostClients.Nanoseconds() < MinKeepLostClients.Nanoseconds() ||
		c.Server.KeepLostClients.Nanoseconds() > MaxKeepLostClients.Nanoseconds()) {
		return fmt.Errorf("expected 'Keep Lost Clients' can be in range [%v, %v], actual: %v", MinKeepLostClients, MaxKeepLostClients, c.Server.KeepLostClients)
	}

	return nil
}

func generateJWTSecret() (string, error) {
	data := make([]byte, 10)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("can't generate API JWT secret: %s", err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}
