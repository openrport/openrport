# Rport Frontend
Rport comes with a web-based graphical user interface (frontend) which is distributed as a separate bundle.

## WARNING
Only the rport command-line tools – rport server and rport client – are released under the open-source MIT license. The optional graphical user interface **is NOT open-source**, and free to use only under certain circumstances.

In short, the following is not covered by the [license](https://downloads.rport.io/license.html) and requires acquiring a commercial license. 
* Building a SaaS product or offering a hosted version of rport, either paid or free.
* Running rport and the UI and granting customers access to it, either paid or free.

Free usage in a company is allowed, as long as only employees of the company have access to rport.  [Read the full license](https://downloads.rport.io/license.html).
The uncompressed source code is not published.

## Installing the frontend
The frontend comes as a minified and compressed bundle of Javascript files and all needed assets. The frontend does not require any server-side scripting support. The rport server provides static file serving for that purpose. 

Make sure you have the below options enabled in `[api]` section of the `rportd.conf`.

```
doc_root = "/var/lib/rport/docroot"          # Linux
doc_root = 'C:\Program Files\rport\docroot'  # Windows
```
* Download the latest release of the frontend [here](https://downloads.rport.io/).
* Unpack to the `doc_root` folder.
* Open the API-URL in a browser.
* Log in with a username and password specified for the API authentication.

You are done.
