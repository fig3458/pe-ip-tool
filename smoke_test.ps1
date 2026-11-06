$ErrorActionPreference = "Stop"
$env:PATH = "C:\Users\Administrator\go\bin;" + $env:PATH
$env:GOPATH = "C:\Users\Administrator\gopath"
$env:GOCACHE = "C:\Users\Administrator\gocache"
$env:GOMODCACHE = "C:\Users\Administrator\gomodcache"
$env:CGO_ENABLED = "0"
$env:GOPROXY = "https://goproxy.cn,direct"

$src = "C:\Windows\System32\config\SYSTEM"
$tmp = Join-Path $env:TEMP "peip_SYSTEM_copy"
Copy-Item $src $tmp -Force

$testGo = @'
package main
import (
  "fmt"
  "os"
  "pe-ip-tool/internal/hive"
)
func main() {
  path := os.Args[1]
  if err := hive.Load(path); err != nil { fmt.Println("LOAD_FAIL", err); os.Exit(1) }
  defer hive.Unload()
  ads, err := hive.ListAdapters()
  if err != nil { fmt.Println("LIST_FAIL", err); os.Exit(1) }
  fmt.Println("ADAPTERS", len(ads))
  for i, a := range ads {
    if i >= 5 { break }
    fmt.Printf("%d %s | %s | dhcp=%d ip=%s mask=%s gw=%s dns=%s\n", i, a.Name, a.GUID, a.EnableDHCP, a.IPAddress, a.SubnetMask, a.DefaultGateway, a.NameServer)
  }
  fmt.Println("SMOKE_OK")
}
'@
Set-Content -Path "C:\Users\Administrator\pe-ip-tool\smoke_main.go" -Value $testGo -Encoding UTF8
Push-Location "C:\Users\Administrator\pe-ip-tool"
go build -o smoke.exe .\smoke_main.go 2>&1
# building with extra file conflicts with main.go - use separate folder
Pop-Location
