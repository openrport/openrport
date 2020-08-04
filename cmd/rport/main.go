package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	chclient "github.com/cloudradar-monitoring/rport/client"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

type headerFlags struct {
	http.Header
}

func (flag *headerFlags) String() string {
	out := ""
	for k, v := range flag.Header {
		out += fmt.Sprintf("%s: %s\n", k, v)
	}
	return out
}

func (flag *headerFlags) Set(arg string) error {
	index := strings.Index(arg, ":")
	if index < 0 {
		return fmt.Errorf(`Invalid header (%s). Should be in the format "HeaderName: HeaderContent"`, arg)
	}
	if flag.Header == nil {
		flag.Header = http.Header{}
	}
	key := arg[0:index]
	value := arg[index+1:]
	flag.Header.Set(key, strings.TrimSpace(value))
	return nil
}

var clientHelp = `
  Usage: rport [options] <server> [remote] [remote] [remote] ...

  <server> is the URL to the rport server.

  <remote>s are remote connections tunneled through the server, each of
  which come in the form:

    <local-interface>:<local-port>:<remote-host>:<remote-port>

  which does reverse port forwarding, sharing <remote-host>:<remote-port>
  from the client to the server's <local-interface>:<local-port>.
  Only IPv4 tunnels are supported.

  Examples:

    ./rport <SERVER>:<PORT> 2222:127.0.0.1:22 
    forwards port 2222 of the server to port 22 of the client

    ./rport <SERVER>:<PORT> 3000 
    forwards port 3000 of the server to port 3000 of the client

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
    using IPv6 server address. Forwards port 3389 of the server
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
    disconnection. Defaults to 5 minutes.

    --proxy, An optional HTTP CONNECT or SOCKS5 proxy which will be
    used to reach the rport server. Authentication can be specified
    inside the URL.
    For example, http://admin:password@my-server.com:8081
             or: socks://admin:password@my-server.com:1080

    --header, Set a custom header in the form "HeaderName: HeaderContent".
    Can be used multiple times. (e.g --header "Foo: Bar" --header "Hello: World")

    --hostname, Optionally set the 'Host' header (defaults to the host
    found in the server url).

    --id, Optionally set the 'ID' header (defaults to auto generated id).

    --name, Optionally set the 'Name' header (defaults to unset).

    --tag, Optionally set a tag.
    Can be used multiple times. (e.g --tag "foobaz" --tag "bingo")

    -v, Specify log level. Values: "error", "info", "debug" (defaults to "error")

    -l, Specifies log file path. (defaults to empty string: log printed to stdout)

    --help, This help text

    --version, Print version info and exit

  Signals:
    The rport process is listening for:
      a SIGUSR2 to print process stats, and
      a SIGHUP to short-circuit the client reconnect timer

`

func main() {
	config := chclient.Config{Headers: http.Header{}, LogOutput: os.Stdout}
	flag.StringVar(&config.Fingerprint, "fingerprint", "", "")
	flag.StringVar(&config.Auth, "auth", "", "")
	flag.DurationVar(&config.KeepAlive, "keepalive", 0, "")
	flag.IntVar(&config.MaxRetryCount, "max-retry-count", -1, "")
	flag.DurationVar(&config.MaxRetryInterval, "max-retry-interval", 0, "")
	flag.StringVar(&config.Proxy, "proxy", "", "")
	flag.Var(&headerFlags{config.Headers}, "header", "")
	flag.StringVar(&config.ID, "id", "", "")
	flag.StringVar(&config.Name, "name", "", "")
	flag.Var(&config.Tags, "tag", "")
	hostname := flag.String("hostname", "", "")
	logLevelStr := flag.String("v", "error", "")
	logFilePath := flag.String("l", "", "")
	version := flag.Bool("version", false, "")

	flag.Usage = func() {
		fmt.Print(clientHelp)
		os.Exit(1)
	}
	flag.Parse()

	if *version {
		fmt.Println(chshare.BuildVersion)
		os.Exit(1)
	}

	//pull out options, put back remaining args
	args := flag.Args()
	if len(args) < 1 {
		log.Fatalf("Server address is required. See --help")
	}
	config.Server = args[0]
	config.Remotes = args[1:]
	//default auth
	if config.Auth == "" {
		config.Auth = os.Getenv("AUTH")
	}
	//move hostname onto headers
	if *hostname != "" {
		config.Headers.Set("Host", *hostname)
	}

	config.LogLevel = tryParseLogLevel(*logLevelStr)

	var logFile *os.File
	if *logFilePath != "" {
		logFile = tryOpenLogFile(*logFilePath)
		config.LogOutput = logFile
	}
	defer func() {
		if logFile != nil {
			_ = logFile.Close()
		}
	}()

	//ready
	c, err := chclient.NewClient(&config)
	if err != nil {
		log.Fatal(err)
	}

	go chshare.GoStats()

	if err = c.Run(); err != nil {
		log.Fatal(err)
	}
}

func tryParseLogLevel(s string) chshare.LogLevel {
	var logLevel, err = chshare.ParseLogLevel(s)
	if err != nil {
		log.Fatalf("can't parse log level: %s", err)
	}
	return logLevel
}

func tryOpenLogFile(path string) *os.File {
	logFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("can't open log file: %s", err)
	}
	return logFile
}
