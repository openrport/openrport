$file = "tacoscript-latest-windows-amd64.zip"
iwr https://github.com/cloudradar-monitoring/tacoscript/releases/download/latest/tacoscript-latest-windows-amd64.zip `
-OutFile $file
$dest = "C:\Program Files\tacoscript"
$destBin = "$($dest)\bin"

if(!(Test-Path -path $dest))
{
 New-Item -ItemType directory -Path $dest
 Write-Host "Folder path has been created successfully at: " $dest
 }

if(!(Test-Path -path $destBin))
{
 New-Item -ItemType directory -Path $destBin
 Write-Host "Folder path has been created successfully at: " $destBin
 }

Expand-Archive -Path $file -DestinationPath $dest -force

Get-ChildItem "$($dest)\tacoscript.exe" -file | Move-Item -destination $destBin -force

Get-ChildItem -Path $destBin

$ENV:PATH="$ENV:PATH;$($destBin)"

[Environment]::SetEnvironmentVariable(
    "Path",
    [Environment]::GetEnvironmentVariable(
        "Path", [EnvironmentVariableTarget]::Machine
    ) + ";$($destBin)",
    [EnvironmentVariableTarget]::Machine
)

gci env:Path

tacoscript --version