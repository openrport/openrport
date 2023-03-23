---
title: 'Concepts practically explained'
weight: 0
slug: quick-start
---

{{< hint type=warning title="This is not an installation tutorial!" >}}
If you want to **install and run rport (server and client), switch to the [installation tutorial](https://kb.rport.io/install-the-rport-server)**.  
Our end-user knowledge [kb.rport.io](https://kb.rport.io) base focuses on installation and server maintenance from a user's perspective.

The below documentation is a practical demonstration of the rport concepts.
It's intended to be read by developers and experienced users who want to learn what happens behind the scenes.
{{< /hint>}}

## Build and installation

### Binaries

We provide pre-compiled binaries. You can download them [here](https://github.com/realvnc-labs/rport/releases).

### From source

1. Build from source (Linux or macOS):

   ```bash
   make all
   ```

   `rport` and `rportd` binaries will appear in directory.

2. Build using Docker:

   ```bash
   make docker-goreleaser
   ```

   will create binaries for all supported platforms in `./dist` directory.

## Usage

`rportd` should be executed on the machine, acting as a server.

`rport` is a client app which will try to establish long-running connection to the server.

Minimal setup:

1. Execute `./rportd --addr 0.0.0.0:9999 --auth rport:password123 --data-dir /var/tmp` on a server.
2. Execute `./rport --auth rport:password123 <SERVER_IP>:9999 2222:22` on a client or  
   `./rport --auth rport:password123 <SERVER_IP>:9999 22` and the server tunnel port will be randomly chosen for you.
3. Now end-users can connect to `<SERVER_IP>:2222` (e.g. using a SSH Connection). The connection will be proxied to
   the client machine.

See `./rportd --help` and `./rport --help` for more options, like:

- Specifying certificate fingerprint to validate server authority
- Client session authentication using user:password pair
- Restricting, which users can connect
- Specifying additional intermediate HTTP proxy
- Using POSIX signals to control running apps
- Setting custom HTTP headers
- Using IPv6 addresses when starting a server

## Run the server without installation

If you quickly want to run the rport server without installation, run the following commands from any unprivileged user account.

```shell
curl -LOJ https://downloads.rport.io/rport/stable/latest.php?arch=Linux_x86_64
tar vxzf rport_*_Linux_x86_64.tar.gz rportd
KEY=$(openssl rand -hex 18)
./rportd --log-level info --data-dir /var/tmp/ --key $KEY --auth user1:1234
```

Rportd will be listening on the default port 8080 for client connections.
Grab the generated fingerprint from `/var/tmp/rportd-fingerprint.txt` and use it for secure client connections.

## Install and run the rport server

On a machine connected to the public internet and ideally with an FQDN registered to a public DNS install and run the server.
Assume, the server is called `node1.example.com`.

### A note on security

{{< hint type=warning >}}
**Do not run the server as root!** This is an unnecessary risk. Rportd should always use an unprivileged user.
{{< /hint >}}

While using rport without a **fingerprint** is possible, it's highly recommended to not skip this part.
The fingerprint ensures you connect only to trusted servers. If you omit this step a man in the middle can bring up a
rport server and hijack your tunnels. If you do ssh or rdp through the tunnel, a hijacked tunnel will not expose your
credentials because the data inside the tunnel is still encrypted. But if you use rport for unencrypted protocols like
HTTP, sniffing credentials would be possible.

You might wonder why the rport server does not provide encryption on the transport layer (TLS, SSL, HTTPS).
**Encryption is always enabled.** Your connections are encrypted and secured by SSH over HTTP. When you start up the
rport server, it will generate an in-memory ECDSA public/private key pair. Adding TLS by putting an SSL
reverse proxy is possible so you get SSH over HTTPS.

### Install the server

For a proper installation execute the following steps.

```bash
curl -LOJ https://downloads.rport.io/rportd/stable/latest.php?arch=Linux_x86_64
sudo tar vxzf rportd_*_Linux_x86_64.tar.gz -C /usr/local/bin/ rportd
sudo useradd -d /var/lib/rport -m -U -r -s /bin/false rport
sudo mkdir /etc/rport/
sudo mkdir /var/log/rport/
sudo chown rport /var/log/rport/
sudo tar vxzf rportd_*_Linux_x86_64.tar.gz -C /etc/rport/ rportd.example.conf
sudo cp /etc/rport/rportd.example.conf /etc/rport/rportd.conf
```

Create a new unique key for the server instance. Store this key and don't change it. You will use it later. Otherwise,
your fingerprint will change and your clients might be rejected. Open the `/etc/rport/rportd.conf` with an editor.
Add a random string as `key_seed`. You can use `openssl rand -hex 18` to generate one.
Or just execute the following commands to generate and enter a new key to your configuration file.

```shell
KEY_SEED=$(openssl rand -hex 18)
sed -i "s/key_seed = .*/key_seed =\"${KEY_SEED}\"/g" /etc/rport/rportd.conf
```

All other default settings are suitable for a quick and secure start.

Change to the rport user account and check your rportd starts without errors.

```shell
ubuntu@node1:~$ sudo -u rport -s /bin/bash
rport@node1:/home/ubuntu$ rportd -c /etc/rport/rportd.conf --log-level info &
```

For the first testing leave the console open and observe the log with `tail -f /var/log/rport/rportd.log`.
Copy the generated fingerprint from `/var/lib/rport/rportd-fingerprint.txt` to your clipboard. Try your first client connection now.

## Run the server with systemd

If all works fine stop the rport server and integrate it into systemd.

```shell
sudo rportd --service install --service-user rport --config /etc/rport/rportd.conf
```

A file `/etc/systemd/system/rportd.service` will be created and systemd is ready to manage rportd.

```shell
sudo systemctl start rportd
sudo systemctl enable rportd # Optionally start rportd on boot
```

## Connect a client

Assume, the client is called `client1.local.localdomain`.
On your client just install the client binary

```shell
curl -LOJ https://downloads.rport.io/rport/stable/latest.php?arch=Linux_x86_64
sudo tar vxzf rport_*_Linux_x86_64.tar.gz -C /usr/local/bin/ rport
```

Create an ad hoc tunnel that will forward the port 2222 of `node1.example.com` to the to local port 22 of `client1.local.localdomain`.

```shell
rport --auth user1:1234 --fingerprint <YOUR_FINGERPRINT> --data-dir=/tmp node1.example.com:8080 2222:0.0.0.0:22
```

Observing the log of the server you get a confirmation about the newly created tunnel.

Now you can access your machine behind a firewall through the tunnel. Try `ssh -p 2222 node1.example.com` and you will
come out on the machine where the tunnel has been initiated.

## Run a Linux client with systemd

For a proper and permanent installation of the client execute the following steps.

```shell
curl -LOJ https://downloads.rport.io/rport/stable/latest.php?arch=Linux_x86_64
sudo tar vxzf rport_*_Linux_x86_64.tar.gz -C /usr/local/bin/ rport
sudo useradd -d /var/lib/rport -U -m -r -s /bin/false rport
sudo mkdir /etc/rport/
sudo mkdir /var/log/rport/
sudo chown rport /var/log/rport/
sudo tar vxzf rport_*_Linux_x86_64.tar.gz -C /etc/rport/ rport.example.conf
sudo cp /etc/rport/rport.example.conf /etc/rport/rport.conf
sudo rport --service install --service-user rport --config /etc/rport/rport.conf
```

Open the config file `/etc/rport/rport.conf` and adjust it to your needs. (See below.)
Finally, start the rport client and optionally register it in the auto-start.

```shell
systemctl start rport
systemctl enable rport
```

A very minimalistic client configuration `rport.conf` can look like this:

```shell
[client]
server = "node1.example.com:8080"
fingerprint = "<YOUR_FINGERPRINT>"
auth = "user1:1234"
remotes = ['2222:22']
```

This will establish a permanent tunnel and the local port 22 (SSH) of the client becomes available on port 2222 of the rport server.

## Run a Windows client

On Microsoft Windows [download the latest client binary](https://downloads.rport.io/rport/stable/latest.php?arch=Windows_x86_64)
and extract it ideally to `C:\Program Files\rport`.
Rename the `rport.example.conf` to `rport.conf` and store it in `C:\Program Files\rport` too.
Open the `rport.conf` file with a text editor. On older Windows use an editor that supports unix line breaks,
like [notepad++](https://notepad-plus-plus.org/).

A very minimalistic client configuration `rport.conf` can look like this:

```shell
[client]
server = "node1.example.com:8080"
fingerprint = "<YOUR_FINGERPRINT>"
auth = "user1:1234"
remotes = ['3300:3389']
```

This will establish a permanent tunnel and the local port 3389 (remote desktop) of the client becomes available on port
3300 of the rport server.

Before registering rport as a windows service, check your connection manually.

Open a command prompt with administrative rights and type in:

```shell
cd "C:\Program Files\rport"
rport.exe -c rport.conf
```

If you don't get errors on the console, try a remote desktop connection to the rport server on port 3300.
Stop the client with CTRL-C and register it as a service and start it.

```shell
rport.exe --service install -c rport.conf
sc query rport
sc start rport
```

The windows service will be created with "Startup type = automatic". If you don't want the rport client to start on boot,
you must manually disable it using for example `sc config rport start=disabled`.

## Run clients on other operating systems

Please refer to [clients on other operating systems](no05-client-on-other-os.md).

## Configuration files

Config files can be used to set up both the rport server and clients. In order to use it an arg `--config`(or `-c`)
should be passed to a command with a path to the file. Configuration examples `rportd.example.conf`
([view online](https://github.com/realvnc-labs/rport/blob/master/rportd.example.conf)) and `rport.example.conf`
([view online](https://github.com/realvnc-labs/rport/blob/master/rport.example.conf)) can be found in the
release archive or in the source.

NOTE: command arguments and env variables will override values from the config file.

In order to load the configuration from a file run:

```shell
rportd -c /etc/rport/rportd.conf
rport -c /etc/rport/rport.conf
```

## Using authentication

To prevent anyone who knows the address and the port of your rport server to use it for tunneling, using client
authentication is required.

Using a static username password pair is the most basic option. See the comments in the
[rportd.example.conf](https://github.com/realvnc-labs/rport/blob/master/rportd.example.conf) and read more about
all supported [authentication options](no03-client-auth.md).

On the client start the tunnel this way

```shell
rport --auth user1:1234 --fingerprint <YOUR_FINGERPRINT> node1.example.com:8080 2222:0.0.0.0:22
```

*Note that in this early version the order of the command line options is still important. This might change later.*

## Install a web-based frontend

Rport comes with a user-friendly web-based frontend. The frontend has it's own none-open-source repository.
The installation is quick and easy. [Learn more](/docs/content/get-started/no07-frontend.md)

## Install the command-line interface

You can also manage clients, tunnels, and command from a user-friendly command-line utility. It's available as a
stand-alone static binary for Windows and Linux.
See [https://github.com/realvnc-labs/rportcli](https://github.com/realvnc-labs/rportcli).
The command-line utility does not cover all API capabilities yet. But it's already a very useful tool making rport even
more powerful.
