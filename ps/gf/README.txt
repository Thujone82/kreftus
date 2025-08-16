# gf - Get Forecast (NWS Edition)

## Author
Kreft&Cursor

## Description
`gf.ps1` is a PowerShell script that retrieves and displays detailed weather information for a specified location using the National Weather Service API. It can accept a US zip code or a "City, State" string as input.

The script first uses a geocoding service to determine the latitude and longitude of the location, then fetches the current weather, daily forecasts, and weather alerts from the National Weather Service.

## Features
- **Flexible Location Input:** Accepts 5-digit zip codes or city/state names (e.g., "Portland, OR").
- **Interactive Prompt:** If no location is provided, the script displays a welcome screen and prompts for input.
- **Comprehensive Weather Data:** Displays a wide range of information, including:
  - Current temperature and conditions.
  - Detailed daily and tomorrow forecasts.
  - Wind speed and direction.
  - Weather alerts and warnings.
- **Smart Color-Coding:** Important metrics are color-coded for quick assessment:
  - **Temperature:** Turns red if below 33°F or above 89°F.
  - **Wind:** Turns red if wind speed is 16 mph or greater.
- **Weather Alerts:** Automatically displays any active weather alerts (e.g., warnings, watches) for the location.
- **Quick Link:** Provides a direct URL to the weather.gov forecast map for the location.
- **Smart Exit:** If run from an environment other than a standard command prompt (like by double-clicking), it will pause and wait for user input before closing the window.

## Requirements
- PowerShell
- An active internet connection.
- No API key required - uses the free National Weather Service API.

## How to Run
1.  Open a PowerShell terminal.
2.  Navigate to the directory where `gw.ps1` is located.
3.  Run the script using one of the formats below.

*Note: To run PowerShell scripts, you may need to adjust your execution policy. You can do this by running `Set-ExecutionPolicy Bypass` from an administrator PowerShell prompt.*

## Configuration
The script uses a user agent string "202508161459PDX" for API requests, which is stored in the configuration file. This can be modified if needed.

**Configuration File Locations:**
- **Windows:** `C:\Users\<YourUsername>\AppData\Roaming\gw\gw.ini`
- **Linux/macOS:** `/home/<YourUsername>/.config/gw/gw.ini`

## Parameters

- `Location` [string] (Positional: 0)
  - The location for which to retrieve weather. Can be a 5-digit US zip code or a "City, State" string.
  - If omitted, the script will prompt you for it.

- `-Help` [switch]
  - Displays a detailed help and usage message in the console.

- `-Verbose` [switch]
  - A built-in PowerShell parameter that, when used with this script, will display the URLs being called for geocoding and weather data. Useful for debugging.

- `-Terse` or `-t` [switch]
  - Provides a less busy, streamlined view.
  - **Simplifies Alerts:** For any active weather alerts, only the main title and the start/end times are shown, hiding the detailed description.

## Examples

### Example 1: Get weather by zip code
```powershell
.\gw.ps1 97219
```

### Example 2: Get terse weather by city and state
```powershell
.\gw.ps1 -t "Portland, OR"
```

### Example 3: View help information
```powershell
.\gw.ps1 -Help
```

## Notes
- The National Weather Service API is free and requires no API key.
- The API is limited to locations within the United States.
- This script uses the National Weather Service API endpoints:
  - `/points/{lat},{lon}` - Get metadata for a location
  - `/gridpoints/{office}/{gridX},{gridY}/forecast` - Get forecast data
  - `/gridpoints/{office}/{gridX},{gridY}/forecast/hourly` - Get hourly forecast
  - `/alerts/active` - Get active weather alerts

## Features Removed from Original OpenWeatherMap Version
Due to differences in the National Weather Service API, the following features are not available:
- UV Index data
- Humidity data
- Sunrise/Sunset times
- Moonrise/Moonset times
- Moon phase information
- Temperature trend indicators (rising/falling)
- Detailed weather overview reports
- Rain/Snow precipitation amounts

## API Information
- **Base URL:** https://api.weather.gov/
- **User Agent:** 202508161459PDX
- **Format:** GeoJSON
- **Rate Limits:** None specified, but please be respectful of the service