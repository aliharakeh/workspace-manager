#!/usr/bin/env pwsh
# Builds everything into build/bin: the app, its daemon, and the MCP server.
# The app and wm-mcp start wmd.exe from the folder they live in, so those three must stay together.
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

New-Item -ItemType Directory -Force build/bin | Out-Null

# The MCP server talks over stdio, so it stays a console program.
go build -ldflags "-s -w" -o build/bin/wm-mcp.exe ./cmd/wm-mcp
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

wails3 build
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host ""
Write-Host "Built into build/bin:"
Get-ChildItem build/bin |
  Select-Object Name, @{ n = "MB"; e = { [math]::Round($_.Length / 1MB, 1) } } |
  Format-Table
