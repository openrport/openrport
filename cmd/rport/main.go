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

    --auth, An optional username and password (client authentication) in the form: "<user>:<password>".
    Highly recommended. Required if client authentication in enabled on the rport server.

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
    If not set, a random id will be crated that changes on every client start.
    The server rejects connections on duplicated ids.

    --name, An optional client name to better identify the client.
    Useful if you use numeric ids to make client identification easier.
    For example, --name "my_win_vm_1"
    Defaults to unset.

    --tag, -t, Optional values to give your clients attributes.
    Used for filtering clients on the server.
    Can be used multiple times. (e.g --tag "foobaz" --tag "bingo")

    --verbose, -v, Specify log level. Values: "error", "info", "debug" (defaults to "error")

    --log-file, -l, Specifies log file path. (defaults to empty string: log printed to stdout)

    --config, -c, An optional arg to define a path to a config file. If it is set then
    configuration will be loaded from the file. Note: command arguments and env variables will override them.
    Config file should be in TOML format. You can find an example "rport.example.conf" in the release archive.

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

	viperCfg.SetDefault("logging.log_level", "error")
	viperCfg.SetDefault("connection.max_retry_count", -1)

	// map config fields to CLI args:
	_ = viperCfg.BindPFlag("client.fingerprint", pFlags.Lookup("fingerprint"))
	_ = viperCfg.BindPFlag("client.auth", pFlags.Lookup("auth"))
	_ = viperCfg.BindPFlag("client.proxy", pFlags.Lookup("proxy"))
	_ = viperCfg.BindPFlag("client.id", pFlags.Lookup("id"))
	_ = viperCfg.BindPFlag("client.name", pFlags.Lookup("name"))
	_ = viperCfg.BindPFlag("client.tags", pFlags.Lookup("tag"))

	_ = viperCfg.BindPFlag("logging.log_file", pFlags.Lookup("log-file"))
	_ = viperCfg.BindPFlag("logging.log_level", pFlags.Lookup("verbose"))

	_ = viperCfg.BindPFlag("connection.keep_alive", pFlags.Lookup("keepalive"))
	_ = viperCfg.BindPFlag("connection.max_retry_count", pFlags.Lookup("max-retry-count"))
	_ = viperCfg.BindPFlag("connection.max_retry_interval", pFlags.Lookup("max-retry-interval"))
	_ = viperCfg.BindPFlag("connection.hostname", pFlags.Lookup("hostname"))
	_ = viperCfg.BindPFlag("connection.headers", pFlags.Lookup("header"))
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
		config.Client.Server = args[0]
		config.Client.Remotes = args[1:]
	}

	if config.Client.Server == "" {
		log.Fatalf("Server address is required. See --help")
	}

	err = config.ParseAndValidate()
	if err != nil {
		log.Fatal(err)
	}

	err = config.Logging.LogOutput.Start()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		config.Logging.LogOutput.Shutdown()
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
