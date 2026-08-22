Portland Big Pipe Report Script
================================

DESCRIPTION
-----------
This PowerShell script fetches and displays real-time statistics about the 
Portland Big Pipe system, which is a combined sewer overflow (CSO) control 
system. The script retrieves 15-minute interval data from the City of Portland's 
website and presents it in an easy-to-read format with color-coded statistics 
and visual sparkline graphs.

REQUIREMENTS
------------
- Windows PowerShell 5.1 or later (or PowerShell Core 6.0+)
- Internet connection to access https://www.portlandoregon.gov/bes/bigpipe/data.cfm
- Console that supports Unicode block characters and colors

USAGE
-----
Simply run the script from PowerShell:

    .\pipe.ps1

Or from any directory:

    powershell -ExecutionPolicy Bypass -File .\pipe.ps1

The script will:
1. Clear the screen (unless specific output is requested)
2. Fetch the latest data from the Portland Big Pipe website
3. Parse all available data points (up to 72 hours)
4. Calculate statistics
5. Display the formatted report

COMMAND-LINE ARGUMENTS
----------------------
The script supports command-line arguments to display specific output lines:

    .\pipe.ps1 -banner          # Show banner line only
    .\pipe.ps1 -b               # Alias for -banner
    .\pipe.ps1 -level           # Show current level line only
    .\pipe.ps1 -l               # Alias for -level
    .\pipe.ps1 -capacity        # Show 100% Duration line only (when at 100%)
    .\pipe.ps1 -cap             # Alias for -capacity
    .\pipe.ps1 -lastfull        # Show Last Full line only (when not at 100%)
    .\pipe.ps1 -lf              # Alias for -lastfull
    .\pipe.ps1 -sma             # Show 12/24/72H SMA line only
    .\pipe.ps1 -hl12            # Show 12H High/Low line only
    .\pipe.ps1 -hl24            # Show 24H High/Low line only
    .\pipe.ps1 -hl72            # Show 72H High/Low line only
    .\pipe.ps1 -s12             # Show 12H sparkline only
    .\pipe.ps1 -s24             # Show 24H sparkline only
    .\pipe.ps1 -s72             # Show 72H sparkline only

Multiple arguments can be combined to display multiple lines:

    .\pipe.ps1 -l -sma          # Show level and SMA lines
    .\pipe.ps1 -b -l -s12       # Show banner, level, and 12H sparkline
    .\pipe.ps1 -hl12 -hl24 -hl72 # Show all three High/Low lines
    .\pipe.ps1 -s12 -s24 -s72   # Show all three sparklines

When command-line arguments are used:
- The screen is NOT cleared (useful for scripts and pipelines)
- Only the requested output lines are displayed
- The script exits after displaying the requested output

When no arguments are provided, the full report is displayed (default behavior).

OUTPUT FORMAT
-------------
The script displays:

1. Header: "*** Portland Big Pipe Report ***" (in green)

2. Statistics (all values aligned at column 16):
   - Current Level: Most recent percentage value (single color-coded value)
   - 100% Duration: Amount of time the pipe has been at 100% capacity (only shown
     when current level is 100%, displayed in magenta). Calculated from first 100%
     reading timestamp to current Pacific time, accounting for data lag and missing samples
   - Last Full: Amount of time since the pipe was last at 100% capacity (only shown
     when current level is less than 100% but there was a previous 100% reading,
     displayed in magenta). Calculated from last 100% reading timestamp to current
     Pacific time
   - 12/24/72H SMA: Three simple moving averages displayed as "$x/$y/$z"
     (each value color-coded separately)
   - 12H High/Low: Maximum and minimum of the last 12 hours as "$x/$y"
     (each value color-coded separately)
   - 24H High/Low: Maximum and minimum of the last 24 hours as "$x/$y"
     (each value color-coded separately)
   - 72H High/Low: Maximum and minimum of the last 72 hours as "$x/$y"
     (each value color-coded separately)

3. Sparklines: Three visual graphs showing historical trends:
   - 12H: Last 12 hours (30-minute bins, 24 glyphs)
   - 24H: Last 24 hours (1-hour bins, 24 glyphs)
   - 72H: Last 72 hours (3-hour bins, 24 glyphs)
   - Missing samples are automatically interpolated to create continuous sparklines

COLOR CODING
------------
All percentage values (statistics and sparkline glyphs) are color-coded based 
on the fill level:

    <= 5%    : White   (very low)
    > 5-20%  : Green   (low)
    > 20-50% : Cyan    (moderate)
    > 50-80% : Yellow  (high)
    > 80-95% : Red     (very high)
    > 95%    : Magenta (critical)

Labels are always displayed in white. The header is displayed in green.

DATA SOURCE
-----------
The script fetches data from:
https://www.portlandoregon.gov/bes/bigpipe/data.cfm

Data is updated every 15 minutes and may have up to a 45-minute lag time.

TECHNICAL DETAILS
-----------------
- Data Parsing: Uses regex to extract data from HTML table structure
- Time Range: Processes up to 72 hours of historical data
- Timezone Handling: All time calculations use Pacific timezone, automatically
  converting from system timezone if needed
- Sparklines: Uses Unicode block characters (U+2581 through U+2588) to create
  visual bar charts
- Time-Based Binning: Data is grouped into fixed time intervals for sparkline display:
  * 12H view: 30-minute time bins (24 bins covering 12 hours)
  * 24H view: 1-hour time bins (24 bins covering 24 hours)
  * 72H view: 3-hour time bins (24 bins covering 72 hours)
- Missing Data Handling: Missing samples are automatically interpolated using linear
  interpolation between adjacent known values, ensuring continuous sparklines without gaps
- 100% Duration Calculation: Uses first 100% reading timestamp to current Pacific time,
  correctly handling missing samples and data lag

ERROR HANDLING
--------------
The script will exit with an error if:
- The website is unreachable
- No data points can be parsed from the page
- Network connectivity issues occur

ERROR CODES
-----------
Exit code 1: Failed to parse data or reach data source

NOTES
-----
- The script requires an active internet connection
- Data availability depends on the City of Portland's website
- Sparklines are displayed using Unicode characters that may not render correctly
  in all terminals or fonts
- The script clears the screen before displaying results (only when no command-line
  arguments are provided)
- Command-line arguments are useful for scripting and integration with other tools

VERSION
-------
v2.3 - Enhanced Statistics Display with Command-Line Arguments

Current version includes:
- HTML table parsing
- Color-coded statistics and sparklines
- Aligned output formatting
- Multiple time range views (12H, 24H, 72H)
- Combined statistics display format:
  * 12/24/72H SMA on single line with three color-coded values
  * High/Low statistics for 12H, 24H, and 72H time ranges
  * Each percentage value individually color-coded
- Enhanced statistics calculations:
  * 24H SMA (Simple Moving Average)
  * 72H SMA (Simple Moving Average)
  * 12H High/Low (max/min of last 12 hours)
  * 24H High/Low (max/min of last 24 hours)
  * 72H High/Low (max/min of last 72 hours, updated from entire dataset)
  * 100% Duration tracking (calculates time at full capacity using Pacific timezone,
    based on first 100% reading timestamp to current Pacific time, handles missing samples)
  * Last Full tracking (calculates time since last 100% reading when pipe is not at
    full capacity, using Pacific timezone)
- Time-based sparkline generation:
  * Uses fixed time intervals (30 min for 12H, 1 hour for 24H, 3 hours for 72H)
  * Handles missing samples correctly by binning based on timestamps
  * Interpolates missing data to create continuous sparklines without gaps
  * All time calculations use Pacific timezone for accuracy
- Command-line arguments for selective output:
  * -banner / -b: Display banner line
  * -level / -l: Display current level line
  * -capacity / -cap: Display 100% Duration line (when at 100%)
  * -lastfull / -lf: Display Last Full line (when not at 100% but was previously)
  * -sma: Display 12/24/72H SMA line
  * -hl12, -hl24, -hl72: Display individual High/Low lines
  * -s12, -s24, -s72: Display individual sparklines
  * Multiple arguments can be combined for custom output

AUTHOR
------
Script for monitoring Portland Big Pipe system data.
