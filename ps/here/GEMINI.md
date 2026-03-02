# Gemini Project File

## Project: here (IP Geolocation)

**Author:** Kreft&Cursor  
**Date:** 2026-02-19  
**Version:** 1.2

---

### Description

`here.ps1` is a PowerShell script that retrieves and displays the machine's approximate geographical location using IP-based geolocation. It queries a public API (ip-api.com) for coordinates and timezone based on the machine's public IP (or an optional IP argument), then presents location details, network information, and astronomical data (sunrise, sunset, day length, solar noon, solar irradiance, moon phase) in a colorized, sectioned format. All astronomical values are computed locally from coordinates; no additional APIs are used for sun or moon data.

### Key Functionality

- **No API Key Required:** Uses ip-api.com free tier; no registration or API key.
- **Optional IP Argument:** Query own public IP (default) or a specific IP via `.\here.ps1 [IPAddress]`.
- **Single API Call:** One HTTP GET to ip-api.com returns country, region, city, lat/lon, timezone, ISP, and public IP.
- **Astronomical Data:** Sunrise and sunset (NOAA algorithm), day length, solar noon, clear-sky solar irradiance (GHI in W/m²) at current time, and moon phase with next full/new moon dates. All computed from coordinates and timezone; displayed in the Astronomical Information section.
- **Timezone Handling:** IANA timezone from API; resolved to Windows `TimeZoneInfo` for local time and sun/moon calculations (with fallback mapping for common IANA IDs).
- **Polar Handling:** Polar night (no sunrise/sunset) and polar day (no sunset) detected and displayed with clear labels; irradiance still computed (0 at night, varying by hour angle during polar day if shown).
- **Colorized Output:** Cyan headers, yellow labels, white values, gray status/notes; red/dark red for errors, dark yellow for timeouts.
- **Error Handling:** 5-second timeout, try-catch for API failures, clear messages when location cannot be determined.

### Technical Implementation

The script follows a linear process:

1. **API Request:** Builds URL `http://ip-api.com/json/` or `http://ip-api.com/json/{IPAddress}`; uses `Invoke-RestMethod` with 5-second timeout.
2. **Response Parsing:** Reads JSON for status, country, regionName, city, lat, lon, timezone, query (IP), isp. On failure or non-OK status, exits with error message.
3. **Location Display:** Outputs location block using `Write-ModernHeader` and `Write-ModernRow` (Country, Region, City, Latitude, Longitude, Public IP, Provider).
4. **Astronomical Calculations:** For the location's current date and timezone: `Get-SunriseSunset` (NOAA), `Get-MoonPhase` (reference new moon + lunar cycle), `Get-ResolvedTimeZoneInfo` for local time and UTC conversion. When sunrise/sunset exist (non-polar), computes day length and solar noon from sunrise/sunset times, then `Get-SolarIrradiance` with location's current time converted to UTC.
5. **Astronomical Display:** Writes Astronomical Information section: Timezone, Local Time, Sunrise/Sunset (or polar messages), Day Length, Solar Noon, Irradiance (only when non-polar), Moon Phase, Next Full/New moon when applicable. Ends with accuracy disclaimer.

### API Endpoints Used

- **Geolocation:** `http://ip-api.com/json/` (own IP) or `http://ip-api.com/json/{IPAddress}` (specific IP)
  - Method: GET
  - Response: JSON (country, regionName, city, lat, lon, timezone, query, isp, status)
  - No API key; 5-second timeout in script

### Configuration

- **Timeout:** 5 seconds for the geolocation request (hardcoded).
- **No configuration file:** All settings are in-script.
- **Encoding:** Script may use UTF-8 for any special characters in output; no BOM requirement documented (unlike gf.ps1).

### Features Added/Enhanced

- **Sunrise/Sunset:** NOAA-based calculation from lat/lon and date; returns local times via `Get-ResolvedTimeZoneInfo`; polar night/day handled.
- **Day Length / Solar Noon:** Derived from sunrise/sunset (solar noon = midpoint); shown only when both sunrise and sunset exist.
- **Solar Irradiance (v1.2):** Clear-sky GHI (W/m²) at current time via `Get-SolarIrradiance`; same NOAA solar position math; displayed after Solar Noon as "Irradiance: X W/m²"; only when non-polar.
- **Moon Phase:** Astronomical method (reference new moon Jan 6, 2000 18:14 UTC; cycle 29.53058867 days); phase name and optional Next Full/Next New moon dates.

### Benefits

- **Single dependency:** One external call (geolocation); all astronomical data local.
- **Portable:** No API keys; works from any machine with outbound HTTP.
- **Consistent formatting:** Same row/header style as other kreftus scripts; easy to parse visually.
- **Accurate astronomy:** NOAA algorithms for sun; standard lunar cycle for moon; timezone-aware local time and UTC conversion for irradiance.

### Sunrise and Sunset (NOAA)

Sunrise and sunset times are computed with the same NOAA-style formulas used in gf.ps1: fractional year (γ), equation of time, solar declination, hour angle at sunrise/sunset (zenith 90.833° including refraction), solar noon in UTC minutes, then conversion to local time via `Get-ResolvedTimeZoneInfo`.

**Function:** `Get-SunriseSunset`  
**Location:** Lines 77–148

**Parameters:**
- `[double]$Latitude`, `[double]$Longitude` – Location
- `[DateTime]$Date` – Date for the calculation (local date used by caller)
- `[string]$TimeZoneId` – IANA or Windows timezone ID

**Algorithm (summary):**
- Zenith = 90.833° (standard atmospheric refraction).
- γ = 2π × (dayOfYear − 1) / 365.
- Equation of time (minutes) and solar declination (radians): NOAA coefficient series in γ.
- cos(H) for sunrise/sunset from sin(lat)·sin(δ) + cos(lat)·cos(δ)·cos(H) = cos(90.833°); solve for H.
- If cos(H) > 1 → polar night (no sunrise/sunset); if cos(H) < −1 → polar day.
- Solar noon (UTC minutes from midnight): 720 − 4×Longitude − equationOfTime.
- Sunrise/sunset UTC minutes: solarNoon ± 4×H_deg (15° per hour); normalized to [0, 1440); converted to UTC `DateTime`, then to local time with `TimeZoneInfo.ConvertTimeFromUtc`.

**Return value:** Hashtable with `Sunrise`, `Sunset` (local `DateTime` or `$null`), `IsPolarNight`, `IsPolarDay`, and (for non-polar) `SolarNoonUtcMin`.

**Display:** In the Astronomical section; polar cases show "Polar Night (No Sunrise)" / "Polar Day (No Sunset)" instead of times. Day length and solar noon (and irradiance) are only computed when both sunrise and sunset exist.

### Solar Irradiance (v1.2)

Solar irradiance is clear-sky global horizontal irradiance (GHI) in W/m² at the location’s current time. It is shown after the Solar Noon row in the Astronomical Information section, only when sunrise and sunset are available (non-polar).

**Function:** `Get-SolarIrradiance`  
**Location:** Lines 152–178

**Parameters:**
- `[double]$Latitude`, `[double]$Longitude` – Location
- `[DateTime]$Date` – Time for which to compute irradiance (caller passes UTC; function uses `$Date.ToUniversalTime()` for consistency)
- `[string]$TimeZoneId` – Used by callers for context; calculation is UTC-based

**Algorithm:**
1. **UTC and date:** `$utcNow = $Date.ToUniversalTime()`; day-of-year from UTC date.
2. **NOAA solar position (same as sunrise/sunset):** γ = 2π×(dayOfYear−1)/365; equation of time (minutes); solar declination (radians); solar noon in UTC minutes: 720 − 4×Longitude − equationOfTime.
3. **Hour angle:** `utcMinutesFromMidnight = hour×60 + minute + second/60`; `H_deg = (utcMinutesFromMidnight − solarNoonUtcMin) / 4` (15° per hour from solar noon).
4. **Solar zenith:** cos(θ) = sin(lat)·sin(δ) + cos(lat)·cos(δ)·cos(H_rad). Inputs clamped to [−1, 1] for safety.
5. **GHI:** If cos(θ) ≤ 0 (sun below horizon), return 0. Otherwise GHI = 1000 × cos(θ); return rounded integer (simple clear-sky model; no clouds or aerosols).

**Display integration:**
- Caller (main script, lines 360–364): Only when not polar and sunrise/sunset exist, gets current local time, converts to UTC with `[TimeZoneInfo]::ConvertTimeToUtc($currentLocalTime, $tzInfo)`, calls `Get-SolarIrradiance` with that UTC `DateTime`, then `Write-ModernRow "Irradiance" "$irradianceWm2 W/m²"` after the Solar Noon row. Uses same color scheme (yellow label, white value).

**Edge cases:**
- **Polar night:** Sun never rises; irradiance would be 0; row is not shown because the irradiance block is inside the non-polar sunrise/sunset branch.
- **Polar day:** Script currently does not show irradiance in polar day (irradiance block is only in the branch where both sunrise and sunset exist). If added later, calculation would still be valid (hour angle varies through the day).
- **Missing timezone/coords:** Caller only calls when `$sunriseTime` and `$sunsetTime` exist and timezone is resolved.

**Data flow:**
1. After geolocation, script has `$GeoLocation.Latitude`, `$GeoLocation.Longitude`, `$timeZoneId`, and `$currentDate` (system local).
2. `Get-SunriseSunset` and `Get-MoonPhase` run; `$tzInfo = Get-ResolvedTimeZoneInfo -TimeZoneId $timeZoneId`; `$currentLocalTime = [TimeZoneInfo]::ConvertTime($currentDate, $tzInfo)`.
3. In the non-polar branch, after writing Solar Noon: `$nowUtc = [TimeZoneInfo]::ConvertTimeToUtc($currentLocalTime, $tzInfo)`; `$irradianceWm2 = Get-SolarIrradiance -Latitude ... -Longitude ... -Date $nowUtc -TimeZoneId $timeZoneId`; `Write-ModernRow "Irradiance" "$irradianceWm2 W/m²"`.

**Notes:** No external API for irradiance; clear-sky estimate only. Formula 1000×cos(zenith) approximates typical GHI at zenith ~1000 W/m²; real conditions vary with atmosphere and clouds.

### Moon Phase

Moon phase is computed from a known new moon reference and the synodic lunar cycle; returns phase name and optional next full moon and next new moon dates.

**Function:** `Get-MoonPhase`  
**Location:** Lines 184–255

**Parameters:** `[DateTime]$Date` – Date (and time) for which to compute phase; converted to UTC internally for consistency.

**Reference and cycle:**
- Reference new moon: January 6, 2000 18:14 UTC.
- Lunar cycle: 29.53058867 days (synodic month).

**Algorithm:**
- `daysSince = ($Date.ToUniversalTime() - $knownNewMoon).TotalDays`
- `currentCycle = daysSince % lunarCycle` (position in 0..29.53…)
- `phase = currentCycle / lunarCycle` (0–1)

**Phase name thresholds (here.ps1):**
- New Moon: phase < 0.125
- Waxing Crescent: 0.125–0.25
- First Quarter: 0.25–0.375
- Waxing Gibbous: 0.375–0.48
- Full Moon: 0.48–0.52
- Waning Gibbous: 0.52–0.75
- Last Quarter: 0.75–0.875
- Waning Crescent: 0.875–1.0

**Next full moon:** `daysUntilNextFullMoon = (14.77 - currentCycle) % lunarCycle`; if ≤ 0 add one cycle; `nextFullMoonDate = $Date.AddDays($daysUntilNextFullMoon).ToString("MM/dd/yyyy")`.  
**Next new moon:** `daysUntilNextNewMoon = lunarCycle - currentCycle`; same date formatting.

**Return value:** Hashtable: `Name`, `IsFullMoon`, `IsNewMoon`, `ShowNextFullMoon`, `ShowNextNewMoon`, `NextFullMoon`, `NextNewMoon`.

**Display:** Moon Phase row always; "Next Full" row when `ShowNextFullMoon`; "Next New" row when `ShowNextNewMoon` (logic in script lines 370–375).

### Timezone Resolution

The script converts the API’s IANA timezone (e.g. `America/Los_Angeles`) to a .NET `TimeZoneInfo` so that sunrise/sunset and local time can be computed and displayed correctly.

**Function:** `Get-ResolvedTimeZoneInfo`  
**Location:** Lines 47–74

**Parameter:** `[string]$TimeZoneId` – IANA or Windows timezone ID (e.g. from ip-api.com).

**Logic:**
1. If `TimeZoneId` is null/whitespace, return `[TimeZoneInfo]::Local`.
2. Try `[TimeZoneInfo]::FindSystemTimeZoneById($TimeZoneId)` (works for Windows IDs).
3. If that fails, look up in a small IANA→Windows map (e.g. `America/Los_Angeles` → `Pacific Standard Time`); try `FindSystemTimeZoneById` again.
4. If still no match, return `[TimeZoneInfo]::Local`.

**Usage:** Used by `Get-SunriseSunset` to convert UTC sunrise/sunset to local time; by main script to get `$currentLocalTime` and to convert local time to UTC for `Get-SolarIrradiance`.

## Usage Examples

### Basic Execution
```powershell
.\here.ps1
```

### Query Specific IP
```powershell
.\here.ps1 1.1.1.1
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
Public IP : 71.34.69.187
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
- **Connection Timeout:** 5-second timeout on `Invoke-RestMethod`; script catches and shows timeout message (e.g. "The request timed out.").
- **API Failures:** Response JSON may include `status`; non-OK status triggers failure path with appropriate message.
- **Network Issues:** Try-catch around REST call; "Failed to connect to the IP geolocation service." or similar when connection fails.
- **No Data:** If API returns failure or no valid location, script outputs "Could not determine machine location." in red and does not render location or astronomical sections.

### Error Messages
- `"Geolocation API request failed."` – API returned error or invalid response.
- `"Failed to connect to the IP geolocation service."` – Network or connectivity failure.
- `"The request timed out."` – Request exceeded 5-second timeout.
- `"Could not determine machine location."` – Final fallback when location cannot be resolved.

### Astronomical Edge Cases
- **Polar night/day:** `Get-SunriseSunset` returns `IsPolarNight` or `IsPolarDay`; display shows explanatory text instead of times; day length, solar noon, and irradiance are not shown in polar branches.
- **Timezone resolution failure:** `Get-ResolvedTimeZoneInfo` falls back to `[TimeZoneInfo]::Local`; sunrise/sunset and local time may be wrong if API timezone is not in the IANA→Windows map and `FindSystemTimeZoneById` fails.

## Accuracy Considerations

- **Method:** Location is derived from the machine’s public IP (or the optional IP argument) via ip-api.com; not GPS or device location.
- **Accuracy:** Depends on the geolocation database and ISP assignment; typically city- or region-level.
- **Variability:** Results can vary by provider and database; same IP may resolve differently over time or across services.
- **Limitations:** Not suitable for precision (e.g. street-level or legal); script shows disclaimer: "Accuracy is based on your ISP's IP address assignment, not GPS."
- **Astronomical data:** Sunrise, sunset, irradiance, and moon phase are computed from the returned coordinates and timezone; their accuracy is limited by the accuracy of that location and timezone, not by the astronomy formulas.

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
- **Entry:** `param([string]$IPAddress)`; single code path: call `Get-MachineIPGeoLocation`, then on success display location block and astronomical block (lines 248–385).
- **Display helpers:** `Write-ModernHeader` (lines 23–38), `Write-ModernRow` (lines 41–44) – cyan header/borders, yellow label, white value.
- **Timezone:** `Get-ResolvedTimeZoneInfo` (lines 47–74) – IANA/Windows ID to `TimeZoneInfo` with fallback map.
- **Sun/moon:** `Get-SunriseSunset` (77–148), `Get-SolarIrradiance` (152–178), `Get-MoonPhase` (184–255).
- **Geolocation:** `Get-MachineIPGeoLocation` (248–320) – builds ip-api.com URL, `Invoke-RestMethod` with 5s timeout, returns PSCustomObject (Latitude, Longitude, City, Region, Country, Timezone, PublicIP, Provider) or $null; displays location and astronomical sections (302–378).
- **Astronomical display logic (328–378):** Current date and timezone; get sunrise/sunset and moon phase; resolve `$tzInfo` and `$currentLocalTime`; write Astronomical header; Timezone, Local Time; polar vs normal sunrise/sunset; in normal branch: Day Length, Solar Noon, then Irradiance via `Get-SolarIrradiance` with `$nowUtc`; Moon Phase; Next Full/Next New when applicable; accuracy disclaimer.
- **Error handling:** Try-catch around API call; timeout; status check on response; "Could not determine machine location" on failure.
- **Encoding:** No explicit UTF-8 BOM requirement documented; script is plain ASCII plus optional Unicode in output (e.g. W/m²).

### Performance
- **API response time:** ~1–3 seconds typical for ip-api.com.
- **Timeout:** 5 seconds (script-level).
- **Memory:** Minimal; single API response and a few date/time/astronomical values.
- **Network:** One HTTP GET per run (or none if cached by shell/user).

## License
Part of the kreftus project. See main project LICENSE file for details.

### Recent Enhancements (v1.1–v1.2)
- **Astronomical block (v1.1):** Sunrise, sunset, day length, solar noon, timezone, local time, moon phase with next full/new moon; NOAA-based sun, reference-new-moon lunar cycle; polar night/day handling.
- **Solar irradiance (v1.2):** Clear-sky GHI (W/m²) at current time via `Get-SolarIrradiance`; displayed after Solar Noon in Astronomical section; only when non-polar; same NOAA solar position math as sunrise/sunset.

## Version History
- **v1.0**: Initial release with modern formatting and clean colorized output
- **v1.1**: Added astronomical information including sunrise, sunset, moon phase, day length, solar noon, timezone, and local time calculations
- **v1.2**: Added solar irradiance (clear-sky GHI in W/m²) displayed after Solar Noon using `Get-SolarIrradiance`