package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	chserver "github.com/cloudradar-monitoring/rport/server"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

var serverHelp = `
  Usage: rportd [options]

  Options:

    --listen, Defines the IP address the HTTP server listens on.
    (defaults to the environment variable RPORT_LISTEN and falls back to 0.0.0.0).

    --port, -p, Defines the HTTP listening port (defaults to the environment
    variable RPORT_PORT and fallsback to port 8080).

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

    -v, Enable verbose logging

    --help, This help text

    --version, Print version info and exit

  Signals:
    The rportd process is listening for SIGUSR2 to print process stats

`

func main() {
	listenInterface := flag.String("listen", "", "")
	p := flag.String("p", "", "")
	port := flag.String("port", "", "")
	key := flag.String("key", "", "")
	authfile := flag.String("authfile", "", "")
	auth := flag.String("auth", "", "")
	proxy := flag.String("proxy", "", "")
	apiAddr := flag.String("api-addr", "", "")
	verbose := flag.Bool("v", false, "")
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

	if *listenInterface == "" {
		*listenInterface = os.Getenv("RPORT_LISTEN")
	}
	if *listenInterface == "" {
		*listenInterface = "0.0.0.0"
	}
	if *port == "" {
		*port = *p
	}
	if *port == "" {
		*port = os.Getenv("RPORT_PORT")
	}
	if *port == "" {
		*port = "8080"
	}
	if *key == "" {
		*key = os.Getenv("RPORT_KEY")
	}

	if *apiAddr == "" {
		*apiAddr = os.Getenv("RPORT_API_ADDR")
	}

	s, err := chserver.NewServer(&chserver.Config{
		KeySeed:  *key,
		AuthFile: *authfile,
		Auth:     *auth,
		Proxy:    *proxy,
		Verbose:  *verbose,
	})
	if err != nil {
		log.Fatal(err)
	}

	go chshare.GoStats()

	if err = s.Run(*listenInterface+":"+*port, *apiAddr); err != nil {
		log.Fatal(err)
	}
}
