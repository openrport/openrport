package chclient

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/cloudradar-monitoring/rport/share/files"

	"github.com/cloudradar-monitoring/rport/client/system"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/clientconfig"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
)

const DefaultMonitoringInterval = 60 * time.Second

var (
	allowDenyOrder = [2]string{"allow", "deny"}
	denyAllowOrder = [2]string{"deny", "allow"}
)

type ClientConfigHolder struct {
	*clientconfig.Config
}

func (c *ClientConfigHolder) ParseAndValidate(skipScriptsDirValidation bool) error {
	if err := c.parseHeaders(); err != nil {
		return err
	}
	if err := c.parseServerURL(); err != nil {
		return err
	}
	if err := c.parseFallbackServers(); err != nil {
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

	if c.Client.DataDir == "" {
		return errors.New("'data directory path' cannot be empty")
	}

	if err := c.parseRemoteCommands(); err != nil {
		return fmt.Errorf("remote commands: %v", err)
	}

	c.Client.AuthUser, c.Client.AuthPass = chshare.ParseAuth(c.Client.Auth)

	if err := c.parseRemoteScripts(skipScriptsDirValidation); err != nil {
		return err
	}

	if err := c.ParseAndValidateMonitoring(); err != nil {
		return err
	}

	if err := c.ParseAndValidateFilePushConfig(); err != nil {
		return err
	}

	if err := c.ParseAndValidateConnection(); err != nil {
		return err
	}

	return nil
}

func (c *ClientConfigHolder) ParseAndValidateMonitoring() error {
	if c.Monitoring.Interval < DefaultMonitoringInterval {
		c.Monitoring.Interval = DefaultMonitoringInterval
	}

	if len(c.Monitoring.NetLan) > 0 {
		lanCard, err := models.DecodeCard(c.Monitoring.NetLan)
		if err != nil {
			return err
		}
		c.Monitoring.LanCard = lanCard
	}

	if len(c.Monitoring.NetWan) > 0 {
		wanCard, err := models.DecodeCard(c.Monitoring.NetWan)
		if err != nil {
			return err
		}
		c.Monitoring.WanCard = wanCard
	}
	return nil
}

func (c *ClientConfigHolder) ParseAndValidateConnection() error {
	if !c.Connection.WatchdogIntegration {
		return nil
	}
	if c.Connection.KeepAlive == 0 {
		return errors.New("watchdog integration requires 'keep_alive > 0'")
	}
	if c.Connection.MaxRetryCount > 0 {
		return errors.New("watchdog integration requires 'max_retry_count = -1' (=disabled)")
	}
	return nil
}

func (c *ClientConfigHolder) ParseAndValidateFilePushConfig() error {
	for _, globPattern := range c.FileReceptionConfig.Protected {
		_, err := filepath.Match(globPattern, "/test")
		if err != nil {
			return fmt.Errorf("invalid glob pattern %s: %v", globPattern, err)
		}
	}

	return nil
}

func (c *ClientConfigHolder) parseHeaders() error {
	c.Connection.HTTPHeaders = http.Header{}
	for _, h := range c.Connection.HeadersRaw {
		name, val, err := parseHeader(h)
		if err != nil {
			return err
		}
		c.Connection.HTTPHeaders.Set(name, val)
	}
	if c.Connection.Hostname != "" {
		c.Connection.HTTPHeaders.Set("Host", c.Connection.Hostname)
	}
	if len(c.Connection.HTTPHeaders.Values("User-Agent")) == 0 {
		c.Connection.HTTPHeaders.Set("User-Agent", fmt.Sprintf("rport %s", chshare.BuildVersion))
	}
	return nil
}

func (c *ClientConfigHolder) parseServerURL() error {
	if c.Client.Server == "" {
		return errors.New("server address is required")
	}

	url, err := c.parseURL(c.Client.Server)
	if err != nil {
		return fmt.Errorf("invalid server address: %v", err)
	}

	c.Client.Server = url
	return nil
}

func (c *ClientConfigHolder) parseFallbackServers() error {
	for i := range c.Client.FallbackServers {
		url, err := c.parseURL(c.Client.FallbackServers[i])
		if err != nil {
			return fmt.Errorf("invalid fallback server address: %v", err)
		}
		c.Client.FallbackServers[i] = url
	}
	return nil
}

func (ClientConfigHolder) parseURL(urlStr string) (string, error) {
	//apply default scheme
	if !strings.Contains(urlStr, "://") {
		urlStr = "http://" + urlStr
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
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
	return u.String(), nil
}

func (c *ClientConfigHolder) parseProxyURL() error {
	if p := c.Client.Proxy; p != "" {
		proxyURL, err := url.Parse(p)
		if err != nil {
			return fmt.Errorf("invalid proxy URL: %v", err)
		}
		if proxyURL.Scheme == "https" {
			return fmt.Errorf("https proxies not (yet) supported")
		}
		c.Client.ProxyURL = proxyURL
	}
	return nil
}

func (c *ClientConfigHolder) parseRemotes() error {
	if err := c.parseTunnelsConfig(); err != nil {
		return fmt.Errorf("invalid tunnels config: %w", err)
	}

	for _, ta := range c.Client.TunnelAllowed {
		_, _, err := ParseTunnelAllowed(ta)
		if err != nil {
			return fmt.Errorf(`invalid "tunnel_allowed" config: %v`, err)
		}
	}

	for _, s := range c.Client.Remotes {
		r, err := models.DecodeRemote(s)
		if err != nil {
			return fmt.Errorf("failed to decode remote %q: %v", s, err)
		}

		allowed, err := TunnelIsAllowed(c.Client.TunnelAllowed, r.Remote())
		if err != nil {
			return fmt.Errorf("failed to check if remote %q is allowed: %v", s, err)
		}
		if !allowed {
			return fmt.Errorf(`remote %q is not allowed by "tunnel_allowed" config`, s)
		}

		r = c.applyTunnelsConfig(r)

		c.Client.Tunnels = append(c.Client.Tunnels, r)
	}
	return nil
}

func (c *ClientConfigHolder) parseTunnelsConfig() error {
	if c.Tunnels.HostHeader != "" && !c.Tunnels.ReverseProxy {
		return errors.New("host-header requires enabled reverse-proxy")
	}

	return nil
}

func (c *ClientConfigHolder) applyTunnelsConfig(r *models.Remote) *models.Remote {
	if c.Tunnels.Scheme != "" {
		r.Scheme = &c.Tunnels.Scheme
	}

	r.HTTPProxy = c.Tunnels.ReverseProxy
	r.HostHeader = c.Tunnels.HostHeader

	return r
}

func parseHeader(h string) (string, string, error) {
	index := strings.Index(h, ":")
	if index < 0 {
		return "", "", fmt.Errorf(`invalid header %q. Should be in the format "HeaderName: HeaderContent"`, h)
	}
	return h[0:index], strings.TrimSpace(h[index+1:]), nil
}

func (c *ClientConfigHolder) parseRemoteCommands() error {
	if c.RemoteCommands.SendBackLimit < 0 {
		return fmt.Errorf("send back limit can not be negative: %d", c.RemoteCommands.SendBackLimit)
	}

	allow, err := parseRegexpList(c.RemoteCommands.Allow)
	if err != nil {
		return fmt.Errorf("allow regexp: %v", err)
	}
	c.RemoteCommands.AllowRegexp = allow

	deny, err := parseRegexpList(c.RemoteCommands.Deny)
	if err != nil {
		return fmt.Errorf("deny regexp: %v", err)
	}
	c.RemoteCommands.DenyRegexp = deny

	if c.RemoteCommands.Order != allowDenyOrder && c.RemoteCommands.Order != denyAllowOrder {
		return fmt.Errorf("invalid order: %v", c.RemoteCommands.Order)
	}

	return nil
}

func (c *ClientConfigHolder) GetScriptsDir() string {
	return filepath.Join(c.Client.DataDir, "scripts")
}

func (c *ClientConfigHolder) GetUploadDir() string {
	return filepath.Join(c.Client.DataDir, files.DefaultUploadTempFolder)
}

func (c *ClientConfigHolder) GetProtectedUploadDirs() []string {
	return c.FileReceptionConfig.Protected
}

func (c *ClientConfigHolder) IsFileReceptionEnabled() bool {
	return c.FileReceptionConfig.Enabled
}

func (c *ClientConfigHolder) parseRemoteScripts(skipScriptsDirValidation bool) error {
	if skipScriptsDirValidation {
		return nil
	}

	if c.RemoteScripts.Enabled && !c.RemoteCommands.Enabled {
		return errors.New("remote scripts execution requires remote commands to be enabled")
	}

	if !c.RemoteScripts.Enabled {
		return nil
	}

	err := system.ValidateScriptDir(c.GetScriptsDir())

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

func PrepareDirs(c *ClientConfigHolder) error {
	logger := logger.NewLogger("client", c.Logging.LogOutput, c.Logging.LogLevel)

	if err := os.MkdirAll(c.Client.DataDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create dir %q: %s", c.Client.DataDir, err)
	}

	logger.Debugf("Data directory path: %q", c.Client.DataDir)

	scriptDir := c.GetScriptsDir()
	if _, err := os.Stat(scriptDir); os.IsNotExist(err) {
		err := os.Mkdir(scriptDir, system.DefaultDirMode)
		if err != nil {
			return err
		}
	}

	return nil
}
