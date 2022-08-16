---
title: "Vault"
weight: 13
slug: vault
aliases:
  - /docs/no13-vault.html
---
{{< toc >}}
Rport provides a secure storage which allows to persist arbitrary data related to the clients and the environment.
All data is stored encrypted so user can safely store passwords used for log in to remote systems there.
Typical vault workflow looks as following:

- Administrator initialises vault by calling [vault init api](https://apidoc.rport.io/master/#tag/Vault)
  and giving a password in input. RPort stores password in a secure place. From this moment Rport vault will remain
  unlocked and can accept requests for reading or changing data.

- Administrator can lock vault by using [vault lock api](https://apidoc.rport.io/master/#operation/VaultAdminSesamDelete).
  RPort removes password and rejects all access requests to the vault. The same happens when server restarts.
  The vault database is persisted to hdd, so it will remain after server restarts. However, we would also recommend to
  back up the vault database additionally to survive also possible disk failures.

- Administrator can unlock vault by using [vault unlock api](https://apidoc.rport.io/master/#operation/VaultAdminSesamPost).
  He should provide password which he used on init stage. If a wrong password is provided, RPort will reject the vault access.
  In case of a correct password vault will be unlocked and can be used for reading or changing secure data.
  
- Any authorized user can store new key value pairs by calling [vault store api](https://apidoc.rport.io/master/#operation/VaultPost)

- Any authorized user can list or search for stored vault entries by calling [vault list api](https://apidoc.rport.io/master/#operation/VaultGet)

- Any authorized user can get the stored secure value by provided id [vault get api](https://apidoc.rport.io/master/#operation/VaultItemGet)

Any user belonging to the Administrators group can init, lock and unlock Vault. Any authorized user can read, store,
delete values in an unlocked and initialized Vault. The only exception from this rule is if a value is stored with a
non-empty `required_group` field, in this case the access will be allowed only to the users belonging to the specified
`required_group` value.

## Admin API Usage

The `/vault-admin` endpoints allow you to initialize, lock, unlock and read status of RPort vault.

### Initialize

This operation creates a new vault database and provisions it with some status information.
> _Administrator access required_

```shell
curl -X POST 'http://localhost:3000/api/v1/vault-admin/init' \
-u admin:foobaz \
-H 'Content-Type: application/json' \
--data-raw '{
 "password": "1234"
}'
```

Password length must be between 4 and 32 bytes, shorter and longer passwords are rejected.

You need to init database every time when the rport server is restarted.

### Status

This API allows to read the current status of the RPort vault:

```shell
curl -X GET 'http://localhost:3000/api/v1/vault-admin' \
-u admin:foobaz \
-H 'Content-Type: application/json'
```

The response will be

```json
{
    "data": {
        "init": "setup-completed",
        "status": "unlocked"
    }
}
```

The `init` field shows the initialization status of the vault. `setup-completed` means that the vault is already
initialized or `uninitialized` otherwise. The `status` field shows the lock status of the vault. It can be either
`unlocked` or `locked`. `unlocked` status means that the vault is fully functional and can be used to store, read or
modify data securely. `unlocked` status means that any access to vault database will be rejected till administrator unlocks it again.

### Lock

This operation locks the vault, meaning that the password will be removed from server's memory and vault won't accept any requests.
> _Administrator access required_

```shell
curl -X DELETE 'http://localhost:3000/api/v1/vault-admin/sesame' \
-u admin:foobaz \
-H 'Content-Type: application/json'
```

### Unlock

This operation unlocks the vault. Administrator has to provide same password he used for the vault initialization.
Rport will check the password validity and reject the request if a wrong password is provided. If administrator loses
the password, all access to the data will be lost, so it should be kept in a secure place. If a correct password is
provided, the vault will become fully functional.

> _Administrator access required_

```shell
curl -X POST 'http://localhost:3000/api/v1/vault-admin/sesame' \
-u admin:foobaz \
-H 'Content-Type: application/json' \
--data-raw '{
 "password": "1234"
}'
```

## User API Usage

### List

This API allows to list all entries with `id`, `client_id`, `created_by`, `created_at`, `key` fields but without
the encrypted `value` field.

```shell
curl -X GET 'http://localhost:3000/api/v1/vault' \
-u admin:foobaz \
-H 'Content-Type: application/json'
```

The response will be

```json
{
    "data": [
        {
            "id": 1,
            "client_id": "client123",
            "created_by": "admin",
            "created_at": "2021-05-18T09:26:10+03:00",
            "key": "one"
        },
        {
            "id": 2,
            "client_id": "client123",
            "created_by": "admin",
            "created_at": "2021-05-18T09:30:27+03:00",
            "key": "two"
        }
    ]
}
```

### Sort

You can sort entries by `id`, `client_id`, `created_by`, `created_at`, `key` fields. Example:  
`http://localhost:3000/api/v1/vault?sort=created_at` - gives you entries sorted by date of creation in ascending order.

To change the sorting order by adding `-` to a field name. Example:  
`http://localhost:3000/api/v1/vault?sort=-created_at` - gives you entries sorted by date of creation where the newest
entries will be listed first.

You can sort by multiple fields and any combination of sort directions: Example:  
`http://localhost:3000/api/v1/vault?sort=client_id&sort=-created_at` - gives you entries sorted by key. If multiple
entries have same `client_id`, they will be sorted by date of creation in descending order.

### Filter

You can filter entries by `id`, `client_id`, `created_by`, `created_at`, `key` fields. Example:  
`http://localhost:3000/api/v1/vault?filter[key]=one` will list you entries with the key=one.

Note: If you use curl to test filters, you should switch off URL globbing parser by providing `-g` flag
(see curl documentation for the details), e.g.:

```shell
curl -g -X GET 'http://localhost:3000/api/v1/vault?filter[created_by]=admin' \
-u admin:foobaz \
-H 'Content-Type: application/json'
```

You can combine filters for multiple fields:
`http://localhost:3000/api/v1/vault?filter[client_id]=client123&filter[created_by]=admin` -
gives you list of entries for client `client123` and created by `admin`

You can also specify multiple filter values e.g.
`http://localhost:3000/api/v1/vault?filter[client_id]=client123,client3` - gives you list of entries for client
`client123` or `client3`

You can also combine both sort and filter queries in a single request:  
`http://localhost:3000/api/v1/vault?sort=created_at&filter[client_id]=client123` - gives you entries for client
`client123` sorted by `created_at` in order of creation.

_Filters based on DateTime columns are not implemented at the moment._

### Read a secured value

You can get a single document with all fields and the decrypted value.

```shell
curl -X GET 'http://localhost:3000/api/v1/vault/1' \
-u admin:foobaz \
-H 'Content-Type: application/json'
```

The response will be

```json
{
    "data": {
        "client_id": "client123",
        "required_group": "",
        "key": "three",
        "value": "345",
        "type": "secret",
        "id": 1,
        "created_at": "2021-05-18T09:46:07+03:00",
        "updated_at": "2021-05-18T09:46:07+03:00",
        "created_by": "admin",
        "updated_by": "admin"
    }
}
```

In the "value" field you will find the decrypted secure value. If `required_group` value of the stored vault entry is
not empty, only users of this group can read this value, e.g. if `required_group` = 'Administrators' and the current
user doesn't belong to this group, an error will be returned.

### Add a new secured value

You can create a new secured value:

```shell
curl -X POST 'http://localhost:3000/api/v1/vault' \
-u admin:foobaz \
-H 'Content-Type: application/json' \
--data-raw '{
 "client_id": "client3",
 "required_group": "",
 "key": "four",
 "value": "4",
 "type": "string"
}'
```

The response will contain the id of the added element, which you can use then in the read value, deletion or changing APIs:

```json
{
    "data": {
        "id": 5
    }
}
```

### Fields info

`client_id`
: text, optional, Used to tie a document to a specific client where 0 means the document can be accessed from any client.

`required_group`
: text, optional, if filled, users not belonging to this group are not allowed to store or read the decrypted value.

`key`
:  text, required, some string to identify the document

`value`
: text, required representing the encrypted "body" of the document. All other columns hold clear text values.
  his column stores the encrypted data.

`type`
: text, required  ENUM('text', 'secret', 'markdown', 'string') Type of the secret value.

### Change a vault entry

You need to provide all fields like those you used to create a vault entry. Partial updates are not supported.
Additionally, you need to provide `id` of a stored value in the request url. You can get it by using the listing API.
You get the id also when you store a new value.

```shell
curl -X PUT 'http://localhost:3000/api/v1/vault/1' \
-u admin:foobaz \
-H 'Content-Type: application/json' \
--data-raw '{
 "client_id": "client3",
 "required_group": "",
 "key": "four",
 "value": "4",
 "type": "string"
}'
```

The response will contain the id of the added element:

```json
{
    "data": {
        "id": 1
    }
}
```

If `required_group` value of the entry you want to change is not empty, only users of this group can change this value,
otherwise an error will be returned.

### Delete a vault entry

To delete a vault entry, you need to provide id of an existing vault entry. You can get it by listing vault keys.
Additionally, id is provided when you create a new vault entry. You can delete a vault entry by calling the following API:

```shell
curl -X DELETE 'http://localhost:3000/api/v1/vault/1' \
-u admin:foobaz
```

If `required_group` value of the entry you want to delete is not empty, only users of this group can change this value,
otherwise an error will be returned.

## Create clear text backups of the vault

If you lose the passphrase of the vault, accessing the data is not possible anymore. A lost password can only be
recovered by so-called brute-force password probing.

Consider creating clear-text backups of the vault. Backups are performed via the API. You need a user of the
Administrator group and an API token for that user.

Below you find a simple script that dumps all entries of the vault to json text files.

```bash
USER=admin
TOKEN=e83d40e4-e237-43d6-bb99-35972ded631b
URL=http://localhost:3000/api/v1/vault

# Get all vault document ids
FOLDER=./vault-backup
mkdir ${FOLDER}
IDS=$(curl -s -u ${USER}:${TOKEN} ${URL}|jq .data[].id)
# Iterate over list of document ids
for ID in $IDS; do 
  curl -s -u ${USER}:${TOKEN} ${URL}/${ID} -o ${FOLDER}/${ID}.json
done
# Pack and compress
tar czf vault-backup.tar.gz ${FOLDER}
# Securely delete exported files
find ${FOLDER} -type f -exec shred {} \;
rm -rf ${FOLDER}
```
