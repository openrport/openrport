$file = "tacoscript-latest-windows-amd64.zip"
iwr https://github.com/cloudradar-monitoring/tacoscript/releases/download/latest/tacoscript-latest-windows-amd64.zip `
-OutFile $file
$dest = "C:\Program Files\tacoscript\bin"

if(!(Test-Path -path $dest))
{
 New-Item -ItemType directory -Path $dest
 Write-Host "Folder path has been created successfully at: " $dest
 }

Expand-Archive -Path $file -DestinationPath $dest -force

Get-ChildItem "$($dest)\tacoscript.exe" -file | Move-Item -destination "$($dest)\bin" -force

$ENV:PATH="$ENV:PATH;$($dest)\bin"

[Environment]::SetEnvironmentVariable(
    "Path",
    [Environment]::GetEnvironmentVariable(
        "Path", [EnvironmentVariableTarget]::Machine
    ) + ";$($dest)\bin",
    [EnvironmentVariableTarget]::Machine
)