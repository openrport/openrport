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

	chserver "github.com/cloudradar-monitoring/rport/server"
	"github.com/cloudradar-monitoring/rport/server/api/message"
	"github.com/cloudradar-monitoring/rport/server/auditlog"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/files"
)

const (
	DefaultKeepLostClients           = time.Hour
	DefaultCleanClientsInterval      = 1 * time.Minute
	DefaultMaxRequestBytes           = 10 * 1024  // 10 KB
	DefaultMaxRequestBytesClient     = 512 * 1024 // 512KB
	DefaultCheckPortTimeout          = 2 * time.Second
	DefaultUsedPorts                 = "20000-30000"
	DefaultExcludedPorts             = "1-1024"
	DefaultServerAddress             = "0.0.0.0:8080"
	DefaultLogLevel                  = "info"
	DefaultRunRemoteCmdTimeoutSec    = 60
	DefaultMonitoringDataStorageDays = 30
)

var serverHelp = `
  Usage: rportd [options]

  Examples:

    ./rportd --addr=0.0.0.0:9999 --auth=clientAuth1:1234
    starts server, listening to client connections on port 9999

    ./rportd --addr="[2a01:4f9:c010:b278::1]:9999" --api-addr=0.0.0.0:9000 --api-auth=admin:1234 --auth=clientAuth1:1234
    starts server, listening to client connections on IPv6 interface,
    also enabling HTTP API, available at http://0.0.0.0:9000/

    ./rportd -c /etc/rport/rportd.conf
    starts server with configuration loaded from the file

  Options:

    --addr, -a, Defines the IP address and port the HTTP server listens on.
    This is where the rport clients connect to.
    Defaults: "0.0.0.0:8080"

    --use-ports, Defines port numbers or ranges of server ports,
    separated with comma that would be used for automatic and manual port assignment.
    Creating reverse tunnels will fail if the requested server port is not listed here.
    Defaults to 20000-30000
    e.g.: --use-ports=20000-30000,9999

    --exclude-ports, -e, Defines port numbers or ranges of server ports,
    separated with comma that would not be used for automatic and manual port assignment.
    Values that are not included in the --use-ports are ignored.
    Defaults to 1-1024. If no ports should be excluded then set it to ""(empty string).
    e.g.: --exclude-ports=1-1024,8080 or -e 22,443,80,8080,5000-5999

    --key, An optional string to seed the generation of a ECDSA public
    and private key pair. All communications will be secured using this
    key pair. Share the subsequent fingerprint with clients to enable detection
    of man-in-the-middle attacks. If not specified, a new key is generate each run.
    Use "openssl rand -hex 18" to generate a secure key seed

    --authfile, An optional path to a json file with client credentials.
    This is for authentication of the rport tunnel clients.
    The file should contain a map with clients credentials defined like:
      {
        "<client-auth-id1>": "<password1>"
        "<client-auth-id2>": "<password2>"
      }

    --auth, An optional string representing a single client auth credentials, in the form of <client-auth-id>:<password>.
    This is equivalent to creating an authfile with {"<client-auth-id>":"<password>"}.
    Use either "authfile", "auth-table" or "auth". If multiple auth options are enabled, rportd exits with an error.

    --auth-table, An optional name of a database table for client authentication.
    Requires a global database connection. The table must be created manually.
    Use either "authfile", "auth-table" or "auth". If multiple auth options are enabled, rportd exits with an error.

    --auth-write, If you want to delegate the creation and maintenance to an external tool
    you should set this value to "false". The API will reject all writing access to the
    client auth with HTTP 403. Applies only to --authfile and --auth-table. Default is "true".

    --auth-multiuse-creds, When using --authfile creating separate credentials for each client is recommended.
    It increases security because you can lock out clients individually.
    If auth-multiuse-creds is false a client is rejected if another client with the same id is connected
    or has been connected within the --keep-lost-clients interval.
    Defaults: true

    --equate-clientauthid-clientid, Having set "--auth-multiuse-creds=false", you can omit specifying a client-id.
    You can use the client-auth-id as client-id to slim down the client configuration.
    Defaults: false

    --proxy, Specifies another HTTP server to proxy requests to when
    rportd receives a normal HTTP request. Useful for hiding rportd in
    plain sight.

    --api-addr, Defines the IP address and port the API server listens on.
    e.g. "0.0.0.0:7777". Defaults to empty string: API not available.

    --api-doc-root, Specifies local directory path. If specified, rportd will serve
    files from this directory on the same API address (--api-addr).

    --api-authfile, Defines a path to a JSON file that contains users, password, and groups for accessing the API.
    Passwords must be bcrypt encrypted. This file should be structured like:
    [
      {
        "username": "admin",
        "password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu",
        "groups": ["admins", "users", "gods"]
      },
      {
        "username": "minion",
        "password": "$2y$40$eqwLZekPE/pxLb4g9n8rU.OLIdPnWnOo.q5wqqA0LyYf3ihonenlu",
        "groups": ["users"]
      }
    ]

    --api-auth, Defines <user>:<password> authentication pair for accessing API
    e.g. "admin:1234". Defaults to empty string: authorization not required.

    --api-auth-user-table, An optional arg to specify database table to use for API auth users.

    --api-auth-group-table, An optional arg to specify database table to use for API auth groups.

    --api-cert-file, An optional arg to specify certificate file for API with https.
    Https will be activated if both cert and key file are set.

    --api-key-file, An optional arg to specify private key file for API with https.
    Https will be activated if both cert and key file are set.

    --api-access-log-file, An optional arg to specify file for writing api access logs.

    --db-type, An optional arg to specify database type. Values 'mysql' or 'sqlite'.

    --db-host, An optional arg to specify host for mysql database.

    --db-user, An optional arg to specify user for mysql database.

    --db-password, An optional arg to specify password for mysql database.

    --db-name, An optional arg to specify name for mysql database or file for sqlite.

    --data-dir, An optional arg to define a local directory path to store internal data.
    By default, "/var/lib/rportd" is used on Linux, 'C:\ProgramData\rportd' is used on Windows.
    If the directory doesn't exist, it will be created. On Linux you must create this directory
    because an unprivileged user don't have the right to create a directory in /var/lib.
    Ideally this directory is the homedir of the rport user and has been created along with the user.
    Example: useradd -r -d /var/lib/rportd -m -s /bin/false -U -c "System user for rport client and server" rport

    --keep-lost-clients, An optional arg to define a duration to keep info(clients, tunnels, etc)
    about active and disconnected clients.
    By default is "1h". To disable it set it to "0".
    It can contain "h"(hours), "m"(minutes), "s"(seconds).

    --cleanup-clients-interval, An optional
    arg to define an interval to clean up internal storage from obsolete disconnected clients.
    By default, '1m' is used. It can contain "h"(hours), "m"(minutes), "s"(seconds).

    --check-port-timeout, An optional arg to define a timeout to check whether a remote destination of a requested
    new tunnel is available, i.e. whether a given remote port is open on a client machine. By default, "2s" is used.

    --run-remote-cmd-timeout-sec, An optional arg to define a timeout in seconds to observe the remote command execution.
    Defaults: 60

    --api-jwt-secret, Defines JWT secret used to generate new tokens.
    Defaults to auto-generated value.

    --max-request-bytes, An optional arg to define a limit for data that can be sent by API requests.
    By default is set to 10240(10Kb).

    --max-request-bytes-client, An optional arg to define a limit for data that can be sent by rport clients.
    By default is set to 524288(512Kb).

    --allow-root, An optional arg to allow running rportd as root. There is no technical requirement to run the rport
    server under the root user. Running it as root is an unnecessary security risk.

    --service, Manages rportd running as a service. Possible commands are "install", "uninstall", "start" and "stop".
    The only arguments compatible with --service are --service-user and --config, others will be ignored.

    --service-user, An optional arg specifying user to run rportd service under. Only on linux. Defaults to rport.

    --log-level, Specify log level. Values: "error", "info", "debug" (defaults to "info")

    --log-file, -l, Specifies log file path. (defaults to empty string: log printed to stdout)

    --config, -c, An optional arg to define a path to a config file. If it is set then
    configuration will be loaded from the file. Note: command arguments and env variables will override them.
    Config file should be in TOML format. You can find an example "rportd.example.conf" in the release archive.

    --monitoring-data-storage-days, The number of days, client monitoring data is stored on server (defaults to 30)

    --tunnel-proxy-cert-file, An optional arg to specify certificate file for http tunnel proxy.
    Https tunnel proxy can be activated if both cert and key file are set.

    --tunnel-proxy-key-file, An optional arg to specify key file for http tunnel proxy.
    Https tunnel proxy can be activated if both cert and key file are set.

    --novnc-root, Specifies local directory path. If specified, rportd will serve
    novnc javascript app from this directory.

    --help, -h, This help text

    --version, Print version info and exit

  Signals:
    The rportd process is listening for SIGUSR2 to print process stats

`

var (
	RootCmd  *cobra.Command
	cfgPath  *string
	viperCfg *viper.Viper
	cfg      = &chserver.Config{}

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

	pFlags.StringP("addr", "a", "", "")
	pFlags.String("url", "", "")
	pFlags.String("key", "", "")
	pFlags.String("authfile", "", "")
	pFlags.String("auth", "", "")
	pFlags.String("auth-table", "", "")
	pFlags.String("proxy", "", "")
	pFlags.String("api-addr", "", "")
	pFlags.String("api-authfile", "", "")
	pFlags.String("api-auth", "", "")
	pFlags.String("api-auth-user-table", "", "")
	pFlags.String("api-auth-group-table", "", "")
	pFlags.String("api-jwt-secret", "", "")
	pFlags.String("api-doc-root", "", "")
	pFlags.String("api-cert-file", "", "")
	pFlags.String("api-key-file", "", "")
	pFlags.String("api-access-log-file", "", "")
	pFlags.String("db-type", "", "")
	pFlags.String("db-name", "", "")
	pFlags.String("db-host", "", "")
	pFlags.String("db-user", "", "")
	pFlags.String("db-password", "", "")
	pFlags.StringP("log-file", "l", "", "")
	pFlags.String("log-level", "", "")
	pFlags.StringSlice("use-ports", nil, "")
	pFlags.StringSliceP("exclude-ports", "e", nil, "")
	pFlags.String("data-dir", "", "")
	pFlags.Duration("keep-lost-clients", 0, "")
	pFlags.Duration("save-clients-interval", 0, "")
	pFlags.Duration("cleanup-clients-interval", 0, "")
	pFlags.Int64("max-request-bytes", 0, "")
	pFlags.Int64("max-request-bytes-client", 0, "")
	pFlags.Duration("check-port-timeout", 0, "")
	pFlags.Bool("auth-write", false, "")
	pFlags.Bool("auth-multiuse-creds", false, "")
	pFlags.Bool("equate-clientauthid-clientid", false, "")
	pFlags.Int("run-remote-cmd-timeout-sec", 0, "")
	pFlags.Bool("allow-root", false, "")
	pFlags.Int64("monitoring-data-storage-days", 0, "")
	pFlags.String("tunnel-proxy-cert-file", "", "")
	pFlags.String("tunnel-proxy-key-file", "", "")
	pFlags.String("novnc-root", "", "")

	cfgPath = pFlags.StringP("config", "c", "", "")
	svcCommand = pFlags.String("service", "", "")
	if runtime.GOOS != "windows" {
		svcUser = pFlags.String("service-user", "rport", "")
	}

	RootCmd.SetUsageFunc(func(*cobra.Command) error {
		fmt.Print(serverHelp)
		os.Exit(1)
		return nil
	})

	viperCfg = viper.New()
	viperCfg.SetConfigType("toml")

	viperCfg.SetDefault("logging.log_level", DefaultLogLevel)
	viperCfg.SetDefault("server.address", DefaultServerAddress)
	viperCfg.SetDefault("server.used_ports", []string{DefaultUsedPorts})
	viperCfg.SetDefault("server.excluded_ports", []string{DefaultExcludedPorts})
	viperCfg.SetDefault("server.data_dir", chserver.DefaultDataDirectory)
	viperCfg.SetDefault("server.keep_lost_clients", DefaultKeepLostClients)
	viperCfg.SetDefault("server.cleanup_clients_interval", DefaultCleanClientsInterval)
	viperCfg.SetDefault("server.max_request_bytes", DefaultMaxRequestBytes)
	viperCfg.SetDefault("server.max_request_bytes_client", DefaultMaxRequestBytesClient)
	viperCfg.SetDefault("server.check_port_timeout", DefaultCheckPortTimeout)
	viperCfg.SetDefault("server.auth_write", true)
	viperCfg.SetDefault("server.auth_multiuse_creds", true)
	viperCfg.SetDefault("server.run_remote_cmd_timeout_sec", DefaultRunRemoteCmdTimeoutSec)
	viperCfg.SetDefault("server.client_login_wait", 2)
	viperCfg.SetDefault("server.max_failed_login", 5)
	viperCfg.SetDefault("server.ban_time", 3600)
	viperCfg.SetDefault("server.enable_ws_test_endpoints", false)
	viperCfg.SetDefault("api.user_header", "Authentication-User")
	viperCfg.SetDefault("api.default_user_group", "Administrators")
	viperCfg.SetDefault("api.user_login_wait", 2)
	viperCfg.SetDefault("api.max_failed_login", 10)
	viperCfg.SetDefault("api.ban_time", 600)
	viperCfg.SetDefault("api.two_fa_token_ttl_seconds", 600)
	viperCfg.SetDefault("api.two_fa_send_timeout", 10*time.Second)
	viperCfg.SetDefault("api.two_fa_send_to_type", message.ValidationNone)
	viperCfg.SetDefault("api.enable_audit_log", true)
	viperCfg.SetDefault("api.totp_enabled", false)
	viperCfg.SetDefault("api.audit_log_rotation", auditlog.RotationMonthly)
	viperCfg.SetDefault("monitoring.data_storage_days", DefaultMonitoringDataStorageDays)
	viperCfg.SetDefault("api.totp_login_session_ttl", time.Minute*10)
	viperCfg.SetDefault("api.totp_account_name", "RPort")
}

func bindPFlags() {
	pFlags := RootCmd.PersistentFlags()
	// map config fields to CLI args:
	// _ is used to ignore errors to pass linter check
	_ = viperCfg.BindPFlag("server.address", pFlags.Lookup("addr"))
	_ = viperCfg.BindPFlag("server.url", pFlags.Lookup("url"))
	_ = viperCfg.BindPFlag("server.key_seed", pFlags.Lookup("key"))
	_ = viperCfg.BindPFlag("server.auth", pFlags.Lookup("auth"))
	_ = viperCfg.BindPFlag("server.auth_file", pFlags.Lookup("authfile"))
	_ = viperCfg.BindPFlag("server.auth_table", pFlags.Lookup("auth-table"))
	_ = viperCfg.BindPFlag("server.auth_multiuse_creds", pFlags.Lookup("auth-multiuse-creds"))
	_ = viperCfg.BindPFlag("server.equate_clientauthid_clientid", pFlags.Lookup("equate-clientauthid-clientid"))
	_ = viperCfg.BindPFlag("server.auth_write", pFlags.Lookup("auth-write"))
	_ = viperCfg.BindPFlag("server.proxy", pFlags.Lookup("proxy"))
	_ = viperCfg.BindPFlag("server.used_ports", pFlags.Lookup("use-ports"))
	_ = viperCfg.BindPFlag("server.excluded_ports", pFlags.Lookup("exclude-ports"))
	_ = viperCfg.BindPFlag("server.data_dir", pFlags.Lookup("data-dir"))
	_ = viperCfg.BindPFlag("server.keep_lost_clients", pFlags.Lookup("keep-lost-clients"))
	_ = viperCfg.BindPFlag("server.cleanup_clients_interval", pFlags.Lookup("cleanup-clients-interval"))
	_ = viperCfg.BindPFlag("server.max_request_bytes", pFlags.Lookup("max-request-bytes"))
	_ = viperCfg.BindPFlag("server.max_request_bytes_client", pFlags.Lookup("max-request-bytes-client"))
	_ = viperCfg.BindPFlag("server.check_port_timeout", pFlags.Lookup("check-port-timeout"))
	_ = viperCfg.BindPFlag("server.run_remote_cmd_timeout_sec", pFlags.Lookup("run-remote-cmd-timeout-sec"))
	_ = viperCfg.BindPFlag("server.allow_root", pFlags.Lookup("allow-root"))
	_ = viperCfg.BindPFlag("server.tunnel_proxy_cert_file", pFlags.Lookup("tunnel-proxy-cert-file"))
	_ = viperCfg.BindPFlag("server.tunnel_proxy_key_file", pFlags.Lookup("tunnel-proxy-key-file"))
	_ = viperCfg.BindPFlag("server.novnc_root", pFlags.Lookup("novnc-root"))

	_ = viperCfg.BindPFlag("logging.log_file", pFlags.Lookup("log-file"))
	_ = viperCfg.BindPFlag("logging.log_level", pFlags.Lookup("log-level"))

	_ = viperCfg.BindPFlag("api.address", pFlags.Lookup("api-addr"))
	_ = viperCfg.BindPFlag("api.auth", pFlags.Lookup("api-auth"))
	_ = viperCfg.BindPFlag("api.auth_file", pFlags.Lookup("api-authfile"))
	_ = viperCfg.BindPFlag("api.auth_user_table", pFlags.Lookup("api-auth-user-table"))
	_ = viperCfg.BindPFlag("api.auth_group_table", pFlags.Lookup("api-auth-group-table"))
	_ = viperCfg.BindPFlag("api.jwt_secret", pFlags.Lookup("api-jwt-secret"))
	_ = viperCfg.BindPFlag("api.doc_root", pFlags.Lookup("api-doc-root"))
	_ = viperCfg.BindPFlag("api.cert_file", pFlags.Lookup("api-cert-file"))
	_ = viperCfg.BindPFlag("api.key_file", pFlags.Lookup("api-key-file"))
	_ = viperCfg.BindPFlag("api.access_log_file", pFlags.Lookup("api-access-log-file"))
	_ = viperCfg.BindPFlag("database.db_type", pFlags.Lookup("db-type"))
	_ = viperCfg.BindPFlag("database.db_name", pFlags.Lookup("db-name"))
	_ = viperCfg.BindPFlag("database.db_host", pFlags.Lookup("db-host"))
	_ = viperCfg.BindPFlag("database.db_user", pFlags.Lookup("db-user"))
	_ = viperCfg.BindPFlag("database.db_password", pFlags.Lookup("db-password"))

	_ = viperCfg.BindPFlag("monitoring.data_storage_days", pFlags.Lookup("monitoring-data-storage-days"))
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func decodeAndValidateConfig() error {
	if *cfgPath != "" {
		viperCfg.SetConfigFile(*cfgPath)
	} else {
		viperCfg.AddConfigPath(".")
		viperCfg.SetConfigName("rportd.conf")
	}

	if err := chshare.DecodeViperConfig(viperCfg, cfg); err != nil {
		return err
	}

	err := cfg.ParseAndValidate()
	if err != nil {
		return err
	}

	return nil
}

func runMain(*cobra.Command, []string) {
	if svcCommand != nil && *svcCommand != "" {
		// validate config file without command line args before installing it for the service
		// other service commands do not change config file specified at install
		if *svcCommand == "install" {
			err := decodeAndValidateConfig()
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

	err := decodeAndValidateConfig()
	if err != nil {
		log.Fatalf("Invalid config: %v. See --help", err)
	}

	if !cfg.Server.AllowRoot && chshare.IsRunningAsRoot() {
		log.Fatal("By default running as root is not allowed.")
	}

	err = cfg.Logging.LogOutput.Start()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		cfg.Logging.LogOutput.Shutdown()
	}()

	s, err := chserver.NewServer(cfg, files.NewFileSystem())
	if err != nil {
		log.Fatal(err)
	}

	if !service.Interactive() {
		err = runAsService(s, *cfgPath)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	go chshare.GoStats()

	if err = s.Run(); err != nil {
		log.Fatal(err)
	}
}
