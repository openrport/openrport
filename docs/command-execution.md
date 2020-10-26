## Command execution
Via the API you can execute a command on connected clients. 
The command and the response are transferred through the web socket connection. A tunnel is not needed.
### Execute on a single host
Example:
```
SESSIONID=my-client
curl -s -u admin:foobaz http://localhost:3000/api/v1/sessions/$SESSIONID/commands -H "Content-Type: application/json" -X POST \
--data-raw '{
  "command": "date",
  "timeout_sec": 10
}'|jq
```
You will get back a job id.
Now execute a query to get the result of the command.

```
JOBID=f72b69fd-f418-40c3-ab62-4ce2c2022c58
curl -s -u admin:foobaz http://localhost:3000/api/v1/sessions/$SESSIONID/commands/$JOBID|jq
{
    "data": {
        "jid": "f72b69fd-f418-40c3-ab62-4ce2c2022c58",
        "status": "successful",
        "finished_at": "2020-10-15T15:30:12.937267522Z",
        "sid": "my-client",
        "command": "date",
        "shell": "/bin/sh",
        "pid": 908526,
        "started_at": "2020-10-15T15:30:12.934238782Z",
        "created_by": "admin",
        "timeout_sec": 10,
        "result": {
            "stdout": "Thu Oct 15 15:30:12 UTC 2020\n",
            "stderr": ""
        }
    }
}
```

The rport client supervises the command for the given {timeout_sec} (in seconds) period. If the timeout is exceeded the command state is considered 'unknown' but the command keeps running. 



### Execute on multiple hosts
(Not implemented yet.)

### Securing your environment (NOT IMPLEMENTED YET)
The commands are executed from the account that runs rport.
On Linux this by default an unprivileged user. Do not run rport as root.
On Windows, rport runs as a local service account that by default has administrative rights.

On the client using the `rport.conf` you can configure and limit the execution of remote commands.
```
[remote-commands]
## Enable or disable remote commands.
## Defaults: True
#enabled = True

## Only allow commands matching the following regular expressions.
## The filter is applied to the command sent and not to the full path where the command is located.
## Defaults: ['.*']
#allow = ['.*']

## Deny any commands matching one of the following regular expressions
## The filter is applied to the command sent.
## With the below default filter only single commands are allowed.
## Defaults: ['(\||<|>|;|\\|\n)']
#deny = ['(\||<|>|;|\\\n)']

## Order of which filter is applied first.
## Defaults: ['deny','allow']
#order = ['deny','allow']

## Limit the maximum length of the command output that is sent back.
## Applies to the sum of stdout and stderr.
## If exceeded {send_back_limit} bytes are sent.
## Defaults: 4096
#send_back_limit = 4096
```

**Examples:**

On Linux only allow commands in `/usr/bin` and `/usr/local/bin` and command prefixed with `sudo -n`.
```
allow = [
    '^\/usr\/bin\/.*',
    '^\/usr\/local\/bin\/.*',
    '^sudo -n .*'
]
```

On Windows try this
```
allow = [
    '^C:\\\Windows\\\System32.*',
    '^C:\\\Users\\\Administrator\\\scripts\\\*.\.bat'
]
```
Using the above examples requires sending commands with a full path. 