Write-Output "Install upgrade sequence msi test"
Write-Output "---------------------------------"
$ErrorActionPreference = 'Stop'

Write-Output "  - build rport msi ver. 0.9.10"
.github/scripts/msi-build.ps1 -major 0 -minor 9 -patch 10 -SignMsi:$false

Write-Output "  - install rport ver 0.9.10"
Start-Process msiexec.exe -Wait -ArgumentList '/i rport-client.msi /qn /quiet /log msi-install0.9.10.log'
if (Select-String -Path "msi-install0.9.10.log" -Pattern "The same or a newer version of this product is already installed" -SimpleMatch -Quiet)
{
    Write-Error "install rport ver 0.9.10 failed: same or a newer version..."
}
 
Write-Output "  - adding and editing rport.conf"
Add-Content "C:\Program Files\RPort\rport.conf" "`n# Hello, I was edited by the user in version 0.9.10"

Write-Output "  - build rport msi ver. 0.9.11"
.github/scripts/msi-build.ps1 0 9 11 -SignMsi:$false

Write-Output "  - upgrade rport ver 0.9.11 (patch upgrade)"
Start-Process msiexec.exe -Wait -ArgumentList '/i rport-client.msi /qn /quiet /log msi-install0.9.11.log'
if (Select-String -Path "msi-install0.9.11.log" -Pattern "The same or a newer version of this product is already installed" -SimpleMatch -Quiet)
{
    Write-Error "upgrade to rport ver 0.9.11 failed: same or a newer version..."
}

Write-Output "  - check rport.conf is still there "
$files = Get-ChildItem "C:\Program Files\RPort"|Select-Object -Property Name
if (-not($files.name.Contains('rport.conf')))
{
    Write-Error "rport.conf was removed from an upgrade"
}

Write-Output "  - check rport.conf edited is the same "
if (Select-String -Path "C:\Program Files\RPort\rport.conf" -Pattern "Hello, I was edited by the user in version 0.9.10" -SimpleMatch -Quiet)
{
    Write-Output "  - found rport.conf edited content, all good!"
}
else
{
    Write-Error "rport.conf does not include user modifications"
}