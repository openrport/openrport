#!/usr/bin/env bash
set -e
test -e .pkg-env && . .pkg-env # Source local test env

pwd
ls -la

pkill ssh-agent||true
eval $(ssh-agent)
ssh-add - <<<"$REPO_SSH_PRIV_KEY"
test -e ~.ssh || mkdir -p ~/.ssh
grep -q "$REPO_SSH_HOST" ~/.ssh/known_hosts || echo "$REPO_SSH_KNOWN_HOSTS">>~/.ssh/known_hosts
echo "👷 Copying to repo server ... "
scp -P ${REPO_SSH_PORT} *.deb "${REPO_SSH_USER}"@"${REPO_SSH_HOST}":~/incoming/
scp -P ${REPO_SSH_PORT} *.rpm "${REPO_SSH_USER}"@"${REPO_SSH_HOST}":~/incoming/
echo "✅ All files copied"
echo "👷 Triggering package publishing ... "
ssh -p "${REPO_SSH_PORT}" "${REPO_SSH_USER}"@"${REPO_SSH_HOST}" "~/bin/update-repos.sh"
echo "✅ All packages published"
