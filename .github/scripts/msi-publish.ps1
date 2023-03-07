Write-Output "[*] Compressing rport.exe to zip"
$zip = "rport-$($env:GITHUB_REF_NAME)_x86_64.zip"
Compress-Archive rport.exe -DestinationPath $zip
Get-ChildItem *.zip

Write-Output "[*] Uploading $($zip)"
& curl.exe -fs https://$env:DOWNLOAD_SERVER/exec/upload.php `
 -H "Authentication: $env:MSI_UPLOAD_TOKEN" `
 -F file=@$zip -F dest_dir="rport/unstable/msi"

Write-Output "[*] Uploading MSI to download server"
$upload = "rport-$($env:GITHUB_REF_NAME)_x86_64.msi"
Copy-Item rport-client.msi $upload
Get-ChildItem -File *.msi
& curl.exe -V
& curl.exe -fs https://$env:DOWNLOAD_SERVER/exec/upload.php `
 -H "Authentication: $env:MSI_UPLOAD_TOKEN" `
 -F file=@$upload -F dest_dir="rport/unstable/msi"