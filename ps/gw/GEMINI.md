# Gemini Project File

## Project: gw (Get Weather)

**Author:** Kreft&Gemini[Gemini 2.5 Pro (preview)]
**Date:** 2025-08-02
**Version:** 1.0

---

### Description

`gw` is a command-line weather utility for PowerShell that provides detailed, real-time weather information for any specified location. It leverages the OpenWeatherMap One Call API 3.0 to fetch a comprehensive set of data, including current conditions, daily forecasts, and active weather alerts.

The script is designed for ease of use, accepting flexible location inputs like US zip codes or "City, State" strings. It features a one-time, guided setup for API key configuration and stores the key securely in a user-specific configuration directory. The output is color-coded to highlight important metrics, making it easy to assess conditions at a glance.

### Key Functionality

- **API Key Management:** Automatically prompts for an OpenWeatherMap API key on the first run, validates it, and saves it to `gw.ini` for future use.
- **Flexible Location Input:** Can determine latitude and longitude from either a 5-digit US zip code or a "City, State" formatted string.
- **Comprehensive Data Display:** Shows current temperature, high/low forecast, humidity, UV Index, wind speed/direction, sunrise/sunset times, and moon phase.
- **Detailed Weather Report:** Includes a multi-paragraph, human-readable weather summary for the location.
- **Color-Coded Metrics:** Key data points (temperature, wind speed, UV index) change color to red to indicate potentially hazardous conditions.
- **Weather Alerts:** Automatically fetches and displays any active weather alerts (e.g., warnings, watches) from official sources.
- **Terse Mode (`-t`):** Offers a streamlined, less verbose output that hides the detailed report and simplifies alert descriptions for quicker checks.
- **Interactive & Scriptable:** Can be run with command-line arguments or interactively, where it will prompt the user for a location.
- **Smart Exit:** Pauses for user input before closing if run outside of a standard terminal (e.g., by double-clicking).

### How to Run

The script is executed from a PowerShell terminal.

**Get Weather:**
- **By Zip Code:** `.\gw.ps1 97219`
- **By City, State:** `.\gw.ps1 "Portland, OR"`
- **Terse Mode:** `.\gw.ps1 -t "Portland, OR"`

**Help:**
- `.\gw.ps1 -Help`

### Dependencies

- Windows PowerShell
- An active internet connection
- A free API key from [OpenWeatherMap One Call API 3.0](https://openweathermap.org/api/one-call-3)

### File Structure

- `gw.ps1`: The main executable script.
- `gw.ini`: Configuration file for storing the API key (auto-generated in the user's AppData or .config directory).
- `README.txt`: Detailed user documentation.
