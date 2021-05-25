# Scripts
Rport allows to store your scripts for later reuse, so you can share them with your teammates and to have access to them from anywhere.

You can manage script with the [REST API](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/cloudradar-monitoring/rport/master/api-doc.yml#/Scripts).

## API Usage
The `/scripts` endpoints allow you to create, update, delete and list scripts.

### Create
To create a script, provide following input:

```
curl -X POST 'http://localhost:3000/api/v1/scripts' \
-u admin:foobaz \
-H 'Content-Type: application/json' \
--data-raw '{
	"name": "current_directory",
	"interpreter": "/bin/sh",
	"is_sudo": true,
	"cwd": "/root",
	"script": "pwd"
}'
```
### Update
You can provide any parameters that you want to change. 
```
curl -X PUT 'http://localhost:3000/api/v1/users/user1' \
-u admin:foobaz \
-H 'Content-Type: application/json' \
--data-raw '{
    "password": "1234567"
    "groups":
    {
        "Users"
    }
}'
```
This will change password and remove user from Administrators group. To add user to a new group, you should provide all current user groups + a new one e.g.
```
{
    "groups":
    {
        "Users",
        "Administrators",
        "New Group"
    }
}
```

Please note that all changes to the user affecting credentials will have an immediate effect in most cases disregard if you use JWT or basic password auth (e.g. deletion user from Administrators group), so you should use this API carefully.
If you change a password, user will still be able to login with an old JWT token, so the change will work till the next login.

### List all users
```
curl -s -u admin:foobaz http://localhost:3000/api/v1/users
{
    "data": [
        {
            "username": "admin",
            "groups": [
                "Administrators",
                "Users"
            ]
        },
        {
            "username": "root",
            "groups": [
                "Users"
            ]
        }
    ]
}
```
Because of security consideration, users list won't return hashed passwords.
### Delete
```
curl -u admin:foobaz -X DELETE 'http://localhost:3000/api/v1/users/user1'
```
