---
title: "Software- and Container-Inventory"
weight: 15
slug: supervising-system-inventory
aliases:
  - /docs/no15-inventory.html
---
{{< toc >}}

## Preface

Starting with version 0.9.15 the rport client can fetch a list of software- and container inventory for the underlying operating system.
The summary is sent to the RPort server. On the RPort server API each client will contain an `inventory` object that contains data as
shown in the example below.

```json
{
  "inventory": {
      "refreshed": "2024-09-18T13:29:40.834378519Z",
      "software_inventory": [
        {
          "package": "curl",
          "version": "8.5.0-2ubuntu10.4",
          "summary": "command line tool for transferring data with URL syntax"
        },
        {
          "package": "docker-ce",
          "version": "5:27.2.1-1~ubuntu.24.04~noble",
          "summary": "Docker: the open-source application container engine"
        }
      ],
      "container_inventory": [
        {
          "container_id": "ef5bde85d185571548ecc8de319a1ccd59e8a623ff3880f4e30c6fbe1b31abb8",
          "container_name": "/stoic_goldberg",
          "status": "Up 23 seconds",
          "image_id": "sha256:b1e9cef3f2977f8bdd19eb9ae04f83b315f80fe4f5c5651fedf41482c12432f7",
          "image_name": "ubuntu:latest"
        }
      ]
}
```

## Supported operating systems

The following operating systems are supported for the inventory supervision:

* [Software-Invetory] Ubuntu and Debian by using `dpkg`
* [Software-Invetory] Microsoft Windows by using the Windows Registry
* [Container-Invetory] All operating systems using the GO-Module for Docker

## Enable/disable

Fetching the inventory is enabled by default. In the `[client]` section, you will find a block as shown in
the `rport.conf` file.

```text
## Inventory of software and containers
## Rport can regularly read all installed packages / programs as 
## well as running containers (e.g. docker) and transfer them together
## to the rport server. On Debian/Ubuntu systems, the rport user 
## must be given permission to use the docker daemon.
## https://oss.openrport.io/advanced/supervising-system-inventory/
## How often should the inventory list be sent to the server after the client starts
## Set 0 to disable.
## Supported time units: h (hours), m (minutes)
## Default: inventory_interval = '4h'
#inventory_interval = '4h'
```

Setting `inventory_interval = '0'` disables the feature.

## Permissions

On some Linux distributions, only the root user is allowed to look for running docker containers. Because the rport client runs
with its own unprivileged user, you hve to give the permissions to that user.

### Debian and Ubuntu

Run the following command:

```text
sudo usermod -aG docker rport
```