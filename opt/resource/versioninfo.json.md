# versioninfo.json

This file is used by [goversioninfo](https://github.com/josephspurrier/goversioninfo) to generate the
Microsoft Windows File Properties/Version Info.

`versioninfo.json` is a template. Values are inserted before compiling the Windows version of the rport client.
See [msi-build.ps1](/.github/scripts/msi-build.ps1)
 
Example:
```shell
cd ./cmd/rport
goversioninfo.exe
```

This creates the file `resource.syso` and `go build` will pick it up to embed the data into the .exe file.

If `goversioninfo` is not installed on your system, install with:
```powershell
go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
$ENV:PATH="$ENV:PATH;$($env:home)\go\bin"
```