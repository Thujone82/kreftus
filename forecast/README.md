# Forecast Web App

A Progressive Web App (PWA) version of the PowerShell GetForecast script, providing detailed weather information using the National Weather Service API.

## Features

- **No API Key Required**: Uses free National Weather Service API
- **Multiple Display Modes**: Full, Terse, Hourly, Daily, Rain, and Wind modes
- **PWA Support**: Installable as a web app with offline support
- **Auto-Update**: Automatically detects app updates via manifest version checking
- **Responsive Design**: Works on all screen sizes and aspect ratios
- **Location Detection**: Supports zip codes, "City, State", or automatic "here" detection
- **Color-Coded Metrics**: Visual indicators for temperature, wind, precipitation, humidity, and dew point
- **Weather Calculations**: Wind chill, heat index, sunrise/sunset, moon phase calculations
- **Sunrise/Sunset/Day Length**: Displays sunrise, sunset, and day length for each day in Daily and History modes (calculated astronomically)
- **NOAA Resources**: Conditionally displays NOAA tide station information and links when a station is found within 100 miles

## Setup

1. **Add PWA Icons**: Create the following icon files in the `icons/` directory:
   - `icon-192.png` (192x192 pixels)
   - `icon-512.png` (512x512 pixels)

2. **Deploy**: The app is a static web app and can be served from any web server.

3. **HTTPS Required**: PWA features (service worker, installation) require HTTPS in production.

## Usage

1. Open `index.html` in a modern web browser
2. Enter a location (zip code or "City, State") or click "Here" for automatic detection
3. Select a display mode using the mode buttons
4. Use the refresh button to manually update weather data
5. Toggle auto-update to enable/disable automatic data refreshing (every 10 minutes)

## Reset Feature

If you experience issues with corrupted favorites that cannot be removed, you can perform a complete reset:

1. Add `?reset=true` to the URL (e.g., `https://kreft.us/forecast/?reset=true`)
2. The app will automatically:
   - Clear all favorites
   - Clear all cached weather data
   - Clear all NOAA station cache
   - Clear last viewed location
   - Reset to default mode
   - Reload the page with a clean state

**Note**: This action cannot be undone. All saved favorites and cached data will be permanently deleted.

## Display Modes

- **Full**: Complete weather information (current conditions, forecasts, hourly, 7-day, alerts, location info)
- **Terse**: Current conditions + today's forecast + alerts summary
- **Hourly**: 12-hour hourly forecast (scrollable to 48 hours)
- **Daily**: Enhanced 7-day forecast with sunrise/sunset/day length for each day and detailed forecasts
- **Rain**: 96-hour rain likelihood sparklines (5 days max)
- **Wind**: 96-hour wind direction glyphs (5 days max)
- **History**: Historical observations with sunrise/sunset/day length for each day. When the station provides it, barometric pressure (inHg) is shown after wind with color coding; when pressure data is not available for a day, the pressure row is omitted (nothing is shown).

## NOAA Resources

When a NOAA tide station is found within 100 miles of the location, the app displays:

- **NOAA Station**: Station name, clickable station ID link, distance in miles, and cardinal direction from weather location
- **NOAA Resources**: Links to Tide Prediction, Datums, and Water Levels (if supported)

The station ID is a clickable link that opens the NOAA station homepage for detailed information.

**Technical Implementation:**
- Uses the official NOAA CO-OPS Metadata API (`https://api.tidesandcurrents.noaa.gov/mdapi/prod/webapi/stations.json`)
- Fetches all available tide stations and calculates distance using the Haversine formula
- Returns the closest station within 100 miles
- No hardcoded station list - searches all available NOAA stations dynamically

## Auto-Update Mechanism

The app automatically checks for updates by comparing the version in `manifest.json` with the cached version. When a new version is detected:

1. The service worker detects the version change
2. A notification appears to the user
3. User can click "Reload" to get the new version

To trigger an update, simply change the `version` field in `manifest.json` (e.g., from "1.0.0" to "1.0.1").

## Technical Details

- **APIs Used**:
  - National Weather Service API (weather.gov)
  - NOAA CO-OPS Metadata API (tide station discovery)
  - OpenStreetMap Nominatim API (geocoding)
  - ip-api.com (IP-based geolocation fallback)
  - Browser Geolocation API (automatic location detection)

- **Technologies**:
  - Vanilla JavaScript (no frameworks)
  - HTML5
  - CSS3 with CSS Variables
  - Service Workers (PWA)
  - IndexedDB (localStorage for preferences)

- **Browser Support**:
  - Modern browsers with Service Worker support
  - Chrome, Firefox, Edge, Safari (latest versions)

## Color Coding

- **Temperature**: Blue (<33°F), Red (>89°F), White (normal)
- **Wind Speed**: White (≤5mph), Yellow (6-9mph), Red (10-14mph), Magenta (≥15mph)
- **Precipitation**: Red (>50%), Yellow (21-50%), White (≤20%)
- **Humidity**: Cyan (<30%), White (30-60%), Yellow (61-70%), Red (>70%)
- **Dew Point**: Cyan (<40°F), White (40-54°F), Yellow (55-64°F), Red (≥65°F)
- **Hour Labels**: Yellow when the majority of that hour is during daytime (determined by checking if the hour midpoint falls between sunrise and sunset), White otherwise. This helps quickly distinguish daytime vs nighttime hours at a glance in the hourly forecast.

## File Structure

```
forecast/
├── index.html          # Main HTML structure
├── manifest.json       # PWA manifest with version
├── service-worker.js   # Service worker for offline support
├── css/
│   └── style.css       # All styling
├── js/
│   ├── app.js          # Main application logic
│   ├── api.js          # API integration
│   ├── weather.js      # Weather data processing
│   ├── display.js      # Display mode rendering
│   └── utils.js        # Utility functions
└── icons/
    ├── icon-192.png    # PWA icon (192x192)
    └── icon-512.png    # PWA icon (512x512)
```

## License

Same as the parent project.

