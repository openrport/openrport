# Rport Frontend
Rport comes with a web-based graphical user interface (frontend) which is distributed as a separate bundle.

::: warning
Only the rport command-line tools – rport server and rport client – are released under the open-source MIT license. The optional graphical user interface **is NOT open-source**, and free to use only under certain circumstances.

In short, the following is not covered by the [license](https://downloads.rport.io/frontend/license.html) and requires acquiring a commercial license.
* Building a SaaS product or offering a hosted version of rport, either paid or free.
* Running rport and the UI and granting customers access to it, either paid or free.

Free usage in a company is allowed, as long as only employees of the company have access to rport.  [Read the full license](https://downloads.rport.io/frontend/license.html).
The uncompressed source code is not published.
:::

## Installing the frontend
The frontend comes as a minified and compressed bundle of Javascript files and all needed assets. The frontend does not require any server-side scripting support. The rport server provides static file serving for that purpose. 

By default the built-in websserver listens only on localhost. Serving a web frontend on localhost is not very useful. Change the listen address of the API to "0.0.0.0:3000" or any port you like. 

Make sure you have the below options enabled in `[api]` section of the `rportd.conf`.
```
[api]
  address = "0.0.0.0:3000"  
  doc_root = "/var/lib/rport/docroot"
```

::: danger
Usually you run rportd and the web frontend on a public server directly exposed to the internet. Running the API and serving the frontend on unencrypted HTTP is dangerous. Always use HTTPs. The built-in web server supports HTTPs. To quickly generate certificates, follow [this guide](https://oss.rport.io/docs/no08-https-howto.html).
:::

* Create the doc root folder, `mkdir /var/lib/rport/docroot`
* Download the latest release of the frontend from [https://downloads.rport.io/frontend/stable](https://downloads.rport.io/frontend/stable/?sort=time&order=desc).
* Unpack to the doc root folder.
* Open the API-URL in a browser.
* Log in with a username and password specified for the [API authentication](https://github.com/cloudradar-monitoring/rport/blob/master/docs/api-auth.md).

You are done.