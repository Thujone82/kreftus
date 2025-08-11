<#
.SYNOPSIS
    Builds the 'bmon' Go application for multiple platforms.

.DESCRIPTION
    This script automates the build process for the 'bmon' (Bitcoin Monitor) application.
    It performs the following steps:
    1. Checks for the 'windres' tool, required for embedding icons in Windows executables.
    2. Cleans up any previous build artifacts by deleting the './bin' directory.
    3. Tidies the Go module dependencies.
    4. Creates a structured './bin' directory for the compiled binaries.
    5. Compiles the application for four target platforms, embedding an icon for Windows builds.
    6. Strips debugging information from the binaries to reduce their size.

.NOTES
    Requires Go and 'windres' (from a MinGW-w64 toolchain) to be in the system's PATH.
    The script expects 'main.go', 'bmon.rc', and 'bitcoin_small.ico' to be in the same directory.
#>

$PSDefaultParameterValues['Out-File:Encoding'] = 'utf8'
# Optional flag parsing: use UPX only when '-upx' is provided
$useUpx = $args -contains '-upx'

$ErrorActionPreference = "Stop"

Write-Host "Starting build process for bmon..." -ForegroundColor Cyan

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
if (-not (Test-Path "bmon.rc") -or -not (Test-Path "bitcoin_small.ico")) {
    Write-Error "bmon.rc and/or bitcoin_small.ico not found. Cannot build Windows executables with an icon."
    exit 1
}

try {
    Write-Host "Building for Windows 32-bit (x86)..."
    Write-Host "  -> Generating 32-bit resources..." -ForegroundColor DarkGray
    $env:GOOS = "windows"; $env:GOARCH = "386"; $env:CGO_ENABLED = "0"
    windres -F pe-i386 -i bmon.rc -o bmon.syso -I .
    Write-Host "  -> go build (windows/386) with -v" -ForegroundColor DarkGray
    go build -v -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o ".\bin\win\x86\bmon.exe" .

    Write-Host "Building for Windows 64-bit (amd64)..."
    Write-Host "  -> Generating 64-bit resources..." -ForegroundColor DarkGray
    $env:GOOS = "windows"; $env:GOARCH = "amd64"; $env:CGO_ENABLED = "0"
    windres -F pe-x86-64 -i bmon.rc -o bmon.syso -I .
    Write-Host "  -> go build (windows/amd64) with -v" -ForegroundColor DarkGray
    go build -v -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o ".\bin\win\x64\bmon.exe" .
}
finally {
    if (Test-Path "bmon.syso") { Remove-Item "bmon.syso" -Force }
}

# --- 4. Linux Builds ---
Write-Host "Building for Linux 32-bit (x86)..."
$env:GOOS = "linux"; $env:GOARCH = "386"; $env:CGO_ENABLED = "0"; Write-Host "  -> go build (linux/386) with -v" -ForegroundColor DarkGray; go build -v -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o ".\bin\linux\x86\bmon" .

Write-Host "Building for Linux 64-bit (amd64)..."
$env:GOOS = "linux"; $env:GOARCH = "amd64"; $env:CGO_ENABLED = "0"; Write-Host "  -> go build (linux/amd64) with -v" -ForegroundColor DarkGray; go build -v -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o ".\bin\linux\amd64\bmon" .

# --- 5. Optional UPX compression (only when -upx is specified) ---
if ($useUpx) {
    $upxCmd = Get-Command upx -ErrorAction SilentlyContinue
    if ($upxCmd -and $upxCmd.Path) {
        Write-Host "Compressing binaries with UPX (--best --lzma)..." -ForegroundColor Yellow
        $binaries = @(
            ".\bin\win\x86\bmon.exe",
            ".\bin\win\x64\bmon.exe",
            ".\bin\linux\x86\bmon",
            ".\bin\linux\amd64\bmon"
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
