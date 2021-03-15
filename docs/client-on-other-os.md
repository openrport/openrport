# Install and run a rport client
## On Mac OS (intel based)
Open the terminal and as an unprivileged user download the binary and put it in `/usr/local/bin`
```
curl -OL https://github.com/cloudradar-monitoring/rport/releases/download/0.1.25/rport_0.1.25_Darwin_x86_64.tar.gz
test -e /usr/local/bin/||sudo mkdir /usr/local/bin
sudo tar xzf rport_0.1.25_Darwin_x86_64.tar.gz -C /usr/local/bin/ rport
sudo mkdir /etc/rport
tar xzf rport_0.1.25_Darwin_x86_64.tar.gz rport.example.conf
sudo mv rport.example.conf /etc/rport/rport.conf
sudo mkdir /var/log/rport
```

Now open the configuration file with an editor and enter your server URL, credentials, and fingerprint.
```
sudo vim /etc/rport/rport.conf
```

Before registering a service, test it with `rport -c /etc/rport/rport.conf`. You should not get any output and the new client should appear on the server.

For registering the service you have two options. 
1. You run it with your own user. 
2. You create a so-called daemon user on Mac OS [following this guide](https://gist.github.com/mwf/20cbb260ad2490d7faaa).

Register and run the service.
```
sudo rport --service install --service-user <USERNAME> -c /etc/rport/rport.conf
sudo rport --service start
```

If you are in doubt the service has been created run `sudo launchctl list|grep rport`. It should list the pid on the first column to indicate rport is running.
```
$ sudo launchctl list|grep "rport$"
9942	0	rport
```
If you get an output like this, the installation of the service has succeeded but rport cannot start.

```
$ sudo launchctl list|grep "rport$"
-	0	rport
```

Missing write permissions to the folder `/usr/local/var/log/` are most likely the reason.
Open `/Library/LaunchDaemons/rport.plist` with an editor and use `tmp` as log directory for the start-up logs.
```
<?xml version='1.0' encoding='UTF-8'?>
<!DOCTYPE plist PUBLIC "-//Apple Computer//DTD PLIST 1.0//EN"
"http://www.apple.com/DTDs/PropertyList-1.0.dtd" >
<plist version='1.0'>
  <dict>
    <key>Label</key>
    <string>rport</string>
    <key>ProgramArguments</key>
    <array>
      <string>/usr/local/bin/rport</string>

      <string>-c</string>

      <string>/etc/rport/rport.conf</string>

    </array>
    <key>UserName</key>
    <string>hero</string>


    <key>SessionCreate</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>RunAtLoad</key>
    <false/>
    <key>Disabled</key>
    <false/>

    <key>StandardOutPath</key>
    <string>/tmp/rport.out.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/rport.err.log</string>

  </dict>
</plist>
```
Now reload the service definition and check if rport starts.
```
sudo launchctl unload /Library/LaunchDaemons/rport.plist
sudo launchctl load /Library/LaunchDaemons/rport.plist
sudo launchctl start rport
sudo launchctl list|grep "rport$"
```

By default, rport starts at boot.
## On Mac OS (M1/Arm based)
Coming soon.

## On OpenWRT
Coming soon.