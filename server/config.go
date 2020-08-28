package chserver

import "C"
import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/jpillora/requestlog"

	"github.com/cloudradar-monitoring/rport/server/ports"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

type APIConfig struct {
	Address   string `mapstructure:"address"`
	Auth      string `mapstructure:"auth"`
	JWTSecret string `mapstructure:"jwt_secret"`
	DocRoot   string `mapstructure:"doc_root"`
}

type Config struct {
	LogOutput chshare.LogOutput `mapstructure:"log_file"`
	LogLevel  chshare.LogLevel  `mapstructure:"log_level"`

	ListenAddress    string    `mapstructure:"address"`
	URL              string    `mapstructure:"url"`
	KeySeed          string    `mapstructure:"key_seed"`
	AuthFile         string    `mapstructure:"auth_file"`
	Auth             string    `mapstructure:"auth"`
	Proxy            string    `mapstructure:"proxy"`
	API              APIConfig `mapstructure:"api"`
	ExcludedPortsRaw []string  `mapstructure:"excluded_ports"`

	excludedPorts mapset.Set
}

func (c *Config) InitRequestLogOptions() *requestlog.Options {
	o := requestlog.DefaultOptions
	o.Writer = c.LogOutput.File
	o.Filter = func(r *http.Request, code int, duration time.Duration, size int64) bool {
		return c.LogLevel == chshare.LogLevelInfo || c.LogLevel == chshare.LogLevelDebug
	}
	return &o
}

func (c *Config) GetExcludedPorts() mapset.Set {
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

	return nil
}

func generateJWTSecret() (string, error) {
	data := make([]byte, 10)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("can't generate API JWT secret: %s", err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}
