<#
.SYNOPSIS
    Retrieves the machine's approximate geographical location using a public IP Geolocation API.

.DESCRIPTION
    This script uses Invoke-RestMethod to query a third-party web service 
    (ip-api.com) for the geographical location based on the machine's public IP address.

.NOTES
    - Requires an active internet connection.
    - Location accuracy is based on the IP address, not GPS, so it may be less precise.
#>

function Write-ModernHeader ($Text) {
    Write-Host ("=" * ($Text.Length + 8)) -ForegroundColor Cyan
    Write-Host ("    $Text    ") -ForegroundColor Black -BackgroundColor Cyan
    Write-Host ("=" * ($Text.Length + 8)) -ForegroundColor Cyan
}

function Write-ModernRow ($Key, $Value) {
    Write-Host ("{0,-12}: " -f $Key) -NoNewline -ForegroundColor Yellow
    Write-Host $Value -ForegroundColor White
}

function Get-MachineIPGeoLocation {
    $IPGeolocationAPI = "http://ip-api.com/json/"
    try {
        Write-Host "Querying public IP geolocation service..." -ForegroundColor Gray
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
$GeoLocation = Get-MachineIPGeoLocation

if ($GeoLocation) {
    Write-ModernHeader "IP Location Found (Approximate)"
    Write-ModernRow "Country"   $GeoLocation.Country
    Write-ModernRow "Region"    $GeoLocation.Region
    Write-ModernRow "City"      $GeoLocation.City
    Write-ModernRow "Latitude"  $GeoLocation.Latitude
    Write-ModernRow "Longitude" $GeoLocation.Longitude
    Write-ModernRow "Public IP" $GeoLocation.PublicIP
    Write-ModernRow "Provider"  $GeoLocation.Provider
    Write-Host ""
    Write-Host "Accuracy is based on your ISP's IP address assignment, not GPS." -ForegroundColor DarkGray
}
else {
    Write-Host ""
    Write-Host "Could not determine machine location." -ForegroundColor Red
}
