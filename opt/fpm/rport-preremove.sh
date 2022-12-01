#!/bin/sh
#
# pre remove script for rport (client)
#
set -e
if systemctl status >/dev/null 2>&1; then
  true
else
  # Systemd is not running. Nothing to do.
  exit 0
fi
if deb-systemd-invoke --help >/dev/null 2>&1; then
  deb-systemd-invoke stop rport.service || true
elif
  systemctl --version >/dev/null 2>&1
then
  systemctl stop rport.service || true
fi

# Looking for user removal? We don't do that to not leave files without owners behind.
