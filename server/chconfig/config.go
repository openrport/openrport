package chconfig

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/realvnc-labs/rport/db/sqlite"
	rportplus "github.com/realvnc-labs/rport/plus"
	"github.com/realvnc-labs/rport/server/caddy"

	"github.com/realvnc-labs/rport/share/files"

	mapset "github.com/deckarep/golang-set"
	"github.com/jpillora/requestlog"
	"github.com/pkg/errors"

	"github.com/realvnc-labs/rport/server/api/message"
	auditlog "github.com/realvnc-labs/rport/server/auditlog/config"
	"github.com/realvnc-labs/rport/server/bearer"
	"github.com/realvnc-labs/rport/server/clients/clienttunnel"
	"github.com/realvnc-labs/rport/server/ports"
	chshare "github.com/realvnc-labs/rport/share"
	"github.com/realvnc-labs/rport/share/email"
	"github.com/realvnc-labs/rport/share/logger"
)

type APIConfig struct {
	Address                string   `mapstructure:"address"`
	BaseURL                string   `mapstructure:"base_url"`
	EnableAcme             bool     `mapstructure:"enable_acme"`
	Auth                   string   `mapstructure:"auth"`
	AuthFile               string   `mapstructure:"auth_file"`
	AuthUserTable          string   `mapstructure:"auth_user_table"`
	AuthGroupTable         string   `mapstructure:"auth_group_table"`
	AuthGroupDetailsTable  string   `mapstructure:"auth_group_details_table"`
	AuthHeader             string   `mapstructure:"auth_header"`
	UserHeader             string   `mapstructure:"user_header"`
	CreateMissingUsers     bool     `mapstructure:"create_missing_users"`
	DefaultUserGroup       string   `mapstructure:"default_user_group"`
	JWTSecret              string   `mapstructure:"jwt_secret"`
	DocRoot                string   `mapstructure:"doc_root"`
	CertFile               string   `mapstructure:"cert_file"`
	KeyFile                string   `mapstructure:"key_file"`
	AccessLogFile          string   `mapstructure:"access_log_file"`
	UserLoginWait          float32  `mapstructure:"user_login_wait"`
	MaxFailedLogin         int      `mapstructure:"max_failed_login"`
	BanTime                int      `mapstructure:"ban_time"`
	MaxTokenLifeTimeHours  int      `mapstructure:"max_token_lifetime"`
	PasswordMinLength      int      `mapstructure:"password_min_length"`
	PasswordZxcvbnMinscore int      `mapstructure:"password_zxcvbn_minscore"`
	TLSMin                 string   `mapstructure:"tls_min"`
	EnableWsTestEndpoints  bool     `mapstructure:"enable_ws_test_endpoints"`
	MaxRequestBytes        int64    `mapstructure:"max_request_bytes"`
	MaxFilePushSize        int64    `mapstructure:"max_filepush_size"`
	CORS                   []string `mapstructure:"cors"`

	TwoFATokenDelivery       string                 `mapstructure:"two_fa_token_delivery"`
	TwoFATokenTTLSeconds     int                    `mapstructure:"two_fa_token_ttl_seconds"`
	TwoFASendTimeout         time.Duration          `mapstructure:"two_fa_send_timeout"`
	TwoFASendToType          message.ValidationType `mapstructure:"two_fa_send_to_type"`
	TwoFASendToRegex         string                 `mapstructure:"two_fa_send_to_regex"`
	TwoFASendToRegexCompiled *regexp.Regexp

	AuditLog                auditlog.Config `mapstructure:",squash"`
	TotPEnabled             bool            `mapstructure:"totp_enabled"`
	TotPLoginSessionTimeout time.Duration   `mapstructure:"totp_login_session_ttl"`
	TotPAccountName         string          `mapstructure:"totp_account_name"`
}

func (c *APIConfig) IsTwoFAOn() bool {
	return c.TwoFATokenDelivery != ""
}

func (c *APIConfig) parseAndValidate2FASendToType() error {
	if c.TwoFASendToType != message.ValidationNone &&
		c.TwoFASendToType != message.ValidationEmail &&
		c.TwoFASendToType != message.ValidationRegex {
		return fmt.Errorf("invalid api.two_fa_send_to_type: %q", c.TwoFASendToType)
	}

	if c.TwoFASendToType == message.ValidationRegex {
		regex, err := regexp.Compile(c.TwoFASendToRegex)
		if err != nil {
			return fmt.Errorf("invalid api.two_fa_send_to_regex: %v", err)
		}
		c.TwoFASendToRegexCompiled = regex
	}

	return nil
}

const (
	MinKeepDisconnectedClients = time.Second
	MaxKeepDisconnectedClients = 7 * 24 * time.Hour
	DefaultVaultDBName         = "vault.sqlite.db"

	socketPrefix = "socket:"
)

type LogConfig struct {
	LogOutput logger.LogOutput `mapstructure:"log_file"`
	LogLevel  logger.LogLevel  `mapstructure:"log_level"`
}

type ServerConfig struct {
	ListenAddress                        string                                 `mapstructure:"address"`
	URL                                  []string                               `mapstructure:"url"`
	PairingURL                           string                                 `mapstructure:"pairing_url"`
	KeySeed                              string                                 `mapstructure:"key_seed"`
	Auth                                 string                                 `mapstructure:"auth"`
	AuthFile                             string                                 `mapstructure:"auth_file"`
	AuthTable                            string                                 `mapstructure:"auth_table"`
	Proxy                                string                                 `mapstructure:"proxy"`
	UsedPortsRaw                         []string                               `mapstructure:"used_ports"`
	ExcludedPortsRaw                     []string                               `mapstructure:"excluded_ports"`
	DataDir                              string                                 `mapstructure:"data_dir"`
	SqliteWAL                            bool                                   `mapstructure:"sqlite_wal"`
	MaxConcurrentSSHConnectionHandshakes int                                    `mapstructure:"max_concurrent_ssh_handshakes"`
	PurgeDisconnectedClients             bool                                   `mapstructure:"purge_disconnected_clients"`
	CleanupLostClients                   bool                                   `mapstructure:"cleanup_lost_clients" replaced_by:"PurgeDisconnectedClients"`
	KeepLostClients                      time.Duration                          `mapstructure:"keep_lost_clients" replaced_by:"KeepDisconnectedClients"`
	KeepDisconnectedClients              time.Duration                          `mapstructure:"keep_disconnected_clients"`
	CleanupClientsInterval               time.Duration                          `mapstructure:"cleanup_clients_interval" replaced_by:"PurgeDisconnectedClientsInterval"`
	PurgeDisconnectedClientsInterval     time.Duration                          `mapstructure:"purge_disconnected_clients_interval"`
	CheckClientsConnectionInterval       time.Duration                          `mapstructure:"check_clients_connection_interval"`
	CheckClientsConnectionTimeout        time.Duration                          `mapstructure:"check_clients_connection_timeout"`
	MaxRequestBytesClient                int64                                  `mapstructure:"max_request_bytes_client"`
	CheckPortTimeout                     time.Duration                          `mapstructure:"check_port_timeout"`
	RunRemoteCmdTimeoutSec               int                                    `mapstructure:"run_remote_cmd_timeout_sec"`
	AuthWrite                            bool                                   `mapstructure:"auth_write"`
	AuthMultiuseCreds                    bool                                   `mapstructure:"auth_multiuse_creds"`
	EquateClientauthidClientid           bool                                   `mapstructure:"equate_clientauthid_clientid"`
	AllowRoot                            bool                                   `mapstructure:"allow_root"`
	ClientLoginWait                      float32                                `mapstructure:"client_login_wait"`
	MaxFailedLogin                       int                                    `mapstructure:"max_failed_login"`
	BanTime                              int                                    `mapstructure:"ban_time"`
	InternalTunnelProxyConfig            clienttunnel.InternalTunnelProxyConfig `mapstructure:",squash"`
	JobsMaxResults                       int                                    `mapstructure:"jobs_max_results"`
	AcmeHTTPPort                         int                                    `mapstructure:"acme_http_port"`

	// DEPRECATED, only here for backwards compatibility
	MaxRequestBytes       int64 `mapstructure:"max_request_bytes"`
	MaxFilePushSize       int64 `mapstructure:"max_filepush_size"`
	EnableWsTestEndpoints bool  `mapstructure:"enable_ws_test_endpoints"`

	allowedPorts mapset.Set
	AuthID       string
	AuthPassword string
}

type DatabaseConfig struct {
	Type     string `mapstructure:"db_type"`
	Host     string `mapstructure:"db_host"`
	User     string `mapstructure:"db_user"`
	Password string `mapstructure:"db_password"`
	Name     string `mapstructure:"db_name"`

	Driver string
	Dsn    string
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
	err := p.ValidateReceiver(context.Background(), c.UserKey)
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

type MonitoringConfig struct {
	DataStorageDuration string `mapstructure:"data_storage_duration"`
	DataStorageDays     int64  `mapstructure:"data_storage_days"`
	Enabled             bool   `mapstructure:"enabled"`

	// cached version of DataStorageDuration as real time.Duration
	duration time.Duration `mapstructure:"-"`
}

func (mc *MonitoringConfig) GetDataStorageDuration() (duration time.Duration) {
	return mc.duration
}

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Caddy      caddy.Config     `mapstructure:"caddy-integration"`
	Logging    LogConfig        `mapstructure:"logging"`
	API        APIConfig        `mapstructure:"api"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Pushover   PushoverConfig   `mapstructure:"pushover"`
	SMTP       SMTPConfig       `mapstructure:"smtp"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`

	PlusConfig rportplus.PlusConfig `mapstructure:",squash"`
}

var (
	CheckClientsConnectionIntervalMinimum = time.Minute * 2
)

func (c *Config) GetVaultDBPath() string {
	return path.Join(c.Server.DataDir, DefaultVaultDBName)
}

func (c *Config) GetUploadDir() string {
	return filepath.Join(c.Server.DataDir, files.DefaultUploadTempFolder)
}

func (s *ServerConfig) GetSQLiteDataSourceOptions() sqlite.DataSourceOptions {
	return sqlite.DataSourceOptions{WALEnabled: s.SqliteWAL}
}

func (c *Config) InitRequestLogOptions() *requestlog.Options {
	o := requestlog.DefaultOptions
	o.Writer = c.Logging.LogOutput.File
	o.Filter = func(r *http.Request, code int, duration time.Duration, size int64) bool {
		return c.Logging.LogLevel == logger.LogLevelInfo || c.Logging.LogLevel == logger.LogLevelDebug
	}
	return &o
}

func (c *Config) AllowedPorts() mapset.Set {
	return c.Server.allowedPorts
}

func (c *Config) ParseAndValidate(mLog *logger.MemLogger) error {
	rpl, err := ConfigReplaceDeprecated(&c.Server)
	for old, new := range rpl {
		mLog.Infof("server setting '%s' is deprecated and will be removed soon. Use '%s' instead.", old, new)
	}
	if err != nil {
		return err
	}

	if c.Server.MaxRequestBytes > 0 {
		c.API.MaxRequestBytes = c.Server.MaxRequestBytes
		mLog.Info("server setting 'max_request_bytes' is deprecated and will be removed soon. Use the setting in api section instead.")
	}
	if c.Server.MaxFilePushSize > 0 {
		c.API.MaxFilePushSize = c.Server.MaxFilePushSize
		mLog.Info("server setting 'max_filepush_size' is deprecated and will be removed soon. Use the setting in api section instead.")
	}
	if c.Server.EnableWsTestEndpoints {
		c.API.EnableWsTestEndpoints = c.Server.EnableWsTestEndpoints
		mLog.Info("server setting 'enable_ws_test_endpoints' is deprecated and will be removed soon. Use the setting in api section instead.")
	}

	if err := c.Server.parseAndValidateURLs(); err != nil {
		return err
	}

	if err := c.Server.parseAndValidatePorts(); err != nil {
		return err
	}

	if err := c.Server.InternalTunnelProxyConfig.ParseAndValidate(); err != nil {
		return err
	}
	c.Server.InternalTunnelProxyConfig.CORS = parseAndValidateCORS(mLog, c.Server.InternalTunnelProxyConfig.CORS)

	filesAPI := files.NewFileSystem()
	serverLogLevel := c.Logging.LogLevel.String()

	if err := c.Caddy.ParseAndValidate(c.Server.DataDir, serverLogLevel, filesAPI); err != nil {
		// caddy integration is not critical, so continue running with caddy integration disabled
		mLog.Errorf("caddy integration not enabled due to error: %v", err)
	}

	if c.Server.DataDir == "" {
		return errors.New("'data directory path' cannot be empty")
	}

	if c.Server.PurgeDisconnectedClients && c.Server.KeepDisconnectedClients != 0 && (c.Server.KeepDisconnectedClients.Nanoseconds() < MinKeepDisconnectedClients.Nanoseconds() ||
		c.Server.KeepDisconnectedClients.Nanoseconds() > MaxKeepDisconnectedClients.Nanoseconds()) {
		return fmt.Errorf("expected 'Keep Lost Clients' can be in range [%v, %v], actual: %v", MinKeepDisconnectedClients, MaxKeepDisconnectedClients, c.Server.KeepDisconnectedClients)
	}

	if err := c.parseAndValidateClientAuth(); err != nil {
		return err
	}

	if err := c.parseAndValidateAPI(mLog); err != nil {
		return fmt.Errorf("API: %v", err)
	}

	if err := c.Database.ParseAndValidate(); err != nil {
		return err
	}

	maxProcs := runtime.GOMAXPROCS(0)

	mLog.Debugf("max_concurrent_ssh_handshakes = %d", c.Server.MaxConcurrentSSHConnectionHandshakes)

	if c.Server.MaxConcurrentSSHConnectionHandshakes > (maxProcs * 2) {
		mLog.Infof("warning: allowing too many concurrent ssh handhakes ('max_concurrent_ssh_handshakes') will slow down the server significantly and cause operational reliability issues. Please use a value less than or equal to the MAX_PROCS (%d)", maxProcs)
	}

	if c.Server.CheckClientsConnectionInterval < CheckClientsConnectionIntervalMinimum {
		c.Server.CheckClientsConnectionInterval = CheckClientsConnectionIntervalMinimum
		mLog.Errorf("'check_clients_status_interval' too fast. Using the minimum possible of %s", CheckClientsConnectionIntervalMinimum)
	}

	if err := c.Monitoring.parseAndValidateMonitoring(mLog); err != nil {
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
		c.Server.AuthID, c.Server.AuthPassword = chshare.ParseAuth(c.Server.Auth)
		if c.Server.AuthID == "" || c.Server.AuthPassword == "" {
			return fmt.Errorf("invalid client auth credentials, expected '<client-id>:<password>', got %q", c.Server.Auth)
		}
	}

	return nil
}

func (c *Config) parseAndValidateAPI(mLog *logger.MemLogger) error {
	if c.API.Address != "" {
		// API enabled
		err := c.parseAndValidateAPIAuth()
		if err != nil {
			return err
		}
		err = c.parseAndValidateBaseURL()
		if err != nil {
			return err
		}
		err = c.parseAndValidateAPIHTTPSOptions(false, false)
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

		err = c.parseAndValidateTotPSecret()
		if err != nil {
			return err
		}

		if c.API.TLSMin != "" && c.API.TLSMin != "1.2" && c.API.TLSMin != "1.3" {
			return errors.New("TLS must be either 1.2 or 1.3")
		}

		if c.API.MaxTokenLifeTimeHours < 0 || (time.Duration(c.API.MaxTokenLifeTimeHours)*time.Hour) > bearer.DefaultMaxTokenLifetime {
			return fmt.Errorf("max_token_lifetime outside allowable ranges. must be between 0 and %.0f", bearer.DefaultMaxTokenLifetime.Hours())
		}

		c.API.CORS = parseAndValidateCORS(mLog, c.API.CORS)
	} else {
		// API disabled
		if c.API.DocRoot != "" {
			return errors.New("to use document root you need to specify API address")
		}
	}
	err := c.API.AuditLog.Validate()
	if err != nil {
		return err
	}

	err = c.validateAPIWhenCaddyIntegration()
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) parseAndValidateBaseURL() error {
	u, err := url.Parse(c.API.BaseURL)
	if c.API.BaseURL != "" {
		if err != nil {
			return fmt.Errorf("base_url must be a valid url: %w", err)
		}
	}
	if c.API.EnableAcme {
		if u.Host == "" {
			return errors.New("base_url must have a host when acme is enabled")
		}
	}
	return nil
}

func (c *Config) validateAPIWhenCaddyIntegration() (err error) {
	caddyConfig := c.Caddy
	if !caddyConfig.Enabled {
		return nil
	}

	if !caddyConfig.APIReverseProxyEnabled() {
		// Check if the API and the tunnel subdomains are on the same port
		matchingPorts, err := matchingPorts(c.API.Address, caddyConfig.HostAddress)
		if err != nil {
			return err
		}
		if matchingPorts {
			return errors.New("API and tunnel subdomains are on the same port. The api_hostname and api_port must be configured")
		}
	}

	return nil
}

func (c *Config) WriteCaddyBaseConfig(caddyConfig *caddy.Config) (bc *caddy.BaseConfig, err error) {
	// targetAPIPort is required if the API reverse proxy is being used
	_, targetAPIPort, err := net.SplitHostPort(c.API.Address)
	if err != nil {
		return nil, err
	}

	bc, err = caddyConfig.MakeBaseConfig(targetAPIPort)
	if err != nil {
		return nil, err
	}

	baseConfigBytes, err := caddyConfig.GetBaseConf(bc)
	if err != nil {
		return nil, err
	}

	filename := caddyConfig.MakeBaseConfFilename()

	// note this remove is required as the caddy base config file is created with
	// read only permissions and write file won't overwrite the existing contents
	err = os.Remove(filename)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	err = os.WriteFile(filename, baseConfigBytes, 0400)
	if err != nil {
		return nil, err
	}

	return bc, nil
}

func (c *Config) CaddyEnabled() bool {
	return c.Caddy.Enabled
}

func matchingPorts(address1 string, address2 string) (matching bool, err error) {
	_, port1, err := net.SplitHostPort(address1)
	if err != nil {
		return false, err
	}
	_, port2, err := net.SplitHostPort(address2)
	if err != nil {
		return false, err
	}
	return port1 == port2, nil
}

func (mc *MonitoringConfig) parseAndValidateMonitoring(mLog *logger.MemLogger) (err error) {
	if !mc.Enabled {
		return nil
	}

	if mc.DataStorageDays > 0 {
		mLog.Infof("monitoring setting 'data_storage_days' is deprecated and will be removed soon. Use 'data_storage_duration' only instead.")
	}

	// we need to do this conversion as time.Duration doesn't support days
	mc.duration, err = convertHourOrDayStringToDuration("data_storage_duration", mc.DataStorageDuration)
	if err != nil {
		return err
	}

	if mc.Enabled && mc.GetDataStorageDuration() < time.Hour {
		return errors.New("monitoring results must be stored for at least 1 hour")
	}
	return nil
}

func convertHourOrDayStringToDuration(desc string, inputStr string) (duration time.Duration, err error) {
	if len(inputStr) == 0 {
		return 0, fmt.Errorf("'%s' must not be empty value", desc)
	}
	if len(inputStr) == 1 {
		return 0, fmt.Errorf("'%s' must include value and units, use suffix d (=days) or h (=hours)", desc)
	}

	// byte strings are assumed
	units := inputStr[len(inputStr)-1:]
	isHours := strings.EqualFold(units, "h")
	isDays := strings.EqualFold(units, "d")

	if !isHours && !isDays {
		return 0, fmt.Errorf("'%s' must include units of either d (=days) or h (=hours)", desc)
	}

	value := inputStr[:len(inputStr)-1]
	dur, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("'%s' must be simple value: %w", desc, err)
	}

	if isHours {
		duration = time.Duration(dur) * time.Hour
	}
	if isDays {
		duration = time.Duration(dur) * time.Hour * 24
	}

	return duration, nil
}

func (c *Config) parseAndValidateTotPSecret() error {
	if c.API.TwoFATokenDelivery != "" && c.API.TotPEnabled {
		return errors.New("conflicting 2FA configuration, two factor auth and totp_enabled options cannot be both enabled")
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
	default:
		// if the setting is a valid executable we set script delivery
		if _, err := exec.LookPath(c.API.TwoFATokenDelivery); err == nil {
			return c.API.parseAndValidate2FASendToType()
		}
		// if the setting is a valid url, we set url delivery
		if err := validateHTTPorHTTPSURL(c.API.TwoFATokenDelivery); err == nil {
			if c.API.BaseURL == "" {
				return errors.New("base_url is required for url two_fa_token_delivery")
			}
			return nil
		}
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

	if c.API.AuthHeader != "" {
		if c.API.Auth != "" {
			return errors.New("'auth_header' cannot be used with single user 'auth'")
		}
		if c.API.UserHeader == "" {
			return errors.New("'user_header' must be set when 'auth_header' is set")
		}
	}

	return nil
}

func (c *Config) parseAndValidateAPIHTTPSOptions(mustBeConfigured bool, skipLoadCheck bool) error {
	if !mustBeConfigured && c.API.CertFile == "" && c.API.KeyFile == "" {
		return nil
	}
	if c.API.CertFile != "" && c.API.KeyFile == "" {
		return errors.New("when 'cert_file' is set, 'key_file' must be set as well")
	}
	if c.API.CertFile == "" && c.API.KeyFile != "" {
		return errors.New("when 'key_file' is set, 'cert_file' must be set as well")
	}
	if c.API.EnableAcme {
		return errors.New("cert_file, key_file and enable_acme cannot be used together")
	}
	if !skipLoadCheck {
		_, err := tls.LoadX509KeyPair(c.API.CertFile, c.API.KeyFile)
		if err != nil {
			return fmt.Errorf("invalid 'cert_file', 'key_file': %v", err)
		}
	}
	return nil
}

func (s *ServerConfig) parseAndValidatePorts() error {
	usedPorts, err := ports.TryParsePortRanges(s.UsedPortsRaw)
	if err != nil {
		return fmt.Errorf("can't parse 'used_ports': %s", err)
	}

	excludedPorts, err := ports.TryParsePortRanges(s.ExcludedPortsRaw)
	if err != nil {
		return fmt.Errorf("can't parse 'excluded_ports': %s", err)
	}

	s.allowedPorts = usedPorts.Difference(excludedPorts)

	if s.allowedPorts.Cardinality() == 0 {
		return errors.New("invalid 'used_ports', 'excluded_ports': at least one port should be available for port assignment")
	}

	return nil
}

func (s *ServerConfig) parseAndValidateURLs() error {
	if len(s.URL) == 0 {
		s.URL = []string{"http://" + s.ListenAddress}
	}

	for _, v := range s.URL {
		if err := validateHTTPorHTTPSURL(v); err != nil {
			return errors.Wrap(err, "server.URL")
		}
	}
	if len(s.PairingURL) != 0 {
		if err := validateHTTPorHTTPSURL(s.PairingURL); err != nil {
			return errors.Wrap(err, "server.pairingURL")
		}
	}

	return nil
}

func validateHTTPorHTTPSURL(testURL string) error {
	u, err := url.ParseRequestURI(testURL)
	if err != nil {
		return fmt.Errorf("invalid url %s: %w", testURL, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid url %s: schema must be http or https", testURL)
	}

	if u.Host == "" {
		return fmt.Errorf("invalid url %s, must be absolute url", testURL)
	}
	return nil
}

func (d *DatabaseConfig) ParseAndValidate() error {
	switch d.Type {
	case "":
		return nil
	case "mysql":
		d.Driver = "mysql"
		d.Dsn = ""
		if d.User != "" {
			d.Dsn += d.User
			if d.Password != "" {
				d.Dsn += ":"
				d.Dsn += d.Password
			}
			d.Dsn += "@"
		}
		if d.Host != "" {
			if strings.HasPrefix(d.Host, socketPrefix) {
				d.Dsn += fmt.Sprintf("unix(%s)", strings.TrimPrefix(d.Host, socketPrefix))
			} else {
				d.Dsn += fmt.Sprintf("tcp(%s)", d.Host)
			}
		}
		d.Dsn += "/"
		d.Dsn += d.Name
	case "sqlite":
		d.Driver = "sqlite3"
		d.Dsn = d.Name
	default:
		return fmt.Errorf("invalid 'db_type', expected 'mysql' or 'sqlite', got %q", d.Type)
	}

	return nil
}

func (d *DatabaseConfig) DsnForLogs() string {
	if d.Password != "" {
		// hide the password
		return strings.Replace(d.Dsn, ":"+d.Password, ":***", 1)
	}
	return d.Dsn
}

func generateJWTSecret() (string, error) {
	data := make([]byte, 10)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("can't generate API JWT secret: %s", err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

func parseAndValidateCORS(mLog *logger.MemLogger, cors []string) []string {
	result := []string{}
	for _, c := range cors {
		err := validateCORSOrigin(c)
		if err != nil {
			mLog.Errorf("invalid cors origin %q: %v", c, err)
			continue
		}
		result = append(result, c)
	}
	return result
}

func validateCORSOrigin(c string) error {
	if c == "*" {
		return nil
	}
	u, err := url.Parse(c)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid scheme: %s", u.Scheme)
	}
	if u.Host == "" {
		return errors.New("must have a host")

	}
	if u.Path != "" {
		return errors.New("must not have a path")
	}
	if u.RawQuery != "" {
		return errors.New("must not have query params")
	}
	if u.Fragment != "" {
		return errors.New("must not contain a fragment")
	}
	return nil
}
