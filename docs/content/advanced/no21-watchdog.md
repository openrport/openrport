---
title: 'Watchdog Integration'
weight: 21
slug: watchdog-integration
---

{{< toc >}}

## Watchdog integration

Starting with RPort version 0.8.3 you can enable a watchdog integration on the rport client.

The rport client works stable and reliable and broken connections are always resumed.
There is no need to enable watchdog integration by default.
It should only be activated in exceptional cases when you notice the rport client process is running, but no connection
attempts are made.

On the `rport.conf` go the the `[connection]` section and set `watchdog_integration = true`.
Also make sure you have `max_retry_count` disabled and `keep_alive` must be enabled.

## State.json file

With the watchdog integration enabled, the rport client will create and constantly update a file `state.json` in the
data directory. The file looks like the following example:

```json
{
  "last_update": "2022-08-15T10:37:25.643839+02:00",
  "last_update_ts": 1660552645,
  "last_state": "connected",
  "last_message": "ping to rport server 81.151.79.171:8080 succeeded within 14.682823ms"
}
```

The first two fields indicate when the file has been updated for the last time. These are the most important field,
and your supervisor logic must be based one of the update times. As long as the last_update constantly increases,
the rport client is working correctly.

The field `last_state` can have one of the following values:

`initialized`
: the watchdog integration was started but no rport client state is yet known.

`connected`
: the rport client is connected to the server.

`reconnecting`
: the rport client is not connected but trying to reconnect.

The field `last_message` can have one of the following values:

`ping to {server} succeeded within {latency}`
: a ping succeeded

`Retrying in {time}...`
: a connection failed

`connected to {server} within {latency}`
: a connection succeeded

## Update events

The `state.json` file gets updated on the following events:

1. connection succeeded
2. connection retry, according to setting `max_retry_interval`
3. ping to the server, according to setting `keep_alive`

While your server is connected, the file gets updates every time a ping to the server is sent using the `keep_alive`
interval. With the client being disconnected, the file is updated on each reconnect attempt which can have a maximum
interval set by `max_retry_interval`.

## Implement your watchdog

Your watchdog implementation should restart the rport client when is "hangs". Either the last update is older than
the current time minus the keep alive interval or older than the max retry interval.

### Simple PowerShell example

The Windows service does not have a built-in watchdog. So you must implement a schedules task that checks the state.json
file regularly.

```powershell
$stateFile = 'C:\Program Files\rport\data\state.json'
$threshHoldSec = 600

$now = ((Get-Date -UFormat %s) - [int](get-date -Uformat %Z)*3600)
$lastUpdate = (Get-Content $stateFile| ConvertFrom-Json).last_update_ts
$diff = $now - $lastUpdate
if ($diff -gt $threshHoldSec){
    Write-Output "RPort hangs. No activity deteced for $($diff) seconds."
    Restart-Service rport
} else {
    Write-Output "RPort is running fine. Last activity $($diff) seconds ago."
}
```

🐶 A fully-featured rport watchdog for Windows implemented in PowerShell can be
[downloaded here](https://github.com/realvnc-labs/rport-win-watchdog). It comes ready-to-use with an easy
installation.

### Using the systemd watchdog

On Linux systemd comes with a built-in watchdog. Systemd does not use the state.json file. The file is written anyway,
so you can observe what's happening behind the scenes.

To enable the systemd watchdog you must enter a line `WatchdogSec=N` to the service file
`/etc/systemd/system/rport.service`. This causes systemd to create a unix socket where watchdog updates can be pushed to.
The rport client will automatically recognize the presence of the socket and updates will be pushed on the events listed
above.

If you enable debug logging, you will get a confirmation like `Using NOTIFY_SOCKET /run/systemd/notify for systemd watchdog integration` in the log file.

If systemd does not detect any updates on the socket within the `WatchdogSec` period, it will restart the rport client.
This means you must set `WatchdogSec` a bit longer than `max_retry_interval` and `keep_alive`. See above.

Setting `keep_alive` and `max_retry_interval` both the 3 minutes and `WatchdocSec` to 200 are good values.
The shorter you set `keep_alive` and `max_retry_interval` the faster the watchdog can act. But the more bandwidth the
rport client consumes in idle mode.

Full systemd service example:

```text
[Unit]
Description=Create reverse tunnels with ease.
ConditionFileIsExecutable=/usr/local/bin/rport

[Service]
ExecStart=/usr/local/bin/rport "-c" "/etc/rport/rport.conf"
LimitNOFILE=1048576
User=rport
Restart=always
RestartSec=120
WatchdogSec=200

[Install]
WantedBy=multi-user.target

[Unit]
StartLimitIntervalSec=5
StartLimitBurst=10
```
