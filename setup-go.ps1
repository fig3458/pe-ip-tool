$ErrorActionPreference = "Stop"
Get-Process curl -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
$zip = "C:\Users\Administrator\go.zip"
$dest = "C:\Users\Administrator"
$goBin = "C:\Users\Administrator\go\bin\go.exe"

function Test-ZipOk([string]$path) {
    if (-not (Test-Path $path)) { return $false }
    $len = (Get-Item $path).Length
    if ($len -lt 70000000) { return $false }
    $fs = [IO.File]::OpenRead($path)
    try {
        $b0 = $fs.ReadByte(); $b1 = $fs.ReadByte()
        if ($b0 -ne 0x50 -or $b1 -ne 0x4B) { return $false }
    } finally { $fs.Close() }
    return $true
}

if (-not (Test-ZipOk $zip)) {
    Write-Host "Downloading Go..."
    Remove-Item $zip -Force -ErrorAction SilentlyContinue
    Remove-Item "C:\Users\Administrator\go" -Recurse -Force -ErrorAction SilentlyContinue
    $ Progressive = $false
    Invoke-WebRequest -Uri "https://mirrors.aliyun.com/golang/go1.23.4.windows-amd64.zip" -OutFile $zip -UseBasicParsing
}

if (-not (Test-ZipOk $zip)) {
    throw "go.zip invalid"
}

Write-Host "Extracting..."
Remove-Item "C:\Users\Administrator\go" -Recurse -Force -ErrorAction SilentlyContinue
Expand-Archive -Path $zip -DestinationPath $dest -Force
& $goBin version
if ($LASTEXITCODE -ne 0) { throw "go version failed" }
Write-Host "GO_READY"
