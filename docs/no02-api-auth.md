# API Authentication

## Authentication Mechanisms
The Rportd API support two ways of authentication.
1. HTTP Basic Auth
2. Bearer Token Auth
### HTTP Basic Auth
The API claims to be REST compliant. Submitting credentials on each request using an HTTP basic auth header is therefore possible, for example
```
curl -s -u admin:foobaz http://localhost:3000/api/v1/clients|jq
```

### Bearer Token Auth
Using HTTP Basic auth you can request a token at [`login` endpoint](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/cloudradar-monitoring/rport/master/api-doc.yml#/default/get_login) to authenticate further requests with a token.
Example:
```
curl -s -u admin:foobaz http://localhost:3000/api/v1/login|jq
{
 "data": {
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIxMzI2MDU3MjgzMTA4OTc4NTg1OSJ9.6HSANk3aRleJbAMvfJhUc4grieupRdfU62MMX_L6wEA"
 }
}
```
The token has a default lifetime of 600 seconds(10 minutes). Using the query parameter `token-lifetime=3600`(in seconds) you can request a defined lifetime.

Having a valid token you can execute requests, using an `Authorization: Bearer: <TOKEN>` header. For example.
```
# Get and store the token
curl -s -u admin:foobaz http://localhost:3000/api/v1/login?token-lifetime=3600|jq -r .data.token > .token

# Request using the stored toeken
curl -s -H "Authorization: Bearer $(cat .token)" http://localhost:3000/api/v1/clients|jq
```

Rportd holds the tokens in memory. Restarting rportd deletes (expires) them all.

Tokens are based on JWT. For your security, you should enter a unique `jwt_secret` into the `rportd.conf`. Do not use the provided sample secret in a production environment.

## Storing credentials, managing users
The Rportd can read user credentials from three different sources.
1. A "hardcoded" single user with a plaintext password
2. A user file with bcrypt encoded passwords
3. A database table with bcrypt encoded passwords

Which one you chose is an either-or decision. A mixed-mode is not supported.

### Hardcoded single user
To use just a single user enter the following line to the `rportd.config` in the `[api]` section.
```
auth = "admin:foobaz"
```
Quite simple. Now you can log in to the API using the username `admin` and the password `foobaz`.

### User File
If you want to have more than one user, create a json file with the following structure.
```
[
    {
        "username": "Admin",
        "password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu",
        "groups": [
            "Administrators",
            "Bunnies"
        ]
    },
    {
        "username": "Bunny",
        "password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu",
        "groups": [
            "Bunnies"
        ]
    }
]
```
Using `/var/lib/rport/api-auth.json` or `C:\Program Files\rport\api-auth.json` is a good choice.

Enter the following line to your `rportd.config` in the `[api]` section.
```
auth_file = "/var/lib/rport/api-auth.json"           # Linux
auth_file = 'C:\Program Files\rport\api-auth.json'   # Windows
```
Make sure no other auth option is enabled.
Reload rportd to activate the changes.

The file is read only on start or reload `kill -SIGUSR1 <pid>`. Changes to the file, while rportd is running, have no effect.

To generate bcrypt hashes use for example the command `htpasswd` from the Apache Utils.
```
htpasswd -nbB password 'Super-Secrete$Passw0rD'
password:$2y$05$Wgzg0fwtiCNYfP69k2uYKuYbmmFtd5RPK7W7mkgemuGkfXB2kgcdW
```
Copy the second part after the colon to the `api-auth.json` file. This is the hash of the password.

htpasswd.exe for Windows can be extracted from this [ZIP file](https://de.apachehaus.com/downloads/httpd-2.4.46-o111g-x86-vc15.zip) or this [ZIP File](https://www.apachelounge.com/download/VS16/binaries/httpd-2.4.46-win64-VS16.zip) or use this [Online Hash Generator](https://bcrypt-generator.com/).

### Database
If you want to integrate rport into and existing user base or if you want to implement some kind of registration, reading credentials from a database might be handy.
Rport has no special demands on the database or the table layout.

The tables must be created manually.

Each time a http basic auth request is received, rport executes these two queries.
```
SELECT username,password FROM {user-table} WHERE username='{username}' LIMIT 1;
SELECT DISTINCT(group) FROM {group-table} WHERE username='{username}';
```
The password must be bcrypt-hashed.

To use the database authentication you must setup a global database connection in the `[database]` section of `rportd.config` first.
Only MySQL/MariaDB and SQLite3 are supported at the moment. The [example config](https://github.com/cloudradar-monitoring/rport/blob/master/rportd.example.conf) contains all explanations on how to set up the database connection.

Having the database set up, enter the following two lines to the `[api]` section of the `rportd.config` to specify the table names.
```
auth_user_table = "users"
auth_group_table = "groups"
```
Reload rportd to apply all changes.

#### Examples
Some simple example of a table layout.
Change column types and lengths to your needs.

:::: code-group
::: code-group-item MySQL
```mysql
CREATE TABLE `users` (
  `username` varchar(150) NOT NULL,
  `password` varchar(255) NOT NULL,
  UNIQUE KEY `username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
CREATE TABLE `groups` (
  `username` varchar(150) NOT NULL,
  `group` varchar(150) NOT NULL,
  UNIQUE KEY `username_group` (`username`,`group`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
```
:::
::: code-group-item SQLite3
```sqlite
CREATE TABLE "users" (
  "username" TEXT(150) NOT NULL,
  "password" TEXT(255) NOT NULL
);
CREATE UNIQUE INDEX "main"."username"
ON "users" (
  "username" ASC
);
CREATE TABLE "groups" (
  "username" TEXT(150) NOT NULL,
  "group" TEXT(150) NOT NULL
);
CREATE UNIQUE INDEX "main"."username_group"
ON "groups" (
  "username" ASC,
  "group" ASC
);
```
:::
::::
