package cliboilerplate

var ClientHelp = `
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

    ./rport --scheme http --enable-reverse-proxy <SERVER>:<PORT> 8080
    Makes the local port 8080 available via HTTPS on a random port of the server.

    ./rport -c /etc/rport/rport.conf
    starts client with configuration loaded from the file

  Options:
    NOTE: The order of options is important. <SERVER>:<PORT> and <REMOTES> aka the tunnels
    must be the last options on the command line.

    --fingerprint, A *strongly recommended* fingerprint string
    to perform host-key validation against the server's public key.
    You may provide just a prefix of the key or the entire string.
    Fingerprint mismatches will close the connection. Alternatively,
    export the fingerprint to the environment variable RPORT_FINGERPRINT.

    --auth, Required client authentication credentials in the form: "<client-auth-id>:<password>".
    Alternatively, export credentials to the environment variable RPORT_AUTH.

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

    --use-system-id, By default rport reads /etc/machine-id (Linux) or the ComputerSystemProduct UUID (Windows)
    to get a unique id for the client identification.
    NOTE: all history for a client is stored based on this id.
    --id, An optional hard-coded client ID to better identify the client.
    If not set, a random id will be created that changes on every client start.
    That's why it's highly recommended to set it with a value that was generated on the first
    start or just set it on the very beginning. So on client restart all his history will be preserved.
    The server rejects connections on duplicated ids.

    --use-hostname, By default rport reads the local hostname to identify the system in a human-readable way.
    --name, An optional client name to better identify the client.
    Useful if you use numeric ids to make client identification easier.
    For example, --name "my_win_vm_1"

    --tag, -t, Optional values to give your clients attributes.
    Used for filtering clients on the server.
    Can be used multiple times. (e.g --tag "foobaz" --tag "bingo")

    --allow-root, An optional arg to allow running rport as root. There is no technical requirement to run the rport
    client under the root user. Running it as root is an unnecessary security risk.

    --service, Manages rport running as a service. Possible commands are "install", "uninstall", "start" and "stop".
    The only arguments compatible with --service are --service-user and --config, others will be ignored.

    --service-user, An optional arg specifying user to run rport service under. Only on linux. Defaults to rport.

    --log-level, Specify log level. Values: "error", "info", "debug" (defaults to "info")

    --log-file, -l, Specifies log file path. (defaults to empty string: log printed to stdout)

    --remote-commands-enabled, Enable or disable remote commands.
    Defaults: true

    --remote-scripts-enabled, Enable or disable remote scripts.
    Defaults: false

    --data-dir, Temporary directory to store temp client data.
    Defaults: /var/lib/rport (unix) or C:\Program Files\rport (windows)

    --remote-commands-send-back-limit, Limit the maximum length of the command or script output that is sent back.
    Applies to the stdout and stderr separately. If exceeded the specified number of bytes are sent.
    Defaults: 2048

    --updates-interval, How often after the rport client has started pending updates are summarized.
    Defaults: 4h

    --fallback-server, Set fallback server(s) to which the client tries to connect if the main server is not reachable.

    --server-switchback-interval, If connected to fallback server, try every interval to switch back to the main server.
    Defaults: 2m

    --monitoring-enabled, Enable or disable gathering of monitoring data.
    Defaults: true

   --monitoring-interval, the interval time in seconds, when monitoring data is gathered
   Defaults: 60s

   --monitoring-fs-type-include, list of filesystem types to include in list of mountpoints
   --monitoring-fs-path-exclude, list of filesystem path to exclude from list of mountpoints
   --monitoring-fs-path-exclude-recurse, enable or disable recursive handling
   --monitoring-fs-identify-mountpoints-by-device, enable or disable the identification of mountpoints by device

   --monitoring-pm-enabled, enable or disable process-monitoring
   --monitoring-pm-kerneltasks-enabled, enable or disable monitoring of kerneltasks
   --monitoring-pm-max-number-processes, maximum number of processes in process monitoring list

   --monitoring-net-lan, enable monitoring of lan network card
   --monitoring-net-wan, enable monitoring of wan network card

    --scheme, Flag all <REMOTES> aka tunnels to be used by a URI scheme, for example http, rdp or vnc.

    --enable-reverse-proxy, Start one or more reverse proxies on top of the tunnel(s) to make them
    available via HTTPs with the server-side certificates. Requires '--scheme' to be http or https.
    Note: --scheme refers to the local protocol. The rport server will always use https for the proxy.

    --host-header, Inject a static header "host: " with the specified value when using --enable-reverse-proxy.
    By default the FQDN of the rport server is sent.

    --config, -c, An optional arg to define a path to a config file. If it is set then
    configuration will be loaded from the file. Note: command arguments and env variables will override them.
    MonitoringConfig file should be in TOML format. You can find an example "rport.example.conf" in the release archive.

    --help, This help text

    --version, Print version info and exit

   Environment Variables:
    RPORT_AUTH
    RPORT_FINGERPRINT

  Signals:
    The rport process is listening for:
      a SIGUSR2 to print process stats, and
      a SIGHUP to short-circuit the client reconnect timer

`
