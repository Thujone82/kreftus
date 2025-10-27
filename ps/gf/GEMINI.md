# Gemini Project File

## Project: gf (Get Forecast) - NWS Edition

**Author:** Kreft&Cursor
**Date:** 2025-01-27
**Version:** 2.1

---

### Description

`gf` is a command-line weather utility for PowerShell that provides detailed, real-time weather information for any specified location within the United States. It leverages the National Weather Service API to fetch comprehensive weather data, including current conditions, daily forecasts, and active weather alerts.

The script is designed for ease of use, accepting flexible location inputs like US zip codes, "City, State" strings, or the special "here" keyword for automatic location detection. It uses free geocoding services to determine coordinates and then fetches weather data from the official National Weather Service API. The output is color-coded to highlight important metrics, making it easy to assess conditions at a glance.

### Key Functionality

- **No API Key Required:** Uses the free National Weather Service API which requires no registration or API key.
- **Flexible Location Input:** Can determine latitude and longitude from either a 5-digit US zip code, a "City, State" formatted string, or the "here" keyword for automatic location detection.
- **Automatic Location Detection:** Uses ip-api.com to automatically detect the user's current location based on their IP address when "here" is specified.
- **Comprehensive Data Display:** Shows current temperature, conditions, wind chill and heat index calculations (using NWS formulas), detailed forecasts for today and tomorrow, wind information, sunrise and sunset times (calculated astronomically), moon phase information with emoji and next full moon date, rain likelihood forecasts with visual sparklines, and wind outlook forecasts with direction glyphs.
- **Weather Alerts:** Automatically fetches and displays any active weather alerts (e.g., warnings, watches) from official sources.
- **Color-Coded Metrics:** Key data points (temperature, wind speed) change color (blue for cold, red for hot) to indicate potentially hazardous conditions. Rain likelihood sparklines use color coding (white for very low, cyan for low, green for light, yellow for medium, red for high probability). Wind outlook glyphs use color coding (white for calm, yellow for light breeze, red for moderate wind, magenta for strong wind) with peak wind hours highlighted using inverted colors. **Humidity:** Uses meteorological comfort thresholds based on relative humidity percentage. Low humidity (<30%) can cause dry skin, static electricity, and respiratory discomfort (cyan). Comfortable range (30-60%) is ideal for human comfort (white). Elevated humidity (61-70%) begins to feel muggy and can affect perceived temperature (yellow). High humidity (>70%) is oppressive, significantly increases heat index, and can be dangerous in hot weather (red). **Dew Point:** More reliable than humidity for assessing comfort as it's independent of temperature. Dew point represents the temperature at which air becomes saturated and condensation forms. Values below 40°F indicate very dry air (cyan), 40-54°F is comfortable (white), 55-64°F feels sticky and muggy (yellow), and 65°F+ is oppressive and can be dangerous when combined with high temperatures (red). Dew points above 70°F are rare but extremely uncomfortable.
- **Multiple Display Modes:**
  - **Full Mode (default):** Shows all available weather information
  - **Terse Mode (`-t`):** Shows only current conditions and today's forecast (plus alerts)
  - **Hourly Mode (`-h`):** Shows only the 12-hour hourly forecast
  - **7-Day Mode (`-7`):** Shows only the 7-day forecast summary
  - **Enhanced Daily Mode (`-d`):** Shows comprehensive 7-day forecast with detailed wind information, windchill/heat index, and word-wrapped detailed forecasts
  - **Rain Forecast Mode (`-r` or `-rain`):** Shows rain likelihood forecast with visual sparklines for 96 hours
  - **Wind Forecast Mode (`-w` or `-wind`):** Shows wind outlook forecast with direction glyphs for 96 hours
  - **No-Interactive Mode (`-x`):** Exits immediately after displaying data (perfect for scripting)
- **Interactive Mode:** When run from non-terminal environments, provides keyboard shortcuts for dynamic view switching:
  - **[H]** - Switch to hourly forecast only
  - **[D]** - Switch to 7-day forecast only
  - **[T]** - Switch to terse mode
  - **[R]** - Switch to rain forecast mode (sparklines)
  - **[W]** - Switch to wind forecast mode (direction glyphs)
  - **[F]** - Return to full display
  - **[Enter]** or **[Esc]** - Exit the script
  - **Ctrl+C** will also exit the script
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

**Color Threshold Implementation:** The script implements meteorological comfort thresholds for humidity and dew point based on established meteorological standards. Humidity thresholds are based on relative humidity percentage impact on human comfort and health. Dew point thresholds are based on the temperature at which air becomes saturated, providing a more reliable comfort indicator than humidity alone. These thresholds are applied in the Show-CurrentConditions function using conditional logic that evaluates numeric values and assigns appropriate PowerShell color names (Cyan, White, Yellow, Red) for terminal display.

**Exponential Backoff Retry Logic:** The script implements robust error handling through exponential backoff retry logic in the Update-WeatherData function. This addresses temporary service unavailability (HTTP 503 errors) by implementing a retry mechanism with increasing delays between attempts. The algorithm uses a base delay of 1 second with exponential growth: 1s, 2s, 4s, 8s, 16s, 32s, 64s, 128s, 256s, 512s (capped at 512 seconds). The implementation includes screen clearing between retry attempts for a clean user experience, progress indication showing current attempt number, and graceful exit after maximum retry attempts (10) with a clear "Service Unavailable" message. This prevents the script from flooding the terminal with error messages during service outages while providing respectful retry behavior that doesn't overwhelm the NWS API servers.

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
- Moonrise/Moonset times
- Temperature trend indicators (rising/falling)
- Detailed weather overview reports
- Rain/Snow precipitation amounts

### Features Added/Enhanced

- **Sunrise/Sunset Times:** Calculated using NOAA astronomical algorithms based on location coordinates and time zone
- **Moon Phase Information:** Astronomical moon phase calculation with emoji display and next full moon date
- **Humidity Data:** Available in current conditions display

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

# Get weather using automatic location detection
.\gf.ps1 here

# Get terse output (current conditions + today's forecast only)
.\gf.ps1 -t "Seattle, WA"

# Get terse output using automatic location detection
.\gf.ps1 here -t

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
- **[R]** - **Rain View:** Switch to rain forecast mode with sparklines
- **[W]** - **Wind View:** Switch to wind forecast mode with direction glyphs
- **[G]** - **Get/Refresh:** Manually refresh weather data (auto-refreshes every 10 minutes)
- **[U]** - **Update Toggle:** Toggle automatic updates on/off
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
- **Interactive Mode:** Enters interactive mode after display (use -x to exit immediately)

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
- **Peak Wind Highlighting:** Hours with the highest wind speed for each day are displayed with inverted colors (black text on colored background)
- **Interactive Mode:** Enters interactive mode after display (use -x to exit immediately)

#### Wind Forecast Use Cases:

- **Outdoor Planning:** Identify calm periods for outdoor activities
- **Wind Assessment:** Understand wind patterns and direction changes
- **Activity Scheduling:** Plan wind-dependent activities (sailing, flying, etc.)
- **Quick Assessment:** Visual overview of wind conditions at a glance
- **Direction Analysis:** Track wind direction changes throughout the day
- **Peak Wind Identification:** Instantly spot the strongest wind hours for each day

#### Technical Implementation:

The wind forecast mode uses the same NWS hourly forecast data but processes it differently:
- Groups hourly data by day for display
- Maps wind direction to directional glyphs (N, NE, E, SE, S, SW, W, NW)
- Applies color coding based on wind speed thresholds
- Uses different glyph sets for light winds (<7mph) vs strong winds (≥7mph)
- Handles missing data gracefully with blank spaces
- Uses 96-hour data limit for comprehensive coverage

### Enhanced Daily Mode (v2.1)

The Enhanced Daily Mode (`-d`) provides a comprehensive 7-day forecast with detailed wind information, windchill/heat index calculations, and word-wrapped detailed forecasts. This mode is designed for users who need detailed weather information for comprehensive weekly planning.

#### Enhanced Daily Mode Features:

- **3-Line Display Per Day:** Each day shows summary line, day detailed forecast, and night detailed forecast
- **Wind Information:** Wind speed and direction with color coding (red for high wind ≥16mph)
- **Temperature Indices:** Windchill (≤50°F) and Heat Index (≥80°F) calculations when applicable
- **Precipitation Clarity:** Precipitation percentage with "Precip" label for clarity
- **Word Wrapping:** Detailed forecasts automatically wrap to terminal width
- **Smart Period Detection:** Automatically detects day vs night periods for single-period days
- **Consistent Styling:** All detailed forecast text uses configurable gray color
- **Interactive Mode:** Enters interactive mode after display (use -x to exit immediately)

#### Enhanced Daily Mode Display Format:

```
Sun: ☁ H:47°F 2 to 8mph SSW (36% Precip)
  Day: A chance of rain showers before 5am. Mostly cloudy, with a low around 47. 
       South southwest wind 2 to 6 mph. Chance of precipitation is 30%.
  Night: Mostly clear, with a low around 43. North wind around 3 mph.

Mon: ☀ H:62°F L:43°F 5mph ESE (10% Precip)
  Day: Mostly sunny. High near 62, with temperatures falling to around 60 in 
       the afternoon. East southeast wind around 5 mph.
  Night: Mostly clear, with a low around 43. North wind around 3 mph.
```

#### Enhanced Daily Mode Use Cases:

- **Comprehensive Planning:** Detailed weekly weather assessment for complex planning
- **Outdoor Activities:** Wind and temperature information for activity planning
- **Travel Planning:** Detailed forecasts for multi-day trips and events
- **Weather Analysis:** In-depth understanding of weather patterns and conditions
- **Professional Use:** Detailed weather information for work planning and scheduling

#### Technical Implementation:

The Enhanced Daily Mode uses advanced processing techniques:

- **Wind Speed Extraction:** Parses wind speed from NWS forecast data and applies color coding
- **Temperature Index Calculations:** Implements NWS formulas for windchill and heat index
- **Text Wrapping:** Uses `Format-TextWrap` function with terminal width detection
- **Period Detection:** Time-based logic to determine day vs night periods (6 PM to 6 AM = night)
- **Color Management:** Centralized `$detailedForecastColor` variable for consistent styling
- **Interactive Integration:** Full integration with interactive mode keyboard shortcuts

#### Text Wrapping Implementation:

The Enhanced Daily Mode implements sophisticated text wrapping to ensure detailed forecasts display properly across different terminal widths:

**Core Wrapping Function:**
- **Function:** `Format-TextWrap` (lines 644-683)
- **Algorithm:** Word-boundary splitting with intelligent line building
- **Width Calculation:** `$terminalWidth - $labelLength` to account for label space
- **Array Return:** Returns array of wrapped lines for multi-line display

**Wrapping Process:**
1. **Terminal Width Detection:** `$Host.UI.RawUI.WindowSize.Width`
2. **Label Length Calculation:** Accounts for "  Day: " (7 chars) and "  Night: " (9 chars)
3. **Text Processing:** Splits detailed forecast text on word boundaries
4. **Line Building:** Constructs lines that fit within calculated width
5. **Array Output:** Returns array of properly wrapped lines

**Indentation Strategy:**
- **First Line:** Uses label prefix ("  Day: " or "  Night: ")
- **Subsequent Lines:** Padded with spaces to align with text start
  - Day lines: 7-space indentation to align with "  Day: " text
  - Night lines: 9-space indentation to align with "  Night: " text
- **Visual Alignment:** Maintains consistent left margin for wrapped text

**Color Coding:**
- **Labels:** White color for "Day:" and "Night:" labels
- **Forecast Text:** Configurable gray color via `$detailedForecastColor` variable
- **Consistency:** All detailed forecast text uses same color scheme

**Terminal Width Handling:**
- **Dynamic Adaptation:** Automatically adjusts to current terminal width
- **Narrow Terminal Support:** Gracefully handles very narrow terminals
- **Word Boundary Respect:** Never breaks words mid-character
- **Overflow Prevention:** Ensures text never exceeds terminal boundaries

**Implementation Example:**
```powershell
# Get terminal width for text wrapping
$terminalWidth = $Host.UI.RawUI.WindowSize.Width

# Day detailed forecast with wrapping
$dayLabel = "  Day: "
$wrappedDayForecast = Format-TextWrap -Text $dayForecastText -Width ($terminalWidth - $dayLabel.Length)

Write-Host $dayLabel -ForegroundColor White -NoNewline
Write-Host $wrappedDayForecast[0] -ForegroundColor $detailedForecastColor
# Additional wrapped lines with proper indentation
for ($i = 1; $i -lt $wrappedDayForecast.Count; $i++) {
    Write-Host ("       " + $wrappedDayForecast[$i]) -ForegroundColor $detailedForecastColor
}
```

#### Color Coding System:

- **Wind Speed:** Red for high wind (≥16mph), default color for normal wind
- **Temperature:** Red for extreme temperatures (<33°F or >89°F)
- **Precipitation:** Red for high probability (>80%), yellow for medium (40-80%), default for low
- **Windchill:** Blue color when applicable (≤50°F and difference >1°F)
- **Heat Index:** Red color when applicable (≥80°F and difference >1°F)
- **Detailed Forecasts:** Configurable gray color (default: "Gray")

### Moon Phase Information (v2.1)

The moon phase feature provides astronomical moon phase information with visual emoji representation and next full moon date calculation. This feature enhances the weather display with lunar information that's useful for planning outdoor activities, understanding natural lighting conditions, and general astronomical awareness.

#### Moon Phase Features:

- **8 Moon Phases:** Complete lunar cycle representation with accurate phase names
  - **New Moon** (🌑): 0-12.5% of lunar cycle
  - **Waxing Crescent** (🌒): 12.5-25% of lunar cycle  
  - **First Quarter** (🌓): 25-37.5% of lunar cycle
  - **Waxing Gibbous** (🌔): 37.5-62.5% of lunar cycle
  - **Full Moon** (🌕): 62.5-75% of lunar cycle
  - **Waning Gibbous** (🌖): 75-87.5% of lunar cycle
  - **Last Quarter** (🌗): 87.5-93.75% of lunar cycle
  - **Waning Crescent** (🌘): 93.75-100% of lunar cycle

- **Astronomical Calculation:** Uses precise astronomical method with:
  - **Reference New Moon:** January 6, 2000 18:14 UTC (known astronomical event)
  - **Lunar Cycle:** 29.53058867 days (average synodic month)
  - **Phase Detection:** Simple percentage-based phase determination
  - **Next Full Moon:** Calculates next full moon date (occurs at ~14.77 days in cycle)

- **Visual Display:**
  - **Emoji Representation:** Each phase displays appropriate moon emoji
  - **Phase Name:** Full descriptive name of current moon phase
  - **Next Full Moon:** Shows "Next Full Moon: MM/DD/YYYY" only when not currently full moon
  - **Gray Color:** Displays in gray color for subtle integration
  - **Position:** Appears after sunset information in current conditions

#### Moon Phase Use Cases:

- **Outdoor Planning:** Understand natural lighting conditions for evening activities
- **Astronomical Awareness:** Track lunar cycle for astronomical observations
- **Activity Scheduling:** Plan moon-dependent activities (stargazing, photography, etc.)
- **Natural Lighting:** Assess available moonlight for outdoor activities
- **Lunar Calendar:** Track moon phases for personal or cultural reasons
- **Photography Planning:** Plan moon photography sessions and timing

#### Technical Implementation:

The moon phase calculation uses a simple but accurate astronomical method:

**Function:** `Get-MoonPhase`
**Location:** After `Get-SunriseSunset` function (around line 788)
**Algorithm:** Simple astronomical calculation with known reference point

**Mathematical Formula:**
```
Days Since Reference = (Current Date - Reference New Moon).TotalDays
Current Cycle Position = Days Since Reference % 29.53058867
Phase = Current Cycle Position / 29.53058867
```

**Phase Detection Logic:**
```powershell
if ($phase -lt 0.125) return "New Moon"
elseif ($phase -lt 0.25) return "Waxing Crescent"  
elseif ($phase -lt 0.375) return "First Quarter"
elseif ($phase -lt 0.625) return "Waxing Gibbous"
elseif ($phase -lt 0.75) return "Full Moon"
elseif ($phase -lt 0.875) return "Waning Gibbous"
elseif ($phase -lt 0.9375) return "Last Quarter"
else return "Waning Crescent"
```

**Next Full Moon Calculation:**
```powershell
$daysUntilNextFullMoon = (14.77 - $currentCycle) % $lunarCycle
if ($daysUntilNextFullMoon -le 0) {
    $daysUntilNextFullMoon += $lunarCycle
}
$nextFullMoonDate = $Date.AddDays($daysUntilNextFullMoon).ToString("MM/dd/yyyy")
```

**Display Integration:**
- **Function:** `Show-CurrentConditions` (line 1251-1257)
- **Parameters:** `MoonPhase`, `MoonEmoji`, `IsFullMoon`, `NextFullMoonDate`
- **Display Logic:** Shows moon phase info after sunset line in gray color
- **Conditional Display:** Only shows "Next Full Moon" when not currently full moon

**Data Flow:**
1. **Calculation:** `Get-MoonPhase` called in main script after sunrise/sunset calculation
2. **Storage:** Results stored in `$moonPhaseInfo` hashtable
3. **Passing:** Moon phase data passed to all `Show-CurrentConditions` calls
4. **Display:** Moon phase information rendered in current conditions section

**Error Handling:**
- **Graceful Degradation:** Missing moon phase data doesn't break display
- **Null Safety:** Handles missing or invalid date inputs
- **Fallback:** Returns empty values for invalid calculations

**Performance Considerations:**
- **Lightweight Calculation:** Simple arithmetic operations for fast execution
- **Single Calculation:** Moon phase calculated once per script run
- **Memory Efficient:** Minimal memory footprint for moon phase data
- **No External APIs:** No additional network requests required

#### Benefits:

- **Astronomical Accuracy:** Uses scientifically validated lunar cycle calculations
- **Visual Clarity:** Emoji representation makes moon phase instantly recognizable
- **Practical Utility:** Next full moon date helps with planning and scheduling
- **Seamless Integration:** Fits naturally into existing weather display
- **No Dependencies:** Self-contained calculation requiring no external services
- **Cultural Relevance:** Moon phases have cultural and practical significance worldwide

### Recent Enhancements (v2.1)

- **Moon Phase Information:** Added astronomical moon phase calculation with emoji display and next full moon date
- **Rain Forecast Mode:** Added visual sparkline representation of rain likelihood over 96 hours
- **Wind Forecast Mode:** Added visual directional glyph representation of wind patterns over 96 hours
- **Interactive Integration:** Both rain and wind modes now fully integrated into interactive mode
- **Enhanced Color Coding:** Implemented color-coded sparklines for rain probability and wind speed visualization
- **Peak Wind Highlighting:** Added visual feedback mechanism that inverts colors for peak wind hours
- **Extended Forecast Coverage:** Both rain and wind modes use 96-hour data instead of standard 12-hour limit
- **Improved Visual Design:** Better sparkline characters and directional glyphs that don't interfere with each other
- **Auto-Refresh Functionality:** Weather data automatically refreshes every 10 minutes in interactive mode
- **Auto-Update Toggle:** Added 'U' key to toggle automatic updates on/off during interactive mode
- **Command Line Control:** Added `-u` flag to start with automatic updates disabled
- **Manual Refresh Control:** Added 'G' key for manual data refresh while preserving current view mode
- **Dynamic Period Names:** Forecast sections now use actual NWS period names instead of hardcoded labels
- **NWS Resources Links:** Added clickable links to official NWS resources with custom display text
- **Control Bar Toggle:** Added command line flag and interactive key to hide/show control bar
- **Comprehensive Documentation:** Updated README and project documentation

### Auto-Refresh Technical Implementation

The auto-refresh functionality provides seamless data updates in interactive mode:

#### Key Components:
- **Timer Tracking:** Uses `$dataFetchTime` to track when data was last fetched
- **Staleness Detection:** Checks if current time exceeds 10-minute threshold (`$dataStaleThreshold = 600`)
- **Refresh Function:** `Update-WeatherData` re-fetches all API endpoints and updates global variables
- **View Preservation:** Maintains current display mode (hourly/daily/terse/rain/wind/full) during refresh
- **Error Handling:** Graceful fallback to existing data if refresh fails

#### Implementation Details:
- **Automatic Refresh:** Triggered in interactive loop before `ReadKey()` when data is stale
- **Manual Refresh:** 'G' key handler calls refresh function and re-renders current view
- **Data Consistency:** All weather variables updated atomically to prevent partial updates
- **User Feedback:** Visual indicators show refresh status and completion
- **Performance:** Reuses existing API job infrastructure for consistent behavior

#### Benefits:
- **Always Current:** Weather data never becomes outdated during long interactive sessions
- **Seamless Experience:** Users don't need to restart the script to get fresh data
- **Flexible Control:** Manual refresh available for immediate updates when needed
- **View Continuity:** Current display mode preserved across refreshes
- **User Control:** Toggle auto-updates on/off as needed with 'U' key
- **Battery Friendly:** Disable auto-updates to save battery on mobile devices
- **Network Conscious:** Turn off auto-updates when on limited data connections

### Auto-Update Toggle Feature (v2.1)

The auto-update toggle provides users with control over automatic data refreshing:

#### Key Features:
- **Toggle Control:** Press 'U' key to toggle automatic updates on/off
- **Visual Feedback:** Clear status messages show current auto-update state
- **Timer Reset:** When re-enabling updates, the refresh timer resets to current time
- **Command Line Option:** Use `-u` flag to start with auto-updates disabled
- **State Preservation:** Current display mode maintained when toggling updates

#### Implementation Details:
- **Status Display:** Green text for "Automatic Updates Enabled", Yellow for "Automatic Updates Disabled"
- **Timer Management:** `$dataFetchTime` reset when re-enabling to prevent immediate refresh
- **Conditional Logic:** Auto-refresh only occurs when `$autoUpdateEnabled` is true
- **User Feedback:** 800ms status message display with appropriate color coding
- **Mode Continuity:** Current view (hourly/daily/terse/rain/wind/full) preserved during toggle

#### Use Cases:
- **Battery Conservation:** Disable auto-updates on mobile devices to save battery
- **Data Usage Control:** Turn off updates when on limited data connections
- **Manual Control:** Users who prefer to manually refresh data only when needed
- **Network Issues:** Disable updates when experiencing network connectivity problems
- **Focused Sessions:** Turn off updates during focused work sessions to avoid interruptions

### NWS Resources Links Feature (v2.1)

The NWS Resources feature provides direct access to official National Weather Service resources through clickable links in the Location Information section.

#### Key Features:

- **Custom Link Text:** Displays "Forecast" and "Radar" instead of full URLs
- **ANSI Escape Sequences:** Uses modern terminal hyperlink support for clickable links
- **Dynamic Radar Station:** Automatically determines the appropriate radar station for the location
- **Fallback Support:** Includes fallback display for terminals that don't support ANSI sequences

#### Technical Implementation:

**Data Extraction:**
- **Radar Station ID:** Extracted from `$pointsData.properties.radarStation` in NWS points API response
- **Location Coordinates:** Uses existing latitude and longitude variables
- **Auto-Refresh Support:** Radar station preserved during data refresh operations

**Link Construction:**
- **Forecast Link:** `https://forecast.weather.gov/MapClick.php?lat={lat}&lon={lon}`
- **Graph Link:** `https://forecast.weather.gov/MapClick.php?lat={lat}&lon={lon}&unit=0&lg=english&FcstType=graphical`
- **Radar Link:** `https://radar.weather.gov/ridge/standard/{RADAR_STATION}_loop.gif`
- **Display Format:** "NWS Resources: Forecast | Graph | Radar"

**ANSI Escape Sequence Format:**
```powershell
# Forecast link
$forecastUrl = "https://forecast.weather.gov/MapClick.php?lat=$lat&lon=$lon"
Write-Host "`e]8;;$forecastUrl`e\Forecast`e]8;;`e\" -ForegroundColor Cyan -NoNewline

# Graph link
$graphUrl = "https://forecast.weather.gov/MapClick.php?lat=$lat&lon=$lon&unit=0&lg=english&FcstType=graphical"
Write-Host "`e]8;;$graphUrl`e\Graph`e]8;;`e\" -ForegroundColor Cyan -NoNewline

# Radar link  
$radarUrl = "https://radar.weather.gov/ridge/standard/${radarStation}_loop.gif"
Write-Host "`e]8;;$radarUrl`e\Radar`e]8;;`e\" -ForegroundColor Cyan
```

**Function Updates:**
- **Show-LocationInfo:** Added `[string]$RadarStation` parameter
- **Show-FullWeatherReport:** Added `[string]$RadarStation` parameter
- **Update-WeatherData:** Added radar station extraction and preservation

**Terminal Compatibility:**
- **Supported:** Windows Terminal, PowerShell 7+, VS Code integrated terminal, most modern terminals
- **Fallback:** Full URLs displayed for terminals without ANSI support
- **Visual Feedback:** Links appear underlined and clickable in supported terminals

#### Benefits:

- **Direct Access:** One-click access to official NWS forecast and radar pages
- **Location-Specific:** Radar link automatically uses the correct local radar station
- **Clean Interface:** Custom text instead of long URLs improves readability
- **Modern UX:** Clickable links provide modern terminal experience
- **Official Sources:** Links directly to authoritative NWS resources

#### Use Cases:

- **Quick Forecast Check:** Click "Forecast" to view detailed NWS forecast page
- **Graphical Analysis:** Click "Graph" to view detailed graphical forecast with charts and graphs
- **Radar Analysis:** Click "Radar" to view current radar imagery for the area
- **Weather Planning:** Access official NWS resources for detailed weather analysis
- **Professional Use:** Direct access to authoritative weather information sources

### Control Bar Toggle Feature (v2.1)

The Control Bar Toggle feature provides users with the ability to hide the interactive control bar for a cleaner display experience.

#### Key Features:

- **Command Line Control:** Start with control bar hidden using `-b` or `-NoBar` flag
- **Interactive Toggle:** Use 'B' key during interactive mode to toggle control bar on/off
- **State Persistence:** Control bar state is maintained across mode changes and data refreshes
- **Status Feedback:** Visual confirmation when toggling control bar state

#### Technical Implementation:

**Command Line Parameter:**
```powershell
[Alias('b')]
[switch]$NoBar
```

**State Management:**
```powershell
# Initialize control bar visibility (can be toggled with 'b' key)
$script:showControlBar = -not $NoBar.IsPresent
```

**Control Function Modification:**
```powershell
function Show-InteractiveControls {
    # Don't show controls if bar is hidden
    if (-not $script:showControlBar) {
        return
    }
    # ... existing control bar display code ...
}
```

**Interactive Toggle Handler:**
```powershell
'b' { # B key - Toggle control bar
    $script:showControlBar = -not $script:showControlBar
    # Show status message and re-render current view
}
```

#### Benefits:

- **Cleaner Display:** Hide control bar for distraction-free weather viewing
- **Flexible Usage:** Toggle on/off as needed during interactive sessions
- **State Persistence:** Control bar setting maintained across all mode changes
- **User Control:** Both command line and interactive control options

#### Use Cases:

- **Clean Screenshots:** Hide control bar for cleaner weather display captures
- **Distraction-Free Viewing:** Focus on weather data without interface clutter
- **Presentation Mode:** Hide controls when displaying weather information to others
- **Custom Workflows:** Start with hidden bar and toggle as needed

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

## Wind Chill and Heat Index Calculations

The script includes sophisticated wind chill and heat index calculations using official NWS formulas to provide accurate "feels like" temperature information.

### Wind Chill Calculation

**Function:** `Get-WindChill`
**Formula:** NWS Wind Chill Formula
**Conditions:** Only calculated when temperature ≤ 50°F and wind speed ≥ 3 mph
**Display:** Blue color when difference > 1°F from actual temperature

**Mathematical Formula:**
```
Wind Chill = 35.74 + (0.6215 × T) - (35.75 × V^0.16) + (0.4275 × T × V^0.16)
```
Where:
- T = Temperature in Fahrenheit
- V = Wind speed in mph

**Implementation Details:**
- Uses `[Math]::Pow()` for power calculations
- Rounds result to nearest integer
- Returns `$null` if conditions not met
- Displayed as `[value°F]` in blue when significant difference

### Heat Index Calculation

**Function:** `Get-HeatIndex`
**Formula:** NWS Rothfusz Regression
**Conditions:** Only calculated when temperature ≥ 80°F
**Display:** Red color when difference > 1°F from actual temperature

**Mathematical Formula:**
```
HI = -42.379 + (2.04901523 × T) + (10.14333127 × RH) - (0.22475541 × T × RH) 
     - (0.00683783 × T²) - (0.05481717 × RH²) + (0.00122874 × T² × RH) 
     + (0.00085282 × T × RH²) - (0.00000199 × T² × RH²)
```
Where:
- T = Temperature in Fahrenheit
- RH = Relative Humidity percentage

**Adjustment Factors:**
- **Low Humidity (RH < 13%):** Additional adjustment for dry conditions
- **High Humidity (RH > 85%):** Additional adjustment for very humid conditions

**Implementation Details:**
- Uses full Rothfusz regression for temperatures ≥ 80°F
- Includes humidity adjustment factors for extreme conditions
- Rounds result to nearest integer
- Returns `$null` if conditions not met
- Displayed as `[value°F]` in red when significant difference

### Display Integration

**Temperature Display Format:**
- **Wind Chill:** `Temperature: 45°F [37°F] ↗️` (blue wind chill)
- **Heat Index:** `Temperature: 85°F [92°F] ↘️` (red heat index)
- **Normal:** `Temperature: 65°F →` (no additional values)

**Color Coding:**
- **Wind Chill:** Blue (`-ForegroundColor Blue`)
- **Heat Index:** Red (`-ForegroundColor Red`)
- **Integration:** Seamlessly works with existing temperature trend icons

**Thresholds:**
- **Wind Chill:** Only displays when temp ≤ 50°F and difference > 1°F
- **Heat Index:** Only displays when temp ≥ 80°F and difference > 1°F
- **Significance:** Only shows when the calculated value differs significantly from actual temperature

### Technical Implementation

**Function Location:** After `Get-WindSpeed` function (around line 868)
**Integration Point:** `Show-CurrentConditions` function temperature display section
**Data Sources:** 
- Temperature: `$currentTemp` (from NWS API)
- Wind Speed: `$currentWind` (parsed by `Get-WindSpeed`)
- Humidity: `$currentHumidity` (from NWS API)

**Error Handling:**
- Graceful handling of missing or invalid data
- Returns `$null` for invalid conditions
- No display when calculation not applicable

This implementation provides users with accurate "feels like" temperature information using official NWS standards, enhancing the weather display with scientifically validated comfort metrics.
