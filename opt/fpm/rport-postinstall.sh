#!/bin/sh
#
# Post install script for rport (client)
#
set -e
# Create a use if not exists
RPORT_USER=rport
if id "${RPORT_USER}" >/dev/null 2>&1; then
  true
else
  if useradd --help >/dev/null 2>&1; then
    useradd -r -d /var/lib/rport -M -s /bin/false -U -c "System user for rport" "$RPORT_USER"
  elif adduser --help >/dev/null 2>&1; then
    addgroup rport
    adduser -h /var/lib/rport -s /bin/false -G rport -S -D "$RPORT_USER"
  else
    echo "No command found to add a user"
    false
  fi
fi

chown "$RPORT_USER" /var/lib/rport
chown "$RPORT_USER" /var/log/rport
set -e
if deb-systemd-invoke --help >/dev/null 2>&1; then
  deb-systemd-invoke enable rport.service
elif
  systemctl --version >/dev/null 2>&1
then
  systemctl enable rport.service
fi
