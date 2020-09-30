# Managing tunnels
## Manage tunnel client-side

## Manage tunnel server-side
On the server, you can supervise and manage the attached clients through the API.
### List

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
The above example shows one client connected with an active tunnel. The second client is in standby mode.

### Create
Now use `PUT /api/v1/sessions/{id}/tunnels?local={port}&remote={port}` to request a new tunnel for a client session.
For example,
```
CLIENTID=2ba9174e-640e-4694-ad35-34a2d6f3986b
LOCAL_PORT=4000 
REMOTE_PORT=22
curl -u admin:foobaz -X PUT "http://localhost:3000/api/v1/sessions/$CLIENTID/tunnels?local=$LOCAL_PORT&remote=$REMOTE_PORT"
```
The ports are defined from the servers' perspective. "Local" refers to the local ports of the rport server. "Remote" refers to the ports and interfaces of the client.
The above example opens port 4000 on the rport server and forwards to the port 22 of the client.

Using `curl -s -u admin:foobaz -X GET "http://localhost:3000/api/v1/sessions"` or `curl -s -u admin:foobaz -X GET "http://localhost:3000/api/v1/sessions"|jq ".data[] | select(.id==\"$CLIENTID\")|.tunnels"` again confirms the tunnel has been established.
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
The above example makes the tunnel available without restrictions. Learn more about access control (ACL) below.

If you omit the local port a random free port on the rport server is selected. For example,
```
curl -s -u admin:foobaz -X PUT "http://localhost:3000/api/v1/sessions/$CLIENTID/tunnels?remote=22"|jq
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

#### Tunnel access control
To increase the security of remote access, you can control how it is allowed to use a tunnel by limiting the tunnel usage to ip-addresses or network segments.

```
CLIENTID=2ba9174e-640e-4694-ad35-34a2d6f3986b
LOCAL_PORT=4000 
REMOTE_PORT=22
ACL=213.90.90.123,189.20.90.0/24
curl -u admin:foobaz -X PUT "http://localhost:3000/api/v1/sessions/$CLIENTID/tunnels?local=$LOCAL_PORT&remote=$REMOTE_PORT&acl=$ACL"
```
A list of single ip-addresses or network segments separated by a comma is accepted.

### Delete

Using a DELETE request with the tunnel id allows terminating a tunnel.

```
CLIENTID=2ba9174e-640e-4694-ad35-34a2d6f3986b
TUNNELID=1 
curl -u admin:foobaz -X DELETE "http://localhost:3000/api/v1/sessions/$CLIENTID/tunnels/$TUNNELID"
```