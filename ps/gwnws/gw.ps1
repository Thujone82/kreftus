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
    .\gw.ps1 97219 -Verbose

.EXAMPLE
    .\gw.ps1 "Portland, OR" -Verbose

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
    $defaultConfig = @{ "nws" = @{ "user_agent" = "202508161459PDX" } }
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
    if (-not $ini.nws) { $ini.nws = @{} }
    if (-not $ini.nws.user_agent) { $ini.nws.user_agent = "202508161459PDX" }
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

function Initialize-GwConfiguration {
    $configPath = Get-GwConfigPath
    if (-not $configPath) {
        # Error already shown in Get-GwConfigPath
        exit 1
    }
 
    $config = Get-GwConfiguration -FilePath $configPath
    $userAgent = $config.nws.user_agent
    
    return $userAgent
}

if ($Help -or ($Terse.IsPresent -and -not $Location)) {
    Write-Host "Usage: .\gw.ps1 [ZipCode | `"City, State`"] [-Verbose]" -ForegroundColor Green
    Write-Host " • Provide a 5-digit zipcode or a City, State (e.g., 'Portland, OR')." -ForegroundColor Cyan
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
    Write-Host "  .\gw.ps1 97219 -Verbose" -ForegroundColor Cyan
    Write-Host "  .\gw.ps1 `"Portland, OR`" -Verbose" -ForegroundColor Cyan
    Write-Host "  .\gw.ps1 -Help" -ForegroundColor Cyan
    return
}

# Force TLS 1.2.
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$userAgent = Initialize-GwConfiguration

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
            Write-Host ' GetWeather ' -ForegroundColor Green  -NoNewline; Write-Host "  ________________  " -ForegroundColor Cyan
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

Write-Verbose "Grid info: Office=$office, GridX=$gridX, GridY=$gridY"

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

# Output the results.
Write-Host "*** $city, $state Current Conditions ***" -ForegroundColor $titleColor
Write-Host "Currently: $currentConditions" -ForegroundColor $defaultColor
Write-Host "Temperature: $currentTemp°F" -ForegroundColor $tempColor
Write-Host "Wind: $currentWind $currentWindDir" -ForegroundColor $windColor
Write-Host "Time: $($currentTimeLocal.ToString('MM/dd/yyyy HH:mm'))" -ForegroundColor $infoColor

Write-Host ""
Write-Host "*** Today's Forecast ***" -ForegroundColor $titleColor
$wrappedForecast = Format-TextWrap -Text $todayForecast -Width $Host.UI.RawUI.WindowSize.Width
$wrappedForecast | ForEach-Object { Write-Host $_ -ForegroundColor $defaultColor }

Write-Host ""
Write-Host "*** Tomorrow's Forecast ***" -ForegroundColor $titleColor
$wrappedTomorrow = Format-TextWrap -Text $tomorrowForecast -Width $Host.UI.RawUI.WindowSize.Width
$wrappedTomorrow | ForEach-Object { Write-Host $_ -ForegroundColor $defaultColor }

# --- Alert Handling ---
if ($alertsData -and $alertsData.features.Count -gt 0) {
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

Write-Host ""
Write-Host "https://forecast.weather.gov/MapClick.php?lat=$lat&lon=$lon" -ForegroundColor Cyan

if ($parentName -notmatch '^(WindowsTerminal.exe|PowerShell|cmd)') {
    Write-Verbose "Parent:$parentName"
    Read-Host -Prompt "Hit Enter to Exit"
}