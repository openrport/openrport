param (
    [int]$major = (&{If($env:GITHUB_REF_NAME) {  [int]($env:GITHUB_REF_NAME.Split(".")[0]) } Else { $(throw "-major is required.") }}),
    [int]$minor = (&{If($env:GITHUB_REF_NAME) {  [int]($env:GITHUB_REF_NAME.Split(".")[1]) } Else { $(throw "-minor is required.") }}),
    [int]$patch = (&{If($env:GITHUB_REF_NAME) {  [int]($env:GITHUB_REF_NAME.Split(".")[2]) } Else { $(throw "-patch is required.") }}),
    [switch]$SignMsi = $false,
    [string]$msiFileName = "rport-client.msi"
)
Write-Output "Making $msiFileName ver $major.$minor.$patch"
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
go build -ldflags "-s -w -X github.com/openrport/openrport/share.BuildVersion=$($env:GITHUB_REF_NAME)" -o rport.exe ./cmd/rport
Get-ChildItem -File *.exe
.\rport.exe --version

Write-Output "[*] Creating wixobj's"
& 'C:\Program Files (x86)\WiX Toolset v3.11\bin\candle.exe' -dPlatform=x64 -ext WixUtilExtension opt/resource/*.wxs
Start-Sleep 2

Write-Output "[*] Creating MSI"
& 'C:\Program Files (x86)\WiX Toolset v3.11\bin\light.exe' `
  -loc opt/resource/Product_en-us.wxl `
  -ext WixUtilExtension -ext WixUIExtension -sval `
  -out $msiFileName LicenseAgreementDlg_HK.wixobj WixUI_HK.wixobj Product.wixobj

if ($SignMsi)
{
    Write-Output "Signing the MSI..."
    Write-Output "------------------"

    Start-Sleep 2
    Get-ChildItem -File *.msi

    Write-Output "[*] Decode the base64 encoded pfx from the env variable and store it into a file"
    if (!$env:CS_PFX)
    {
        Write-Error "CS_PFX env is null or empty, signing cannot continue"
    }
    $binary = [Convert]::FromBase64String($env:CS_PFX)
    # "-Encoding Byte" is replaced by "-AsByteStream" 
    Set-Content -Path cs.pfx -Value $binary -AsByteStream
    Get-ChildItem -File *.pfx

    Write-Output "[*] Validate the certificate has been decoded correctly"
    $PfxData=Get-PfxData -FilePath ".\cs.pfx"
    $SigningCert=$PfxData.EndEntityCertificates[0]
    Write-Output "    $SigningCert"

    Write-Output "[*] Signing the generated MSI"
    & 'C:\Program Files (x86)\Windows Kits\10\bin\10.0.22621.0\x86\signtool.exe' sign /f cs.pfx /tr http://timestamp.digicert.com /td SHA256 /fd SHA256 /a $msiFileName

    Write-Output "[*] Check signature of the generated MSI"
    & 'C:\Program Files (x86)\Windows Kits\10\bin\10.0.22621.0\x86\signtool.exe' verify /v /pa $msiFileName

    Start-Sleep 2

    Write-Output "[*] Displaying MSI summary"
    Install-Module MSI -Force
    Get-MSISummaryInfo $msiFileName
    Get-AuthenticodeSignature $msiFileName|Format-List
}