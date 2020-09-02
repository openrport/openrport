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

    --authfile, An optional path to a users.json file. This file should
    be an object with users defined like:
      {
        "<user:pass>": ["<addr-regex>","<addr-regex>"]
      }
    when <user> connects, their <pass> will be verified and then
    each of the remote addresses will be compared against the list
    of address regular expressions for a match.

    --auth, An optional string representing a single user with full
    access, in the form of <user:pass>. This is equivalent to creating an
    authfile with {"<user:pass>": [""]}.

    --proxy, Specifies another HTTP server to proxy requests to when
    rportd receives a normal HTTP request. Useful for hiding rportd in
    plain sight.

    --api-addr, Defines the IP address and port the API server listens on.
    e.g. "0.0.0.0:7777". (defaults to the environment variable RPORT_API_ADDR
    and fallsback to empty string: API not available)

    --api-doc-root, Specifies local directory path. If specified, rportd will serve
    files from this directory on the same API address (--api-addr).

    --api-auth, Defines <user:password> authentication pair for accessing API
    e.g. "admin:1234". (defaults to the environment variable RPORT_API_AUTH
    and fallsback to empty string: authorization not required).

    --data-dir, Defines a local directory path to store internal data.
    By default, "/var/lib/rportd" is used on Linux, "C:\ProgramData\rportd" is used on Windows.

    --keep-lost-clients, Defines a duration to keep disconnected clients. For example,
    "--keep-lost-clients=1h30m". It can contain "h"(hours), "m"(minutes), "s"(seconds).

    --save-clients-interval, Only valid if --keep-lost-clients is specified. Defines an
    interval to flush info about active and disconnected clients to disk. By default,
    1 second is used.  It can contain "h"(hours), "m"(minutes), "s"(seconds).

    --cleanup-clients-interval, Only valid if --keep-lost-clients is specified. Defines an
    interval to clean up internal storage from obsolete disconnected clients. By default,
    5 minutes is used.  It can contain "h"(hours), "m"(minutes), "s"(seconds).

    --csr-file-name, Defines a file name in --data-dir directory to store active and
    disconnected clients. By default, "csr.json" is used.

    --api-jwt-secret, Defines JWT secret used to generate new tokens.
    (defaults to the environment variable RPORT_API_JWT_SECRET and fallsback
    to auto-generated value).

    --verbose, -v, Specify log level. Values: "error", "info", "debug" (defaults to "error")

    --log-file, -l, Specifies log file path. (defaults to empty string: log printed to stdout)

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
	viperCfg.BindPFlag("log_file", pFlags.Lookup("log-file"))
	viperCfg.BindPFlag("log_level", pFlags.Lookup("verbose"))
	viperCfg.BindPFlag("address", pFlags.Lookup("addr"))
	viperCfg.BindPFlag("url", pFlags.Lookup("url"))
	viperCfg.BindPFlag("key_seed", pFlags.Lookup("key"))
	viperCfg.BindPFlag("auth_file", pFlags.Lookup("authfile"))
	viperCfg.BindPFlag("auth", pFlags.Lookup("auth"))
	viperCfg.BindPFlag("proxy", pFlags.Lookup("proxy"))
	viperCfg.BindPFlag("api.address", pFlags.Lookup("api-addr"))
	viperCfg.BindPFlag("api.auth", pFlags.Lookup("api-auth"))
	viperCfg.BindPFlag("api.jwt_secret", pFlags.Lookup("api-jwt-secret"))
	viperCfg.BindPFlag("api.doc_root", pFlags.Lookup("api-doc-root"))
	viperCfg.BindPFlag("excluded_ports", pFlags.Lookup("exclude-ports"))
	viperCfg.BindPFlag("data_dir", pFlags.Lookup("data-dir"))
	viperCfg.BindPFlag("csr_file_name", pFlags.Lookup("csr-file-name"))
	viperCfg.BindPFlag("keep_lost_clients", pFlags.Lookup("keep-lost-clients"))
	viperCfg.BindPFlag("save_clients_interval", pFlags.Lookup("save-clients-interval"))
	viperCfg.BindPFlag("cleanup_clients_interval", pFlags.Lookup("cleanup-clients-interval"))

	// map ENV variables
	viperCfg.BindEnv("address", "RPORT_ADDR")
	viperCfg.BindEnv("url", "RPORT_URL")
	viperCfg.BindEnv("key_seed", "RPORT_KEY")
	viperCfg.BindEnv("api.address", "RPORT_API_ADDR")
	viperCfg.BindEnv("api.auth", "RPORT_API_AUTH")
	viperCfg.BindEnv("api.jwt_secret", "RPORT_JWT_SECRET")
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
		go scheduler.Run(ctx, s.Logger, sessions.NewSaveToFileTask(s.Logger, repo, cfg.CSRFilePath()), cfg.SaveClients)
		s.Infof("Task to save clients to disk will run with interval %v", cfg.SaveClients)
	}

	if err = s.Run(); err != nil {
		log.Fatal(err)
	}
}
