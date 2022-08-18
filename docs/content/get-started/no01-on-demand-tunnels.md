---
title: "On-demand tunnels"
weight: 1
slug: on-demand-tunnels
aliases:
  - /docs/no01-on-demand-tunnels.html
---
{{< toc >}}

Initializing the creation of a tunnel from the client is nice but not a perfect solution for secure and reliable remote
access to a large number of machines. Most of the time the tunnel wouldn't be used. Network resources would be wasted
and a port is exposed to the internet for an unnecessarily long time. Rport provides the option to establish tunnels
from the server only when you need them.

## Activate the API

Using the provided `rportd.example.conf` the internal management API is enabled by default listening on <http://localhost:3000>.

Set up `[api]` config params. For example:

   ```yaml
   # specify non-empty api.address to enable API support
   [api]
     # Defines the IP address and port the API server listens on
     address = "127.0.0.1:3000"
     # Defines <user:password> authentication pair for accessing API
     auth = "admin:foobaz"
     jwt_secret = "quei1too2Jae3xootu"
   ```

This opens the API and enables HTTP basic authentication with a single user "admin:foobaz" who has access to the API.
Restart the rportd after any changes to the configuration. Read more about API [authentication options](no02-api-auth.md).

{{< hint type=warning >}}
Do not run the API on public servers with the default credentials. Change the `auth=` settings and generate your own
`jwt_secret`, e.g. by using the command `pwgen 24 1` or `openssl rand -hex 12`.
{{< /hint >}}

{{< hint type=tip >}}
If you expose your API to the public internet, it's highly recommended to enable HTTPS. Read the [quick HTTPS howto](no08-https-howto.md).
{{< /hint >}}
By default the built-in web server can run the API only on ports above 1024. To circumvent this limit, refer to
[running the API on privileged ports.](#running-the-api-on-a-privileged-port)

Test if you set up the API properly by querying its status with `curl -s -u admin:foobaz http://localhost:3000/api/v1/status`.

{{< hint type=tip >}}
The API always returns a minified json formatted response. The large output is hard to read. In all further examples,
we use the command-line tool [jq](https://stedolan.github.io/jq/) to reformat the json with line breaks and indentation
for better readability. `jq`is included in almost any distribution, for Windows you can download it
[here](https://stedolan.github.io/jq/download/).
{{< /hint >}}

Example of a human-readable API status

```shell
~# curl -s -u admin:foobaz http://localhost:3000/api/v1/status |jq
{
  "data": {
    "connect_url": ["http://0.0.0.0:8080"],
    "fingerprint": "2a:c8:79:09:80:ba:7c:60:05:e5:2c:99:6d:75:56:24",
    "clients_connected": 3,
    "clients_disconnected": 1,
    "version": "0.1.28"
  }
}
```

## Connect a client without a tunnel

Invoke the rport client without specifying a tunnel but with some extra data.

```shell
rport --id my-client-1 \
  --fingerprint <YOUR_FINGERPRINT> \
  --tag Linux --tag "Office Berlin" \
  --name "My Test VM" --auth user1:1234 \
  node1.example.com:8080
```

*Add auth and fingerprint as already explained.*

This attaches the client to the message queue of the server without creating a tunnel.

## Manage clients and tunnels through the API

On the server, you can supervise the attached clients using
`curl -s -u admin:foobaz http://localhost:3000/api/v1/clients`.
Here is an example:

```shell
curl -s -u admin:foobaz http://localhost:3000/api/v1/clients|jq
[
  {
    "id": "my-client-1",
    "name": "My Test VM",
    "os": "Linux my-devvm-v3 5.4.0-37-generic #41-Ubuntu SMP Wed Jun 3 18:57:02 UTC 2020 x86_64 x86_64 x86_64 GNU/Linux",
    "os_arch": "amd64",
    "os_family": "debian",
    "os_kernel": "linux",
    "os_full_name": "Debian",
    "os_version": "5.4.0-37",
    "os_virtualization_system":"KVM",
    "os_virtualization_role":"guest",
    "cpu_family":"59",
    "cpu_model":"6",
    "cpu_model_name":"Intel(R) Xeon(R) Silver 4110 CPU @ 2.10GHz",
    "cpu_vendor":"Intel",
    "num_cpus":16,
    "mem_total":67020316672,
    "timezone":"UTC (UTC+00:00)",
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
    "os_arch": "amd64",
    "os_family": "alpine",
    "os_kernel": "linux",
    "os_full_name": "Alpine Linux",
    "os_version": "4.19.80-0",
    "os_virtualization_system":"",
    "os_virtualization_role":"",
    "cpu_family":"6",
    "cpu_model":"79",
    "cpu_model_name":"Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz",
    "cpu_vendor":"Intel",
    "num_cpus":4,
    "mem_total":8363900928,
    "timezone":"CEST (UTC+02:00)",
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
Read more about the [management of tunnel via the API](/docs/content/get-started/no09-managing-tunnels.md)
or read the [API docs](https://apidoc.rport.io/master/).

## Running the API on a privileged port

For security reasons the rport server runs as an unprivileged user and you should not change this. But unprivileged
users are not allowed to open port below 1024. Instead of changing user privileges you can change the capabilities of
the rportd binary and allow it to use any port even when not running as root.

```shell
sudo setcap CAP_NET_BIND_SERVICE=+eip /usr/local/bin/rportd
```

That's all.
Now you can use "0.0.0.0:443" as API address.

You need to run the above command everytime you change the rpotd binary, for example after every update.
