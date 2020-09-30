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
curl -LSs https://github.com/cloudradar-monitoring/rport/releases/download/0.1.19/rport_0.1.19_Linux_x86_64.tar.gz|\
tar vxzf - -C /usr/local/bin/
```

Create a key for the server instance. Store this key and don't change it. You will use it later. Otherwise, your fingerprint will change and your clients might be rejected. 
```
openssl rand -hex 18
```  

Start the server as a background task.
```
nohup rportd --key <YOUR_KEY> --addr 0.0.0.0:19075 &>/tmp/rportd.log &
```
For the first testing leave the console open and observe the log with `tail -f /tmp/rportd.log`. 

### Connect a client
We call the client `client1.local.localdomain`.
On your client just install the client binary 
```
curl -LSs https://github.com/cloudradar-monitoring/rport/releases/download/0.1.19/rport_0.1.19_Linux_x86_64.tar.gz|\
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

## Configuration files
Config files can be used to set up both the rport server and clients. In order to use it an arg `--config`(or `-c`) should be passed to a command with a path to the file. Configuration examples `rportd.example.conf` ([view online](rportd.example.conf)) and `rport.example.conf` ([view online](rport.example.conf)) can be found in the release archive or in the source.

NOTE: command arguments and env variables will override values from the config file.

In order to load the configuration from a file run:
```
rportd -c /etc/rport/rportd.conf
rport -c /etc/rport/rport.conf
```

## Proper client and server installation
### Don't use the root user
Client and server don't require running as root in Linux. You should avoid this. Create an unprivileged system-user instead.
```
useradd -r -d /var/lib/rport -m -s /bin/false -U -c "System user for rport client and server" rport
mkdir /var/log/rport/
chown rport:root /var/log/rport/
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
ExecStart=/usr/local/bin/rportd --config /etc/rport/rportd.conf
ExecReload=kill -SIGUSR1 $MAINPID
Restart=on-failure
RestartSec=5
StandardOutput=file:/var/log/rport/rportd.log
StandardError=file:/var/log/rport/rportd.log

[Install]
WantedBy=multi-user.target
```

Start it and enable the auto-start on boot
```
systemctl daemon-reload
systemctl start rportd
systemctl enable rportd
```

### Using authentication
Anyone who knows the address and the port of your rport server can use it for tunneling. In most cases, this is not desired. Your rport server could be abused for example to publish content under your IP address. Therefore, using rport with authentication is highly recommended.

Using a static username password pair is the most basic option. See the comments in the [rportd.example.conf](rportd.example.conf) and read more about all supported [authentication options](docs/client-auth.md). 

On the client start the tunnel this way
`rport --auth rport:password123 --fingerprint <YOUR_FINGERPRINT> node1.example.com:19075 2222:0.0.0.0:22`
*Note that in this early version the order of the command line options is still important. This might change later.*


## On-demand tunnels using the API
Initializing the creation of a tunnel from the client is nice but not a perfect solution for secure and reliable remote access to a large number of machines.
Most of the time the tunnel wouldn't be used. Network resources would be wasted and a port is exposed to the internet for an unnecessarily long time.
Rport provides the option to establish tunnels from the server only when you need them.

#### Step 1: activate the API
The internal management API is disabled by default. To activate it use a config file that is described
in ["Configuration files"](https://github.com/cloudradar-monitoring/rport#configuration-files) section.
Set up `[api]` config params. For example:
   ```
   # specify non-empty api.address to enable API support
   [api]
     # Defines the IP address and port the API server listens on
     address = "127.0.0.1:3000"
     # Defines <user:password> authentication pair for accessing API
     auth = "admin:foobaz"
   ```
This opens the API and enables HTTP basic authentication with a single user "admin:foobaz" who has access to the API.
To enable access to multiple users and to mange them in the file use "api.auth_file" config param (or "--api-authfile" rportd command arg).
Restart the rportd after any changes to the configuration.
Read more about the supported [api authentication options](docs/api-auth.md).

Read the [Swagger API docs](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/cloudradar-monitoring/rport/master/api-doc.yml).

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
Read more about the [management of tunnel via the API](docs/managing-tunnels.md) or read the [Swagger API docs](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/cloudradar-monitoring/rport/master/api-doc.yml).

#### Step 4: Install a web-based frontend
Rport comes with a user-friendly web-based frontend. The frontend has it's own none-open-source repository. The installation is quick and easy. [Learn more](docs/frontend.md).

### Versioning model
rport uses `<major>.<minor>.<buildnumber>` version pattern for compatibility with a maximum number of package managers.

Starting from version 1.0.0 packages with even <minor> number are considered stable.


### Credits
Forked from [jpillora/chisel](https://github.com/jpillora/chisel)
