# API Authentication

## Authentication Mechanisms
The Rportd API support two ways of authentication.
1. HTTP Basic Auth
2. Bearer Token Auth
3. Two-Factor Auth
### HTTP Basic Auth
The API claims to be REST compliant. Submitting credentials on each request using an HTTP basic auth header is therefore possible, for example
```
curl -s -u admin:foobaz http://localhost:3000/api/v1/clients|jq
```

With the two-factor authentication enabled, HTTP basic authentication with a username and user's password stops working. But you can create a static API token per user to activate HTTP basic auth again. Users must submit the personal API token instead of the password, for example
```
curl -s -u admin:e83d40e4-e237-43d6-bb99-35972ded631b http://localhost:3000/api/v1/clients|jq
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

# Request using the stored token
curl -s -H "Authorization: Bearer $(cat .token)" http://localhost:3000/api/v1/clients|jq
```

Rportd holds the tokens in memory. Restarting rportd deletes (expires) them all.

Tokens are based on JWT. For your security, you should enter a unique `jwt_secret` into the `rportd.conf`. Do not use the provided sample secret in a production environment.

### Two-Factor Auth
If you want an extra layer of security, you can enable 2FA. It allows you to confirm your login with a verification code sent by a chosen delivery method.
Supported delivery methods:
1. email (requires [SMTP setup](no15-messaging.md#smtp))
2. [pushover.net](https://pushover.net) (requires [Pushover setup](no15-messaging.md#pushover))
3. Custom [script](no15-messaging.md#script)

By default, 2FA is disabled.

#### How to enable 2FA?
1. Choose the desired delivery method. Enter the following lines to the `rportd.config` in the `[api]` section, for example:
   ```
   two_fa_token_delivery = 'smtp'
   two_fa_token_ttl_seconds = 600
   ```
   Use either `'smtp'`, `'pushover'` or provide a path to a binary or script executable.

   `two_fa_token_ttl_seconds` is an optional param for a lifetime of 2FA verification code. By default, 600 seconds.


2. Set up a valid [SMTP](no15-messaging.md#smtp) or [Pushover](no15-messaging.md#pushover) config.
3. 2FA is not available if you use [a single static user-password pair](no02-api-auth.md#hardcoded-single-user).
4. Your user-password store ([json file](no02-api-auth.md#user-file) or [DB table](no02-api-auth.md#database)) needs an additional field `two_fa_send_to`.
   It should hold an email or pushover user key that is used to send 2FA verification code to a user.
5. Your user's `two_fa_send_to` field needs to contain a valid email or pushover user key.

   When using an executable for delivery, `two_fa_send_to_type` can be used to specify how the `two_fa_send_to` is validated on changes.
   This setting is ignored when using SMTP or Pushover for token delivery.
   Use `2fa_send_to_type = 'email'` to accept only valid email address.
   Or use a regular expression, for example
   ```
   2fa_send_to_type = 'regex'
   2fa_send_to_regex = '[a-z0-9]{10}'
   ```
6. Restart the server.

#### How to use it?
1. Using 2FA will disable HTTP basic auth on all API endpoints except [`/login`](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/cloudradar-monitoring/rport/master/api-doc.yml#/Login/get_login).
Login endpoints trigger sending 2FA verification code to a user. For example,
```
curl -s -u admin:foobaz http://localhost:3000/api/v1/login|jq
{
  "data": {
    "token": null,
    "two_fa": {
      "send_to": "my.email@gmail.com",
      "delivery_method": "email"
    }
  }
}
```
2. Wait for an email with `Rport 2FA` subject with a content like:
```
Verification code: 05Nfqm (valid 10m0s)
```
3. Verify this code using [`/verify-2fa`](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/cloudradar-monitoring/rport/master/api-doc.yml#/Login/post_verify_2fa) endpoint.
It returns an auth JWT token that can be further used for any requests as listed in [here](no02-api-auth.md#bearer-token-auth). For example,
```
curl -s http://localhost:3000/api/v1/verify-2fa -H "Content-Type: application/json" -X POST \
--data-raw '{
"username": "admin",
"token": "05Nfqm"
}'|jq
{
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFkbWluIiwianRpIjoiMTcwMTc0MjY4MTkxNTQwMDA2NjQifQ.IhOK2leOdCXK5jvAO9aWEcpZ0kanpSkSbRpufha8soc",
    "two_fa": null
  }
}
```

## Storing credentials, managing users
The Rportd can read user credentials from three different sources.
1. A "hardcoded" single user with a plaintext password
2. A user file with bcrypt encoded passwords
3. A database table with bcrypt encoded passwords

Which one you chose is an either-or decision. A mixed-mode is not supported.

If you are planning to manage API users through the API or if you want to manage users comfortably using the graphical frontend, you must store users in a database.

### Hardcoded single user
To use just a single user enter the following line to the `rportd.config` in the `[api]` section.
```
auth = "admin:foobaz"
```
Quite simple. Now you can log in to the API using the username `admin` and the password `foobaz`.

### User File
If you want to have more than one user, create a json file with the following structure.
:::: code-group
::: code-group-item 2FA off
```json
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
:::
::: code-group-item 2FA on
```json
[
    {
        "username": "Admin",
        "password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu",
        "groups": [
            "Administrators",
            "Bunnies"
        ],
        "two_fa_send_to": "my.email@gmail.com"
    },
    {
        "username": "Bunny",
        "password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu",
        "groups": [
            "Bunnies"
        ],
        "two_fa_send_to": "super.bunny@gmail.com"
    }
]
```
:::
::::

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
htpasswd -nbB password 'Super-Secret$Passw0rD'
password:$2y$05$Wgzg0fwtiCNYfP69k2uYKuYbmmFtd5RPK7W7mkgemuGkfXB2kgcdW
```
Copy the second part after the colon to the `api-auth.json` file. This is the hash of the password.

htpasswd.exe for Windows can be extracted from this [ZIP file](https://de.apachehaus.com/downloads/httpd-2.4.46-o111g-x86-vc15.zip) or this [ZIP File](https://www.apachelounge.com/download/VS16/binaries/httpd-2.4.46-win64-VS16.zip) or use this [Online Hash Generator](https://bcrypt-generator.com/).

### Database
If you want to integrate rport into and existing user base or if you want to implement some kind of registration, reading credentials from a database might be handy.
Rport has no special demands on the database or the table layout.

The tables must be created manually.

Each time a http basic auth request is received, rport executes these two queries.
:::: code-group
::: code-group-item 2FA off
```
SELECT username,password FROM {user-table} WHERE username='{username}' LIMIT 1;
SELECT DISTINCT(group) FROM {group-table} WHERE username='{username}';
```
:::
::: code-group-item 2FA on
```
SELECT username,password,two_fa_send_to FROM {user-table} WHERE username='{username}' LIMIT 1;
SELECT DISTINCT(group) FROM {group-table} WHERE username='{username}';
```
:::
::::
The password must be bcrypt-hashed.

To use the database authentication you must setup a global database connection in the `[database]` section of `rportd.config` first.
Only MySQL/MariaDB and SQLite3 are supported at the moment. The [example config](https://github.com/cloudradar-monitoring/rport/blob/master/rportd.example.conf) contains all explanations on how to set up the database connection.

Having the database set up, enter the following two lines to the `[api]` section of the `rportd.config` to specify the table names.
```
auth_user_table = "users"
auth_group_table = "groups"
```
Reload rportd to apply all changes.

#### MySQL Example
Create table. Change column types and lengths to your needs.
:::: code-group
::: code-group-item 2FA off
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
::: code-group-item 2FA on
```mysql
CREATE TABLE `users` (
  `username` varchar(150) NOT NULL,
  `password` varchar(255) NOT NULL,
  `two_fa_send_to` varchar(150),
  `token` char(36) default NULL,
  UNIQUE KEY `username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
CREATE TABLE `groups` (
  `username` varchar(150) NOT NULL,
  `group` varchar(150) NOT NULL,
  UNIQUE KEY `username_group` (`username`,`group`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
```
:::
::::


#### SQLite Example
Enter the following line to the `rportd.conf` file in the `[api]` and `[database]` section:
```
[api]
  #auth = "admin:foobaz" <-- Must be disabled
  auth_user_table = "users"
  auth_group_table = "groups"
[database]
  db_type = "sqlite"
  db_name = "/var/lib/rport/database.sqlite3"
```

Create the database and set the ownership. Restart rport afterwards.
```
touch /var/lib/rport/database.sqlite3
chown rport:rport /var/lib/rport/database.sqlite3
systemctl restart rportd
```

Now connect to the database and create the tables.
*Change column types and lengths to your needs.*
```
sqlite3 /var/lib/rport/database.sqlite3
SQLite version 3.31.1 2020-01-27 19:55:54
Enter ".help" for usage hints.
sqlite>
```

:::: code-group
::: code-group-item 2FA off
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
::: code-group-item 2FA on
```sqlite
CREATE TABLE "users" (
  "username" TEXT(150) NOT NULL,
  "password" TEXT(255) NOT NULL,
  "token" TEXT(36) DEFAULT NULL,
  "two_fa_send_to" TEXT(150)
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

Sqlite does not print any confirmation. To confirm your tables have been created execute:
```
sqlite> SELECT name FROM sqlite_master WHERE type='table';
users
groups
```

Now insert the first user:
:::: code-group
::: code-group-item 2FA off
```
sqlite> INSERT INTO users VALUES('admin','$2y$05$zfvuP4PvjsNWTqRFLdswEeRzETE2KiZONJQyVn7T3ZV5qcYAlmNWO');
sqlite> INSERT INTO groups VALUES('admin','Administrators');
```
:::
::: code-group-item 2FA on
```
sqlite> INSERT INTO users VALUES('admin','$2y$05$zfvuP4PvjsNWTqRFLdswEeRzETE2KiZONJQyVn7T3ZV5qcYAlmNWO','my.email@gmail.com');
sqlite> INSERT INTO groups VALUES('admin','Administrators');
```
:::
::::

This creates a user `admin` with the password `password`. To use another password, create the appropriate bcrypt hash [here](https://bcrypt-generator.com/).

#### API Usage examples

To verify the user is able to authenticate execute:
```
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
```
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
```
curl -Ss -X PUT https://localhost/api/v1/users/Willy \
-u admin:password \
-H "content-type:application/json" \
--data-raw '{"password": "4321ssap"}'
```

## Enabling 2FA with an Authenticator app
You can enable 2FA with an authenticator app e.g. [Google Authenticator](https://play.google.com/store/apps/details?id=com.google.android.apps.authenticator2&hl=de&gl=US) 
or [Microsoft Authenticator](https://www.microsoft.com/de-de/security/mobile-authenticator-app?rtc=1).

To activate 2FA  based on [Time-based One Time Password](https://en.wikipedia.org/wiki/Time-based_One-time_Password_algorithm)
you should provide a secret key in `totp_secret` option in `[api]` configuration section.

You can generate secret key by executing following command with the rportd binary:
```
rportd totp
```

You can copy the generated code to an Authenticator app of your choice or you can also generate a qr code for better usability:

```
rportd totp --totp-image bin/qr.jpg #to generate a qr code in jpeg format
rportd totp --totp-image bin/qr.png #to generate a qr code in png format
```

Copy the generated secret code to the `totp_secret` option in the rport configuration and use the image to add a account in an authenticator app.

Optionally you can change issuer and account name that will be encrypted in the gr code. Those values will be used to identify an account in an Authenticator app.

```
rportd totp --totp-image bin/qr.png --totp-account no@mail.me --totp-issuer mycomp
```

Time based one-time password 2FA auth method conflicts with a non-empty `two_fa_token_delivery` option, as rport won't know which method to use. In this case please make sure, that `two_fa_token_delivery` option is empty before providing a `totp_secret` option othewise an error will occur.

Please note that the generated secret key is associated with the generated qr code.  
It means that you cannot generate a qr code to an existing secret key or generate a secret key based on an existing qr image.

To verify one time passwords generated by an Authenticator application, you can call `verify-2fa` API and use the generated code as 2fa token.
Verify this code using [`/verify-2fa`](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/cloudradar-monitoring/rport/master/api-doc.yml#/Login/post_verify_2fa) endpoint.
   It returns an auth JWT token that can be further used for any requests as listed in [here](no02-api-auth.md#bearer-token-auth). For example,
```
curl -s http://localhost:3000/api/v1/verify-2fa -H "Content-Type: application/json" -X POST \
--data-raw '{
"username": "admin",
"token": "34976"
}'|jq
{
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFkbWluIiwianRpIjoiMTcwMTc0MjY4MTkxNTQwMDA2NjQifQ.IhOK2leOdCXK5jvAO9aWEcpZ0kanpSkSbRpufha8soc",
    "two_fa": null
  }
}
```

Before using `verify-2fa` API, users should call `login` API, otherwise they could just provide a valid code from an Authenticator app and bypass login.
This gives additional protection in cases if e.g. someone knows the one time password but doens't have login and password.

A user has a limited time to provide an Authenticator's code after login. This time is set in `totp_login_session_ttl` option.

If a user provides valid login and password, waits more than `totp_login_session_ttl` time and provides a valid totp code, his login attempt will be rejected.
You're flexible in selecting ms, seconds, minutes and hours for the `totp_login_session_ttl` option, so you can set it as e.g. "3600000ms", "3600s" or "60m" or "1h".
