---
title: "Tunnels on Subdomains"
weight: 22
slug: tunnels-on-subdomains
---
{{< toc >}}

## Introduction

Starting with rport version 0.9.5 you can access http-based tunnels on subdomains.

This feature was introduced to comply with strict firewall rules that block outgoing connections to none-default ports.
After a tunnel has been created you access it on a random port according to the port range reserved for tunnels.
With the new feature an additional random subdomain is created typically on port 443 as an alternative way to access the
tunnel.

That means you can access a tunnel on `https://rport.example.com:34567` or on `https://pio0toazivaeGheif2.tunnels.example.com`

Subdomains can be created for http(s) based tunnels such as RDP or VNC via browser or remote web-uis. SSH over web is
currently not supported.

For the subdomain to tunnel routing rportd uses [caddy](https://caddyserver.com/) as reverse-proxy. Starting, stopping
and configuring the caddy server is entirely done by rportd. No manual caddy configuration is needed.

## Prerequisites

### Caddy

You must have caddy installed on the same host with rportd. Using caddy inside a separated docker container is not
supported. Rportd takes full control over caddy and the communication is handled over a unix socket.

Either install caddy manually by just downloading it

```bash
sudo curl -L "https://caddyserver.com/api/download?os=linux&arch=amd64" -o /usr/local/bin/caddy
sudo chmod +x /usr/local/bin/caddy
```

or install [via package manager](https://caddyserver.com/docs/install#debian-ubuntu-raspbian). This way you make sure
you get caddy updates together with OS updates.

Once installed execute

```bash
sudo setcap 'cap_net_bind_service=+ep' $(which caddy)
```

{{< hint type=important title="CRUCIAL PRECONDITIONS">}}
**Allow privileged port binding**\
Because either `rportd` nor `caddy` run with root privileges, by default caddy cannot bind to port 443 or any port
below 1024. You must explicitly allow this by executing `sudo setcap 'cap_net_bind_service=+ep' $(which caddy)`.
If the command `setcap` is not available, install with `sudo apt-get install libcap2-bin`.

**Disable caddy systemd unit**\
When installing caddy via package manager, caddy will start automatically on system start. This might conflict with
caddy being run as subprocess from rportd. You would be well advised to switch caddy systemd autostart off with
`systemctl disable caddy`

{{< /hint >}}

### Wildcard DNS record

To make the random subdomain available you must create a wildcard DNS record. This is usually done by entering `*.` as
prefix for a subdomain. For example, if you want to use `*.tunnels.example.com` you must create a new host `*.tunnels`
for the domain `example.com`. Using the API and UI inside the range of the wildcard is supported, for example `*.example.com`
for the tunnel subdomains, and `rport.example.com`. In such case the second DNS record is not needed.

### Wildcard SSL Certificate

For the dynamic creation of the subdomains you need a wildcard SSL certificate that covers the DNS record. You can buy
a commercial certificate or use Let's encrypt. For the latter you must confirm your ownership of the domain by a DNS
record. Look at the example below:

```bash
certbot certonly --manual --preferred-challenges dns -d *.tunnels.example.com
```

Before executing the above command, make sure you are logged in to the admin panel of your DNS.
The command will display a DNS text record you must create, and then it waits for your confirmation.
Before continuing open a new terminal and make sure your DNS TXT record has become available using for example `dig`:

```shell
dig -t txt _acme-challenge.tunnels.example.com
```

If your record is listed as indicated by certbot, continue.

Doublecheck your certificate is a wildcard certificate with:

```bash
openssl x509 -noout -subject -in  /etc/letsencrypt/live/<YOUR-DOMAIN>/fullchain.pem
```

Make sure the `rport` **user has read-rights on the certificate** and key file. By default, Let's encrypt creates files  
readable just for root. Consider giving read access to anyone or create a user group.

```bash
find /etc/letsencrypt/archive/ -type d -exec chmod o+rx {} \;
find /etc/letsencrypt/archive/ -type f -exec chmod o+r {} \;
find /etc/letsencrypt/live/ -type d -exec chmod o+rx {} \;
find /etc/letsencrypt/live/ -type f -exec chmod o+r {} \;
```

## Configure rportd

For the configuration of the caddy integration you have two options.

1. **split port setup**: The rportd-API/UI and the subdomains will listen on different TCP ports. Subdomains will typically
   use 443, and the API/UI any other free port, e.g. 8443. Caddy will route subdomains only, the API/UI has its
   independent built-in web server.
2. **shared port setup**: The rportd-API/UI and the subdomains will listen on the same TCP port, typically 443.
   Caddy will route all http connection. The built-in API/UI web server acts as a backend behind the reverse proxy.

### Common and split port setup

For any kind of setup, add a configuration like this to your `/etc/rport/rportd.conf` file.

```toml
[caddy-integration]
  ## Enable https tunnels on random subdomains. 
  ## See https://oss.rport.io/advanced/tunnels-on-subdomains/
  ## Note: no defaults currently.

  ## Specifies the path to the caddy executable. mandatory.
  # caddy="/usr/bin/caddy"
  ## The bind address where caddy should listen for subdomain tunnels connections. mandatory.
  # address="0.0.0.0:8443"
  ## All caddy subdomain tunnels will have the domain prefix listed below. mandatory.
  # subdomain_prefix="tunnels.example.com"
  ## An SSL wildcard certificate is required that matches the subdomain prefix above. mandatory.
  # cert_file="/etc/letsencrypt/live/<YOUR-DOMAIN>/fullchain.pem"
  # key_file="/etc/letsencrypt/live/<YOUR-DOMAIN>/privkey.pem"
```

* For the `subdomain_prefix` do not enter the `*` sign. If your DNS record is `*.rport.example.com` enter
  `rport.example.com`.
* Make sure the port of the `address` does not conflict with the port of the address in the `[api]` section.

If you want to run the subdomains managed by caddy and the rport API/UI on different TCP ports, you can stop here and
restart rportd.

### Shared port setup

Additionally, to the above setting, add the following lines to the `[caddy-integration]` block.

```toml
  ## If you want to run the API and the tunnel subdomains on the same HTTPs port,
  ## you must specify a hostname for the API.
  api_hostname = "rport-api.example.com"
  ## Even if the above api_hostname is  inside the validity of the above certificate, 
  ## you must specify wich certificate to use for the API.
  api_cert_file = "/etc/ssl/certs/rport/api.crt"
  api_key_file = "/etc/ssl/certs/rport/api.key"
  ## Port of the API with TLS switched off. Port must match the port of "[api] address"
  api_port = "3000"
```

The `address` on the `[api]` must match the `api_port` on the `[caddy-integration]` section.
Also, **the api must have SSL/TLS switched off**, by commenting out the paths to the key and certificate.

For the above example the corresponding `[api]` config is:

```toml
  [api]
  ## Defines the IP address and port the API server listens on.
  ## Specify non-empty {address} to enable API support.
  address = "0.0.0.0:3000"
  ## <snip snap>
  ## If both cert_file and key_file are specified, then rportd will use them to serve the API with https.
  ## Intermediate certificates should be included in cert_file if required.
  #cert_file = "/etc/letsencrypt/live/rport/fullchain.pem"
  #key_file = "/etc/letsencrypt/live/rport/privkey.pem"
```

{{< hint type=caution title="Attention to hostname changes">}}
If you change the hostname of your rport server in the `rportd.conf` file you very likely must change it in the
email-based two-factor sender script too. Check `/usr/local/bin/2fa-sender.sh`.
{{< /hint>}}

### Troubleshoot

1. If rport doesn't start with the new configuration, inspect the start errors with

   ```bash
   journalctl -u rportd.service --no-pager -e
   ```

2. Also, look at `/var/log/rport/rportd.log`.
3. Consider increasing the `log_level` to `debug` in the `/etc/rport/rportd.conf` file.
4. To query the current caddy routing and the active subdomains, use:

    ```bash
    curl http://localhost/config/apps/http/servers/srv0/routes \
    --unix-socket /var/lib/rport/caddy-admin.sock -H "host:unix"|jq
    ```

## Use it

* Create a tunnel for RDP and activate "Enable RDP via browser" or
* Create a tunnel for VNC and select "Enable NoVNC (VNC via Browser)" or
* Create a tunnel for HTTP/HTTPs and activate "Enable HTTP Reverse proxy"

All the above tunnel settings will trigger the creation of a subdomain. After tunnel creation you will notice that the
API returns a field `tunnel_url`. The "access tunnel" button of the UI will point to that URL instead of pointing to the
random port on the main domain of the rport server.
