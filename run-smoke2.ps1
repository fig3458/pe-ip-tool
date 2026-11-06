$ErrorActionPreference = "Stop"
$env:PATH = "C:\Users\Administrator\go\bin;" + $env:PATH
$env:GOPATH = "C:\Users\Administrator\gopath"
$env:GOCACHE = "C:\Users\Administrator\gocache"
$env:GOMODCACHE = "C:\Users\Administrator\gomodcache"
$env:CGO_ENABLED = "0"
$env:GOPROXY = "https://goproxy.cn,direct"

Set-Location "C:\Users\Administrator\pe-ip-tool"
go build -o smoke.exe ./cmd/smoke
$h = Join-Path $env:TEMP "peip_SYSTEM_copy"
if (-not (Test-Path $h)) {
    & reg.exe save "HKLM\SYSTEM" $h /y | Out-Null
}
& .\smoke.exe $h
