Write-Output "Install upgrade sequence msi test"
Write-Output "---------------------------------"
$ErrorActionPreference = 'Stop'

Write-Output "[*] Installing goversioninfo"
go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
$ENV:PATH="$ENV:PATH;$($env:home)\go\bin"

Write-Output "[*] Installing WIX"
choco install wixtoolset

Write-Output "  - build rport msi ver. 0.1.2"
.github/scripts/msi-buildver.ps1 0 1 2

Write-Output "  - install rport ver 0.1.2"
Start-Process msiexec.exe -Wait -ArgumentList '/i rport-client-ver0.1.2.msi /qn /quiet /log msi-install0.1.2.log'
if (Select-String -Path "msi-install0.1.2.log" -Pattern "The same or a newer version of this product is already installed" -SimpleMatch -Quiet)
{
    Write-Error "install rport ver 0.1.2 failed: same or a newer version..."
}
 
Write-Output "  - editing rport.conf"
Add-Content "C:\Program Files\RPort\rport.conf" "`n# Hello, I was added by the user in version 0.1.2"
Get-Content "C:\Program Files\RPort\rport.conf" -Tail 3

Write-Output "  - build rport msi ver. 1.3.4"
.github/scripts/msi-buildver.ps1 1 3 4

Write-Output "  - upgrade rport ver 1.3.4 (major upgrade)"
Start-Process msiexec.exe -Wait -ArgumentList '/i rport-client-ver1.3.4.msi /qn /quiet /log msi-install1.3.4.log'
if (Select-String -Path "msi-install1.3.4.log" -Pattern "The same or a newer version of this product is already installed" -SimpleMatch -Quiet)
{
    Write-Error "upgrade to rport ver 1.3.4 failed: same or a newer version..."
}

Write-Output "  - check rport.conf is still there "
$files = Get-ChildItem "C:\Program Files\RPort"|Select-Object -Property Name
if (-not($files.name.Contains('rport.conf')))
{
    Write-Error "rport.conf was removed from an upgrade"
}

Write-Output "  - check rport.conf edited is the same "
Get-Content "C:\Program Files\RPort\rport.conf" -Tail 3
if (Select-String -Path "C:\Program Files\RPort\rport.conf" -Pattern "Hello, I was added by the user in version 0.1.2" -SimpleMatch -Quiet)
{
    Write-Output "  - found rport.conf edited content, all good!"
}
else
{
    Write-Error "rport.conf does not include user modifications"
}