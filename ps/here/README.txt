===============================================================================
                                HERE.PS1
                    IP Geolocation Information Script
===============================================================================

DESCRIPTION:
    This PowerShell script retrieves the machine's approximate geographical 
    location using a public IP Geolocation API. It provides a clean, modern
    colorized output showing location details, coordinates, and network 
    information with professional formatting.

FEATURES:
    - Modern colorized output with clean formatting
    - Real-time API query feedback
    - Organized information display with modern headers
    - Comprehensive error handling with timeout detection
    - Clean ASCII-based formatting for universal compatibility

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
    Country   - Country name
    Region    - State/Province/Region
    City      - City name
    Latitude  - Geographic latitude coordinate
    Longitude - Geographic longitude coordinate
    Public IP - Your public IP address
    Provider  - Internet Service Provider name

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
    
    Accuracy is based on your ISP's IP address assignment, not GPS.

TROUBLESHOOTING:
    - Ensure internet connectivity
    - Check firewall settings
    - Verify PowerShell execution policy
    - Try running as administrator if blocked

VERSION: 1.0
AUTHOR:  Generated for kreftus project
DATE:    Current
LICENSE: See project LICENSE file
