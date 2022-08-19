---
title: "Securing the rport server"
weight: 10
slug: securing-the-rport-server
aliases:
  - /docs/no10-securing-the-server.html
---
{{< toc >}}
Your rport server usually exposes two TCP ports to the public internet:

* one for the client connections
* one for the API and the web frontend

Both require authentication and unless you use weak passwords you are safe.
But even if login attempts are prevented on the HTTP level they produce some load on your server.

Learn more about the default security measures and how to add a second level of security using [fail2ban](https://www.fail2ban.org/).

## Client Connection Listener

### The client connection listener is secured by default

By default, the following options are activated for the client connection listener.

```text
## Protect your server against password guessing.
## Force clients to wait N seconds (float) between unsuccessful login attempts.
## This is per client auth id.
## A message like
##    'client-listener: Failed login attempt for client auth id "abc", forcing to wait for {client_login_wait}s ({ip})'
## is logged to the info log.
## Consider changing the log_level to 'info' to trace failed login attempts.
## Learn more https://oss.rport.io/docs/no10-securing-the-server.html
## Defaults: 2.0
#client_login_wait = 2.0

## After {max_failed_login} consecutive failed login-in attempts ban the source IP address for {ban_time} seconds.
## HTTP Status 423 is returned.
## A message like
##     'Maximum of {max_failed_login} login attempts reached. Visitor ({remote-ip}) banned. Ban expiry: 2021-04-16T11:22:26+00:00'
## is logged to the info log.
## Banning happens on HTTP level.
## Consider banning on network level using fail2ban.
## Learn more https://oss.rport.io/docs/no10-securing-the-server.html
## Defaults: max_failed_login = 5, ban_time = 3600
#max_failed_login = 5
#ban_time = 3600
```

These are good settings to protect your server against password guessing.
The counters for failed logins are constantly increasing and only reset by a successful login or a server restart.

For example, if a client fails to log in for the fifth time, any login attempts of the IP address are blocked for one hour.
If there are more clients on the same network with correct credentials but sharing common internet access and these
clients are restarted, they are banned too.

{{< hint type=warning >}}
Because rejecting connections on failed logins is handled properly by the server, they are not considered an error and
not logged to the log file when you are on the `error` log level. Consider changing the `log_level` to `info` to trace
failed logins and to eventually activate fail2ban.
{{< /hint >}}

### Using fail2ban for additional security

#### Ban password guesser

#### Ban scanners

After a short period, you will notice HTTP requests to arbitrary files and folders on the client connect port that are
answered with HTTP 404. The internet is full of scanners searching for vulnerable web applications.
You can safely ban any IP address that produces HTTP 404. A rport client will never do this.

Search your log file for the following pattern:
`egrep " 404 [0-9]+\w+ \(.*\)" /var/log/rport/rportd.log`

#### Fail2ban configuration

{{< hint type=tip >}} Remember
All fail2ban rules require running rport with `log_level = 'info'`.
{{< /hint >}}

To create rules and filters to ban both, password guessers and scanners, proceed with the following setup.

A fail2ban filter `/etc/fail2ban/filter.d/rportd-client-connect.conf` can look like this.

```text
# Fail2Ban filter for rportd client connect
[Definition]
# Identify scanners
failregex = 404 [0-9]+\w+ \(<HOST>\)
# Identify password guesser
```

Test the definition with
`fail2ban-regex /var/log/rport/rportd.log /etc/fail2ban/filter.d/rport-client-connect.conf`
It should output something like

```text
Lines: 16900 lines, 0 ignored, 755 matched, 16145 missed
[processed in 1.04 sec]
```

That means fail2ban has found 755 requests that will be banned.

Create a ban action in `/etc/fail2ban/jail.d/rportd-client-connect.conf`.
Change `port` to the port your rportd is listening for client connections. `grep "url =" /etc/rport/rportd.conf`

```text
# service name
[rportd-client-connect]
# turn on /off
enabled  = true
# ports to ban (numeric or text)
port     = 8080
# filter from previous step
filter   = rportd-client-connect
# file to parse
logpath  = /var/log/rport/rportd.log
# ban rule:
# ban all IPs that have created two 404 request during the last 20 seconds for 1hour
maxretry = 2
findtime = 20
# ban on 10 minutes
bantime = 3600
```

Restart fail2ban to activate the new configuration with `service fail2ban restart` and check the status.

```shell
root@localhost:~# fail2ban-client status
Status
|- Number of jail: 2
`- Jail list: rportd-client-connect, sshd
fail2ban-server[1905]: Server ready
root@localhost:~# fail2ban-client status
Status
|- Number of jail: 2
`- Jail list: rportd-client-connect, sshd
root@localhost:~# fail2ban-client status rportd-client-connect
Status for the jail: rportd-client-connect
|- Filter
|  |- Currently failed: 0
|  |- Total failed: 0
|  `- File list: /var/log/rport/rportd.log
`- Actions
   |- Currently banned: 0
   |- Total banned: 0
   `- Banned IP list: 
```

{{< hint type=warning >}}
Fail2ban ships with a lot of default rules and the SSH is enabled by default.
To enable fail2ban only for rportd and to disable the ssh rule, make sure only rportd rules are in `/etc/fail2ban/jail.d/`.
Either delete `/etc/fail2ban/jail.d/defaults-debian.conf` or open the file and comment out all lines.
Use `fail2ban-client status` to verify which rules are active.
{{< /hint >}}

## Securing the API

@todo: Finish this chapter.
