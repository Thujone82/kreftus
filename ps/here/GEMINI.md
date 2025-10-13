# HERE.PS1 - IP Geolocation Script

## Overview
A professional PowerShell script that retrieves and displays machine geographical location information using IP-based geolocation services. Features colorized output, real-time status indicators, and organized information presentation.

## Features

### 🎨 Professional Display
- **Colorized Output**: Cyan headers, yellow sections, green success indicators
- **Status Indicators**: `[OK]`, `[FAIL]`, `[ERROR]`, `[!]`, `[X]` with appropriate colors
- **Clean Formatting**: ASCII-based borders and consistent spacing
- **Real-time Feedback**: Live status updates during API queries

### 📍 Information Sections
- **Location**: City, Region, Country
- **Coordinates**: Latitude and Longitude
- **Network**: Public IP Address and ISP Provider

### 🛡️ Error Handling
- Connection timeout detection
- API failure message handling
- Network connectivity validation
- Clear error status with color coding

## Technical Details

### API Integration
- **Service**: ip-api.com (free tier)
- **Method**: HTTP GET request
- **Format**: JSON response
- **Timeout**: 5 seconds
- **Auto-detection**: Uses machine's public IP

### PowerShell Features
- **Version**: 5.1+ compatible
- **Execution Policy**: Bypass recommended
- **Error Handling**: Try-catch with specific error types
- **Output**: PSCustomObject for structured data

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
```

## Color Scheme

| Color | Usage | Example |
|-------|-------|---------|
| `Cyan` | Headers and titles | Main title, section headers |
| `Yellow` | Section labels | "Location", "Coordinates" |
| `Green` | Success indicators | `[OK]` status |
| `Red` | Error messages | `[ERROR]`, `[FAIL]` |
| `Gray` | Data labels and notes | Field names, footer info |
| `White` | Data values | Actual location data |

## Status Indicators

| Indicator | Color | Meaning |
|-----------|-------|---------|
| `[OK]` | Green | Successful data retrieval |
| `[FAIL]` | Red | API returned error |
| `[ERROR]` | Red | Connection failed |
| `[!]` | Yellow | Request timeout warning |
| `[X]` | Red | Location could not be determined |

## Accuracy Considerations

- **Method**: IP-based geolocation (not GPS)
- **Accuracy**: Depends on ISP IP assignment
- **Variability**: Results may vary by provider
- **Limitations**: Not as precise as GPS coordinates

## Requirements

- PowerShell 5.1 or later
- Active internet connection
- Windows operating system
- No API key required

## Troubleshooting

### Common Issues
1. **Connection Timeout**: Check internet connectivity
2. **Execution Policy**: Use `-ExecutionPolicy Bypass`
3. **Firewall**: Ensure outbound HTTP access
4. **API Limits**: Free tier has usage limits

### Error Messages
- `Connection failed. Check your internet connection.`
- `API Error: [message]`
- `Request timed out.`
- `Could not determine machine location.`

## Development Notes

### Code Structure
- Function-based design with `Get-MachineIPGeoLocation`
- Error handling with try-catch blocks
- Colorized output using `Write-Host` with `-ForegroundColor`
- Structured data return using `PSCustomObject`

### Performance
- **API Response Time**: ~1-3 seconds typical
- **Timeout**: 5 seconds maximum
- **Memory Usage**: Minimal (single API call)

## License
Part of the kreftus project. See main project LICENSE file for details.

## Version History
- **v1.0**: Initial release with professional formatting and colorized output
