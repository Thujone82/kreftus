# Portland Heritage Trees scraper + geocoder.
#
# 1. Fetches https://www.portland.gov/trees/heritage/heritage-trees-year and
#    parses the table of registered heritage trees.
# 2. For each tree, resolves Portland-addressable coordinates via the free
#    OpenStreetMap Nominatim API (no API key required) and stores lat/lng
#    directly in the bundled snapshot so the PWA doesn't have to geocode live.
# 3. Writes heritage/data/trees.json.
#
# Design notes:
#  - Coordinates are cached from the previous snapshot: if a tree's ID and
#    location text haven't changed, its lat/lng are reused without another
#    API call. If you hand-edit geocodeAddress or lat/lng (or use -Update to
#    enter manual coords), those rows are also kept across runs unless the
#    snapshot row was removed or you pass -Force to re-geocode everything.
#    When the City site text for name / location / year differs from the last
#    snapshot, interactive runs prompt: keep snapshot (stored in `manualKeep`
#    so it won't ask again) or overwrite from the City. `-NoInteractive` keeps
#    the live City text for those deltas without prompting.
#  - Nominatim's Acceptable Use Policy: max 1 request/second, include a
#    descriptive User-Agent that identifies the application. The script sleeps
#    ~1100 ms between calls to stay polite.
#  - Every geocode request is restricted to the Portland metro viewbox and the
#    returned point is distance-checked against Portland center; results that
#    land outside the metro are rejected with status OUT_OF_AREA so stray
#    matches like "2393 SW Park -> Austin, TX" don't poison the snapshot.
#  - On geocoding failure, the script stops and shows tree details plus
#    research URLs (Google Maps, Google Search, OSM) and prompts for an
#    alternate address. User can also open any research link in their default
#    browser, retry the same address, skip, mark removed, or quit.
#  - Use -NoInteractive to mark failures without prompting (for CI / batch).
#    Listing deltas (name/location/year vs snapshot) also default to City text
#    without prompts when -NoInteractive is set.
#  - Use -Force to re-geocode every tree even if cached.
#  - Use -UserAgent to override the Nominatim User-Agent (include an email or
#    URL that identifies you, per OSM policy).
#  - Use -Update to skip scraping and edit individual trees in the existing
#    snapshot: pick a tree by number, then re-geocode it or hand-edit any
#    field (coords, location, name, year, removed, geocodeAddress, ...).
#    An optional tree number can be supplied (positional) to jump straight to
#    that tree without being prompted first, e.g. '-Update 366'.
#
# Usage:
#   pwsh -File heritage/heritage.ps1
#   pwsh -File heritage/heritage.ps1 -Force -NoInteractive
#   pwsh -File heritage/heritage.ps1 -UserAgent "PDXHeritageTrees (me@example.com)"
#   pwsh -File heritage/heritage.ps1 -Update
#   pwsh -File heritage/heritage.ps1 -Update 366

param(
    [string]$Url       = 'https://www.portland.gov/trees/heritage/heritage-trees-year',
    [string]$Output    = '',
    [string]$UserAgent = 'PDXHeritageTreesScraper (https://github.com/kreftus/kreftus)',
    [int]   $DelayMs   = 1100,
    [switch]$NoInteractive,
    [switch]$Force,
    [switch]$Update,
    # Must not be named $Tree: case-insensitive clash with scraper body $tree
    # would coerce PSCustomObject rows to [string] when assigning $tree = ...
    [Parameter(Position = 0)]
    [string]$UpdateTree = ''
)

# Portland metro geofence. Nominatim viewbox format is "left,top,right,bottom"
# i.e. west longitude, north latitude, east longitude, south latitude.
$script:PortlandCenterLat = 45.5152
$script:PortlandCenterLng = -122.6784
$script:PortlandViewBox   = '-123.25,45.80,-122.25,45.20'
$script:PortlandMaxMiles  = 75.0

$ErrorActionPreference = 'Stop'

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
if (-not $Output) { $Output = Join-Path $scriptDir 'data\trees.json' }

# --- Helpers ---------------------------------------------------------------

function Clean-HtmlText {
    param([string]$Html)
    if ($null -eq $Html) { return '' }
    $t = $Html
    $t = [regex]::Replace($t, '<[^>]+>', '')
    $t = $t -replace '&nbsp;', ' '
    $t = $t -replace '&amp;',  '&'
    $t = $t -replace '&lt;',   '<'
    $t = $t -replace '&gt;',   '>'
    $t = $t -replace '&quot;', '"'
    $t = $t -replace '&#39;',  "'"
    $t = $t -replace '&rsquo;', "'"
    $t = $t -replace '&lsquo;', "'"
    $t = $t -replace '&ldquo;', '"'
    $t = $t -replace '&rdquo;', '"'
    $t = $t -replace '&ndash;', '-'
    $t = $t -replace '&mdash;', '-'
    $t = [regex]::Replace($t, '\s+', ' ')
    return $t.Trim()
}

function Normalize-Name {
    param([string]$Raw)
    $text = Clean-HtmlText -Html $Raw
    $text = [regex]::Replace($text, '\s*[-\u2013\u2014]+\s*', ' - ')
    return $text.Trim()
}

function Parse-RemovedYear {
    param([string]$Location)
    if ([string]::IsNullOrWhiteSpace($Location)) { return $null }
    # The City writes "removed" annotations in several shapes, all of which
    # need to be caught so those trees are marked removed and not geocoded:
    #   "Removed from list in 2024"
    #   "Removed from list 2023"
    #   "Removed in 2024"
    #   "Removed 2025"
    #   "1961 SW Vista Ave (private, front yard) - removed in 2025"
    #   "252 NW Maywood Dr (removed in 2015)"
    #   "2607 NE Wasco St (right-of-way, removed from list in 2020)"
    $m = [regex]::Match($Location, 'removed\s*(?:from\s+list\s*)?(?:in\s+)?(\d{4})', 'IgnoreCase')
    if ($m.Success) { return [int]$m.Groups[1].Value }
    return $null
}

function Pad-Id { param([int]$N) return ('{0:D3}' -f $N) }

# Read the tree id field whether the row is a PSCustomObject (JSON) or an
# ordered hashtable (update mode). Plain strings in trees.json (corruption)
# return $null.
function Get-TreeIdScalar {
    param($Tree)
    if ($null -eq $Tree) { return $null }
    if ($Tree -is [string]) { return $null }
    if ($Tree -is [System.Collections.IDictionary]) {
        foreach ($k in @('id', 'Id', 'ID')) {
            $has = if ($Tree -is [hashtable]) { $Tree.ContainsKey($k) } else { $Tree.Contains($k) }
            if ($has) { return $Tree[$k] }
        }
        return $null
    }
    $p = $Tree.PSObject.Properties['id']
    if ($null -ne $p) { return $p.Value }
    $p2 = $Tree.PSObject.Properties['Id']
    if ($null -ne $p2) { return $p2.Value }
    return $null
}

# Canonical id "001".."999" for hashtable keys and comparisons.
function Normalize-TreeId {
    param($Id)
    if ($null -eq $Id) { return $null }
    if ($Id -is [System.Collections.IDictionary]) {
        foreach ($k in @('id', 'Id', 'ID')) {
            $has = if ($Id -is [hashtable]) { $Id.ContainsKey($k) } else { $Id.Contains($k) }
            if ($has) { return (Normalize-TreeId $Id[$k]) }
        }
        return $null
    }
    # Use [UInt32] not [uint]: Windows PowerShell 5.1 has no [uint] accelerator and
    # `-is [uint]` throws "Unable to find type [uint]" instead of evaluating false.
    if ($Id -is [byte] -or $Id -is [sbyte] -or $Id -is [int16] -or $Id -is [uint16] -or
        $Id -is [int] -or $Id -is [UInt32] -or $Id -is [long] -or $Id -is [uint64]) {
        $n = [int64]$Id
        if ($n -lt 1) { return $null }
        return ('{0:D3}' -f $n)
    }
    if ($Id -is [double] -or $Id -is [float] -or $Id -is [decimal]) {
        $d = [double]$Id
        if ([double]::IsNaN($d)) { return $null }
        $n = [int64][math]::Round($d)
        if ($n -lt 1) { return $null }
        return ('{0:D3}' -f $n)
    }
    $s = ([string]$Id).Trim().TrimStart('#')
    if (-not $s) { return $null }
    $digits = $s -replace '[^\d]', ''
    if (-not $digits) { return $null }
    $n2 = 0
    if (-not [int]::TryParse($digits, [ref]$n2) -or $n2 -lt 1) { return $null }
    return ('{0:D3}' -f $n2)
}

function Sanitize-Location {
    param([string]$Loc)
    if (-not $Loc) { return '' }
    $t = $Loc -replace '\([^)]*\)', ' '
    $t = [regex]::Replace($t, '\s+', ' ')
    return $t.Trim()
}

function Build-Address {
    param([string]$Loc)
    $clean = Sanitize-Location $Loc
    if (-not $clean) { return $null }
    if ($clean -match 'Removed from list') { return $null }
    if ($clean -match '\bPortland\b') {
        if ($clean -notmatch '\bOR\b|\bOregon\b') { return "$clean, OR, USA" }
        if ($clean -notmatch '\bUSA\b|\bUS\b|United States') { return "$clean, USA" }
        return $clean
    }
    return "$clean, Portland, OR, USA"
}

# True when the previous snapshot row looks user-tuned so we should not
# replace lat/lng/geocodeAddress (and we carry location forward) when the City
# HTML drifts: custom geocode, manual coords, different listing text vs scrape
# while the snapshot already has coordinates, etc. Use -Force to take City
# text and re-geocode from scratch.
function Test-ManualGeocodeHint {
    param($Prev, $CurrentT)
    if ($null -eq $Prev) { return $false }
    if ($null -ne $Prev.removed) { return $false }
    if ($Prev.geocodeFormatted -eq 'manual coords') { return $true }
    $autoAddr = Build-Address $CurrentT.location
    if ([string]::IsNullOrWhiteSpace($autoAddr)) { return $false }
    $ga = [string]$Prev.geocodeAddress
    if ([string]::IsNullOrWhiteSpace($ga)) { return $false }
    return ($ga.Trim().ToLowerInvariant() -ne $autoAddr.Trim().ToLowerInvariant())
}

function Format-DeltaSnippet {
    param([string]$Text, [int]$MaxLen = 90)
    if ($null -eq $Text) { return '(null)' }
    $t = $Text.Replace("`r", '').Replace("`n", ' | ')
    if ($t.Length -le $MaxLen) { return $t }
    return $t.Substring(0, $MaxLen) + '...'
}

function Get-ManualKeepSet {
    param($Tree)
    $h = @{}
    if ($null -eq $Tree) { return $h }
    $raw = $null
    if ($Tree -is [System.Collections.IDictionary]) {
        foreach ($k in @('manualKeep', 'ManualKeep')) {
            $has = if ($Tree -is [hashtable]) { $Tree.ContainsKey($k) } else { $Tree.Contains($k) }
            if ($has) { $raw = $Tree[$k]; break }
        }
    } else {
        $p = $Tree.PSObject.Properties['manualKeep']
        if ($null -ne $p) { $raw = $p.Value }
    }
    if ($null -eq $raw) { return $h }
    foreach ($item in @($raw)) {
        if ($null -eq $item) { continue }
        $s = ([string]$item).Trim().ToLowerInvariant()
        if ($s -in @('name', 'location', 'year')) { $h[$s] = $true }
    }
    return $h
}

function Set-TreeManualKeep {
    param($Tree, [string[]]$FieldNames)
    $uniq = [System.Collections.Generic.List[string]]::new()
    foreach ($f in $FieldNames) {
        if (-not $f) { continue }
        $n = $f.Trim().ToLowerInvariant()
        if ($n -notin @('name', 'location', 'year')) { continue }
        if (-not $uniq.Contains($n)) { [void]$uniq.Add($n) }
    }
    $arr = @($uniq | Sort-Object)
    $val = if ($arr.Count -gt 0) { $arr } else { $null }
    if ($Tree -is [System.Collections.IDictionary]) {
        $Tree['manualKeep'] = $val
    } else {
        if ($null -ne $Tree.PSObject.Properties['manualKeep']) {
            $Tree.manualKeep = $val
        } else {
            $Tree | Add-Member -NotePropertyName manualKeep -NotePropertyValue $val -Force
        }
    }
}

function Add-TreeManualKeepField {
    param($Tree, [string]$Field)
    $mk = Get-ManualKeepSet -Tree $Tree
    $list = [System.Collections.Generic.List[string]]::new()
    foreach ($x in $mk.Keys) { [void]$list.Add([string]$x) }
    $fn = $Field.Trim().ToLowerInvariant()
    if (-not $list.Contains($fn)) { [void]$list.Add($fn) }
    Set-TreeManualKeep -Tree $Tree -FieldNames @($list)
}

function Remove-TreeManualKeepField {
    param($Tree, [string]$Field)
    $mk = Get-ManualKeepSet -Tree $Tree
    $fn = $Field.Trim().ToLowerInvariant()
    $list = [System.Collections.Generic.List[string]]::new()
    foreach ($x in $mk.Keys) {
        if ([string]$x -ne $fn) { [void]$list.Add([string]$x) }
    }
    Set-TreeManualKeep -Tree $Tree -FieldNames @($list)
}

function Copy-LockedFieldsFromPrev {
    param($Tree, $Prev)
    if ($null -eq $Prev) { return }
    $mk = Get-ManualKeepSet -Tree $Prev
    if ($mk.Count -eq 0) { return }
    foreach ($k in @($mk.Keys)) {
        switch ($k) {
            'name' {
                if ($null -ne $Prev.name) { $Tree.name = $Prev.name }
            }
            'location' {
                if ($null -ne $Prev.location) { $Tree.location = $Prev.location }
            }
            'year' {
                if ($null -ne $Prev.year) { $Tree.year = [int]$Prev.year }
            }
        }
    }
    $names = @($mk.Keys | ForEach-Object { [string]$_ })
    Set-TreeManualKeep -Tree $Tree -FieldNames $names
}

function Test-ListingFieldDelta {
    param($Prev, $Tree, [string]$FieldKey)
    switch ($FieldKey) {
        'name' {
            return ([string]$Prev.name).Trim() -cne ([string]$Tree.name).Trim()
        }
        'location' {
            return ([string]$Prev.location).Trim() -cne ([string]$Tree.location).Trim()
        }
        'year' {
            $py = 0
            $ty = 0
            [void][int]::TryParse([string]$Prev.year, [ref]$py)
            [void][int]::TryParse([string]$Tree.year, [ref]$ty)
            return ($py -ne $ty)
        }
        default { return $false }
    }
}

function Invoke-ListingDeltaWorkflow {
    param($Prev, $Tree, [string]$BannerPrefix, [switch]$NoInteractive, $Stats)
    if ($null -eq $Prev) { return $false }
    if ($null -ne $Prev.removed) { return $false }
    $changed = $false
    Copy-LockedFieldsFromPrev -Tree $Tree -Prev $Prev
    $defs = @(
        @{ Key = 'name';     Label = 'Tree / species name' }
        @{ Key = 'location'; Label = 'City listing (address line)' }
        @{ Key = 'year';     Label = 'Registration year' }
    )
    foreach ($def in $defs) {
        $k = $def.Key
        $mk = Get-ManualKeepSet -Tree $Tree
        if ($mk.ContainsKey($k)) { continue }
        if (-not (Test-ListingFieldDelta -Prev $Prev -Tree $Tree -FieldKey $k)) { continue }
        if ($NoInteractive) {
            $Stats.deltaAutoupdate++
            continue
        }
        Write-Host ''
        Write-Host ("{0}  LISTING DELTA - {1}" -f $BannerPrefix, $def.Label) -ForegroundColor Yellow
        $snap = switch ($k) {
            'location' { [string]$Prev.location }
            'name'     { [string]$Prev.name }
            'year'     { "$($Prev.year)" }
        }
        $city = switch ($k) {
            'location' { [string]$Tree.location }
            'name'     { [string]$Tree.name }
            'year'     { "$($Tree.year)" }
        }
        Write-Host ("  Snapshot:  {0}" -f (Format-DeltaSnippet $snap)) -ForegroundColor DarkGray
        Write-Host ("  City site: {0}" -f (Format-DeltaSnippet $city)) -ForegroundColor DarkGray
        Write-Host '  [K]eep snapshot (manual; saved to manualKeep - no future prompt for this field)' -ForegroundColor Cyan
        Write-Host '  [U]pdate from City (overwrite this field from the live scrape)' -ForegroundColor Cyan
        $choice = ''
        while ($true) {
            Write-Host '  > ' -NoNewline -ForegroundColor Cyan
            $ans = Read-Host
            if ($null -eq $ans) { $ans = '' }
            $choice = $ans.Trim().ToLowerInvariant()
            if (-not $choice) {
                Write-Host '  Please enter K or U.' -ForegroundColor Red
                continue
            }
            $c0 = $choice.Substring(0, 1)
            if ($c0 -eq 'k' -or $c0 -eq 'u') { break }
            Write-Host '  Please enter K or U.' -ForegroundColor Red
        }
        if ($choice.StartsWith('k')) {
            switch ($k) {
                'name'     { $Tree.name     = $Prev.name }
                'location' { $Tree.location = $Prev.location }
                'year'     { if ($null -ne $Prev.year) { $Tree.year = [int]$Prev.year } }
            }
            Add-TreeManualKeepField -Tree $Tree -Field $k
            $Stats.deltaKept++
            $changed = $true
        } else {
            Remove-TreeManualKeepField -Tree $Tree -Field $k
            $Stats.deltaUpdated++
            $changed = $true
        }
    }
    return $changed
}

function Get-MilesFromPortland {
    param([double]$Lat, [double]$Lng)
    $R = 3958.7613  # Earth radius in miles
    $lat1 = $script:PortlandCenterLat * [Math]::PI / 180.0
    $lat2 = $Lat * [Math]::PI / 180.0
    $dLat = ($Lat - $script:PortlandCenterLat) * [Math]::PI / 180.0
    $dLng = ($Lng - $script:PortlandCenterLng) * [Math]::PI / 180.0
    $a = [Math]::Sin($dLat / 2) * [Math]::Sin($dLat / 2) +
         [Math]::Cos($lat1) * [Math]::Cos($lat2) *
         [Math]::Sin($dLng / 2) * [Math]::Sin($dLng / 2)
    $c = 2 * [Math]::Atan2([Math]::Sqrt($a), [Math]::Sqrt(1 - $a))
    return $R * $c
}

function Invoke-Geocode {
    param(
        [string]$Address,
        [string]$UserAgent,
        [switch]$Unbounded   # allow caller to override the Portland viewbox
    )
    if (-not $Address) { return @{ ok = $false; status = 'EMPTY_ADDRESS' } }
    try {
        $encoded = [uri]::EscapeDataString($Address)
        $geoUrl = "https://nominatim.openstreetmap.org/search?q=$encoded&format=json&limit=1&countrycodes=us&addressdetails=1"
        if (-not $Unbounded) {
            # Restrict Nominatim's search to the Portland metro bounding box.
            # bounded=1 tells Nominatim to treat the viewbox as a hard filter
            # rather than a ranking hint, so unrelated matches in other states
            # (e.g. "2393 SW Park" -> Austin, TX) are never returned.
            $geoUrl += "&viewbox=$($script:PortlandViewBox)&bounded=1"
        }
        $headers = @{ 'User-Agent' = $UserAgent; 'Accept' = 'application/json' }
        $resp = Invoke-RestMethod -Uri $geoUrl -Headers $headers -Method Get -TimeoutSec 30
        if ($resp -and @($resp).Count -gt 0) {
            $first = @($resp)[0]
            $lat = [double]$first.lat
            $lng = [double]$first.lon
            # Belt-and-suspenders: even with bounded=1, reject anything that
            # ends up more than $PortlandMaxMiles from Portland center.
            $miles = Get-MilesFromPortland -Lat $lat -Lng $lng
            if ($miles -gt $script:PortlandMaxMiles) {
                return @{
                    ok      = $false
                    status  = 'OUT_OF_AREA'
                    message = ("result {0:F6}, {1:F6} is {2:F1} mi from Portland: {3}" -f $lat, $lng, $miles, $first.display_name)
                    lat     = $lat
                    lng     = $lng
                    formatted = $first.display_name
                }
            }
            return @{
                ok        = $true
                lat       = $lat
                lng       = $lng
                formatted = $first.display_name
                status    = 'OK'
                miles     = $miles
            }
        }
        return @{ ok = $false; status = 'ZERO_RESULTS' }
    } catch {
        $msg = $_.Exception.Message
        # Distinguish rate-limit style failures so we can back off.
        if ($msg -match '429' -or $msg -match 'Too Many Requests') {
            return @{ ok = $false; status = 'RATE_LIMITED'; message = $msg }
        }
        return @{ ok = $false; status = 'REQUEST_ERROR'; message = $msg }
    }
}

function Get-ResearchUrls {
    param($Tree)
    $locEnc = [uri]::EscapeDataString($Tree.location)
    return [ordered]@{
        GMaps  = "https://www.google.com/maps/search/?api=1&query=$locEnc"
        Google = "https://www.google.com/search?q=$locEnc"
        OSM    = "https://www.openstreetmap.org/search?query=$locEnc"
    }
}

function Show-TreeDetails {
    param($Tree, [string]$LastAddress)
    Write-Host "    ------------------------------------------------------------" -ForegroundColor DarkGray
    Write-Host "    Tree #$($Tree.id) (added $($Tree.year))" -ForegroundColor Yellow
    Write-Host "      Name:     $($Tree.name)"
    Write-Host "      Location: $($Tree.location)"
    if ($LastAddress) { Write-Host "      Tried:    $LastAddress" -ForegroundColor DarkGray }
    $urls = Get-ResearchUrls -Tree $Tree
    Write-Host "    Research links (address lookup only):" -ForegroundColor Yellow
    Write-Host "      [m] Google Maps:   $($urls.GMaps)"
    Write-Host "      [g] Google:        $($urls.Google)"
    Write-Host "      [o] OpenStreetMap: $($urls.OSM)"
    Write-Host "    ------------------------------------------------------------" -ForegroundColor DarkGray
}

function Save-Snapshot {
    param($Trees, [string]$OutputPath, [string]$SourceUrl)
    $outDir = Split-Path -Parent $OutputPath
    if (-not (Test-Path $outDir)) { New-Item -ItemType Directory -Path $outDir -Force | Out-Null }
    foreach ($tr in @($Trees)) {
        if ($null -eq $tr) {
            throw 'Save-Snapshot: null tree row (refusing to write corrupt JSON).'
        }
        if ($tr -is [string]) {
            throw 'Save-Snapshot: string tree row (in-memory snapshot is corrupt). Restore heritage/data/trees.json from backup or git, then re-run.'
        }
        $idNorm = Normalize-TreeId (Get-TreeIdScalar $tr)
        if (-not $idNorm) {
            throw 'Save-Snapshot: tree row has no usable id (refusing to write corrupt JSON).'
        }
        if ($tr -is [System.Collections.IDictionary]) {
            $tr['id'] = $idNorm
        } else {
            $tr.id = $idNorm
        }
    }
    $snapshot = [ordered]@{
        sourceUrl = $SourceUrl
        scrapedAt = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
        count     = @($Trees).Count
        trees     = $Trees
    }
    # Write to a temp file and swap it in so Ctrl+C mid-write can't corrupt the
    # JSON. Windows PowerShell's Move-Item -Force has a long-standing quirk
    # ("Cannot create a file when that file already exists.") when the target
    # exists, so we prefer [IO.File]::Replace which is an atomic NTFS swap.
    $tmp  = $OutputPath + '.tmp'
    $json = $snapshot | ConvertTo-Json -Depth 8
    try {
        $probe = $json | ConvertFrom-Json
        if ($probe.trees) {
            $first = @($probe.trees)[0]
            if ($first -is [string]) {
                throw 'Round-trip validation failed: trees[0] is a string after ConvertTo-Json (would corrupt JSON on disk).'
            }
        }
    } catch {
        throw "Save-Snapshot: serialized JSON failed validation: $_"
    }
    Set-Content -Path $tmp -Value $json -Encoding UTF8
    $fullTmp = (Resolve-Path -LiteralPath $tmp).Path
    $fullDst = if (Test-Path -LiteralPath $OutputPath) {
        (Resolve-Path -LiteralPath $OutputPath).Path
    } else {
        [System.IO.Path]::GetFullPath($OutputPath)
    }
    try {
        if (Test-Path -LiteralPath $fullDst) {
            [System.IO.File]::Replace($fullTmp, $fullDst, $null)
        } else {
            [System.IO.File]::Move($fullTmp, $fullDst)
        }
    } catch {
        # Fallback: delete destination then move. Not atomic, but robust on
        # file systems that don't implement Replace (e.g. some network shares).
        if (Test-Path -LiteralPath $fullDst) { Remove-Item -LiteralPath $fullDst -Force }
        Move-Item -LiteralPath $fullTmp -Destination $fullDst -Force
    }
}

function ConvertTo-TreeHash {
    param($PsTree)
    if ($PsTree -is [string]) {
        throw 'trees[] row is a plain string (heritage/data/trees.json is corrupt). Restore from git or a backup, then retry.'
    }
    # Turn a PSCustomObject (loaded via ConvertFrom-Json) into an ordered
    # hashtable so it serializes identically on write and supports free
    # property assignment/addition.
    $h = [ordered]@{}
    $fields = @(
        'id','year','name','location','removed',
        'lat','lng',
        'geocodeStatus','geocodeAddress','geocodeFormatted','geocodeError',
        'manualKeep'
    )
    foreach ($f in $fields) { $h[$f] = $null }
    foreach ($prop in $PsTree.PSObject.Properties) {
        $h[$prop.Name] = $prop.Value
    }
    return $h
}

function Format-NullableNumber {
    param($Value, [string]$Fmt = 'F6')
    if ($null -eq $Value) { return '(null)' }
    try { return ([double]$Value).ToString($Fmt) } catch { return "$Value" }
}

function Show-TreeForUpdate {
    param($Tree)
    Write-Host ""
    Write-Host ("#{0}  {1}" -f $Tree.id, $Tree.name) -ForegroundColor White
    Write-Host ("  year:             {0}" -f $Tree.year)
    Write-Host ("  location:         {0}" -f $Tree.location)
    $removed = if ($null -eq $Tree.removed) { '(none)' } else { "$($Tree.removed)" }
    Write-Host ("  removed:          {0}" -f $removed)
    if ($null -ne $Tree.lat -and $null -ne $Tree.lng) {
        $miles = Get-MilesFromPortland -Lat ([double]$Tree.lat) -Lng ([double]$Tree.lng)
        $coordsLine = "  coords:           {0}, {1}  ({2:F1} mi from Portland)" -f `
            (Format-NullableNumber $Tree.lat), (Format-NullableNumber $Tree.lng), $miles
        if ($miles -gt $script:PortlandMaxMiles) {
            Write-Host $coordsLine -ForegroundColor Red
        } else {
            Write-Host $coordsLine -ForegroundColor DarkGray
        }
    } else {
        Write-Host "  coords:           (null, null)" -ForegroundColor DarkGray
    }
    Write-Host ("  geocodeStatus:    {0}" -f $Tree.geocodeStatus)
    if ($Tree.geocodeAddress)   { Write-Host ("  geocodeAddress:   {0}" -f $Tree.geocodeAddress) }
    if ($Tree.geocodeFormatted) { Write-Host ("  geocodeFormatted: {0}" -f $Tree.geocodeFormatted) -ForegroundColor DarkGray }
    if ($Tree.geocodeError)     { Write-Host ("  geocodeError:     {0}" -f $Tree.geocodeError) -ForegroundColor DarkRed }
}

function Read-OptionalString {
    param([string]$Label, [string]$Current)
    Write-Host ("  New {0} (blank = keep '{1}'): " -f $Label, $Current) -NoNewline -ForegroundColor Cyan
    $answer = Read-Host
    if ($null -eq $answer) { return $Current }
    if (-not $answer) { return $Current }
    return $answer.Trim()
}

function Read-OptionalInt {
    param([string]$Label, $Current, [int]$Min = 1, [int]$Max = 2100)
    $label2 = if ($null -eq $Current) { '(null)' } else { "$Current" }
    Write-Host ("  New {0} (blank = keep {1}, '-' = clear): " -f $Label, $label2) -NoNewline -ForegroundColor Cyan
    $answer = Read-Host
    if ($null -eq $answer) { return $Current }
    $trim = $answer.Trim()
    if (-not $trim) { return $Current }
    if ($trim -eq '-') { return $null }
    $n = 0
    if ([int]::TryParse($trim, [ref]$n) -and $n -ge $Min -and $n -le $Max) { return $n }
    Write-Host "  Not a valid integer in range; keeping previous value." -ForegroundColor Yellow
    return $Current
}

function Read-Coordinate {
    param([string]$Label, $Current, [double]$Min, [double]$Max)
    $label2 = if ($null -eq $Current) { '(null)' } else { ([double]$Current).ToString('F6') }
    Write-Host ("  New {0} (blank = keep {1}, '-' = clear): " -f $Label, $label2) -NoNewline -ForegroundColor Cyan
    $answer = Read-Host
    if ($null -eq $answer) { return $Current }
    $trim = $answer.Trim()
    if (-not $trim) { return $Current }
    if ($trim -eq '-') { return $null }
    $d = 0.0
    if ([double]::TryParse($trim, [ref]$d) -and $d -ge $Min -and $d -le $Max) { return $d }
    Write-Host "  Not a valid coordinate; keeping previous value." -ForegroundColor Yellow
    return $Current
}

function Read-LatMaybePair {
    # Reads the 'lat' field but also accepts a combined "lat, lng" paste
    # (like Google Maps' share format, e.g. "45.574557, -122.666124"). When
    # a valid pair is entered the caller can skip the separate lng prompt.
    # Returns a hashtable { lat = <value-or-null>; lng = <value-or-null>; gotBoth = <bool> }.
    param(
        $CurrentLat,
        [double]$LatMin, [double]$LatMax,
        [double]$LngMin, [double]$LngMax
    )
    $latStr = if ($null -eq $CurrentLat) { '(null)' } else { ([double]$CurrentLat).ToString('F6') }
    Write-Host ("  New lat (blank = keep {0}, '-' = clear, 'lat, lng' sets both): " -f $latStr) -NoNewline -ForegroundColor Cyan
    $answer = Read-Host
    if ($null -eq $answer)        { return @{ lat = $CurrentLat; lng = $null; gotBoth = $false } }
    $trim = $answer.Trim()
    if (-not $trim)               { return @{ lat = $CurrentLat; lng = $null; gotBoth = $false } }
    if ($trim -eq '-')            { return @{ lat = $null;       lng = $null; gotBoth = $false } }

    if ($trim.Contains(',')) {
        $parts = $trim.Split(@(','), 2)
        $latPart = $parts[0].Trim()
        $lngPart = if ($parts.Length -gt 1) { $parts[1].Trim() } else { '' }
        $dLat = 0.0; $dLng = 0.0
        $latOk = [double]::TryParse($latPart, [ref]$dLat) -and $dLat -ge $LatMin -and $dLat -le $LatMax
        $lngOk = [double]::TryParse($lngPart, [ref]$dLng) -and $dLng -ge $LngMin -and $dLng -le $LngMax
        if ($latOk -and $lngOk) {
            return @{ lat = $dLat; lng = $dLng; gotBoth = $true }
        }
        Write-Host "  Paired 'lat, lng' invalid or out of range; keeping previous lat and asking for lng separately." -ForegroundColor Yellow
        return @{ lat = $CurrentLat; lng = $null; gotBoth = $false }
    }

    $d = 0.0
    if ([double]::TryParse($trim, [ref]$d) -and $d -ge $LatMin -and $d -le $LatMax) {
        return @{ lat = $d; lng = $null; gotBoth = $false }
    }
    Write-Host "  Not a valid coordinate; keeping previous value." -ForegroundColor Yellow
    return @{ lat = $CurrentLat; lng = $null; gotBoth = $false }
}

function Update-Tree-Regeocode {
    param($Tree, [string]$UserAgent, [int]$DelayMs)
    $startAddress = if ($Tree.geocodeAddress) { $Tree.geocodeAddress } else { Build-Address $Tree.location }
    Write-Host ("  Current geocode address: {0}" -f $startAddress) -ForegroundColor DarkGray
    Write-Host "  Enter address to geocode (blank = use current, '!' = unbounded search): " -NoNewline -ForegroundColor Cyan
    $answer = Read-Host
    $unbounded = $false
    $address = $startAddress
    if ($answer) {
        $trim = $answer.Trim()
        if ($trim -eq '!') {
            $unbounded = $true
            Write-Host "  (Unbounded mode) address: " -NoNewline -ForegroundColor Cyan
            $address = (Read-Host).Trim()
            if (-not $address) { Write-Host "  no address provided; aborting." -ForegroundColor Yellow; return }
        } elseif ($trim.StartsWith('!')) {
            $unbounded = $true
            $address = $trim.Substring(1).Trim()
        } else {
            $address = $trim
        }
    }
    if (-not $address) { Write-Host "  no address to geocode; aborting." -ForegroundColor Yellow; return }

    Write-Host ("  geocoding '{0}'..." -f $address) -NoNewline
    if ($unbounded) {
        $result = Invoke-Geocode -Address $address -UserAgent $UserAgent -Unbounded
    } else {
        $result = Invoke-Geocode -Address $address -UserAgent $UserAgent
    }
    Start-Sleep -Milliseconds $DelayMs
    if ($result.ok) {
        $Tree.lat              = $result.lat
        $Tree.lng              = $result.lng
        $Tree.geocodeStatus    = 'ok'
        $Tree.geocodeAddress   = $address
        $Tree.geocodeFormatted = $result.formatted
        $Tree.geocodeError     = $null
        Write-Host " OK" -ForegroundColor Green
        Write-Host ("  -> {0:F6}, {1:F6}  ({2:F1} mi from Portland)" -f $result.lat, $result.lng, $result.miles) -ForegroundColor DarkGreen
        if ($result.formatted) { Write-Host ("  -> {0}" -f $result.formatted) -ForegroundColor DarkGray }
    } else {
        Write-Host (" {0}" -f $result.status) -ForegroundColor Red
        if ($result.message) { Write-Host ("  {0}" -f $result.message) -ForegroundColor DarkRed }
        Write-Host "  Tree not updated." -ForegroundColor Yellow
    }
}

function Resolve-TreeEntry {
    # Parse a user-entered tree number ("366", "#366", "  366 ", ...) and find
    # the matching tree in the loaded snapshot. Emits diagnostic output for
    # invalid/missing entries and returns $null when no tree matches.
    param(
        $Trees,
        [string]$Entry,
        [switch]$Silent
    )
    if ($null -eq $Entry) { return $null }
    $trimmed = $Entry.Trim().TrimStart('#')
    if (-not $trimmed) { return $null }
    $n = 0
    if (-not [int]::TryParse($trimmed, [ref]$n) -or $n -lt 1) {
        if (-not $Silent) { Write-Host "  Not a valid tree number." -ForegroundColor Yellow }
        return $null
    }
    $id = "{0:D3}" -f $n
    foreach ($t in $Trees) {
        $tid = Normalize-TreeId (Get-TreeIdScalar $t)
        if ($tid -eq $id) { return $t }
    }
    if (-not $Silent) { Write-Host ("  No tree #{0} in snapshot." -f $id) -ForegroundColor Yellow }
    return $null
}

function Invoke-UpdateMode {
    param([string]$OutputPath, [string]$UserAgent, [int]$DelayMs, [string]$SourceUrl, [string]$InitialTree = '')

    if (-not (Test-Path $OutputPath)) {
        Write-Host ("No existing snapshot at {0}. Run the scraper first." -f $OutputPath) -ForegroundColor Red
        return
    }

    Write-Host ""
    Write-Host ("== Update mode ==  loading {0}" -f $OutputPath) -ForegroundColor Cyan
    $raw = Get-Content -Path $OutputPath -Raw | ConvertFrom-Json
    $sourceUrlFromFile = if ($raw.sourceUrl) { [string]$raw.sourceUrl } else { $SourceUrl }
    $rawTrees = $raw.trees
    if ($null -eq $rawTrees) { $rawTrees = $raw.Trees }
    if ($null -eq $rawTrees) {
        Write-Error ("{0} has no trees or Trees array." -f $OutputPath)
        exit 1
    }
    $trees = @()
    foreach ($pt in $rawTrees) {
        if ($pt -is [string]) {
            Write-Error ("{0} is corrupt: trees[] contains plain strings (not JSON objects). Restore from git or a backup, then retry." -f $OutputPath)
            exit 1
        }
        $trees += ,(ConvertTo-TreeHash -PsTree $pt)
    }
    Write-Host ("   loaded {0} trees" -f $trees.Count)

    # If the caller supplied a tree number (e.g. -Update 366), try to jump
    # straight into editing that tree on the first pass. On success we skip
    # the initial prompt; on failure we fall through to the normal loop.
    $pendingTree = $null
    if ($InitialTree) {
        $pendingTree = Resolve-TreeEntry -Trees $trees -Entry $InitialTree
        if ($null -ne $pendingTree) {
            Write-Host ("   starting with tree #{0}" -f $pendingTree.id) -ForegroundColor DarkGray
        }
    }

    while ($true) {
        if ($pendingTree) {
            $tree = $pendingTree
            $pendingTree = $null
        } else {
            Write-Host ""
            Write-Host "Enter tree # to update (blank or 'q' to quit): " -NoNewline -ForegroundColor Cyan
            $entry = Read-Host
            if ($null -eq $entry) { break }
            $trimmed = $entry.Trim()
            if (-not $trimmed -or $trimmed -eq 'q' -or $trimmed -eq 'Q') { break }
            $tree = Resolve-TreeEntry -Trees $trees -Entry $trimmed
            if ($null -eq $tree) { continue }
        }

        $dirty = $false
        # Label required: `break` inside `switch` only leaves the switch, not this
        # loop. Option [s] must exit the loop so the outer prompt can ask for the
        # next tree #.
        :treeEdit while ($true) {
            Show-TreeForUpdate -Tree $tree
            Write-Host ""
            Write-Host "  What to update?" -ForegroundColor Yellow
            Write-Host "    [g] re-geocode (prompts for address, bounded to Portland)"
            Write-Host "    [c] enter lat/lng directly"
            Write-Host "    [a] edit geocodeAddress string"
            Write-Host "    [l] edit location (City-listed address; sets manualKeep.location)"
            Write-Host "    [n] edit name (sets manualKeep.name)"
            Write-Host "    [y] edit year (sets manualKeep.year)"
            Write-Host "    [r] set/clear removed year"
            Write-Host "    [x] clear geocoding (mark pending for next scraper run)"
            Write-Host "    [s] save and pick another tree"
            Write-Host "    [q] save and quit update mode"
            Write-Host "  > " -NoNewline -ForegroundColor Cyan
            $cmd = Read-Host
            if ($null -eq $cmd) { break }
            switch ($cmd.Trim().ToLowerInvariant()) {
                'g' {
                    Update-Tree-Regeocode -Tree $tree -UserAgent $UserAgent -DelayMs $DelayMs
                    $dirty = $true
                }
                'c' {
                    $latResult = Read-LatMaybePair -CurrentLat $tree.lat `
                        -LatMin 20.0 -LatMax 55.0 -LngMin -140.0 -LngMax -100.0
                    $newLat = $latResult.lat
                    if ($latResult.gotBoth) {
                        $newLng = $latResult.lng
                        Write-Host ("  lng parsed from paired input: {0}" -f ([double]$newLng).ToString('F6')) -ForegroundColor DarkGray
                    } else {
                        $newLng = Read-Coordinate -Label 'lng' -Current $tree.lng -Min -140.0 -Max -100.0
                    }
                    $tree.lat = $newLat
                    $tree.lng = $newLng
                    if ($null -ne $newLat -and $null -ne $newLng) {
                        $tree.geocodeStatus = 'ok'
                        $tree.geocodeError  = $null
                        if (-not $tree.geocodeFormatted) { $tree.geocodeFormatted = 'manual coords' }
                        $miles = Get-MilesFromPortland -Lat ([double]$newLat) -Lng ([double]$newLng)
                        if ($miles -gt $script:PortlandMaxMiles) {
                            Write-Host ("  WARNING: {0:F1} mi from Portland center; the PWA will still show it." -f $miles) -ForegroundColor Yellow
                        }
                    }
                    $dirty = $true
                }
                'a' { $tree.geocodeAddress = Read-OptionalString -Label 'geocodeAddress' -Current ([string]$tree.geocodeAddress); $dirty = $true }
                'l' {
                    $newLocation = Read-OptionalString -Label 'location' -Current ([string]$tree.location)
                    if ($newLocation -cne [string]$tree.location) {
                        $tree.location = $newLocation
                        Add-TreeManualKeepField -Tree $tree -Field 'location'
                    } else {
                        $tree.location = $newLocation
                    }
                    $dirty = $true
                }
                'n' {
                    $newName = Read-OptionalString -Label 'name' -Current ([string]$tree.name)
                    if ($newName -cne [string]$tree.name) {
                        $tree.name = $newName
                        Add-TreeManualKeepField -Tree $tree -Field 'name'
                    } else {
                        $tree.name = $newName
                    }
                    $dirty = $true
                }
                'y' {
                    $newYear = Read-OptionalInt -Label 'year' -Current $tree.year -Min 1800 -Max 2100
                    $oldYear = if ($null -eq $tree.year) { $null } else { [int]$tree.year }
                    if ($null -eq $oldYear -and $null -eq $newYear) {
                        $tree.year = $newYear
                    } elseif ($oldYear -ne $newYear) {
                        $tree.year = $newYear
                        Add-TreeManualKeepField -Tree $tree -Field 'year'
                    } else {
                        $tree.year = $newYear
                    }
                    $dirty = $true
                }
                'r' {
                    $tree.removed = Read-OptionalInt -Label 'removed year' -Current $tree.removed -Min 1800 -Max 2100
                    if ($null -ne $tree.removed) { $tree.geocodeStatus = 'skipped-removed' }
                    $dirty = $true
                }
                'x' {
                    $tree.lat              = $null
                    $tree.lng              = $null
                    $tree.geocodeStatus    = 'pending'
                    $tree.geocodeFormatted = $null
                    $tree.geocodeError     = $null
                    Write-Host "  geocoding cleared." -ForegroundColor DarkGray
                    $dirty = $true
                }
                's' {
                    if ($dirty) {
                        Save-Snapshot -Trees $trees -OutputPath $OutputPath -SourceUrl $sourceUrlFromFile
                        Write-Host "  saved." -ForegroundColor Green
                    }
                    break treeEdit
                }
                'q' { if ($dirty) { Save-Snapshot -Trees $trees -OutputPath $OutputPath -SourceUrl $sourceUrlFromFile; Write-Host "  saved." -ForegroundColor Green }; return }
                default { Write-Host "  Unknown option." -ForegroundColor Yellow }
            }
        }
    }

    Save-Snapshot -Trees $trees -OutputPath $OutputPath -SourceUrl $sourceUrlFromFile
    Write-Host ""
    Write-Host ("Wrote: {0}" -f (Resolve-Path $OutputPath))
}

function Prompt-RemovedYear {
    $defaultYear = (Get-Date).Year
    while ($true) {
        Write-Host "    Year removed (default $defaultYear): " -NoNewline -ForegroundColor Cyan
        $answer = Read-Host
        if (-not $answer) { return $defaultYear }
        $year = 0
        if ([int]::TryParse($answer.Trim(), [ref]$year) -and $year -ge 1900 -and $year -le 2100) {
            return $year
        }
        Write-Host "    Please enter a 4-digit year (1900-2100) or blank for $defaultYear." -ForegroundColor Red
    }
}

function Prompt-Alternate {
    param($Tree, [string]$LastAddress)
    $urls = Get-ResearchUrls -Tree $Tree
    while ($true) {
        Write-Host "    Enter alt address | [m]aps [g]oogle [o]sm [r]etry [x]mark removed [s]kip [q]uit > " -NoNewline -ForegroundColor Cyan
        $answer = Read-Host
        if ($null -eq $answer) { return $null }
        $trimmed = $answer.Trim()
        if (-not $trimmed -or $trimmed -eq 's') { return $null }
        switch ($trimmed.ToLowerInvariant()) {
            'q' { throw 'User quit.' }
            'r' { return $LastAddress }
            'm' { Start-Process $urls.GMaps;  continue }
            'g' { Start-Process $urls.Google; continue }
            'o' { Start-Process $urls.OSM;    continue }
            'x' {
                $y = Prompt-RemovedYear
                return @{ action = 'mark-removed'; year = $y }
            }
            default { return $trimmed }
        }
    }
}

# --- Update mode ----------------------------------------------------------

if ($Update) {
    Invoke-UpdateMode -OutputPath $Output -UserAgent $UserAgent -DelayMs $DelayMs -SourceUrl $Url -InitialTree $UpdateTree
    return
}

# --- Fetch -----------------------------------------------------------------

Write-Host ""
Write-Host "== Fetching heritage tree list ==" -ForegroundColor Cyan
Write-Host "   $Url"
try {
    $response = Invoke-WebRequest -Uri $Url -UseBasicParsing -UserAgent $UserAgent
} catch {
    Write-Error "Failed to fetch ${Url}: $_"
    exit 1
}
$html = $response.Content

$tbodyMatch = [regex]::Match($html, '<tbody>(?<body>.*?)</tbody>', [System.Text.RegularExpressions.RegexOptions]::Singleline)
if (-not $tbodyMatch.Success) { Write-Error "Could not locate <tbody>."; exit 1 }
$tbody = $tbodyMatch.Groups['body'].Value

$rowPattern = '<tr[^>]*>\s*<th[^>]*>(?<id>[^<]+?)</th>\s*<td[^>]*>(?<year>[^<]*?)</td>\s*<td[^>]*>(?<name>.*?)</td>\s*<td[^>]*>(?<loc>.*?)</td>\s*</tr>'
$rowMatches = [regex]::Matches($tbody, $rowPattern, [System.Text.RegularExpressions.RegexOptions]::Singleline)
Write-Host "   Parsed $($rowMatches.Count) rows." -ForegroundColor DarkGray

$trees = New-Object System.Collections.Generic.List[object]
$seenIds = @{}
$skippedParse = 0
foreach ($m in $rowMatches) {
    $idText   = Clean-HtmlText -Html $m.Groups['id'].Value
    $yearText = Clean-HtmlText -Html $m.Groups['year'].Value
    $nameRaw  = $m.Groups['name'].Value
    $locRaw   = $m.Groups['loc'].Value

    $idNum = 0
    if (-not [int]::TryParse(($idText -replace '[^\d]', ''), [ref]$idNum) -or $idNum -le 0) { $skippedParse++; continue }
    $yearNum = 0
    [void][int]::TryParse(($yearText -replace '[^\d]', ''), [ref]$yearNum)

    $nameNorm = Normalize-Name -Raw $nameRaw
    $location = Clean-HtmlText -Html $locRaw
    $removed  = Parse-RemovedYear -Location $location

    $id = Pad-Id -N $idNum
    if ($seenIds.ContainsKey($id)) { $skippedParse++; continue }
    $seenIds[$id] = $true

    $tree = [pscustomobject][ordered]@{
        id               = $id
        year             = $yearNum
        name             = $nameNorm
        location         = $location
        removed          = $removed
        lat              = $null
        lng              = $null
        geocodeStatus    = $null
        geocodeAddress   = $null
        geocodeFormatted = $null
        geocodeError     = $null
        manualKeep       = $null
    }
    [void]$trees.Add($tree)
}
$trees = @($trees | Sort-Object -Property @{ Expression = {
    $nk = Normalize-TreeId (Get-TreeIdScalar $_)
    if (-not $nk) { return [int]::MaxValue }
    return [int]$nk
}})

# --- Load previous snapshot for coordinate cache ---------------------------

$prevMap = @{}
$prevCount = 0
if (Test-Path $Output) {
    try {
        $prev = Get-Content -Raw -Path $Output | ConvertFrom-Json
        $prevTrees = $prev.trees
        if ($null -eq $prevTrees) { $prevTrees = $prev.Trees }
        if ($prevTrees) {
            $prevCount = $prevTrees.Count
            foreach ($pt in $prevTrees) {
                if ($null -eq $pt) { continue }
                if ($pt -is [string]) {
                    Write-Warning "Previous snapshot has a string trees[] row (corrupt JSON). Skipping that cache entry. Restore heritage/data/trees.json from git or a backup."
                    continue
                }
                $key = Normalize-TreeId (Get-TreeIdScalar $pt)
                if (-not $key) {
                    Write-Warning 'Previous snapshot has a tree row with missing or unusable id; skipping that cache entry.'
                    continue
                }
                $prevMap[$key] = $pt
            }
            Write-Host "   Cached $prevCount records from previous snapshot." -ForegroundColor DarkGray
        }
    } catch {
        Write-Warning "Could not parse previous snapshot: $_"
    }
}

# --- Geocode loop ----------------------------------------------------------

Write-Host ""
Write-Host "== Geocoding $($trees.Count) trees via Nominatim ==" -ForegroundColor Cyan
Write-Host "   User-Agent: $UserAgent" -ForegroundColor DarkGray
Write-Host "   Delay between requests: ${DelayMs}ms" -ForegroundColor DarkGray

$total = $trees.Count
$i = 0
$stats = [ordered]@{
    reused           = 0
    geocoded         = 0
    manual           = 0
    failed           = 0
    skipped          = 0
    added            = 0
    removedNow       = 0
    deltaKept        = 0
    deltaUpdated     = 0
    deltaAutoupdate  = 0
}

$userQuit = $false

try {
    foreach ($t in $trees) {
        $i++
        $idKey = Normalize-TreeId (Get-TreeIdScalar $t)
        if (-not $idKey) {
            Write-Error ('Geocode loop: tree row has no usable id after parse: {0}' -f ($t | Out-String))
            exit 1
        }
        $prefix = "[{0,3}/{1,3}] #{2}" -f $i, $total, $idKey
        Write-Host ""
        Write-Host ("{0}  {1}" -f $prefix, $t.name) -ForegroundColor White

        $prev = $null
        if ($prevMap.ContainsKey($idKey)) { $prev = $prevMap[$idKey] }
        if (-not $prev) { $stats.added++ }

        $deltaDirty = $false
        if ($null -ne $prev) {
            $deltaDirty = Invoke-ListingDeltaWorkflow -Prev $prev -Tree $t -BannerPrefix $prefix -NoInteractive:$NoInteractive -Stats $stats
            if ($deltaDirty) {
                Save-Snapshot -Trees $trees -OutputPath $Output -SourceUrl $Url
            }
        }

        # Persist user-supplied data across runs. If the previous snapshot
        # already has a good result for this tree (same location text, coords
        # or removed year) carry that forward so the user doesn't have to
        # re-correct anything they fixed in an earlier run.
        $locSame = ($null -ne $prev) -and ((Sanitize-Location $prev.location) -eq (Sanitize-Location $t.location))
        $manualHint = Test-ManualGeocodeHint -Prev $prev -CurrentT $t
        $prevHasCoords = ($null -ne $prev) -and ($null -ne $prev.lat) -and ($null -ne $prev.lng)
        $reusePrevGeocode = (-not $Force) -and ($null -ne $prev) -and ($null -eq $prev.removed) -and
            $prevHasCoords -and ($locSame -or $manualHint)

        # 1) Manual "mark removed": City text doesn't yet mention removal but
        #    the previous snapshot has a removed year -> keep it.
        if ($null -eq $t.removed -and $null -ne $prev -and $null -ne $prev.removed -and $locSame) {
            $t.removed = [int]$prev.removed
        }

        # Removed trees - no geocode. Save only if this changed state.
        if ($null -ne $t.removed) {
            if ($t.location) { Write-Host ("        {0}" -f $t.location) -ForegroundColor DarkGray }
            $wasAlreadyRemoved = ($null -ne $prev) -and ($null -ne $prev.removed) -and ($locSame) -and ($prev.geocodeStatus -eq 'skipped-removed')
            $t.geocodeStatus = 'skipped-removed'
            Write-Host ("        removed from list in {0}; no geocode" -f $t.removed) -ForegroundColor DarkGray
            if ($prev -and $null -eq $prev.removed) { $stats.removedNow++ }
            $stats.skipped++
            if (-not $wasAlreadyRemoved) { Save-Snapshot -Trees $trees -OutputPath $Output -SourceUrl $Url }
            continue
        }

        # 2) Previous successful geocode (including a manual-address fix) ->
        #    reuse the coords and any saved manual address when the City-listed
        #    location still matches (sanitized) OR the snapshot shows a custom
        #    geocode / manual coords (see Test-ManualGeocodeHint). -Force skips.
        if ($reusePrevGeocode) {
            $t.lat = [double]$prev.lat
            $t.lng = [double]$prev.lng
            $t.geocodeStatus = 'ok'
            if ($prev.PSObject.Properties['geocodeAddress'])   { $t.geocodeAddress   = $prev.geocodeAddress }
            if ($prev.PSObject.Properties['geocodeFormatted']) { $t.geocodeFormatted = $prev.geocodeFormatted }
            $t.geocodeError = $null
            # Only substitute snapshot `location` when the user marked it manualKeep.
            # Do not use locSame alone: Sanitize-Location can match when raw text still
            # differs (e.g. case), and then copying prev would undo a listing-delta [U].
            $mkLoc = Get-ManualKeepSet -Tree $t
            if ($mkLoc.ContainsKey('location') -and $null -ne $prev.location -and
                -not [string]::IsNullOrWhiteSpace([string]$prev.location)) {
                $t.location = $prev.location
            }
            $tag = if ($manualHint) { ' (manual)' } elseif ($prev.geocodeAddress -and $prev.geocodeAddress -ne (Build-Address $t.location)) { ' (manual)' } else { '' }
            if ($t.location) { Write-Host ("        {0}" -f $t.location) -ForegroundColor DarkGray }
            Write-Host ("        cached{0}  -> {1:F6}, {2:F6}" -f $tag, $t.lat, $t.lng) -ForegroundColor DarkGreen
            $stats.reused++
            continue
        }

        # Manual geocode hint but no reusable coords (rare): use snapshot listing
        # only when `location` is in manualKeep (same rule as reuse block above).
        if ($manualHint -and (Get-ManualKeepSet -Tree $t).ContainsKey('location') -and ($null -ne $prev) -and
            ($null -eq $prev.removed) -and ($null -ne $prev.location) -and
            -not [string]::IsNullOrWhiteSpace([string]$prev.location)) {
            $t.location = $prev.location
        }
        if ($t.location) { Write-Host ("        {0}" -f $t.location) -ForegroundColor DarkGray }

        $address = Build-Address $t.location
        if (-not $address) {
            $t.geocodeStatus = 'skipped-no-address'
            Write-Host "        no usable address; skipping" -ForegroundColor DarkGray
            $stats.skipped++
            if (-not ($prev -and $prev.geocodeStatus -eq 'skipped-no-address' -and $locSame)) {
                Save-Snapshot -Trees $trees -OutputPath $Output -SourceUrl $Url
            }
            continue
        }

        Write-Host "        geocoding..." -NoNewline
        $result = Invoke-Geocode -Address $address -UserAgent $UserAgent
        $currentAddress = $address
        $triedManually = $false
        $markedRemoved = $false

        while (-not $result.ok) {
            Write-Host (" {0}" -f $result.status) -ForegroundColor Red
            if ($result.message) { Write-Host ("        {0}" -f $result.message) -ForegroundColor DarkRed }

            if ($result.status -eq 'RATE_LIMITED') {
                Write-Host "        rate-limited by Nominatim; backing off 10s..." -ForegroundColor Yellow
                Start-Sleep -Seconds 10
                Write-Host "        retrying..." -NoNewline
                $result = Invoke-Geocode -Address $currentAddress -UserAgent $UserAgent
                continue
            }

            if ($NoInteractive) { break }

            Show-TreeDetails -Tree $t -LastAddress $currentAddress
            try {
                $answer = Prompt-Alternate -Tree $t -LastAddress $currentAddress
            } catch {
                Write-Host "Quit requested - saving progress and exiting." -ForegroundColor Yellow
                $userQuit = $true
                break
            }
            if ($null -eq $answer -or ($answer -is [string] -and -not $answer)) { break }
            if ($answer -is [hashtable] -and $answer.action -eq 'mark-removed') {
                $t.removed = [int]$answer.year
                $t.geocodeStatus = 'skipped-removed'
                $t.geocodeError = $null
                Write-Host ("        marked removed in {0}; no geocode" -f $t.removed) -ForegroundColor DarkGray
                if ($prev -and $null -eq $prev.removed) { $stats.removedNow++ }
                $stats.skipped++
                $markedRemoved = $true
                break
            }
            $currentAddress = [string]$answer
            $triedManually = $true
            Write-Host ("        geocoding '{0}'..." -f $currentAddress) -NoNewline
            $result = Invoke-Geocode -Address $currentAddress -UserAgent $UserAgent
        }

        if ($userQuit) {
            Save-Snapshot -Trees $trees -OutputPath $Output -SourceUrl $Url
            break
        }

        if ($markedRemoved) {
            Save-Snapshot -Trees $trees -OutputPath $Output -SourceUrl $Url
            Start-Sleep -Milliseconds $DelayMs
            continue
        }

        if ($result.ok) {
            $t.lat = $result.lat
            $t.lng = $result.lng
            $t.geocodeStatus = 'ok'
            $t.geocodeAddress = $currentAddress
            $t.geocodeFormatted = $result.formatted
            $t.geocodeError = $null
            $suffix = if ($triedManually) { ' (manual address)' } else { '' }
            Write-Host (" OK{0}" -f $suffix) -ForegroundColor Green
            Write-Host ("        {0:F6}, {1:F6}" -f $result.lat, $result.lng) -ForegroundColor DarkGreen
            if ($result.formatted) { Write-Host ("        {0}" -f $result.formatted) -ForegroundColor DarkGray }
            $stats.geocoded++
            if ($triedManually) { $stats.manual++ }
        } else {
            $t.geocodeStatus = 'failed'
            $t.geocodeError = $result.status
            Write-Host "        marked failed; re-run to retry" -ForegroundColor Yellow
            $stats.failed++
        }

        Save-Snapshot -Trees $trees -OutputPath $Output -SourceUrl $Url
        Start-Sleep -Milliseconds $DelayMs
    }
}
finally {
    # Always flush the latest state on the way out (including Ctrl+C / errors).
    try { Save-Snapshot -Trees $trees -OutputPath $Output -SourceUrl $Url } catch { }
}

# --- Summary ---------------------------------------------------------------

Write-Host ""
Write-Host "*** Portland Heritage Trees ***" -ForegroundColor Green
Write-Host ("Wrote:        {0}" -f (Resolve-Path $Output))
Write-Host ("Trees:        {0}" -f $total)
Write-Host ("Geocoded:     {0} new" -f $stats.geocoded) -ForegroundColor Green
Write-Host ("  (of those)  {0} used a manual address" -f $stats.manual) -ForegroundColor DarkGray
Write-Host ("Cached:       {0} reused from previous snapshot" -f $stats.reused) -ForegroundColor DarkGray
Write-Host ("Skipped:      {0} removed / unaddressable" -f $stats.skipped) -ForegroundColor DarkGray
$failColor = if ($stats.failed -gt 0) { 'Yellow' } else { 'Green' }
Write-Host ("Failed:       {0}" -f $stats.failed) -ForegroundColor $failColor
if ($prevCount -gt 0) {
    Write-Host ("Added:        {0} new vs previous snapshot" -f $stats.added) -ForegroundColor DarkGray
    Write-Host ("Newly removed:{0}" -f $stats.removedNow) -ForegroundColor DarkGray
}
if (($stats.deltaKept + $stats.deltaUpdated + $stats.deltaAutoupdate) -gt 0) {
    Write-Host ("Listing delta: {0} kept (manualKeep), {1} updated from City, {2} auto City (-NoInteractive)" -f `
        $stats.deltaKept, $stats.deltaUpdated, $stats.deltaAutoupdate) -ForegroundColor DarkGray
}
