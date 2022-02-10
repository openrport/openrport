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


