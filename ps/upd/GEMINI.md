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

## Recent Updates (v1.1)

### New Features Added

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

