# Forecast Web App

A Progressive Web App (PWA) providing detailed weather information using the National Weather Service API. No API key required.

## Features

- **No API Key Required**: Uses free National Weather Service API
- **Optional AQI (AirNow)**: Add your own AirNow API key in Settings to display an AQI line in Current Conditions using official AQI category colors; categories 5/6 are emphasized with white text on colored badges and white borders for contrast. Uses AirNow [Current Observations by Zip Code or Lat/Long](https://docs.airnowapi.org/webservices) (`/aq/observation/current/ziplatLong`); rate limit is 500 requests per hour per key.
- **Multiple Display Modes**: Full, Daily, Hourly, Rain, Wind, and History
- **PWA Support**: Installable as a web app with offline support and update detection
- **Saved Locations**: Save favorite locations and switch between them; locations bar open/closed state is remembered. Number hotkeys load the first 20 favorites in drawer order: `1`–`0` for slots 1–10, `Shift+1`–`Shift+0` for slots 11–20 (ignored while typing, renaming a favorite, or when Settings is open).
- **Keyboard shortcuts** (global; same ignore rules as location hotkeys): mode keys **F** Full, **D** Daily, **R** Rain, **W** Wind, **H** Hourly (**H** again while Hourly is active switches to History), **L** toggle Locations bar; section navigation **.** next / **,** previous (see [Section navigation](#section-navigation) below).
- **Settings (gear button)**: Accent colors (primary/secondary), Reset Colors, Standard/Metric units, AM/PM or 24-hour time, Compact/Normal density, Feels-Like vs **WBGT** (optional estimated wet-bulb globe temperature when warm—see below), Auto-Update Data, optional AQI (Enable AQI + AirNow API key with inline validation), Extras (Enable Radar—off by default: NWS ridge loop GIF in Full mode above hourly, cached by the service worker for offline; Enable Solar Irradiance; Enable Magic Hours; Enable per Location Colors); Reset Forecast clears all data and settings to defaults. You can also double-click the header icon to open Settings.
- **Control Bar**: Favorite (save location), current location (pin), Locations (open/close saved locations), Refresh, Share (copy or share URL), Settings (gear)
- **Share**: Copy shareable link or use Web Share API when available; URL can include location and mode
- **Units**: Standard (°F, mph, inHg, ft, in) or Metric (°C, m/s, hPa, m, mm) for all displayed values—temperature, wind, pressure, elevation, station distance, tide heights, precipitation, and cloud ceiling (ft or m) in History
- **Time Format**: 12-hour (AM/PM) or 24-hour for all time displays
- **Responsive Design**: Works on all screen sizes and aspect ratios
- **Location Input**: Zip code, "City, State", or use the pin button for automatic current location
- **Color-Coded Metrics**: Temperature, wind, precipitation, humidity, dew point, pressure
- **Weather Calculations**: Wind chill, NWS heat index (default warm “feels like”), optional **estimated WBGT** when you enable the Feels-Like → WBGT toggle (not instrument-grade: Stull wet-bulb plus a simplified globe term using clear-sky solar × a **forecast-text** cloud heuristic); sunrise/sunset, moon phase, and optional Magic Hours timing for photography
- **Sunrise/Sunset/Day Length**: Shown for each day in Daily and History (astronomical calculation)
- **NOAA Tide Stations**: When a station is within 100 miles, shows station name, distance (mi/km), cardinal direction, links to Tide Prediction/Datums/Levels, and last/next tide (height in ft or m, time)
- **Accessibility**: ARIA labels and dialog roles for screen readers; keyboard support (Escape closes Settings; mode, location, and section shortcuts as above)

## Keyboard shortcuts

Shortcuts are ignored while focus is in an input, textarea, or select; while renaming a favorite inline; or while the Settings dialog is open.

| Key | Action |
|-----|--------|
| **F** | Full mode |
| **D** | Daily mode |
| **R** | Rain mode |
| **W** | Wind mode |
| **H** | Hourly mode (press again while Hourly is active to switch to History) |
| **L** | Open or close the Locations drawer |
| **1**–**0** | Load favorites 1–10 (drawer order) |
| **Shift+1**–**Shift+0** | Load favorites 11–20 |
| **.** | Next section (see below) |
| **,** | Previous section (see below) |

Mode buttons show their key in the tooltip (e.g. Full (**F**)). The first 20 location buttons in the drawer show number hints in their tooltips.

### Section navigation

**`.`** and **,** scroll the page so the target lands flush with the top of the viewport. Works in **Full**, **Daily**, **Hourly**, **Rain**, **Wind**, and **History**.

Forward (**`.`**): page top (header + controls) → mode title → each content anchor in order.

Backward (**`,`**): the reverse. From the first content anchor, one more **,** returns to page top.

| Mode | Anchors (in order after page top) |
|------|-----------------------------------|
| **Full** | Current Conditions → Today → Tomorrow (if shown) → Radar (if enabled) → Hourly table → 7-Day Summary → Alerts (if any) → Location Information |
| **Daily** | 7-Day Forecast title → each day block |
| **History** | Observations title → each day block |
| **Rain** / **Wind** | Outlook title → each day row (not the hour legend row) |
| **Hourly** | Hourly title → each 12-hour table page (up to 48 hours). **`.`** on the last page does nothing further. On short pages that fit without scrolling, **.** / **,** change hour pages directly. **,** from the first hour page goes to the Hourly title, then page top |

On the Hourly table, the **Time** column header includes the calendar day(s) for the visible rows in the location’s timezone, e.g. `Time (3rd)` or `Time (3rd-4th)` when hours span midnight. Use **Refresh** if cached data is stale and the dates look wrong.

## Setup

Install Forecast as an app for a home-screen or dock icon, faster launch, and offline support. The live app is at [https://kreft.us/forecast/](https://kreft.us/forecast/) and must be served over **HTTPS** (or opened on **localhost** during development).

### iPhone and iPad (iOS)

1. Open Forecast in **Safari**. Other iOS browsers cannot install PWAs.
2. Tap **Share**, then **Add to Home Screen**.
3. Tap **Add**. Launch Forecast from your home screen like a native app.

If you are on iPhone and not already installed, open **Settings** (gear button)—the app shows a short install reminder there.

### Android

1. Open Forecast in **Chrome** (recommended), **Edge**, or **Samsung Internet**.
2. Tap **Settings** (gear button). If **Install Forecast** appears, tap it and confirm.
3. If that button is not shown, use the browser menu (**⋮**) and choose **Install app** or **Add to Home screen**, or tap the install icon in the address bar when Chrome offers it.

### PC and Mac

1. Open Forecast in a supported browser (**Chrome** or **Edge** on Windows, Mac, or Linux; **Safari** on Mac).
2. **Chrome / Edge**: Open **Settings** (gear button) and use **Install Forecast** when it appears, or click the install control in the address bar, or use the browser menu (**⋮** or **…**) → **Install Forecast** / **Install app**.
3. **Safari (Mac)**: Use **File → Add to Dock** (macOS Sonoma or later), or **Share → Add to Dock**.

Desktop **Firefox** has limited PWA support; use Chrome or Edge for the full install experience.

## Usage

1. Open the app at [https://kreft.us/forecast/](https://kreft.us/forecast/) in a modern browser.
2. Enter a location (zip or "City, State") and click **Load**, or click the **pin** button to use your current location.
3. Use the **mode** buttons (Full, Daily, Hourly, Rain, Wind, History) to switch views, or press **F**, **D**, **H**, **R**, or **W** ( **H** toggles Hourly/History when Hourly is active). Press **L** to open or close the Locations bar. Press **.** and **,** to move between page top, the mode title, and each section or day—or between 12-hour pages in Hourly mode (see [Section navigation](#section-navigation)).
4. **Star** saves the current location to the Locations bar; **Locations** opens/closes the saved locations list.
5. **Refresh** updates weather data; **Share** copies or shares the current page URL (with location and mode).
6. Click the **gear** button in the control bar to open **Settings**: accent colors, Reset Colors, Standard/Metric, AM/PM vs 24H, Compact/Normal density, Feels-Like vs WBGT, Auto-Update Data, optional AQI setup, Extras (Enable Radar for Full-mode NWS loop with offline cache; Enable Solar Irradiance; Enable Magic Hours; Enable per Location Colors). **Reset Forecast** clears all favorites, cache, and settings and reloads. Alternatively, double-click the header icon to open Settings.
7. To enable AQI, toggle **Enable AQI**, paste your **AirNow API Key**, and wait for a green check mark after validation. Register a key at [Request an AirNow API Key](https://docs.airnowapi.org/account/request/). AQI uses AirNow’s current observations API (`/aq/observation/current/ziplatLong`); each key is limited to **500 requests per hour** (see [AirNow Web Services](https://docs.airnowapi.org/webservices)).

## Reset Feature

A full reset clears everything and restores defaults:

1. In Settings, click **Reset Forecast** and confirm, or add `?reset=true` to the URL (e.g. `https://kreft.us/forecast/?reset=true`).
2. The app will:
   - Clear all saved locations (favorites)
   - Clear all cached weather and NOAA data
   - Clear last viewed location and stored location
  - Reset display mode, units, time format, accent colors, radar (to disabled), irradiance, magic hours, Feels-Like/WBGT (to default Feels-Like), auto-update, per-location colors (to disabled), and locations bar state to defaults
   - Reload the page

**Note**: This cannot be undone.

## Display Modes

- **Full**: Current conditions (header shows ⚠️ before "Current Conditions" when the location has active NWS alerts), optional AQI line (when enabled and key is valid) with official AQI colors and high-contrast category 5/6 badges, forecast text, optional **Radar** section when **Enable Radar** is on (same NWS ridge loop as the Location Information “Radar” link; image is not scaled above its native size; the GIF is treated as stale on the same cadence as weather data—refreshed after weather fetches, on a background timer, and when the tab becomes visible, with cache-busted requests so the loop updates while offline fallback still uses the last cached frame), hourly table, 7-day summary, alerts, location info (elevation, NWS/NOAA links, tides when available). Use **.** / **,** to jump between sections (see [Section navigation](#section-navigation)).
- **Daily**: 7-day forecast with sunrise/sunset/day length per day, high/low temps, wind, precipitation chance, detailed text. **.** / **,** step through each day.
- **Hourly**: Hourly table in 12-hour pages (up to 48 hours total) with ↑/↓ links or **.** / **,** to change pages. The **Time** column header shows the calendar day(s) for the visible rows (e.g. `Time (3rd)` or `Time (3rd-4th)`). Other columns: temp, wind, precip %, forecast.
- **Rain**: Rain outlook with likelihood over the next ~96 hours (up to 5 days). **.** / **,** step through each day row.
- **Wind**: Wind direction/speed over the same period. **.** / **,** step through each day row.
- **History**: Historical observations by day with sunrise/sunset/day length. Per day: high/low temp (with wind chill, or estimated WBGT, or heat index when applicable—matching the Feels-Like/WBGT setting), wind (avg/gust or max), pressure (inHg or hPa), precip (in or mm), humidity, conditions, clouds (amount and base height in ft or m per units). Cloud codes: SKC, FEW, SCT, BKN, OVC. Rows omitted when data is not available. **.** / **,** step through each day.

All numeric values (temp, wind, pressure, elevation, distance, tide height, precip depth) follow the **Standard** or **Metric** setting. Times follow **AM/PM** or **24H**.

## Magic Hours (Photography)

When **Enable Magic Hours** is turned on in Settings -> Extras (disabled by default), Current Conditions shows:

- `Next Golden Hour:`
- `Next Blue Hour:`

These lines appear immediately before `Updated:` and are calculated from the location's sun elevation angle:

- **Golden Hour** band: from **+6 degrees** to **-4 degrees**
- **Blue Hour** band: from **-4 degrees** to **-8 degrees**

If the current time is already inside one of those bands, the app shows **Active Until** with the local end time.

Photography science notes:

- **Golden Hour:** with the sun near the horizon, sunlight travels through more atmosphere. Rayleigh scattering removes more blue wavelengths, so warmer red/orange/yellow light dominates.
- **Blue Hour:** with the sun below the horizon, upper atmosphere still receives sunlight. Ozone (Chappuis absorption) reduces red/orange components, leaving a cooler, richer blue ambience that often balances well with city lighting.

## NOAA Resources

When a NOAA tide station is within 100 miles of the location:

- **NOAA Station**: Name, clickable station ID, distance (mi or km), cardinal direction.
- **NOAA Resources**: Links to Tide Prediction, Datums, and (if supported) Water Levels.
- **Tides**: Last and next tide with height (ft or m) and time.

Implementation uses the NOAA CO-OPS Metadata API, computes distance (Haversine), and picks the closest station within 100 miles. Locations with no station within 100 miles are remembered (by coordinates and in cache); subsequent loads skip the station lookup for that location. The console logs when the out-of-range flag is used.

## Per-Location Colors

When **Enable per Location Colors** (under Extras in Settings) is off, one global accent theme applies to all locations. When on:

- **Saved locations**: In Settings, the color pickers show the current location name (e.g. "Anchorage Primary Accent Color"); changes apply only to that location.
- **Non-saved location**: The pickers show "Global Primary Accent Color" / "Global Secondary Accent Color"; changes update the global default only.
- Disabling the option clears all per-location overrides and reverts the UI to the global theme; color choosers in Settings are synced to global values.
- **Reset Forecast** (or Reset Colors in global mode) resets theme behavior; per-location colors return to disabled after a full reset.

## Auto-Update Mechanism

The app checks for new versions (e.g. via `manifest.json` version and service worker). When an update is detected, a “New version available!” message appears; the user can click **Reload** to load the new version. When **Auto-Update Data** is on, stale data can refresh in the background. When **Auto-Update Data** is off, only a manual **Refresh** updates weather data for the currently selected location (other cached favorites are not refreshed). For releases, use `forecast/version.ps1` to keep `service-worker.js`, `manifest.json`, and `index.html` cache-busting versions aligned.

## Technical Details

- **APIs**: National Weather Service (weather.gov), NOAA CO-OPS (tide stations), OpenStreetMap Nominatim (geocoding), Browser Geolocation (current location), and IP geolocation fallback chain for here: ip-api.com -> ipwho.is -> ipapi.co.
- **Location preference**: User-provided locations (zip/city,state/favorites) are always preferred; IP geolocation is only fallback for here when browser geolocation is unavailable or fails.
- **Stack**: Vanilla JavaScript, HTML5, CSS3 (variables, responsive layout), Service Worker (PWA), localStorage (preferences, favorites, cache keys).
- **Browser support**: Modern browsers with Service Worker support (Chrome, Firefox, Edge, Safari).

## Color Coding

- **Temperature**: Blue (cold), Red (hot), default (normal).
- **Wind**: Calm / light / moderate / strong bands with distinct colors.
- **Precipitation chance**: Red (>50%), Yellow (21–50%), default (≤20%).
- **Humidity / Dew point**: Ranges with cyan, yellow, red as appropriate.
- **Pressure (History)**: Color by value for inHg (metric uses same logic on converted hPa).
- **Hour labels (Hourly)**: Yellow for hours mostly in daytime (sunrise–sunset), default otherwise. **Time** column header includes ordinal calendar day(s) for the visible page (location timezone).

## File Structure

```
forecast/
|-- index.html           # Main HTML
|-- manifest.json        # PWA manifest (version, icons, start_url)
|-- service-worker.js    # Cache and update (VERSION)
|-- CACHE_VERIFICATION.md
|-- css/
|   `-- style.css
|-- js/
|   |-- app.js           # App logic, state, UI, Settings
|   |-- api.js           # NWS, geocoding, NOAA, IP
|   |-- weather.js       # Data parsing and aggregation
|   |-- display.js       # Per-mode rendering and unit formatting
|   `-- utils.js         # Conversions, time, sun/moon
|-- version.ps1          # Release helper: updates versions in service-worker.js, manifest.json, and index.html
`-- icons/               # PWA icons (e.g. 192px, 512px)
```

## License

Same as the parent project.
