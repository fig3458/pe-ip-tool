$ErrorActionPreference = "Stop"
$env:PATH = "C:\Users\Administrator\go\bin;" + $env:PATH
$env:GOPATH = "C:\Users\Administrator\gopath"
$env:GOCACHE = "C:\Users\Administrator\gocache"
$env:GOMODCACHE = "C:\Users\Administrator\gomodcache"
$env:CGO_ENABLED = "0"
$env:GOPROXY = "https://goproxy.cn,direct"

$hiveCopy = Join-Path $env:TEMP "peip_SYSTEM_copy"
if (-not (Test-Path $hiveCopy)) {
    & reg.exe save "HKLM\SYSTEM" $hiveCopy /y | Out-Null
}
Set-Location "C:\Users\Administrator\pe-ip-tool\smoketest"
go mod tidy
go build -o smoke.exe .
& .\smoke.exe $hiveCopy
