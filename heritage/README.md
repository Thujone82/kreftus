# PDX Heritage Trees

A small, installable PWA field guide to Portland's registered **Heritage Trees**.
It keeps a local database of every tree on the City registry, plots them on a
Google Map, and lets you mark which ones you've found, timestamp the find, and
take notes &mdash; all stored privately in your browser.

The initial tree list is scraped from
[portland.gov/trees/heritage/heritage-trees-year](https://www.portland.gov/trees/heritage/heritage-trees-year)
by the included PowerShell script and bundled as a JSON snapshot
(`data/trees.json`). The app reads that JSON on first run, geocodes each tree's
address with the Google Geocoding API, and then runs entirely offline from your
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
- A free **Google Maps API key** with these APIs enabled:
  - **Maps JavaScript API** (for the interactive map and markers)
  - **Geocoding API** (for converting tree addresses into coordinates once)

Google's monthly free credit comfortably covers the one-time geocoding of
~400 trees plus normal day-to-day map use.

## Setup

### 1. Get a free Google Maps API key

1. Go to the
   [Google Cloud Console &rarr; Credentials](https://console.cloud.google.com/google/maps-apis/credentials)
   and sign in.
2. Create a new project (or select an existing one) and click
   **Create Credentials &rarr; API Key**.
3. Enable the two APIs above on the project
   (**APIs &amp; Services &rarr; Library**).
4. *(Recommended)* Restrict the key to those two APIs and to your site's URL
   under **Application restrictions** and **API restrictions**.

### 2. First launch

1. Open `heritage/index.html` in a browser (or deploy the `heritage/`
   folder to your site at `/heritage/`).
2. The welcome screen walks you through pasting your API key. The key is
   stored in `localStorage` under `pdxHeritageGoogleApiKey` and is only ever
   sent to Google.
3. After you save the key, the app:
   - Loads `data/trees.json` (~400 trees) into IndexedDB.
   - Streams geocoding requests against your Google API key to resolve each
     tree's coordinates. A top progress bar shows `Geocoding trees... N / M`.
   - Initializes the map once the first markers are available.

Geocoding runs at about 5 requests/second and takes ~90 seconds for a fresh
install. The app is usable during geocoding &mdash; markers appear as they
resolve.

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

The bundled `data/trees.json` is produced by the included PowerShell scraper,
which parses the HTML table on
[portland.gov/trees/heritage/heritage-trees-year](https://www.portland.gov/trees/heritage/heritage-trees-year).
The script lives at `../ps/heritage/heritage.ps1` relative to this folder, so
from the repository root:

```powershell
pwsh -File ps/heritage/heritage.ps1
```

It writes `heritage/data/trees.json` with a header object:

```json
{
  "sourceUrl": "https://www.portland.gov/trees/heritage/heritage-trees-year",
  "scrapedAt": "2026-04-16T17:13:10Z",
  "count": 397,
  "trees": [
    { "id": "001", "year": 1993,
      "name": "Ulmus americana - American elm",
      "location": "Removed from list in 2024",
      "removed": 2024 },
    ...
  ]
}
```

Tree `id` is the Tree # zero-padded to 3 digits (`"001"`). Running the scraper
against an already-populated snapshot prints a summary of new and newly-removed
trees since the previous run.

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
  README.md                  # this file
  css/
    styles.css               # PNW woodsy theme
  js/
    db.js                    # IndexedDB wrapper (trees, meta stores)
    wiki.js                  # species -> Wikipedia URL
    sync.js                  # fetch trees.json, diff-merge preserving user data
    geocode.js               # throttled Geocoding API queue
    map.js                   # map, markers, info window, camera logic
    nearby.js                # 10-nearest list + walking nav
    ui.js                    # progress bar, toast, modal helpers
    sw-register.js           # service worker + update banner
    app.js                   # boot & glue
  data/
    trees.json               # bundled snapshot produced by the PS1 scraper
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
- **Geocoding is slow / stalling** &mdash; the queue runs at a polite ~5 req/s.
  If Google returns `OVER_QUERY_LIMIT` the app backs off automatically. Open
  **Settings &rarr; Retry failed geocoding** to re-attempt just the ones that
  failed.
- **Marker missing for a tree** &mdash; some addresses ("Removed from list in
  2024", or a non-addressable reference like "NW Corner of SW Park &amp; Main
  (right-of-way)") will not resolve. Use **Retry failed geocoding** from
  Settings and, if it still fails, add a note on the tree manually.
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
