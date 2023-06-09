package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/realvnc-labs/rport/cmd/rportd/servicemanagement"
	"github.com/realvnc-labs/rport/share/logger"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	chserver "github.com/realvnc-labs/rport/server"
	"github.com/realvnc-labs/rport/server/api/message"
	auditlog "github.com/realvnc-labs/rport/server/auditlog/config"
	"github.com/realvnc-labs/rport/server/chconfig"
	chshare "github.com/realvnc-labs/rport/share"
	"github.com/realvnc-labs/rport/share/files"
)

const (
	DefaultKeepDisconnectedClients          = time.Hour
	DefaultPurgeDisconnectedClientsInterval = 1 * time.Minute
	DefaultCheckClientsConnectionInterval   = 5 * time.Minute
	DefaultCheckClientsConnectionTimeout    = 30 * time.Second
	DefaultMaxRequestBytes                  = 10 * 1024       // 10 KB
	DefaultMaxRequestBytesClient            = 512 * 1024      // 512KB
	DefaultMaxFilePushBytes                 = int64(10 << 20) // 10M
	DefaultCheckPortTimeout                 = 2 * time.Second
	DefaultUsedPorts                        = "20000-30000"
	DefaultExcludedPorts                    = "1-1024"
	DefaultServerAddress                    = "0.0.0.0:8080"
	DefaultLogLevel                         = "info"
	DefaultRunRemoteCmdTimeoutSec           = 60
	DefaultMonitoringDataStorageDuration    = "7d"
	DefaultPairingURL                       = "https://pairing.rport.io"
)

var (
	DefaultMaxConcurrentSSHConnectionHandshakes = calcMaxConcurrentSSHConnectionHandshakes()
)

func calcMaxConcurrentSSHConnectionHandshakes() (max int) {
	maxProcs := runtime.GOMAXPROCS(0)
	if maxProcs == 1 {
		return maxProcs
	}
	return maxProcs / 2
}

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

    ./rportd user
    commands for user management, run './rportd user help' for more options

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
    or has been connected within the purge_disconnected_clients_interval interval.
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

    --max-filepush-bytes, An optional arg to define a limit for the file size that can be uploaded to server.
    By default is set to 10000000(10Mb).

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

    --guacd-address, Specifies network address (host:port) of guacd daemon. If specified, rportd will serve
    remote desktop connections in browser using Guacamole protocol.

    --help, -h, This help text

    --version, Print version info and exit

  Signals:
    The rportd process is listening for SIGUSR2 to print process stats

`

var (
	RootCmd  *cobra.Command
	cfgPath  *string
	viperCfg *viper.Viper
	cfg      = &chconfig.Config{}

	svcCommand *string
	svcUser    *string
)

func init() {
	// Assign root cmd late to avoid initialization loop
	RootCmd = &cobra.Command{
		Use:     "rportd",
		Version: chshare.BuildVersion,
		Run:     runMain,
	}

	// lFlags are used only when starting server
	// pFlags are used when running subcommands like user as well
	lFlags := RootCmd.Flags()
	pFlags := RootCmd.PersistentFlags()

	lFlags.StringP("addr", "a", "", "")
	lFlags.String("url", "", "")
	lFlags.String("key", "", "")
	lFlags.String("authfile", "", "")
	lFlags.String("auth", "", "")
	lFlags.String("auth-table", "", "")
	lFlags.String("proxy", "", "")
	lFlags.String("api-addr", "", "")
	lFlags.String("api-authfile", "", "")
	lFlags.String("api-auth", "", "")
	lFlags.String("api-auth-user-table", "", "")
	lFlags.String("api-auth-group-table", "", "")
	lFlags.String("api-jwt-secret", "", "")
	lFlags.String("api-doc-root", "", "")
	lFlags.String("api-cert-file", "", "")
	lFlags.String("api-key-file", "", "")
	lFlags.String("api-access-log-file", "", "")
	lFlags.String("db-type", "", "")
	lFlags.String("db-name", "", "")
	lFlags.String("db-host", "", "")
	lFlags.String("db-user", "", "")
	lFlags.String("db-password", "", "")
	lFlags.StringP("log-file", "l", "", "")
	lFlags.String("log-level", "", "")
	lFlags.StringSlice("use-ports", nil, "")
	lFlags.StringSliceP("exclude-ports", "e", nil, "")
	lFlags.String("data-dir", "", "")
	lFlags.Duration("save-clients-interval", 0, "")
	lFlags.Int64("max-request-bytes", 0, "")
	lFlags.Int64("max-filepush-bytes", 0, "")
	lFlags.Int64("max-request-bytes-client", 0, "")
	lFlags.Duration("check-port-timeout", 0, "")
	lFlags.Bool("auth-write", false, "")
	lFlags.Bool("auth-multiuse-creds", false, "")
	lFlags.Bool("equate-clientauthid-clientid", false, "")
	lFlags.Int("run-remote-cmd-timeout-sec", 0, "")
	lFlags.Bool("allow-root", false, "")
	lFlags.Int64("monitoring-data-storage-days", 0, "")
	lFlags.String("tunnel-proxy-cert-file", "", "")
	lFlags.String("tunnel-proxy-key-file", "", "")
	lFlags.String("novnc-root", "", "")
	lFlags.String("guacd-address", "", "")

	cfgPath = pFlags.StringP("config", "c", "", "location of the config file")
	svcCommand = lFlags.String("service", "", "")
	if runtime.GOOS != "windows" {
		svcUser = lFlags.String("service-user", "rport", "")
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
	viperCfg.SetDefault("server.sqlite_wal", true)
	viperCfg.SetDefault("server.keep_disconnected_clients", DefaultKeepDisconnectedClients)
	viperCfg.SetDefault("server.max_concurrent_ssh_handshakes", DefaultMaxConcurrentSSHConnectionHandshakes)
	viperCfg.SetDefault("server.purge_disconnected_clients_interval", DefaultPurgeDisconnectedClientsInterval)
	viperCfg.SetDefault("server.check_clients_connection_interval", DefaultCheckClientsConnectionInterval)
	viperCfg.SetDefault("server.check_clients_connection_timeout", DefaultCheckClientsConnectionTimeout)
	viperCfg.SetDefault("server.max_request_bytes_client", DefaultMaxRequestBytesClient)
	viperCfg.SetDefault("server.check_port_timeout", DefaultCheckPortTimeout)
	viperCfg.SetDefault("server.auth_write", true)
	viperCfg.SetDefault("server.auth_multiuse_creds", true)
	viperCfg.SetDefault("server.run_remote_cmd_timeout_sec", DefaultRunRemoteCmdTimeoutSec)
	viperCfg.SetDefault("server.client_login_wait", 2)
	viperCfg.SetDefault("server.max_failed_login", 5)
	viperCfg.SetDefault("server.pairing_url", DefaultPairingURL)
	viperCfg.SetDefault("server.ban_time", 3600)
	viperCfg.SetDefault("server.jobs_max_results", 10000)
	viperCfg.SetDefault("server.tls_min", "1.3")
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
	viperCfg.SetDefault("monitoring.data_storage_duration", DefaultMonitoringDataStorageDuration)
	viperCfg.SetDefault("monitoring.enabled", true)
	viperCfg.SetDefault("api.max_request_bytes", DefaultMaxRequestBytes)
	viperCfg.SetDefault("api.max_filepush_size", DefaultMaxFilePushBytes)
	viperCfg.SetDefault("api.enable_ws_test_endpoints", false)
	viperCfg.SetDefault("api.totp_login_session_ttl", time.Minute*10)
	viperCfg.SetDefault("api.totp_account_name", "RPort")
	viperCfg.SetDefault("api.password_min_length", 14)
	viperCfg.SetDefault("api.password_zxcvbn_minscore", 0)
	viperCfg.SetDefault("api.tls_min", "1.3")
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
	_ = viperCfg.BindPFlag("server.max_request_bytes_client", pFlags.Lookup("max-request-bytes-client"))
	_ = viperCfg.BindPFlag("server.check_port_timeout", pFlags.Lookup("check-port-timeout"))
	_ = viperCfg.BindPFlag("server.run_remote_cmd_timeout_sec", pFlags.Lookup("run-remote-cmd-timeout-sec"))
	_ = viperCfg.BindPFlag("server.allow_root", pFlags.Lookup("allow-root"))
	_ = viperCfg.BindPFlag("server.tunnel_proxy_cert_file", pFlags.Lookup("tunnel-proxy-cert-file"))
	_ = viperCfg.BindPFlag("server.tunnel_proxy_key_file", pFlags.Lookup("tunnel-proxy-key-file"))
	_ = viperCfg.BindPFlag("server.novnc_root", pFlags.Lookup("novnc-root"))
	_ = viperCfg.BindPFlag("server.guacd_address", pFlags.Lookup("guacd-address"))

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
	_ = viperCfg.BindPFlag("api.max_request_bytes", pFlags.Lookup("max-request-bytes"))
	_ = viperCfg.BindPFlag("api.max_filepush_size", pFlags.Lookup("max-filepush-bytes"))
	_ = viperCfg.BindPFlag("database.db_type", pFlags.Lookup("db-type"))
	_ = viperCfg.BindPFlag("database.db_name", pFlags.Lookup("db-name"))
	_ = viperCfg.BindPFlag("database.db_host", pFlags.Lookup("db-host"))
	_ = viperCfg.BindPFlag("database.db_user", pFlags.Lookup("db-user"))
	_ = viperCfg.BindPFlag("database.db_password", pFlags.Lookup("db-password"))

	_ = viperCfg.BindPFlag("monitoring.data_storage_duration", pFlags.Lookup("monitoring-data-storage-duration"))
	_ = viperCfg.BindPFlag("monitoring.enabled", pFlags.Lookup("monitoring-enabled"))
	_ = viperCfg.BindPFlag("monitoring.data_storage_days", pFlags.Lookup("monitoring-data-storage-days"))
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func decodeAndValidateConfig(mLog *logger.MemLogger) error {
	if *cfgPath != "" {
		viperCfg.SetConfigFile(*cfgPath)
	} else {
		viperCfg.AddConfigPath(".")
		viperCfg.SetConfigName("rportd.conf")
	}

	if err := chshare.DecodeViperConfig(viperCfg, cfg, nil); err != nil {
		return err
	}

	err := cfg.ParseAndValidate(mLog)
	if err != nil {
		return err
	}

	return nil
}

func runMain(*cobra.Command, []string) {
	// Create an in-memory logger while the real logger is not loaded yet
	mLog := logger.NewMemLogger()
	if svcCommand != nil && *svcCommand != "" {
		// validate config file without command line args before installing it for the service
		// other service commands do not change config file specified at install
		if *svcCommand == "install" {
			err := decodeAndValidateConfig(&mLog)
			if err != nil {
				log.Fatalf("Invalid config: %v. Check your config file.", err)
			}
		}

		err := servicemanagement.HandleSvcCommand(*svcCommand, *cfgPath, svcUser)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	// Bind command line arguments late, so they're not included in validation for service install
	bindPFlags()

	err := decodeAndValidateConfig(&mLog)
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
	// Flush the in-memory logger
	mLog.Flush(logger.NewLogger("server-startup", cfg.Logging.LogOutput, cfg.Logging.LogLevel))

	filesAPI := files.NewFileSystem()

	// this ctx will be used to co-ordinate shutdown of the various server go-routines
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	plusManager, err := chserver.EnablePlusIfAvailable(ctx, cfg, filesAPI)
	if err != nil && err != chserver.ErrPlusNotEnabled {
		log.Fatal(err)
	}

	s, err := chserver.NewServer(ctx, cfg, &chserver.ServerOpts{
		FilesAPI:    filesAPI,
		PlusManager: plusManager,
	})
	if err != nil {
		log.Fatal(err)
	}

	if !service.Interactive() {
		err = servicemanagement.RunAsService(s, *cfgPath)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	go chshare.GoStats()

	err = s.Run(ctx)
	s.Logger.Debugf("run finished")

	if err != nil {
		// the standard context canceled message is somewhat unclear. just let the user know that
		// the server has shutdown.
		if strings.Contains(err.Error(), "context canceled") {
			log.Fatal("server shutdown")
		}

		log.Fatal(err)
	}
}
