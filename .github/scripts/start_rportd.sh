#!/usr/bin/env bash
set -e
echo "🕵️‍ Testing if rportd is executable and accepting connections"

test -e /tmp/rportd-data && rm -rf /tmp/rportd-data
mkdir /tmp/rportd-data
cat<<EOF>rportd.conf
[server]
  address = "127.0.0.1:8080"
  key_seed = "5448e69530b4b97fb510f96ff1550500b093"
  #-> Fingerprint: 36:98:56:12:f3:dc:e5:8d:ac:96:48:23:b6:f0:42:15
  auth = "client1:foobaz"
  data_dir = "/tmp/rportd-data"
  used_ports = ['2000-2009']
[logging]
  log_file = "/tmp/rportd-data/rportd.log"
  log_level = "debug"
EOF

echo "✅️ Created config file"
cat rportd.conf

echo -n "RPortd "
./rportd --version
./rportd -c rportd.conf &
sleep 3
echo -n "RPortd pid "
pidof rportd
echo "✅ rportd is running"