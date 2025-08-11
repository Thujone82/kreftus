<#
.SYNOPSIS
    Builds the 'larry' Go terminal game for Windows and Linux.

.DESCRIPTION
    Mirrors the structure used in your other Go projects.
    - Cleans previous bin output
    - go mod tidy
    - Builds for Windows (x86, x64) and Linux (x86, amd64)
    - Places artifacts under ./bin/...
#>

[CmdletBinding()]
param(
    [switch]$upx
)

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

Write-Host "Starting build process for larry..." -ForegroundColor Cyan

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
New-Item -Path "./bin/win/x86" -ItemType Directory -Force | Out-Null
New-Item -Path "./bin/win/x64" -ItemType Directory -Force | Out-Null
New-Item -Path "./bin/linux/x86" -ItemType Directory -Force | Out-Null
New-Item -Path "./bin/linux/amd64" -ItemType Directory -Force | Out-Null

$ldflags = "-s -w"

try {
    # 4) Windows builds (console app)
    # Windows builds with icon (if windres and larry.ico/larry.rc are present)
    $rcFile = Join-Path $PSScriptRoot 'larry.rc'
    $icoFile = Join-Path $PSScriptRoot 'larry.ico'
    $hasIcon = (Test-Path $rcFile) -and (Test-Path $icoFile) -and (Get-Command windres -ErrorAction SilentlyContinue)

    if ($hasIcon) {
        Write-Host "Building for Windows (x86) with icon..." -ForegroundColor Cyan
        $env:GOOS = "windows"; $env:GOARCH = "386"
        windres -F pe-i386 -I $PSScriptRoot -i $rcFile -o larry.syso
        Write-Host "  -> go build (windows/386) with -v" -ForegroundColor DarkGray
        go build -v -ldflags "$ldflags" -o "./bin/win/x86/larry.exe" .

        Write-Host "Building for Windows (x64) with icon..." -ForegroundColor Cyan
        $env:GOOS = "windows"; $env:GOARCH = "amd64"
        windres -F pe-x86-64 -I $PSScriptRoot -i $rcFile -o larry.syso
        Write-Host "  -> go build (windows/amd64) with -v" -ForegroundColor DarkGray
        go build -v -ldflags "$ldflags" -o "./bin/win/x64/larry.exe" .

        if (Test-Path "larry.syso") { Remove-Item "larry.syso" -Force }
    } else {
        Write-Host "Building for Windows (x86)..." -ForegroundColor Cyan
        $env:GOOS = "windows"; $env:GOARCH = "386"; Write-Host "  -> go build (windows/386) with -v" -ForegroundColor DarkGray; go build -v -ldflags "$ldflags" -o "./bin/win/x86/larry.exe" .

        Write-Host "Building for Windows (x64)..." -ForegroundColor Cyan
        $env:GOOS = "windows"; $env:GOARCH = "amd64"; Write-Host "  -> go build (windows/amd64) with -v" -ForegroundColor DarkGray; go build -v -ldflags "$ldflags" -o "./bin/win/x64/larry.exe" .
    }

    # 5) Linux builds
    Write-Host "Building for Linux (x86)..." -ForegroundColor Cyan
    $env:GOOS = "linux"; $env:GOARCH = "386"; Write-Host "  -> go build (linux/386) with -v" -ForegroundColor DarkGray; go build -v -ldflags "$ldflags" -o "./bin/linux/x86/larry" .

    Write-Host "Building for Linux (amd64)..." -ForegroundColor Cyan
    $env:GOOS = "linux"; $env:GOARCH = "amd64"; Write-Host "  -> go build (linux/amd64) with -v" -ForegroundColor DarkGray; go build -v -ldflags "$ldflags" -o "./bin/linux/amd64/larry" .
}
finally {
    # Clean env overrides
    Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
}

# 6) Optional UPX compression (only when -upx is specified)
if ($upx.IsPresent) {
    $upxCmd = Get-Command upx -ErrorAction SilentlyContinue
    if ($upxCmd -and $upxCmd.Path) {
        Write-Host "Compressing binaries with UPX (--best --lzma)..." -ForegroundColor Yellow
        $binaries = @(
            "./bin/win/x86/larry.exe",
            "./bin/win/x64/larry.exe",
            "./bin/linux/x86/larry",
            "./bin/linux/amd64/larry"
        )
        foreach ($bin in $binaries) {
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


