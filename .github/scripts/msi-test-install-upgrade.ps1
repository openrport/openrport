Write-Output "install upgrade sequence msi test"
Write-Output "----------------------------------------------"
$ErrorActionPreference = 'Stop'
Remove-Item *.log
Write-Output "----------------------------------------------------------------------------------------------------"
Write-Output "  - build rport msi ver. 0.1.2"
.github/scripts/msi-buildver.ps1 0 1 2
Write-Output "  - install rport ver 0.1.2"
Start-Process msiexec.exe -Wait -ArgumentList '/i rport-client-ver0.1.2.msi /qn /quiet /log msi-install0.1.2.log'
# New-Item "C:\Program Files\RPort\test123.conf"
Write-Output "  - editing rport.conf"
Add-Content "C:\Program Files\RPort\rport.conf" "`n# Hello, I was added by the user in version 0.1.2"
Write-Output "  - build rport msi ver. 1.3.4"
.github/scripts/msi-buildver.ps1 1 3 4
Write-Output "  - upgrade rport ver 1.3.4 (major upgrade)"
Start-Process msiexec.exe -Wait -ArgumentList '/i rport-client-ver1.3.4.msi /qn /quiet /log msi-install1.3.4.log'
Write-Output "  - check rport.conf is still there "
$files = Get-ChildItem "C:\Program Files\RPort"|Select-Object -Property Name
if (-not($files.name.Contains('rport.conf')))
{
    Write-Error "rport.conf was removed from an upgrade"
}
Write-Output "  - check rport.conf edited is the same "
if (Select-String -Path "C:\Program Files\RPort\rport.conf" -Pattern "Hello, I was added by the user in version 0.1.2" -SimpleMatch -Quiet)
{
    Write-Output "  - found rport.conf edited content, all good!"
}
else
{
    Write-Error "rport.conf does not content user modifications"
}

# Write-Output "  - Uninstall rport."
# Start-Process msiexec.exe -Wait -ArgumentList '/x rport-client-ver1.3.4.msi /qn /quiet /log msi-uninstall.log'
Write-Output "----------------------------------------------------------------------------------------------------"