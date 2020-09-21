package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	chserver "github.com/cloudradar-monitoring/rport/server"
	"github.com/cloudradar-monitoring/rport/server/scheduler"
	"github.com/cloudradar-monitoring/rport/server/sessions"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

const (
	DefaultCSRFileName          = "csr.json"
	DefaultCacheClientsInterval = 1 * time.Second
	DefaultCleanClientsInterval = 5 * time.Minute
	DefaultMaxRequestBytes      = 2 * 1024 // 2 KB
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

    --authfile, An optional path to a json file with clients credentials.
    This is for authentication of the rport tunnel clients.
    The file should contain a map with clients credentials defined like:
      {
        "<client1-id>": "<password1>"
        "<client2-id>": "<password2>"
      }

    --auth, An optional string representing a single client with full
    access, in the form of <client-id>:<password>. This is equivalent to creating an
    authfile with {"<client-id>":"<password>"}.

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
    [{
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
    By default, 5 minutes is used. It can contain "h"(hours), "m"(minutes), "s"(seconds).

    --csr-file-name, An optional arg to define a file name in --data-dir directory to store
    info about active and disconnected clients. By default, "csr.json" is used.

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
	ctx := context.Background()

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

	var keepLostClients *time.Duration
	if cfg.KeepLostClients > 0 {
		keepLostClients = &cfg.KeepLostClients
	}
	initSessions, err := sessions.GetInitStateFromFile(cfg.CSRFilePath(), keepLostClients)
	if err != nil {
		if len(initSessions) == 0 {
			log.Printf("Failed to get init CSR state from file %q: %v\n", cfg.CSRFilePath(), err)
		} else {
			log.Printf("Partial failure. Successfully read %d sessions from file %q. Error: %v\n", len(initSessions), cfg.CSRFilePath(), err)
		}
		// proceed further
	}
	repo := sessions.NewSessionRepository(initSessions, keepLostClients)

	s, err := chserver.NewServer(cfg, repo)
	if err != nil {
		log.Fatal(err)
	}

	s.Infof("data directory path: %q", cfg.DataDir)
	// create --data-dir path if not exist
	if makedirErr := os.MkdirAll(cfg.DataDir, os.ModePerm); makedirErr != nil {
		log.Printf("ERROR: failed to create --data-dir %q: %v\n", cfg.DataDir, makedirErr)
	}

	go chshare.GoStats()
	if keepLostClients != nil {
		s.Infof("Variable to keep lost clients is set. Enables keeping disconnected clients for period: %v", cfg.KeepLostClients)
		s.Infof("csr file path: %q", cfg.CSRFilePath())

		go scheduler.Run(ctx, s.Logger, sessions.NewCleanupTask(s.Logger, repo), cfg.CleanupClients)
		s.Infof("Task to cleanup obsolete clients will run with interval %v", cfg.CleanupClients)
		// TODO(m-terel): add graceful shutdown of background task
		go scheduler.Run(ctx, s.Logger, sessions.NewSaveToFileTask(s.Logger, repo, cfg.CSRFilePath()), cfg.SaveClients)
		s.Infof("Task to save clients to disk will run with interval %v", cfg.SaveClients)
	}

	if err = s.Run(); err != nil {
		log.Fatal(err)
	}
}
