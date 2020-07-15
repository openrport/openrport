#!/usr/bin/env bash

set -xe

if [ -z ${RELEASE_MODE} ]; then
  echo "RELEASE_MODE env variable is empty"
  exit 1
fi

PROJECT_NAME=github.com/cloudradar-monitoring/rport
PROJECT_DIR=/go/src/${PROJECT_NAME}

# fetch release changelog so we can preserve it when releasing
CHANGELOGRAW=$(curl -H "Authorization: token ${GITHUB_TOKEN}" https://api.github.com/repos/cloudradar-monitoring/rport/releases | jq ".[0].body")

# update release status
PRERELEASE="--pre-release"
if [ ${RELEASE_MODE} = "stable" ]; then
  PRERELEASE=
fi
echo -e ${CHANGELOGRAW} | sed -e 's/^"//' -e 's/"$//' | github-release edit --user cloudradar-monitoring --repo rport --tag ${CIRCLE_TAG} ${PRERELEASE} --description -
