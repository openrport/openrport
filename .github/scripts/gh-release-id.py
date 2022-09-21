#!/usr/bin/env python3

import os
import sys

import requests

try:
    github_token = os.environ["GITHUB_TOKEN"]
except:
    sys.stderr.write("⛔ Environment variable GITHUB_TOKEN not set\n")
    sys.exit(1)

try:
    tag = sys.argv[1]
except IndexError:
    sys.stderr.write("⛔ No tag given as first argument\n")
    sys.exit(1)

headers = {"Authorization": "Bearer " + github_token}
r = requests.get(
    "https://api.github.com/repos/cloudradar-monitoring/rport/releases?page=1&per_page=5",
    headers=headers,
)
if r.status_code != 200:
    sys.stderr.write("⛔ Got status %d: %s" % (r.status_code, r.content))
    sys.exit(1)

for release in r.json():
    if tag == release['tag_name']:
        print(release['id'])
        sys.exit(0)

sys.stderr.write("⛔ No release_id found for tag %s\n" % tag)
sys.exit(1)
