set -e
pwd
ls -la
#
# Create a tar package
#
if [ -e  ~/.plus.env ];then
  .  ~/.plus.env
else
  echo "⛔ ~/.plus.env not found. Exit"
  false
fi
cd rport-plus
export PLUS_ARTIFACT=rport-plus_${PLUS_LATEST}@${GITHUB_REF_NAME}_Linux_$(uname -m).tar.gz
echo "🚚Will create ${PLUS_ARTIFACT} release file"≤
echo "rport-plus v${PLUS_LATEST}; compiled for rportd ${GITHUB_REF_NAME}; built on $(date)" > version.txt
tar czf ${PLUS_ARTIFACT} README.md license.txt version.txt rport-plus.so
tar tzf ${PLUS_ARTIFACT}
ls -la ${PLUS_ARTIFACT}
echo "✅ Successfully created artifact ${PLUS_ARTIFACT}"
#
# Get the release id of the rport (main repo) tag
#
dpkg -l|grep -q python3-requests || apt-get -y install python3-requests
RELEASE_ID=$(../.github/scripts/gh-release-id.py ${GITHUB_REF_NAME})
echo "🚩 RELEASE_ID=$RELEASE_ID"
#
# Upload a file to the existing release assets
#
echo "🚚 Will upload a new asset to the existing release"
curl -v -s --fail -T ${PLUS_ARTIFACT} \
 -H "Authorization: token ${GITHUB_TOKEN}" -H "Content-Type: $(file -b --mime-type ${PLUS_ARTIFACT})" \
 -H "Accept: application/vnd.github.v3+json" \
 "https://uploads.github.com/repos/cloudradar-monitoring/rport/releases/${RELEASE_ID}/assets?name=$(basename ${PLUS_ARTIFACT})"|tee upload.log|jq
