---
title: "Client Groups"
weight: 4
slug: client-groups
aliases:
  - /docs/no04-clients-groups.html
  - /docs/get-started/no04-client-groups.md
---
{{< toc >}}

## Define client groups

Rport client group can be created by:

1. adding single clients to it;
2. dynamic criteria using wildcards.

Managing client groups is done via the [API](https://apidoc.rport.io/master/#tag/Client-Groups).
The `/client-groups` endpoints allow you to create, update, delete and list them.

As listed in the [API docs](https://apidoc.rport.io/master/#tag/Client-Groups) Client Group is defined by:

* `id` - unique group identifier
* `description` - group description
* `params` - parameters that define what clients belong to a current group.

Each parameter can be specified by:

* exact match of the property **(ignoring case)**. For example,

  ```text
  params: {
    "client_id": ["test-win2019-tk01", "qa-lin-ubuntu16"]
  }
  ```

  Means only clients with `id` equals to `test-win2019-tk01` or `qa-lin-ubuntu16` belong to a current group.

* dynamic criteria using wildcards **(ignoring case)**. For example,

  ```text
  params: {
    "os_family": ["linux*", "*win*"]
  }
  ```

  Means all clients with `os_family` that starts with `linux` OR that contains `win` belong to a current group.

  NOTE: if few different parameters are given then a client belongs to this group only if client properties match all
  the given group parameters. If client parameter has multiple values (like `tags`, `ipv4`, `ipv6`, etc. ) then it belongs
  to a group if at least one client param matches one of group parameters.
  For example:

  ```text
    params: {
      "tag": ["QA", "my-tag*"],
      "os_family": ["linux*", "ubuntu*"]
    }
  ```

  Means clients belong to this group only if **both** conditions are met:
  1. has `tag` equals to `QA` **OR** `tag` that starts with `my-tag`;
  2. its `os_family` starts with `linux` or `ubuntu`.

  Example specifing logical operators:

  ```text
    params: {
      "tags": { "and": [ "Linux", "Datacenter 3" ] }
    }
  ```

  Means clients belong to this group only if following condition is met:
  1. has a `tag` equals to `Linux` **AND** a `tag` that equals to `Datacenter 3`;
  **OR** operator can be specified in the same way

* `client_ids` - read-only field that is populated with IDs of active clients that belong to this group.

## Manage client groups via the API

Here are some examples how to manage client groups.

### Create

```shell
curl -X POST 'http://localhost:3000/api/v1/client-groups' \
-u admin:foobaz \
-H 'Content-Type: application/json' \
--data-raw '{
    "id": "group-1",
    "description": "This is my super client group.",
    "params":
    {
        "tag": ["QA"],
        "os_family": ["linux*", "ubuntu*"]
    }
}'
```

### Update

Note all the parameters will be overridden.

```shell
curl -X PUT 'http://localhost:3000/api/v1/client-groups/group-1' \
-u admin:foobaz \
-H 'Content-Type: application/json' \
--data-raw '{
    "id": "group-1",
    "description": "This is my super client group.",
    "params":
    {
        "tag": ["QA", "my-tag*"],
        "os_family": ["linux*", "ubuntu*"]
    }
}'
```

### List all client groups

```shell
curl -s -u admin:foobaz http://localhost:3000/api/v1/client-groups/group-1|jq
{
  "data": {
    "id": "group-1",
    "description": "This is my super client group.",
    "params": {
      "client_id": null,
      "name": null,
      "os": null,
      "os_arch": null,
      "os_family": [
        "linux*",
        "ubuntu*"
      ],
      "os_kernel": null,
      "hostname": null,
      "ipv4": null,
      "ipv6": null,
      "tag": [
        "QA",
        "my-tag*"
      ],
      "version": null,
      "address": null,
      "client_auth_id": null
    },
    "client_ids": [
      "qa-lin-ubuntu16",
      "qa-lin-ubuntu19",
      "qa-lin-ubuntu23"
    ]
  }
}
```

### Delete

```shell
curl -u admin:foobaz -X DELETE 'http://localhost:3000/api/v1/client-groups/group-1'
```
