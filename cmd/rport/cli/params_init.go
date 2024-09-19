package cli

import (
	"os"
	"time"

	chclient "github.com/openrport/openrport/client"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func BindPFlagsToViperConfig(pFlags *pflag.FlagSet, viperCfg *viper.Viper) {

	// map config fields to CLI args:
	_ = viperCfg.BindPFlag("client.fingerprint", pFlags.Lookup("fingerprint"))
	_ = viperCfg.BindPFlag("client.auth", pFlags.Lookup("auth"))
	_ = viperCfg.BindPFlag("client.proxy", pFlags.Lookup("proxy"))
	_ = viperCfg.BindPFlag("client.use_system_id", pFlags.Lookup("use-system-id"))
	_ = viperCfg.BindPFlag("client.id", pFlags.Lookup("id"))
	_ = viperCfg.BindPFlag("client.use_hostname", pFlags.Lookup("use-hostname"))
	_ = viperCfg.BindPFlag("client.name", pFlags.Lookup("name"))
	_ = viperCfg.BindPFlag("client.tags", pFlags.Lookup("tag"))
	_ = viperCfg.BindPFlag("client.allow_root", pFlags.Lookup("allow-root"))
	_ = viperCfg.BindPFlag("client.updates_interval", pFlags.Lookup("updates-interval"))
	_ = viperCfg.BindPFlag("client.inventory_interval", pFlags.Lookup("inventory-interval"))
	_ = viperCfg.BindPFlag("client.fallback_servers", pFlags.Lookup("fallback-server"))
	_ = viperCfg.BindPFlag("client.server_switchback_interval", pFlags.Lookup("server-switchback-interval"))
	_ = viperCfg.BindPFlag("client.data_dir", pFlags.Lookup("data-dir"))
	_ = viperCfg.BindPFlag("client.bind_interface", pFlags.Lookup("bind-interface"))

	_ = viperCfg.BindPFlag("logging.log_file", pFlags.Lookup("log-file"))
	_ = viperCfg.BindPFlag("logging.log_level", pFlags.Lookup("log-level"))

	_ = viperCfg.BindPFlag("connection.keep_alive", pFlags.Lookup("keepalive"))
	_ = viperCfg.BindPFlag("connection.max_retry_count", pFlags.Lookup("max-retry-count"))
	_ = viperCfg.BindPFlag("connection.max_retry_interval", pFlags.Lookup("max-retry-interval"))
	_ = viperCfg.BindPFlag("connection.hostname", pFlags.Lookup("hostname"))
	_ = viperCfg.BindPFlag("connection.headers", pFlags.Lookup("header"))

	_ = viperCfg.BindPFlag("remote-commands.enabled", pFlags.Lookup("remote-commands-enabled"))
	_ = viperCfg.BindPFlag("remote-scripts.enabled", pFlags.Lookup("remote-scripts-enabled"))
	_ = viperCfg.BindPFlag("remote-commands.send_back_limit", pFlags.Lookup("remote-commands-send-back-limit"))

	_ = viperCfg.BindPFlag("monitoring.enabled", pFlags.Lookup("monitoring-enabled"))
	_ = viperCfg.BindPFlag("monitoring.interval", pFlags.Lookup("monitoring-interval"))
	_ = viperCfg.BindPFlag("monitoring.fs_type_include", pFlags.Lookup("monitoring-fs-type-include"))
	_ = viperCfg.BindPFlag("monitoring.fs_path_exclude", pFlags.Lookup("monitoring-fs-path-exclude"))
	_ = viperCfg.BindPFlag("monitoring.fs_path_exclude_recurse", pFlags.Lookup("monitoring-fs-path-exclude-recurse"))
	_ = viperCfg.BindPFlag("monitoring.fs_identify_mountpoints_by_device", pFlags.Lookup("monitoring-fs-identify-mountpoints-by-device"))
	_ = viperCfg.BindPFlag("monitoring.pm_enabled", pFlags.Lookup("monitoring-pm-enabled"))
	_ = viperCfg.BindPFlag("monitoring.pm_kerneltasks_enabled", pFlags.Lookup("monitoring-pm-kerneltasks-enabled"))
	_ = viperCfg.BindPFlag("monitoring.pm_max_number_processes", pFlags.Lookup("monitoring-pm-max-number-processes"))
	_ = viperCfg.BindPFlag("monitoring.net_lan", pFlags.Lookup("monitoring-net-lan"))
	_ = viperCfg.BindPFlag("monitoring.net_wan", pFlags.Lookup("monitoring-net-wan"))

	_ = viperCfg.BindPFlag("file-reception.protected", pFlags.Lookup("file-reception-protected"))
	_ = viperCfg.BindPFlag("file-reception.enabled", pFlags.Lookup("file-reception-enabled"))
}

func SetPFlags(pFlags *pflag.FlagSet) {
	// needed always
	pFlags.StringP("config", "c", "", "")
	pFlags.String("service", "", "")

	// needed ony when installing service to configure OS service
	pFlags.String("service-user", "rport", "")

	// needed only when run as rport client
	pFlags.String("scheme", "", "")
	pFlags.Bool("enable-reverse-proxy", false, "")
	pFlags.String("host-header", "", "")

	// present in config file
	pFlags.String("fingerprint", os.Getenv("RPORT_FINGERPRINT"), "")
	pFlags.String("auth", os.Getenv("RPORT_AUTH"), "")
	pFlags.Duration("keepalive", 0, "")
	pFlags.Int("max-retry-count", 0, "")
	pFlags.Duration("max-retry-interval", 0, "")
	pFlags.String("proxy", "", "")
	pFlags.StringArray("header", []string{}, "")
	pFlags.Bool("use-system-id", true, "")
	pFlags.String("id", "", "")
	pFlags.Bool("use-hostname", true, "")
	pFlags.String("name", "", "")
	pFlags.StringArrayP("tag", "t", []string{}, "")
	pFlags.String("hostname", "", "")
	pFlags.StringP("log-file", "l", "", "")
	pFlags.String("log-level", "", "")
	pFlags.Bool("allow-root", false, "")
	pFlags.Bool("remote-commands-enabled", false, "")
	pFlags.Bool("remote-scripts-enabled", false, "")
	pFlags.String("data-dir", chclient.DefaultDataDir, "")
	pFlags.Int("remote-commands-send-back-limit", 0, "")
	pFlags.Duration("updates-interval", 0, "")
	pFlags.StringArray("fallback-server", []string{}, "")
	pFlags.Duration("server-switchback-interval", 0, "")
	pFlags.Bool("monitoring-enabled", false, "")
	pFlags.Duration("monitoring-interval", 0, "")
	pFlags.StringArray("monitoring-fs-type-include", []string{}, "")
	pFlags.StringArray("monitoring-fs-path-exclude", []string{}, "")
	pFlags.Bool("monitoring-fs-path-exclude-recurse", false, "")
	pFlags.Bool("monitoring-fs-identify-mountpoints-by-device", false, "")
	pFlags.Bool("monitoring-pm-enabled", false, "")
	pFlags.Bool("monitoring-pm-kerneltasks-enabled", false, "")
	pFlags.Int("monitoring-pm-max-number-processes", 0, "")
	pFlags.StringArray("monitoring-net-lan", []string{}, "")
	pFlags.StringArray("monitoring-net-wan", []string{}, "")
	pFlags.StringArray("file-reception-protected", []string{}, "")
	pFlags.Bool("file-reception-enabled", true, "")
	pFlags.String("bind-interface", "", "")
}

func SetViperConfigDefaults(viperCfg *viper.Viper) {

	viperCfg.SetDefault("logging.log_level", "info")

	viperCfg.SetDefault("connection.max_retry_count", -1)
	viperCfg.SetDefault("connection.keep_alive", "3m")
	viperCfg.SetDefault("connection.keep_alive_timeout", "30s")

	viperCfg.SetDefault("remote-commands.allow", []string{"^/usr/bin/.*", "^/usr/local/bin/.*", `^C:\\Windows\\System32\\.*`})
	viperCfg.SetDefault("remote-commands.deny", []string{`(\||<|>|;|,|\n|&)`})
	viperCfg.SetDefault("remote-commands.order", []string{"allow", "deny"})
	viperCfg.SetDefault("remote-commands.send_back_limit", 4194304)
	viperCfg.SetDefault("remote-commands.enabled", true)
	viperCfg.SetDefault("remote-scripts.enabled", false)

	viperCfg.SetDefault("client.server_switchback_interval", 2*time.Minute)
	viperCfg.SetDefault("client.updates_interval", 4*time.Hour)
	viperCfg.SetDefault("client.inventory_interval", 4*time.Hour)
	viperCfg.SetDefault("client.data_dir", chclient.DefaultDataDir)
	viperCfg.SetDefault("client.attributes_file_path", "")
	viperCfg.SetDefault("client.ip_refresh_min", 30)

	viperCfg.SetDefault("monitoring.enabled", true)
	viperCfg.SetDefault("monitoring.interval", chclient.DefaultMonitoringInterval)
	viperCfg.SetDefault("monitoring.fs_type_include", []string{"ext3", "ext4", "xfs", "jfs", "ntfs", "btrfs", "hfs", "apfs", "exfat", "smbfs", "nfs"})
	viperCfg.SetDefault("monitoring.fs_identify_mountpoints_by_device", true)
	viperCfg.SetDefault("monitoring.pm_enabled", true)
	viperCfg.SetDefault("monitoring.pm_kerneltasks_enabled", true)
	viperCfg.SetDefault("monitoring.pm_max_number_processes", 500)

	viperCfg.SetDefault("file-reception.protected", chclient.FileReceptionGlobs)
	viperCfg.SetDefault("file-reception.enabled", true)
}
