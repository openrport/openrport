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

### Params:

- _name_ any text to identify the script
- _interpreter_  how will the script be executed on the client, e.g. /bin/sh, cmd.exe, powershell
- _sudo_ true or false if this script should be executed under a sudo user
- _cwd_ an optional directory where the script will be executed
- _script_ the text of the script to execute

### Update
You should know the script unique id to update it e.g. `4943d682-7874-4f7a-999c-b4ff5493fc3f`. 
You can use scripts list API to find ID of a corresponding script.

```
curl -X PUT 'http://localhost:3000/api/v1/scripts/4943d682-7874-4f7a-999c-b4ff5493fc3f' \
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
Please note, that you should provide all parameters as partial updates are not supported. 

### List scripts
This API allows to list all stored scripts.

```
curl -X GET 'http://localhost:3000/api/v1/scripts' \
-u admin:foobaz \
-H 'Content-Type: application/json'
```

The response will be

```
{
    "data": [
        {
            "id": "4943d682-7874-4f7a-999c-b4ff5493fc3f",
            "name": "current directory",
            "created_by": "admin",
            "created_at": "2021-05-18T09:30:27+03:00",
            "interpreter": "/bin/bash",
            "is_sudo": true,
            "cwd": "/root",
            "script": "pwd",
        },
        {
            "id": "4943d682-7874-4f7a-999c-12r2343241",
            "name": "hostname",
            "created_by": "admin",
            "created_at": "2021-05-19T19:31:57+03:00",
            "interpreter": "/bin/sh",
            "is_sudo": false,
            "cwd": "/root",
            "script": "hostname",
        },
    ]
}
```

### Sort
You can sort entries by `id`,`name`,`created_by`,`created_at` fields:

e.g. `http://localhost:3000/api/v1/scripts?sort=created_at` - gives you scripts sorted by date of creation in ascending order.

To change the sorting order by adding `-` to a field name.
e.g. `http://localhost:3000/api/v1/scripts?sort=-created_at` - gives you scripts sorted by date of creation where the newest entries will be listed first.

You can sort by multiple fields and any combination of sort directions:
e.g. `http://localhost:3000/api/v1/scripts?sort=created_at&sort=-name` - gives you entries sorted by creation date. If multiple entries are created at the same time, they will be sorted by name in descending order.

### Filter

You can filter entries by `id`,`name`,`created_by`,`created_at` fields:

`http://localhost:3000/api/v1/scripts?filter[name]=current_directory` will list scripts with the name `current_directory`.

You can combine filters for multiple fields:
`http://localhost:3000/api/v1/scripts?filter[name]=current_directory&filter[created_by]=admin` - gives you a list of scripts with name `current_directory` and created by `admin`.

You can also specify multiple filter values e.g.
`http://localhost:3000/api/v1/scripts?filter[name]=script1,scriptX` - gives you scripts `script1` or `scriptX`.

You can also combine both sort and filter queries in a single request:

`http://localhost:3000/api/v1/scripts?sort=created_at&filter[created_by]=admin` - gives you scripts created by `admin` sorted by `created_at` in order of creation.

### Delete
You should know the script unique id to delete it e.g. `4943d682-7874-4f7a-999c-b4ff5493fc3f`.
You can use scripts list API to find ID of a corresponding script.

```
curl -u admin:foobaz -X DELETE 'http://localhost:3000/api/v1/scripts/4943d682-7874-4f7a-999c-b4ff5493fc3f'
```

### Show a single script
To show a single script you need to know it's id e.g. `4943d682-7874-4f7a-999c-b4ff5493fc3f`.
You can use scripts list API to find ID of a corresponding script.

```
curl -u admin:foobaz -XGET 'http://localhost:3000/api/v1/scripts/4943d682-7874-4f7a-999c-b4ff5493fc3f'
```

The response will be:
```
{
    "data": {
        "id": "4943d682-7874-4f7a-999c-b4ff5493fc3f",
        "name": "current directory",
        "created_by": "admin",
        "created_at": "2021-05-18T09:30:27+03:00",
        "interpreter": "/bin/bash",
        "is_sudo": true,
        "cwd": "/root",
        "script": "pwd"
    }
}
```
