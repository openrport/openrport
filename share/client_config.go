package chshare

import (
	"net/http"
	"net/url"
	"regexp"
	"time"
)

type Config struct {
	Client         ClientConfig     `mapstructure:"client"`
	Connection     ConnectionConfig `mapstructure:"connection"`
	Logging        LogConfig        `mapstructure:"logging"`
	RemoteCommands CommandsConfig   `mapstructure:"remote-commands"`
	RemoteScripts  ScriptsConfig    `mapstructure:"remote-scripts"`
	Monitoring     MonitoringConfig `mapstructure:"monitoring"`
}

type ClientConfig struct {
	Server                   string        `mapstructure:"server"`
	FallbackServers          []string      `mapstructure:"fallback_servers"`
	ServerSwitchbackInterval time.Duration `mapstructure:"server_switchback_interval"`
	Fingerprint              string        `mapstructure:"fingerprint"`
	Auth                     string        `mapstructure:"auth"`
	Proxy                    string        `mapstructure:"proxy"`
	ID                       string        `mapstructure:"id"`
	Name                     string        `mapstructure:"name"`
	Tags                     []string      `mapstructure:"tags"`
	Remotes                  []string      `mapstructure:"remotes"`
	AllowRoot                bool          `mapstructure:"allow_root"`
	UpdatesInterval          time.Duration `mapstructure:"updates_interval"`
	DataDir                  string        `mapstructure:"data_dir"`

	ProxyURL *url.URL
	Tunnels  []*Remote
	AuthUser string
	AuthPass string
}

type ConnectionConfig struct {
	KeepAlive        time.Duration `mapstructure:"keep_alive"`
	MaxRetryCount    int           `mapstructure:"max_retry_count"`
	MaxRetryInterval time.Duration `mapstructure:"max_retry_interval"`
	HeadersRaw       []string      `mapstructure:"headers"`
	Hostname         string        `mapstructure:"hostname"`

	HTTPHeaders http.Header
}

type LogConfig struct {
	LogOutput LogOutput `mapstructure:"log_file"`
	LogLevel  LogLevel  `mapstructure:"log_level"`
}

type CommandsConfig struct {
	Enabled       bool      `mapstructure:"enabled"`
	SendBackLimit int       `mapstructure:"send_back_limit"`
	Allow         []string  `mapstructure:"allow"`
	Deny          []string  `mapstructure:"deny"`
	Order         [2]string `mapstructure:"order"`

	AllowRegexp []*regexp.Regexp
	DenyRegexp  []*regexp.Regexp
}

type ScriptsConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type MonitoringConfig struct {
	Enabled                       bool          `mapstructure:"enabled"`
	Interval                      time.Duration `mapstructure:"interval"`
	FSTypeInclude                 []string      `mapstructure:"fs_type_include"`
	FSPathExclude                 []string      `mapstructure:"fs_path_exclude"`
	FSPathExcludeRecurse          bool          `mapstructure:"fs_path_exclude_recurse"`
	FSIdentifyMountpointsByDevice bool          `mapstructure:"fs_identify_mountpoints_by_device"`
	PMEnabled                     bool          `mapstructure:"pm_enabled"`
	PMKerneltasksEnabled          bool          `mapstructure:"pm_kerneltasks_enabled"`
	PMMaxNumberProcesses          uint          `mapstructure:"pm_max_number_processes"`
}
