Write-Output "Installing and Uninstalling rport..."
Write-Output "------------------------------------"
$ErrorActionPreference = 'Stop'

Start-Process msiexec.exe -Wait -ArgumentList '/I rport-client.msi /quiet'
$files = Get-ChildItem "C:\Program Files\RPort"|Select-Object -Property Name
if(-not ($files.name.Contains('rport.conf'))){
  Write-Error "rport.conf not installed"
}
if(-not ($files.name.Contains('rport.exe'))){
  Write-Error "rport.exe not installed"
}

Get-service -erroraction 'silentlycontinue' | findstr rport > installedServices.txt
# above will create a txt containing:
$expectedServicesContent = @'
Stopped  rport.exe          Rport Client

'@
$whatServiceWasInstalled = Get-Content "installedServices.txt" -raw
if ($whatServiceWasInstalled -eq $expectedServicesContent) {
	Write-Output 'Service Install looks good'
} else {
	Write-Output "wrong services installed"
	exit 1
}

Start-Process msiexec.exe -Wait -ArgumentList '/x rport-client.msi /quiet FORCEREMOVEPRODUCTDIR=YES'

# this will fail only if the folder was not removed from the installer
mkdir 'C:\Program Files\RPort' > $null
