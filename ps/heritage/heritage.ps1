# Portland Heritage Trees Scraper
# Fetches https://www.portland.gov/trees/heritage/heritage-trees-year, parses the
# table of registered heritage trees, and writes heritage/data/trees.json as a
# bundled snapshot for the PDX Heritage Trees PWA.
#
# Columns in the source table:
#   Tree #   Year   Name (species - common)   Location
#
# The "Location" column may contain "Removed from list in YYYY" for retired trees.
#
# Output structure:
#   {
#     "sourceUrl": "...",
#     "scrapedAt":  "2026-04-16T15:22:00Z",
#     "trees": [
#       { "id": "001", "year": 1993, "name": "Ulmus americana - American elm",
#         "location": "Removed from list in 2024", "removed": 2024 },
#       ...
#     ]
#   }
#
# Usage:
#   pwsh -File ps/heritage/heritage.ps1            # default output
#   pwsh -File ps/heritage/heritage.ps1 -Output path\to\trees.json

param(
    [string]$Url    = 'https://www.portland.gov/trees/heritage/heritage-trees-year',
    [string]$Output = ''
)

# Resolve repo root (this file lives in <repo>/ps/heritage/)
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot  = Resolve-Path (Join-Path $scriptDir '..\..')
if (-not $Output) {
    $Output = Join-Path $repoRoot 'heritage\data\trees.json'
}

# ---- Helpers ---------------------------------------------------------------

function Clean-HtmlText {
    param([string]$Html)
    if ($null -eq $Html) { return '' }
    $t = $Html
    # Drop all tags
    $t = [regex]::Replace($t, '<[^>]+>', '')
    # Decode common entities
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
    # Collapse whitespace
    $t = [regex]::Replace($t, '\s+', ' ')
    return $t.Trim()
}

function Normalize-Name {
    param([string]$Raw)
    # Raw HTML of the Name cell uses <em>Species</em> then a dash then common name.
    # Sometimes the dash or trailing space is inside the <em> tag. We strip tags
    # and re-normalize to "Species - Common" with exactly one " - " separator.
    $text = Clean-HtmlText -Html $Raw
    # Replace any number of hyphens/en-dashes/em-dashes surrounded by optional
    # whitespace with a single " - ".
    $text = [regex]::Replace($text, '\s*[-\u2013\u2014]+\s*', ' - ')
    return $text.Trim()
}

function Parse-RemovedYear {
    param([string]$Location)
    if ([string]::IsNullOrWhiteSpace($Location)) { return $null }
    $m = [regex]::Match($Location, 'Removed from list in\s+(\d{4})', 'IgnoreCase')
    if ($m.Success) { return [int]$m.Groups[1].Value }
    return $null
}

function Pad-Id {
    param([int]$N)
    return ('{0:D3}' -f $N)
}

# ---- Fetch -----------------------------------------------------------------

Write-Host "Fetching $Url ..." -ForegroundColor Cyan
try {
    $response = Invoke-WebRequest -Uri $Url -UseBasicParsing -UserAgent 'Mozilla/5.0 (PDXHeritageTreesScraper)'
} catch {
    Write-Error "Failed to fetch ${Url}: $_"
    exit 1
}
$html = $response.Content

# ---- Locate tbody ----------------------------------------------------------

$tbodyMatch = [regex]::Match($html, '<tbody>(?<body>.*?)</tbody>', [System.Text.RegularExpressions.RegexOptions]::Singleline)
if (-not $tbodyMatch.Success) {
    Write-Error "Could not locate <tbody> in the heritage trees page."
    exit 1
}
$tbody = $tbodyMatch.Groups['body'].Value

# ---- Parse rows ------------------------------------------------------------

# Row pattern - tolerant of variations:
#   <th ...>ID</th><td ...>YEAR</td><td ...>NAME_HTML</td><td ...>LOCATION_HTML</td>
# Some rows use <th scope="row"> or <th colspan="1" rowspan="1"> or plain <th>.
$rowPattern = '<tr[^>]*>\s*<th[^>]*>(?<id>[^<]+?)</th>\s*<td[^>]*>(?<year>[^<]*?)</td>\s*<td[^>]*>(?<name>.*?)</td>\s*<td[^>]*>(?<loc>.*?)</td>\s*</tr>'
$rowMatches = [regex]::Matches($tbody, $rowPattern, [System.Text.RegularExpressions.RegexOptions]::Singleline)

Write-Host "Found $($rowMatches.Count) candidate rows." -ForegroundColor DarkGray

$trees = New-Object System.Collections.Generic.List[object]
$seenIds = @{}
$skipped = 0

foreach ($m in $rowMatches) {
    $idText   = Clean-HtmlText -Html $m.Groups['id'].Value
    $yearText = Clean-HtmlText -Html $m.Groups['year'].Value
    $nameRaw  = $m.Groups['name'].Value
    $locRaw   = $m.Groups['loc'].Value

    $idNum = 0
    if (-not [int]::TryParse(($idText -replace '[^\d]', ''), [ref]$idNum) -or $idNum -le 0) {
        $skipped++; continue
    }
    $yearNum = 0
    [void][int]::TryParse(($yearText -replace '[^\d]', ''), [ref]$yearNum)

    $name     = Normalize-Name -Raw $nameRaw
    $location = Clean-HtmlText -Html $locRaw
    $removed  = Parse-RemovedYear -Location $location

    $id = Pad-Id -N $idNum
    if ($seenIds.ContainsKey($id)) {
        # Duplicate id - keep the first occurrence, count as skipped.
        $skipped++; continue
    }
    $seenIds[$id] = $true

    $tree = [ordered]@{
        id       = $id
        year     = $yearNum
        name     = $name
        location = $location
        removed  = $removed
    }
    $trees.Add([pscustomobject]$tree) | Out-Null
}

# Sort by numeric id
$sorted = $trees | Sort-Object -Property @{Expression = { [int]$_.id }}

# ---- Compare with previous snapshot (if any) ------------------------------

$prevCount = 0
$added     = 0
$removedNow = 0
if (Test-Path $Output) {
    try {
        $prev = Get-Content -Raw -Path $Output | ConvertFrom-Json
        if ($prev.trees) {
            $prevCount = $prev.trees.Count
            $prevMap = @{}
            foreach ($pt in $prev.trees) { $prevMap[$pt.id] = $pt }
            foreach ($t in $sorted) {
                if (-not $prevMap.ContainsKey($t.id)) { $added++ }
                elseif ($null -eq $prevMap[$t.id].removed -and $null -ne $t.removed) { $removedNow++ }
            }
        }
    } catch {
        Write-Warning "Could not parse previous snapshot at ${Output}: $_"
    }
}

# ---- Write JSON ------------------------------------------------------------

$outDir = Split-Path -Parent $Output
if (-not (Test-Path $outDir)) { New-Item -ItemType Directory -Path $outDir -Force | Out-Null }

$snapshot = [ordered]@{
    sourceUrl = $Url
    scrapedAt = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
    count     = $sorted.Count
    trees     = $sorted
}

$json = $snapshot | ConvertTo-Json -Depth 6
# ConvertTo-Json escapes forward slashes as \/ on some PS versions - leave as-is.
Set-Content -Path $Output -Value $json -Encoding UTF8

# ---- Summary ---------------------------------------------------------------

Write-Host ""
Write-Host "*** Portland Heritage Trees ***" -ForegroundColor Green
Write-Host ("Wrote:       {0}" -f (Resolve-Path $Output)) -ForegroundColor White
Write-Host ("Trees:       {0}" -f $sorted.Count) -ForegroundColor White
Write-Host ("Skipped:     {0}" -f $skipped) -ForegroundColor DarkGray
$removedCount = ($sorted | Where-Object { $null -ne $_.removed }).Count
Write-Host ("Removed:     {0}" -f $removedCount) -ForegroundColor DarkGray
if ($prevCount -gt 0) {
    Write-Host ("Previous:    {0}" -f $prevCount) -ForegroundColor DarkGray
    Write-Host ("New:         {0}" -f $added) -ForegroundColor Yellow
    Write-Host ("Newly removed: {0}" -f $removedNow) -ForegroundColor Yellow
}
