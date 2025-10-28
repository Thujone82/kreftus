# UPDate Tool - Development Documentation

## Overview

UPDate is a PowerShell-based file update management tool developed to streamline the process of updating files from various sources (local paths or URLs) to configured destination directories. The tool features an interactive CLI interface with color-coded output and support for automated execution via command-line arguments.

## Development History

### Initial Requirements

The tool was developed with the following specifications:
- Interactive command-line interface with color-coded output
- Job-based configuration system for update tasks
- Support for both local file copying and URL downloads
- Multiple job selection capability
- Command-line argument support for automation
- Persistent configuration storage

### Technical Challenges & Solutions

#### 1. Variable Persistence in Recursive Function Calls

**Challenge:** PowerShell's script execution model was resetting the `$script:SelectedJobs` array when functions were called recursively, preventing multiple job selections from persisting.

**Solution:** 
- Implemented conditional variable initialization using `Get-Variable` to check if the variable already exists
- Replaced recursive function calls with a `while ($true)` loop in `Show-UpdateScreen`
- Used explicit array construction `@()` when adding/removing items to ensure proper array type

```powershell
# Initialize only if it doesn't exist
if (-not (Get-Variable -Name "SelectedJobs" -Scope Script -ErrorAction SilentlyContinue)) {
    $script:SelectedJobs = @()
}

# Proper array manipulation
$script:SelectedJobs = @($script:SelectedJobs) + @($Index)
```

#### 2. Command-Line Argument Parsing

**Challenge:** Distinguishing between job names and PowerShell's built-in parameters (like `-Verbose`) when processing command-line arguments.

**Solution:**
- Created a whitelist of PowerShell common parameters to exclude
- Manual argument processing via `$args` to capture job names
- Added explicit `-Verbose` parameter to the script's param block

```powershell
$builtInParams = @('Debug', 'ErrorAction', 'WarningAction', ...)
foreach ($arg in $args) {
    if ($arg -match '^-(.+)$') {
        $paramName = $matches[1]
        if ($paramName -notin $builtInParams) {
            $JobNames += $paramName
        }
    }
}
```

#### 3. Configuration Storage Format

**Challenge:** Initial implementation used `.psd1` (PowerShell Data File) which required `Export-PowerShellDataFile` cmdlet not available in all PowerShell versions.

**Solution:**
- Migrated to JSON format for broader compatibility
- Implemented conversion between hashtables and PSCustomObjects
- Used `ConvertTo-Json` and `ConvertFrom-Json` for serialization

```powershell
# Export: Hashtable -> PSCustomObject -> JSON
$jobsForJson = foreach ($job in $script:Jobs) {
    [PSCustomObject]@{
        Name = $job.Name
        Remote = $job.Remote
        Local = $job.Local
    }
}

# Import: JSON -> PSCustomObject -> Hashtable
foreach ($job in $configData.Jobs) {
    $script:Jobs += @{
        Name = $job.Name
        Remote = $job.Remote
        Local = $job.Local
    }
}
```

#### 4. UI Consistency and Color Coding

**Challenge:** Maintaining consistent color coding across different UI elements while ensuring hotkeys are visually distinct.

**Solution:**
- Referenced existing project tools (`gf.ps1`) for color coding patterns
- Implemented granular `Write-Host` calls with `-NoNewline` for precise control
- Created dedicated color wrapper functions for consistency

```powershell
Write-Host "Enable [" -ForegroundColor White -NoNewline
Write-Host $jobRange -ForegroundColor Cyan -NoNewline
Write-Host "] [" -ForegroundColor White -NoNewline
Write-Host "A" -ForegroundColor Cyan -NoNewline
Write-Host "]ll" -ForegroundColor White
```

## Architecture

### Core Components

1. **Configuration Management**
   - `Import-Config`: Load jobs from JSON file
   - `Export-Config`: Save jobs to JSON file
   - `Initialize-Config`: Create config file if it doesn't exist

2. **Job Management**
   - `Set-JobSelection`: Toggle individual job selection
   - `Select-AllJobs`: Select all configured jobs
   - `Get-JobStatus`: Check if a job is selected
   - `Get-JobByName`: Find job index by name (case-insensitive)

3. **File Operations**
   - `Invoke-UpdateJob`: Execute a single update job
   - `Start-SelectedJobs`: Execute all selected jobs
   - Supports both local file copy and HTTP/HTTPS downloads

4. **UI Components**
   - `Show-UpdateScreen`: Main interactive screen (loop-based)
   - `Show-JobsScreen`: Jobs management screen
   - `Show-Header`: Display colored headers
   - `Read-SingleKey`: Capture single key input

5. **Job Editing**
   - `New-Job`: Create new update job
   - `Edit-Job`: Modify existing job

### Data Flow

```
Command Line Args → Test-CommandLineArgs → Pre-select Jobs
                                         ↓
                                    Auto Execute?
                                    ↙         ↘
                                  Yes          No
                                   ↓           ↓
                          Start-SelectedJobs  Show-UpdateScreen
                                               ↓
                                    Interactive Loop
                                    (User Selection)
                                               ↓
                                    Start-SelectedJobs
```

### State Management

- **Global State Variables:**
  - `$script:Jobs`: Array of job hashtables
  - `$script:SelectedJobs`: Array of selected job indices
  - `$script:ConfigPath`: Path to configuration file

- **Session Persistence:**
  - Job selections persist throughout the interactive session
  - Configuration changes are saved immediately to disk

## Best Practices Applied

1. **PowerShell Cmdlet Naming Conventions**
   - Used approved verbs: `Import`, `Export`, `Start`, `Set`, `Get`
   - Avoided unapproved verbs like `Load`, `Save`, `Toggle`, `Execute`

2. **Error Handling**
   - Try-catch blocks around file operations
   - Validation of user input
   - Graceful handling of missing configuration files

3. **User Experience**
   - Clear visual feedback with color coding
   - Confirmation prompts for destructive operations
   - Verbose mode for troubleshooting
   - Automatic directory creation when needed

4. **Code Organization**
   - Logical function grouping
   - Consistent parameter naming
   - Comprehensive inline comments
   - Separation of UI and business logic

## Testing Scenarios

### Manual Testing Performed

1. **Job Selection:**
   - Single job selection/deselection
   - Multiple job selection
   - Select all functionality
   - Extended selection sequences (10+ toggles)

2. **Command-Line Arguments:**
   - Pre-selection by job name
   - Auto-execution with `-a` flag
   - Verbose mode with `-Verbose`
   - Multiple job names in single command
   - Handling of non-existent job names

3. **File Operations:**
   - Local file copying
   - URL downloads
   - Non-existent source handling
   - Directory creation
   - Permission errors

4. **UI Navigation:**
   - Main screen to Jobs screen
   - Jobs screen to Main screen
   - Job editing workflow
   - New job creation
   - Esc key navigation (exit and return)
   - Job removal workflow

## Dynamic Date Placeholder System

### Overview

The UPDate tool supports dynamic date/time placeholders in both Remote and Local paths, allowing users to specify date format patterns wrapped in square brackets `[...]` that get replaced with the current date/time when jobs execute. This feature enables time-sensitive file operations like downloading weather radar images, accessing timestamped log files, creating time-based backups, and organizing downloads into date-based folder structures.

### Technical Implementation

#### Core Function: `Expand-DatePlaceholders`

```powershell
function Expand-DatePlaceholders {
    param([string]$Path)
    
    $currentDate = Get-Date
    $result = $Path
    
    # Find all [dateformat] patterns
    $pattern = '\[([^\]]+)\]'
    $patternMatches = [regex]::Matches($Path, $pattern)
    
    foreach ($match in $patternMatches) {
        $formatString = $match.Groups[1].Value
        $dateValue = $currentDate.ToString($formatString)
        $result = $result.Replace($match.Value, $dateValue)
    }
    
    return $result
}
```

#### Regex Pattern Analysis

The regex pattern `\[([^\]]+)\]` works as follows:
- `\[` - Matches literal opening bracket `[`
- `([^\]]+)` - Captures one or more characters that are not closing brackets
- `\]` - Matches literal closing bracket `]`

This pattern ensures that:
- Only properly formatted bracket pairs are matched
- Nested brackets are not supported (by design)
- Empty brackets `[]` are not matched (no capture group content)

#### .NET DateTime Format Support

The function leverages .NET's `DateTime.ToString(string format)` method, which supports:

**Standard Format Specifiers:**
- `yyyy` - 4-digit year (2025)
- `yy` - 2-digit year (25)
- `MM` - 2-digit month with leading zero (01-12)
- `M` - 1-2 digit month (1-12)
- `dd` - 2-digit day with leading zero (01-31)
- `d` - 1-2 digit day (1-31)
- `HH` - 2-digit hour 24-hour format (00-23)
- `hh` - 2-digit hour 12-hour format (01-12)
- `mm` - 2-digit minute (00-59)
- `ss` - 2-digit second (00-59)
- `tt` - AM/PM designator

**Custom Format Strings:**
- `[yyyy-MM-dd]` - ISO date format (2025-01-15)
- `[MMddHH]` - Compact date/time (011523)
- `[HH:mm:ss]` - Time format (14:30:45)
- `[yyyyMMdd_HHmmss]` - Full timestamp (20250115_143045)

#### Integration Points

The function is called in `Invoke-UpdateJob` before processing both Remote and Local paths:

```powershell
$remote = Expand-DatePlaceholders -Path $Job.Remote
$local = Expand-DatePlaceholders -Path $Job.Local
```

This ensures that:
- Date expansion occurs once per job execution
- All placeholders in a single job use the same timestamp
- Both Remote and Local paths support dynamic date placeholders
- The expanded paths are used for both URL downloads and local file operations

### Real-World Use Cases

#### Weather Radar Images
```
Remote: http://radar.weather.gov/ridge/RadarImg/N0R/[MMddHH]/[MMddHH]_N0R_0.gif
Expands to: http://radar.weather.gov/ridge/RadarImg/N0R/011523/011523_N0R_0.gif
```

#### Log File Archival
```
Remote: C:\logs\application-[yyyy-MM-dd].log
Expands to: C:\logs\application-2025-01-15.log
```

#### API Data with Timestamps
```
Remote: https://api.example.com/data?date=[yyyyMMdd]&time=[HHmmss]
Expands to: https://api.example.com/data?date=20250115&time=143045
```

#### Backup File Naming
```
Remote: backup-[yyyy-MM-dd_HH-mm-ss].zip
Expands to: backup-2025-01-15_14-30-45.zip
```

#### Monthly Download Organization
```
Remote: https://example.com/monthly-report.pdf
Local: C:\downloads\[yyyy-MM]\
Expands to: C:\downloads\2025-01\
Result: Downloads to monthly folders for organization
```

#### Daily Archive Structure
```
Remote: https://api.example.com/data.json
Local: C:\data\[yyyy-MM-dd]\
Expands to: C:\data\2025-01-15\
Result: Creates daily folders for data organization
```

#### Time-Based Log Archival
```
Remote: C:\logs\application.log
Local: C:\archive\[yyyy-MM]\logs\
Expands to: C:\archive\2025-01\logs\
Result: Archives logs into monthly folders
```

### Performance Considerations

- **Regex Compilation:** The regex pattern is compiled once per function call
- **String Operations:** Multiple string replacements per path (one per placeholder)
- **Date Calculation:** Single `Get-Date` call per job execution
- **Memory Impact:** Minimal - only string manipulation operations

### Error Handling

The function includes implicit error handling through .NET's `DateTime.ToString()` method:
- Invalid format strings throw `FormatException`
- Malformed brackets are ignored (no regex match)
- Empty format strings result in empty replacement

### Testing Scenarios

#### Basic Functionality
- Single placeholder: `[yyyy]` → `2025`
- Multiple placeholders: `[MM]-[dd]-[yyyy]` → `01-15-2025`
- Mixed content: `file-[HHmmss].txt` → `file-143045.txt`

#### Edge Cases
- No placeholders: `static-file.txt` → `static-file.txt` (unchanged)
- Empty brackets: `file[].txt` → `file[].txt` (unchanged)
- Invalid format: `[invalid]` → throws exception
- Special characters: `[yyyy-MM-dd_HH:mm:ss]` → `2025-01-15_14:30:45`

#### Real-World Patterns
- Weather services: `[MMddHH]` for radar images
- Log rotation: `[yyyy-MM-dd]` for daily logs
- API endpoints: `[yyyyMMdd]` for date-based queries
- Backup systems: `[yyyy-MM-dd_HH-mm-ss]` for timestamped archives

### Future Enhancement Possibilities

1. **Timezone Support:** Allow specification of different timezones
2. **Relative Dates:** Support for `[yesterday]`, `[last-week]` patterns
3. **Custom Functions:** User-defined date calculation functions
4. **Validation:** Pre-execution validation of date format strings
5. **Caching:** Cache expanded paths for repeated executions
6. **Logging:** Track which placeholders were expanded and their values

## Recent Updates

### v1.2 - Transfer Size Formatting Enhancement

#### Problem Identified
The original implementation always displayed transfer sizes in MB, which resulted in confusing output for small files. For example, a 5.39 KB file would display as "0 MB", making it appear as if no data was transferred.

#### Solution Implemented
Implemented intelligent transfer size formatting that automatically selects the most appropriate unit based on file size:

```powershell
# Format size with appropriate units
$sizeStr = if ($totalBytesTransferred -ge 1GB) {
    $sizeGB = [math]::Round($totalBytesTransferred / 1GB, 2)
    "$sizeGB GB"
} elseif ($totalBytesTransferred -ge 1MB) {
    $sizeMB = [math]::Round($totalBytesTransferred / 1MB, 2)
    "$sizeMB MB"
} else {
    $sizeKB = [math]::Round($totalBytesTransferred / 1KB, 2)
    "$sizeKB KB"
}
```

#### Technical Details
- **Thresholds**: 1 MB (1,048,576 bytes) and 1 GB (1,073,741,824 bytes)
- **Precision**: 2 decimal places for all units
- **Consistency**: Both transfer size and speed use the same logic
- **Performance**: Minimal overhead - only string formatting operations

#### User Experience Impact
- **Before**: "Transfer: 0 MB in 496ms (5.39 KB/s)" - confusing for small files
- **After**: "Transfer: 5.39 KB in 496ms (5.39 KB/s)" - clear and accurate

#### Testing Performed
- Verified formatting for files ranging from 512 bytes to 2.5 GB
- Confirmed proper unit selection at threshold boundaries
- Validated decimal precision and rounding behavior

### v1.1 - Enhanced Navigation and Job Management

1. **Esc Key Navigation**
   - Main screen: Esc key exits the application
   - Jobs screen: Esc key returns to main screen
   - Provides consistent navigation pattern

2. **Job Removal Functionality**
   - Added "Remove Job? [y/N]" prompt in job editing
   - Yellow-colored prompt for visibility
   - Only 'y' or 'Y' triggers removal, all other inputs continue with editing
   - Removed jobs are immediately saved to configuration

3. **Enhanced Auto Mode (-a)**
   - Removed forced key press requirement
   - Removed screen clear to leave output visible
   - Script completes automatically without user interaction
   - Output remains on screen for review

4. **Improved Output Display**
   - Replaced "Copied:" message with "Source:" and "Destination:" lines
   - Provides clearer information about file operations
   - Better visibility of what was copied and where

5. **Job Name Validation**
   - Enforces strict naming rules for job names
   - Prevents spaces and special characters
   - Ensures compatibility with file systems and command-line usage
   - Provides clear error messages and persistent prompting

### Technical Implementation Details

#### Esc Key Handling
```powershell
# Main screen exit
{ $_ -eq [char]27 } {  # Esc key
    Write-White "Exiting..."
    return
}

# Jobs screen return (already implemented)
{ $_ -eq [char]27 -or $_ -eq [char]13 } {  # Esc or Enter
    Show-UpdateScreen
}
```

#### Job Removal Logic
```powershell
# Ask if user wants to remove the job
Write-Host "Remove Job? [y/N]" -ForegroundColor Yellow -NoNewline
$removeJob = Read-Host " "
if ($removeJob -eq 'y' -or $removeJob -eq 'Y') {
    # Remove the job from the array
    $script:Jobs = $script:Jobs | Where-Object { $_ -ne $script:Jobs[$Index] }
    if (Export-Config) {
        Write-Green "Job '$($job.Name)' removed successfully."
    }
    return
}
```

#### Auto Mode Improvements
```powershell
# Before: Required key press and cleared screen
if ($Auto) {
    Start-SelectedJobs
    Write-Host "Press any key to continue..." -ForegroundColor Yellow
    $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
    Clear-Screen
    return $true
}

# After: Clean completion without interaction
if ($Auto) {
    Start-SelectedJobs
    return $true
}
```

#### Job Name Validation Implementation
```powershell
# Validation function with regex patterns
function Test-JobName {
    param([string]$Name)
    
    if ([string]::IsNullOrWhiteSpace($Name)) {
        return $false
    }
    
    # Check for spaces using \s pattern
    if ($Name -match '\s') {
        return $false
    }
    
    # Check for special characters (allow only alphanumeric, hyphens, underscores)
    if ($Name -match '[^a-zA-Z0-9_-]') {
        return $false
    }
    
    return $true
}

# New-Job function with validation loop
do {
    $name = Read-Host "Name"
    if ([string]::IsNullOrWhiteSpace($name)) {
        Write-Red "Name cannot be empty."
        continue
    }
    if (-not (Test-JobName $name)) {
        Write-Red "Job name must contain only letters, numbers, hyphens, and underscores. No spaces or special characters allowed."
        continue
    }
    break
} while ($true)

# Edit-Job function with conditional validation
do {
    $name = Read-Host "Name [$($job.Name)]"
    if ([string]::IsNullOrWhiteSpace($name)) {
        $name = $job.Name  # Keep original if empty
        break
    }
    if (-not (Test-JobName $name)) {
        Write-Red "Job name must contain only letters, numbers, hyphens, and underscores. No spaces or special characters allowed."
        continue
    }
    break
} while ($true)
```

**Validation Rules:**
- **Allowed Characters:** Letters (a-z, A-Z), numbers (0-9), hyphens (-), underscores (_)
- **Prohibited Characters:** Spaces, special characters (@, #, $, %, etc.)
- **Empty Input:** Handled differently in New-Job (error) vs Edit-Job (keep original)
- **Regex Patterns:** 
  - `\s` detects any whitespace characters
  - `[^a-zA-Z0-9_-]` detects any character not in the allowed set
- **User Experience:** Clear error messages with specific guidance on valid characters

## Future Enhancement Possibilities

1. **Job Reordering:** Allow users to change job display order
2. **Job Categories:** Group jobs into categories for better organization
3. **Scheduled Execution:** Integration with Windows Task Scheduler
4. **Backup Before Update:** Automatic backup of existing files before update
5. **Update History:** Log of executed updates with timestamps
6. **Job Dependencies:** Define job execution order based on dependencies
7. **Wildcard Support:** Allow multiple file updates with wildcards
8. **Progress Indicators:** Show download progress for large files
9. **Configuration Export/Import:** Share job configurations between machines

## Lessons Learned

1. **PowerShell Script Scope:** Script-level variables require careful management in recursive scenarios
2. **Array Type Preservation:** Explicit array construction prevents type coercion issues
3. **Loop vs Recursion:** For UI refresh operations, loops are more predictable than recursion
4. **Parameter Handling:** Manual argument parsing provides more control than automatic binding
5. **Cross-Version Compatibility:** JSON is more portable than PowerShell-specific formats

## Dependencies

- PowerShell 5.1 or higher
- .NET Framework (for `Invoke-WebRequest` and `System.IO.File`)
- Windows operating system (for console UI features)

## File Structure

```
ps/upd/
├── upd.ps1           # Main script
├── upd.json          # Configuration file (auto-generated)
├── README.txt        # User documentation
└── GEMINI.md         # Development documentation (this file)
```

## Performance Considerations

- Job selection operations are O(n) where n is number of jobs
- Configuration save operations occur only on job creation/editing
- No background processes or threading
- Memory footprint scales linearly with number of configured jobs

## Security Considerations

- No credential storage
- Local file operations respect Windows permissions
- URL downloads use PowerShell's built-in web request security
- No code execution from configuration files
- Configuration stored in plain text (JSON)

---

*This tool was developed as part of the Kreftus project, emphasizing clean code, user experience, and maintainability.*

