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

## "attributes_file_path"

under **client** category in client configuration file
specifies additional configuration file with both tags and labels

### tags

can be also specified in client configuration file for backwards compatibility
but if __attributes_file_path__ is specified __tags__ in client configuration file will be ignored.

### labels

can only be specified only in additional attributes file  

### attributes file can be written in json, toml and yaml formats

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

## filtering

clients can be filtered by tags and labels like text through additional filter parameter

`/api/v1/clients?filter[tags]=server`
`/api/v1/clients?filter[labels]=city: Cologne`

with possible use of wildcards

`/api/v1/clients?filter[tags]=ser*`
`/api/v1/clients?filter[labels]=*: Cologne`

though remember to encode space (" ") into proper url %20
