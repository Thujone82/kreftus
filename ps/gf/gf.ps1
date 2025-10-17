<#
.SYNOPSIS
    A PowerShell script to retrieve and display detailed weather information for a specified location.
    
.DESCRIPTION
    This script accepts a US zip code or a "City, State" string as input. It uses the
    National Weather Service API to retrieve and display detailed weather information.

    The script uses three free APIs, none of which require API keys:
    - OpenStreetMap Nominatim API: For geocoding (converting locations to coordinates)
    - ip-api.com: For IP-based geolocation when no location is specified
    - National Weather Service API: For weather data, forecasts, and alerts

    When 'here' is specified, the script automatically detects your location using 
    ip-api.com to determine coordinates based on your public IP address. For specified 
    locations, OpenStreetMap Nominatim API is used to convert the location to coordinates, 
    then the National Weather Service API fetches current weather, daily forecasts, and alerts.
    Sunrise and sunset times are calculated astronomically using NOAA algorithms.
    
.PARAMETER Location
    The location for which to retrieve weather. Can be a 5-digit US zip code or a "City, State" string, or 'here'.
    If omitted, the script will prompt you for it.
    
.PARAMETER Help
    Displays usage information for this script.
    
.EXAMPLE
    .\gf.ps1 97219 -Verbose

.EXAMPLE
    .\gf.ps1 "Portland, OR" -Verbose

.NOTES
    This script uses three free APIs, none of which require API keys:
    - National Weather Service API (weather.gov) - Weather data for US locations
    - OpenStreetMap Nominatim API (nominatim.openstreetmap.org) - Geocoding for US locations
    - ip-api.com - IP-based geolocation for automatic location detection
    
    The weather data is limited to locations within the United States due to the 
    National Weather Service API's coverage area.

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
    [switch]$Wind,

    [Alias('u')]
    [switch]$NoAutoUpdate
)

# --- Helper Functions ---

# Force TLS 1.2.
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

# Global error handler for 503 Server Unavailable errors
$ErrorActionPreference = "Stop"
trap {
    if ($_.Exception.Message -match "503" -or $_.Exception.Message -match "Service Unavailable") {
        Write-Host "The server is not currently available, try again later." -ForegroundColor Red
        exit 1
    } else {
        Write-Host "An error occurred: $($_.Exception.Message)" -ForegroundColor Red
        exit 1
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
    Write-Host "Usage: .\gf.ps1 [ZipCode | `"City, State`" | here] [Options] [-Verbose]" -ForegroundColor Green
    Write-Host " • Provide a 5-digit zipcode or a City, State (e.g., 'Portland, OR')." -ForegroundColor Cyan
    Write-Host " • Use 'here' to automatically detect your location based on IP address." -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Options:" -ForegroundColor Blue
    Write-Host "  -t, -Terse    Show only current conditions and today's forecast" -ForegroundColor Cyan
    Write-Host "  -h, -Hourly   Show only the hourly forecast (up to $($script:MAX_HOURLY_FORECAST_HOURS) hours)" -ForegroundColor Cyan
    Write-Host "  -d, -Daily    Show only the $($script:MAX_DAILY_FORECAST_DAYS)-day forecast summary" -ForegroundColor Cyan
    Write-Host "  -r, -Rain     Show rain likelihood forecast with sparklines" -ForegroundColor Cyan
    Write-Host "                • 96-hour visual sparklines with color-coded intensity" -ForegroundColor Gray
    Write-Host "                • White (0%), White (1-10%), Cyan (11-33%), Green (34-40%), Yellow (41-80%), Red (81%+)" -ForegroundColor Gray
    Write-Host "  -w, -Wind     Show wind outlook forecast with direction glyphs" -ForegroundColor Cyan
    Write-Host "                • 96-hour directional glyphs with color-coded wind speed" -ForegroundColor Gray
    Write-Host "                • White (≤5mph), Yellow (6-9mph), Red (10-14mph), Magenta (15mph+)" -ForegroundColor Gray
    Write-Host "                • Peak wind hours highlighted with inverted colors" -ForegroundColor Gray
    Write-Host "  -u, -NoAutoUpdate Start with automatic updates disabled" -ForegroundColor Cyan
    Write-Host "  -x, -NoInteractive Exit immediately (no interactive mode)" -ForegroundColor Cyan
    Write-Host ""
         Write-Host "Interactive Mode:" -ForegroundColor Blue
     Write-Host "  When run interactively (not from terminal), the script enters interactive mode." -ForegroundColor Cyan
     Write-Host "  Use keyboard shortcuts to switch between display modes:" -ForegroundColor Cyan
     Write-Host "    [H] - Switch to hourly forecast only (with scrolling)" -ForegroundColor Cyan
     Write-Host "    [D] - Switch to $($script:MAX_DAILY_FORECAST_DAYS)-day forecast only" -ForegroundColor Cyan
     Write-Host "    [T] - Switch to terse mode (current + today)" -ForegroundColor Cyan
     Write-Host "    [R] - Switch to rain forecast mode (sparklines)" -ForegroundColor Cyan
     Write-Host "    [W] - Switch to wind forecast mode (direction glyphs)" -ForegroundColor Cyan
    Write-Host "    [G] - Refresh weather data (auto-refreshes every 10 minutes)" -ForegroundColor Cyan
    Write-Host "    [U] - Toggle automatic updates on/off" -ForegroundColor Cyan
    Write-Host "    [F] - Return to full display" -ForegroundColor Cyan
    Write-Host "    [Enter] or [Esc] - Exit the script" -ForegroundColor Cyan
     Write-Host "  In hourly mode, use [↑] and [↓] arrows to scroll through all 48 hours" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "This script retrieves weather info from National Weather Service API (geocoding via OpenStreetMap) and outputs:" -ForegroundColor Blue
    Write-Host " • Location (City, State)" -ForegroundColor Cyan
    Write-Host " • Current Conditions" -ForegroundColor Cyan
    Write-Host " • Temperature with forecast range (red if <33°F or >89°F)" -ForegroundColor Cyan
    Write-Host " • Humidity" -ForegroundColor Cyan
    Write-Host " • Wind (with gust if available; red if wind speed >=16 mph)" -ForegroundColor Cyan
    Write-Host " • Sunrise and Sunset times (calculated astronomically)" -ForegroundColor Cyan
    Write-Host " • Detailed Forecast" -ForegroundColor Cyan
    Write-Host " • Weather Alerts" -ForegroundColor Cyan
    Write-Host " • Observation timestamp" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Blue
    Write-Host "  .\gf.ps1 97219 -Verbose" -ForegroundColor Cyan
    Write-Host "  .\gf.ps1 `"Portland, OR`" -Verbose" -ForegroundColor Cyan
    Write-Host "  .\gf.ps1 here -Verbose" -ForegroundColor Cyan
    Write-Host "  .\gf.ps1 97219 -t" -ForegroundColor Cyan
    Write-Host "  .\gf.ps1 here -t" -ForegroundColor Cyan
    Write-Host "  .\gf.ps1 `"Portland, OR`" -h" -ForegroundColor Cyan
    Write-Host "  .\gf.ps1 97219 -d For Daily Forecast" -ForegroundColor Cyan
    Write-Host "  .\gf.ps1 97219 -h -x For Hourly Forecast and Exit" -ForegroundColor Cyan
    Write-Host "  .\gf.ps1 97219 -r For Rain Outlook" -ForegroundColor Cyan
    Write-Host "  .\gf.ps1 97219 -w For Wind Outlook" -ForegroundColor Cyan
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

# Function to detect current location using IP address
function Get-CurrentLocation {
    try {
        $ipApiUrl = "http://ip-api.com/json/"
        Write-Verbose "GET: $ipApiUrl"
        $response = Invoke-RestMethod -Uri $ipApiUrl -Method Get -ErrorAction Stop
        
        if ($response.status -eq "success") {
            return @{
                Lat = [double]$response.lat
                Lon = [double]$response.lon
                City = $response.city
                State = $response.regionName
            }
        } else {
            throw "Failed to detect location: $($response.message)"
        }
    } catch {
        throw "Unable to detect your location automatically: $($_.Exception.Message)"
    }
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

# Function to refresh weather data
function Update-WeatherData {
    param(
        [string]$Lat,
        [string]$Lon,
        [hashtable]$Headers
    )
    
    try {
        Write-Host "Refreshing weather data..." -ForegroundColor Yellow
        
        # Re-fetch points data
        $pointsUrl = "https://api.weather.gov/points/$lat,$lon"
        $pointsJob = Start-ApiJob -Url $pointsUrl -Headers $headers -JobName "PointsData"
        Wait-Job -Job $pointsJob | Out-Null
        
        if ($pointsJob.State -ne 'Completed') { 
            throw "Points data job failed: $($pointsJob | Receive-Job)"
        }
        
        $pointsJson = $pointsJob | Receive-Job
        if ([string]::IsNullOrWhiteSpace($pointsJson)) { 
            throw "Empty response from points API"
        }
        
        $pointsData = $pointsJson | ConvertFrom-Json
        Remove-Job -Job $pointsJob
        
        # Extract grid information
        $office = $pointsData.properties.cwa
        $gridX = $pointsData.properties.gridX
        $gridY = $pointsData.properties.gridY
        
        # Re-fetch forecast and hourly data
        $forecastUrl = "https://api.weather.gov/gridpoints/$office/$gridX,$gridY/forecast"
        $hourlyUrl = "https://api.weather.gov/gridpoints/$office/$gridX,$gridY/forecast/hourly"
        
        $forecastJob = Start-ApiJob -Url $forecastUrl -Headers $headers -JobName "ForecastData"
        $hourlyJob = Start-ApiJob -Url $hourlyUrl -Headers $headers -JobName "HourlyData"
        
        $jobsToWaitFor = @($forecastJob, $hourlyJob)
        Wait-Job -Job $jobsToWaitFor | Out-Null
        
        # Check forecast job
        if ($forecastJob.State -ne 'Completed') { 
            throw "Forecast data job failed: $($forecastJob | Receive-Job)"
        }
        
        $forecastJson = $forecastJob | Receive-Job
        if ([string]::IsNullOrWhiteSpace($forecastJson)) { 
            throw "Empty response from forecast API"
        }
        
        $forecastData = $forecastJson | ConvertFrom-Json
        
        # Check hourly job
        if ($hourlyJob.State -ne 'Completed') { 
            throw "Hourly data job failed: $($hourlyJob | Receive-Job)"
        }
        
        $hourlyJson = $hourlyJob | Receive-Job
        if ([string]::IsNullOrWhiteSpace($hourlyJson)) { 
            throw "Empty response from hourly API"
        }
        
        $hourlyData = $hourlyJson | ConvertFrom-Json
        Remove-Job -Job $jobsToWaitFor
        
        # Re-fetch alerts
        $alertsData = $null
        try {
            $alertsUrl = "https://api.weather.gov/alerts/active?point=$lat,$lon"
            Write-Verbose "GET: $alertsUrl"
            $alertsResponse = Invoke-RestMethod -Uri $alertsUrl -Method Get -Headers $headers -ErrorAction Stop
            $alertsData = $alertsResponse
        }
        catch {
            # Alerts are optional, continue without them
        }
        
        # Update global variables
        $script:forecastData = $forecastData
        $script:hourlyData = $hourlyData
        $script:alertsData = $alertsData
        
        # Update current conditions
        $script:currentPeriod = $hourlyData.properties.periods[0]
        $script:currentTemp = $currentPeriod.temperature
        $script:currentConditions = $currentPeriod.shortForecast
        $script:currentWind = $currentPeriod.windSpeed
        $script:currentWindDir = $currentPeriod.windDirection
        $script:currentTime = $script:dataFetchTime
        $script:currentTimeLocal = $script:dataFetchTime
        $script:currentHumidity = $currentPeriod.relativeHumidity.value
        $script:currentPrecipProb = $currentPeriod.probabilityOfPrecipitation.value
        $script:currentIcon = $currentPeriod.icon
        
        # Update forecast data
        $script:todayPeriod = $forecastData.properties.periods[0]
        $script:todayForecast = $todayPeriod.detailedForecast
        $script:todayPeriodName = $todayPeriod.name
        
        $script:tomorrowPeriod = $forecastData.properties.periods[1]
        $script:tomorrowForecast = $tomorrowPeriod.detailedForecast
        $script:tomorrowPeriodName = $tomorrowPeriod.name
        
        # Update fetch time
        $script:dataFetchTime = Get-Date
        
        Write-Host "Data refreshed successfully" -ForegroundColor Green
        return $true
    }
    catch {
        Write-Host "Failed to refresh data: $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
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

        # Check if user wants automatic location detection
        if ($Location -ieq "here") {
            Write-Verbose "Detecting location automatically..."
            $locationData = Get-CurrentLocation
            $lat = $locationData.Lat
            $lon = $locationData.Lon
            $city = $locationData.City
            $state = $locationData.State
            Write-Verbose "Detected location: $city, $state (Lat: $lat, Lon: $lon)"
        } else {
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
Write-Verbose "GET: $pointsUrl"

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

# Extract forecast URLs directly from Points API response
# This ensures we use the correct office code (e.g., AER vs AFC)
$forecastUrl = $pointsData.properties.forecast
$hourlyUrl = $pointsData.properties.forecastHourly

# Extract grid information (for reference/debugging)
$office = $pointsData.properties.cwa
$gridX = $pointsData.properties.gridX
$gridY = $pointsData.properties.gridY
$gridId = $pointsData.properties.gridId

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

Write-Verbose "Grid info: Office=$office (CWA), GridId=$gridId, GridX=$gridX, GridY=$gridY"
Write-Verbose "Location info: County=$county, TimeZone=$timeZone, Radar=$radarStation"
Write-Verbose "Forecast URLs extracted from Points API response"

# --- CONCURRENTLY FETCH FORECAST AND HOURLY DATA ---
Write-Verbose "Starting API calls for forecast data."
Write-Verbose "GET: $forecastUrl"
Write-Verbose "GET: $hourlyUrl"

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

# --- TIMER TRACKING FOR AUTO-REFRESH ---
$dataFetchTime = Get-Date
$dataStaleThreshold = 600  # 10 minutes in seconds

# --- FETCH ALERTS ---
$alertsData = $null
try {
    $alertsUrl = "https://api.weather.gov/alerts/active?point=$lat,$lon"
    Write-Verbose "Fetching alerts from: $alertsUrl"
    Write-Verbose "GET: $alertsUrl"
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

# Helper: resolve a TimeZoneInfo from an ID (supports IANA -> Windows mapping for common US zones)
function Get-ResolvedTimeZoneInfo {
    param([string]$TimeZoneId)

    if (-not $TimeZoneId -or [string]::IsNullOrWhiteSpace($TimeZoneId)) {
        return [System.TimeZoneInfo]::Local
    }

    try { return [System.TimeZoneInfo]::FindSystemTimeZoneById($TimeZoneId) } catch {}

    $ianaToWindows = @{
        'America/Los_Angeles' = 'Pacific Standard Time'
        'America/Denver'      = 'Mountain Standard Time'
        'America/Phoenix'     = 'US Mountain Standard Time'
        'America/Chicago'     = 'Central Standard Time'
        'America/New_York'    = 'Eastern Standard Time'
        'America/Anchorage'   = 'Alaskan Standard Time'
        'Pacific/Honolulu'    = 'Hawaiian Standard Time'
        'America/Boise'       = 'Mountain Standard Time'
        'America/Detroit'     = 'Eastern Standard Time'
        'America/Indiana/Indianapolis' = 'US Eastern Standard Time'
    }

    if ($ianaToWindows.ContainsKey($TimeZoneId)) {
        try { return [System.TimeZoneInfo]::FindSystemTimeZoneById($ianaToWindows[$TimeZoneId]) } catch {}
    }

    return [System.TimeZoneInfo]::Local
}

# Function to calculate sunrise and sunset (NOAA-based, returns local times using provided time zone)
function Get-SunriseSunset {
    param(
        [double]$Latitude,
        [double]$Longitude,
        [DateTime]$Date,
        [string]$TimeZoneId
    )

    # Constants
    $zenithDegrees = 90.833 # Includes standard atmospheric refraction

    # Helpers
    function ToRadians([double]$deg) { return [Math]::PI * $deg / 180.0 }
    function ToDegrees([double]$rad) { return 180.0 * $rad / [Math]::PI }

    $latRad = ToRadians $Latitude
    $dayOfYear = $Date.DayOfYear

    # Fractional year (radians) for day N at 12:00 (good enough for sunrise/sunset)
    $gamma = 2.0 * [Math]::PI * ($dayOfYear - 1) / 365.0

    # Equation of time (minutes) - NOAA approximation
    $equationOfTime = 229.18 * (0.000075 + 0.001868 * [Math]::Cos($gamma) - 0.032077 * [Math]::Sin($gamma) - 0.014615 * [Math]::Cos(2*$gamma) - 0.040849 * [Math]::Sin(2*$gamma))

    # Solar declination (radians) - NOAA series
    $declination = 0.006918 - 0.399912 * [Math]::Cos($gamma) + 0.070257 * [Math]::Sin($gamma) - 0.006758 * [Math]::Cos(2*$gamma) + 0.000907 * [Math]::Sin(2*$gamma) - 0.002697 * [Math]::Cos(3*$gamma) + 0.00148 * [Math]::Sin(3*$gamma)

    # Hour angle for the sun at sunrise/sunset
    $cosH = ([Math]::Cos((ToRadians $zenithDegrees)) - [Math]::Sin($latRad) * [Math]::Sin($declination)) / ([Math]::Cos($latRad) * [Math]::Cos($declination))

    if ($cosH -gt 1) {
        # Polar night - no sunrise
        return @{ Sunrise = $null; Sunset = $null; IsPolarNight = $true; IsPolarDay = $false }
    }
    if ($cosH -lt -1) {
        # Polar day - no sunset
        return @{ Sunrise = $null; Sunset = $null; IsPolarNight = $false; IsPolarDay = $true }
    }

    $H = [Math]::Acos([Math]::Min(1.0, [Math]::Max(-1.0, $cosH))) # Clamp for safety
    $Hdeg = ToDegrees $H

    # Solar noon in minutes from UTC midnight
    $solarNoonUtcMin = 720.0 - 4.0 * $Longitude - $equationOfTime

    # Sunrise/Sunset in minutes from UTC midnight
    $sunriseUtcMin = $solarNoonUtcMin - 4.0 * $Hdeg
    $sunsetUtcMin  = $solarNoonUtcMin + 4.0 * $Hdeg

    # Normalize to 0..1440 range to build DateTime values
    while ($sunriseUtcMin -lt 0) { $sunriseUtcMin += 1440 }
    while ($sunriseUtcMin -ge 1440) { $sunriseUtcMin -= 1440 }
    while ($sunsetUtcMin -lt 0) { $sunsetUtcMin += 1440 }
    while ($sunsetUtcMin -ge 1440) { $sunsetUtcMin -= 1440 }

    # Build UTC DateTimes (note: sunset can be next-day local, conversion handles it)
    $utcMidnight = [DateTime]::new($Date.Year, $Date.Month, $Date.Day, 0, 0, 0, [System.DateTimeKind]::Utc)
    $sunriseUtc = $utcMidnight.AddMinutes($sunriseUtcMin)
    $sunsetUtc  = $utcMidnight.AddMinutes($sunsetUtcMin)

    # Convert to target time zone (falls back to local if mapping fails)
    $tzInfo = Get-ResolvedTimeZoneInfo -TimeZoneId $TimeZoneId
    $sunriseLocal = [System.TimeZoneInfo]::ConvertTimeFromUtc($sunriseUtc, $tzInfo)
    $sunsetLocal  = [System.TimeZoneInfo]::ConvertTimeFromUtc($sunsetUtc,  $tzInfo)

    return @{
        Sunrise = $sunriseLocal
        Sunset  = $sunsetLocal
        IsPolarDay = $false
        IsPolarNight = $false
    }
}

# Extract current weather conditions from the first period of hourly data
# This represents the most recent weather observation
$currentPeriod = $hourlyData.properties.periods[0]
$currentTemp = $currentPeriod.temperature
$currentConditions = $currentPeriod.shortForecast
$currentWind = $currentPeriod.windSpeed
$currentWindDir = $currentPeriod.windDirection
# Use our API call time for the observation timestamp (when we fetched the data)
$currentTime = $dataFetchTime
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
$todayPeriodName = $todayPeriod.name

# Extract tomorrow's detailed forecast (second period in forecast data)
$tomorrowPeriod = $forecastData.properties.periods[1]
$tomorrowForecast = $tomorrowPeriod.detailedForecast
$tomorrowPeriodName = $tomorrowPeriod.name



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

# Function: Determine if a weather period is during day or night using calculated sunrise/sunset times
# More accurate than NWS API isDaytime property as it uses astronomical calculations
function Test-IsDaytimeAstronomical {
    param(
        [object]$Period,
        [DateTime]$SunriseTime,
        [DateTime]$SunsetTime
    )
    
    # Parse the period start time
    $periodTime = [DateTime]::Parse($period.startTime)
    
    # Extract just the time portion for comparison and round to the hour
    $periodTimeOnly = $periodTime.TimeOfDay
    $sunriseTimeOnly = [TimeSpan]::FromHours([Math]::Round($SunriseTime.TimeOfDay.TotalHours))
    $sunsetTimeOnly = [TimeSpan]::FromHours([Math]::Round($SunsetTime.TimeOfDay.TotalHours))
    
    # Handle cases where sunset is the next day (after midnight) - polar regions
    if ($sunsetTimeOnly -lt $sunriseTimeOnly) {
        # Sunset is the next day, so daytime is from sunrise to midnight OR midnight to sunset
        return ($periodTimeOnly -ge $sunriseTimeOnly) -or ($periodTimeOnly -lt $sunsetTimeOnly)
    } else {
        # Normal case: sunset is same day as sunrise
        return $periodTimeOnly -ge $sunriseTimeOnly -and $periodTimeOnly -lt $sunsetTimeOnly
    }
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
        [string]$SunriseTime,
        [string]$SunsetTime,
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

    # Display sunrise and sunset times
    if ($SunriseTime) {
        Write-Host "Sunrise: $SunriseTime" -ForegroundColor Yellow
    }
    if ($SunsetTime) {
        Write-Host "Sunset: $SunsetTime" -ForegroundColor Yellow
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
        [bool]$IsInteractive = $false,
        [DateTime]$SunriseTime = $null,
        [DateTime]$SunsetTime = $null
    )
    
    Write-Host ""
    Write-Host "*** Hourly ***" -ForegroundColor $TitleColor
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
        
        # Determine if this period is during day or night using astronomical calculations
        if ($SunriseTime -and $SunsetTime) {
            $isPeriodDaytime = Test-IsDaytimeAstronomical $period $SunriseTime $SunsetTime
        } else {
            # Fallback to NWS API isDaytime property if sunrise/sunset not available
            $isPeriodDaytime = Test-IsDaytime $period
        }
        $periodIcon = Get-WeatherIcon $period.icon $isPeriodDaytime $precipProb
        
        # Color code temperature
        $tempColor = if ([int]$temp -lt $script:COLD_TEMP_THRESHOLD -or [int]$temp -gt $script:HOT_TEMP_THRESHOLD) { $AlertColor } else { $DefaultColor }
        
        # Color code precipitation probability
        $precipColor = if ($precipProb -gt $script:HIGH_PRECIP_THRESHOLD) { $AlertColor } elseif ($precipProb -gt $script:MEDIUM_PRECIP_THRESHOLD) { "Yellow" } else { $DefaultColor }
        
        # Build and write the line piece-by-piece for easier colorization
        $timePart = "$hourDisplay "
        $iconPart = "$periodIcon"
        $tempPart = " $temp°F "
        $windPart = "$wind $($windDir.PadRight(3))"
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
        [int]$MaxDays = $script:MAX_DAILY_FORECAST_DAYS,
        [DateTime]$SunriseTime = $null,
        [DateTime]$SunsetTime = $null
    )
    
    Write-Host ""
    Write-Host "*** 7-Day Summary ***" -ForegroundColor $TitleColor
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
        
        # Determine if this period is during day or night using astronomical calculations
        if ($SunriseTime -and $SunsetTime) {
            $isPeriodDaytime = Test-IsDaytimeAstronomical $period $SunriseTime $SunsetTime
        } else {
            # Fallback to NWS API isDaytime property if sunrise/sunset not available
            $isPeriodDaytime = Test-IsDaytime $period
        }
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
        Write-Host "Mode: " -ForegroundColor Green -NoNewline
        Write-Host "Scroll[" -ForegroundColor White -NoNewline; Write-Host "↑↓" -ForegroundColor Cyan -NoNewline; Write-Host "], " -ForegroundColor White -NoNewline
        Write-Host "F" -ForegroundColor Cyan -NoNewline; Write-Host "ull " -ForegroundColor White -NoNewline
        Write-Host "T" -ForegroundColor Cyan -NoNewline; Write-Host "erse " -ForegroundColor White -NoNewline
        Write-Host "D" -ForegroundColor Cyan -NoNewline; Write-Host "aily " -ForegroundColor White -NoNewline
        Write-Host "H" -ForegroundColor Cyan -NoNewline; Write-Host "ourly " -ForegroundColor White -NoNewline
        Write-Host "R" -ForegroundColor Cyan -NoNewline; Write-Host "ain " -ForegroundColor White -NoNewline
        Write-Host "W" -ForegroundColor Cyan -NoNewline; Write-Host "ind" -ForegroundColor White
    } else {
        Write-Host "Mode: " -ForegroundColor Green -NoNewline
        Write-Host "F" -ForegroundColor Cyan -NoNewline; Write-Host "ull " -ForegroundColor White -NoNewline
        Write-Host "T" -ForegroundColor Cyan -NoNewline; Write-Host "erse " -ForegroundColor White -NoNewline
        Write-Host "D" -ForegroundColor Cyan -NoNewline; Write-Host "aily " -ForegroundColor White -NoNewline
        Write-Host "H" -ForegroundColor Cyan -NoNewline; Write-Host "ourly " -ForegroundColor White -NoNewline
        Write-Host "R" -ForegroundColor Cyan -NoNewline; Write-Host "ain " -ForegroundColor White -NoNewline
        Write-Host "W" -ForegroundColor Cyan -NoNewline; Write-Host "ind" -ForegroundColor White
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
        [string]$TodayPeriodName,
        [string]$TomorrowForecast,
        [string]$TomorrowPeriodName,
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
        Show-CurrentConditions -City $City -State $State -WeatherIcon $WeatherIcon -CurrentConditions $CurrentConditions -CurrentTemp $CurrentTemp -TempColor $TempColor -CurrentTempTrend $CurrentTempTrend -CurrentWind $CurrentWind -WindColor $WindColor -CurrentWindDir $CurrentWindDir -WindGust $WindGust -CurrentHumidity $CurrentHumidity -CurrentDewPoint $CurrentDewPoint -CurrentPrecipProb $CurrentPrecipProb -CurrentTimeLocal $dataFetchTime -SunriseTime $($sunriseTime.ToString('h:mm tt')) -SunsetTime $($sunsetTime.ToString('h:mm tt')) -DefaultColor $DefaultColor -AlertColor $AlertColor -TitleColor $TitleColor -InfoColor $InfoColor
    }

    if ($ShowTodayForecast) {
        Show-ForecastText -Title $TodayPeriodName -ForecastText $TodayForecast -TitleColor $TitleColor -DefaultColor $DefaultColor
    }

    if ($ShowTomorrowForecast) {
        Show-ForecastText -Title $TomorrowPeriodName -ForecastText $TomorrowForecast -TitleColor $TitleColor -DefaultColor $DefaultColor
    }

    if ($ShowHourlyForecast) {
        Show-HourlyForecast -HourlyData $HourlyData -TitleColor $TitleColor -DefaultColor $DefaultColor -AlertColor $AlertColor -IsInteractive $false -SunriseTime $SunriseTime -SunsetTime $SunsetTime
    }

    if ($ShowSevenDayForecast) {
        Show-SevenDayForecast -ForecastData $ForecastData -TitleColor $TitleColor -DefaultColor $DefaultColor -AlertColor $AlertColor -SunriseTime $SunriseTime -SunsetTime $SunsetTime
    }

    if ($ShowAlerts) {
        Show-WeatherAlerts -AlertsData $AlertsData -AlertColor $AlertColor -DefaultColor $DefaultColor -InfoColor $InfoColor -ShowDetails $ShowAlertDetails
    }



    if ($ShowLocationInfo) {
        Show-LocationInfo -County $County -TimeZone $TimeZone -RadarStation $RadarStation -Lat $Lat -Lon $Lon -TitleColor $TitleColor -DefaultColor $DefaultColor
    }
}

# Function to map rain percentage to sparkline character and color
# Color coding: White (0%), White (1-10%), Cyan (11-33%), Green (34-40%), Yellow (41-80%), Red (81%+)
function Get-RainSparkline {
    param([int]$RainPercent)
    
    if ($RainPercent -eq 0) { return @{Char=" "; Color="White"} }
    elseif ($RainPercent -le 10) { return @{Char="▁"; Color="White"} }
    elseif ($RainPercent -le 33) { return @{Char="▂"; Color="Cyan"} }
    elseif ($RainPercent -le 44) { return @{Char="▃"; Color="Green"} }
    elseif ($RainPercent -le 66) { return @{Char="▄"; Color="Yellow"} }
    elseif ($RainPercent -le 80) { return @{Char="▅"; Color="Yellow"} }
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
        "N" = 0; "NNE" = 0; "NNW" = 7
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
        $glyphs = @("▽", "◺", "◁", "◸", "△", "◹", "▷", "◿")
    } else {
        $glyphs = @("▼", "◣", "◀", "◤", "▲", "◥", "▶", "◢")
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
        [string]$DefaultColor,
        [string]$City
    )
    
    Write-Host ""
    $headerText = "*** $City Rain Outlook ***"
    $padding = [Math]::Max(0, (34 - $headerText.Length) / 2)
    $paddedHeader = " " * $padding + $headerText
    Write-Host $paddedHeader -ForegroundColor $TitleColor
    
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
        
        # Get color for the highest percentage (White: 0%, Cyan: 1-33%, Yellow: 34-66%, Red: 67%+)
        $maxRainColor = if ($maxRainPercent -le 10) { "White" }
                       elseif ($maxRainPercent -le 33) { "Cyan" }
                       elseif ($maxRainPercent -le 44) { "Green" }
                       elseif ($maxRainPercent -le 80) { "Yellow" }
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
        [string]$DefaultColor,
        [string]$City
    )
    
    Write-Host ""
    $headerText = "*** $City Wind Outlook ***"
    $padding = [Math]::Max(0, (34 - $headerText.Length) / 2)
    $paddedHeader = " " * $padding + $headerText
    Write-Host $paddedHeader -ForegroundColor Green
    
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
                
                # Check if this hour matches the peak wind speed for the day
                # Visual feedback: Invert colors (black text on colored background) for peak wind hours
                if ($windSpeed -eq $maxWindSpeed) {
                    # Invert colors for peak wind hours - makes them stand out visually
                    Write-Host $windGlyphData.Char -ForegroundColor Black -BackgroundColor $windGlyphData.Color -NoNewline
                } else {
                    # Normal colors for non-peak hours
                    Write-Host $windGlyphData.Char -ForegroundColor $windGlyphData.Color -NoNewline
                }
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

# Use our API call time for the display timestamp
$currentTimeLocal = $dataFetchTime
Write-Verbose "Using API call time: $currentTimeLocal"

# Calculate sunrise and sunset times
$sunTimes = Get-SunriseSunset -Latitude $lat -Longitude $lon -Date (Get-Date) -TimeZoneId $timeZone
$sunriseTime = $sunTimes.Sunrise
$sunsetTime = $sunTimes.Sunset

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
    # Rain mode: Show only rain likelihood forecast with sparklines
    $showCurrentConditions = $false
    $showTodayForecast = $false
    $showTomorrowForecast = $false
    $showHourlyForecast = $false
    $showSevenDayForecast = $false
    $showAlerts = $false
    $showLocationInfo = $false
}
elseif ($Wind.IsPresent) {
    # Wind mode: Show only wind outlook forecast with direction glyphs
    $showCurrentConditions = $false
    $showTodayForecast = $false
    $showTomorrowForecast = $false
    $showHourlyForecast = $false
    $showSevenDayForecast = $false
    $showAlerts = $false
    $showLocationInfo = $false
}

# Determine if it's currently day or night using NWS API isDaytime property
$isCurrentlyDaytime = Test-IsDaytime $currentPeriod

# Output the results.
$weatherIcon = Get-WeatherIcon $currentIcon $isCurrentlyDaytime $currentPrecipProb

# Display the weather report using the refactored function
if ($Rain.IsPresent) {
    # Rain mode: Show only rain likelihood forecast with sparklines
    Show-RainForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city
} elseif ($Wind.IsPresent) {
    # Wind mode: Show only wind outlook forecast with direction glyphs
    Show-WindForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city
} else {
    Show-FullWeatherReport -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $dataFetchTime -TodayForecast $todayForecast -TodayPeriodName $todayPeriodName -TomorrowForecast $tomorrowForecast -TomorrowPeriodName $tomorrowPeriodName -HourlyData $hourlyData -ForecastData $forecastData -AlertsData $alertsData -County $county -TimeZone $timeZone -RadarStation $radarStation -Lat $lat -Lon $lon -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor -ShowCurrentConditions $showCurrentConditions -ShowTodayForecast $showTodayForecast -ShowTomorrowForecast $showTomorrowForecast -ShowHourlyForecast $showHourlyForecast -ShowSevenDayForecast $showSevenDayForecast -ShowAlerts $showAlerts -ShowAlertDetails $showAlertDetails -ShowLocationInfo $showLocationInfo
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
    $isRainMode = $false  # State tracking for rain forecast mode
    $isWindMode = $false  # State tracking for wind forecast mode
    $isTerseMode = $false  # State tracking for terse mode
    $autoUpdateEnabled = -not $NoAutoUpdate.IsPresent  # State tracking for auto-updates
    if (-not $autoUpdateEnabled) {
        Write-Verbose "Auto-updates disabled via command line flag"
    }
    $hourlyScrollIndex = 0
    $totalHourlyPeriods = [Math]::Min($hourlyData.properties.periods.Count, 48)  # Limit to 48 hours
    
    # Initialize mode state tracking
    Write-Verbose "Interactive mode initialized - Hourly: $isHourlyMode, Rain: $isRainMode, Wind: $isWindMode, Auto-Update: $autoUpdateEnabled"
    
    # If starting in hourly mode, show hourly forecast first
    if ($isHourlyMode) {
        Clear-Host
        Show-HourlyForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true -SunriseTime $sunriseTime -SunsetTime $sunsetTime
        Show-InteractiveControls -IsHourlyMode $true
    } elseif ($Rain.IsPresent) {
        # If starting in rain mode, show rain forecast first
        Clear-Host
        $isRainMode = $true
        Show-RainForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city
        Show-InteractiveControls
    } elseif ($Wind.IsPresent) {
        # If starting in wind mode, show wind forecast first
        Clear-Host
        $isWindMode = $true
        Show-WindForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city
        Show-InteractiveControls
    } else {
        # Interactive mode: Listen for keyboard input to switch between display modes
        Show-InteractiveControls
    }
    
    while ($true) {
        try {
            # Check if data is stale and refresh if needed (only if auto-update is enabled)
            if ($autoUpdateEnabled) {
                $timeSinceLastFetch = (Get-Date) - $dataFetchTime
                if ($timeSinceLastFetch.TotalSeconds -gt $dataStaleThreshold) {
                    Write-Verbose "Auto-refresh triggered - data is stale ($([math]::Round($timeSinceLastFetch.TotalSeconds, 1)) seconds old)"
                    $refreshSuccess = Update-WeatherData -Lat $lat -Lon $lon -Headers $headers
                    if ($refreshSuccess) {
                        # Re-render current view with fresh data
                        Clear-Host
                        if ($isHourlyMode) {
                            Show-HourlyForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true -SunriseTime $sunriseTime -SunsetTime $sunsetTime
                            Show-InteractiveControls -IsHourlyMode $true
                        } elseif ($isRainMode) {
                            Show-RainForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city
                            Show-InteractiveControls
                        } elseif ($isWindMode) {
                            Show-WindForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city
                            Show-InteractiveControls
                        } else {
                            # Preserve current mode - show terse mode if in terse mode
                            if ($isTerseMode) {
                                Show-CurrentConditions -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $dataFetchTime -SunriseTime $($sunriseTime.ToString('h:mm tt')) -SunsetTime $($sunsetTime.ToString('h:mm tt')) -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor
                                Show-ForecastText -Title $todayPeriodName -ForecastText $todayForecast -TitleColor $titleColor -DefaultColor $defaultColor
                                Show-WeatherAlerts -AlertsData $alertsData -AlertColor $alertColor -DefaultColor $defaultColor -InfoColor $infoColor -ShowDetails $false
                                Show-InteractiveControls
                            } else {
                                Show-FullWeatherReport -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $dataFetchTime -TodayForecast $todayForecast -TodayPeriodName $todayPeriodName -TomorrowForecast $tomorrowForecast -TomorrowPeriodName $tomorrowPeriodName -HourlyData $hourlyData -ForecastData $forecastData -AlertsData $alertsData -County $county -TimeZone $timeZone -RadarStation $radarStation -Lat $lat -Lon $lon -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor -ShowCurrentConditions $showCurrentConditions -ShowTodayForecast $showTodayForecast -ShowTomorrowForecast $showTomorrowForecast -ShowHourlyForecast $showHourlyForecast -ShowSevenDayForecast $showSevenDayForecast -ShowAlerts $showAlerts -ShowAlertDetails $showAlertDetails -ShowLocationInfo $showLocationInfo
                                Show-InteractiveControls
                            }
                        }
                    }
                }
            }
            
            # Check for key input (non-blocking) - using same approach as bmon.ps1
            if ([System.Console]::KeyAvailable) {
                $keyInfo = [System.Console]::ReadKey($true)
                
                # Handle keyboard input for interactive mode
                switch ($keyInfo.KeyChar) {
                'h' { # H key - Switch to hourly forecast only
                    Clear-Host
                    $isHourlyMode = $true
                    $isRainMode = $false
                    $isWindMode = $false
                    $isTerseMode = $false
                    $hourlyScrollIndex = 0  # Reset to first 12 hours
                    Show-HourlyForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true -SunriseTime $sunriseTime -SunsetTime $sunsetTime
                    Show-InteractiveControls -IsHourlyMode $true
                }
                'd' { # D key - Switch to 7-day forecast only
                    Clear-Host
                    $isHourlyMode = $false
                    $isRainMode = $false
                    $isWindMode = $false
                    $isTerseMode = $false
                    Show-SevenDayForecast -ForecastData $forecastData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -SunriseTime $sunriseTime -SunsetTime $sunsetTime
                    Show-InteractiveControls
                }
                't' { # T key - Switch to terse mode (current + today + alerts)
                    Clear-Host
                    $isHourlyMode = $false
                    $isRainMode = $false
                    $isWindMode = $false
                    $isTerseMode = $true
                    Show-CurrentConditions -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $dataFetchTime -SunriseTime $($sunriseTime.ToString('h:mm tt')) -SunsetTime $($sunsetTime.ToString('h:mm tt')) -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor
                    Show-ForecastText -Title $todayPeriodName -ForecastText $todayForecast -TitleColor $titleColor -DefaultColor $defaultColor
                    Show-WeatherAlerts -AlertsData $alertsData -AlertColor $alertColor -DefaultColor $defaultColor -InfoColor $infoColor -ShowDetails $false
                    Show-InteractiveControls
                }
                'f' { # F key - Switch to full weather report
                    Clear-Host
                    $isHourlyMode = $false
                    $isRainMode = $false
                    $isWindMode = $false
                    $isTerseMode = $false
                    Show-FullWeatherReport -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $dataFetchTime -TodayForecast $todayForecast -TodayPeriodName $todayPeriodName -TomorrowForecast $tomorrowForecast -TomorrowPeriodName $tomorrowPeriodName -HourlyData $hourlyData -ForecastData $forecastData -AlertsData $alertsData -County $county -TimeZone $timeZone -RadarStation $radarStation -Lat $lat -Lon $lon -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor -ShowCurrentConditions $true -ShowTodayForecast $true -ShowTomorrowForecast $true -ShowHourlyForecast $true -ShowSevenDayForecast $true -ShowAlerts $true -ShowAlertDetails $true -ShowLocationInfo $true
                    Show-InteractiveControls
                }
                'r' { # R key - Switch to rain forecast mode
                    Clear-Host
                    $isHourlyMode = $false
                    $isRainMode = $true
                    $isWindMode = $false
                    $isTerseMode = $false
                    Show-RainForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city
                    Show-InteractiveControls
                }
                'u' { # U key - Toggle automatic updates
                    $autoUpdateEnabled = -not $autoUpdateEnabled
                    Write-Verbose "Auto-update toggled: $autoUpdateEnabled"
                    $statusMessage = if ($autoUpdateEnabled) { 
                        $dataFetchTime = Get-Date  # Reset timer when re-enabling
                        "Automatic Updates Enabled" 
                    } else { 
                        "Automatic Updates Disabled" 
                    }
                    
                    # Show status message briefly
                    Write-Host "`n$statusMessage" -ForegroundColor $(if ($autoUpdateEnabled) { "Green" } else { "Yellow" })
                    Start-Sleep -Milliseconds 800
                    
                    # Re-render current view
                    Clear-Host
                    if ($isHourlyMode) {
                        Show-HourlyForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true -SunriseTime $sunriseTime -SunsetTime $sunsetTime
                        Show-InteractiveControls -IsHourlyMode $true
                    } elseif ($isRainMode) {
                        Show-RainForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city
                        Show-InteractiveControls
                    } elseif ($isWindMode) {
                        Show-WindForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city
                        Show-InteractiveControls
                    } else {
                        # Preserve current mode - show terse mode if in terse mode
                        if ($isTerseMode) {
                            Show-CurrentConditions -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $dataFetchTime -SunriseTime $($sunriseTime.ToString('h:mm tt')) -SunsetTime $($sunsetTime.ToString('h:mm tt')) -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor
                            Show-ForecastText -Title $todayPeriodName -ForecastText $todayForecast -TitleColor $titleColor -DefaultColor $defaultColor
                            Show-WeatherAlerts -AlertsData $alertsData -AlertColor $alertColor -DefaultColor $defaultColor -InfoColor $infoColor -ShowDetails $false
                            Show-InteractiveControls
                        } else {
                            Show-FullWeatherReport -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $dataFetchTime -TodayForecast $todayForecast -TodayPeriodName $todayPeriodName -TomorrowForecast $tomorrowForecast -TomorrowPeriodName $tomorrowPeriodName -HourlyData $hourlyData -ForecastData $forecastData -AlertsData $alertsData -County $county -TimeZone $timeZone -RadarStation $radarStation -Lat $lat -Lon $lon -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor -ShowCurrentConditions $showCurrentConditions -ShowTodayForecast $showTodayForecast -ShowTomorrowForecast $showTomorrowForecast -ShowHourlyForecast $showHourlyForecast -ShowSevenDayForecast $showSevenDayForecast -ShowAlerts $showAlerts -ShowAlertDetails $showAlertDetails -ShowLocationInfo $showLocationInfo
                            Show-InteractiveControls
                        }
                    }
                }
                'w' { # W key - Switch to wind forecast mode
                    Clear-Host
                    $isHourlyMode = $false
                    $isRainMode = $false
                    $isWindMode = $true
                    $isTerseMode = $false
                    Show-WindForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city
                    Show-InteractiveControls
                }
                'g' { # G key - Refresh weather data
                    $refreshSuccess = Update-WeatherData -Lat $lat -Lon $lon -Headers $headers
                    if ($refreshSuccess) {
                        # Re-render current view with fresh data
                        Clear-Host
                        if ($isHourlyMode) {
                            Show-HourlyForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true -SunriseTime $sunriseTime -SunsetTime $sunsetTime
                            Show-InteractiveControls -IsHourlyMode $true
                        } elseif ($isRainMode) {
                            Show-RainForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city
                            Show-InteractiveControls
                        } elseif ($isWindMode) {
                            Show-WindForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city
                            Show-InteractiveControls
                        } else {
                            Show-FullWeatherReport -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $dataFetchTime -TodayForecast $todayForecast -TodayPeriodName $todayPeriodName -TomorrowForecast $tomorrowForecast -TomorrowPeriodName $tomorrowPeriodName -HourlyData $hourlyData -ForecastData $forecastData -AlertsData $alertsData -County $county -TimeZone $timeZone -RadarStation $radarStation -Lat $lat -Lon $lon -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor -ShowCurrentConditions $showCurrentConditions -ShowTodayForecast $showTodayForecast -ShowTomorrowForecast $showTomorrowForecast -ShowHourlyForecast $showHourlyForecast -ShowSevenDayForecast $showSevenDayForecast -ShowAlerts $showAlerts -ShowAlertDetails $showAlertDetails -ShowLocationInfo $showLocationInfo
                            Show-InteractiveControls
                        }
                    }
                }
                { $keyInfo.Key -eq 'UpArrow' } { # Up arrow - Scroll up in hourly mode
                    if ($isHourlyMode) {
                        $newIndex = $hourlyScrollIndex - $script:MAX_HOURLY_FORECAST_HOURS
                        if ($newIndex -ge 0) {
                            $hourlyScrollIndex = $newIndex
                            Clear-Host
                            Show-HourlyForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true -SunriseTime $sunriseTime -SunsetTime $sunsetTime
                            Show-InteractiveControls -IsHourlyMode $true
                        }
                    }
                }
                { $keyInfo.Key -eq 'DownArrow' } { # Down arrow - Scroll down in hourly mode
                    if ($isHourlyMode) {
                        $newIndex = $hourlyScrollIndex + $script:MAX_HOURLY_FORECAST_HOURS
                        if ($newIndex -lt $totalHourlyPeriods) {
                            $hourlyScrollIndex = $newIndex
                            Clear-Host
                            Show-HourlyForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true -SunriseTime $sunriseTime -SunsetTime $sunsetTime
                            Show-InteractiveControls -IsHourlyMode $true
                        }
                    }
                }
                { $keyInfo.Key -eq 'Enter' } { # Enter key - Exit interactive mode
                    Write-Host "Exiting..." -ForegroundColor Yellow
                    return
                }
                { $keyInfo.Key -eq 'NumPadEnter' } { # NumPad Enter key - Exit interactive mode
                    Write-Host "Exiting..." -ForegroundColor Yellow
                    return
                }
                { $keyInfo.Key -eq 'Escape' } { # Esc key - Exit interactive mode
                    Write-Host "Exiting..." -ForegroundColor Yellow
                    return
                }
                default {
                    # Ignore unhandled keys (like spacebar, etc.)
                    # Do nothing - just continue the loop
                }
            }
            } else {
                # No key available - sleep briefly to prevent CPU spinning
                Start-Sleep -Milliseconds 100
            }
        }
        catch {
            Write-Host "Interactive mode not supported in this environment. Exiting..." -ForegroundColor Yellow
            return
        }
    }
}
