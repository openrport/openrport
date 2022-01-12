package chserver

import (
	"html/template"
	"net/http"
)

func (al *APIListener) getWsPrefix() string {
	p := "ws://"
	if al.config.API.CertFile != "" && al.config.API.KeyFile != "" {
		p = "wss://"
	}

	return p
}

func (al *APIListener) wsCommands(w http.ResponseWriter, r *http.Request) {
	wsPrefix := al.getWsPrefix()
	_ = homeTemplate.Execute(w, wsPrefix+r.Host+"/api/v1/ws/commands")
}

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script>
window.addEventListener("load", function(evt) {
   var output = document.getElementById("output");
   var input = document.getElementById("input");
   var token = document.getElementById("token");
   var ws;
   var print = function(message) {
       var d = document.createElement("div");
       d.textContent = message;
       output.appendChild(d);
   };
   document.getElementById("open").onclick = function(evt) {
       if (ws) {
           return false;
       }

       var wsURL = "{{.}}"+"?access_token=" + token.value;
       print("WS url: " + wsURL);
       ws = new WebSocket(wsURL);
       ws.onopen = function(evt) {
           print("OPEN");
       }
       ws.onclose = function(evt) {
           print("CLOSE");
           ws = null;
       }
       ws.onmessage = function(evt) {
           print("RESPONSE: " + evt.data);
       }
       ws.onerror = function(evt) {
           print("ERROR: " + evt.data);
       }
       return false;
   };
   document.getElementById("send").onclick = function(evt) {
       if (!ws) {
           return false;
       }
       print("SEND: " + input.value);
       ws.send(input.value);
       return false;
   };
   document.getElementById("close").onclick = function(evt) {
       if (!ws) {
           return false;
       }
       ws.close();
       return false;
   };
});
</script>
</head>
<body>
<table>
<tr><td valign="top" width="50%">
<p>Click "Open" to create a connection to the server,
"Send" to send a message to the server and "Close" to close the connection.
<p>
<form>
<button id="open">Open</button>
<button id="close">Close</button>
<p><textarea id="token" rows="3" cols="60" placeholder="Enter token here..."></textarea><p>
<textarea id="input" rows="5" cols="60" placeholder="Enter JSON request here...">
{
  "command": "/usr/bin/whoami",
  "client_ids": ["qa-lin-debian9", "qa-lin-debian10", "qa-lin-centos8", "qa-lin-ubuntu18", "qa-lin-ubuntu16"],
  "timeout_sec": 60,
  "abort_on_error": false,
  "execute_concurrently": true
}
</textarea>
<p>
<button id="send">Send</button>
</form>
</td><td valign="top" width="50%">
<pre id="output"></pre>
</td></tr></table>
</body>
</html>
`))

func (al *APIListener) wsScripts(w http.ResponseWriter, r *http.Request) {
	wsPrefix := al.getWsPrefix()
	_ = scriptsTemplate.Execute(w, wsPrefix+r.Host+"/api/v1/ws/scripts")
}

var scriptsTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script>
window.addEventListener("load", function(evt) {
   var ws;
   var print = function(message) {
       var outCont = document.getElementById("output");
       outCont.value += message + "\n";
   };
   document.getElementById("open").onclick = function(evt) {
	   var token = document.getElementById("token");
	   var params = {
			access_token: token.value,
      };
       if (ws) {
           return false;
       }
		var queryString = Object.keys(params).map(function(key) {
			return key + '=' + params[key]
		}).join('&');
		
       var wsURL = "{{.}}"+"?" + queryString;

       print("WS url: " + wsURL);
       ws = new WebSocket(wsURL);
       ws.onopen = function(evt) {
           print("OPEN");
       }
       ws.onclose = function(evt) {
           print("CLOSE");
           ws = null;
       }
       ws.onmessage = function(evt) {
            const data = JSON.parse(evt.data);
			var str = JSON.stringify(data, null, 2);
			print("RESP:");
			print(str);
       }
       ws.onerror = function(evt) {
           print("ERROR: " + evt.data);
       }
       return false;
   };
   document.getElementById("send").onclick = function(evt) {
	   var input = document.getElementById("input");
	   var script = document.getElementById("script");
       if (!ws) {
           return false;
       } 
	   var inputObj = JSON.parse(input.value);
	   inputObj.script = btoa(script.value);
       var inputStr = JSON.stringify(inputObj, null, 2);

       print("SEND: " + inputStr);
       ws.send(inputStr);
       input.value = inputStr;

       return false;
   };
   document.getElementById("close").onclick = function(evt) {
       if (!ws) {
           return false;
       }
       ws.close();
       return false;
   };
});
</script>
</head>
<body>
<table>
<tr><td valign="top" width="50%">
<p>Click "Open" to create a connection to the server,
"Send" to send a message to the server and "Close" to close the connection.
<p>
<form>
<button id="open">Open</button>
<button id="close">Close</button>
<p><label for="token">Token</label>
<br/>
<textarea id="token" name="token" rows="3" cols="60"></textarea>
</p>
<p><label for="token">Script</label>
<br/>
<textarea id="script" name="script" rows="3" cols="60"></textarea>
</p>
<label for="input">Input</label><br/>
<textarea id="input" rows="5" cols="60">
{
  "script": "cHdk",
  "client_ids": ["qa-lin-windows1", "qa-lin-windows2"],
  "timeout_sec": 60,
  "abort_on_error": false,
  "execute_concurrently": true,
  "interpreter":"powershell"
}
</textarea>
</p>
<button id="send">Send</button>
</form>
</td><td valign="top" width="45%">
<p>Output</p>
<textarea id="output" rows="35" cols="50" style="width:90%;"></textarea>
</td></tr></table>
</body>
</html>
`))

func (al *APIListener) wsUploads(w http.ResponseWriter, r *http.Request) {
	wsPrefix := al.getWsPrefix()
	_ = uploadsTemplate.Execute(w, wsPrefix+r.Host+"/api/v1/ws/uploads")
}

var uploadsTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script>
window.addEventListener("load", function(evt) {
   var ws;
   var print = function(message) {
       var outCont = document.getElementById("output");
       outCont.value += message + "\n";
   };
   document.getElementById("open").onclick = function(evt) {
	   var token = document.getElementById("token");
	   var params = {
			access_token: token.value,
      };
       if (ws) {
           return false;
       }
		var queryString = Object.keys(params).map(function(key) {
			return key + '=' + params[key]
		}).join('&');
		
       var wsURL = "{{.}}"+"?" + queryString;

       print("WS url: " + wsURL);
       ws = new WebSocket(wsURL);
       ws.onopen = function(evt) {
           print("OPEN");
       }
       ws.onclose = function(evt) {
           print("CLOSE");
           ws = null;
       }
       ws.onmessage = function(evt) {
            const data = JSON.parse(evt.data);
			var str = JSON.stringify(data, null, 2);
			print("RESP:");
			print(str);
       }
       ws.onerror = function(evt) {
           print("ERROR: " + evt.data);
       }
       return false;
   };
   document.getElementById("close").onclick = function(evt) {
       if (!ws) {
           return false;
       }
       ws.close();
       return false;
   };
});
</script>
</head>
<body>
<table>
<tr><td valign="top" width="50%">
<p>Click "Open" to create a connection to the server,
<p>
<form>
<p><label for="token">Token</label>
<br/>
<textarea id="token" name="token" rows="3" cols="60"></textarea>
</p>
<button id="open">Open</button>
<button id="close">Close</button>
</form>
</td><td valign="top" width="45%">
<p>Output</p>
<textarea id="output" rows="35" cols="50" style="width:90%;"></textarea>
</td></tr></table>
</body>
</html>
`))
