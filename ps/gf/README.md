# gf — Get Forecast (NWS Edition)

**Author:** Kreft&Cursor

## Description
`gf.ps1` is a PowerShell script that retrieves and displays detailed weather information for a specified location using the National Weather Service API. It can accept a US zip code or a "City, State" string as input.

The script first uses OpenStreetMap Nominatim to geocode the location, then fetches current weather, forecasts, alerts, and optional AirNow AQI from the National Weather Service and related APIs.

## Features
- **Flexible Location Input:** Accepts 5-digit zip codes, city/state names (e.g., "Portland, OR"), or "here" for automatic location detection.
- **Automatic Location Detection:** Use "here" to automatically detect your location based on your IP address with provider fallback (`ip-api.com` -> `ipwho.is` -> `ipapi.co`).
- **Interactive Prompt:** If no location is provided, the script displays a welcome screen and prompts for input.
- **Comprehensive Weather Data:** Displays a wide range of information, including:
  - Current temperature and conditions.
  - Wind chill and heat index calculations (NWS formulas), or estimated outdoor WBGT with `-wbgt` (aligned with the forecast web app).
  - Temperature trend indicators (↗️ rising, ↘️ falling, → steady) when observation data supports them.
  - Detailed daily and tomorrow forecasts.
  - Wind speed and direction.
  - Sunrise and sunset times (calculated astronomically).
  - Solar irradiance (clear-sky GHI in W/m² at current time plus peak at solar noon with time), displayed in white when available (full and terse views).
  - Optional Magic Hours lines (`-m` / `-Magic`) in Current Conditions immediately before `Updated:`:
    - `Golden Hour:` / `Next Golden Hour:`
    - `Blue Hour:` / `Next Blue Hour:`
    - Active windows display `Active Until HH:mm`; inactive windows display `HH:mm-HH:mm`.
  - Moon phase information with emoji and next full moon date.
  - **All times displayed in location's timezone:** Hourly forecasts, sunrise, sunset, and update times are shown in the destination location's local timezone, not your system's timezone.
  - Weather alerts and warnings.
  - AQI line (AirNow) after Wind when configured: requires your own **AirNow API key** in the persisted Windows **User** environment variable **`AirNowAPI`** (not stored in the script). Use **`.\gf.ps1 -aqi`** to set or validate the key (see [Parameters](#parameters)). When the variable is unset, AQI is omitted.
  - Rain likelihood forecast with visual sparklines.
  - Wind outlook forecast with direction glyphs.
- **Smart Color-Coding:** Important metrics are color-coded for quick assessment:
  - **Temperature:** Turns blue if below 33°F and red if above 89°F.
  - **Wind Chill:** Displayed in blue when temperature <= 50°F and difference > 1°F.
  - **Heat Index / WBGT:** Heat index in alert color when temp ≥ 80°F and difference > 1°F; with `-wbgt`, estimated outdoor WBGT bracket uses the same temperature color bands as dry-bulb (including warm band from 75°F).
  - **Wind (current conditions):** Wind speed and direction glyph colored by intensity — White (≤5 mph), Yellow (6–9), Red (10–14), Magenta (15+). Gust parenthetical uses alert color when present.
  - **Alert header:** When active alerts apply, the current-conditions title is prefixed with ⚠️ (and 🌡 for heat-related alerts), matching the forecast web app.
  - **Hour Labels:** Hour labels in the hourly forecast (e.g., "08:00", "09:00") are colored yellow when the majority of that hour is during daytime, otherwise displayed in white. This helps quickly identify daytime vs nighttime hours at a glance.
  - **Rain Likelihood:** Color-coded sparklines — White (0–10%), Cyan (11–33%), Green (34–44%), Yellow (45–80%), Red (81%+).
  - **Wind Outlook:** Color-coded directional glyphs — White (≤5 mph), Yellow (6–9), Red (10–14), Magenta (15+); peak wind hours highlighted with inverted colors.
  - **Humidity:** Color-coded based on comfort levels:
    - Cyan: Very dry (<30%)
    - White: Comfortable (30-60%)
    - Yellow: Getting humid (61-70%)
    - Red: Very humid (>70%)
  - **Dew Point:** Color-coded based on comfort levels:
    - Cyan: Very dry, crisp air (<40°F)
    - White: Comfortable, pleasant (40-54°F)
    - Yellow: Getting sticky/muggy (55-64°F)
    - Red: Oppressive, very uncomfortable (≥65°F)
  - **Pressure (Observations only):** Barometric pressure in inHg: Cyan (<29.50), White (29.50–30.20), Yellow (>30.20), Red (extreme: <29.0 or >30.5)
  - **Clouds (Observations only):** When data is available, "Clouds:" is shown on the same line as Conditions (white label, gray data). Codes: SKC (clear), FEW (few), SCT (scattered), BKN (broken), OVC (overcast), VV (vertical visibility). Omitted when not available
  - **AQI (Current conditions):**
    - `AQI:` label is always White.
    - `CategoryName` color uses highest category number from O3/PM2.5:
      - 1 Good = Green
      - 2 Moderate = Cyan
      - 3 Unhealthy for Sensitive Groups = Yellow
      - 4 Unhealthy = Yellow
      - 5 Very Unhealthy = Red
      - 6 Hazardous = Magenta
      - 7 Unavailable = suppress AQI line
    - `O3[...]` and `PM2.5[...]` are each colored from their own category numbers.
    - In terse mode, AQI is shown only when highest category is 2-6; otherwise suppressed.
- **Robust Error Handling:** Implements exponential backoff retry logic for service unavailability, automatically retrying up to 10 times with increasing delays (1s to 512s) before gracefully exiting with a clear error message.
- **Weather Alerts:** Automatically displays any active weather alerts (e.g., warnings, watches) for the location. Test/monitoring-only alerts are filtered out (v2.2).
- **NWS Resources:** Provides clickable links to official NWS resources:
  - **Forecast:** Direct link to weather.gov forecast map for the location
  - **Graph:** Direct link to NWS graphical forecast with detailed charts
  - **Radar:** Direct link to NWS radar imagery for the local radar station
- **NOAA Resources:** Conditionally displays NOAA tide station information when a station is found within 100 miles:
  - **NOAA Station:** Shows station name, clickable station ID link, distance in miles, and cardinal direction from weather location
  - **Tide Prediction:** Direct link to NOAA tide predictions for the station
  - **Datums:** Direct link to station datums information
  - **Water Levels:** Direct link to water level data (only if station supports it)
  - **API-Based Discovery:** Uses official NOAA CO-OPS Metadata API to search all available stations dynamically
  - **Tide Predictions Display:** Shows "Tides: Last[↑/↓]: {Height}ft@{Time} Next[↑/↓]: {Height}ft@{Time}" with last and next high/low tide predictions
  - **Adjacent Day Fetching:** Automatically fetches tomorrow's predictions if no "next" tide found for today, and yesterday's predictions if no "last" tide found for today, ensuring complete tide information is always available
- **Enhanced Daily Mode:** Comprehensive 7-day forecast with detailed information:
  - Sunrise, sunset, and day length for each day (calculated astronomically)
  - Wind speed and direction with color-coded intensity (same bands as wind outlook)
  - Windchill, heat index, or estimated WBGT (with `-wbgt`) when applicable
  - Precipitation probability with "Precip" label for clarity
  - Word-wrapped detailed forecasts for both day and night periods
  - Smart day/night period detection for single-period days
  - Consistent gray color for all detailed forecast text
- **Interactive control bar:** On-screen hotkey hints (toggle with **B** or start hidden with `-b`). Use `-x` for one-shot output without the control bar loop.
- **CLI tolerance:** Unrecognized switches (e.g. accidental `-c`) produce a yellow warning and the script continues (v2.2).
- **Moon Phase Information:** Displays current moon phase with emoji and next full moon date:
  - Shows 8 moon phases: New Moon, Waxing Crescent, First Quarter, Waxing Gibbous, Full Moon, Waning Gibbous, Last Quarter, Waning Crescent
  - Uses astronomical calculation based on known new moon reference (January 6, 2000)
  - Displays appropriate emoji for each phase: 🌑🌒🌓🌔🌕🌖🌗🌘
  - Shows "Next Full Moon: MM/DD/YYYY" only when not currently a full moon
  - Appears in gray color after sunset information
- **Interactive mode:** When run interactively, keyboard shortcuts switch display modes:
  - **H** — Hourly forecast (12-hour page; **↑**/**↓** scroll through up to 48 hours)
  - **D** — 7-day forecast only
  - **T** — Terse mode (current conditions + today's forecast)
  - **R** — Rain forecast mode (sparklines)
  - **W** — Wind forecast mode (direction glyphs)
  - **O** — Observations mode (historical weather data)
  - **G** — Refresh weather data (auto-refreshes every 10 minutes)
  - **U** — Toggle automatic updates on/off
  - **B** — Toggle control bar on/off
  - **F** — Return to full display
  - **Enter** or **Esc** — Exit the script (**Ctrl+C** also exits)

## Requirements
- PowerShell
- An active internet connection.
- **NWS / geocoding:** No API key required for weather (National Weather Service and OpenStreetMap Nominatim).
- **AQI (optional):** If you want the AirNow AQI line, set the **User** environment variable **`AirNowAPI`** to your key from https://docs.airnowapi.org/account/request/ or run **`.\gf.ps1 -aqi`**.

## How to Run
1. Open a PowerShell terminal.
2. Navigate to the directory where `gf.ps1` is located.
3. Run the script using one of the formats below.

> **Note:** To run PowerShell scripts, you may need to adjust your execution policy (e.g. `Set-ExecutionPolicy Bypass` from an administrator PowerShell prompt).

## Configuration
The script uses a user agent string "GetForecast/1.0 (081625PDX)" for API requests. This is hardcoded in the script and does not require a configuration file.

### Essential Configuration Tips
To ensure proper emoji display, you must ensure your terminal and PowerShell are set to use the correct encoding and a font that supports emoji glyphs.

#### 1. Use a Unicode-Compliant Font
The default console font may not contain all the necessary emoji glyphs. You should use a font that is designed for terminal use and includes broad Unicode support.

**Cascadia Code PL / Cascadia Mono PL:** These are Microsoft's recommended fonts. The "PL" (Powerline) versions include extra glyphs for powerline symbols, which also helps with general Unicode support.

**A Nerd Font:** If you use a customized prompt (like Oh My Posh), you'll need to install a Nerd Font (like Cascadia Code NF) which is patched with thousands of extra glyphs, including extensive emoji sets.

To change the font in Windows Terminal:

1. Open Windows Terminal settings (Ctrl+,).
2. Select the PowerShell profile on the left.
3. Go to the Appearance section.
4. In the Font face dropdown, select a font like Cascadia Code PL or a Nerd Font.


#### 2. Change console encoding to UTF-8

Emojis require UTF-8 encoding. The script attempts to set this automatically; you can also add the following to your PowerShell profile:

```powershell
# Set console output encoding to UTF-8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

# Set PowerShell output encoding for cmdlets
$OutputEncoding = [System.Text.Encoding]::UTF8
```

## Parameters

| Parameter | Alias | Description |
|-----------|-------|-------------|
| `Location` | — | US zip or `City, State`. Prompted if omitted. |
| `-Help` | — | Show detailed help and exit. |
| `-Verbose` | — | Debug output for API calls, AQI diagnostics, suppression reasons. |
| `-AqiSetup` | `-aqi` | AQI setup screen only; stores key in User env `AirNowAPI`. [Request a key](https://docs.airnowapi.org/account/request/) |
| `-Terse` | `-t` | Current conditions + today's forecast (+ alerts). |
| `-Hourly` | `-h` | 12-hour hourly forecast (scroll up to 48h in interactive mode). |
| `-Daily` | `-d` | Enhanced 7-day forecast with wind, indices, and wrapped text. |
| `-Rain` | `-r` | Rain likelihood sparklines (96 hours). |
| `-Wind` | `-w` | Wind outlook glyphs (96 hours). |
| `-Observations` | `-o` | Historical observations (7 days). |
| `-NoAutoUpdate` | `-u` | Start with auto-update disabled (10-minute default). |
| `-Magic` | `-m` | Golden/Blue hour lines before `Updated:`. |
| `-NoInteractive` | `-x` | Display once and exit (scripting). |
| `-NoBar` | `-b` | Start with the interactive control bar hidden. |
| `-UseWbgt` | `-wbgt` | Use estimated outdoor WBGT instead of heat index (warm band from 75°F). |
| `-Noaa` | — | Override NOAA tide station ID (ignores 100-mile limit). |

### Parameter details

- `Location` [string] (Positional: 0)
  - The location for which to retrieve weather. Can be a 5-digit US zip code or a "City, State" string.
  - If omitted, the script will prompt you for it.

- `-Help` [switch]
  - Displays a detailed help and usage message in the console.

- `-Verbose` [switch]
  - A built-in PowerShell parameter that displays debugging details for API calls and processing.
  - Includes AQI diagnostics:
    - AirNow request line (`GET:` with API key redacted)
    - Number of AQI rows returned
    - Parsed AQI summary (category, O3, PM2.5)
    - Suppression reasons (empty/null payload, unavailable category 7, terse-mode suppression)

- `-aqi` (alias) or `-AqiSetup` [switch]
  - Opens the **AQI Setup** screen only, then exits (no weather fetch).
  - Stores your key in the **User** environment variable **`AirNowAPI`** using `[Environment]::SetEnvironmentVariable(..., 'User')` so **new PowerShell sessions** keep the key (not session-only).
  - Validates the key with a test request to AirNow at fixed coordinates (Portland, OR area: 45.5202471, -122.674194).
  - Request an AirNow key: https://docs.airnowapi.org/account/request/

- `-Terse` or `-t` [switch]
  - Shows only current conditions and today's forecast (plus alerts if they exist).
  - Combines sunrise and sunset into a single `Sunrise-Sunset: start-end` line and omits the Dew Point line for a tighter layout, while still showing irradiance when available.
  - Provides a streamlined, focused view for quick weather checks.

- `-Hourly` or `-h` [switch]
  - Shows only the 12-hour hourly forecast on first display.
  - In interactive mode, **↑** and **↓** scroll through up to 48 hours in 12-hour pages.
  - Enters interactive mode after display (use `-x` to exit immediately).
  - Perfect for planning activities throughout the day.

- `-Daily` or `-d` [switch]
  - Shows enhanced 7-day forecast with detailed wind information, windchill/heat index/WBGT (with `-wbgt`), and word-wrapped detailed forecasts.
  - Displays 3 lines per day: summary line with wind info, day detailed forecast, and night detailed forecast.
  - Includes color-coded wind speeds, precipitation percentages, and temperature indices.
  - Enters interactive mode after display (use -x to exit immediately).
  - Perfect for comprehensive weekly planning with detailed weather information.

- `-Rain` or `-r` [switch]
  - Shows rain likelihood forecast with sparklines for the next 96 hours (4 days).
  - Displays up to 5 days of hourly rain probability data in a visual sparkline format.
  - Uses color-coded sparklines: White (very low), Cyan (low), Green (light), Yellow (medium), Red (high) rain likelihood.
  - Enters interactive mode after display (use -x to exit immediately).
  - Perfect for quick rain planning and outdoor activity scheduling.

- `-Wind` or `-w` [switch]
  - Shows wind outlook forecast with direction glyphs for the next 96 hours (4 days).
  - Displays up to 5 days of hourly wind direction and speed data in a visual glyph format.
  - Uses color-coded directional glyphs: White (calm), Yellow (light), Red (moderate), Magenta (strong).
  - Peak wind hours are highlighted with inverted colors (black text on colored background) for easy identification.
  - Enters interactive mode after display (use -x to exit immediately).
  - Perfect for wind assessment and outdoor activity planning.

- `-Observations` or `-o` [switch]
  - Shows historical weather observations for the last 7 days.
  - Displays daily aggregates including high/low temperatures, average and maximum wind speeds, wind direction, barometric pressure (inHg) with color coding, humidity, total precipitation, general conditions, and cloud summary (Clouds:) on the same line as Conditions when the station provides cloud data (white label, gray data; omitted when not). Cloud codes: SKC, FEW, SCT, BKN, OVC, VV.
  - Includes moon phase information and windchill/heat index/WBGT (with `-wbgt`) when applicable.
  - Only shows days that have actual observation data available.
  - Enters interactive mode after display (use -x to exit immediately).
  - Perfect for reviewing recent weather patterns and historical conditions.

- `-NoAutoUpdate` or `-u` [switch]
  - Starts with automatic updates disabled.
  - Auto-updates are enabled by default (every 10 minutes).
  - Can be toggled on/off during interactive mode with the 'U' key.

- `-Magic` or `-m` [switch]
  - Enables Golden/Blue hour timing in Current Conditions.
  - Displays the lines immediately before `Updated:`.
  - Labels are `Golden Hour` / `Blue Hour` while active, otherwise `Next Golden Hour` / `Next Blue Hour`.

- `-NoInteractive` or `-x` [switch]
  - Exits immediately after displaying weather data (no interactive mode or control bar).
  - Perfect for scripting and automation scenarios.
  - Can be combined with other display flags (e.g., `-h -x` for hourly view then exit).

- `-NoBar` or `-b` [switch]
  - Starts with the interactive control bar hidden.
  - Toggle visibility anytime in interactive mode with the **B** key.

- `-UseWbgt` or `-wbgt` [switch]
  - Uses estimated outdoor WBGT instead of heat index for the feels-like bracket when temp > 50°F.
  - Bracket color follows the same temperature bands as dry-bulb (Blue <33°F / default / alert >89°F), with web-app rules including a warm band from 75°F.
  - Applies in current conditions, hourly, daily, and observations displays.

- `-Noaa` [string]
  - Overrides automatic NOAA station selection with a specific station ID.
  - When specified, uses the given station ID regardless of distance (ignores 100-mile limit).
  - Still calculates and displays distance from location to the specified station.
  - Useful for accessing specific tide stations or stations beyond the normal 100-mile radius.
  - Example: `-Noaa 9440083` for Vancouver, WA station.
  - **Tip — finding a station ID:** Open the [NOAA Tides & Currents map](https://tidesandcurrents.noaa.gov/map/), pan/zoom to your area, and click a station marker. The popup shows the numeric **Station ID** (e.g. `9440083`); use that value with `-Noaa`. You can also open the station’s detail page from the map — the ID appears in the page URL as `id=9440083`.

## Examples

### Example 1: Get weather by zip code
```powershell
.\gf.ps1 97219
```

### Example 2: Get weather using automatic location detection
```powershell
.\gf.ps1 here
```

### Example 3: Get terse weather by city and state
```powershell
.\gf.ps1 -t "Portland, OR"
```

### Example 4: Get terse weather using automatic location detection
```powershell
.\gf.ps1 here -t
```

### Example 5: Get hourly forecast only
```powershell
.\gf.ps1 -h "Portland, OR"
```

### Example 6: Get enhanced 7-day forecast (daily mode)
```powershell
.\gf.ps1 -d "Portland, OR"
```

### Example 7: Get hourly forecast and exit immediately (for scripting)
```powershell
.\gf.ps1 -h -x "Portland, OR"
```

### Example 8: Get terse forecast and exit immediately
```powershell
.\gf.ps1 -t -x 97219
```

### Example 9: Get rain likelihood forecast with sparklines
```powershell
.\gf.ps1 -r 97219
```

### Example 10: Get rain forecast for city and state
```powershell
.\gf.ps1 -rain "Portland, OR"
```

### Example 11: Get wind outlook forecast with direction glyphs
```powershell
.\gf.ps1 -w 97219
```

### Example 12: Get wind forecast for city and state
```powershell
.\gf.ps1 -wind "Portland, OR"
```

### Example 13: Get historical observations
```powershell
.\gf.ps1 -o 97219
```

### Example 14: Get observations for city and state
```powershell
.\gf.ps1 -observations "Portland, OR"
```

### Example 15: View help information
```powershell
.\gf.ps1 -Help
```

### Example 15a: AirNow API key setup (AQI optional)
```powershell
.\gf.ps1 -aqi
```

### Example 16: Start with control bar hidden
```powershell
.\gf.ps1 -b "Portland, OR"
```

### Example 17: Use specific NOAA station by ID
```powershell
.\gf.ps1 "Vancouver, WA" -Noaa 9440083
```

> **Tip:** Locate station IDs on the [NOAA Tides & Currents map](https://tidesandcurrents.noaa.gov/map/) — click a marker to see the ID in the popup, or in the station page URL (`id=…`).

### Example 18: Enable Magic Hours in Current Conditions
```powershell
.\gf.ps1 "Portland, OR" -m
```

### Example 19: Use WBGT instead of heat index
```powershell
.\gf.ps1 "Portland, OR" -wbgt
```

## Full Display Mode

When no display-mode flag is set (`-t`, `-h`, `-d`, `-r`, `-w`, or `-o`), the script shows the **full report**:

- Current conditions (with optional Magic Hours, moon phase, irradiance, AQI)
- Today's and tomorrow's detailed forecast text
- 12-hour hourly forecast table
- 7-day forecast summary (compact; use **D** or `-d` for the enhanced daily view)
- Active weather alerts (with details)
- Location metadata (timezone, coordinates, elevation, radar station)
- Clickable NWS forecast/graph/radar links and NOAA tide resources when available

Unless `-x` is passed, the script then enters interactive mode with the control bar so you can switch views with hotkeys.

## Observations Mode

The observations mode (`-o` or `-observations`) provides historical weather data from the National Weather Service observation stations API. This mode displays daily aggregates of weather conditions for the last 7 days, showing only days that have actual observation data available.

### Observations Mode Features:

- **Historical Data:** Shows weather observations from the last 7 days
- **Sunrise/Sunset/Day Length:** Displays sunrise time, sunset time, and day length for each observation day (calculated astronomically)
- **Daily Aggregates:** Displays high/low temperatures, average and maximum wind speeds, wind direction, barometric pressure (inHg) after wind with color coding, humidity, total precipitation, general conditions, and cloud summary (Clouds:) on the same line as Conditions when available (white label, gray data; omitted when not). Cloud codes: SKC, FEW, SCT, BKN, OVC, VV
- **Moon Phase Information:** Includes moon phase emoji and information for each day
- **Windchill/Heat Index/WBGT:** Calculates windchill (≤50°F), heat index (≥80°F), or estimated WBGT (with `-wbgt`) when applicable
- **Data Filtering:** Only displays days that have actual observation data (skips days with no data)
- **Color Coding:** Uses the same color coding rules as other modes (temperature, wind speed, etc.)
- **Interactive Mode:** Enters interactive mode after display (use -x to exit immediately)
- **Pagination Support:** Automatically fetches all pages of observation data when preloading in full mode
- **Precipitation Accuracy:** Precipitation values correctly converted from millimeters to inches for accurate display

### Observations Mode Usage:

```powershell
# Basic observations display
.\gf.ps1 -o 97219

# Observations for city/state
.\gf.ps1 -observations "Portland, OR"

# Observations with verbose output
.\gf.ps1 -o 97219 -Verbose
```

### Observations Mode Display Format:

Each day shows:
- **Day Name and Date:** Day of week and date (e.g., "Thursday (11/06)")
- **High/Low Temperatures:** Maximum and minimum temperatures for the day
- **Wind Information:** Maximum wind speed (and average if different), with cardinal direction
- **Precipitation:** Total precipitation for the day in inches (if any), accurately converted from millimeters
- **Humidity:** Average relative humidity percentage
- **Conditions:** Most common weather condition description for the day
- **Clouds:** Cloud summary on the same line as Conditions when the station provides cloud data (e.g. "Conditions: Cloudy Clouds: BKN 2,400 ft, OVC 3,900 ft"); omitted when not available. Codes: SKC (clear), FEW (few), SCT (scattered), BKN (broken), OVC (overcast), VV (vertical visibility)
- **Moon Phase:** Moon phase emoji and information

### Observations Mode Use Cases:

- **Weather Review:** Review recent weather patterns and conditions
- **Historical Analysis:** Analyze weather trends over the past week
- **Activity Planning:** Understand recent weather conditions for planning future activities
- **Data Verification:** Verify forecast accuracy by comparing to actual observations
- **Pattern Recognition:** Identify weather patterns and trends

## Interactive Mode

By default, after weather data is displayed, the script enters **interactive mode** when the console supports keyboard input (PowerShell, cmd, Windows Terminal, and similar). Use `-x` to skip this and exit immediately.

Interactive mode shows a control bar with hotkey hints (hide with **B** or start hidden with `-b`) and lets you switch views without restarting.

### How to Use Interactive Mode:

1. **Run the script** (with or without a location; omit `-x` unless scripting)
2. **Wait for the weather data to load** and display
3. **Use keyboard shortcuts** to switch between views:
   - **H** — Hourly forecast (12-hour page; **↑**/**↓** scroll up to 48 hours)
   - **D** — Enhanced 7-day forecast
   - **T** — Terse mode (current + today)
   - **R** — Rain forecast (sparklines)
   - **W** — Wind forecast (glyphs)
   - **O** — Observations (historical data)
   - **G** — Refresh weather data
   - **U** — Toggle automatic updates
   - **B** — Toggle control bar on/off
   - **F** — Full display
   - **Enter**, **Esc**, or **Ctrl+C** — Exit the script

### Interactive Mode Benefits:

- **Quick View Switching:** No need to restart the script to see different data
- **Efficient Planning:** Switch between hourly and daily views for different planning needs
- **Focused Information:** Get exactly the weather data you need without scrolling through everything
- **Auto-Refresh:** Weather data automatically refreshes every 10 minutes to keep information current
- **Manual Refresh:** Press **G** to manually refresh data at any time
- **Hourly Scrolling:** In hourly view, **↑**/**↓** move through 12-hour pages covering up to 48 hours

### When Interactive Mode Is Skipped:

- Pass `-x` or `-NoInteractive` for one-shot/scripting use
- Console key input is unavailable (redirected stdin, unsupported host) — the script exits after the initial display

**Note:** Rain (`-r`), wind (`-w`), and hourly (`-h`) modes also enter interactive mode unless `-x` is set.

## Rain Forecast Mode

The rain forecast mode (`-r` or `-rain`) provides a unique visual representation of rain likelihood over the next 96 hours using sparklines. This mode is perfect for planning outdoor activities and understanding precipitation patterns.

### Rain Forecast Features:

- **96-Hour Coverage:** Shows rain probability for the next 4 days (96 hours)
- **Visual Sparklines:** Each character represents one hour of rain likelihood
- **Color-Coded Intensity:**
  - **White** ( ): No rain likelihood (0%)
  - **White** (▁): Very low rain likelihood (1-10%)
  - **Cyan** (▂): Low rain likelihood (11-33%)
  - **Green** (▃): Light rain likelihood (34–44%)
  - **Yellow** (▄▅): Medium rain likelihood (45–80%)
  - **Red** (▇): High rain likelihood (81%+)
- **Day-by-Day Display:** Up to 5 days shown with abbreviated day names
- **Hourly Precision:** Each sparkline character represents one hour (00:00 to 23:00)
- **Interactive Mode:** Enters interactive mode after display (use `-x` to exit immediately)

### Rain Forecast Usage:

```powershell
# Basic rain forecast
.\gf.ps1 -r 97219

# Rain forecast for city/state
.\gf.ps1 -rain "Portland, OR"

# Rain forecast with verbose output
.\gf.ps1 -r 97219 -Verbose
```

### Reading the Rain Forecast:

- **Day Labels:** White text showing abbreviated day names (Mon, Tue, Wed, etc.)
- **Sparkline Characters:** Visual representation of rain likelihood for each hour
- **Color Coding:** Instantly identify high-risk periods for rain
- **Time Alignment:** Each character represents one hour, aligned from 00:00 to 23:00
- **Missing Data:** Hours without forecast data show as blank spaces

This mode is particularly useful for:
- Planning outdoor events and activities
- Understanding precipitation timing
- Identifying dry periods for travel
- Quick visual assessment of rain patterns

## Wind Forecast Mode

The wind forecast mode (`-w` or `-wind`) provides a unique visual representation of wind patterns over the next 96 hours using directional glyphs. This mode is perfect for understanding wind patterns, planning outdoor activities, and assessing wind conditions.

### Wind Forecast Features:

- **96-Hour Coverage:** Shows wind direction and speed for the next 4 days (96 hours)
- **Visual Direction Glyphs:** Each character represents one hour of wind direction and speed
- **Color-Coded Intensity:**
  - **White**: Calm (≤5 mph)
  - **Yellow**: Light breeze (6–9 mph)
  - **Red**: Moderate wind (10–14 mph)
  - **Magenta**: Strong wind (15+ mph)
- **Day-by-Day Display:** Up to 5 days shown with abbreviated day names
- **Hourly Precision:** Each glyph represents one hour (00:00 to 23:00)
- **Peak Wind Highlighting:** Hours with the highest wind speed for each day are displayed with inverted colors (black text on colored background)
- **Interactive Mode:** Enters interactive mode after display (use `-x` to exit immediately)

### Wind Forecast Usage:

```powershell
# Basic wind forecast
.\gf.ps1 -w 97219

# Wind forecast for city/state
.\gf.ps1 -wind "Portland, OR"

# Wind forecast with verbose output
.\gf.ps1 -w 97219 -Verbose
```

### Reading the Wind Forecast:

- **Day Labels:** White text showing abbreviated day names (Mon, Tue, Wed, etc.)
- **Max Wind Speed:** Highest wind speed for each day with color coding
- **Direction Glyphs:** Visual representation of wind direction and intensity for each hour
- **Color Coding:** Instantly identify wind intensity levels
- **Peak Wind Highlighting:** Hours with the highest wind speed for each day are highlighted with inverted colors
- **Time Alignment:** Each glyph represents one hour, aligned from 00:00 to 23:00
- **Missing Data:** Hours without forecast data show as blank spaces

This mode is particularly useful for:
- Planning outdoor activities and wind-dependent sports
- Understanding wind patterns and direction changes
- Identifying calm periods for outdoor events
- Quick visual assessment of wind conditions
- Planning wind-dependent activities (sailing, flying, etc.)

## Loading Messages

The script provides dynamic loading messages that update during the data fetching process:

- **Geocoding:** Displays "Geocoding ($Location)..." when geocoding starts
- **Forecast Loading:** Updates to "Loading $location Forecast..." when fetching forecast data
- **Hourly Loading:** Updates to "Loading $location Hourly..." when fetching hourly data
- **Screen Clearing:** All loading messages are cleared before displaying weather data for a clean presentation

These messages provide clear feedback about the script's progress and help users understand what data is being loaded.

## Changelog

- **v2.2** — Soft-warn on unrecognized CLI options (e.g. accidental `-c`) and continue; suppress NWS test/monitoring-only alerts; current-conditions header shows ⚠️/🌡 when alerts are active (matches forecast web); fix alert section when API returns a single GeoJSON feature.
- **v2.1** — Moon phase, rain/wind sparklines, observations mode, auto-refresh, solar irradiance, and related enhancements (see GEMINI.md).

## Notes
- The National Weather Service API is free and requires no API key.
- The API is limited to locations within the United States.
- Precipitation values are automatically converted from millimeters (NWS API standard) to inches for display.
- This script uses OpenStreetMap Nominatim for geocoding (US locations).
- This script uses the National Weather Service API endpoints:
  - `/points/{lat},{lon}` - Get metadata for a location
  - `/gridpoints/{office}/{gridX},{gridY}/forecast` - Get forecast data
  - `/gridpoints/{office}/{gridX},{gridY}/forecast/hourly` - Get hourly forecast
  - `/alerts/active` - Get active weather alerts
  - `/points/{lat},{lon}/stations` - Get observation stations for a location
  - `/stations/{stationId}/observations` - Get historical observations from a station
- This script also uses the AirNow API endpoint for AQI:
  - `https://www.airnowapi.org/aq/observation/latLong/current/?format=application/json&latitude={lat}&longitude={lon}&distance=25&API_KEY={key}`

## Features Added/Enhanced
- **Sunrise/Sunset Times:** Calculated using NOAA astronomical algorithms based on location coordinates and time zone. During polar night or polar day, sun times use `MM/dd HH:mm` format. All displayed times (hourly forecasts, sunrise, sunset, update times) are shown in the destination location's local timezone, not your system's timezone.
- **Solar Irradiance:** Clear-sky global horizontal irradiance (GHI) in W/m² at the current time plus peak at location solar noon with time, displayed as "Irradiance: XW/m2 [Peak YW/m2 @ h:mm]" in white when available (including terse mode). Estimate only (NWS does not provide irradiance).
- **Temperature Trend:** Rising/falling/steady icons on the current temperature line when supported by observation data.
- **Estimated WBGT:** Optional `-wbgt` feels-like bracket using Stull wet-bulb + simplified globe term (matches forecast web heuristic).
- **Alert Header Icons:** ⚠️ and 🌡 prefixes on the current-conditions title when relevant alerts are in effect (v2.2).
- **Humidity Data:** Available in current conditions display

## API Information
- **NWS Base URL:** https://api.weather.gov/
- **Geocoding:** OpenStreetMap Nominatim (https://nominatim.openstreetmap.org/) — no API key required
- **User Agent:** GetForecast/1.0 (081625PDX)
- **Format:** GeoJSON
- **Rate Limits:** None specified, but please be respectful of the service
- **AirNow (optional AQI):** `https://www.airnowapi.org/aq/observation/latLong/current/` — requires your own API key in the **`AirNowAPI`** User environment variable; the script never embeds a key.