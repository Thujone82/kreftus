===============================================================================
                                HERE.PS1
                    IP Geolocation Information Script
===============================================================================

DESCRIPTION:
    This PowerShell script retrieves the machine's approximate geographical 
    location using a public IP Geolocation API. It provides a professional,
    colorized output showing location details, coordinates, and network 
    information.

FEATURES:
    - Professional colorized output with status indicators
    - Real-time API query feedback
    - Organized information display (Location, Coordinates, Network)
    - Error handling with clear status messages
    - Clean ASCII-based formatting for universal compatibility

REQUIREMENTS:
    - PowerShell 5.1 or later
    - Active internet connection
    - Windows operating system

USAGE:
    .\here.ps1
    
    Or from any directory:
    PowerShell -ExecutionPolicy Bypass -File "C:\path\to\here.ps1"

OUTPUT SECTIONS:
    Location    - City, Region, Country
    Coordinates - Latitude and Longitude
    Network     - Public IP Address and ISP Provider

ACCURACY NOTE:
    Location accuracy is based on your ISP's IP address assignment, not GPS.
    Results may vary depending on your internet service provider's network
    infrastructure and IP geolocation database accuracy.

API USED:
    ip-api.com (free tier)
    - No API key required
    - Automatic IP detection
    - JSON response format

ERROR HANDLING:
    - Connection timeout detection
    - API failure messages
    - Network connectivity checks
    - Clear error status indicators

STATUS INDICATORS:
    [OK]     - Successful data retrieval
    [FAIL]   - API returned error
    [ERROR]  - Connection failed
    [!]      - Request timeout warning
    [X]      - Location could not be determined

COLOR SCHEME:
    Cyan     - Headers and titles
    Yellow   - Section headers
    Green    - Success indicators
    Red      - Error messages
    Gray     - Data labels and notes
    White    - Data values

EXAMPLES:
    PS C:\Users\Username> .\here.ps1
    
    Querying geolocation service...
    [OK] Location data received
    
      ================================================================
                        GEOLOCATION INFORMATION                      
      ================================================================
      
      Location
          City      : Portland
          Region    : Oregon
          Country   : United States
      
      Coordinates
          Latitude  : 45.4805
          Longitude : -122.6363
      
      Network
          Public IP : 71.34.69.157
          Provider  : CenturyLink
      
      ----------------------------------------------------------------
      [i] Location accuracy is based on ISP IP assignment, not GPS
      ----------------------------------------------------------------

TROUBLESHOOTING:
    - Ensure internet connectivity
    - Check firewall settings
    - Verify PowerShell execution policy
    - Try running as administrator if blocked

VERSION: 1.0
AUTHOR:  Generated for kreftus project
DATE:    Current
LICENSE: See project LICENSE file
