#Requires -Version 5.1
<#
.SYNOPSIS
    Updates local trimps/ from upstream Trimps.github.io, localises by removing .git, restores PWA components, and reports file changes.
.DESCRIPTION
    Clones https://github.com/Trimps/Trimps.github.io into a temp dir, copies into trimps/ (overwrite),
    removes trimps/.git, restores manifest.json, sw.js, and index.html PWA edits. Reports Added/Removed/Updated files or "No changes."
#>

$ErrorActionPreference = 'Stop'
$trimpsPath = Join-Path $PSScriptRoot 'trimps'

# --- Require git ---
if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Write-Error "git is required but not found. Install Git and ensure it is on PATH."
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
    }
    return $h
}

$before = Get-TrimpsFileHashes -BasePath $trimpsPath

# --- Clone into temp ---
$cloneDir = Join-Path $env:TEMP "trimps_clone_$(Get-Date -Format 'yyyyMMddHHmmss')"
try {
    & git clone --depth 1 'https://github.com/Trimps/Trimps.github.io.git' $cloneDir
    if ($LASTEXITCODE -ne 0) { throw "git clone exited with $LASTEXITCODE" }
} catch {
    if (Test-Path -LiteralPath $cloneDir) { Remove-Item -LiteralPath $cloneDir -Recurse -Force -ErrorAction SilentlyContinue }
    throw "Clone failed: $_"
}

try {
    # --- Ensure trimps exists ---
    if (-not (Test-Path -LiteralPath $trimpsPath)) {
        New-Item -ItemType Directory -Path $trimpsPath -Force | Out-Null
    }
    # --- Copy all items (including .git) from clone into trimps, except .vscode ---
    Get-ChildItem -Path $cloneDir -Force | Where-Object { $_.Name -ne '.vscode' } | ForEach-Object {
        Copy-Item -LiteralPath $_.FullName -Destination $trimpsPath -Recurse -Force
    }
} finally {
    Remove-Item -LiteralPath $cloneDir -Recurse -Force -ErrorAction SilentlyContinue
}

# --- Remove trimps/.git to localise; remove .vscode so it is not present ---
$gitPath = Join-Path $trimpsPath '.git'
if (Test-Path -LiteralPath $gitPath) {
    Remove-Item -LiteralPath $gitPath -Recurse -Force
}
$vscodePath = Join-Path $trimpsPath '.vscode'
if (Test-Path -LiteralPath $vscodePath) {
    Remove-Item -LiteralPath $vscodePath -Recurse -Force
}

# --- Restore PWA: manifest.json ---
$manifestContent = '{"name":"Trimps","short_name":"Trimps","start_url":"./","display":"standalone","scope":"./","icons":[{"src":"favicon.ico","type":"image/x-icon","sizes":"any"}]}'
Set-Content -LiteralPath (Join-Path $trimpsPath 'manifest.json') -Value $manifestContent -Encoding UTF8 -NoNewline

# --- Restore PWA: sw.js ---
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
Set-Content -LiteralPath (Join-Path $trimpsPath 'sw.js') -Value $swContent -Encoding UTF8

# --- Restore PWA: index.html (idempotent) ---
$indexPath = Join-Path $trimpsPath 'index.html'
$indexHtml = Get-Content -LiteralPath $indexPath -Raw -Encoding UTF8
if ($indexHtml -notmatch 'manifest\.json') {
    $indexHtml = $indexHtml -replace '(\t<link rel="icon" href="favicon\.ico"[^>]*>)', ('$1' + "`r`n`t<link rel=`"manifest`" href=`"manifest.json`">")
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
}
Set-Content -LiteralPath $indexPath -Value $indexHtml -Encoding UTF8 -NoNewline

# --- Capture "after" state ---
$after = Get-TrimpsFileHashes -BasePath $trimpsPath

# --- Diff and report ---
$added   = [string[]]($after.Keys | Where-Object { -not $before.ContainsKey($_) })
$removed = [string[]]($before.Keys | Where-Object { -not $after.ContainsKey($_) })
$updated = [string[]]($before.Keys | Where-Object { $after.ContainsKey($_) -and $before[$_] -ne $after[$_] })

if ($added.Count -eq 0 -and $removed.Count -eq 0 -and $updated.Count -eq 0) {
    Write-Output 'No changes.'
} else {
    if ($added.Count -gt 0) {
        Write-Output 'Added:'
        $added | ForEach-Object { Write-Output "  $_" }
    }
    if ($removed.Count -gt 0) {
        Write-Output 'Removed:'
        $removed | ForEach-Object { Write-Output "  $_" }
    }
    if ($updated.Count -gt 0) {
        Write-Output 'Updated:'
        $updated | ForEach-Object { Write-Output "  $_" }
    }
}
