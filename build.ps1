#!/usr/bin/env pwsh
# Builds the daemon, the MCP server and the GUI into build/bin.
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

# Windows locks running executables, so the daemon must stop before wmd is rebuilt.
if (Get-Process wmd -ErrorAction SilentlyContinue) {
  Get-Process wmd -ErrorAction SilentlyContinue | Stop-Process -Force
  Start-Sleep -Milliseconds 500
}

New-Item -ItemType Directory build/bin -Force | Out-Null

go build -o build/bin/wmd.exe ./server/cmd/wmd
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
go build -o build/bin/wm-mcp.exe ./mcp/cmd/wm-mcp
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

# wails joins -o with gui/build/bin, so ../../../ points at the repo root.
Push-Location gui
try {
  wails build -o ../../../build/bin/workspace-manager.exe
  if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
} finally {
  Pop-Location
}

# wails dev runs the app from gui/build/bin; put the daemon next to it.
Copy-Item build/bin/wmd.exe gui/build/bin/wmd.exe -Force

Write-Host ""
Write-Host "Built into build/bin:"
Get-ChildItem build/bin |
  Select-Object Name, @{ n = "MB"; e = { [math]::Round($_.Length / 1MB, 1) } } |
  Format-Table
