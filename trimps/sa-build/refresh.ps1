# Rebuild self-contained trimps/sa.html from the community sheet snapshot.
# A user of the planner only needs sa.html — this folder is the authoring pipeline.
# Usage (from this folder or via full path):
#   .\refresh.ps1           # reuse cached workbook if present
#   .\refresh.ps1 -Fresh    # re-download the spreadsheet, then rebuild
param(
  [switch]$Fresh
)

$ErrorActionPreference = 'Stop'
$here = $PSScriptRoot
$trimps = Split-Path -Parent $here

if (-not (Get-Command python -ErrorAction SilentlyContinue)) {
  throw 'python not found on PATH'
}

if ($Fresh) {
  $cache = Join-Path $here '_sa.xlsx'
  if (Test-Path -LiteralPath $cache) {
    Remove-Item -LiteralPath $cache -Force
    Write-Host 'Removed cached workbook.'
  }
}

Write-Host 'Snapshotting community SA sheet and item catalog...'
$buildArgs = @((Join-Path $here 'build_data.py'))
if ($Fresh) { $buildArgs += '--fresh' }
& python @buildArgs
if ($LASTEXITCODE) { throw "snapshot failed ($LASTEXITCODE)" }

Write-Host 'Assembling self-contained sa.html...'
& python (Join-Path $here 'assemble.py')
if ($LASTEXITCODE) { throw "assemble failed ($LASTEXITCODE)" }

$out = Join-Path $trimps 'sa.html'
$size = (Get-Item -LiteralPath $out).Length
Write-Host "Wrote $out ($size bytes)"
