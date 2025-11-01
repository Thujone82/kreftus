# RC.PS1 - Run Continuously Script

## Overview
A flexible PowerShell utility designed to execute a given command repeatedly at a specified interval. It is ideal for simple, recurring tasks, monitoring, or any scenario where a command needs to be run in a loop without the complexity of setting up a formal scheduled task.

## Features

### 🔄 Continuous Execution
- **Infinite Loop**: Runs any valid PowerShell command string in an infinite loop until manually stopped
- **Configurable Interval**: The time between executions can be easily set in minutes
- **Interactive Mode**: If run without parameters, the script will prompt for the command and interval
- **Manual Stop**: Press Ctrl+C to stop the script at any time

### ⚙️ Execution Modes

#### Standard Mode (Default)
- After a command finishes, the script waits for the specified interval before the next run
- Simple but can lead to timing "drift" if the command's execution time varies
- Timer starts after the command completes

#### Precision Mode (`-p`)
- Establishes a fixed-interval "grid" based on the script's start time
- Accounts for the command's execution time to ensure each new run starts at a predictable, precise moment (e.g., exactly every 10 minutes at :00, :10, :20, etc.)
- If a command runs longer than its interval, the script will immediately start the next iteration to get back on schedule
- Prevents timing drift by aligning to a grid

#### Silent Mode (`-s`)
- Suppresses status output messages such as execution timing and wait periods
- Still displays the actual command output and any errors
- Ideal for logging or when you only want to see the command results

#### Clear Mode (`-c`)
- Clears the screen before executing the command in each iteration
- Provides a clean output display for each run
- Useful for monitoring scenarios where you want to see only the current command output

## Technical Details

### Parameters
- **Command** [string] (Positional: 0)
  - The PowerShell command to execute on each iteration
  - If the command contains spaces, it must be enclosed in quotes
  - Required (will be prompted for if not provided)

- **Period** [int] (Positional: 1)
  - The time to wait between command executions, in minutes
  - Default value is 5

- **-Precision** or **-p** [switch]
  - Enables "Precision Mode"
  - Uses a fixed-interval schedule to prevent timing drift

- **-Silent** or **-s** [switch]
  - Enables "Silent Mode"
  - Suppresses status output messages while preserving command output and errors

- **-Clear** or **-c** [switch]
  - Enables "Clear Mode"
  - Clears the screen before executing the command in each iteration

### PowerShell Features
- **Version**: PowerShell 5.1+ compatible
- **Execution Policy**: Bypass recommended
- **Error Handling**: Try-catch blocks for command execution errors
- **Screen Clearing**: Uses `Clear-Host` cmdlet for screen clearing functionality
- **Timing**: Uses `Get-Date` and `Start-Sleep` for interval management

## Usage Examples

### Basic Execution
```powershell
.\rc.ps1 "Get-Process -Name 'chrome' | Stop-Process -Force" 1
```
This command will attempt to stop all 'chrome' processes every 1 minute.

### Running Another Script
```powershell
.\rc.ps1 "gw Portland" 10
```
Runs the `gw.ps1` script with its own parameter "Portland" every 10 minutes.

### High-Precision Logging
```powershell
.\rc.ps1 ".\my-data-logger.ps1" 10 -Precision
```
Runs 'my-data-logger.ps1' on a fixed 10-minute schedule. If the script starts at 10:00:00 and the logger takes 20 seconds to run, `rc.ps1` will calculate the remaining time and sleep, ensuring the next run starts at exactly 10:10:00.

### Silent Mode
```powershell
.\rc.ps1 "Get-Date" 1 -Silent
```
Runs 'Get-Date' every minute in silent mode, suppressing status messages while still showing the date output.

### Clear Mode
```powershell
.\rc.ps1 "Get-Date" 1 -Clear
```
Runs 'Get-Date' every minute with the screen cleared before each execution, providing a clean output display for monitoring.

### Combined Modes
```powershell
.\rc.ps1 ".\my-monitor.ps1" 5 -Precision -Silent
```
Runs 'my-monitor.ps1' every 5 minutes with both precision timing and silent output, ideal for background monitoring tasks.

### Interactive Mode
```powershell
.\rc.ps1
```
When run without parameters, the script will prompt for:
- Command to execute
- Period in minutes (default: 5)
- Precision Mode (y/n, default: n)

## Technical Implementation

### Precision Mode Algorithm
- Calculates total elapsed minutes since script start
- Determines number of intervals completed using floor division
- Calculates next target time based on grid alignment
- Adjusts sleep time to account for command execution duration

### Clear Mode Implementation
- Uses PowerShell's `Clear-Host` cmdlet before each command execution
- Executed conditionally when `-Clear` switch is present
- Provides clean output for each iteration

### Error Handling
- Commands are executed in try-catch blocks
- Errors are displayed as warnings without stopping the loop
- Script continues running even if individual commands fail

## Color Scheme

| Color | Usage | Example |
|-------|-------|---------|
| `Yellow` | Script title | "*** Run Continuously v1 ***" |
| `Cyan` | Precision mode messages | Precision mode status messages |
| Default | Status messages | Execution timing, wait periods |

## Requirements

- PowerShell 5.1 or later
- Windows operating system
- No additional dependencies
- No API keys required

## Use Cases

- **Process Monitoring**: Continuously monitor processes or services
- **Logging**: Run logging scripts at regular intervals
- **Data Collection**: Collect data from systems at fixed intervals
- **Cleanup Tasks**: Run cleanup commands periodically
- **System Checks**: Perform system health checks on a schedule
- **Display Updates**: Show updated information with clean screen display

## Notes

- To stop the script, press `Ctrl+C` in the terminal window where it is running
- Precision mode ensures accurate scheduling but may run immediately if a command exceeds its interval
- Clear mode provides a clean display but may not be suitable for logging scenarios where you need to see history
- Silent mode suppresses all status messages but still shows command output and errors

## Version History

- **v1.3**: Added Clear Mode (`-c` switch) for screen clearing functionality before each command execution
- **v1.0**: Initial release with continuous execution, precision mode, and silent mode

