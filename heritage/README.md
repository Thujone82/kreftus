# PDX Heritage Trees

A small, installable PWA field guide to Portland's registered **Heritage Trees**.
It keeps a local database of every tree on the City registry, plots them on an
OpenStreetMap-backed map, and lets you mark which ones you've found, timestamp
the find, and take notes &mdash; all stored privately in your browser.

**No API keys required.** The basemap uses [Leaflet](https://leafletjs.com/)
with the [CARTO Voyager](https://carto.com/basemaps/) tile style (OpenStreetMap
data under a CARTO-designed palette that suits the Pacific Northwest look).
Geocoding is handled offline by the included PowerShell scraper, which calls
OpenStreetMap's free [Nominatim](https://nominatim.openstreetmap.org/) service
and bundles the resolved coordinates alongside the tree list in
`data/trees.json`. The app reads that JSON on first run, so the map populates
instantly without any live geocoding in the browser. Everything then runs
entirely offline from your browser's IndexedDB.

## Features

- **Full registry** &mdash; every tree with Tree #, year added, species / common
  name, location, and "Removed from list in YYYY" where applicable.
- **Interactive map** &mdash; Leaflet + CARTO Voyager basemap (OSM data) with
  colored markers:
  - Green: trees you have marked as **found**
  - Amber: trees not yet found
  - Gray: trees removed from the registry
- **Tap-to-inspect** &mdash; click a marker for an info card with the species
  (linked to Wikipedia), the address, a **Mark as found** button, and a
  **Notes** field.
- **Nearby** &mdash; lists the 10 closest trees to your current location and
  launches walking directions in Google Maps (https deep link that works on
  iOS, Android, Windows, and desktop browsers alike).
- **Check for app update** &mdash; a single action in Settings that refreshes
  the tree database from `data/trees.json` *and* asks the service worker to
  look for a new app shell. New/removed trees are diff-merged **without**
  touching your found marks or notes.
- **Installable PWA** &mdash; runs offline after first load; the service worker
  keeps the app shell fresh without wiping your data on upgrade.

## Requirements

- A modern browser (Chrome, Edge, Firefox, Safari) with IndexedDB and
  service worker support.
- No API keys. No Google Cloud account. No billing setup.
- PowerShell 7+ (`pwsh`) if you want to refresh the tree list with
  `heritage.ps1`. The script uses OpenStreetMap's Nominatim service and needs
  no API key either.

## Setup

Open `heritage/index.html` in a browser, or deploy the entire `heritage/`
folder to your site at `/heritage/`. That's the whole setup.

On first launch the app:

1. Loads `data/trees.json` (~400 trees, coordinates included) into IndexedDB.
2. Initializes the Leaflet map and drops all markers immediately.
3. Runs a **fallback** browser-side geocode pass only for any tree whose
   address couldn't be resolved during the snapshot build (usually none). A top
   progress bar shows `Geocoding trees... N / M` during that rare case.

In the common case a fresh install is interactive within a few seconds,
because the heavy lifting happened offline inside `heritage.ps1`.

### Map layer

- **Library:** Leaflet 1.9.4, loaded from `unpkg.com` and cached by the
  service worker after first use.
- **Tiles:** CARTO Voyager raster tiles
  (`https://{a,b,c,d}.basemaps.cartocdn.com/rastertiles/voyager/...`). CARTO
  explicitly permits free use of their basemaps for personal projects and
  hobby apps. Attribution for OpenStreetMap contributors and CARTO is shown in
  the map's bottom-right and again in **Settings &rarr; Map**.
- **Offline:** tiles are cached cache-first in `heritage-tiles-v1`, so areas
  you've previously panned to stay available without a connection.
- **Swapping tile layers:** change `TILE_URL` / `TILE_ATTRIBUTION` at the top
  of `js/map.js` &mdash; everything else is provider-agnostic.

## Usage

- **Map** &mdash; pan and zoom freely. If your browser allows geolocation, the
  app drops a blue dot for your current location. If you're within 20 mi of
  Portland, the camera zooms to show both you and the nearest tree (with a
  small pad). Otherwise it fits the entire tree set.
- **Tap a marker** &mdash; the info card opens with:
  - The **species** (italic) linked to a Wikipedia search
  - The **common name**, tree #, year added, and location
  - **Mark as found** &rarr; records the current timestamp. Found trees turn
    green. An **Undo** button appears with a localized find timestamp.
  - **Notes** &mdash; type freely; notes autosave when you leave the field.
- **Nearby** &mdash; bottom bar button. Shows the 10 closest trees with
  walking distance. Tapping Navigate opens walking directions in Google Maps.
- **Recenter** &mdash; re-runs the camera logic (user within 20 mi of Portland
  vs. fit-all).
- **Settings (gear icon)** &mdash; view basemap attribution and data stats
  (total / found / removed / last updated).
- **Check for app update** (Settings &rarr; App) &mdash; one tap does two
  things:
  1. Refreshes `data/trees.json` from the network and diff-merges it into the
     local database. New trees are inserted with their pre-geocoded
     coordinates, trees newly flagged "Removed from list in YYYY" turn gray,
     and any canonical-field changes (year/name/location) propagate.
  2. Pings the service worker to look for a new app shell; if one is found,
     the **New app version available &mdash; Reload** banner appears.

  **Your found marks, find dates, and notes are never overwritten** by either
  step. If the network is unavailable the tree refresh is skipped silently
  and the app-version check still runs.

## Data source

The bundled `data/trees.json` is produced by `heritage/heritage.ps1`, which

1. Parses the HTML table on
   [portland.gov/trees/heritage/heritage-trees-year](https://www.portland.gov/trees/heritage/heritage-trees-year).
2. Geocodes every tree's location via OpenStreetMap's free **Nominatim** API
   (no key, 1 request/second per OSM's usage policy). Every request is
   restricted to the Portland metro bounding box (`viewbox=-123.25,45.80,
   -122.25,45.20&bounded=1`) and the returned point is double-checked against
   Portland center &mdash; anything more than 75 mi away is rejected with
   status `OUT_OF_AREA` so stray matches like `"2393 SW Park" &rarr; Austin,
   TX` never get written to the snapshot.
3. Re-uses coordinates from the previous `data/trees.json` for any tree whose
   ID and location haven't changed &mdash; re-runs only touch new/changed
   entries.

From the repository root:

```powershell
pwsh -File heritage/heritage.ps1
```

A fresh run of ~400 trees takes about 7&ndash;8 minutes (pacing is dictated by
Nominatim's 1 req/sec policy). Subsequent runs, once coordinates are cached,
complete in a few seconds.

#### `heritage.ps1` parameter reference

| Parameter         | Type      | Default                                                                 | Purpose                                                                                       |
| ----------------- | --------- | ----------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| `-Url`            | `string`  | `https://www.portland.gov/trees/heritage/heritage-trees-year`           | Page to scrape. Override only if the City moves the registry.                                 |
| `-Output`         | `string`  | `<script dir>\data\trees.json`                                          | Destination snapshot path. Parent directories are created as needed.                          |
| `-UserAgent`      | `string`  | `PDXHeritageTreesScraper (https://github.com/kreftus/kreftus)`          | Value sent to Nominatim. Per OSM policy, include an email or URL that identifies you.         |
| `-DelayMs`        | `int`     | `1100`                                                                  | Milliseconds to wait between Nominatim calls. Raise if you hit rate limits.                   |
| `-NoInteractive`  | `switch`  | off                                                                     | Never prompt on failures. Failing trees are marked `failed` so a later run can retry them.    |
| `-Force`          | `switch`  | off                                                                     | Re-geocode every tree even if the previous snapshot had usable coordinates.                   |
| `-Update`         | `switch`  | off                                                                     | Skip scraping/geocoding entirely; open the existing `trees.json` for per-tree manual editing. |
| `-Tree` *(pos 0)* | `string`  | *(empty)*                                                               | Only meaningful with `-Update`. Jumps straight to this tree number (e.g. `366`, `#366`).      |

Examples:

```powershell
# First ever run (or after deleting data/trees.json): scrape + geocode everything,
# prompting on any failure.
pwsh -File heritage/heritage.ps1

# CI / batch mode: no prompts, mark failures instead.
pwsh -File heritage/heritage.ps1 -NoInteractive

# Re-geocode everything from scratch, slower pacing for a flaky network.
pwsh -File heritage/heritage.ps1 -Force -DelayMs 2000

# Be a good OSM citizen on a hosted runner.
pwsh -File heritage/heritage.ps1 -UserAgent "PDXHeritageTrees (me@example.com)"

# Open update mode and jump straight to tree #366.
pwsh -File heritage/heritage.ps1 -Update 366
```

### Per-tree progress and interactive fallback on failure

The script prints one block per tree, e.g.

```
[ 42/397] #042 Quercus garryana - Oregon white oak
          1234 NE Example St
          geocoding... OK
          45.542100, -122.645800
          1234 NE Example St, Portland, OR 97211, USA
```

If Nominatim returns `ZERO_RESULTS` (or any other non-OK status), the script
stops on that tree and shows:

- the tree's **ID, year, full name, and raw location**;
- direct **address-lookup research links** &mdash; Google Maps, Google Search,
  and OpenStreetMap search;
- a prompt to type an alternate address, open any of the research links in
  your browser (`m` / `g` / `o`), retry the same address (`r`), **mark the
  tree as removed** (`x`, then enter the year), skip (`s` or blank), or quit
  (`q`).

The `x` option is the manual escape hatch for "removed" annotations that the
parser missed. The parser already catches every variant the City uses today
&mdash; `"Removed from list in 2024"`, `"Removed from list 2023"`,
`"Removed in 2025"`, `"Removed 2025"`, inline `"1961 SW Vista Ave (private,
front yard) - removed in 2025"`, parenthesized `"252 NW Maywood Dr (removed in
2015)"`, and combined `"2607 NE Wasco St (right-of-way, removed from list in
2020)"`. If the City invents a new phrasing, `x` lets you handle it on the
spot; the `removed` year you enter is persisted into `trees.json`.

Any address you enter is stored in `geocodeAddress` alongside the resulting
coordinates so later runs keep the manual override until the City's listed
address changes.

### Progress is saved incrementally

`heritage/data/trees.json` is rewritten after **every tree that changes state**
(new geocode, manual address fix, `x`-marked removed, or auto-detected removal).
The final snapshot is also flushed from a `finally` block, so pressing `q` at
the prompt &mdash; or Ctrl+C, closing the window, a network error, or anything
else that ends the run early &mdash; still persists everything you've corrected
so far. Re-running the script picks up exactly where you left off: trees with
valid coordinates (manual or otherwise) are reused from the snapshot and only
the untouched failures prompt you again. Pass `-Force` if you want to ignore
the cache and re-geocode everything from scratch.

### Fixing a single tree with `-Update`

If you notice a tree pinned in the wrong place (for example after a Nominatim
mis-match or a change on the City's page), you don't have to re-scrape
everything &mdash; run the script in update mode and edit just that tree:

```powershell
# Interactive: asks which tree to edit.
pwsh -File heritage/heritage.ps1 -Update

# Jump directly to tree #366 (accepts "366", "#366", or whitespace around it).
pwsh -File heritage/heritage.ps1 -Update 366
```

Update mode skips the HTML fetch entirely. It loads the existing
`data/trees.json`, asks for a tree number (or uses the one passed positionally
after `-Update`), and shows every stored field including
distance-from-Portland. After you finish editing that tree the script returns
to the "Enter tree #" prompt so you can keep working &mdash; press Enter or
`q` to quit. If the tree number supplied positionally isn't in the snapshot,
the script warns and falls through to the normal prompt.

From the edit menu you can:

| Key | Action                                                                 |
| --- | ---------------------------------------------------------------------- |
| `g` | Re-geocode (asks for an address; `!` prefix to bypass the Portland viewbox for rare out-of-metro trees) |
| `c` | Enter `lat`/`lng` directly (blank keeps, `-` clears)                   |
| `a` | Edit the stored `geocodeAddress` string                                |
| `l` | Edit `location` (the City-listed address)                              |
| `n` | Edit `name` (`"Genus species - common name"`)                          |
| `y` | Edit the registry `year`                                               |
| `r` | Set or clear the `removed` year (setting it also marks `skipped-removed`) |
| `x` | Clear all geocoding (coords &rarr; null, status &rarr; `pending`)      |
| `s` | Save and pick another tree                                             |
| `q` | Save and quit                                                          |

Every save goes through the same atomic write that the main loop uses, so
update mode is safe to run while the PWA is open &mdash; reload the page
afterwards to pick up the new position.

### Snapshot format

```json
{
  "sourceUrl": "https://www.portland.gov/trees/heritage/heritage-trees-year",
  "scrapedAt": "2026-04-16T17:13:10Z",
  "count": 397,
  "trees": [
    { "id": "001", "year": 1993,
      "name": "Ulmus americana - American elm",
      "location": "Removed from list in 2024",
      "removed": 2024,
      "lat": null, "lng": null,
      "geocodeStatus": "skipped-removed" },
    { "id": "042", "year": 2001,
      "name": "Quercus garryana - Oregon white oak",
      "location": "1234 NE Example St",
      "removed": null,
      "lat": 45.5421, "lng": -122.6458,
      "geocodeStatus": "ok",
      "geocodeAddress": "1234 NE Example St, Portland, OR" }
  ]
}
```

Tree `id` is the Tree # zero-padded to 3 digits (`"001"`). The script's closing
summary reports how many were cached vs. newly geocoded vs. failed.

> The PWA itself does **not** scrape portland.gov directly: the City's page is
> served from a different origin and CORS blocks a browser fetch. Refreshing
> the snapshot is an intentional offline step.

## Why the tree list uses a bundled JSON (and not a live fetch)

Browsers block cross-origin `fetch()` calls without CORS headers, and the
City's page does not send them. Bundling a pre-scraped JSON file has two
welcome side effects: the app loads instantly even on a slow connection, and
the registry snapshot stays pinned to a version you control rather than
silently drifting.

## Updates and your data

There are two independent "update" concepts:

1. **Tree list updates** (the data) &mdash; driven by the **Check for
   updates** button. Only canonical fields (`year`, `name`, `location`,
   `removed`) can be overwritten. `found`, `foundDate`, `notes`, and existing
   `lat`/`lng` are only changed if a tree's address text changes (in which
   case a re-geocode is queued but your found/notes still survive).
2. **App updates** (the code) &mdash; driven by the service worker. When a new
   version of the app is deployed, a small banner reads **"New app version
   available &mdash; Reload"**. Tapping Reload activates the waiting service
   worker and reloads the page. All user data lives in IndexedDB, which is
   **never** touched by the service worker lifecycle, so your found marks and
   notes survive app upgrades.

## File structure

```
heritage/
  index.html                 # PWA shell (map, nav, settings modal)
  manifest.json              # PWA manifest (start_url /heritage/)
  service-worker.js          # NF shell, SWR assets, NF data, cache-first tiles
  heritage.ps1               # scraper + geocoder -> data/trees.json
  version.ps1                # bumps the app version across files
  README.md                  # this file
  css/
    styles.css               # PNW woodsy theme
  js/
    db.js                    # IndexedDB wrapper (trees, meta stores)
    wiki.js                  # species -> Wikipedia URL
    sync.js                  # fetch trees.json, diff-merge preserving user data
    geocode.js               # fallback Nominatim queue (rarely runs in browser)
    map.js                   # Leaflet map, markers, popup, camera logic
    nearby.js                # 10-nearest list + Google Maps navigate links
    ui.js                    # progress bar, toast, modal helpers
    sw-register.js           # service worker + update banner
    app.js                   # boot & glue
  data/
    trees.json               # bundled pre-geocoded snapshot
  icons/
    icon-192.png / icon-512.png (PWA / favicon / Apple touch)
```

## Troubleshooting

- **Blank map, nothing loads** &mdash; open DevTools &rarr; Network and filter
  on `cartocdn.com`. If those requests fail, a privacy extension
  (uBlock, Brave Shields, etc.) or corporate filter may be blocking the tile
  host. Whitelisting `basemaps.cartocdn.com` and `unpkg.com` fixes it.
- **Tiles load but markers don't** &mdash; open DevTools &rarr; Application
  &rarr; IndexedDB and confirm `heritage-db` / `trees` has rows. If it's
  empty, delete it and reload; the app will repopulate from `data/trees.json`.
- **Marker missing for a tree** &mdash; some addresses ("Removed from list in
  2024", or a non-addressable reference like "NW Corner of SW Park &amp; Main
  (right-of-way)") won't resolve. Re-run `heritage/heritage.ps1`: when it gets
  to that tree it prints the details and research links and lets you enter a
  better address. The edit is persisted in `trees.json` and picked up by the
  app the next time you tap **Settings &rarr; Check for app update**. A
  silent browser-side Nominatim fallback also runs on boot for any tree
  still missing coordinates, so a stuck marker usually self-heals within a
  minute of the next launch.
- **Nominatim 403 / 429 / "blocked"** &mdash; OpenStreetMap rate-limits the
  free tier aggressively. Pass `-UserAgent "PDXHeritageTrees (your@email)"`
  so they can reach you, and consider raising `-DelayMs` or running the
  script in smaller chunks.
- **PWA shows old UI after an update** &mdash; tap **Reload** on the update
  banner, or open **Settings &rarr; Check for app update**. The shell uses a
  network-first strategy specifically to avoid this.

## Versioning

The app version lives in four places &mdash; they must stay in sync or
browsers will serve stale assets after an update:

- `service-worker.js`      &rarr; `const VERSION = 'x.y.z';`
- `manifest.json`          &rarr; `"version": "x.y.z"`
- `js/app.js`              &rarr; `const APP_VERSION = 'x.y.z';`
- `index.html`             &rarr; `?v=x.y.z` cache-busting query on every
  `css/*.css` and `js/*.js` asset reference

These are never bumped automatically; use `version.ps1` whenever you change
the app in a way that warrants a new service worker (asset list, caching
behavior, UI changes the user should see immediately, etc.).

### `version.ps1` &mdash; one-shot version bumper

`version.ps1` rewrites all four locations atomically and preserves each
file's original UTF-8 BOM state. If any target file is missing or its
version marker can't be found, the script throws and exits non-zero without
writing partial updates.

```powershell
# Interactive: prints current version, prompts for the new value.
pwsh -File heritage/version.ps1

# Explicit version (sanitized; see rules below).
pwsh -File heritage/version.ps1 1.1.0
pwsh -File heritage/version.ps1 1.7.12b

# Increment flags (mutually exclusive with each other and with an explicit version).
pwsh -File heritage/version.ps1 -Rev     # 1.2.3 -> 1.2.4
pwsh -File heritage/version.ps1 -Minor   # 1.2.3 -> 1.3.0
pwsh -File heritage/version.ps1 -Major   # 1.2.3 -> 2.0.0

# Show built-in help without touching any files.
pwsh -File heritage/version.ps1 -Help
```

#### Parameter reference

| Parameter          | Type     | Purpose                                                                                                   |
| ------------------ | -------- | --------------------------------------------------------------------------------------------------------- |
| `-Version` *(pos 0)* | `string` | Explicit version string. Sanitized to `[A-Za-z0-9._-]` (spaces and URL-unsafe characters are stripped). |
| `-Rev`             | `switch` | Reads the current version from `service-worker.js` and increments only the revision.                      |
| `-Minor`           | `switch` | Increments the minor and resets the revision to `0`.                                                      |
| `-Major`           | `switch` | Increments the major and resets minor/revision to `0`.                                                    |
| `-Help`            | `switch` | Prints full usage text and exits without modifying any files.                                             |

#### Rules and behavior

- `-Rev`, `-Minor`, and `-Major` are **mutually exclusive** &mdash; specifying
  more than one aborts the run.
- Combining `-Rev`/`-Minor`/`-Major` with an explicit `-Version` value is an
  error; use either an explicit version **or** one increment flag.
- Increment modes parse the existing version as `^(\d+)\.(\d+)\.(\d+)`; any
  trailing suffix is ignored. For example `1.2.3-test` with `-Rev` becomes
  `1.2.4`.
- Explicit versions are normalized by stripping whitespace first, then any
  character outside `A-Z a-z 0-9 . _ -`. If the resulting string is empty the
  script throws.
- When running interactively (no explicit version and no increment flag), a
  blank prompt cancels with `Version must not be blank, update cancelled`
  and exits `1`.
- The script prints step-by-step `old -> new` diffs for every patched file
  and a final green summary. Re-running with the same version is safe &mdash;
  already-current files report `already set to x.y.z` in dark yellow instead
  of overwriting.

## License

Tree data is published by the
[City of Portland](https://www.portland.gov/trees/heritage) and used here for
personal, educational use. The app code in this folder is provided as-is.
