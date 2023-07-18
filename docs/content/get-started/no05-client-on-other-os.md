---
title: "Install and run an rport client"
weight: 5
slug: install-and-run-rport-client
aliases:
- /docs/no05-client-on-other-os.html
---
{{< toc >}}

## On macOS (any)

### Download and create config

Open the terminal and as an unprivileged user download the binary and put it in `/usr/local/bin`

```shell
curl -L "https://downloads.rport.io/rport/stable/latest.php?filter=darwin_$(uname -m)" -o rport-mac.tar.gz
test -e /usr/local/bin/||sudo mkdir /usr/local/bin
sudo tar xzf rport-mac.tar.gz -C /usr/local/bin/ rport
rport --version
sudo mkdir /etc/rport
sudo tar xzf rport-mac.tar.gz -C /etc/rport rport.example.conf
sudo mv /etc/rport/rport.example.conf /etc/rport/rport.conf
sudo mkdir /private/var/log/rport
sudo chown ${USER} /private/var/log/rport/
sudo mkdir /private/var/lib/rport
sudo chown ${USER} /private/var/lib/rport
```

Now open the configuration file with an editor and enter your server URL, credentials, and fingerprint.

```shell
sudo vim /etc/rport/rport.conf
```

Or automate the above step using `sed`.

```shell
FINGERPRINT="2b:c8:79:09:40:ba:7c:60:05:e5:2c:93:6d:75:56:10"
CONNECT_URL="http://rport.example.com:8080"
CLIENT_ID="000002-techdev"
PASSWORD="thahs4chiwio3WieZe"
CONFIG_FILE=/etc/rport/rport.conf
sudo sed -i '' "s|#*server = .*|server = \"${CONNECT_URL}\"|g" "$CONFIG_FILE"
sudo sed -i '' "s/#*auth = .*/auth = \"${CLIENT_ID}:${PASSWORD}\"/g" "$CONFIG_FILE"
sudo sed -i '' "s/#*fingerprint = .*/fingerprint = \"${FINGERPRINT}\"/g" "$CONFIG_FILE"
sudo sed -i '' "s/#*log_file = .*C.*Program Files.*/""/g" "$CONFIG_FILE"
sudo sed -i '' "s/#*log_file = /log_file = /g" "$CONFIG_FILE"
```

Before registering a service, test it with `rport -c /etc/rport/rport.conf`. You should not get any output and the new
client should appear on the server.

For registering the service you have three options.

1. You run it with your own user.
2. You create a so-called system user on macOS, see below.
3. You run it from the built-in `deamon` user account.

The third option is not recommended if you want to run scripts with sudo privileges because you would give to many
rights to the `daemon` user. It's better to create a dedicated user `rport`.

### Create an rport system users

```bash
cat<<"EOF"|sudo bash
set -e
username="rport"
if id ${username} >/dev/null 2>&1;then
  echo "User ${username} exists"
  false
fi
realname="RPort"
echo "Adding system user $username with real name ${realname}"

for (( uid = 500;; --uid )) ; do
echo "."
if ! id -u $uid &>/dev/null; then
if ! dscl /Local/Default -ls Groups gid | grep -q [^0-9]$uid\$ ; then
dscl /Local/Default -create Groups/_$username
dscl /Local/Default -create Groups/_$username Password \*
dscl /Local/Default -create Groups/_$username PrimaryGroupID $uid
dscl /Local/Default -create Groups/_$username RealName "$realname"
dscl /Local/Default -create Groups/_$username RecordName _$username $username
dscl /Local/Default -create Users/_$username

dscl /Local/Default -create Users/_$username NFSHomeDirectory /private/var/lib/rport
dscl /Local/Default -create Users/_$username Password \*
dscl /Local/Default -create Users/_$username PrimaryGroupID $uid
dscl /Local/Default -create Users/_$username RealName "$realname"
dscl /Local/Default -create Users/_$username RecordName _$username $username
dscl /Local/Default -create Users/_$username UniqueID $uid
dscl /Local/Default -create Users/_$username UserShell /bin/sh
dscl /Local/Default -delete /Users/_$username AuthenticationAuthority
dscl /Local/Default -delete /Users/_$username PasswordPolicyOptions
break
fi
fi
done
chown rport /private/var/log/rport/
EOF
```

### Register and run the service

```shell
sudo rport --service install --service-user <USERNAME> -c /etc/rport/rport.conf
sudo rport --service start
```

If you are in doubt the service has been created run `sudo launchctl list|grep rport`. It should list the pid on the
first column to indicate rport is running.

```shell
$ sudo launchctl list|grep "rport$"
9942 0 rport
```

If you get an output like this, the installation of the service has succeeded but rport cannot start.

```shell
$ sudo launchctl list|grep "rport$"
- 0 rport
```

Missing write permissions to the log folder is most likely the reason.
Open `/Library/LaunchDaemons/rport.plist` with an editor and use `tmp` as log directory for the start-up logs.

```xml
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

```shell
sudo launchctl unload /Library/LaunchDaemons/rport.plist
sudo launchctl load /Library/LaunchDaemons/rport.plist
sudo launchctl start rport
sudo launchctl list|grep "rport$"
```

By default, rport starts at boot.
