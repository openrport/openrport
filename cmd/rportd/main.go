package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	chserver "github.com/cloudradar-monitoring/rport/server"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

const (
	DefaultCSRFileName             = "csr.json"
	DefaultCacheClientsInterval    = 1 * time.Second
	DefaultSaveClientsAuthInterval = 5 * time.Second
	DefaultCleanClientsInterval    = 3 * time.Second
	DefaultMaxRequestBytes         = 2 * 1024 // 2 KB
	DefaultCheckPortTimeout        = 2 * time.Second
)

var serverHelp = `
  Usage: rportd [options]

  Examples:

    ./rportd --addr=0.0.0.0:9999 
    starts server, listening to client connections on port 9999

    ./rportd --addr="[2a01:4f9:c010:b278::1]:9999" --api-addr=0.0.0.0:9000 --api-auth=admin:1234
    starts server, listening to client connections on IPv6 interface,
    also enabling HTTP API, available at http://0.0.0.0:9000/

  Options:

    --addr, -a, Defines the IP address and port the HTTP server listens on.
    (defaults to the environment variable RPORT_ADDR and falls back to 0.0.0.0:8080).

    --url, Defines full client connect URL. Defaults to "http://{addr}"

    --exclude-ports, -e, Defines port numbers or ranges of server ports,
    separated with comma that would not be used for automatic port assignment.
    Defaults to 1-1000.
    e.g.: --exclude-ports=1-1000,8080 or -e 22,443,80,8080,5000-5999

    --key, An optional string to seed the generation of a ECDSA public
    and private key pair. All communications will be secured using this
    key pair. Share the subsequent fingerprint with clients to enable detection
    of man-in-the-middle attacks (defaults to the RPORT_KEY environment
    variable, otherwise a new key is generate each run).

    --authfile, An optional path to a json file with client credentials.
    This is for authentication of the rport tunnel clients.
    The file should contain a map with clients credentials defined like:
      {
        "<client1-id>": "<password1>"
        "<client2-id>": "<password2>"
      }

    --auth, An optional string representing a single client with full access, in the form of <client-id>:<password>.
    This is equivalent to creating an authfile with {"<client-id>":"<password>"}.
    Use either "authfile" or "auth". Not both.

    --auth-write, If you want to delegate the creation and maintenance to an external tool
    you should set this value to "false". The API will reject all writing access to the
    client auth with HTTP 403. Applies only to --authfile and --auth-table. Default is "true".

    --auth-multiuse-creds, When using --authfile creating separate credentials for each client is recommended.
    It increases security because you can lock out clients individually. 
    If auth-multiuse-creds is false a client is rejected if another client with the same username is connected
    or has been connected within the --keep-lost-clients interval.
    Defaults: true

    --equate-authusername-clientid, Having set "--auth-multiuse-creds=false", you can omit specifying a client-id.
    You can us the authentication username as client-id to slim down the client configuration.
    Defaults: false

    --save-clients-auth-interval, Applicable only if --authfile is specified and --auth-write is true.
    An optional arg to define an interval to flush rport clients auth info to disk. By default, '5s' is used.

    --proxy, Specifies another HTTP server to proxy requests to when
    rportd receives a normal HTTP request. Useful for hiding rportd in
    plain sight.

    --api-addr, Defines the IP address and port the API server listens on.
    e.g. "0.0.0.0:7777". (defaults to the environment variable RPORT_API_ADDR
    and fallsback to empty string: API not available)

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
    e.g. "admin:1234". (defaults to the environment variable RPORT_API_AUTH
    and fallsback to empty string: authorization not required).

    --data-dir, An optional arg to define a local directory path to store internal data.
    By default, "/var/lib/rportd" is used on Linux, "C:\ProgramData\rportd" is used on Windows.

    --keep-lost-clients, An optional arg to define a duration to keep info(sessions, tunnels, etc)
    about active and disconnected clients. Enables to identify disconnected clients
    at server restart and to reestablish previous tunnels on reconnect.
    By default is "0"(is disabled). For example, "--keep-lost-clients=1h30m".
    It can contain "h"(hours), "m"(minutes), "s"(seconds).

    --save-clients-interval, Applicable only if --keep-lost-clients is specified. An optional arg to define
    an interval to flush info (sessions, tunnels, etc) about active and disconnected clients to disk.
    By default, 1 second is used. It can contain "h"(hours), "m"(minutes), "s"(seconds).

    --cleanup-clients-interval, Applicable only if --keep-lost-clients is specified. An optional
    arg to define an interval to clean up internal storage from obsolete disconnected clients.
    By default, '3s' is used. It can contain "h"(hours), "m"(minutes), "s"(seconds).

    --csr-file-name, An optional arg to define a file name in --data-dir directory to store
    info about active and disconnected clients. By default, "csr.json" is used.

    --check-port-timeout, An optional arg to define a timeout to check whether a remote destination of a requested
    new tunnel is available, i.e. whether a given remote port is open on a client machine. By default, "2s" is used.

    --api-jwt-secret, Defines JWT secret used to generate new tokens.
    (defaults to the environment variable RPORT_API_JWT_SECRET and fallsback
    to auto-generated value).

    --max-request-bytes, An optional arg to define a limit for data that can be sent by rport clients and API requests.
    By default is set to 2048(2Kb).

    --verbose, -v, Specify log level. Values: "error", "info", "debug" (defaults to "error")

    --log-file, -l, Specifies log file path. (defaults to empty string: log printed to stdout)

    --config, -c, An optional arg to define a path to a config file. If it is set then
    configuration will be loaded from the file. Note: command arguments and env variables will override them.
    Config file should be in TOML format. You can find an example "rportd.example.conf" in the release archive.

    --help, -h, This help text

    --version, Print version info and exit

  Signals:
    The rportd process is listening for SIGUSR2 to print process stats

`

var (
	RootCmd = &cobra.Command{
		Version: chshare.BuildVersion,
		Run:     runMain,
	}

	cfgPath  *string
	viperCfg *viper.Viper
	cfg      = &chserver.Config{}
)

func init() {
	pFlags := RootCmd.PersistentFlags()

	pFlags.StringP("addr", "a", "", "")
	pFlags.String("url", "", "")
	pFlags.String("key", "", "")
	pFlags.String("authfile", "", "")
	pFlags.String("auth", "", "")
	pFlags.String("proxy", "", "")
	pFlags.String("api-addr", "", "")
	pFlags.String("api-authfile", "", "")
	pFlags.String("api-auth", "", "")
	pFlags.String("api-jwt-secret", "", "")
	pFlags.String("api-doc-root", "", "")
	pFlags.StringP("log-file", "l", "", "")
	pFlags.StringP("verbose", "v", "", "")
	pFlags.StringSliceP("exclude-ports", "e", []string{}, "")
	pFlags.String("data-dir", chserver.DefaultDataDirectory, "")
	pFlags.String("csr-file-name", DefaultCSRFileName, "")
	pFlags.Duration("keep-lost-clients", 0, "")
	pFlags.Duration("save-clients-interval", DefaultCacheClientsInterval, "")
	pFlags.Duration("cleanup-clients-interval", DefaultCleanClientsInterval, "")
	pFlags.Int64("max-request-bytes", DefaultMaxRequestBytes, "")
	pFlags.Duration("check-port-timeout", DefaultCheckPortTimeout, "")
	pFlags.Bool("auth-write", true, "")
	pFlags.Bool("auth-multiuse-creds", true, "")
	pFlags.Bool("equate-authusername-clientid", false, "")
	pFlags.Duration("save-clients-auth-interval", DefaultSaveClientsAuthInterval, "")

	cfgPath = pFlags.StringP("config", "c", "", "")

	RootCmd.SetUsageFunc(func(*cobra.Command) error {
		fmt.Print(serverHelp)
		os.Exit(1)
		return nil
	})

	viperCfg = viper.New()
	viperCfg.SetConfigType("toml")

	viperCfg.SetDefault("log_level", "error")
	viperCfg.SetDefault("address", "0.0.0.0:8080")
	viperCfg.SetDefault("excluded_ports", "0-1000")

	// map config fields to CLI args:
	// _ is used to ignore errors to pass linter check
	_ = viperCfg.BindPFlag("log_file", pFlags.Lookup("log-file"))
	_ = viperCfg.BindPFlag("log_level", pFlags.Lookup("verbose"))
	_ = viperCfg.BindPFlag("address", pFlags.Lookup("addr"))
	_ = viperCfg.BindPFlag("url", pFlags.Lookup("url"))
	_ = viperCfg.BindPFlag("key_seed", pFlags.Lookup("key"))
	_ = viperCfg.BindPFlag("auth_file", pFlags.Lookup("authfile"))
	_ = viperCfg.BindPFlag("auth", pFlags.Lookup("auth"))
	_ = viperCfg.BindPFlag("proxy", pFlags.Lookup("proxy"))
	_ = viperCfg.BindPFlag("api.address", pFlags.Lookup("api-addr"))
	_ = viperCfg.BindPFlag("api.auth", pFlags.Lookup("api-auth"))
	_ = viperCfg.BindPFlag("api.auth_file", pFlags.Lookup("api-authfile"))
	_ = viperCfg.BindPFlag("api.jwt_secret", pFlags.Lookup("api-jwt-secret"))
	_ = viperCfg.BindPFlag("api.doc_root", pFlags.Lookup("api-doc-root"))
	_ = viperCfg.BindPFlag("excluded_ports", pFlags.Lookup("exclude-ports"))
	_ = viperCfg.BindPFlag("data_dir", pFlags.Lookup("data-dir"))
	_ = viperCfg.BindPFlag("csr_file_name", pFlags.Lookup("csr-file-name"))
	_ = viperCfg.BindPFlag("keep_lost_clients", pFlags.Lookup("keep-lost-clients"))
	_ = viperCfg.BindPFlag("save_clients_interval", pFlags.Lookup("save-clients-interval"))
	_ = viperCfg.BindPFlag("cleanup_clients_interval", pFlags.Lookup("cleanup-clients-interval"))
	_ = viperCfg.BindPFlag("max_request_bytes", pFlags.Lookup("max-request-bytes"))
	_ = viperCfg.BindPFlag("check_port_timeout", pFlags.Lookup("check-port-timeout"))
	_ = viperCfg.BindPFlag("auth_write", pFlags.Lookup("auth-write"))
	_ = viperCfg.BindPFlag("auth_multiuse_creds", pFlags.Lookup("auth-multiuse-creds"))
	_ = viperCfg.BindPFlag("equate_authusername_clientid", pFlags.Lookup("equate-authusername-clientid"))
	_ = viperCfg.BindPFlag("save_clients_auth_interval", pFlags.Lookup("save-clients-auth-interval"))

	// map ENV variables
	_ = viperCfg.BindEnv("address", "RPORT_ADDR")
	_ = viperCfg.BindEnv("url", "RPORT_URL")
	_ = viperCfg.BindEnv("key_seed", "RPORT_KEY")
	_ = viperCfg.BindEnv("api.address", "RPORT_API_ADDR")
	_ = viperCfg.BindEnv("api.auth", "RPORT_API_AUTH")
	_ = viperCfg.BindEnv("api.jwt_secret", "RPORT_JWT_SECRET")
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func tryDecodeConfig() error {
	if *cfgPath != "" {
		viperCfg.SetConfigFile(*cfgPath)
	} else {
		viperCfg.AddConfigPath(".")
		viperCfg.SetConfigName("rportd.conf")
	}

	return chshare.DecodeViperConfig(viperCfg, cfg)
}

func runMain(*cobra.Command, []string) {
	err := tryDecodeConfig()
	if err != nil {
		log.Fatal(err)
	}

	err = cfg.ParseAndValidate()
	if err != nil {
		log.Fatal(err)
	}

	err = cfg.LogOutput.Start()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		cfg.LogOutput.Shutdown()
	}()

	s, err := chserver.NewServer(cfg)
	if err != nil {
		log.Fatal(err)
	}

	go chshare.GoStats()

	if err = s.Run(); err != nil {
		log.Fatal(err)
	}
}
