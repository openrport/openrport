package chserver

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/jpillora/requestlog"

	"github.com/cloudradar-monitoring/rport/server/api/message"
	"github.com/cloudradar-monitoring/rport/server/ports"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/email"
)

type APIConfig struct {
	Address              string  `mapstructure:"address"`
	Auth                 string  `mapstructure:"auth"`
	AuthFile             string  `mapstructure:"auth_file"`
	AuthUserTable        string  `mapstructure:"auth_user_table"`
	AuthGroupTable       string  `mapstructure:"auth_group_table"`
	JWTSecret            string  `mapstructure:"jwt_secret"`
	DocRoot              string  `mapstructure:"doc_root"`
	CertFile             string  `mapstructure:"cert_file"`
	KeyFile              string  `mapstructure:"key_file"`
	AccessLogFile        string  `mapstructure:"access_log_file"`
	UserLoginWait        float32 `mapstructure:"user_login_wait"`
	MaxFailedLogin       int     `mapstructure:"max_failed_login"`
	BanTime              int     `mapstructure:"ban_time"`
	TwoFATokenDelivery   string  `mapstructure:"two_fa_token_delivery"`
	TwoFATokenTTLSeconds int     `mapstructure:"two_fa_token_ttl_seconds"`
}

func (c *APIConfig) IsTwoFAOn() bool {
	return c.TwoFATokenDelivery != ""
}

const (
	MinKeepLostClients = time.Second
	MaxKeepLostClients = 7 * 24 * time.Hour

	socketPrefix = "socket:"
)

type LogConfig struct {
	LogOutput chshare.LogOutput `mapstructure:"log_file"`
	LogLevel  chshare.LogLevel  `mapstructure:"log_level"`
}

type ServerConfig struct {
	ListenAddress              string        `mapstructure:"address"`
	URL                        string        `mapstructure:"url"`
	KeySeed                    string        `mapstructure:"key_seed"`
	Auth                       string        `mapstructure:"auth"`
	AuthFile                   string        `mapstructure:"auth_file"`
	AuthTable                  string        `mapstructure:"auth_table"`
	Proxy                      string        `mapstructure:"proxy"`
	ExcludedPortsRaw           []string      `mapstructure:"excluded_ports"`
	DataDir                    string        `mapstructure:"data_dir"`
	KeepLostClients            time.Duration `mapstructure:"keep_lost_clients"`
	SaveClients                time.Duration `mapstructure:"save_clients_interval"`
	CleanupClients             time.Duration `mapstructure:"cleanup_clients_interval"`
	MaxRequestBytes            int64         `mapstructure:"max_request_bytes"`
	CheckPortTimeout           time.Duration `mapstructure:"check_port_timeout"`
	RunRemoteCmdTimeoutSec     int           `mapstructure:"run_remote_cmd_timeout_sec"`
	AuthWrite                  bool          `mapstructure:"auth_write"`
	AuthMultiuseCreds          bool          `mapstructure:"auth_multiuse_creds"`
	EquateClientauthidClientid bool          `mapstructure:"equate_clientauthid_clientid"`
	AllowRoot                  bool          `mapstructure:"allow_root"`
	ClientLoginWait            float32       `mapstructure:"client_login_wait"`
	MaxFailedLogin             int           `mapstructure:"max_failed_login"`
	BanTime                    int           `mapstructure:"ban_time"`
	EnableWsTestEndpoints      bool          `mapstructure:"enable_ws_test_endpoints"`

	excludedPorts mapset.Set
	authID        string
	authPassword  string
}

type DatabaseConfig struct {
	Type     string `mapstructure:"db_type"`
	Host     string `mapstructure:"db_host"`
	User     string `mapstructure:"db_user"`
	Password string `mapstructure:"db_password"`
	Name     string `mapstructure:"db_name"`

	driver string
	dsn    string
}

type PushoverConfig struct {
	APIToken string `mapstructure:"api_token"`
	UserKey  string `mapstructure:"user_key"`
}

func (c *PushoverConfig) Validate() error {
	if c.APIToken == "" {
		return errors.New("pushover.api_token is required")
	}

	p := message.NewPushoverService(c.APIToken)
	err := p.ValidateReceiver(c.UserKey)
	if err != nil {
		return fmt.Errorf("invalid pushover.api_token and pushover.user_key: %v", err)

	}

	return nil
}

type SMTPConfig struct {
	Server       string `mapstructure:"server"`
	AuthUsername string `mapstructure:"auth_username"`
	AuthPassword string `mapstructure:"auth_password"`
	SenderEmail  string `mapstructure:"sender_email"`
	Secure       bool   `mapstructure:"secure"`
}

func (c *SMTPConfig) Validate() error {
	if c.Server == "" {
		return errors.New("smtp.server is required")
	}
	host, _, err := net.SplitHostPort(c.Server)
	if err != nil {
		return fmt.Errorf("invalid smtp.server, expected to be server and port separated by a colon. e.g. 'smtp.gmail.com:587'; error: %v", err)
	}

	if err := email.Validate(c.SenderEmail); err != nil {
		return fmt.Errorf("invalid smtp.sender_email: %v", err)
	}

	var client *smtp.Client
	if c.Secure {
		tlsConfig := &tls.Config{
			ServerName: host,
			MinVersion: tls.VersionTLS12,
		}
		conn, err := tls.Dial("tcp", c.Server, tlsConfig)
		if err != nil {
			return fmt.Errorf("could not connect to smtp.server using TLS: %v", err)
		}

		client, err = smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("could not init smtp client to smtp.server: %v", err)
		}
		defer client.Close()
	} else {
		client, err = smtp.Dial(c.Server)
		if err != nil {
			return fmt.Errorf("could not connect to smtp.server: %v", err)
		}
		defer client.Close()

		// use TLS if available
		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsConfig := &tls.Config{
				ServerName: host,
				MinVersion: tls.VersionTLS12,
			}
			if err = client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("failed to start tls: %v", err)
			}
		}
	}

	if c.AuthUsername != "" || c.AuthPassword != "" {
		err = client.Auth(smtp.PlainAuth("", c.AuthUsername, c.AuthPassword, host))
		if err != nil {
			return fmt.Errorf("failed to connect to smtp server using provided auth_username and auth_password: %v", err)
		}
	}

	return nil
}

type VaultConfig struct {
	DBName string `mapstructure:"db_name"`
}

func (vc VaultConfig) GetDatabasePath() string {
	return vc.DBName
}

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Logging  LogConfig      `mapstructure:"logging"`
	API      APIConfig      `mapstructure:"api"`
	Database DatabaseConfig `mapstructure:"database"`
	Pushover PushoverConfig `mapstructure:"pushover"`
	SMTP     SMTPConfig     `mapstructure:"smtp"`
	Vault    VaultConfig    `mapstructure:"vault"`
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

	excludedPorts, err := ports.TryParsePortRanges(c.Server.ExcludedPortsRaw)
	if err != nil {
		return fmt.Errorf("can't parse excluded ports: %s", err)
	}
	c.Server.excludedPorts = excludedPorts

	if c.Server.DataDir == "" {
		return errors.New("'data directory path' cannot be empty")
	}

	if c.Server.KeepLostClients != 0 && (c.Server.KeepLostClients.Nanoseconds() < MinKeepLostClients.Nanoseconds() ||
		c.Server.KeepLostClients.Nanoseconds() > MaxKeepLostClients.Nanoseconds()) {
		return fmt.Errorf("expected 'Keep Lost Clients' can be in range [%v, %v], actual: %v", MinKeepLostClients, MaxKeepLostClients, c.Server.KeepLostClients)
	}

	if err := c.parseAndValidateClientAuth(); err != nil {
		return err
	}

	if err := c.parseAndValidateAPI(); err != nil {
		return fmt.Errorf("API: %v", err)
	}

	if err := c.Database.ParseAndValidate(); err != nil {
		return err
	}

	return nil
}

func (c *Config) parseAndValidateClientAuth() error {
	if c.Server.Auth == "" && c.Server.AuthFile == "" && c.Server.AuthTable == "" {
		return errors.New("client authentication must be enabled: set either 'auth', 'auth_file' or 'auth_table'")
	}

	if c.Server.AuthFile != "" && c.Server.Auth != "" {
		return errors.New("'auth_file' and 'auth' are both set: expected only one of them")
	}
	if c.Server.AuthFile != "" && c.Server.AuthTable != "" {
		return errors.New("'auth_file' and 'auth_table' are both set: expected only one of them")
	}
	if c.Server.Auth != "" && c.Server.AuthTable != "" {
		return errors.New("'auth' and 'auth_table' are both set: expected only one of them")
	}

	if c.Server.AuthTable != "" && c.Database.Type == "" {
		return errors.New("'db_type' must be set when 'auth_table' is set")
	}

	if c.Server.Auth != "" {
		c.Server.authID, c.Server.authPassword = chshare.ParseAuth(c.Server.Auth)
		if c.Server.authID == "" || c.Server.authPassword == "" {
			return fmt.Errorf("invalid client auth credentials, expected '<client-id>:<password>', got %q", c.Server.Auth)
		}
	}

	return nil
}

func (c *Config) parseAndValidateAPI() error {
	if c.API.Address != "" {
		// API enabled
		err := c.parseAndValidateAPIAuth()
		if err != nil {
			return err
		}
		err = c.parseAndValidateAPIHTTPSOptions()
		if err != nil {
			return err
		}
		if c.API.JWTSecret == "" {
			c.API.JWTSecret, err = generateJWTSecret()
			if err != nil {
				return err
			}
		}
		err = c.parseAndValidate2FA()
		if err != nil {
			return err
		}
	} else {
		// API disabled
		if c.API.DocRoot != "" {
			return errors.New("to use document root you need to specify API address")
		}

	}
	return nil
}

func (c *Config) parseAndValidate2FA() error {
	if c.API.TwoFATokenDelivery == "" {
		return nil
	}

	if c.API.Auth != "" {
		return errors.New("2FA is not available if you use a single static user-password pair")
	}

	// TODO: to do better handling, maybe with using enums
	switch c.API.TwoFATokenDelivery {
	case "pushover":
		return c.Pushover.Validate()
	case "smtp":
		return c.SMTP.Validate()
	}

	return fmt.Errorf("unknown 2fa token delivery method: %s", c.API.TwoFATokenDelivery)
}

func (c *Config) parseAndValidateAPIAuth() error {
	if c.API.AuthFile == "" && c.API.Auth == "" && c.API.AuthUserTable == "" {
		return errors.New("authentication must be enabled: set either 'auth', 'auth_file' or 'auth_user_table'")
	}

	if c.API.AuthFile != "" && c.API.Auth != "" {
		return errors.New("'auth_file' and 'auth' are both set: expected only one of them")
	}

	if c.API.AuthUserTable != "" && c.API.Auth != "" {
		return errors.New("'auth_user_table' and 'auth' are both set: expected only one of them")
	}

	if c.API.AuthUserTable != "" && c.API.AuthFile != "" {
		return errors.New("'auth_user_table' and 'auth_file' are both set: expected only one of them")
	}

	if c.API.AuthUserTable != "" && c.API.AuthGroupTable == "" {
		return errors.New("when 'auth_user_table' is set, 'auth_group_table' must be set as well")
	}

	if c.API.AuthUserTable != "" && c.Database.Type == "" {
		return errors.New("'db_type' must be set when 'auth_user_table' is set")
	}

	return nil
}

func (c *Config) parseAndValidateAPIHTTPSOptions() error {
	if c.API.CertFile == "" && c.API.KeyFile == "" {
		return nil
	}
	if c.API.CertFile != "" && c.API.KeyFile == "" {
		return errors.New("when 'cert_file' is set, 'key_file' must be set as well")
	}
	if c.API.CertFile == "" && c.API.KeyFile != "" {
		return errors.New("when 'key_file' is set, 'cert_file' must be set as well")
	}
	_, err := tls.LoadX509KeyPair(c.API.CertFile, c.API.KeyFile)
	if err != nil {
		return fmt.Errorf("invalid 'cert_file', 'key_file': %v", err)
	}
	return nil
}

func (d *DatabaseConfig) ParseAndValidate() error {
	switch d.Type {
	case "":
		return nil
	case "mysql":
		d.driver = "mysql"
		d.dsn = ""
		if d.User != "" {
			d.dsn += d.User
			if d.Password != "" {
				d.dsn += ":"
				d.dsn += d.Password
			}
			d.dsn += "@"
		}
		if d.Host != "" {
			if strings.HasPrefix(d.Host, socketPrefix) {
				d.dsn += fmt.Sprintf("unix(%s)", strings.TrimPrefix(d.Host, socketPrefix))
			} else {
				d.dsn += fmt.Sprintf("tcp(%s)", d.Host)
			}
		}
		d.dsn += "/"
		d.dsn += d.Name
	case "sqlite":
		d.driver = "sqlite3"
		d.dsn = d.Name
	default:
		return fmt.Errorf("invalid 'db_type', expected 'mysql' or 'sqlite', got %q", d.Type)
	}

	return nil
}

func (d *DatabaseConfig) dsnForLogs() string {
	if d.Password != "" {
		// hide the password
		return strings.Replace(d.dsn, ":"+d.Password, ":***", 1)
	}
	return d.dsn
}

func generateJWTSecret() (string, error) {
	data := make([]byte, 10)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("can't generate API JWT secret: %s", err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}
