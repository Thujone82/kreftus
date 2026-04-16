# PDX Heritage Trees

A small, installable PWA field guide to Portland's registered **Heritage Trees**.
It keeps a local database of every tree on the City registry, plots them on a
Google Map, and lets you mark which ones you've found, timestamp the find, and
take notes &mdash; all stored privately in your browser.

The initial tree list is scraped from
[portland.gov/trees/heritage/heritage-trees-year](https://www.portland.gov/trees/heritage/heritage-trees-year)
by the included PowerShell script (`heritage/heritage.ps1`), which **also
geocodes every address via the free OpenStreetMap Nominatim API** (no API key
needed) and bundles the result as a JSON snapshot (`data/trees.json`). The
app reads that JSON on first run and &mdash; because the coordinates are
already resolved &mdash; the map populates instantly without any live
geocoding in the browser. Everything then runs entirely offline from your
browser's IndexedDB.

## Features

- **Full registry** &mdash; every tree with Tree #, year added, species / common
  name, location, and "Removed from list in YYYY" where applicable.
- **Interactive map** &mdash; Google Maps with colored markers:
  - Green: trees you have marked as **found**
  - Amber: trees not yet found
  - Gray: trees removed from the registry
- **Tap-to-inspect** &mdash; click a marker for an info card with the species
  (linked to Wikipedia), the address, a **Mark as found** button, and a
  **Notes** field.
- **Nearby** &mdash; lists the 10 closest trees to your current location and
  launches walking directions in Google Maps on tap.
- **Check for updates** &mdash; re-reads the bundled snapshot and diff-merges
  any new or removed trees into your local database **without** touching your
  found marks or notes.
- **Installable PWA** &mdash; runs offline after first load; the service worker
  keeps the app shell fresh without wiping your data on upgrade.

## Requirements

- A modern browser (Chrome, Edge, Firefox, Safari) with IndexedDB,
  `localStorage`, and service worker support.
- A free **Google Maps API key** with **Maps JavaScript API** enabled
  (that's the only Google API the app needs &mdash; coordinates are resolved
  by the scraper via OpenStreetMap).
- PowerShell 7+ (`pwsh`) to refresh the tree list with `heritage.ps1`. The
  script uses OpenStreetMap's Nominatim service and needs no API key of any
  kind.

Google's monthly free credit covers normal map use by a long way. There are
no runtime geocoding charges because the scraper resolves every address
offline.

## Setup

### 1. Get a free Google Maps API key

1. Go to the
   [Google Cloud Console &rarr; Credentials](https://console.cloud.google.com/google/maps-apis/credentials)
   and sign in.
2. Create a new project (or select an existing one) and click
   **Create Credentials &rarr; API Key**.
3. Enable **Maps JavaScript API** on the project
   (**APIs &amp; Services &rarr; Library**). That's the only API the PWA uses.
4. *(Recommended)* Restrict the key to *Maps JavaScript API* and to your site's
   URL under **Application restrictions** and **API restrictions**.

### 2. First launch

1. Open `heritage/index.html` in a browser (or deploy the `heritage/`
   folder to your site at `/heritage/`).
2. The welcome screen walks you through pasting your API key. The key is
   stored in `localStorage` under `pdxHeritageGoogleApiKey` and is only ever
   sent to Google.
3. After you save the key, the app:
   - Loads `data/trees.json` (~400 trees, coordinates included) into IndexedDB.
   - Initializes the map and drops all markers immediately.
   - Runs a **fallback** geocode pass only for any tree whose address couldn't
     be resolved during the snapshot build (usually none). A top progress bar
     shows `Geocoding trees... N / M` during that rare case.

In the common case a fresh install is interactive within a few seconds,
because the heavy lifting happened offline inside `heritage.ps1`.

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
  walking distance. Tapping a row opens Google Maps walking directions.
- **Recenter** &mdash; re-runs the camera logic (user within 20 mi of Portland
  vs. fit-all).
- **Check updates** &mdash; re-reads `data/trees.json` and diff-merges:
  - New trees are inserted and queued for geocoding.
  - Trees that are now marked "Removed from list in YYYY" get their `removed`
    year set and their markers turn gray.
  - Changed addresses trigger a re-geocode (lat/lng will be refreshed).
  - **Your found marks, find dates, and notes are never overwritten.**
- **Settings (gear icon)** &mdash; edit your API key, view stats
  (total / found / removed / last updated), retry failed geocodes, and check
  for an app update.

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

# Optional flags
#   -Force           re-geocode every tree even if cached
#   -NoInteractive   don't prompt on failures (mark them failed instead)
#   -Update          skip scraping; edit one tree at a time (see below)
#   -DelayMs 1100    delay between Nominatim requests (default 1100 ms)
#   -UserAgent "..." override the User-Agent (please include an email or URL
#                    so OSM can reach you if there's a problem)
```

A fresh run of ~400 trees takes about 7&ndash;8 minutes (pacing is dictated by
Nominatim's 1 req/sec policy). Subsequent runs, once coordinates are cached,
complete in a few seconds.

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
pwsh -File heritage/heritage.ps1 -Update
```

Update mode skips the HTML fetch entirely. It loads the existing
`data/trees.json`, asks for a tree number (e.g. `158`, `#158`, or `1`), and
shows every stored field including distance-from-Portland. You can then
choose what to do:

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
  index.html                 # PWA shell (setup screen, map, nav, modal)
  manifest.json              # PWA manifest (start_url /heritage/)
  service-worker.js          # network-first shell, SWR assets, NF data
  heritage.ps1               # scraper + geocoder -> data/trees.json
  version.ps1                # bumps the app version across files
  README.md                  # this file
  css/
    styles.css               # PNW woodsy theme
  js/
    db.js                    # IndexedDB wrapper (trees, meta stores)
    wiki.js                  # species -> Wikipedia URL
    sync.js                  # fetch trees.json, diff-merge preserving user data
    geocode.js               # fallback Geocoding API queue (rarely runs)
    map.js                   # map, markers, info window, camera logic
    nearby.js                # 10-nearest list + walking nav
    ui.js                    # progress bar, toast, modal helpers
    sw-register.js           # service worker + update banner
    app.js                   # boot & glue
  data/
    trees.json               # bundled pre-geocoded snapshot
  icons/
    icon.svg / icon-192.svg / icon-512.svg
```

## Troubleshooting

- **"That doesn't look like a Google Maps API key"** &mdash; keys start with
  `AIza`. If yours is different (e.g. a legacy format), enter it anyway; the
  check is advisory.
- **Map doesn't load** &mdash; open the browser console. A 403 or
  `InvalidKeyMapError` usually means either the key is mistyped or the
  **Maps JavaScript API** isn't enabled on the Cloud project.
- **Marker missing for a tree** &mdash; some addresses ("Removed from list in
  2024", or a non-addressable reference like "NW Corner of SW Park &amp; Main
  (right-of-way)") will not resolve. Re-run `heritage/heritage.ps1`: when it
  gets to that tree it prints the details and research links and lets you
  enter a better address. The edit is persisted in `trees.json` and picked up
  by the app on its next **Check for updates**. As a last resort the app also
  has **Settings &rarr; Retry failed geocoding** which uses the browser's
  fallback geocoder.
- **Nominatim 403 / 429 / "blocked"** &mdash; OpenStreetMap rate-limits the
  free tier aggressively. Pass `-UserAgent "PDXHeritageTrees (your@email)"`
  so they can reach you, and consider raising `-DelayMs` or running the
  script in smaller chunks.
- **PWA shows old UI after an update** &mdash; tap **Reload** on the update
  banner, or open **Settings &rarr; Check for app update**. The shell uses a
  network-first strategy specifically to avoid this.

## Versioning

The app version lives in three places:

- `service-worker.js`      &rarr; `const VERSION = '1.0.0'`
- `manifest.json`          &rarr; `"version": "1.0.0"`
- `index.html`             &rarr; `?v=1.0.0` query on each asset

These are bumped together only when the app changes in a way that warrants a
new service worker (asset list, caching behavior, etc.). They are not bumped
automatically.

## License

Tree data is published by the
[City of Portland](https://www.portland.gov/trees/heritage) and used here for
personal, educational use. The app code in this folder is provided as-is.
