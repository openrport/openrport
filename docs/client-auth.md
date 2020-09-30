## Client Authentication
The Rportd can read client credentials from three different sources. 
1. A "hardcoded" single pair of client credentials
2. A file with client credentials
3. A database table (NOT SUPPORTED YET)

Which one you choose is an either-or decision. A mixed-mode is not supported.
If (2) or (3) is enabled then managing clients can be done via the API.

### Using static credentials
To use just a single pair of client credentials enter the following line to the server config(`rportd.config`) in the `[server]` section.
```
auth = "admin:123456"
```
Make sure no other auth option is enabled. 
Reload rportd to activate the changes. 
Quite simple. Now you can run a client using the username `admin` and the password `123456`. It can be done in two ways:
1. Use a command arg: `--auth admin:123456`
2. Enter the following line to the client config(`rport.config`) in the `[client]` section. 
```
auth = "admin:123456"
```

### Using a file
If you want to have more than one pair of client credentials, create a json file with the following structure.
```
{
    "admin":   "123456",
    "client1": "yienei5Ch",
    "client2": "ieRi1Noo2"
}
``` 
Using `/var/lib/rport/client-auth.json` or `C:\Program Files\rport\client-auth.json` is a good choice. 

Enter the following line to your `rportd.config` in the `[server]` section.
```
auth_file = "/var/lib/rport/client-auth.json"           # Linux
auth_file = "C:\Program Files\rport\client-auth.json"   # Windows
```
Make sure no other auth option is enabled. 
Reload rportd to activate the changes. 

The file is read only on start. Changes to the file, while rportd is running, have no effect.

### Using a database table
(NOT SUPPORTED YET)

### Manage client credentials via the API

The `/clients` endpoint allows you to manage clients and credentials through the API.
This option is disabled, if you use a single static username password pair.
If you want to delegate the management of client to a third-party app writing directly to the auth-file or the database, consider turning the endpoint off by activating the following lines in the `rportd.conf`.
```
## If you want to delegate the creation and maintenance to an external tool
## you should turn {auth_write} off. 
## The API will reject all writing access to the client auth with HTTP 403.
## Applies only to auth_file and auth_table
## Default: true
auth_write = false
```

List all client credentials.

```
curl -s -u admin:foobaz http://localhost:3000/api/v1/clients|jq
{
  "data": [
    {
      "id": "client1",
      "password": "yienei5Ch"
    },
    {
      "id": "client2",
      "password": "ieRi1Noo2"
    }
  ]
}
```

Add a new client

```
curl -X POST 'http://node2.rport.io:3000/api/v1/clients' \
-u admin:foobaz \
--data-raw '{ 
    "id":"client3",
    "password":"hase243345"
}'
```

Bear in mind, that the auth-json file is written asynchronously with a delay. To avoid concurrent writing caused by parallel requests, new clients are stored in memory and flushed to disk every 5 seconds. You can change this interval in the `rportd.conf`.

Caution: As writing the client-auth file happens asynchronously in the background, errors are not reported to the API clients. They are logged in `rportd.log`. Make sure the rport user has write access to the file.