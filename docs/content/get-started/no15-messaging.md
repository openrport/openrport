---
title: "2FA-Messaging"
weight: 15
slug: 2fa-messaging
aliases:
  - /docs/no15-messaging.html
  - /docs/content/get-started/no15-messaging.md
---
{{< toc >}}
Some features require the rport server to send messages, e.g., [2FA Auth](no02-api-auth.md#two-factor-auth)
requires sending a verification code to a user.

## SMTP

To enable sending emails, enter the following lines to the `rportd.config`, for example:

```text
[smtp]
  server = 'smtp.example.com:2525'
  sender_email = 'rport@gmail.com'
  auth_username = 'john.doe'
  auth_password = 'secret'
  secure = false
```

Required:
`server`
: smtp server and port separated by a colon, e.g. `smtp.example.com:2525`.
  If you use port 465 with Implicit(Forced) TLS then `secure` param should be enabled.

`sender_email`
:an email that is used by rport server as its sender.

Optional:

`auth_username`
: a username for authentication;

`auth_password`
: a password for the username;

`secure`
: `true|false`, set to `true` if Implicit(Forced) TLS must be used.

## Pushover

Follow a [link](https://support.pushover.net/i7-what-is-pushover-and-how-do-i-use-it) to have a quick Pushover intro.

In order to enable sending messages via pushover:

1. [Register](https://pushover.net/signup) pushover account (if you don't have an existing account).
2. [Register](https://pushover.net/apps/build) pushover API token that will be used to send messages by rport server (if you don't have it yet).
3. Enter the following lines to the `rportd.config`, for example:

    ```text
    [pushover]
      api_token = "afapzrcv5801jeaw71b4odjyn1m2e5"
      user_key = "pgcjszdyures33k4m4e12e9ggc1syo"
    ```

4. Use any of [pushover device clients](https://pushover.net/clients) to receive the messages.

## Script

You can create a custom script to send the 2FA verification code. This way you can use messengers like Telegram and
many others. Inside the `[api]` section of `rportd.conf` insert the full path to an executable script for
`two_fa_token_delivery` parameter. For example:

```text
two_fa_token_delivery = '/usr/local/bin/2fa-sender'
```

The token and the recipient's details are passed as environmental variables to the script.
Create the file `/usr/local/bin/2fa-sender` with the following content, and make the script executable with `chmod +x`.

```bash
#!/bin/bash
date > /tmp/2fa-sender.txt
printenv >> /tmp/2fa-sender.txt
```

Try to log in, and the script creates the following output.

```text
Fri Aug 13 13:36:25 UTC 2021
RPORT_2FA_SENDTO=email@example.com
RPORT_2FA_TOKEN_TTL=600
RPORT_2FA_TOKEN=7SM7j2
RPORT_2FA_USER_AGENT='Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:99.0) Gecko/20100101 Firefox/99.0'
RPORT_2FA_REMOTE_ADDRESS=::1
snip..snap
```

The value of `RPORT_2FA_SENDTO` may vary. It's the value specified in the 2fa_sendto column of the user table or the auth file.

Additionally, you can specify how the api should validate updates of the 2fa_sendto. This prevents users entering values
that cannot be processed by your script. Use `two_fa_send_to_type = 'email'`  to accept only valid email address or
specify a regular expression.

If the script exits with an exit code other than `0` the API request returns HTTP Status code 500 along with the STDERR
output of the script.

{{< hint type=tip >}}
When handing over the token using curl, consider using the `-f` option of curl. On any other http status code than 200
curl will exit with a non-zero status code. This way the rport server knows about a failed request, and the API includes
the error for further processing.
{{< /hint >}}

### Telegram example

A script that sends the token via Telegram can work like this example.
You must [create a bot](https://core.telegram.org/bots#6-botfather) first and grab the token of it.

```bash
#!/bin/sh
BOT_TOKEN="<YOUR_BOT_TOKEN>"
URL="https://api.telegram.org/bot${BOT_TOKEN}/sendMessage"
curl -fs -X POST $URL \
  -d chat_id=$RPORT_2FA_SENDTO \
  -d text="Your RPort 2fa token: $RPORT_2FA_TOKEN (valid for $RPORT_2FA_TOKEN_TTL seconds)"
```

## URL

Rport will make a POST request to the specified URL.
The token and the recipient's details are passed as form values.
This method requires `base_url` to be set in server config.

Example using free 2FA service:

```text
two_fa_token_delivery = 'https://free-2fa-sender.rport.io'
```
