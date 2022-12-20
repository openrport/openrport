#!/usr/bin/env bash
#
# Create and publish Debian deb and RedHat rpm packages for rport and rportd
#
set -e
# Check the version is set.
if [ -z "$GITHUB_REF_NAME" ]; then
  echo "Fatal error. Variable GITHUB_REF_NAME not set. This variable becomes the package version."
  false
fi

export DEBIAN_FRONTEND=noninteractive
# Install fpm and its dependencies, if missing
which gem || sudo -En apt-get install -y ruby
which fpm || sudo -En gem install fpm
which rpmbuild || sudo -En apt-get install -y rpm

# Install lintian, if missing
which lintian || sudo -En apt-get install -y lintian

# Install chglog, if missing
test -e ~/go/bin/chglog || go install github.com/goreleaser/chglog/cmd/chglog@latest

# Create the changelog file
echo "👷‍ Creating a changelog ..."
chglog init --owner "RealVNC Limited <support@rport.io>"
chglog format --template rpm >changelog
chglog format --template deb | gzip -n --best -c >changelog.gz

# Creating empty dir
test -e /tmp/empty/rport && rm -rf /tmp/empty/rport
mkdir -p /tmp/empty/rport

# We must set the description here because the fpm conf file does not support linebreaks,
# but debian packages expect the description to have at least one linebreak.
DESCRIPTION="Remote access and remote management of heterogeneous IT environments
RPort provides remote access to and script execution on systems
behind firewalls and NAT"

# Clean up (not needed on GitHub Actions but handy on local runs)
rm *.deb 2>/dev/null || true
rm *.rpm 2>/dev/null || true

# Loop over specified folders in the dist folder to create packages
SOURCES="amd64_v1 arm64 386 arm_7" # Refers to dist/rport_linux_${SOURCE}
for SRC in $SOURCES; do
  #
  # Create deb packages
  #
  DST_ARCH_DEB=${SRC//amd64_v1/amd64}
  DST_ARCH_DEB=${DST_ARCH_DEB//arm_7/armhf}
  DST_ARCH_DEB=${DST_ARCH_DEB//386/i386}
  echo "👷‍ Creating deb package for $DST_ARCH_DEB now..."
  fpm --fpm-options-file ./opt/fpm/fpm.conf \
    -t deb \
    --description "$DESCRIPTION" \
    -p rport_${GITHUB_REF_NAME}_${DST_ARCH_DEB}.deb \
    --version ${GITHUB_REF_NAME} \
    --architecture $DST_ARCH_DEB \
    --depends libc6 --depends adduser \
    --deb-changelog ./changelog.gz \
    dist/rport_linux_${SRC}/rport=/usr/bin/rport \
    ./opt/systemd/rport.service=/lib/systemd/system/rport.service \
    ./rport.example.conf=/etc/rport/rport.conf \
    ./opt/fpm/copyright=/usr/share/doc/rport/copyright \
    /tmp/empty/rport=/var/log \
    /tmp/empty/rport=/var/lib

  #
  # Create the rpm packages
  [ "$SRC" == "arm_7" ] && continue # rpm distros don't support 32bit arm
  DST_ARCH_RPM=${SRC//amd64_v1/x86_64}
  DST_ARCH_RPM=${DST_ARCH_RPM//386/i686}
  DST_ARCH_RPM=${DST_ARCH_RPM//arm64/aarch64}
  echo "👷‍ Creating rpm package for $DST_ARCH_RPM now..."
  fpm --fpm-options-file ./opt/fpm/fpm.conf \
    -t rpm \
    --description "$DESCRIPTION" \
    -p rport-${GITHUB_REF_NAME}-1.${DST_ARCH_RPM}.rpm \
    --version ${GITHUB_REF_NAME} \
    --architecture $DST_ARCH_RPM \
    --rpm-changelog ./changelog \
    dist/rport_linux_${SRC}/rport=/usr/bin/rport \
    ./opt/systemd/rport.service=/lib/systemd/system/rport.service \
    ./rport.example.conf=/etc/rport/rport.conf \
    /tmp/empty/rport=/var/log \
    /tmp/empty/rport=/var/lib
done
echo "✅ Created $(ls -l *.deb|wc -l) deb packages."
echo "✅ Created $(ls -l *.rpm|wc -l) rpm packages."
# Clean up (not needed on GitHub Actions but handy on local runs)
rm -f changelog.*

# Validate deb
echo "👷Verifying debian package with lintian ..."
lintian --no-tag-display-limit -X binaries rport_*_amd64.deb
echo "👷 Checking if deb installs and uninstalls ..."
sudo dpkg -i rport_*_amd64.deb
sudo dpkg -r rport
echo "✅ Check passed"

# Validate rpms with rpmlint
echo "👷 Verifying rpm package with rpmlint ..."
docker run --name rpmlint --mount type=bind,source="$(pwd)",target=/src rockylinux:9 \
  bash -c "dnf -y install rpmlint >/dev/null 2>&1 && rpmlint -f /src/opt/fpm/rpmlint.conf /src/rport-*.x86_64.rpm"
docker rm rpmlint
# Check the package installs and uninstalls without errors.
# Inside docker it's a weak check but it will still catch most obvious errors
echo "👷 Checking if rpm installs and uninstalls ..."
docker run --name rpminstall --mount type=bind,source="$(pwd)",target=/src rockylinux:9 \
  bash -c "rpm -i /src/rport-*.x86_64.rpm && rpm -e rport"
docker rm rpminstall
echo "✅ Check passed"
ls -la