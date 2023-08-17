---
title: "external IP Address determination"
weight: 23
slug: "ip-address-determination"
---
{{< toc >}}

## Introduction

Starting with version 0.9.13 the rport client can determine its external IP addresses (ipV4 and ipV6) by asking an
external API.  
This is more reliable than extracting it from the TCP headers on the server. You will get correct external IP addresses
even if the rport server runs behind a reverse proxy.

Using an external API comes with downsides:

* It creates additional network traffic.
* The API might log your request making the identification of your network possible.

The latter is mitigated by sending an empty user agent that will not identify the rport client on first sight.

For privacy reasons the feature is disabled by default.

## Supported APIs

Any API that accepts ipv6 and ipv4 requests and returns a JSON-formatted string like the below is suitable.

```json
{"ip":"<IP ADDRESS>"}
```

Additional fields in the response don't harm. They are ignored.

The following API URLs are known to be compatible:

* `https://api.my-ip.io/ip.json`
* `https://api.myip.com`
* `https://api.seeip.org/jsonip`
* `https://myip.rport.io`

{{< hint type=note title="ipv4 and ipv6" >}}
Before entering the API URL to the `rport.conf` make sure they accept ipv4 **and ipv6** requests.  
Test with `curl -4 <URL>` and `curl -6 <URL>`.
{{< /hint >}}

## Run your own IP API

Public APIs might disappear suddenly, or you get blocked because of too many requests. Therefore, consider running your
own API on some external webserver.

Put the following PHP script on a web server. That's all you need.

```php
<?php
header('Content-Type: application/json; charset=utf-8');
echo json_encode(['ip'=>$_SERVER['REMOTE_ADDR']]);
```

If you want to **protect your IP API** with HTTP basic auth, you can specify the user and password in the `rport.conf`
file for the `ip_api_url` using `https://user:password@api.example.com`.
