[CmdletBinding()]
param (
      [int] $major
    , [int] $minor
    , [int] $patch
)

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
Write-Output $versionInfo|convertTo-Json
# Write the file used for the build process
$versionInfo|ConvertTo-Json|Out-File -Path cmd/rport/versioninfo.json
# Convert the versioninfo.json to resource.syso
Set-Location ./cmd/rport
goversioninfo.exe
Set-Location ../../

Write-Output "[*] Building rport.exe for windows"
go build -ldflags "-s -w -X github.com/realvnc-labs/rport/share.BuildVersion=$major.$minor.$patch" -o rport.exe ./cmd/rport
Get-ChildItem -File *.exe

Write-Output "[*] Creating wixobj's"
& 'C:\Program Files (x86)\WiX Toolset v3.11\bin\candle.exe' -dPlatform=x64 -ext WixUtilExtension opt/resource/*.wxs

Write-Output "[*] Creating MSI"
& 'C:\Program Files (x86)\WiX Toolset v3.11\bin\light.exe' `
  -loc opt/resource/Product_en-us.wxl `
  -ext WixUtilExtension -ext WixUIExtension -sval `
  -out rport-client-ver"$major.$minor.$patch".msi LicenseAgreementDlg_HK.wixobj WixUI_HK.wixobj Product.wixobj
Get-ChildItem -File *.msi

exit 0

Write-Output "  - install rport ver 0.1.2"
Start-Process msiexec.exe -Wait -ArgumentList '/i rport-client.msi /qn '

Write-Output "  - editing rport.conf"
Write-Output "  - build rport msi ver. 1.3.0"
Write-Output "  - upgrade rport ver 1.3.0 (major upgrade)"
Write-Output "  - check rport.conf edited is the same "
Write-Output "  - Uninstall rport."
Write-Output "--------------------------------------------------"

# New-Item 'C:\Program Files\RPort\ciccio.conf'
Start-Process msiexec.exe -Wait -ArgumentList '/i rport-client.msi /qn /quiet /log msi-install.log'

# $files = Get-ChildItem "C:\Program Files\RPort"|Select-Object -Property Name
# if (-not($files.name.Contains('ciccio.conf')))
# {
#     Write-Error "rport.conf was overwritten / removed from an upgrade"
# }
Start-Process msiexec.exe -Wait -ArgumentList '/x rport-client.msi /qn /quiet /log msi-uninstall.log'

# if (Test-Path 'C:\Program Files\RPort')
# {
#     Write-Error "Folder was not removed after MSI uninstallation"
# }



# Write-Output "Installing and Uninstalling rport..."
# Write-Output "------------------------------------"
# $ErrorActionPreference = 'Stop'

# Get-ChildItem *.msi
# Start-Process msiexec.exe -Wait -ArgumentList '/i rport-client.msi /qn /quiet /log msi-install.log'
# Get-ChildItem *.log
# Get-Content msi-install.log

# $files = Get-ChildItem "C:\Program Files\RPort"|Select-Object -Property Name

# if (-not($files.name.Contains('rport.conf')))
# {
#     Write-Error "rport.conf not installed"
# }

# if (-not($files.name.Contains('rport.exe')))
# {
#     Write-Error "rport.exe not installed"
# }

# if (-not(get-service 'RPort client'))
# {
#     Write-Output "Service not installed"
# }

# Start-Process msiexec.exe -Wait -ArgumentList '/x rport-client.msi /qn /quiet /log msi-uninstall.log'
# #Get-Content msi-uninstall.log

# if (Test-Path 'C:\Program Files\RPort')
# {
#     Write-Error "Folder was not removed after MSI uninstallation"
# }
