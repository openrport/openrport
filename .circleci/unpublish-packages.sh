#!/usr/bin/env bash

set -xe

github-release delete --user cloudradar-monitoring --repo rport --tag ${CIRCLE_TAG}
