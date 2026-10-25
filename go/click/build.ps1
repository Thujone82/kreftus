<#
.SYNOPSIS
    Builds the Click Windows mouse-macro app.

.DESCRIPTION
    Windows only. Cleans ./bin, tidies modules, and builds x86 and x64
    GUI executables (no console window). Pass -upx to compress with UPX.
#>

[CmdletBinding()]
param(
    [switch]$upx
)

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

Write-Host "Starting build process for click..." -ForegroundColor Cyan

if (Test-Path "./bin") {
    Write-Host "Cleaning old bin directory..." -ForegroundColor DarkGray
    Remove-Item -Path "./bin" -Recurse -Force
}

Write-Host "Tidying Go modules..." -ForegroundColor Cyan
go mod tidy

Write-Host "Preparing output directories..." -ForegroundColor Cyan
New-Item -Path "./bin/win/x86" -ItemType Directory -Force | Out-Null
New-Item -Path "./bin/win/x64" -ItemType Directory -Force | Out-Null

$ldflags = "-s -w -H=windowsgui"
$rcFile = Join-Path $PSScriptRoot "click.rc"
$icoFile = Join-Path $PSScriptRoot "click.ico"
$hasIcon = (Test-Path $rcFile) -and (Test-Path $icoFile) -and (Get-Command windres -ErrorAction SilentlyContinue)

try {
    $env:GOOS = "windows"

    if ($hasIcon) {
        Write-Host "Building for Windows (x86) with icon..." -ForegroundColor Cyan
        $env:GOARCH = "386"
        windres -F pe-i386 -I $PSScriptRoot -i $rcFile -o click.syso
        Write-Host "  -> go build (windows/386)" -ForegroundColor DarkGray
        go build -ldflags "$ldflags" -o "./bin/win/x86/click.exe" .

        Write-Host "Building for Windows (x64) with icon..." -ForegroundColor Cyan
        $env:GOARCH = "amd64"
        windres -F pe-x86-64 -I $PSScriptRoot -i $rcFile -o click.syso
        Write-Host "  -> go build (windows/amd64)" -ForegroundColor DarkGray
        go build -ldflags "$ldflags" -o "./bin/win/x64/click.exe" .
    } else {
        if (-not (Get-Command windres -ErrorAction SilentlyContinue)) {
            Write-Host "windres not found; building without click.ico." -ForegroundColor DarkGray
        }
        Write-Host "Building for Windows (x86)..." -ForegroundColor Cyan
        $env:GOARCH = "386"
        Write-Host "  -> go build (windows/386)" -ForegroundColor DarkGray
        go build -ldflags "$ldflags" -o "./bin/win/x86/click.exe" .

        Write-Host "Building for Windows (x64)..." -ForegroundColor Cyan
        $env:GOARCH = "amd64"
        Write-Host "  -> go build (windows/amd64)" -ForegroundColor DarkGray
        go build -ldflags "$ldflags" -o "./bin/win/x64/click.exe" .
    }
}
finally {
    if (Test-Path "click.syso") { Remove-Item "click.syso" -Force }
    Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
}

if ($upx.IsPresent) {
    $upxCmd = Get-Command upx -ErrorAction SilentlyContinue
    if ($upxCmd -and $upxCmd.Path) {
        Write-Host "Compressing binaries with UPX (--best --lzma)..." -ForegroundColor Yellow
        foreach ($bin in @("./bin/win/x86/click.exe", "./bin/win/x64/click.exe")) {
            if (Test-Path $bin) {
                Write-Host "  -> upx $bin" -ForegroundColor DarkGray
                & $upxCmd.Path --best --lzma $bin
            }
        }
        Write-Host "UPX compression completed." -ForegroundColor Green
    } else {
        Write-Host "-upx specified but UPX not found in PATH. Skipping compression." -ForegroundColor DarkGray
    }
} else {
    Write-Host "UPX compression disabled by default. Pass -upx to enable." -ForegroundColor DarkGray
}

Write-Host "Build process completed successfully!" -ForegroundColor Green
