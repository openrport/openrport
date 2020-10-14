package chclient

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
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
	AllowRoot   bool     `mapstructure:"allow_root"`

	proxyURL *url.URL
	remotes  []*chshare.Remote
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
	if err := c.parseHeaders(); err != nil {
		return err
	}
	if err := c.parseServerURL(); err != nil {
		return err
	}
	if err := c.parseProxyURL(); err != nil {
		return err
	}
	if err := c.parseRemotes(); err != nil {
		return err
	}
	if c.Connection.MaxRetryInterval < time.Second {
		c.Connection.MaxRetryInterval = 5 * time.Minute
	}
	return nil
}

func (c *Config) parseHeaders() error {
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

func (c *Config) parseServerURL() error {
	if c.Client.Server == "" {
		return fmt.Errorf("Server address is required. See --help")
	}

	//apply default scheme
	if !strings.Contains(c.Client.Server, "://") {
		c.Client.Server = "http://" + c.Client.Server
	}

	u, err := url.Parse(c.Client.Server)
	if err != nil {
		return fmt.Errorf("Invalid server address: %v", err)
	}
	//apply default port
	if !regexp.MustCompile(`:\d+$`).MatchString(u.Host) {
		if u.Scheme == "https" || u.Scheme == "wss" {
			u.Host = u.Host + ":443"
		} else {
			u.Host = u.Host + ":80"
		}
	}
	//swap to websockets scheme
	u.Scheme = strings.Replace(u.Scheme, "http", "ws", 1)
	c.Client.Server = u.String()
	return nil
}

func (c *Config) parseProxyURL() error {
	if p := c.Client.Proxy; p != "" {
		proxyURL, err := url.Parse(p)
		if err != nil {
			return fmt.Errorf("Invalid proxy URL (%s)", err)
		}
		c.Client.proxyURL = proxyURL
	}
	return nil
}

func (c *Config) parseRemotes() error {
	for _, s := range c.Client.Remotes {
		r, err := chshare.DecodeRemote(s)
		if err != nil {
			return fmt.Errorf("Failed to decode remote '%s': %s", s, err)
		}
		c.Client.remotes = append(c.Client.remotes, r)
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
