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
  param([string]$Title)
  $width = 48
  $padLeftCount = [int][math]::Floor(($width - $Title.Length) / 2)
  $padRightCount = $width - $padLeftCount - $Title.Length
  
  $border = "=" * $width
  $consoleWidth = $host.UI.RawUI.BufferSize.Width
  $remainingWidth = $consoleWidth - $border.Length
  
  Write-Host ""
  # Top border
  Write-Host $border -NoNewline -ForegroundColor Cyan
  if ($remainingWidth -gt 0) {
    Write-Host (" " * $remainingWidth) -NoNewline -BackgroundColor Black
  }
  Write-Host ""
  
  # Title row
  Write-Host ("=" * $padLeftCount) -NoNewline -ForegroundColor Cyan -BackgroundColor Cyan
  Write-Host $Title -NoNewline -ForegroundColor Black -BackgroundColor Cyan
  Write-Host ("=" * $padRightCount) -NoNewline -ForegroundColor Cyan -BackgroundColor Cyan
  if ($remainingWidth -gt 0) {
    Write-Host (" " * $remainingWidth) -NoNewline -BackgroundColor Black
  }
  Write-Host ""
  
  # Bottom border
  Write-Host $border -NoNewline -ForegroundColor Cyan
  if ($remainingWidth -gt 0) {
    Write-Host (" " * $remainingWidth) -NoNewline -BackgroundColor Black
  }
  Write-Host ""
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
  # --- Copy only files that are new or changed (skip .git, .vscode); skip PWA-owned files only when we already have them ---
  # Do not overwrite index.html, manifest.json, or sw.js when they already exist - we own them and restore below
  $pwaOwnedFiles = @{ 'index.html' = $true; 'manifest.json' = $true; 'sw.js' = $true }
  $cloneBaseLen = (Resolve-Path -LiteralPath $cloneDir).Path.TrimEnd('\').Length + 1
  $copyCount = 0
  $skipCount = 0
  Get-ChildItem -Path $cloneDir -Recurse -File -Force | Where-Object { $_.FullName -notmatch '\.(git|vscode)\\' } | ForEach-Object {
    $rel = $_.FullName.Substring($cloneBaseLen)
    $dest = Join-Path $trimpsPath $rel
    if ($pwaOwnedFiles.ContainsKey($rel) -and (Test-Path -LiteralPath $dest)) {
      Write-Verbose "Skipped (PWA Owned): $rel"
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

Write-Step "Restoring PWA components (manifest.json, sw.js, index.html)..."
# --- Restore PWA: manifest.json (only write if missing or content differs); use UTF-8 no BOM for stable compare ---
$manifestContent = '{"name":"Trimps","short_name":"Trimps","start_url":"./","display":"standalone","scope":"./","icons":[{"src":"favicon.ico","type":"image/x-icon","sizes":"any"}]}'
$manifestPath = Join-Path $trimpsPath 'manifest.json'
$utf8NoBom = [System.Text.UTF8Encoding]::new($false)
$needWriteManifest = -not (Test-Path -LiteralPath $manifestPath)
if (-not $needWriteManifest) {
  $needWriteManifest = [System.IO.File]::ReadAllText($manifestPath, $utf8NoBom) -ne $manifestContent
}
if ($needWriteManifest) {
  [System.IO.File]::WriteAllText($manifestPath, $manifestContent, $utf8NoBom)
  Write-Host "   [+] Wrote manifest.json" -ForegroundColor DarkGray
}

# --- Restore PWA: sw.js (only write if missing or content differs) ---
$swContent = @'
const CACHE_NAME = 'trimps-v1';

const PRECACHE_URLS = [
  './',
  'index.html'
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

self.addEventListener('fetch', function (event) {
  var request = event.request;
  if (request.method !== 'GET') return;
  if (new URL(request.url).origin !== self.location.origin) return;

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
  $needWriteSw = [System.IO.File]::ReadAllText($swPath, $utf8NoBom) -ne $swContent
}
if ($needWriteSw) {
  [System.IO.File]::WriteAllText($swPath, $swContent, $utf8NoBom)
  Write-Host "   [+] Wrote sw.js" -ForegroundColor DarkGray
}

# --- Restore PWA: index.html (idempotent; only write if we made edits) ---
$indexPath = Join-Path $trimpsPath 'index.html'
$indexHtml = Get-Content -LiteralPath $indexPath -Raw -Encoding UTF8
$indexChanged = $false
if ($indexHtml -notmatch 'manifest\.json') {
  $indexHtml = $indexHtml -replace '(\t<link rel="icon" href="favicon\.ico"[^>]*>)', ('$1' + "`r`n`t<link rel=`"manifest`" href=`"manifest.json`">")
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
  Write-Host "   [~] Injected PWA tags into index.html" -ForegroundColor DarkGray
}

# --- Capture "after" state ---
$after = Get-TrimpsFileHashes -BasePath $trimpsPath

# --- Diff and report ---
$added = [string[]]($after.Keys | Where-Object { -not $before.ContainsKey($_) })
$removed = [string[]]($before.Keys | Where-Object { -not $after.ContainsKey($_) })
$updated = [string[]]($before.Keys | Where-Object { $after.ContainsKey($_) -and $before[$_] -ne $after[$_] })

Write-Header "Update Summary"

if ($added.Count -eq 0 -and $removed.Count -eq 0 -and $updated.Count -eq 0) {
  Write-Output 'No changes.'
  Write-Host "No changes detected. The local trimps/ directory is already up-to-date." -ForegroundColor DarkGray
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

Write-Host "`nUpdate complete.`n" -ForegroundColor Cyan
