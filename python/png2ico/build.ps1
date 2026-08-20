<#
    Build script for png2ico (png2ico.py).

    Produces two distributable outputs:
      1. png2ico.pyz  — compressed Python zipapp (run with: python png2ico.pyz)
      2. png2ico.exe  — standalone Windows executable (PyInstaller)

    Requires Python 3.11+ in PATH. Before building, upgrades pip / packaging tools
    (and PyInstaller when building the exe) so mid-build "pip is out of date" notices
    are not missed. Installs PyInstaller automatically when missing.

    Usage:
      ./build.ps1           # both png2ico.pyz and png2ico.exe
      ./build.ps1 -pyz      # png2ico.pyz only
      ./build.ps1 -exe      # png2ico.exe only
      ./build.ps1 -exe -upx # png2ico.exe with optional UPX compression
#>

$ErrorActionPreference = "Stop"

$buildPyz = $args -contains '-pyz'
$buildExe = $args -contains '-exe'
$useUpx = $args -contains '-upx'

if (-not $buildPyz -and -not $buildExe) {
    $buildPyz = $true
    $buildExe = $true
}

function Invoke-PipUpgrade {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Packages,
        [Parameter(Mandatory = $true)]
        [string]$Label
    )
    Write-Host $Label -ForegroundColor Cyan
    Write-Host ("  packages: " + ($Packages -join ', ')) -ForegroundColor DarkGray
    python -m pip install --upgrade @Packages | Out-Host
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to upgrade: $($Packages -join ', ')"
        exit 1
    }
}

Write-Host "Starting build process for png2ico..." -ForegroundColor Cyan

# --- Prerequisites ---
$pythonCmd = Get-Command python -ErrorAction SilentlyContinue
if (-not $pythonCmd) {
    Write-Error "Python is required but was not found in PATH."
    exit 1
}

if (-not (Test-Path "png2ico.py")) {
    Write-Error "png2ico.py not found. Run this script from python/png2ico/."
    exit 1
}

# --- Tooling updates (before any install/build work) ---
# pip itself often prints "You should consider upgrading" during later installs;
# refresh it first so the notice is acted on instead of buried in build output.
Invoke-PipUpgrade -Packages @('pip', 'setuptools', 'wheel') -Label "Checking build tooling for updates..."

$buildRoot = ".\build"
$iconPath = ".\png2ico.ico"
$preparedIconPath = Join-Path $buildRoot "png2ico-embedded.ico"

# --- Cleanup (only outputs being rebuilt) ---
Write-Host "Cleaning previous build outputs..." -ForegroundColor DarkGray
$cleanupPaths = @()
if ($buildPyz) {
    $cleanupPaths += ".\png2ico.pyz", "$buildRoot\zipapp"
}
if ($buildExe) {
    $cleanupPaths += ".\png2ico.exe", $preparedIconPath, "$buildRoot\pyinstaller"
}
foreach ($path in $cleanupPaths) {
    if (Test-Path $path) {
        Remove-Item -Path $path -Recurse -Force
    }
}
New-Item -Path $buildRoot -ItemType Directory -Force | Out-Null

# --- Dependencies ---
Write-Host "Installing dependencies from requirements.txt..."
python -m pip install -r requirements.txt | Out-Host
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to install requirements."
    exit 1
}

if ($buildPyz) {
    Write-Host "png2ico.py -> png2ico.pyz..." -ForegroundColor Cyan
    $zipappDir = Join-Path $buildRoot "zipapp"
    New-Item -Path $zipappDir -ItemType Directory -Force | Out-Null

    Copy-Item -Path ".\png2ico.py" -Destination (Join-Path $zipappDir "__main__.py")
    Copy-Item -Path ".\png2ico" -Destination (Join-Path $zipappDir "png2ico") -Recurse -Force

    Get-ChildItem -Path $zipappDir -Recurse -Directory -Filter "__pycache__" -ErrorAction SilentlyContinue |
        Remove-Item -Recurse -Force

    python -m zipapp $zipappDir -o ".\png2ico.pyz" -p "." -c

    if ($LASTEXITCODE -ne 0) {
        Write-Error "zipapp build failed."
        exit 1
    }

    if (-not (Test-Path ".\png2ico.pyz")) {
        Write-Error "zipapp finished but png2ico.pyz was not created."
        exit 1
    }

    Write-Host "png2ico.pyz build complete." -ForegroundColor Green
    Write-Host "  Run with: python png2ico.pyz" -ForegroundColor DarkGray
}

if (-not $buildExe) {
    exit 0
}

Invoke-PipUpgrade -Packages @('pyinstaller') -Label "Ensuring PyInstaller is installed and up to date..."

Write-Host "png2ico.py -> png2ico.exe..." -ForegroundColor Cyan
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

# Do not --collect-submodules PIL: that pulls ImageMath/ImageQt/ImageTk and
# then NumPy + OpenBLAS (~20 MB) even though png2ico never uses them.
# PyInstaller's Pillow hook already collects _imaging. Exclude optional
# Pillow deps that Image.py probes at import time.
$pyInstallerArgs = @(
    "--onefile",
    "--name", "png2ico",
    "--icon", $iconFile,
    "--console",
    "--clean",
    "--noconfirm",
    "--noupx",
    "--distpath", ".",
    "--workpath", "$buildRoot\pyinstaller",
    "--specpath", "$buildRoot\pyinstaller",
    "--hidden-import", "PIL.Image",
    "--hidden-import", "PIL.PngImagePlugin",
    "--exclude-module", "numpy",
    "--exclude-module", "yaml",
    "--exclude-module", "tkinter",
    "--exclude-module", "PIL.ImageQt",
    "--exclude-module", "PIL.ImageTk",
    "png2ico.py"
)

python -m PyInstaller @pyInstallerArgs

if ($LASTEXITCODE -ne 0) {
    Write-Error "PyInstaller build failed."
    exit 1
}

if (-not (Test-Path ".\png2ico.exe")) {
    Write-Error "PyInstaller finished but png2ico.exe was not created."
    exit 1
}

if ($useUpx) {
    $upxCmd = Get-Command upx -ErrorAction SilentlyContinue
    if ($upxCmd -and $upxCmd.Path) {
        Write-Host "Compressing png2ico.exe with UPX (--best --lzma)..." -ForegroundColor Yellow
        & $upxCmd.Path --best --lzma ".\png2ico.exe"
        if ($LASTEXITCODE -ne 0) {
            Write-Error "UPX compression failed."
            exit 1
        }
        Write-Host "Re-applying icon after UPX..." -ForegroundColor DarkGray
        python prepare_icon.py reapply ".\png2ico.exe" $preparedIconPath
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

Write-Host "png2ico.exe build complete." -ForegroundColor Green
Write-Host "  Standalone executable (icon: png2ico.ico)" -ForegroundColor DarkGray
