---
title: 'API Authentication'
weight: 2
slug: api-authentication
aliases:
  - /docs/no02-api-auth.html
  - /docs/content/get-started/no02-api-auth.md
---

{{< toc >}}

## Authentication Mechanisms

The Rportd API support three ways of authentication.

1. HTTP Basic Auth with username and password
2. Bearer Token Auth
3. HTTP Basic Auth with username and personal API Token

### HTTP Basic Auth with username and password

The API claims to be REST compliant. Submitting credentials on each request using an HTTP basic auth header is therefore
possible, for example:

```shell
curl -s -u admin:foobaz http://localhost:3000/api/v1/clients|jq
```

With the two-factor authentication enabled, HTTP basic authentication with a username and user's password stops working.
But you can create an API token per user (see below) to activate HTTP basic auth again.

### Bearer Token Auth

Using HTTP Basic auth you can request a token at [`login` endpoint](https://apidoc.rport.io/master/#tag/Login) to
authenticate further requests with a token.
Example:

```shell
curl -s -u admin:foobaz http://localhost:3000/api/v1/login|jq
{
 "data": {
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIxMzI2MDU3MjgzMTA4OTc4NTg1OSJ9.6HSANk3aRleJbAMvfJhUc4grieupRdfU62MMX_L6wEA"
 }
}
```

The token has a default lifetime of 600 seconds(10 minutes). Using the query parameter `token-lifetime=3600`(in seconds)
you can request a defined lifetime.

Having a valid token you can execute requests, using an `Authorization: Bearer: <TOKEN>` header. For example:

```shell
# Get and store the token
curl -s -u admin:foobaz http://localhost:3000/api/v1/login?token-lifetime=3600|jq -r .data.token > .token

# Request using the stored token
curl -s -H "Authorization: Bearer $(cat .token)" http://localhost:3000/api/v1/clients|jq
```

Tokens are based on JWT. For your security, you should enter a unique `jwt_secret` into the `rportd.conf`. Do not use
the provided sample secret in a production environment.

Tokens can be issued only for certain pages. If you have 2fa enabled (either with a deliverable code or an Authenticator
app), rport issues a token, which can be used only in the `/verify-2fa` API. This ensures that your login attempt is
connected to the 2fa code verification. On the other hand, tokens which are issued after successful 2fa code validation
cannot be used to call the `/verify-2fa` API.

### HTTP Basic Auth with username and personal API Token

For the integration of third-party applications or for the development of scripts RPort supports the creation of
personal API tokens. These tokens can be used for HTTP basic authentication.

Users must submit the personal API token instead of the password, for example:

```shell
curl -s -u admin:e83d40e4-e237-43d6-bb99-35972ded631b http://localhost:3000/api/v1/clients|jq
```

Prior to RPort 0.9.11 each user could have only a single API token. Starting with 0.9.11 users can have an unlimited
number of API token. Tokens have a scope and an expiry date.

To generate personal API token navigate to the `Settings` -> `API Tokens` on the user interface, or generate tokens
[using the API](https://apidoc.rport.io/master/#tag/Profile-and-Info/operation/MetTokenPost).

## Two-Factor Auth

If you want an extra layer of security, you can enable 2FA. It allows you to confirm your login with a verification code
sent by a chosen delivery method.

Supported delivery methods:

1. email (requires [SMTP setup](/docs/content/get-started/no15-messaging.md#smtp))
2. [pushover.net](https://pushover.net) (requires [Pushover setup](/docs/content/get-started/no15-messaging.md#pushover))
3. Custom [script](/docs/content/get-started/no15-messaging.md#script)
4. Custom [URL](/docs/content/get-started/no15-messaging.md#url)

By default, 2FA is disabled.

### How to enable 2FA?

Note: 2FA is not available if you use [a single static user-password pair](/docs/content/get-started/no02-api-auth.md#hardcoded-single-user).

1. Choose the desired delivery method. Enter the following lines to the `rportd.config` in the `[api]` section, for example:

   ```text
   two_fa_token_delivery = 'smtp'
   two_fa_token_ttl_seconds = 600
   ```

   Use either `'smtp'`, `'pushover'` or provide a path to a binary or script executable.

   `two_fa_token_ttl_seconds` is an optional param for a lifetime of 2FA verification code. By default, 600 seconds.

2. Set up a valid [SMTP](/docs/content/get-started/no15-messaging.md#smtp) or [Pushover](/docs/content/get-started/no15-messaging.md#pushover) config.
3. Your user-password store ([json file](/docs/content/get-started/no02-api-auth.md#user-file) or
   [DB table](/docs/content/get-started/no02-api-auth.md#database)) needs an additional field `two_fa_send_to`.
   It should hold an email or pushover user key that is used to send 2FA verification code to a user.
4. Your user's `two_fa_send_to` field needs to contain a valid email or pushover user key.

   When using an executable for delivery, `two_fa_send_to_type` can be used to specify how the `two_fa_send_to` is validated on changes.
   This setting is ignored when using SMTP or Pushover for token delivery.
   Use `2fa_send_to_type = 'email'` to accept only valid email address.
   Or use a regular expression, for example

   ```text
   2fa_send_to_type = 'regex'
   2fa_send_to_regex = '[a-z0-9]{10}'
   ```

5. Restart the server.

#### How to use it?

1. Using 2FA will disable HTTP basic auth on all API endpoints except [`/login`](https://apidoc.rport.io/master/#tag/Login).
   Login endpoints trigger sending 2FA verification code to a user. For example,

```shell
curl -s -u admin:foobaz http://localhost:3000/api/v1/login|jq
{
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFwIiwiSz.2dbLWzej7XqwAAWiQajBDPPO15Vz2VHDA",
    "two_fa": {
      "send_to": "my.email@gmail.com",
      "delivery_method": "email"
    }
  }
}
```

Copy the login token. Please note, that it cannot be used as a bearer token auth for all endpoints rather than `/verify-2fa`.
The login token has a limited time validity, which is defined in the `[api]` section of the `rportd.config` as
`totp_login_session_ttl` (default value is 10 minutes).

1. Wait for an email with `Rport 2FA` subject with a content like:

```shell
Verification code: 05Nfqm (valid 10m0s)
```

1. Verify this code using [`/verify-2fa`](https://apidoc.rport.io/master/#operation/Verify2faPost) endpoint and provide
   token that you received after login as Bearer Auth header (e.g. `Authorization: Bearer eyJhbGciOiJIUz...SNIP...SNAP`).
2. If login token is valid and the provided code is correct, this API returns an auth JWT token that can be further used
   for any requests as listed in [here](no02-api-auth.md#bearer-token-auth). For example:

```shell
curl -s http://localhost:3000/api/v1/verify-2fa -H "Content-Type: application/json" -X POST \
-H "Authorization: Bearer eyJhbGciO...snip...snap" \
--data-raw '{
"username": "admin",
"token": "05Nfqm"
}'|jq
{
  "data": {
    "token": "eyJhbGciOiJIUzI1Ni...snip...snap",
    "two_fa": null
  }
}
```

Please note that this token cannot be used to call `/verify-2fa` which requires a valid login token.

## Storing credentials, managing users

The Rportd can read user credentials from three different sources.

1. A "hardcoded" single user with a plaintext password
2. A user file with bcrypt encoded passwords
3. A database table with bcrypt encoded passwords

Which one you chose is an either-or decision. A mixed-mode is not supported.

If you are planning to manage API users through the API or if you want to manage users comfortably using the graphical
frontend, you must store users in a database.

### Hardcoded single user

To use just a single user enter the following line to the `rportd.config` in the `[api]` section.

```text
auth = "admin:foobaz"
```

Quite simple. Now you can log in to the API using the username `admin` and the password `foobaz`.

### User File

If you want to have more than one user, create a json file with a structure as shown on the following examples.

{{< tabs "user-file" >}}
{{< tab "2FA off" >}}

```json
[
    {
        "username": "Admin",
        "password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu",
        "groups": ["Administrators", "Bunnies"]
    },
    {
        "username": "Bunny",
        "password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu",
        "groups": ["Bunnies"]
    }
]
```

{{< /tab >}}
{{< tab "2FA on" >}}

```json
[
    {
        "username": "Admin",
        "password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu",
        "groups": ["Administrators", "Bunnies"],
        "two_fa_send_to": "my.email@gmail.com"
    },
    {
        "username": "Bunny",
        "password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu",
        "groups": ["Bunnies"],
        "two_fa_send_to": "super.bunny@gmail.com"
    }
]
```

{{< /tab >}}
{{< tab "time based one time password (TotP) on" >}}

```json
[
    {
        "username": "Admin",
        "password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu",
        "groups": ["Administrators", "Bunnies"],
        "totp_secret": ""
    },
    {
        "username": "Bunny",
        "password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu",
        "groups": ["Bunnies"],
        "totp_secret": ""
    }
]
```

{{< /tab >}}
{{< /tabs >}}
Please note, that the values for `totp_secret` field are added when you use `/me/totp-secret` api.

Using `/var/lib/rport/api-auth.json` is a good choice.

Enter the following line to your `rportd.config` in the `[api]` section.

```text
auth_file = "/var/lib/rport/api-auth.json"
```

Make sure no other auth option is enabled.
Reload rportd to activate the changes.

The file is read only on start or reload `kill -SIGUSR1 <pid>`. Changes to the file, while rportd is running, have no effect.

To generate bcrypt hashes use for example the command `htpasswd` from the Apache Utils.

```shell
htpasswd -nbB password 'Super-Secret$Passw0rD'
password:$2y$05$Wgzg0fwtiCNYfP69k2uYKuYbmmFtd5RPK7W7mkgemuGkfXB2kgcdW
```

Copy the second part after the colon to the `api-auth.json` file. This is the hash of the password.

### Database

If you want to integrate rport into and existing user base or if you want to implement some kind of registration,
reading credentials from a database might be handy. Rport has no special demands on the database or the table layout.

The password must be bcrypt-hashed.

To use the database authentication you must set up a global database connection in the `[database]` section of `rportd.config` first.
Only MySQL/MariaDB and SQLite3 are supported at the moment.
The [example config](https://github.com/realvnc-labs/rport/blob/master/rportd.example.conf) contains all
explanations on how to set up the database connection.

Having the database set up, enter the following two lines to the `[api]` section of the `rportd.config` to specify the table names.

```toml
[api]
  #auth = "admin:foobaz" <-- Must be disabled
  auth_user_table = "users"
  auth_user_table = "users"
  auth_group_table = "groups"
  group_details_table = "group_details"
```

Reload rportd to apply all changes.

{{< tabs "db-tables" >}}
{{< tab "MySQL/Maria DB" >}}

Set up the database details in the `rportd.conf`:

```toml
[database]
  ## For MySQL or MariaDB.
  db_type = "mysql"

  ## Only for MySQL/Mariadb, ignored for Sqlite.
  db_host = "localhost:3306"
  #db_host = "socket:/var/run/mysqld/mysqld.sock"

  ## Credentials, only for MySQL/Mariadb, ignored for Sqlite.
  db_user = "rport"
  db_password = "rport"

  ## For MySQL/MariaDB name of the database.
  db_name = "rport"
```

Create tables.

```sql
CREATE TABLE `users` (
  `username` varchar(150) NOT NULL,
  `password` varchar(255) NOT NULL,
  `password_expired` bool NOT NULL default false,
  `two_fa_send_to` varchar(150),
  `token` varchar(128) default NULL,
  `totp_secret` longtext,
  UNIQUE KEY `username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
CREATE TABLE `groups` (
  `username` varchar(150) NOT NULL,
  `group` varchar(150) NOT NULL,
  UNIQUE KEY `username_group` (`username`,`group`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
CREATE TABLE `group_details` (
     `name` varchar(150) NOT NULL,
     `permissions` longtext DEFAULT '{}',
     UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
```

If your database was created prior to Version 0.9.1

```sql
ALTER TABLE `users` ADD `password_expired` bool NOT NULL DEFAULT(false);
```

{{< /tab >}}
{{< tab "SQLite" >}}
Enter the following line to the `rportd.conf` file in `[database]` section:

```toml
[database]
  db_type = "sqlite"
  db_name = "/var/lib/rport/database.sqlite3"
```

Create the database and set the ownership. Restart rport afterwards.

```shell
touch /var/lib/rport/database.sqlite3
chown rport:rport /var/lib/rport/database.sqlite3
systemctl restart rportd
```

connect to the database and create the tables. Change column types and lengths to your needs.

```shell
sqlite3 /var/lib/rport/database.sqlite3
SQLite version 3.31.1 2020-01-27 19:55:54
Enter ".help" for usage hints.
sqlite>
```

```sql
CREATE TABLE "users" (
  "username" TEXT(150) NOT NULL,
  "password" TEXT(255) NOT NULL,
  "password_expired" BOOLEAN NOT NULL CHECK (password_expired IN (0, 1)) DEFAULT 0,
  "token" TEXT(36) DEFAULT NULL,
  "two_fa_send_to" TEXT(150),
  "totp_secret" TEXT
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
CREATE TABLE "group_details" (name TEXT, permissions TEXT);
CREATE UNIQUE INDEX "main"."group_details_name" ON "group_details" ("name" ASC);
CREATE TABLE "group_details" (
    "name" TEXT(150) NOT NULL,
    "permissions" TEXT DEFAULT "{}"
    "tunnels_restricted" TEXT DEFAULT "{}"
    "commands_restricted" TEXT DEFAULT "{}"
);
CREATE UNIQUE INDEX "main"."name" ON "group_details" (
    "name" ASC
);
```

If your database was created prior to Version 0.9.1

```sql
ALTER TABLE `users` ADD `password_expired` BOOLEAN NOT NULL CHECK (password_expired IN (0, 1)) DEFAULT 0;
```

Sqlite does not print any confirmation. To confirm your tables have been created execute:

```text
sqlite> SELECT name FROM sqlite_master WHERE type='table';
users
groups
group_details
```

{{< /tab >}}
{{< /tabs >}}

Insert the first user:
{{< tabs "create-user-sqlite" >}}
{{< tab "2FA off" >}}

```sql
INSERT INTO users VALUES('admin','$2y$05$zfvuP4PvjsNWTqRFLdswEeRzETE2KiZONJQyVn7T3ZV5qcYAlmNWO');
INSERT INTO groups VALUES('admin','Administrators');
```

{{< /tab >}}
{{< tab "2FA on" >}}

```sql
INSERT INTO users VALUES('admin','$2y$05$zfvuP4PvjsNWTqRFLdswEeRzETE2KiZONJQyVn7T3ZV5qcYAlmNWO','my.email@gmail.com');
INSERT INTO groups VALUES('admin','Administrators');
```

{{< /tab >}}
{{< /tabs >}}

This creates a user `admin` with the password `password`. To use another password, create the appropriate bcrypt hash
[here](https://bcrypt-generator.com/) or use `htpasswd` on the command line.

### Extended group permissions additional fields

To enable the extended group permissions feature, the database must be upgraded with the following SQL statement:

```sql
ALTER TABLE `group_details` ADD COLUMN `tunnels_restricted` TEXT DEFAULT '{}';
ALTER TABLE `group_details` ADD COLUMN `commands_restricted` TEXT DEFAULT '{}';
```

Upon start, the "Extended group permissions" feature will be in trial mode, if the fields are present in the database, but there is no Plus license installed.

### API Usage examples

To verify the user is able to authenticate execute:

```shell
curl -Ss http://localhost:3000/api/v1/users -u admin:password|jq
{
  "data": [
    {
      "username": "admin",
      "groups": [
        "Administrators"
      ]
    }
  ]
}
```

Create a new user:

```shell
curl -Ss -X POST http://localhost:3000/api/v1/users \
-u admin:password \
-H "content-type:application/json" \
--data-raw '{
  "username": "Willy",
  "password": "pass1234",
  "groups": [
    "Administrators"
  ]
}'
```

Change the password of an existing user:

```shell
curl -Ss -X PUT https://localhost/api/v1/users/Willy \
-u admin:password \
-H "content-type:application/json" \
--data-raw '{"password": "4321ssap"}'
```

### Manage user from the command line

Starting with RPort 0.9.11 the ability to manage users from the command line has been introduced. The allows adding
users or changing passwords independent of the underlying storage mechanism.

{{< hint type=important title="don't execute from root account" >}}
When managing users from the command line, rportd will directly write to the configured files or databases. The command
line interface is not a wrapper for API calls. Therefor it's highly recommended to use the cli only from the user
account that rportd is using, usually `rport`.  
💣 By executing `rportd user [command]` from the root account you run **the risk of messing up file permissions** and make the files
unreadable for the running daemon process.

💁‍♂️ Change the user account with `su - rport -s /bin/bash` first.

{{< /hint >}}

To learn more about user management via the cli, execute `rportd user help`, which will display the following messages:

```text
Add, change or delete api users

Usage:
  rportd user [command]

Available Commands:
  add         add a new user
  change      change a user
  delete      delete a user

Flags:
  -h, --help              help for user
  -u, --username string   username [required]

Global Flags:
  -c, --config string   location of the config file

Use "rportd user [command] --help" for more information about a command.
```

To change the password of an existing user, execute

```shell
rportd user change -u <USERNAME> -p -c /etc/rport/rportd.conf
```

## Enabling 2FA with an Authenticator app (TotP auth)

You can enable 2FA with an authenticator app e.g. [Google Authenticator](https://play.google.com/store/apps/details?id=com.google.android.apps.authenticator2&hl=de&gl=US)
or [Microsoft Authenticator](https://www.microsoft.com/de-de/security/mobile-authenticator-app?rtc=1).

TotP authentication works similar to the mentioned 2FA with one difference: the 2fa code is generated by an authenticator
app rather than sent by email or sms. To make sure that the code generated with a client app is consistent with the server
you should generate a totP secret key, which is stored on the server and can be used to validate totP client codes.

### (1) activate

To activate 2FA based on [Time-based One Time Password](https://en.wikipedia.org/wiki/Time-based_One-time_Password_algorithm)
you should set `totp_enabled` option to true in the `[api]` configuration section.
Please note, that `totp_enabled` option cannot be combined with the email or sms based 2FA auth enabled (see `two_fa_token_delivery` option).

Another limitation is that you cannot use this auth method with [a single static user-password pair](no02-api-auth.md#hardcoded-single-user).

If you use a mysql/sqlite database for storing users data, you should also have `totp_secret` column there
(see the sql queries above).

Use `/login` API to pass the first factor authentication. There you will get a login token which you can use to create a
secret key as well as validate the one time code generated by your authenticator app.

```json
{
    "data": {
        "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InVzZXIiLCJqdGkiOiIxOTA2MTc2.LzSZDGo0vlsDi-Puy3w_vVZab50",
        "two_fa": {
            "send_to": "",
            "delivery_method": "totp_authenticator_app",
            "totp_key_status": "pending"
        }
    }
}
```

Note, that `delivery_method` has `totp_authenticator_app` value which indicates, that the server has totp authentication
enabled. The `totp_key_status` field indicates the status of a TotP secret key. If user hasn't created it yet, the status
will be `pending` and `exists` otherwise.

Note, that you can use this token only for `/verify-2fa` endpoint and for creating a new secret `/me/totp-secret`.
The reason for this, that 2FA requires additional authentication step by providing one time password generated by an
Authenticator app.

### (2) generate secret

To generate a new secret send a POST request to the `/me/totp-secret` API with the Bearer Authorization header
containing the token you got from the `/login` endpoint:
`curl -s -XPOST -H "Authorization: Bearer eyJhbGc...SNIP...SNAP" http://localhost:3000/api/v1/me/totp-secret`
As a result you will get a secret key in text format and a qr code as a base64 encoded png image e.g.:
`{ "secret": "54E4WYG5XSNZ37KI4CLILAVZKCMZ5MY7", "qr": "iVBORw0...snip...snap" }`

The secret key will be stored in the users' database for the current user and can only be used with the combination with his login.
Please note that you can generate a secret key only once, so if a user has it already, the creation attempt will be rejected.
Please write down the returned secret key or scan the generated qr code, otherwise you might lose access to your account.

If you've lost it, you have the following possibilities to recover access to your account:

- if you have access to another Administrator account, you can delete totP secret of any user including yourself by using totp administrative api (see below)
- if you have database access, you can erase value of `totp_secret` column in the `users` table of the corresponding user e.g.:

```sql
  UPDATE users SET totp_secret = '' WHERE `username` = 'your@mail.com';
```

- if you still have a valid auth token (the one you got from the second factor auth check from `/verify-2fa`), you can
- call `/me/totp-secret` endpoint with DELETE method to remove totP secret of the current user. Because of security reasons,
- you cannot use a login token here (the one you got from `/login` endpoint).

To display the qr code as an image you can use following script:

```shell
echo '<base64-code-of-image>' | base64 --decode > qr.png
open qr.png
```

You can open `gr.png` image and scan it with your authenticator app. If everything works well, you will see one time
passwords generated every 30 seconds. After activation, you will see Rport account in your Authenticator app. You can
change this name by changing configuration key `totp_account_name` to a name of your choice.

### (3) send one-time tokens

Now you can call `/verify-2fa` endpoint to pass second factor auth. For that you would need login token from the above
as well as code from the previous paragraph.

```shell
curl -s http://localhost:3000/api/v1/verify-2fa -H "Content-Type: application/json" -X POST \
-H "Authorization: Bearer eyJhbGciOi...snip..snap" \
--data-raw '{
"username": "admin",
"token": "123334"
}'|jq
{
  "data": {
    "token": "eyJhbGciOiJIUzI1Ni...snip...snap",
    "two_fa": null
  }
}
```

User has a limited time to provide an Authenticator's code after login. This time is set in `totp_login_session_ttl` option.

If a user provides valid login and password, waits more than `totp_login_session_ttl` time and provides a valid totp code,
his login attempt will be rejected. You're flexible in selecting ms, seconds, minutes and hours for the `totp_login_session_ttl`
option, so you can set it as e.g. "3600000ms", "3600s" or "60m" or "1h".

### (4) manage secrets through the API

You can manage totp secret key with the following endpoints:

Delete a totP secret of the current user:

```shell
curl -s http://localhost:3000/api/v1/me/totp-secret -H "Content-Type: application/json" -X DELETE \
-H "Authorization: Bearer eyJhbGciOiJIU...snip...snap"
```

Read an existing totP secret of the current user:

```shell
curl -s http://localhost:3000/api/v1/me/totp-secret -H "Content-Type: application/json" -X GET \
-H "Authorization: Bearer eyJhbGciOiJIUz...snip...snap"
```

Read an existing totP secret of the current user:

```shell
curl -s http://localhost:3000/api/v1/me/totp-secret -H "Content-Type: application/json" -X POST \
-H "Authorization: Bearer eyJhbGciOiJIUz...snip...snap"
```

{{< hint type=note >}}
Note, that for all above endpoints you can use JWT token that you get after the second factor authentication from
`/verify-2fa` endpoint. Additionally, you can use login token to create a new secret if user has no other secret
(see above).
{{< /hint >}}

All totP secret management endpoints are available only if `totp_enabled` option is true.

If you have an Administrator account, you can also delete totP secret of all users including yourself. The only
limitation is that as with all other management endpoints user must pass 2fa to get a valid token from the
`/verify-2fa` endpoint.

```shell
curl -s http://localhost:3000/api/v1/users/no@mail.com/totp-secret -H "Content-Type: application/json" -X DELETE \
-H "Authorization: Bearer eyJhbGciOiJIUz...snip...snap"
```

## Delegated authentication

Staring with rportd 0.5.0 you can delegate the authentication to a reverse proxy. This allows you to use a variety of
authentication backends for example such supported by Netscaler, Keycloak, the
[Apache Auth Plugins](https://httpd.apache.org/docs/2.4/howto/auth.html) or by the [Caddy Auth Portal](https://github.com/greenpau/caddy-auth-portal).

If the reverse proxy sends a specific header and a header that includes a username in the value, rport considers the
user as authenticated.
Using the delegated authentication does not liberate you from performing a `/api/v1/login` request to retrieve the JWT
and sending this JWT on all requests.

If you are using the RPort frontend, you can send authenticated useser directly to
`https://<SERVER-FQDN>/auth/?auto-login=1&remember-me=24h`. This caused the login form to be auto-submitted.

To enable the delegated authentication, activate the following settings in your `rportd.conf`.

```text
  ## The rport server can treat all requests as pre-authenticated by a reverse proxy based on a http header.
  ## This option is enabled if auth_header is set.
  ## If the header exists, the request is considered valid and a session is created.
  ## Inside the same or a different header, the username must be submitted.
  auth_header = "My-User"
  user_header = "My-User"
  ## If the user doesn't exist yet, it can be created on-the-fly.
  ## Disabled by default
  create_missing_users = true
  ## If users are created on-the-fly to which user group do they belong?
  default_user_group = "Administrators"
```

### Simple example with Caddy basic authentication

Below you have a simple configuration file for the Caddy web server. It assumes your rportd API is listening on localhost:3000.

```text
http://rport.example.com:8080 {
 # Forward all requests to Rport
 reverse_proxy * 127.0.0.1:3000 {
  header_up My-User jopapa
 }
 log {
  output file /tmp/proxy.log
 }
 basicauth /api/v1/login {
  # require password foobaz
  jomama JDJhJDE0JEkycVRhTlkwTHZxekZsZTViVTRsMy5zLjE5ZVdWQTYyWnRJeC9tYm5pOVRmOVliNUVFazUu
 }
}
```

If you request the JWT you must specify the username and password expected by the reverse proxy (`jomama` + `foobaz`).
The rportd user backend is not asked.

`curl -u jomama:foobaz "http://rport.example.com:8080/api/v1/login?token-lifetime=9999" -v -o auth.json`

If you do the request via the web user interface you also need to enter the username and password specified by the proxy.

In this example, you must log in as user "jomama" but rport treats you as "jopapa" because the reverse proxy sends a
static hard coded username in the `My-User` header.

{{< hint type=warning >}}
Note that the reverse proxy must only require authentication on the `/api/v1/login` URI. If you require basic
authentication for all resources, the frontend will break because ajax requests are performed in the background with
the default authentication bearer header that will overwrite potential authentication basic headers.
{{< /hint >}}

### Real-world use case

The above example is just a demonstration. In a real-world example, the user would visit an authentication portal first
that will place a cookie. Based on that cookie, the reverse proxy will deny or forward requests to the rportd backend.

The following Nginx example simulates an authentication portal.

```text
# Reverse proxy
#
server {
 listen 8888 default_server;
 listen [::]:8888 default_server;
 server_name _;
        # Reject all request if cookie is missing
        if ($http_cookie !~ 'secretvalue') {
            return 401;
        }

 location / {
                # Proxy forward to rportd
                proxy_pass http://127.0.0.1:3000/;
                proxy_set_header My-User gagaman;
 }
}

# Auth portal
#
server {
        listen 8889 default_server;
        listen [::]:8889 default_server;

        root /var/www/html;

        # Add index.php to the list if you are using PHP
        index index.html index.htm index.nginx-debian.html;

        server_name _;

        location / {
              add_header Set-Cookie "letmein=secretvalue;max-age=3153600000;path=/;domain=.example.com";
        }
}

```

{{< hint type=warning >}}
Do not use any of the examples in production as they all are not using encryption and all your authentication data would
be transferred plain text, that means easily sniffable.
{{< /hint >}}
