Write-Output "Installing and Uninstalling rport..."
Write-Output "------------------------------------"
$ErrorActionPreference = 'Stop'

Get-ChildItem *.msi
Start-Process msiexec.exe -Wait -ArgumentList '/i rport-client.msi /qn /quiet /log msi-install.log'
Get-ChildItem *.log
Get-Content msi-install.log

$files = Get-ChildItem "C:\Program Files\RPort"|Select-Object -Property Name

if (-not($files.name.Contains('rport.conf')))
{
    Write-Error "rport.conf not installed"
}

if (-not($files.name.Contains('rport.exe')))
{
    Write-Error "rport.exe not installed"
}

if (-not(get-service 'RPort client'))
{
    Write-Output "Service not installed"
}

Start-Process msiexec.exe -Wait -ArgumentList '/x rport-client.msi /qn /quiet /log msi-uninstall.log'
#Get-Content msi-uninstall.log

if (Test-Path 'C:\Program Files\RPort')
{
    Write-Error "Folder was not removed after MSI uninstallation"
}
