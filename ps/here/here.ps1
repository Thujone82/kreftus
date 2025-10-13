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

function Get-MachineIPGeoLocation {
    $IPGeolocationAPI = "http://ip-api.com/json/"

    try {
        Write-Host "`n  " -NoNewline
        Write-Host "Querying geolocation service" -ForegroundColor Cyan -NoNewline
        Write-Host "..." -ForegroundColor DarkGray
        
        # Invoke the web request. The API automatically detects the public IP.
        $Response = Invoke-RestMethod -Uri $IPGeolocationAPI -Method Get -TimeoutSec 5

        # Check for success status from the API
        if ($Response.status -eq "success") {
            Write-Host "  [" -NoNewline -ForegroundColor DarkGray
            Write-Host "OK" -ForegroundColor Green -NoNewline
            Write-Host "] Location data received" -ForegroundColor Gray
            
            [PSCustomObject]@{
                Latitude    = $Response.lat
                Longitude   = $Response.lon
                City        = $Response.city
                Region      = $Response.regionName
                Country     = $Response.country
                PublicIP    = $Response.query
                Provider    = $Response.isp
            }
        } else {
            Write-Host "`n  [" -NoNewline -ForegroundColor DarkGray
            Write-Host "FAIL" -ForegroundColor Red -NoNewline
            Write-Host "] API Error: $($Response.message)" -ForegroundColor Red
            return $null
        }
    }
    catch {
        Write-Host "`n  [" -NoNewline -ForegroundColor DarkGray
        Write-Host "ERROR" -ForegroundColor Red -NoNewline
        Write-Host "] Connection failed. Check your internet connection." -ForegroundColor Red
        
        # If the error is specific to a timeout, you can include that information
        if ($_.Exception.Message -like "*timeout*") {
            Write-Host "  [" -NoNewline -ForegroundColor DarkGray
            Write-Host "!" -ForegroundColor Yellow -NoNewline
            Write-Host "] Request timed out" -ForegroundColor Yellow
        }
        return $null
    }
}

# Execute the function and display the result
$GeoLocation = Get-MachineIPGeoLocation

if ($GeoLocation) {
    # Header
    Write-Host "`n"
    Write-Host "  =================================================================" -ForegroundColor DarkCyan
    Write-Host "                    GEOLOCATION INFORMATION                      " -ForegroundColor Cyan
    Write-Host "  =================================================================" -ForegroundColor DarkCyan
    Write-Host ""
    
    # Display formatted information
    Write-Host "  " -NoNewline
    Write-Host "Location" -ForegroundColor Yellow -NoNewline
    Write-Host ""
    Write-Host "    " -NoNewline
    Write-Host "City      : " -ForegroundColor Gray -NoNewline
    Write-Host $GeoLocation.City -ForegroundColor White
    Write-Host "    " -NoNewline
    Write-Host "Region    : " -ForegroundColor Gray -NoNewline
    Write-Host $GeoLocation.Region -ForegroundColor White
    Write-Host "    " -NoNewline
    Write-Host "Country   : " -ForegroundColor Gray -NoNewline
    Write-Host $GeoLocation.Country -ForegroundColor White
    
    Write-Host ""
    Write-Host "  " -NoNewline
    Write-Host "Coordinates" -ForegroundColor Yellow
    Write-Host "    " -NoNewline
    Write-Host "Latitude  : " -ForegroundColor Gray -NoNewline
    Write-Host $GeoLocation.Latitude -ForegroundColor White
    Write-Host "    " -NoNewline
    Write-Host "Longitude : " -ForegroundColor Gray -NoNewline
    Write-Host $GeoLocation.Longitude -ForegroundColor White
    
    Write-Host ""
    Write-Host "  " -NoNewline
    Write-Host "Network" -ForegroundColor Yellow
    Write-Host "    " -NoNewline
    Write-Host "Public IP : " -ForegroundColor Gray -NoNewline
    Write-Host $GeoLocation.PublicIP -ForegroundColor White
    Write-Host "    " -NoNewline
    Write-Host "Provider  : " -ForegroundColor Gray -NoNewline
    Write-Host $GeoLocation.Provider -ForegroundColor White
    
    # Footer
    Write-Host ""
    Write-Host "  ----------------------------------------------------------------" -ForegroundColor DarkGray
    Write-Host "  [i] Location accuracy is based on ISP IP assignment, not GPS" -ForegroundColor DarkGray
    Write-Host "  ----------------------------------------------------------------" -ForegroundColor DarkGray
    Write-Host ""
} else {
    Write-Host "`n  [X] Could not determine machine location`n" -ForegroundColor Red
}