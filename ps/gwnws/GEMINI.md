# Gemini Project File

## Project: gw (Get Weather) - NWS Edition

**Author:** Kreft&Gemini[Gemini 2.5 Pro (preview)]
**Date:** 2025-08-16
**Version:** 2.0

---

### Description

`gw` is a command-line weather utility for PowerShell that provides detailed, real-time weather information for any specified location within the United States. It leverages the National Weather Service API to fetch comprehensive weather data, including current conditions, daily forecasts, and active weather alerts.

The script is designed for ease of use, accepting flexible location inputs like US zip codes or "City, State" strings. It uses free geocoding services to determine coordinates and then fetches weather data from the official National Weather Service API. The output is color-coded to highlight important metrics, making it easy to assess conditions at a glance.

### Key Functionality

- **No API Key Required:** Uses the free National Weather Service API which requires no registration or API key.
- **Flexible Location Input:** Can determine latitude and longitude from either a 5-digit US zip code or a "City, State" formatted string.
- **Comprehensive Data Display:** Shows current temperature, conditions, detailed forecasts for today and tomorrow, and wind information.
- **Weather Alerts:** Automatically fetches and displays any active weather alerts (e.g., warnings, watches) from official sources.
- **Color-Coded Metrics:** Key data points (temperature, wind speed) change color to red to indicate potentially hazardous conditions.
- **Terse Mode (`-t`):** Offers a streamlined, less verbose output that simplifies alert descriptions for quicker checks.
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

The script stores a user agent string in the configuration file for API requests. The default user agent is "202508161459PDX" as requested.

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
.\gw.ps1 97219

# Get weather for a city and state
.\gw.ps1 "Portland, OR"

# Get terse output
.\gw.ps1 -t "Seattle, WA"

# View help
.\gw.ps1 -Help
```

### Future Enhancements

Potential improvements could include:
- Support for international locations (would require different weather APIs)
- Additional weather data points if they become available in the NWS API
- Integration with other weather services for missing data points
- Enhanced alert filtering and categorization
