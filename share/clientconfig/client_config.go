package clientconfig

import (
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/openrport/openrport/share/logger"
	"github.com/openrport/openrport/share/models"
)

type Config struct {
	Client                   ClientConfig        `json:"client" mapstructure:"client"`
	Connection               ConnectionConfig    `json:"connection" mapstructure:"connection"`
	Logging                  LogConfig           `json:"logging" mapstructure:"logging"`
	RemoteCommands           CommandsConfig      `json:"remote_commands" mapstructure:"remote-commands"`
	RemoteScripts            ScriptsConfig       `json:"remote_scripts" mapstructure:"remote-scripts"`
	Monitoring               MonitoringConfig    `json:"monitoring" mapstructure:"monitoring"`
	Tunnels                  TunnelsConfig       `json:"-"`
	InterpreterAliasesConfig map[string]any      `json:"-" mapstructure:"interpreter-aliases"`
	FileReceptionConfig      FileReceptionConfig `json:"file_reception" mapstructure:"file-reception"`

	InterpreterAliases          map[string]string                   `json:"interpreter_aliases"`
	InterpreterAliasesEncodings map[string]InterpreterAliasEncoding `json:"interpreter_aliases_encodings"`
}

type ClientConfig struct {
	AttributesFilePath       string            `json:"-" mapstructure:"attributes_file_path"`
	Server                   string            `json:"server" mapstructure:"server"`
	FallbackServers          []string          `json:"fallback_servers" mapstructure:"fallback_servers"`
	ServerSwitchbackInterval time.Duration     `json:"server_switchback_interval" mapstructure:"server_switchback_interval"`
	Fingerprint              string            `json:"fingerprint" mapstructure:"fingerprint"`
	Auth                     string            `json:"auth" mapstructure:"auth"`
	Proxy                    string            `json:"proxy" mapstructure:"proxy"`
	ID                       string            `json:"id" mapstructure:"id"`
	UseSystemID              bool              `json:"use_system_id" mapstructure:"use_system_id"`
	Name                     string            `json:"name" mapstructure:"name"`
	UseHostname              bool              `json:"use_hostname" mapstructure:"use_hostname"`
	Tags                     []string          `json:"tags" mapstructure:"tags"`
	Labels                   map[string]string `json:"labels" mapstructure:"labels"`
	Remotes                  []string          `json:"remotes" mapstructure:"remotes"`
	TunnelAllowed            []string          `json:"tunnel_allowed" mapstructure:"tunnel_allowed"`
	AllowRoot                bool              `json:"allow_root" mapstructure:"allow_root"`
	UpdatesInterval          time.Duration     `json:"updates_interval" mapstructure:"updates_interval"`
	InventoryInterval        time.Duration     `json:"inventory_interval" mapstructure:"inventory_interval"`
	DataDir                  string            `json:"data_dir" mapstructure:"data_dir"`
	BindInterface            string            `json:"bind_interface" mapstructure:"bind_interface"`
	IPAPIURL                 string            `json:"ip_api_url" mapstructure:"ip_api_url"`
	IPRefreshMin             time.Duration     `json:"ip_refresh_min" mapstructure:"ip_refresh_min"`

	ProxyURL *url.URL         `json:"proxy_url"`
	Tunnels  []*models.Remote `json:"tunnels"`
	AuthUser string           `json:"auth_user"`
	AuthPass string           `json:"auth_pass"`
}

type TunnelsConfig struct {
	Scheme       string `json:"scheme"`
	ReverseProxy bool   `json:"reverse_proxy"`
	HostHeader   string `json:"host_header"`
}

type ConnectionConfig struct {
	KeepAlive           time.Duration `json:"keep_alive" mapstructure:"keep_alive"`
	KeepAliveTimeout    time.Duration `json:"keep_alive_timeout" mapstructure:"keep_alive_timeout"`
	MaxRetryCount       int           `json:"max_retry_count" mapstructure:"max_retry_count"`
	MaxRetryInterval    time.Duration `json:"max_retry_interval" mapstructure:"max_retry_interval"`
	HeadersRaw          []string      `json:"headers" mapstructure:"headers"`
	Hostname            string        `json:"hostname" mapstructure:"hostname"`
	WatchdogIntegration bool          `json:"watchdog_integration" mapstructure:"watchdog_integration"`

	HTTPHeaders http.Header `json:"http_headers"`
}

type LogConfig struct {
	LogOutput logger.LogOutput `json:"log_file" mapstructure:"log_file"`
	LogLevel  logger.LogLevel  `json:"log_level" mapstructure:"log_level"`
}

type CommandsConfig struct {
	Enabled       bool      `json:"enabled" mapstructure:"enabled"`
	SendBackLimit int       `json:"send_back_limit" mapstructure:"send_back_limit"`
	Allow         []string  `json:"allow" mapstructure:"allow"`
	Deny          []string  `json:"deny" mapstructure:"deny"`
	Order         [2]string `json:"order" mapstructure:"order"`

	AllowRegexp []*regexp.Regexp `json:"allow_regexp"`
	DenyRegexp  []*regexp.Regexp `json:"deny_regexp"`
}

type ScriptsConfig struct {
	Enabled bool `json:"enabled" mapstructure:"enabled"`
}

type MonitoringConfig struct {
	Enabled                       bool          `json:"enabled" mapstructure:"enabled"`
	Interval                      time.Duration `json:"interval" mapstructure:"interval"`
	FSTypeInclude                 []string      `json:"fs_type_include" mapstructure:"fs_type_include"`
	FSPathExclude                 []string      `json:"fs_path_exclude" mapstructure:"fs_path_exclude"`
	FSPathExcludeRecurse          bool          `json:"fs_path_exclude_recurse" mapstructure:"fs_path_exclude_recurse"`
	FSIdentifyMountpointsByDevice bool          `json:"fs_identify_mountpoints_by_device" mapstructure:"fs_identify_mountpoints_by_device"`
	PMEnabled                     bool          `json:"pm_enabled" mapstructure:"pm_enabled"`
	PMKerneltasksEnabled          bool          `json:"pm_kerneltasks_enabled" mapstructure:"pm_kerneltasks_enabled"`
	PMMaxNumberProcesses          uint          `json:"pm_max_number_processes" mapstructure:"pm_max_number_processes"`
	NetLan                        []string      `json:"net_lan" mapstructure:"net_lan"`
	NetWan                        []string      `json:"net_wan" mapstructure:"net_wan"`

	LanCard *models.NetworkCard `json:"lan_card"`
	WanCard *models.NetworkCard `json:"wan_card"`
}

type FileReceptionConfig struct {
	Protected []string `json:"protected" mapstructure:"protected"`
	Enabled   bool     `json:"enabled" mapstructure:"enabled"`
}

type InterpreterAliasEncoding struct {
	InputEncoding  string `json:"input_encoding"`
	OutputEncoding string `json:"output_encoding"`
}
