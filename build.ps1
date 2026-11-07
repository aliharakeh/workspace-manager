#!/usr/bin/env pwsh
# Builds the GUI, MCP server and daemon modes into one executable.
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

New-Item -ItemType Directory -Force build/bin | Out-Null

wails3 build
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host ""
Write-Host "Built into build/bin:"
Get-ChildItem build/bin |
  Select-Object Name, @{ n = "MB"; e = { [math]::Round($_.Length / 1MB, 1) } } |
  Format-Table
