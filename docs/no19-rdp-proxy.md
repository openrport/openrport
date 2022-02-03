# RDP-Proxy 

## Preface
Starting with version 0.6.0 RPort has the ability to create a tunnel to a remote RDP server with a built-in HTTPS proxy. 
RPort uses the Apache Guacamole Server to connect to the remote RDP Server, which brings the remote desktop into your browser without the need of an RDP viewer.
If a rdp tunnel with https proxy is created, RPort first creates the tunnel to the remote machine and makes it available through the https proxy with path "/".
Pointing the browser to this URL creates a websocket tunnel to connect the browser with the guacamole server and starts the connection to the remote RDP server.

## Prerequisites
* Apache Guacamole Server has to be running on 127.0.0.1:4822

## Install Apache Guacamole Server

* Either build Guacamole Server from source and run it, which is described [here](http://guacamole.incubator.apache.org/doc/gug/installing-guacamole.html).
* Or use one of the provided Docker images for guacd, e.g. from [linuxserver.io](https://docs.linuxserver.io/images/docker-guacd)
```
docker pull lscr.io/linuxserver/guacd
docker run -d --name=guacd -p 4822:4822 --net=host --restart unless-stopped lscr.io/linuxserver/guacd
```
Important: docker run with `--net=host` to connect to RPort tunnel on 127.0.0.1


