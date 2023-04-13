param (
    [int]$major = (&{If($env:GITHUB_REF_NAME) {  [int]($env:GITHUB_REF_NAME.Split(".")[0]) } Else { $(throw "-major is required.") }}),
    [int]$minor = (&{If($env:GITHUB_REF_NAME) {  [int]($env:GITHUB_REF_NAME.Split(".")[1]) } Else { $(throw "-minor is required.") }}),
    [int]$patch = (&{If($env:GITHUB_REF_NAME) {  [int]($env:GITHUB_REF_NAME.Split(".")[2]) } Else { $(throw "-patch is required.") }}),
    [switch]$SignMsi = $true
)
Write-Output "Making the MSI... ver $major.$minor.$patch"
Write-Output "--------------------------------------"
$ErrorActionPreference = 'Stop'
Get-ChildItem env:
$PSVersionTable

Write-Output "[*] Installing goversioninfo"
go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
$ENV:PATH="$ENV:PATH;$($env:home)\go\bin"
# See ./cmd/rpport/versioninfo.json.md

Write-Output "[*] Installing WIX"
choco install wixtoolset

Write-Output "[*] Versioning the build"
# Read the template
$versionInfo = (Get-Content -Raw opt/resource/versioninfo.json | ConvertFrom-Json)
# Set values
$versionInfo.FixedFileInfo.FileVersion.Major = $major
$versionInfo.FixedFileInfo.FileVersion.Minor = $minor
$versionInfo.FixedFileInfo.FileVersion.Patch = $patch
$versionInfo.FixedFileInfo.ProductVersion.Major = $major
$versionInfo.FixedFileInfo.ProductVersion.Minor = $minor
$versionInfo.FixedFileInfo.ProductVersion.Patch = $patch
$versionInfo.StringFileInfo.FileVersion = $major.$minor.$patch
$versionInfo.StringFileInfo.ProductVersion = $major.$minor.$patch
Write-Output $versionInfo|convertTo-Json
# Write the file used for the build process
$versionInfo|ConvertTo-Json|Out-File -Path cmd/rport/versioninfo.json
# Convert the versioninfo.json to resource.syso
Set-Location ./cmd/rport
goversioninfo.exe
Set-Location ../../

Write-Output "[*] Building rport.exe for windows"
go build -ldflags "-s -w -X github.com/realvnc-labs/rport/share.BuildVersion=$($env:GITHUB_REF_NAME)" -o rport.exe ./cmd/rport
Get-ChildItem -File *.exe
.\rport.exe --version

Write-Output "[*] Creating wixobj's"
& 'C:\Program Files (x86)\WiX Toolset v3.11\bin\candle.exe' -dPlatform=x64 -ext WixUtilExtension opt/resource/*.wxs
Start-Sleep 2

Write-Output "[*] Creating MSI"
& 'C:\Program Files (x86)\WiX Toolset v3.11\bin\light.exe' `
  -loc opt/resource/Product_en-us.wxl `
  -ext WixUtilExtension -ext WixUIExtension -sval `
  -out rport-client.msi LicenseAgreementDlg_HK.wixobj WixUI_HK.wixobj Product.wixobj

if ($SignMsi)
{
    Write-Output "Signing the MSI..."
    Write-Output "------------------"

    Start-Sleep 2
    Get-ChildItem -File *.msi

    Write-Output "[*] Creating a self signed certificate"
    $cert = New-SelfSignedCertificate -DnsName rport.io -CertStoreLocation cert:\LocalMachine\My -type CodeSigning
    $MyPassword = ConvertTo-SecureString -String "MyPassword" -Force -AsPlainText
    Export-PfxCertificate -cert $cert -FilePath mycert.pfx -Password $MyPassword

    Write-Output "[*] Signing the generated MSI"
    & 'C:\Program Files (x86)\Windows Kits\10\bin\10.0.22621.0\x86\signtool.exe' sign /fd SHA256 /f mycert.pfx /p MyPassword rport-client.msi
    Start-Sleep 2

    Write-Output "[*] Displaying MSI summary"
    Install-Module MSI -Force
    Get-MSISummaryInfo rport-client.msi
    Get-AuthenticodeSignature rport-client.msi|Format-List
}