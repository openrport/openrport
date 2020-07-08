# rport
Create reverse tunnels with ease.

## Build and installation
### From source
1) Build from source (Linux or Mac OS/X.):
    ```bash
    make all
    ```
    `rport` and `rportd` binaries will appear in directory.  

2) Build using Docker:
    ```bash
    make docker-goreleaser
    ```
    will create binaries for all supported platforms in ./dist directory.

## Usage
`rportd` should be executed on the machine, acting as a server.

`rport` is a client app which will try to establish long-running connection to the server.

Minimal setup:
1) Execute `./rportd -p 9999` on a server.
1) Execute `./rport <SERVER_IP>:9999 3389:3389` on a client.
1) Now end-users can connect to `<SERVER_IP>:3389` (e.g. using Remote Desktop Connection). The connection will be proxied to client machine.

See `./rportd --help` and `./rport --help` for more options, like:
- Specifying certificate fingerprint to validate server authority
- Client session authentication using user:password pair
- Restricting, which users can connect
- Specifying additional intermediate HTTP proxy
- Using POSIX signals to control running apps
- Setting custom HTTP headers

## Quickstart guide
### Install and run the rport server
On a machine connected to the public internet and ideally with an FQDN registered to a public DNS install and run the server.
The server is called node1.example.com in this example.

Install client and server
```
curl -LSs https://github.com/cloudradar-monitoring/rport/releases/download/0.1.0/rport_0.1.0-SNAPSHOT-7323e7c_linux_amd64.tar.gz|\
tar vxzf - -C /usr/local/bin/
````

Create a key for the server instance. Store this key and don't change it. Otherwise, your fingerprint will change and your clients might be rejected. 
```
openssl rand -hex 18
```  

Start the server as a background task.
```
nohup rportd --key <YOUR_KEY> -p 19075 &>/tmp/rportd.log &
```
For the first testing leave the console open and observe the log with `tail -f /tmp/rportd.log`. Note the fingerprint. You will use it later. 

To safely store and reuse the key use these commands.
```
echo "RPORT_KEY=$(openssl rand -hex 18)">/etc/default/rport
. /etc/default/rport
export RPORT_KEY=$RPORT_KEY
nohup rportd -p 19075 &>/tmp/rportd.log &
```
rportd reads the key from the environment so it does not appear in the process list or the history. 

### Connect a client
We call the client `client1.local.localdomain`.
On your client just install the client binary 
```
curl -LSs https://github.com/cloudradar-monitoring/rport/releases/download/0.1.0/rport_0.1.0-SNAPSHOT-7323e7c_linux_amd64.tar.gz|\
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

### Using systemd
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

Create a config file `/etc/default/rport` like this exmaple.
```
RPORT_KEY=<YOUR_KEY>
HOST=0.0.0.0
PORT=19075
```

Start it
```
systemctl daemon-reload
service rportd restart
```

### Using authentication
Anyone who knows the address and the port of your rport server can use it for tunneling. In most cases, this is not desired. Your rport server could be abused for example to publish content under your IP address. Therefore using rport with authentication is highly recommended.

For the server `rportd --auth rport:password123` is the most basic option. All clients must use the username `rport` and the given password. 

On the client start the tunnel this way
`rport --auth rport:password123 --fingerprint <YOUR_FINGERPRINT> node2.rport.io:19075 2222:0.0.0.0:22`
*Note that in this early version the order of the command line options is still important. This might change later.*

If you want to maintain multiple users with different passwords, create a json-file `/etc/rportd-auth.json` with credentials, for example
```
{
    "user1:foobaz": [
        ".*"
    ],
    "user2:bingo": [
        "210\\.211\\.212.*",
        "107\\.108\\.109.*"
    ],
    "rport:password123": [
        "^999"
    ]
}
```
*For now, rportd reads only the user and password. The optional filters to limit the tunnels to match a regex are under construction.*
*Rportd reads the file immediately after writing without the need for a sighub. This might change in the future.*

Start the server with `rportd --authfile /etc/rport-auth.json`. Change the `ExecStart` line of the systemd service file accordingly. 

### Credits
Forked from [jpillora/chisel](https://github.com/jpillora/chisel)