$ErrorActionPreference = "Stop"
$env:PATH = "C:\Users\Administrator\go\bin;" + $env:PATH
$env:GOPATH = "C:\Users\Administrator\gopath"
$env:GOCACHE = "C:\Users\Administrator\gocache"
$env:GOMODCACHE = "C:\Users\Administrator\gomodcache"
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:GOPROXY = "https://goproxy.cn,direct"
New-Item -ItemType Directory -Force -Path $env:GOPATH, $env:GOCACHE, $env:GOMODCACHE | Out-Null
Set-Location "C:\Users\Administrator\pe-ip-tool"
go mod tidy
go build -ldflags "-s -w -H windowsgui" -o pe-ip-tool.exe .
if (-not (Test-Path ".\pe-ip-tool.exe")) { throw "build failed" }
Get-Item ".\pe-ip-tool.exe" | Format-List Name, Length, FullName
Write-Host "BUILD_OK"
