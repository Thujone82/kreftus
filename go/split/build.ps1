<#
.SYNOPSIS
    Builds the 'Split' video splitter/joiner binaries for Windows, Linux, and macOS.

.DESCRIPTION
    Cross-compiles CGo-free GUI + CLI binaries directly from Windows into ./bin:
      - Windows: ./bin/win/x86/split.exe, ./bin/win/x64/split.exe (with icon + -H=windowsgui)
      - Linux:   ./bin/linux/x64/split,   ./bin/linux/arm64/split (purego GTK3 + WebKit2GTK)
      - macOS:   ./bin/mac/arm64/split,   ./bin/mac/x64/split     (purego Cocoa + WKWebView)
    Pass -upx to compress Windows and Linux binaries with UPX.
#>

[CmdletBinding()]
param(
    [switch]$upx
)

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

Write-Host "Starting multi-platform build process for split..." -ForegroundColor Cyan

if (Test-Path "./bin") {
    Write-Host "Cleaning old bin directory..." -ForegroundColor DarkGray
    Get-ChildItem -Path "./bin" -Recurse -File -ErrorAction SilentlyContinue | Remove-Item -Force -ErrorAction SilentlyContinue
    Remove-Item -Path "./bin" -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Host "Tidying Go modules..." -ForegroundColor Cyan
go mod tidy
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$targets = @(
    "./bin/win/x86",
    "./bin/win/x64",
    "./bin/linux/x64",
    "./bin/linux/arm64",
    "./bin/mac/arm64",
    "./bin/mac/x64"
)
Write-Host "Preparing output directories..." -ForegroundColor Cyan
foreach ($dir in $targets) {
    New-Item -Path $dir -ItemType Directory -Force | Out-Null
}

$winLdflags = "-s -w -H=windowsgui"
$unixLdflags = "-s -w"
$rcFile = Join-Path $PSScriptRoot "split.rc"
$icoFile = Join-Path $PSScriptRoot "split.ico"
$hasIcon = (Test-Path $rcFile) -and (Test-Path $icoFile) -and (Get-Command windres -ErrorAction SilentlyContinue)
$sysoFile = "split_windows.syso"

$env:CGO_ENABLED = "0"
try {
    # 1. Windows builds (x86 & x64)
    $env:GOOS = "windows"
    if ($hasIcon) {
        Write-Host "Building for Windows (x86) with icon..." -ForegroundColor Cyan
        $env:GOARCH = "386"
        windres -F pe-i386 -I $PSScriptRoot -i $rcFile -o $sysoFile
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        go build -ldflags "$winLdflags" -o "./bin/win/x86/split.exe" .
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

        Write-Host "Building for Windows (x64) with icon..." -ForegroundColor Cyan
        $env:GOARCH = "amd64"
        windres -F pe-x86-64 -I $PSScriptRoot -i $rcFile -o $sysoFile
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        go build -ldflags "$winLdflags" -o "./bin/win/x64/split.exe" .
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    } else {
        if (-not (Get-Command windres -ErrorAction SilentlyContinue)) {
            Write-Host "windres not found; building Windows binaries without split.ico." -ForegroundColor DarkGray
        }
        Write-Host "Building for Windows (x86)..." -ForegroundColor Cyan
        $env:GOARCH = "386"
        go build -ldflags "$winLdflags" -o "./bin/win/x86/split.exe" .
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

        Write-Host "Building for Windows (x64)..." -ForegroundColor Cyan
        $env:GOARCH = "amd64"
        go build -ldflags "$winLdflags" -o "./bin/win/x64/split.exe" .
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    }
    if (Test-Path $sysoFile) { Remove-Item $sysoFile -Force }

    # 2. Linux builds (x64 & arm64)
    $env:GOOS = "linux"
    Write-Host "Building for Linux (x64)..." -ForegroundColor Cyan
    $env:GOARCH = "amd64"
    go build -ldflags "$unixLdflags" -o "./bin/linux/x64/split" .
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    Write-Host "Building for Linux (arm64)..." -ForegroundColor Cyan
    $env:GOARCH = "arm64"
    go build -ldflags "$unixLdflags" -o "./bin/linux/arm64/split" .
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    # 3. macOS builds (arm64 & x64)
    $env:GOOS = "darwin"
    Write-Host "Building for macOS (arm64 / Apple Silicon)..." -ForegroundColor Cyan
    $env:GOARCH = "arm64"
    go build -ldflags "$unixLdflags" -o "./bin/mac/arm64/split" .
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    Write-Host "Building for macOS (x64 / Intel)..." -ForegroundColor Cyan
    $env:GOARCH = "amd64"
    go build -ldflags "$unixLdflags" -o "./bin/mac/x64/split" .
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
finally {
    if (Test-Path $sysoFile) { Remove-Item $sysoFile -Force }
    if (Test-Path "split.syso") { Remove-Item "split.syso" -Force }
    Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
    Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue
}

if ($upx.IsPresent) {
    $upxCmd = Get-Command upx -ErrorAction SilentlyContinue
    if ($upxCmd -and $upxCmd.Path) {
        Write-Host "Compressing Windows & Linux binaries with UPX (--best --lzma)..." -ForegroundColor Yellow
        foreach ($bin in @("./bin/win/x86/split.exe", "./bin/win/x64/split.exe", "./bin/linux/x64/split", "./bin/linux/arm64/split")) {
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
