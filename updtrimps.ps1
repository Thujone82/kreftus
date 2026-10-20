#Requires -Version 5.1
<#
.SYNOPSIS
    Updates local trimps/ from upstream Trimps.github.io, localises by removing .git, restores PWA components, and reports file changes.
.DESCRIPTION
    Clones https://github.com/Trimps/Trimps.github.io into a temp dir, copies into trimps/ (overwrite),
    removes trimps/.git, restores manifest.json, sw.js, and index.html PWA edits. Reports Added/Removed/Updated files or "No changes."
#>

[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$trimpsPath = Join-Path $PSScriptRoot 'trimps'

function Write-Header {
  param([string]$Text)
  $border = "=" * ($Text.Length + 8)
  Write-Host $border -NoNewline -ForegroundColor Cyan
  $consoleWidth = $host.UI.RawUI.BufferSize.Width
  $remainingWidth = $consoleWidth - $border.Length
  if ($remainingWidth -gt 0) {
    Write-Host (" " * $remainingWidth) -NoNewline -BackgroundColor Black
  }
  Write-Host ""
  Write-Host ("    $Text    ") -NoNewline -ForegroundColor Black -BackgroundColor Cyan
  $remainingAfterTitle = $consoleWidth - ($Text.Length + 8)
  if ($remainingAfterTitle -gt 0) {
    Write-Host (" " * $remainingAfterTitle) -NoNewline -BackgroundColor Black
  }
  Write-Host ""
  Write-Host $border -NoNewline -ForegroundColor Cyan
  if ($remainingWidth -gt 0) {
    Write-Host (" " * $remainingWidth) -NoNewline -BackgroundColor Black
  }
  Write-Host ""
  [Console]::ResetColor()
}

function Write-Step {
  param([string]$Message)
  Write-Host " * " -ForegroundColor Yellow -NoNewline
  Write-Host $Message -ForegroundColor White
}

Write-Header "Trimps Update Tool"

# --- Require git ---
if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
  Write-Error "git is required but not found. Install Git and ensure it is on PATH."
  Write-Host "Error: git is required but not found. Install Git and ensure it is on PATH." -ForegroundColor Red
  exit 1
}

# --- Capture "before" state (relative path -> hash), exclude .git and .vscode ---
function Get-TrimpsFileHashes {
  param([string]$BasePath)
  $h = @{}
  if (-not (Test-Path -LiteralPath $BasePath)) { return $h }
  $baseLen = (Resolve-Path -LiteralPath $BasePath).Path.TrimEnd('\').Length + 1
  Get-ChildItem -Path $BasePath -Recurse -File -Force | Where-Object { $_.FullName -notmatch '\.(git|vscode)\\' } | ForEach-Object {
    $rel = $_.FullName.Substring($baseLen)
    $h[$rel] = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash
    Write-Verbose "Hashed local file: $rel"
  }
  return $h
}

Write-Step "Scanning local trimps directory for existing files..."
$before = Get-TrimpsFileHashes -BasePath $trimpsPath
Write-Host "   Found $($before.Count) local files." -ForegroundColor DarkGray

# --- Clone into temp ---
Write-Step "Fetching upstream data from Trimps.github.io..."
$cloneDir = Join-Path $env:TEMP "trimps_clone_$(Get-Date -Format 'yyyyMMddHHmmss')"
try {
  & git clone -q --depth 1 'https://github.com/Trimps/Trimps.github.io.git' $cloneDir
  if ($LASTEXITCODE -ne 0) { throw "git clone exited with $LASTEXITCODE" }
}
catch {
  if (Test-Path -LiteralPath $cloneDir) { Remove-Item -LiteralPath $cloneDir -Recurse -Force -ErrorAction SilentlyContinue }
  throw "Clone failed: $_"
}

Write-Host "   [+] Remote fetch complete." -ForegroundColor DarkGray

Write-Step "Updating local files from upstream sources..."
try {
  # --- Ensure trimps exists ---
  if (-not (Test-Path -LiteralPath $trimpsPath)) {
    New-Item -ItemType Directory -Path $trimpsPath -Force | Out-Null
  }
  # --- Copy only files that are new or changed (skip .git, .vscode); skip local-owned files when they already exist ---
  # Do not overwrite index.html, manifest.json, sw.js, or css/kreftus-local.css when they already exist - we own them and restore below
  $localOwnedFiles = @{
    'index.html' = $true
    'manifest.json' = $true
    'sw.js' = $true
    'css\kreftus-local.css' = $true
    'css/kreftus-local.css' = $true
    'icons\192.png' = $true
    'icons/192.png' = $true
    'icons\512.png' = $true
    'icons/512.png' = $true
  }
  $cloneBaseLen = (Resolve-Path -LiteralPath $cloneDir).Path.TrimEnd('\').Length + 1
  $copyCount = 0
  $skipCount = 0
  Get-ChildItem -Path $cloneDir -Recurse -File -Force | Where-Object { $_.FullName -notmatch '\.(git|vscode)\\' } | ForEach-Object {
    $rel = $_.FullName.Substring($cloneBaseLen)
    $dest = Join-Path $trimpsPath $rel
    if ($localOwnedFiles.ContainsKey($rel) -and (Test-Path -LiteralPath $dest)) {
      Write-Verbose "Skipped (local owned): $rel"
      $skipCount++
      return 
    }
    $doCopy = $true
    if (Test-Path -LiteralPath $dest) {
      $srcHash = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash
      $destHash = (Get-FileHash -LiteralPath $dest -Algorithm SHA256).Hash
      if ($srcHash -eq $destHash) { $doCopy = $false }
    }
    if ($doCopy) {
      $destDir = Split-Path -Parent $dest
      if (-not (Test-Path -LiteralPath $destDir)) { New-Item -ItemType Directory -Path $destDir -Force | Out-Null }
      Copy-Item -LiteralPath $_.FullName -Destination $dest -Force
      Write-Verbose "Copied: $rel"
      $copyCount++
    }
    else {
      Write-Verbose "Skipped identical: $rel"
      $skipCount++
    }
  }
  Write-Host "   [+] Copied $copyCount files, skipped $skipCount identical files." -ForegroundColor DarkGray
}
finally {
  Remove-Item -LiteralPath $cloneDir -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Step "Cleaning up unnecessary repository files..."
$gitPath = Join-Path $trimpsPath '.git'
if (Test-Path -LiteralPath $gitPath) {
  Remove-Item -LiteralPath $gitPath -Recurse -Force
  Write-Host "   [-] Removed .git directory" -ForegroundColor DarkGray
}
$vscodePath = Join-Path $trimpsPath '.vscode'
if (Test-Path -LiteralPath $vscodePath) {
  Remove-Item -LiteralPath $vscodePath -Recurse -Force
  Write-Host "   [-] Removed .vscode directory" -ForegroundColor DarkGray
}

Write-Step "Restoring local components (PWA + kreftus-local.css)..."
$utf8NoBom = [System.Text.UTF8Encoding]::new($false)

# --- Restore local CSS overlay (only write if missing or content differs) ---
# Use ASCII-only content so PS 5.1 script encoding (no BOM) cannot corrupt the here-string
$localCssContent = @'
/* Local overrides - not from upstream; restored by updtrimps.ps1 */

/*
 * Stretch SA from the top of the window to just above the Save toolbar
 * so the items list can fill leftover height instead of leaving a gap.
 * Header stays a 2-column grid (Huffy | Enemy).
 */
#tooltipDiv.tooltipExtraBiggest {
  display: flex;
  flex-direction: column;
  top: 0 !important;
  bottom: 2.6vw;
  max-width: 100vw;
  overflow: hidden;
  box-sizing: border-box;
}

#tooltipDiv.tooltipExtraBiggest > #tipText {
  flex: 1 1 0%;
  min-height: 0;
  max-width: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
}

#tooltipDiv.tooltipExtraBiggest #tipText > .noselect {
  flex: 1 1 0%;
  min-height: 0;
  max-width: 100%;
  overflow-x: clip;
  overflow-y: hidden;
  display: grid;
  grid-template-columns: 1fr 1fr;
  column-gap: 2%;
  grid-template-rows: repeat(8, max-content) minmax(0, 1fr);
  box-sizing: border-box;
}

#tooltipDiv.tooltipExtraBiggest #autoDust,
#tooltipDiv.tooltipExtraBiggest #abLevelButtons,
#tooltipDiv.tooltipExtraBiggest #autoBattleStatsText,
#tooltipDiv.tooltipExtraBiggest #autoBattleMenuButtons,
#tooltipDiv.tooltipExtraBiggest #autoBattleNotes,
#tooltipDiv.tooltipExtraBiggest #autoItemsDiv {
  grid-column: 1 / -1;
}

#tooltipDiv.tooltipExtraBiggest .autoBattleTopName,
#tooltipDiv.tooltipExtraBiggest .autoBattleBarHolder {
  width: auto;
  max-width: 100%;
  margin-left: 0;
  margin-right: 0;
  box-sizing: border-box;
}

/*
 * SA fight bars: never scroll. Also size text to the bar so a 1px line-height
 * mismatch cannot create a scrollable overflow box.
 */
#tooltipDiv.tooltipExtraBiggest .autoBattleBarHolder,
#tooltipDiv.tooltipExtraBiggest .autoBattleBarHolder .progress-bar,
#tooltipDiv.tooltipExtraBiggest .autoBattleBarHolder .innerFightBar,
#tooltipDiv.tooltipExtraBiggest .autoBattleBarHolder .noselect {
  overflow: hidden !important;
  scrollbar-width: none !important;
  -ms-overflow-style: none !important;
}

#tooltipDiv.tooltipExtraBiggest .autoBattleBarHolder .progress-bar::-webkit-scrollbar,
#tooltipDiv.tooltipExtraBiggest .autoBattleBarHolder .innerFightBar::-webkit-scrollbar,
#tooltipDiv.tooltipExtraBiggest .autoBattleBarHolder .noselect::-webkit-scrollbar {
  display: none !important;
  width: 0 !important;
  height: 0 !important;
}

#tooltipDiv.tooltipExtraBiggest .autoBattleBarHolder .progress-bar {
  height: auto !important;
  line-height: 1.25em !important;
}

#tooltipDiv.tooltipExtraBiggest .autoBattleBarHolder .innerFightBar {
  font-size: 1em !important;
  line-height: 1.25em !important;
}

/* Stat columns: 50% width + padding must stay inside the panel */
#tooltipDiv.tooltipExtraBiggest .autoStats {
  box-sizing: border-box;
  max-width: 50%;
}

/* Tab row: wrap/shrink when narrow instead of forcing horizontal overflow */
#tooltipDiv.tooltipExtraBiggest #autoBattleMenuButtons {
  display: flex;
  flex-wrap: wrap;
  max-width: 100%;
  box-sizing: border-box;
}

#tooltipDiv.tooltipExtraBiggest #autoBattleMenuButtons .btn {
  flex: 1 1 auto;
  min-width: 0;
  max-width: 100%;
  box-sizing: border-box;
  white-space: normal;
}

#tooltipDiv.tooltipExtraBiggest #abLevelButtons {
  max-width: 100%;
  box-sizing: border-box;
  overflow-x: hidden;
}

/*
 * Items list: last grid row fills leftover popup height and scrolls.
 */
#tooltipDiv.tooltipExtraBiggest #autoItemsDiv,
#tooltipDiv.tooltipExtraBiggest #autoItemsDiv.modeLg {
  grid-row: 9;
  min-height: 0;
  max-height: none;
  max-width: 100%;
  overflow-y: auto !important;
  overflow-x: hidden;
  box-sizing: border-box;
}

/*
 * Only item rows (divs that contain .autoItemContainer) use flex.
 * Misc tab .abMiscBox children must stay inline-block so presets sit side-by-side.
 */
#tooltipDiv.tooltipExtraBiggest #autoItemsDiv > div:has(.autoItemContainer) {
  display: flex;
  flex-wrap: wrap;
  max-width: 100%;
  box-sizing: border-box;
}

#tooltipDiv.tooltipExtraBiggest #autoItemsDiv .autoItemContainer {
  box-sizing: border-box;
  width: calc(100% / 7);
  max-width: calc(100% / 7);
  margin-left: 0;
  padding: 0 0.05em;
}

#tooltipDiv.tooltipExtraBiggest #autoItemsDiv .autoItem {
  box-sizing: border-box;
  margin-left: 0;
  margin-right: 0;
  max-width: 100%;
}

/*
 * Restore inline-block flow for Misc / Bonuses / Contracts boxes.
 * (An earlier rule set display:flex on all #autoItemsDiv > div, which
 * stacked these full-width instead of in rows.)
 */
#tooltipDiv.tooltipExtraBiggest #autoItemsDiv > .abMiscBox,
#tooltipDiv.tooltipExtraBiggest #autoItemsDiv > .autoBonusBox,
#tooltipDiv.tooltipExtraBiggest #autoItemsDiv > .contractBox {
  display: inline-block;
  box-sizing: border-box;
  vertical-align: top;
}

#tooltipDiv.tooltipExtraBiggest #autoItemsDiv > button.autoBonusBox {
  box-sizing: border-box;
  vertical-align: top;
}
'@
# Normalize newlines + trailing whitespace so LF vs CRLF does not force a rewrite
function Test-TextContentDiffers {
  param([string]$Existing, [string]$Expected)
  $norm = {
    param([string]$s)
    return (($s -replace "`r`n", "`n" -replace "`r", "`n").TrimEnd() + "`n")
  }
  return (& $norm $Existing) -ne (& $norm $Expected)
}
$localCssPath = Join-Path $trimpsPath 'css\kreftus-local.css'
$localCssDir = Split-Path -Parent $localCssPath
if (-not (Test-Path -LiteralPath $localCssDir)) {
  New-Item -ItemType Directory -Path $localCssDir -Force | Out-Null
}
$needWriteLocalCss = -not (Test-Path -LiteralPath $localCssPath)
if (-not $needWriteLocalCss) {
  $needWriteLocalCss = Test-TextContentDiffers ([System.IO.File]::ReadAllText($localCssPath, $utf8NoBom)) $localCssContent
}
if ($needWriteLocalCss) {
  [System.IO.File]::WriteAllText($localCssPath, $localCssContent.TrimEnd() + "`n", $utf8NoBom)
  Write-Host "   [+] Wrote css/kreftus-local.css" -ForegroundColor DarkGray
}

# --- Restore PWA icons (192/512 PNG) from favicon if missing ---
# favicon.ico is a misnamed 16x16 PNG; Chrome/Android install requires explicit 192 and 512 PNG icons
$iconsDir = Join-Path $trimpsPath 'icons'
$icon192 = Join-Path $iconsDir '192.png'
$icon512 = Join-Path $iconsDir '512.png'
$faviconPath = Join-Path $trimpsPath 'favicon.ico'
if ((-not (Test-Path -LiteralPath $icon192)) -or (-not (Test-Path -LiteralPath $icon512))) {
  if (Test-Path -LiteralPath $faviconPath) {
    Add-Type -AssemblyName System.Drawing
    if (-not (Test-Path -LiteralPath $iconsDir)) { New-Item -ItemType Directory -Path $iconsDir -Force | Out-Null }
    $src = [System.Drawing.Image]::FromFile($faviconPath)
    try {
      foreach ($size in @(192, 512)) {
        $outPath = Join-Path $iconsDir ("$size.png")
        if (Test-Path -LiteralPath $outPath) { continue }
        $bmp = New-Object System.Drawing.Bitmap $size, $size
        $g = [System.Drawing.Graphics]::FromImage($bmp)
        $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::NearestNeighbor
        $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::Half
        $g.Clear([System.Drawing.Color]::Transparent)
        $g.DrawImage($src, 0, 0, $size, $size)
        $bmp.Save($outPath, [System.Drawing.Imaging.ImageFormat]::Png)
        $g.Dispose()
        $bmp.Dispose()
        Write-Host "   [+] Wrote icons/$size.png" -ForegroundColor DarkGray
      }
    } finally {
      $src.Dispose()
    }
  }
}

# --- Restore PWA: manifest.json (only write if missing or content differs); use UTF-8 no BOM for stable compare ---
$manifestContent = @'
{
  "name": "Trimps",
  "short_name": "Trimps",
  "start_url": "./",
  "display": "standalone",
  "scope": "./",
  "icons": [
    {
      "src": "icons/192.png",
      "sizes": "192x192",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "icons/512.png",
      "sizes": "512x512",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "favicon.ico",
      "sizes": "16x16",
      "type": "image/png",
      "purpose": "any"
    }
  ]
}
'@
$manifestPath = Join-Path $trimpsPath 'manifest.json'
$needWriteManifest = -not (Test-Path -LiteralPath $manifestPath)
if (-not $needWriteManifest) {
  $needWriteManifest = Test-TextContentDiffers ([System.IO.File]::ReadAllText($manifestPath, $utf8NoBom)) $manifestContent
}
if ($needWriteManifest) {
  [System.IO.File]::WriteAllText($manifestPath, $manifestContent.TrimEnd() + "`n", $utf8NoBom)
  Write-Host "   [+] Wrote manifest.json" -ForegroundColor DarkGray
}

# --- Restore PWA: sw.js (only write if missing or content differs) ---
$swContent = @'
const CACHE_NAME = 'trimps-v14';

const PRECACHE_URLS = [
  './',
  'index.html',
  'manifest.json',
  'css/kreftus-local.css',
  'css/kreftus-local.css?14',
  'icons/192.png',
  'icons/512.png',
  'favicon.ico'
];

self.addEventListener('install', function (event) {
  event.waitUntil(
    caches.open(CACHE_NAME).then(function (cache) {
      return cache.addAll(PRECACHE_URLS);
    }).then(function () {
      return self.skipWaiting();
    })
  );
});

self.addEventListener('activate', function (event) {
  event.waitUntil(
    caches.keys().then(function (cacheNames) {
      return Promise.all(
        cacheNames
          .filter(function (name) { return name !== CACHE_NAME; })
          .map(function (name) { return caches.delete(name); })
      );
    }).then(function () {
      return self.clients.claim();
    })
  );
});

function isStyleRequest(request, url) {
  return request.destination === 'style' || /kreftus-local\.css/i.test(url.pathname + url.search);
}

self.addEventListener('fetch', function (event) {
  var request = event.request;
  if (request.method !== 'GET') return;
  var url = new URL(request.url);
  if (url.origin !== self.location.origin) return;

  // Always prefer fresh CSS so layout fixes are not stuck behind cache-first.
  if (isStyleRequest(request, url)) {
    event.respondWith(
      fetch(request).then(function (response) {
        if (response && response.status === 200) {
          var clone = response.clone();
          caches.open(CACHE_NAME).then(function (cache) {
            cache.put(request, clone);
          });
        }
        return response;
      }).catch(function () {
        return caches.match(request);
      })
    );
    return;
  }

  event.respondWith(
    caches.match(request).then(function (cached) {
      if (cached) return cached;
      return fetch(request).then(function (response) {
        if (!response || response.status !== 200 || response.type === 'error') return response;
        var clone = response.clone();
        caches.open(CACHE_NAME).then(function (cache) {
          cache.put(request, clone);
        });
        return response;
      });
    })
  );
});
'@
$swPath = Join-Path $trimpsPath 'sw.js'
$needWriteSw = -not (Test-Path -LiteralPath $swPath)
if (-not $needWriteSw) {
  $needWriteSw = Test-TextContentDiffers ([System.IO.File]::ReadAllText($swPath, $utf8NoBom)) $swContent
}
if ($needWriteSw) {
  [System.IO.File]::WriteAllText($swPath, $swContent, $utf8NoBom)
  Write-Host "   [+] Wrote sw.js" -ForegroundColor DarkGray
}

# --- Restore PWA + local CSS link in index.html (idempotent; only write if we made edits) ---
$indexPath = Join-Path $trimpsPath 'index.html'
$indexHtml = Get-Content -LiteralPath $indexPath -Raw -Encoding UTF8
$indexChanged = $false
if ($indexHtml -match 'rel="icon" href="favicon\.ico"' -and $indexHtml -notmatch 'type="image/png" sizes="16x16"') {
  $indexHtml = $indexHtml -replace '<link rel="icon" href="favicon\.ico"[^>]*>', '<link rel="icon" href="favicon.ico" type="image/png" sizes="16x16">'
  $indexChanged = $true
}
if ($indexHtml -notmatch 'rel="manifest"') {
  $indexHtml = $indexHtml -replace '(\t<link rel="icon" href="favicon\.ico"[^>]*>)', ('$1' + "`r`n`t<link rel=`"apple-touch-icon`" href=`"icons/192.png`">`r`n`t<link rel=`"manifest`" href=`"manifest.json`">")
  $indexChanged = $true
}
elseif ($indexHtml -notmatch 'apple-touch-icon') {
  $indexHtml = $indexHtml -replace '(<link rel="manifest" href="manifest\.json">)', ("`t<link rel=`"apple-touch-icon`" href=`"icons/192.png`">`r`n" + '$1')
  $indexChanged = $true
}
if ($indexHtml -notmatch 'kreftus-local\.css') {
  $indexHtml = $indexHtml -replace '(<link rel="stylesheet" type="text/css" href="css/style\.css[^"]*"\s*/>)', ('$1' + "`r`n`t<link rel=`"stylesheet`" type=`"text/css`" href=`"css/kreftus-local.css?14`" />")
  $indexChanged = $true
}
elseif ($indexHtml -match 'href="css/kreftus-local\.css(\?[^"]*)?"' -and $indexHtml -notmatch 'kreftus-local\.css\?14"') {
  $indexHtml = $indexHtml -replace 'href="css/kreftus-local\.css(\?[^"]*)?"', 'href="css/kreftus-local.css?14"'
  $indexChanged = $true
}
if ($indexHtml -notmatch 'serviceWorker') {
  $swScript = @'

	<script>
		if ('serviceWorker' in navigator) {
			navigator.serviceWorker.register('sw.js', { scope: './' }).then(
				function (reg) { /* registered */ },
				function (err) { console.warn('SW registration failed:', err); }
			);
		}
	</script>
'@
  $indexHtml = $indexHtml -replace '\s*</body>', "$swScript`r`n`t</body>"
  $indexChanged = $true
}
if ($indexChanged) {
  [System.IO.File]::WriteAllText($indexPath, $indexHtml, $utf8NoBom)
  Write-Host "   [~] Injected PWA / local CSS tags into index.html" -ForegroundColor DarkGray
}

# --- Capture "after" state ---
$after = Get-TrimpsFileHashes -BasePath $trimpsPath

# --- Diff and report ---
$added = [string[]]($after.Keys | Where-Object { -not $before.ContainsKey($_) })
$removed = [string[]]($before.Keys | Where-Object { -not $after.ContainsKey($_) })
$updated = [string[]]($before.Keys | Where-Object { $after.ContainsKey($_) -and $before[$_] -ne $after[$_] })

Write-Header "Update Summary"
[Console]::ResetColor()

if ($added.Count -eq 0 -and $removed.Count -eq 0 -and $updated.Count -eq 0) {
  Write-Output 'No changes.'
  $noChgMsg = "No changes detected. The local trimps/ directory is already up-to-date."
  Write-Host $noChgMsg -NoNewline -ForegroundColor DarkGray
  $summaryCw = [Math]::Max(80, $host.UI.RawUI.BufferSize.Width)
  $summaryRemain = $summaryCw - $noChgMsg.Length
  if ($summaryRemain -gt 0) { Write-Host (" " * $summaryRemain) -NoNewline -BackgroundColor Black }
  Write-Host ""
}
else {
  if ($added.Count -gt 0) {
    Write-Output 'Added:'
    Write-Host "Added ($($added.Count) files):" -ForegroundColor Green
    $added | ForEach-Object { 
      Write-Output "  $_"
      Write-Host "  + $_" -ForegroundColor Green
    }
  }
  if ($removed.Count -gt 0) {
    if ($added.Count -gt 0) { Write-Host "" }
    Write-Output 'Removed:'
    Write-Host "Removed ($($removed.Count) files):" -ForegroundColor Red
    $removed | ForEach-Object { 
      Write-Output "  $_"
      Write-Host "  - $_" -ForegroundColor Red
    }
  }
  if ($updated.Count -gt 0) {
    if ($added.Count -gt 0 -or $removed.Count -gt 0) { Write-Host "" }
    Write-Output 'Updated:'
    Write-Host "Updated ($($updated.Count) files):" -ForegroundColor Yellow
    $updated | ForEach-Object { 
      Write-Output "  $_"
      Write-Host "  ~ $_" -ForegroundColor Yellow
    }
  }
}

$completeMsg = "Update complete."
$cw = [Math]::Max(80, $host.UI.RawUI.BufferSize.Width)
Write-Host "`n$completeMsg" -NoNewline -ForegroundColor DarkGray
$remain = $cw - $completeMsg.Length
if ($remain -gt 0) { Write-Host (" " * $remain) -NoNewline -BackgroundColor Black }
Write-Host ""
[Console]::ResetColor()
