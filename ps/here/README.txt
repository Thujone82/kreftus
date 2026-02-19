===============================================================================
                                HERE.PS1
                    IP Geolocation Information Script
===============================================================================

DESCRIPTION:
    This PowerShell script retrieves the machine's approximate geographical 
    location using a public IP Geolocation API. It provides a clean, modern
    colorized output showing location details, coordinates, network information,
    and astronomical data (sunrise, sunset, moon phase, solar information) with
    professional formatting.

FEATURES:
    - Modern colorized output with clean formatting
    - Real-time API query feedback
    - Organized information display with modern headers
    - Comprehensive error handling with timeout detection
    - Clean ASCII-based formatting for universal compatibility
    - Astronomical calculations (sunrise, sunset, moon phase)
    - Day length, solar noon, and solar irradiance (clear-sky GHI in W/m²) calculations
    - Timezone-aware local time display
    - Polar day/night handling for extreme latitudes

REQUIREMENTS:
    - PowerShell 5.1 or later
    - Active internet connection
    - Windows operating system

USAGE:
    .\here.ps1
    
    Or from any directory:
    PowerShell -ExecutionPolicy Bypass -File "C:\path\to\here.ps1"

OUTPUT FORMAT:
    The script displays information in a clean, modern format with:
    - Cyan-colored header with title
    - Yellow labels for each data field
    - White values for the actual data
    - Gray informational notes

INFORMATION DISPLAYED:
    Location Information:
    Country   - Country name
    Region    - State/Province/Region
    City      - City name
    Latitude  - Geographic latitude coordinate
    Longitude - Geographic longitude coordinate
    Public IP - Your public IP address
    Provider  - Internet Service Provider name
    
    Astronomical Information:
    Timezone  - IANA timezone identifier
    Local Time - Current local time at location
    Sunrise   - Calculated sunrise time (NOAA algorithm)
    Sunset    - Calculated sunset time (NOAA algorithm)
    Day Length - Duration between sunrise and sunset
    Solar Noon - Time when sun reaches highest point
    Irradiance  - Clear-sky solar irradiance at current time (W/m²)
    Moon Phase - Current moon phase (text description)

ACCURACY NOTE:
    Location accuracy is based on your ISP's IP address assignment, not GPS.
    Results may vary depending on your internet service provider's network
    infrastructure and IP geolocation database accuracy.

API USED:
    ip-api.com (free tier)
    - No API key required
    - Automatic IP detection
    - JSON response format
    - 5-second timeout

ERROR HANDLING:
    - Connection timeout detection (5 seconds)
    - API failure message handling
    - Network connectivity validation
    - Clear error messages with appropriate colors

COLOR SCHEME:
    Cyan     - Headers and borders
    Yellow   - Field labels
    White    - Data values
    Red      - Error messages
    Gray     - Status messages and notes
    DarkRed  - Detailed error messages
    DarkYellow - Timeout warnings

EXAMPLE OUTPUT:
    PS C:\Users\Username> .\here.ps1
    
    Querying public IP geolocation service...
    
    ================================================
        IP Location Found (Approximate)        
    ================================================
    Country   : United States
    Region    : Oregon
    City      : Portland
    Latitude  : 45.4805
    Longitude : -122.6363
    Public IP : 71.34.69.157
    Provider  : CenturyLink
    
    ================================================
        Astronomical Information        
    ================================================
    Timezone  : America/Los_Angeles
    Local Time: 12:27 PM, January 01, 2026
    Sunrise   : 7:50 AM
    Sunset    : 4:36 PM
    Day Length: 8 hours 46 minutes
    Solar Noon: 12:13 PM
    Irradiance : 258 W/m²
    Moon Phase: Waxing Gibbous
    
    Accuracy is based on your ISP's IP address assignment, not GPS.

TROUBLESHOOTING:
    - Ensure internet connectivity
    - Check firewall settings
    - Verify PowerShell execution policy
    - Try running as administrator if blocked

ASTRONOMICAL CALCULATIONS:
    - Sunrise and sunset times calculated using NOAA algorithms
    - Solar irradiance (clear-sky GHI) at current time in W/m² (simple zenith-angle model)
    - Moon phase calculated using astronomical method (reference date: Jan 6, 2000)
    - All calculations performed locally using coordinates (no additional APIs)
    - Automatic timezone resolution (IANA to Windows timezone conversion)
    - Handles edge cases: polar day/night scenarios

VERSION: 1.2
AUTHOR:  Generated for kreftus project
DATE:    Current
LICENSE: See project LICENSE file

CHANGELOG:
    v1.2 - Added solar irradiance (clear-sky GHI in W/m²) after Solar Noon
    v1.1 - Added astronomical information (sunrise, sunset, moon phase, 
           day length, solar noon, timezone, local time)
    v1.0 - Initial release with modern formatting and clean colorized output
