Write-Output "install upgrade sequence msi test"
Write-Output "----------------------------------------------"
$ErrorActionPreference = 'Stop'

Write-Output "----------------------------------------------------------------------------------------------------"
Write-Output "  - build rport msi ver. 0.1.2"
.github/scripts/msi-buildver.ps1 0 1 2
Write-Output "  - install rport ver 0.1.2"
Start-Process msiexec.exe -Wait -ArgumentList '/i rport-client0.1.2.msi /qn '
Write-Output "  - editing rport.conf"
Write-Output "  - build rport msi ver. 1.3.4"
.github/scripts/msi-buildver.ps1 1 3 4
Write-Output "  - upgrade rport ver 1.3.4 (major upgrade)"
Start-Process msiexec.exe -Wait -ArgumentList '/i rport-client1.3.4.msi /qn /quiet'
Write-Output "  - check rport.conf edited is the same "
Write-Output "  - Uninstall rport."
Start-Process msiexec.exe -Wait -ArgumentList '/x rport-client1.3.4.msi /qn /quiet'
Write-Output "----------------------------------------------------------------------------------------------------"