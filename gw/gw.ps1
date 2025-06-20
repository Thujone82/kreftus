<#
.SYNOPSIS
    Retrieves and displays weather information using OpenWeatherMap One Call API 3.0.
    
.DESCRIPTION
    This script accepts a zipcode (5-digit or 5-digit+4) or a "City, State" string.
    For zip codes it calls the geocoding endpoint at:
       http://api.openweathermap.org/geo/1.0/zip
    For city names it calls:
       http://api.openweathermap.org/geo/1.0/direct
    Once geographic coordinates are obtained, it calls the One Call API 3.0 endpoint:
       https://api.openweathermap.org/data/3.0/onecall
    The returned data includes current weather, daily forecasts (for today's min/max),
    wind (including gust), UV index, sunrise/sunset, moonrise/moonset, observation 
    time, weather report and Geo link to verify report location.
    
    Output text is colored as follows:
      - Temperature in red if below 33°F or above 89°F.
      - Wind in red if wind speed is >=16 mph.
      - UV index in red if it is >=6.
    
.PARAMETER Location
    The zipcode or "City, State" for which to retrieve the weather information.
    
.PARAMETER Help
    Displays usage information for this script.
    
.EXAMPLE
    .\gw.ps1 97219 -Verbose
    Retrieves weather for zipcode 97219.
    
    
.EXAMPLE
    .\gw.ps1 "Portland, OR" -Verbose
    Retrieves weather for Portland, OR.
    
.NOTES
    API Key: OpenWeatherMap 3.0 api key must be configured below.  Reg requires CC but can
    be configured in a way that the service is free by capping calls to match the 1000 limit.

    To execute PS Scripts run the following from an admin prompt "Set-ExecutionPolicy bypass"
    Execute with ./<scriptname>, or simply <scriptname> if placed in your %PATH%
#>

param(
    [Parameter(Mandatory = $false, Position = 0)]
    [string]$Location,

    [Parameter(Mandatory = $false)]
    [switch]$Help
)


if ($Help) {
    Write-Host "Usage: .\gw.ps1 [ZipCode | `"City, State`"] [-Verbose]" -ForegroundColor Green
    Write-Host "  Provide a 5-digit zipcode or a City, State (e.g., 'Portland, OR')." -ForegroundColor Cyan
    Write-Host ""
    Write-Host "This script retrieves weather info from OpenWeatherMap One Call API 3.0 and outputs:" -ForegroundColor Blue
    Write-Host "  Location (City, Country)" -ForegroundColor Cyan
    Write-Host "  Overview" -ForegroundColor Cyan
    Write-Host "  Conditions" -ForegroundColor Cyan
    Write-Host "  Temperature with forecast range (red if <33°F or >89°F)" -ForegroundColor Cyan
    Write-Host "  Humidity" -ForegroundColor Cyan
    Write-Host "  Wind (with gust if available; red if wind speed >=16 mph)" -ForegroundColor Cyan
    Write-Host "  UV Index (red if >=6)" -ForegroundColor Cyan
    Write-Host "  Sunrise and Sunset times" -ForegroundColor Cyan
    Write-Host "  Moonrise and Moonset times" -ForegroundColor Cyan
    Write-Host "  Weather Report" -ForegroundColor Cyan
    Write-Host "  Observation timestamp" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Blue
    Write-Host "  .\gw.ps1 97219 -Verbose" -ForegroundColor Cyan
    Write-Host "  .\gw.ps1 `"Portland, OR`" -Verbose" -ForegroundColor Cyan
    Write-Host "  .\gw.ps1 -Help" -ForegroundColor Cyan
    return
}

# Force TLS 1.2.
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

# Function to get the parent process name using CIM
function Get-ParentProcessName {
    $proc = Get-CimInstance Win32_Process -Filter "ProcessId=$PID"
    if ($proc) {
        $parentProc = Get-CimInstance Win32_Process -Filter "ProcessId=$($proc.ParentProcessId)"
        return $parentProc.Name
    }
    return $null
}

# Define common header.
$headers = @{
    "Accept"     = "application/json"
    "User-Agent" = "curl/7.64.1"
}

# Determine the parent's process name.
$parentName = Get-ParentProcessName
Write-Verbose "Proc:$parentName"

if (-not $Location) {
    if ($VerbosePreference -ne 'Continue') {
       Clear-Host
    }
    Write-Host '    \|/     ' -ForegroundColor Yellow -NoNewline; Write-Host "    .-~~~~~~-.      " -ForegroundColor Cyan
    Write-Host '  -- O --   ' -ForegroundColor Yellow -NoNewline; Write-Host "   /_)      ( \     " -ForegroundColor Cyan
    Write-Host '    /|\     ' -ForegroundColor Yellow -NoNewline; Write-Host "  (   ( )    ( )    " -ForegroundColor Cyan
    Write-Host '            ' -ForegroundColor Yellow -NoNewline; Write-Host "   `-~~~~~~~~~-`    " -ForegroundColor Cyan
    Write-Host '  Welcome   ' -ForegroundColor Green  -NoNewline; Write-Host "     ''    ''       " -ForegroundColor Cyan
    Write-Host '     to     ' -ForegroundColor Green  -NoNewline; Write-Host "    ''    ''        " -ForegroundColor Cyan
    Write-Host '  KWeather  ' -ForegroundColor Green  -NoNewline; Write-Host "  ________________  " -ForegroundColor Cyan
    Write-Host '            ' -ForegroundColor Yellow -NoNewline; Write-Host "~~~~~~~~~~~~~~~~~~~~" -ForegroundColor Cyan
    Write-Host ""
    $Location = Read-Host "Enter a location (Zip Code or City, State)"
}

Write-Verbose "Input provided: $Location"

$apiKey = "118479e789f099418d17a8299fe267de"

# --- GEOCODING ---
if ($Location -match "^\d{5}(-\d{4})?$") {
    Write-Verbose "Input identified as a zipcode."
    $geoUrl = "http://api.openweathermap.org/geo/1.0/zip?zip=$Location,us&appid=$apiKey"
    Write-Verbose "Geocoding URL (zip): $geoUrl"
    try {
        $geoData = Invoke-RestMethod "$GeoUrl" -ErrorAction Stop
    }
    catch {
        Write-Error "Failed to retrieve geocoding data for zipcode '$Location'."
        exit
    }
    $lat = $geoData.lat
    $lon = $geoData.lon
    $city = $geoData.name
    $country = $geoData.country
}
else {
    Write-Verbose "Input assumed to be a City, State."
    #if ($Location -notmatch ",") {
        $Location = "$Location,us"
    #}
    $encodedLocation = [uri]::EscapeDataString($Location)
    $geoUrl = "http://api.openweathermap.org/geo/1.0/direct?q=$encodedLocation&limit=1&appid=$apiKey"
    Write-Verbose "Geocoding URL (direct): $geoUrl"
    try {
        $geoData = Invoke-RestMethod "$GeoUrl" -ErrorAction Stop
        if ($geoData.Count -eq 0) {
            Write-Error "No geocoding results found for '$Location'."
            exit
        }
    }
    catch {
        Write-Error "Failed to retrieve geocoding data for '$Location'."
        exit
    }
    $lat = $geoData.lat
    $lon = $geoData.lon
    $city = $geoData.name
    $country = $geoData.country
}

Write-Verbose "Geocoding result: City: $city, Country: $country, Lat: $lat, Lon: $lon"

# --- CALL ONE CALL API 3.0 ---
$weatherUrl = "https://api.openweathermap.org/data/3.0/onecall?lat=$lat&lon=$lon&appid=$apiKey&units=imperial&lang=en&exclude=minutely"
Write-Verbose "Weather API URL: $weatherUrl"
try {
    $weatherResponse = Invoke-WebRequest -Uri $weatherUrl -Method Get -Headers $headers
    $weatherJson = $weatherResponse.Content
    if ([string]::IsNullOrWhiteSpace($weatherJson)) {
        throw "Empty response from weather API."
    }
    Write-Verbose "Raw weather response:`n$weatherJson"
    Write-Verbose ""
    $weatherData = $weatherJson | ConvertFrom-Json
}
catch {
    Write-Error "Failed to retrieve weather data."
    exit
}

# Ensure current data exists.
if (-not $weatherData.current) {
    Write-Error "Unexpected JSON structure: 'current' data not found."
    exit
}
$overviewUrl = "https://api.openweathermap.org/data/3.0/onecall/overview?lat=$lat&lon=$lon&appid=$apiKey&units=imperial&lang=en"
Write-Verbose "Overview API URL: $overviewUrl"
try {
    $overviewResponse = Invoke-WebRequest -Uri $overviewUrl -Method Get -Headers $headers
    $overviewJson = $overviewResponse.Content
    if ([string]::IsNullOrWhiteSpace($overviewJson)) {
        throw "Empty response from overview API."
    }
    Write-Verbose "Raw overview response:`n$overviewJson"
    Write-Verbose ""
    $overviewData = $overviewJson | ConvertFrom-Json
}
catch {
    Write-Error "Failed to retrieve overview data."
    exit
}

# Ensure overview data exists.
if (-not $overviewData.weather_overview) {
    Write-Error "Unexpected JSON structure: 'overview' data not found."
    exit
}

function Wrap-Text {
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

# Build variables.
$current = $weatherData.current
$daily = $weatherData.daily[0]
$tomorrow = $weatherData.daily[1]
$nextday = $tomorrow.summary
if ($weatherData.alerts -ne $null) {
    $alert = $weatherData.alerts[0]
}
$temperature = $current.temp
$windSpeed = $current.wind_speed
$windDeg = $current.wind_deg
$windGust = $current.wind_gust
$humidity = $current.humidity
$sunrise = $current.sunrise
$sunset = $current.sunset
$moonrise = $daily.moonrise
$moonset = $daily.moonset
$uv = $current.uvi
$conditions = $current.weather[0].main
if ($current.rain -ne $null) {
    $rain = $current.rain
    $conditions = "[$rain.1h mm/H]"
}
if ($current.snow -ne $null) {
    $snow = $current.snow
    $conditions = "[$snow.1h mm/H]"
}
$report = $overviewData.weather_overview
$wrappedReport = Wrap-Text -Text $report -Width $Host.UI.RawUI.WindowSize.Width
if ($current.dt) {
    $observedTime = ([System.DateTimeOffset]::FromUnixTimeSeconds($current.dt)).LocalDateTime
}
else {
    $observedTime = "N/A"
}
if ($sunrise) {
$sunriseTime = ([System.DateTimeOffset]::FromUnixTimeSeconds($sunrise)).LocalDateTime
}
else {
    $sunriseTime = "N/A"
}
if ($sunset) {
$sunsetTime = ([System.DateTimeOffset]::FromUnixTimeSeconds($sunset)).LocalDateTime
}
else {
    $sunsetTime = "N/A"
}
if ($moonrise) {
$moonriseTime = ([System.DateTimeOffset]::FromUnixTimeSeconds($moonrise)).LocalDateTime
}
else {
    $moonriseTime = "N/A"
}
if ($moonset) {
$moonsetTime = ([System.DateTimeOffset]::FromUnixTimeSeconds($moonset)).LocalDateTime
}
else {
    $moonsetTime = "N/A"
}

# Use the first daily forecast for min and max temperatures.
if ($weatherData.daily -and $weatherData.daily.Count -gt 0) {
    $daily    = $weatherData.daily[0]
    $minTemp  = $daily.temp.min
    $maxTemp  = $daily.temp.max
    $forecast = $daily.summary

    # Convert the moon_phase (a value between 0 and 1) into a text description.
    $moonPhaseValue = $daily.moon_phase
    if ($moonPhaseValue -lt 0.0625 -or $moonPhaseValue -ge 0.9375) {
        $moonPhaseDescription = "New Moon"
    }
    elseif ($moonPhaseValue -lt 0.1875) {
        $moonPhaseDescription = "Waxing Crescent"
    }
    elseif ($moonPhaseValue -lt 0.3125) {
        $moonPhaseDescription = "First Quarter"
    }
    elseif ($moonPhaseValue -lt 0.4375) {
        $moonPhaseDescription = "Waxing Gibbous"
    }
    elseif ($moonPhaseValue -lt 0.5625) {
        $moonPhaseDescription = "Full Moon"
    }
    elseif ($moonPhaseValue -lt 0.6875) {
        $moonPhaseDescription = "Waning Gibbous"
    }
    elseif ($moonPhaseValue -lt 0.8125) {
        $moonPhaseDescription = "Third Quarter"
    }
    else {
        $moonPhaseDescription = "Waning Crescent"
    }
}
else {
    $minTemp = "N/A"
    $maxTemp = "N/A"
    $moonPhaseDescription = "N/A"
    $forecast = "N/A"
}

# --- Temperature indicator from hourly forecast ---
if ($weatherData.hourly -and $weatherData.hourly.Count -gt 1) {
    $nextHour = $weatherData.hourly[2]
    $nextTemp = $nextHour.temp
    $tempDiff = [double]$nextTemp - [double]$temperature
    Write-Verbose "Temp delta:$tempDiff"
    $compTime = ([System.DateTimeOffset]::FromUnixTimeSeconds($nextHour.dt)).LocalDateTime
    Write-Verbose "Compare Time:$compTime"
    if ($tempDiff -ge .67) {
        $tempIndicator = "(Rising)"
    }
    elseif ($tempDiff -le -.67) {
        $tempIndicator = "(Falling)"
    }
    else {
        $tempIndicator = ""
    }
}
else {
    $tempIndicator = ""
}

# Function: Convert wind degrees to cardinal direction.
function Get-CardinalDirection ($deg) {
    $val = [math]::Floor(($deg / 22.5) + 0.5)
    $directions = @("N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW")
    return $directions[($val % 16)]
}
$windDir = Get-CardinalDirection $windDeg

# Prepare output strings.

if ($windGust -and $windGust -ne $null) {
    $windLabel = "Wind[Gust]:"
    $displayWind = "$windSpeed mph [$windGust mph] $windDir"
} else {
    $windLabel = "Wind:"
    $displayWind = "$windSpeed mph $windDir"
}

# Define colors.
$defaultColor = "DarkCyan"
$alertColor = "Red"
$titleColor = "Green"
$infoColor = "Blue"
$sunColor = "DarkYellow"
$moonColor = "DarkGray"

if ([int]$temperature -lt 33 -or [int]$temperature -gt 89) {
    $tempColor = $alertColor
} else {
    $tempColor = $defaultColor
}

if ([int]$windSpeed -ge 16) {
    $windColor = $alertColor
} else {
    $windColor = $defaultColor
}

if ([double]$uv -ge 6) {
    $uvColor = $alertColor
} else {
    $uvColor = $defaultColor
}

if ($VerbosePreference -ne 'Continue') {
    Clear
}

# Output the results.
Write-Host "*** $city Current Conditions ***" -ForegroundColor $titleColor
Write-Host "Forecast: $forecast" -ForegroundColor $infoColor
Write-Host "Currently: $conditions" -ForegroundColor $defaultColor
Write-Host "Temp [L/H]: $temperature$tempIndicator [$minTemp/$maxTemp]" -ForegroundColor $tempColor
Write-Host "Humidity: $humidity" -ForegroundColor $defaultColor
Write-Host "UV Index: $uv" -ForegroundColor $uvColor
Write-Host "$windLabel $displayWind" -ForegroundColor $windColor
Write-Host "Tomorrow: $nextday" -ForegroundColor Cyan
Write-Host "Sunrise: $sunriseTime" -ForegroundColor $sunColor
Write-Host "Sunset: $sunsetTime" -ForegroundColor $sunColor
Write-Host "Moonrise: $MoonriseTime" -ForegroundColor $MoonColor
Write-Host "Moonset: $MoonsetTime" -ForegroundColor $MoonColor
Write-Host "Moon Phase: $moonPhaseDescription" -ForegroundColor $MoonColor
Write-Host "Observed: $observedTime" -ForegroundColor $infoColor
Write-Host ""
Write-Host "*** $city Weather Report ***" -ForegroundColor $titleColor
$wrappedReport | ForEach-Object { Write-Host $_ -ForegroundColor $defaultColor }
Write-Host ""
Write-Host "https://forecast.weather.gov/MapClick.php?lat=$lat&lon=$lon" -ForegroundColor Cyan
# Alert Handling
if ($alert -ne $null) {
    $sender = $alert.sender_name
    $event = $alert.event
    $alertStart = ([System.DateTimeOffset]::FromUnixTimeSeconds($alert.start)).LocalDateTime
    $alertEnd = ([System.DateTimeOffset]::FromUnixTimeSeconds($alert.end)).LocalDateTime
    $alertDesc = $alert.description
    Write-Host ""
    Write-Host "*** $event - $sender ***" -ForegroundColor $alertColor
    Write-Host "$alertDesc" -ForegroundColor $defaultColor
    Write-Host "Starts: $alertStart" -ForegroundColor $infoColor
    Write-Host "Ends: $alertEnd" -ForegroundColor $infoColor
}
# If the script wasn't started from a typical command shell (powershell or cmd),
# prompt the user to hit Enter before exiting.
if ($parentName -notmatch '^(WindowsTerminal.exe|PowerShell|cmd)') {
    Write-Verbose "Parent:$parentName"
    Read-Host -Prompt "Hit Enter to Exit"
}