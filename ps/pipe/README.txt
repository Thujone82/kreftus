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
- Sparklines: Uses Unicode block characters (U+2581 through U+2588) to create
  visual bar charts
- Binning: Data is averaged into bins for sparkline display:
  * 12H view: 2 samples per glyph (30-minute intervals)
  * 24H view: 4 samples per glyph (1-hour intervals)
  * 72H view: 12 samples per glyph (3-hour intervals)

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
v2.1 - Enhanced Statistics Display with Command-Line Arguments

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
- Command-line arguments for selective output:
  * -banner / -b: Display banner line
  * -level / -l: Display current level line
  * -sma: Display 12/24/72H SMA line
  * -hl12, -hl24, -hl72: Display individual High/Low lines
  * -s12, -s24, -s72: Display individual sparklines
  * Multiple arguments can be combined for custom output

AUTHOR
------
Script for monitoring Portland Big Pipe system data.
