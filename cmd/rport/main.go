package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	chclient "github.com/cloudradar-monitoring/rport/client"
	config2 "github.com/cloudradar-monitoring/rport/client/monitoring/config"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

var clientHelp = `
  Usage: rport [options] [<server> [remote] [remote] [remote] ...]

  <server>, rportd server address. Mandatory IP address and port divided by a colon, unless --config(-c) is used.

  <remote>s are remote connections tunneled through the server, each of
  which come in the form:

    <local-interface>:<local-port>:<remote-host>:<remote-port>
    or
    <remote-host>:<remote-port>

  which does reverse port forwarding, sharing <remote-host>:<remote-port>
  from the client to the server's <local-interface>:<local-port>.
  If local part is omitted, a randomly chosen server port will be assigned.
  Only IPv4 addresses are supported.
  If not set, client connects without active tunnel(s) waiting for tunnels to be initialized by the server.

  Examples:

    ./rport <SERVER>:<PORT> 2222:127.0.0.1:22
    forwards port 2222 of the server to port 22 of the client

    ./rport <SERVER>:<PORT> 3000
    forwards randomly-assigned free port of the server to port 3000 of the client

    ./rport <SERVER>:<PORT> example.com:3000
    forwards port 3000 of the server to port 3000 of example.com
    originating the connection from the client

    ./rport <SERVER>:<PORT> 3000:google.com:80
    forwards port 3000 of the server to port 80 of google.com
    originating the connection from the client

    ./rport <SERVER>:<PORT> 192.168.0.5:3000:google.com:80
    server will listen on 192.168.0.5 interface forwarding all packets
    from port 3000 to port 80 of google.com
    originating the connection from the client

    ./rport "[2a01:4f9:c010:b278::1]:9999" 3389
    using IPv6 server address. Forwards randomly-assigned free port of the server
    to port 3389 of the client

    ./rport -c /etc/rport/rport.conf
    starts client with configuration loaded from the file

  Options:

    --fingerprint, A *strongly recommended* fingerprint string
    to perform host-key validation against the server's public key.
    You may provide just a prefix of the key or the entire string.
    Fingerprint mismatches will close the connection.

    --auth, Required client authentication credentials in the form: "<client-auth-id>:<password>".

    --keepalive, An optional keepalive interval. Since the underlying
    transport is HTTP, in many instances we'll be traversing through
    proxies, often these proxies will close idle connections. You must
    specify a time with a unit, for example '30s' or '2m'. Defaults
    to '0s' (disabled).

    --max-retry-count, Maximum number of times to retry before exiting.
    Defaults to unlimited (-1).

    --max-retry-interval, Maximum wait time before retrying after a
    disconnection. Defaults to 5 minutes ('5m').

    --proxy, An optional HTTP CONNECT or SOCKS5 proxy which will be
    used to reach the rport server. Authentication can be specified
    inside the URL.
    For example, http://admin:password@my-server.com:8081
             or: socks://admin:password@my-server.com:1080

    --header, Set a custom header in the form "HeaderName: HeaderContent".
    Can be used multiple times. (e.g --header "User-Agent: test1" --header "Authorization: Basic XXXXXX")

    --hostname, Optionally set the 'Host' header (defaults to the host
    found in the server url).

    --id, An optional client ID to better identify the client.
    If not set, a random id will be created that changes on every client start.
    The server rejects connections on duplicated ids.

    --name, An optional client name to better identify the client.
    Useful if you use numeric ids to make client identification easier.
    For example, --name "my_win_vm_1"
    Defaults to unset.

    --tag, -t, Optional values to give your clients attributes.
    Used for filtering clients on the server.
    Can be used multiple times. (e.g --tag "foobaz" --tag "bingo")

    --allow-root, An optional arg to allow running rport as root. There is no technical requirement to run the rport
    client under the root user. Running it as root is an unnecessary security risk.

    --service, Manages rport running as a service. Possible commands are "install", "uninstall", "start" and "stop".
    The only arguments compatible with --service are --service-user and --config, others will be ignored.

    --service-user, An optional arg specifying user to run rport service under. Only on linux. Defaults to rport.

    --log-level, Specify log level. Values: "error", "info", "debug" (defaults to "error")

    --log-file, -l, Specifies log file path. (defaults to empty string: log printed to stdout)

    --remote-commands-enabled, Enable or disable remote commands.
    Defaults: true

    --remote-scripts-enabled, Enable or disable remote scripts.
    Defaults: false

    --data-dir, Temporary directory to store temp client data.
    Defaults: /var/lib/rport (unix) or C:\Program Files\rport (windows)

    --remote-commands-send-back-limit, Limit the maximum length of the command or script output that is sent back.
    Applies to the stdout and stderr separately. If exceeded the specified number of bytes are sent.
    Defaults: 2048

    --updates-interval, How often after the rport client has started pending updates are summarized.
    Defaults: 4h

    --fallback-server, Set fallback server(s) to which the client tries to connect if the main server is not reachable.

    --server-switchback-interval, If connected to fallback server, try every interval to switch back to the main server.
    Defaults: 2m

    --monitoring-enabled, Enable or disable gathering of monitoring data.
    Defaults: true

   --monitoring-interval, the interval time in seconds, when monitoring data is gathered
   Defaults: 60s

   --monitoring-fs-type-include, list of filesystem types to include in list of mountpoints
   --monitoring-fs-path-exclude, list of filesystem path to exclude from list of mountpoints
   --monitoring-fs-path-exclude-recurse, enable or disable recursive handling
   --monitoring-fs-identify-mountpoints-by-device, enable or disable the identification of mountpoints by device

   --monitoring-pm-enabled, enable or disable process-monitoring
   --monitoring-pm-kerneltasks-enabled, enable or disable monitoring of kerneltasks
   --monitoring-pm-max-number-processes, maximum number of processes in process monitoring list

    --config, -c, An optional arg to define a path to a config file. If it is set then
    configuration will be loaded from the file. Note: command arguments and env variables will override them.
    MonitoringConfig file should be in TOML format. You can find an example "rport.example.conf" in the release archive.

    --help, This help text

    --version, Print version info and exit

  Signals:
    The rport process is listening for:
      a SIGUSR2 to print process stats, and
      a SIGHUP to short-circuit the client reconnect timer

`

var (
	RootCmd *cobra.Command

	cfgPath  *string
	viperCfg *viper.Viper
	config   = &chclient.Config{}

	svcCommand *string
	svcUser    *string
)

func init() {
	// Assign root cmd late to avoid initialization loop
	RootCmd = &cobra.Command{
		Version: chshare.BuildVersion,
		Run:     runMain,
	}

	pFlags := RootCmd.PersistentFlags()

	pFlags.String("fingerprint", "", "")
	pFlags.String("auth", "", "")
	pFlags.Duration("keepalive", 0, "")
	pFlags.Int("max-retry-count", 0, "")
	pFlags.Duration("max-retry-interval", 0, "")
	pFlags.String("proxy", "", "")
	pFlags.StringArray("header", []string{}, "")
	pFlags.String("id", "", "")
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

	cfgPath = pFlags.StringP("config", "c", "", "")
	svcCommand = pFlags.String("service", "", "")
	if runtime.GOOS != "windows" {
		svcUser = pFlags.String("service-user", "rport", "")
	}

	RootCmd.SetUsageFunc(func(*cobra.Command) error {
		fmt.Print(clientHelp)
		os.Exit(1)
		return nil
	})

	viperCfg = viper.New()
	viperCfg.SetConfigType("toml")

	viperCfg.SetDefault("client.server_switchback_interval", 2*time.Minute)
	viperCfg.SetDefault("logging.log_level", "error")
	viperCfg.SetDefault("connection.max_retry_count", -1)
	viperCfg.SetDefault("remote-commands.allow", []string{"^/usr/bin/.*", "^/usr/local/bin/.*", `^C:\\Windows\\System32\\.*`})
	viperCfg.SetDefault("remote-commands.deny", []string{`(\||<|>|;|,|\n|&)`})
	viperCfg.SetDefault("remote-commands.order", []string{"allow", "deny"})
	viperCfg.SetDefault("remote-commands.send_back_limit", 4194304)
	viperCfg.SetDefault("remote-commands.enabled", true)
	viperCfg.SetDefault("remote-scripts.enabled", false)
	viperCfg.SetDefault("client.updates_interval", 4*time.Hour)
	viperCfg.SetDefault("client.data_dir", chclient.DefaultDataDir)
	viperCfg.SetDefault("monitoring.enabled", true)
	viperCfg.SetDefault("monitoring.interval", config2.DefaultMonitoringInterval)
	viperCfg.SetDefault("monitoring.fs_type_include", []string{"ext3", "ext4", "xfs", "jfs", "ntfs", "btrfs", "hfs", "apfs", "exfat", "smbfs", "nfs"})
	viperCfg.SetDefault("monitoring.fs_identify_mountpoints_by_device", true)
	viperCfg.SetDefault("monitoring.pm_enabled", true)
	viperCfg.SetDefault("monitoring.pm_kerneltasks_enabled", true)
	viperCfg.SetDefault("monitoring.pm_max_number_processes", 500)
}

func bindPFlags() {
	pFlags := RootCmd.PersistentFlags()
	// map config fields to CLI args:
	_ = viperCfg.BindPFlag("client.fingerprint", pFlags.Lookup("fingerprint"))
	_ = viperCfg.BindPFlag("client.auth", pFlags.Lookup("auth"))
	_ = viperCfg.BindPFlag("client.proxy", pFlags.Lookup("proxy"))
	_ = viperCfg.BindPFlag("client.id", pFlags.Lookup("id"))
	_ = viperCfg.BindPFlag("client.name", pFlags.Lookup("name"))
	_ = viperCfg.BindPFlag("client.tags", pFlags.Lookup("tag"))
	_ = viperCfg.BindPFlag("client.allow_root", pFlags.Lookup("allow-root"))
	_ = viperCfg.BindPFlag("client.updates_interval", pFlags.Lookup("updates-interval"))
	_ = viperCfg.BindPFlag("client.fallback_servers", pFlags.Lookup("fallback-server"))
	_ = viperCfg.BindPFlag("client.server_switchback_interval", pFlags.Lookup("server-switchback-interval"))
	_ = viperCfg.BindPFlag("client.data_dir", pFlags.Lookup("data-dir"))

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
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func decodeConfig(args []string) error {
	if *cfgPath != "" {
		viperCfg.SetConfigFile(*cfgPath)
	} else {
		viperCfg.AddConfigPath(".")
		viperCfg.SetConfigName("rport.conf")
	}

	if err := chshare.DecodeViperConfig(viperCfg, config); err != nil {
		return err
	}

	if len(args) > 0 {
		config.Client.Server = args[0]
		config.Client.Remotes = args[1:]
	}

	return nil
}

func runMain(cmd *cobra.Command, args []string) {
	if svcCommand != nil && *svcCommand != "" {
		// validate config file without command line args before installing it for the service
		// other service commands do not change config file specified at install
		if *svcCommand == "install" {
			err := decodeConfig(nil)
			if err != nil {
				log.Fatalf("Invalid config: %v. Check your config file.", err)
			}

			err = config.ParseAndValidate(true)
			if err != nil {
				log.Fatalf("Invalid config: %v. Check your config file.", err)
			}
		}
		err := handleSvcCommand(*svcCommand, *cfgPath, svcUser)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	// Bind command line arguments late, so they're not included in validation for service install
	bindPFlags()

	err := decodeConfig(nil)
	if err != nil {
		log.Fatalf("Invalid config: %v. Check your config file.", err)
	}
	err = config.Logging.LogOutput.Start()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		config.Logging.LogOutput.Shutdown()
	}()

	err = chclient.PrepareDirs(config)
	if err != nil {
		log.Fatalf("Invalid config: %v. Check your config file.", err)
	}

	err = config.ParseAndValidate(false)
	if err != nil {
		log.Fatalf("Invalid config: %v. Check your config file.", err)
	}

	if !config.Client.AllowRoot && chshare.IsRunningAsRoot() {
		log.Fatal("By default running as root is not allowed.")
	}

	c := chclient.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	if !service.Interactive() {
		err = runAsService(c, *cfgPath)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	go chshare.GoStats()

	if err = c.Run(); err != nil {
		log.Fatal(err)
	}
}
