## Client Authentication
The Rportd can read client auth credentials from three different sources.
1. A "hardcoded" single pair of client credentials
2. A file with client credentials
3. A database table

Which one you choose is an either-or decision. A mixed-mode is not supported.
If (2) or (3) is enabled then managing client auth credentials can be done via the API.

### Using static credentials
To use just a single pair of client auth credentials enter the following line to the server config(`rportd.config`) in the `[server]` section.
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
If you want to have more than one pair of client auth credentials, create a json file with the following structure.
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
auth_file = 'C:\Program Files\rport\client-auth.json'   # Windows
```
Make sure no other auth option is enabled.
Reload rportd to activate the changes.

The file is read only on start. Changes to the file, while rportd is running, have no effect.

### Using a database table
Clients auth credentials can be read from and written to a database table.

To use the database client authentication you must setup a global database connection in the `[database]` section of `rportd.conf` first.
Only MySQL/MariaDB and SQLite3 are supported at the moment. The [example config](../rportd.example.conf) contains all explanations on how to set up the database connection.

The tables must be created manually.

**MySQL/MariaDB**

```sql
CREATE TABLE `clients_auth` (
  `id` varchar(100) PRIMARY KEY,
  `password` varchar(100) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
```

**SQLite3**

```sql
CREATE TABLE `clients_auth` (
  `id` varchar(100) PRIMARY KEY,
  `password` varchar(100) NOT NULL
);
```

Having the database set up, enter the following to the `[server]` section of the `rportd.conf` to specify the table names.
```
auth_table = "clients_auth"
```
Reload rportd to apply all changes.


### Manage client credentials via the API

The [`/clients-auth` endpoint](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/cloudradar-monitoring/rport/master/api-doc.yml#/Rport%20Client%20Auth%20Credentials) allows you to manage client auth credentials through the API.
This option is disabled, if you use a single static username password pair.
If you want to delegate the management of client auth credentials to a third-party app writing directly to the auth-file or the database, consider turning the endpoint off by activating the following lines in the `rportd.conf`.
```
## If you want to delegate the creation and maintenance to an external tool
## you should turn {auth_write} off.
## The API will reject all writing access to the client auth with HTTP 403.
## Applies only to auth_file and auth_table
## Default: true
auth_write = false
```

List all client auth credentials.

```
curl -s -u admin:foobaz http://localhost:3000/api/v1/clients-auth|jq
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

Add a new client auth credentials

```
curl -X POST 'http://localhost:3000/api/v1/clients-auth' \
-u admin:foobaz \
-H 'Content-Type: application/json' \
--data-raw '{
    "id":"client3",
    "password":"hase243345"
}'
```
