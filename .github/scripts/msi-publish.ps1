Write-Output "[*] Compressing rport.exe to zip"
$zip = "rport-$($env:GITHUB_REF_NAME)_x86_64.zip"
Compress-Archive rport.exe -DestinationPath $zip
Get-ChildItem *.zip

Get-ChildItem -File *.exe

Write-Output "[*] Uploading $($zip)"
& curl.exe -v -fs https://$env:DOWNLOAD_SERVER/exec/upload.php `
 -H "Authentication: $env:MSI_UPLOAD_TOKEN" `
 -F file=@$zip -F dest_dir="rport/unstable/msi"

Write-Output "[*] Uploading MSI to download server"
$upload = "rport_$($env:GITHUB_REF_NAME)_windows_x86_64.msi"
Copy-Item rport-client.msi $upload
Get-ChildItem -File *.msi
& curl.exe -V
& curl.exe -fs https://$env:DOWNLOAD_SERVER/exec/upload.php `
 -H "Authentication: $env:MSI_UPLOAD_TOKEN" `
 -F file=@$upload -F dest_dir="rport/unstable/msi"

# Publish the MSI to the release
# The release might not be ready yet because goreleaser runs on a separate action that ususally takes longer
Import-Module -Name .\.github\scripts\gh-release-id.psm1 -Force
$releaseId = Get-ReleaseId -tag $env:GITHUB_REF_NAME -waitSec 1200
Write-Output "[*] Will upload $($upload) to the existing release $($releaseId)"
& curl.exe -v -s --fail -T $($upload) `
-H "Authorization: token $($env:GITHUB_TOKEN)" `
-H "Content-Type: application/x-msi" `
-H "Accept: application/vnd.github.v3+json" `
"https://uploads.github.com/repos/openrport/openrport/releases/$($releaseId)/assets?name=$($upload)"