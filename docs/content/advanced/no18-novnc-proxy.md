---
title: "noVNC-Proxy"
weight: 18
slug: novnc-proxy
aliases:
  - /docs/no18-novnc-proxy.html
---
{{< toc >}}

## Preface

Starting with version 0.5.10 RPort has the ability to create a tunnel to a remote VNC server with a built-in VNC-to-HTTPS
proxy. The rport server exposes the "vnc signal" on an encrypted HTTPS port instead of using the unencrypted VNC protocol.
If a vnc tunnel with proxy is created, rport first creates the tunnel to the remote machine, makes the remote VNC port
available only on localhost and spawns a vnc proxy that makes the VNC signal accessible over HTTPS on the proxy port with
path "/". Pointing the browser to this URL loads the novnc javascript app and the session starts.

## Prerequisites

* noVNC javascript app has to be available on local filesystem
* RPort server configuration `novnc_root` must point to the noVNC javascript app directory
* RPort server configuration `tunnel_proxy_cert_file` and `tunnel_proxy_key_file` must be set up. NoVNC depends on the
* generic built-in TLS reverse proxy, which is enabled by providing a key and a certificate.

## Install noVNC javascript app

Rport is tested with noVNC v1.3.0.zip, so this version is recommended.

* Download noVNC from github <https://github.com/novnc/noVNC/archive/refs/tags/v1.3.0.zip>
* Extract the content of the zip file to a local directory e.g. "/home/{user}/rportd/noVNC-1.3.0"

If your rport server runs with the suggested default settings, the NoVNC can be downloaded as follows.

```shell
cd /tmp
curl -LO https://github.com/novnc/noVNC/archive/refs/tags/v1.3.0.zip
unzip -d /var/lib/rport v1.3.0.zip
rm -f v1.3.0.zip
```

You will then have all needed files in `/var/lib/rport/noVNC-1.3.0`.

## Server configuration

Provide a value for `novnc_root` in the `[server]` section of the `rportd.conf`. It has to be the directory where the
noVNC javascript app is located e.g.

```text
[server]
 ...snip ...snap
 ## If specified, rportd will serve novnc javascript app from this directory.
 novnc_root = "/var/lib/rport/noVNC-1.3.0"
 ```
