<#
.SYNOPSIS
    Retrieves the machine's approximate geographical location using a public IP Geolocation API.

.DESCRIPTION
    This script uses Invoke-RestMethod to query a third-party web service 
    (ip-api.com) for the geographical location based on the machine's public IP address.
    Optionally accepts an IP address as a command line argument to query a specific IP.

.PARAMETER IPAddress
    Optional IP address to query. If not provided, queries the machine's own public IP.

.NOTES
    - Requires an active internet connection.
    - Location accuracy is based on the IP address, not GPS, so it may be less precise.
    - Usage: .\here.ps1 [IPAddress]
    - Example: .\here.ps1 1.1.1.1
#>

param([string]$IPAddress)

function Write-ModernHeader ($Text) {
    $border = "=" * ($Text.Length + 8)
    # Write border, then fill rest of line with black background
    Write-Host $border -NoNewline -ForegroundColor Cyan
    $consoleWidth = $host.UI.RawUI.BufferSize.Width
    $remainingWidth = $consoleWidth - $border.Length
    if ($remainingWidth -gt 0) {
        Write-Host (" " * $remainingWidth) -NoNewline -BackgroundColor Black
    }
    Write-Host ""
    Write-Host ("    $Text    ") -ForegroundColor Black -BackgroundColor Cyan
    Write-Host $border -NoNewline -ForegroundColor Cyan
    if ($remainingWidth -gt 0) {
        Write-Host (" " * $remainingWidth) -NoNewline -BackgroundColor Black
    }
    Write-Host ""
}

function Write-ModernRow ($Key, $Value) {
    Write-Host ("{0,-12}: " -f $Key) -NoNewline -ForegroundColor Yellow
    Write-Host $Value -ForegroundColor White
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
        SolarNoonUtcMin = $solarNoonUtcMin
    }
}

# Function to calculate moon phase using simple astronomical method
# Uses reference new moon date (January 6, 2000 18:14 UTC) and lunar cycle of 29.53058867 days
# Returns moon phase name (text only, no emoji) and next full/new moon information
function Get-MoonPhase {
    param([DateTime]$Date)
    
    $knownNewMoon = [DateTime]::new(2000, 1, 6, 18, 14, 0, [System.DateTimeKind]::Utc)
    $lunarCycle = 29.53058867
    
    # Calculate phase (0-1 range)
    $daysSince = ($Date.ToUniversalTime() - $knownNewMoon).TotalDays
    $currentCycle = $daysSince % $lunarCycle
    $phase = $currentCycle / $lunarCycle
    
    # Determine phase name using corrected astronomical method
    # Phase ranges: New (0-0.125), Waxing Crescent (0.125-0.25), First Quarter (0.25-0.375),
    # Waxing Gibbous (0.375-0.48), Full (0.48-0.52), Waning Gibbous (0.52-0.75),
    # Last Quarter (0.75-0.875), Waning Crescent (0.875-1.0)
    
    $phaseName = ""
    
    if ($phase -lt 0.125) {
        $phaseName = "New Moon"
    } elseif ($phase -lt 0.25) {
        $phaseName = "Waxing Crescent"
    } elseif ($phase -lt 0.375) {
        $phaseName = "First Quarter"
    } elseif ($phase -lt 0.48) {
        $phaseName = "Waxing Gibbous"
    } elseif ($phase -lt 0.52) {
        $phaseName = "Full Moon"
    } elseif ($phase -lt 0.75) {
        $phaseName = "Waning Gibbous"
    } elseif ($phase -lt 0.875) {
        $phaseName = "Last Quarter"
    } else {
        $phaseName = "Waning Crescent"
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
        IsFullMoon = $isFullMoon
        IsNewMoon = $isNewMoon
        ShowNextFullMoon = $showNextFullMoon
        ShowNextNewMoon = $showNextNewMoon
        NextFullMoon = $nextFullMoonDate
        NextNewMoon = $nextNewMoonDate
    }
}

function Get-MachineIPGeoLocation {
    param([string]$IPAddress)
    
    # Build API URL - append IP if provided, otherwise query own IP
    if ($IPAddress) {
        $IPGeolocationAPI = "http://ip-api.com/json/$IPAddress"
        Write-Host "Querying geolocation for IP: $IPAddress..." -ForegroundColor Gray
    } else {
        $IPGeolocationAPI = "http://ip-api.com/json/"
        Write-Host "Querying public IP geolocation service..." -ForegroundColor Gray
    }
    
    try {
        $Response = Invoke-RestMethod -Uri $IPGeolocationAPI -Method Get -TimeoutSec 5
        if ($Response.status -eq "success") {
            [PSCustomObject]@{
                Latitude  = $Response.lat
                Longitude = $Response.lon
                City      = $Response.city
                Region    = $Response.regionName
                Country   = $Response.country
                PublicIP  = $Response.query
                Provider  = $Response.isp
                Timezone  = $Response.timezone
            }
        } else {
            Write-Host "Geolocation API request failed." -ForegroundColor Red
            Write-Host "Message: $($Response.message)" -ForegroundColor DarkRed
            return $null
        }
    }
    catch {
        Write-Host "Failed to connect to the IP geolocation service." -ForegroundColor Red
        if ($_.Exception.Message -like "*timeout*") {
            Write-Host "The request timed out." -ForegroundColor DarkYellow
        }
        return $null
    }
}

# Script Execution
$GeoLocation = Get-MachineIPGeoLocation -IPAddress $IPAddress

if ($GeoLocation) {
    Write-ModernHeader "IP Location Found (Approximate)"
    Write-ModernRow "Country"   $GeoLocation.Country
    Write-ModernRow "Region"    $GeoLocation.Region
    Write-ModernRow "City"      $GeoLocation.City
    Write-ModernRow "Latitude"  $GeoLocation.Latitude
    Write-ModernRow "Longitude" $GeoLocation.Longitude
    Write-ModernRow "Public IP" $GeoLocation.PublicIP
    Write-ModernRow "Provider"  $GeoLocation.Provider
    
    # Calculate astronomical data
    $currentDate = Get-Date
    $timeZoneId = $GeoLocation.Timezone
    
    # Calculate sunrise and sunset
    $sunTimes = Get-SunriseSunset -Latitude $GeoLocation.Latitude -Longitude $GeoLocation.Longitude -Date $currentDate -TimeZoneId $timeZoneId
    $sunriseTime = $sunTimes.Sunrise
    $sunsetTime = $sunTimes.Sunset
    
    # Calculate moon phase
    $moonPhaseInfo = Get-MoonPhase -Date $currentDate
    $moonPhase = $moonPhaseInfo.Name
    
    # Get timezone info for current local time
    $tzInfo = Get-ResolvedTimeZoneInfo -TimeZoneId $timeZoneId
    $currentLocalTime = [System.TimeZoneInfo]::ConvertTime($currentDate, $tzInfo)
    
    Write-Host ""
    Write-ModernHeader "Astronomical Information"
    
    # Display timezone
    if ($timeZoneId) {
        Write-ModernRow "Timezone" $timeZoneId
    }
    
    # Display current local time
    Write-ModernRow "Local Time" $currentLocalTime.ToString("h:mm tt, MMMM dd, yyyy")
    
    # Display sunrise and sunset
    if ($sunTimes.IsPolarNight) {
        Write-ModernRow "Sunrise" "Polar Night (No Sunrise)"
        Write-ModernRow "Sunset" "Polar Night (No Sunset)"
    } elseif ($sunTimes.IsPolarDay) {
        Write-ModernRow "Sunrise" "Polar Day (No Sunset)"
        Write-ModernRow "Sunset" "Polar Day (No Sunset)"
    } else {
        if ($sunriseTime) {
            Write-ModernRow "Sunrise" $sunriseTime.ToString("h:mm tt")
        }
        if ($sunsetTime) {
            Write-ModernRow "Sunset" $sunsetTime.ToString("h:mm tt")
        }
        
        # Calculate and display day length
        if ($sunriseTime -and $sunsetTime) {
            $dayLength = $sunsetTime - $sunriseTime
            
            # Handle case where sunset is on the next day (negative duration)
            if ($dayLength.TotalHours -lt 0) {
                $dayLength = $dayLength.Add([TimeSpan]::FromDays(1))
            }
            
            $hours = [Math]::Floor($dayLength.TotalHours)
            $minutes = $dayLength.Minutes
            Write-ModernRow "Day Length" "$hours hours $minutes minutes"
            
            # Calculate and display solar noon (midpoint between sunrise and sunset)
            $solarNoon = $sunriseTime.AddMinutes($dayLength.TotalMinutes / 2)
            Write-ModernRow "Solar Noon" $solarNoon.ToString("h:mm tt")
        }
    }
    
    # Display moon phase
    Write-ModernRow "Moon Phase" $moonPhase
    
    # Display next full moon or new moon based on cycle
    if ($moonPhaseInfo.ShowNextFullMoon -and $moonPhaseInfo.NextFullMoon) {
        Write-ModernRow "Next Full" $moonPhaseInfo.NextFullMoon
    }
    if ($moonPhaseInfo.ShowNextNewMoon -and $moonPhaseInfo.NextNewMoon) {
        Write-ModernRow "Next New" $moonPhaseInfo.NextNewMoon
    }
    
    Write-Host ""
    Write-Host "Accuracy is based on your ISP's IP address assignment, not GPS." -ForegroundColor DarkGray
}
else {
    Write-Host ""
    Write-Host "Could not determine machine location." -ForegroundColor Red
}
