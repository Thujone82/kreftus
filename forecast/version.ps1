<#
.SYNOPSIS
Updates Forecast version across release files and injects asset versions.

.DESCRIPTION
Usage examples:
  ./version.ps1 1.8
  version 1.8

The script updates:
  - forecast/service-worker.js  (const VERSION = 'x.y.z';)
  - forecast/manifest.json      ("version": "x.y.z")
  - forecast/index.html         (?v=... cache-busting links for css/js assets)

It prints step-by-step status and exits non-zero on failure.
#>

[CmdletBinding()]
param(
    [Parameter(Position = 0, Mandatory = $false)]
    [string]$Version
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Write-Step {
    param(
        [string]$Message,
        [string]$Color = 'Cyan'
    )
    Write-Host "==> $Message" -ForegroundColor $Color
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
        [bool]$HasUtf8Bom
    )

    $encoding = [System.Text.UTF8Encoding]::new($HasUtf8Bom)
    [System.IO.File]::WriteAllText($Path, $Text, $encoding)
}

function Replace-OrFail {
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
        Write-TextPreserveBom -Path $Path -Text $updated -HasUtf8Bom $readState.HasUtf8Bom
    }
    return [pscustomobject]@{
        Changed = $changed
        OldValue = $oldValue
        NewValue = $newValue
    }
}

try {
    $forecastRoot = if ($PSScriptRoot) { $PSScriptRoot } else { (Get-Location).Path }
    $serviceWorkerPath = Join-Path $forecastRoot 'service-worker.js'
    $manifestPath = Join-Path $forecastRoot 'manifest.json'
    $indexPath = Join-Path $forecastRoot 'index.html'

    if ([string]::IsNullOrWhiteSpace($Version)) {
        $currentVersion = Get-CurrentVersionFromServiceWorker -Path $serviceWorkerPath
        Write-Host "Current version is $currentVersion" -ForegroundColor Yellow
        $Version = Read-Host "Enter updated version"
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
    Write-Step "Target version: $sanitizedVersion" 'Yellow'

    Write-Step "Updating service-worker VERSION in $serviceWorkerPath"
    $swResult = Replace-OrFail `
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
    $manifestResult = Replace-OrFail `
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

    Write-Step "Updating index.html cache-busting links in $indexPath"
    $assetUpdates = @(
        @{ Description = 'style.css version link'; Pattern = '<link\s+rel="stylesheet"\s+href="css/style\.css\?v=[^"]+">'},
        @{ Description = 'utils.js version link'; Pattern = '<script\s+src="js/utils\.js\?v=[^"]+"></script>'},
        @{ Description = 'api.js version link'; Pattern = '<script\s+src="js/api\.js\?v=[^"]+"></script>'},
        @{ Description = 'weather.js version link'; Pattern = '<script\s+src="js/weather\.js\?v=[^"]+"></script>'},
        @{ Description = 'display.js version link'; Pattern = '<script\s+src="js/display\.js\?v=[^"]+"></script>'},
        @{ Description = 'app.js version link'; Pattern = '<script\s+src="js/app\.js\?v=[^"]+"></script>'}
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
    Write-Host " - index.html links:  $sanitizedVersion" -ForegroundColor Gray
}
catch {
    Write-Host ""
    Write-Host "Version update failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
