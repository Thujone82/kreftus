# Forecast Web App

A Progressive Web App (PWA) providing detailed weather information using the National Weather Service API. No API key required.

## Features

- **No API Key Required**: Uses free National Weather Service API
- **Multiple Display Modes**: Full, Daily, Hourly, Rain, Wind, and History
- **PWA Support**: Installable as a web app with offline support and update detection
- **Saved Locations**: Save favorite locations and switch between them; locations bar open/closed state is remembered
- **Settings (gear or double-click header icon)**: Accent colors (primary/secondary), Reset Colors, Standard/Metric units, AM/PM or 24-hour time, Auto-Update Data, Enable Solar Irradiance; Reset Forecast clears all data and settings to defaults
- **Control Bar**: Favorite (save location), current location (pin), Locations (open/close saved locations), Refresh, Share (copy or share URL), Settings (gear)
- **Share**: Copy shareable link or use Web Share API when available; URL can include location and mode
- **Units**: Standard (°F, mph, inHg, ft, in) or Metric (°C, m/s, hPa, m, mm) for all displayed values—temperature, wind, pressure, elevation, station distance, tide heights, precipitation
- **Time Format**: 12-hour (AM/PM) or 24-hour for all time displays
- **Responsive Design**: Works on all screen sizes and aspect ratios
- **Location Input**: Zip code, "City, State", or use the pin button for automatic current location
- **Color-Coded Metrics**: Temperature, wind, precipitation, humidity, dew point, pressure
- **Weather Calculations**: Wind chill, heat index, sunrise/sunset, moon phase
- **Sunrise/Sunset/Day Length**: Shown for each day in Daily and History (astronomical calculation)
- **NOAA Tide Stations**: When a station is within 100 miles, shows station name, distance (mi/km), cardinal direction, links to Tide Prediction/Datums/Levels, and last/next tide (height in ft or m, time)
- **Accessibility**: ARIA labels and dialog roles for screen readers; keyboard support (Escape closes Settings)

## Setup

1. **PWA Icons**: Place icon files in the `icons/` directory (e.g. 192×192 and 512×512 PNG). The app references `light-icon-192.png` and `light-icon-512.png` in the HTML; the manifest may reference `dark-icon-*` or the same—ensure the paths in `manifest.json` and `index.html` match your files.

2. **Deploy**: Static web app; serve from any web server. HTTPS is required in production for PWA features (service worker, install prompt).

3. **Cache busting (optional)**: Run `node forecast/scripts/inject-version.js` from the repo root before deploy. It reads `VERSION` from `service-worker.js` and replaces `{{VERSION}}` in `index.html` so one version drives both the service worker cache and asset query params. See `CACHE_VERIFICATION.md` for details.

## Usage

1. Open the app in a modern browser.
2. Enter a location (zip or "City, State") and click **Load**, or click the **pin** button to use your current location.
3. Use the **mode** buttons (Full, Daily, Hourly, Rain, Wind, History) to switch views.
4. **Star** saves the current location to the Locations bar; **Locations** opens/closes the saved locations list.
5. **Refresh** updates weather data; **Share** copies or shares the current page URL (with location and mode).
6. **Gear** (or double-click the header icon) opens **Settings**: accent colors, Reset Colors, Standard/Metric, AM/PM vs 24H, Auto-Update Data, Enable Solar Irradiance. **Reset Forecast** clears all favorites, cache, and settings and reloads.

## Reset Feature

A full reset clears everything and restores defaults:

1. In Settings, click **Reset Forecast** and confirm, or add `?reset=true` to the URL (e.g. `https://yoursite.com/forecast/?reset=true`).
2. The app will:
   - Clear all saved locations (favorites)
   - Clear all cached weather and NOAA data
   - Clear last viewed location and stored location
   - Reset display mode, units, time format, accent colors, irradiance, auto-update, and locations bar state to defaults
   - Reload the page

**Note**: This cannot be undone.

## Display Modes

- **Full**: Current conditions, forecast text, hourly table, 7-day summary, alerts, location info (elevation, NWS/NOAA links, tides when available).
- **Daily**: 7-day forecast with sunrise/sunset/day length per day, high/low temps, wind, precipitation chance, detailed text.
- **Hourly**: Scrollable hourly table (time, temp, wind, precip %, forecast); nav to earlier/later hours.
- **Rain**: Rain outlook with likelihood over the next ~96 hours (up to 5 days).
- **Wind**: Wind direction/speed over the same period.
- **History**: Historical observations by day with sunrise/sunset/day length. Per day: high/low temp (with wind chill/heat index when applicable), wind (avg/gust or max), pressure (inHg or hPa), precip (in or mm), humidity, conditions, clouds (when provided). Cloud codes: SKC, FEW, SCT, BKN, OVC. Rows omitted when data is not available.

All numeric values (temp, wind, pressure, elevation, distance, tide height, precip depth) follow the **Standard** or **Metric** setting. Times follow **AM/PM** or **24H**.

## NOAA Resources

When a NOAA tide station is within 100 miles of the location:

- **NOAA Station**: Name, clickable station ID, distance (mi or km), cardinal direction.
- **NOAA Resources**: Links to Tide Prediction, Datums, and (if supported) Water Levels.
- **Tides**: Last and next tide with height (ft or m) and time.

Implementation uses the NOAA CO-OPS Metadata API, computes distance (Haversine), and picks the closest station within 100 miles.

## Auto-Update Mechanism

The app checks for new versions (e.g. via `manifest.json` version and service worker). When an update is detected, a “New version available!” message appears; the user can click **Reload** to load the new version. Bump `VERSION` in `service-worker.js` and `version` in `manifest.json` for releases; run `scripts/inject-version.js` before deploy to keep asset cache busting in sync.

## Technical Details

- **APIs**: National Weather Service (weather.gov), NOAA CO-OPS (tide stations), OpenStreetMap Nominatim (geocoding), ip-api.com (IP geolocation fallback), Browser Geolocation (current location).
- **Stack**: Vanilla JavaScript, HTML5, CSS3 (variables, responsive layout), Service Worker (PWA), localStorage (preferences, favorites, cache keys).
- **Browser support**: Modern browsers with Service Worker support (Chrome, Firefox, Edge, Safari).

## Color Coding

- **Temperature**: Blue (cold), Red (hot), default (normal).
- **Wind**: Calm / light / moderate / strong bands with distinct colors.
- **Precipitation chance**: Red (>50%), Yellow (21–50%), default (≤20%).
- **Humidity / Dew point**: Ranges with cyan, yellow, red as appropriate.
- **Pressure (History)**: Color by value for inHg (metric uses same logic on converted hPa).
- **Hour labels (Hourly)**: Yellow for hours mostly in daytime (sunrise–sunset), default otherwise.

## File Structure

```
forecast/
├── index.html           # Main HTML
├── manifest.json        # PWA manifest (version, icons, start_url)
├── service-worker.js    # Cache and update (VERSION)
├── CACHE_VERIFICATION.md
├── css/
│   └── style.css
├── js/
│   ├── app.js           # App logic, state, UI, Settings
│   ├── api.js           # NWS, geocoding, NOAA, IP
│   ├── weather.js       # Data parsing and aggregation
│   ├── display.js       # Per-mode rendering and unit formatting
│   └── utils.js         # Conversions, time, sun/moon
├── scripts/
│   └── inject-version.js  # Deploy: inject VERSION into index.html
└── icons/               # PWA icons (e.g. 192px, 512px)
```

## License

Same as the parent project.
