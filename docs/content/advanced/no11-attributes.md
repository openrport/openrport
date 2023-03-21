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

## Using an attributes file

You can maintain attributes (tags and labels) inside a separate file.
This file can be formatted as JSON, YAML or TOML using the file extensions `.josn`, `yaml`, or `toml`.

Add the `attributes_file_path` into the `[client]` section of your `rport.conf` file.  
The following example shows how to activate attributes read from a file.

```toml
## A list of of tags and labels to give your clients attributes maintained in a separate file.
## See https://oss.rport.io/advanced/attributes/
attributes_file_path = "/var/lib/rport/client_attributes.yaml"
#attributes_file_path = "C:\Program Files\rport\client_attributes.(yaml|json|toml)"
```

The attributes file could look like the below example.

{{< tabs "attribute_examples" >}}
{{< tab "Yaml" >}}

```yaml
tags:
  - win
  - server
  - vm
labels:
  country: Germany
  city: Cologne
  datacenter: NetCologne GmbH
```

{{< /tab >}}

{{< tab "Json" >}}

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

{{< /tab >}}
{{< tab "Toml" >}}

```toml
tags = [ "win", "server", "vm" ]
labels = { "country" = Germany, "city" = Cologne, "datacenter" = "NetCologne GmbH" }
```

{{< /tab >}}

{{< /tabs >}}

The file is read only on rport client start. On every file change a restart of the rport client is required.

## Using tags in the main config

As an alternative to the attributes file you can still use the "old-style" (=< 0.9.6) tags directly inserted to the
`rport.conf` file.  
`attributes_file_path` has precedence. To enable reading tags from the main configuration file you must remove
or disable `attributes_file_path`.

## Filtering

Clients can be filtered by tags and labels like text through additional filter parameter

`/api/v1/clients?filter[tags]=server`  
`/api/v1/clients?filter[labels]=city: Cologne`

with the possible use of wildcards

`/api/v1/clients?filter[tags]=ser*`  
`/api/v1/clients?filter[labels]=*: Cologne`

Though remember to encode space (" ") into proper url %20.

[Read the API documentation](https://apidoc.rport.io/master/#tag/Clients-and-Tunnels/operation/ClientsGet).
