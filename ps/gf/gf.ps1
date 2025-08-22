<#
.SYNOPSIS
    A PowerShell script to retrieve and display detailed weather information for a specified location.
    
.DESCRIPTION
    This script accepts a US zip code or a "City, State" string as input. It uses the
    National Weather Service API to retrieve and display detailed weather information.

    The script first uses a geocoding service to determine the latitude and longitude of the location,
    then fetches the current weather, daily forecasts, and weather alerts.

    No API key is required as the National Weather Service API is free and open.
    
.PARAMETER Location
    The location for which to retrieve weather. Can be a 5-digit US zip code or a "City, State" string.
    If omitted, the script will prompt you for it.
    
.PARAMETER Help
    Displays usage information for this script.
    
.EXAMPLE
    .\gf.ps1 97219 -Verbose

.EXAMPLE
    .\gf.ps1 "Portland, OR" -Verbose

.NOTES
    This script uses the free National Weather Service API which requires no API key.
    The API is limited to locations within the United States.

    To execute PS Scripts run the following from an admin prompt "Set-ExecutionPolicy bypass"
    Execute with ./<scriptname>, or simply <scriptname> if placed in your %PATH%
#>

param(
    [Parameter(Mandatory = $false, Position = 0)]
    [string]$Location,

    [Parameter(Mandatory = $false)]
    [switch]$Help,

    [Alias('t')]
    [switch]$Terse,

    [Alias('h')]
    [switch]$Hourly,

    [Alias('7')]
    [switch]$SevenDay,

    [Alias('d')]
    [switch]$Daily,

    [Alias('x')]
    [switch]$NoInteractive
)

# --- Helper Functions ---

if ($Help -or (($Terse.IsPresent -or $Hourly.IsPresent -or $SevenDay.IsPresent -or $Daily.IsPresent -or $NoInteractive.IsPresent) -and -not $Location)) {
    Write-Host "Usage: .\gf.ps1 [ZipCode | `"City, State`"] [Options] [-Verbose]" -ForegroundColor Green
    Write-Host " • Provide a 5-digit zipcode or a City, State (e.g., 'Portland, OR')." -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Options:" -ForegroundColor Blue
    Write-Host "  -t, -Terse    Show only current conditions and today's forecast" -ForegroundColor Cyan
    Write-Host "  -h, -Hourly   Show only the 12-hour hourly forecast" -ForegroundColor Cyan
    Write-Host "  -7, -SevenDay Show only the 7-day forecast summary" -ForegroundColor Cyan
    Write-Host "  -d, -Daily    Same as -7 (7-day forecast summary)" -ForegroundColor Cyan
    Write-Host "  -x, -NoInteractive Exit immediately (no interactive mode)" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Interactive Mode:" -ForegroundColor Blue
    Write-Host "  When run interactively (not from terminal), the script enters interactive mode." -ForegroundColor Cyan
    Write-Host "  Use keyboard shortcuts to switch between display modes:" -ForegroundColor Cyan
    Write-Host "    [H] - Switch to hourly forecast only" -ForegroundColor Cyan
    Write-Host "    [D] - Switch to 7-day forecast only" -ForegroundColor Cyan
    Write-Host "    [T] - Switch to terse mode (current + today)" -ForegroundColor Cyan
    Write-Host "    [F] - Return to full display" -ForegroundColor Cyan
    Write-Host "    [Enter] - Exit the script" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "This script retrieves weather info from National Weather Service API and outputs:" -ForegroundColor Blue
    Write-Host " • Location (City, State)" -ForegroundColor Cyan
    Write-Host " • Current Conditions" -ForegroundColor Cyan
    Write-Host " • Temperature with forecast range (red if <33°F or >89°F)" -ForegroundColor Cyan
    Write-Host " • Humidity" -ForegroundColor Cyan
    Write-Host " • Wind (with gust if available; red if wind speed >=16 mph)" -ForegroundColor Cyan
    Write-Host " • Detailed Forecast" -ForegroundColor Cyan
    Write-Host " • Weather Alerts" -ForegroundColor Cyan
    Write-Host " • Observation timestamp" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Blue
    Write-Host "  .\gf.ps1 97219 -Verbose" -ForegroundColor Cyan
    Write-Host "  .\gf.ps1 `"Portland, OR`" -Verbose" -ForegroundColor Cyan
    Write-Host "  .\gf.ps1 97219 -t" -ForegroundColor Cyan
    Write-Host "  .\gf.ps1 `"Portland, OR`" -h" -ForegroundColor Cyan
         Write-Host "  .\gf.ps1 97219 -7" -ForegroundColor Cyan
     Write-Host "  .\gf.ps1 97219 -h -x" -ForegroundColor Cyan
     Write-Host "  .\gf.ps1 -Help" -ForegroundColor Cyan
    return
}

# Force TLS 1.2.
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

# Define User-Agent for NWS API requests
$userAgent = "GetForecast/1.0 (081625PDX)"

# Function to get the parent process name using CIM
function Get-ParentProcessName {
    $proc = Get-CimInstance Win32_Process -Filter "ProcessId=$PID"
    if ($proc) {
        $parentProc = Get-CimInstance Win32_Process -Filter "ProcessId=$($proc.ParentProcessId)"
        return $parentProc.Name
    }
    return $null
}

# Define common header for NWS API.
$headers = @{
    "Accept"     = "application/geo+json"
    "User-Agent" = $userAgent
}

# Determine the parent's process name.
$parentName = Get-ParentProcessName
Write-Verbose "Proc:$parentName"

$isInteractive = (-not $Location)
$lat = $null
$lon = $null
$city = $null
$state = $null

while ($true) { # Loop for location input and geocoding
    try {
        if (-not $Location) {
            if ($VerbosePreference -ne 'Continue') { Clear-Host }
            Write-Host '    \|/     ' -ForegroundColor Yellow -NoNewline; Write-Host "    .-~~~~~~-.      " -ForegroundColor Cyan
            Write-Host '  -- O --   ' -ForegroundColor Yellow -NoNewline; Write-Host "   /_)      ( \     " -ForegroundColor Cyan
            Write-Host '    /|\     ' -ForegroundColor Yellow -NoNewline; Write-Host "  (   ( )    ( )    " -ForegroundColor Cyan
            Write-Host '            ' -ForegroundColor Yellow -NoNewline; Write-Host "   `-~~~~~~~~~-`    " -ForegroundColor Cyan
            Write-Host '  Welcome   ' -ForegroundColor Green  -NoNewline; Write-Host "     ''    ''       " -ForegroundColor Cyan
            Write-Host '     to     ' -ForegroundColor Green  -NoNewline; Write-Host "    ''    ''        " -ForegroundColor Cyan
            Write-Host ' GetForecast ' -ForegroundColor Green  -NoNewline; Write-Host "  ________________  " -ForegroundColor Cyan
            Write-Host '            ' -ForegroundColor Yellow -NoNewline; Write-Host "~~~~~~~~~~~~~~~~~~~~" -ForegroundColor Cyan
            Write-Host ""
            $Location = Read-Host "Enter a location (Zip Code or City, State)"
            if ([string]::IsNullOrEmpty($Location)) { exit } # Exit if user enters nothing
        }

        Write-Verbose "Input provided: $Location"

        # --- GEOCODING ---
        if ($Location -match "^\d{5}(-\d{4})?$") {
            Write-Verbose "Input identified as a zipcode."
            # Use a free geocoding service for zip codes
            $geoUrl = "https://api.zippopotam.us/us/$Location"
            Write-Verbose "Geocoding URL (zip): $geoUrl"
            $geoData = Invoke-RestMethod "$geoUrl" -ErrorAction Stop
            if (-not $geoData) { throw "No geocoding results found for zipcode '$Location'." }
            $lat = [double]$geoData.places[0].latitude
            $lon = [double]$geoData.places[0].longitude
            $city = $geoData.places[0].'place name'
            $state = $geoData.places[0].'state abbreviation'
        }
        else {
            Write-Verbose "Input assumed to be a City, State."
            # Use a free geocoding service for city/state
            $locationForApi = if ($Location -match ",") { $Location } else { "$Location,US" }
            $encodedLocation = [uri]::EscapeDataString($locationForApi)
            $geoUrl = "https://nominatim.openstreetmap.org/search?q=$encodedLocation&format=json&limit=1&countrycodes=us"
            Write-Verbose "Geocoding URL (direct): $geoUrl"
            $geoData = Invoke-RestMethod "$geoUrl" -ErrorAction Stop
            if ($geoData.Count -eq 0) { throw "No geocoding results found for '$Location'." }
            $lat = [double]$geoData[0].lat
            $lon = [double]$geoData[0].lon
            $city = $geoData[0].name
            # Try to get state from address, fallback to display_name parsing
            if ($geoData[0].address.state) {
                $state = $geoData[0].address.state
            } else {
                # Parse state from display_name if available
                $displayName = $geoData[0].display_name
                if ($displayName -match ", ([A-Z]{2}),") {
                    $state = $matches[1]
                }
            }
        }
        break # Exit loop on successful geocoding
    }
    catch {
        Write-Host "Location not found, try again" -ForegroundColor Red
        if (-not $isInteractive) {
            exit 1
        }
        $Location = $null # Clear location to re-prompt
        Start-Sleep -Seconds 1
    }
}

Write-Verbose "Geocoding result: City: $city, State: $state, Lat: $lat, Lon: $lon"

# --- FETCH NWS POINTS DATA ---
Write-Verbose "Starting API call for NWS points data."

$pointsUrl = "https://api.weather.gov/points/$lat,$lon"

$pointsJob = Start-Job -ScriptBlock {
    param($url, $hdrs)
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    try {
        $response = Invoke-RestMethod -Uri $url -Method Get -Headers $hdrs -ErrorAction Stop
        return $response | ConvertTo-Json -Depth 10
    }
    catch {
        throw "Failed to retrieve points data from job: $($_.Exception.Message)"
    }
} -ArgumentList $pointsUrl, $headers

Wait-Job -Job $pointsJob | Out-Null

if ($pointsJob.State -ne 'Completed') { 
    Write-Error "The points data job failed: $($pointsJob | Receive-Job)"; 
    Remove-Job -Job $pointsJob; 
    exit 1 
}

$pointsJson = $pointsJob | Receive-Job
if ([string]::IsNullOrWhiteSpace($pointsJson)) { 
    Write-Error "Empty response from points API job."; 
    Remove-Job -Job $pointsJob; 
    exit 1 
}

$pointsData = $pointsJson | ConvertFrom-Json
Write-Verbose "Raw points response:`n$pointsJson"

Remove-Job -Job $pointsJob

# Extract grid information
$office = $pointsData.properties.cwa
$gridX = $pointsData.properties.gridX
$gridY = $pointsData.properties.gridY

# Extract additional location information
$timeZone = $pointsData.properties.timeZone
$radarStation = $pointsData.properties.radarStation

# Try to get county information from different possible locations
$county = $null
if ($pointsData.properties.relativeLocation.properties.county) {
    $county = $pointsData.properties.relativeLocation.properties.county
    # Clean up county name if it's a URL
    if ($county -match "county/([^/]+)$") {
        $county = $matches[1]
    }
} elseif ($pointsData.properties.county) {
    $county = $pointsData.properties.county
    # Clean up county name if it's a URL
    if ($county -match "county/([^/]+)$") {
        $county = $matches[1]
    }
}

Write-Verbose "Grid info: Office=$office, GridX=$gridX, GridY=$gridY"
Write-Verbose "Location info: County=$county, TimeZone=$timeZone, Radar=$radarStation"

# --- CONCURRENTLY FETCH FORECAST AND HOURLY DATA ---
Write-Verbose "Starting API calls for forecast data."

$forecastUrl = "https://api.weather.gov/gridpoints/$office/$gridX,$gridY/forecast"
$hourlyUrl = "https://api.weather.gov/gridpoints/$office/$gridX,$gridY/forecast/hourly"

$forecastJob = Start-Job -ScriptBlock {
    param($url, $hdrs)
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    try {
        $response = Invoke-RestMethod -Uri $url -Method Get -Headers $hdrs -ErrorAction Stop
        return $response | ConvertTo-Json -Depth 10
    }
    catch {
        throw "Failed to retrieve forecast data from job: $($_.Exception.Message)"
    }
} -ArgumentList $forecastUrl, $headers

$hourlyJob = Start-Job -ScriptBlock {
    param($url, $hdrs)
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    try {
        $response = Invoke-RestMethod -Uri $url -Method Get -Headers $hdrs -ErrorAction Stop
        return $response | ConvertTo-Json -Depth 10
    }
    catch {
        throw "Failed to retrieve hourly data from job: $($_.Exception.Message)"
    }
} -ArgumentList $hourlyUrl, $headers

$jobsToWaitFor = @($forecastJob, $hourlyJob)
Wait-Job -Job $jobsToWaitFor | Out-Null

# --- COLLECT RESULTS AND HANDLE ERRORS ---
if ($forecastJob.State -ne 'Completed') { 
    Write-Error "The forecast data job failed: $($forecastJob | Receive-Job)"; 
    Remove-Job -Job $jobsToWaitFor; 
    exit 1 
}

$forecastJson = $forecastJob | Receive-Job
if ([string]::IsNullOrWhiteSpace($forecastJson)) { 
    Write-Error "Empty response from forecast API job."; 
    Remove-Job -Job $jobsToWaitFor; 
    exit 1 
}

$forecastData = $forecastJson | ConvertFrom-Json
Write-Verbose "Raw forecast response:`n$forecastJson"

if ($hourlyJob.State -ne 'Completed') { 
    Write-Error "The hourly data job failed: $($hourlyJob | Receive-Job)"; 
    Remove-Job -Job $jobsToWaitFor; 
    exit 1 
}

$hourlyJson = $hourlyJob | Receive-Job
if ([string]::IsNullOrWhiteSpace($hourlyJson)) { 
    Write-Error "Empty response from hourly API job."; 
    Remove-Job -Job $jobsToWaitFor; 
    exit 1 
}

$hourlyData = $hourlyJson | ConvertFrom-Json
Write-Verbose "Raw hourly response:`n$hourlyJson"

Remove-Job -Job $jobsToWaitFor

# --- FETCH ALERTS ---
$alertsData = $null
try {
    $alertsUrl = "https://api.weather.gov/alerts/active?point=$lat,$lon"
    Write-Verbose "Fetching alerts from: $alertsUrl"
    $alertsResponse = Invoke-RestMethod -Uri $alertsUrl -Method Get -Headers $headers -ErrorAction Stop
    $alertsData = $alertsResponse
    Write-Verbose "Alerts found: $($alertsData.features.Count)"
}
catch {
    Write-Verbose "No alerts found or error fetching alerts: $($_.Exception.Message)"
}

function Format-TextWrap {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)]
        [string]$Text,

        [Parameter(Mandatory)]
        [ValidateRange(1,1000)]
        [int]$Width
    )

    # Split into words (preserving non-empty tokens)
    $words = $Text -split '\s+'

    $lines = @()
    $currentLine = ''

    foreach ($word in $words) {
        if ($currentLine.Length -eq 0) {
            # start a new line
            $currentLine = $word
        }
        elseif ($currentLine.Length + 1 + $word.Length -le $Width) {
            # fits on current line
            $currentLine += ' ' + $word
        }
        else {
            # flush current line, start a new one
            $lines += $currentLine
            $currentLine = $word
        }
    }

    # add the last line
    if ($currentLine) {
        $lines += $currentLine
    }

    return $lines
}

# Extract current conditions from hourly data
$currentPeriod = $hourlyData.properties.periods[0]
$currentTemp = $currentPeriod.temperature
$currentConditions = $currentPeriod.shortForecast
$currentWind = $currentPeriod.windSpeed
$currentWindDir = $currentPeriod.windDirection
$currentTime = $currentPeriod.startTime
$currentHumidity = $currentPeriod.relativeHumidity.value
$currentDewPoint = $currentPeriod.dewpoint.value
$currentPrecipProb = $currentPeriod.probabilityOfPrecipitation.value
$currentTempTrend = $currentPeriod.temperatureTrend
$currentIcon = $currentPeriod.icon

# Check for wind gusts in the wind speed string
$windGust = $null
if ($currentWind -match "(\d+)\s*to\s*(\d+)\s*mph") {
    $windGust = $matches[2]
    $currentWind = "$($matches[1]) mph"
}

# Extract today's forecast
$todayPeriod = $forecastData.properties.periods[0]
$todayForecast = $todayPeriod.detailedForecast

# Extract tomorrow's forecast
$tomorrowPeriod = $forecastData.properties.periods[1]
$tomorrowForecast = $tomorrowPeriod.detailedForecast

# Function: Convert wind degrees to cardinal direction.
function Get-CardinalDirection ($deg) {
    $val = [math]::Floor(($deg / 22.5) + 0.5)
    $directions = @("N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW")
    return $directions[($val % 16)]
}

# Get wind speed numeric value from string
function Get-WindSpeed ($windString) {
    if ($windString -match "(\d+)") {
        return [int]$matches[1]
    }
    return 0
}

# Function to determine if it's currently day or night based on NWS API isDaytime property
function Test-IsDaytime {
    param(
        [object]$Period
    )
    
    # Use the NWS API's isDaytime property if available
    if ($Period.PSObject.Properties['isDaytime']) {
        return $Period.isDaytime
    }
    
    # Fallback: use current local time to estimate day/night
    $currentHour = (Get-Date).Hour
    return $currentHour -ge 6 -and $currentHour -lt 18
}

# Convert weather icon to emoji (no padding - we'll handle alignment differently)
function Get-WeatherIcon ($iconUrl, $isDaytime = $true) {
    if (-not $iconUrl) { return "" }
    
    # Extract weather condition from icon URL
    if ($iconUrl -match "/([^/]+)\?") {
        $condition = $matches[1]
        
        # Map conditions to emoji with day/night variants
        $emoji = switch -Wildcard ($condition) {
            "*skc*" { if ($isDaytime) { "☀️" } else { "🌙" } }  # Clear - sun during day, moon at night
            "*few*" { if ($isDaytime) { "🌤️" } else { "🌙" } }  # Few clouds - sun with clouds during day, moon at night
            "*sct*" { if ($isDaytime) { "⛅" } else { "☁️" } }   # Scattered clouds - sun with clouds during day, just clouds at night
            "*bkn*" { "☁️" }  # Broken clouds - same for day/night
            "*ovc*" { "☁️" }  # Overcast - same for day/night
            "*rain*" { "🌧️" } # Rain - same for day/night
            "*snow*" { "❄️" }  # Snow - same for day/night
            "*fzra*" { "🧊" }  # Freezing rain - same for day/night
            "*tsra*" { "⛈️" }  # Thunderstorm - same for day/night
            "*fog*" { "🌫️" }   # Fog - same for day/night
            "*haze*" { "🌫️" }  # Haze - same for day/night
            "*smoke*" { "💨" } # Smoke - same for day/night
            "*dust*" { "💨" }  # Dust - same for day/night
            "*wind*" { "💨" }  # Windy - same for day/night
            default { if ($isDaytime) { "🌡️" } else { "🌙" } }   # Default - thermometer during day, moon at night
        }
        
        return $emoji
    }
    
    # Default fallback
    return if ($isDaytime) { "🌡️" } else { "🌙" }
}

# Function to detect if we're in Cursor/VS Code terminal
function Test-CursorTerminal {
    return $parentName -match 'Code' -or 
           $parentName -match 'Cursor' -or 
           $env:TERM_PROGRAM -eq 'vscode' -or
           $env:VSCODE_PID -or
           $env:CURSOR_PID
}

# Function to get emoji width based on terminal environment
function Get-EmojiWidth {
    param([string]$Emoji)
    
    # In Cursor's terminal, emojis render as single-width
    if (Test-CursorTerminal) {
        return 1
    }
    
    # In regular PowerShell, treat all emojis as double-width for consistency
    return 2
}

# Function to calculate the display width of a string (accounting for emoji width)
function Get-StringDisplayWidth {
    param([string]$Text)
    
    $width = 0
    $i = 0
    while ($i -lt $Text.Length) {
        $char = $Text[$i]
        $codePoint = [int][char]$char
        
        # Check if this is a surrogate pair (emoji)
        if ($codePoint -ge 0xD800 -and $codePoint -le 0xDBFF -and $i + 1 -lt $Text.Length) {
            $nextChar = $Text[$i + 1]
            $nextCodePoint = [int][char]$nextChar
            if ($nextCodePoint -ge 0xDC00 -and $nextCodePoint -le 0xDFFF) {
                # This is a surrogate pair - get the emoji
                $emoji = $char + $nextChar
                $width += Get-EmojiWidth $emoji
                $i += 2
                continue
            }
        }
        
        # Regular character - check if it's an emoji
        if ($codePoint -ge 0x1F600 -and $codePoint -le 0x1F64F) { # Emoji
            $width += Get-EmojiWidth $char
        } elseif ($codePoint -ge 0x1F300 -and $codePoint -le 0x1F5FF) { # Misc Symbols and Pictographs
            $width += Get-EmojiWidth $char
        } elseif ($codePoint -ge 0x1F680 -and $codePoint -le 0x1F6FF) { # Transport and Map Symbols
            $width += Get-EmojiWidth $char
        } elseif ($codePoint -ge 0x1F1E0 -and $codePoint -le 0x1F1FF) { # Regional Indicator Symbols
            $width += Get-EmojiWidth $char
        } elseif ($codePoint -ge 0x2600 -and $codePoint -le 0x26FF) { # Misc Symbols
            $width += Get-EmojiWidth $char
        } elseif ($codePoint -ge 0x2700 -and $codePoint -le 0x27BF) { # Dingbats
            $width += Get-EmojiWidth $char
        } else {
            $width += 1
        }
        $i++
    }
    return $width
}

# Function to build a formatted hourly forecast line with proper alignment
function Format-HourlyLine {
    param(
        [string]$Time,
        [string]$Icon,
        [string]$Temp,
        [string]$Wind,
        [string]$WindDir,
        [int]$PrecipProb,
        [string]$Forecast
    )
    
    # Build the line components
    $timePart = "$Time "
    $iconPart = "$Icon"
    $tempPart = " $Temp°F "
    $windPart = "$Wind $WindDir"
    $precipPart = if ($PrecipProb -gt 0) { " ($PrecipProb%)" } else { "" }
    $forecastPart = " - $Forecast"
    
    if (Test-CursorTerminal) {
        # In Cursor, use a simple fixed-width approach
        # Time is 5 chars + space = 6 chars, then add fixed spaces after icon
        $padding = "  "  # Fixed 2 spaces after icon
        $completeLine = $timePart + $iconPart + $padding + $tempPart + $windPart + $precipPart + $forecastPart
    } else {
        # In regular terminals, use the width calculation approach
        $targetTempColumn = 8
        $lineBeforeTemp = $timePart + $iconPart
        $currentDisplayWidth = Get-StringDisplayWidth $lineBeforeTemp
        $spacesNeeded = $targetTempColumn - $currentDisplayWidth
        $padding = " " * [Math]::Max(0, $spacesNeeded)
        $completeLine = $lineBeforeTemp + $padding + $tempPart + $windPart + $precipPart + $forecastPart
    }
    
    return $completeLine
}

# Function to build a formatted daily forecast line with proper alignment
function Format-DailyLine {
    param(
        [string]$DayName,
        [string]$Icon,
        [string]$Temp,
        [string]$NightTemp,
        [string]$Forecast,
        [int]$PrecipProb
    )
    
    # Build the line components
    $dayPart = "$DayName`: "
    $iconPart = "$Icon"
    $tempPart = if ($NightTemp) { " H:$Temp°F L:$NightTemp°F" } else { " $Temp°F" }
    $forecastPart = " - $Forecast"
    $precipPart = if ($PrecipProb -gt 0) { " ($PrecipProb% precip)" } else { "" }
    
    if (Test-CursorTerminal) {
        # In Cursor, use a simple fixed-width approach
        # Each day name is 4 chars + ": " = 6 chars, then add fixed spaces after icon
        $padding = "  "  # Fixed 2 spaces after icon
        $completeLine = $dayPart + $iconPart + $padding + $tempPart + $forecastPart + $precipPart
    } else {
        # In regular terminals, use the width calculation approach
        $targetTempColumn = 6
        $lineBeforeTemp = $dayPart + $iconPart
        $currentDisplayWidth = Get-StringDisplayWidth $lineBeforeTemp
        $spacesNeeded = $targetTempColumn - $currentDisplayWidth
        $padding = " " * [Math]::Max(0, $spacesNeeded)
        $completeLine = $lineBeforeTemp + $padding + $tempPart + $forecastPart + $precipPart
    }
    
    return $completeLine
}

$windSpeed = Get-WindSpeed $currentWind

# Convert time strings to local time
$currentTimeLocal = ([DateTime]::Parse($currentTime)).ToLocalTime()

# Define colors.
$defaultColor = "DarkCyan"
$alertColor = "Red"
$titleColor = "Green"
$infoColor = "Blue"

if ([int]$currentTemp -lt 33 -or [int]$currentTemp -gt 89) {
    $tempColor = $alertColor
} else {
    $tempColor = $defaultColor
}

if ($windSpeed -ge 16) {
    $windColor = $alertColor
} else {
    $windColor = $defaultColor
}

if ($VerbosePreference -ne 'Continue') {
    Clear-Host
}

# Determine display mode
$showCurrentConditions = $true
$showTodayForecast = $true
$showTomorrowForecast = $true
$showHourlyForecast = $true
$showSevenDayForecast = $true
$showAlerts = $true
$showLocationInfo = $true

if ($Terse.IsPresent) {
    # Terse mode: only current conditions and today's forecast
    $showTomorrowForecast = $false
    $showHourlyForecast = $false
    $showSevenDayForecast = $false
    $showLocationInfo = $false
}
elseif ($Hourly.IsPresent) {
    # Hourly mode: only hourly forecast
    $showCurrentConditions = $false
    $showTodayForecast = $false
    $showTomorrowForecast = $false
    $showSevenDayForecast = $false
    $showAlerts = $false
    $showLocationInfo = $false
}
elseif ($SevenDay.IsPresent -or $Daily.IsPresent) {
    # 7-day mode: only 7-day forecast
    $showCurrentConditions = $false
    $showTodayForecast = $false
    $showTomorrowForecast = $false
    $showHourlyForecast = $false
    $showAlerts = $false
    $showLocationInfo = $false
}

# Determine if it's currently day or night using NWS API isDaytime property
$isCurrentlyDaytime = Test-IsDaytime $currentPeriod

# Output the results.
$weatherIcon = Get-WeatherIcon $currentIcon $isCurrentlyDaytime

if ($showCurrentConditions) {
    Write-Host "*** $city, $state Current Conditions ***" -ForegroundColor $titleColor
    Write-Host "Currently: $weatherIcon $currentConditions" -ForegroundColor $defaultColor
    Write-Host "Temperature: $currentTemp°F" -ForegroundColor $tempColor -NoNewline
    if ($currentTempTrend) {
        $trendIcon = switch ($currentTempTrend) {
            "rising" { "↗️" }
            "falling" { "↘️" }
            "steady" { "→" }
            default { "" }
        }
        Write-Host " $trendIcon " -ForegroundColor $defaultColor -NoNewline
    }
    Write-Host ""

    Write-Host "Wind: $currentWind $currentWindDir" -ForegroundColor $windColor -NoNewline
    if ($windGust) {
        Write-Host " (gusts to $windGust mph)" -ForegroundColor $alertColor -NoNewline
    }
    Write-Host ""

    Write-Host "Humidity: $currentHumidity%" -ForegroundColor $defaultColor
    Write-Host "Dew Point: $([math]::Round($currentDewPoint * 9/5 + 32, 1))°F" -ForegroundColor $defaultColor
    if ($currentPrecipProb -gt 0) {
        Write-Host "Precipitation: $currentPrecipProb% chance" -ForegroundColor $defaultColor
    }

    Write-Host "Time: $($currentTimeLocal.ToString('MM/dd/yyyy HH:mm'))" -ForegroundColor $infoColor
}

if ($showTodayForecast) {
    Write-Host ""
    Write-Host "*** Today's Forecast ***" -ForegroundColor $titleColor
    $wrappedForecast = Format-TextWrap -Text $todayForecast -Width $Host.UI.RawUI.WindowSize.Width
    $wrappedForecast | ForEach-Object { Write-Host $_ -ForegroundColor $defaultColor }
}

if ($showTomorrowForecast) {
    Write-Host ""
    Write-Host "*** Tomorrow's Forecast ***" -ForegroundColor $titleColor
    $wrappedTomorrow = Format-TextWrap -Text $tomorrowForecast -Width $Host.UI.RawUI.WindowSize.Width
    $wrappedTomorrow | ForEach-Object { Write-Host $_ -ForegroundColor $defaultColor }
}

# --- Hourly Forecast ---
if ($showHourlyForecast) {
    Write-Host ""
    Write-Host "*** Hourly Forecast ***" -ForegroundColor $titleColor
    $hourlyPeriods = $hourlyData.properties.periods
    $hourCount = 0

    foreach ($period in $hourlyPeriods) {
        if ($hourCount -ge 12) { break }
        
        $periodTime = [DateTime]::Parse($period.startTime)
        $hourDisplay = $periodTime.ToString("HH:mm")
        $temp = $period.temperature
        $shortForecast = $period.shortForecast
        $wind = $period.windSpeed
        $windDir = $period.windDirection
        $precipProb = $period.probabilityOfPrecipitation.value
        
        # Determine if this period is during day or night using NWS API isDaytime property
        $isPeriodDaytime = Test-IsDaytime $period
        $periodIcon = Get-WeatherIcon $period.icon $isPeriodDaytime
        
        # Color code temperature
        $tempColor = if ([int]$temp -lt 33 -or [int]$temp -gt 89) { $alertColor } else { $defaultColor }
        
        # Color code precipitation probability
        $precipColor = if ($precipProb -gt 50) { $alertColor } elseif ($precipProb -gt 20) { "Yellow" } else { $defaultColor }
        
        # Build the formatted line
        $formattedLine = Format-HourlyLine -Time $hourDisplay -Icon $periodIcon -Temp $temp -Wind $wind -WindDir $windDir -PrecipProb $precipProb -Forecast $shortForecast
        
        # Split the line into parts for color coding
        $tempStart = $formattedLine.IndexOf(" $temp°F ")
        $precipStart = $formattedLine.IndexOf(" ($precipProb%)")
        
        if ($tempStart -ge 0) {
            # Write everything before temperature
            Write-Host $formattedLine.Substring(0, $tempStart) -ForegroundColor $defaultColor -NoNewline
            # Write temperature with color
            Write-Host " $temp°F " -ForegroundColor $tempColor -NoNewline
            
            # Handle precipitation probability color coding
            if ($precipStart -ge 0) {
                # Write everything between temperature and precipitation
                $afterTemp = $formattedLine.Substring($tempStart + " $temp°F ".Length, $precipStart - ($tempStart + " $temp°F ".Length))
                Write-Host $afterTemp -ForegroundColor $defaultColor -NoNewline
                # Write precipitation probability with color
                Write-Host " ($precipProb%)" -ForegroundColor $precipColor -NoNewline
                # Write everything after precipitation
                $afterPrecip = $formattedLine.Substring($precipStart + " ($precipProb%)".Length)
                Write-Host $afterPrecip -ForegroundColor $defaultColor
            } else {
                # Write everything after temperature (no precipitation probability)
                $afterTemp = $formattedLine.Substring($tempStart + " $temp°F ".Length)
                Write-Host $afterTemp -ForegroundColor $defaultColor
            }
        } else {
            # Fallback if temperature not found
            Write-Host $formattedLine -ForegroundColor $defaultColor
        }
        
        $hourCount++
    }
}

# --- 7-Day Forecast Summary ---
if ($showSevenDayForecast) {
    Write-Host ""
    Write-Host "*** 7-Day Forecast Summary ***" -ForegroundColor $titleColor
    $forecastPeriods = $forecastData.properties.periods
    $dayCount = 0
    $processedDays = @{}

    foreach ($period in $forecastPeriods) {
        if ($dayCount -ge 7) { break }
        
        $periodTime = [DateTime]::Parse($period.startTime)
        $dayName = $periodTime.ToString("ddd")
        
        # Skip if we've already processed this day
        if ($processedDays.ContainsKey($dayName)) { continue }
        
        $temp = $period.temperature
        $shortForecast = $period.shortForecast
        $precipProb = $period.probabilityOfPrecipitation.value
        
        # Determine if this period is during day or night using NWS API isDaytime property
        $isPeriodDaytime = Test-IsDaytime $period
        $periodIcon = Get-WeatherIcon $period.icon $isPeriodDaytime
        
        # Find the corresponding night period for high/low
        $nightTemp = $null
        foreach ($nightPeriod in $forecastPeriods) {
            $nightTime = [DateTime]::Parse($nightPeriod.startTime)
            $nightDayName = $nightTime.ToString("ddd")
            $nightPeriodName = $nightPeriod.name
            
            if ($nightDayName -eq $dayName -and ($nightPeriodName -match "Night" -or $nightPeriodName -match "Overnight")) {
                $nightTemp = $nightPeriod.temperature
                break
            }
        }
        
        # Color code temperature
        $tempColor = if ([int]$temp -lt 33 -or [int]$temp -gt 89) { $alertColor } else { $defaultColor }
        
        # Color code precipitation probability
        $precipColor = if ($precipProb -gt 50) { $alertColor } elseif ($precipProb -gt 20) { "Yellow" } else { $defaultColor }
        
        # Build the formatted line
        $formattedLine = Format-DailyLine -DayName $dayName -Icon $periodIcon -Temp $temp -NightTemp $nightTemp -Forecast $shortForecast -PrecipProb $precipProb
        
        # Split the line into parts for color coding
        $tempStart = $formattedLine.IndexOf(" H:$temp°F")
        if ($tempStart -lt 0) {
            $tempStart = $formattedLine.IndexOf(" $temp°F")
        }
        
        if ($tempStart -ge 0) {
            # Write everything before temperature
            Write-Host $formattedLine.Substring(0, $tempStart) -ForegroundColor $defaultColor -NoNewline
            
            # Write temperature with color
            if ($nightTemp) {
                Write-Host " H:$temp°F L:$nightTemp°F" -ForegroundColor $tempColor -NoNewline
            } else {
                Write-Host " $temp°F" -ForegroundColor $tempColor -NoNewline
            }
            
            # Write everything after temperature
            $tempEnd = if ($nightTemp) { " H:$temp°F L:$nightTemp°F".Length } else { " $temp°F".Length }
            $afterTemp = $formattedLine.Substring($tempStart + $tempEnd)
            Write-Host $afterTemp -ForegroundColor $defaultColor
        } else {
            # Fallback if temperature not found
            Write-Host $formattedLine -ForegroundColor $defaultColor
        }
        
        $processedDays[$dayName] = $true
        $dayCount++
    }
}

# --- Alert Handling ---
if ($showAlerts -and $alertsData -and $alertsData.features.Count -gt 0) {
    Write-Host ""
    Write-Host "*** Active Weather Alerts ***" -ForegroundColor $alertColor
    foreach ($alert in $alertsData.features) {
        $alertProps = $alert.properties
        $alertEvent = $alertProps.event
        $alertHeadline = $alertProps.headline
        $alertDesc = $alertProps.description
        $alertStart = ([DateTime]::Parse($alertProps.effective)).ToLocalTime()
        $alertEnd = ([DateTime]::Parse($alertProps.expires)).ToLocalTime()
        
        Write-Host "*** $alertEvent ***" -ForegroundColor $alertColor
        Write-Host "$alertHeadline" -ForegroundColor $defaultColor
        if (-not $Terse.IsPresent) {
            $wrappedAlert = Format-TextWrap -Text $alertDesc -Width $Host.UI.RawUI.WindowSize.Width
            $wrappedAlert | ForEach-Object { Write-Host $_ -ForegroundColor $defaultColor }
        }
        Write-Host "Effective: $($alertStart.ToString('MM/dd/yyyy HH:mm'))" -ForegroundColor $infoColor
        Write-Host "Expires: $($alertEnd.ToString('MM/dd/yyyy HH:mm'))" -ForegroundColor $infoColor
        Write-Host ""
    }
}

if ($showLocationInfo) {
    Write-Host ""
    Write-Host "*** Location Information ***" -ForegroundColor $titleColor
    if ($county) {
        Write-Host "County: $county" -ForegroundColor $defaultColor
    }
    Write-Host "Time Zone: $timeZone" -ForegroundColor $defaultColor
    Write-Host "Radar Station: $radarStation" -ForegroundColor $defaultColor
    Write-Host "Coordinates: $lat, $lon" -ForegroundColor $defaultColor

    Write-Host ""
    Write-Host "https://forecast.weather.gov/MapClick.php?lat=$lat&lon=$lon" -ForegroundColor Cyan
}

# Check if we're in an interactive environment that supports ReadKey
$isInteractiveEnvironment = $false

# Check if we're in a Windows terminal environment
if ($parentName -match '^(WindowsTerminal.exe|PowerShell|cmd)') {
    $isInteractiveEnvironment = $true
}
# Check if we're in an SSH session with a proper terminal
elseif ($env:SSH_CONNECTION -and $Host.UI.RawUI.WindowSize.Width -gt 0) {
    $isInteractiveEnvironment = $true
}
# Check if we have a proper terminal size (indicating interactive terminal)
elseif ($Host.UI.RawUI.WindowSize.Width -gt 0 -and $Host.UI.RawUI.WindowSize.Height -gt 0) {
    $isInteractiveEnvironment = $true
}

if ($isInteractiveEnvironment -and -not $NoInteractive.IsPresent) {
    Write-Verbose "Parent:$parentName - Interactive environment detected"
    
    # Interactive mode - listen for keys to switch display modes
    Write-Host ""
    Write-Host "*** Interactive Mode ***" -ForegroundColor $titleColor
    Write-Host "Hourly[" -ForegroundColor White -NoNewline; Write-Host "H" -ForegroundColor Cyan -NoNewline; Write-Host "], 7-Day[" -ForegroundColor White -NoNewline; Write-Host "D" -ForegroundColor Cyan -NoNewline; Write-Host "], Terse[" -ForegroundColor White -NoNewline; Write-Host "T" -ForegroundColor Cyan -NoNewline; Write-Host "], Full[" -ForegroundColor White -NoNewline; Write-Host "F" -ForegroundColor Cyan -NoNewline; Write-Host "], Exit[" -ForegroundColor White -NoNewline; Write-Host "Enter" -ForegroundColor Cyan -NoNewline; Write-Host "]" -ForegroundColor White
    
    while ($true) {
        try {
            $key = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
            

            
            switch ($key.VirtualKeyCode) {
            72 { # H key
                Clear-Host
                Write-Host "*** $city, $state - Hourly Forecast Only ***" -ForegroundColor $titleColor
                # Show only hourly forecast
                Write-Host ""
                Write-Host "*** Hourly Forecast ***" -ForegroundColor $titleColor
                $hourlyPeriods = $hourlyData.properties.periods
                $hourCount = 0

                foreach ($period in $hourlyPeriods) {
                    if ($hourCount -ge 12) { break }
                    
                    $periodTime = [DateTime]::Parse($period.startTime)
                    $hourDisplay = $periodTime.ToString("HH:mm")
                    $temp = $period.temperature
                    $shortForecast = $period.shortForecast
                    $wind = $period.windSpeed
                    $windDir = $period.windDirection
                    $precipProb = $period.probabilityOfPrecipitation.value
                    
                    # Determine if this period is during day or night using NWS API isDaytime property
                    $isPeriodDaytime = Test-IsDaytime $period
                    $periodIcon = Get-WeatherIcon $period.icon $isPeriodDaytime
                    
                    # Color code temperature
                    $tempColor = if ([int]$temp -lt 33 -or [int]$temp -gt 89) { $alertColor } else { $defaultColor }
                    
                    # Color code precipitation probability
                    $precipColor = if ($precipProb -gt 50) { $alertColor } elseif ($precipProb -gt 20) { "Yellow" } else { $defaultColor }
                    
                    # Build the formatted line
                    $formattedLine = Format-HourlyLine -Time $hourDisplay -Icon $periodIcon -Temp $temp -Wind $wind -WindDir $windDir -PrecipProb $precipProb -Forecast $shortForecast
                    
                    # Split the line into parts for color coding
                    $tempStart = $formattedLine.IndexOf(" $temp°F ")
                    $precipStart = $formattedLine.IndexOf(" ($precipProb%)")
                    
                    if ($tempStart -ge 0) {
                        # Write everything before temperature
                        Write-Host $formattedLine.Substring(0, $tempStart) -ForegroundColor $defaultColor -NoNewline
                        # Write temperature with color
                        Write-Host " $temp°F " -ForegroundColor $tempColor -NoNewline
                        
                        # Handle precipitation probability color coding
                        if ($precipStart -ge 0) {
                            # Write everything between temperature and precipitation
                            $afterTemp = $formattedLine.Substring($tempStart + " $temp°F ".Length, $precipStart - ($tempStart + " $temp°F ".Length))
                            Write-Host $afterTemp -ForegroundColor $defaultColor -NoNewline
                            # Write precipitation probability with color
                            Write-Host " ($precipProb%)" -ForegroundColor $precipColor -NoNewline
                            # Write everything after precipitation
                            $afterPrecip = $formattedLine.Substring($precipStart + " ($precipProb%)".Length)
                            Write-Host $afterPrecip -ForegroundColor $defaultColor
                        } else {
                            # Write everything after temperature (no precipitation probability)
                            $afterTemp = $formattedLine.Substring($tempStart + " $temp°F ".Length)
                            Write-Host $afterTemp -ForegroundColor $defaultColor
                        }
                    } else {
                        # Fallback if temperature not found
                        Write-Host $formattedLine -ForegroundColor $defaultColor
                    }
                    
                    $hourCount++
                }
                Write-Host ""
                Write-Host "Hourly[" -ForegroundColor White -NoNewline; Write-Host "H" -ForegroundColor Cyan -NoNewline; Write-Host "], 7-Day[" -ForegroundColor White -NoNewline; Write-Host "D" -ForegroundColor Cyan -NoNewline; Write-Host "], Terse[" -ForegroundColor White -NoNewline; Write-Host "T" -ForegroundColor Cyan -NoNewline; Write-Host "], Full[" -ForegroundColor White -NoNewline; Write-Host "F" -ForegroundColor Cyan -NoNewline; Write-Host "], Exit[" -ForegroundColor White -NoNewline; Write-Host "Enter" -ForegroundColor Cyan -NoNewline; Write-Host "]" -ForegroundColor White
            }
            68 { # D key
                Clear-Host
                Write-Host "*** $city, $state - 7-Day Forecast Only ***" -ForegroundColor $titleColor
                # Show only 7-day forecast
                Write-Host ""
                Write-Host "*** 7-Day Forecast Summary ***" -ForegroundColor $titleColor
                $forecastPeriods = $forecastData.properties.periods
                $dayCount = 0
                $processedDays = @{}

                foreach ($period in $forecastPeriods) {
                    if ($dayCount -ge 7) { break }
                    
                    $periodTime = [DateTime]::Parse($period.startTime)
                    $dayName = $periodTime.ToString("ddd")
                    
                    # Skip if we've already processed this day
                    if ($processedDays.ContainsKey($dayName)) { continue }
                    
                    $temp = $period.temperature
                    $shortForecast = $period.shortForecast
                    $precipProb = $period.probabilityOfPrecipitation.value
                    
                    # Determine if this period is during day or night using NWS API isDaytime property
                    $isPeriodDaytime = Test-IsDaytime $period
                    $periodIcon = Get-WeatherIcon $period.icon $isPeriodDaytime
                    
                    # Find the corresponding night period for high/low
                    $nightTemp = $null
                    foreach ($nightPeriod in $forecastPeriods) {
                        $nightTime = [DateTime]::Parse($nightPeriod.startTime)
                        $nightDayName = $nightTime.ToString("ddd")
                        $nightPeriodName = $nightPeriod.name
                        
                        if ($nightDayName -eq $dayName -and ($nightPeriodName -match "Night" -or $nightPeriodName -match "Overnight")) {
                            $nightTemp = $nightPeriod.temperature
                            break
                        }
                    }
                    
                    # Color code temperature
                    $tempColor = if ([int]$temp -lt 33 -or [int]$temp -gt 89) { $alertColor } else { $defaultColor }
                    
                    # Color code precipitation probability
                    $precipColor = if ($precipProb -gt 50) { $alertColor } elseif ($precipProb -gt 20) { "Yellow" } else { $defaultColor }
                    
                    # Build the formatted line
                    $formattedLine = Format-DailyLine -DayName $dayName -Icon $periodIcon -Temp $temp -NightTemp $nightTemp -Forecast $shortForecast -PrecipProb $precipProb
                    
                    # Split the line into parts for color coding
                    $tempStart = $formattedLine.IndexOf(" H:$temp°F")
                    if ($tempStart -lt 0) {
                        $tempStart = $formattedLine.IndexOf(" $temp°F")
                    }
                    
                    if ($tempStart -ge 0) {
                        # Write everything before temperature
                        Write-Host $formattedLine.Substring(0, $tempStart) -ForegroundColor $defaultColor -NoNewline
                        
                        # Write temperature with color
                        if ($nightTemp) {
                            Write-Host " H:$temp°F L:$nightTemp°F" -ForegroundColor $tempColor -NoNewline
                        } else {
                            Write-Host " $temp°F" -ForegroundColor $tempColor -NoNewline
                        }
                        
                        # Write everything after temperature
                        $tempEnd = if ($nightTemp) { " H:$temp°F L:$nightTemp°F".Length } else { " $temp°F".Length }
                        $afterTemp = $formattedLine.Substring($tempStart + $tempEnd)
                        Write-Host $afterTemp -ForegroundColor $defaultColor
                    } else {
                        # Fallback if temperature not found
                        Write-Host $formattedLine -ForegroundColor $defaultColor
                    }
                    
                    $processedDays[$dayName] = $true
                    $dayCount++
                }
                Write-Host ""
                Write-Host "Hourly[" -ForegroundColor White -NoNewline; Write-Host "H" -ForegroundColor Cyan -NoNewline; Write-Host "], 7-Day[" -ForegroundColor White -NoNewline; Write-Host "D" -ForegroundColor Cyan -NoNewline; Write-Host "], Terse[" -ForegroundColor White -NoNewline; Write-Host "T" -ForegroundColor Cyan -NoNewline; Write-Host "], Full[" -ForegroundColor White -NoNewline; Write-Host "F" -ForegroundColor Cyan -NoNewline; Write-Host "], Exit[" -ForegroundColor White -NoNewline; Write-Host "Enter" -ForegroundColor Cyan -NoNewline; Write-Host "]" -ForegroundColor White
            }
            84 { # T key
                Clear-Host
                Write-Host "*** $city, $state - Terse Mode ***" -ForegroundColor $titleColor
                # Show current conditions and today's forecast only
                Write-Host "Currently: $weatherIcon $currentConditions" -ForegroundColor $defaultColor
                Write-Host "Temperature: $currentTemp°F" -ForegroundColor $tempColor -NoNewline
                if ($currentTempTrend) {
                    $trendIcon = switch ($currentTempTrend) {
                        "rising" { "↗️" }
                        "falling" { "↘️" }
                        "steady" { "→" }
                        default { "" }
                    }
                    Write-Host " $trendIcon " -ForegroundColor $defaultColor -NoNewline
                }
                Write-Host ""

                Write-Host "Wind: $currentWind $currentWindDir" -ForegroundColor $windColor -NoNewline
                if ($windGust) {
                    Write-Host " (gusts to $windGust mph)" -ForegroundColor $alertColor -NoNewline
                }
                Write-Host ""

                Write-Host "Humidity: $currentHumidity%" -ForegroundColor $defaultColor
                Write-Host "Dew Point: $([math]::Round($currentDewPoint * 9/5 + 32, 1))°F" -ForegroundColor $defaultColor
                if ($currentPrecipProb -gt 0) {
                    Write-Host "Precipitation: $currentPrecipProb% chance" -ForegroundColor $defaultColor
                }

                Write-Host "Time: $($currentTimeLocal.ToString('MM/dd/yyyy HH:mm'))" -ForegroundColor $infoColor

                Write-Host ""
                Write-Host "*** Today's Forecast ***" -ForegroundColor $titleColor
                $wrappedForecast = Format-TextWrap -Text $todayForecast -Width $Host.UI.RawUI.WindowSize.Width
                $wrappedForecast | ForEach-Object { Write-Host $_ -ForegroundColor $defaultColor }

                # Show alerts if they exist
                if ($alertsData -and $alertsData.features.Count -gt 0) {
                    Write-Host ""
                    Write-Host "*** Active Weather Alerts ***" -ForegroundColor $alertColor
                    foreach ($alert in $alertsData.features) {
                        $alertProps = $alert.properties
                        $alertEvent = $alertProps.event
                        $alertHeadline = $alertProps.headline
                        $alertStart = ([DateTime]::Parse($alertProps.effective)).ToLocalTime()
                        $alertEnd = ([DateTime]::Parse($alertProps.expires)).ToLocalTime()
                        
                        Write-Host "*** $alertEvent ***" -ForegroundColor $alertColor
                        Write-Host "$alertHeadline" -ForegroundColor $defaultColor
                        Write-Host "Effective: $($alertStart.ToString('MM/dd/yyyy HH:mm'))" -ForegroundColor $infoColor
                        Write-Host "Expires: $($alertEnd.ToString('MM/dd/yyyy HH:mm'))" -ForegroundColor $infoColor
                        Write-Host ""
                    }
                }
                Write-Host ""
                Write-Host "Hourly[" -ForegroundColor White -NoNewline; Write-Host "H" -ForegroundColor Cyan -NoNewline; Write-Host "], 7-Day[" -ForegroundColor White -NoNewline; Write-Host "D" -ForegroundColor Cyan -NoNewline; Write-Host "], Terse[" -ForegroundColor White -NoNewline; Write-Host "T" -ForegroundColor Cyan -NoNewline; Write-Host "], Full[" -ForegroundColor White -NoNewline; Write-Host "F" -ForegroundColor Cyan -NoNewline; Write-Host "], Exit[" -ForegroundColor White -NoNewline; Write-Host "Enter" -ForegroundColor Cyan -NoNewline; Write-Host "]" -ForegroundColor White
            }
            70 { # F key
                Clear-Host
                # Force full view regardless of initial flags
                $showCurrentConditions = $true
                $showTodayForecast = $true
                $showTomorrowForecast = $true
                $showHourlyForecast = $true
                $showSevenDayForecast = $true
                $showAlerts = $true
                $showLocationInfo = $true
                
                # Show full display (re-run the original display logic)
                if ($showCurrentConditions) {
                    Write-Host "*** $city, $state Current Conditions ***" -ForegroundColor $titleColor
                    Write-Host "Currently: $weatherIcon $currentConditions" -ForegroundColor $defaultColor
                    Write-Host "Temperature: $currentTemp°F" -ForegroundColor $tempColor -NoNewline
                    if ($currentTempTrend) {
                        $trendIcon = switch ($currentTempTrend) {
                            "rising" { "↗️" }
                            "falling" { "↘️" }
                            "steady" { "→" }
                            default { "" }
                        }
                        Write-Host " $trendIcon " -ForegroundColor $defaultColor -NoNewline
                    }
                    Write-Host ""

                    Write-Host "Wind: $currentWind $currentWindDir" -ForegroundColor $windColor -NoNewline
                    if ($windGust) {
                        Write-Host " (gusts to $windGust mph)" -ForegroundColor $alertColor -NoNewline
                    }
                    Write-Host ""

                    Write-Host "Humidity: $currentHumidity%" -ForegroundColor $defaultColor
                    Write-Host "Dew Point: $([math]::Round($currentDewPoint * 9/5 + 32, 1))°F" -ForegroundColor $defaultColor
                    if ($currentPrecipProb -gt 0) {
                        Write-Host "Precipitation: $currentPrecipProb% chance" -ForegroundColor $defaultColor
                    }

                    Write-Host "Time: $($currentTimeLocal.ToString('MM/dd/yyyy HH:mm'))" -ForegroundColor $infoColor
                }

                if ($showTodayForecast) {
                    Write-Host ""
                    Write-Host "*** Today's Forecast ***" -ForegroundColor $titleColor
                    $wrappedForecast = Format-TextWrap -Text $todayForecast -Width $Host.UI.RawUI.WindowSize.Width
                    $wrappedForecast | ForEach-Object { Write-Host $_ -ForegroundColor $defaultColor }
                }

                if ($showTomorrowForecast) {
                    Write-Host ""
                    Write-Host "*** Tomorrow's Forecast ***" -ForegroundColor $titleColor
                    $wrappedTomorrow = Format-TextWrap -Text $tomorrowForecast -Width $Host.UI.RawUI.WindowSize.Width
                    $wrappedTomorrow | ForEach-Object { Write-Host $_ -ForegroundColor $defaultColor }
                }

                if ($showHourlyForecast) {
                    Write-Host ""
                    Write-Host "*** Hourly Forecast ***" -ForegroundColor $titleColor
                    $hourlyPeriods = $hourlyData.properties.periods
                    $hourCount = 0

                    foreach ($period in $hourlyPeriods) {
                        if ($hourCount -ge 12) { break }
                        
                        $periodTime = [DateTime]::Parse($period.startTime)
                        $hourDisplay = $periodTime.ToString("HH:mm")
                        $temp = $period.temperature
                        $shortForecast = $period.shortForecast
                        $wind = $period.windSpeed
                        $windDir = $period.windDirection
                        $precipProb = $period.probabilityOfPrecipitation.value
                        
                        # Determine if this period is during day or night using NWS API isDaytime property
                        $isPeriodDaytime = Test-IsDaytime $period
                        $periodIcon = Get-WeatherIcon $period.icon $isPeriodDaytime
                        
                        # Color code temperature
                        $tempColor = if ([int]$temp -lt 33 -or [int]$temp -gt 89) { $alertColor } else { $defaultColor }
                        
                        # Color code precipitation probability
                        $precipColor = if ($precipProb -gt 50) { $alertColor } elseif ($precipProb -gt 20) { "Yellow" } else { $defaultColor }
                        
                        # Build the formatted line
                        $formattedLine = Format-HourlyLine -Time $hourDisplay -Icon $periodIcon -Temp $temp -Wind $wind -WindDir $windDir -PrecipProb $precipProb -Forecast $shortForecast
                        
                        # Split the line into parts for color coding
                        $tempStart = $formattedLine.IndexOf(" $temp°F ")
                        $precipStart = $formattedLine.IndexOf(" ($precipProb%)")
                        
                        if ($tempStart -ge 0) {
                            # Write everything before temperature
                            Write-Host $formattedLine.Substring(0, $tempStart) -ForegroundColor $defaultColor -NoNewline
                            # Write temperature with color
                            Write-Host " $temp°F " -ForegroundColor $tempColor -NoNewline
                            
                            # Handle precipitation probability color coding
                            if ($precipStart -ge 0) {
                                # Write everything between temperature and precipitation
                                $afterTemp = $formattedLine.Substring($tempStart + " $temp°F ".Length, $precipStart - ($tempStart + " $temp°F ".Length))
                                Write-Host $afterTemp -ForegroundColor $defaultColor -NoNewline
                                # Write precipitation probability with color
                                Write-Host " ($precipProb%)" -ForegroundColor $precipColor -NoNewline
                                # Write everything after precipitation
                                $afterPrecip = $formattedLine.Substring($precipStart + " ($precipProb%)".Length)
                                Write-Host $afterPrecip -ForegroundColor $defaultColor
                            } else {
                                # Write everything after temperature (no precipitation probability)
                                $afterTemp = $formattedLine.Substring($tempStart + " $temp°F ".Length)
                                Write-Host $afterTemp -ForegroundColor $defaultColor
                            }
                        } else {
                            # Fallback if temperature not found
                            Write-Host $formattedLine -ForegroundColor $defaultColor
                        }
                        
                        $hourCount++
                    }
                }

                if ($showSevenDayForecast) {
                    Write-Host ""
                    Write-Host "*** 7-Day Forecast Summary ***" -ForegroundColor $titleColor
                    $forecastPeriods = $forecastData.properties.periods
                    $dayCount = 0
                    $processedDays = @{}

                    foreach ($period in $forecastPeriods) {
                        if ($dayCount -ge 7) { break }
                        
                        $periodTime = [DateTime]::Parse($period.startTime)
                        $dayName = $periodTime.ToString("ddd")
                        
                        # Skip if we've already processed this day
                        if ($processedDays.ContainsKey($dayName)) { continue }
                        
                        $temp = $period.temperature
                        $shortForecast = $period.shortForecast
                        $precipProb = $period.probabilityOfPrecipitation.value
                        
                        # Determine if this period is during day or night using NWS API isDaytime property
                        $isPeriodDaytime = Test-IsDaytime $period
                        $periodIcon = Get-WeatherIcon $period.icon $isPeriodDaytime
                        
                        # Find the corresponding night period for high/low
                        $nightTemp = $null
                        foreach ($nightPeriod in $forecastPeriods) {
                            $nightTime = [DateTime]::Parse($nightPeriod.startTime)
                            $nightDayName = $nightTime.ToString("ddd")
                            $nightPeriodName = $nightPeriod.name
                            
                            if ($nightDayName -eq $dayName -and ($nightPeriodName -match "Night" -or $nightPeriodName -match "Overnight")) {
                                $nightTemp = $nightPeriod.temperature
                                break
                            }
                        }
                        
                        # Color code temperature
                        $tempColor = if ([int]$temp -lt 33 -or [int]$temp -gt 89) { $alertColor } else { $defaultColor }
                        
                        # Color code precipitation probability
                        $precipColor = if ($precipProb -gt 50) { $alertColor } elseif ($precipProb -gt 20) { "Yellow" } else { $defaultColor }
                        
                        # Build the formatted line
                        $formattedLine = Format-DailyLine -DayName $dayName -Icon $periodIcon -Temp $temp -NightTemp $nightTemp -Forecast $shortForecast -PrecipProb $precipProb
                        
                        # Split the line into parts for color coding
                        $tempStart = $formattedLine.IndexOf(" H:$temp°F")
                        if ($tempStart -lt 0) {
                            $tempStart = $formattedLine.IndexOf(" $temp°F")
                        }
                        
                        if ($tempStart -ge 0) {
                            # Write everything before temperature
                            Write-Host $formattedLine.Substring(0, $tempStart) -ForegroundColor $defaultColor -NoNewline
                            
                            # Write temperature with color
                            if ($nightTemp) {
                                Write-Host " H:$temp°F L:$nightTemp°F" -ForegroundColor $tempColor -NoNewline
                            } else {
                                Write-Host " $temp°F" -ForegroundColor $tempColor -NoNewline
                            }
                            
                            # Write everything after temperature
                            $tempEnd = if ($nightTemp) { " H:$temp°F L:$nightTemp°F".Length } else { " $temp°F".Length }
                            $afterTemp = $formattedLine.Substring($tempStart + $tempEnd)
                            Write-Host $afterTemp -ForegroundColor $defaultColor
                        } else {
                            # Fallback if temperature not found
                            Write-Host $formattedLine -ForegroundColor $defaultColor
                        }
                        
                        $processedDays[$dayName] = $true
                        $dayCount++
                    }
                }

                if ($showAlerts -and $alertsData -and $alertsData.features.Count -gt 0) {
                    Write-Host ""
                    Write-Host "*** Active Weather Alerts ***" -ForegroundColor $alertColor
                    foreach ($alert in $alertsData.features) {
                        $alertProps = $alert.properties
                        $alertEvent = $alertProps.event
                        $alertHeadline = $alertProps.headline
                        $alertDesc = $alertProps.description
                        $alertStart = ([DateTime]::Parse($alertProps.effective)).ToLocalTime()
                        $alertEnd = ([DateTime]::Parse($alertProps.expires)).ToLocalTime()
                        
                        Write-Host "*** $alertEvent ***" -ForegroundColor $alertColor
                        Write-Host "$alertHeadline" -ForegroundColor $defaultColor
                        if (-not $Terse.IsPresent) {
                            $wrappedAlert = Format-TextWrap -Text $alertDesc -Width $Host.UI.RawUI.WindowSize.Width
                            $wrappedAlert | ForEach-Object { Write-Host $_ -ForegroundColor $defaultColor }
                        }
                        Write-Host "Effective: $($alertStart.ToString('MM/dd/yyyy HH:mm'))" -ForegroundColor $infoColor
                        Write-Host "Expires: $($alertEnd.ToString('MM/dd/yyyy HH:mm'))" -ForegroundColor $infoColor
                        Write-Host ""
                    }
                }

                if ($showLocationInfo) {
                    Write-Host ""
                    Write-Host "*** Location Information ***" -ForegroundColor $titleColor
                    if ($county) {
                        Write-Host "County: $county" -ForegroundColor $defaultColor
                    }
                    Write-Host "Time Zone: $timeZone" -ForegroundColor $defaultColor
                    Write-Host "Radar Station: $radarStation" -ForegroundColor $defaultColor
                    Write-Host "Coordinates: $lat, $lon" -ForegroundColor $defaultColor

                    Write-Host ""
                    Write-Host "https://forecast.weather.gov/MapClick.php?lat=$lat&lon=$lon" -ForegroundColor Cyan
                }
                Write-Host ""
                Write-Host "Hourly[" -ForegroundColor White -NoNewline; Write-Host "H" -ForegroundColor Cyan -NoNewline; Write-Host "], 7-Day[" -ForegroundColor White -NoNewline; Write-Host "D" -ForegroundColor Cyan -NoNewline; Write-Host "], Terse[" -ForegroundColor White -NoNewline; Write-Host "T" -ForegroundColor Cyan -NoNewline; Write-Host "], Full[" -ForegroundColor White -NoNewline; Write-Host "F" -ForegroundColor Cyan -NoNewline; Write-Host "], Exit[" -ForegroundColor White -NoNewline; Write-Host "Enter" -ForegroundColor Cyan -NoNewline; Write-Host "]" -ForegroundColor White
            }
            13 { # Enter key
                Write-Host "Exiting..." -ForegroundColor Yellow
                return
            }
            28 { # NumPad Enter key
                Write-Host "Exiting..." -ForegroundColor Yellow
                return
            }
        }
        }
        catch {
            Write-Host "Interactive mode not supported in this environment. Exiting..." -ForegroundColor Yellow
            return
        }
    }
}