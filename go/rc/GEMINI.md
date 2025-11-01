# RC - Run Continuously (Go Edition)

## Overview
A cross-platform command-line utility written in Go that executes a given command string in a loop at a specified interval. It is a direct port of the `rc.ps1` PowerShell script, designed to be a lightweight, dependency-free executable that works on Windows, Linux, and macOS.

## Features

### 🔄 Continuous Execution
- **Infinite Loop**: Runs any command string that can be executed by the system's shell in an infinite loop until manually stopped
- **Configurable Interval**: The time between executions can be easily set in minutes
- **Interactive Mode**: If run without parameters, the application will prompt for the command and interval
- **Manual Stop**: Press Ctrl+C to stop the application at any time
- **Cross-Platform**: Compiles to native executables for Windows and Linux with no external dependencies

### ⚙️ Execution Modes

#### Standard Mode (Default)
- After a command finishes, the application waits for the specified interval before the next run
- Simple but can lead to timing "drift" if the command's execution time varies
- Timer starts after the command completes

#### Precision Mode (`-p` or `-precision`)
- Establishes a fixed-interval "grid" based on the application's start time
- Accounts for the command's execution time to ensure each new run starts at a predictable, precise moment (e.g., exactly every 10 minutes at :00, :10, :20, etc.)
- If a command runs longer than its interval, the application will immediately start the next iteration to get back on schedule
- Prevents timing drift by aligning to a grid

#### Silent Mode (`-s` or `-silent`)
- Suppresses status output messages such as execution timing and wait periods
- Still displays the actual command output and any errors
- Ideal for logging or when you only want to see the command results

#### Clear Mode (`-c` or `-clear`)
- Clears the screen before executing the command in each iteration
- Provides a clean output display for each run
- Uses platform-specific methods: `cls` command on Windows, ANSI escape sequences on Unix-like systems
- Useful for monitoring scenarios where you want to see only the current command output

## Technical Details

### Parameters
- **Command** [string] (Positional: 0)
  - The command string to execute on each iteration
  - If the command contains spaces, it must be enclosed in quotes
  - Required (will be prompted for if not provided)

- **Period** [int] (Positional: 1, or use `-period`)
  - The time to wait between command executions, in minutes
  - Default value is 5

- **-p** or **-precision** [switch]
  - Enables "Precision Mode"
  - Uses a fixed-interval schedule to prevent timing drift

- **-s** or **-silent** [switch]
  - Enables "Silent Mode"
  - Suppresses status output messages while preserving command output and errors

- **-c** or **-clear** [switch]
  - Enables "Clear Mode"
  - Clears the screen before executing the command in each iteration

### Platform Support
- **Windows**: Uses `cmd.exe` for command execution and `cls` command for screen clearing
- **Linux/macOS**: Uses `sh` for command execution and ANSI escape sequences (`\033[2J\033[H`) for screen clearing

### Go Features
- **Version**: Requires Go 1.16+ (for building from source)
- **Dependencies**: Uses `github.com/fatih/color` for colorized output
- **Error Handling**: Graceful error handling with try-catch equivalent behavior
- **Screen Clearing**: Platform-specific implementation using `os/exec` for Windows or ANSI sequences for Unix
- **Timing**: Uses Go's `time` package for interval management and precision timing

## Usage Examples

### Basic Execution
```sh
./rc "go run main.go" 1
```
Runs 'go run main.go' every 1 minute.

### Running Another Script
```sh
./rc "gw Portland" 10
```
Runs the `gw` script with its own parameter "Portland" every 10 minutes.

### High-Precision Logging
```sh
./rc -p -period 10 "./my-data-logger.sh"
```
Runs './my-data-logger.sh' on a fixed 10-minute schedule. If the script starts at 10:00:00 and the logger takes 20 seconds to run, `rc` will calculate the remaining time and sleep, ensuring the next run starts at exactly 10:10:00.

### Silent Mode
```sh
./rc -s -period 1 "date"
```
Runs 'date' every minute in silent mode, suppressing status messages while still showing the date output.

### Clear Mode
```sh
./rc -c -period 1 "date"
```
Runs 'date' every minute with the screen cleared before each execution, providing a clean output display for monitoring.

### Combined Modes
```sh
./rc -p -s -period 5 "./my-monitor.sh"
```
Runs './my-monitor.sh' every 5 minutes with both precision timing and silent output, ideal for background monitoring tasks.

### Interactive Mode
```sh
./rc
```
When run without parameters, the application will prompt for:
- Command to execute
- Period in minutes (default: 5)
- Precision Mode (y/n, default: n)
- Clear Mode (y/n, default: n)

## Technical Implementation

### Precision Mode Algorithm
- Calculates total elapsed minutes since application start
- Determines number of intervals completed using floor division
- Calculates next target time based on grid alignment
- Adjusts sleep time to account for command execution duration

### Clear Mode Implementation
- **Windows**: Executes `cmd /C cls` command to clear the screen
- **Unix/Linux/macOS**: Outputs ANSI escape sequence `\033[2J\033[H` to clear and reset cursor
- Executed conditionally when `-c` or `-clear` flag is present
- Provides clean output for each iteration

### Error Handling
- Commands are executed with error handling
- Errors are displayed as warnings without stopping the loop
- Application continues running even if individual commands fail

## Color Scheme

| Color | Usage | Example |
|-------|-------|---------|
| `Yellow` | Titles and warnings | "*** Run Continuously v1 ***", error messages |
| `Cyan` | Precision mode messages | Precision mode status messages |
| `White` | Status messages | Execution timing, wait periods |

## Requirements

- Go 1.16+ (for building from source)
- No additional dependencies for compiled executables
- Windows or Linux/macOS operating system
- No API keys required

## Building

Use the included `build.ps1` script to compile native executables:
```powershell
.\build.ps1
```

This will create executables in the `bin/` directory for:
- Windows (x64 and x86)
- Linux (amd64 and x86)

## Use Cases

- **Process Monitoring**: Continuously monitor processes or services
- **Logging**: Run logging scripts at regular intervals
- **Data Collection**: Collect data from systems at fixed intervals
- **Cleanup Tasks**: Run cleanup commands periodically
- **System Checks**: Perform system health checks on a schedule
- **Display Updates**: Show updated information with clean screen display

## Notes

- To stop the application, press `Ctrl+C` in the terminal window where it is running
- Precision mode ensures accurate scheduling but may run immediately if a command exceeds its interval
- Clear mode provides a clean display but may not be suitable for logging scenarios where you need to see history
- Silent mode suppresses all status messages but still shows command output and errors
- The compiled executable is platform-specific - use the appropriate binary for your operating system

## Version History

- **v1.3**: Added Clear Mode (`-c` and `-clear` flags) for screen clearing functionality before each command execution
- **v1.0**: Initial release with continuous execution, precision mode, and silent mode

