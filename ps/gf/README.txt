# gf - Get Forecast (NWS Edition)

## Author
Kreft&Cursor

## Description
`gf.ps1` is a PowerShell script that retrieves and displays detailed weather information for a specified location using the National Weather Service API. It can accept a US zip code or a "City, State" string as input.

The script first uses a geocoding service to determine the latitude and longitude of the location, then fetches the current weather, daily forecasts, and weather alerts from the National Weather Service.

## Features
- **Flexible Location Input:** Accepts 5-digit zip codes, city/state names (e.g., "Portland, OR"), or "here" for automatic location detection.
- **Automatic Location Detection:** Use "here" to automatically detect your location based on your IP address.
- **Interactive Prompt:** If no location is provided, the script displays a welcome screen and prompts for input.
- **Comprehensive Weather Data:** Displays a wide range of information, including:
  - Current temperature and conditions.
  - Detailed daily and tomorrow forecasts.
  - Wind speed and direction.
  - Sunrise and sunset times (calculated astronomically).
  - Weather alerts and warnings.
  - Rain likelihood forecast with visual sparklines.
  - Wind outlook forecast with direction glyphs.
- **Smart Color-Coding:** Important metrics are color-coded for quick assessment:
  - **Temperature:** Turns red if below 33°F or above 89°F.
  - **Wind:** Turns red if wind speed is 16 mph or greater.
  - **Rain Likelihood:** Color-coded sparklines show rain probability at a glance (white for very low, cyan for low, green for light, yellow for medium, red for high).
  - **Wind Outlook:** Color-coded directional glyphs show wind patterns and intensity, with peak wind hours highlighted using inverted colors.
- **Weather Alerts:** Automatically displays any active weather alerts (e.g., warnings, watches) for the location.
- **Quick Link:** Provides a direct URL to the weather.gov forecast map for the location.
- **Smart Exit:** If run from an environment other than a standard command prompt (like by double-clicking), it will pause and wait for user input before closing the window.
- **Interactive Mode:** When run interactively, provides keyboard shortcuts to switch between different display modes:
     - **[H]** - Switch to hourly forecast only
   - **[D]** - Switch to 7-day forecast only  
   - **[T]** - Switch to terse mode (current conditions + today's forecast)
   - **[R]** - Switch to rain forecast mode (sparklines)
   - **[W]** - Switch to wind forecast mode (direction glyphs)
   - **[G]** - Refresh weather data (auto-refreshes every 10 minutes)
   - **[U]** - Toggle automatic updates on/off
   - **[F]** - Return to full display
   - **[Enter]** or **[Esc]** - Exit the script
   - **Ctrl+C** will also exit the script

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


#### 2. Change Console Encoding to UTF-8 [Note: The script attempts to do this for you]
Emojis are complex Unicode characters (often outside the Basic Multilingual Plane) that require UTF-8 encoding. While modern Windows Terminal handles this well, you can explicitly set the output encoding in your PowerShell profile script to prevent issues, especially when redirecting output.

Execute In PowerShell 

# Set console output encoding to UTF-8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

# Set PowerShell output encoding for cmdlets
$OutputEncoding = [System.Text.Encoding]::UTF8


## Parameters

- `Location` [string] (Positional: 0)
  - The location for which to retrieve weather. Can be a 5-digit US zip code or a "City, State" string.
  - If omitted, the script will prompt you for it.

- `-Help` [switch]
  - Displays a detailed help and usage message in the console.

- `-Verbose` [switch]
  - A built-in PowerShell parameter that, when used with this script, will display the URLs being called for geocoding and weather data. Useful for debugging.

- `-Terse` or `-t` [switch]
  - Shows only current conditions and today's forecast (plus alerts if they exist).
  - Provides a streamlined, focused view for quick weather checks.

- `-Hourly` or `-h` [switch]
  - Shows only the 12-hour hourly forecast.
  - Perfect for planning activities throughout the day.

- `-SevenDay` or `-7` [switch]
  - Shows only the 7-day forecast summary.
  - Great for weekly planning.

- `-Daily` or `-d` [switch]
  - Same as `-7` (shows only the 7-day forecast summary).
  - Alternative flag for the same functionality.

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

- `-NoAutoUpdate` or `-u` [switch]
  - Starts with automatic updates disabled.
  - Auto-updates are enabled by default (every 10 minutes).
  - Can be toggled on/off during interactive mode with the 'U' key.

- `-NoInteractive` or `-x` [switch]
  - Exits immediately after displaying weather data (no interactive mode).
  - Perfect for scripting and automation scenarios.
  - Can be combined with other display flags (e.g., `-h -x` for hourly view then exit).

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

### Example 6: Get 7-day forecast only
```powershell
.\gf.ps1 -7 "Portland, OR"
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

### Example 13: View help information
```powershell
.\gf.ps1 -Help
```

## Interactive Mode

When you run the script by double-clicking it or from a non-terminal environment, it enters **Interactive Mode**. This mode allows you to switch between different display views using keyboard shortcuts without having to restart the script.

### How to Use Interactive Mode:

1. **Run the script interactively** (double-click the .ps1 file or run from Windows Explorer)
2. **Wait for the weather data to load** and display
3. **Use keyboard shortcuts** to switch between views:
       - **H** - Switch to hourly forecast only (12-hour view)
    - **D** - Switch to 7-day forecast only (weekly view)
    - **T** - Switch to terse mode (current conditions + today's forecast)
    - **R** - Switch to rain forecast mode (sparklines)
    - **W** - Switch to wind forecast mode (direction glyphs)
    - **G** - Refresh weather data (auto-refreshes every 10 minutes)
    - **U** - Toggle automatic updates on/off
    - **F** - Return to full display (all information)
    - **Enter** - Exit the script

### Interactive Mode Benefits:

- **Quick View Switching:** No need to restart the script to see different data
- **Efficient Planning:** Switch between hourly and daily views for different planning needs
- **Focused Information:** Get exactly the weather data you need without scrolling through everything
- **Auto-Refresh:** Weather data automatically refreshes every 10 minutes to keep information current
- **Manual Refresh:** Press 'G' to manually refresh data at any time
- **User-Friendly:** Perfect for users who prefer mouse/keyboard interaction over command-line options

### When Interactive Mode Activates:

Interactive mode automatically activates when the script detects it's not running from a standard terminal environment (PowerShell, Command Prompt, or Windows Terminal). This typically happens when:
- Double-clicking the .ps1 file
- Running from Windows Explorer
- Running from a GUI application
- Running from certain development environments

**Note:** Interactive mode can be disabled using the `-x` or `-NoInteractive` flag, which is useful for scripting and automation scenarios.

## Rain Forecast Mode

The rain forecast mode (`-r` or `-rain`) provides a unique visual representation of rain likelihood over the next 96 hours using sparklines. This mode is perfect for planning outdoor activities and understanding precipitation patterns.

### Rain Forecast Features:

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
  - **White**: Calm conditions (≤5mph)
  - **Yellow**: Light breeze (>5mph and ≤9mph)
  - **Red**: Moderate wind (>9mph and ≤14mph)
  - **Magenta**: Strong wind (>14mph)
- **Day-by-Day Display:** Up to 5 days shown with abbreviated day names
- **Hourly Precision:** Each glyph represents one hour (00:00 to 23:00)
- **Peak Wind Highlighting:** Hours with the highest wind speed for each day are displayed with inverted colors (black text on colored background)
- **Automatic Exit:** No interactive mode - displays data and exits immediately

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
- Moonrise/Moonset times
- Moon phase information
- Temperature trend indicators (rising/falling)
- Detailed weather overview reports
- Rain/Snow precipitation amounts

## Features Added/Enhanced
- **Sunrise/Sunset Times:** Calculated using NOAA astronomical algorithms based on location coordinates and time zone
- **Humidity Data:** Available in current conditions display

## API Information
- **Base URL:** https://api.weather.gov/
- **User Agent:** GetForecast/1.0 (081625PDX)
- **Format:** GeoJSON
- **Rate Limits:** None specified, but please be respectful of the service