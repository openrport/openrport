#!/usr/bin/env bash

#
# Create a debian package from a file located in ./dist previously created by goreleaser.
# Fpm and go-fpm have proven to be unreliable.
#

set -e

INPUT_ARCH=$1
BIN_FILE=./dist/rport_linux_${INPUT_ARCH}/rport
DST_ARCH=${INPUT_ARCH//amd64_v1/amd64}
DST_ARCH=${DST_ARCH//arm_7/armhf}
DST_ARCH=${DST_ARCH//386/i386}

if [ -e ${BIN_FILE} ]; then
    true
else
    echo "🛑 ERROR: ${BIN_FILE} not found."
    false
fi

if [ -n "${GITHUB_REF_NAME}" ]; then
    VERSION=${GITHUB_REF_NAME}
else
    echo "🛑 ERROR: \$GITHUB_REF_NAME missing"
    false
fi

#
# Create folder structure
#
PKG_NAME=rport
PKG_ROOT=/tmp/deb-build
test -e $PKG_ROOT && sudo rm -rf $PKG_ROOT
mkdir -p ${PKG_ROOT}/etc/rport
mkdir -p ${PKG_ROOT}/lib/systemd/system
mkdir -p ${PKG_ROOT}/var/lib/rport
mkdir -p ${PKG_ROOT}/var/log/rport
mkdir -p ${PKG_ROOT}/usr/bin
mkdir -p ${PKG_ROOT}/usr/share/doc/${PKG_NAME}

#
# Fix file modes
#
find ${PKG_ROOT} -type d -exec chmod 0755 {} \;
find ${PKG_ROOT} -type f -exec chmod 0644 {} \;

#
# Copy files
#
cp ${BIN_FILE} ${PKG_ROOT}/usr/bin/rport
cp ./rport.example.conf ${PKG_ROOT}/etc/rport/rport.conf
cp ./opt/systemd/rport.service ${PKG_ROOT}/lib/systemd/system/rport.service

#
# Prepare for packaging
#
INSTALLED_SIZE=$(du -sb ${PKG_ROOT}/ | awk '{print $1}')
mkdir ${PKG_ROOT}/DEBIAN
chmod 0755 ${PKG_ROOT}/DEBIAN

echo "👷 Building debian package from $BIN_FILE for $DST_ARCH"

#
# Create the package control file
#
cat <<EOF >${PKG_ROOT}/DEBIAN/control
Package: rport
Version: ${VERSION}
Maintainer: RealVNC Limited <info@rport.io>
Depends: libc6, sudo, passwd
Installed-Size: ${INSTALLED_SIZE}
Architecture: ${DST_ARCH}
Section: misc
Priority: optional
Homepage: https://github.com/realvnc-labs/rport/
Description: Remote access and remote management of heterogeneous IT infrastructures
 RPort provides remote access to and script execution on systems
 behind firewalls and NAT
EOF

#
# List of config files
#
cat <<EOF >${PKG_ROOT}/DEBIAN/conffiles
/etc/rport/rport.conf
EOF

#
# Create a changelog, even dummy
#
cat <<EOF | gzip -n --best -c >${PKG_ROOT}/usr/share/doc/${PKG_NAME}/changelog.gz
rport (${VERSION}); urgency=low

  * new version created by GitHub action
  * Full changelog https://github.com/realvnc-labs/rport/releases/tag/${GITHUB_REF_NAME}
EOF
chmod 0644 ${PKG_ROOT}/usr/share/doc/${PKG_NAME}/changelog.gz

cat <<EOF >${PKG_ROOT}/usr/share/doc/${PKG_NAME}/copyright
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Source: https://github.com/realvnc-labs/rport/releases/tag/${GITHUB_REF_NAME}
Copyright: $(date +%Y)
License: MIT

Files: *
Copyright: $(date +%Y)
License:  MIT
EOF
chmod 0644 ${PKG_ROOT}/usr/share/doc/${PKG_NAME}/copyright

#
# Create a postinst script
#
cat <<EOF >${PKG_ROOT}/DEBIAN/postinst
#!/bin/sh
#
# Post install script for rport (client)
#
set -e
# Create a user if not exists
RPORT_USER=rport
if id "\${RPORT_USER}" >/dev/null 2>&1; then
  true
else
  useradd -r -d /var/lib/rport -M -s /bin/false -U -c "System user for rport" "\$RPORT_USER"
fi

chown "\$RPORT_USER" /var/lib/rport
chown "\$RPORT_USER" /var/log/rport
systemctl daemon-reload
deb-systemd-invoke enable rport.service
EOF
chmod 0555 ${PKG_ROOT}/DEBIAN/postinst

#
# Create a prerm script
#
cat <<EOF >${PKG_ROOT}/DEBIAN/prerm
#!/bin/sh
#
# pre remove script for rport (client)
#
set -e
deb-systemd-invoke stop rport.service || true
# Looking for user removal? We don't do that to not leave files without owners behind.
EOF
chmod 0555 ${PKG_ROOT}/DEBIAN/prerm

#
# Fix file owner
#
sudo chown -R root:root ${PKG_ROOT}

#
# Build the debian package
#
PKG_FILE=./${PKG_NAME}_${VERSION}_${DST_ARCH}.deb
dpkg-deb -Zgzip -v --build ${PKG_ROOT}
mv ${PKG_ROOT}.deb ${PKG_FILE}
echo "✅ Created $PKG_FILE"

## Check the content of the package
dpkg-deb -c ${PKG_FILE}

# Validate deb
echo "👷 Verifying debian package with lintian ..."
lintian --no-tag-display-limit -X binaries ${PKG_FILE}
echo "✅ Check passed"

if [ $(uname -m) == "x86_64" ] && [ ${DST_ARCH} == "amd64" ]; then
    echo "👷 Checking if deb installs and uninstalls ..."
    cp rport_${VERSION}_amd64.deb /tmp
    sudo dpkg -i /tmp/rport_${VERSION}_amd64.deb
    dpkg -l|grep rport
    sudo dpkg -r rport
    echo "✅ Check passed"
fi

echo "======================================================================================================"
echo ""
echo "  Successfully created debian package ${PKG_FILE}"
echo ""
echo "======================================================================================================"
