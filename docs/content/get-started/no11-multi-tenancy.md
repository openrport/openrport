---
title: "Multi tenancy"
weight: 11
slug: multi-tenancy
aliases:
  - /docs/no11-multi-tenancy.html
---
{{< toc >}}
Multi tenancy can be achieved by running multiple isolated instances of the RPort server on a single host.
By invoking `rportd` multiple time with different configuration files, you get completely isolated server instances.

## Run it with systemd

Below you find an example of systemd service file, that manages multiple instances.
Store the file in `/etc/systemd/system/rportd@.service`. (The `@` sign in the file name is crucial.)

```text
[Unit]
Description=Rport Server Instance %i
ConditionFileIsExecutable=/usr/local/bin/rportd

[Service]
StartLimitInterval=5
StartLimitBurst=10
ExecStart=/usr/local/bin/rportd "-c" "/etc/rport/instances/rportd.%i.conf"
LimitNOFILE=1048576
User=rport
Restart=always
RestartSec=120
EnvironmentFile=-/etc/sysconfig/rportd

[Install]
WantedBy=multi-user.target
```

Now create a folder `/etc/rport/instances/` and put a configuration file per instance in this folder.
Start and stop the instances with `systemctl start rportd@<INSTANCE-NAME>`.
