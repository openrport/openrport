# Vault
Rport provides a secure storage which allows to persist arbitrary data related to the clients and the environment.
All data is stored encrypted so user can safely store passwords used for log in to remote systems there.
Typical vault workflow looks as following:

- Administrator initialises vault by calling [vault init api](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/cloudradar-monitoring/rport/master/api-doc.yml#/Vault Init)
and giving a password in input. RPort stores password in a secure place. From this moment Rport vault will remain unlocked and can accept requests for reading or changing data.

- Administrator can lock vault by using [vault lock api](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/cloudradar-monitoring/rport/master/api-doc.yml#/Vault Lock).
RPort removes password and rejects all access requests to the vault. The same happens when server restarts. The vault database is persisted to hdd, so it will remain after server restarts. However, we would also recommend to back up the vault database additionally to survive also possible disk failures.

- Administrator can unlock vault by using [vault unlock api](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/cloudradar-monitoring/rport/master/api-doc.yml#/Vault Unlock).
He should provide password which he used on init stage. If a wrong password is provided, RPort will reject the vault access. In case of a correct password vault will be unlocked and can be used for reading or changing secure data.
  
A user, who doesn't belong to the Administrators group, is not allowed to use the vault management api.

## API Usage
The `/vault-admin` endpoints allow you to initialize, lock, unlock and read status of RPort vault.

### Initialize
[Administrator access]
This operation creates a new vault database and provisions it with some status information.

```
curl -X POST 'http://localhost:3000/api/v1/vault-admin/init' \
-u admin:foobaz \
-H 'Content-Type: application/json' \
--data-raw '{
	"password": "1234"
}'
```

Password length must be between 4 and 32 bytes, shorter and longer passwords are rejected.

### Status
[Any user access]
This API allows to read the current status of the RPort vault:

```
curl -X GET 'http://localhost:3000/api/v1/vault-admin' \
-u admin:foobaz \
-H 'Content-Type: application/json'
```

The response will be

```
{
    "data": {
        "init": "setup-completed",
        "status": "unlocked"
    }
}
```

The `init` field shows the initialization status of the vault. `setup-completed` means that the vault is already initialized or `uninitialized` otherwise.
The `status` field shows the lock status of the vault. It can be either `unlocked` or `locked`. `unlocked` status means that the vault is fully functional
and can be used to store, read or modify data securely. `unlocked` status means that any access to vault database will be rejected till administrator unlocks it again.

### Lock
[Administrator access]
This operation locks the vault, meaning that the password will be removed from server's memory and vault won't accept any requests.

```
curl -X DELETE 'http://localhost:3000/api/v1/vault-admin/sesame' \
-u admin:foobaz \
-H 'Content-Type: application/json'
```

### Unlock
[Administrator access]
This operation unlocks the vault. Administrator has to provide same password he used for the vault initialization. Rport will check the password validity and reject the request if a wrong password is provided. If administrator looses the password, all access to the data will be lost, so it should be kept in a secure place. If a correct password is provided, the vault will become fully functional.

```
curl -X POST 'http://localhost:3000/api/v1/vault-admin/sesame' \
-u admin:foobaz \
-H 'Content-Type: application/json' \
--data-raw '{
	"password": "1234"
}'
```
