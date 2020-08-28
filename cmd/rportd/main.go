package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"math"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set"

	"github.com/cloudradar-monitoring/rport/constant"
	chserver "github.com/cloudradar-monitoring/rport/server"
	"github.com/cloudradar-monitoring/rport/server/csr"
	"github.com/cloudradar-monitoring/rport/server/scheduler"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

const (
	DefaultCSRFileName          = "csr.json"
	DefaultKeepLostClients      = time.Hour
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
	ctx := context.Background()

	a := flag.String("a", "", "")
	listenAddr := flag.String("addr", "", "")
	url := flag.String("url", "", "")
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
	e := flag.String("e", "1-1000", "")
	excludePorts := flag.String("exclude-ports", "", "")
	dataDir := flag.String("clients-file", constant.DataDirectory, "")
	csrFileName := flag.String("csr-filename", DefaultCSRFileName, "")
	keepLostClients := flag.Duration("keep-lost-clients", DefaultKeepLostClients, "")
	saveClients := flag.Duration("save-clients", DefaultCacheClientsInterval, "")
	cleanupClients := flag.Duration("cleanup-clients", DefaultCleanClientsInterval, "")

	if dataDir == nil || *dataDir == "" {
		log.Fatal("--data-dir cannot be empty")
	}
	if err := os.MkdirAll(*dataDir, os.ModePerm); err != nil {
		log.Printf("ERROR: failed to create --data-dir %q: %v\n", *dataDir, err)
	}

	if csrFileName == nil || *csrFileName == "" {
		log.Fatal("--csr-filename cannot be empty")
	}
	csrFile := *dataDir + "/" + *csrFileName
	log.Printf("csr file: %q\n", csrFile)

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
	if *url == "" {
		*url = "http://" + *listenAddr
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

	if *excludePorts == "" {
		*excludePorts = *e
	}

	config := &chserver.Config{
		URL:           tryParseURL(*url),
		KeySeed:       *key,
		AuthFile:      *authfile,
		Auth:          *auth,
		Proxy:         *proxy,
		APIAuth:       *apiAuth,
		APIJWTSecret:  *apiJWTSecret,
		DocRoot:       *docRoot,
		LogOutput:     os.Stdout,
		LogLevel:      tryParseLogLevel(*logLevelStr),
		ExcludedPorts: tryParseExcludedPorts(*excludePorts),
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

	initSessions, err := csr.GetInitStateFromFile(csrFile, keepLostClients)
	if err != nil {
		log.Printf("Failed to get init CSR state from file: %v\n", err)
		// proceed further
	}
	repo := csr.NewSessionRepository(initSessions, keepLostClients)

	s, err := chserver.NewServer(config, repo)
	if err != nil {
		log.Fatal(err)
	}

	go chshare.GoStats()
	if keepLostClients != nil {
		go scheduler.Run(ctx, s.Logger, csr.NewCleanupTask(s.Logger, repo), *cleanupClients)
	}
	go scheduler.Run(ctx, s.Logger, csr.NewSaveToFileTask(s.Logger, repo, csrFile), *saveClients)

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

func tryParseExcludedPorts(val string) mapset.Set {
	result := mapset.NewThreadUnsafeSet()
	values := strings.Split(val, ",")
	for _, v := range values {
		rangeParts := strings.Split(v, "-")
		if len(rangeParts) == 1 {
			result.Add(tryParsePortNumber(rangeParts[0]))
		} else if len(rangeParts) == 2 {
			result = result.Union(tryParsePortNumberRange(rangeParts[0], rangeParts[1]))
		} else {
			log.Fatalf("can't parse exclude-ports parameter: incorrect range %s", v)
		}
	}
	return result
}

func tryParsePortNumber(portNumberStr string) int {
	num, err := strconv.Atoi(portNumberStr)
	if err != nil {
		log.Fatalf("can't parse port number %s: %s", portNumberStr, err)
	}
	if num < 0 || num > math.MaxUint16 {
		log.Fatalf("invalid port number: %d", num)
	}
	return num
}

func tryParsePortNumberRange(rangeStart, rangeEnd string) mapset.Set {
	start := tryParsePortNumber(rangeStart)
	end := tryParsePortNumber(rangeEnd)
	if start > end {
		log.Fatalf("invalid port range %s-%s", rangeStart, rangeEnd)
	}

	return chshare.SetFromRange(start, end)
}

func tryOpenLogFile(path string) *os.File {
	logFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("can't open log file: %s", err)
	}
	return logFile
}

func tryParseURL(urlRaw string) string {
	u, err := url.Parse(urlRaw)
	if err != nil {
		log.Fatalf("invalid --url: %s", err)
	}
	if u.Host == "" {
		log.Fatalf("invalid --url: must be absolute url")
	}
	return urlRaw
}
