# Cache Verification Summary

## Version Source

Version is defined in two places and must be kept in sync:

- **`service-worker.js`**: `const VERSION = '1.2.4'` — used for cache names and SW update check.
- **`manifest.json`**: `"version": "1.2.4"` — used by the app’s periodic update check and PWA metadata.

The config modal shows **Forecast v{VERSION}** at the bottom; version is read by fetching `service-worker.js` and parsing the `VERSION` constant (no extra network call for “updated ago”).

---

## Files Cached in Service Worker

All static assets are cached in the `STATIC_ASSETS` array in `service-worker.js`:

1. `/forecast/` — Root path
2. `/forecast/index.html` — Main HTML file
3. `/forecast/css/style.css` — Stylesheet
4. `/forecast/js/app.js` — Main application logic
5. `/forecast/js/api.js` — API interactions
6. `/forecast/js/weather.js` — Weather data processing
7. `/forecast/js/display.js` — Display rendering
8. `/forecast/js/utils.js` — Utility functions
9. `/forecast/manifest.json` — PWA manifest
10. `/forecast/icons/light-icon-192.png` — 192px icon
11. `/forecast/icons/light-icon-512.png` — 512px icon

---

## Cache-Busting Parameters

Script and stylesheet URLs in `index.html` use the same **VERSION** as the service worker. The file contains placeholders `?v={{VERSION}}` that are replaced at deploy time.

**Deploy-time script:** Run `node forecast/scripts/inject-version.js` (from repo root) before deploying. The script reads `VERSION` from `forecast/service-worker.js` and replaces all `{{VERSION}}` in `forecast/index.html` with that value. Then only `VERSION` in `service-worker.js` and `version` in `manifest.json` need to be updated on release; asset cache busting stays in sync automatically.

---

## Cache Names

Defined in `service-worker.js` from the `VERSION` constant:

- **`forecast-static-v{VERSION}`** — Static assets (HTML, JS, CSS, manifest, icons).
- **`forecast-data-v{VERSION}`** — API responses (weather, geocoding, IP).

`CACHE_NAME` (`forecast-v{VERSION}`) is defined but not used for storage; only the static and data caches are used.

---

## Fetch Strategies

### Network-first (updates preferred)

- **JS, HTML, CSS, JSON** (including `/forecast/` and manifest): `fetch(..., { cache: 'no-cache' })`, then on success the response is cloned and stored in `STATIC_CACHE`. On network failure, fallback to cache.
- **API requests** (api.weather.gov, nominatim.openstreetmap.org, ip-api.com, ipwho.is, ipapi.co): fetch from network first; on success (status 200) clone and store in `DATA_CACHE`. On network failure, use cache or return offline JSON.

### Cache-first

- **Other static assets** (e.g. icons): `caches.match(request)` then `fetch(request)` if missing.

---

## Update Mechanism

### Service worker install/activate

1. **Install**: Precache all `STATIC_ASSETS` into `STATIC_CACHE`, then `skipWaiting()`.
2. **Activate**:
   - Delete any cache whose name starts with `forecast-` and is not the current `STATIC_CACHE` or `DATA_CACHE`.
   - Post `CLEAR_CACHE` (reason: “Service worker updated”) to all clients.
   - `clients.claim()`.

### App response to CLEAR_CACHE

When the app receives `CLEAR_CACHE` from the service worker:

- It calls `clearAllCachedData()` (clears localStorage weather/cache data).
- If a location is set, it triggers a refresh of weather data after a short delay so new features (e.g. NOAA station data) are loaded.

### Update detection

1. **Periodic check (app)**  
   Every 5 minutes the app runs `checkForUpdate()`: fetches `/forecast/manifest.json?t=...`, compares `manifest.version` to `localStorage.getItem('forecastVersion')` (default `'1.0.0'`). If different, it shows the update notification, stores the new version, and sends `CHECK_UPDATE` to the service worker.

2. **Service worker updatefound**  
   When a new service worker is installed and reaches `installed` while a controller already exists, the app shows the update notification.

3. **Service worker CHECK_UPDATE**  
   When the app sends `CHECK_UPDATE`, the service worker runs its own `checkForUpdate()`: fetches manifest, compares `manifest.version` to its `VERSION` constant. If different, it posts `UPDATE_AVAILABLE` (with `version`) to all clients; the app shows the update notification.

### User notification and reload

- **Notification**: “New version available!” with a **Reload** button.
- **Reload button**:
  1. `caches.keys()` then `caches.delete()` for every cache name.
  2. `navigator.serviceWorker.getRegistration()` then `registration.unregister()`.
  3. `window.location.reload(true)` to load the page with fresh files.

---

## Verification Checklist

- [x] All JS files listed in `STATIC_ASSETS`
- [x] All JS files have cache-busting query params in `index.html`
- [x] CSS file in `STATIC_ASSETS` and has cache-busting param in `index.html`
- [x] `index.html` and `manifest.json` in `STATIC_ASSETS`
- [x] Icons in `STATIC_ASSETS` match paths used in app (`light-icon-192.png`, `light-icon-512.png`)
- [x] Network-first for JS/HTML/CSS/JSON
- [x] Network-first for API domains with cache fallback
- [x] Cache-first for other static assets
- [x] Old caches deleted on activate when version changes
- [x] `CLEAR_CACHE` on activate; app clears localStorage cache and can refresh data
- [x] Reload button clears all caches, unregisters SW, and reloads

---

## Testing the Update Flow

1. Bump **both** `VERSION` in `service-worker.js` and `version` in `manifest.json` (e.g. `1.2.4` → `1.2.5`).
2. Optionally bump cache-busting params in `index.html` for changed files (e.g. `?v=5`).
3. Deploy; the app will detect the change within 5 minutes (or when the user triggers a check), or immediately when a new SW installs (`updatefound`).
4. “New version available!” should appear; clicking **Reload** should clear caches, unregister the SW, and reload with the new version.
