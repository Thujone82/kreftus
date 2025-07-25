'# gw - Get Weather

## Author
Kreft&Gemini

## Description
`gw.ps1` is a PowerShell script that retrieves and displays detailed weather information for a specified location using the OpenWeatherMap One Call API 3.0. It can accept a US zip code or a "City, State" string as input.

The script first uses a geocoding API to determine the latitude and longitude of the location, then fetches the current weather, daily forecasts, and a descriptive weather report.

## Features
- **Flexible Location Input:** Accepts 5-digit zip codes or city/state names (e.g., "Portland, OR").
- **Interactive Prompt:** If no location is provided, the script displays a welcome screen and prompts for input.
- **Comprehensive Weather Data:** Displays a wide range of information, including:
  - Current temperature with the day's high/low forecast.
  - A text summary of the day's forecast (e.g., "Clear sky throughout the day").
  - Current conditions (e.g., "Clear", "Clouds").
  - Humidity and UV Index.
  - Wind speed, gust speed, and cardinal direction.
  - Sunrise, sunset, moonrise, and moonset times.
  - Current moon phase.
  - A detailed, paragraph-style weather report.
- **Smart Color-Coding:** Important metrics are color-coded for quick assessment:
  - **Temperature:** Turns red if below 33°F or above 89°F.
  - **Wind:** Turns red if wind speed is 16 mph or greater.
  - **UV Index:** Turns red if the index is 6 or higher.
- **Weather Alerts:** Automatically displays any active weather alerts (e.g., warnings, watches) for the location.
- **Quick Link:** Provides a direct URL to the weather.gov forecast map for the location.
- **Smart Exit:** If run from an environment other than a standard command prompt (like by double-clicking), it will pause and wait for user input before closing the window.

## Requirements
- PowerShell
- An active internet connection.
- A free one-call-3 API key from OpenWeatherMap.
TIP: Set maximum calls per day to 1000 to prevent charges.

## How to Run
1.  Open a PowerShell terminal.
2.  Navigate to the directory where `gw.ps1` is located.
3.  Run the script using one of the formats below.

*Note: To run PowerShell scripts, you may need to adjust your execution policy. You can do this by running `Set-ExecutionPolicy Bypass` from an administrator PowerShell prompt.*

## Configuration & First-Time Setup
On the first run, `gw.ps1` will detect that no API key is configured and will guide you through a one-time setup process.

1.  You will be prompted to enter your free **One Call API 3.0 Key** from OpenWeatherMap.
2.  The script will validate the key to ensure it's working correctly.
3.  Once validated, the key will be saved to a `gw.ini` configuration file.

This file is stored in a standard user configuration directory on your system. You will not be prompted for the key again unless the file is deleted or the key becomes invalid.

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
  - Provides a less busy, streamlined view. This mode is also faster as it suppresses an API call.
  - **Removes:** The entire "Weather Report" section, including the descriptive paragraph and the forecast.weather.gov link.
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
- The OpenWeatherMap API key is currently hardcoded within the script. You can register for a free key (with usage limits) on the OpenWeatherMap website if you wish to change it.