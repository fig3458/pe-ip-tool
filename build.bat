@echo off
setlocal
cd /d "%~dp0"

if exist "%USERPROFILE%\go\bin\go.exe" set "PATH=%USERPROFILE%\go\bin;%PATH%"
where go >nul 2>&1
if errorlevel 1 (
  echo Go not found. Install Go or extract portable Go to %%USERPROFILE%%\go
  exit /b 1
)

set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
go mod tidy
go build -ldflags="-s -w -H windowsgui" -o pe-ip-tool.exe .
if errorlevel 1 exit /b 1
dir pe-ip-tool.exe
echo OK
