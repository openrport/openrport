package chclient

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type ConnectionConfig struct {
	KeepAlive        time.Duration `mapstructure:"keep_alive"`
	MaxRetryCount    int           `mapstructure:"max_retry_count"`
	MaxRetryInterval time.Duration `mapstructure:"max_retry_interval"`
	HeadersRaw       []string      `mapstructure:"headers"`
	Hostname         string        `mapstructure:"hostname"`

	headers http.Header
}

type LogConfig struct {
	LogOutput chshare.LogOutput `mapstructure:"log_file"`
	LogLevel  chshare.LogLevel  `mapstructure:"log_level"`
}

type ClientConfig struct {
	Server      string   `mapstructure:"server"`
	Fingerprint string   `mapstructure:"fingerprint"`
	Auth        string   `mapstructure:"auth"`
	Proxy       string   `mapstructure:"proxy"`
	ID          string   `mapstructure:"id"`
	Name        string   `mapstructure:"name"`
	Tags        []string `mapstructure:"tags"`
	Remotes     []string `mapstructure:"remotes"`
}

func (c *ConnectionConfig) Headers() http.Header {
	return c.headers
}

type Config struct {
	Client     ClientConfig     `mapstructure:"client"`
	Connection ConnectionConfig `mapstructure:"connection"`
	Logging    LogConfig        `mapstructure:"logging"`
}

func (c *Config) ParseAndValidate() error {
	c.Connection.headers = http.Header{}
	for _, h := range c.Connection.HeadersRaw {
		name, val, err := parseHeader(h)
		if err != nil {
			return err
		}
		c.Connection.headers.Set(name, val)
	}
	if c.Connection.Hostname != "" {
		c.Connection.headers.Set("Host", c.Connection.Hostname)
	}
	if len(c.Connection.headers.Values("User-Agent")) == 0 {
		c.Connection.headers.Set("User-Agent", fmt.Sprintf("rport %s", chshare.BuildVersion))
	}
	return nil
}

func parseHeader(h string) (string, string, error) {
	index := strings.Index(h, ":")
	if index < 0 {
		return "", "", fmt.Errorf(`invalid header (%s). Should be in the format "HeaderName: HeaderContent"`, h)
	}
	return h[0:index], strings.TrimSpace(h[index+1:]), nil
}
