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
- _interpreter_  script syntax interpreter which is used for execution, possible values are sh, cmd.exe, powershell, taco, default values: sh (under Linux) and cmd.exe (under Windows)
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
On the client using the `rport.conf` you can enable or disable execution of remote scripts.
```
[remote-scripts]
## Enable or disable remote scripts.
## Defaults: true
#enabled = true
```

Please note that scripts execution requires commands execution to be enabled (check https://oss.rport.io/docs/no06-command-execution.html#securing-your-environment or `[remote-commands]` enabled flag of configuration).

Similar to command execution, you can run scripts both by calling a REST or websocket interface. 
In all cases the scripts are executed by the following algorithm:
- Rport server calls each client to create a script file. The script file will get a random unique name and will be placed in the folder specified as `script_dir` in the configuration:

```
[remote-scripts]
script_dir = '/var/lib/rport/scripts'
```

- Rport calls the existing command API to execute the script on the target client, e.g.:
```
sh /tmp/f68a779d-1d46-414a-b165-d8d2df5f348c.sh #Linux/macOS

cmd C:\Users\me\AppData\Local\Temp\f68a779d-1d46-414a-b165-d8d2df5f348c.bat #Windows non-powershell execution

powershell -executionpolicy bypass -file C:\Users\me\AppData\Local\Temp\f68a779d-1d46-414a-b165-d8d2df5f348c.ps1 #Windows powershell execution
```

- Rport deletes the temp script file in any case disregard if script execution fails or not.

**NOTE**
To execute scripts you need to allow execution of the corresponding script command (see https://oss.rport.io/docs/no06-command-execution.html#securing-your-environment)

### Script execution via REST interface
To execute script on a certain client you would need a client id e.g. `b6b28b13-be4a-4a78-bb39-eca132c434fb` and an access token.
The script's payload should be base64 encoded:

```
curl -X POST 'http://localhost:3000/api/v1/clients/4943d682-7874-4f7a-999c-b4ff5493fc3f/scripts' \
-H 'Authorization: Bearer eyJhbGcidfasjflInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImdfasfdjoiMTEzMzkyNjMxNTA0MDYwOTU1MCJ9.JG4whDXeDKDuZqgVA \
-H 'Content-Type: application/javascript' \
--data-raw '{
  "script": "cHdkCg==",
  "timeout_sec": 60,
  "interpreter":"powershell"
}'
```

as a result you will get a unique job id for the executable script e.g.

```
{
    "data": {
        "jid": "24bd86ba-1fd4-48c1-9620-6879f196b8de"
    }
}
```

to customize script execution you can provide additional JSON fields in the request body:

- `interpreter:powershell` to execute script with powershell under Windows
- `is_sudo:true` to execute script as sudo under Linux or macOS
- `cwd:/tmp/script` to change the default folder for the script execution
- `timeout:10s` to set the timeout for the corresponding script execution

Here is an example of a script execution with all parameters enabled:
```
curl -X POST 'http://localhost:3000/api/v1/clients/4943d682-7874-4f7a-999c-b4ff5493fc3f/scripts' \
-H 'Authorization: Bearer eyJhbGcidfasjflInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImdfasfdjoiMTEzMzkyNjMxNTA0MDYwOTU1MCJ9.JG4whDXeDKDuZqgVA \
-H 'Content-Type: application/x-www-form-urlencoded' \
--data-raw '{
  "script": "cHdkCg==",
  "interpreter": "cmd",
  "timeout_sec": 60,
  "execute_concurrently": false,
  "abort_on_error": true,
  "cwd": "string",
  "is_sudo": true
}'
```

you can check the status of the command execution by calling [client commands API](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/cloudradar-monitoring/rport/master/api-doc.yml#/Commands/get_clients__client_id__commands__job_id_).

You can execute a script on multiple clients by calling `scripts` API, in this case you should provide client ids in the input body:

```
curl -X POST 'http://localhost:3000/api/scripts' \
-H 'Authorization: Bearer eyJhbGcidfasjflInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImdfasfdjoiMTEzMzkyNjMxNTA0MDYwOTU1MCJ9.JG4whDXeDKDuZqgVA \
-H 'Content-Type: application/javascript' \
--data-raw '{
  "script": "cHdkCg==",
  "client_ids": [
    "8e995525-b18f-44a4-ae83-f3b8fd5a5ff8",
    "eabb80c4-fa17-46e5-a965-0ea442fa0e83"
  ],
  "timeout_sec": 60,
  "execute_concurrently": false,
  "abort_on_error": true,
  "interpreter":"powershell"
}'
```

### Script execution via websocket interface
You can use [our testing API for Websockets](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/cloudradar-monitoring/rport/master/api-doc.yml#/Scripts/get_ws_scripts). 
To use this API, enable testing endpoints by setting `enable_ws_test_endpoints` flag to true in the `[server]` section of configuration file:
```
[server]
...
  enable_ws_test_endpoints = true
```

Restart Rport server and go to: `{YOUR_RPORT_ADDRESS}/api/v1/test/scripts/ui`

Put an access token and a client ids in the corresponding fields. You can also provide `interpreter`, `is_sudo`, `cwd`, `timeout` parameters as described above.

Click Open to start websocket connection.

Put the input data in JSON format with the base64 encoded script to the input field and click Send. The payload will be transmitted via Websocket protocol. Once the clients finish the execution, they will send back the response which you'll see in the Output field.

### Execution of taco scripts
[Taco scripts](https://github.com/cloudradar-monitoring/tacoscript) interpreter can be used to execute scripts in a Saltstack similar format for both Windows and Linux machines. Tacoscript interpreter doesn't require additional libraries or tools to be installed in the system and it has capabilities for:
- conditional execution depending on command exit codes, present/missing files, host system information (e.g. os version)
- installing/uninstalling/upgrading packages
- creating files
- dependant executions (e.g. script A depends on execution of script B etc)
- reserved values from the information about the host system
- reusable variables

To execute a taco script, you need to specify `taco` as a script interpreter, e.g.

```
curl -X POST 'http://localhost:3000/api/v1/clients/4943d682-7874-4f7a-999c-b4ff5493fc3f/scripts' \
-H 'Authorization: Bearer eyJhbGcidfasjflInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImdfasfdjoiMTEzMzkyNjMxNTA0MDYwOTU1MCJ9.JG4whDXeDKDuZqgVA \
-H 'Content-Type: application/x-www-form-urlencoded' \
--data-raw '{
  "script": "IwojIEZpcnN0IGV4YW1wbGUgb2YgYSB0YWNvc2NyaXB0IGZvbGxvd2luZyB0aGUgc3ludGF4IG9mIFNhbHQKIyBidXQgbm90IGltcGxlbWVudGluZyBhbGwgb3B0aW9ucwojIGh0dHBzOi8vZG9jcy5zYWx0c3RhY2suY29tL2VuL2xhdGVzdC9yZWYvc3RhdGVzL2FsbC9zYWx0LnN0YXRlcy5jbWQuaHRtbAojCiMgdW5pcXVlIGlkIG9mIHRoZSB0YXNrLCBjYW4gYmUgYW55IHN0cmluZwpkYXRlIGNvbW1hbmQ6CiAgY21kLnJ1bjoKICAgIC0gbmFtZXM6CiAgICAgIC0gZGF0ZQo=",
  "interpreter": "taco"
}'
```
Where the base64 encoded script looks like this:

```
#
# First example of a tacoscript following the syntax of Salt
# but not implementing all options
# https://docs.saltstack.com/en/latest/ref/states/all/salt.states.cmd.html
#
# unique id of the task, can be any string
date command:
  cmd.run:
    - names:
      - date
```
As a result this script will output the current date. 
In order to execute taco scripts, there should be taco binary avalable in the system path (see here [the installation instructions](https://github.com/cloudradar-monitoring/tacoscript#installation))
