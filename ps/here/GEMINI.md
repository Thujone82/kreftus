# HERE.PS1 - IP Geolocation Script

## Overview
A clean and modern PowerShell script that retrieves and displays machine geographical location information using IP-based geolocation services. Features modern colorized output, real-time status feedback, organized information presentation with professional formatting, and astronomical calculations including sunrise, sunset, moon phase, and solar information.

## Features

### 🎨 Modern Display
- **Clean Formatting**: Modern header with cyan borders and black text on cyan background
- **Colorized Output**: Cyan headers, yellow labels, white values, gray status messages
- **Professional Layout**: Consistent spacing and alignment for all data fields
- **Real-time Feedback**: Live status updates during API queries

### 📍 Information Display
- **Country**: Country name
- **Region**: State/Province/Region name  
- **City**: City name
- **Latitude**: Geographic latitude coordinate
- **Longitude**: Geographic longitude coordinate
- **Public IP**: Your public IP address
- **Provider**: Internet Service Provider name

### 🌅 Astronomical Information
- **Timezone**: IANA timezone identifier from geolocation API
- **Local Time**: Current local time at the location
- **Sunrise**: Calculated sunrise time (NOAA algorithm)
- **Sunset**: Calculated sunset time (NOAA algorithm)
- **Day Length**: Duration between sunrise and sunset (hours and minutes)
- **Solar Noon**: Time when sun reaches highest point in sky
- **Moon Phase**: Current moon phase (text description, e.g., "New Moon", "Waxing Crescent")

### 🛡️ Error Handling
- Connection timeout detection (5 seconds)
- API failure message handling
- Network connectivity validation
- Clear error messages with appropriate color coding
- Specific timeout warning messages
- Polar day/night handling for extreme latitudes

### 🌙 Astronomical Calculations
- **Sunrise/Sunset**: NOAA-based astronomical calculations using coordinates
- **Moon Phase**: Astronomical method using reference new moon date and lunar cycle
- **Timezone Resolution**: Automatic IANA to Windows timezone conversion
- **Local Calculations**: All calculations performed locally (no additional API calls)
- **Edge Cases**: Handles polar day/night scenarios where sunrise/sunset don't occur

## Technical Details

### API Integration
- **Service**: ip-api.com (free tier)
- **Method**: HTTP GET request
- **Format**: JSON response
- **Timeout**: 5 seconds
- **Auto-detection**: Uses machine's public IP
- **No API Key**: Required

### PowerShell Features
- **Version**: 5.1+ compatible
- **Execution Policy**: Bypass recommended
- **Error Handling**: Try-catch with specific error types
- **Output**: PSCustomObject for structured data
- **Functions**: Modular design with `Write-ModernHeader` and `Write-ModernRow`

## Usage Examples

### Basic Execution
```powershell
.\here.ps1
```

### Remote Execution
```powershell
PowerShell -ExecutionPolicy Bypass -File "C:\path\to\here.ps1"
```

### Expected Output
```
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
Moon Phase: Waxing Gibbous

Accuracy is based on your ISP's IP address assignment, not GPS.
```

## Color Scheme

| Color | Usage | Example |
|-------|-------|---------|
| `Cyan` | Headers and borders | Main title border, section headers |
| `Black on Cyan` | Header text | Title text background |
| `Yellow` | Field labels | "Country", "Region", "City" |
| `White` | Data values | Actual location data |
| `Red` | Error messages | Connection failures |
| `Gray` | Status messages | "Querying..." messages |
| `DarkRed` | Detailed errors | API error details |
| `DarkYellow` | Timeout warnings | Timeout notifications |

## Error Handling

### Error Types
- **Connection Timeout**: 5-second timeout with specific warning
- **API Failures**: Status-based error detection
- **Network Issues**: Connection failure detection
- **No Data**: Clear message when location cannot be determined

### Error Messages
- `"Geolocation API request failed."`
- `"Failed to connect to the IP geolocation service."`
- `"The request timed out."`
- `"Could not determine machine location."`

## Accuracy Considerations

- **Method**: IP-based geolocation (not GPS)
- **Accuracy**: Depends on ISP IP assignment
- **Variability**: Results may vary by provider
- **Limitations**: Not as precise as GPS coordinates
- **Note**: Accuracy disclaimer displayed in output

## Requirements

- PowerShell 5.1 or later
- Active internet connection
- Windows operating system
- No API key required
- No additional dependencies

## Troubleshooting

### Common Issues
1. **Connection Timeout**: Check internet connectivity
2. **Execution Policy**: Use `-ExecutionPolicy Bypass`
3. **Firewall**: Ensure outbound HTTP access
4. **API Limits**: Free tier has usage limits

### Error Messages
- `"Failed to connect to the IP geolocation service."`
- `"Geolocation API request failed."`
- `"The request timed out."`
- `"Could not determine machine location."`

## Development Notes

### Code Structure
- Function-based design with `Get-MachineIPGeoLocation`
- Helper functions: `Write-ModernHeader` and `Write-ModernRow`
- Astronomical functions: `Get-SunriseSunset`, `Get-MoonPhase`, `Get-ResolvedTimeZoneInfo`
- Error handling with try-catch blocks
- Colorized output using `Write-Host` with `-ForegroundColor`
- Structured data return using `PSCustomObject`
- Local astronomical calculations (no external API dependencies)

### Performance
- **API Response Time**: ~1-3 seconds typical
- **Timeout**: 5 seconds maximum
- **Memory Usage**: Minimal (single API call)
- **Network**: Single HTTP GET request

## License
Part of the kreftus project. See main project LICENSE file for details.

## Version History
- **v1.0**: Initial release with modern formatting and clean colorized output
- **v1.1**: Added astronomical information including sunrise, sunset, moon phase, day length, solar noon, timezone, and local time calculations