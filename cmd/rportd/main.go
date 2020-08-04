package main

import (
	"crypto/rand"
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"os"

	chserver "github.com/cloudradar-monitoring/rport/server"
	chshare "github.com/cloudradar-monitoring/rport/share"
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
    rport receives a normal HTTP request. Useful for hiding rport in
    plain sight.

    --api-addr, Defines the IP address and port the API server listens on.
    e.g. "0.0.0.0:7777". (defaults to the environment variable RPORT_API_ADDR
    and fallsback to empty string: API not available)

    --doc-root, Specifies local directory path. If specified, rportd will serve
    files from this directory on the same API address (--api-addr).

    --api-auth, Defines "user:password"" authentication pair for accessing API
    e.g. "admin:1234". (defaults to the environment variable RPORT_API_AUTH
    and fallsback to empty string: authorization not required).

    --api-jwt-secret, Defines JWT secret used to generate new tokens.
    (defaults to the environment variable RPORT_AUTH_JWT_SECRET and fallsback
    to auto-generated value).

    -v, Specify log level. Values: "error", "info", "debug" (defaults to "error")

    -l, Specifies log file path. (defaults to empty string: log printed to stdout)

    --help, This help text

    --version, Print version info and exit

  Signals:
    The rportd process is listening for SIGUSR2 to print process stats

`

func main() {
	a := flag.String("a", "", "")
	listenAddr := flag.String("addr", "", "")
	key := flag.String("key", "", "")
	authfile := flag.String("authfile", "", "")
	auth := flag.String("auth", "", "")
	proxy := flag.String("proxy", "", "")
	apiAddr := flag.String("api-addr", "", "")
	apiAuth := flag.String("api-auth", "", "")
	apiJWTSecret := flag.String("api-jwt-secret", "", "")
	docRoot := flag.String("doc-root", "", "")
	logLevelStr := flag.String("v", "error", "")
	logFilePath := flag.String("l", "", "")
	version := flag.Bool("version", false, "")

	flag.Usage = func() {
		fmt.Print(serverHelp)
		os.Exit(1)
	}
	flag.Parse()

	if *version {
		fmt.Println(chshare.BuildVersion)
		os.Exit(1)
	}

	if flag.NArg() > 0 {
		fmt.Println("Unsupported command or argument.")
		fmt.Print(serverHelp)
		os.Exit(1)
	}

	if *listenAddr == "" {
		*listenAddr = *a
	}
	if *listenAddr == "" {
		*listenAddr = os.Getenv("RPORT_ADDR")
	}
	if *listenAddr == "" {
		*listenAddr = "0.0.0.0:8080"
	}
	if *key == "" {
		*key = os.Getenv("RPORT_KEY")
	}
	if *apiAddr == "" {
		*apiAddr = os.Getenv("RPORT_API_ADDR")
	}
	if *apiAuth == "" {
		*apiAuth = os.Getenv("RPORT_API_AUTH")
	}
	if *apiJWTSecret == "" {
		*apiJWTSecret = os.Getenv("RPORT_API_JWT_SECRET")
	}
	if *apiJWTSecret == "" {
		*apiJWTSecret = generateJWTSecret()
	}

	if *docRoot != "" && *apiAddr == "" {
		log.Fatal("To use --doc-root you need to specify API address (see --api-addr)")
	}

	config := &chserver.Config{
		KeySeed:      *key,
		AuthFile:     *authfile,
		Auth:         *auth,
		Proxy:        *proxy,
		APIAuth:      *apiAuth,
		APIJWTSecret: *apiJWTSecret,
		DocRoot:      *docRoot,
		LogOutput:    os.Stdout,
		LogLevel:     tryParseLogLevel(*logLevelStr),
	}

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

	s, err := chserver.NewServer(config)
	if err != nil {
		log.Fatal(err)
	}

	go chshare.GoStats()

	if err = s.Run(*listenAddr, *apiAddr); err != nil {
		log.Fatal(err)
	}
}

func generateJWTSecret() string {
	data := make([]byte, 10)
	if _, err := rand.Read(data); err != nil {
		log.Fatalf("can't generate API JWT secret: %s", err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
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
