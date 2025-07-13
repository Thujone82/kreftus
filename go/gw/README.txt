# gw - Get Weather (Go Edition)

## Author
Kreft&Gemini

## Description
`gw` is a native, cross-platform command-line application that retrieves and displays detailed weather information for a specified location using the OpenWeatherMap One Call API 3.0. It can accept a US zip code or a "City, State" string as input.

This Go version is compiled for Windows and Linux for maximum performance and portability.

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
- **Smart Color-Coding:** Important metrics are color-coded for quick assessment.
- **Weather Alerts:** Automatically displays any active weather alerts for the location.
- **Quick Link:** Provides a direct URL to the weather.gov forecast map for the location.
- **Smart Exit:** Pauses for user input before closing if run by double-clicking.

## Requirements
- An active internet connection.
- A free one-call-3 API key from OpenWeatherMap.
TIP: Set maximum calls per day to 1000 to prevent charges.

## How to Run
1.  Download the appropriate binary for your system (Windows or Linux).
2.  Open a terminal or command prompt.
3.  Navigate to the directory where the `gw` executable is located.
4.  Run the application using one of the formats below.

## Configuration & First-Time Setup
On the first run, `gw` will detect that no API key is configured and will guide you through a one-time setup process.

1.  You will be prompted to enter your free **One Call API 3.0 Key** from OpenWeatherMap.
2.  The application will validate the key to ensure it's working correctly.
3.  Once validated, the key will be saved to a `gw.ini` configuration file.

This file is stored in a standard user configuration directory on your system. You will not be prompted for the key again unless the file is deleted or the key becomes invalid.

**Configuration File Locations:**
- **Windows:** `C:\Users\<YourUsername>\AppData\Roaming\gw\gw.ini`
- **Linux:** `/home/<YourUsername>/.config/gw/gw.ini`

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
```shell
./gw 97219
```

### Example 2: Get terse weather by city and state
```shell
./gw -t "Portland, OR"
```

### Example 3: View help information
```shell
./gw -h
```