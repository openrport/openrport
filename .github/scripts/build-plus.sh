#
# Create the private key that gives access to the private rport-plus repo
#
test -e ~/.ssh || mkdir ~/.ssh
echo "${RPORT_PLUS_PRIV_KEY}" > ~/.ssh/rport-plus-key
chmod 0400 ~/.ssh/*
#
# Checkout the repo
#
export GIT_SSH_COMMAND="ssh -i ~/.ssh/rport-plus-key"
git clone git@github.com:cloudradar-monitoring/rport-plus.git
cd rport-plus
pwd
git status
#
# Checkout the latest tag
#
PLUS_LATEST=$(git ls-remote --tags origin|tail -n1|awk '{print $2}'|cut -d'/' -f3)
echo "Will checkout rport-plus branch ${PLUS_LATEST}"
git checkout tags/"${PLUS_LATEST}" -b v"${PLUS_LATEST}"
echo "✅ Successfully checked out rport-plus${PLUS_LATEST}"
echo ::set-output name=repo_name::$(basename `git rev-parse --show-toplevel`)
#
# Build the plugin
#
make build
ls -la rport-plus.so
echo "=================================================================="
echo "✅ Successfully built rport-plus.so version ${PLUS_LATEST}"
echo "=================================================================="
echo PLUS_LATEST="$PLUS_LATEST" > ~/.plus.env
