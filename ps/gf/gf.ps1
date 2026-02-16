<#
.ENCODING
    This file MUST be saved as UTF-8 with BOM. Do not change the encoding or script errors may occur (e.g. with glyphs/emoji).
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
    Moon phase information is calculated using astronomical methods with emoji display.
    
    The script includes wind chill and heat index calculations using NWS formulas:
    - Wind Chill: Displayed in blue when temperature <= 50°F and difference > 1°F
    - Heat Index: Displayed in red when temperature >= 80°F and difference > 1°F
    
    Color-coded weather indicators:
    - Temperature: Blue (<33°F), Red (>89°F), White (normal range)
    - Wind Speed: White (≤5 mph), Yellow (6-9 mph), Red (10-14 mph), Magenta (≥15 mph)
    - Precipitation: Red (>50%), Yellow (21-50%), White (≤20%)
    - Humidity: Cyan (<30%), White (30-60%), Yellow (61-70%), Red (>70%)
    - Dew Point: Cyan (<40°F), White (40-54°F), Yellow (55-64°F), Red (≥65°F)
    - Pressure (Observations): Cyan (<29.50 inHg), White (29.50-30.20), Yellow (>30.20), Alert (extreme)
    - Clouds (Observations): When the station provides cloud data, "Clouds:" is shown on the same line as Conditions (label white, data gray). Codes: SKC (clear), FEW (few), SCT (scattered), BKN (broken), OVC (overcast), VV (vertical visibility; sky obscured). Omitted when not available.
    
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
    [switch]$NoAutoUpdate,

    [Alias('b')]
    [switch]$NoBar,

    [Alias('o')]
    [switch]$Observations,

    [Parameter(Mandatory = $false)]
    [string]$Noaa
)

# --- Helper Functions ---

# Initialize control bar visibility (can be toggled with 'b' key)
$script:showControlBar = -not $NoBar.IsPresent

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
if ($Help -or (($Terse.IsPresent -or $Hourly.IsPresent -or $Daily.IsPresent -or $Rain.IsPresent -or $Wind.IsPresent -or $Observations.IsPresent -or $NoInteractive.IsPresent) -and -not $Location)) {
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
    Write-Host "  -o, -Observations    Show historical weather observations" -ForegroundColor Cyan
    Write-Host "                • Clouds on same line as Conditions when available (SKC=clear, FEW=few, SCT=scattered, BKN=broken, OVC=overcast, VV=vertical visibility)" -ForegroundColor Gray
    Write-Host "                • Pressure (inHg): Cyan (<29.50), White (29.50-30.20), Yellow (>30.20), Alert (extreme)" -ForegroundColor Gray
    Write-Host "  -u, -NoAutoUpdate Start with automatic updates disabled" -ForegroundColor Cyan
    Write-Host "  -b, -NoBar    Start with control bar hidden" -ForegroundColor Cyan
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
     Write-Host "    [O] - Switch to observations historical data" -ForegroundColor Cyan
    Write-Host "    [G] - Refresh weather data (auto-refreshes every 10 minutes)" -ForegroundColor Cyan
    Write-Host "    [U] - Toggle automatic updates on/off" -ForegroundColor Cyan
    Write-Host "    [B] - Toggle control bar on/off" -ForegroundColor Cyan
    Write-Host "    [F] - Return to full display" -ForegroundColor Cyan
    Write-Host "    [Enter] or [Esc] - Exit the script" -ForegroundColor Cyan
     Write-Host "  In hourly mode, use [↑] and [↓] arrows to scroll through all 48 hours" -ForegroundColor Cyan
     Write-Host "  Note: All times (hourly, sunrise, sunset) are displayed in the location's timezone" -ForegroundColor Gray
    Write-Host ""
    Write-Host "This script retrieves weather info from National Weather Service API (geocoding via OpenStreetMap) and outputs:" -ForegroundColor Blue
    Write-Host " • Location (City, State)" -ForegroundColor Cyan
    Write-Host " • Current Conditions" -ForegroundColor Cyan
    Write-Host " • Temperature with forecast range (Blue <33°F / Red >89°F)" -ForegroundColor Cyan
    Write-Host " • Wind Chill (Blue when temp <= 50°F and difference > 1°F)" -ForegroundColor Cyan
    Write-Host " • Heat Index (Red when temp >= 80°F and difference > 1°F)" -ForegroundColor Cyan
    Write-Host " • Humidity" -ForegroundColor Cyan
    Write-Host " • Wind (with gust if available; red if wind speed >=16 mph)" -ForegroundColor Cyan
    Write-Host " • Sunrise and Sunset times (calculated astronomically)" -ForegroundColor Cyan
    Write-Host " • Detailed Forecast" -ForegroundColor Cyan
    Write-Host " • Weather Alerts" -ForegroundColor Cyan
    Write-Host " • Forecast Fetch timestamp" -ForegroundColor Cyan
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
    Write-Host "  .\gf.ps1 97219 -o For Observations" -ForegroundColor Cyan
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

# Helper function to clear screen with optional delay in verbose mode
function Clear-HostWithDelay {
    if ($VerbosePreference -ne 'Continue') {
        Clear-Host
    }
}

# Function to resolve timezone ID to TimeZoneInfo object
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

# Function to process observations data into daily aggregates
function Convert-ObservationsData {
    param(
        [object]$ObservationsData,
        [string]$TimeZone
    )
    
    try {
        # Process observations and group by day
        $dailyData = @{}
        
        Write-Verbose "Processing $($ObservationsData.features.Count) observations"
        
        foreach ($observation in $ObservationsData.features) {
            try {
                # Parse the observation timestamp (API returns in UTC)
                $obsTimeOffset = [DateTimeOffset]::Parse($observation.properties.timestamp)
                
                # Convert to local timezone if provided
                if ($TimeZone) {
                    $localTimeZone = Get-ResolvedTimeZoneInfo -TimeZoneId $TimeZone
                    if ($localTimeZone) {
                        $obsTimeLocal = [System.TimeZoneInfo]::ConvertTime($obsTimeOffset, $localTimeZone)
                        $obsDate = $obsTimeLocal.ToString("yyyy-MM-dd")
                    } else {
                        # Fallback to UTC if timezone conversion fails
                        Write-Verbose "Timezone conversion failed, using UTC for observation"
                        $obsDate = $obsTimeOffset.UtcDateTime.ToString("yyyy-MM-dd")
                    }
                } else {
                    # Fallback to UTC if no timezone provided
                    $obsDate = $obsTimeOffset.UtcDateTime.ToString("yyyy-MM-dd")
                }
            } catch {
                Write-Verbose "Error parsing observation timestamp: $($_.Exception.Message)"
                continue
            }
            
            if (-not $dailyData.ContainsKey($obsDate)) {
                $dailyData[$obsDate] = @{
                    Date = $obsDate
                    Temperatures = @()
                    WindSpeeds = @()
                    WindGusts = @()
                    WindDirections = @()
                    Humidities = @()
                    Precipitations = @()
                    Conditions = @()
                    Pressures = @()
                    CloudSummaries = @()
                }
            }
            
            $props = $observation.properties
            
            # Extract temperature (convert from Celsius to Fahrenheit if needed)
            if ($props.temperature -and $props.temperature.value) {
                $tempC = $props.temperature.value
                $tempF = ($tempC * 9/5) + 32
                $dailyData[$obsDate].Temperatures += $tempF
            }
            
            # Extract wind speed (sustained wind, convert from km/h to mph)
            if ($props.windSpeed -and $props.windSpeed.value) {
                $windSpeedKmh = $props.windSpeed.value
                $windSpeedMph = $windSpeedKmh * 0.621371
                $dailyData[$obsDate].WindSpeeds += $windSpeedMph
            }
            
            # Extract wind gust (peak wind, convert from km/h to mph)
            if ($props.windGust -and $props.windGust.value) {
                $windGustKmh = $props.windGust.value
                $windGustMph = $windGustKmh * 0.621371
                $dailyData[$obsDate].WindGusts += $windGustMph
            }
            
            # Extract wind direction
            if ($props.windDirection -and $props.windDirection.value) {
                $dailyData[$obsDate].WindDirections += $props.windDirection.value
            }
            
            # Extract sea-level pressure (NWS API returns Pascals; convert to inHg: inHg = Pa / 3386.389)
            if ($props.seaLevelPressure -and $null -ne $props.seaLevelPressure.value) {
                $pressurePa = $props.seaLevelPressure.value
                $pressureInHg = $pressurePa / 3386.389
                $dailyData[$obsDate].Pressures += $pressureInHg
            }
            
            # Extract humidity
            if ($props.relativeHumidity -and $props.relativeHumidity.value) {
                $dailyData[$obsDate].Humidities += $props.relativeHumidity.value
            }
            
            # Extract precipitation - try multiple fields for better accuracy
            # precipitationLastHour is most common, but also check other time periods
            # NWS API returns these values in millimeters, so we convert to inches
            $precipValue = $null
            if ($props.precipitationLastHour -and $props.precipitationLastHour.value) {
                # Convert from millimeters to inches (1 mm = 0.0393701 inches)
                $precipValue = $props.precipitationLastHour.value * 0.0393701
            } elseif ($props.precipitationLast3Hours -and $props.precipitationLast3Hours.value) {
                # Convert from millimeters to inches and divide by 3 to get hourly equivalent
                $precipValue = ($props.precipitationLast3Hours.value * 0.0393701) / 3
            } elseif ($props.precipitationLast6Hours -and $props.precipitationLast6Hours.value) {
                # Convert from millimeters to inches and divide by 6 to get hourly equivalent
                $precipValue = ($props.precipitationLast6Hours.value * 0.0393701) / 6
            }
            if ($null -ne $precipValue) {
                $dailyData[$obsDate].Precipitations += $precipValue
            }
            
            # Extract conditions
            if ($props.textDescription) {
                $dailyData[$obsDate].Conditions += $props.textDescription
            }
            
            # Extract cloud layers summary (amount + base height in ft)
            if ($props.cloudLayers -and $props.cloudLayers.Count -gt 0) {
                $parts = @()
                foreach ($layer in $props.cloudLayers) {
                    $amount = if ($layer.amount) { $layer.amount.Trim() } else { '?' }
                    $baseM = if ($layer.base -and $null -ne $layer.base.value) { $layer.base.value } else { $null }
                    $baseFt = if ($null -ne $baseM) { [Math]::Round($baseM * 3.28084) } else { $null }
                    $ftStr = if ($null -ne $baseFt) { "{0:N0} ft" -f $baseFt } else { "? ft" }
                    $parts += "$amount $ftStr"
                }
                $summary = $parts -join ', '
                if ($summary) { $dailyData[$obsDate].CloudSummaries += $summary }
            }
        }
        
        # Calculate daily aggregates
        $result = @()
        
        # Get current date in the target timezone (same as observations)
        if ($TimeZone) {
            $localTimeZone = Get-ResolvedTimeZoneInfo -TimeZoneId $TimeZone
            if ($localTimeZone) {
                $nowInLocalTz = [System.TimeZoneInfo]::ConvertTime([DateTimeOffset]::Now, $localTimeZone)
            } else {
                $nowInLocalTz = [DateTimeOffset]::Now
            }
        } else {
            $nowInLocalTz = [DateTimeOffset]::Now
        }
        
        for ($i = 6; $i -ge 0; $i--) {
            $targetDate = $nowInLocalTz.AddDays(-$i)
            $date = $targetDate.ToString("yyyy-MM-dd")
            Write-Verbose "Checking for date: $date (day offset: -$i)"
            if ($dailyData.ContainsKey($date)) {
                Write-Verbose "Found data for date: $date"
                $dayData = $dailyData[$date]
                $result += @{
                    Date = $date
                    HighTemp = if ($dayData.Temperatures.Count -gt 0) { [Math]::Round(($dayData.Temperatures | Measure-Object -Maximum).Maximum, 1) } else { $null }
                    LowTemp = if ($dayData.Temperatures.Count -gt 0) { [Math]::Round(($dayData.Temperatures | Measure-Object -Minimum).Minimum, 1) } else { $null }
                    AvgWindSpeed = if ($dayData.WindSpeeds.Count -gt 0) { [Math]::Round(($dayData.WindSpeeds | Measure-Object -Average).Average, 1) } else { $null }
                    MaxWindSpeed = if ($dayData.WindSpeeds.Count -gt 0) { [Math]::Round(($dayData.WindSpeeds | Measure-Object -Maximum).Maximum, 1) } else { $null }
                    MaxWindGust = if ($dayData.WindGusts.Count -gt 0) { [Math]::Round(($dayData.WindGusts | Measure-Object -Maximum).Maximum, 1) } else { $null }
                    WindDirection = if ($dayData.WindDirections.Count -gt 0) { [Math]::Round(($dayData.WindDirections | Measure-Object -Average).Average, 0) } else { $null }
                    AvgHumidity = if ($dayData.Humidities.Count -gt 0) { [Math]::Round(($dayData.Humidities | Measure-Object -Average).Average, 1) } else { $null }
                    TotalPrecipitation = if ($dayData.Precipitations.Count -gt 0) { [Math]::Round(($dayData.Precipitations | Measure-Object -Sum).Sum, 2) } else { 0 }
                    Conditions = if ($dayData.Conditions.Count -gt 0) { ($dayData.Conditions | Group-Object | Sort-Object Count -Descending | Select-Object -First 1).Name } else { "N/A" }
                    Pressure = if ($dayData.Pressures.Count -gt 0) { [Math]::Round(($dayData.Pressures | Measure-Object -Average).Average, 2) } else { $null }
                    CloudSummary = if ($dayData.CloudSummaries.Count -gt 0) {
                        $nonEmpty = $dayData.CloudSummaries | Where-Object { $_ -and $_.ToString().Trim() }
                        if ($nonEmpty) { ($nonEmpty | Group-Object | Sort-Object Count -Descending | Select-Object -First 1).Name } else { $null }
                    } else { $null }
                }
            } else {
                # No data for this day
                $result += @{
                    Date = $date
                    HighTemp = $null
                    LowTemp = $null
                    AvgWindSpeed = $null
                    MaxWindSpeed = $null
                    MaxWindGust = $null
                    WindDirection = $null
                    AvgHumidity = $null
                    TotalPrecipitation = 0
                    Conditions = "N/A"
                    Pressure = $null
                    CloudSummary = $null
                }
            }
        }
        
        Write-Verbose "Processed $($dailyData.Keys.Count) unique dates from observations"
        Write-Verbose "Result array contains $($result.Count) days"
        
        # Debug: Show what dates we're looking for vs what we found
        Write-Verbose "Dates in dailyData: $($dailyData.Keys -join ', ')"
        Write-Verbose "Dates in result: $(($result | ForEach-Object { $_.Date }) -join ', ')"
        
        # Return the result array (Show-Observations will filter out days with no data)
        return $result
    }
    catch {
        Write-Verbose "Error processing observations data: $($_.Exception.Message)"
        Write-Verbose "Error details: $($_.Exception.GetType().FullName) at line $($_.InvocationInfo.ScriptLineNumber)"
        Write-Verbose "Stack trace: $($_.ScriptStackTrace)"
        return $null
    }
}

# Helper function to fetch all observations with pagination support
function Get-AllObservationsWithPagination {
    param(
        [string]$ObservationsUrl,
        [hashtable]$Headers
    )
    
    try {
        # Collect all observations from all pages
        $allFeatures = @()
        $currentUrl = $ObservationsUrl
        $pageCount = 0
        $maxPages = 50  # Safety limit to prevent infinite loops
        
        while ($currentUrl -and $pageCount -lt $maxPages) {
            $pageCount++
            Write-Verbose "Fetching observations page ${pageCount}: $currentUrl"
            
            $observationsJob = Start-ApiJob -Url $currentUrl -Headers $Headers -JobName "Observations"
            Wait-Job -Job $observationsJob | Out-Null
            
            if ($observationsJob.State -ne 'Completed') {
                Write-Verbose "Failed to fetch observations page ${pageCount}: $($observationsJob | Receive-Job)"
                Remove-Job -Job $observationsJob -Force
                break
            }
            
            $observationsJson = $observationsJob | Receive-Job
            Remove-Job -Job $observationsJob
            
            if ([string]::IsNullOrWhiteSpace($observationsJson)) {
                Write-Verbose "Empty response from observations API page $pageCount"
                break
            }
            
            $observationsData = $observationsJson | ConvertFrom-Json
            
            # Add features from this page to our collection
            if ($observationsData.features) {
                $allFeatures += $observationsData.features
                Write-Verbose "Collected $($observationsData.features.Count) observations from page $pageCount (total: $($allFeatures.Count))"
            }
            
            # Check for next page
            $currentUrl = $null
            if ($observationsData.pagination -and $observationsData.pagination.next) {
                $currentUrl = $observationsData.pagination.next
                Write-Verbose "Found pagination link for next page"
            }
        }
        
        if ($allFeatures.Count -eq 0) {
            Write-Verbose "No observations collected from any page"
            return $null
        }
        
        Write-Verbose "Collected total of $($allFeatures.Count) observations from $pageCount page(s)"
        
        # Create a combined observations data object
        $combinedObservationsData = @{
            type = "FeatureCollection"
            features = $allFeatures
        }
        
        return $combinedObservationsData
    }
    catch {
        Write-Verbose "Error in Get-AllObservationsWithPagination: $($_.Exception.Message)"
        return $null
    }
}

# Function to fetch NWS observations (kept for backward compatibility, but now uses async pattern in Update-WeatherData)
function Get-NWSObservations {
    param(
        [object]$PointsData,
        [hashtable]$Headers,
        [string]$TimeZone
    )
    
    try {
        # Get observation stations from points data
        $observationStationsUrl = $PointsData.properties.observationStations
        if (-not $observationStationsUrl) {
            Write-Verbose "No observation stations URL found in points data"
            return $null
        }
        
        Write-Verbose "Fetching observation stations from: $observationStationsUrl"
        $stationsJob = Start-ApiJob -Url $observationStationsUrl -Headers $Headers -JobName "ObservationStations"
        Wait-Job -Job $stationsJob | Out-Null
        
        if ($stationsJob.State -ne 'Completed') {
            Write-Verbose "Failed to fetch observation stations: $($stationsJob | Receive-Job)"
            Remove-Job -Job $stationsJob -Force
            return $null
        }
        
        $stationsJson = $stationsJob | Receive-Job
        Remove-Job -Job $stationsJob
        
        if ([string]::IsNullOrWhiteSpace($stationsJson)) {
            Write-Verbose "Empty response from observation stations API"
            return $null
        }
        
        $stationsData = $stationsJson | ConvertFrom-Json
        
        # Get the first station ID
        if ($stationsData.features.Count -eq 0) {
            Write-Verbose "No observation stations found"
            return $null
        }
        
        $stationId = $stationsData.features[0].properties.stationIdentifier
        Write-Verbose "Using observation station: $stationId"
        
        # Calculate time range (last 7 days for observations)
        $endTime = Get-Date
        $startTime = $endTime.AddDays(-7)
        
        # Format times in ISO 8601 format for API
        $startTimeStr = $startTime.ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
        $endTimeStr = $endTime.ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
        
        # Fetch observations with pagination support
        $observationsUrl = "https://api.weather.gov/stations/$stationId/observations?start=$startTimeStr&end=$endTimeStr"
        Write-Verbose "GET: $observationsUrl"
        Write-Verbose "Fetching historical observations from NWS observation stations API"
        
        # Use helper function to fetch all observations with pagination
        $observationsData = Get-AllObservationsWithPagination -ObservationsUrl $observationsUrl -Headers $Headers
        
        if ($null -eq $observationsData) {
            return $null
        }
        
        # Use Convert-ObservationsData to process the observations
        return Convert-ObservationsData -ObservationsData $observationsData -TimeZone $TimeZone
    }
    catch {
        Write-Verbose "Error fetching observations: $($_.Exception.Message)"
        Write-Verbose "Error details: $($_.Exception.GetType().FullName) at line $($_.InvocationInfo.ScriptLineNumber)"
        Write-Verbose "Stack trace: $($_.ScriptStackTrace)"
        return $null
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
            $response = Invoke-RestMethod -Uri $url -Method Get -Headers $hdrs -ErrorAction Stop -TimeoutSec 30
            return $response | ConvertTo-Json -Depth 10
        }
        catch {
            $errorInfo = @{
                Error = $true
                Message = $_.Exception.Message
                InnerException = if ($_.Exception.InnerException) { $_.Exception.InnerException.Message } else { $null }
                StatusCode = if ($_.Exception.Response) { $_.Exception.Response.StatusCode } else { $null }
                StatusDescription = if ($_.Exception.Response) { $_.Exception.Response.StatusDescription } else { $null }
            }
            # Write error to error stream so it can be captured
            Write-Error -Message $errorInfo.Message -Exception $_.Exception
            # Also return as JSON for structured error handling
            return ($errorInfo | ConvertTo-Json -Compress)
        }
    } -ArgumentList $Url, $Headers
    
    return $job
}

# Function to refresh weather data with exponential backoff
# Implements retry logic with exponential backoff to handle temporary service unavailability
# - Maximum 10 retry attempts
# - Exponential delay: 1s, 2s, 4s, 8s, 16s, 32s, 64s, 128s, 256s, 512s
# - Clears screen between retries for clean user experience
# - Gracefully exits with "Service Unavailable" message after max retries
function Update-WeatherData {
    param(
        [string]$Lat,
        [string]$Lon,
        [hashtable]$Headers,
        [string]$TimeZone,
        [bool]$UseRetryLogic = $true
    )
    
    $maxRetries = if ($UseRetryLogic) { 10 } else { 1 }  # Only retry if UseRetryLogic is true
    $baseDelay = 1  # Start with 1 second
    $retryCount = 0
    
    while ($retryCount -lt $maxRetries) {
    try {
            if ($retryCount -gt 0) {
                Clear-HostWithDelay
                Write-Host "Retrying weather data refresh... (Attempt $($retryCount + 1)/$maxRetries)" -ForegroundColor Yellow
            } else {
        Write-Host "Refreshing weather data..." -ForegroundColor Yellow
            }
        
        # Re-fetch points data
        $pointsUrl = "https://api.weather.gov/points/$lat,$lon"
        $pointsJob = Start-ApiJob -Url $pointsUrl -Headers $headers -JobName "PointsData"
        
        # Wait for job with timeout
        $jobResult = Wait-Job -Job $pointsJob -Timeout 30
        if (-not $jobResult) {
            Stop-Job -Job $pointsJob
            Remove-Job -Job $pointsJob -Force
            throw "Points data job timed out after 30 seconds"
        }
        
        if ($pointsJob.State -eq 'Failed') {
            $errorMsg = try { 
                $pointsJob | Receive-Job -ErrorAction SilentlyContinue
                "Job failed with state: $($pointsJob.State)"
            } catch { 
                "Job failed with state: $($pointsJob.State)" 
            }
            Remove-Job -Job $pointsJob -Force
            throw "Points data job failed: $errorMsg"
        } elseif ($pointsJob.State -ne 'Completed') { 
            $errorMsg = try { $pointsJob | Receive-Job } catch { "Job failed with state: $($pointsJob.State)" }
            Remove-Job -Job $pointsJob -Force
            throw "Points data job failed: $errorMsg"
        }
        
        $pointsJson = $pointsJob | Receive-Job
        Remove-Job -Job $pointsJob
        
        if ([string]::IsNullOrWhiteSpace($pointsJson)) { 
            throw "Empty response from points API"
        }
        
        $pointsData = $pointsJson | ConvertFrom-Json
        
        # Update script-scoped points data for observations refresh
        $script:pointsDataForObservations = $pointsData
        if ($TimeZone) {
            $script:timeZoneForObservations = $TimeZone
        }
        
        # Extract grid information
        $office = $pointsData.properties.cwa
        $gridX = $pointsData.properties.gridX
        $gridY = $pointsData.properties.gridY
            
            # Extract radar station
            $script:radarStation = $pointsData.properties.radarStation
            Write-Verbose "Radar Station: $script:radarStation"
        
        # Use URLs directly from Points API response (same as initial load)
        # This ensures we use the exact URLs the API provides, avoiding potential URL construction issues
        $forecastUrl = $pointsData.properties.forecast
        $hourlyUrl = $pointsData.properties.forecastHourly
        
        $forecastJob = Start-ApiJob -Url $forecastUrl -Headers $headers -JobName "ForecastData"
        $hourlyJob = Start-ApiJob -Url $hourlyUrl -Headers $headers -JobName "HourlyData"
        
        # Start alerts job in parallel
        $alertsUrl = "https://api.weather.gov/alerts/active?point=$lat,$lon"
        Write-Verbose "GET: $alertsUrl"
        $alertsJob = Start-ApiJob -Url $alertsUrl -Headers $headers -JobName "AlertsData"
        
        # Start observations stations job in parallel if timezone is provided
        $stationsJob = $null
        if ($TimeZone -and $pointsData.properties.observationStations) {
            $observationStationsUrl = $pointsData.properties.observationStations
            Write-Verbose "Fetching observation stations from: $observationStationsUrl"
            $stationsJob = Start-ApiJob -Url $observationStationsUrl -Headers $headers -JobName "ObservationStations"
        }
        
        $jobsToWaitFor = @($forecastJob, $hourlyJob, $alertsJob)
        if ($stationsJob) {
            $jobsToWaitFor += $stationsJob
        }
        
        # Wait for jobs (same as initial load - wait indefinitely, no timeout)
        # Jobs have their own 30-second timeout in Start-ApiJob
        Wait-Job -Job $jobsToWaitFor | Out-Null
        
        # Check forecast job - allow partial failure
        $forecastData = $null
        $forecastFailed = $false
        if ($forecastJob.State -eq 'Failed') {
            # Capture detailed error information from job
            $errorDetails = @()
            try {
                # First, try to get output (might contain error JSON)
                $jobOutput = $forecastJob | Receive-Job -ErrorVariable jobErrors -ErrorAction SilentlyContinue 2>&1
                
                # Check if output contains error JSON
                foreach ($item in $jobOutput) {
                    if ($item -is [System.Management.Automation.ErrorRecord]) {
                        $errorDetails += $item.Exception.Message
                        if ($item.Exception.InnerException) {
                            $errorDetails += "Inner: $($item.Exception.InnerException.Message)"
                        }
                        if ($item.Exception.Response) {
                            $errorDetails += "Response: $($item.Exception.Response.StatusCode) $($item.Exception.Response.StatusDescription)"
                        }
                    } elseif ($item) {
                        $str = $item.ToString()
                        # Check if it's a JSON error object
                        try {
                            $errorObj = $str | ConvertFrom-Json -ErrorAction SilentlyContinue
                            if ($errorObj.Error) {
                                $errorDetails += "Message: $($errorObj.Message)"
                                if ($errorObj.InnerException) {
                                    $errorDetails += "Inner: $($errorObj.InnerException)"
                                }
                                if ($errorObj.StatusCode) {
                                    $errorDetails += "HTTP: $($errorObj.StatusCode) $($errorObj.StatusDescription)"
                                }
                            } else {
                                if ($str -and $str -ne "") {
                                    $errorDetails += $str
                                }
                            }
                        } catch {
                            if ($str -and $str -ne "") {
                                $errorDetails += $str
                            }
                        }
                    }
                }
                
                # Also check error stream
                if ($jobErrors) {
                    foreach ($err in $jobErrors) {
                        $errorDetails += $err.Exception.Message
                        if ($err.Exception.InnerException) {
                            $errorDetails += "Inner: $($err.Exception.InnerException.Message)"
                        }
                    }
                }
                
                # Check job's error collection
                if ($forecastJob.ChildJobs) {
                    foreach ($childJob in $forecastJob.ChildJobs) {
                        if ($childJob.Error) {
                            foreach ($err in $childJob.Error) {
                                $errorDetails += $err.Exception.Message
                                if ($err.Exception.InnerException) {
                                    $errorDetails += "Inner: $($err.Exception.InnerException.Message)"
                                }
                            }
                        }
                    }
                }
            } catch {
                $errorDetails += "Error capturing job error: $($_.Exception.Message)"
            }
            
            $errorMsg = if ($errorDetails.Count -gt 0) {
                ($errorDetails | Select-Object -First 5) -join "; "
            } else {
                "Job failed with state: $($forecastJob.State) - No error details captured"
            }
            
            Write-Verbose "Forecast API failed for location: $lat,$lon (Office: $office, Grid: $gridX,$gridY) - URL: $forecastUrl - Error: $errorMsg"
            Remove-Job -Job $forecastJob -Force
            $forecastFailed = $true
        } elseif ($forecastJob.State -ne 'Completed') {
            Write-Verbose "Forecast API failed for location: $lat,$lon (Office: $office, Grid: $gridX,$gridY) - URL: $forecastUrl - Job in unexpected state: $($forecastJob.State)"
            Remove-Job -Job $forecastJob -Force
            $forecastFailed = $true
        } else {
            $forecastJson = $forecastJob | Receive-Job
            if ([string]::IsNullOrWhiteSpace($forecastJson)) { 
                Write-Verbose "Forecast API failed for location: $lat,$lon (Office: $office, Grid: $gridX,$gridY) - URL: $forecastUrl - Empty response from forecast API"
                Remove-Job -Job $forecastJob -Force
                $forecastFailed = $true
            } else {
                $forecastData = $forecastJson | ConvertFrom-Json
                Write-Verbose "Forecast data retrieved successfully"
                
                # Extract elevation from forecast data
                $script:elevationMeters = $forecastData.properties.elevation.value
                $script:elevationFeet = [math]::Round($script:elevationMeters * 3.28084, 0)
                Write-Verbose "Elevation: $script:elevationMeters meters ($script:elevationFeet feet)"
            }
        }
        
        # Check hourly job - allow partial failure
        $hourlyData = $null
        $hourlyFailed = $false
        if ($hourlyJob.State -eq 'Failed') {
            # Capture detailed error information from job
            $errorDetails = @()
            try {
                # First, try to get output (might contain error JSON)
                $jobOutput = $hourlyJob | Receive-Job -ErrorVariable jobErrors -ErrorAction SilentlyContinue 2>&1
                
                # Check if output contains error JSON
                foreach ($item in $jobOutput) {
                    if ($item -is [System.Management.Automation.ErrorRecord]) {
                        $errorDetails += $item.Exception.Message
                        if ($item.Exception.InnerException) {
                            $errorDetails += "Inner: $($item.Exception.InnerException.Message)"
                        }
                        if ($item.Exception.Response) {
                            $errorDetails += "Response: $($item.Exception.Response.StatusCode) $($item.Exception.Response.StatusDescription)"
                        }
                    } elseif ($item) {
                        $str = $item.ToString()
                        # Check if it's a JSON error object
                        try {
                            $errorObj = $str | ConvertFrom-Json -ErrorAction SilentlyContinue
                            if ($errorObj.Error) {
                                $errorDetails += "Message: $($errorObj.Message)"
                                if ($errorObj.InnerException) {
                                    $errorDetails += "Inner: $($errorObj.InnerException)"
                                }
                                if ($errorObj.StatusCode) {
                                    $errorDetails += "HTTP: $($errorObj.StatusCode) $($errorObj.StatusDescription)"
                                }
                            } else {
                                if ($str -and $str -ne "") {
                                    $errorDetails += $str
                                }
                            }
                        } catch {
                            if ($str -and $str -ne "") {
                                $errorDetails += $str
                            }
                        }
                    }
                }
                
                # Also check error stream
                if ($jobErrors) {
                    foreach ($err in $jobErrors) {
                        $errorDetails += $err.Exception.Message
                        if ($err.Exception.InnerException) {
                            $errorDetails += "Inner: $($err.Exception.InnerException.Message)"
                        }
                    }
                }
                
                # Check job's error collection
                if ($hourlyJob.ChildJobs) {
                    foreach ($childJob in $hourlyJob.ChildJobs) {
                        if ($childJob.Error) {
                            foreach ($err in $childJob.Error) {
                                $errorDetails += $err.Exception.Message
                                if ($err.Exception.InnerException) {
                                    $errorDetails += "Inner: $($err.Exception.InnerException.Message)"
                                }
                            }
                        }
                    }
                }
            } catch {
                $errorDetails += "Error capturing job error: $($_.Exception.Message)"
            }
            
            $errorMsg = if ($errorDetails.Count -gt 0) {
                ($errorDetails | Select-Object -First 5) -join "; "
            } else {
                "Job failed with state: $($hourlyJob.State) - No error details captured"
            }
            
            Write-Verbose "Hourly API failed for location: $lat,$lon (Office: $office, Grid: $gridX,$gridY) - URL: $hourlyUrl - Error: $errorMsg"
            Remove-Job -Job $hourlyJob -Force
            $hourlyFailed = $true
        } elseif ($hourlyJob.State -ne 'Completed') {
            Write-Verbose "Hourly API failed for location: $lat,$lon (Office: $office, Grid: $gridX,$gridY) - URL: $hourlyUrl - Job in unexpected state: $($hourlyJob.State)"
            Remove-Job -Job $hourlyJob -Force
            $hourlyFailed = $true
        } else {
            $hourlyJson = $hourlyJob | Receive-Job
            if ([string]::IsNullOrWhiteSpace($hourlyJson)) { 
                Write-Verbose "Hourly API failed for location: $lat,$lon (Office: $office, Grid: $gridX,$gridY) - URL: $hourlyUrl - Empty response from hourly API"
                Remove-Job -Job $hourlyJob -Force
                $hourlyFailed = $true
            } else {
                $hourlyData = $hourlyJson | ConvertFrom-Json
                Write-Verbose "Hourly data retrieved successfully"
            }
        }
        
        # Process alerts job - allow partial failure
        $alertsData = $null
        if ($alertsJob.State -eq 'Failed') {
            Write-Verbose "Alerts API failed - continuing without alerts"
            Remove-Job -Job $alertsJob -Force
        } elseif ($alertsJob.State -ne 'Completed') {
            Write-Verbose "Alerts API job in unexpected state: $($alertsJob.State) - continuing without alerts"
            Remove-Job -Job $alertsJob -Force
        } else {
            $alertsJson = $alertsJob | Receive-Job
            Remove-Job -Job $alertsJob
            if (-not [string]::IsNullOrWhiteSpace($alertsJson)) {
                try {
                    $alertsData = $alertsJson | ConvertFrom-Json
                    Write-Verbose "Alerts data retrieved successfully"
                } catch {
                    Write-Verbose "Failed to parse alerts data: $($_.Exception.Message)"
                }
            }
        }
        
        # Process stations job and fetch observations data if timezone is provided
        $stationId = $null
        if ($stationsJob) {
            if ($stationsJob.State -eq 'Failed') {
                Write-Verbose "Observation stations API failed - continuing without observations"
                Remove-Job -Job $stationsJob -Force
            } elseif ($stationsJob.State -ne 'Completed') {
                Write-Verbose "Observation stations job in unexpected state: $($stationsJob.State) - continuing without observations"
                Remove-Job -Job $stationsJob -Force
            } else {
                $stationsJson = $stationsJob | Receive-Job
                Remove-Job -Job $stationsJob
                if (-not [string]::IsNullOrWhiteSpace($stationsJson)) {
                    try {
                        $stationsData = $stationsJson | ConvertFrom-Json
                        if ($stationsData.features.Count -gt 0) {
                            $stationId = $stationsData.features[0].properties.stationIdentifier
                            Write-Verbose "Using observation station: $stationId"
                            
                            # Calculate time range (last 7 days for observations)
                            $endTime = Get-Date
                            $startTime = $endTime.AddDays(-7)
                            
                            # Format times in ISO 8601 format for API
                            $startTimeStr = $startTime.ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
                            $endTimeStr = $endTime.ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
                            
                            # Fetch observations with pagination support
                            $observationsUrl = "https://api.weather.gov/stations/$stationId/observations?start=$startTimeStr&end=$endTimeStr"
                            Write-Verbose "GET: $observationsUrl"
                            Write-Verbose "Fetching historical observations from NWS observation stations API"
                            
                            # Use helper function to fetch all observations with pagination
                            $observationsData = Get-AllObservationsWithPagination -ObservationsUrl $observationsUrl -Headers $headers
                            
                            if ($null -ne $observationsData) {
                                Write-Verbose "Processing $($observationsData.features.Count) observations"
                                $script:observationsData = Convert-ObservationsData -ObservationsData $observationsData -TimeZone $TimeZone
                                if ($null -ne $script:observationsData) {
                                    Write-Verbose "Observations data processed successfully"
                                } else {
                                    Write-Verbose "Observations data processing returned null"
                                }
                            } else {
                                Write-Verbose "No observations data collected"
                                $script:observationsData = $null
                            }
                            
                            # Observations processed inline with pagination support
                        } else {
                            Write-Verbose "No observation stations found"
                        }
                    } catch {
                        Write-Verbose "Failed to parse stations data: $($_.Exception.Message)"
                    }
                }
            }
        }
        
        # Observations are now processed inline above with pagination support
        # No need to wait for or process a job here
        if ($TimeZone -and $pointsData -and $null -eq $script:observationsData) {
            # If stations job wasn't started but timezone is provided, set observations to null
            $script:observationsData = $null
        }
        
        # Update global variables (only if data is available)
        if ($forecastData) {
            $script:forecastData = $forecastData
        }
        if ($hourlyData) {
            $script:hourlyData = $hourlyData
        }
        $script:alertsData = $alertsData
        
        # Update current conditions (only if hourly data is available)
        if ($hourlyData) {
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
            # Recalculate rising/falling trend from current vs next hour
            $script:currentTempTrend = $null
            $hourlyPeriods = $hourlyData.properties.periods
            if ($hourlyPeriods.Count -gt 1) {
                $nextHourPeriod = $hourlyPeriods[1]
                $nextHourTemp = $nextHourPeriod.temperature
                $tempDiff = [double]$nextHourTemp - [double]$script:currentTemp
                if ($tempDiff -gt 0.1) { $script:currentTempTrend = "rising" }
                elseif ($tempDiff -lt -0.1) { $script:currentTempTrend = "falling" }
                else { $script:currentTempTrend = "steady" }
            } else {
                $script:currentTempTrend = "steady"
            }
        }
        
        # Update forecast data (only if forecast data is available)
        if ($forecastData) {
            $script:todayPeriod = $forecastData.properties.periods[0]
            $script:todayForecast = $todayPeriod.detailedForecast
            $script:todayPeriodName = $todayPeriod.name
            
            $script:tomorrowPeriod = $forecastData.properties.periods[1]
            $script:tomorrowForecast = $tomorrowPeriod.detailedForecast
            $script:tomorrowPeriodName = $tomorrowPeriod.name
        }
        
        # Update fetch time
        $script:dataFetchTime = Get-Date
        
        # Refresh NOAA station data in parallel (non-blocking, don't fail if it errors)
        try {
            # Start fetching NOAA stations.json in parallel
            $noaaStationsJob = Start-Job -ScriptBlock {
                param($lat, $lon)
                [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
                try {
                    $apiUrl = "https://api.tidesandcurrents.noaa.gov/mdapi/prod/webapi/stations.json"
                    Write-Verbose "NOAA Tide API call: GET $apiUrl"
                    $apiResponse = Invoke-RestMethod -Uri $apiUrl -Method Get -ErrorAction Stop -TimeoutSec 30
                    $stationsCount = if ($apiResponse.stations) { $apiResponse.stations.Count } else { 0 }
                    Write-Verbose "NOAA Tide Stations API response received successfully: $stationsCount stations"
                    return @{
                        Success = $true
                        Stations = $apiResponse.stations
                        Lat = $lat
                        Lon = $lon
                    }
                } catch {
                    Write-Verbose "NOAA Tide Stations API call failed: $($_.Exception.Message)"
                    return @{
                        Success = $false
                        Error = $_.Exception.Message
                        Lat = $lat
                        Lon = $lon
                    }
                }
            } -ArgumentList $lat, $lon
            
            # Wait briefly for NOAA stations (non-blocking - don't delay refresh)
            $jobResult = Wait-Job -Job $noaaStationsJob -Timeout 2
            if ($jobResult) {
                $script:noaaStationsData = $noaaStationsJob | Receive-Job
                Remove-Job -Job $noaaStationsJob
                if ($script:noaaStationsData -and $script:noaaStationsData.Success) {
                    Write-Verbose "NOAA stations data refreshed successfully"
                    # Process and store NOAA station
                    if ($Noaa -and $Noaa.Trim() -ne "") {
                        # Use override station ID
                        $script:noaaStation = Get-NoaaTideStationById -StationId $Noaa.Trim() -Lat $lat -Lon $lon
                    } else {
                        # Normal station selection
                        $script:noaaStation = Get-NoaaTideStation -Lat $lat -Lon $lon -PreFetchedStations $script:noaaStationsData.Stations
                    }
                    if ($script:noaaStation) {
                        # Fetch tide predictions for the station
                        try {
                            $script:noaaStation.tideData = Get-NoaaTidePredictions -StationId $script:noaaStation.stationId -TimeZone $TimeZone
                        } catch {
                            Write-Verbose "Error refreshing tide predictions: $($_.Exception.Message)"
                        }
                    }
                }
            } else {
                # Job still running - remove any previous stored job to avoid leak, then store new reference
                if ($null -ne $script:noaaStationsJob) {
                    Remove-Job -Job $script:noaaStationsJob -Force -ErrorAction SilentlyContinue
                    $script:noaaStationsJob = $null
                }
                $script:noaaStationsJob = $noaaStationsJob
                Write-Verbose "NOAA stations refresh still in progress (non-blocking)"
            }
        } catch {
            Write-Verbose "Error refreshing NOAA station data: $($_.Exception.Message)"
            # Continue without NOAA data - don't fail the refresh
        }
        
        # Determine if refresh was successful
        # Success if we got points data (critical) and at least one of forecast or hourly data
        $hasPointsData = $true  # We already validated points data above
        $hasForecastData = $null -ne $forecastData
        $hasHourlyData = $null -ne $hourlyData
        
        # Show warnings for partial failures (only if data was previously available)
        if ($forecastFailed -and $null -ne $script:forecastData) {
            Write-Host "Warning: Forecast data unavailable - displaying previous forecast" -ForegroundColor Yellow
        }
        if ($hourlyFailed -and $null -ne $script:hourlyData) {
            Write-Host "Warning: Hourly data unavailable - displaying previous hourly forecast" -ForegroundColor Yellow
        }
        
        if ($hasPointsData -and ($hasForecastData -or $hasHourlyData)) {
            if ($hasForecastData -and $hasHourlyData) {
                Write-Host "Data refreshed successfully" -ForegroundColor Green
            }
            Write-Verbose "Refresh success: Points=$hasPointsData, Forecast=$hasForecastData, Hourly=$hasHourlyData"
            return $true
        } else {
            Write-Host "Partial data refresh - some services unavailable" -ForegroundColor Yellow
            Write-Verbose "Partial refresh: Points=$hasPointsData, Forecast=$hasForecastData, Hourly=$hasHourlyData"
            return $true  # Still consider this a success since we have some data
        }
    }
    catch {
            $retryCount++
            
            if (-not $UseRetryLogic) {
                # For auto-refresh calls, fail fast without retries
                Write-Verbose "Update-WeatherData failed (no retry): $($_.Exception.Message)"
                return $false
            }
            
            if ($retryCount -ge $maxRetries) {
                Clear-HostWithDelay
                Write-Host "Service Unavailable" -ForegroundColor Red
                Write-Host "The weather service is currently unavailable. Please try again later." -ForegroundColor Yellow
                Write-Host "Press any key to exit..." -ForegroundColor Cyan
                $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
                exit 1
            }
            
            # Calculate exponential backoff delay: 1, 2, 4, 8, 16, 32, 64, 128, 256, 512 seconds
            $delay = [Math]::Min($baseDelay * [Math]::Pow(2, $retryCount - 1), 512)
        Write-Host "Failed to refresh data: $($_.Exception.Message)" -ForegroundColor Red
            Write-Host "Retrying in $delay seconds..." -ForegroundColor Yellow
            Start-Sleep -Seconds $delay
        }
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
            if ($VerbosePreference -ne 'Continue') { 
                Clear-Host 
            }
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
            if ($VerbosePreference -ne 'Continue') {
                Clear-Host
            }
            Write-Host "Detecting location..." -ForegroundColor Yellow
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
            
            # Display geocoding message
            if ($VerbosePreference -ne 'Continue') {
                Clear-Host
            }
            Write-Host "Geocoding ($Location)..." -ForegroundColor Yellow
            
            # Prepare location for API query
            $locationForApi = if ($Location -match ",") { $Location } else { "$Location,US" }
            $encodedLocation = [uri]::EscapeDataString($locationForApi)
            $geoUrl = "https://nominatim.openstreetmap.org/search?q=$encodedLocation&format=json&limit=1&countrycodes=us&addressdetails=1"
            Write-Verbose "Geocoding URL: $geoUrl"
            
            $nominatimHeaders = @{
                "User-Agent" = "GetForecast/1.0 (Weather Script)"
            }
            $geoData = Invoke-RestMethod -Uri "$geoUrl" -Headers $nominatimHeaders -ErrorAction Stop
            if ($geoData.Count -eq 0) { throw "No geocoding results found for '$Location'." }
            
            # Log Nominatim response for debugging
            Write-Verbose "Nominatim API response received successfully"
            Write-Verbose "Response type: $($geoData[0].type)"
            Write-Verbose "Response name: $($geoData[0].name)"
            Write-Verbose "Response display_name: $($geoData[0].display_name)"
            if ($geoData[0].address) {
                Write-Verbose "Address object present with fields:"
                $geoData[0].address.PSObject.Properties | ForEach-Object {
                    Write-Verbose "  address.$($_.Name) = $($_.Value)"
                }
            } else {
                Write-Verbose "No address object in response"
            }
            
            $lat = [double]$geoData[0].lat
            $lon = [double]$geoData[0].lon
            
            # Extract city and state from address object (now that we have addressdetails=1)
            $city = $null
            $state = $null
            
            if ($geoData[0].address) {
                # Extract city from address object (prioritize city over neighborhood/suburb)
                if ($geoData[0].address.city) {
                    $city = $geoData[0].address.city
                    Write-Verbose "Found city from address.city: $city"
                } elseif ($geoData[0].address.town) {
                    $city = $geoData[0].address.town
                    Write-Verbose "Found city from address.town: $city"
                } elseif ($geoData[0].address.village) {
                    $city = $geoData[0].address.village
                    Write-Verbose "Found city from address.village: $city"
                } elseif ($geoData[0].address.municipality) {
                    $city = $geoData[0].address.municipality
                    Write-Verbose "Found city from address.municipality: $city"
                }
                
                # Extract state from address object
                if ($geoData[0].address.state_code -and $geoData[0].address.state_code.Length -eq 2) {
                    $state = $geoData[0].address.state_code.ToUpper()
                    Write-Verbose "Found state from address.state_code: $state"
                } elseif ($geoData[0].address.state) {
                    $stateName = $geoData[0].address.state
                    Write-Verbose "Found state name from address.state: $stateName"
                    # Map full state names to abbreviations
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
                        "District of Columbia" = "DC"
                    }
                    if ($stateMap.ContainsKey($stateName)) {
                        $state = $stateMap[$stateName]
                        Write-Verbose "Mapped state name '$stateName' to abbreviation: $state"
                    }
                }
            }
            
            # Fallback: Parse from display_name if address object didn't provide needed fields
            if (-not $city) {
                $displayName = $geoData[0].display_name
                Write-Verbose "Parsing city from display_name (fallback): $displayName"
                
                # Simple fallback: try to extract from display_name before state
                $stateNames = @("Alabama", "Alaska", "Arizona", "Arkansas", "California", "Colorado", "Connecticut", 
                               "Delaware", "Florida", "Georgia", "Hawaii", "Idaho", "Illinois", "Indiana", "Iowa",
                               "Kansas", "Kentucky", "Louisiana", "Maine", "Maryland", "Massachusetts", "Michigan", 
                               "Minnesota", "Mississippi", "Missouri", "Montana", "Nebraska", "Nevada", "New Hampshire", 
                               "New Jersey", "New Mexico", "New York", "North Carolina", "North Dakota", "Ohio",
                               "Oklahoma", "Oregon", "Pennsylvania", "Rhode Island", "South Carolina", "South Dakota", 
                               "Tennessee", "Texas", "Utah", "Vermont", "Virginia", "Washington", "West Virginia", 
                               "Wisconsin", "Wyoming")
                
                foreach ($stateName in $stateNames) {
                    if ($displayName -match ", ([^,]+), $stateName,") {
                        $potentialCity = $matches[1].Trim()
                        if (-not ($potentialCity -match "County$") -and -not ($stateNames -contains $potentialCity)) {
                            $city = $potentialCity
                            Write-Verbose "Found city from display_name (before state '$stateName'): $city"
                            break
                        }
                    }
                }
                
                # Final fallback: use name field
                if (-not $city) {
                    $city = $geoData[0].name
                    Write-Verbose "Using name field as fallback: $city"
                }
            }
            
            # Fallback: Parse state from display_name if address object didn't provide it
            if (-not $state -or $state -eq "US") {
                $displayName = $geoData[0].display_name
                Write-Verbose "Parsing state from display_name (fallback): $displayName"
                
                if ($displayName -match ", ([A-Z]{2}), United States$") {
                    $state = $matches[1]
                } elseif ($displayName -match ", ([A-Z]{2})$") {
                    $state = $matches[1]
                } else {
                    # Try to extract full state name from display_name and map it
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
                        "District of Columbia" = "DC"
                    }
                    foreach ($stateName in $stateMap.Keys) {
                        if ($displayName -match ", $([regex]::Escape($stateName)),") {
                            $state = $stateMap[$stateName]
                            Write-Verbose "Mapped state name '$stateName' from display_name to abbreviation: $state"
                            break
                        }
                    }
                }
                
                # Final fallback: extract state from original input if it contains a comma
                if (-not $state -or $state -eq "US") {
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

# --- START NOAA STATIONS FETCH IN PARALLEL (doesn't depend on NWS data) ---
Write-Verbose "Starting NOAA stations.json fetch in parallel"
$noaaStationsJob = Start-Job -ScriptBlock {
    param($lat, $lon)
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    try {
        $apiUrl = "https://api.tidesandcurrents.noaa.gov/mdapi/prod/webapi/stations.json"
        Write-Verbose "NOAA Tide API call: GET $apiUrl"
        $apiResponse = Invoke-RestMethod -Uri $apiUrl -Method Get -ErrorAction Stop -TimeoutSec 30
        $stationsCount = if ($apiResponse.stations) { $apiResponse.stations.Count } else { 0 }
        Write-Verbose "NOAA Tide Stations API response received successfully: $stationsCount stations"
        return @{
            Success = $true
            Stations = $apiResponse.stations
            Lat = $lat
            Lon = $lon
        }
    } catch {
        Write-Verbose "NOAA Tide Stations API call failed: $($_.Exception.Message)"
        return @{
            Success = $false
            Error = $_.Exception.Message
            Lat = $lat
            Lon = $lon
        }
    }
} -ArgumentList $lat, $lon

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

# Extract additional location information
$timeZone = $pointsData.properties.timeZone
$radarStation = $pointsData.properties.radarStation

Write-Verbose "Grid info: Office=$office (CWA), GridX=$gridX, GridY=$gridY"
Write-Verbose "Location info: TimeZone=$timeZone, RadarStation=$radarStation"
Write-Verbose "Forecast URLs extracted from Points API response"

# --- CONCURRENTLY FETCH FORECAST AND HOURLY DATA ---
Write-Verbose "Starting API calls for forecast data."
Write-Verbose "GET: $forecastUrl"
Write-Verbose "GET: $hourlyUrl"

# Display loading messages for forecast and hourly data
$locationDisplay = if ($state) { "$city, $state" } else { $city }
if ($VerbosePreference -ne 'Continue') {
    Clear-Host
}
Write-Host "Calling API for $locationDisplay Forecast..." -ForegroundColor Cyan

$forecastJob = Start-ApiJob -Url $forecastUrl -Headers $headers -JobName "ForecastData"

if ($VerbosePreference -ne 'Continue') {
    Clear-Host
}
Write-Host "Calling API for $locationDisplay Hourly..." -ForegroundColor Cyan

$hourlyJob = Start-ApiJob -Url $hourlyUrl -Headers $headers -JobName "HourlyData"
if ($VerbosePreference -ne 'Continue') {
    Clear-Host
}
Write-Host "Loading $locationDisplay Data..." -ForegroundColor Yellow


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

# Set script-scoped variable for refresh operations
$script:forecastData = $forecastData

# Extract elevation from forecast data
$elevationMeters = $forecastData.properties.elevation.value
$elevationFeet = [math]::Round($elevationMeters * 3.28084, 0)
Write-Verbose "Elevation: $elevationMeters meters ($elevationFeet feet)"

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

# Set script-scoped variable for refresh operations
$script:hourlyData = $hourlyData

Remove-Job -Job $jobsToWaitFor

# Preload observations data after initial load (for better UX when switching to Observations mode)
# Store points data and headers for later use
$script:observationsData = $null
$script:observationsDataLoading = $false
$script:observationsPreloadAttempted = $false
$script:pointsDataForObservations = $pointsData
$script:headersForObservations = $headers
$script:timeZoneForObservations = $timeZone

# Collect initial NOAA stations job (started earlier in parallel) to avoid job leak and use for Location Info
$noaaJobResult = Wait-Job -Job $noaaStationsJob -Timeout 15
if ($noaaJobResult) {
    $script:noaaStationsData = $noaaStationsJob | Receive-Job
    Remove-Job -Job $noaaStationsJob -Force
    if ($script:noaaStationsData -and $script:noaaStationsData.Success) {
        Write-Verbose "NOAA stations data loaded from initial fetch"
    }
} else {
    Remove-Job -Job $noaaStationsJob -Force
    Write-Verbose "NOAA stations initial job timed out or failed; will fetch on demand if needed"
}

if ($Observations.IsPresent) {
    # If Observations mode is requested at startup, fetch immediately
    Write-Verbose "Fetching observations data for Observations mode"
    Write-Host "Loading Historical Data..." -ForegroundColor Yellow
    $script:observationsData = Get-NWSObservations -PointsData $pointsData -Headers $headers -TimeZone $timeZone
    Clear-HostWithDelay
    if ($null -eq $script:observationsData) {
        Write-Verbose "Failed to fetch observations data"
    } else {
        Write-Verbose "Observations data retrieved successfully: $($script:observationsData.Count) days"
    }
}

# --- TIMER TRACKING FOR AUTO-REFRESH ---
$script:dataFetchTime = Get-Date
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

    # Ensure we always return an array, even if it's a single element
    return ,$lines
}

function Get-TruncatedCityName {
    param(
        [string]$CityName,
        [int]$MaxLength = 20
    )
    
    if ([string]::IsNullOrWhiteSpace($CityName)) {
        return ""
    }
    
    $words = $CityName -split ' '
    $result = $words[0]
    
    # If first word alone exceeds limit, return it anyway
    if ($result.Length -gt $MaxLength) {
        return $result
    }
    
    # Try to add more words while staying within limit
    for ($i = 1; $i -lt $words.Count; $i++) {
        $nextWord = $words[$i]
        $potentialLength = $result.Length + 1 + $nextWord.Length  # +1 for space
        
        if ($potentialLength -le $MaxLength) {
            $result += " " + $nextWord
        } else {
            break
        }
    }
    
    return $result
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
        # Polar night - no sunrise today, find next sunrise and last sunset
        $nextSunrise = $null
        $lastSunset = $null
        $maxDaysToCheck = 180  # Check up to 6 months ahead/back
        
        # Find next sunrise (forward)
        for ($dayOffset = 1; $dayOffset -le $maxDaysToCheck; $dayOffset++) {
            $checkDate = $Date.AddDays($dayOffset)
            $checkDayOfYear = $checkDate.DayOfYear
            $checkGamma = 2.0 * [Math]::PI * ($checkDayOfYear - 1) / 365.0
            $checkDeclination = 0.006918 - 0.399912 * [Math]::Cos($checkGamma) + 0.070257 * [Math]::Sin($checkGamma) - 0.006758 * [Math]::Cos(2*$checkGamma) + 0.000907 * [Math]::Sin(2*$checkGamma) - 0.002697 * [Math]::Cos(3*$checkGamma) + 0.00148 * [Math]::Sin(3*$checkGamma)
            $checkCosH = ([Math]::Cos((ToRadians $zenithDegrees)) - [Math]::Sin($latRad) * [Math]::Sin($checkDeclination)) / ([Math]::Cos($latRad) * [Math]::Cos($checkDeclination))
            
            if ($checkCosH -le 1) {
                # Found a day with sunrise
                $checkH = [Math]::Acos([Math]::Min(1.0, [Math]::Max(-1.0, $checkCosH)))
                $checkHdeg = ToDegrees $checkH
                $checkEquationOfTime = 229.18 * (0.000075 + 0.001868 * [Math]::Cos($checkGamma) - 0.032077 * [Math]::Sin($checkGamma) - 0.014615 * [Math]::Cos(2*$checkGamma) - 0.040849 * [Math]::Sin(2*$checkGamma))
                $checkSolarNoonUtcMin = 720.0 - 4.0 * $Longitude - $checkEquationOfTime
                $checkSunriseUtcMin = $checkSolarNoonUtcMin - 4.0 * $checkHdeg
                
                while ($checkSunriseUtcMin -lt 0) { $checkSunriseUtcMin += 1440 }
                while ($checkSunriseUtcMin -ge 1440) { $checkSunriseUtcMin -= 1440 }
                
                $checkUtcMidnight = [DateTime]::new($checkDate.Year, $checkDate.Month, $checkDate.Day, 0, 0, 0, [System.DateTimeKind]::Utc)
                $checkSunriseUtc = $checkUtcMidnight.AddMinutes($checkSunriseUtcMin)
                $tzInfo = Get-ResolvedTimeZoneInfo -TimeZoneId $TimeZoneId
                $nextSunrise = [System.TimeZoneInfo]::ConvertTimeFromUtc($checkSunriseUtc, $tzInfo)
                break
            }
        }
        
        # Find last sunset (backward)
        for ($dayOffset = -1; $dayOffset -ge -$maxDaysToCheck; $dayOffset--) {
            $checkDate = $Date.AddDays($dayOffset)
            $checkDayOfYear = $checkDate.DayOfYear
            $checkGamma = 2.0 * [Math]::PI * ($checkDayOfYear - 1) / 365.0
            $checkDeclination = 0.006918 - 0.399912 * [Math]::Cos($checkGamma) + 0.070257 * [Math]::Sin($checkGamma) - 0.006758 * [Math]::Cos(2*$checkGamma) + 0.000907 * [Math]::Sin(2*$checkGamma) - 0.002697 * [Math]::Cos(3*$checkGamma) + 0.00148 * [Math]::Sin(3*$checkGamma)
            $checkCosH = ([Math]::Cos((ToRadians $zenithDegrees)) - [Math]::Sin($latRad) * [Math]::Sin($checkDeclination)) / ([Math]::Cos($latRad) * [Math]::Cos($checkDeclination))
            
            if ($checkCosH -le 1) {
                # Found a day with sunset
                $checkH = [Math]::Acos([Math]::Min(1.0, [Math]::Max(-1.0, $checkCosH)))
                $checkHdeg = ToDegrees $checkH
                $checkEquationOfTime = 229.18 * (0.000075 + 0.001868 * [Math]::Cos($checkGamma) - 0.032077 * [Math]::Sin($checkGamma) - 0.014615 * [Math]::Cos(2*$checkGamma) - 0.040849 * [Math]::Sin(2*$checkGamma))
                $checkSolarNoonUtcMin = 720.0 - 4.0 * $Longitude - $checkEquationOfTime
                $checkSunsetUtcMin = $checkSolarNoonUtcMin + 4.0 * $checkHdeg
                
                while ($checkSunsetUtcMin -lt 0) { $checkSunsetUtcMin += 1440 }
                while ($checkSunsetUtcMin -ge 1440) { $checkSunsetUtcMin -= 1440 }
                
                $checkUtcMidnight = [DateTime]::new($checkDate.Year, $checkDate.Month, $checkDate.Day, 0, 0, 0, [System.DateTimeKind]::Utc)
                $checkSunsetUtc = $checkUtcMidnight.AddMinutes($checkSunsetUtcMin)
                $tzInfo = Get-ResolvedTimeZoneInfo -TimeZoneId $TimeZoneId
                $lastSunset = [System.TimeZoneInfo]::ConvertTimeFromUtc($checkSunsetUtc, $tzInfo)
                break
            }
        }
        
        return @{ Sunrise = $nextSunrise; Sunset = $lastSunset; IsPolarNight = $true; IsPolarDay = $false }
    }
    if ($cosH -lt -1) {
        # Polar day - no sunset today, find next sunset and last sunrise
        $nextSunset = $null
        $lastSunrise = $null
        $maxDaysToCheck = 180  # Check up to 6 months ahead/back
        
        # Find next sunset (forward)
        for ($dayOffset = 1; $dayOffset -le $maxDaysToCheck; $dayOffset++) {
            $checkDate = $Date.AddDays($dayOffset)
            $checkDayOfYear = $checkDate.DayOfYear
            $checkGamma = 2.0 * [Math]::PI * ($checkDayOfYear - 1) / 365.0
            $checkDeclination = 0.006918 - 0.399912 * [Math]::Cos($checkGamma) + 0.070257 * [Math]::Sin($checkGamma) - 0.006758 * [Math]::Cos(2*$checkGamma) + 0.000907 * [Math]::Sin(2*$checkGamma) - 0.002697 * [Math]::Cos(3*$checkGamma) + 0.00148 * [Math]::Sin(3*$checkGamma)
            $checkCosH = ([Math]::Cos((ToRadians $zenithDegrees)) - [Math]::Sin($latRad) * [Math]::Sin($checkDeclination)) / ([Math]::Cos($latRad) * [Math]::Cos($checkDeclination))
            
            if ($checkCosH -le 1) {
                # Found a day with sunset
                $checkH = [Math]::Acos([Math]::Min(1.0, [Math]::Max(-1.0, $checkCosH)))
                $checkHdeg = ToDegrees $checkH
                $checkEquationOfTime = 229.18 * (0.000075 + 0.001868 * [Math]::Cos($checkGamma) - 0.032077 * [Math]::Sin($checkGamma) - 0.014615 * [Math]::Cos(2*$checkGamma) - 0.040849 * [Math]::Sin(2*$checkGamma))
                $checkSolarNoonUtcMin = 720.0 - 4.0 * $Longitude - $checkEquationOfTime
                $checkSunsetUtcMin = $checkSolarNoonUtcMin + 4.0 * $checkHdeg
                
                while ($checkSunsetUtcMin -lt 0) { $checkSunsetUtcMin += 1440 }
                while ($checkSunsetUtcMin -ge 1440) { $checkSunsetUtcMin -= 1440 }
                
                $checkUtcMidnight = [DateTime]::new($checkDate.Year, $checkDate.Month, $checkDate.Day, 0, 0, 0, [System.DateTimeKind]::Utc)
                $checkSunsetUtc = $checkUtcMidnight.AddMinutes($checkSunsetUtcMin)
                $tzInfo = Get-ResolvedTimeZoneInfo -TimeZoneId $TimeZoneId
                $nextSunset = [System.TimeZoneInfo]::ConvertTimeFromUtc($checkSunsetUtc, $tzInfo)
                break
            }
        }
        
        # Find last sunrise (backward)
        for ($dayOffset = -1; $dayOffset -ge -$maxDaysToCheck; $dayOffset--) {
            $checkDate = $Date.AddDays($dayOffset)
            $checkDayOfYear = $checkDate.DayOfYear
            $checkGamma = 2.0 * [Math]::PI * ($checkDayOfYear - 1) / 365.0
            $checkDeclination = 0.006918 - 0.399912 * [Math]::Cos($checkGamma) + 0.070257 * [Math]::Sin($checkGamma) - 0.006758 * [Math]::Cos(2*$checkGamma) + 0.000907 * [Math]::Sin(2*$checkGamma) - 0.002697 * [Math]::Cos(3*$checkGamma) + 0.00148 * [Math]::Sin(3*$checkGamma)
            $checkCosH = ([Math]::Cos((ToRadians $zenithDegrees)) - [Math]::Sin($latRad) * [Math]::Sin($checkDeclination)) / ([Math]::Cos($latRad) * [Math]::Cos($checkDeclination))
            
            if ($checkCosH -le 1) {
                # Found a day with sunrise
                $checkH = [Math]::Acos([Math]::Min(1.0, [Math]::Max(-1.0, $checkCosH)))
                $checkHdeg = ToDegrees $checkH
                $checkEquationOfTime = 229.18 * (0.000075 + 0.001868 * [Math]::Cos($checkGamma) - 0.032077 * [Math]::Sin($checkGamma) - 0.014615 * [Math]::Cos(2*$checkGamma) - 0.040849 * [Math]::Sin(2*$checkGamma))
                $checkSolarNoonUtcMin = 720.0 - 4.0 * $Longitude - $checkEquationOfTime
                $checkSunriseUtcMin = $checkSolarNoonUtcMin - 4.0 * $checkHdeg
                
                while ($checkSunriseUtcMin -lt 0) { $checkSunriseUtcMin += 1440 }
                while ($checkSunriseUtcMin -ge 1440) { $checkSunriseUtcMin -= 1440 }
                
                $checkUtcMidnight = [DateTime]::new($checkDate.Year, $checkDate.Month, $checkDate.Day, 0, 0, 0, [System.DateTimeKind]::Utc)
                $checkSunriseUtc = $checkUtcMidnight.AddMinutes($checkSunriseUtcMin)
                $tzInfo = Get-ResolvedTimeZoneInfo -TimeZoneId $TimeZoneId
                $lastSunrise = [System.TimeZoneInfo]::ConvertTimeFromUtc($checkSunriseUtc, $tzInfo)
                break
            }
        }
        
        return @{ Sunrise = $lastSunrise; Sunset = $nextSunset; IsPolarNight = $false; IsPolarDay = $true }
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

# Helper function to format day length as "Xh Ym"
function Format-DayLength {
    param(
        [object]$Sunrise,
        [object]$Sunset
    )
    
    if ($null -eq $Sunrise -or $null -eq $Sunset) {
        return "N/A"
    }
    
    # Calculate day length: simply subtract sunrise from sunset
    # If sunset is earlier than sunrise, add 24 hours (sunset is next day)
    $duration = $Sunset - $Sunrise
    if ($duration.TotalMinutes -lt 0) {
        # Sunset is next day, add 24 hours (1440 minutes)
        $totalMinutes = [Math]::Round($duration.TotalMinutes) + (24 * 60)
    } else {
        $totalMinutes = [Math]::Round($duration.TotalMinutes)
    }
    
    $hours = [Math]::Floor($totalMinutes / 60)
    $minutes = $totalMinutes % 60
    
    return "${hours}h ${minutes}m"
}

# Function to calculate moon phase using simple astronomical method
# Uses reference new moon date (January 6, 2000 18:14 UTC) and lunar cycle of 29.53058867 days
# Returns moon phase name, emoji, and next full moon date for display in current conditions
function Get-MoonPhase {
    param([DateTime]$Date)
    
    $knownNewMoon = [DateTime]::new(2000, 1, 6, 18, 14, 0, [System.DateTimeKind]::Utc)
    $lunarCycle = 29.53058867
    
    # Calculate phase (0-1 range)
    $daysSince = ($Date.ToUniversalTime() - $knownNewMoon).TotalDays
    $currentCycle = $daysSince % $lunarCycle
    $phase = $currentCycle / $lunarCycle
    
    # Determine phase name and emoji using corrected astronomical method
    # Phase ranges: New (0-0.125), Waxing Crescent (0.125-0.25), First Quarter (0.25-0.375),
    # Waxing Gibbous (0.375-0.48), Full (0.48-0.52), Waning Gibbous (0.52-0.75),
    # Last Quarter (0.75-0.875), Waning Crescent (0.875-1.0)
    
    $phaseName = ""
    $emoji = ""
    
    if ($phase -lt 0.125) {
        $phaseName = "New Moon"
        $emoji = "🌑"
    } elseif ($phase -lt 0.25) {
        $phaseName = "Waxing Crescent"
        $emoji = "🌒"
    } elseif ($phase -lt 0.375) {
        $phaseName = "First Quarter"
        $emoji = "🌓"
    } elseif ($phase -lt 0.48) {
        $phaseName = "Waxing Gibbous"
        $emoji = "🌔"
    } elseif ($phase -lt 0.52) {
        $phaseName = "Full Moon"
        $emoji = "🌕"
    } elseif ($phase -lt 0.75) {
        $phaseName = "Waning Gibbous"
        $emoji = "🌖"
    } elseif ($phase -lt 0.875) {
        $phaseName = "Last Quarter"
        $emoji = "🌗"
    } else {
        $phaseName = "Waning Crescent"
        $emoji = "🌘"
    }
    
    # Calculate next full moon and new moon dates
    $isFullMoon = ($phase -ge 0.48 -and $phase -lt 0.52)
    $isNewMoon = ($phase -lt 0.125)
    $showNextFullMoon = ($phase -lt 0.48)  # Before Full Moon
    $showNextNewMoon = ($phase -ge 0.52)   # At/After Full Moon
    
    # Calculate next full moon
    $daysUntilNextFullMoon = (14.77 - $currentCycle) % $lunarCycle
    if ($daysUntilNextFullMoon -le 0) {
        $daysUntilNextFullMoon += $lunarCycle
    }
    $nextFullMoonDate = $Date.AddDays($daysUntilNextFullMoon).ToString("MM/dd/yyyy")
    
    # Calculate next new moon
    $daysUntilNextNewMoon = $lunarCycle - $currentCycle
    $nextNewMoonDate = $Date.AddDays($daysUntilNextNewMoon).ToString("MM/dd/yyyy")
    
    return @{
        Name = $phaseName
        Emoji = $emoji
        IsFullMoon = $isFullMoon
        IsNewMoon = $isNewMoon
        ShowNextFullMoon = $showNextFullMoon
        ShowNextNewMoon = $showNextNewMoon
        NextFullMoon = $nextFullMoonDate
        NextNewMoon = $nextNewMoonDate
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
# Calculate trend by comparing current temp to next hour temp (future-looking trend)
# Always use the actual calculated difference - never trust API's temperatureTrend when it contradicts the data
$currentTempTrend = $null
$hourlyPeriods = $hourlyData.properties.periods
if ($hourlyPeriods.Count -gt 1) {
    $nextHourPeriod = $hourlyPeriods[1]
    $nextHourTemp = $nextHourPeriod.temperature
    
    $tempDiff = [double]$nextHourTemp - [double]$currentTemp
    Write-Verbose "Temperature trend calculation: Current=$currentTemp°F, Next=$nextHourTemp°F, Diff=$tempDiff°F"
    
    # Always check the sign of the actual temperature difference first
    if ($tempDiff -gt 0.1) {
        # Next hour is warmer - temperature is rising
        $currentTempTrend = "rising"
        Write-Verbose "Calculated temperature trend (future-looking): $currentTempTrend"
    }
    elseif ($tempDiff -lt -0.1) {
        # Next hour is cooler - temperature is falling
        $currentTempTrend = "falling"
        Write-Verbose "Calculated temperature trend (future-looking): $currentTempTrend"
    }
    else {
        # Very small change (within 0.1 degrees) - temperature is steady
        $currentTempTrend = "steady"
        Write-Verbose "Small change detected ($tempDiff°F). Temperature is steady."
    }
} else {
    $currentTempTrend = "steady"
    Write-Verbose "Insufficient hourly data for trend calculation"
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

# Function: Calculate wind chill using NWS formula
function Get-WindChill {
    param(
        [double]$TempF,
        [double]$WindSpeedMph
    )
    
    # Wind chill only applies when temp <= 50°F and wind speed >= 3 mph
    if ($TempF -gt 50 -or $WindSpeedMph -lt 3) {
        return $null
    }
    
    # NWS Wind Chill Formula
    $windChill = 35.74 + (0.6215 * $TempF) - (35.75 * [Math]::Pow($WindSpeedMph, 0.16)) + (0.4275 * $TempF * [Math]::Pow($WindSpeedMph, 0.16))
    return [Math]::Round($windChill)
}

# Function: Calculate heat index using NWS Rothfusz regression
function Get-HeatIndex {
    param(
        [double]$TempF,
        [double]$Humidity
    )
    
    # Heat index only applies when temp >= 80°F
    if ($TempF -lt 80) {
        return $null
    }
    
    # NWS Heat Index Formula (Rothfusz regression)
    $T = $TempF
    $RH = $Humidity
    
    # Simple formula for initial estimate
    $HI = 0.5 * ($T + 61.0 + (($T - 68.0) * 1.2) + ($RH * 0.094))
    
    # If >= 80°F, use full Rothfusz regression
    if ($HI -ge 80) {
        $HI = -42.379 + (2.04901523 * $T) + (10.14333127 * $RH) - (0.22475541 * $T * $RH) - (0.00683783 * $T * $T) - (0.05481717 * $RH * $RH) + (0.00122874 * $T * $T * $RH) + (0.00085282 * $T * $RH * $RH) - (0.00000199 * $T * $T * $RH * $RH)
        
        # Adjustments for low/high RH
        if ($RH -lt 13 -and $T -ge 80 -and $T -le 112) {
            $HI = $HI - ((13 - $RH) / 4) * [Math]::Sqrt((17 - [Math]::Abs($T - 95)) / 17)
        }
        elseif ($RH -gt 85 -and $T -ge 80 -and $T -le 87) {
            $HI = $HI + (($RH - 85) / 10) * ((87 - $T) / 5)
        }
    }
    
    return [Math]::Round($HI)
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
        [object]$SunriseTime,
        [object]$SunsetTime
    )
    
    # Handle polar night/day cases where sunrise or sunset is null
    if ($null -eq $SunriseTime -or $null -eq $SunsetTime) {
        # Fallback to NWS API isDaytime property for polar regions
        if ($Period.PSObject.Properties['isDaytime']) {
            return $Period.isDaytime
        }
        # If no isDaytime property, use simple time-based heuristic (6 AM to 6 PM)
        $periodTime = [DateTime]::Parse($Period.startTime)
        $currentHour = $periodTime.Hour
        return $currentHour -ge 6 -and $currentHour -lt 18
    }
    
    # Parse the period start time
    $periodTime = [DateTime]::Parse($Period.startTime)
    
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
        [string]$InfoColor,
        [string]$MoonPhase,
        [string]$MoonEmoji,
        [bool]$IsFullMoon,
        [string]$NextFullMoonDate,
        [bool]$IsNewMoon,
        [bool]$ShowNextFullMoon,
        [bool]$ShowNextNewMoon,
        [string]$NextNewMoonDate
    )
    
    Write-Host "*** $city, $state Current Conditions ***" -ForegroundColor $TitleColor
    Write-Host "Currently: $weatherIcon $currentConditions" -ForegroundColor $DefaultColor
    Write-Host "Temperature: $currentTemp°F" -ForegroundColor $TempColor -NoNewline

    # Calculate and display wind chill or heat index
    $tempNum = [double]$currentTemp
    if ($tempNum -le 50) {
        $windSpeedNum = Get-WindSpeed $currentWind
        $windChill = Get-WindChill $tempNum $windSpeedNum
        if ($null -ne $windChill -and ($tempNum - $windChill) -gt 1) {
            Write-Host " [$windChill°F]" -ForegroundColor Blue -NoNewline
        }
    }
    elseif ($tempNum -ge 80) {
        $humidityNum = [double]$currentHumidity
        $heatIndex = Get-HeatIndex $tempNum $humidityNum
        if ($null -ne $heatIndex -and ($heatIndex - $tempNum) -gt 1) {
            Write-Host " [$heatIndex°F]" -ForegroundColor Red -NoNewline
        }
    }

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

    # Wind line and direction glyph use same color rules as wind modal (from Get-WindGlyph)
    $currentWindSpeed = Get-WindSpeed $currentWind
    $windGlyphData = Get-WindGlyph -WindDirection $currentWindDir -WindSpeed $currentWindSpeed
    Write-Host "Wind: $currentWind $currentWindDir " -ForegroundColor $windGlyphData.Color -NoNewline
    Write-Host $windGlyphData.Char -ForegroundColor $windGlyphData.Color -NoNewline
    if ($windGust) {
        Write-Host " (gusts to $windGust mph)" -ForegroundColor $AlertColor -NoNewline
    }
    Write-Host ""

    # Apply humidity color scheme based on comfort levels
    $humidityValue = [double]$currentHumidity
    $humidityColor = if ($humidityValue -lt 30) { "Cyan" }
                    elseif ($humidityValue -le 60) { "White" }
                    elseif ($humidityValue -le 70) { "Yellow" }
                    else { "Red" }
    
    Write-Host "Humidity: $currentHumidity%" -ForegroundColor $humidityColor
    
    # Display dew point if available
    if ($null -ne $currentDewPoint) {
        try {
            $dewPointCelsius = [double]$currentDewPoint
            $dewPointF = [math]::Round($dewPointCelsius * 9/5 + 32, 1)
            
            # Apply dew point color scheme based on comfort levels
            $dewPointColor = if ($dewPointF -lt 40) { "Cyan" }
                            elseif ($dewPointF -le 54) { "White" }
                            elseif ($dewPointF -le 64) { "Yellow" }
                            else { "Red" }
            
            Write-Host "Dew Point: $dewPointF°F" -ForegroundColor $dewPointColor
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
    
    # Display moon phase information
    if ($MoonPhase -and $MoonEmoji) {
        Write-Host "Moon Phase: $MoonEmoji $MoonPhase" -ForegroundColor Gray
    }
    if ($ShowNextFullMoon -and $NextFullMoonDate) {
        Write-Host "Next Full Moon: $NextFullMoonDate" -ForegroundColor Gray
    }
    if ($ShowNextNewMoon -and $NextNewMoonDate) {
        Write-Host "Next New Moon: $NextNewMoonDate" -ForegroundColor Gray
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
    $wrappedForecast | ForEach-Object { Write-Host $_ -ForegroundColor $detailedForecastColor }
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
        [object]$SunriseTime = $null,
        [object]$SunsetTime = $null,
        [string]$City = "",
        [bool]$ShowCityInTitle = $false,
        [string]$TimeZone = ""
    )
    
    Write-Host ""
    if ($ShowCityInTitle -and $City) {
        # Extract as many words as fit within 20 characters to keep title short
        $cityName = Get-TruncatedCityName -CityName $City -MaxLength 20
        $titleText = "*** $cityName Hourly ***"
    } else {
        $titleText = "*** Hourly ***"
    }
    Write-Host $titleText -ForegroundColor $TitleColor
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
        
        # Parse the time and convert to destination timezone
        $periodTimeOffset = [DateTimeOffset]::Parse($period.startTime)
        
        # Convert to destination timezone if provided
        if ($TimeZone) {
            $destTimeZone = Get-ResolvedTimeZoneInfo -TimeZoneId $TimeZone
            if ($destTimeZone) {
                # Convert DateTimeOffset to destination timezone
                $periodTimeLocal = [System.TimeZoneInfo]::ConvertTime($periodTimeOffset, $destTimeZone)
                $hourDisplay = $periodTimeLocal.ToString("HH:mm")
            } else {
                # Fallback if timezone conversion fails - use offset time directly
                $hourDisplay = $periodTimeOffset.ToString("HH:mm")
            }
        } else {
            # Fallback if no timezone provided - use offset time directly  
            $hourDisplay = $periodTimeOffset.ToString("HH:mm")
        }
        
        # Determine the time to use for hour midpoint calculation
        # Use timezone-converted time if available, otherwise use the period offset time
        $timeForMidpoint = if ($TimeZone -and $destTimeZone -and $periodTimeLocal) {
            $periodTimeLocal
        } else {
            $periodTimeOffset.DateTime
        }
        
        # Calculate hour midpoint (HH:30) to determine if majority of hour is during daytime
        $hourMidpoint = $timeForMidpoint.Date.AddHours($timeForMidpoint.Hour).AddMinutes(30)
        $hourMidpointTimeOnly = $hourMidpoint.TimeOfDay
        
        # Determine if hour midpoint is during daytime
        $isHourMidpointDaytime = $false
        if ($SunriseTime -and $SunsetTime) {
            # Extract time-of-day portions for comparison
            $sunriseTimeOnly = $SunriseTime.TimeOfDay
            $sunsetTimeOnly = $SunsetTime.TimeOfDay
            
            # Handle cases where sunset is the next day (after midnight)
            if ($sunsetTimeOnly -lt $sunriseTimeOnly) {
                # Sunset is the next day, so daytime is from sunrise to midnight OR midnight to sunset
                $isHourMidpointDaytime = ($hourMidpointTimeOnly -ge $sunriseTimeOnly) -or ($hourMidpointTimeOnly -lt $sunsetTimeOnly)
            } else {
                # Normal case: sunset is same day as sunrise
                $isHourMidpointDaytime = $hourMidpointTimeOnly -ge $sunriseTimeOnly -and $hourMidpointTimeOnly -lt $sunsetTimeOnly
            }
        } else {
            # For polar regions or when sunrise/sunset unavailable, use period daytime property as fallback
            if ($period.PSObject.Properties['isDaytime']) {
                $isHourMidpointDaytime = $period.isDaytime
            } else {
                # Fallback to simple time-based heuristic (6 AM to 6 PM)
                $isHourMidpointDaytime = $hourMidpoint.Hour -ge 6 -and $hourMidpoint.Hour -lt 18
            }
        }
        
        # Set hour label color: Yellow if majority of hour is during daytime, otherwise White
        $hourLabelColor = if ($isHourMidpointDaytime) { "Yellow" } else { "White" }
        
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
        $tempColor = if ([int]$temp -lt $script:COLD_TEMP_THRESHOLD) { "Blue" } elseif ([int]$temp -gt $script:HOT_TEMP_THRESHOLD) { $AlertColor } else { $DefaultColor }
        
        # Color code precipitation probability
        $precipColor = if ($precipProb -gt $script:HIGH_PRECIP_THRESHOLD) { $AlertColor } elseif ($precipProb -gt $script:MEDIUM_PRECIP_THRESHOLD) { "Yellow" } else { $DefaultColor }
        
        # Calculate windchill or heat index for hourly display
        $tempNum = [double]$temp
        $windchillHeatIndex = ""
        $windchillHeatIndexColor = ""
        
        if ($tempNum -le 50) {
            $windSpeedNum = Get-WindSpeed $wind
            $windChill = Get-WindChill $tempNum $windSpeedNum
            if ($null -ne $windChill -and ($tempNum - $windChill) -gt 1) {
                $windchillHeatIndex = " [$windChill°F]"
                $windchillHeatIndexColor = "Blue"
            }
        }
        elseif ($tempNum -ge 80) {
            $humidityNum = [double]$period.relativeHumidity.value
            $heatIndex = Get-HeatIndex $tempNum $humidityNum
            if ($null -ne $heatIndex -and ($heatIndex - $tempNum) -gt 1) {
                $windchillHeatIndex = " [$heatIndex°F]"
                $windchillHeatIndexColor = "Red"
            }
        }
        
        # Build and write the line piece-by-piece for easier colorization
        $timePart = "$hourDisplay "
        $iconPart = "$periodIcon"
        $tempPart = " $temp°F"
        # Normalize spacing around dashes in wind speed: ensure exactly one space on each side
        $windNormalized = $wind -replace '\s*-\s*', ' - '
        # Pad wind direction for alignment, but we'll trim trailing spaces before the dash
        $windPart = "$windNormalized $($windDir.PadRight(3))"
        $precipPart = if ($precipProb -gt 0) { " ($precipProb%)" } else { "" }
        $forecastPart = " - $shortForecast"
        
        # Apply wind mode color scheme to hourly wind display
        $hourlyWindSpeed = Get-WindSpeed $wind
        $windDisplayColor = if ($hourlyWindSpeed -le 5) { "White" }
                           elseif ($hourlyWindSpeed -le 9) { "Yellow" }
                           elseif ($hourlyWindSpeed -le 14) { "Red" }
                           else { "Magenta" }

        # Calculate padding for alignment
        $targetTempColumn = 8
        $lineBeforeTemp = $timePart + $iconPart
        $currentDisplayWidth = Get-StringDisplayWidth $lineBeforeTemp
        $spacesNeeded = $targetTempColumn - $currentDisplayWidth
        $padding = " " * [Math]::Max(0, $spacesNeeded)

        Write-Host $timePart -ForegroundColor $hourLabelColor -NoNewline
        Write-Host $iconPart -ForegroundColor $DefaultColor -NoNewline
        Write-Host $padding -ForegroundColor $DefaultColor -NoNewline
        Write-Host $tempPart -ForegroundColor $tempColor -NoNewline
        if ($windchillHeatIndex) {
            Write-Host $windchillHeatIndex -ForegroundColor $windchillHeatIndexColor -NoNewline
        }
        Write-Host " " -ForegroundColor $DefaultColor -NoNewline
        # Trim trailing spaces from windPart to ensure exactly one space before the dash
        $windPartTrimmed = $windPart.TrimEnd()
        Write-Host $windPartTrimmed -ForegroundColor $windDisplayColor -NoNewline
        if ($precipPart) { 
            Write-Host $precipPart -ForegroundColor $precipColor -NoNewline 
        }
        # Ensure exactly one space before the dash in forecastPart
        Write-Host $forecastPart -ForegroundColor $DefaultColor
        
        $hourCount++
    }
}

# Function to format date with ordinal suffix (e.g., "Jan 11th:")
function Format-DateWithOrdinal {
    param(
        [DateTime]$Date
    )
    
    $month = $Date.ToString("MMM")
    $day = $Date.Day
    
    # Determine ordinal suffix
    $suffix = if ($day -ge 11 -and $day -le 13) {
        "th"  # 11th, 12th, 13th
    } elseif ($day % 10 -eq 1) {
        "st"  # 1st, 21st, 31st
    } elseif ($day % 10 -eq 2) {
        "nd"  # 2nd, 22nd
    } elseif ($day % 10 -eq 3) {
        "rd"  # 3rd, 23rd
    } else {
        "th"  # 4th, 5th, 6th, etc.
    }
    
    return "$month $day$suffix`:"
}

# Function to display 7-day forecast
function Show-SevenDayForecast {
    param(
        [object]$ForecastData,
        [string]$TitleColor,
        [string]$DefaultColor,
        [string]$AlertColor,
        [int]$MaxDays = $script:MAX_DAILY_FORECAST_DAYS,
        [object]$SunriseTime = $null,
        [object]$SunsetTime = $null,
        [bool]$IsEnhancedMode = $false,
        [string]$City = "",
        [bool]$ShowCityInTitle = $false,
        [double]$Latitude = 0,
        [double]$Longitude = 0,
        [string]$TimeZone = ""
    )
    
    Write-Host ""
    if ($ShowCityInTitle -and $City) {
        # Extract as many words as fit within 20 characters to keep title short
        $cityName = Get-TruncatedCityName -CityName $City -MaxLength 20
        $titleText = if ($IsEnhancedMode) { "*** $cityName 7-Day Forecast ***" } else { "*** $cityName 7-Day Summary ***" }
    } else {
        $titleText = if ($IsEnhancedMode) { "*** 7-Day Forecast ***" } else { "*** 7-Day Summary ***" }
    }
    Write-Host $titleText -ForegroundColor $TitleColor
    $forecastPeriods = $forecastData.properties.periods
    $dayCount = 0
    $processedDays = @{}

    foreach ($period in $forecastPeriods) {
        if ($dayCount -ge $MaxDays) { break }
        
        # Validate period has required properties
        if (-not $period.startTime) {
            Write-Verbose "Skipping period with missing startTime"
            continue
        }
        
        try {
            $periodTime = [DateTime]::Parse($period.startTime)
        } catch {
            Write-Verbose "Failed to parse period startTime: $($period.startTime). Error: $($_.Exception.Message)"
            continue
        }
        
        $dayName = if ($IsEnhancedMode) { $periodTime.ToString("dddd") } else { $periodTime.ToString("ddd") }
        
        # Skip if we've already processed this day
        if ($processedDays.ContainsKey($dayName)) { continue }
        
        $temp = $period.temperature
        $shortForecast = $period.shortForecast
        $precipProb = $period.probabilityOfPrecipitation.value
        
        # For daily forecast, always use daytime icons regardless of actual time
        # This ensures consistent daytime icons in the 7-day summary
        $periodIcon = Get-WeatherIcon $period.icon $true $precipProb
        
        # Calculate moon phase for this specific day (current time + day offset)
        $moonPhaseInfo = Get-MoonPhase -Date (Get-Date).AddDays($dayCount)
        $moonEmoji = $moonPhaseInfo.Emoji
        
        # Find the corresponding night period for high/low and detailed forecast
        $nightTemp = $null
        $nightDetailedForecast = $null
        
        # Get the current period's start time to find the next period for the same day
        $currentPeriodTime = $periodTime  # Use already parsed periodTime
        $currentDay = $currentPeriodTime.ToString("yyyy-MM-dd")
        
        # Debug: Check the period structure
        
        # Look for the next period on the same day (which should be the night period)
        foreach ($nightPeriod in $forecastPeriods) {
            if (-not $nightPeriod.startTime) { continue }
            try {
                $nightTime = [DateTime]::Parse($nightPeriod.startTime)
            } catch {
                Write-Verbose "Failed to parse night period startTime: $($nightPeriod.startTime)"
                continue
            }
            $nightDay = $nightTime.ToString("yyyy-MM-dd")
            
            # Check if this is the next period on the same day
            if ($nightDay -eq $currentDay -and $nightTime -gt $currentPeriodTime) {
                $nightTemp = $nightPeriod.temperature
                $nightDetailedForecast = $nightPeriod.detailedForecast
                break
            }
        }
        
        # Color code temperature
        $tempColor = if ([int]$temp -lt $script:COLD_TEMP_THRESHOLD) { "Blue" } elseif ([int]$temp -gt $script:HOT_TEMP_THRESHOLD) { $AlertColor } else { $DefaultColor }
        
        if ($IsEnhancedMode) {
            # Enhanced Daily mode display
            # Extract wind information
            $windSpeed = Get-WindSpeed $period.windSpeed
            $windColor = if ($windSpeed -ge $script:WIND_ALERT_THRESHOLD) { $AlertColor } else { $DefaultColor }
            $windDisplay = $period.windSpeed -replace '\s+mph', 'mph'
            
            # Calculate windchill or heat index
            $tempNum = [double]$temp
            $windChillHeatIndex = ""
            $windChillHeatIndexColor = ""
            if ($tempNum -le 50) {
                $windChill = Get-WindChill $tempNum $windSpeed
                if ($null -ne $windChill -and ($tempNum - $windChill) -gt 1) {
                    $windChillHeatIndex = " [$windChill°F]"
                    $windChillHeatIndexColor = "Blue"
                }
            } elseif ($tempNum -ge 80) {
                $humidityNum = [double]$period.relativeHumidity.value
                $heatIndex = Get-HeatIndex $tempNum $humidityNum
                if ($null -ne $heatIndex -and ($heatIndex - $tempNum) -gt 1) {
                    $windChillHeatIndex = " [$heatIndex°F]"
                    $windChillHeatIndexColor = "Red"
                }
            }
            
            # Color code precipitation probability
            $precipColor = if ($precipProb -gt $script:HIGH_PRECIP_THRESHOLD) { $AlertColor } elseif ($precipProb -gt $script:MEDIUM_PRECIP_THRESHOLD) { "Yellow" } else { $DefaultColor }
            
            # Calculate sunrise/sunset for this specific day (use date only, not time)
            $daySunTimes = $null
            $sunriseStr = ""
            $sunsetStr = ""
            $dayLengthStr = ""
            if ($Latitude -ne 0 -and $Longitude -ne 0 -and $TimeZone -and $null -ne $currentPeriodTime) {
                # Use just the date portion (midnight) for accurate sunrise/sunset calculation
                $dayDate = $currentPeriodTime.Date
                $daySunTimes = Get-SunriseSunset -Latitude $Latitude -Longitude $Longitude -Date $dayDate -TimeZoneId $TimeZone
                if ($daySunTimes.Sunrise) {
                    # Format sunrise: date/time (MM/dd HH:mm) if polar night/day, otherwise time (24-hour format)
                    # During polar night: shows next sunrise; during polar day: shows last sunrise
                    if ($daySunTimes.IsPolarNight -or $daySunTimes.IsPolarDay) {
                        $sunriseStr = $daySunTimes.Sunrise.ToString('MM/dd HH:mm')
                    } else {
                        $sunriseStr = $daySunTimes.Sunrise.ToString('HH:mm')
                    }
                    # Show sunset if available (during polar night: last sunset; during polar day: next sunset)
                    if ($daySunTimes.Sunset) {
                        if ($daySunTimes.IsPolarNight -or $daySunTimes.IsPolarDay) {
                            $sunsetStr = $daySunTimes.Sunset.ToString('MM/dd HH:mm')
                        } else {
                            $sunsetStr = $daySunTimes.Sunset.ToString('HH:mm')
                        }
                        # Only show day length if not polar night/day (normal day)
                        if (-not $daySunTimes.IsPolarNight -and -not $daySunTimes.IsPolarDay) {
                            $dayLengthStr = Format-DayLength -Sunrise $daySunTimes.Sunrise -Sunset $daySunTimes.Sunset
                        }
                    }
                }
            }
            
            # Display enhanced format with proper padding to align to column 10
            $dayNameWithColon = "$dayName`:"
            $targetColumn = 10
            $currentWidth = Get-StringDisplayWidth $dayNameWithColon
            $paddingNeeded = $targetColumn - $currentWidth
            $padding = " " * [Math]::Max(0, $paddingNeeded)
            
            Write-Host $dayNameWithColon -ForegroundColor Yellow -NoNewline
            Write-Host $padding -ForegroundColor Yellow -NoNewline
            
            # Display sunrise/sunset/day length if available (on same line, no blank line after)
            if ($sunriseStr) {
                Write-Host "Sunrise: " -ForegroundColor $DefaultColor -NoNewline
                Write-Host "$sunriseStr" -ForegroundColor Gray -NoNewline
                if ($sunsetStr) {
                    Write-Host " Sunset: " -ForegroundColor $DefaultColor -NoNewline
                    Write-Host "$sunsetStr" -ForegroundColor Gray -NoNewline
                    if ($dayLengthStr) {
                        Write-Host " Day Length: " -ForegroundColor $DefaultColor -NoNewline
                        Write-Host "$dayLengthStr" -ForegroundColor Gray
                    } else {
                        Write-Host ""  # Newline if no day length (polar night/day)
                    }
                } else {
                    Write-Host ""  # Newline if no sunset
                }
            }
            
            # Format date with ordinal suffix (e.g., "Jan 11th:")
            $dateStr = Format-DateWithOrdinal -Date $periodTime
            $dateWidth = Get-StringDisplayWidth $dateStr
            $targetColumn = 10
            $paddingNeeded = $targetColumn - $dateWidth
            $datePadding = " " * [Math]::Max(0, $paddingNeeded)
            
            # Display date in white, then padding, then temperature
            Write-Host $dateStr -ForegroundColor White -NoNewline
            Write-Host $datePadding -ForegroundColor White -NoNewline
            Write-Host "H:$temp°F" -ForegroundColor $tempColor -NoNewline
            if ($windChillHeatIndex) {
                Write-Host $windChillHeatIndex -ForegroundColor $windChillHeatIndexColor -NoNewline
            }
            if ($nightTemp) {
                Write-Host " L:$nightTemp°F" -ForegroundColor $tempColor -NoNewline
            }
            Write-Host " $windDisplay $($period.windDirection)" -ForegroundColor $windColor -NoNewline
            if ($precipProb -gt 0) {
                Write-Host " ($precipProb%☔️)" -ForegroundColor $precipColor -NoNewline
            }
            Write-Host ""
            
            # Get terminal width for text wrapping
            $terminalWidth = $Host.UI.RawUI.WindowSize.Width
            
            # Determine if current period is day or night based on time
            $currentHour = $currentPeriodTime.Hour
            $isCurrentPeriodNight = ($currentHour -ge 18 -or $currentHour -lt 6)  # Evening (6 PM) to morning (6 AM)
            
            # If we have both day and night periods, show both
            if ($nightDetailedForecast) {
                # Day detailed forecast with wrapping
                $dayLabel = "$periodIcon Day:   "
                $dayForecastText = if ($period.detailedForecast) { $period.detailedForecast } else { "No detailed forecast available" }
                
                $wrappedDayForecast = Format-TextWrap -Text $dayForecastText -Width ($terminalWidth - (Get-StringDisplayWidth $dayLabel))
                
                Write-Host $dayLabel -ForegroundColor White -NoNewline
                Write-Host $wrappedDayForecast[0] -ForegroundColor $detailedForecastColor
                # Additional wrapped lines with proper indentation
                for ($i = 1; $i -lt $wrappedDayForecast.Count; $i++) {
                    Write-Host ("          " + $wrappedDayForecast[$i]) -ForegroundColor $detailedForecastColor
                }
                
                # Night detailed forecast with wrapping
                $nightLabel = "$moonEmoji Night: "
                
                $wrappedNightForecast = Format-TextWrap -Text $nightDetailedForecast -Width ($terminalWidth - (Get-StringDisplayWidth $nightLabel))
                
                Write-Host $nightLabel -ForegroundColor White -NoNewline
                Write-Host $wrappedNightForecast[0] -ForegroundColor $detailedForecastColor
                # Additional wrapped lines with proper indentation
                for ($i = 1; $i -lt $wrappedNightForecast.Count; $i++) {
                    Write-Host ("          " + $wrappedNightForecast[$i]) -ForegroundColor $detailedForecastColor
                }
            } else {
                # Only one period available - determine if it's day or night
                $singlePeriodLabel = if ($isCurrentPeriodNight) { "$moonEmoji Night: " } else { "$periodIcon Day:   " }
                $singlePeriodText = if ($period.detailedForecast) { $period.detailedForecast } else { "No detailed forecast available" }
                
                $wrappedSingleForecast = Format-TextWrap -Text $singlePeriodText -Width ($terminalWidth - (Get-StringDisplayWidth $singlePeriodLabel))
                
                Write-Host $singlePeriodLabel -ForegroundColor White -NoNewline
                Write-Host $wrappedSingleForecast[0] -ForegroundColor $detailedForecastColor
                # Additional wrapped lines with proper indentation
                $indentSpaces = if ($isCurrentPeriodNight) { "          " } else { "          " }
                for ($i = 1; $i -lt $wrappedSingleForecast.Count; $i++) {
                    Write-Host ($indentSpaces + $wrappedSingleForecast[$i]) -ForegroundColor $detailedForecastColor
                }
            }
        } else {
            # Standard Full mode display (unchanged)
            $formattedLine = Format-DailyLine -DayName $dayName -Icon $periodIcon -Temp $temp -NightTemp $nightTemp -Forecast $shortForecast -PrecipProb $precipProb
        
        # Split the line into parts for color coding
        $tempStart = $formattedLine.IndexOf(" H:$temp°F")
        if ($tempStart -lt 0) {
            $tempStart = $formattedLine.IndexOf(" $temp°F")
        }
        
        if ($tempStart -ge 0) {
            # Write day name in yellow, then the rest in default color
            $dayNameEnd = $formattedLine.IndexOf(": ")
            if ($dayNameEnd -ge 0) {
                Write-Host $formattedLine.Substring(0, $dayNameEnd + 2) -ForegroundColor Yellow -NoNewline
                Write-Host $formattedLine.Substring($dayNameEnd + 2, $tempStart - $dayNameEnd - 2) -ForegroundColor $DefaultColor -NoNewline
            } else {
                Write-Host $formattedLine.Substring(0, $tempStart) -ForegroundColor $DefaultColor -NoNewline
            }
            
            # Write temperature with color
            if ($nightTemp) {
                Write-Host " H:$temp°F L:$nightTemp°F" -ForegroundColor $tempColor -NoNewline
            } else {
                Write-Host " $temp°F" -ForegroundColor $tempColor -NoNewline
            }
            
            # Write everything after temperature with proper precipitation color coding
            $tempEnd = if ($nightTemp) { " H:$temp°F L:$nightTemp°F".Length } else { " $temp°F".Length }
            $afterTemp = $formattedLine.Substring($tempStart + $tempEnd)
            
            # Check if there's precipitation data and apply color coding
            if ($precipProb -gt 0) {
                # Find the precipitation part in the line
                $precipStart = $afterTemp.IndexOf("($precipProb%☔️)")
                if ($precipStart -ge 0) {
                    # Write everything before precipitation
                    Write-Host $afterTemp.Substring(0, $precipStart) -ForegroundColor $DefaultColor -NoNewline
                    
                    # Write precipitation with proper color
                    $precipColor = if ($precipProb -gt $script:HIGH_PRECIP_THRESHOLD) { $AlertColor } elseif ($precipProb -gt $script:MEDIUM_PRECIP_THRESHOLD) { "Yellow" } else { $DefaultColor }
                    Write-Host "($precipProb%☔️)" -ForegroundColor $precipColor -NoNewline
                    
                    # Write everything after precipitation
                    $precipEnd = "($precipProb%☔️)".Length
                    if ($precipStart + $precipEnd -lt $afterTemp.Length) {
                        Write-Host $afterTemp.Substring($precipStart + $precipEnd) -ForegroundColor $DefaultColor
                    } else {
                        Write-Host ""
                    }
                } else {
                    Write-Host $afterTemp -ForegroundColor $DefaultColor
                }
            } else {
                Write-Host $afterTemp -ForegroundColor $DefaultColor
            }
        } else {
            # Fallback if temperature not found
            Write-Host $formattedLine -ForegroundColor $DefaultColor
        }
        }
        
        $processedDays[$dayName] = $true
        $dayCount++
    }
}

# Function to convert wind direction degrees to cardinal direction
function Get-CardinalDirection {
    param([double]$Degrees)
    
    $directions = @("N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW")
    $index = [Math]::Round($Degrees / 22.5) % 16
    return $directions[$index]
}

# Function to display historical weather observations
function Show-Observations {
    param(
        [array]$ObservationsData,
        [string]$TitleColor,
        [string]$DefaultColor,
        [string]$AlertColor,
        [string]$City = "",
        [bool]$ShowCityInTitle = $false,
        [string]$TimeZone = "",
        [double]$Latitude = 0,
        [double]$Longitude = 0
    )
    
    if ($ShowCityInTitle -and $City) {
        $cityName = Get-TruncatedCityName -CityName $City -MaxLength 20
        $titleText = "*** $cityName Observations ***"
    } else {
        $titleText = "*** Observations ***"
    }
    Write-Host $titleText -ForegroundColor $TitleColor
    
    if (-not $ObservationsData -or $ObservationsData.Count -eq 0) {
        Write-Host "No historical observations available." -ForegroundColor $DefaultColor
        return
    }
    
    $detailedForecastColor = "Gray"
    
    # Reverse the order so most recent observations appear first
    $reversedObservations = if ($ObservationsData -is [Array]) {
        $ObservationsData | Sort-Object { [DateTime]::Parse($_.Date) } -Descending
    } else {
        $ObservationsData
    }
    
    foreach ($dayData in $reversedObservations) {
        # Skip days with no actual data (all N/A)
        if ($null -eq $dayData.HighTemp -and $null -eq $dayData.LowTemp -and $null -eq $dayData.AvgWindSpeed) {
            continue
        }
        
        $date = [DateTime]::Parse($dayData.Date)
        $dayName = $date.ToString("dddd")
        $dateStr = $date.ToString("MM/dd")
        
        # Calculate moon phase for this day
        $moonPhaseInfo = Get-MoonPhase -Date $date
        $moonEmoji = $moonPhaseInfo.Emoji
        
        # Get high and low temperatures
        $highTemp = $dayData.HighTemp
        $lowTemp = $dayData.LowTemp
        
        # Color code temperature (use high temp for color)
        $tempColor = if ($null -ne $highTemp) {
            if ([int]$highTemp -lt $script:COLD_TEMP_THRESHOLD) { "Blue" } 
            elseif ([int]$highTemp -gt $script:HOT_TEMP_THRESHOLD) { $AlertColor } 
            else { $DefaultColor }
        } else {
            $DefaultColor
        }
        
        # Wind information
        $avgWindSpeed = $dayData.AvgWindSpeed
        $maxWindSpeed = $dayData.MaxWindSpeed
        $maxWindGust = $dayData.MaxWindGust
        $windDirection = $dayData.WindDirection
        
        # Calculate windchill or heat index (use avg wind speed for calculations)
        $windChillHeatIndex = ""
        $windChillHeatIndexColor = ""
        if ($null -ne $highTemp -and $null -ne $avgWindSpeed) {
            $tempNum = [double]$highTemp
            $windSpeedNum = [double]$avgWindSpeed
            if ($tempNum -le 50) {
                $windChill = Get-WindChill $tempNum $windSpeedNum
                if ($null -ne $windChill -and ($tempNum - $windChill) -gt 1) {
                    $windChillHeatIndex = " [$windChill°F]"
                    $windChillHeatIndexColor = "Blue"
                }
            } elseif ($tempNum -ge 80) {
                $humidityNum = if ($null -ne $dayData.AvgHumidity) { [double]$dayData.AvgHumidity } else { 0 }
                $heatIndex = Get-HeatIndex $tempNum $humidityNum
                if ($null -ne $heatIndex -and ($heatIndex - $tempNum) -gt 1) {
                    $windChillHeatIndex = " [$heatIndex°F]"
                    $windChillHeatIndexColor = "Red"
                }
            }
        }
        
        # Precipitation
        $totalPrecip = $dayData.TotalPrecipitation
        $precipDisplay = if ($totalPrecip -gt 0) { " ($totalPrecip`" precip)" } else { "" }
        
        # Wind display - color code avg and gust separately
        # Calculate colors separately for avg and gust
        $avgWindColor = $DefaultColor
        $gustWindColor = $DefaultColor
        if ($null -ne $avgWindSpeed) {
            $avgWindSpeedNum = [Math]::Round($avgWindSpeed, 0)
            $avgWindColor = if ($avgWindSpeedNum -le 5) { "White" }
                           elseif ($avgWindSpeedNum -le 9) { "Yellow" }
                           elseif ($avgWindSpeedNum -le 14) { "Red" }
                           else { "Magenta" }
        }
        if ($null -ne $maxWindGust) {
            $maxWindGustNum = [Math]::Round($maxWindGust, 0)
            $gustWindColor = if ($maxWindGustNum -le 5) { "White" }
                            elseif ($maxWindGustNum -le 9) { "Yellow" }
                            elseif ($maxWindGustNum -le 14) { "Red" }
                            else { "Magenta" }
        } elseif ($null -ne $maxWindSpeed) {
            $maxWindSpeedNum = [Math]::Round($maxWindSpeed, 0)
            $gustWindColor = if ($maxWindSpeedNum -le 5) { "White" }
                            elseif ($maxWindSpeedNum -le 9) { "Yellow" }
                            elseif ($maxWindSpeedNum -le 14) { "Red" }
                            else { "Magenta" }
        }
        
        # Calculate sunrise/sunset for this specific observation date
        # Extract date components and create a date object for accurate calculation
        $daySunTimes = $null
        $sunriseStr = ""
        $sunsetStr = ""
        $dayLengthStr = ""
        if ($TimeZone -and $Latitude -ne 0 -and $Longitude -ne 0) {
            # Parse the date string (format: "YYYY-MM-DD") and extract components
            $parsedDate = [DateTime]::Parse($dayData.Date)
            # Create a new date using just the year, month, day (timezone doesn't matter for date-only calculation)
            $dayDate = [DateTime]::new($parsedDate.Year, $parsedDate.Month, $parsedDate.Day)
            $daySunTimes = Get-SunriseSunset -Latitude $Latitude -Longitude $Longitude -Date $dayDate -TimeZoneId $TimeZone
            if ($daySunTimes.Sunrise) {
                # Format sunrise: date/time (MM/dd HH:mm) if polar night/day, otherwise time (24-hour format)
                # During polar night: shows next sunrise; during polar day: shows last sunrise
                if ($daySunTimes.IsPolarNight -or $daySunTimes.IsPolarDay) {
                    $sunriseStr = $daySunTimes.Sunrise.ToString('MM/dd HH:mm')
                } else {
                    $sunriseStr = $daySunTimes.Sunrise.ToString('HH:mm')
                }
                # Show sunset if available (during polar night: last sunset; during polar day: next sunset)
                if ($daySunTimes.Sunset) {
                    if ($daySunTimes.IsPolarNight -or $daySunTimes.IsPolarDay) {
                        $sunsetStr = $daySunTimes.Sunset.ToString('MM/dd HH:mm')
                    } else {
                        $sunsetStr = $daySunTimes.Sunset.ToString('HH:mm')
                    }
                    # Only show day length if not polar night/day (normal day)
                    if (-not $daySunTimes.IsPolarNight -and -not $daySunTimes.IsPolarDay) {
                        $dayLengthStr = Format-DayLength -Sunrise $daySunTimes.Sunrise -Sunset $daySunTimes.Sunset
                    }
                }
            }
        }
        
        # Display enhanced format with proper padding
        $dayNameWithColon = "$dayName ($dateStr):"
        $targetColumn = 10
        $currentWidth = Get-StringDisplayWidth $dayNameWithColon
        $paddingNeeded = $targetColumn - $currentWidth
        $padding = " " * [Math]::Max(0, $paddingNeeded)
        
        Write-Host $dayNameWithColon -ForegroundColor White -NoNewline
        Write-Host $padding -ForegroundColor White -NoNewline
        
        # Display sunrise/sunset/day length if available (on same line, no blank line after)
        if ($sunriseStr) {
            Write-Host " Sunrise: " -ForegroundColor $DefaultColor -NoNewline
            Write-Host "$sunriseStr" -ForegroundColor Gray -NoNewline
            if ($sunsetStr) {
                Write-Host " Sunset: " -ForegroundColor $DefaultColor -NoNewline
                Write-Host "$sunsetStr" -ForegroundColor Gray -NoNewline
                if ($dayLengthStr) {
                    Write-Host " Day Length: " -ForegroundColor $DefaultColor -NoNewline
                    Write-Host "$dayLengthStr" -ForegroundColor Gray
                } else {
                    Write-Host ""  # Newline if no day length (polar night/day)
                }
            } else {
                Write-Host ""  # Newline if no sunset
            }
        }
        
        # Temperature display
        if ($null -ne $highTemp) {
            Write-Host " H:$highTemp°F" -ForegroundColor $tempColor -NoNewline
        } else {
            Write-Host "H:N/A" -ForegroundColor $DefaultColor -NoNewline
        }
        
        if ($windChillHeatIndex) {
            Write-Host $windChillHeatIndex -ForegroundColor $windChillHeatIndexColor -NoNewline
        }
        
        if ($null -ne $lowTemp) {
            Write-Host " L:$lowTemp°F" -ForegroundColor $tempColor -NoNewline
        } else {
            Write-Host " L:N/A" -ForegroundColor $DefaultColor -NoNewline
        }
        
        # Wind display - color code avg and gust separately
        $windDirStr = if ($null -ne $windDirection) { Get-CardinalDirection $windDirection } else { "" }
        if ($null -ne $avgWindSpeed) {
            $avgWindSpeedStr = [Math]::Round($avgWindSpeed, 0).ToString()
            
            # Show average with separate color
            Write-Host " avg ${avgWindSpeedStr}mph" -ForegroundColor $avgWindColor -NoNewline
            
            # Show gust with separate color if available
            if ($null -ne $maxWindGust) {
                $maxWindGustStr = [Math]::Round($maxWindGust, 0).ToString()
                Write-Host " gust ${maxWindGustStr}mph" -ForegroundColor $gustWindColor -NoNewline
            } elseif ($null -ne $maxWindSpeed) {
                # Show max with separate color if it differs significantly
                if ([Math]::Abs($maxWindSpeed - $avgWindSpeed) -gt 1) {
                    $maxWindSpeedStr = [Math]::Round($maxWindSpeed, 0).ToString()
                    Write-Host " max ${maxWindSpeedStr}mph" -ForegroundColor $gustWindColor -NoNewline
                }
            }
            
            # Wind direction
            if ($windDirStr) {
                Write-Host " $windDirStr" -ForegroundColor $DefaultColor -NoNewline
            }
        } elseif ($null -ne $maxWindSpeed) {
            # Fallback to max if avg not available
            $maxWindSpeedStr = [Math]::Round($maxWindSpeed, 0).ToString()
            Write-Host " max ${maxWindSpeedStr}mph" -ForegroundColor $gustWindColor -NoNewline
            if ($windDirStr) {
                Write-Host " $windDirStr" -ForegroundColor $DefaultColor -NoNewline
            }
        } elseif ($null -ne $maxWindGust) {
            # Fallback to gust if available
            $maxWindGustStr = [Math]::Round($maxWindGust, 0).ToString()
            Write-Host " gust ${maxWindGustStr}mph" -ForegroundColor $gustWindColor -NoNewline
            if ($windDirStr) {
                Write-Host " $windDirStr" -ForegroundColor $DefaultColor -NoNewline
            }
        }
        
        # Pressure display (inHg) - color by range: low (Cyan), normal (White), high (Yellow), extreme (Magenta)
        $pressureInHg = $dayData.Pressure
        if ($null -ne $pressureInHg) {
            $pressureColor = if ($pressureInHg -lt 29.0 -or $pressureInHg -gt 30.5) { $AlertColor }
                            elseif ($pressureInHg -lt 29.50) { "Cyan" }
                            elseif ($pressureInHg -le 30.20) { $DefaultColor }
                            else { "Yellow" }
            Write-Host " P:$pressureInHg inHg" -ForegroundColor $pressureColor -NoNewline
        } else {
            Write-Host " P:N/A" -ForegroundColor $DefaultColor -NoNewline
        }
        
        # Precipitation display
        if ($precipDisplay) {
            Write-Host $precipDisplay -ForegroundColor $DefaultColor -NoNewline
        }
        
        # Humidity display
        if ($null -ne $dayData.AvgHumidity) {
            $humidityStr = [Math]::Round($dayData.AvgHumidity, 0).ToString()
            Write-Host " ($humidityStr% RH)" -ForegroundColor $DefaultColor -NoNewline
        }
        
        Write-Host ""
        
        # Conditions display (Clouds on same line when available: "Conditions: X Clouds: Y")
        $conditionsLabel = "$moonEmoji Conditions: "
        $conditionsValue = if ($dayData.Conditions -ne "N/A") { $dayData.Conditions } else { "N/A" }
        if ($dayData.CloudSummary -and $dayData.CloudSummary.ToString().Trim()) {
            $conditionsValue = "$conditionsValue Clouds: $($dayData.CloudSummary)"
        }
        $terminalWidth = $Host.UI.RawUI.WindowSize.Width
        $wrappedConditions = Format-TextWrap -Text $conditionsValue -Width ($terminalWidth - (Get-StringDisplayWidth $conditionsLabel))
        Write-Host $conditionsLabel -ForegroundColor White -NoNewline
        $firstLine = $wrappedConditions[0]
        if ($firstLine -match '^(.+?) Clouds: (.+)$') {
            Write-Host $Matches[1] -ForegroundColor $detailedForecastColor -NoNewline
            Write-Host " Clouds: " -ForegroundColor White -NoNewline
            Write-Host $Matches[2] -ForegroundColor $detailedForecastColor
        } else {
            Write-Host $firstLine -ForegroundColor $detailedForecastColor
        }
        for ($i = 1; $i -lt $wrappedConditions.Count; $i++) {
            Write-Host ("          " + $wrappedConditions[$i]) -ForegroundColor $detailedForecastColor
        }
        
        Write-Host ""
    }
}

# Function to display weather alerts
function Show-WeatherAlerts {
    param(
        [object]$AlertsData,
        [string]$AlertColor,
        [string]$DefaultColor,
        [string]$InfoColor,
        [bool]$ShowDetails = $true,
        [string]$TimeZone = $null
    )
    # Use local copy from PSBoundParameters to avoid unbound parameter reference (can crash in some hosts)
    $showDetails = if ($PSBoundParameters.ContainsKey('ShowDetails')) { $ShowDetails } else { $true }
    
    if ($alertsData -and $alertsData.features.Count -gt 0) {
        if ($showDetails) {
            Write-Host ""
            Write-Host "*** Active Weather Alerts ***" -ForegroundColor $AlertColor
        } else {
            Write-Host ""
        }
        for ($i = 0; $i -lt $alertsData.features.Count; $i++) {
            $alert = $alertsData.features[$i]
            $alertProps = $alert.properties
            $alertEvent = $alertProps.event
            $alertHeadline = $alertProps.headline
            $alertDesc = $alertProps.description
            
            # Parse alert times - API returns ISO 8601 format (typically UTC with 'Z' suffix)
            # The alert description text already contains times in local time, so we must
            # convert the effective/expires times to the location's timezone to match
            # Note: 'expires' is when the alert message expires, 'ends' is when the event actually ends
            # We use 'ends' if available to match the description text, otherwise fall back to 'expires'
            $alertStartOffset = [DateTimeOffset]::Parse($alertProps.effective)
            if ($alertProps.ends) {
                $alertEndOffset = [DateTimeOffset]::Parse($alertProps.ends)
            } else {
                $alertEndOffset = [DateTimeOffset]::Parse($alertProps.expires)
            }
            
            # Always convert to location's timezone to match the alert description text
            # The description text uses local time, so effective/expires must also be local
            if ($TimeZone) {
                $locationTimeZone = Get-ResolvedTimeZoneInfo -TimeZoneId $TimeZone
                if ($locationTimeZone) {
                    # Convert DateTimeOffset to location's timezone
                    $alertStart = [System.TimeZoneInfo]::ConvertTime($alertStartOffset, $locationTimeZone)
                    $alertEnd = [System.TimeZoneInfo]::ConvertTime($alertEndOffset, $locationTimeZone)
                } else {
                    # Fallback to local time if timezone resolution fails
                    $alertStart = $alertStartOffset.LocalDateTime
                    $alertEnd = $alertEndOffset.LocalDateTime
                }
            } else {
                # Fallback to local time if no timezone provided
                $alertStart = $alertStartOffset.LocalDateTime
                $alertEnd = $alertEndOffset.LocalDateTime
            }
            
            if ($showDetails) {
                # Full mode: label on own line, headline, details, Effective, Expires on separate lines
                Write-Host "*** $alertEvent ***" -ForegroundColor $AlertColor
                Write-Host "$alertHeadline" -ForegroundColor $DefaultColor
                $wrappedAlert = Format-TextWrap -Text $alertDesc -Width $Host.UI.RawUI.WindowSize.Width
                $wrappedAlert | ForEach-Object { Write-Host $_ -ForegroundColor $DefaultColor }
                Write-Host "Effective: $($alertStart.ToString('MM/dd/yyyy HH:mm'))" -ForegroundColor $InfoColor
                Write-Host "Expires: $($alertEnd.ToString('MM/dd/yyyy HH:mm'))" -ForegroundColor $InfoColor
            } else {
                # Terse mode: label + Expires on same line only (red then blue)
                Write-Host -NoNewline "*** $alertEvent *** " -ForegroundColor $AlertColor
                Write-Host "Expires: $($alertEnd.ToString('MM/dd/yyyy HH:mm'))" -ForegroundColor $InfoColor
            }
            
            # Only add blank line if this is not the last alert
            if ($i -lt $alertsData.features.Count - 1) {
                Write-Host ""
            }
        }
    }
}

# Function to calculate distance between two coordinates using Haversine formula
function Get-DistanceMiles {
    param(
        [double]$Lat1,
        [double]$Lon1,
        [double]$Lat2,
        [double]$Lon2
    )
    
    # Earth radius in miles
    $R = 3959
    
    # Convert degrees to radians
    $lat1Rad = [Math]::PI * $Lat1 / 180.0
    $lon1Rad = [Math]::PI * $Lon1 / 180.0
    $lat2Rad = [Math]::PI * $Lat2 / 180.0
    $lon2Rad = [Math]::PI * $Lon2 / 180.0
    
    # Calculate differences
    $dLat = $lat2Rad - $lat1Rad
    $dLon = $lon2Rad - $lon1Rad
    
    # Haversine formula
    $a = [Math]::Sin($dLat / 2) * [Math]::Sin($dLat / 2) + 
         [Math]::Cos($lat1Rad) * [Math]::Cos($lat2Rad) * 
         [Math]::Sin($dLon / 2) * [Math]::Sin($dLon / 2)
    $c = 2 * [Math]::Atan2([Math]::Sqrt($a), [Math]::Sqrt(1 - $a))
    $distance = $R * $c
    
    return $distance
}

# Function to calculate bearing (direction) from point 1 to point 2 in degrees (0-360)
function Get-Bearing {
    param(
        [double]$Lat1,
        [double]$Lon1,
        [double]$Lat2,
        [double]$Lon2
    )
    
    # Convert degrees to radians
    $lat1Rad = [Math]::PI * $Lat1 / 180.0
    $lat2Rad = [Math]::PI * $Lat2 / 180.0
    $dLon = [Math]::PI * ($Lon2 - $Lon1) / 180.0
    
    $y = [Math]::Sin($dLon) * [Math]::Cos($lat2Rad)
    $x = [Math]::Cos($lat1Rad) * [Math]::Sin($lat2Rad) - 
         [Math]::Sin($lat1Rad) * [Math]::Cos($lat2Rad) * [Math]::Cos($dLon)
    
    $bearing = [Math]::Atan2($y, $x)
    $bearing = $bearing * 180.0 / [Math]::PI
    $bearing = ($bearing + 360) % 360  # Normalize to 0-360
    
    return $bearing
}

# Function to search NOAA tide stations by coordinates
function Get-NoaaTideStation {
    param(
        [double]$Lat,
        [double]$Lon,
        [array]$PreFetchedStations = $null
    )
    
    try {
        Write-Verbose "Searching NOAA tide stations for coordinates: $Lat,$Lon"
        
        # Use pre-fetched stations if provided, otherwise fetch from API
        $apiResponse = $null
        if ($PreFetchedStations) {
            Write-Verbose "Using pre-fetched stations data ($($PreFetchedStations.Count) stations)"
            $apiResponse = @{ stations = $PreFetchedStations }
        } else {
            # Use NOAA CO-OPS Metadata API to get all stations and filter by distance
            Write-Verbose "Fetching stations from NOAA CO-OPS Metadata API..."
            try {
                $apiUrl = "https://api.tidesandcurrents.noaa.gov/mdapi/prod/webapi/stations.json"
                Write-Verbose "NOAA Tide API call: GET $apiUrl"
                $apiResponse = Invoke-RestMethod -Uri $apiUrl -Method Get -ErrorAction Stop -TimeoutSec 30
                $stationsCount = if ($apiResponse.stations) { $apiResponse.stations.Count } else { 0 }
                Write-Verbose "NOAA Tide Stations API response received successfully: $stationsCount stations"
            } catch {
                Write-Verbose "Error fetching stations from API: $($_.Exception.Message)"
                return $null
            }
        }
        
        if ($apiResponse -and $apiResponse.stations) {
            Write-Verbose "Found $($apiResponse.stations.Count) stations from API"
            $closestStation = $null
            $minDistance = 1000000
            $maxDistanceMiles = 100
            $allNearbyStations = @()  # Track all stations within 100 miles for verbose logging
            
            foreach ($station in $apiResponse.stations) {
                # API uses 'lng' for longitude, not 'lon'
                if ($station.lat -and $station.lng) {
                    $stationLat = [double]$station.lat
                    $stationLon = [double]$station.lng
                    
                    $distance = Get-DistanceMiles -Lat1 $Lat -Lon1 $Lon -Lat2 $stationLat -Lon2 $stationLon
                    
                    if ($distance -le $maxDistanceMiles) {
                        # Track all stations within 100 miles
                        $allNearbyStations += @{
                            stationId = $station.id.ToString()
                            name = $station.name
                            lat = $stationLat
                            lon = $stationLon
                            distance = $distance
                        }
                        
                        # Update closest station if this one is closer
                        if ($distance -lt $minDistance) {
                            $minDistance = $distance
                            $closestStation = @{
                                stationId = $station.id.ToString()
                                name = $station.name
                                lat = $stationLat
                                lon = $stationLon
                                distance = $distance
                            }
                        }
                    }
                }
            }
            
            # Log the closest stations (sorted by distance)
            if ($allNearbyStations.Count -gt 0) {
                # Sort by distance (ascending - shortest first) - explicitly numeric sort
                $sortedStations = $allNearbyStations | Sort-Object -Property @{Expression={[double]$_.distance}; Ascending=$true}
                $topStations = $sortedStations | Select-Object -First 5
                $stationCount = $topStations.Count
                
                $headerText = if ($stationCount -eq 1) {
                    "Top 1 closest NOAA station within 100 miles:"
                } else {
                    "Top $stationCount closest NOAA stations within 100 miles:"
                }
                Write-Verbose $headerText
                
                foreach ($station in $topStations) {
                    $isSelected = ($closestStation -and $station.stationId -eq $closestStation.stationId)
                    $marker = if ($isSelected) { " [SELECTED]" } else { "" }
                    Write-Verbose "  $($station.name) ($($station.stationId)) at $([Math]::Round($station.distance, 2)) miles$marker"
                }
            }
            
            if ($closestStation) {
                Write-Verbose "Found closest NOAA station via API: $($closestStation.name) ($($closestStation.stationId)) at $([Math]::Round($closestStation.distance, 2)) miles"
                
                # Check for water level support via products endpoint (most reliable method)
                try {
                    $productsUrl = "https://api.tidesandcurrents.noaa.gov/mdapi/prod/webapi/stations/$($closestStation.stationId)/products.json"
                    Write-Verbose "NOAA Tide API call: GET $productsUrl"
                    $products = Invoke-RestMethod -Uri $productsUrl -Method Get -ErrorAction Stop -TimeoutSec 10
                    $productsCount = if ($products.products) { $products.products.Count } else { 0 }
                    Write-Verbose "NOAA Tide Products API response received successfully: $productsCount products"
                    if ($products.products) {
                        # Check if any product name contains "Water Level" or "Water Levels"
                        $waterLevelProducts = $products.products | Where-Object { 
                            $_.name -match "Water Level"
                        }
                        # Force to array to get accurate count (Where-Object may return single object)
                        $closestStation.supportsWaterLevels = (@($waterLevelProducts).Count -gt 0)
                        Write-Verbose "Water levels support from products endpoint: $($closestStation.supportsWaterLevels)"
                    } else {
                        $closestStation.supportsWaterLevels = $false
                        Write-Verbose "No products found, assuming no water levels support"
                    }
                } catch {
                    Write-Verbose "Could not fetch products endpoint, assuming no water levels support: $($_.Exception.Message)"
                    $closestStation.supportsWaterLevels = $false
                }
                
                return $closestStation
            } else {
                Write-Verbose "No stations found within 100 miles via API"
                return $null
            }
        } else {
            Write-Verbose "API response does not contain stations data"
            return $null
        }
    }
    catch {
        Write-Verbose "Error searching NOAA tide stations: $($_.Exception.Message)"
        return $null
    }
}

# Old web scraping code removed - API is the only method now

# Function to get NOAA tide station by ID (override mode)
function Get-NoaaTideStationById {
    param(
        [string]$StationId,
        [double]$Lat,
        [double]$Lon
    )
    
    try {
        Write-Verbose "Fetching NOAA station by ID: $StationId"
        
        # Fetch station details from NOAA CO-OPS Metadata API
        try {
            $apiUrl = "https://api.tidesandcurrents.noaa.gov/mdapi/prod/webapi/stations.json"
            Write-Verbose "NOAA Tide API call: GET $apiUrl"
            $apiResponse = Invoke-RestMethod -Uri $apiUrl -Method Get -ErrorAction Stop -TimeoutSec 30
            $stationsCount = if ($apiResponse.stations) { $apiResponse.stations.Count } else { 0 }
            Write-Verbose "NOAA Tide Stations API response received successfully: $stationsCount stations"
        } catch {
            Write-Verbose "Error fetching stations from API: $($_.Exception.Message)"
            return $null
        }
        
        if ($apiResponse -and $apiResponse.stations) {
            # Find the station by ID
            $station = $apiResponse.stations | Where-Object { $_.id.ToString() -eq $StationId }
            
            if ($station) {
                $stationLat = [double]$station.lat
                $stationLon = [double]$station.lng
                
                # Calculate distance from location to station
                $distance = Get-DistanceMiles -Lat1 $Lat -Lon1 $Lon -Lat2 $stationLat -Lon2 $stationLon
                
                Write-Verbose "Found NOAA station: $($station.name) ($StationId) at coordinates $stationLat,$stationLon"
                Write-Verbose "Distance from location: $([Math]::Round($distance, 2)) miles"
                
                $stationInfo = @{
                    stationId = $StationId
                    name = $station.name
                    lat = $stationLat
                    lon = $stationLon
                    distance = $distance
                }
                
                # Check for water level support via products endpoint
                try {
                    $productsUrl = "https://api.tidesandcurrents.noaa.gov/mdapi/prod/webapi/stations/$StationId/products.json"
                    Write-Verbose "NOAA Tide API call: GET $productsUrl"
                    $products = Invoke-RestMethod -Uri $productsUrl -Method Get -ErrorAction Stop -TimeoutSec 10
                    $productsCount = if ($products.products) { $products.products.Count } else { 0 }
                    Write-Verbose "NOAA Tide Products API response received successfully: $productsCount products"
                    if ($products.products) {
                        $waterLevelProducts = $products.products | Where-Object { 
                            $_.name -match "Water Level"
                        }
                        $stationInfo.supportsWaterLevels = (@($waterLevelProducts).Count -gt 0)
                        Write-Verbose "Water levels support from products endpoint: $($stationInfo.supportsWaterLevels)"
                    } else {
                        $stationInfo.supportsWaterLevels = $false
                        Write-Verbose "No products found, assuming no water levels support"
                    }
                } catch {
                    Write-Verbose "Could not fetch products endpoint, assuming no water levels support: $($_.Exception.Message)"
                    $stationInfo.supportsWaterLevels = $false
                }
                
                return $stationInfo
            } else {
                Write-Verbose "Station ID $StationId not found in NOAA stations database"
                return $null
            }
        } else {
            Write-Verbose "API response does not contain stations data"
            return $null
        }
    }
    catch {
        Write-Verbose "Error fetching NOAA station by ID: $($_.Exception.Message)"
        return $null
    }
}

# Function to display location information
function Show-LocationInfo {
    param(
        [string]$TimeZone,
        [double]$Lat,
        [double]$Lon,
        [int]$ElevationFeet,
        [string]$RadarStation,
        [string]$TitleColor,
        [string]$DefaultColor
    )
    
    Write-Host ""
    Write-Host "*** Location Information ***" -ForegroundColor $TitleColor
    
    # Calculate UTC offset for display
    $utcOffsetStr = ""
    if ($TimeZone) {
        try {
            $tzInfo = Get-ResolvedTimeZoneInfo -TimeZoneId $TimeZone
            if ($tzInfo) {
                $utcOffset = $tzInfo.GetUtcOffset([DateTime]::Now)
                $offsetHours = $utcOffset.TotalHours
                $offsetSign = if ($offsetHours -ge 0) { "+" } else { "" }
                $utcOffsetStr = " (UTC$offsetSign$([Math]::Round($offsetHours, 0)))"
            }
        } catch {
            Write-Verbose "Could not calculate UTC offset for timezone: $($_.Exception.Message)"
        }
    }
    
    Write-Host "Time Zone: $timeZone$utcOffsetStr" -ForegroundColor $DefaultColor
    Write-Host "Coordinates: $lat, $lon" -ForegroundColor $DefaultColor
    Write-Host "Elevation: ${elevationFeet}ft" -ForegroundColor $DefaultColor
    
    # Display NWS Resources with clickable links
    Write-Host "NWS Resources: " -ForegroundColor $DefaultColor -NoNewline
    # Forecast link
    $forecastUrl = "https://forecast.weather.gov/MapClick.php?lat=$lat&lon=$lon"
    Write-Host "$([char]27)]8;;$forecastUrl$([char]27)\Forecast$([char]27)]8;;$([char]27)\" -ForegroundColor Blue -NoNewline
    Write-Host " | " -ForegroundColor $DefaultColor -NoNewline
    
    # Graph link
    $graphUrl = "https://forecast.weather.gov/MapClick.php?lat=$lat&lon=$lon&unit=0&lg=english&FcstType=graphical"
    Write-Host "$([char]27)]8;;$graphUrl$([char]27)\Graph$([char]27)]8;;$([char]27)\" -ForegroundColor Blue -NoNewline
    Write-Host " | " -ForegroundColor $DefaultColor -NoNewline
    
    # Radar link
    $radarUrl = "https://radar.weather.gov/ridge/standard/${radarStation}_loop.gif"
    Write-Host "$([char]27)]8;;$radarUrl$([char]27)\Radar$([char]27)]8;;$([char]27)\" -ForegroundColor Blue
    
    # Display NOAA Station and Resources
    try {
        $noaaStation = $null
        
        # Check if -Noaa parameter is set to override station selection
        if ($Noaa -and $Noaa.Trim() -ne "") {
            Write-Verbose "Using NOAA station override: $Noaa"
            $noaaStation = Get-NoaaTideStationById -StationId $Noaa.Trim() -Lat $lat -Lon $lon
            if (-not $noaaStation) {
                Write-Verbose "Warning: Could not find NOAA station with ID '$Noaa'. Station may not exist or API call failed."
            }
        } else {
            # Normal station selection - check if we have a refreshed NOAA station from Update-WeatherData
            if ($script:noaaStation) {
                $noaaStation = $script:noaaStation
                Write-Verbose "Using refreshed NOAA station data"
            } else {
                # Check if we have pre-fetched NOAA stations data
                $preFetchedStations = $null
                if ($script:noaaStationsData -and $script:noaaStationsData.Success) {
                    $preFetchedStations = $script:noaaStationsData.Stations
                    Write-Verbose "Using pre-fetched NOAA stations data"
                } else {
                    # If job is still running, wait briefly for it
                    if ($script:noaaStationsJob -and $script:noaaStationsJob.State -ne 'Completed' -and $script:noaaStationsJob.State -ne 'Failed') {
                        Write-Verbose "Waiting for NOAA stations job to complete..."
                        $jobResult = Wait-Job -Job $script:noaaStationsJob -Timeout 5
                        if ($jobResult) {
                            $script:noaaStationsData = $script:noaaStationsJob | Receive-Job
                            Remove-Job -Job $script:noaaStationsJob
                            $script:noaaStationsJob = $null
                            if ($script:noaaStationsData -and $script:noaaStationsData.Success) {
                                $preFetchedStations = $script:noaaStationsData.Stations
                                Write-Verbose "NOAA stations data fetched after brief wait"
                            }
                        }
                    }
                }
                
                $noaaStation = Get-NoaaTideStation -Lat $lat -Lon $lon -PreFetchedStations $preFetchedStations
            }
        }
        if ($noaaStation) {
            # Display NOAA Station information first
            # Display NOAA Station information with clickable station ID
            Write-Host "NOAA Station: " -ForegroundColor $DefaultColor -NoNewline
            Write-Host "$($noaaStation.name) (" -ForegroundColor Gray -NoNewline
            $stationHomeUrl = "https://tidesandcurrents.noaa.gov/stationhome.html?id=$($noaaStation.stationId)"
            Write-Host "$([char]27)]8;;$stationHomeUrl$([char]27)\$($noaaStation.stationId)$([char]27)]8;;$([char]27)\" -ForegroundColor Blue -NoNewline
            
            # Calculate bearing and cardinal direction from location to station
            $bearing = Get-Bearing -Lat1 $lat -Lon1 $lon -Lat2 $noaaStation.lat -Lon2 $noaaStation.lon
            $cardinalDir = Get-CardinalDirection -Degrees $bearing
            $distanceStr = "$([Math]::Round($noaaStation.distance, 2))mi"
            
            Write-Host ") " -ForegroundColor Gray -NoNewline
            Write-Host "$distanceStr $cardinalDir" -ForegroundColor $DefaultColor
            
            # Display NOAA Resources
            Write-Host "NOAA Resources: " -ForegroundColor $DefaultColor -NoNewline
            # Tide Prediction link
            $tideUrl = "https://tidesandcurrents.noaa.gov/noaatidepredictions.html?id=$($noaaStation.stationId)"
            Write-Host "$([char]27)]8;;$tideUrl$([char]27)\Tide Prediction$([char]27)]8;;$([char]27)\" -ForegroundColor Blue -NoNewline
            Write-Host " | " -ForegroundColor $DefaultColor -NoNewline
            
            # Datums link
            $datumsUrl = "https://tidesandcurrents.noaa.gov/datums.html?id=$($noaaStation.stationId)"
            Write-Host "$([char]27)]8;;$datumsUrl$([char]27)\Datums$([char]27)]8;;$([char]27)\" -ForegroundColor Blue -NoNewline
            
            # Check if water levels are supported (already set in Get-NoaaTideStation via API)
            if ($noaaStation.supportsWaterLevels) {
                Write-Host " | " -ForegroundColor $DefaultColor -NoNewline
                $waterLevelsUrl = "https://tidesandcurrents.noaa.gov/waterlevels.html?id=$($noaaStation.stationId)"
                Write-Host "$([char]27)]8;;$waterLevelsUrl$([char]27)\Levels$([char]27)]8;;$([char]27)\" -ForegroundColor Blue
            } else {
                Write-Host ""
            }
            
            # Fetch and display tide predictions (can be done in parallel with other operations)
            try {
                # Log the API call that will be made (before starting the job, since job verbose output isn't captured)
                $now = Get-Date
                $beginDate = $now.AddDays(-1).ToString("yyyyMMdd")
                $endDate = $now.AddDays(1).ToString("yyyyMMdd")
                $rangeApiUrl = "https://api.tidesandcurrents.noaa.gov/api/prod/datagetter?product=predictions&datum=mllw&station=$($noaaStation.stationId)&begin_date=$beginDate&end_date=$endDate&interval=hilo&format=json&units=english&time_zone=lst_ldt"
                Write-Verbose "NOAA Tide API call: GET $rangeApiUrl"
                Write-Verbose "NOAA Tide API: Fetching predictions for date range (yesterday through tomorrow) in a single call"
                
                # Start tide predictions fetch asynchronously if we have time
                $tidePredictionsJob = Start-Job -ScriptBlock {
                    param($stationId, $timeZone)
                    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
                    
                    function Get-NoaaTidePredictionsForDateRange {
                        param([string]$StationId, [string]$BeginDate, [string]$EndDate)
                        try {
                            # Use begin_date and end_date to get a date range (yesterday through tomorrow)
                            $apiUrl = "https://api.tidesandcurrents.noaa.gov/api/prod/datagetter?product=predictions&datum=mllw&station=$StationId&begin_date=$BeginDate&end_date=$EndDate&interval=hilo&format=json&units=english&time_zone=lst_ldt"
                            Write-Verbose "NOAA Tide API call: GET $apiUrl"
                            $response = Invoke-RestMethod -Uri $apiUrl -Method Get -ErrorAction Stop -TimeoutSec 10
                            if ($response.predictions) {
                                $predictionsCount = $response.predictions.Count
                                Write-Verbose "NOAA Tide Predictions API response received successfully: $predictionsCount predictions for date range $BeginDate to $EndDate"
                                return $response.predictions
                            }
                            Write-Verbose "NOAA Tide Predictions API response received: 0 predictions for date range $BeginDate to $EndDate"
                            return $null
                        } catch {
                            Write-Verbose "NOAA Tide Predictions API call failed for date range ${BeginDate} to ${EndDate}: $($_.Exception.Message)"
                            return $null
                        }
                    }
                    
                    try {
                        $verboseMessages = @()
                        $now = Get-Date
                        $verboseMessages += "Current time (reference): $($now.ToString('yyyy-MM-dd HH:mm:ss'))"
                        $verboseMessages += ""
                        
                        # Make a single API call for yesterday through tomorrow (3-day range)
                        $yesterdayDate = $now.AddDays(-1)
                        $tomorrowDate = $now.AddDays(1)
                        $beginDate = $yesterdayDate.ToString("yyyyMMdd")
                        $endDate = $tomorrowDate.ToString("yyyyMMdd")
                        
                        $verboseMessages += "Fetching tide predictions for date range: $beginDate to $endDate (yesterday through tomorrow)"
                        $allPredictions = Get-NoaaTidePredictionsForDateRange -StationId $stationId -BeginDate $beginDate -EndDate $endDate
                        
                        if (-not $allPredictions -or $allPredictions.Count -eq 0) {
                            return @{ Success = $false; Error = "No predictions returned from API" }
                        }
                        
                        $totalPredictions = $allPredictions.Count
                        $apiCalls = @("range: $beginDate to $endDate")
                        $verboseMessages += ""
                        
                        $verboseMessages += "All predictions ($($allPredictions.Count) tides) from date range:"
                        foreach ($pred in $allPredictions) {
                            $predTime = if ($pred.t -match "\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}") {
                                [DateTime]::ParseExact($pred.t, "yyyy-MM-dd HH:mm", $null)
                            } else { $null }
                            $timeStr = if ($predTime) { $predTime.ToString("yyyy-MM-dd HH:mm") } else { $pred.t }
                            $isFuture = if ($predTime) { ($predTime -gt $now) } else { $false }
                            $futureStr = if ($isFuture) { " [FUTURE]" } else { " [PAST]" }
                            # Add each tide prediction as a separate array element
                            $verboseMessages += "  $timeStr : $($pred.type) $($pred.v)ft$futureStr"
                        }
                        
                        $verboseMessages += ""
                        
                        # Process all predictions to find last and next tide
                        $lastTide = $null
                        $nextTide = $null
                        
                        foreach ($prediction in $allPredictions) {
                            $timeStr = $prediction.t
                            if ($timeStr -match "\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}") {
                                $tideTime = [DateTime]::ParseExact($timeStr, "yyyy-MM-dd HH:mm", $null)
                                $height = [double]$prediction.v
                                $type = $prediction.type
                                $tideInfo = @{ Time = $tideTime; Height = $height; Type = $type }
                                
                                # Find last tide (most recent past tide)
                                if ($tideTime -le $now) {
                                    if ($null -eq $lastTide -or $tideTime -gt $lastTide.Time) {
                                        $lastTide = $tideInfo
                                    }
                                }
                                
                                # Find next tide (earliest future tide)
                                if ($tideTime -gt $now) {
                                    if ($null -eq $nextTide -or $tideTime -lt $nextTide.Time) {
                                        $nextTide = $tideInfo
                                    }
                                }
                            }
                        }
                        
                        $tideData = @{ LastTide = $lastTide; NextTide = $nextTide }
                        $verboseMessages += "Summary:"
                        if ($lastTide) {
                            $verboseMessages += "  Found last tide: $($lastTide.Time.ToString('yyyy-MM-dd HH:mm')) $($lastTide.Type) $($lastTide.Height)ft"
                        } else {
                            $verboseMessages += "  No last tide found"
                        }
                        
                        if ($nextTide) {
                            $verboseMessages += "  Found next tide: $($nextTide.Time.ToString('yyyy-MM-dd HH:mm')) $($nextTide.Type) $($nextTide.Height)ft"
                        } else {
                            $verboseMessages += "  No next tide found"
                        }
                        
                        # Verbose messages are already built above during processing
                        
                        return @{ Success = $true; TideData = $tideData; TotalPredictions = $totalPredictions; ApiCalls = $apiCalls; VerboseMessages = $verboseMessages }
                    } catch {
                        return @{ Success = $false; Error = $_.Exception.Message }
                    }
                } -ArgumentList $noaaStation.stationId, $timeZone
                
                # Process tide predictions (wait briefly if needed)
                $tideData = $null
                $jobResult = Wait-Job -Job $tidePredictionsJob -Timeout 5
                if ($jobResult) {
                    $tideJobData = $tidePredictionsJob | Receive-Job
                    Remove-Job -Job $tidePredictionsJob
                    if ($tideJobData.Success -and $tideJobData.TideData) {
                        # Log successful response with details from job
                        $predictionCount = if ($tideJobData.TotalPredictions) { $tideJobData.TotalPredictions } else { "unknown" }
                        $datesCalled = if ($tideJobData.ApiCalls) { ($tideJobData.ApiCalls -join ", ") } else { "today" }
                        Write-Verbose "NOAA Tide Predictions API response received successfully: $predictionCount predictions retrieved (dates: $datesCalled) for station $($noaaStation.stationId)"
                        
                        # Output verbose messages from the job - each on its own line
                        if ($tideJobData.VerboseMessages -and $tideJobData.VerboseMessages.Count -gt 0) {
                            # Ensure we have an array and output each message separately
                            $messages = if ($tideJobData.VerboseMessages -is [array]) {
                                $tideJobData.VerboseMessages
                            } else {
                                @($tideJobData.VerboseMessages)
                            }
                            
                            foreach ($msg in $messages) {
                                $msgStr = if ($null -eq $msg) { "" } else { $msg.ToString().Trim() }
                                if ([string]::IsNullOrWhiteSpace($msgStr)) {
                                    # Output blank line for spacing
                                    Write-Verbose ""
                                } else {
                                    # Output each message separately to ensure line breaks
                                    Write-Verbose $msgStr
                                }
                            }
                        }
                        
                        $tideData = $tideJobData.TideData
                    }
                } else {
                    # Job still running, process synchronously as fallback
                    Write-Verbose "Tide predictions job taking longer, processing synchronously"
                    Remove-Job -Job $tidePredictionsJob -Force
                    $tideData = Get-NoaaTidePredictions -StationId $noaaStation.stationId -TimeZone $timeZone
                }
                
                if ($tideData) {
                    Write-Host "Tides: " -ForegroundColor $DefaultColor -NoNewline
                    
                    # Display last tide if available
                    if ($tideData.LastTide) {
                        $lastHeight = "$([Math]::Round($tideData.LastTide.Height, 2))ft"
                        $lastTime = $tideData.LastTide.Time.ToString("HHmm")
                        $lastArrow = if ($tideData.LastTide.Type -eq "L") { "↓" } else { "↑" }
                        Write-Host "Last${lastArrow}: ${lastHeight}@${lastTime}" -ForegroundColor Gray -NoNewline
                    }
                    
                    # Add space between last and next if both are present
                    if ($tideData.LastTide -and $tideData.NextTide) {
                        Write-Host " " -ForegroundColor Gray -NoNewline
                    }
                    
                    # Display next tide if available
                    if ($tideData.NextTide) {
                        $nextHeight = "$([Math]::Round($tideData.NextTide.Height, 2))ft"
                        $nextTime = $tideData.NextTide.Time.ToString("HHmm")
                        $nextArrow = if ($tideData.NextTide.Type -eq "H") { "↑" } else { "↓" }
                        Write-Host "Next${nextArrow}: ${nextHeight}@${nextTime}" -ForegroundColor Gray -NoNewline
                    }
                    
                    Write-Host ""  # New line
                }
            } catch {
                Write-Verbose "Error fetching tide predictions: $($_.Exception.Message)"
                # Silently fail - don't display tide info if there's an error
            }
        }
    }
    catch {
        Write-Verbose "Error fetching NOAA station data: $($_.Exception.Message)"
        # Silently fail - don't display NOAA Resources if there's an error
    }
}

# Function to fetch NOAA tide predictions for a specific date
function Get-NoaaTidePredictionsForDate {
    param(
        [string]$StationId,
        [string]$Date  # Format: "today", "YYYYMMDD", or date string
    )
    
    try {
        # Use begin_date and end_date for specific dates (more reliable than date parameter)
        # For "today", use the date parameter; for specific dates, use begin_date/end_date
        if ($Date -eq "today") {
            $apiUrl = "https://api.tidesandcurrents.noaa.gov/api/prod/datagetter?product=predictions&datum=mllw&station=$StationId&date=$Date&interval=hilo&format=json&units=english&time_zone=lst_ldt"
        } else {
            # For specific dates, use begin_date and end_date to ensure correct date range
            $apiUrl = "https://api.tidesandcurrents.noaa.gov/api/prod/datagetter?product=predictions&datum=mllw&station=$StationId&begin_date=$Date&end_date=$Date&interval=hilo&format=json&units=english&time_zone=lst_ldt"
        }
        Write-Verbose "NOAA Tide API call: GET $apiUrl"
        
        $response = Invoke-RestMethod -Uri $apiUrl -Method Get -ErrorAction Stop -TimeoutSec 10
        
        if (-not $response.predictions -or $response.predictions.Count -eq 0) {
            Write-Verbose "NOAA Tide Predictions API response received: 0 predictions for date '$Date'"
            return $null
        }
        
        $predictionsCount = $response.predictions.Count
        Write-Verbose "NOAA Tide Predictions API response received successfully: $predictionsCount predictions for date '$Date'"
        return $response.predictions
    }
    catch {
        Write-Verbose "Error fetching tide predictions for date '$Date': $($_.Exception.Message)"
        return $null
    }
}

# Function to fetch NOAA tide predictions (synchronous version for fallback)
function Get-NoaaTidePredictions {
    param(
        [string]$StationId,
        [string]$TimeZone
    )
    
    try {
        $now = Get-Date
        
        # Make a single API call for yesterday through tomorrow (3-day range)
        $yesterdayDate = $now.AddDays(-1)
        $tomorrowDate = $now.AddDays(1)
        $beginDate = $yesterdayDate.ToString("yyyyMMdd")
        $endDate = $tomorrowDate.ToString("yyyyMMdd")
        
        Write-Verbose "Fetching tide predictions for date range: $beginDate to $endDate"
        $allPredictions = Get-NoaaTidePredictionsForDateRange -StationId $StationId -BeginDate $beginDate -EndDate $endDate
        
        if (-not $allPredictions -or $allPredictions.Count -eq 0) {
            Write-Verbose "No tide predictions returned from API"
            return $null
        }
        
        # Process all predictions to find last and next tide
        $lastTide = $null
        $nextTide = $null
        
        foreach ($prediction in $allPredictions) {
            $timeStr = $prediction.t
            if ($timeStr -match "\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}") {
                $tideTime = [DateTime]::ParseExact($timeStr, "yyyy-MM-dd HH:mm", $null)
                $height = [double]$prediction.v
                $type = $prediction.type
                $tideInfo = @{ Time = $tideTime; Height = $height; Type = $type }
                
                # Find last tide (most recent past tide)
                if ($tideTime -le $now) {
                    if ($null -eq $lastTide -or $tideTime -gt $lastTide.Time) {
                        $lastTide = $tideInfo
                    }
                }
                
                # Find next tide (earliest future tide)
                if ($tideTime -gt $now) {
                    if ($null -eq $nextTide -or $tideTime -lt $nextTide.Time) {
                        $nextTide = $tideInfo
                    }
                }
            }
        }
        
        $tideData = @{ LastTide = $lastTide; NextTide = $nextTide }
        return $tideData
    }
    catch {
        Write-Verbose "Error fetching tide predictions: $($_.Exception.Message)"
        return $null
    }
}

# Function to display interactive mode controls
function Show-InteractiveControls {
    param(
        [bool]$IsHourlyMode = $false,
        [bool]$IsRainMode = $false,
        [bool]$IsWindMode = $false,
        [bool]$IsTerseMode = $false,
        [bool]$IsDailyMode = $false,
        [bool]$IsFullMode = $false,
        [bool]$IsObservationsMode = $false
    )
    
    # Don't show controls if bar is hidden
    if (-not $script:showControlBar) {
        return
    }
    
    Write-Host ""
    if ($IsHourlyMode) {
        Write-Host "Mode: " -ForegroundColor Green -NoNewline
        Write-Host "Scroll[" -ForegroundColor White -NoNewline; Write-Host "↑↓" -ForegroundColor Cyan -NoNewline; Write-Host "], " -ForegroundColor White -NoNewline
        if (-not $IsFullMode) { Write-Host "F" -ForegroundColor Cyan -NoNewline; Write-Host "ull " -ForegroundColor White -NoNewline }
        if (-not $IsTerseMode) { Write-Host "T" -ForegroundColor Cyan -NoNewline; Write-Host "erse " -ForegroundColor White -NoNewline }
        if (-not $IsDailyMode) { Write-Host "D" -ForegroundColor Cyan -NoNewline; Write-Host "aily " -ForegroundColor White -NoNewline }
        if (-not $IsRainMode) { Write-Host "R" -ForegroundColor Cyan -NoNewline; Write-Host "ain " -ForegroundColor White -NoNewline }
        if (-not $IsWindMode) { Write-Host "W" -ForegroundColor Cyan -NoNewline; Write-Host "ind" -ForegroundColor White }
    } else {
        Write-Host "Mode: " -ForegroundColor Green -NoNewline
        if (-not $IsFullMode) { Write-Host "F" -ForegroundColor Cyan -NoNewline; Write-Host "ull " -ForegroundColor White -NoNewline }
        if (-not $IsTerseMode) { Write-Host "T" -ForegroundColor Cyan -NoNewline; Write-Host "erse " -ForegroundColor White -NoNewline }
        if (-not $IsDailyMode) { Write-Host "D" -ForegroundColor Cyan -NoNewline; Write-Host "aily " -ForegroundColor White -NoNewline }
        if (-not $IsHourlyMode) { Write-Host "H" -ForegroundColor Cyan -NoNewline; Write-Host "ourly " -ForegroundColor White -NoNewline }
        if (-not $IsRainMode) { Write-Host "R" -ForegroundColor Cyan -NoNewline; Write-Host "ain " -ForegroundColor White -NoNewline }
        if (-not $IsWindMode) { Write-Host "W" -ForegroundColor Cyan -NoNewline; Write-Host "ind " -ForegroundColor White -NoNewline }
        if (-not $IsObservationsMode) { Write-Host "O" -ForegroundColor Cyan -NoNewline; Write-Host "bservations " -ForegroundColor White -NoNewline }
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
        [string]$TimeZone,
        [double]$Lat,
        [double]$Lon,
        [int]$ElevationFeet,
        [string]$RadarStation,
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
        [bool]$ShowLocationInfo = $true,
        [string]$MoonPhase,
        [string]$MoonEmoji,
        [bool]$IsFullMoon,
        [string]$NextFullMoonDate,
        [bool]$IsNewMoon,
        [bool]$ShowNextFullMoon,
        [bool]$ShowNextNewMoon,
        [string]$NextNewMoonDate,
        [object]$SunriseTime = $null,
        [object]$SunsetTime = $null,
        [bool]$IsPolarNight = $false,
        [bool]$IsPolarDay = $false
    )
    
    if ($ShowCurrentConditions) {
        # Format sunrise: date/time in 24-hour format if polar night/day, otherwise time in 12-hour format
        # During polar night: shows next sunrise; during polar day: shows last sunrise
        $sunriseTimeStr = if ($null -ne $SunriseTime) { 
            if ($IsPolarNight -or $IsPolarDay) { 
                $SunriseTime.ToString('MM/dd HH:mm') 
            } else { 
                $SunriseTime.ToString('h:mm tt') 
            }
        } else { 
            "N/A" 
        }
        # Format sunset: date/time in 24-hour format if polar night/day, otherwise time in 12-hour format
        # During polar night: shows last sunset; during polar day: shows next sunset
        $sunsetTimeStr = if ($null -ne $SunsetTime) { 
            if ($IsPolarNight -or $IsPolarDay) { 
                $SunsetTime.ToString('MM/dd HH:mm') 
            } else { 
                $SunsetTime.ToString('h:mm tt') 
            }
        } else { 
            "N/A" 
        }
        Show-CurrentConditions -City $City -State $State -WeatherIcon $WeatherIcon -CurrentConditions $CurrentConditions -CurrentTemp $CurrentTemp -TempColor $TempColor -CurrentTempTrend $CurrentTempTrend -CurrentWind $CurrentWind -WindColor $WindColor -CurrentWindDir $CurrentWindDir -WindGust $WindGust -CurrentHumidity $CurrentHumidity -CurrentDewPoint $CurrentDewPoint -CurrentPrecipProb $CurrentPrecipProb -CurrentTimeLocal $dataFetchTime -SunriseTime $sunriseTimeStr -SunsetTime $sunsetTimeStr -DefaultColor $DefaultColor -AlertColor $AlertColor -TitleColor $TitleColor -InfoColor $InfoColor -MoonPhase $MoonPhase -MoonEmoji $MoonEmoji -IsFullMoon $IsFullMoon -NextFullMoonDate $NextFullMoonDate -IsNewMoon $IsNewMoon -ShowNextFullMoon $ShowNextFullMoon -ShowNextNewMoon $ShowNextNewMoon -NextNewMoonDate $NextNewMoonDate
    }

    if ($ShowTodayForecast) {
        Show-ForecastText -Title $TodayPeriodName -ForecastText $TodayForecast -TitleColor $TitleColor -DefaultColor $DefaultColor
    }

    if ($ShowTomorrowForecast) {
        Show-ForecastText -Title $TomorrowPeriodName -ForecastText $TomorrowForecast -TitleColor $TitleColor -DefaultColor $DefaultColor
    }

    if ($ShowHourlyForecast) {
        Show-HourlyForecast -HourlyData $HourlyData -TitleColor $TitleColor -DefaultColor $DefaultColor -AlertColor $AlertColor -IsInteractive $false -SunriseTime $SunriseTime -SunsetTime $SunsetTime -City $City -ShowCityInTitle $true -TimeZone $TimeZone
    }

    if ($ShowSevenDayForecast) {
        Show-SevenDayForecast -ForecastData $ForecastData -TitleColor $TitleColor -DefaultColor $DefaultColor -AlertColor $AlertColor -SunriseTime $SunriseTime -SunsetTime $SunsetTime -City $City -ShowCityInTitle $true -Latitude $Lat -Longitude $Lon -TimeZone $TimeZone
    }

    if ($ShowAlerts) {
        Show-WeatherAlerts -AlertsData $AlertsData -AlertColor $AlertColor -DefaultColor $DefaultColor -InfoColor $InfoColor -ShowDetails $ShowAlertDetails -TimeZone $TimeZone
    }

    if ($ShowLocationInfo) {
        Show-LocationInfo -TimeZone $TimeZone -Lat $Lat -Lon $Lon -ElevationFeet $ElevationFeet -RadarStation $RadarStation -TitleColor $TitleColor -DefaultColor $DefaultColor
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
        [string]$City,
        [string]$TimeZone = ""
    )
    
    Write-Host ""
    # Extract as many words as fit within 20 characters to keep title short
    $cityName = Get-TruncatedCityName -CityName $City -MaxLength 20
    $headerText = "*** $cityName Rain Outlook ***"
    $padding = [Math]::Max(0, (34 - $headerText.Length) / 2)
    $paddedHeader = " " * $padding + $headerText
    Write-Host $paddedHeader -ForegroundColor $TitleColor
    
    $hourlyPeriods = $hourlyData.properties.periods
    $totalHours = [Math]::Min($hourlyPeriods.Count, 96)  # Use 96 hours for rain mode
    
    # Group periods by day using destination timezone
    $daysData = @{}
    $destTimeZone = $null
    if ($TimeZone) {
        $destTimeZone = Get-ResolvedTimeZoneInfo -TimeZoneId $TimeZone
    }
    
    foreach ($period in $hourlyPeriods[0..($totalHours-1)]) {
        # Parse and convert to destination timezone
        $periodTimeOffset = [DateTimeOffset]::Parse($period.startTime)
        if ($destTimeZone) {
            $periodTimeLocal = [System.TimeZoneInfo]::ConvertTime($periodTimeOffset, $destTimeZone)
        } else {
            $periodTimeLocal = $periodTimeOffset
        }
        
        $dayKey = $periodTimeLocal.ToString("yyyy-MM-dd")
        $hour = $periodTimeLocal.Hour
        
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
        
        # Parse day key as date for day name (using destination timezone if available)
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
        [string]$City,
        [string]$TimeZone = ""
    )
    
    Write-Host ""
    # Extract as many words as fit within 20 characters to keep title short
    $cityName = Get-TruncatedCityName -CityName $City -MaxLength 20
    $headerText = "*** $cityName Wind Outlook ***"
    $padding = [Math]::Max(0, (34 - $headerText.Length) / 2)
    $paddedHeader = " " * $padding + $headerText
    Write-Host $paddedHeader -ForegroundColor Green
    
    $hourlyPeriods = $hourlyData.properties.periods
    $totalHours = [Math]::Min($hourlyPeriods.Count, 96)  # Use 96 hours for wind mode
    
    # Group periods by day using destination timezone
    $daysData = @{}
    $destTimeZone = $null
    if ($TimeZone) {
        $destTimeZone = Get-ResolvedTimeZoneInfo -TimeZoneId $TimeZone
    }
    
    foreach ($period in $hourlyPeriods[0..($totalHours-1)]) {
        # Parse and convert to destination timezone
        $periodTimeOffset = [DateTimeOffset]::Parse($period.startTime)
        if ($destTimeZone) {
            $periodTimeLocal = [System.TimeZoneInfo]::ConvertTime($periodTimeOffset, $destTimeZone)
        } else {
            $periodTimeLocal = $periodTimeOffset
        }
        
        $dayKey = $periodTimeLocal.ToString("yyyy-MM-dd")
        $hour = $periodTimeLocal.Hour
        
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
    $precipPart = if ($PrecipProb -gt 0) { " ($PrecipProb%☔️)" } else { "" }
    
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

$tzInfo = $null
if ($timeZone) {
    $tzInfo = Get-ResolvedTimeZoneInfo -TimeZoneId $timeZone
}
# Calculate sunrise and sunset times using the location's timezone date (not the user's local date)
$locationToday = if ($tzInfo) { [System.TimeZoneInfo]::ConvertTime((Get-Date), $tzInfo).Date } else { (Get-Date).Date }
$sunTimes = Get-SunriseSunset -Latitude $lat -Longitude $lon -Date $locationToday -TimeZoneId $timeZone
$sunriseTime = $sunTimes.Sunrise
$sunsetTime = $sunTimes.Sunset
$isPolarNight = $sunTimes.IsPolarNight
$isPolarDay = $sunTimes.IsPolarDay

# Store in script scope for interactive mode
$script:sunriseTime = $sunriseTime
$script:sunsetTime = $sunsetTime
$script:isPolarNight = $isPolarNight
$script:isPolarDay = $isPolarDay

# Ensure sunrise/sunset are properly typed (can be null for polar regions)
if ($null -ne $sunriseTime -and $sunriseTime -isnot [DateTime]) {
    $sunriseTime = [DateTime]$sunriseTime
}
if ($null -ne $sunsetTime -and $sunsetTime -isnot [DateTime]) {
    $sunsetTime = [DateTime]$sunsetTime
}

# Calculate moon phase
$moonPhaseInfo = Get-MoonPhase -Date (Get-Date)

# Define color scheme for weather display
$defaultColor = "DarkCyan"
$alertColor = "Red"
$titleColor = "Green"
$infoColor = "Blue"
$detailedForecastColor = "Gray"

# Apply color coding based on weather conditions
# Temperature: Blue if too cold (<33°F), Red if too hot (>89°F)
if ([int]$currentTemp -lt $script:COLD_TEMP_THRESHOLD) {
    $tempColor = "Blue"
} elseif ([int]$currentTemp -gt $script:HOT_TEMP_THRESHOLD) {
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
elseif ($Observations.IsPresent) {
    # Observations mode: Show only historical observations
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

# Clear loading message before displaying data
Clear-HostWithDelay

# Display the weather report using the refactored function
if ($Rain.IsPresent) {
    # Rain mode: Show only rain likelihood forecast with sparklines
    Show-RainForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city -TimeZone $timeZone
} elseif ($Wind.IsPresent) {
    # Wind mode: Show only wind outlook forecast with direction glyphs
    Show-WindForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city -TimeZone $timeZone
} elseif ($Daily.IsPresent) {
    # Daily mode: Show enhanced 7-day forecast with wind info and detailed forecasts
    Show-SevenDayForecast -ForecastData $forecastData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -SunriseTime $sunriseTime -SunsetTime $sunsetTime -IsEnhancedMode $true -City $city -ShowCityInTitle $true -Latitude $lat -Longitude $lon -TimeZone $timeZone
    # Exit only if -x flag is present, otherwise continue to interactive mode
    if ($NoInteractive.IsPresent) {
        exit 0
    }
} elseif ($Observations.IsPresent) {
    # Observations mode: Show historical observations
    if ($null -ne $script:observationsData) {
        Show-Observations -ObservationsData $script:observationsData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -City $city -ShowCityInTitle $true -TimeZone $timeZone -Latitude $lat -Longitude $lon
    } else {
        Write-Host "No historical observations available." -ForegroundColor $defaultColor
    }
    # Exit only if -x flag is present, otherwise continue to interactive mode
    if ($NoInteractive.IsPresent) {
        exit 0
    }
} else {
    Show-FullWeatherReport -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $dataFetchTime -TodayForecast $todayForecast -TodayPeriodName $todayPeriodName -TomorrowForecast $tomorrowForecast -TomorrowPeriodName $tomorrowPeriodName -HourlyData $hourlyData -ForecastData $forecastData -AlertsData $alertsData -TimeZone $timeZone -Lat $lat -Lon $lon -ElevationFeet $elevationFeet -RadarStation $radarStation -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor -ShowCurrentConditions $showCurrentConditions -ShowTodayForecast $showTodayForecast -ShowTomorrowForecast $showTomorrowForecast -ShowHourlyForecast $showHourlyForecast -ShowSevenDayForecast $showSevenDayForecast -ShowAlerts $showAlerts -ShowAlertDetails $showAlertDetails -ShowLocationInfo $showLocationInfo -MoonPhase $moonPhaseInfo.Name -MoonEmoji $moonPhaseInfo.Emoji -IsFullMoon $moonPhaseInfo.IsFullMoon -NextFullMoonDate $moonPhaseInfo.NextFullMoon -IsNewMoon $moonPhaseInfo.IsNewMoon -ShowNextFullMoon $moonPhaseInfo.ShowNextFullMoon -ShowNextNewMoon $moonPhaseInfo.ShowNextNewMoon -NextNewMoonDate $moonPhaseInfo.NextNewMoon -SunriseTime $sunriseTime -SunsetTime $sunsetTime -IsPolarNight $isPolarNight -IsPolarDay $isPolarDay
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

if ($isInteractiveEnvironment -and -not $NoInteractive.IsPresent) {
    Write-Verbose "Parent:$parentName - Interactive environment detected"
    
    # Interactive mode variables
    $isHourlyMode = $Hourly.IsPresent  # State tracking for hourly forecast mode
    $isRainMode = $Rain.IsPresent  # State tracking for rain forecast mode
    $isWindMode = $Wind.IsPresent  # State tracking for wind forecast mode
    $isTerseMode = $Terse.IsPresent  # State tracking for terse mode
    $isDailyMode = $Daily.IsPresent  # State tracking for daily forecast mode
    $isObservationsMode = $Observations.IsPresent  # State tracking for observations mode
    $autoUpdateEnabled = -not $NoAutoUpdate.IsPresent  # State tracking for auto-updates
    if (-not $autoUpdateEnabled) {
        Write-Verbose "Auto-updates disabled via command line flag"
    }
    $hourlyScrollIndex = 0
    $totalHourlyPeriods = [Math]::Min($script:hourlyData.properties.periods.Count, 48)  # Limit to 48 hours
    
    # Initialize mode state tracking
    Write-Verbose "Interactive mode initialized - Hourly: $isHourlyMode, Rain: $isRainMode, Wind: $isWindMode, Terse: $isTerseMode, Daily: $isDailyMode, Auto-Update: $autoUpdateEnabled"
    
    # Check if interactive mode is supported
    if ([System.Console]::IsInputRedirected) {
        Write-Host "Interactive mode not supported: Input is redirected. Exiting..." -ForegroundColor Yellow
        return
    }
    
    # Additional console capability check
    try {
        [System.Console]::KeyAvailable | Out-Null
        Write-Verbose "Console key detection test passed"
    } catch {
        Write-Host "Interactive mode not supported: Console key detection failed. Exiting..." -ForegroundColor Yellow
        return
    }
    
    # If starting in hourly mode, show hourly forecast first
    if ($isHourlyMode) {
        Clear-HostWithDelay
        Show-HourlyForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true -SunriseTime $sunriseTime -SunsetTime $sunsetTime -City $city -ShowCityInTitle $true -TimeZone $timeZone
        Show-InteractiveControls -IsHourlyMode $true -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $false
    } elseif ($Rain.IsPresent) {
        # If starting in rain mode, show rain forecast first
        Clear-HostWithDelay
        $isRainMode = $true
        Show-RainForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city -TimeZone $timeZone
        Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
    } elseif ($Wind.IsPresent) {
        # If starting in wind mode, show wind forecast first
        Clear-HostWithDelay
        $isWindMode = $true
        Show-WindForecast -HourlyData $hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city -TimeZone $timeZone
        Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
    } elseif ($Observations.IsPresent) {
        # If starting in Observations mode, show observations first
        Clear-HostWithDelay
        $isObservationsMode = $true
        if ($null -ne $script:observationsData) {
            Show-Observations -ObservationsData $script:observationsData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -City $city -ShowCityInTitle $true -TimeZone $timeZone -Latitude $lat -Longitude $lon
        } else {
            Write-Host "No historical observations available." -ForegroundColor $defaultColor
        }
        Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
    } else {
        # Interactive mode: Listen for keyboard input to switch between display modes
        Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
    }
    
    while ($true) {
        try {
            # Remove any orphaned completed/failed jobs to prevent memory growth over long runs
            Get-Job -ErrorAction SilentlyContinue | Where-Object {
                ($_.State -eq 'Completed' -or $_.State -eq 'Failed') -and
                $_ -ne $script:observationsPreloadJob -and $_ -ne $script:noaaStationsJob
            } | Remove-Job -Force -ErrorAction SilentlyContinue

            # Preload observations data in background if not already loaded (non-blocking)
            # Only attempt preload once to avoid infinite loops
            if ($null -eq $script:observationsData -and -not $script:observationsDataLoading -and -not $Observations.IsPresent -and $null -eq $script:observationsPreloadJob -and -not $script:observationsPreloadAttempted) {
                Write-Verbose "Preloading observations data in background for Observations mode"
                $script:observationsDataLoading = $true
                $script:observationsPreloadAttempted = $true
                $script:observationsPreloadJob = Start-Job -ScriptBlock {
                    param($pointsDataJson, $headersJson)
                    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
                    $errorInfo = @{}
                    try {
                        # Parse input data
                        try {
                            $pointsData = $pointsDataJson | ConvertFrom-Json
                            $headersObj = $headersJson | ConvertFrom-Json
                            # Convert headers PSCustomObject back to hashtable (JSON deserialization creates PSCustomObject)
                            $headers = @{}
                            $headersObj.PSObject.Properties | ForEach-Object {
                                $headers[$_.Name] = $_.Value
                            }
                            $errorInfo["Step"] = "ParseInput"
                            $errorInfo["Success"] = $true
                        } catch {
                            $errorInfo["Step"] = "ParseInput"
                            $errorInfo["Error"] = $_.Exception.Message
                            $errorInfo["Success"] = $false
                            return (@{ Error = $true; ErrorInfo = $errorInfo } | ConvertTo-Json -Compress)
                        }
                        
                        # Get observation stations URL
                        $observationStationsUrl = $pointsData.properties.observationStations
                        if (-not $observationStationsUrl) {
                            $errorInfo["Step"] = "GetStationsUrl"
                            $errorInfo["Error"] = "No observation stations URL found in points data"
                            $errorInfo["Success"] = $false
                            return (@{ Error = $true; ErrorInfo = $errorInfo } | ConvertTo-Json -Compress)
                        }
                        $errorInfo["StationsUrl"] = $observationStationsUrl
                        
                        # Fetch observation stations
                        try {
                            $stationsResponse = Invoke-RestMethod -Uri $observationStationsUrl -Method Get -Headers $headers -ErrorAction Stop -TimeoutSec 30
                            $errorInfo["Step"] = "FetchStations"
                            $errorInfo["Success"] = $true
                            $errorInfo["StationsCount"] = if ($stationsResponse.features) { $stationsResponse.features.Count } else { 0 }
                        } catch {
                            $errorInfo["Step"] = "FetchStations"
                            $errorInfo["Error"] = $_.Exception.Message
                            $errorInfo["StatusCode"] = if ($_.Exception.Response) { $_.Exception.Response.StatusCode } else { $null }
                            $errorInfo["Success"] = $false
                            return (@{ Error = $true; ErrorInfo = $errorInfo } | ConvertTo-Json -Compress)
                        }
                        
                        if (-not $stationsResponse.features -or $stationsResponse.features.Count -eq 0) {
                            $errorInfo["Step"] = "ValidateStations"
                            $errorInfo["Error"] = "No observation stations found in response"
                            $errorInfo["Success"] = $false
                            return (@{ Error = $true; ErrorInfo = $errorInfo } | ConvertTo-Json -Compress)
                        }
                        
                        $stationId = $stationsResponse.features[0].properties.stationIdentifier
                        $errorInfo["StationId"] = $stationId
                        
                        # Calculate time range
                        $endTime = Get-Date
                        $startTime = $endTime.AddDays(-7)
                        $startTimeStr = $startTime.ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
                        $endTimeStr = $endTime.ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
                        $errorInfo["StartTime"] = $startTimeStr
                        $errorInfo["EndTime"] = $endTimeStr
                        
                        # Fetch observations with pagination support
                        $observationsUrl = "https://api.weather.gov/stations/$stationId/observations?start=$startTimeStr&end=$endTimeStr"
                        $errorInfo["ObservationsUrl"] = $observationsUrl
                        
                        try {
                            # Collect all observations from all pages
                            $allFeatures = @()
                            $currentUrl = $observationsUrl
                            $pageCount = 0
                            $maxPages = 50  # Safety limit to prevent infinite loops
                            
                            while ($currentUrl -and $pageCount -lt $maxPages) {
                                $pageCount++
                                
                                $observationsResponse = Invoke-RestMethod -Uri $currentUrl -Method Get -Headers $headers -ErrorAction Stop -TimeoutSec 30
                                
                                # Add features from this page to our collection
                                if ($observationsResponse.features) {
                                    $allFeatures += $observationsResponse.features
                                }
                                
                                # Check for next page
                                $currentUrl = $null
                                if ($observationsResponse.pagination -and $observationsResponse.pagination.next) {
                                    $currentUrl = $observationsResponse.pagination.next
                                }
                            }
                            
                            if ($allFeatures.Count -eq 0) {
                                $errorInfo["Step"] = "FetchObservations"
                                $errorInfo["Error"] = "No observations collected from any page"
                                $errorInfo["Success"] = $false
                                return (@{ Error = $true; ErrorInfo = $errorInfo } | ConvertTo-Json -Compress)
                            }
                            
                            # Create a combined observations data object
                            $combinedObservationsData = @{
                                type = "FeatureCollection"
                                features = $allFeatures
                            }
                            
                            $errorInfo["Step"] = "FetchObservations"
                            $errorInfo["Success"] = $true
                            $errorInfo["ObservationsCount"] = $allFeatures.Count
                            $errorInfo["PagesFetched"] = $pageCount
                            
                            # Return success with data
                            return (@{ Error = $false; Data = ($combinedObservationsData | ConvertTo-Json -Depth 10); ErrorInfo = $errorInfo } | ConvertTo-Json -Compress)
                        } catch {
                            $errorInfo["Step"] = "FetchObservations"
                            $errorInfo["Error"] = $_.Exception.Message
                            $errorInfo["StatusCode"] = if ($_.Exception.Response) { $_.Exception.Response.StatusCode } else { $null }
                            $errorInfo["Success"] = $false
                            return (@{ Error = $true; ErrorInfo = $errorInfo } | ConvertTo-Json -Compress)
                        }
                    } catch {
                        $errorInfo["Step"] = "Unexpected"
                        $errorInfo["Error"] = $_.Exception.Message
                        $errorInfo["ExceptionType"] = $_.Exception.GetType().FullName
                        $errorInfo["Success"] = $false
                        return (@{ Error = $true; ErrorInfo = $errorInfo } | ConvertTo-Json -Compress)
                    }
                } -ArgumentList ($script:pointsDataForObservations | ConvertTo-Json -Depth 10), ($script:headersForObservations | ConvertTo-Json)
            }
            
            # Check if preload job completed
            if ($null -ne $script:observationsPreloadJob) {
                if ($script:observationsPreloadJob.State -eq 'Completed') {
                    $preloadResult = $script:observationsPreloadJob | Receive-Job
                    Remove-Job -Job $script:observationsPreloadJob
                    $script:observationsPreloadJob = $null
                    $script:observationsDataLoading = $false
                    
                    if ($null -ne $preloadResult -and -not [string]::IsNullOrWhiteSpace($preloadResult)) {
                        try {
                            $preloadJson = $preloadResult | ConvertFrom-Json
                            
                            # Check if job returned an error
                            if ($preloadJson.Error) {
                                $errorInfo = $preloadJson.ErrorInfo
                                Write-Verbose "Observations preload job failed at step: $($errorInfo.Step)"
                                Write-Verbose "Preload error: $($errorInfo.Error)"
                                if ($errorInfo.StatusCode) {
                                    Write-Verbose "Preload HTTP status: $($errorInfo.StatusCode)"
                                }
                                if ($errorInfo.StationsUrl) {
                                    Write-Verbose "Preload stations URL: $($errorInfo.StationsUrl)"
                                }
                                if ($errorInfo.ObservationsUrl) {
                                    Write-Verbose "Preload observations URL: $($errorInfo.ObservationsUrl)"
                                }
                                if ($errorInfo.StationId) {
                                    Write-Verbose "Preload station ID: $($errorInfo.StationId)"
                                }
                                if ($errorInfo.StationsCount) {
                                    Write-Verbose "Preload stations found: $($errorInfo.StationsCount)"
                                }
                                if ($errorInfo.ExceptionType) {
                                    Write-Verbose "Preload exception type: $($errorInfo.ExceptionType)"
                                }
                                # Set empty array to prevent retry
                                $script:observationsData = @()
                                    } else {
                                        # Success - process the data
                                        if ($preloadJson.Data) {
                                            $observationsData = $preloadJson.Data | ConvertFrom-Json
                                            $script:observationsData = Convert-ObservationsData -ObservationsData $observationsData -TimeZone $script:timeZoneForObservations
                                    if ($null -ne $script:observationsData) {
                                        Write-Verbose "Observations data preloaded and processed successfully: $($script:observationsData.Count) days"
                                        if ($preloadJson.ErrorInfo) {
                                            $errorInfo = $preloadJson.ErrorInfo
                                            Write-Verbose "Preload used station: $($errorInfo.StationId)"
                                            Write-Verbose "Preload observations count: $($errorInfo.ObservationsCount)"
                                        }
                                    } else {
                                        Write-Verbose "Observations preload data processing returned null"
                                    }
                                } else {
                                    Write-Verbose "Observations preload job returned success but no data field"
                                    $script:observationsData = @()
                                }
                            }
                        } catch {
                            Write-Verbose "Failed to parse preload job result: $($_.Exception.Message)"
                            Write-Verbose "Preload result was: $($preloadResult.Substring(0, [Math]::Min(200, $preloadResult.Length)))"
                            $script:observationsData = @()
                        }
                    } else {
                        Write-Verbose "Observations preload job returned null or empty result - will not retry"
                        $script:observationsData = @()
                    }
                } elseif ($script:observationsPreloadJob.State -eq 'Failed') {
                    $jobError = $null
                    try {
                        $jobError = $script:observationsPreloadJob | Receive-Job -ErrorVariable jobErrors 2>&1
                    } catch {
                        $jobError = $_.Exception.Message
                    }
                    Write-Verbose "Observations preload job failed with state: $($script:observationsPreloadJob.State)"
                    if ($jobError) {
                        Write-Verbose "Preload job error output: $jobError"
                    }
                    Remove-Job -Job $script:observationsPreloadJob -Force
                    $script:observationsPreloadJob = $null
                    $script:observationsDataLoading = $false
                    $script:observationsData = @()
                } elseif ($script:observationsPreloadJob.State -eq 'Running') {
                    # Job still running - no action needed (don't spam verbose messages)
                } else {
                    Write-Verbose "Observations preload job in unexpected state: $($script:observationsPreloadJob.State)"
                }
            }
            
            # Check if data is stale and refresh if needed (only if auto-update is enabled and interactive mode is supported)
            if ($autoUpdateEnabled -and -not [System.Console]::IsInputRedirected) {
                $timeSinceLastFetch = (Get-Date) - $script:dataFetchTime
                if ($timeSinceLastFetch.TotalSeconds -gt $dataStaleThreshold) {
                    Write-Verbose "Auto-refresh triggered - data is stale ($([math]::Round($timeSinceLastFetch.TotalSeconds, 1)) seconds old)"
                    $refreshSuccess = Update-WeatherData -Lat $lat -Lon $lon -Headers $headers -TimeZone $timeZone -UseRetryLogic $false
                    if ($refreshSuccess) {
                        # Recalculate moon phase and sunrise/sunset for current time
                        $moonPhaseInfo = Get-MoonPhase -Date (Get-Date)
                        # Use location's current date (not user's local date) for sunrise/sunset
                        $locationToday = if ($timeZone) { 
                            $tzInfo = Get-ResolvedTimeZoneInfo -TimeZoneId $timeZone
                            [System.TimeZoneInfo]::ConvertTime((Get-Date), $tzInfo).Date 
                        } else { 
                            (Get-Date).Date 
                        }
                        $sunTimes = Get-SunriseSunset -Latitude $lat -Longitude $lon -Date $locationToday -TimeZoneId $timeZone
                        $sunriseTime = $sunTimes.Sunrise
                        $sunsetTime = $sunTimes.Sunset
                        # Sync script-scoped display vars to local so re-render uses fresh data (including currentTempTrend)
                        $currentTemp = $script:currentTemp
                        $currentConditions = $script:currentConditions
                        $currentTempTrend = $script:currentTempTrend
                        $currentWind = $script:currentWind
                        $currentWindDir = $script:currentWindDir
                        $currentHumidity = $script:currentHumidity
                        $currentPrecipProb = $script:currentPrecipProb
                        $todayForecast = $script:todayForecast
                        $todayPeriodName = $script:todayPeriodName
                        $tomorrowForecast = $script:tomorrowForecast
                        $tomorrowPeriodName = $script:tomorrowPeriodName
                        
                        # Re-render current view with fresh data
                        Clear-HostWithDelay
                        if ($isHourlyMode) {
                            Show-HourlyForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true -SunriseTime $script:sunriseTime -SunsetTime $script:sunsetTime -City $city -ShowCityInTitle $true -TimeZone $timeZone
                            Show-InteractiveControls -IsHourlyMode $true -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $false
                        } elseif ($isRainMode) {
                            Show-RainForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city -TimeZone $timeZone
                            Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                        } elseif ($isWindMode) {
                            Show-WindForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city -TimeZone $timeZone
                            Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                        } elseif ($isTerseMode) {
                            # Preserve current mode - show terse mode if in terse mode
                            # Format sunrise: date/time in 24-hour format if polar night/day, otherwise time in 12-hour format
                            # During polar night: shows next sunrise; during polar day: shows last sunrise
                            $sunriseTimeStr = if ($null -ne $script:sunriseTime) { 
                                if ($script:isPolarNight -or $script:isPolarDay) { 
                                    $script:sunriseTime.ToString('MM/dd HH:mm') 
                                } else { 
                                    $script:sunriseTime.ToString('h:mm tt') 
                                }
                            } else { 
                                "N/A" 
                            }
                            # Format sunset: date/time in 24-hour format if polar night/day, otherwise time in 12-hour format
                            # During polar night: shows last sunset; during polar day: shows next sunset
                            $sunsetTimeStr = if ($null -ne $script:sunsetTime) { 
                                if ($script:isPolarNight -or $script:isPolarDay) { 
                                    $script:sunsetTime.ToString('MM/dd HH:mm') 
                                } else { 
                                    $script:sunsetTime.ToString('h:mm tt') 
                                }
                            } else { 
                                "N/A" 
                            }
                            Show-CurrentConditions -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $script:dataFetchTime -SunriseTime $sunriseTimeStr -SunsetTime $sunsetTimeStr -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor -MoonPhase $moonPhaseInfo.Name -MoonEmoji $moonPhaseInfo.Emoji -IsFullMoon $moonPhaseInfo.IsFullMoon -NextFullMoonDate $moonPhaseInfo.NextFullMoon -IsNewMoon $moonPhaseInfo.IsNewMoon -ShowNextFullMoon $moonPhaseInfo.ShowNextFullMoon -ShowNextNewMoon $moonPhaseInfo.ShowNextNewMoon -NextNewMoonDate $moonPhaseInfo.NextNewMoon
                            Show-ForecastText -Title $todayPeriodName -ForecastText $todayForecast -TitleColor $titleColor -DefaultColor $defaultColor
                            Show-WeatherAlerts -AlertsData $script:alertsData -AlertColor $alertColor -DefaultColor $defaultColor -InfoColor $infoColor -ShowDetails $false -TimeZone $timeZone
                            Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                        } elseif ($isDailyMode) {
                            # Preserve current mode - show daily mode if in daily mode
                            Show-SevenDayForecast -ForecastData $script:forecastData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -SunriseTime $script:sunriseTime -SunsetTime $script:sunsetTime -IsEnhancedMode $true -City $city -ShowCityInTitle $true -Latitude $lat -Longitude $lon -TimeZone $timeZone
                            Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                        } elseif ($isObservationsMode) {
                            # Preserve current mode - show Observations mode if in Observations mode
                            if ($null -ne $script:observationsData) {
                                Show-Observations -ObservationsData $script:observationsData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -City $city -ShowCityInTitle $true -TimeZone $timeZone -Latitude $lat -Longitude $lon
                            } else {
                                Write-Host "No historical observations available." -ForegroundColor $defaultColor
                            }
                            Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                        } elseif ($isHourlyMode) {
                            # Preserve current mode - show hourly mode if in hourly mode
                            Show-HourlyForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true -SunriseTime $script:sunriseTime -SunsetTime $script:sunsetTime -City $city -ShowCityInTitle $true -TimeZone $timeZone
                            Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $false
                        } elseif ($isRainMode) {
                            # Preserve current mode - show rain mode if in rain mode
                            Show-RainForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city -TimeZone $timeZone
                            Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $false
                        } elseif ($isWindMode) {
                            # Preserve current mode - show wind mode if in wind mode
                            Show-WindForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city -TimeZone $timeZone
                            Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $false
                        } elseif ($isTerseMode) {
                            # Preserve current mode - show terse mode if in terse mode
                            # Format sunrise: date/time in 24-hour format if polar night/day, otherwise time in 12-hour format
                            # During polar night: shows next sunrise; during polar day: shows last sunrise
                            $sunriseTimeStr = if ($null -ne $script:sunriseTime) { 
                                if ($script:isPolarNight -or $script:isPolarDay) { 
                                    $script:sunriseTime.ToString('MM/dd HH:mm') 
                                } else { 
                                    $script:sunriseTime.ToString('h:mm tt') 
                                }
                            } else { 
                                "N/A" 
                            }
                            # Format sunset: date/time in 24-hour format if polar night/day, otherwise time in 12-hour format
                            # During polar night: shows last sunset; during polar day: shows next sunset
                            $sunsetTimeStr = if ($null -ne $script:sunsetTime) { 
                                if ($script:isPolarNight -or $script:isPolarDay) { 
                                    $script:sunsetTime.ToString('MM/dd HH:mm') 
                                } else { 
                                    $script:sunsetTime.ToString('h:mm tt') 
                                }
                            } else { 
                                "N/A" 
                            }
                            Show-CurrentConditions -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $script:dataFetchTime -SunriseTime $sunriseTimeStr -SunsetTime $sunsetTimeStr -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor -MoonPhase $moonPhaseInfo.Name -MoonEmoji $moonPhaseInfo.Emoji -IsFullMoon $moonPhaseInfo.IsFullMoon -NextFullMoonDate $moonPhaseInfo.NextFullMoon -IsNewMoon $moonPhaseInfo.IsNewMoon -ShowNextFullMoon $moonPhaseInfo.ShowNextFullMoon -ShowNextNewMoon $moonPhaseInfo.ShowNextNewMoon -NextNewMoonDate $moonPhaseInfo.NextNewMoon
                            Show-ForecastText -Title $todayPeriodName -ForecastText $todayForecast -TitleColor $titleColor -DefaultColor $defaultColor
                            Show-WeatherAlerts -AlertsData $script:alertsData -AlertColor $alertColor -DefaultColor $defaultColor -InfoColor $infoColor -ShowDetails $false -TimeZone $timeZone
                            Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $false
                        } else {
                            # Default to full weather report if no specific mode is set
                            Show-FullWeatherReport -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $script:dataFetchTime -TodayForecast $todayForecast -TodayPeriodName $todayPeriodName -TomorrowForecast $tomorrowForecast -TomorrowPeriodName $tomorrowPeriodName -HourlyData $script:hourlyData -ForecastData $script:forecastData -AlertsData $script:alertsData -TimeZone $timeZone -Lat $lat -Lon $lon -ElevationFeet $elevationFeet -RadarStation $radarStation -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor -ShowCurrentConditions $true -ShowTodayForecast $true -ShowTomorrowForecast $true -ShowHourlyForecast $true -ShowSevenDayForecast $true -ShowAlerts $true -ShowAlertDetails $true -ShowLocationInfo $true -MoonPhase $moonPhaseInfo.Name -MoonEmoji $moonPhaseInfo.Emoji -IsFullMoon $moonPhaseInfo.IsFullMoon -NextFullMoonDate $moonPhaseInfo.NextFullMoon -IsNewMoon $moonPhaseInfo.IsNewMoon -ShowNextFullMoon $moonPhaseInfo.ShowNextFullMoon -ShowNextNewMoon $moonPhaseInfo.ShowNextNewMoon -NextNewMoonDate $moonPhaseInfo.NextNewMoon -SunriseTime $script:sunriseTime -SunsetTime $script:sunsetTime -IsPolarNight $script:isPolarNight -IsPolarDay $script:isPolarDay
                                Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                            }
                        }
                    }
                }
            
            # Check for key input (non-blocking) - using same approach as bmon.ps1
            try {
                # Check if console supports key input
                if (-not [System.Console]::IsInputRedirected -and [System.Console]::KeyAvailable) {
                    $keyInfo = [System.Console]::ReadKey($true)
                
                # Handle keyboard input for interactive mode
                switch ($keyInfo.KeyChar) {
                'h' { # H key - Switch to hourly forecast only
                    Clear-HostWithDelay
                    $isHourlyMode = $true
                    $isRainMode = $false
                    $isWindMode = $false
                    $isTerseMode = $false
                    $isDailyMode = $false
                    $hourlyScrollIndex = 0  # Reset to first 12 hours
                            Show-HourlyForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true -SunriseTime $script:sunriseTime -SunsetTime $script:sunsetTime -City $city -ShowCityInTitle $true -TimeZone $timeZone
                    Show-InteractiveControls -IsHourlyMode $true -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $false
                }
                'd' { # D key - Switch to 7-day forecast only
                    Clear-HostWithDelay
                    $isHourlyMode = $false
                    $isRainMode = $false
                    $isWindMode = $false
                    $isTerseMode = $false
                    $isDailyMode = $true
                    Show-SevenDayForecast -ForecastData $script:forecastData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -SunriseTime $script:sunriseTime -SunsetTime $script:sunsetTime -IsEnhancedMode $true -City $city -ShowCityInTitle $true -Latitude $lat -Longitude $lon -TimeZone $timeZone
                    Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                }
                't' { # T key - Switch to terse mode (current + today + alerts)
                    Clear-HostWithDelay
                    $isHourlyMode = $false
                    $isRainMode = $false
                    $isWindMode = $false
                    $isTerseMode = $true
                    $isDailyMode = $false
                    # Format sunrise/sunset: date/time if polar night/day, otherwise time
                    $sunriseTimeStr = if ($null -ne $script:sunriseTime) { 
                        if ($script:isPolarNight -or $script:isPolarDay) { 
                            $script:sunriseTime.ToString('MM/dd HH:mm') 
                        } else { 
                            $script:sunriseTime.ToString('h:mm tt') 
                        }
                    } else { 
                        "N/A" 
                    }
                    $sunsetTimeStr = if ($null -ne $script:sunsetTime) { 
                        if ($script:isPolarNight -or $script:isPolarDay) { 
                            $script:sunsetTime.ToString('MM/dd HH:mm') 
                        } else { 
                            $script:sunsetTime.ToString('h:mm tt') 
                        }
                    } else { 
                        "N/A" 
                    }
                    Show-CurrentConditions -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $script:dataFetchTime -SunriseTime $sunriseTimeStr -SunsetTime $sunsetTimeStr -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor -MoonPhase $moonPhaseInfo.Name -MoonEmoji $moonPhaseInfo.Emoji -IsFullMoon $moonPhaseInfo.IsFullMoon -NextFullMoonDate $moonPhaseInfo.NextFullMoon -IsNewMoon $moonPhaseInfo.IsNewMoon -ShowNextFullMoon $moonPhaseInfo.ShowNextFullMoon -ShowNextNewMoon $moonPhaseInfo.ShowNextNewMoon -NextNewMoonDate $moonPhaseInfo.NextNewMoon
                    Show-ForecastText -Title $todayPeriodName -ForecastText $todayForecast -TitleColor $titleColor -DefaultColor $defaultColor
                    Show-WeatherAlerts -AlertsData $script:alertsData -AlertColor $alertColor -DefaultColor $defaultColor -InfoColor $infoColor -ShowDetails $false -TimeZone $timeZone
                    Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                }
                'f' { # F key - Switch to full weather report
                    Clear-HostWithDelay
                    $isHourlyMode = $false
                    $isRainMode = $false
                    $isWindMode = $false
                    $isTerseMode = $false
                    $isDailyMode = $false
                    $isObservationsMode = $false
                    Show-FullWeatherReport -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $script:dataFetchTime -TodayForecast $todayForecast -TodayPeriodName $todayPeriodName -TomorrowForecast $tomorrowForecast -TomorrowPeriodName $tomorrowPeriodName -HourlyData $script:hourlyData -ForecastData $script:forecastData -AlertsData $script:alertsData -TimeZone $timeZone -Lat $lat -Lon $lon -ElevationFeet $elevationFeet -RadarStation $radarStation -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor -ShowCurrentConditions $true -ShowTodayForecast $true -ShowTomorrowForecast $true -ShowHourlyForecast $true -ShowSevenDayForecast $true -ShowAlerts $true -ShowAlertDetails $true -ShowLocationInfo $true -MoonPhase $moonPhaseInfo.Name -MoonEmoji $moonPhaseInfo.Emoji -IsFullMoon $moonPhaseInfo.IsFullMoon -NextFullMoonDate $moonPhaseInfo.NextFullMoon -IsNewMoon $moonPhaseInfo.IsNewMoon -ShowNextFullMoon $moonPhaseInfo.ShowNextFullMoon -ShowNextNewMoon $moonPhaseInfo.ShowNextNewMoon -NextNewMoonDate $moonPhaseInfo.NextNewMoon -SunriseTime $script:sunriseTime -SunsetTime $script:sunsetTime -IsPolarNight $script:isPolarNight -IsPolarDay $script:isPolarDay
                    Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                }
                'r' { # R key - Switch to rain forecast mode
                    Clear-HostWithDelay
                    $isHourlyMode = $false
                    $isRainMode = $true
                    $isWindMode = $false
                    $isTerseMode = $false
                    $isDailyMode = $false
                    Show-RainForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city -TimeZone $timeZone
                    Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                }
                'u' { # U key - Toggle automatic updates
                    $autoUpdateEnabled = -not $autoUpdateEnabled
                    Write-Verbose "Auto-update toggled: $autoUpdateEnabled"
                    $statusMessage = if ($autoUpdateEnabled) { 
                        $script:dataFetchTime = Get-Date  # Reset timer when re-enabling
                        "Automatic Updates Enabled" 
                    } else { 
                        "Automatic Updates Disabled" 
                    }
                    
                    # Show status message briefly
                    Write-Host "`n$statusMessage" -ForegroundColor $(if ($autoUpdateEnabled) { "Green" } else { "Yellow" })
                    Start-Sleep -Milliseconds 800
                    
                    # Re-render current view
                    Clear-HostWithDelay
                    if ($isHourlyMode) {
                            Show-HourlyForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true -SunriseTime $script:sunriseTime -SunsetTime $script:sunsetTime -City $city -ShowCityInTitle $true -TimeZone $timeZone
                        Show-InteractiveControls -IsHourlyMode $true -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $false
                    } elseif ($isRainMode) {
                        Show-RainForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city -TimeZone $timeZone
                        Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                    } elseif ($isWindMode) {
                        Show-WindForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city -TimeZone $timeZone
                        Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                    } elseif ($isDailyMode) {
                        Show-SevenDayForecast -ForecastData $script:forecastData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -SunriseTime $script:sunriseTime -SunsetTime $script:sunsetTime -IsEnhancedMode $true -City $city -ShowCityInTitle $true -Latitude $lat -Longitude $lon -TimeZone $timeZone
                        Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                    } else {
                        # Preserve current mode - show terse mode if in terse mode
                        if ($isTerseMode) {
                            # Format sunrise: date/time in 24-hour format if polar night/day, otherwise time in 12-hour format
                            # During polar night: shows next sunrise; during polar day: shows last sunrise
                            $sunriseTimeStr = if ($null -ne $script:sunriseTime) { 
                                if ($script:isPolarNight -or $script:isPolarDay) { 
                                    $script:sunriseTime.ToString('MM/dd HH:mm') 
                                } else { 
                                    $script:sunriseTime.ToString('h:mm tt') 
                                }
                            } else { 
                                "N/A" 
                            }
                            # Format sunset: date/time in 24-hour format if polar night/day, otherwise time in 12-hour format
                            # During polar night: shows last sunset; during polar day: shows next sunset
                            $sunsetTimeStr = if ($null -ne $script:sunsetTime) { 
                                if ($script:isPolarNight -or $script:isPolarDay) { 
                                    $script:sunsetTime.ToString('MM/dd HH:mm') 
                                } else { 
                                    $script:sunsetTime.ToString('h:mm tt') 
                                }
                            } else { 
                                "N/A" 
                            }
                            Show-CurrentConditions -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $script:dataFetchTime -SunriseTime $sunriseTimeStr -SunsetTime $sunsetTimeStr -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor -MoonPhase $moonPhaseInfo.Name -MoonEmoji $moonPhaseInfo.Emoji -IsFullMoon $moonPhaseInfo.IsFullMoon -NextFullMoonDate $moonPhaseInfo.NextFullMoon -IsNewMoon $moonPhaseInfo.IsNewMoon -ShowNextFullMoon $moonPhaseInfo.ShowNextFullMoon -ShowNextNewMoon $moonPhaseInfo.ShowNextNewMoon -NextNewMoonDate $moonPhaseInfo.NextNewMoon
                            Show-ForecastText -Title $todayPeriodName -ForecastText $todayForecast -TitleColor $titleColor -DefaultColor $defaultColor
                            Show-WeatherAlerts -AlertsData $script:alertsData -AlertColor $alertColor -DefaultColor $defaultColor -InfoColor $infoColor -ShowDetails $false -TimeZone $timeZone
                            Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                        } else {
                            Show-FullWeatherReport -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $script:dataFetchTime -TodayForecast $todayForecast -TodayPeriodName $todayPeriodName -TomorrowForecast $tomorrowForecast -TomorrowPeriodName $tomorrowPeriodName -HourlyData $script:hourlyData -ForecastData $script:forecastData -AlertsData $script:alertsData -TimeZone $timeZone -Lat $lat -Lon $lon -ElevationFeet $elevationFeet -RadarStation $radarStation -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor -ShowCurrentConditions $showCurrentConditions -ShowTodayForecast $showTodayForecast -ShowTomorrowForecast $showTomorrowForecast -ShowHourlyForecast $showHourlyForecast -ShowSevenDayForecast $showSevenDayForecast -ShowAlerts $showAlerts -ShowAlertDetails $showAlertDetails -ShowLocationInfo $showLocationInfo -MoonPhase $moonPhaseInfo.Name -MoonEmoji $moonPhaseInfo.Emoji -IsFullMoon $moonPhaseInfo.IsFullMoon -NextFullMoonDate $moonPhaseInfo.NextFullMoon -IsNewMoon $moonPhaseInfo.IsNewMoon -ShowNextFullMoon $moonPhaseInfo.ShowNextFullMoon -ShowNextNewMoon $moonPhaseInfo.ShowNextNewMoon -NextNewMoonDate $moonPhaseInfo.NextNewMoon -SunriseTime $sunriseTime -SunsetTime $sunsetTime
                            Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                        }
                    }
                }
                'b' { # B key - Toggle control bar
                    $script:showControlBar = -not $script:showControlBar
                    Write-Verbose "Control bar toggled: $script:showControlBar"
                    $statusMessage = if ($script:showControlBar) { 
                        "Control Bar Enabled" 
                    } else { 
                        "Control Bar Disabled" 
                    }
                    
                    # Show status message briefly
                    Write-Host "`n$statusMessage" -ForegroundColor $(if ($script:showControlBar) { "Green" } else { "Yellow" })
                    Start-Sleep -Milliseconds 800
                    
                    # Re-render current view - use exact same logic as 'u' key handler
                    Clear-HostWithDelay
                    if ($isHourlyMode) {
                            Show-HourlyForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true -SunriseTime $script:sunriseTime -SunsetTime $script:sunsetTime -City $city -ShowCityInTitle $true -TimeZone $timeZone
                        Show-InteractiveControls -IsHourlyMode $true -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $false
                    } elseif ($isRainMode) {
                        Show-RainForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city -TimeZone $timeZone
                        Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                    } elseif ($isWindMode) {
                        Show-WindForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city -TimeZone $timeZone
                        Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                    } elseif ($isDailyMode) {
                        Show-SevenDayForecast -ForecastData $script:forecastData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -SunriseTime $script:sunriseTime -SunsetTime $script:sunsetTime -IsEnhancedMode $true -City $city -ShowCityInTitle $true -Latitude $lat -Longitude $lon -TimeZone $timeZone
                        Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                    } elseif ($isObservationsMode) {
                        if ($null -ne $script:observationsData -and ($script:observationsData -isnot [Array] -or $script:observationsData.Count -gt 0)) {
                            Show-Observations -ObservationsData $script:observationsData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -City $city -ShowCityInTitle $true -TimeZone $timeZone -Latitude $lat -Longitude $lon
                        } else {
                            Write-Host "No historical observations available." -ForegroundColor $defaultColor
                        }
                        Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                    } else {
                        # Preserve current mode - show terse mode if in terse mode
                        if ($isTerseMode) {
                            # Format sunrise: date/time in 24-hour format if polar night/day, otherwise time in 12-hour format
                            # During polar night: shows next sunrise; during polar day: shows last sunrise
                            $sunriseTimeStr = if ($null -ne $script:sunriseTime) { 
                                if ($script:isPolarNight -or $script:isPolarDay) { 
                                    $script:sunriseTime.ToString('MM/dd HH:mm') 
                                } else { 
                                    $script:sunriseTime.ToString('h:mm tt') 
                                }
                            } else { 
                                "N/A" 
                            }
                            # Format sunset: date/time in 24-hour format if polar night/day, otherwise time in 12-hour format
                            # During polar night: shows last sunset; during polar day: shows next sunset
                            $sunsetTimeStr = if ($null -ne $script:sunsetTime) { 
                                if ($script:isPolarNight -or $script:isPolarDay) { 
                                    $script:sunsetTime.ToString('MM/dd HH:mm') 
                                } else { 
                                    $script:sunsetTime.ToString('h:mm tt') 
                                }
                            } else { 
                                "N/A" 
                            }
                            Show-CurrentConditions -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $script:dataFetchTime -SunriseTime $sunriseTimeStr -SunsetTime $sunsetTimeStr -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor -MoonPhase $moonPhaseInfo.Name -MoonEmoji $moonPhaseInfo.Emoji -IsFullMoon $moonPhaseInfo.IsFullMoon -NextFullMoonDate $moonPhaseInfo.NextFullMoon -IsNewMoon $moonPhaseInfo.IsNewMoon -ShowNextFullMoon $moonPhaseInfo.ShowNextFullMoon -ShowNextNewMoon $moonPhaseInfo.ShowNextNewMoon -NextNewMoonDate $moonPhaseInfo.NextNewMoon
                            Show-ForecastText -Title $todayPeriodName -ForecastText $todayForecast -TitleColor $titleColor -DefaultColor $defaultColor
                            Show-WeatherAlerts -AlertsData $script:alertsData -AlertColor $alertColor -DefaultColor $defaultColor -InfoColor $infoColor -ShowDetails $false -TimeZone $timeZone
                            Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                        } else {
                            # Full mode - all mode flags are false
                            Show-FullWeatherReport -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $script:dataFetchTime -TodayForecast $todayForecast -TodayPeriodName $todayPeriodName -TomorrowForecast $tomorrowForecast -TomorrowPeriodName $tomorrowPeriodName -HourlyData $script:hourlyData -ForecastData $script:forecastData -AlertsData $script:alertsData -TimeZone $timeZone -Lat $lat -Lon $lon -ElevationFeet $elevationFeet -RadarStation $radarStation -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor -ShowCurrentConditions $true -ShowTodayForecast $true -ShowTomorrowForecast $true -ShowHourlyForecast $true -ShowSevenDayForecast $true -ShowAlerts $true -ShowAlertDetails $true -ShowLocationInfo $true -MoonPhase $moonPhaseInfo.Name -MoonEmoji $moonPhaseInfo.Emoji -IsFullMoon $moonPhaseInfo.IsFullMoon -NextFullMoonDate $moonPhaseInfo.NextFullMoon -IsNewMoon $moonPhaseInfo.IsNewMoon -ShowNextFullMoon $moonPhaseInfo.ShowNextFullMoon -ShowNextNewMoon $moonPhaseInfo.ShowNextNewMoon -NextNewMoonDate $moonPhaseInfo.NextNewMoon -SunriseTime $script:sunriseTime -SunsetTime $script:sunsetTime -IsPolarNight $script:isPolarNight -IsPolarDay $script:isPolarDay
                            Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                        }
                    }
                }
                'w' { # W key - Switch to wind forecast mode
                    Clear-HostWithDelay
                    $isHourlyMode = $false
                    $isRainMode = $false
                    $isWindMode = $true
                    $isTerseMode = $false
                    $isDailyMode = $false
                    $isObservationsMode = $false
                    Show-WindForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city -TimeZone $timeZone
                    Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                }
                'o' { # O key - Switch to observations mode
                    Clear-HostWithDelay
                    $isHourlyMode = $false
                    $isRainMode = $false
                    $isWindMode = $false
                    $isTerseMode = $false
                    $isDailyMode = $false
                    $isObservationsMode = $true
                    # Check if preload job is still running
                    if ($null -ne $script:observationsPreloadJob) {
                        if ($script:observationsPreloadJob.State -eq 'Running') {
                            Write-Host "Waiting for preloaded data..." -ForegroundColor Yellow
                            Wait-Job -Job $script:observationsPreloadJob | Out-Null
                        }
                        if ($script:observationsPreloadJob.State -eq 'Completed') {
                            $preloadResult = $script:observationsPreloadJob | Receive-Job
                            Remove-Job -Job $script:observationsPreloadJob
                            $script:observationsPreloadJob = $null
                            $script:observationsDataLoading = $false
                            if ($null -ne $preloadResult -and -not [string]::IsNullOrWhiteSpace($preloadResult)) {
                                try {
                                    $preloadJson = $preloadResult | ConvertFrom-Json
                                    
                                    # Check if job returned an error
                                    if ($preloadJson.Error) {
                                        $errorInfo = $preloadJson.ErrorInfo
                                        Write-Verbose "Observations preload job failed at step: $($errorInfo.Step)"
                                        Write-Verbose "Preload error: $($errorInfo.Error)"
                                        $script:observationsData = @()
                                    } else {
                                        # Success - process the data
                                        if ($preloadJson.Data) {
                                            $observationsData = $preloadJson.Data | ConvertFrom-Json
                                            $script:observationsData = Convert-ObservationsData -ObservationsData $observationsData -TimeZone $script:timeZoneForObservations
                                            if ($null -ne $script:observationsData) {
                                                Write-Verbose "Observations data preloaded and processed successfully: $($script:observationsData.Count) days"
                                            } else {
                                                Write-Verbose "Observations preload data processing returned null"
                                                $script:observationsData = @()
                                            }
                                        } else {
                                            Write-Verbose "Observations preload job returned success but no data field"
                                            $script:observationsData = @()
                                        }
                                    }
                                } catch {
                                    Write-Verbose "Failed to parse preload job result: $($_.Exception.Message)"
                                    Write-Verbose "Preload result was: $($preloadResult.Substring(0, [Math]::Min(200, $preloadResult.Length)))"
                                    $script:observationsData = @()
                                }
                            } else {
                                Write-Verbose "Observations preload job returned null or empty result"
                                $script:observationsData = @()
                            }
                        } elseif ($script:observationsPreloadJob.State -eq 'Failed') {
                            Remove-Job -Job $script:observationsPreloadJob -Force
                            $script:observationsPreloadJob = $null
                            $script:observationsDataLoading = $false
                        }
                        # Clear screen after preload completes to remove "Waiting for preloaded data..." message
                        Clear-HostWithDelay
                    }
                    # Fetch observations if not already fetched (check for null or empty array)
                    if ($null -eq $script:observationsData -or ($script:observationsData -is [Array] -and $script:observationsData.Count -eq 0)) {
                        # Reset the empty array flag if it was set
                        if ($script:observationsData -is [Array] -and $script:observationsData.Count -eq 0) {
                            $script:observationsData = $null
                        }
                        if ($script:observationsDataLoading) {
                            # If loading flag is stuck, reset it
                            $script:observationsDataLoading = $false
                        }
                        Write-Host "Loading Historical Data..." -ForegroundColor Yellow
                        $script:observationsDataLoading = $true
                        # Use script-scoped variables if available, otherwise use current scope variables
                        $pointsDataToUse = if ($null -ne $script:pointsDataForObservations) { $script:pointsDataForObservations } else { $pointsData }
                        $headersToUse = if ($null -ne $script:headersForObservations) { $script:headersForObservations } else { $headers }
                        $timeZoneToUse = if ($null -ne $script:timeZoneForObservations) { $script:timeZoneForObservations } else { $timeZone }
                        try {
                            $script:observationsData = Get-NWSObservations -PointsData $pointsDataToUse -Headers $headersToUse -TimeZone $timeZoneToUse
                        } catch {
                            Write-Verbose "Error fetching observations: $($_.Exception.Message)"
                            $script:observationsData = $null
                        } finally {
                            $script:observationsDataLoading = $false
                        }
                        Clear-HostWithDelay
                    }
                    if ($null -ne $script:observationsData -and ($script:observationsData -isnot [Array] -or $script:observationsData.Count -gt 0)) {
                        Show-Observations -ObservationsData $script:observationsData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -City $city -ShowCityInTitle $true -TimeZone $timeZone -Latitude $lat -Longitude $lon
                    } else {
                        Write-Host "No historical observations available." -ForegroundColor $defaultColor
                    }
                    Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                }
                'g' { # G key - Refresh weather data
                    $refreshSuccess = Update-WeatherData -Lat $lat -Lon $lon -Headers $headers -TimeZone $timeZone -UseRetryLogic $true
                    if ($refreshSuccess) {
                        # Recalculate moon phase and sunrise/sunset for current time (using location's date)
                        $moonPhaseInfo = Get-MoonPhase -Date (Get-Date)
                        $locationToday = if ($timeZone) { 
                            $tzInfo = Get-ResolvedTimeZoneInfo -TimeZoneId $timeZone
                            [System.TimeZoneInfo]::ConvertTime((Get-Date), $tzInfo).Date 
                        } else { 
                            (Get-Date).Date 
                        }
                        $sunTimes = Get-SunriseSunset -Latitude $lat -Longitude $lon -Date $locationToday -TimeZoneId $timeZone
                        $sunriseTime = $sunTimes.Sunrise
                        $sunsetTime = $sunTimes.Sunset
                        # Sync script-scoped display vars to local so re-render uses fresh data (including currentTempTrend)
                        $currentTemp = $script:currentTemp
                        $currentConditions = $script:currentConditions
                        $currentTempTrend = $script:currentTempTrend
                        $currentWind = $script:currentWind
                        $currentWindDir = $script:currentWindDir
                        $currentHumidity = $script:currentHumidity
                        $currentPrecipProb = $script:currentPrecipProb
                        $todayForecast = $script:todayForecast
                        $todayPeriodName = $script:todayPeriodName
                        $tomorrowForecast = $script:tomorrowForecast
                        $tomorrowPeriodName = $script:tomorrowPeriodName
                        
                        # Re-render current view with fresh data
                        Clear-HostWithDelay
                        if ($isHourlyMode) {
                            Show-HourlyForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true -SunriseTime $script:sunriseTime -SunsetTime $script:sunsetTime -City $city -ShowCityInTitle $true -TimeZone $timeZone
                            Show-InteractiveControls -IsHourlyMode $true -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $false
                        } elseif ($isRainMode) {
                            Show-RainForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city -TimeZone $timeZone
                            Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                        } elseif ($isWindMode) {
                            Show-WindForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -City $city -TimeZone $timeZone
                            Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                        } elseif ($isTerseMode) {
                            # Format sunrise: date/time in 24-hour format if polar night/day, otherwise time in 12-hour format
                            # During polar night: shows next sunrise; during polar day: shows last sunrise
                            $sunriseTimeStr = if ($null -ne $script:sunriseTime) { 
                                if ($script:isPolarNight -or $script:isPolarDay) { 
                                    $script:sunriseTime.ToString('MM/dd HH:mm') 
                                } else { 
                                    $script:sunriseTime.ToString('h:mm tt') 
                                }
                            } else { 
                                "N/A" 
                            }
                            # Format sunset: date/time in 24-hour format if polar night/day, otherwise time in 12-hour format
                            # During polar night: shows last sunset; during polar day: shows next sunset
                            $sunsetTimeStr = if ($null -ne $script:sunsetTime) { 
                                if ($script:isPolarNight -or $script:isPolarDay) { 
                                    $script:sunsetTime.ToString('MM/dd HH:mm') 
                                } else { 
                                    $script:sunsetTime.ToString('h:mm tt') 
                                }
                            } else { 
                                "N/A" 
                            }
                            Show-CurrentConditions -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $script:dataFetchTime -SunriseTime $sunriseTimeStr -SunsetTime $sunsetTimeStr -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor -MoonPhase $moonPhaseInfo.Name -MoonEmoji $moonPhaseInfo.Emoji -IsFullMoon $moonPhaseInfo.IsFullMoon -NextFullMoonDate $moonPhaseInfo.NextFullMoon -IsNewMoon $moonPhaseInfo.IsNewMoon -ShowNextFullMoon $moonPhaseInfo.ShowNextFullMoon -ShowNextNewMoon $moonPhaseInfo.ShowNextNewMoon -NextNewMoonDate $moonPhaseInfo.NextNewMoon
                            Show-ForecastText -Title $todayPeriodName -ForecastText $todayForecast -TitleColor $titleColor -DefaultColor $defaultColor
                            Show-WeatherAlerts -AlertsData $script:alertsData -AlertColor $alertColor -DefaultColor $defaultColor -InfoColor $infoColor -ShowDetails $false -TimeZone $timeZone
                            Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                        } elseif ($isDailyMode) {
                            Show-SevenDayForecast -ForecastData $script:forecastData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -SunriseTime $script:sunriseTime -SunsetTime $script:sunsetTime -IsEnhancedMode $true -City $city -ShowCityInTitle $true -Latitude $lat -Longitude $lon -TimeZone $timeZone
                            Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                        } elseif ($isObservationsMode) {
                            # Preserve current mode - show Observations mode if in Observations mode
                            if ($null -ne $script:observationsData) {
                                Show-Observations -ObservationsData $script:observationsData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -City $city -ShowCityInTitle $true -TimeZone $timeZone -Latitude $lat -Longitude $lon
                            } else {
                                Write-Host "No historical observations available." -ForegroundColor $defaultColor
                            }
                            Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                        } else {
                            Show-FullWeatherReport -City $city -State $state -WeatherIcon $weatherIcon -CurrentConditions $currentConditions -CurrentTemp $currentTemp -TempColor $tempColor -CurrentTempTrend $currentTempTrend -CurrentWind $currentWind -WindColor $windColor -CurrentWindDir $currentWindDir -WindGust $windGust -CurrentHumidity $currentHumidity -CurrentDewPoint $currentDewPoint -CurrentPrecipProb $currentPrecipProb -CurrentTimeLocal $script:dataFetchTime -TodayForecast $todayForecast -TodayPeriodName $todayPeriodName -TomorrowForecast $tomorrowForecast -TomorrowPeriodName $tomorrowPeriodName -HourlyData $script:hourlyData -ForecastData $script:forecastData -AlertsData $script:alertsData -TimeZone $timeZone -Lat $lat -Lon $lon -ElevationFeet $elevationFeet -RadarStation $radarStation -DefaultColor $defaultColor -AlertColor $alertColor -TitleColor $titleColor -InfoColor $infoColor -ShowCurrentConditions $showCurrentConditions -ShowTodayForecast $showTodayForecast -ShowTomorrowForecast $showTomorrowForecast -ShowHourlyForecast $showHourlyForecast -ShowSevenDayForecast $showSevenDayForecast -ShowAlerts $showAlerts -ShowAlertDetails $showAlertDetails -ShowLocationInfo $showLocationInfo -MoonPhase $moonPhaseInfo.Name -MoonEmoji $moonPhaseInfo.Emoji -IsFullMoon $moonPhaseInfo.IsFullMoon -NextFullMoonDate $moonPhaseInfo.NextFullMoon -IsNewMoon $moonPhaseInfo.IsNewMoon -ShowNextFullMoon $moonPhaseInfo.ShowNextFullMoon -ShowNextNewMoon $moonPhaseInfo.ShowNextNewMoon -NextNewMoonDate $moonPhaseInfo.NextNewMoon -SunriseTime $sunriseTime -SunsetTime $sunsetTime
                            Show-InteractiveControls -IsHourlyMode $isHourlyMode -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $(-not $isHourlyMode -and -not $isRainMode -and -not $isWindMode -and -not $isTerseMode -and -not $isDailyMode -and -not $isObservationsMode)
                        }
                    }
                }
                { $keyInfo.Key -eq 'UpArrow' } { # Up arrow - Scroll up in hourly mode
                    if ($isHourlyMode) {
                        $newIndex = $hourlyScrollIndex - $script:MAX_HOURLY_FORECAST_HOURS
                        if ($newIndex -ge 0) {
                            $hourlyScrollIndex = $newIndex
                            Clear-HostWithDelay
                            Show-HourlyForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true -SunriseTime $script:sunriseTime -SunsetTime $script:sunsetTime -City $city -ShowCityInTitle $true -TimeZone $timeZone
                            Show-InteractiveControls -IsHourlyMode $true -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $false
                        }
                    }
                }
                { $keyInfo.Key -eq 'DownArrow' } { # Down arrow - Scroll down in hourly mode
                    if ($isHourlyMode) {
                        $newIndex = $hourlyScrollIndex + $script:MAX_HOURLY_FORECAST_HOURS
                        if ($newIndex -lt $totalHourlyPeriods) {
                            $hourlyScrollIndex = $newIndex
                            Clear-HostWithDelay
                            Show-HourlyForecast -HourlyData $script:hourlyData -TitleColor $titleColor -DefaultColor $defaultColor -AlertColor $alertColor -StartIndex $hourlyScrollIndex -IsInteractive $true -SunriseTime $script:sunriseTime -SunsetTime $script:sunsetTime -City $city -ShowCityInTitle $true -TimeZone $timeZone
                            Show-InteractiveControls -IsHourlyMode $true -IsRainMode $isRainMode -IsWindMode $isWindMode -IsTerseMode $isTerseMode -IsDailyMode $isDailyMode -IsObservationsMode $isObservationsMode -IsFullMode $false
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
            } catch {
                Write-Host "Key detection error: $($_.Exception.Message)" -ForegroundColor Red
                Write-Host "Interactive mode not supported in this environment. Exiting..." -ForegroundColor Yellow
                return
            }
        }
        catch {
            Write-Host "Interactive mode error: $($_.Exception.Message)" -ForegroundColor Red
            Write-Host "Interactive mode not supported in this environment. Exiting..." -ForegroundColor Yellow
            return
        }
    }
}
