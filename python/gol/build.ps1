<#
    Build script for GoLPy (gol.py).

    Produces two distributable outputs:
      1. gol.pyz  — compressed Python zipapp (run with: python gol.pyz)
      2. gol.exe  — standalone Windows executable (PyInstaller)

    Requires Python 3.11+ in PATH. Installs PyInstaller automatically when missing.

    Usage:
      ./build.ps1           # both gol.pyz and gol.exe
      ./build.ps1 -pyz      # gol.pyz only
      ./build.ps1 -exe      # gol.exe only
      ./build.ps1 -exe -upx # gol.exe with optional UPX compression
#>

$ErrorActionPreference = "Stop"

$buildPyz = $args -contains '-pyz'
$buildExe = $args -contains '-exe'
$useUpx = $args -contains '-upx'

if (-not $buildPyz -and -not $buildExe) {
    $buildPyz = $true
    $buildExe = $true
}

Write-Host "Starting build process for gol..." -ForegroundColor Cyan

$pythonCmd = Get-Command python -ErrorAction SilentlyContinue
if (-not $pythonCmd) {
    Write-Error "Python is required but was not found in PATH."
    exit 1
}

if (-not (Test-Path "gol.py")) {
    Write-Error "gol.py not found. Run this script from python/gol/."
    exit 1
}

$buildRoot = ".\build"
$iconPath = Join-Path $buildRoot "icon-32.ico"
$preparedIconPath = Join-Path $buildRoot "gol-embedded.ico"

Write-Host "Cleaning previous build outputs..." -ForegroundColor DarkGray
$cleanupPaths = @()
if ($buildPyz) {
    $cleanupPaths += ".\gol.pyz", "$buildRoot\zipapp"
}
if ($buildExe) {
    $cleanupPaths += ".\gol.exe", $preparedIconPath, "$buildRoot\pyinstaller"
}
foreach ($path in $cleanupPaths) {
    if (Test-Path $path) {
        Remove-Item -Path $path -Recurse -Force
    }
}
New-Item -Path $buildRoot -ItemType Directory -Force | Out-Null

$iconPngSource = Join-Path $buildRoot "32x32.png"
$assetsDir = Join-Path ".\gol" "assets"
$iconPngAsset = Join-Path $assetsDir "32x32.png"
if (Test-Path $iconPngSource) {
    New-Item -Path $assetsDir -ItemType Directory -Force | Out-Null
    Copy-Item -Path $iconPngSource -Destination $iconPngAsset -Force
}

Write-Host "Installing dependencies from requirements.txt..."
python -m pip install -r requirements.txt | Out-Host
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to install requirements."
    exit 1
}

if ($buildPyz) {
    Write-Host "gol.py -> gol.pyz..." -ForegroundColor Cyan
    $zipappDir = Join-Path $buildRoot "zipapp"
    New-Item -Path $zipappDir -ItemType Directory -Force | Out-Null

    Copy-Item -Path ".\gol.py" -Destination (Join-Path $zipappDir "__main__.py")
    Copy-Item -Path ".\gol" -Destination (Join-Path $zipappDir "gol") -Recurse -Force

    Get-ChildItem -Path $zipappDir -Recurse -Directory -Filter "__pycache__" -ErrorAction SilentlyContinue |
        Remove-Item -Recurse -Force

    python -m zipapp $zipappDir -o ".\gol.pyz" -p "." -c

    if ($LASTEXITCODE -ne 0) {
        Write-Error "zipapp build failed."
        exit 1
    }

    if (-not (Test-Path ".\gol.pyz")) {
        Write-Error "zipapp finished but gol.pyz was not created."
        exit 1
    }

    Write-Host "gol.pyz build complete." -ForegroundColor Green
    Write-Host "  Run with: python gol.pyz" -ForegroundColor DarkGray
}

if (-not $buildExe) {
    exit 0
}

Write-Host "Checking for PyInstaller..."
python -c "import PyInstaller" 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "PyInstaller not found. Installing..." -ForegroundColor Yellow
    python -m pip install pyinstaller | Out-Host
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to install PyInstaller."
        exit 1
    }
}

Write-Host "gol.py -> gol.exe..." -ForegroundColor Cyan
if (-not (Test-Path $iconPath)) {
    Write-Error "Icon not found: $iconPath"
    exit 1
}

Write-Host "Preparing icon for Windows embedding..." -ForegroundColor DarkGray
python prepare_icon.py $iconPath $preparedIconPath
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to prepare icon."
    exit 1
}
$iconFile = (Resolve-Path $preparedIconPath).Path
$patternsJson = (Resolve-Path ".\gol\patterns.json").Path

$pyInstallerArgs = @(
    "--onefile",
    "--name", "gol",
    "--icon", $iconFile,
    "--clean",
    "--noconfirm",
    "--noupx",
    "--distpath", ".",
    "--workpath", "$buildRoot\pyinstaller",
    "--specpath", "$buildRoot\pyinstaller",
    "--hidden-import", "pygame",
    "--collect-submodules", "pygame"
)
foreach ($dataSpec in @("$patternsJson;gol")) {
    $pyInstallerArgs += "--add-data", $dataSpec
}
if (Test-Path $iconPngAsset) {
    $pyInstallerArgs += "--add-data", "$(Resolve-Path $iconPngAsset);gol/assets"
}
$pyInstallerArgs += "gol.py"

python -m PyInstaller @pyInstallerArgs

if ($LASTEXITCODE -ne 0) {
    Write-Error "PyInstaller build failed."
    exit 1
}

if (-not (Test-Path ".\gol.exe")) {
    Write-Error "PyInstaller finished but gol.exe was not created."
    exit 1
}

if ($useUpx) {
    $upxCmd = Get-Command upx -ErrorAction SilentlyContinue
    if ($upxCmd -and $upxCmd.Path) {
        Write-Host "Compressing gol.exe with UPX (--best --lzma)..." -ForegroundColor Yellow
        & $upxCmd.Path --best --lzma ".\gol.exe"
        if ($LASTEXITCODE -ne 0) {
            Write-Error "UPX compression failed."
            exit 1
        }
        Write-Host "Re-applying icon after UPX..." -ForegroundColor DarkGray
        python prepare_icon.py reapply ".\gol.exe" $preparedIconPath
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to re-apply icon after UPX."
            exit 1
        }
        Write-Host "UPX compression completed." -ForegroundColor Green
    } else {
        Write-Host "-upx specified but UPX not found in PATH. Skipping compression." -ForegroundColor DarkGray
    }
} else {
    Write-Host "UPX compression disabled by default. Pass -upx to enable." -ForegroundColor DarkGray
}

Write-Host "gol.exe build complete." -ForegroundColor Green
Write-Host "  Standalone executable (icons: build/icon-32.ico, build/32x32.png)" -ForegroundColor DarkGray
