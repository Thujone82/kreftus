<#
.SYNOPSIS
  Master build menu for Go projects.

.DESCRIPTION
  Enumerates subfolders that contain a 'build.ps1' and shows an interactive
  menu. Press the number for each project to toggle selection. Press 'U' to
  toggle UPX compression (adds '-upx' when invoking child builds). Press 'A'
  to select all, 'N' to select none. Press Enter to run selected builds one by
  one, streaming their output. Displays a wrap-up with success/failure counts.
#>

$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

function Get-Projects {
    $dirs = Get-ChildItem -Directory | Sort-Object Name
    $projects = @()
    $index = 1
    foreach ($d in $dirs) {
        $buildPath = Join-Path $d.FullName 'build.ps1'
        if (Test-Path $buildPath) {
            $projects += [pscustomobject]@{
                Index = $index
                Name  = $d.Name
                Path  = $d.FullName
            }
            $index++
        }
    }
    return $projects
}

function Get-ConsoleWidth {
    try { return [Console]::WindowWidth } catch { return 80 }
}

function Write-PaddedLine {
    param(
        [string]$Text,
        [ConsoleColor]$Foreground = [Console]::ForegroundColor,
        [ConsoleColor]$Background = [Console]::BackgroundColor
    )
    $width = Get-ConsoleWidth
    $pad = if ($Text.Length -lt $width) { ' ' * ($width - $Text.Length) } else { '' }
    $oldFg = [Console]::ForegroundColor
    $oldBg = [Console]::BackgroundColor
    try {
        [Console]::ForegroundColor = $Foreground
        [Console]::BackgroundColor = $Background
        Write-Host ($Text + $pad)
    } finally {
        [Console]::ForegroundColor = $oldFg
        [Console]::BackgroundColor = $oldBg
    }
}

function Show-Menu {
    param(
        [array]$Projects,
        [hashtable]$SelectionMap,
        [bool]$UpxEnabled
    )
    Clear-Host
    $title = "*** Go Projects Master Build ***"
    Write-PaddedLine $title ([ConsoleColor]::Yellow)
    Write-Host ""
    $upxText = $UpxEnabled ? 'ON' : 'OFF'
    $upxColor = $UpxEnabled ? 'Green' : 'DarkGray'
    Write-Host ("UPX Compression: " + $upxText) -ForegroundColor $upxColor
    Write-Host ""
    foreach ($p in $Projects) {
        $prefix = " [$($p.Index)] $($p.Name)"
        if ($SelectionMap.ContainsKey($p.Index) -and $SelectionMap[$p.Index]) {
            Write-PaddedLine $prefix ([ConsoleColor]::Black) ([ConsoleColor]::DarkGreen)
        } else {
            Write-Host $prefix -ForegroundColor White
        }
    }
    Write-Host ""
    # Sleek control bar with bracketed hotkeys, matching bmon style
    $range = "1-$($Projects.Count)"
    Write-Host -NoNewline "Toggle Project[" -ForegroundColor White
    Write-Host -NoNewline $range -ForegroundColor Cyan
    Write-Host -NoNewline "] " -ForegroundColor White
    Write-Host -NoNewline "UPX[" -ForegroundColor White
    Write-Host -NoNewline "U" -ForegroundColor Cyan
    Write-Host -NoNewline "] " -ForegroundColor White
    Write-Host -NoNewline "All[" -ForegroundColor White
    Write-Host -NoNewline "A" -ForegroundColor Cyan
    Write-Host -NoNewline "] " -ForegroundColor White
    Write-Host -NoNewline "None[" -ForegroundColor White
    Write-Host -NoNewline "N" -ForegroundColor Cyan
    Write-Host -NoNewline "] " -ForegroundColor White
    Write-Host -NoNewline "Build[" -ForegroundColor Green
    Write-Host -NoNewline "Enter" -ForegroundColor Cyan
    Write-Host -NoNewline "] " -ForegroundColor Green
    Write-Host -NoNewline "Exit[" -ForegroundColor DarkYellow
    Write-Host -NoNewline "Esc" -ForegroundColor Cyan
    Write-Host "]" -ForegroundColor DarkYellow
}

function Invoke-ProjectBuild {
    param(
        [string]$ProjectPath,
        [string]$ProjectName,
        [bool]$UpxEnabled
    )
    Write-Host ""; Write-Host ("─" * (Get-ConsoleWidth)) -ForegroundColor DarkGray
    Write-Host ("Building " + $ProjectName + " …") -ForegroundColor Yellow
    Push-Location $ProjectPath
    try {
        $argsList = @('-NoProfile','-ExecutionPolicy','Bypass','-File','build.ps1')
        if ($UpxEnabled) { $argsList += '-upx' }
        Write-Host ("Executing: pwsh " + ($argsList -join ' ')) -ForegroundColor DarkGray
        & pwsh @argsList
        $exitCode = $LASTEXITCODE
        if ($exitCode -eq 0) {
            Write-Host ("✔ Completed: " + $ProjectName) -ForegroundColor Green
            return $true
        } else {
            Write-Host ("✖ Failed: " + $ProjectName + " (exit $exitCode)") -ForegroundColor Red
            return $false
        }
    } catch {
        Write-Host ("✖ Failed: " + $ProjectName + " - " + $_.Exception.Message) -ForegroundColor Red
        return $false
    } finally {
        Pop-Location
    }
}

$projects = Get-Projects
if ($projects.Count -eq 0) {
    Write-Host "No subfolders with build.ps1 found." -ForegroundColor Red
    exit 1
}

$selection = @{}
$projects | ForEach-Object { $selection[$_.Index] = $false }
$upxEnabled = $false

Show-Menu -Projects $projects -SelectionMap $selection -UpxEnabled $upxEnabled

while ($true) {
    $key = [Console]::ReadKey($true)
    if ($key.Key -eq 'Enter') { break }
    if ($key.Key -eq 'Escape') { Write-Host "Exiting."; exit }

    $ch = $key.KeyChar
    switch -regex ($ch) {
        '^[1-9]$' {
            $idx = [int]$ch - 48
            if ($idx -ge 1 -and $idx -le $projects.Count) {
                $selection[$idx] = -not $selection[$idx]
            }
        }
        '^[aA]$' { $projects | ForEach-Object { $selection[$_.Index] = $true } }
        '^[nN]$' { $projects | ForEach-Object { $selection[$_.Index] = $false } }
        '^[uU]$' { $upxEnabled = -not $upxEnabled }
    }
    Show-Menu -Projects $projects -SelectionMap $selection -UpxEnabled $upxEnabled
}

$chosen = $projects | Where-Object { $selection[$_.Index] }
if ($chosen.Count -eq 0) {
    Write-Host "No projects selected. Nothing to build." -ForegroundColor Yellow
    exit 0
}

Write-Host ""; $upxState = $upxEnabled ? 'ON' : 'OFF'; Write-Host ("Starting builds (UPX=" + $upxState + ")…") -ForegroundColor Cyan

$successCount = 0
$failed = @()
foreach ($p in $chosen) {
    if (Invoke-ProjectBuild -ProjectPath $p.Path -ProjectName $p.Name -UpxEnabled $upxEnabled) {
        $successCount++
    } else {
        $failed += $p.Name
    }
}

Write-Host ""; Write-Host ("─" * (Get-ConsoleWidth)) -ForegroundColor DarkGray
Write-Host ("Builds completed: $successCount of $($chosen.Count) succeeded.") -ForegroundColor ($(if ($failed.Count -eq 0) { 'Green' } else { 'Yellow' }))
if ($failed.Count -gt 0) {
    Write-Host ("Failed: " + ($failed -join ', ')) -ForegroundColor Red
}


