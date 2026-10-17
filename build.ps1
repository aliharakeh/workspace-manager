#!/usr/bin/env pwsh
# Builds everything into build/bin: the app, its daemon, the MCP server and the
# VS Code extension (.vsix). The app and wm-mcp start wmd.exe from the folder they
# live in, so those three must stay together.
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

New-Item -ItemType Directory -Force build/bin | Out-Null

go build -ldflags "-s -w -H windowsgui" -o build/bin/wmd.exe ./cmd/wmd
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

# The MCP server talks over stdio, so it stays a console program.
go build -ldflags "-s -w" -o build/bin/wm-mcp.exe ./cmd/wm-mcp
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

wails build
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

# The VS Code extension; its build writes the .vsix to build/bin.
Push-Location vscode
try {
  if (-not (Test-Path node_modules)) {
    npm install
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
  }
  npm run build
  if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
} finally {
  Pop-Location
}

Write-Host ""
Write-Host "Built into build/bin:"
Get-ChildItem build/bin |
  Select-Object Name, @{ n = "MB"; e = { [math]::Round($_.Length / 1MB, 1) } } |
  Format-Table
