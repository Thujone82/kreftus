<#
.SYNOPSIS
    Rebuilds the self-contained Trimps Surky Perk Optimizer (surky.html).
.DESCRIPTION
    Authoring & build pipeline utility for surky.html. Optionally fetches the latest upstream
    nsheetz/perks index.html snapshot (fetch_upstream.py), validates upstream & overlay JavaScript
    syntax via node --check, and bundles upstream/index.html + head-inject.html + body-inject.html
    + theme.css + overlay.js into a single self-contained surky.html file with CRLF line endings.
    Portable: surky.html + surky-build/ are completely self-contained and require no parent repo files.
.PARAMETER Fresh
    Downloads the latest upstream index.html and commit metadata from github.com/nsheetz/perks before rebuilding.
.PARAMETER Ref
    Git branch or commit SHA to fetch when -Fresh is used (defaults to 'master').
.PARAMETER AssembleOnly
    Skips upstream snapshot checks and immediately validates JS and assembles surky.html from local files.
.PARAMETER OutFile
    Custom output path for the assembled HTML file (defaults to ..\surky.html relative to surky-build/).
.PARAMETER Help
    Displays this documentation and usage guide.
.EXAMPLE
    .\refresh.ps1
    Validates JS syntax and assembles ..\surky.html from the committed upstream snapshot and local overlays.
.EXAMPLE
    .\refresh.ps1 -Fresh
    Downloads the latest upstream nsheetz/perks index.html, reports any perk/preset diffs, and rebuilds surky.html.
.EXAMPLE
    .\refresh.ps1 -AssembleOnly
    Fast rebuild of surky.html after editing theme.css or overlay.js.
.EXAMPLE
    .\refresh.ps1 -OutFile C:\inetpub\wwwroot\surky.html
    Builds surky.html directly to a custom hosting path.
.EXAMPLE
    .\refresh.ps1 -Help
    Displays this help message.
#>
[CmdletBinding()]
param(
    [switch]$Fresh,
    [string]$Ref = 'master',
    [switch]$AssembleOnly,
    [string]$OutFile = '',
    [switch]$Help
)

# ==============================================================================
# Script Name:  refresh.ps1
# Description:  Surky Perk Optimizer (surky.html) Snapshot & Overlay Pipeline
# ==============================================================================

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$BoxWidth = 68

$Here          = $PSScriptRoot
$ParentDir     = Split-Path -Parent $Here
$SnapFile      = Join-Path (Join-Path $Here 'upstream') 'index.html'
$MetaFile      = Join-Path $Here 'upstream.json'
$ThemeFile     = Join-Path $Here 'theme.css'
$OverlayFile   = Join-Path $Here 'overlay.js'
$HeadInjFile   = Join-Path $Here 'head-inject.html'
$BodyInjFile   = Join-Path $Here 'body-inject.html'
$TempUpstreamJs = Join-Path $Here '_upstream-main.js'
$TargetOutFile = if ($OutFile) { [System.IO.Path]::GetFullPath($OutFile) } else { Join-Path $ParentDir 'surky.html' }

# --- UI Helper Functions (matching sa-build/refresh.ps1 visual style) ---
function Write-BoxHeader {
    param(
        [string]$Title,
        [string]$TitleColor = 'White',
        [string]$BorderColor = 'Cyan',
        [int]$Width = $BoxWidth
    )
    $inner = $Width - 2
    $pad = $inner - $Title.Length
    if ($pad -lt 0) { $pad = 0 }
    $leftPad = [Math]::Floor($pad / 2)
    $rightPad = [Math]::Ceiling($pad / 2)
    Write-Host "║" -NoNewline -ForegroundColor $BorderColor
    Write-Host (" " * $leftPad) -NoNewline
    Write-Host $Title -NoNewline -ForegroundColor $TitleColor
    Write-Host (" " * $rightPad) -NoNewline
    Write-Host "║" -ForegroundColor $BorderColor
}

function Write-BoxRow {
    param(
        [string]$Label,
        [string]$Value,
        [string]$LabelColor = 'DarkCyan',
        [string]$ValueColor = 'Gray',
        [string]$BorderColor = 'Cyan',
        [int]$Width = $BoxWidth
    )
    $contentWidth = $Width - 4
    if (($Label.Length + $Value.Length) -gt $contentWidth) {
        $avail = $contentWidth - $Label.Length
        if ($avail -gt 3) { $Value = $Value.Substring(0, $avail - 3) + '...' }
    }
    $pad = $contentWidth - $Label.Length - $Value.Length
    if ($pad -lt 0) { $pad = 0 }
    Write-Host "║ " -NoNewline -ForegroundColor $BorderColor
    Write-Host $Label -NoNewline -ForegroundColor $LabelColor
    Write-Host $Value -NoNewline -ForegroundColor $ValueColor
    Write-Host ((' ' * $pad) + ' ║') -ForegroundColor $BorderColor
}

function Write-BoxSection {
    param(
        [string]$Title,
        [string]$BorderColor = 'Green',
        [string]$TitleColor = 'White',
        [int]$Width = $BoxWidth
    )
    $tag = "─── [ $Title ] "
    $rem = ($Width - 2) - $tag.Length
    if ($rem -lt 0) { $rem = 0 }
    $rightBar = "─" * $rem
    Write-Host "╟" -NoNewline -ForegroundColor $BorderColor
    Write-Host "─── [ " -NoNewline -ForegroundColor $BorderColor
    Write-Host $Title -NoNewline -ForegroundColor $TitleColor
    Write-Host " ] $rightBar" -NoNewline -ForegroundColor $BorderColor
    Write-Host "╢" -ForegroundColor $BorderColor
}

function Format-ByteSize {
    param([long]$Bytes)
    $abs = [Math]::Abs($Bytes)
    if ($abs -ge 1MB) { return "{0:N2} MB ({1:N0} B)" -f ($Bytes / 1MB), $Bytes }
    if ($abs -ge 1KB) { return "{0:N1} KB ({1:N0} B)" -f ($Bytes / 1KB), $Bytes }
    return "{0:N0} B" -f $Bytes
}

function Format-SignedDelta {
    param([long]$Delta, [string]$Unit = 'B')
    if ($Delta -gt 0) { return "+{0:N0} {1}" -f $Delta, $Unit }
    if ($Delta -lt 0) { return "{0:N0} {1}" -f $Delta, $Unit }
    return "0 $Unit"
}

function Get-FileSnapshot {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        return @{ Exists = $false; Size = 0L; Hash = 'none'; Lines = 0; Raw = '' }
    }
    $item = Get-Item -LiteralPath $Path
    $hash = (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.Substring(0, 8).ToLower()
    $raw  = [System.IO.File]::ReadAllText($Path, [System.Text.Encoding]::UTF8)
    $lines = ([regex]::Matches($raw, "`n")).Count + 1
    return @{
        Exists = $true
        Size   = [long]$item.Length
        Hash   = $hash
        Lines  = $lines
        Raw    = $raw
    }
}

function Get-SurkyFeatures {
    param([string]$HtmlRaw)
    $perks   = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
    $presets = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
    $auxKeys = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
    if (-not $HtmlRaw) {
        return @{ Perks = $perks; Presets = $presets; AuxKeys = $auxKeys }
    }
    foreach ($m in [regex]::Matches($HtmlRaw, 'id="input-([a-z0-9-]+)-enabled"')) {
        $name = $m.Groups[1].Value
        [void]$perks.Add($name)
    }
    foreach ($m in [regex]::Matches($HtmlRaw, '<option\s+value="([^"]+)"')) {
        [void]$presets.Add($m.Groups[1].Value)
    }
    foreach ($m in [regex]::Matches($HtmlRaw, 'keyName:\s*"([^"]+)"')) {
        [void]$auxKeys.Add($m.Groups[1].Value)
    }
    return @{ Perks = $perks; Presets = $presets; AuxKeys = $auxKeys }
}

function Compare-SurkyUpstream {
    param([string]$OldRaw, [string]$NewRaw)
    $oldF = Get-SurkyFeatures -HtmlRaw $OldRaw
    $newF = Get-SurkyFeatures -HtmlRaw $NewRaw
    $diffs = [System.Collections.Generic.List[string]]::new()
    if ($OldRaw -and ($OldRaw -ne $NewRaw)) {
        foreach ($p in $newF.Perks)   { if (-not $oldF.Perks.Contains($p))   { $diffs.Add("+perk:$p") } }
        foreach ($p in $oldF.Perks)   { if (-not $newF.Perks.Contains($p))   { $diffs.Add("-perk:$p") } }
        foreach ($p in $newF.Presets) { if (-not $oldF.Presets.Contains($p)) { $diffs.Add("+preset:$p") } }
        foreach ($p in $oldF.Presets) { if (-not $newF.Presets.Contains($p)) { $diffs.Add("-preset:$p") } }
        foreach ($a in $newF.AuxKeys) { if (-not $oldF.AuxKeys.Contains($a)) { $diffs.Add("+aux:$a") } }
        foreach ($a in $oldF.AuxKeys) { if (-not $newF.AuxKeys.Contains($a)) { $diffs.Add("-aux:$a") } }
    }
    return @{
        Changed     = ($OldRaw -ne $NewRaw)
        PerkCount   = $newF.Perks.Count
        PresetCount = $newF.Presets.Count
        AuxCount    = $newF.AuxKeys.Count
        Diffs       = $diffs
    }
}

function Write-StreamLine {
    param(
        [string]$Text,
        [string]$Color = 'White'
    )
    Write-Host "│ " -NoNewline -ForegroundColor DarkCyan
    Write-Host $Text -ForegroundColor $Color
}

# --- Explicit Help Intercept ---
if ($Help) {
    Get-Help $PSCommandPath -Full
    return
}

# --- Action & Snapshot Context ---
$NeedFetch = $Fresh -or (-not (Test-Path -LiteralPath $SnapFile))
$ActionTitle = if ($AssembleOnly) {
    "Assemble Only (Fast Overlay Rebuild)"
} elseif ($Fresh) {
    "Fresh Upstream Download & Full Rebuild"
} else {
    "Cached Upstream Snapshot & Full Rebuild"
}

$UpstreamInfo = "Not present (will download from github.com/nsheetz/perks)"
if (Test-Path -LiteralPath $MetaFile) {
    try {
        $metaObj = Get-Content -LiteralPath $MetaFile -Raw -Encoding UTF8 | ConvertFrom-Json
        $sha7 = if ($metaObj.commit) { $metaObj.commit.Substring(0, [Math]::Min(7, $metaObj.commit.Length)) } else { "unknown" }
        $cDate = if ($metaObj.commitDate -is [DateTime]) {
            $metaObj.commitDate.ToString("yyyy-MM-dd")
        } elseif ($metaObj.commitDate -and $metaObj.commitDate.Length -ge 10) {
            $metaObj.commitDate.Substring(0, 10)
        } else {
            "$($metaObj.commitDate)"
        }
        $UpstreamInfo = "$($metaObj.repo)@$sha7 ($cDate)"
    } catch {}
}

$SnapStatus = if (Test-Path -LiteralPath $SnapFile) {
    $si = Get-Item -LiteralPath $SnapFile
    "{0:N1} KB (modified {1})" -f ($si.Length / 1KB), $si.LastWriteTime.ToString("yyyy-MM-dd HH:mm")
} else {
    "Missing (will fetch upstream/index.html)"
}

# --- Top Banner ---
$Bar = '═' * ($BoxWidth - 2)
Write-Host ""
Write-Host "╔$Bar╗" -ForegroundColor Cyan
Write-BoxHeader "TRIMPS SURKY PERK OPTIMIZER BUILD PIPELINE" "White" "Cyan" $BoxWidth
Write-Host "╠$Bar╣" -ForegroundColor Cyan
Write-BoxRow "Action   :" " $ActionTitle" "DarkCyan" "Yellow" "Cyan" $BoxWidth
Write-BoxRow "Upstream :" " $UpstreamInfo" "DarkCyan" "Gray" "Cyan" $BoxWidth
Write-BoxRow "Snapshot :" " $SnapStatus" "DarkCyan" "Gray" "Cyan" $BoxWidth
Write-BoxRow "Target   :" " $TargetOutFile" "DarkCyan" "Gray" "Cyan" $BoxWidth
Write-Host "╚$Bar╝" -ForegroundColor Cyan
Write-Host ""

$Stopwatch = [System.Diagnostics.Stopwatch]::StartNew()

try {
    if (-not (Get-Command python -ErrorAction SilentlyContinue)) {
        throw "Python executable not found on PATH."
    }

    # Pre-build snapshots for diff & change analysis
    $PreSnap = Get-FileSnapshot -Path $SnapFile
    $PreHtml = Get-FileSnapshot -Path $TargetOutFile

    # Live stream frame
    $StreamTag = "─── [ LIVE BUILD STREAM ] "
    $StreamBar = "─" * ($BoxWidth - 2 - $StreamTag.Length)
    Write-Host "┌${StreamTag}${StreamBar}┐" -ForegroundColor DarkCyan

    # Step 1: Upstream Snapshot
    if ($AssembleOnly -and (Test-Path -LiteralPath $SnapFile)) {
        Write-StreamLine "[1/3] Upstream : Skipped (-AssembleOnly active, reusing upstream/index.html)" "DarkGray"
    } elseif ($NeedFetch) {
        Write-StreamLine "[1/3] Upstream : Fetching nsheetz/perks index.html (fetch_upstream.py)..." "Yellow"
        $fetchArgs = @('-u', (Join-Path $Here 'fetch_upstream.py'), '--ref', $Ref)
        & python @fetchArgs 2>&1 | ForEach-Object {
            $line = $_.ToString()
            $color = if ($line -match '^Snapshot\s*:') { 'Green' } elseif ($line -match '^(Upstream|Commit)\s*:') { 'Gray' } else { 'White' }
            Write-StreamLine "      $line" $color
        }
        if ($LASTEXITCODE -ne 0) {
            throw "fetch_upstream.py failed with exit code $LASTEXITCODE"
        }
    } else {
        Write-StreamLine "[1/3] Upstream : Using committed snapshot ($UpstreamInfo)" "Gray"
    }

    # Step 2: JavaScript Syntax Validation
    Write-StreamLine "[2/3] Validate : Checking upstream inline JS & overlay.js syntax..." "Yellow"
    if (Get-Command node -ErrorAction SilentlyContinue) {
        try {
            & python (Join-Path $Here 'assemble.py') --extract-upstream-js $TempUpstreamJs 2>&1 | ForEach-Object {
                Write-StreamLine "      $($_.ToString())" "Red"
            }
            if ($LASTEXITCODE -ne 0) {
                throw "Failed to extract upstream inline JS for validation (exit code $LASTEXITCODE)"
            }
            & node --check $TempUpstreamJs 2>&1 | ForEach-Object {
                Write-StreamLine "      $($_.ToString())" "Red"
            }
            if ($LASTEXITCODE -ne 0) {
                throw "JavaScript syntax check failed in upstream inline script (exit code $LASTEXITCODE)"
            }
            & node --check $OverlayFile 2>&1 | ForEach-Object {
                Write-StreamLine "      $($_.ToString())" "Red"
            }
            if ($LASTEXITCODE -ne 0) {
                throw "JavaScript syntax check failed in overlay.js (exit code $LASTEXITCODE)"
            }
            Write-StreamLine "      Syntax   : OK (upstream JS & overlay.js verified via node --check)" "Green"
        } finally {
            if (Test-Path -LiteralPath $TempUpstreamJs) {
                Remove-Item -LiteralPath $TempUpstreamJs -Force -ErrorAction SilentlyContinue
            }
        }
    } else {
        Write-StreamLine "      Syntax   : Skipped (node not found on PATH)" "DarkGray"
    }

    # Step 3: Self-Contained HTML Assembly
    $outLeaf = Split-Path -Leaf $TargetOutFile
    Write-StreamLine "[3/3] Assemble : Bundling self-contained $outLeaf (assemble.py)..." "Yellow"
    & python (Join-Path $Here 'assemble.py') --out $TargetOutFile 2>&1 | ForEach-Object {
        $line = $_.ToString()
        $color = if ($line -match '^Assemble\s*:') {
            'Green'
        } elseif ($line -match '^(Inputs|Anchors|Strip)\s*:') {
            'Gray'
        } elseif ($line -match '^Warning\s*:') {
            'Yellow'
        } else {
            'White'
        }
        Write-StreamLine "      $line" $color
    }
    if ($LASTEXITCODE -ne 0) {
        throw "assemble.py failed with exit code $LASTEXITCODE"
    }

    Write-Host ("└" + ("─" * ($BoxWidth - 2)) + "┘") -ForegroundColor DarkCyan
    Write-Host ""

    $Stopwatch.Stop()
    $ElapsedStr = "{0:N2}s" -f $Stopwatch.Elapsed.TotalSeconds

    # --- Post-build Snapshots & Diff Computation ---
    $PostSnap = Get-FileSnapshot -Path $SnapFile
    $PostHtml = Get-FileSnapshot -Path $TargetOutFile
    $UpDiff   = Compare-SurkyUpstream -OldRaw $PreSnap.Raw -NewRaw $PostSnap.Raw

    $BareLfCount = ([regex]::Matches($PostHtml.Raw, "(?<!`r)`n")).Count
    $EolStatus   = if ($BareLfCount -eq 0) { "CRLF verified" } else { "Mixed ($BareLfCount bare LF)" }

    $ThemeSize   = (Get-Item -LiteralPath $ThemeFile).Length
    $OverlaySize = (Get-Item -LiteralPath $OverlayFile).Length
    $InjectSize  = (Get-Item -LiteralPath $HeadInjFile).Length + (Get-Item -LiteralPath $BodyInjFile).Length

    $HtmlChanged   = ($PreHtml.Hash -ne $PostHtml.Hash)
    $HtmlSizeDelta = $PostHtml.Size - $PreHtml.Size
    $HtmlLineDelta = $PostHtml.Lines - $PreHtml.Lines
    $SnapChanged   = ($PreSnap.Hash -ne $PostSnap.Hash)
    $SnapSizeDelta = $PostSnap.Size - $PreSnap.Size

    $SyncedCommit = "Unknown"
    if (Test-Path -LiteralPath $MetaFile) {
        try {
            $m = Get-Content -LiteralPath $MetaFile -Raw -Encoding UTF8 | ConvertFrom-Json
            $s7 = if ($m.commit) { $m.commit.Substring(0, [Math]::Min(7, $m.commit.Length)) } else { "unknown" }
            $mDate = if ($m.commitDate -is [DateTime]) { $m.commitDate.ToString("yyyy-MM-dd") } elseif ($m.commitDate -and $m.commitDate.Length -ge 10) { $m.commitDate.Substring(0, 10) } else { "$($m.commitDate)" }
            $SyncedCommit = "$($m.repo)@$s7 · $mDate ($($m.commitMessage))"
        } catch {}
    }

    # --- Completion & Wrap-Up Stats Box ---
    $CompleteBar = '═' * ($BoxWidth - 2)
    Write-Host "╔$CompleteBar╗" -ForegroundColor Green
    Write-BoxHeader "✔  BUILD & SNAPSHOT COMPLETE" "White" "Green" $BoxWidth
    Write-Host "╠$CompleteBar╣" -ForegroundColor Green

    $OverallState = if ($HtmlChanged -or $SnapChanged) {
        "Completed with updates in $ElapsedStr"
    } else {
        "Completed in $ElapsedStr (in sync, 0 B delta)"
    }
    Write-BoxRow "Status   :" " $OverallState" "DarkGreen" "Green" "Green" $BoxWidth
    Write-BoxRow "Output   :" " $(Format-ByteSize $PostHtml.Size) · $($PostHtml.Lines) lines · $EolStatus" "DarkGreen" "White" "Green" $BoxWidth
    Write-BoxRow "Payload  :" (" upstream {0:N1}KB | theme {1:N1}KB | overlay {2:N1}KB | html {3:N1}KB" -f ($PostSnap.Size/1KB), ($ThemeSize/1KB), ($OverlaySize/1KB), ($InjectSize/1KB)) "DarkGreen" "Gray" "Green" $BoxWidth
    Write-BoxRow "Dataset  :" " $($UpDiff.PerkCount) U2 perks · $($UpDiff.PresetCount) presets · $($UpDiff.AuxCount) aux fields" "DarkGreen" "Gray" "Green" $BoxWidth
    Write-BoxRow "Upstream :" " $SyncedCommit" "DarkGreen" "White" "Green" $BoxWidth

    # Section: Changes Made During This Run
    Write-BoxSection "CHANGES MADE (THIS RUN)" "Green" "White" $BoxWidth
    if ($SnapChanged) {
        $sDeltaStr = Format-SignedDelta $SnapSizeDelta 'B'
        Write-BoxRow "Snapshot :" " Updated ($sDeltaStr, sha $($PreSnap.Hash) -> $($PostSnap.Hash))" "DarkGreen" "Yellow" "Green" $BoxWidth
        if ($UpDiff.Diffs.Count -gt 0) {
            Write-BoxRow "Diffs    :" " $($UpDiff.Diffs -join ', ')" "DarkGreen" "Cyan" "Green" $BoxWidth
        }
    } else {
        Write-BoxRow "Snapshot :" " Unchanged (sha $($PostSnap.Hash))" "DarkGreen" "Gray" "Green" $BoxWidth
    }

    $outLabel = ("{0,-9}:" -f $outLeaf)
    if ($HtmlChanged) {
        $hBytesStr = Format-SignedDelta $HtmlSizeDelta 'B'
        $hLinesStr = Format-SignedDelta $HtmlLineDelta 'lines'
        Write-BoxRow $outLabel " Rebuilt ($hBytesStr, $hLinesStr, sha $($PreHtml.Hash) -> $($PostHtml.Hash))" "DarkGreen" "Yellow" "Green" $BoxWidth
    } else {
        Write-BoxRow $outLabel " Unchanged (sha $($PostHtml.Hash) matches prior build)" "DarkGreen" "Gray" "Green" $BoxWidth
    }

    # Section: Git Working Tree Status vs HEAD (only if inside a git repository)
    if (Get-Command git -ErrorAction SilentlyContinue) {
        $gitTop = (& git -C $Here rev-parse --show-toplevel 2>$null)
        if ($LASTEXITCODE -eq 0 -and $gitTop) {
            $gitRoot = [System.IO.Path]::GetFullPath($gitTop.Trim())
            $gitRows = @(& git -C $gitRoot status --porcelain -- $TargetOutFile $Here 2>$null)
            $numStat = @{}
            & git -C $gitRoot diff --numstat HEAD -- $TargetOutFile $Here 2>$null | ForEach-Object {
                $parts = $_ -split '\s+'
                if ($parts.Count -ge 3) {
                    $numStat[$parts[2]] = "+$($parts[0])/-$($parts[1]) lines"
                }
            }

            Write-BoxSection "GIT WORKING TREE (VS HEAD)" "Green" "White" $BoxWidth
            if ($gitRows.Count -eq 0) {
                Write-BoxRow "Repo Sync    :" " Clean (all Surky files match HEAD commit)" "DarkGreen" "Gray" "Green" $BoxWidth
            } else {
                foreach ($row in $gitRows) {
                    $st = $row.Substring(0, 2).Trim()
                    $relPath = $row.Substring(3).Trim()
                    $shortName = Split-Path -Leaf $relPath
                    $tag = switch ($st) {
                        'M'  { 'Modified' }
                        'A'  { 'Added' }
                        '??' { 'Untracked' }
                        default { $st }
                    }
                    $statInfo = if ($numStat.ContainsKey($relPath)) { " ($($numStat[$relPath]))" } else { "" }
                    $valColor = if ($st -eq '??') { 'DarkYellow' } else { 'Yellow' }
                    Write-BoxRow ("{0,-13}:" -f $shortName) " $tag$statInfo" "DarkGreen" $valColor "Green" $BoxWidth
                }
            }
        }
    }

    Write-Host "╚$CompleteBar╝" -ForegroundColor Green
    Write-Host ""

} catch {
    $Stopwatch.Stop()
    $ErrorBar = '═' * ($BoxWidth - 2)
    Write-Host ""
    Write-Host "╔$ErrorBar╗" -ForegroundColor Red
    Write-BoxHeader "✖  EXECUTION ERROR ENCOUNTERED" "White" "Red" $BoxWidth
    Write-Host "╠$ErrorBar╣" -ForegroundColor Red
    Write-BoxRow "Action :" " $ActionTitle" "DarkRed" "Yellow" "Red" $BoxWidth
    Write-BoxRow "Details:" " $($_.ToString())" "DarkRed" "Red" "Red" $BoxWidth
    Write-Host "╚$ErrorBar╝" -ForegroundColor Red
    Write-Host ""
    exit 1
}
