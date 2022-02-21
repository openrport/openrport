# RDP-Proxy 

## Preface
Starting with version 0.6.0 RPort has the ability to create a tunnel to a remote RDP server with a built-in HTTPS proxy. 
RPort uses the Apache Guacamole Server to connect to the remote RDP Server, which brings the remote desktop into your browser without the need of an RDP viewer.
If a rdp tunnel with https proxy is created, RPort first creates the tunnel to the remote machine and makes it available through the https proxy with path "/".
Pointing the browser to this URL creates a websocket tunnel to connect the browser with the guacamole server and starts the connection to the remote RDP server.

## Prerequisites
* Apache Guacamole Server 1.4.X has to be running on 127.0.0.1

::: tip
👉 **You don't need a fully featured Guacamole installation.** Only the `guacd` is needed, which is a lightweight and easy to install daemon. It does not require any configuration or special maintenance.
:::

## Install Apache Guacamole Server

To run the guacamole server `guacd` you have the following options:

* Use the pre-compiled packges for Ubuntu or Debian we have perpared to run the `guacd` just for RPort. [downloads and instructions](https://bitbucket.org/cloudradar/rport-guacamole/src/main/)
* Build the Guacamole Server from source and run it, which is described [here](http://guacamole.incubator.apache.org/doc/gug/installing-guacamole.html).
* Use one of the provided Docker images for guacd, e.g. from [linuxserver.io](https://docs.linuxserver.io/images/docker-guacd)
  ```
  docker pull lscr.io/linuxserver/guacd
  docker run -d --name=guacd -p 4822:4822 --net=host --restart unless-stopped lscr.io/linuxserver/guacd
  ```
  Important: docker run with `--net=host` to connect to RPort tunnel on 127.0.0.1

## Frontend Options
If the browser links to the exposed proxy port with "/", query parameters can be set to control the rdp connection. 

E.g. `/?username=Administrator&width=800&height=600&security=nla&keyboard=de-de-qwertz`

* `username` is the pre-filled login user on the remote machine. For security reasons, the password cannot be injected.
* `width` for the required screen width and `height` for the required screen height. If `width` and `height` are omitted, 1024 x 768 are the default values.
* `security` with one of the following values: `any`, `nla`, `nla-ext`, `tls`, `vmconnect`, `rdp`. `nla` is the most commonly used security option on MS Windows. Read more on the [Guacamole documentation](https://guacamole.apache.org/doc/gug/configuring-guacamole.html#authentication-and-security).
* `keyboard` to identify the local keyboard of the user, not the desired remote keyboard, with one of the following options. 
   * Brazilian (Portuguese), `pt-br-qwerty`
   * English (UK) `en-gb-qwerty`
   * English (US) `en-us-qwerty`
   * French `fr-fr-azerty`
   * French (Belgian) `fr-be-azerty`
   * French (Swiss) `fr-ch-qwertz`
   * German `de-de-qwertz`
   * German (Swiss) `de-ch-qwertz`
   * Hungarian `hu-hu-qwertz`
   * Italian `it-it-qwerty`
   * Japanese `ja-jp-qwerty`
   * Norwegian `no-no-qwerty`
   * Spanish `es-es-qwerty`
   * Spanish (Latin American) `es-latam-qwerty`
   * Swedish `sv-se-qwerty`
   * Turkish-Q `tr-tr-qwerty`
   
   Read more on the [Guacamole documentation](https://guacamole.apache.org/doc/gug/configuring-guacamole.html#session-settings)
