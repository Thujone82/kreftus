<#
    Build script for TemPy (tp.py).

    Produces two distributable outputs:
      1. tp.pyz  — compressed Python zipapp (run with: python tp.pyz)
      2. tp.exe  — standalone Windows executable (PyInstaller)

    Requires Python 3.11+ in PATH. Installs PyInstaller automatically when missing.

    Usage:
      ./build.ps1           # both tp.pyz and tp.exe
      ./build.ps1 -pyz      # tp.pyz only
      ./build.ps1 -exe      # tp.exe only
      ./build.ps1 -exe -upx # tp.exe with optional UPX compression
#>

$ErrorActionPreference = "Stop"

$buildPyz = $args -contains '-pyz'
$buildExe = $args -contains '-exe'
$useUpx = $args -contains '-upx'

if (-not $buildPyz -and -not $buildExe) {
    $buildPyz = $true
    $buildExe = $true
}

Write-Host "Starting build process for tp..." -ForegroundColor Cyan

# --- Prerequisites ---
$pythonCmd = Get-Command python -ErrorAction SilentlyContinue
if (-not $pythonCmd) {
    Write-Error "Python is required but was not found in PATH."
    exit 1
}

if (-not (Test-Path "tp.py")) {
    Write-Error "tp.py not found. Run this script from python/tp/."
    exit 1
}

$buildRoot = ".\build"
$iconPath = Join-Path $buildRoot "thermo.ico"
$preparedIconPath = Join-Path $buildRoot "thermo-embedded.ico"

# --- Cleanup (only outputs being rebuilt) ---
Write-Host "Cleaning previous build outputs..." -ForegroundColor DarkGray
$cleanupPaths = @()
if ($buildPyz) {
    $cleanupPaths += ".\tp.pyz", "$buildRoot\zipapp"
}
if ($buildExe) {
    $cleanupPaths += ".\tp.exe", $preparedIconPath, "$buildRoot\pyinstaller"
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
    Write-Host "tp.py -> tp.pyz..." -ForegroundColor Cyan
    $zipappDir = Join-Path $buildRoot "zipapp"
    New-Item -Path $zipappDir -ItemType Directory -Force | Out-Null

    Copy-Item -Path ".\tp.py" -Destination (Join-Path $zipappDir "__main__.py")
    Copy-Item -Path ".\tp" -Destination (Join-Path $zipappDir "tp") -Recurse -Force

    Get-ChildItem -Path $zipappDir -Recurse -Directory -Filter "__pycache__" -ErrorAction SilentlyContinue |
        Remove-Item -Recurse -Force

    python -m zipapp $zipappDir -o ".\tp.pyz" -p "." -c

    if ($LASTEXITCODE -ne 0) {
        Write-Error "zipapp build failed."
        exit 1
    }

    if (-not (Test-Path ".\tp.pyz")) {
        Write-Error "zipapp finished but tp.pyz was not created."
        exit 1
    }

    Write-Host "tp.pyz build complete." -ForegroundColor Green
    Write-Host "  Run with: python tp.pyz" -ForegroundColor DarkGray
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

Write-Host "tp.py -> tp.exe..." -ForegroundColor Cyan
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

$pyInstallerArgs = @(
    "--onefile",
    "--name", "tp",
    "--icon", $iconFile,
    "--clean",
    "--noconfirm",
    "--noupx",
    "--distpath", ".",
    "--workpath", "$buildRoot\pyinstaller",
    "--specpath", "$buildRoot\pyinstaller",
    "--hidden-import", "bleak",
    "--hidden-import", "textual",
    "--collect-submodules", "textual",
    "--collect-submodules", "bleak",
    "tp.py"
)

python -m PyInstaller @pyInstallerArgs

if ($LASTEXITCODE -ne 0) {
    Write-Error "PyInstaller build failed."
    exit 1
}

if (-not (Test-Path ".\tp.exe")) {
    Write-Error "PyInstaller finished but tp.exe was not created."
    exit 1
}

if ($useUpx) {
    $upxCmd = Get-Command upx -ErrorAction SilentlyContinue
    if ($upxCmd -and $upxCmd.Path) {
        Write-Host "Compressing tp.exe with UPX (--best --lzma)..." -ForegroundColor Yellow
        & $upxCmd.Path --best --lzma ".\tp.exe"
        if ($LASTEXITCODE -ne 0) {
            Write-Error "UPX compression failed."
            exit 1
        }
        Write-Host "Re-applying icon after UPX..." -ForegroundColor DarkGray
        python prepare_icon.py reapply ".\tp.exe" $preparedIconPath
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

Write-Host "tp.exe build complete." -ForegroundColor Green
Write-Host "  Standalone executable (icon: build/thermo.ico)" -ForegroundColor DarkGray
