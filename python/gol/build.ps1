<#
    Build script for GoLPy (gol.py).

    Produces distributable outputs:
      1. gol.pyz      — compressed Python zipapp (run with: python gol.pyz)
      2. gol.exe      — standalone GUI executable (PyInstaller + pygame)
      3. gol-tui.exe  — standalone TUI executable (PyInstaller + textual, no pygame)

    Requires Python 3.11+ in PATH. Installs PyInstaller automatically when missing.

    Usage:
      ./build.ps1              # gol.pyz, gol.exe, then gol-tui.exe
      ./build.ps1 -pyz         # gol.pyz only
      ./build.ps1 -exe         # gol.exe only
      ./build.ps1 -tui         # gol-tui.exe only
      ./build.ps1 -exe -upx    # gol.exe with optional UPX compression
#>

$ErrorActionPreference = "Stop"

$buildPyz = $args -contains '-pyz'
$buildGuiExe = $args -contains '-exe'
$buildTuiExe = $args -contains '-tui'
$useUpx = $args -contains '-upx'
$explicitTarget = $buildPyz -or $buildGuiExe -or $buildTuiExe

if (-not $explicitTarget) {
    $buildPyz = $true
    $buildGuiExe = $true
    $buildTuiExe = $true
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
if ($buildGuiExe) {
    $cleanupPaths += ".\gol.exe", "$buildRoot\pyinstaller-gui"
}
if ($buildTuiExe) {
    $cleanupPaths += ".\gol-tui.exe", "$buildRoot\pyinstaller-tui"
}
if ($buildGuiExe -or $buildTuiExe) {
    $cleanupPaths += $preparedIconPath
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
    Copy-Item -Path ".\gol_tui.py" -Destination (Join-Path $zipappDir "gol_tui.py") -Force
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

if (-not $buildGuiExe -and -not $buildTuiExe) {
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

function Invoke-GolPyInstaller {
    param(
        [string]$ExeName,
        [string]$EntryScript,
        [string]$WorkSubdir,
        [string[]]$HiddenImports,
        [string[]]$CollectSubmodules,
        [string[]]$ExcludeModules = @(),
        [switch]$IncludeIconAsset
    )

    if (-not $ExeName -or -not $EntryScript) {
        Write-Error "Invoke-GolPyInstaller requires ExeName and EntryScript."
        exit 1
    }

    Write-Host "$EntryScript -> $ExeName..." -ForegroundColor Cyan

    $pyInstallerArgs = @(
        "--onefile",
        "--name", $ExeName,
        "--icon", $iconFile,
        "--clean",
        "--noconfirm",
        "--noupx",
        "--distpath", ".",
        "--workpath", (Join-Path $buildRoot $WorkSubdir),
        "--specpath", (Join-Path $buildRoot $WorkSubdir)
    )
    foreach ($module in $HiddenImports) {
        $pyInstallerArgs += "--hidden-import", $module
    }
    foreach ($module in $CollectSubmodules) {
        $pyInstallerArgs += "--collect-submodules", $module
    }
    foreach ($module in $ExcludeModules) {
        $pyInstallerArgs += "--exclude-module", $module
    }
    $pyInstallerArgs += "--add-data", "$patternsJson;gol"
    if ($IncludeIconAsset -and (Test-Path $iconPngAsset)) {
        $pyInstallerArgs += "--add-data", "$(Resolve-Path $iconPngAsset);gol/assets"
    }
    $pyInstallerArgs += $EntryScript

    python -m PyInstaller @pyInstallerArgs
    if ($LASTEXITCODE -ne 0) {
        Write-Error "PyInstaller build failed for $ExeName."
        exit 1
    }

    $outputPath = Join-Path "." "$ExeName.exe"
    if (-not (Test-Path $outputPath)) {
        Write-Error "PyInstaller finished but $outputPath was not created."
        exit 1
    }

    if ($useUpx -and $ExeName -eq "gol") {
        $upxCmd = Get-Command upx -ErrorAction SilentlyContinue
        if ($upxCmd -and $upxCmd.Path) {
            Write-Host "Compressing $ExeName.exe with UPX (--best --lzma)..." -ForegroundColor Yellow
            & $upxCmd.Path --best --lzma $outputPath
            if ($LASTEXITCODE -ne 0) {
                Write-Error "UPX compression failed."
                exit 1
            }
            Write-Host "Re-applying icon after UPX..." -ForegroundColor DarkGray
            python prepare_icon.py reapply $outputPath $preparedIconPath
            if ($LASTEXITCODE -ne 0) {
                Write-Error "Failed to re-apply icon after UPX."
                exit 1
            }
            Write-Host "UPX compression completed." -ForegroundColor Green
        } else {
            Write-Host "-upx specified but UPX not found in PATH. Skipping compression." -ForegroundColor DarkGray
        }
    }

    Write-Host "$ExeName.exe build complete." -ForegroundColor Green
}

if ($buildGuiExe) {
    Invoke-GolPyInstaller -ExeName 'gol' -EntryScript 'gol.py' -WorkSubdir 'pyinstaller-gui' `
        -HiddenImports @('pygame') -CollectSubmodules @('pygame') -ExcludeModules @('textual') -IncludeIconAsset
    Write-Host "  GUI executable (pygame; icons: build/icon-32.ico, build/32x32.png)" -ForegroundColor DarkGray
} elseif ($useUpx) {
    Write-Host "UPX applies to gol.exe only; pass -exe with -upx." -ForegroundColor DarkGray
}

if ($buildTuiExe) {
    Invoke-GolPyInstaller -ExeName 'gol-tui' -EntryScript 'gol_tui.py' -WorkSubdir 'pyinstaller-tui' `
        -HiddenImports @('textual') -CollectSubmodules @('textual') -ExcludeModules @('pygame', 'pygame_ce')
    Write-Host "  TUI executable (textual only; no pygame)" -ForegroundColor DarkGray
}
