package chclient

import (
	"errors"
	"fmt"
	"log"
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
	Server          string        `mapstructure:"server"`
	Fingerprint     string        `mapstructure:"fingerprint"`
	Auth            string        `mapstructure:"auth"`
	Proxy           string        `mapstructure:"proxy"`
	ID              string        `mapstructure:"id"`
	Name            string        `mapstructure:"name"`
	Tags            []string      `mapstructure:"tags"`
	Remotes         []string      `mapstructure:"remotes"`
	AllowRoot       bool          `mapstructure:"allow_root"`
	UpdatesInterval time.Duration `mapstructure:"updates_interval"`

	proxyURL *url.URL
	remotes  []*chshare.Remote
	authUser string
	authPass string
}

func (c *ConnectionConfig) Headers() http.Header {
	return c.headers
}

var (
	allowDenyOrder = [2]string{"allow", "deny"}
	denyAllowOrder = [2]string{"deny", "allow"}
)

type CommandsConfig struct {
	Enabled       bool      `mapstructure:"enabled"`
	SendBackLimit int       `mapstructure:"send_back_limit"`
	Allow         []string  `mapstructure:"allow"`
	Deny          []string  `mapstructure:"deny"`
	Order         [2]string `mapstructure:"order"`

	allowRegexp []*regexp.Regexp
	denyRegexp  []*regexp.Regexp
}

type ScriptsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Dir     string `mapstructure:"script_dir"`
}

type Config struct {
	Client         ClientConfig     `mapstructure:"client"`
	Connection     ConnectionConfig `mapstructure:"connection"`
	Logging        LogConfig        `mapstructure:"logging"`
	RemoteCommands CommandsConfig   `mapstructure:"remote-commands"`
	RemoteScripts  ScriptsConfig    `mapstructure:"remote-scripts"`
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

	if err := c.parseRemoteCommands(); err != nil {
		return fmt.Errorf("remote commands: %v", err)
	}

	c.Client.authUser, c.Client.authPass = chshare.ParseAuth(c.Client.Auth)

	if err := c.parseRemoteScripts(); err != nil {
		return err
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
		return errors.New("server address is required")
	}

	//apply default scheme
	if !strings.Contains(c.Client.Server, "://") {
		c.Client.Server = "http://" + c.Client.Server
	}

	u, err := url.Parse(c.Client.Server)
	if err != nil {
		return fmt.Errorf("invalid server address: %v", err)
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
			return fmt.Errorf("invalid proxy URL: %v", err)
		}
		c.Client.proxyURL = proxyURL
	}
	return nil
}

func (c *Config) parseRemotes() error {
	for _, s := range c.Client.Remotes {
		r, err := chshare.DecodeRemote(s)
		if err != nil {
			return fmt.Errorf("failed to decode remote %q: %v", s, err)
		}
		c.Client.remotes = append(c.Client.remotes, r)
	}
	return nil
}

func parseHeader(h string) (string, string, error) {
	index := strings.Index(h, ":")
	if index < 0 {
		return "", "", fmt.Errorf(`invalid header %q. Should be in the format "HeaderName: HeaderContent"`, h)
	}
	return h[0:index], strings.TrimSpace(h[index+1:]), nil
}

func (c *Config) parseRemoteCommands() error {
	if c.RemoteCommands.SendBackLimit < 0 {
		return fmt.Errorf("send back limit can not be negative: %d", c.RemoteCommands.SendBackLimit)
	}

	allow, err := parseRegexpList(c.RemoteCommands.Allow)
	if err != nil {
		return fmt.Errorf("allow regexp: %v", err)
	}
	c.RemoteCommands.allowRegexp = allow

	deny, err := parseRegexpList(c.RemoteCommands.Deny)
	if err != nil {
		return fmt.Errorf("deny regexp: %v", err)
	}
	c.RemoteCommands.denyRegexp = deny

	if c.RemoteCommands.Order != allowDenyOrder && c.RemoteCommands.Order != denyAllowOrder {
		return fmt.Errorf("invalid order: %v", c.RemoteCommands.Order)
	}

	return nil
}

func (c *Config) parseRemoteScripts() error {
	if c.RemoteScripts.Enabled && !c.RemoteCommands.Enabled {
		return errors.New("remote scripts execution requires remote commands to be enabled")
	}

	err := ValidateScriptDir(c.RemoteScripts.Dir)
	// we allow to start a client if the script dir is not good because clients might never run scripts
	if err != nil {
		log.Printf("ERROR: %v\n", err)
	}

	return nil
}

func parseRegexpList(regexpList []string) ([]*regexp.Regexp, error) {
	res := make([]*regexp.Regexp, 0, len(regexpList))
	for _, cur := range regexpList {
		r, err := regexp.Compile(cur)
		if err != nil {
			return nil, fmt.Errorf("invalid regular expression %q: %v", cur, err)
		}
		res = append(res, r)
	}
	return res, nil
}
