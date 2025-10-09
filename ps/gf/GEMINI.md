# Gemini Project File

## Project: gf (Get Forecast) - NWS Edition

**Author:** Kreft&Cursor
**Date:** 2025-01-27
**Version:** 2.1

---

### Description

`gf` is a command-line weather utility for PowerShell that provides detailed, real-time weather information for any specified location within the United States. It leverages the National Weather Service API to fetch comprehensive weather data, including current conditions, daily forecasts, and active weather alerts.

The script is designed for ease of use, accepting flexible location inputs like US zip codes or "City, State" strings. It uses free geocoding services to determine coordinates and then fetches weather data from the official National Weather Service API. The output is color-coded to highlight important metrics, making it easy to assess conditions at a glance.

### Key Functionality

- **No API Key Required:** Uses the free National Weather Service API which requires no registration or API key.
- **Flexible Location Input:** Can determine latitude and longitude from either a 5-digit US zip code or a "City, State" formatted string.
- **Comprehensive Data Display:** Shows current temperature, conditions, detailed forecasts for today and tomorrow, wind information, rain likelihood forecasts with visual sparklines, and wind outlook forecasts with direction glyphs.
- **Weather Alerts:** Automatically fetches and displays any active weather alerts (e.g., warnings, watches) from official sources.
- **Color-Coded Metrics:** Key data points (temperature, wind speed) change color to red to indicate potentially hazardous conditions. Rain likelihood sparklines use color coding (white for very low, cyan for low, green for light, yellow for medium, red for high probability). Wind outlook glyphs use color coding (white for calm, yellow for light breeze, red for moderate wind, magenta for strong wind).
- **Multiple Display Modes:**
  - **Full Mode (default):** Shows all available weather information
  - **Terse Mode (`-t`):** Shows only current conditions and today's forecast (plus alerts)
  - **Hourly Mode (`-h`):** Shows only the 12-hour hourly forecast
  - **7-Day Mode (`-7` or `-d`):** Shows only the 7-day forecast summary
  - **Rain Forecast Mode (`-r` or `-rain`):** Shows rain likelihood forecast with visual sparklines for 96 hours
  - **Wind Forecast Mode (`-w` or `-wind`):** Shows wind outlook forecast with direction glyphs for 96 hours
  - **No-Interactive Mode (`-x`):** Exits immediately after displaying data (perfect for scripting)
- **Interactive Mode:** When run from non-terminal environments, provides keyboard shortcuts for dynamic view switching:
  - **[H]** - Switch to hourly forecast only
  - **[D]** - Switch to 7-day forecast only
  - **[T]** - Switch to terse mode
  - **[F]** - Return to full display
  - **[Enter]** - Exit the script
- **Interactive & Scriptable:** Can be run with command-line arguments or interactively, where it will prompt the user for a location.
- **Smart Exit:** Pauses for user input before closing if run outside of a standard terminal (e.g., by double-clicking).

### Technical Implementation

The script follows a multi-step process:

1. **Geocoding:** Uses free services (zippopotam.us for zip codes, Nominatim for city/state) to convert location input to coordinates.
2. **NWS Points Lookup:** Calls the NWS `/points/{lat},{lon}` endpoint to get grid metadata for the location.
3. **Forecast Data:** Fetches both regular forecast and hourly forecast data from the NWS gridpoints endpoints.
4. **Alerts:** Retrieves any active weather alerts for the location.
5. **Data Processing:** Parses and formats the GeoJSON responses for display.
6. **Output:** Displays formatted weather information with color coding and text wrapping.

### API Endpoints Used

- **Geocoding:** 
  - `https://api.zippopotam.us/us/{zipcode}` (for zip codes)
  - `https://nominatim.openstreetmap.org/search` (for city/state)
- **NWS Points:** `https://api.weather.gov/points/{lat},{lon}`
- **Forecast:** `https://api.weather.gov/gridpoints/{office}/{gridX},{gridY}/forecast`
- **Hourly:** `https://api.weather.gov/gridpoints/{office}/{gridX},{gridY}/forecast/hourly`
- **Alerts:** `https://api.weather.gov/alerts/active?point={lat},{lon}`

### Configuration

The script uses a hardcoded user agent string "GetForecast/1.0 (081625PDX)" for API requests. No configuration file is required.

### Features Removed from Original Version

Due to differences between the OpenWeatherMap and National Weather Service APIs, the following features are not available in this version:

- UV Index data
- Humidity data  
- Sunrise/Sunset times
- Moonrise/Moonset times
- Moon phase information
- Temperature trend indicators (rising/falling)
- Detailed weather overview reports
- Rain/Snow precipitation amounts

### Benefits of NWS API

- **Free and Open:** No API key required, no usage limits
- **Official Data:** Direct from the National Weather Service
- **US Coverage:** Comprehensive coverage of all US territories
- **Reliable:** Government-operated service with high uptime
- **Detailed Alerts:** Official weather warnings and watches

### Usage Examples

```powershell
# Get weather for a zip code
.\gf.ps1 97219

# Get weather for a city and state
.\gf.ps1 "Portland, OR"

# Get terse output (current conditions + today's forecast only)
.\gf.ps1 -t "Seattle, WA"

# Get hourly forecast only
.\gf.ps1 -h "Portland, OR"

# Get 7-day forecast only
.\gf.ps1 -7 "Portland, OR"

# Alternative 7-day forecast command
.\gf.ps1 -d "Portland, OR"

# Get hourly forecast and exit immediately (for scripting)
.\gf.ps1 -h -x "Portland, OR"

# Get terse forecast and exit immediately
.\gf.ps1 -t -x 97219

# Get rain likelihood forecast with sparklines
.\gf.ps1 97219 -r

# Get rain forecast for city and state
.\gf.ps1 -rain "Portland, OR"

# Get wind outlook forecast with direction glyphs
.\gf.ps1 97219 -w

# Get wind forecast for city and state
.\gf.ps1 -wind "Portland, OR"

# View help
.\gf.ps1 -Help
```

### Interactive Mode

The script features an advanced **Interactive Mode** that activates when run from non-terminal environments (such as double-clicking the script file). This mode provides a dynamic, user-friendly interface for exploring weather data.

#### How Interactive Mode Works:

1. **Automatic Activation:** When the script detects it's not running from a standard terminal (PowerShell, Command Prompt, Windows Terminal), it automatically enters interactive mode
2. **Manual Override:** Interactive mode can be disabled using the `-x` or `-NoInteractive` flag for scripting scenarios
2. **Display Options:** After showing the initial weather data, the script presents keyboard shortcuts for different view modes
3. **Dynamic Switching:** Users can switch between different display modes without restarting the script
4. **Persistent Session:** The script remains active until the user chooses to exit

#### Interactive Mode Controls:

- **[H]** - **Hourly View:** Switch to 12-hour hourly forecast display
- **[D]** - **Daily View:** Switch to 7-day forecast summary display  
- **[T]** - **Terse View:** Switch to streamlined view (current conditions + today's forecast)
- **[F]** - **Full View:** Return to complete weather information display
- **[Enter]** - **Exit:** Close the script and return to the system

#### Benefits of Interactive Mode:

- **User-Friendly Interface:** Perfect for users who prefer GUI-like interaction over command-line options
- **Efficient Data Exploration:** Quickly switch between different weather perspectives without multiple script executions
- **Contextual Planning:** View hourly data for immediate planning, then switch to weekly for long-term planning
- **Reduced Cognitive Load:** Focus on specific weather aspects without information overload
- **Accessibility:** Makes weather data accessible to users unfamiliar with command-line interfaces

#### Technical Implementation:

The interactive mode uses PowerShell's `$Host.UI.RawUI.ReadKey()` method to capture keyboard input without requiring the Enter key. It implements a state machine that:
- Monitors for specific virtual key codes (H=72, D=68, T=84, F=70, Enter=13)
- Dynamically re-renders the display based on the selected mode
- Maintains all weather data in memory for instant switching
- Provides clear visual feedback about available options

#### Use Cases:

- **Quick Weather Checks:** Double-click the script and use [T] for immediate current conditions
- **Planning Activities:** Use [H] to see hourly breakdown for day planning
- **Weekly Planning:** Use [D] to see the 7-day outlook for weekly scheduling
- **Comprehensive Review:** Use [F] to see all available weather information
- **Scripting & Automation:** Use `-x` flag for automated weather checks in scripts, cron jobs, or scheduled tasks

### Rain Forecast Mode (v2.1)

The rain forecast mode (`-r` or `-rain`) is a unique feature that provides visual representation of rain likelihood over the next 96 hours using sparklines. This mode is particularly useful for planning outdoor activities and understanding precipitation patterns.

#### Rain Forecast Features:

- **96-Hour Coverage:** Shows rain probability for the next 4 days (96 hours)
- **Visual Sparklines:** Each character represents one hour of rain likelihood
- **Color-Coded Intensity:**
  - **White** ( ): No rain likelihood (0%)
  - **White** (▁): Very low rain likelihood (1-10%)
  - **Cyan** (▂): Low rain likelihood (11-33%)
  - **Green** (▃): Light rain likelihood (34-40%)
  - **Yellow** (▄▅): Medium rain likelihood (41-80%)
  - **Red** (▇): High rain likelihood (81%+)
- **Day-by-Day Display:** Up to 5 days shown with abbreviated day names
- **Hourly Precision:** Each sparkline character represents one hour (00:00 to 23:00)
- **Automatic Exit:** No interactive mode - displays data and exits immediately

#### Rain Forecast Use Cases:

- **Outdoor Planning:** Identify dry periods for events and activities
- **Travel Planning:** Understand precipitation timing for trips
- **Quick Assessment:** Visual overview of rain patterns at a glance
- **Activity Scheduling:** Plan outdoor activities during low-rain periods
- **Precipitation Analysis:** Understand rain intensity and timing patterns

#### Technical Implementation:

The rain forecast mode uses the same NWS hourly forecast data but processes it differently:
- Groups hourly data by day for display
- Maps rain probability percentages to sparkline characters
- Applies color coding based on probability thresholds
- Handles missing data gracefully with blank spaces
- Uses 96-hour data limit for comprehensive coverage

### Wind Forecast Mode (v2.1)

The wind forecast mode (`-w` or `-wind`) provides a unique visual representation of wind patterns over the next 96 hours using directional glyphs. This mode is particularly useful for understanding wind patterns, planning outdoor activities, and assessing wind conditions.

#### Wind Forecast Features:

- **96-Hour Coverage:** Shows wind direction and speed for the next 4 days (96 hours)
- **Visual Direction Glyphs:** Each character represents one hour of wind direction and speed
- **Color-Coded Intensity:**
  - **White**: Calm conditions (≤5mph)
  - **Yellow**: Light breeze (>5mph and ≤9mph)
  - **Red**: Moderate wind (>9mph and ≤14mph)
  - **Magenta**: Strong wind (>14mph)
- **Day-by-Day Display:** Up to 5 days shown with abbreviated day names
- **Hourly Precision:** Each glyph represents one hour (00:00 to 23:00)
- **Automatic Exit:** No interactive mode - displays data and exits immediately

#### Wind Forecast Use Cases:

- **Outdoor Planning:** Identify calm periods for outdoor activities
- **Wind Assessment:** Understand wind patterns and direction changes
- **Activity Scheduling:** Plan wind-dependent activities (sailing, flying, etc.)
- **Quick Assessment:** Visual overview of wind conditions at a glance
- **Direction Analysis:** Track wind direction changes throughout the day

#### Technical Implementation:

The wind forecast mode uses the same NWS hourly forecast data but processes it differently:
- Groups hourly data by day for display
- Maps wind direction to directional glyphs (N, NE, E, SE, S, SW, W, NW)
- Applies color coding based on wind speed thresholds
- Uses different glyph sets for light winds (<7mph) vs strong winds (≥7mph)
- Handles missing data gracefully with blank spaces
- Uses 96-hour data limit for comprehensive coverage

### Recent Enhancements (v2.1)

- **Rain Forecast Mode:** Added visual sparkline representation of rain likelihood over 96 hours
- **Wind Forecast Mode:** Added visual directional glyph representation of wind patterns over 96 hours
- **Enhanced Color Coding:** Implemented color-coded sparklines for rain probability and wind speed visualization
- **Extended Forecast Coverage:** Both rain and wind modes use 96-hour data instead of standard 12-hour limit
- **Improved Visual Design:** Better sparkline characters and directional glyphs that don't interfere with each other
- **Comprehensive Documentation:** Updated README and project documentation

### Future Enhancements

Potential improvements could include:
- Support for international locations (would require different weather APIs)
- Additional weather data points if they become available in the NWS API
- Integration with other weather services for missing data points
- Enhanced alert filtering and categorization
- Additional display modes for specific use cases
- Export functionality for weather data
- Integration with calendar/scheduling applications
- Interactive rain forecast mode with scrolling capabilities
- Additional sparkline visualizations for other weather metrics (temperature, wind, etc.)
