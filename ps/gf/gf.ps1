<#
.SYNOPSIS
    A PowerShell script to retrieve and display detailed weather information for a specified location.
    
.DESCRIPTION
    This script accepts a US zip code or a "City, State" string as input. It uses the
    National Weather Service API to retrieve and display detailed weather information.

    The script first uses the OpenStreetMap Nominatim API for geocoding to determine the latitude 
    and longitude of the location, then fetches the current weather, daily forecasts, and weather alerts.

    No API key is required as both the National Weather Service API and OpenStreetMap Nominatim API are free and open.
    
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
    This script uses the free National Weather Service API and OpenStreetMap Nominatim API, 
    both of which require no API key. The APIs are limited to locations within the United States.

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

    [Alias('d')]
    [switch]$Daily,

    [Alias('x')]
    [switch]$NoInteractive,

    [Alias('r')]
    [switch]$Rain,

    [Alias('w')]
    [switch]$Wind
)

# --- Helper Functions ---

# Force TLS 1.2.
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

# Global error handler for 503 Server Unavailable errors
$ErrorActionPreference = "Stop"
trap {
    if ($_.Exception.Message -match "503.*Server Unavailable" -or $_.Exception.Message -match "503 Server Unavailable") {
        Write-Host "The server is not currently available, try again later." -ForegroundColor Red
        exit 1
    } else {
        throw
    }
}

# Ensure Unicode output (degree symbol, emoji) renders correctly
try {
    [System.Console]::OutputEncoding = [System.Text.Encoding]::UTF8
    [System.Console]::InputEncoding  = [System.Text.Encoding]::UTF8
    $OutputEncoding = [System.Text.Encoding]::UTF8
    $PSDefaultParameterValues['Out-File:Encoding'] = 'utf8'
} catch {}

# --- CONSTANTS ---
# Temperature thresholds for color coding (°F)
$script:COLD_TEMP_THRESHOLD = 33
$script:HOT_TEMP_THRESHOLD = 89

# Wind speed threshold for alert color (mph)
$script:WIND_ALERT_THRESHOLD = 16

# Precipitation probability thresholds (%)
$script:HIGH_PRECIP_THRESHOLD = 50
$script:MEDIUM_PRECIP_THRESHOLD = 20

# API configuration
$script:USER_AGENT = "GetForecast/1.0 (081625PDX)"
$script:MAX_HOURLY_FORECAST_HOURS = 12  # Max 96 available in API
$script:MAX_DAILY_FORECAST_DAYS = 7

# Define User-Agent for NWS API requests
$userAgent = $script:USER_AGENT

# --- HELP LOGIC ---
if ($Help -or (($Terse.IsPresent -or $Hourly.IsPresent -or $Daily.IsPresent -or $Rain.IsPresent -or $Wind.IsPresent -or $NoInteractive.IsPresent) -and -not $Location)) {
    Write-Host "Usage: .\gf.ps1 [ZipCode | `"City, State`"] [Options] [-Verbose]" -ForegroundColor Green
    Write-Host " • Provide a 5-digit zipcode or a City, State (e.g., 'Portland, OR')." -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Options:" -ForegroundColor Blue
    Write-Host "  -t, -Terse    Show only current conditions and today's forecast" -ForegroundColor Cyan
    Write-Host "  -h, -Hourly   Show only the hourly forecast (up to $($script:MAX_HOURLY_FORECAST_HOURS) hours)" -ForegroundColor Cyan
    Write-Host "  -d, -Daily    Show only the $($script:MAX_DAILY_FORECAST_DAYS)-day forecast summary" -ForegroundColor Cyan
    Write-Host "  -r, -Rain     Show rain likelihood forecast with sparklines (exits immediately)" -ForegroundColor Cyan
    Write-Host "  -w, -Wind     Show wind outlook forecast with direction glyphs (exits immediately)" -ForegroundColor Cyan
    Write-Host "  -x, -NoInteractive Exit immediately (no interactive mode)" -ForegroundColor Cyan
    Write-Host ""
         Write-Host "Interactive Mode:" -ForegroundColor Blue
     Write-Host "  When run interactively (not from terminal), the script enters interactive mode." -ForegroundColor Cyan
     Write-Host "  Use keyboard shortcuts to switch between display modes:" -ForegroundColor Cyan
     Write-Host "    [H] - Switch to hourly forecast only (with scrolling)" -ForegroundColor Cyan
     Write-Host "    [D] - Switch to $($script:MAX_DAILY_FORECAST_DAYS)-day forecast only" -ForegroundColor Cyan
     Write-Host "    [T] - Switch to terse mode (current + today)" -ForegroundColor Cyan
     Write-Host "    [F] - Return to full display" -ForegroundColor Cyan
     Write-Host "    [Enter] - Exit the script" -ForegroundColor Cyan
     Write-Host "  In hourly mode, use [↑] and [↓] arrows to scroll through all 48 hours" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "This script retrieves weather info from National Weather Service API (geocoding via OpenStreetMap) and outputs:" -ForegroundColor Blue
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
    Write-Host "  .\gf.ps1 97219 -d For Daily Forecast" -ForegroundColor Cyan
    Write-Host "  .\gf.ps1 97219 -h -x For Hourly Forecast and Exit" -ForegroundColor Cyan
    Write-Host "  .\gf.ps1 97219 -r For Rain Likelihood Forecast" -ForegroundColor Cyan
    Write-Host "  .\gf.ps1 97219 -w For Wind Outlook Forecast" -ForegroundColor Cyan
    Write-Host "  .\gf.ps1 -Help" -ForegroundColor Cyan
    return
}

# Function to get the parent process name using CIM
function Get-ParentProcessName {
    $proc = Get-CimInstance Win32_Process -Filter "ProcessId=$PID"
    if ($proc) {
        $parentProc = Get-CimInstance Win32_Process -Filter "ProcessId=$($proc.ParentProcessId)"
        return $parentProc.Name
    }
    return $null
}

# Function to create and execute API jobs
function Start-ApiJob {
    param(
        [string]$Url,
        [hashtable]$Headers,
        [string]$JobName
    )
    
    Write-Verbose "Starting API job: $JobName"
    
    $job = Start-Job -ScriptBlock {
        param($url, $hdrs)
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        try {
            $response = Invoke-RestMethod -Uri $url -Method Get -Headers $hdrs -ErrorAction Stop
            return $response | ConvertTo-Json -Depth 10
        }
        catch {
            if ($_.Exception.Message -match "503.*Server Unavailable") {
                throw "503 Server Unavailable"
            } else {
                throw "Failed to retrieve data from job: $($_.Exception.Message)"
            }
        }
    } -ArgumentList $Url, $Headers
    
    return $job
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
            Write-Host '     \|/    ' -ForegroundColor Yellow -NoNewline; Write-Host "      .-~~~~~~-.    " -ForegroundColor Gray
            Write-Host '   -- O --  ' -ForegroundColor Yellow -NoNewline; Write-Host "     /_)      ( \   " -ForegroundColor Gray
            Write-Host '     /|\    ' -ForegroundColor Yellow -NoNewline; Write-Host "    (   ( )    ( )  " -ForegroundColor Gray
            Write-Host '            ' -ForegroundColor Yellow -NoNewline; Write-Host "     `-~~~~~~~~~-`  " -ForegroundColor Gray
            Write-Host '   Welcome  ' -ForegroundColor White  -NoNewline; Write-Host "       ''    ''     " -ForegroundColor Cyan
            Write-Host '      to    ' -ForegroundColor White  -NoNewline; Write-Host "      ''    ''      " -ForegroundColor Cyan
            Write-Host ' GetForecast' -ForegroundColor Green  -NoNewline; Write-Host "     ''    ''       " -ForegroundColor Cyan
            Write-Host '~~~~~~~~~~~~~~' -ForegroundColor Yellow -NoNewline; Write-Host "~~~~~~~~~~~~~~~~~~" -ForegroundColor Green
            Write-Host ""
            $Location = Read-Host "Enter a location (Zip Code or City, State)"
            if ([string]::IsNullOrEmpty($Location)) { exit } # Exit if user enters nothing
        }

        Write-Verbose "Input provided: $Location"

        # --- GEOCODING ---
        # Use OpenStreetMap Nominatim API for all geocoding (free, no API key required)
        # This consolidates our API providers and reduces dependencies
        Write-Verbose "Using OpenStreetMap Nominatim API for geocoding."
        
        # Prepare location for API query
        $locationForApi = if ($Location -match ",") { $Location } else { "$Location,US" }
        $encodedLocation = [uri]::EscapeDataString($locationForApi)
        $geoUrl = "https://nominatim.openstreetmap.org/search?q=$encodedLocation&format=json&limit=1&countrycodes=us"
        Write-Verbose "Geocoding URL: $geoUrl"
        
        $geoData = Invoke-RestMethod "$geoUrl" -ErrorAction Stop
        if ($geoData.Count -eq 0) { throw "No geocoding results found for '$Location'." }
        
        $lat = [double]$geoData[0].lat
        $lon = [double]$geoData[0].lon
        
        # Extract city and state based on the type of location returned
        if ($geoData[0].type -eq "postcode") {
            # For zipcodes, parse city and state from display_name
            # Format: "97219, Multnomah, Portland, Multnomah County, Oregon, United States"
            $displayName = $geoData[0].display_name
            Write-Verbose "Parsing zipcode location from display_name: $displayName"
            
            # Extract city (usually the third element after zipcode)
            if ($displayName -match "^\d{5}, [^,]+,\s*([^,]+),") {
                $city = $matches[1].Trim()
            } else {
                # Fallback: use the name field (which contains the zipcode)
                $city = $geoData[0].name
            }
            
            # Extract state from display_name
            # Format: "97219, Multnomah, Portland, Multnomah County, Oregon, United States"
            # Look for state abbreviation in the display_name
            if ($displayName -match ", ([A-Z]{2}), United States$") {
                $state = $matches[1]
            } elseif ($displayName -match ", ([A-Z]{2})$") {
                $state = $matches[1]
            } else {
                # Try to extract state from the full state name in display_name
                # Look for patterns like "Oregon, United States" and map to abbreviation
                $stateMap = @{
                    "Alabama" = "AL"; "Alaska" = "AK"; "Arizona" = "AZ"; "Arkansas" = "AR"; "California" = "CA"
                    "Colorado" = "CO"; "Connecticut" = "CT"; "Delaware" = "DE"; "Florida" = "FL"; "Georgia" = "GA"
                    "Hawaii" = "HI"; "Idaho" = "ID"; "Illinois" = "IL"; "Indiana" = "IN"; "Iowa" = "IA"
                    "Kansas" = "KS"; "Kentucky" = "KY"; "Louisiana" = "LA"; "Maine" = "ME"; "Maryland" = "MD"
                    "Massachusetts" = "MA"; "Michigan" = "MI"; "Minnesota" = "MN"; "Mississippi" = "MS"; "Missouri" = "MO"
                    "Montana" = "MT"; "Nebraska" = "NE"; "Nevada" = "NV"; "New Hampshire" = "NH"; "New Jersey" = "NJ"
                    "New Mexico" = "NM"; "New York" = "NY"; "North Carolina" = "NC"; "North Dakota" = "ND"; "Ohio" = "OH"
                    "Oklahoma" = "OK"; "Oregon" = "OR"; "Pennsylvania" = "PA"; "Rhode Island" = "RI"; "South Carolina" = "SC"
                    "South Dakota" = "SD"; "Tennessee" = "TN"; "Texas" = "TX"; "Utah" = "UT"; "Vermont" = "VT"
                    "Virginia" = "VA"; "Washington" = "WA"; "West Virginia" = "WV"; "Wisconsin" = "WI"; "Wyoming" = "WY"
                }
                
                foreach ($stateName in $stateMap.Keys) {
                    if ($displayName -match ", $stateName, United States$") {
                        $state = $stateMap[$stateName]
                        break
                    }
                }
                
                # If still no match, try fallback methods
                if (-not $state -or $state -eq "US") {
                    # Fallback: extract state from original input if it contains a comma
                    if ($Location -match ", ([A-Z]{2})") {
                        $state = $matches[1]
                    } else {
                        # For zipcodes without state info, we'll need to use a fallback
                        # This is a limitation of the consolidated approach
                        $state = "US"
                    }
                }
            }
        } else {
            # For city/state queries, use the name field for city
            $city = $geoData[0].name
            
            # Parse state from display_name or original input
            $displayName = $geoData[0].display_name
            Write-Verbose "Parsing state from display_name: $displayName"
            if ($displayName -match ", ([A-Z]{2}),") {
                $state = $matches[1]
            } elseif ($displayName -match ", ([A-Z]{2})$") {
                $state = $matches[1]
            } else {
                # Fallback: extract state from original input if it contains a comma
                if ($Location -match ", ([A-Z]{2})") {
                    $state = $matches[1]
                } else {
                    $state = "US"
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

$pointsJob = Start-ApiJob -Url $pointsUrl -Headers $headers -JobName "PointsData"

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
Write-Verbose "Points data retrieved successfully"

Remove-Job -Job $pointsJob

# Extract grid information
$office = $pointsData.properties.cwa
$gridX = $pointsData.properties.gridX
$gridY = $pointsData.properties.gridY

# Extract additional location information
$timeZone = $pointsData.properties.timeZone
$radarStation = $pointsData.properties.radarStation

# Extract county information from NWS API response
$county = $null
if ($pointsData.properties.relativeLocation.properties.county) {
    $county = $pointsData.properties.relativeLocation.properties.county
    # Clean up county name if it's a URL (extract just the county name)
    if ($county -match "county/([^/]+)$") {
        $county = $matches[1]
    }
} elseif ($pointsData.properties.county) {
    $county = $pointsData.properties.county
    # Clean up county name if it's a URL (extract just the county name)
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

$forecastJob = Start-ApiJob -Url $forecastUrl -Headers $headers -JobName "ForecastData"
$hourlyJob = Start-ApiJob -Url $hourlyUrl -Headers $headers -JobName "HourlyData"

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
Write-Verbose "Forecast data retrieved successfully"

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
Write-Verbose "Hourly data retrieved successfully"

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

# Extract current weather conditions from the first period of hourly data
# This represents the most recent weather observation
$currentPeriod = $hourlyData.properties.periods[0]
$currentTemp = $currentPeriod.temperature
$currentConditions = $currentPeriod.shortForecast
$currentWind = $currentPeriod.windSpeed
$currentWindDir = $currentPeriod.windDirection
# Use the API update time for the observation timestamp (more accurate than period start time)
$currentTime = $hourlyData.properties.updateTime
$currentHumidity = $currentPeriod.relativeHumidity.value

# Extract dew point if available in API response
$currentDewPoint = $null
if ($currentPeriod.PSObject.Properties['dewpoint'] -and $null -ne $currentPeriod.dewpoint.value) {
    try {
        $currentDewPoint = [double]$currentPeriod.dewpoint.value
        Write-Verbose "Successfully extracted dew point: $currentDewPoint°C"
    }
    catch {
        Write-Verbose "Invalid dew point value: $($currentPeriod.dewpoint.value)"
        $currentDewPoint = $null
    }
}

$currentPrecipProb = $currentPeriod.probabilityOfPrecipitation.value
$currentIcon = $currentPeriod.icon

# --- Temperature Trend Detection ---
$currentTempTrend = $currentPeriod.temperatureTrend

# Fallback: Calculate trend from hourly data if not provided by API
if (-not $currentTempTrend -or $currentTempTrend -eq "") {
    $hourlyPeriods = $hourlyData.properties.periods
    if ($hourlyPeriods.Count -gt 1) {
        $nextHourPeriod = $hourlyPeriods[1]
        $nextHourTemp = $nextHourPeriod.temperature
        
        $tempDiff = [double]$nextHourTemp - [double]$currentTemp
        Write-Verbose "Temperature trend calculation: Current=$currentTemp°F, Next=$nextHourTemp°F, Diff=$tempDiff°F"
        
        if ($tempDiff -ge 0.67) {
            $currentTempTrend = "rising"
        }
        elseif ($tempDiff -le -0.67) {
            $currentTempTrend = "falling"
        }
        else {
            $currentTempTrend = "steady"
        }
        Write-Verbose "Calculated temperature trend: $currentTempTrend"
    } else {
        $currentTempTrend = "steady"
        Write-Verbose "Insufficient hourly data for trend calculation"
    }
}

# Extract wind gust information from wind speed string
$windGust = $null
if ($currentWind -match "(\d+)\s*to\s*(\d+)\s*mph") {
    $windGust = $matches[2]
    $currentWind = "$($matches[1]) mph"
}

# Extract today's detailed forecast (first period in forecast data)
$todayPeriod = $forecastData.properties.periods[0]
$todayForecast = $todayPeriod.detailedForecast

# Extract tomorrow's detailed forecast (second period in forecast data)
$tomorrowPeriod = $forecastData.properties.periods[1]
$tomorrowForecast = $tomorrowPeriod.detailedForecast



# Function: Extract numeric wind speed from wind speed string
# Handles formats like "10 mph", "5 to 15 mph", etc.
function Get-WindSpeed ($windString) {
    if ($windString -match "(\d+)") {
        return [int]$matches[1]
    }
    return 0
}

# Function: Determine if a weather period is during day or night
# Uses NWS API isDaytime property when available, falls back to time-based estimation
function Test-IsDaytime {
    param(
        [object]$Period
    )
    
    # Use the NWS API's isDaytime property if available (most accurate)
    if ($Period.PSObject.Properties['isDaytime']) {
        return $Period.isDaytime
    }
    
    # Fallback: use current local time to estimate day/night (6 AM to 6 PM = daytime)
    $currentHour = (Get-Date).Hour
    return $currentHour -ge 6 -and $currentHour -lt 18
}

# Function: Convert NWS weather icon URLs to appropriate emoji
# Maps NWS icon conditions to emoji with day/night variants for better visual representation
# Prioritizes precipitation-related conditions when present
function Get-WeatherIcon ($iconUrl, $isDaytime = $true, $precipProb = 0) {
    if (-not $iconUrl) { return "" }
    
    # Extract weather condition from NWS icon URL
    if ($iconUrl -match "/([^/]+)\?") {
        $condition = $matches[1]
        
        # Prioritize precipitation-related conditions when present
        # Check for precipitation conditions first (highest priority)
        if ($condition -match "tsra") { return "⛈️" }  # Thunderstorm
        if ($condition -match "rain" -and $precipProb -ge 50) { return "🌧️" }  # Rain (only if >= 50% chance)
        if ($condition -match "snow") { return "❄️" }  # Snow
        if ($condition -match "fzra") { return "🧊" }  # Freezing rain
        
        # Check for other weather conditions
        if ($condition -match "fog") { return "🌫️" }   # Fog
        if ($condition -match "haze") { return "🌫️" }  # Haze
        if ($condition -match "smoke") { return "💨" } # Smoke
        if ($condition -match "dust") { return "💨" }  # Dust
        if ($condition -match "wind") { return "💨" }  # Windy
        
        # Check for cloud conditions (lower priority than precipitation)
        if ($condition -match "ovc") { return "☁️" }   # Overcast
        if ($condition -match "bkn") { return "☁️" }   # Broken clouds
        if ($condition -match "sct") { 
            if ($isDaytime) { return "⛅" } else { return "☁️" }
        }  # Scattered clouds
        if ($condition -match "few") { 
            if ($isDaytime) { return "🌤️" } else { return "🌙" }
        }  # Few clouds
        if ($condition -match "skc") { 
            if ($isDaytime) { return "☀️" } else { return "🌙" }
        }  # Clear
        
        # Check for other common cloud patterns that might not be caught above
        if ($condition -match "cloud") { return "☁️" }  # Generic cloud
        if ($condition -match "shower") { return "☁️" }  # Showers (cloudy with precipitation)
        if ($condition -match "drizzle") { return "☁️" }  # Drizzle (light rain, cloudy)
        
        # Default fallback - use cloud emoji instead of thermometer for unknown conditions
        if ($isDaytime) { return "☁️" } else { return "🌙" }
    }
    
    # Default fallback if URL parsing fails
    if ($isDaytime) { return "☁️" } else { return "🌙" }
}

# Function: Detect if running in Cursor/VS Code terminal
# Used to adjust emoji rendering behavior for different terminal environments
function Test-CursorTerminal {
    return $parentName -match 'Code' -or 
           $parentName -match 'Cursor' -or 
           $env:TERM_PROGRAM -eq 'vscode' -or
           $env:VSCODE_PID -or
           $env:CURSOR_PID
}

# Function: Get emoji display width based on terminal environment
# Different terminals render emojis with different widths (single vs double)
function Get-EmojiWidth {
    param([string]$Emoji)
    
    # In Cursor's terminal, emojis render as single-width characters
    if (Test-CursorTerminal) {
        return 1
    }
    
    # In regular PowerShell, treat all emojis as double-width for consistent alignment
    return 2
}

# Function: Calculate the display width of a string (accounting for emoji width)
# Handles Unicode emoji characters that may have different display widths than regular characters
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

# Function to display current conditions
function Show-CurrentConditions {
    param(
        [string]$City,
        [string]$State,
        [string]$WeatherIcon,
        [string]$CurrentConditions,
        [string]$CurrentTemp,
        [string]$TempColor,
        [string]$CurrentTempTrend,
        [string]$CurrentWind,
        [string]$WindColor,
        [string]$CurrentWindDir,
        [string]$WindGust,
        [string]$CurrentHumidity,
        [string]$CurrentDewPoint,
        [string]$CurrentPrecipProb,
        [string]$CurrentTimeLocal,
        [string]$DefaultColor,
        [string]$AlertColor,
        [string]$TitleColor,
        [string]$InfoColor
    )
    
    Write-Host "*** $city, $state Current Conditions ***" -ForegroundColor $TitleColor
    Write-Host "Currently: $weatherIcon $currentConditions" -ForegroundColor $DefaultColor
    Write-Host "Temperature: $currentTemp°F" -ForegroundColor $TempColor -NoNewline
    if ($currentTempTrend) {
                 $trendIcon = switch ($currentTempTrend) {
             "rising" { "↗️" }
             "falling" { "↘️" }
             "steady" { "→" }
             default { "" }
         }
        Write-Host " $trendIcon " -ForegroundColor $DefaultColor -NoNewline
    }
    Write-Host ""

    Write-Host "Wind: $currentWind $currentWindDir" -ForegroundColor $WindColor -NoNewline
    if ($windGust) {
        Write-Host " (gusts to $windGust mph)" -ForegroundColor $AlertColor -NoNewline
    }
    Write-Host ""

    Write-Host "Humidity: $currentHumidity%" -ForegroundColor $DefaultColor
    
    # Display dew point if available
    if ($null -ne $currentDewPoint) {
        try {
            $dewPointCelsius = [double]$currentDewPoint
            $dewPointF = [math]::Round($dewPointCelsius * 9/5 + 32, 1)
            Write-Host "Dew Point: $dewPointF°F" -ForegroundColor $DefaultColor
            Write-Verbose "Dew point conversion: $dewPointCelsius°C → $dewPointF°F"
        }
        catch {
            Write-Verbose "Error converting dew point: $($_.Exception.Message)"
        }
    }
    
    if ($currentPrecipProb -gt 0) {
        # Color code precipitation probability
        $precipColor = if ($currentPrecipProb -gt $script:HIGH_PRECIP_THRESHOLD) { $AlertColor } elseif ($currentPrecipProb -gt $script:MEDIUM_PRECIP_THRESHOLD) { "Yellow" } else { $DefaultColor }
        Write-Host "Precipitation: $currentPrecipProb% chance" -ForegroundColor $precipColor
    }

    # Safely display API update time with validation
    try {
        if ($currentTimeLocal -and ($currentTimeLocal -is [DateTime] -or $currentTimeLocal -is [string])) {
            if ($currentTimeLocal -is [string]) {
                # If it's a string, try to parse it again
                $parsedTime = [DateTime]::Parse($currentTimeLocal)
                Write-Host "Updated: $($parsedTime.ToString('MM/dd/yyyy HH:mm'))" -ForegroundColor $InfoColor
            } else {
                Write-Host "Updated: $($currentTimeLocal.ToString('MM/dd/yyyy HH:mm'))" -ForegroundColor $InfoColor
            }
        } else {
            Write-Host "Updated: N/A" -ForegroundColor $InfoColor
        }
    }
    catch {
        Write-Host "Updated: N/A" -ForegroundColor $InfoColor
    }
}

# Function to display forecast text
function Show-ForecastText {
    param(
        [string]$Title,
        [string]$ForecastText,
        [string]$TitleColor,
        [string]$DefaultColor
    )
    
    Write-Host ""
    Write-Host "*** $Title ***" -ForegroundColor $TitleColor
    $wrappedForecast = Format-TextWrap -Text $ForecastText -Width $Host.UI.RawUI.WindowSize.Width
    $wrappedForecast | ForEach-Object { Write-Host $_ -ForegroundColor $DefaultColor }
}

# Function to display hourly forecast with scrolling capability
function Show-HourlyForecast {
    param(
        [object]$HourlyData,
        [string]$TitleColor,
        [string]$DefaultColor,
        [string]$AlertColor,
        [int]$MaxHours = $script:MAX_HOURLY_FORECAST_HOURS,
        [int]$StartIndex = 0,
        [bool]$IsInteractive = $false
    )
    
    Write-Host ""
    Write-Host "*** Hourly Forecast ***" -ForegroundColor $TitleColor
    $hourlyPeriods = $hourlyData.properties.periods
    $totalHours = [Math]::Min($hourlyPeriods.Count, 48)  # Limit to 48 hours
    $endIndex = [Math]::Min($StartIndex + $MaxHours, $totalHours)
    
    # Show scroll indicators only in interactive mode
    if ($IsInteractive) {
        if ($StartIndex -gt 0) {
            Write-Host "↑ Previous hours available (Up arrow)" -ForegroundColor Yellow
        }
        if ($endIndex -lt $totalHours) {
            Write-Host "↓ More hours available (Down arrow)" -ForegroundColor Yellow
        }
        Write-Host "Showing hours $($StartIndex + 1)-$endIndex of $totalHours" -ForegroundColor Cyan
        Write-Host ""
    }
    
    $hourCount = 0
    for ($i = $StartIndex; $i -lt $endIndex; $i++) {
        $period = $hourlyPeriods[$i]
        
        $periodTime = [DateTime]::Parse($period.startTime)
        $hourDisplay = $periodTime.ToString("HH:mm")
        $temp = $period.temperature
        $shortForecast = $period.shortForecast
        $wind = $period.windSpeed
        $windDir = $period.windDirection
        $precipProb = $period.probabilityOfPrecipitation.value
        
        # Determine if this period is during day or night using NWS API isDaytime property
        $isPeriodDaytime = Test-IsDaytime $period
        $periodIcon = Get-WeatherIcon $period.icon $isPeriodDaytime $precipProb
        
        # Color code temperature
        $tempColor = if ([int]$temp -lt $script:COLD_TEMP_THRESHOLD -or [int]$temp -gt $script:HOT_TEMP_THRESHOLD) { $AlertColor } else { $DefaultColor }
        
        # Color code precipitation probability
        $precipColor = if ($precipProb -gt $script:HIGH_PRECIP_THRESHOLD) { $AlertColor } elseif ($precipProb -gt $script:MEDIUM_PRECIP_THRESHOLD) { "Yellow" } else { $DefaultColor }
        
        # Build and write the line piece-by-piece for easier colorization
        $timePart = "$hourDisplay "
        $iconPart = "$periodIcon"
        $tempPart = " $temp°F "
        $windPart = "$wind $windDir"
        $precipPart = if ($precipProb -gt 0) { " ($precipProb%)" } else { "" }
        $forecastPart = " - $shortForecast"

        # Calculate padding for alignment
        $targetTempColumn = 8
        $lineBeforeTemp = $timePart + $iconPart
        $currentDisplayWidth = Get-StringDisplayWidth $lineBeforeTemp
        $spacesNeeded = $targetTempColumn - $currentDisplayWidth
        $padding = " " * [Math]::Max(0, $spacesNeeded)

        Write-Host $timePart -ForegroundColor $DefaultColor -NoNewline
        Write-Host $iconPart -ForegroundColor $DefaultColor -NoNewline
        Write-Host $padding -ForegroundColor $DefaultColor -NoNewline
        Write-Host $tempPart -ForegroundColor $tempColor -NoNewline
        Write-Host $windPart -ForegroundColor $DefaultColor -NoNewline
        if ($precipPart) { Write-Host $precipPart -ForegroundColor $precipColor -NoNewline }
        Write-Host $forecastPart -ForegroundColor $DefaultColor
        
        $hourCount++
    }
}

# Function to display 7-day forecast
function Show-SevenDayForecast {
    param(
        [object]$ForecastData,
        [string]$TitleColor,
        [string]$DefaultColor,
        [string]$AlertColor,
        [int]$MaxDays = $script:MAX_DAILY_FORECAST_DAYS
    )
    
    Write-Host ""
    Write-Host "*** 7-Day Forecast Summary ***" -ForegroundColor $TitleColor
    $forecastPeriods = $forecastData.properties.periods
    $dayCount = 0
    $processedDays = @{}

    foreach ($period in $forecastPeriods) {
        if ($dayCount -ge $MaxDays) { break }
        
        $periodTime = [DateTime]::Parse($period.startTime)
        $dayName = $periodTime.ToString("ddd")
        
        # Skip if we've already processed this day
        if ($processedDays.ContainsKey($dayName)) { continue }
        
        $temp = $period.temperature
        $shortForecast = $period.shortForecast
        $precipProb = $period.probabilityOfPrecipitation.value
        
        # Determine if this period is during day or night using NWS API isDaytime property
        $isPeriodDaytime = Test-IsDaytime $period
        $periodIcon = Get-WeatherIcon $period.icon $isPeriodDaytime $precipProb
        
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
        $tempColor = if ([int]$temp -lt $script:COLD_TEMP_THRESHOLD -or [int]$temp -gt $script:HOT_TEMP_THRESHOLD) { $AlertColor } else { $DefaultColor }
        
        # Build the formatted line
        $formattedLine = Format-DailyLine -DayName $dayName -Icon $periodIcon -Temp $temp -NightTemp $nightTemp -Forecast $shortForecast -PrecipProb $precipProb
        
        # Split the line into parts for color coding
        $tempStart = $formattedLine.IndexOf(" H:$temp°F")
        if ($tempStart -lt 0) {
            $tempStart = $formattedLine.IndexOf(" $temp°F")
        }
        
        if ($tempStart -ge 0) {
            # Write everything before temperature
            Write-Host $formattedLine.Substring(0, $tempStart) -ForegroundColor $DefaultColor -NoNewline
            
            # Write temperature with color
            if ($nightTemp) {
                Write-Host " H:$temp°F L:$nightTemp°F" -ForegroundColor $tempColor -NoNewline
            } else {
                Write-Host " $temp°F" -ForegroundColor $tempColor -NoNewline
            }
            
            # Write everything after temperature
            $tempEnd = if ($nightTemp) { " H:$temp°F L:$nightTemp°F".Length } else { " $temp°F".Length }
            $afterTemp = $formattedLine.Substring($tempStart + $tempEnd)
            Write-Host $afterTemp -ForegroundColor $DefaultColor
        } else {
            # Fallback if temperature not found
            Write-Host $formattedLine -ForegroundColor $DefaultColor
        }
        
        $processedDays[$dayName] = $true
        $dayCount++
    }
}

# Function to display weather alerts
function Show-WeatherAlerts {
    param(
        [object]$AlertsData,
        [string]$AlertColor,
        [string]$DefaultColor,
        [string]$InfoColor,
        [bool]$ShowDetails = $true
    )
    
    if ($alertsData -and $alertsData.features.Count -gt 0) {
        Write-Host ""
        Write-Host "*** Active Weather Alerts ***" -ForegroundColor $AlertColor
        foreach ($alert in $alertsData.features) {
            $alertProps = $alert.properties
            $alertEvent = $alertProps.event
            $alertHeadline = $alertProps.headline
            $alertDesc = $alertProps.description
            $alertStart = ([DateTime]::Parse($alertProps.effective)).ToLocalTime()
            $alertEnd = ([DateTime]::Parse($alertProps.expires)).ToLocalTime()
            
            Write-Host "*** $alertEvent ***" -ForegroundColor $AlertColor
            Write-Host "$alertHeadline" -ForegroundColor $DefaultColor
            if ($ShowDetails) {
                $wrappedAlert = Format-TextWrap -Text $alertDesc -Width $Host.UI.RawUI.WindowSize.Width
                $wrappedAlert | ForEach-Object { Write-Host $_ -ForegroundColor $DefaultColor }
            }
            if ($ShowDetails) {
                Write-Host "Effective: $($alertStart.ToString('MM/dd/yyyy HH:mm'))" -ForegroundColor $InfoColor
            }
            Write-Host "Expires: $($alertEnd.ToString('MM/dd/yyyy HH:mm'))" -ForegroundColor $InfoColor
            Write-Host ""
        }
    }
}

# Function to display location information
function Show-LocationInfo {
    param(
        [string]$County,
        [string]$TimeZone,
        [string]$RadarStation,
        [double]$Lat,
        [double]$Lon,
        [string]$TitleColor,
        [string]$DefaultColor
    )
    
    Write-Host ""
    Write-Host "*** Location Information ***" -ForegroundColor $TitleColor
    if ($county) {
        Write-Host "County: $county" -ForegroundColor $DefaultColor
    }
    Write-Host "Time Zone: $timeZone" -ForegroundColor $DefaultColor
    Write-Host "Radar Station: $radarStation" -ForegroundColor $DefaultColor
    Write-Host "Coordinates: $lat, $lon" -ForegroundColor $DefaultColor

    Write-Host ""
    Write-Host "https://forecast.weather.gov/MapClick.php?lat=$lat&lon=$lon" -ForegroundColor Cyan
}

# Function to display interactive mode controls
function Show-InteractiveControls {
    param([bool]$IsHourlyMode = $false)
    
    Write-Host ""
    if ($IsHourlyMode) {
        Write-Host "Scroll[" -ForegroundColor White -NoNewline; Write-Host "↑↓" -ForegroundColor Cyan -NoNewline; Write-Host "], 7-Day[" -ForegroundColor White -NoNewline; Write-Host "D" -ForegroundColor Cyan -NoNewline; Write-Host "], Terse[" -ForegroundColor White -NoNewline; Write-Host "T" -ForegroundColor Cyan -NoNewline; Write-Host "], Full[" -ForegroundColor White -NoNewline; Write-Host "F" -ForegroundColor Cyan -NoNewline; Write-Host "], Exit[" -ForegroundColor White -NoNewline; Write-Host "Enter" -ForegroundColor Cyan -NoNewline; Write-Host "]" -ForegroundColor White
    } else {
        Write-Host "Hourly[" -ForegroundColor White -NoNewline; Write-Host "H" -ForegroundColor Cyan -NoNewline; Write-Host "], 7-Day[" -ForegroundColor White -NoNewline; Write-Host "D" -ForegroundColor Cyan -NoNewline; Write-Host "], Terse[" -ForegroundColor White -NoNewline; Write-Host "T" -ForegroundColor Cyan -NoNewline; Write-Host "], Full[" -ForegroundColor White -NoNewline; Write-Host "F" -ForegroundColor Cyan -NoNewline; Write-Host "], Exit[" -ForegroundColor White -NoNewline; Write-Host "Enter" -ForegroundColor Cyan -NoNewline; Write-Host "]" -ForegroundColor White
    }
}

# Function to display full weather report
function Show-FullWeatherReport {
    param(
        [string]$City,
        [string]$State,
        [string]$WeatherIcon,
        [string]$CurrentConditions,
        [string]$CurrentTemp,
        [string]$TempColor,
        [string]$CurrentTempTrend,
        [string]$CurrentWind,
        [string]$WindColor,
        [string]$CurrentWindDir,
        [string]$WindGust,
        [string]$CurrentHumidity,
        [string]$CurrentDewPoint,
        [string]$CurrentPrecipProb,
        [string]$CurrentTimeLocal,
        [string]$TodayForecast,
        [string]$TomorrowForecast,
        [object]$HourlyData,
        [object]$ForecastData,
        [object]$AlertsData,
        [string]$County,
        [string]$TimeZone,
        [string]$RadarStation,
        [double]$Lat,
        [double]$Lon,
        [string]$DefaultColor,
        [string]$AlertColor,
        [string]$TitleColor,
        [string]$InfoColor,
        [bool]$ShowCurrentConditions = $true,
        [bool]$ShowTodayForecast = $true,
        [bool]$ShowTomorrowForecast = $true,
        [bool]$ShowHourlyForecast = $true,
        [bool]$ShowSevenDayForecast = $true,
        [bool]$ShowAlerts = $true,
        [bool]$ShowAlertDetails = $true,
        [bool]$ShowLocationInfo = $true
    )
    
    if ($ShowCurrentConditions) {
        Show-CurrentConditions -City $City -State $State -WeatherIcon $WeatherIcon -CurrentConditions $CurrentConditions -CurrentTemp $CurrentTemp -TempColor $TempColor -CurrentTempTrend $CurrentTempTrend -CurrentWind $CurrentWind -WindColor $WindColor -CurrentWindDir $CurrentWindDir -WindGust $WindGust -CurrentHumidity $CurrentHumidity -CurrentDewPoint $CurrentDewPoint -CurrentPrecipProb $CurrentPrecipProb -CurrentTimeLocal $CurrentTimeLocal -DefaultColor $DefaultColor -AlertColor $AlertColor -TitleColor $TitleColor -InfoColor $InfoColor
    }

    if ($ShowTodayForecast) {
        Show-ForecastText -Title "Today's Forecast" -ForecastText $TodayForecast -TitleColor $TitleColor -DefaultColor $DefaultColor
    }

    if ($ShowTomorrowForecast) {
        Show-ForecastText -Title "Tomorrow's Forecast" -ForecastText $TomorrowForecast -TitleColor $TitleColor -DefaultColor $DefaultColor
    }

    if ($ShowHourlyForecast) {
        Show-HourlyForecast -HourlyData $HourlyData -TitleColor $TitleColor -DefaultColor $DefaultColor -AlertColor $AlertColor -IsInteractive $false
    }

    if ($ShowSevenDayForecast) {
        Show-SevenDayForecast -ForecastData $ForecastData -TitleColor $TitleColor -DefaultColor $DefaultColor -AlertColor $AlertColor
    }

    if ($ShowAlerts) {
        Show-WeatherAlerts -AlertsData $AlertsData -AlertColor $AlertColor -DefaultColor $DefaultColor -InfoColor $InfoColor -ShowDetails $ShowAlertDetails
    }



    if ($ShowLocationInfo) {
        Show-LocationInfo -County $County -TimeZone $TimeZone -RadarStation $RadarStation -Lat $Lat -Lon $Lon -TitleColor $TitleColor -DefaultColor $DefaultColor
    }
}

# Function to map rain percentage to sparkline character and color
function Get-RainSparkline {
    param([int]$RainPercent)
    
    if ($RainPercent -eq 0) { return @{Char=" "; Color="White"} }
    elseif ($RainPercent -le 10) { return @{Char="▁"; Color="Cyan"} }
    elseif ($RainPercent -le 33) { return @{Char="▂"; Color="Cyan"} }
    elseif ($RainPercent -le 40) { return @{Char="▃"; Color="Yellow"} }
    elseif ($RainPercent -le 66) { return @{Char="▅"; Color="Yellow"} }
    elseif ($RainPercent -le 80) { return @{Char="▆"; Color="Red"} }
    else { return @{Char="▇"; Color="Red"} }
}

# Function to map wind direction to glyph and get wind speed color
function Get-WindGlyph {
    param(
        [string]$WindDirection,
        [int]$WindSpeed
    )
    
    # Wind direction glyphs (N, NE, E, SE, S, SW, W, NW)
    $directionMap = @{
        "N" = 0; "NNE" = 0; "NNW" = 0
        "NE" = 1; "ENE" = 1
        "E" = 2; "ESE" = 2
        "SE" = 3; "SSE" = 3
        "S" = 4; "SSW" = 4
        "SW" = 5; "WSW" = 5
        "W" = 6; "WNW" = 6
        "NW" = 7
    }
    
    # Get direction index (default to 0 for N if not found)
    $dirIndex = if ($directionMap.ContainsKey($WindDirection)) { $directionMap[$WindDirection] } else { 0 }
    
    # Choose glyph set based on wind speed
    if ($WindSpeed -lt 7) {
        $glyphs = @("△", "◹", "▷", "◿", "▽", "◺", "◁", "◸")
    } else {
        $glyphs = @("▲", "◥", "▶", "◢", "▼", "◣", "◀", "◤")
    }
    
    $glyph = $glyphs[$dirIndex]
    
    # Get color based on wind speed
    $color = if ($WindSpeed -le 5) { "White" }
            elseif ($WindSpeed -le 9) { "Yellow" }
            elseif ($WindSpeed -le 14) { "Red" }
            else { "Magenta" }
    
    return @{Char=$glyph; Color=$color}
}

# Function to display rain likelihood forecast with sparklines
function Show-RainForecast {
    param(
        [object]$HourlyData,
        [string]$TitleColor,
        [string]$DefaultColor
    )
    
    Write-Host ""
    Write-Host "       *** Rain Outlook ***" -ForegroundColor $TitleColor
    
    $hourlyPeriods = $hourlyData.properties.periods
    $totalHours = [Math]::Min($hourlyPeriods.Count, 96)  # Use 96 hours for rain mode
    
    # Group periods by day
    $daysData = @{}
    foreach ($period in $hourlyPeriods[0..($totalHours-1)]) {
        $periodTime = [DateTime]::Parse($period.startTime)
        $dayKey = $periodTime.ToString("yyyy-MM-dd")
        $hour = $periodTime.Hour
        
        if (-not $daysData.ContainsKey($dayKey)) {
            $daysData[$dayKey] = @{}
        }
        $daysData[$dayKey][$hour] = $period
    }
    
    # Get sorted day keys
    $sortedDays = $daysData.Keys | Sort-Object
    $dayCount = 0
    
    foreach ($dayKey in $sortedDays) {
        if ($dayCount -ge 5) { break }  # Limit to 5 days
        
        $periodTime = [DateTime]::Parse($dayKey)
        $dayName = $periodTime.ToString("ddd")
        $dayData = $daysData[$dayKey]
        
        # Find the highest rain percentage for this day
        $maxRainPercent = 0
        foreach ($hour in $dayData.Keys) {
            $period = $dayData[$hour]
            $rainPercent = if ($period.probabilityOfPrecipitation.value) { $period.probabilityOfPrecipitation.value } else { 0 }
            if ($rainPercent -gt $maxRainPercent) {
                $maxRainPercent = $rainPercent
            }
        }
        
        # Cap at 99% to prevent alignment issues
        if ($maxRainPercent -eq 100) {
            $maxRainPercent = 99
        }
        
        # Get color for the highest percentage
        $maxRainColor = if ($maxRainPercent -eq 0) { "White" }
                       elseif ($maxRainPercent -le 33) { "Cyan" }
                       elseif ($maxRainPercent -le 66) { "Yellow" }
                       else { "Red" }
        
        # Write day name and max percentage with color coding and proper padding
        Write-Host "$dayName " -ForegroundColor White -NoNewline
        $paddedPercent = if ($maxRainPercent -lt 10) { " $maxRainPercent%" } else { "$maxRainPercent%" }
        Write-Host "$paddedPercent " -ForegroundColor $maxRainColor -NoNewline
        
        # Build sparkline for this day (24 hours: 00:00 to 23:00)
        for ($hour = 0; $hour -lt 24; $hour++) {
            if ($dayData.ContainsKey($hour)) {
                $period = $dayData[$hour]
                $rainPercent = if ($period.probabilityOfPrecipitation.value) { $period.probabilityOfPrecipitation.value } else { 0 }
                $sparklineData = Get-RainSparkline $rainPercent
                Write-Host $sparklineData.Char -ForegroundColor $sparklineData.Color -NoNewline
            } else {
                Write-Host " " -ForegroundColor $DefaultColor -NoNewline  # Blank for no data
            }
        }
        Write-Host ""  # New line after each day
        $dayCount++
    }
}

# Function to display wind outlook forecast with direction glyphs
function Show-WindForecast {
    param(
        [object]$HourlyData,
        [string]$TitleColor,
        [string]$DefaultColor
    )
    
    Write-Host ""
    Write-Host "       *** Wind Outlook ***" -ForegroundColor Green
    
    $hourlyPeriods = $hourlyData.properties.periods
    $totalHours = [Math]::Min($hourlyPeriods.Count, 96)  # Use 96 hours for wind mode
    
    # Group periods by day
    $daysData = @{}
    foreach ($period in $hourlyPeriods[0..($totalHours-1)]) {
        $periodTime = [DateTime]::Parse($period.startTime)
        $dayKey = $periodTime.ToString("yyyy-MM-dd")
        $hour = $periodTime.Hour
        
        if (-not $daysData.ContainsKey($dayKey)) {
            $daysData[$dayKey] = @{}
        }
        $daysData[$dayKey][$hour] = $period
    }
    
    # Get sorted day keys
    $sortedDays = $daysData.Keys | Sort-Object
    $dayCount = 0
    
    foreach ($dayKey in $sortedDays) {
        if ($dayCount -ge 5) { break }  # Limit to 5 days
        
        $periodTime = [DateTime]::Parse($dayKey)
        $dayName = $periodTime.ToString("ddd")
        $dayData = $daysData[$dayKey]
        
        # Find the highest wind speed for this day
        $maxWindSpeed = 0
        foreach ($hour in $dayData.Keys) {
            $period = $dayData[$hour]
            $windSpeed = Get-WindSpeed $period.windSpeed
            if ($windSpeed -gt $maxWindSpeed) {
                $maxWindSpeed = $windSpeed
            }
        }
        
        # Get color for the highest wind speed
        $maxWindColor = if ($maxWindSpeed -le 5) { "White" }
                       elseif ($maxWindSpeed -le 9) { "Yellow" }
                       elseif ($maxWindSpeed -le 14) { "Red" }
                       else { "Magenta" }
        
        # Write day name and max wind speed with color coding and proper padding
        Write-Host "$dayName " -ForegroundColor White -NoNewline
        $paddedSpeed = if ($maxWindSpeed -lt 10) { " $maxWindSpeed" } else { "$maxWindSpeed" }
        Write-Host "${paddedSpeed}mph " -ForegroundColor $maxWindColor -NoNewline
        
        # Build wind glyphs for this day (24 hours: 00:00 to 23:00)
        for ($hour = 0; $hour -lt 24; $hour++) {
            if ($dayData.ContainsKey($hour)) {
                $period = $dayData[$hour]
                $windSpeed = Get-WindSpeed $period.windSpeed
                $windDirection = $period.windDirection
                $windGlyphData = Get-WindGlyph $windDirection $windSpeed
                Write-Host $windGlyphData.Char -ForegroundColor $windGlyphData.Color -NoNewline
            } else {
                Write-Host " " -ForegroundColor $DefaultColor -NoNewline  # Blank for no data
            }
        }
        Write-Host ""  # New line after each day
        $dayCount++
    }
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

# Parse API update time (API provides local time for the location)
$currentTimeLocal = $null
Write-Verbose "Raw update time from API: $currentTime"
try {
    if ($currentTime -and $currentTime -ne "") {
        # Parse the time from the API (this is already local time for the location)
        $currentTimeLocal = [DateTime]::Parse($currentTime)
        Write-Verbose "Successfully parsed update time: $currentTimeLocal"
    } else {
        Write-Verbose "Update time is null or empty"
    }
}
catch {
    Write-Verbose "Error parsing API update time: $currentTime - $($_.Exception.Message)"
    $currentTimeLocal = $null
}

# Define color scheme for weather display
$defaultColor = "DarkCyan"
$alertColor = "Red"
$titleColor = "Green"
$infoColor = "Blue"

# Apply color coding based on weather conditions
# Temperature: Red if too cold (<33°F) or too hot (>89°F)
if ([int]$currentTemp -lt $script:COLD_TEMP_THRESHOLD -or [int]$currentTemp -gt $script:HOT_TEMP_THRESHOLD) {
    $tempColor = $alertColor
} else {
    $tempColor = $defaultColor
}

# Wind: Red if wind speed is high (=16 mph)
if ($windSpeed -ge $script:WIND_ALERT_THRESHOLD) {
    $windColor = $alertColor
} else {
    $windColor = $defaultColor
}

if ($VerbosePreference -ne 'Continue') {
    Clear-Host
}

# Determine which sections to display based on command-line options
# Default: Show all sections (full weather report)
$showCurrentConditions = $true
$showTodayForecast = $true
$showTomorrowForecast = $true
$showHourlyForecast = $true
$showSevenDayForecast = $true
$showAlerts = $true
$showAlertDetails = $true
$showLocationInfo = $true

if ($Terse.IsPresent) {
    # Terse mode: Show only current conditions and today's forecast
    $showTomorrowForecast = $false
    $showHourlyForecast = $false
    $showSevenDayForecast = $false
    $showAlertDetails = $false
    $showLocationInfo = $false
}
elseif ($Hourly.IsPresent) {
    # Hourly mode: Show only the hourly forecast (up to $($script:MAX_HOURLY_FORECAST_HOURS) hours)
    $showCurrentConditions = $false
    $showTodayForecast = $false
    $showTomorrowForecast = $false
    $showSevenDayForecast = $false
    $showAlerts = $false
    $showLocationInfo = $false
}
elseif ($Daily.IsPresent) {
    # $($script:MAX_DAILY_FORECAST_DAYS)-day mode: Show only the $($script:MAX_DAILY_FORECAST_DAYS)-day forecast summary
    $showCurrentConditions = $false
    $showTodayForecast = $false
    $showTomorrowForecast = $false
    $showHourlyForecast = $false
    $showAlerts = $false
    $showLocationInfo = $false
}
elseif ($Rain.IsPresent) {
    # Rain mode: Show only rain likelihood forecast with sparklines, then exit
    $showCurrentConditions = $false
    $showTodayForecast = $false
    $showTomorrowForecast = $false
    $showHourlyForecast = $false
    $showSevenDayForecast = $false
    $showAlerts = $false
    $showLocationInfo = $false
    $NoInteractive = $true  # Rain mode implies -x (no interactive mode)
}
elseif ($Wind.IsPresent) {
    # Wind mode: Show only wind outlook forecast with direction glyphs, then exit
    $showCurrentConditions = $false
    $showTodayForecast = $false
    $showTomorrowForecast = $false
    $showHourlyForecast = $false
    $showSevenDayForecast = $false
    $showAlerts = $false
    $showLocationInfo = $false
    $NoInteractive = $true  # Wind mode implies -x (no interactive mode)
}

# Determine if it's currently day or night using NWS API isDaytime property
$isCurrentlyDaytime = Test-IsDaytime $currentPeriod

# Output the results.
$weatherIcon = Get-WeatherIcon $currentIcon $isCurrentlyDaytime $currentPrecipProb

# Display the weather report using the refactored function
if ($Rain.IsPresent) {
    # Rain mode: Show only rain likelihood forecast with sparklines
    Show-RainForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor
} elseif ($Wind.IsPresent) {
    # Wind mode: Show only wind outlook forecast with direction glyphs
    Show-WindForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor
} else {
    Show-FullWeatherReport -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $currentTimeLocal -TodayForecast $todayForecast -TomorrowForecast $tomorrowForecast -HourlyData $hourlyData -ForecastData $forecastData -AlertsData $alertsData -County $county -TimeZone $timeZone -RadarStation $radarStation -Lat $lat -Lon $lon -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor -ShowCurrentConditions $showCurrentConditions -ShowTodayForecast $showTodayForecast -ShowTomorrowForecast $showTomorrowForecast -ShowHourlyForecast $showHourlyForecast -ShowSevenDayForecast $showSevenDayForecast -ShowAlerts $showAlerts -ShowAlertDetails $showAlertDetails -ShowLocationInfo $showLocationInfo
}

# Detect if we're in an interactive environment that supports ReadKey
$isInteractiveEnvironment = $false

if ($parentName -match '^(WindowsTerminal.exe|PowerShell|cmd)') {
    $isInteractiveEnvironment = $true
}
elseif ($env:SSH_CONNECTION -and $Host.UI.RawUI.WindowSize.WindowSize.Width -gt 0) {
    $isInteractiveEnvironment = $true
}
elseif ($Host.UI.RawUI.WindowSize.Width -gt 0 -and $Host.UI.RawUI.WindowSize.Height -gt 0) {
    $isInteractiveEnvironment = $true
}

# Set initial mode based on command line flags
$initialHourlyMode = $Hourly.IsPresent

if ($isInteractiveEnvironment -and -not $NoInteractive.IsPresent) {
    Write-Verbose "Parent:$parentName - Interactive environment detected"
    
    # Interactive mode variables
    $isHourlyMode = $initialHourlyMode
    $hourlyScrollIndex = 0
    $totalHourlyPeriods = [Math]::Min($hourlyData.properties.periods.Count, 48)  # Limit to 48 hours
    
    # If starting in hourly mode, show hourly forecast first
    if ($isHourlyMode) {
        Clear-Host
        Show-HourlyForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true
        Show-InteractiveControls -IsHourlyMode $true
    } else {
        # Interactive mode: Listen for keyboard input to switch between display modes
        Show-InteractiveControls
    }
    
    while ($true) {
        try {
            $key = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
            
            # Handle keyboard input for interactive mode
            switch ($key.VirtualKeyCode) {
                72 { # H key - Switch to hourly forecast only
                    Clear-Host
                    $isHourlyMode = $true
                    $hourlyScrollIndex = 0  # Reset to first 12 hours
                    Show-HourlyForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true
                    Show-InteractiveControls -IsHourlyMode $true
                }
                68 { # D key - Switch to 7-day forecast only
                    Clear-Host
                    $isHourlyMode = $false
                    Show-SevenDayForecast -ForecastData $forecastData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor
                    Show-InteractiveControls
                }
                84 { # T key - Switch to terse mode (current + today + alerts)
                    Clear-Host
                    $isHourlyMode = $false
                    Show-CurrentConditions -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $currentTimeLocal -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor
                    Show-ForecastText -Title "Today's Forecast" -ForecastText $todayForecast -TitleColor $titleColor -DefaultColor $defaultColor
                    Show-WeatherAlerts -AlertsData $alertsData -AlertColor $alertColor -DefaultColor $defaultColor -InfoColor $infoColor -ShowDetails $false
                    Show-InteractiveControls
                }
                70 { # F key - Switch to full weather report
                    Clear-Host
                    $isHourlyMode = $false
                    Show-FullWeatherReport -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $currentTimeLocal -TodayForecast $todayForecast -TomorrowForecast $tomorrowForecast -HourlyData $hourlyData -ForecastData $forecastData -AlertsData $alertsData -County $county -TimeZone $timeZone -RadarStation $radarStation -Lat $lat -Lon $lon -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor -ShowCurrentConditions $true -ShowTodayForecast $true -ShowTomorrowForecast $true -ShowHourlyForecast $true -ShowSevenDayForecast $true -ShowAlerts $true -ShowAlertDetails $true -ShowLocationInfo $true
                    Show-InteractiveControls
                }
                38 { # Up arrow - Scroll up in hourly mode
                    if ($isHourlyMode) {
                        $newIndex = $hourlyScrollIndex - $script:MAX_HOURLY_FORECAST_HOURS
                        if ($newIndex -ge 0) {
                            $hourlyScrollIndex = $newIndex
                            Clear-Host
                            Show-HourlyForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true
                            Show-InteractiveControls -IsHourlyMode $true
                        }
                    }
                }
                40 { # Down arrow - Scroll down in hourly mode
                    if ($isHourlyMode) {
                        $newIndex = $hourlyScrollIndex + $script:MAX_HOURLY_FORECAST_HOURS
                        if ($newIndex -lt $totalHourlyPeriods) {
                            $hourlyScrollIndex = $newIndex
                            Clear-Host
                            Show-HourlyForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true
                            Show-InteractiveControls -IsHourlyMode $true
                        }
                    }
                }
                13 { # Enter key - Exit interactive mode
                    Write-Host "Exiting..." -ForegroundColor Yellow
                    return
                }
                28 { # NumPad Enter key - Exit interactive mode
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
