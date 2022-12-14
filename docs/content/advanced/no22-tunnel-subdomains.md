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

You must have caddy installed on the same host with rportd. Using caddy inside a docker container is not supported.
Rportd takes full control over caddy and the communication is handled over a unix socket.

Either install caddy manually by just downloading it

```bash
sudo curl -L "https://caddyserver.com/api/download?os=linux&arch=amd64" -o /usr/local/bin/caddy
sudo chmod +x /usr/local/bin/caddy
sudo setcap 'cap_net_bind_service=+ep' /usr/local/bin/caddy
```

or install [via package manager](https://caddyserver.com/docs/install#debian-ubuntu-raspbian).

The latter makes sure you get caddy updates together with OS updates.

{{< hint type=important >}}
**Disable caddy systemd unit**\
When installing caddy via package manager, caddy will start automatically on system start. This might conflict with
caddy being run as subprocess from rportd. You would be well advised to switch caddy systemd autostart off.

```bash
systemctl disable caddy
```

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
The command will display a DNS text record you must create, and then it waits and constantly checks the DNS record to
become available. This can take several minutes. Don't cancel the certbot command. It's not hanging.

Doublecheck your certificate is a wildcard certificate with:

```bash
openssl x509 -noout -subject -in  /etc/letsencrypt/live/<YOUR-DOMAIN>/fullchain.pem
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
  # subdomain_prefix="tunnels.rport.test"
  ## An SSL wildcard certificate is required that matches the subdomain prefix above. mandatory.
  # cert_file="/var/lib/rport/tunnels.test.crt"
  # key_file="/var/lib/tunnels.rport.test.key"
```

* For the `subdomain_prefix` do not enter the `*` sign. If your DNS record is `*.rport.example.com` enter
  `rport.example.com`.
* Make sure the port of the `address` does not conflict with the port of the address in the `[api]` section.
* Make sure the `rport` user has read-rights on the certificate and key file. By default, Let's encrypt creates files  
  readable just for root. Consider giving read access to anyone or create a user group.

  ```bash
  find /etc/letsencrypt/archive/ -type d -exec chmod o+rx {} \;
  find /etc/letsencrypt/archive/ -type f -exec chmod o+r {} \;
  find /etc/letsencrypt/live/ -type d -exec chmod o+rx {} \;
  find /etc/letsencrypt/live/ -type f -exec chmod o+r {} \;
  ```

If you want to run the subdomains managed by caddy and the rport API/UI on different TCP ports, you can stop here and
restart rportd.

### Shared port setup

Additionally, to the above setting, add the following lines to the `[caddy-integration]` block.

```toml
  ## If you want to run the API and the tunnel subdomains on the same HTTPs port,
  ## you must specify a hostname for the API. 
  #api_hostname = "rport-api.example.com"
  ## If the above api_hostname is not inside the validity of the above certificate, you can optionally specify
  ## wich certificate to use for the API.
  #api_cert_file = "/etc/ssl/certs/rport/api.crt"
  #api_key_file = "/etc/ssl/certs/rport/api.key"
  ## The API will come up on localhost only without TLS on the given port.
  # api_port = "3000"
```

The `address` on the `[api]` section will be ignored and the API will listen on `127.0.0.1` only.

### Troubleshoot

1. If rport doesn't start with the new configuration, inspect the start errors with

   ```bash
   journalctl -u rportd.service --no-pager -e
   ```

2. Also, look at `/var/log/rport/rportd.log`.
3. Consider increasing the `log_level` to `debug` in the `/etc/rport/rportd.conf` file.
