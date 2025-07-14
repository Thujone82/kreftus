<#
.SYNOPSIS
    Builds the 'rc' Go application for multiple platforms.

.DESCRIPTION
    This script automates the build process for the 'rc' (Run Continuously) application.
    It performs the following steps:
    1. Cleans up any previous build artifacts by deleting the './bin' directory.
    2. Initializes a Go module if one doesn't exist and tidies dependencies.
    3. Creates a structured './bin' directory for the compiled binaries.
    4. Compiles the application for four target platforms:
        - Windows 32-bit
        - Windows 64-bit
        - Linux 32-bit
        - Linux 64-bit
    5. Strips debugging information from the binaries to reduce their size.

.NOTES
    Requires Go to be installed and in the system's PATH.
    The script expects 'main.go' to be in the same directory.
#>

$ErrorActionPreference = "Stop"

Write-Host "Starting build process for rc..." -ForegroundColor Cyan

# --- 1. Cleanup and Setup ---
$binDir = ".\bin"
if (Test-Path $binDir) {
    Write-Host "Cleaning up old build directory..."
    Remove-Item -Path $binDir -Recurse -Force
}

Write-Host "Tidying Go modules..."
if (-not (Test-Path "go.mod")) {
    Write-Host "go.mod not found, initializing new module 'rc'..." -ForegroundColor Yellow
    go mod init rc
}

Write-Host "Ensuring dependencies are downloaded..." -ForegroundColor DarkGray
go get github.com/fatih/color

go mod tidy

Write-Host "Creating output directories..."
New-Item -Path ".\bin\win\x86" -ItemType Directory -Force | Out-Null
New-Item -Path ".\bin\win\x64" -ItemType Directory -Force | Out-Null # Using x64 as is standard for amd64
New-Item -Path ".\bin\linux\x86" -ItemType Directory -Force | Out-Null
New-Item -Path ".\bin\linux\amd64" -ItemType Directory -Force | Out-Null

# --- 2. Build Binaries ---
$env:GOOS = "windows"; $env:GOARCH = "386"; go build -ldflags="-s -w" -o ".\bin\win\x86\rc.exe" .
$env:GOOS = "windows"; $env:GOARCH = "amd64"; go build -ldflags="-s -w" -o ".\bin\win\x64\rc.exe" .
$env:GOOS = "linux"; $env:GOARCH = "386"; go build -ldflags="-s -w" -o ".\bin\linux\x86\rc" .
$env:GOOS = "linux"; $env:GOARCH = "amd64"; go build -ldflags="-s -w" -o ".\bin\linux\amd64\rc" .

Write-Host "Build process completed successfully!" -ForegroundColor Green