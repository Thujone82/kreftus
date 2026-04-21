<#
.SYNOPSIS
Updates PDX Heritage Trees version across release files and injects asset versions.

.DESCRIPTION
Usage examples:
  ./version.ps1 1.1
  version 1.1

The script updates:
  - heritage/service-worker.js  (const VERSION = 'x.y.z';)
  - heritage/manifest.json      ("version": "x.y.z")
  - heritage/js/app.js          (const APP_VERSION = 'x.y.z';)
  - heritage/index.html         (?v=... cache-busting links for css/js assets)

It prints step-by-step status and exits non-zero on failure.
#>

[CmdletBinding()]
param(
    [Parameter(Position = 0, Mandatory = $false)]
    [string]$Version,
    [switch]$Rev,
    [switch]$Minor,
    [switch]$Major,
    [switch]$Help
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$script:RollbackSnapshots = @{}

function Write-Step {
    param(
        [string]$Message,
        [string]$Color = 'Cyan'
    )
    Write-Host "==> $Message" -ForegroundColor $Color
}

function Show-VersionScriptHelp {
    $helpText = @'
PDX Heritage Trees version updater

Usage:
  .\heritage\version.ps1 [<version>] [-rev|-minor|-major] [-help]

Description:
  Updates version values in:
    - heritage/service-worker.js  (const VERSION = 'x.y.z')
    - heritage/manifest.json      ("version": "x.y.z")
    - heritage/js/app.js          (const APP_VERSION = 'x.y.z')
    - heritage/index.html         (css/js ?v= cache-busting links)

Modes:
  1) Explicit version:
     .\heritage\version.ps1 1.1.0
     - Uses the provided version string after sanitization.
     - Allowed characters: A-Z a-z 0-9 . _ -
     - Spaces and invalid URL characters are removed.

  2) Increment revision:
     .\heritage\version.ps1 -rev
     - Reads current version from service-worker.js.
     - Increments revision only: Major.Minor.Revision -> Major.Minor.(Revision+1)
     - Example: 1.2.3 -> 1.2.4

  3) Increment minor:
     .\heritage\version.ps1 -minor
     - Reads current version from service-worker.js.
     - Increments minor and resets revision: Major.Minor.Revision -> Major.(Minor+1).0
     - Example: 1.2.3 -> 1.3.0

  4) Increment major:
     .\heritage\version.ps1 -major
     - Reads current version from service-worker.js.
     - Increments major and resets minor/revision: Major.Minor.Revision -> (Major+1).0.0
     - Example: 1.2.3 -> 2.0.0

Version parsing notes:
  - Increment modes parse numeric base as: ^(\d+)\.(\d+)\.(\d+)
  - Any trailing note/suffix is ignored in increment mode.
    Example: 1.2.3-test with -rev becomes 1.2.4

Interactive prompt behavior:
  - If no explicit version and no increment switch is provided,
    the script prompts for "Enter updated version".
  - Blank input cancels with:
    "Version must not be blank, update cancelled"

Rules:
  - -rev, -minor, and -major are mutually exclusive.
  - Do not combine explicit <version> with increment switches.
  - -help shows this message and exits without changing files.
'@
    Write-Host $helpText -ForegroundColor Gray
}

function Get-CurrentVersionFromServiceWorker {
    param(
        [string]$Path
    )

    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Missing file: $Path"
    }

    $content = (Read-TextWithBomState -Path $Path).Text
    $match = [regex]::Match($content, "const\s+VERSION\s*=\s*'([^']+)';")
    if (-not $match.Success) {
        throw "Could not determine current version from service-worker.js"
    }
    return $match.Groups[1].Value
}

function ConvertTo-BaseSemVerOrFail {
    param(
        [string]$RawVersion
    )

    $versionText = if ($null -eq $RawVersion) { '' } else { [string]$RawVersion }
    $match = [regex]::Match($versionText, '^(\d+)\.(\d+)\.(\d+)')
    if (-not $match.Success) {
        throw "Current version '$RawVersion' is not parseable as Major.Minor.Revision"
    }

    return [pscustomobject]@{
        Major = [int]$match.Groups[1].Value
        Minor = [int]$match.Groups[2].Value
        Revision = [int]$match.Groups[3].Value
        Base = "$($match.Groups[1].Value).$($match.Groups[2].Value).$($match.Groups[3].Value)"
    }
}

function Get-IncrementedVersion {
    param(
        [int]$MajorPart,
        [int]$MinorPart,
        [int]$RevisionPart,
        [string]$Mode
    )

    switch ($Mode) {
        'rev' {
            return "$MajorPart.$MinorPart.$([int]($RevisionPart + 1))"
        }
        'minor' {
            return "$MajorPart.$([int]($MinorPart + 1)).0"
        }
        'major' {
            return "$([int]($MajorPart + 1)).0.0"
        }
        default {
            throw "Unknown increment mode '$Mode'"
        }
    }
}

function Read-TextWithBomState {
    param(
        [string]$Path
    )

    $bytes = [System.IO.File]::ReadAllBytes($Path)
    $hasUtf8Bom = $bytes.Length -ge 3 -and $bytes[0] -eq 0xEF -and $bytes[1] -eq 0xBB -and $bytes[2] -eq 0xBF
    $text = [System.Text.Encoding]::UTF8.GetString($bytes)
    if ($text.Length -gt 0 -and [int][char]$text[0] -eq 0xFEFF) {
        $text = $text.Substring(1)
    }
    return [pscustomobject]@{
        Text = $text
        HasUtf8Bom = $hasUtf8Bom
    }
}

function Write-TextPreserveBom {
    param(
        [string]$Path,
        [string]$Text,
        [bool]$HasUtf8Bom,
        [int]$MaxAttempts = 8
    )

    $encoding = [System.Text.UTF8Encoding]::new($HasUtf8Bom)
    for ($attempt = 1; $attempt -le $MaxAttempts; $attempt++) {
        try {
            [System.IO.File]::WriteAllText($Path, $Text, $encoding)
            return
        } catch [System.IO.IOException], [System.UnauthorizedAccessException] {
            if ($attempt -ge $MaxAttempts) {
                throw
            }
            Start-Sleep -Milliseconds (100 * $attempt)
        }
    }
}

function Save-RollbackSnapshotIfMissing {
    param(
        [string]$Path
    )

    if ([string]::IsNullOrWhiteSpace($Path)) { return }
    if ($script:RollbackSnapshots.ContainsKey($Path)) { return }
    if (-not (Test-Path -LiteralPath $Path)) { return }

    $readState = Read-TextWithBomState -Path $Path
    $script:RollbackSnapshots[$Path] = [pscustomobject]@{
        Text = $readState.Text
        HasUtf8Bom = $readState.HasUtf8Bom
    }
}

function Invoke-RollbackSnapshots {
    param()

    if ($null -eq $script:RollbackSnapshots -or $script:RollbackSnapshots.Count -eq 0) {
        return
    }

    Write-Host "Attempting rollback of modified files..." -ForegroundColor Yellow
    foreach ($entry in $script:RollbackSnapshots.GetEnumerator()) {
        $path = [string]$entry.Key
        $snap = $entry.Value
        try {
            Write-TextPreserveBom -Path $path -Text ([string]$snap.Text) -HasUtf8Bom ([bool]$snap.HasUtf8Bom)
            Write-Host " - rolled back: $path" -ForegroundColor DarkYellow
        } catch {
            Write-Host " - rollback failed: $path ($($_.Exception.Message))" -ForegroundColor Red
        }
    }
}

function Set-TextByPatternOrFail {
    param(
        [string]$Path,
        [string]$Pattern,
        [string]$Replacement,
        [string]$Description
    )

    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Missing file: $Path"
    }

    $readState = Read-TextWithBomState -Path $Path
    $content = $readState.Text
    $regex = [regex]::new($Pattern)
    if (-not $regex.IsMatch($content)) {
        throw "No change made for $Description in $Path (pattern not found)."
    }

    $match = $regex.Match($content)
    if (-not $match.Success) {
        throw "No change made for $Description in $Path (pattern not found)."
    }
    $oldValue = $match.Value

    $updated = $regex.Replace($content, $Replacement, 1)
    $changed = $updated -ne $content
    if ($changed) {
        Save-RollbackSnapshotIfMissing -Path $Path
        Write-TextPreserveBom -Path $Path -Text $updated -HasUtf8Bom $readState.HasUtf8Bom
    }
    return [pscustomobject]@{
        Changed = $changed
        OldValue = $oldValue
        NewValue = $Replacement
    }
}

function Update-AssetVersion-OrFail {
    param(
        [string]$Path,
        [string]$Pattern,
        [string]$Description,
        [string]$Version
    )

    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Missing file: $Path"
    }

    $readState = Read-TextWithBomState -Path $Path
    $content = $readState.Text
    $regex = [regex]::new($Pattern)
    $match = $regex.Match($content)
    if (-not $match.Success) {
        throw "No change made for $Description in $Path (pattern not found)."
    }

    $oldValue = $match.Value
    $newValue = [regex]::Replace($oldValue, '\?v=[^"]+', "?v=$Version")
    $updated = $regex.Replace($content, $newValue, 1)
    $changed = $updated -ne $content
    if ($changed) {
        Save-RollbackSnapshotIfMissing -Path $Path
        Write-TextPreserveBom -Path $Path -Text $updated -HasUtf8Bom $readState.HasUtf8Bom
    }
    return [pscustomobject]@{
        Changed = $changed
        OldValue = $oldValue
        NewValue = $newValue
    }
}

try {
    if ($Help) {
        Show-VersionScriptHelp
        exit 0
    }

    $heritageRoot = if ($PSScriptRoot) { $PSScriptRoot } else { (Get-Location).Path }
    $serviceWorkerPath = Join-Path $heritageRoot 'service-worker.js'
    $manifestPath = Join-Path $heritageRoot 'manifest.json'
    $indexPath = Join-Path $heritageRoot 'index.html'
    $appJsPath = Join-Path $heritageRoot 'js\app.js'

    $incrementFlagsUsed = @($Rev, $Minor, $Major).Where({ $_ }).Count
    if ($incrementFlagsUsed -gt 1) {
        throw "Options -rev, -minor, and -major are mutually exclusive. Specify only one."
    }
    if ($incrementFlagsUsed -gt 0 -and -not [string]::IsNullOrWhiteSpace($Version)) {
        throw "Do not provide a version value together with -rev/-minor/-major. Use either explicit version OR one increment flag."
    }

    $currentVersion = Get-CurrentVersionFromServiceWorker -Path $serviceWorkerPath
    $sanitizedVersion = $null

    if ($incrementFlagsUsed -gt 0) {
        $mode = if ($Rev) { 'rev' } elseif ($Minor) { 'minor' } else { 'major' }
        $parsed = ConvertTo-BaseSemVerOrFail -RawVersion $currentVersion
        $sanitizedVersion = Get-IncrementedVersion `
            -MajorPart $parsed.Major `
            -MinorPart $parsed.Minor `
            -RevisionPart $parsed.Revision `
            -Mode $mode
        Write-Step "Current version source: $currentVersion" 'Yellow'
        Write-Step "Parsed base version: $($parsed.Base)" 'Yellow'
        Write-Step "Increment mode: -$mode" 'Yellow'
    } else {
        if ([string]::IsNullOrWhiteSpace($Version)) {
            Write-Host "Current version is $currentVersion" -ForegroundColor Yellow
            $Version = Read-Host "Enter updated version"
            if ([string]::IsNullOrWhiteSpace($Version)) {
                Write-Host ""
                Write-Host "Version must not be blank, update cancelled" -ForegroundColor Red
                exit 1
            }
        }

        # Allow alphanumeric semantic-ish versions (e.g. 1.7.12b), but strip spaces
        # and any characters that are unsafe/invalid for URL query values.
        $inputVersion = $Version
        $trimmedVersion = ($Version -replace '\s+', '')
        $sanitizedVersion = ($trimmedVersion -replace '[^A-Za-z0-9._-]', '')
        if ([string]::IsNullOrWhiteSpace($sanitizedVersion)) {
            throw "Invalid version '$inputVersion'. After filtering invalid characters, no usable version remained."
        }

        if ($sanitizedVersion -ne $inputVersion) {
            Write-Step "Input version normalized from '$inputVersion' to '$sanitizedVersion'" 'Yellow'
        }
    }
    Write-Step "Target version: $sanitizedVersion" 'Yellow'

    Write-Step "Updating service-worker VERSION in $serviceWorkerPath"
    $swResult = Set-TextByPatternOrFail `
        -Path $serviceWorkerPath `
        -Pattern "const\s+VERSION\s*=\s*'[^']+';" `
        -Replacement "const VERSION = '$sanitizedVersion';" `
        -Description 'service-worker VERSION constant'
    Write-Host "   old: $($swResult.OldValue)" -ForegroundColor DarkGray
    Write-Host "   new: $($swResult.NewValue)" -ForegroundColor DarkGray
    if ($swResult.Changed) {
        Write-Host "   OK: service-worker.js updated" -ForegroundColor Green
    } else {
        Write-Host "   OK: service-worker.js already set to $sanitizedVersion" -ForegroundColor DarkYellow
    }

    Write-Step "Updating manifest version in $manifestPath"
    $manifestResult = Set-TextByPatternOrFail `
        -Path $manifestPath `
        -Pattern '"version"\s*:\s*"[^"]+"' `
        -Replacement "`"version`": `"$sanitizedVersion`"" `
        -Description 'manifest version field'
    Write-Host "   old: $($manifestResult.OldValue)" -ForegroundColor DarkGray
    Write-Host "   new: $($manifestResult.NewValue)" -ForegroundColor DarkGray
    if ($manifestResult.Changed) {
        Write-Host "   OK: manifest.json updated" -ForegroundColor Green
    } else {
        Write-Host "   OK: manifest.json already set to $sanitizedVersion" -ForegroundColor DarkYellow
    }

    Write-Step "Updating APP_VERSION in $appJsPath"
    $appResult = Set-TextByPatternOrFail `
        -Path $appJsPath `
        -Pattern "const\s+APP_VERSION\s*=\s*'[^']+';" `
        -Replacement "const APP_VERSION = '$sanitizedVersion';" `
        -Description 'app.js APP_VERSION constant'
    Write-Host "   old: $($appResult.OldValue)" -ForegroundColor DarkGray
    Write-Host "   new: $($appResult.NewValue)" -ForegroundColor DarkGray
    if ($appResult.Changed) {
        Write-Host "   OK: app.js updated" -ForegroundColor Green
    } else {
        Write-Host "   OK: app.js already set to $sanitizedVersion" -ForegroundColor DarkYellow
    }

    Write-Step "Updating index.html cache-busting links in $indexPath"
    $assetUpdates = @(
        @{ Description = 'styles.css version link';    Pattern = '<link\s+rel="stylesheet"\s+href="css/styles\.css\?v=[^"]+">'},
        @{ Description = 'db.js version link';         Pattern = '<script\s+src="js/db\.js\?v=[^"]+"></script>'},
        @{ Description = 'wiki.js version link';       Pattern = '<script\s+src="js/wiki\.js\?v=[^"]+"></script>'},
        @{ Description = 'sync.js version link';       Pattern = '<script\s+src="js/sync\.js\?v=[^"]+"></script>'},
        @{ Description = 'geocode.js version link';    Pattern = '<script\s+src="js/geocode\.js\?v=[^"]+"></script>'},
        @{ Description = 'map.js version link';        Pattern = '<script\s+src="js/map\.js\?v=[^"]+"></script>'},
        @{ Description = 'nearby.js version link';     Pattern = '<script\s+src="js/nearby\.js\?v=[^"]+"></script>'},
        @{ Description = 'search.js version link';     Pattern = '<script\s+src="js/search\.js\?v=[^"]+"></script>'},
        @{ Description = 'found.js version link';      Pattern = '<script\s+src="js/found\.js\?v=[^"]+"></script>'},
        @{ Description = 'ui.js version link';         Pattern = '<script\s+src="js/ui\.js\?v=[^"]+"></script>'},
        @{ Description = 'sw-register.js version link';Pattern = '<script\s+src="js/sw-register\.js\?v=[^"]+"></script>'},
        @{ Description = 'app.js version link';        Pattern = '<script\s+src="js/app\.js\?v=[^"]+"></script>'}
    )

    $indexChangedAny = $false
    foreach ($item in $assetUpdates) {
        $assetResult = Update-AssetVersion-OrFail `
            -Path $indexPath `
            -Pattern $item.Pattern `
            -Description $item.Description `
            -Version $sanitizedVersion
        Write-Host "   old: $($assetResult.OldValue)" -ForegroundColor DarkGray
        Write-Host "   new: $($assetResult.NewValue)" -ForegroundColor DarkGray
        if ($assetResult.Changed) {
            $indexChangedAny = $true
            Write-Host "   OK: $($item.Description) updated" -ForegroundColor Green
        } else {
            Write-Host "   OK: $($item.Description) already set to $sanitizedVersion" -ForegroundColor DarkYellow
        }
    }

    if ($indexChangedAny) {
        Write-Host "   OK: index.html cache-busting links updated" -ForegroundColor Green
    } else {
        Write-Host "   OK: index.html cache-busting links already set to $sanitizedVersion" -ForegroundColor DarkYellow
    }

    Write-Host ""
    Write-Host "Version update completed successfully." -ForegroundColor Green
    Write-Host " - service-worker.js: $sanitizedVersion" -ForegroundColor Gray
    Write-Host " - manifest.json:     $sanitizedVersion" -ForegroundColor Gray
    Write-Host " - js/app.js:         $sanitizedVersion" -ForegroundColor Gray
    Write-Host " - index.html links:  $sanitizedVersion" -ForegroundColor Gray
}
catch {
    Invoke-RollbackSnapshots
    Write-Host ""
    Write-Host "Version update failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
