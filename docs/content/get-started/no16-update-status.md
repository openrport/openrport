---
title: "OS update status (patch level)"
weight: 16
slug: supervising-update-status
aliases:
  - /docs/no16-update-status.html
---
{{< toc >}}

## Preface

Starting with version 0.3 the rport client can fetch a list of pending updates for the underlying operating system.
The summary is sent to the RPort server. This allows you to supervise the patch level of a large number of machines
from a central place. On the RPort server API each client will contain an `update_status` object that contains data as
shown in the example below.

```json
{
  "updates_status": {
    "refreshed": "2021-08-16T06:45:56.485091025Z",
    "updates_available": 109,
    "security_updates_available": 41,
    "update_summaries": [
      {
        "title": "login",
        "description": "login 1:4.8.1-1ubuntu5.20.04.1 Ubuntu:20.04/focal-updates [amd64]",
        "reboot_required": false,
        "is_security_update": false
      },
      {
        "title": "libatomic1",
        "description": "libatomic1 10.3.0-1ubuntu1~20.04 Ubuntu:20.04/focal-updates, Ubuntu:20.04/focal-security [amd64]",
        "reboot_required": false,
        "is_security_update": true
      }
    ],
    "reboot_pending": false
  }
}
```

## Supported operating systems

The following operating systems are supported for the update status supervision:

* Ubuntu and Debian by using `apt-get`
* RedHat, CentOs, and derivates by using `yum` or `dnf`
* SuSE by using `zypper`
* Microsoft Windows by using the Windows Update Manager

## Enable/disable

Fetching the update status is enabled by default. In the `[client]` section, you will find a block as shown in
the `rport.conf` file.

```text
## Supervision and reporting of the pending updates (patch level)
## Rport can constantly summarize pending updates and
## make that summary available on the rport server.
## On Debian/Ubuntu and SuSE Linux sudo rules are needed.
## https://oss.rport.io/docs/no16-update-status.html
## How often after the rport client has started pending updates are summarized
## Set 0 to disable.
## Supported time units: h (hours), m (minutes)
## Default: updates_interval = '4h'
#updates_interval = '4h'
```

Setting `updates_interval = '0'` disables the feature. Triggering a manual refresh of the update status is prevented this way too.

## Sudo rules

On some Linux distributions, only the root user is allowed to look for pending updates. Because the rport client runs
with its own unprivileged users, sudo rules are needed.

### Debian and Ubuntu

Create a file `/etc/sudoers.d/rport-update-status` with the following content:

```text
rport ALL=NOPASSWD: SETENV: /usr/bin/apt-get update -o Debug\:\:NoLocking=true
```

### SuSE Linux

Create a file `/etc/sudoers.d/rport-update-status` with the following content:

```text
rport ALL=NOPASSWD: SETENV: /usr/bin/zypper refresh *
```
