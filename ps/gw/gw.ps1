<#
.SYNOPSIS
    A PowerShell script to retrieve and display detailed weather information for a specified location.
    
.DESCRIPTION
    This script accepts a US zip code or a "City, State" string as input. It uses the
    OpenWeatherMap One Call API 3.0 to retrieve and display detailed weather information.

    The script first uses a geocoding API to determine the latitude and longitude of the location,
    then fetches the current weather, daily forecasts, and a descriptive weather report.

    On the first run, the script will prompt the user for their OpenWeatherMap API key
    and save it to a configuration file for future use.
    
.PARAMETER Location
    The location for which to retrieve weather. Can be a 5-digit US zip code or a "City, State" string.
    If omitted, the script will prompt you for it.
    
.PARAMETER Help
    Displays usage information for this script.
    
.EXAMPLE
    .\gw.ps1 97219 -Verbose

.EXAMPLE
    .\gw.ps1 "Portland, OR" -Verbose

.NOTES
    On the first run, the script will prompt for a free OpenWeatherMap One Call API 3.0 key.
    This key is validated and stored in a `gw.ini` file in the user's configuration directory
    (e.g., %APPDATA%\gw on Windows).

    TIP: Set maximum calls per day to 1000 to prevent charges.

    To execute PS Scripts run the following from an admin prompt "Set-ExecutionPolicy bypass"
    Execute with ./<scriptname>, or simply <scriptname> if placed in your %PATH%
#>

param(
    [Parameter(Mandatory = $false, Position = 0)]
    [string]$Location,

    [Parameter(Mandatory = $false)]
    [switch]$Help,

    [Alias('t')]
    [switch]$Terse
)

# --- Helper Functions ---

function Get-GwConfigPath {
    $appName = "gw"
    $configFileName = "gw.ini"
    $configDir = ""

    if ($env:APPDATA) {
        $configDir = Join-Path -Path $env:APPDATA -ChildPath $appName
    }
    elseif ($env:HOME) {
        # This covers Linux and macOS
        $configDir = Join-Path -Path $env:HOME -ChildPath ".config/$appName"
    }
    else {
        # Fallback for environments without standard config paths
        $configDir = Join-Path -Path $PSScriptRoot -ChildPath ".config"
    }

    if (-not (Test-Path $configDir)) {
        try {
            New-Item -Path $configDir -ItemType Directory -Force -ErrorAction Stop | Out-Null
        }
        catch {
            Write-Error "Failed to create configuration directory at '$configDir'. Please check permissions."
            return $null
        }
    }
    return Join-Path -Path $configDir -ChildPath $configFileName
}

function Get-GwConfiguration {
    param ([string]$FilePath)
    Write-Verbose "Reading INI file from $FilePath"
    # Define the default structure
    $defaultConfig = @{ "openweathermap" = @{ "apikey" = "" } }
    if (-not (Test-Path $FilePath)) {
        return $defaultConfig
    }

    $ini = @{}
    $fileContent = Get-Content $FilePath -ErrorAction SilentlyContinue
    $currentSection = ""
    foreach ($line in $fileContent) {
        $trimmedLine = $line.Trim()
        if ($trimmedLine -match "^\[(.+)\]$") {
            $currentSection = $matches[1].Trim()
            if (-not $ini.ContainsKey($currentSection)) { $ini[$currentSection] = @{} }
        }
        elseif ($trimmedLine -match "^([^#;].*?)=(.*)$" -and $currentSection) {
            $key = $matches[1].Trim()
            $value = $matches[2].Trim()
            $ini[$currentSection][$key] = $value
        }
    }
    # Ensure the structure matches the default to prevent errors
    if (-not $ini.openweathermap) { $ini.openweathermap = @{} }
    if (-not $ini.openweathermap.apikey) { $ini.openweathermap.apikey = "" }
    return $ini
}

function Set-GwConfiguration {
    param ([string]$FilePath, [hashtable]$Configuration)
    $iniContent = @()
    foreach ($sectionKey in $Configuration.Keys | Sort-Object) {
        $iniContent += "[$sectionKey]"
        $section = $Configuration[$sectionKey]
        foreach ($key in $section.Keys | Sort-Object) {
            $iniContent += "$key=$($section[$key])"
        }
        $iniContent += ""
    }
    try {
        Set-Content -Path $FilePath -Value $iniContent -ErrorAction Stop
        Write-Verbose "Configuration saved to: $FilePath"
        return $true
    }
    catch {
        Write-Error "Failed to save configuration to $FilePath. Error: $($_.Exception.Message)"
        return $false
    }
}

function Test-ApiKey {
    param ([string]$ApiKey)
    if ([string]::IsNullOrEmpty($ApiKey)) { return $false }
    $testUrl = "http://api.openweathermap.org/geo/1.0/zip?zip=90210,us&appid=$ApiKey"
    try {
        Invoke-RestMethod -Uri $testUrl -Method Get -TimeoutSec 10 -ErrorAction Stop | Out-Null
        return $true
    }
    catch {
        return $false
    }
}

function Show-FirstRunSetup {
    param ([ref]$Config, [string]$ConfigPath)
    Clear-Host
    Write-Host "*** First Time Setup ***" -ForegroundColor Yellow
    Write-Host "Get Free One Call API 3.0 Key: https://openweathermap.org/api" -ForegroundColor Green
    
    while ($true) {
        $apiKeyInput = Read-Host "Please enter your API Key"
        if (Test-ApiKey -ApiKey $apiKeyInput) {
            $Config.Value.openweathermap.apikey = $apiKeyInput
            Set-GwConfiguration -FilePath $ConfigPath -Configuration $Config.Value
            Write-Host "API Key is valid and has been saved to `"$ConfigPath`"." -ForegroundColor Green
            Read-Host "Press Enter to continue."
            break
        } else {
            Write-Host "Invalid API Key. Please try again." -ForegroundColor Red
        }
    }
    return $Config.Value
}

function Initialize-GwConfiguration {
    $configPath = Get-GwConfigPath
    if (-not $configPath) {
        # Error already shown in Get-GwConfigPath
        exit 1
    }
 
    $config = Get-GwConfiguration -FilePath $configPath
    $apiKey = $config.openweathermap.apikey
 
    if ([string]::IsNullOrEmpty($apiKey) -or -not (Test-ApiKey -ApiKey $apiKey)) {
        if (-not [string]::IsNullOrEmpty($apiKey)) {
            Write-Host "Your previously saved API key is no longer valid." -ForegroundColor Yellow
        }
        $config = Show-FirstRunSetup -Config ([ref]$config) -ConfigPath $configPath
        $apiKey = $config.openweathermap.apikey
    }
    
    return $apiKey
}


if ($Help -or ($Terse.IsPresent -and -not $Location)) {
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

$apiKey = Initialize-GwConfiguration

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

$isInteractive = (-not $Location)
$lat = $null
$lon = $null
$city = $null
$country = $null

while ($true) { # Loop for location input and geocoding
    try {
        if (-not $Location) {
            if ($VerbosePreference -ne 'Continue') { Clear-Host }
            Write-Host '    \|/     ' -ForegroundColor Yellow -NoNewline; Write-Host "    .-~~~~~~-.      " -ForegroundColor Cyan
            Write-Host '  -- O --   ' -ForegroundColor Yellow -NoNewline; Write-Host "   /_)      ( \     " -ForegroundColor Cyan
            Write-Host '    /|\     ' -ForegroundColor Yellow -NoNewline; Write-Host "  (   ( )    ( )    " -ForegroundColor Cyan
            Write-Host '            ' -ForegroundColor Yellow -NoNewline; Write-Host "   `-~~~~~~~~~-`    " -ForegroundColor Cyan
            Write-Host '  Welcome   ' -ForegroundColor Green  -NoNewline; Write-Host "     ''    ''       " -ForegroundColor Cyan # Matched to old style
            Write-Host '     to     ' -ForegroundColor Green  -NoNewline; Write-Host "    ''    ''        " -ForegroundColor Cyan # Matched to old style
            Write-Host ' GetWeather ' -ForegroundColor Green  -NoNewline; Write-Host "  ________________  " -ForegroundColor Cyan # Matched to old style
            Write-Host '            ' -ForegroundColor Yellow -NoNewline; Write-Host "~~~~~~~~~~~~~~~~~~~~" -ForegroundColor Cyan
            Write-Host ""
            $Location = Read-Host "Enter a location (Zip Code or City, State)"
            if ([string]::IsNullOrEmpty($Location)) { exit } # Exit if user enters nothing
        }

        Write-Verbose "Input provided: $Location"

        # --- GEOCODING ---
        if ($Location -match "^\d{5}(-\d{4})?$") {
            Write-Verbose "Input identified as a zipcode."
            $geoUrl = "http://api.openweathermap.org/geo/1.0/zip?zip=$Location,us&appid=$apiKey"
            Write-Verbose "Geocoding URL (zip): $geoUrl"
            $geoData = Invoke-RestMethod "$GeoUrl" -ErrorAction Stop
            if (-not $geoData) { throw "No geocoding results found for zipcode '$Location'." }
            $lat = $geoData.lat
            $lon = $geoData.lon
            $city = $geoData.name
            $country = $geoData.country
        }
        else {
            Write-Verbose "Input assumed to be a City, State."
            $locationForApi = if ($Location -match ",") { $Location } else { "$Location,us" }
            $encodedLocation = [uri]::EscapeDataString($locationForApi)
            $geoUrl = "http://api.openweathermap.org/geo/1.0/direct?q=$encodedLocation&limit=1&appid=$apiKey"
            Write-Verbose "Geocoding URL (direct): $geoUrl"
            $geoData = Invoke-RestMethod "$GeoUrl" -ErrorAction Stop
            if ($geoData.Count -eq 0) { throw "No geocoding results found for '$Location'." }
            $lat = $geoData[0].lat
            $lon = $geoData[0].lon
            $city = $geoData[0].name
            $country = $geoData[0].country
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

Write-Verbose "Geocoding result: City: $city, Country: $country, Lat: $lat, Lon: $lon"

# --- CONCURRENTLY FETCH WEATHER AND OVERVIEW DATA ---
Write-Verbose "Starting API call for weather data."

$weatherUrl = "https://api.openweathermap.org/data/3.0/onecall?lat=$lat&lon=$lon&appid=$apiKey&units=imperial&lang=en&exclude=minutely"

$weatherJob = Start-Job -ScriptBlock {
    param($url, $hdrs)
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    try {
        $response = Invoke-WebRequest -Uri $url -Method Get -Headers $hdrs -ErrorAction Stop
        return $response.Content
    }
    catch {
        throw "Failed to retrieve weather data from job: $($_.Exception.Message)" # Throw will be caught by Receive-Job
    }
} -ArgumentList $weatherUrl, $headers

$overviewJob = $null
if (-not $Terse.IsPresent) {
    Write-Verbose "Terse mode is off, starting API call for overview data."
    $overviewUrl = "https://api.openweathermap.org/data/3.0/onecall/overview?lat=$lat&lon=$lon&appid=$apiKey&units=imperial&lang=en"
    $overviewJob = Start-Job -ScriptBlock {
        param($url, $hdrs)
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        try {
            $response = Invoke-WebRequest -Uri $url -Method Get -Headers $hdrs -ErrorAction Stop
            return $response.Content
        }
        catch {
            throw "Failed to retrieve overview data from job: $($_.Exception.Message)"
        }
    } -ArgumentList $overviewUrl, $headers
}

$jobsToWaitFor = @($weatherJob)
if ($null -ne $overviewJob) { $jobsToWaitFor += $overviewJob }
Wait-Job -Job $jobsToWaitFor | Out-Null

# --- COLLECT RESULTS AND HANDLE ERRORS ---
if ($weatherJob.State -ne 'Completed') { Write-Error "The weather data job failed: $($weatherJob | Receive-Job)"; Remove-Job -Job $jobsToWaitFor; exit 1 }
$weatherJson = $weatherJob | Receive-Job
if ([string]::IsNullOrWhiteSpace($weatherJson)) { Write-Error "Empty response from weather API job."; Remove-Job -Job $jobsToWaitFor; exit 1 }
$weatherData = $weatherJson | ConvertFrom-Json
Write-Verbose "Raw weather response:`n$weatherJson"

$overviewData = $null
if ($null -ne $overviewJob) {
    if ($overviewJob.State -ne 'Completed') { Write-Error "The overview data job failed: $($overviewJob | Receive-Job)"; Remove-Job -Job $jobsToWaitFor; exit 1 }
    $overviewJson = $overviewJob | Receive-Job
    if ([string]::IsNullOrWhiteSpace($overviewJson)) { Write-Error "Empty response from overview API job."; Remove-Job -Job $jobsToWaitFor; exit 1 }
    $overviewData = $overviewJson | ConvertFrom-Json
    Write-Verbose "Raw overview response:`n$overviewJson"
}

Remove-Job -Job $jobsToWaitFor

# Ensure data integrity after fetching.
if (-not $weatherData.current) {
    Write-Error "Unexpected JSON structure: 'current' data not found."
    exit
}
if ((-not $Terse.IsPresent) -and (-not $overviewData.weather_overview)) {
    Write-Error "Unexpected JSON structure: 'overview' data not found."
    exit
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

# Build variables.
$current = $weatherData.current
$daily = $weatherData.daily[0]
$tomorrow = $weatherData.daily[1] 
$nextday = $tomorrow.summary 
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
if ($null -ne $current.rain) {
    $rain = $current.rain
    $conditions = "[$rain.1h mm/H]"
}
if ($null -ne $current.snow) {
    $snow = $current.snow
    $conditions = "[$snow.1h mm/H]"
}
$report = $overviewData.weather_overview
$wrappedReport = Format-TextWrap -Text $report -Width $Host.UI.RawUI.WindowSize.Width
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

if ($windGust -and $null -ne $windGust) {
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
    Clear-Host
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

if (-not $Terse.IsPresent) {
    Write-Host ""
    Write-Host "*** $city Weather Report ***" -ForegroundColor $titleColor
    $wrappedReport | ForEach-Object { Write-Host $_ -ForegroundColor $defaultColor }
    Write-Host ""
    Write-Host "https://forecast.weather.gov/MapClick.php?lat=$lat&lon=$lon" -ForegroundColor Cyan
}

# --- Alert Handling ---
if ($null -ne $weatherData.alerts) {
    foreach ($alertItem in $weatherData.alerts) {
        $alertSender = $alertItem.sender_name
        $alertEvent = $alertItem.event
        $alertStart = ([System.DateTimeOffset]::FromUnixTimeSeconds($alertItem.start)).LocalDateTime
        $alertEnd = ([System.DateTimeOffset]::FromUnixTimeSeconds($alertItem.end)).LocalDateTime
        $alertDesc = $alertItem.description
        Write-Host ""
        Write-Host "*** $alertEvent - $alertSender ***" -ForegroundColor $alertColor
        if (-not $Terse.IsPresent) {
            Write-Host "$alertDesc" -ForegroundColor $defaultColor
        }
        Write-Host "Starts: $alertStart" -ForegroundColor $infoColor
        Write-Host "Ends: $alertEnd" -ForegroundColor $infoColor
    }
}

if ($parentName -notmatch '^(WindowsTerminal.exe|PowerShell|cmd)') {
    Write-Verbose "Parent:$parentName"
    Read-Host -Prompt "Hit Enter to Exit"
}