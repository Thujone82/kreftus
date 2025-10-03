'# rc - Run Continuously (RC v1)

## Version 1.3

## Author
Kreft&Gemini

## Date
2025-07-26

## Description
`rc.ps1` (Run Continuously) is a flexible PowerShell utility designed to execute a given command repeatedly at a specified interval. It is ideal for simple, recurring tasks, monitoring, or any scenario where a command needs to be run in a loop without the complexity of setting up a formal scheduled task.

The script offers two modes for scheduling: a simple delay mode and a high-precision grid-based mode for tasks that require exact timing.

## Features
- **Continuous Execution:** Runs any valid PowerShell command string in an infinite loop until manually stopped.
- **Configurable Interval:** The time between executions can be easily set in minutes.
- **Interactive Mode:** If run without parameters, the script will prompt for the command and interval.
- **Standard Mode (Default):** After a command finishes, the script waits for the specified interval before the next run. This is simple but can lead to timing "drift" if the command's execution time varies.
- **Precision Mode (`-p`):** This mode establishes a fixed-interval "grid" based on the script's start time. It accounts for the command's execution time to ensure each new run starts at a predictable, precise moment (e.g., exactly every 10 minutes at :00, :10, :20, etc.). If a command runs longer than its interval, the script will immediately start the next iteration to get back on schedule.
- **Silent Mode (`-s`):** Suppresses status output messages such as execution timing and wait periods, while still displaying the actual command output and any errors. Ideal for logging or when you only want to see the command results.

## Requirements
- PowerShell

## How to Run
1.  Open a PowerShell terminal.
2.  Navigate to the directory where `rc.ps1` is located.
3.  Run the script using one of the example formats below.
4.  To stop the script, press `Ctrl+C`.

## Parameters

- `Command` [string] (Positional: 0)
  - The PowerShell command to execute on each iteration.
  - If the command contains spaces, it must be enclosed in quotes.
  - This parameter is required (will be prompted for if not provided).

- `Period` [int] (Positional: 1)
  - The time to wait between command executions, in minutes.
  - The default value is 5.

- `-Precision` or `-p` [switch]
  - A switch to enable "Precision Mode".
  - When enabled, the script uses a fixed-interval schedule to prevent timing drift.

- `-Silent` or `-s` [switch]
  - A switch to enable "Silent Mode".
  - When enabled, suppresses status output messages while preserving command output and errors.

## Examples

### Example 1: Simple Task
```powershell
.\rc.ps1 "Get-Process -Name 'chrome' | Stop-Process -Force" 1
```
This command will attempt to stop all 'chrome' processes every 1 minute. The 1-minute timer starts *after* the command finishes.

### Example 2: Running Another Script
```powershell
.\rc.ps1 "gw Portland" 10
```
Runs the `gw.ps1` script (assuming it's in the path or current directory) with its own parameter "Portland" every 10 minutes.

### Example 3: High-Precision Logging
```powershell
.\rc.ps1 ".\my-data-logger.ps1" 10 -Precision
```
Runs 'my-data-logger.ps1' on a fixed 10-minute schedule. If the script starts at 10:00:00 and the logger takes 20 seconds to run, `rc.ps1` will calculate the remaining time and sleep, ensuring the next run starts at exactly 10:10:00.

### Example 4: Silent Mode
```powershell
.\rc.ps1 "Get-Date" 1 -Silent
```
Runs 'Get-Date' every minute in silent mode, suppressing status messages while still showing the date output. Perfect for logging scenarios where you only want to see the command results.

### Example 5: Combined Modes
```powershell
.\rc.ps1 ".\my-monitor.ps1" 5 -Precision -Silent
```
Runs 'my-monitor.ps1' every 5 minutes with both precision timing and silent output, ideal for background monitoring tasks.

## Notes
- To stop the script at any time, press `Ctrl+C` in the terminal window where it is running.