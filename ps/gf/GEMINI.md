# Gemini Project File

## Project: gf (Get Forecast) - NWS Edition

**Author:** Kreft&Cursor
**Date:** 2025-08-16
**Version:** 2.0

---

### Description

`gf` is a command-line weather utility for PowerShell that provides detailed, real-time weather information for any specified location within the United States. It leverages the National Weather Service API to fetch comprehensive weather data, including current conditions, daily forecasts, and active weather alerts.

The script is designed for ease of use, accepting flexible location inputs like US zip codes or "City, State" strings. It uses free geocoding services to determine coordinates and then fetches weather data from the official National Weather Service API. The output is color-coded to highlight important metrics, making it easy to assess conditions at a glance.

### Key Functionality

- **No API Key Required:** Uses the free National Weather Service API which requires no registration or API key.
- **Flexible Location Input:** Can determine latitude and longitude from either a 5-digit US zip code or a "City, State" formatted string.
- **Comprehensive Data Display:** Shows current temperature, conditions, detailed forecasts for today and tomorrow, and wind information.
- **Weather Alerts:** Automatically fetches and displays any active weather alerts (e.g., warnings, watches) from official sources.
- **Color-Coded Metrics:** Key data points (temperature, wind speed) change color to red to indicate potentially hazardous conditions.
- **Multiple Display Modes:**
  - **Full Mode (default):** Shows all available weather information
  - **Terse Mode (`-t`):** Shows only current conditions and today's forecast (plus alerts)
  - **Hourly Mode (`-h`):** Shows only the 12-hour hourly forecast
  - **7-Day Mode (`-7` or `-d`):** Shows only the 7-day forecast summary
  - **No-Interactive Mode (`-x`):** Exits immediately after displaying data (perfect for scripting)
- **Interactive Mode:** When run from non-terminal environments, provides keyboard shortcuts for dynamic view switching:
  - **[H]** - Switch to hourly forecast only
  - **[D]** - Switch to 7-day forecast only
  - **[T]** - Switch to terse mode
  - **[ESC]** - Return to full display
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
- **[ESC]** - **Full View:** Return to complete weather information display
- **[Enter]** - **Exit:** Close the script and return to the system

#### Benefits of Interactive Mode:

- **User-Friendly Interface:** Perfect for users who prefer GUI-like interaction over command-line options
- **Efficient Data Exploration:** Quickly switch between different weather perspectives without multiple script executions
- **Contextual Planning:** View hourly data for immediate planning, then switch to weekly for long-term planning
- **Reduced Cognitive Load:** Focus on specific weather aspects without information overload
- **Accessibility:** Makes weather data accessible to users unfamiliar with command-line interfaces

#### Technical Implementation:

The interactive mode uses PowerShell's `$Host.UI.RawUI.ReadKey()` method to capture keyboard input without requiring the Enter key. It implements a state machine that:
- Monitors for specific virtual key codes (H=72, D=68, T=84, ESC=27, Enter=13)
- Dynamically re-renders the display based on the selected mode
- Maintains all weather data in memory for instant switching
- Provides clear visual feedback about available options

#### Use Cases:

- **Quick Weather Checks:** Double-click the script and use [T] for immediate current conditions
- **Planning Activities:** Use [H] to see hourly breakdown for day planning
- **Weekly Planning:** Use [D] to see the 7-day outlook for weekly scheduling
- **Comprehensive Review:** Use [ESC] to see all available weather information
- **Scripting & Automation:** Use `-x` flag for automated weather checks in scripts, cron jobs, or scheduled tasks

### Future Enhancements

Potential improvements could include:
- Support for international locations (would require different weather APIs)
- Additional weather data points if they become available in the NWS API
- Integration with other weather services for missing data points
- Enhanced alert filtering and categorization
- Additional display modes for specific use cases
- Export functionality for weather data
- Integration with calendar/scheduling applications
