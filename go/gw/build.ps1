<#
.SYNOPSIS
    Builds the 'gw' Go application for multiple platforms.

.DESCRIPTION
    This script automates the build process for the 'gw' (Get Weather) application.
    It performs the following steps:
    1. Checks for the 'rsrc' tool, which is required for embedding icons in Windows executables.
    2. Cleans up any previous build artifacts by deleting the './bin' directory.
    3. Tidies the Go module dependencies using 'go mod tidy'.
    4. Creates a structured './bin' directory for the compiled binaries.
    5. Compiles the application for four target platforms:
        - Windows 32-bit (with gw.ico)
        - Windows 64-bit (with gw.ico)
        - Linux 32-bit
        - Linux 64-bit
    6. Strips debugging information from the binaries to reduce their size.

.NOTES
    Requires Go to be installed and in the system's PATH.
    Requires 'windres' for Windows builds, which is part of a MinGW-w64 toolchain.
    On Windows, you can install it via MSYS2 or Chocolatey (choco install mingw).

    The script expects 'gw.go' and 'gw.ico' to be in the same directory.
#>

$ErrorActionPreference = "Stop"

Write-Host "Starting build process for gw..." -ForegroundColor Cyan

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
if (-not (Test-Path "gw.rc") -or -not (Test-Path "gw.ico")) {
    Write-Error "gw.rc and/or gw.ico not found. Cannot build Windows executables with an icon."
    exit 1
}

try {
    Write-Host "Building for Windows 32-bit (x86)..."
    Write-Host "  -> Generating 32-bit resources..." -ForegroundColor DarkGray
    $env:GOOS = "windows"; $env:GOARCH = "386"
    windres -F pe-i386 -i gw.rc -o gw.syso -I .
    go build -ldflags="-s -w" -o ".\bin\win\x86\gw.exe" .

    Write-Host "Building for Windows 64-bit (amd64)..."
    Write-Host "  -> Generating 64-bit resources..." -ForegroundColor DarkGray
    $env:GOOS = "windows"; $env:GOARCH = "amd64"
    windres -F pe-x86-64 -i gw.rc -o gw.syso -I .
    go build -ldflags="-s -w" -o ".\bin\win\x64\gw.exe" .
}
finally {
    if (Test-Path "gw.syso") { Remove-Item "gw.syso" -Force }
}

# --- 4. Linux Builds ---
Write-Host "Building for Linux 32-bit (x86)..."
$env:GOOS = "linux"; $env:GOARCH = "386"; go build -ldflags="-s -w" -o ".\bin\linux\x86\gw" "gw.go"

Write-Host "Building for Linux 64-bit (amd64)..."
$env:GOOS = "linux"; $env:GOARCH = "amd64"; go build -ldflags="-s -w" -o ".\bin\linux\amd64\gw" "gw.go"

# --- 5. Optional UPX compression ---
$upxCmd = Get-Command upx -ErrorAction SilentlyContinue
if ($upxCmd -and $upxCmd.Path) {
    Write-Host "Compressing binaries with UPX (--best --lzma)..." -ForegroundColor Yellow
    $binaries = @(
        ".\bin\win\x86\gw.exe",
        ".\bin\win\x64\gw.exe",
        ".\bin\linux\x86\gw",
        ".\bin\linux\amd64\gw"
    )
    foreach ($bin in $binaries) {
        if (Test-Path $bin) {
            & $upxCmd.Path --best --lzma $bin | Out-Null
        }
    }
    Write-Host "UPX compression completed." -ForegroundColor Green
} else {
    Write-Host "UPX not found in PATH. Skipping binary compression." -ForegroundColor DarkGray
}

Write-Host "Build process completed successfully!" -ForegroundColor Green