# Scripts
Rport allows to store your scripts for later reuse, so you can share them with your teammates and to have access to them from anywhere.

## Scripts management
You can manage script with the [REST API](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/cloudradar-monitoring/rport/master/api-doc.yml#/Scripts).

The `/library/scripts` endpoints allow you to create, update, delete and list scripts.

### Create
To create a script, provide following input:

```
curl -X POST 'http://localhost:3000/api/v1/library/scripts' \
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
curl -X PUT 'http://localhost:3000/api/v1/library/scripts/4943d682-7874-4f7a-999c-b4ff5493fc3f' \
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
curl -X GET 'http://localhost:3000/api/v1/library/scripts' \
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

e.g. `http://localhost:3000/api/v1/library/scripts?sort=created_at` - gives you scripts sorted by date of creation in ascending order.

To change the sorting order by adding `-` to a field name.
e.g. `http://localhost:3000/api/v1/library/scripts?sort=-created_at` - gives you scripts sorted by date of creation where the newest entries will be listed first.

You can sort by multiple fields and any combination of sort directions:
e.g. `http://localhost:3000/api/v1/library/scripts?sort=created_at&sort=-name` - gives you entries sorted by creation date. If multiple entries are created at the same time, they will be sorted by name in descending order.

### Filter

You can filter entries by `id`,`name`,`created_by`,`created_at` fields:

`http://localhost:3000/api/v1/library/scripts?filter[name]=current_directory` will list scripts with the name `current_directory`.

You can combine filters for multiple fields:
`http://localhost:3000/api/v1/library/scripts?filter[name]=current_directory&filter[created_by]=admin` - gives you a list of scripts with name `current_directory` and created by `admin`.

You can also specify multiple filter values e.g.
`http://localhost:3000/api/v1/library/scripts?filter[name]=script1,scriptX` - gives you scripts `script1` or `scriptX`.

You can also combine both sort and filter queries in a single request:

`http://localhost:3000/api/v1/library/scripts?sort=created_at&filter[created_by]=admin` - gives you scripts created by `admin` sorted by `created_at` in order of creation.

### Delete
You should know the script unique id to delete it e.g. `4943d682-7874-4f7a-999c-b4ff5493fc3f`.
You can use scripts list API to find ID of a corresponding script.

```
curl -u admin:foobaz -X DELETE 'http://localhost:3000/api/v1/library/scripts/4943d682-7874-4f7a-999c-b4ff5493fc3f'
```

### Show a single script
To show a single script you need to know it's id e.g. `4943d682-7874-4f7a-999c-b4ff5493fc3f`.
You can use scripts list API to find ID of a corresponding script.

```
curl -u admin:foobaz -XGET 'http://localhost:3000/api/v1/library/scripts/4943d682-7874-4f7a-999c-b4ff5493fc3f'
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

## Scripts execution
Similar to command execution, you can run scripts both by calling a REST api or triggering scripts via websocket interface. 
In all cases the scripts are executed by the following algorithm:
- Rport server creates a temp script file in a target client. Depending on the client's OS it will create either a shell script file for Linux/macOS or a cmd/powershell script for Windows. The script file will get a random unique name and will be placed in the OS default temp folder e.g. `/tmp` in Linux.
- Rport calls the existing command API to execute the script on the target client, e.g.:
```
sh /tmp/f68a779d-1d46-414a-b165-d8d2df5f348c.sh #Linux/macOS

cmd C:\Users\me\AppData\Local\Temp\f68a779d-1d46-414a-b165-d8d2df5f348c.bat #Windows non-powershell execution

powershell -executionpolicy bypass -file C:\Users\me\AppData\Local\Temp\f68a779d-1d46-414a-b165-d8d2df5f348c.ps1 #Windows powershell execution
```

- Rport also deletes the temp script file in any case disregard if script execution fails or not. To achieve this we append the removal instruction to the command above as following:

```
sh /tmp/f68a779d-1d46-414a-b165-d8d2df5f348c.sh; rm /tmp/f68a779d-1d46-414a-b165-d8d2df5f348c.sh

cmd C:\Users\me\AppData\Local\Temp\f68a779d-1d46-414a-b165-d8d2df5f348c.bat & del C:\Users\me\AppData\Local\Temp\f68a779d-1d46-414a-b165-d8d2df5f348c.bat

powershell -executionpolicy bypass -file C:\Users\me\AppData\Local\Temp\f68a779d-1d46-414a-b165-d8d2df5f348c.ps1; del C:\Users\me\AppData\Local\Temp\f68a779d-1d46-414a-b165-d8d2df5f348c.bat
```

The `;` or `&` concatenation assures that deletion command will be executed no matter if the previous command finishes with success or failure.

**NOTE**
To execute scripts you need to allow execution of the corresponding script command (see https://oss.rport.io/docs/no06-command-execution.html#securing-your-environment)

### Script execution via REST interface
To execute script on a certain client you would need a client id e.g. `b6b28b13-be4a-4a78-bb39-eca132c434fb` and access token:

```
curl -X POST 'http://localhost:3000/api/v1/clients/4943d682-7874-4f7a-999c-b4ff5493fc3f/scripts' \
-H 'Authorization: Bearer eyJhbGcidfasjflInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImdfasfdjoiMTEzMzkyNjMxNTA0MDYwOTU1MCJ9.JG4whDXeDKDuZqgVA \
-H 'Content-Type: application/x-www-form-urlencoded' \
--data-raw 'touch test.txt
echo "Hello world" > test.txt
cat test.txt'
```

as a result you will get a unique job id for the executable script e.g.

```
{
    "data": {
        "jid": "24bd86ba-1fd4-48c1-9620-6879f196b8de"
    }
}
```

to customize script execution you can provide additional URL parameters:

- `isPowershell=true` to execute script with powershell under Windows
- `isSudo=true` to execute script as sudo under Linux or macOS
- `cwd=/tmp/script` to change the default folder for the script execution
- `timeout=10s` to set the timeout for the corresponding script execution

Here is an example of a script execution with all parameters enabled:
```
curl -X POST 'http://localhost:3000/api/v1/clients/4943d682-7874-4f7a-999c-b4ff5493fc3f/scripts?isPowershell=true&isSudo=false&cwd=/tmp/script&timeout=10s' \
-H 'Authorization: Bearer eyJhbGcidfasjflInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImdfasfdjoiMTEzMzkyNjMxNTA0MDYwOTU1MCJ9.JG4whDXeDKDuZqgVA \
-H 'Content-Type: application/x-www-form-urlencoded' \
--data-raw 'touch test.txt
echo "Hello world" > test.txt
cat test.txt'
```

you can check the status of the command execution by calling [client commands API](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/cloudradar-monitoring/rport/master/api-doc.yml#/Commands/get_clients__client_id__commands__job_id_).

### Script execution via websocket interface
You can use [our testing API for Websockets](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/cloudradar-monitoring/rport/master/api-doc.yml#/Scripts/get_ws_scripts). 
To use this API, enable testing endpoints by setting `enable_ws_test_endpoints` flag to true in the `[server]` section of configuration file:
```
[server]
...
  enable_ws_test_endpoints = true
```

Restart Rport server and go to: `{YOUR_RPORT_ADDRESS}/api/v1/test/scripts/ui`

Put an access token and a client id in the corresponding fields. You can also provide `isPowershell`, `isSudo`, `cwd`, `timeout` parameters as described above.

Click Open to start websocket connection.

Put your script to the script field and click Send. The script text will be transmitted via Websocket protocol. Once the client finishes the execution it will send back the response which you'll see in the Output field. 
