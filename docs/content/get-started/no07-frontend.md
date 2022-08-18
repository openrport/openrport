---
title: "Rport Frontend"
weight: 07
slug: rport-frontend
aliases:
  - /docs/no07-frontend.html
  - /docs/content/get-started/no07-frontend.md
---
{{< toc >}}
Rport comes with a web-based graphical user interface (frontend) which is distributed as a separate bundle.

{{< hint type=warning >}}
Only the rport command-line tools – rport server and rport client – are released under the open-source MIT license.
The optional graphical user interface **is NOT open-source**, and free to use only under certain circumstances.
{{< /hint >}}

In short, the following is not covered by the [license](https://downloads.rport.io/frontend/license.html) and requires
acquiring a commercial license.

* Building a SaaS product or offering a hosted version of rport, either paid or free.
* Running rport and the UI and granting customers access to it, either paid or free.

Free usage in a company is allowed, as long as only employees of the company have access to rport.  
[Read the full license](https://downloads.rport.io/frontend/license.html). The uncompressed source code is not published.

## Installing the frontend

The frontend comes as a minified and compressed bundle of Javascript files and all needed assets. The frontend does not
require any server-side scripting support. The rport server provides static file serving for that purpose.

By default, the built-in web server listens only on localhost. Serving a web frontend on localhost is not very useful.
Change the listen address of the API to "0.0.0.0:3000" or any port you like.

Make sure you have the below options enabled in `[api]` section of the `rportd.conf`.

```text
[api]
  address = "0.0.0.0:3000"  
  doc_root = "/var/lib/rport/docroot"
```

{{< hint type=warning >}}
Usually you run rportd and the web frontend on a public server directly exposed to the internet. Running the API and
serving the frontend on unencrypted HTTP is dangerous. Always use HTTPs. The built-in web server supports HTTPs.
To quickly generate certificates, follow [this guide](/docs/get-started/no08-https-howto.md).
{{< /hint >}}

* Create the doc root folder. Usually `/var/lib/rport/docroot` is used.
* Download the latest release of the frontend from [https://downloads.rport.io/frontend/stable](https://downloads.rport.io/frontend/stable/?sort=time&order=desc).
* Unpack to the doc root folder.

```bash
mkdir /var/lib/rport/docroot
cd /var/lib/rport/docroot
wget -q https://downloads.rport.io/frontend/stable/latest.php -O rport-frontend.zip
unzip -qq rport-frontend.zip && rm -f rport-frontend.zip
cd ~
chown -R rport:rport /var/lib/rport/docroot
```

Now open the API-URL in a browser. Log in with a username and password specified for the
[API authentication](/docs/content/get-started/no02-api-auth.md).

You are done.
