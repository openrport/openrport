# rport
Create reverse tunnels with ease.

## At a glance
Rport helps you to manage your remote servers without the hassle of VPNs, chained SSH connections, jump-hosts, or the use of commercial tools like TeamViewer and its clones. 

Rport acts as server and client establishing permanent or on-demand secure tunnels to devices inside protected intranets behind a firewall. 

All operating systems provide secure and well-established mechanisms for remote management, being SSH and Remote Desktop the most widely used. Rport makes them accessible easily and securely. 

**Is Rport a replacement for TeamViewer?**
Yes and no. It depends on your needs.
TeamViewer and a couple of similar products are focused on giving access to a remote graphical desktop bypassing the Remote Desktop implementation of Microsoft. They fall short in a heterogeneous environment where access to headless Linux machines is needed. But they are without alternatives for Windows Home Editions.
Apart from remote management, they offer supplementary services like Video Conferences, desktop sharing, screen mirroring, or spontaneous remote assistance for desktop users.

**Goal of Rport**
Rport focuses only on remote management of those operating systems where an existing login mechanism can be used. It can be used for Linux and Windows, but also appliances and IoT devices providing a web-based configuration. 
From a technological perspective, [Ngrok](https://ngrok.com/) and [openport.io](https://openport.io) are similar products. Rport differs from them in many aspects.
* Rport is 100% open source. Client and Server. Remote management is a matter of trust and security. Rport is fully transparent.
* Rport will come with a user interface making the management of remote systems easy and user-friendly.
* Rport is made for all operating systems with native and small binaries. No need for Python or similar heavyweights.
* Rport allows you to self-host the server.
* Rport allows clients to wait in standby mode without an active tunnel. Tunnels can be requested on-demand by the user remotely.


## Build and installation
We provide [pre-compiled binaries](https://github.com/cloudradar-monitoring/rport/releases).
### From source
1) Build from source (Linux or Mac OS/X):
    ```bash
    make all
    ```
    `rport` and `rportd` binaries will appear in directory.  

2) Build using Docker:
    ```bash
    make docker-goreleaser
    ```
    will create binaries for all supported platforms in `./dist` directory.

## Usage
`rportd` should be executed on the machine, acting as a server.

`rport` is a client app which will try to establish long-running connection to the server.

Minimal setup:
1) Execute `./rportd --addr 0.0.0.0:9999` on a server.
1) Execute `./rport <SERVER_IP>:9999 3389:3389` on a client or `./rport <SERVER_IP>:9999 3389` and the server tunnel port will be randomly chosen for you.
1) Now end-users can connect to `<SERVER_IP>:3389` (e.g. using Remote Desktop Connection). The connection will be proxied to client machine.

See `./rportd --help` and `./rport --help` for more options, like:
- Specifying certificate fingerprint to validate server authority
- Client session authentication using user:password pair
- Restricting, which users can connect
- Specifying additional intermediate HTTP proxy
- Using POSIX signals to control running apps
- Setting custom HTTP headers
- Using IPv6 addresses when starting a server

## Quickstart guide
### Install and run the rport server
On a machine connected to the public internet and ideally with an FQDN registered to a public DNS install and run the server.
The server is called node1.example.com in this example.

Install client and server
```
curl -LSs https://github.com/cloudradar-monitoring/rport/releases/download/0.1.17/rport_0.1.17_Linux_x86_64.tar.gz|\
tar vxzf - -C /usr/local/bin/
```

Create a key for the server instance. Store this key and don't change it. Otherwise, your fingerprint will change and your clients might be rejected. 
```
openssl rand -hex 18
```  

Start the server as a background task.
```
nohup rportd --key <YOUR_KEY> --addr 0.0.0.0:19075 &>/tmp/rportd.log &
```
For the first testing leave the console open and observe the log with `tail -f /tmp/rportd.log`. Note the fingerprint. You will use it later. 

To safely store and reuse the key use these commands.
```
echo "RPORT_KEY=$(openssl rand -hex 18)">/etc/default/rport
. /etc/default/rport
export RPORT_KEY=$RPORT_KEY
nohup rportd --addr 0.0.0.0:19075 &>/tmp/rportd.log &
```
rportd reads the key from the environment so it does not appear in the process list or the history. 

### Connect a client
We call the client `client1.local.localdomain`.
On your client just install the client binary 
```
curl -LSs https://github.com/cloudradar-monitoring/rport/releases/download/0.1.17/rport_0.1.17_Linux_x86_64.tar.gz|\
tar vxzf - rport -C /usr/local/bin/
```

Create an ad hoc tunnel that will forward the port 2222 of node1.example.com to the to local port 22 of client1.local.localdomain.
`rport node1.example.com:19075 2222:0.0.0.0:22`
Observing the log of the server you get a confirmation about the newly created tunnel.

Now you can access your machine behind a firewall through the tunnel. Try `ssh -p 2222 node1.example.com` and you will come out on the machine where the tunnel has been initiated.

#### Let's improve security by using fingerprints
Copy the fingerprint the server has generated on startup to your clipboard and use it on the client like this 
`rport --fingerprint <YOUR_FINGERPRINT> node1.example.com:19075 2222:0.0.0.0:22`.

This ensures you connect only to trusted servers. If you omit this step a man in the middle can bring up a rport server and hijack your tunnels.
If you do ssh or rdp through the tunnel, a hijacked tunnel will not expose your credentials because the data inside the tunnel is still encrypted. But if you use rport for unencrypted protocols like HTTP, sniffing credentials would be possible.

### Config files
Config files can be used to set up both the rport server and clients. In order to use it an arg `--config`(or `-c`) should be passed to a command with a path to the file. Config examples `rportd.example.conf` and `rport.example.conf` can be found in the release archive or in the source.

NOTE: command arguments and env variables will override values from the config file.

#### Rport server
In order to load configuration from the file run:
```
rportd -c /var/lib/rportd/rportd.conf
```
Here is an example of a config file for the server:
```
# Specifies log file path
log_file = "/tmp/rportd.log"

# Specify log level. Values: 'error', 'info', 'debug'.
# Defaults to 'error'
log_level = "info"

# Defines the IP address and port the HTTP server listens on.
address = "0.0.0.0:7777"

# Defines full client connect URL. Defaults to http://{addr}
url = "http://example.com"

# An optional string to seed the generation of a ECDSA public
# and private key pair.
# key_seed = "youknownothing1"

# An optional path to a json file with clients credentials.
# This is for authentication of the rport tunnel clients.
# The file should contain a map with clients credentials defined like:
# {
#   "<client1-id>": "<password1>"
#   "<client2-id>": "<password2>"
# }
# auth_file = "/etc/rportd/users.json"

# An optional string representing a single client with full access, in the form of <client-id>:<password>.
# This is equivalent to creating an [auth_file] with {"<client-id>":"<password>"}.
auth = "user1:1234"

# Specifies another HTTP server to proxy requests to when
# rportd receives a normal HTTP request
# proxy = "http://intranet.lan:8080/"

# Defines a list of port numbers or ranges of server ports,
# that would not be used for automatic port assignment.
# Defaults to ['1-1000'].
excluded_ports = [
  '1-1000',
  '8888',
  '8080'
]

# An optional param to define a local directory path to store internal data.
# By default, "/var/lib/rportd" is used on Linux, "C:\ProgramData\rportd" is used on Windows.
#data_dir = "/var/lib/rportd"

# An optional param to define a duration to keep info (sessions, tunnels, etc) about active and disconnected clients.
# Enables to identify disconnected clients at server restart and to reestablish previous tunnels on reconnect.
# By default is "0"(is disabled). It can contain "h"(hours), "m"(minutes), "s"(seconds).
#keep_lost_clients = "1h"

# Applicable only if [keep_lost_clients] is specified. An optional param to define an interval
# to flush info (sessions, tunnels, etc) about active and disconnected clients to disk.
# By default, 1 second is used. It can contain "h"(hours), "m"(minutes), "s"(seconds).
# save_clients_interval = "5s"

# Applicable only if [keep_lost_clients] is specified. An optional param to define an
# interval to clean up internal storage from obsolete disconnected clients.
# By default, 5 minutes is used. It can contain "h"(hours), "m"(minutes), "s"(seconds).
# cleanup-clients-interval = "3m"

# An optional param to define a file name in [data_dir] directory to store info about
# active and disconnected clients. By default, "csr.json" is used.
#csr_file_name = "csr.json"

# An optional param to define a limit for data that can be sent by rport clients and API requests.
# By default is set to 2048(2Kb).
max_request_bytes = 2048

# specify non-empty api.address to enable API support
[api]
  # Defines the IP address and port the API server listens on
  address = "0.0.0.0:9999"

  # An optional path to a json file with users credentials for API authentication.
  # The file should contain an array with users credentials defined like:
  # [
  # {
  #   "username": "<username>",
  #   "password": "<password>",
  #   "groups": ["<group>", "<group>"]
  # },
  # {
  #   "username": "admin",
  #   "password": "$2y$05$cIOk1IlsdgdUeZpV464d6OXKI1tF2Yc3MWo55xDu4XhopEJmGb2KC",
  #   "groups": ["admins", "sysadmins", "users", "sudo", "some-group"]
  # }
  # ]
  # Use either [api.auth_file] or [api.auth]. Not both.
  # If both are enabled, [api.auth_file] has precedence and [api.auth] is ignored.
  # If both are disabled, API authentication is turned off.
  # The passwords are expected to be bcrypt hashed and they must be compatible with
  # the passwords generated by: htpasswd -bnBC 10 "" password | tr -d ':'.
  # auth_file = "/etc/rportd/api-auth.json"

  # An optional string representing a single pair of user credentials for accessing API,
  # in the form of "<user>:<password>". Use either [api.auth_file] or [api.auth]. Not both.
  # If both are enabled, [api.auth_file] has precedence and [api.auth] is ignored.
  # If both are disabled, API authentication is turned off.
  auth = "admin:1234"

  # Defines JWT secret used to generate new tokens
  # jwt_secret = "foobar2"

  # If specified, rportd will serve files from this directory on the same API address
  # doc_root = "/var/rportd/www"
```

#### Rport client
In order to load configuration from the file run:
```
rport -c /var/tmp/rport.conf
```
Here is an example of a config file for the client:
```
# Specifies log file path
log_file = "/tmp/rport.log"

# Specify log level. Values: 'error', 'info', 'debug'.
# Defaults to 'error'
log_level = "info"

# rportd server address
server = "rportd.lan:9000"

# fingerprint string to perform host-key validation against the server's public key
# fingerprint = ""

# An optional username and password (client authentication) in the form: "<user>:<pass>"
auth = "user1:1234"

# An optional HTTP CONNECT or SOCKS5 proxy which will be used to reach the rport server.
# Authentication can be specified inside the URL
proxy = "http://admin:password@my-server.com:8081"

# client ID
id = "client id #1"

# client name
name = "my_win_vm_1"

# client tags
tags = ['win', 'server', 'vm']

# remote connections tunneled through the server, each of which come in the form:
#   <local-interface>:<local-port>:<remote-host>:<remote-port>
#   or
#   <remote-host>:<remote-port>
# sharing <remote-host>:<remote-port> from the client to the server's <local-interface>:<local-port>
remotes = [
  '3389:3389',
  '5050'
]

[connection]
  # An optional keepalive interval. You must specify a time with a unit, for example '30s' or '2m'.
  # Defaults to '0s' (disabled)
  keep_alive = '30s'

  # Maximum number of times to retry before exiting. Defaults to unlimited (-1)
  max_retry_count = 10

  # Maximum wait time before retrying after a disconnection. Defaults to 5 minutes
  max_retry_interval = '5m'

  # Optionally set the 'Host' header (defaults to the host found in the server url)
  hostname = "myvm1.lan"

  # Other custom headers in the form "HeaderName: HeaderContent"
  headers = ['User-Agent: test1', 'Authorization: Basic XXXXXX']
```

### Run the server with systemd
Packages for most common distributions and Windows are on our roadmap. In the meantime create a systemd service file in `/etc/systemd/system/rportd.service` with the following lines manually.
``` 
[Unit]
Description=Rport Server Daemon
After=network-online.target
Wants=network-online.target systemd-networkd-wait-online.service

[Service]
User=rport
Group=rport
WorkingDirectory=/var/lib/rport/
EnvironmentFile=/etc/default/rportd
ExecStart=/usr/local/bin/rportd
Restart=on-failure
RestartSec=5
StandardOutput=file:/var/log/rportd/rportd.log
StandardError=file:/var/log/rportd/rportd.log

[Install]
WantedBy=multi-user.target
```

Create a user because rport has no requirement to run as root
```
useradd -m -r -s /bin/false -d /var/lib/rport rport
mkdir /var/log/rportd
chown rport:root /var/log/rportd
```

Create a config file `/etc/default/rport` like this example.
```
# Key to generate the fingerprint
RPORT_KEY=<YOUR_KEY>
# Listen for rport clients connections
RPORT_ADDR=0.0.0.0:19075
```

Start it and enable the auto-start on boot
```
systemctl daemon-reload
systemctl start rportd
systemctl enable rportd
```

### Using authentication
Anyone who knows the address and the port of your rport server can use it for tunneling. In most cases, this is not desired. Your rport server could be abused for example to publish content under your IP address. Therefore using rport with authentication is highly recommended.

For the server `rportd --auth rport:password123` is the most basic option. All clients must use the username `rport` and the given password. 

On the client start the tunnel this way
`rport --auth rport:password123 --fingerprint <YOUR_FINGERPRINT> node1.example.com:19075 2222:0.0.0.0:22`
*Note that in this early version the order of the command line options is still important. This might change later.*

If you want to maintain multiple users with different passwords, create a json-file `/etc/rportd-auth.json` with credentials, for example
```
{
    "user1": "foobaz",
    "user2": "bingo",
    "rport": "password123"
}
```
*For now, rportd reads only the user and password.*
*Rportd re-reads the file immediately after writing without the need for a sighup. This might change in the future.*

Start the server with `rportd --authfile /etc/rport-auth.json`. Change the `ExecStart` line of the systemd service file accordingly. 

### On-demand tunnels using the API
Initializing the creation of a tunnel from the client is nice but not a perfect solution for secure and reliable remote access to a large number of machines.
Most of the time the tunnel wouldn't be used. Network resources would be wasted and a port is exposed to the internet for an unnecessarily long time.
Rport provides the option to establish tunnels from the server only when you need them.

#### Step 1: activate the API
The internal management API is disabled by default. It can be activated in one of two ways:
1) Use a config file that is described in ["Config files"](https://github.com/cloudradar-monitoring/rport#rport-server) section. Set up `[api]` config params. For example:
   ```
   # specify non-empty api.address to enable API support
   [api]
     # Defines the IP address and port the API server listens on
     address = "127.0.0.1:3000"

     # An optional path to a json file with users credentials for API authentication.
     # The file should contain an array with users credentials defined like:
     # [
     # {
     #   "username": "<username>",
     #   "password": "<password>",
     #   "groups": ["<group>", "<group>"]
     # },
     # {
     #   "username": "admin",
     #   "password": "$2y$05$cIOk1IlsdgdUeZpV464d6OXKI1tF2Yc3MWo55xDu4XhopEJmGb2KC",
     #   "groups": ["admins", "sysadmins", "users", "sudo", "some-group"]
     # }
     # ]
     # Use either [api.auth_file] or [api.auth]. Not both.
     # If both are enabled, [api.auth_file] has precedence and [api.auth] is ignored.
     # If both are disabled, API authentication is turned off.
     # The passwords are expected to be bcrypt hashed and they must be compatible with
     # the passwords generated by: htpasswd -bnBC 10 "" password | tr -d ':'.
     # auth_file = "/etc/rportd/api-auth.json"
   
     # An optional string representing a single pair of user credentials for accessing API,
     # in the form of "<user>:<password>". Use either [api.auth_file] or [api.auth]. Not both.
     # If both are enabled, [api.auth_file] has precedence and [api.auth] is ignored.
     # If both are disabled, API authentication is turned off.
     auth = "admin:foobaz"
   ```
2) Extend your rportd configuration file `/etc/default/rport` like this example: 
   ```
   RPORT_KEY=<YOUR_KEY>
   # Listen for rport clients connections
   RPORT_ADDR=0.0.0.0:19075
   
   # Open the management API
   RPORT_API_ADDR=127.0.0.1:3000
   RPORT_API_AUTH=admin:foobaz
   ```
This opens the API and enables HTTP basic authentication with a single user "admin:foobaz" who has access to the API. To enable access to multiple users and to mange them in the file use "api.auth_file" config param (or "--api-authfile" rportd command arg). Restart the rportd after any changes to the configuration.

#### Step 2: Connect a client
Invoke the client without specifying a tunnel but with some extra data.  
```
rport --id 2ba9174e-640e-4694-ad35-34a2d6f3986b \
  --fingerprint c5:26:2b:65:29:a8:0f:ed:ef:77:c9:5c:f1:2a:36:8a \
  --tag Linux --tag "Office Berlin" \
  --name "My Test VM" --auth user1:Aiphei4d \
  node1.example.com:19075
```
*Add auth and fingerprint as already explained.*

This attaches the client to the message queue of the server without creating a tunnel.

#### Step 3: Manage clients and tunnels
On the server, you can supervise the attached clients using 
`curl -s -u admin:foobaz http://localhost:3000/api/v1/sessions`. *Use `jq` for pretty-printing json.*
Here is an example:
```
curl -s -u admin:foobaz http://localhost:3000/api/v1/sessions|jq
[
  {
    "id": "2ba9174e-640e-4694-ad35-34a2d6f3986b",
    "name": "My Test VM",
    "os": "Linux my-devvm-v3 5.4.0-37-generic #41-Ubuntu SMP Wed Jun 3 18:57:02 UTC 2020 x86_64 x86_64 x86_64 GNU/Linux",
    "hostname": "my-devvm-v3",
    "ipv4": [
      "192.168.3.148"
    ],
    "ipv6": [
      "fe80::20c:29ff:fec8:b1f"
    ],
    "tags": [
      "Linux",
      "Office Berlin"
    ],
    "version": "0.1.6",
    "address": "87.123.136.***:63552",
    "tunnels": []
  },
   {
    "id": "aa1210c7-1899-491e-8e71-564cacaf1df8",
    "name": "Random Rport Client",
    "os": "Linux alpine-3-10-tk-01 4.19.80-0-virt #1-Alpine SMP Fri Oct 18 11:51:24 UTC 2019 x86_64 Linux",
    "hostname": "alpine-3-10-tk-01",
    "ipv4": [
      "192.168.122.117"
    ],
    "ipv6": [
      "fe80::b84f:aff:fe59:a0ba"
    ],
    "tags": [
      "Linux",
      "Datacenter 1"
    ],
    "version": "0.1.6",
    "address": "88.198.189.***:43206",
    "tunnels": [
      {
        "lhost": "0.0.0.0",
        "lport": "2222",
        "rhost": "0.0.0.0",
        "rport": "22",
        "id": "1"
      }
    ]
  }
]
```
There is one client connected with an active tunnel. The second client is in standby mode.

Now use `PUT /api/v1/sessions/{id}/tunnels?local={port}&remote={port}` to request a new tunnel for a client session.
For example
```
CLIENTID=2ba9174e-640e-4694-ad35-34a2d6f3986b
LOCAL_PORT=4000 
REMOTE_PORT=22:80
curl -u admin:foobaz -X PUT "http://localhost:3000/api/v1/sessions/$CLIENTID/tunnels?local=$LOCAL_PORT&remote=$REMOTE_PORT"
```
The ports are defined from the servers' perspective. The above example opens port 4000 on the rport server and forwards to the port 22 of the client.

Using `curl -u admin:foobaz -s http://localhost:3000/api/v1/sessions|jq` again confirms the tunnel has been established.
```
"tunnels": [
      {
        "lhost": "0.0.0.0",
        "lport": "4000",
        "rhost": "0.0.0.0",
        "rport": "22",
        "id": "1"
      }
    ]
```

The rport client is not limited to establish tunnels only to the system it runs on. You can use it as a jump host to create tunnels to foreign systems too.

```
CLIENTID=2ba9174e-640e-4694-ad35-34a2d6f3986b
LOCAL_PORT=4001
REMOTE_PORT=192.168.178.1:80
curl -u admin:foobaz -X PUT "http://localhost:3000/api/v1/sessions/$CLIENTID/tunnels?local=$LOCAL_PORT&remote=$REMOTE_PORT"
```
This example forwards port 4001 of the rport server to port 80 of 192.168.178.1 using the rport client in the middle. 
```
"tunnels": [
      {
        "lhost": "0.0.0.0",
        "lport": "4001",
        "rhost": "192.168.178.1",
        "rport": "80",
        "id": "1"
      }
    ]
```

Using a DELETE request with the tunnel id allows terminating a tunnel.

```
CLIENTID=2ba9174e-640e-4694-ad35-34a2d6f3986b
TUNNELID=1 
curl -u admin:foobaz -X DELETE "http://localhost:3000/api/v1/sessions/$CLIENTID/tunnels/$TUNNELID"
```

#### Limitations
The API is very basic still. A UI and many more options will follow soon. Stay connected with us.

Tunnels initiated by the client survive a restart of the server. Tunnels created by the server don't. This will change in the future. 


### Versioning model
rport uses `<major>.<minor>.<buildnumber>` version pattern for compatibility with a maximum number of package managers.

Starting from version 1.0.0 packages with even <minor> number are considered stable.


### Credits
Forked from [jpillora/chisel](https://github.com/jpillora/chisel)
