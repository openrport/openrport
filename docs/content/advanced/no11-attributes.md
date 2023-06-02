---
title: "Describing and filtering clients"
weight: 11
slug: attributes
aliases:
  - /docs/no11-attributes.html
---
{{< toc >}}

Clients can be described for filtering and identification by:

- single dimension tags `["win", "server", "vm"]`
- 2-dimensional labels  `"labels": {"country": "Germany", "city": "Cologne", "datacenter": "NetCologne GmbH" }`

## setting attributes

As of writing this Doc there are 3 ways to set up client's attributes.

1. In client's `rport.conf` config file, under property tags (labels are not supported)
2. As a separate file on the client `attributes file`
3. Through the API - `attributes file` has to be enabled

### 1. Using tags in the main config

As an alternative to the attributes file you can still use the "old-style" (=< 0.9.6) tags directly inserted to the
`rport.conf` file.  
`attributes_file_path` has precedence. To enable reading tags from the main configuration file you must remove
or disable `attributes_file_path`.

### 2. Using an attributes file

You can maintain attributes (tags and labels) inside a separate file.
This file can only be formatted as JSON.

Add the `attributes_file_path` into the `[client]` section of your `rport.conf` file.  
The following example shows how to activate attributes read from a file.

```toml
## A list of of tags and labels to give your clients attributes maintained in a separate file.
## See https://oss.rport.io/advanced/attributes/
attributes_file_path = "/var/lib/rport/client_attributes.yaml"
#attributes_file_path = "C:\Program Files\rport\client_attributes.(yaml|json|toml)"
```

The attributes file could look like the below example.

```json
{ 
  "tags": ["win", "server", "vm"],
  "labels": {
    "country": "Germany",
    "city": "Cologne",
    "datacenter": "NetCologne GmbH"
  }
}

```

The file is read only on rport client start. On every file change a restart of the rport client is required.

### 3. Using the API or UI

Instead of logging in to each remote system, you can update attributes (tags and labels) via the API efficiently.

To manage attributes remotely the following preconditions must be met:

- `attributes_file_path` has to be set, read point 2. Updating tags that are directly inserted into the main
  `rport.conf` file via the API is not supported.
- the path has to be writable by the client when running as daemon
- the client has to be `Active` - currently connected to the server. Attributes are persisted on the client only.
  The server API will reject update attempts if the client is disconnected.

To update attributes via the API you need to send a __PUT__ request with the entire new JSON to

`/api/v1/client/{client_id}/attributes`.

Partial updates, aka PATCH requests, are not supported.  
Read more on the [API documentation](https://apidoc.rport.io/master/#tag/Clients-and-Tunnels/operation/ClientAttributesUpdate).

## Filtering

Clients can be filtered by tags and labels like text through additional filter parameter

`/api/v1/clients?filter[tags]=server`  
`/api/v1/clients?filter[labels]=city: Cologne`

with the possible use of wildcards

`/api/v1/clients?filter[tags]=ser*`  
`/api/v1/clients?filter[labels]=*: Cologne`

Though remember to url-encode space (" ") into `%20`.

Read more on the [API documentation](https://apidoc.rport.io/master/#tag/Clients-and-Tunnels/operation/ClientsGet).
