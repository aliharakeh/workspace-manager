#!/usr/bin/env pwsh
# Builds the app into build/bin.
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

wails build
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host ""
Write-Host "Built into build/bin:"
Get-ChildItem build/bin |
  Select-Object Name, @{ n = "MB"; e = { [math]::Round($_.Length / 1MB, 1) } } |
  Format-Table
