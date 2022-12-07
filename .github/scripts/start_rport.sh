#!/usr/bin/env bash
set -e
echo "🕵️‍ Testing if rport is executable and able to connect"
test -e /tmp/rport-data && rm -rf /tmp/rport-data
mkdir /tmp/rport-data
cat<<EOF>rport.conf
[client]
  server = "127.0.0.1:8080"
  fingerprint = "36:98:56:12:f3:dc:e5:8d:ac:96:48:23:b6:f0:42:15"
  auth = "client1:foobaz"
  id = "github"
  name = "gorunner"
  data_dir = "/tmp/rport-data"
[connection]
  keep_alive = '5s'
[logging]
  log_file = "/tmp/rport-data/rport.log"
  log_level = "debug"
[monitoring]
  enabled = false
[file-reception]
  enabled = false
[interpreter-aliases]
  bash = "/usr/bin/bash"
[remote-scripts]
  enabled = false
[remote-commands]
  enabled = false
EOF

echo -n "RPort "
./rport --version
./rport -c rport.conf &
sleep 1
echo -n "RPort pid "
pidof rport
SUCCESS=1
for C in $(seq 1 10);do
  if test -e /tmp/rport-data/rport.log && grep -qi "client: connected" /tmp/rport-data/rport.log;then
    echo "✅ rport client is running and connected"
    pkill rport ||true
    pkill rportd ||true
    SUCCESS=1
    break
  fi
  echo "${C}: Waiting for client to be connected"
  sleep 1
done
if [ $SUCCESS -eq 1 ];then
  true
else
  echo "❌ Client did not connect to server"
  cat /tmp/rport-data/rport.log
  false
fi
