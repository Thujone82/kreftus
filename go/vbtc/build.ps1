<#
    This build script compiles the vBTC Go application for Windows, macOS, and Linux.
    It automatically sets its location to the script's directory to ensure all
    relative paths are resolved correctly.
#>
Set-Location $PSScriptRoot

# Define the version number in one place for easy updates.
$Version = "1.4"

# --- Build Helper Tool ---
# Capture the host's Go environment settings to ensure our helper tool is always
# built for the platform running this script.
$ZipperSource = "tools/zipper/main.go"
# Use Join-Path to create a full, unambiguous path to the helper executable.
$ZipperExePath = Join-Path $PSScriptRoot "zipper.exe"

Write-Host "Building helper tools..." -ForegroundColor Cyan
$env:GOOS = "windows"; $env:GOARCH = "amd64"; go build -o $ZipperExePath $ZipperSource

# Add error checking right after the build to ensure the helper was created.
if (-not (Test-Path $ZipperExePath)) {
    Write-Host "------------------------------------------------------------------" -ForegroundColor Red
    Write-Host "Build Error: Failed to compile the helper tool 'zipper.exe'." -ForegroundColor Red
    Write-Host "Please check for Go compilation errors in the output above." -ForegroundColor Yellow
    Write-Host "------------------------------------------------------------------" -ForegroundColor Red
    exit 1
}

Write-Host "Checking for build tools..." -ForegroundColor Cyan
# Check for windres command, which is required for embedding the Windows icon.
if (-not (Get-Command windres -ErrorAction SilentlyContinue)) {
    Write-Host "------------------------------------------------------------------" -ForegroundColor Red
    Write-Host "Build Error: 'windres' command not found." -ForegroundColor Red
    Write-Host "This tool is required to embed the icon into the Windows .exe file." -ForegroundColor Yellow
    Write-Host ""
    Write-Host "To fix this, please install the MinGW-w64 toolchain." -ForegroundColor Yellow
    Write-Host "The easiest way is to use the Chocolatey package manager:" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  choco install mingw" -ForegroundColor Green
    Write-Host ""
    Write-Host "After installation, please restart your PowerShell terminal and run this script again." -ForegroundColor Yellow
    Write-Host "------------------------------------------------------------------" -ForegroundColor Red
    exit 1
}

Write-Host "Tidying Go modules..." -ForegroundColor Cyan
go mod tidy

Write-Host "Building for Windows..." -ForegroundColor Cyan
Write-Host "  - Compiling icon resource..."
# Pass the absolute path to the .rc file to avoid any ambiguity for the compiler.
$rcFilePath = Join-Path $PSScriptRoot 'vbtc.rc'
# Use the -I flag to specify the include directory, ensuring windres can find both the .rc file and the .ico file it references.

# Use a try/finally block to ensure the Windows-specific .syso file is always cleaned up.
try {
    windres -I $PSScriptRoot -o vbtc.syso $rcFilePath
    Write-Host "  - Compiling executable (x64)..."
    # Explicitly set the OS and Architecture for the Windows build to avoid environment issues.
    $env:GOOS="windows"; $env:GOARCH="amd64"; go build -ldflags="-s -w" -o bin/pc/x64/vbtc.exe
    Write-Host "  - Compiling executable (x86)..."
    $env:GOOS="windows"; $env:GOARCH="386"; go build -ldflags="-s -w" -o bin/pc/x86/vbtc.exe
}
finally {
    if (Test-Path "vbtc.syso") {
        Remove-Item vbtc.syso -Force
    }
}

Write-Host "Building for macOS (Apple Silicon)..." -ForegroundColor Cyan
$env:GOOS="darwin"; $env:GOARCH="arm64"; go build -ldflags="-s -w" -o bin/mac/arm64/vbtc

Write-Host "Building for macOS (Intel)..." -ForegroundColor Cyan
$env:GOOS="darwin"; $env:GOARCH="amd64"; go build -ldflags="-s -w" -o bin/mac/amd64/vbtc

Write-Host "Building for Linux (x64)..." -ForegroundColor Cyan
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -ldflags="-s -w" -o bin/linux/amd64/vbtc

Write-Host "Building for Linux (x86)..." -ForegroundColor Cyan
$env:GOOS="linux"; $env:GOARCH="386"; go build -ldflags="-s -w" -o bin/linux/x86/vbtc

# --- Package macOS Applications ---
Write-Host "Packaging macOS applications..." -ForegroundColor Cyan

if (-not (Test-Path "vbtc.icns")) {
    Write-Warning "vbtc.icns not found. macOS app bundles will be created without an icon."
}

# Define Info.plist content using a PowerShell Here-String
$infoPlistContent = @"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>vbtc</string>
    <key>CFBundleIconFile</key>
    <string>vbtc.icns</string>
    <key>CFBundleIdentifier</key>
    <string>com.kreftus.vbtc</string>
    <key>CFBundleName</key>
    <string>vBTC</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>$Version</string>
    <key>CFBundleVersion</key>
    <string>$Version</string>
</dict>
</plist>
"@

@( @{ arch = "arm64"; name = "Apple Silicon" }, @{ arch = "amd64"; name = "Intel" } ) | ForEach-Object {
    Write-Host "  - Packaging $($_.name) build..."
    $binaryPath = "bin/mac/$($_.arch)/vbtc"
    $appPath = "bin/mac/$($_.arch)/vbtc.app"
    if (Test-Path $binaryPath) {
        New-Item -Path "$appPath/Contents/MacOS" -ItemType Directory -Force | Out-Null
        New-Item -Path "$appPath/Contents/Resources" -ItemType Directory -Force | Out-Null
        Move-Item -Path $binaryPath -Destination "$appPath/Contents/MacOS/vbtc" -Force
        if (Test-Path "vbtc.icns") { Copy-Item -Path "vbtc.icns" -Destination "$appPath/Contents/Resources/vbtc.icns" -Force }
        $infoPlistContent | Set-Content -Path "$appPath/Contents/Info.plist"
    
        # Compress using the custom zipper to preserve executable permissions.
        $zipPath = "bin/mac/$($_.arch)/vbtc.zip"
        $readmePath = "README.txt"
        Write-Host "    - Compressing to $zipPath..."
        # Execute the custom zipper tool. The paths are relative to the script's location.
        # Use the call operator (&) for robust execution of the helper tool.
        & $ZipperExePath $zipPath $appPath $readmePath
        Remove-Item -Path $appPath -Recurse -Force # Clean up the .app directory
    
    } else {
        Write-Warning "Skipping packaging for $($_.name) because the binary was not found at $binaryPath."
    }
}

Write-Host "Cleaning up helper tools..." -ForegroundColor Cyan
if (Test-Path $ZipperExePath) { Remove-Item $ZipperExePath -Force }

Write-Host "Build complete!" -ForegroundColor Green