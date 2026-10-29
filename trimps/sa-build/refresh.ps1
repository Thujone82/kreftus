<#
.SYNOPSIS
    Rebuilds the self-contained Trimps Spire Assault planner (trimps/sa.html).
.DESCRIPTION
    Authoring & build pipeline utility for trimps/sa.html. Extracts community Spire Assault
    builds from the Google Sheets workbook (_sa.xlsx) and item definitions from objects.js
    into data.json, validates app.js syntax, and bundles shell.html + lz-string.js + data.json
    + app.js into a single self-contained trimps/sa.html file with CRLF line endings.
.PARAMETER Fresh
    Forces a fresh download of the community Spire Assault Google Sheet (_sa.xlsx) before rebuilding.
.PARAMETER AssembleOnly
    Skips running build_data.py and immediately validates app.js and assembles sa.html from existing data.json.
.PARAMETER CleanCache
    Removes the cached _sa.xlsx workbook after the build completes so the working tree stays clean.
.PARAMETER VerboseSheets
    Streams per-level SA build counts in the live build stream.
.PARAMETER Help
    Displays this documentation and usage guide.
.EXAMPLE
    .\refresh.ps1
    Rebuilds data.json (using cached _sa.xlsx if present) and assembles trimps/sa.html.
.EXAMPLE
    .\refresh.ps1 -Fresh
    Re-downloads the community spreadsheet from Google Docs, updates data.json, and rebuilds sa.html.
.EXAMPLE
    .\refresh.ps1 -AssembleOnly
    Fast rebuild of sa.html after editing shell.html or app.js without re-parsing the spreadsheet.
.EXAMPLE
    .\refresh.ps1 -Fresh -CleanCache
    Downloads fresh sheet data, rebuilds sa.html, and deletes the temporary _sa.xlsx cache file.
.EXAMPLE
    .\refresh.ps1 -Help
    Displays this help message.
#>
[CmdletBinding()]
param(
    [switch]$Fresh,
    [switch]$AssembleOnly,
    [switch]$CleanCache,
    [switch]$VerboseSheets,
    [switch]$Help
)

# ==============================================================================
# Script Name:  refresh.ps1
# Description:  Spire Assault Planner (trimps/sa.html) Snapshot & Build Pipeline
# ==============================================================================

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$BoxWidth = 68

$Here       = $PSScriptRoot
$TrimpsDir  = Split-Path -Parent $Here
$RepoRoot   = Split-Path -Parent $TrimpsDir
$CacheFile  = Join-Path $Here '_sa.xlsx'
$DataFile   = Join-Path $Here 'data.json'
$ShellFile  = Join-Path $Here 'shell.html'
$AppFile    = Join-Path $Here 'app.js'
$LzFile     = Join-Path $TrimpsDir 'lz-string.js'
$ObjFile    = Join-Path $TrimpsDir 'objects.js'
$OutFile    = Join-Path $TrimpsDir 'sa.html'
$SheetId    = '17Z3dwnkeAmY2La-LWreybTs4Sm7BAck79EzVF_gkZzs'

# --- UI Helper Functions (kreftus.ps1 visual style) ---
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
        return @{ Exists = $false; Size = 0L; Hash = ''; Lines = 0 }
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

function Compare-SaDataJson {
    param([string]$OldRaw, [string]$NewRaw)
    $result = @{
        Changed       = ($OldRaw -ne $NewRaw)
        LevelCount    = 0
        BuildCount    = 0
        ItemCount     = 0
        AddedBuilds   = [System.Collections.Generic.List[string]]::new()
        RemovedBuilds = [System.Collections.Generic.List[string]]::new()
        UpdatedBuilds = [System.Collections.Generic.List[string]]::new()
        ItemDiffs     = [System.Collections.Generic.List[string]]::new()
    }
    if (-not $NewRaw) { return $result }
    $newObj = $NewRaw | ConvertFrom-Json
    $newItems = @($newObj.items.PSObject.Properties)
    $newLevels = @($newObj.builds.PSObject.Properties)
    $result.ItemCount = $newItems.Count
    $result.LevelCount = $newLevels.Count
    foreach ($prop in $newLevels) {
        $result.BuildCount += @($prop.Value).Count
    }

    if (-not $OldRaw -or -not $result.Changed) { return $result }

    $oldObj = $OldRaw | ConvertFrom-Json
    $oldItemMap = @{}
    foreach ($p in $oldObj.items.PSObject.Properties) {
        $oldItemMap[$p.Name] = ($p.Value | ConvertTo-Json -Compress -Depth 6)
    }
    foreach ($p in $newItems) {
        $newJson = ($p.Value | ConvertTo-Json -Compress -Depth 6)
        if (-not $oldItemMap.ContainsKey($p.Name)) {
            $result.ItemDiffs.Add("+$($p.Name)")
        } elseif ($oldItemMap[$p.Name] -ne $newJson) {
            $result.ItemDiffs.Add("~$($p.Name)")
        }
    }
    foreach ($k in $oldItemMap.Keys) {
        if (-not $newObj.items.PSObject.Properties[$k]) {
            $result.ItemDiffs.Add("-$k")
        }
    }

    $allLevels = [System.Collections.Generic.HashSet[string]]::new()
    foreach ($p in $oldObj.builds.PSObject.Properties) { [void]$allLevels.Add($p.Name) }
    foreach ($p in $newLevels) { [void]$allLevels.Add($p.Name) }

    $sortedLevels = $allLevels | Sort-Object { [int]$_ }
    foreach ($lvl in $sortedLevels) {
        $oldArr = if ($oldObj.builds.PSObject.Properties[$lvl]) { @($oldObj.builds.$lvl) } else { @() }
        $newArr = if ($newObj.builds.PSObject.Properties[$lvl]) { @($newObj.builds.$lvl) } else { @() }

        $oldByName = @{}
        for ($i = 0; $i -lt $oldArr.Count; $i++) {
            $key = "$($oldArr[$i].n)#$i"
            $oldByName[$key] = ($oldArr[$i] | ConvertTo-Json -Compress -Depth 6)
        }
        $newByName = @{}
        for ($i = 0; $i -lt $newArr.Count; $i++) {
            $key = "$($newArr[$i].n)#$i"
            $newByName[$key] = ($newArr[$i] | ConvertTo-Json -Compress -Depth 6)
        }

        foreach ($k in $newByName.Keys) {
            $bName = ($k -split '#')[0]
            if (-not $oldByName.ContainsKey($k)) {
                $result.AddedBuilds.Add("SA ${lvl}: +'$bName'")
            } elseif ($oldByName[$k] -ne $newByName[$k]) {
                $result.UpdatedBuilds.Add("SA ${lvl}: ~'$bName'")
            }
        }
        foreach ($k in $oldByName.Keys) {
            $bName = ($k -split '#')[0]
            if (-not $newByName.ContainsKey($k)) {
                $result.RemovedBuilds.Add("SA ${lvl}: -'$bName'")
            }
        }
    }
    return $result
}

$script:LastWasProgress = $false

function Write-StreamLine {
    param(
        [string]$Text,
        [string]$Color = 'White'
    )
    if ($script:LastWasProgress) {
        Write-Host ("`r" + (" " * ($BoxWidth - 2)) + "`r") -NoNewline
        $script:LastWasProgress = $false
    }
    Write-Host "│ " -NoNewline -ForegroundColor DarkCyan
    Write-Host $Text -ForegroundColor $Color
}

function Write-StreamProgress {
    param(
        [string]$Text,
        [string]$Color = 'Cyan'
    )
    $script:LastWasProgress = $true
    $content = "│       " + $Text
    $padded = $content.PadRight($BoxWidth - 4)
    Write-Host ("`r" + $padded) -NoNewline -ForegroundColor $Color
}

# --- Explicit Help Intercept ---
if ($Help) {
    Get-Help $PSCommandPath -Full
    return
}

# --- Action & Cache Context ---
$ActionTitle = if ($AssembleOnly) {
    "Assemble Only (Skip Workbook Parse)"
} elseif ($Fresh) {
    "Fresh Workbook Download & Full Rebuild"
} else {
    "Cached Snapshot & Full Rebuild"
}

$CacheStatus = if (Test-Path -LiteralPath $CacheFile) {
    $ci = Get-Item -LiteralPath $CacheFile
    "{0:N2} MB (modified {1})" -f ($ci.Length / 1MB), $ci.LastWriteTime.ToString("yyyy-MM-dd HH:mm")
} else {
    "Not present (will download from Google Sheets)"
}

# --- Top Banner ---
$Bar = '═' * ($BoxWidth - 2)
Write-Host ""
Write-Host "╔$Bar╗" -ForegroundColor Cyan
Write-BoxHeader "TRIMPS SPIRE ASSAULT BUILD PIPELINE" "White" "Cyan" $BoxWidth
Write-Host "╠$Bar╣" -ForegroundColor Cyan
Write-BoxRow "Action :" " $ActionTitle" "DarkCyan" "Yellow" "Cyan" $BoxWidth
Write-BoxRow "Sheet  :" " $SheetId" "DarkCyan" "Gray" "Cyan" $BoxWidth
Write-BoxRow "Cache  :" " $CacheStatus" "DarkCyan" "Gray" "Cyan" $BoxWidth
Write-BoxRow "Target :" " $OutFile" "DarkCyan" "Gray" "Cyan" $BoxWidth
Write-Host "╚$Bar╝" -ForegroundColor Cyan
Write-Host ""

$Stopwatch = [System.Diagnostics.Stopwatch]::StartNew()

try {
    if (-not (Get-Command python -ErrorAction SilentlyContinue)) {
        throw "Python executable not found on PATH."
    }

    # Pre-build snapshots for diff & change analysis
    $PreData = Get-FileSnapshot -Path $DataFile
    $PreHtml = Get-FileSnapshot -Path $OutFile

    # Live stream frame
    $StreamTag = "─── [ LIVE BUILD STREAM ] "
    $StreamBar = "─" * ($BoxWidth - 2 - $StreamTag.Length)
    Write-Host "┌${StreamTag}${StreamBar}┐" -ForegroundColor DarkCyan

    # Step 1: Workbook & Catalog Snapshot
    if ($AssembleOnly) {
        Write-StreamLine "[1/3] Snapshot : Skipped (-AssembleOnly active, reusing data.json)" "DarkGray"
    } else {
        if ($Fresh -and (Test-Path -LiteralPath $CacheFile)) {
            Remove-Item -LiteralPath $CacheFile -Force
            Write-StreamLine "[1/3] Snapshot : Removed cached _sa.xlsx for fresh download..." "Yellow"
        } else {
            Write-StreamLine "[1/3] Snapshot : Extracting SA sheet & item catalog (build_data.py)..." "Yellow"
        }

        $buildArgs = @('-u', (Join-Path $Here 'build_data.py'))
        if ($Fresh) { $buildArgs += '--fresh' }
        if ($VerboseSheets) { $buildArgs += '--verbose' }

        $isInteractiveConsole = (-not [Console]::IsOutputRedirected -and $Host.UI.RawUI)
        $lastReportedPct = -1

        & python @buildArgs 2>&1 | ForEach-Object {
            $line = $_.ToString()
            if ($line -match '^PROGRESS:(\d+):(\d+):(\d+):(\d+):(\d+)') {
                $curr = [int]$Matches[1]
                $total = [int]$Matches[2]
                $lvl = [int]$Matches[3]
                $bld = [int]$Matches[4]
                $pct = [int]$Matches[5]

                Write-Progress -Activity "Extracting SA Sheet Builds" -Status "SA $lvl/160 ($pct%) · $bld builds" -PercentComplete $pct

                if ($isInteractiveConsole) {
                    $filled = [Math]::Floor($pct / 5)
                    $bar = [string]::new([char]0x2588, $filled) + [string]::new([char]0x2591, 20 - $filled)
                    $txt = "Parsing  : [{0}] {1,3}% (SA {2,3}/160 · {3,4} builds)" -f $bar, $pct, $lvl, $bld
                    Write-StreamProgress $txt "Cyan"
                } elseif ($pct -ge ($lastReportedPct + 25) -or $pct -eq 100) {
                    $lastReportedPct = [Math]::Floor($pct / 25) * 25
                    Write-StreamLine "      Parsing  : $pct% (SA $lvl/160 · $bld builds)" "DarkCyan"
                }
            } else {
                $color = 'White'
                if ($line -match '^(Workbook|Sheets|Catalog)\s*:') {
                    $color = 'Gray'
                } elseif ($line -match '^Snapshot\s*:') {
                    $color = 'Green'
                } elseif ($line -match '^\s+SA\s+\d+:') {
                    $color = 'DarkGray'
                } elseif ($line -match '(?i)(error|traceback|exception)') {
                    $color = 'Red'
                }
                Write-StreamLine "      $line" $color
            }
        }
        Write-Progress -Activity "Extracting SA Sheet Builds" -Completed
        if ($LASTEXITCODE -ne 0) {
            throw "build_data.py failed with exit code $LASTEXITCODE"
        }
    }

    # Step 2: JavaScript Syntax Validation
    Write-StreamLine "[2/3] Validate : Checking app.js syntax..." "Yellow"
    if (Get-Command node -ErrorAction SilentlyContinue) {
        & node --check $AppFile 2>&1 | ForEach-Object {
            Write-StreamLine "      $($_.ToString())" "Red"
        }
        if ($LASTEXITCODE -ne 0) {
            throw "JavaScript syntax check failed in app.js (exit code $LASTEXITCODE)"
        }
        Write-StreamLine "      Syntax   : OK (app.js verified via node --check)" "Green"
    } else {
        Write-StreamLine "      Syntax   : Skipped (node not found on PATH)" "DarkGray"
    }

    # Step 3: Self-Contained HTML Assembly
    Write-StreamLine "[3/3] Assemble : Bundling self-contained trimps/sa.html (assemble.py)..." "Yellow"
    & python (Join-Path $Here 'assemble.py') 2>&1 | ForEach-Object {
        $line = $_.ToString()
        $color = if ($line -match '^Assemble\s*:') { 'Green' } elseif ($line -match '^Inputs\s*:') { 'Gray' } else { 'White' }
        Write-StreamLine "      $line" $color
    }
    if ($LASTEXITCODE -ne 0) {
        throw "assemble.py failed with exit code $LASTEXITCODE"
    }

    if ($CleanCache -and (Test-Path -LiteralPath $CacheFile)) {
        Remove-Item -LiteralPath $CacheFile -Force
        Write-StreamLine "      Cleanup  : Removed temporary _sa.xlsx cache file" "Cyan"
    }

    Write-Host ("└" + ("─" * ($BoxWidth - 2)) + "┘") -ForegroundColor DarkCyan
    Write-Host ""

    $Stopwatch.Stop()
    $ElapsedStr = "{0:N2}s" -f $Stopwatch.Elapsed.TotalSeconds

    # --- Post-build Snapshots & Diff Computation ---
    $PostData = Get-FileSnapshot -Path $DataFile
    $PostHtml = Get-FileSnapshot -Path $OutFile
    $DataDiff = Compare-SaDataJson -OldRaw $PreData.Raw -NewRaw $PostData.Raw

    $BareLfCount = ([regex]::Matches($PostHtml.Raw, "(?<!`r)`n")).Count
    $EolStatus   = if ($BareLfCount -eq 0) { "CRLF verified" } else { "Mixed ($BareLfCount bare LF)" }

    $ShellSize = (Get-Item -LiteralPath $ShellFile).Length
    $AppSize   = (Get-Item -LiteralPath $AppFile).Length
    $LzSize    = (Get-Item -LiteralPath $LzFile).Length

    $HtmlChanged   = ($PreHtml.Hash -ne $PostHtml.Hash)
    $HtmlSizeDelta = $PostHtml.Size - $PreHtml.Size
    $HtmlLineDelta = $PostHtml.Lines - $PreHtml.Lines
    $DataChanged   = ($PreData.Hash -ne $PostData.Hash)
    $DataSizeDelta = $PostData.Size - $PreData.Size

    # --- Completion & Wrap-Up Stats Box ---
    $CompleteBar = '═' * ($BoxWidth - 2)
    Write-Host "╔$CompleteBar╗" -ForegroundColor Green
    Write-BoxHeader "✔  BUILD & SNAPSHOT COMPLETE" "White" "Green" $BoxWidth
    Write-Host "╠$CompleteBar╣" -ForegroundColor Green

    $OverallState = if ($HtmlChanged -or $DataChanged) {
        "Completed with updates in $ElapsedStr"
    } else {
        "Completed in $ElapsedStr (in sync, 0 B delta)"
    }
    Write-BoxRow "Status   :" " $OverallState" "DarkGreen" "Green" "Green" $BoxWidth
    Write-BoxRow "Output   :" " $(Format-ByteSize $PostHtml.Size) · $($PostHtml.Lines) lines · $EolStatus" "DarkGreen" "White" "Green" $BoxWidth
    Write-BoxRow "Payload  :" (" shell {0:N1}KB | app {1:N1}KB | data {2:N2}MB | lz {3:N1}KB" -f ($ShellSize/1KB), ($AppSize/1KB), ($PostData.Size/1MB), ($LzSize/1KB)) "DarkGreen" "Gray" "Green" $BoxWidth
    Write-BoxRow "Dataset  :" " $($DataDiff.LevelCount) SA levels · $($DataDiff.BuildCount) community builds · $($DataDiff.ItemCount) items" "DarkGreen" "Gray" "Green" $BoxWidth

    # Section: Changes Made During This Run
    Write-BoxSection "CHANGES MADE (THIS RUN)" "Green" "White" $BoxWidth
    if ($DataChanged) {
        $dDeltaStr = Format-SignedDelta $DataSizeDelta 'B'
        Write-BoxRow "data.json:" " Updated ($dDeltaStr, sha $($PreData.Hash) -> $($PostData.Hash))" "DarkGreen" "Yellow" "Green" $BoxWidth
        $changeCount = $DataDiff.AddedBuilds.Count + $DataDiff.RemovedBuilds.Count + $DataDiff.UpdatedBuilds.Count
        if ($changeCount -gt 0) {
            Write-BoxRow "Builds   :" " +$($DataDiff.AddedBuilds.Count) added, ~$($DataDiff.UpdatedBuilds.Count) updated, -$($DataDiff.RemovedBuilds.Count) removed" "DarkGreen" "Cyan" "Green" $BoxWidth
            $details = @($DataDiff.AddedBuilds) + @($DataDiff.UpdatedBuilds) + @($DataDiff.RemovedBuilds)
            $preview = ($details | Select-Object -First 4) -join ', '
            if ($details.Count -gt 4) { $preview += " (+$($details.Count - 4) more)" }
            Write-BoxRow "Detail   :" " $preview" "DarkGreen" "Gray" "Green" $BoxWidth
        }
        if ($DataDiff.ItemDiffs.Count -gt 0) {
            Write-BoxRow "Items    :" " $($DataDiff.ItemDiffs -join ', ')" "DarkGreen" "Cyan" "Green" $BoxWidth
        }
    } else {
        Write-BoxRow "data.json:" " Unchanged (sha $($PostData.Hash))" "DarkGreen" "Gray" "Green" $BoxWidth
    }

    if ($HtmlChanged) {
        $hBytesStr = Format-SignedDelta $HtmlSizeDelta 'B'
        $hLinesStr = Format-SignedDelta $HtmlLineDelta 'lines'
        Write-BoxRow "sa.html  :" " Rebuilt ($hBytesStr, $hLinesStr, sha $($PreHtml.Hash) -> $($PostHtml.Hash))" "DarkGreen" "Yellow" "Green" $BoxWidth
    } else {
        Write-BoxRow "sa.html  :" " Unchanged (sha $($PostHtml.Hash) matches prior build)" "DarkGreen" "Gray" "Green" $BoxWidth
    }

    # Section: Git Working Tree Status vs HEAD
    if (Get-Command git -ErrorAction SilentlyContinue) {
        $gitRows = @(& git -C $RepoRoot status --porcelain -- "trimps/sa.html" "trimps/sa-build" 2>$null)
        $numStat = @{}
        & git -C $RepoRoot diff --numstat HEAD -- "trimps/sa.html" "trimps/sa-build" 2>$null | ForEach-Object {
            $parts = $_ -split '\s+'
            if ($parts.Count -ge 3) {
                $numStat[$parts[2]] = "+$($parts[0])/-$($parts[1]) lines"
            }
        }

        Write-BoxSection "GIT WORKING TREE (VS HEAD)" "Green" "White" $BoxWidth
        if ($gitRows.Count -eq 0) {
            Write-BoxRow "Repo Sync    :" " Clean (all SA files match HEAD commit)" "DarkGreen" "Gray" "Green" $BoxWidth
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