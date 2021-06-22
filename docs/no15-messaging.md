# Messaging
Some features require the rport server to send messages, e.g., [2FA Auth](no02-api-auth.md#two-factor-auth) requires sending a verification code to a user.
It can be done using:
1. email (requires [SMTP](no15-messaging.md#smtp) setup)
2. [pushover.net](https://pushover.net) (requires [Pushover](no15-messaging.md#pushover) setup)

## SMTP
In order to enable sending emails enter the following lines to the `rportd.config`, for example:
```
[smtp]
  server = 'smtp.example.com:2525'
  sender_email = 'rport@gmail.com'
  auth_username = 'john.doe'
  auth_password = 'secret'
  secure = false
```
Required:
- `server` - smtp server and port separated by a colon, e.g. `smtp.example.com:2525`. If you use port 465 with Implicit(Forced) TLS then `secure` param should be enabled.
- `sender_email` - an email that is used by rport server as its sender.

Optional:
- `auth_username` - a username for authentication;
- `auth_password` - a password for the username;
- `secure` - `true|false`, set to `true` if Implicit(Forced) TLS must be used.

## Pushover
Follow a [link](https://support.pushover.net/i7-what-is-pushover-and-how-do-i-use-it) to have a quick Pushover intro.

In order to enable sending messages via pushover:
1. [Register](https://pushover.net/signup) pushover account (if you don't have an existing account).
2. [Register](https://pushover.net/apps/build) pushover API token that will be used to send messages by rport server (if you don't have it yet).
3. Enter the following lines to the `rportd.config`, for example:
```
[pushover]
  api_token = "afapzrcv5801jeaw71b4odjyn1m2e5"
  user_key = "pgcjszdyures33k4m4e12e9ggc1syo"
```
4. Use any of [pushover device clients](https://pushover.net/clients) to receive the messages.