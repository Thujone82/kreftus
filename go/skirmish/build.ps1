<#
.SYNOPSIS
    Builds the 'Skirmish DICE' Go GUI game for Windows and Linux.

.DESCRIPTION
    - Cleans previous bin output
    - go mod tidy
    - Compiles Windows icon with windres if skirmish.ico exists
    - Builds Windows binaries (x64, x86) with -H=windowsgui (no console window)
    - Builds Linux binaries (amd64)
#>

[CmdletBinding()]
param(
    [switch]$upx
)

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

Write-Host "Starting build process for Skirmish DICE..." -ForegroundColor Cyan

# 1) Cleanup bin
if (Test-Path "./bin") {
    Write-Host "Cleaning old bin directory..." -ForegroundColor DarkGray
    Remove-Item -Path "./bin" -Recurse -Force
}

# 2) Tidy modules
Write-Host "Tidying Go modules..." -ForegroundColor Cyan
go mod tidy

# 3) Create output directories
Write-Host "Preparing output directories..." -ForegroundColor Cyan
New-Item -Path "./bin/win/x64" -ItemType Directory -Force | Out-Null
New-Item -Path "./bin/win/x86" -ItemType Directory -Force | Out-Null
New-Item -Path "./bin/linux/amd64" -ItemType Directory -Force | Out-Null

$winLdflags = "-s -w -H=windowsgui"
$linuxLdflags = "-s -w"

try {
    # 4) Windows builds
    $rcFile = Join-Path $PSScriptRoot 'skirmish.rc'
    $icoFile = Join-Path $PSScriptRoot 'skirmish.ico'
    $hasIcon = (Test-Path $rcFile) -and (Test-Path $icoFile) -and (Get-Command windres -ErrorAction SilentlyContinue)

    if ($hasIcon) {
        Write-Host "Building for Windows (x64) with icon..." -ForegroundColor Cyan
        $env:GOOS = "windows"; $env:GOARCH = "amd64"
        windres -F pe-x86-64 -I $PSScriptRoot -i $rcFile -o skirmish.syso
        go build -v -ldflags "$winLdflags" -o "./bin/win/x64/SkirmishDice.exe" .

        Write-Host "Building for Windows (x86) with icon..." -ForegroundColor Cyan
        $env:GOOS = "windows"; $env:GOARCH = "386"
        windres -F pe-i386 -I $PSScriptRoot -i $rcFile -o skirmish.syso
        go build -v -ldflags "$winLdflags" -o "./bin/win/x86/SkirmishDice.exe" .

        if (Test-Path "skirmish.syso") { Remove-Item "skirmish.syso" -Force }
    } else {
        Write-Host "Building for Windows (x64) [GUI only - no console window]..." -ForegroundColor Cyan
        $env:GOOS = "windows"; $env:GOARCH = "amd64"
        go build -v -ldflags "$winLdflags" -o "./bin/win/x64/SkirmishDice.exe" .

        Write-Host "Building for Windows (x86) [GUI only - no console window]..." -ForegroundColor Cyan
        $env:GOOS = "windows"; $env:GOARCH = "386"
        go build -v -ldflags "$winLdflags" -o "./bin/win/x86/SkirmishDice.exe" .
    }

    # Copy companion asset files alongside executable for external tools
    if (Test-Path "skirmish.png") {
        Copy-Item "skirmish.png" "./bin/win/x64/skirmish.png" -Force
        Copy-Item "skirmish.png" "./bin/win/x86/skirmish.png" -Force
    }
    if (Test-Path "skirmish.ico") {
        Copy-Item "skirmish.ico" "./bin/win/x64/skirmish.ico" -Force
        Copy-Item "skirmish.ico" "./bin/win/x86/skirmish.ico" -Force
    }

    # 5) Optional Linux build (requires CGO on Linux hosts)
    try {
        Write-Host "Building for Linux (amd64)..." -ForegroundColor Cyan
        $env:GOOS = "linux"; $env:GOARCH = "amd64"
        go build -v -ldflags "$linuxLdflags" -o "./bin/linux/amd64/SkirmishDice" .
    } catch {
        Write-Host "  -> Note: Linux build skipped (Ebitengine requires Linux Cgo/X11 toolchain)." -ForegroundColor Yellow
    }
}
finally {
    if (Test-Path "skirmish.syso") { Remove-Item "skirmish.syso" -Force }
    Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
}

# 6) Optional UPX compression
if ($upx.IsPresent) {
    $upxCmd = Get-Command upx -ErrorAction SilentlyContinue
    if ($upxCmd -and $upxCmd.Path) {
        Write-Host "Compressing binaries with UPX..." -ForegroundColor Yellow
        $binaries = @(
            "./bin/win/x64/SkirmishDice.exe",
            "./bin/win/x86/SkirmishDice.exe",
            "./bin/linux/amd64/SkirmishDice"
        )
        foreach ($bin in $binaries) {
            if (Test-Path $bin) {
                & $upxCmd.Path --best --lzma $bin
            }
        }
    }
}

Write-Host "Build finished successfully! Binary is located at: ./bin/win/x64/SkirmishDice.exe" -ForegroundColor Green
