Write-Output "Creates a specific version of rport-client.msi"
Write-Output "----------------------------------------------"
$ErrorActionPreference = 'Stop'

$major =  $args[0]
$minor = $args[1]
$patch = $args[2]

Write-Output "--------------------------------------------------"
Write-Output "  - build rport msi ver. $major.$minor.$patch"

# Read the template
$versionInfo = (Get-Content -Raw opt/resource/versioninfo.json | ConvertFrom-Json)
$versionInfo.FixedFileInfo.FileVersion.Major = $major
$versionInfo.FixedFileInfo.FileVersion.Minor = $minor
$versionInfo.FixedFileInfo.FileVersion.Patch = $patch
$versionInfo.FixedFileInfo.ProductVersion.Major = $major
$versionInfo.FixedFileInfo.ProductVersion.Minor = $minor
$versionInfo.FixedFileInfo.ProductVersion.Patch = $patch
$versionInfo.StringFileInfo.FileVersion = "$major.$minor.$patch"
$versionInfo.StringFileInfo.ProductVersion = "$major.$minor.$patch"
# Write the file used for the build process
$versionInfo|ConvertTo-Json|Out-File -Path cmd/rport/versioninfo.json
# Convert the versioninfo.json to resource.syso
Set-Location ./cmd/rport
goversioninfo.exe
Set-Location ../../

Write-Output "[*] Building rport.exe for windows"
go build -ldflags "-s -w -X github.com/realvnc-labs/rport/share.BuildVersion=$major.$minor.$patch" -o rport.exe ./cmd/rport

Write-Output "[*] Creating wixobj's"
& 'C:\Program Files (x86)\WiX Toolset v3.11\bin\candle.exe' -dPlatform=x64 -ext WixUtilExtension opt/resource/*.wxs

Write-Output "[*] Creating MSI"
& 'C:\Program Files (x86)\WiX Toolset v3.11\bin\light.exe' `
  -loc opt/resource/Product_en-us.wxl `
  -ext WixUtilExtension -ext WixUIExtension -sval `
  -out rport-client-ver"$major.$minor.$patch".msi LicenseAgreementDlg_HK.wixobj WixUI_HK.wixobj Product.wixobj