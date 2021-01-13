## Client Groups
Rport client group can be created by:
1. adding single clients to it;
2. dynamic criteria using wildcards.

Managing client groups is done via the [API](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/cloudradar-monitoring/rport/master/api-doc.yml#/Client%20Groups).
The `/client-groups` endpoints allow you to create, update, delete and list them.

As listed in the API docs Client Group is defined by:
* `id` - unique group identifier
* `description` - group description
* `params` - parameters that define what clients belong to a current group.
Each parameter can be specified by:
  * exact match of the property. For example, 
    ```
    params: {
      "client_id": ["test-win2019-tk01", "qa-lin-ubuntu16"]
    }
    ```
    Means only clients with `id` equals to `test-win2019-tk01` or `qa-lin-ubuntu16` belong to a current group.
  * dynamic criteria using wildcards. For example, 
    ```
    params: {
      "os_family": ["linux*"]
    }
    ```
    Means all clients with `os_family` that starts with `linux` belong to a current group.
    
  NOTE: if few different parameters are given then a client belongs to this group
  only if client properties match all the given group parameters. For example,
  ```
    params: {
      "tag": ["QA", "my-tag*"],
      "os_family": ["linux*", "ubuntu*"]
    }
  ```
  Means clients belong to this group only if **both** conditions are met:
  1. has `tag` equals to `QA` or `tag` that starts with `my-tag`;
  2. its `os_family` starts with `linux` or `ubuntu`.

### Manage client groups via the API
Here are some examples how to manage client groups.

#### Create
```
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
#### Update
Note all the parameters will be overridden.
```
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
#### List all client groups.
```
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
    }
  }
}
```
#### Delete
```
curl -u admin:foobaz -X DELETE 'http://localhost:3000/api/v1/client-groups/group-1'
```
