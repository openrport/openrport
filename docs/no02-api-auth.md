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

Tokens are based on JWT. For your security, you should enter a unique `jwt_secret` into the `rportd.conf`. Do not use the provided sample secret in a production environment.

Tokens can be issued only for certain pages. If you have 2fa enabled (either with a deliverable code or an Authenticator app), rport issues a token, which can be used only in the `/verify-2fa` API. This ensures that your login attempt is connected to the 2fa code verification. On the other hand, tokens which are issued after successful 2fa code validation cannot be used to call the `/verify-2fa` API.

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
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFwIiwiSz.2dbLWzej7XqwAAWiQajBDPPO15Vz2VHDA",
    "two_fa": {
      "send_to": "my.email@gmail.com",
      "delivery_method": "email"
    }
  }
}
```
Copy the login token. Please note, that it cannot be used as a bearer token auth for all endpoints rather than `/verify-2fa`.
The login token has a limited time validity, which is defined in the `[api]` section of the `rportd.config` as `totp_login_session_ttl` (default value is 10 minutes).

2. Wait for an email with `Rport 2FA` subject with a content like:
```
Verification code: 05Nfqm (valid 10m0s)
```
3. Verify this code using [`/verify-2fa`](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/cloudradar-monitoring/rport/master/api-doc.yml#/Login/post_verify_2fa) endpoint and provide token that you received after login as Bearer Auth header (e.g. `Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFwIiwiSz.2dbLWzej7XqwAAWiQajBDPPO15Vz2VHDA`). If login token is valid and the provided code is correct,
this API returns an auth JWT token that can be further used for any requests as listed in [here](no02-api-auth.md#bearer-token-auth). For example,
```
curl -s http://localhost:3000/api/v1/verify-2fa -H "Content-Type: application/json" -X POST \
-H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InVzZXIiLCJqdGkiOiIxOTA2MTc2.LzSZDGo0vlsDi-Puy3w_vVZab50" \
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
Please note that this token cannot be used to call `/verify-2fa` which requires a valid login token.

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
::: time based one time password (TotP) on
```json
[
    {
        "username": "Admin",
        "password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu",
        "groups": [
            "Administrators",
            "Bunnies"
        ],
        "totp_secret": ""
    },
    {
        "username": "Bunny",
        "password": "$2y$10$ezwCZekHE/qxMb4g9n6rU.XIIdCnHnOo.q2wqqA8LyYf3ihonenmu",
        "groups": [
            "Bunnies"
        ],
        "totp_secret": ""
    }
]
```
Please note, that the values for `totp_secret` field are added when you use `/me/totp-secret` api.
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
::: code-group-item 2FA off, TotP off
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
::: totP on
```
SELECT username,password,totp_secret FROM {user-table} WHERE username='{username}' LIMIT 1;
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
::: code-group-item 2FA off, time based one time password (TotP) off
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
::: time based one time password (TotP) on
```mysql
CREATE TABLE `users` (
  `username` varchar(150) NOT NULL,
  `password` varchar(255) NOT NULL,
  `totp_secret` longtext,
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
::: code-group-item 2FA off, time based one time password (TotP) off
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
::: ::: time based one time password (TotP) on
```sqlite
CREATE TABLE "users" (
  "username" TEXT(150) NOT NULL,
  "password" TEXT(255) NOT NULL,
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

## Enabling 2FA with an Authenticator app (TotP auth)
You can enable 2FA with an authenticator app e.g. [Google Authenticator](https://play.google.com/store/apps/details?id=com.google.android.apps.authenticator2&hl=de&gl=US) 
or [Microsoft Authenticator](https://www.microsoft.com/de-de/security/mobile-authenticator-app?rtc=1).

TotP authentication works similar to the mentioned 2FA with one difference: the 2fa code is generated by an authenticator app rather than sent by email or sms. To make sure that the code generated with a client app is consistent with the server, you should generate a totP secret key, which is stored on the server and can be used to validate totP client codes.

1. To activate 2FA based on [Time-based One Time Password](https://en.wikipedia.org/wiki/Time-based_One-time_Password_algorithm)
you should set `totp_enabled` option to true in the `[api]` configuration section.
Please note, that `totp_enabled` option cannot be combined with the email or sms based 2FA auth enabled (see `two_fa_token_delivery` option).

Another limitation is that you cannot use this auth method with [a single static user-password pair](no02-api-auth.md#hardcoded-single-user).

If you use a mysql/sqlite database for storing users data, you should also have `totp_secret` column there (see the sql queries above).

2. Use `/login` API to pass first factor authentication. There you will get a login token which you can use to create a secret key as well as validate the one time code generated by your authenticator app.

```
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
Note, that `delivery_method` has `totp_authenticator_app` value which indicates, that the server has totp authentication enabled.
The `totp_key_status` field indicates the status of a TotP secret key. If user hasn't created it yet, the status will be `pending` and `exists` otherwise.

Note, that you can use this token only for `/verify-2fa` endpoint and for creating a new secret `/me/totp-secret`. The reason for this, that 2FA requires additional authentication step by providing one time password generated by an Authenticator app.

3. To generate a new secret send a POST request to the `/me/totp-secret` API with the Bearer Authorization header containing the token you got from the `/login` endpoint:

```
curl -s -XPOST -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InVzZXIiLCJqdGkiOiIxOTA2MTc2.LzSZDGo0vlsDi-Puy3w_vVZab50" http://localhost:3000/api/v1/me/totp-secret
```
As a result you will get a secret key in text format and a qr code as a base64 encoded png image e.g.:

```
{
    "secret": "54E4WYG5XSNZ37KI4CLILAVZKCMZ5MY7",
    "qr": "iVBORw0KGgoAAAANSUhEUgAAA"
}
```
The secret key will be stored in the users' database for the current user and can only be used with the combination with his login.
Please note that you can generate a secret key only once, so if a user has it already, the creation attempt will be rejected. 
Please write down the returned secret key or scan the generated qr code, otherwise you might lose access to your account.

If you've lost it, you have the following possibilities to recover access to your account:
- if you have access to another Administrator account, you can delete totP secret of any user including yourself by using totp administrative api (see below)
- if you have database access, you can erase value of `totp_secret` column in the `users` table of the corresponding user e.g.:
```
   UPDATE users SET totp_secret = '' WHERE `username` = 'your@mail.com';
```
- if you still have a valid auth token (the one you got from the second factor auth check from `/verify-2fa`), you can call `/me/totp-secret` endpoint with DELETE method to remove totP secret of the current user. Because of security reasons, you cannot use a login token here (the one you got from `/login` endpoint).

To display qr code to an image you can use following script:

```
echo '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' | base64 --decode > gr.png
```

4. You can open `gr.png` image and scan it with your authenticator app. If everything works well, you will see one time passwords generated every 30 seconds. After activation you will see Rport account in your Authenticator app. You can change this name by chaning configuration key `totp_account_name` to a name of your choice.

5. Now you can call `/verify-2fa` endpoint to pass second factor auth. For that you would need login token from the above as well as code from the previous paragraph.

```
curl -s http://localhost:3000/api/v1/verify-2fa -H "Content-Type: application/json" -X POST \
-H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InVzZXIiLCJqdGkiOiIxOTA2MTc2.LzSZDGo0vlsDi-Puy3w_vVZab50" \
--data-raw '{
"username": "admin",
"token": "123334"
}'|jq
{
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFkbWluIiwianRpIjoiMTcwMTc0MjY4MTkxNTQwMDA2NjQifQ.IhOK2leOdCXK5jvAO9aWEcpZ0kanpSkSbRpufha8soc",
    "two_fa": null
  }
}
```

User has a limited time to provide an Authenticator's code after login. This time is set in `totp_login_session_ttl` option.

If a user provides valid login and password, waits more than `totp_login_session_ttl` time and provides a valid totp code, his login attempt will be rejected.
You're flexible in selecting ms, seconds, minutes and hours for the `totp_login_session_ttl` option, so you can set it as e.g. "3600000ms", "3600s" or "60m" or "1h".

You can manage totp secret key with the following endpoints:

### To delete a totP secret of the current user:

```
curl -s http://localhost:3000/api/v1/me/totp-secret -H "Content-Type: application/json" -X DELETE \
-H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InVzZXIiLCJqdGkiOiIxOTA2MTc2.LzSZDGo0vlsDi-Puy3w_vVZab50"
```

### To read an existing totP secret of the current user:
```
curl -s http://localhost:3000/api/v1/me/totp-secret -H "Content-Type: application/json" -X GET \
-H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InVzZXIiLCJqdGkiOiIxOTA2MTc2.LzSZDGo0vlsDi-Puy3w_vVZab50"
```

### To create and read a new totP secret for the current user:

### To read an existing totP secret of the current user:
```
curl -s http://localhost:3000/api/v1/me/totp-secret -H "Content-Type: application/json" -X POST \
-H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InVzZXIiLCJqdGkiOiIxOTA2MTc2.LzSZDGo0vlsDi-Puy3w_vVZab50"
```

Please note, that for all above endpoints you can use JWT token that you get after the second factor authentication from `/verify-2fa` endpoint. Additionally, you can use login token to create a new secret if user has no other secret (see above).

All totP secret management endpoints are available only if `totp_enabled` option is true.

### Administrative totP API
If you have an Administrator account, you can also delete totP secret of all users including yourself. The only limitation is that as with all other management endpoints user must pass 2fa to get a valid token from the /verify-2fa` endpoint.

```
curl -s http://localhost:3000/api/v1/users/no@mail.com/totp-secret -H "Content-Type: application/json" -X DELETE \
-H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InVzZXIiLCJqdGkiOiIxOTA2MTc2.LzSZDGo0vlsDi-Puy3w_vVZab50"
```

## Delegated authentication

Staring with rportd 0.5.0 you can delegate the authentication to a reverse proxy. This allows you to use a variety of authentication backends for example such supported by Netscaler, Keycloak, the [Apache Auth Plugins](https://httpd.apache.org/docs/2.4/howto/auth.html) or by the [Caddy Auth Portal](https://github.com/greenpau/caddy-auth-portal). 

If the reverse proxy sends a specific header and a header that includes a username in the value, rport considers the user as authenticated. 
Using the delegated authentication does not liberate you from performing a `/api/v1/login` request to retrieve the JWT and sending this JWT on all requests.

To enable the delegated authentication, activate the following settings in your `rportd.conf`.

```
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
```
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

If you request the JWT you must specify the username and password expected by the reverse proxy (`jomama` + `foobaz`). The rportd user backend is not asked.

`curl -u jomama:foobaz "http://rport.example.com:8080/api/v1/login?token-lifetime=9999" -v -o auth.json`

If you do the request via the web user interface you also need to enter the username and password specified by the proxy.

In this example, you must log in as user "jomama" but rport treats you as "jopapa" because the reverse proxy sends a static hard coded username in the `My-User` header.

::: warning
Note that the reverse proxy must only require authentication on the `/api/v1/login` URI. If you require basic authentication for all resources, the frontend will break because ajax requests are performed in the background with the default authentication bearer header that will overwrite potential authentication basic headers.
:::

### Real-world use case
The above example is just a demonstration. In a real-world example, the user would visit an authentication portal first that will place a cookie. Based on that cookie, the reverse proxy will deny or forward requests to the rportd backend.

The following Nginx example simulates an authentication portal.
```
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
::: danger
Do not use any of the examples in production as they all are not using encryption and all your authentication data would be transferred plain text, that means easily sniffable. 
:::
