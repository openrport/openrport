package chclient

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type ConnectionOptions struct {
	KeepAlive        time.Duration `mapstructure:"keep_alive"`
	MaxRetryCount    int           `mapstructure:"max_retry_count"`
	MaxRetryInterval time.Duration `mapstructure:"max_retry_interval"`
	HeadersRaw       []string      `mapstructure:"headers"`
	Hostname         string        `mapstructure:"hostname"`

	headers http.Header
}

type Config struct {
	LogOutput chshare.LogOutput `mapstructure:"log_file"`
	LogLevel  chshare.LogLevel  `mapstructure:"log_level"`

	Fingerprint string            `mapstructure:"fingerprint"`
	Auth        string            `mapstructure:"auth"`
	Connection  ConnectionOptions `mapstructure:"connection"`
	Server      string            `mapstructure:"server"`
	Proxy       string            `mapstructure:"proxy"`
	ID          string            `mapstructure:"id"`
	Name        string            `mapstructure:"name"`
	Tags        []string          `mapstructure:"tags"`
	Remotes     []string          `mapstructure:"remotes"`
}

func (c *ConnectionOptions) GetHeaders() http.Header {
	return c.headers
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
	return nil
}

func parseHeader(h string) (string, string, error) {
	index := strings.Index(h, ":")
	if index < 0 {
		return "", "", fmt.Errorf(`invalid header (%s). Should be in the format "HeaderName: HeaderContent"`, h)
	}
	return h[0:index], strings.TrimSpace(h[index+1:]), nil
}
