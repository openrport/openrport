#!/usr/bin/env bash

#
# Create an rpm package from a file located in ./dist previously created by goreleaser.
# Fpm and go-fpm have proven to be unreliable.
#

set -e

INPUT_ARCH=$1
BIN_FILE=$(pwd)/dist/rport_linux_${INPUT_ARCH}/rport
DST_ARCH=${INPUT_ARCH//amd64_v1/x86_64}
DST_ARCH=${DST_ARCH//386/i686}
DST_ARCH=${DST_ARCH//arm64/aarch64}

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
# Install dependencies
#
[ -e /usr/bin/sudo ]||dnf -y install sudo
[ -e /usr/bin/which ]||sudo dnf -y install which
which rpmdev-setuptree || sudo dnf install -y rpmdevtools
which rpmlint || sudo dnf install -y rpmlint

#
# Create folder structure
#
test -e ~/rpmbuild && rm -rf ~/rpmbuild
rpmdev-setuptree

#
# Add files
#
#tar czf ~/rpmbuild/SOURCES/rport-${VERSION}.tar.gz --transform "s,dist/rport.*/,rport-${VERSION}/," ${BIN_FILE}
#mkdir ~/rpmbuild/BUILD/rport-${VERSION}
#cp ${BIN_FILE} ~/rpmbuild/BUILD/rport

#
# Create the specs file
#
cat << EOF > ~/rpmbuild/SPECS/rport.spec
%define _rpmfilename %%{NAME}-%%{VERSION}-1.%%{ARCH}.rpm
Name:           rport
Version:        ${VERSION}
Release:        1
Summary:        Remote Management Client for the RPort-Server

License:        MIT
URL:            https://github.com/realvnc-labs/rport
Source0:        https://github.com/realvnc-labs/rport/releases/tag/${GITHUB_REF_NAME}

Requires:       passwd

%description
Remote access and remote management of heterogeneous IT infrastructures
RPort provides remote access to and script execution on systems
behind firewalls and NAT

%prep

%build

%install
echo "BUILDROOT = \$RPM_BUILD_ROOT"
mkdir -p \$RPM_BUILD_ROOT/usr/bin
mkdir -p \$RPM_BUILD_ROOT/etc/rport
mkdir -p \$RPM_BUILD_ROOT/var/lib/rport
mkdir -p \$RPM_BUILD_ROOT/var/log/rport
mkdir -p \$RPM_BUILD_ROOT/lib/systemd/system
cp ${BIN_FILE} \$RPM_BUILD_ROOT/usr/bin/
cp $(pwd)/rport.example.conf \$RPM_BUILD_ROOT/etc/rport/rport.conf
cp $(pwd)/opt/systemd/rport.service \$RPM_BUILD_ROOT/lib/systemd/system/rport.service

%clean
rm -rf \$RPM_BUILD_ROOT

%files
%{_bindir}/%{name}
/lib/systemd/system/rport.service

%dir %attr(755,rport,root) /var/log/rport
%dir %attr(755,rport,root) /var/lib/rport

%config(noreplace) /etc/rport/rport.conf

%pre
# Create a use if not exists
RPORT_USER=rport
if id "\${RPORT_USER}" >/dev/null 2>&1; then
  true
else
  useradd -r -d /var/lib/rport -M -s /bin/false -U -c "System user for rport" "\$RPORT_USER"
fi

%post
systemctl daemon-reload
systemctl enable rport.service

%preun
systemctl stop rport.service || true

%changelog
* $(date) RealVNC Ltd. <developers@rport.io> - ${VERSION}-1
- RPM created by GitHub Action
- Full changelog https://github.com/realvnc-labs/rport/releases/tag/${GITHUB_REF_NAME}
EOF
rpmlint ~/rpmbuild/SPECS/rport.spec
echo "👷 Building RPM package now ..."
rpmbuild -bb --target ${DST_ARCH} ~/rpmbuild/SPECS/rport.spec
echo "👷 Verifying RPM package with rpmlint ..."
cat << EOF > /tmp/rpmlint.conf
# Compiled go binaries are not recognized as binaries. We must ignore this error.
addFilter("E: statically-linked-binary")
EOF
PKG_FILE=$(find ~/rpmbuild/RPMS/ -type f)
rpmlint -f /tmp/rpmlint.conf ${PKG_FILE}
echo "✅ Check passed"

if [ $(uname -m) == "x86_64" ] && [ ${DST_ARCH} == "x86_64" ]; then
    echo "👷 Checking if rpm installs and uninstalls ..."
    # Fake a systemctl binary inside docker containers
    [ -e /usr/bin/systemctl ]|| ln -s /usr/bin/true /usr/bin/systemctl
    cp ${PKG_FILE} /tmp
    (cd /tmp; sudo rpm -i -i ${PKG_FILE})
    rpm -qa|grep rport
    ls -ld /var/log/rport
    ls -ld /var/lib/rport
    (cd /tmp;sudo rpm -e rport)
    echo "✅ Check passed"
fi
mv ${PKG_FILE} .
echo "======================================================================================================"
echo ""
echo "  Successfully created RPM package $(basename ${PKG_FILE})"
echo ""
echo "======================================================================================================"
