---
title: "Managing tunnels"
weight: 09
slug: managing-tunnels
aliases:
  - /docs/no09-managing-tunnels.html
  - /docs/content/get-started/no09-managing-tunnels.md
---
{{< toc >}}

## Manage tunnel client-side

Specify the desired tunnel on the command line, for example

```shell
rport --auth rport:password123 <SERVER_IP>:9999 2222:22
```

Alternatively add tunnels to the configuration file `rport.conf`.

## Manage tunnel server-side

On the server, you can supervise and manage the attached clients through the [API](https://apidoc.rport.io/master/#tag/Clients-and-Tunnels).

### List

`curl -s -u admin:foobaz http://localhost:3000/api/v1/clients`. *Use `jq` for pretty-printing json.*

Here is an example:

```shell
curl -s -u admin:foobaz http://localhost:3000/api/v1/clients|jq
```

Shortened response:

```json
[
  {
    "id": "2ba9174e-640e-4694-ad35-34a2d6f3986b",
    "name": "My Test VM",
    "os": "Linux my-devvm-v3 5.4.0-37-generic #41-Ubuntu ... x86_64 GNU/Linux",
    "tunnels": []
  },
   {
    "id": "aa1210c7-1899-491e-8e71-564cacaf1df8",
    "name": "Random Rport Client",
    "os": "Linux alpine-3-10-tk-01 4.19.80-0-virt #1-Alpine ... x86_64 Linux",
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

The above example shows one client connected with an active tunnel. The second client is in standby mode.

### Create

Now use `PUT /api/v1/clients/{id}/tunnels?local={port}&remote={port}` to request a new tunnel for a client.
For example,

```shell
CLIENTID=2ba9174e-640e-4694-ad35-34a2d6f3986b
LOCAL_PORT=4000
REMOTE_PORT=22
curl -u admin:foobaz -X PUT \
"http://localhost:3000/api/v1/clients/$CLIENTID/tunnels?local=$LOCAL_PORT&remote=$REMOTE_PORT"|jq
```

The ports are defined from the servers' perspective. "Local" refers to the local ports of the rport server.
"Remote" refers to the ports and interfaces of the client. The above example opens port 4000 on the rport server and
forwards to the port 22 of the client.

Using

```shell
curl -s -u admin:foobaz -X GET "http://localhost:3000/api/v1/clients"
```

or

```shell
curl -s -u admin:foobaz -X GET "http://localhost:3000/api/v1/clients" \
|jq ".data[] | select(.id==\"$CLIENTID\")|.tunnels"
```

confirms the tunnel has been established.

```json
{
 "tunnels": [
  {
   "lhost": "0.0.0.0",
   "lport": "4000",
   "rhost": "0.0.0.0",
   "rport": "22",
   "id": "1"
  }
 ]
}
```

The above example makes the tunnel available without restrictions. Learn more about access control (ACL) below.

If you omit the local port a random free port on the rport server is selected. For example,

```shell
curl -s -u admin:foobaz -X PUT "http://localhost:3000/api/v1/clients/$CLIENTID/tunnels?remote=22"|jq
{
  "data": {
    "success": 1,
    "tunnel": {
      "lhost": "0.0.0.0",
      "lport": "38126",
      "rhost": "0.0.0.0",
      "rport": "22",
      "lport_random": true,
      "id": "4"
    }
  }
}
```

The rport client is not limited to establish tunnels only to the system it runs on. You can use it as a jump host to
create tunnels to foreign systems too.

```shell
CLIENTID=2ba9174e-640e-4694-ad35-34a2d6f3986b
LOCAL_PORT=4001
REMOTE_PORT=192.168.178.1:80
curl -u admin:foobaz -X PUT \
"http://localhost:3000/api/v1/clients/$CLIENTID/tunnels?local=$LOCAL_PORT&remote=$REMOTE_PORT"
```

This example forwards port 4001 of the rport server to port 80 of `192.168.178.1` using the rport client in the middle.

```json
{
 "tunnels": [
  {
   "lhost": "0.0.0.0",
   "lport": "4001",
   "rhost": "192.168.178.1",
   "rport": "80",
   "id": "1"
  }
 ]
}
```

Idle tunnels are automatically closed after 5 minutes. You can change `idle-timeout-minutes` parameter to provide a
custom value in minutes.

For example:

```shell
CLIENTID=2ba9174e-640e-4694-ad35-34a2d6f3986b
LOCAL_PORT=4000
REMOTE_PORT=22
curl -u admin:foobaz -X PUT \
"http://localhost:3000/api/v1/clients/$CLIENTID/tunnels?local=$LOCAL_PORT&remote=$REMOTE_PORT&idle-timeout-minutes=10
```

If no `idle-timeout-minutes` parameter is given, the default idle timeout will be 5 minutes.
To disable auto-closing of tunnels, you should provide `skip-idle-timeout` parameter, e.g.:

```shell
CLIENTID=2ba9174e-640e-4694-ad35-34a2d6f3986b
LOCAL_PORT=4000
REMOTE_PORT=22
curl -u admin:foobaz -X PUT \
"http://localhost:3000/api/v1/clients/$CLIENTID/tunnels?local=$LOCAL_PORT&remote=$REMOTE_PORT&skip-idle-timeout=1
```

Please note, that you should not use `skip-idle-timeout` and `idle-timeout-minutes` in the same request, what will cause
a conflicting parameter error.

#### Tunnel access control

To increase the security of remote access, you can control how it is allowed to use a tunnel by limiting the tunnel
usage to ip v4 addresses or network segments (ipv6 is not supported yet).

```shell
CLIENTID=2ba9174e-640e-4694-ad35-34a2d6f3986b
LOCAL_PORT=4000
REMOTE_PORT=22
ACL=213.90.90.123,189.20.90.0/24
curl -u admin:foobaz -X PUT \
"http://localhost:3000/api/v1/clients/$CLIENTID/tunnels?local=$LOCAL_PORT&remote=$REMOTE_PORT&acl=$ACL"
```

A list of single ip-addresses or network segments separated by a comma is accepted.

### Delete

Using a DELETE request with the tunnel id allows terminating a tunnel.

```shell
CLIENTID=2ba9174e-640e-4694-ad35-34a2d6f3986b
TUNNELID=1
curl -u admin:foobaz -X DELETE \
"http://localhost:3000/api/v1/clients/$CLIENTID/tunnels/$TUNNELID"
```

## Reverse proxy for http(s) based tunnels

Starting with RPort version 0.5 the server comes with a built-in http reverse proxy. The reverse proxy runs on top of
tunnels pointing to remote http or https backend.

{{< mermaid class="text-center">}}
flowchart LR
  0[your browser] --> 1[Reverse Proxyon RPort server] --> 2[Tunnel endpoint] --> 3[Tunnel] --> 4[RPort Client] --> 5[Target]
{{< /mermaid >}}
Using the reverse tunnel allows you to access remote web servers (web-based configuration of switches, routers, NAS)
through a secure https communication with valid ssl certificates on the public side.

To enable this feature the rport server needs a key and a certificate in the `[server]` section of the `rportd.conf`
file. If you run the server and API on the same DNS hostname, you can use the same key and certificate for the server and the API.

Running the built-in http proxy without encryption is not supported.

If you have upgraded your RPort server from an older version, insert the following lines into the `[server]` section
of `rportd.conf` manually.

```text
  ## Enable the creation of tunnel proxies with giving certificate- and key-file
  ## Defaults: not enabled
  #tunnel_proxy_cert_file = "/var/lib/rport/server.crt"
  #tunnel_proxy_key_file = "/var/lib/rport/server.key" 
```

By using `http_proxy=1` on [tunnel creation](https://apidoc.rport.io/master/#operation/ClientTunnelsPut), the proxy will
come up together with the tunnel. The rport server will then use two tcp ports. One for the raw tcp tunnel and one for
the http proxy. ACLs are applied to proxy also. The raw tcp tunnel is bound to localhost only, and therefore it's not
accessible from the outside.

{{< hint type=warning >}}
If you want to use the built-in tunnel proxy, accessing your rport server via an FQDN – and not via IP address – is
mandatory. Otherwise, you will always get warnings about invalid SSL certificates.
{{< /hint >}}

### Manipulating the host header

The http header `host` is taken from the client request and passed without modification to the backend (remote target).
If you use your browser for accessing the tunnel proxy, the host is usually set to the FQDN of your rport server.
If the remote side requires a specific header `host` to jump into the right virtual host, you can specify a host header
that will be used for the proxy connection. For example `http_proxy=1&host_header=www.example.com`.

### Example

```bash
curl -s -X PUT -G "${RPORT-SERVER}/api/v1/clients/${CLIENT_ID}/tunnels"  \
 -d remote=192.168.249.1:443 \
 -d scheme=https \
 -d acl=87.79.148.181 \
 -d idle-timeout-minutes=5 \
 -d check_port=1 \
 -d http_proxy=1 \
 -d host_header=fritz.box \
 -H "Authorization: Bearer $TOKEN" \
 -H 'accept: application/json' \
 -H 'Content-Type: application/json'|jq
 ```

*The `-G` switch of curl takes all `-d` key-value-pairs and appends them to the URL concatenaited by an ampersand.*

In the example, the client creates a bridge to the https web-interface of a router and a reverse proxy will be started automatically.
You will get a response like this:

 ```json
 {
  "data": {
    "lhost": "0.0.0.0",
    "lport": "21504",
    "rhost": "192.168.249.1",
    "rport": "443",
    "lport_random": true,
    "scheme": "https",
    "acl": "87.79.148.181",
    "idle_timeout_minutes": 5,
    "http_proxy": true,
    "id": "2"
  }
}
```

Now you can point you browser to `https://{RPORT-SERVER}:21504` to access the web server on the remote side.
