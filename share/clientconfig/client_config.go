package clientconfig

import (
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
)

type Config struct {
	Client         ClientConfig     `json:"client" mapstructure:"client"`
	Connection     ConnectionConfig `json:"connection" mapstructure:"connection"`
	Logging        LogConfig        `json:"logging" mapstructure:"logging"`
	RemoteCommands CommandsConfig   `json:"remote_commands" mapstructure:"remote-commands"`
	RemoteScripts  ScriptsConfig    `json:"remote_scripts" mapstructure:"remote-scripts"`
	Monitoring     MonitoringConfig `json:"monitoring" mapstructure:"monitoring"`
}

type ClientConfig struct {
	Server                   string        `json:"server" mapstructure:"server"`
	FallbackServers          []string      `json:"fallback_servers" mapstructure:"fallback_servers"`
	ServerSwitchbackInterval time.Duration `json:"server_switchback_interval" mapstructure:"server_switchback_interval"`
	Fingerprint              string        `json:"fingerprint" mapstructure:"fingerprint"`
	Auth                     string        `json:"auth" mapstructure:"auth"`
	Proxy                    string        `json:"proxy" mapstructure:"proxy"`
	ID                       string        `json:"id" mapstructure:"id"`
	Name                     string        `json:"name" mapstructure:"name"`
	Tags                     []string      `json:"tags" mapstructure:"tags"`
	Remotes                  []string      `json:"remotes" mapstructure:"remotes"`
	AllowRoot                bool          `json:"allow_root" mapstructure:"allow_root"`
	UpdatesInterval          time.Duration `json:"updates_interval" mapstructure:"updates_interval"`
	DataDir                  string        `json:"data_dir" mapstructure:"data_dir"`

	ProxyURL *url.URL         `json:"proxy_url"`
	Tunnels  []*models.Remote `json:"tunnels"`
	AuthUser string           `json:"auth_user"`
	AuthPass string           `json:"auth_pass"`
}

type ConnectionConfig struct {
	KeepAlive        time.Duration `json:"keep_alive" mapstructure:"keep_alive"`
	MaxRetryCount    int           `json:"max_retry_count" mapstructure:"max_retry_count"`
	MaxRetryInterval time.Duration `json:"max_retry_interval" mapstructure:"max_retry_interval"`
	HeadersRaw       []string      `json:"headers" mapstructure:"headers"`
	Hostname         string        `json:"hostname" mapstructure:"hostname"`

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
