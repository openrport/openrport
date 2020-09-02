package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	chclient "github.com/cloudradar-monitoring/rport/client"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

var clientHelp = `
  Usage: rport [options] [<server> [remote] [remote] [remote] ...]

  <server> is the URL to the rport server.

  <remote>s are remote connections tunneled through the server, each of
  which come in the form:

    <local-interface>:<local-port>:<remote-host>:<remote-port>
    or
    <remote-host>:<remote-port>

  which does reverse port forwarding, sharing <remote-host>:<remote-port>
  from the client to the server's <local-interface>:<local-port>.
  If local part is omitted, a randomly chosen server port will be assigned. 
  Only IPv4 addresses are supported.

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

  Options:

    --fingerprint, A *strongly recommended* fingerprint string
    to perform host-key validation against the server's public key.
    You may provide just a prefix of the key or the entire string.
    Fingerprint mismatches will close the connection.

    --auth, An optional username and password (client authentication)
    in the form: "<user>:<pass>". These credentials are compared to
    the credentials inside the server's --authfile. defaults to the
    AUTH environment variable.

    --keepalive, An optional keepalive interval. Since the underlying
    transport is HTTP, in many instances we'll be traversing through
    proxies, often these proxies will close idle connections. You must
    specify a time with a unit, for example '30s' or '2m'. Defaults
    to '0s' (disabled).

    --max-retry-count, Maximum number of times to retry before exiting.
    Defaults to unlimited.

    --max-retry-interval, Maximum wait time before retrying after a
    disconnection. Defaults to 5 minutes ('5m').

    --proxy, An optional HTTP CONNECT or SOCKS5 proxy which will be
    used to reach the rport server. Authentication can be specified
    inside the URL.
    For example, http://admin:password@my-server.com:8081
             or: socks://admin:password@my-server.com:1080

    --header, Set a custom header in the form "HeaderName: HeaderContent".
    Can be used multiple times. (e.g --header "Foo: Bar" --header "Hello: World")

    --hostname, Optionally set the 'Host' header (defaults to the host
    found in the server url).

    --id, Optionally set the client 'ID' (defaults to auto generated id).

    --name, Optionally set the client 'Name' (defaults to unset).

    --tag, -t, Optionally set client tags.
    Can be used multiple times. (e.g --tag "foobaz" --tag "bingo")

    --verbose, -v, Specify log level. Values: "error", "info", "debug" (defaults to "error")

    --log-file, -l, Specifies log file path. (defaults to empty string: log printed to stdout)

    --help, This help text

    --version, Print version info and exit

  Signals:
    The rport process is listening for:
      a SIGUSR2 to print process stats, and
      a SIGHUP to short-circuit the client reconnect timer

`

var (
	RootCmd = &cobra.Command{
		Version: chshare.BuildVersion,
		Run:     runMain,
	}

	cfgPath  *string
	viperCfg *viper.Viper
	config   = &chclient.Config{}
)

func init() {
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
	pFlags.StringP("verbose", "v", "", "")

	cfgPath = pFlags.StringP("config", "c", "", "")

	RootCmd.SetUsageFunc(func(*cobra.Command) error {
		fmt.Print(clientHelp)
		os.Exit(1)
		return nil
	})

	viperCfg = viper.New()
	viperCfg.SetConfigType("toml")

	viperCfg.SetDefault("log_level", "error")
	viperCfg.SetDefault("connection.max_retry_count", -1)

	// map config fields to CLI args:
	_ = viperCfg.BindPFlag("log_file", pFlags.Lookup("log-file"))
	_ = viperCfg.BindPFlag("log_level", pFlags.Lookup("verbose"))
	_ = viperCfg.BindPFlag("fingerprint", pFlags.Lookup("fingerprint"))
	_ = viperCfg.BindPFlag("auth", pFlags.Lookup("auth"))
	_ = viperCfg.BindPFlag("connection.keep_alive", pFlags.Lookup("keepalive"))
	_ = viperCfg.BindPFlag("connection.max_retry_count", pFlags.Lookup("max-retry-count"))
	_ = viperCfg.BindPFlag("connection.max_retry_interval", pFlags.Lookup("max-retry-interval"))
	_ = viperCfg.BindPFlag("connection.headers", pFlags.Lookup("header"))
	_ = viperCfg.BindPFlag("proxy", pFlags.Lookup("proxy"))
	_ = viperCfg.BindPFlag("id", pFlags.Lookup("id"))
	_ = viperCfg.BindPFlag("name", pFlags.Lookup("name"))
	_ = viperCfg.BindPFlag("tags", pFlags.Lookup("tag"))
	_ = viperCfg.BindPFlag("hostname", pFlags.Lookup("hostname"))

	// map ENV variables
	_ = viperCfg.BindEnv("auth", "AUTH")
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
		viperCfg.SetConfigName("rport.conf")
	}

	return chshare.DecodeViperConfig(viperCfg, config)
}

func runMain(cmd *cobra.Command, args []string) {
	err := tryDecodeConfig()
	if err != nil {
		log.Fatal(err)
	}

	if len(args) > 0 {
		config.Server = args[0]
		config.Remotes = args[1:]
	}

	if config.Server == "" {
		log.Fatalf("Server address is required. See --help")
	}

	err = config.ParseAndValidate()
	if err != nil {
		log.Fatal(err)
	}

	err = config.LogOutput.Start()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		config.LogOutput.Shutdown()
	}()

	c, err := chclient.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	go chshare.GoStats()

	if err = c.Run(); err != nil {
		log.Fatal(err)
	}
}
