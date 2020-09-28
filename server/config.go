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
}

const (
	MinKeepLostClients = time.Second
	MaxKeepLostClients = 7 * 24 * time.Hour
)

type Config struct {
	LogOutput chshare.LogOutput `mapstructure:"log_file"`
	LogLevel  chshare.LogLevel  `mapstructure:"log_level"`

	ListenAddress              string        `mapstructure:"address"`
	URL                        string        `mapstructure:"url"`
	KeySeed                    string        `mapstructure:"key_seed"`
	AuthFile                   string        `mapstructure:"auth_file"`
	Auth                       string        `mapstructure:"auth"`
	Proxy                      string        `mapstructure:"proxy"`
	API                        APIConfig     `mapstructure:"api"`
	ExcludedPortsRaw           []string      `mapstructure:"excluded_ports"`
	DataDir                    string        `mapstructure:"data_dir"`
	CSRFileName                string        `mapstructure:"csr_file_name"`
	KeepLostClients            time.Duration `mapstructure:"keep_lost_clients"`
	SaveClients                time.Duration `mapstructure:"save_clients_interval"`
	CleanupClients             time.Duration `mapstructure:"cleanup_clients_interval"`
	MaxRequestBytes            int64         `mapstructure:"max_request_bytes"`
	CheckPortTimeout           time.Duration `mapstructure:"check_port_timeout"`
	AuthWrite                  bool          `mapstructure:"auth_write"`
	AuthMultiuseCreds          bool          `mapstructure:"auth_multiuse_creds"`
	EquateAuthusernameClientid bool          `mapstructure:"equate_authusername_clientid"`
	SaveClientsAuth            time.Duration `mapstructure:"save_clients_auth_interval"`

	excludedPorts mapset.Set
}

func (c *Config) CSRFilePath() string {
	return c.DataDir + string(os.PathSeparator) + c.CSRFileName
}

func (c *Config) InitRequestLogOptions() *requestlog.Options {
	o := requestlog.DefaultOptions
	o.Writer = c.LogOutput.File
	o.Filter = func(r *http.Request, code int, duration time.Duration, size int64) bool {
		return c.LogLevel == chshare.LogLevelInfo || c.LogLevel == chshare.LogLevelDebug
	}
	return &o
}

func (c *Config) ExcludedPorts() mapset.Set {
	return c.excludedPorts
}

func (c *Config) ParseAndValidate() error {
	if c.URL == "" {
		c.URL = "http://" + c.ListenAddress
	}
	u, err := url.Parse(c.URL)
	if err != nil {
		return fmt.Errorf("invalid connection url %s. %s", u, err)
	}
	if u.Host == "" {
		return fmt.Errorf("invalid connection url %s. must be absolute url", u)
	}

	if c.API.DocRoot != "" && c.API.Address == "" {
		return errors.New("to use document root you need to specify API address")
	}

	excludedPorts, err := ports.TryParsePortRanges(c.ExcludedPortsRaw)
	if err != nil {
		return fmt.Errorf("can't parse excluded ports: %s", err)
	}
	c.excludedPorts = excludedPorts

	if c.API.JWTSecret == "" {
		c.API.JWTSecret, err = generateJWTSecret()
		if err != nil {
			return err
		}
	}

	if c.DataDir == "" {
		return errors.New("'data directory path' cannot be empty")
	}

	if c.CSRFileName == "" {
		return errors.New("'csr filename' cannot be empty")
	}

	if c.KeepLostClients != 0 && (c.KeepLostClients.Nanoseconds() < MinKeepLostClients.Nanoseconds() ||
		c.KeepLostClients.Nanoseconds() > MaxKeepLostClients.Nanoseconds()) {
		return fmt.Errorf("expected 'Keep Lost Clients' can be in range [%v, %v], actual: %v", MinKeepLostClients, MaxKeepLostClients, c.KeepLostClients)
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
