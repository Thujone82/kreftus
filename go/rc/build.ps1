<#
.SYNOPSIS
    Builds the 'rc' Go application for multiple platforms.

.DESCRIPTION
    This script automates the build process for the 'rc' (Run Continuously) application.
    It performs the following steps:
    1. Checks for the 'windres' tool, required for embedding icons in Windows executables.
    2. Cleans up any previous build artifacts by deleting the './bin' directory.
    3. Tidies the Go module dependencies.
    4. Creates a structured './bin' directory for the compiled binaries.
    5. Compiles the application for four target platforms, embedding an icon for Windows builds.
    6. Strips debugging information from the binaries to reduce their size.

.NOTES
    Requires Go and 'windres' (from a MinGW-w64 toolchain) to be in the system's PATH.
    The script expects 'main.go', 'rc.rc', and 'cli_rc_icon.ico' to be in the same directory.
#>

[CmdletBinding()]
param(
    [switch]$upx
)

$ErrorActionPreference = "Stop"

Write-Host "Starting build process for rc..." -ForegroundColor Cyan

# --- 1. Prerequisite Check ---
Write-Host "Checking for 'windres' tool..."
$windresPath = Get-Command windres -ErrorAction SilentlyContinue
if (-not $windresPath) {
    Write-Error "The 'windres' tool is required to embed icons in Windows executables."
    Write-Warning "Please install a MinGW-w64 toolchain (e.g., via MSYS2 or 'choco install mingw')."
    exit 1
}

# --- 2. Cleanup and Setup ---
$binDir = ".\bin"
if (Test-Path $binDir) {
    Write-Host "Cleaning up old build directory..."
    Remove-Item -Path $binDir -Recurse -Force
}

Write-Host "Tidying Go modules..."
go mod tidy

Write-Host "Creating output directories..."
New-Item -Path ".\bin\win\x86" -ItemType Directory -Force | Out-Null
New-Item -Path ".\bin\win\x64" -ItemType Directory -Force | Out-Null # Using x64 as is standard for amd64
New-Item -Path ".\bin\linux\x86" -ItemType Directory -Force | Out-Null
New-Item -Path ".\bin\linux\amd64" -ItemType Directory -Force | Out-Null

# --- 3. Windows Builds (with icon) ---
Write-Host "Generating Windows resources using windres..."
if (-not (Test-Path "rc.rc") -or -not (Test-Path "cli_rc_icon.ico")) {
    Write-Error "rc.rc and/or cli_rc_icon.ico not found. Cannot build Windows executables with an icon."
    exit 1
}

try {
    Write-Host "Building for Windows 32-bit (x86)..."
    Write-Host "  -> Generating 32-bit resources..." -ForegroundColor DarkGray
    $env:GOOS = "windows"; $env:GOARCH = "386"
    windres -F pe-i386 -i rc.rc -o rc.syso -I .
    Write-Host "  -> go build (windows/386) with -v" -ForegroundColor DarkGray
    go build -v -ldflags="-s -w" -o ".\bin\win\x86\rc.exe" .

    Write-Host "Building for Windows 64-bit (amd64)..."
    Write-Host "  -> Generating 64-bit resources..." -ForegroundColor DarkGray
    $env:GOOS = "windows"; $env:GOARCH = "amd64"
    windres -F pe-x86-64 -i rc.rc -o rc.syso -I .
    Write-Host "  -> go build (windows/amd64) with -v" -ForegroundColor DarkGray
    go build -v -ldflags="-s -w" -o ".\bin\win\x64\rc.exe" .
}
finally {
    if (Test-Path "rc.syso") { Remove-Item "rc.syso" -Force }
}

# --- 4. Linux Builds ---
Write-Host "Building for Linux 32-bit (x86)..."
$env:GOOS = "linux"; $env:GOARCH = "386"; Write-Host "  -> go build (linux/386) with -v" -ForegroundColor DarkGray; go build -v -ldflags="-s -w" -o ".\bin\linux\x86\rc" .

Write-Host "Building for Linux 64-bit (amd64)..."
$env:GOOS = "linux"; $env:GOARCH = "amd64"; Write-Host "  -> go build (linux/amd64) with -v" -ForegroundColor DarkGray; go build -v -ldflags="-s -w" -o ".\bin\linux\amd64\rc" .

# --- 5. Optional UPX compression (only when -upx is specified) ---
if ($upx.IsPresent) {
    $upxCmd = Get-Command upx -ErrorAction SilentlyContinue
    if ($upxCmd -and $upxCmd.Path) {
        Write-Host "Compressing binaries with UPX (--best --lzma)..." -ForegroundColor Yellow
        $binaries = @(
            ".\bin\win\x86\rc.exe",
            ".\bin\win\x64\rc.exe",
            ".\bin\linux\x86\rc",
            ".\bin\linux\amd64\rc"
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