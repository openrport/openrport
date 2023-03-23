---
title: 'Monitoring'
weight: 17
slug: monitoring
aliases:
  - /docs/no17-monitoring.html
---

{{< toc >}}

## Preface

Starting with version 0.5 RPort comes with basic monitoring capabilities. The RPort client can collect the following
metrics from the underlying operating system:

* CPU usage (percent)
* Memory usage (percent)
* IO usage (percent)
* List of running processes
* Fill level of hard disks and mount points

To use the built-in monitoring both – server and client – must run at least RPort 0.5.0.

The monitoring is enabled by default. If newer clients are connected to an old server the client disables the monitoring
automatically. All monitoring data is stored on the server in a sqlite3 database file `monitoring.db` inside the data dir
of the RPort server.

## Server configuration options

Select a proper value for `data_storage_days` carefully in the `[monitoring]` section of the `rportd.conf`.
The more clients you have, the more data will be collected. Having hundreds of clients collecting monitoring data, the
database file can quickly grow to 10 Gigabytes or more. Use a symbolic link, if you want to store the `monitoring.db`
file outside the data dir.

## Client configuration options

If you client configuration after an update does not contain a `[monitoring]` section, copy it from the
[rport.example.conf](https://github.com/realvnc-labs/rport/blob/master/rport.example.conf).
To save bandwidth and disk space on the server, you can disable the monitoring for clients completely.
Please refer to the documentation inside the configuration example to explore all options of the monitoring.

## Fetching monitoring data

All collected monitoring data can be fetched using the API. Please refer to our
[API docs](https://apidoc.rport.io/master/#tag/Monitoring).

## Processing monitoring data

At the moment, either the client nor the server processes the monitoring data in any way. Sending alerts based on
thresholds is on our roadmap. Be patient and [stay tuned](https://subscribe.rport.io).
