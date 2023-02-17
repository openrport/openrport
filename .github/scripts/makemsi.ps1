Write-Output "Making the MSI..."
Write-Output "-----------------"
$ErrorActionPreference = 'Stop'

Write-Output "Install goversioninfo"
go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo

Write-Output "Install WIX"
choco install wixtoolset

Write-Output "Building MSI"
Get-Content new.json > cmd\rport\versioninfo.json
Write-Output "Version the client with whatever is in versioninfo.json"
go generate cmd/rport/main.go
Write-Output "Build exe client"
go build -ldflags "-s -w -X {{.Env.PROJECT}}/share.BuildVersion={{.Version}}" -o rport.exe ./cmd/rport/... 
Write-Output "creates wixobj's"
& 'C:\Program Files (x86)\WiX Toolset v3.11\bin\candle.exe' -dPlatform=x64 -ext WixUtilExtension cmd/rport/resource/*.wxs
Write-Output "creates MSI"
& 'C:\Program Files (x86)\WiX Toolset v3.11\bin\light.exe' -loc cmd/rport/resource/Product_en-us.wxl -ext WixUtilExtension -ext WixUIExtension -sval -out rport-client.msi LicenseAgreementDlg_HK.wixobj WixUI_HK.wixobj Product.wixobj

Write-Output "creating a self signed certificate"
$cert = New-SelfSignedCertificate -DnsName selfsignedtest.rport.com -CertStoreLocation cert:\LocalMachine\My -type CodeSigning
$MyPassword = ConvertTo-SecureString -String "MyPassword" -Force -AsPlainText
Export-PfxCertificate -cert $cert -FilePath mycert.pfx -Password $MyPassword

Write-Output "signing the generated MSI"
& 'C:\Program Files (x86)\Windows Kits\10\bin\10.0.22621.0\x86\signtool.exe' sign /fd SHA256 /f mycert.pfx /p MyPassword rport-client.msi