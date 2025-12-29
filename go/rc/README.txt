# rc - RunContinuously v1 (RC v1)

## Version 1.4

## Author
Kreft&Gemini

## Description
`rc` is a cross-platform command-line utility written in Go that executes a given command string in a loop at a specified interval. It is a direct port of the `rc.ps1` PowerShell script, designed to be a lightweight, dependency-free executable.

The application can run in two modes: a standard mode that waits for a fixed duration after a command completes, and a high-precision mode that maintains a strict execution schedule, preventing timing drift over long periods.

## Features
- **Command Execution:** Runs any command string that can be executed by the system's shell (`cmd.exe` on Windows, `sh` on Linux/macOS).
- **Configurable Interval:** Set the wait period between executions in minutes using the `-period` flag.
- **Two Timing Modes:**
  - **Standard Mode (Default):** Waits for the full period *after* the command has finished executing. Simple and straightforward.
  - **Precision Mode (`-p`):** Accounts for the command's execution time to ensure each new run starts on a fixed, predictable schedule. Ideal for tasks requiring consistent timing.
- **Silent Mode (`-s`):** Suppresses status output messages such as execution timing and wait periods, while still displaying the actual command output and any errors. Perfect for logging scenarios.
- **Clear Mode (`-c`):** Clears the screen before executing the command in each iteration, providing a clean output display for each run. Useful for monitoring scenarios where you want to see only the current command output.
- **Skip Mode (`-skip`):** Allows you to skip a specified number of initial executions before starting to run the command. If `-skip 0` is specified, it defaults to 1 (skips the first execution). Useful for delaying the start of command execution while maintaining the timing schedule.
- **Interactive Mode:** If run without any arguments, `rc` will interactively prompt you for the command, period, and timing mode.
- **Cross-Platform:** The included `build.ps1` script compiles native executables for both Windows and Linux.
- **Color-coded Output:** Provides clear, colorized feedback for execution status and timing information.

## Requirements
- Go (for building from source). The compiled executable has no external dependencies.

## How to Run
1.  (Optional) Use the `build.ps1` script to compile the executable for your platform.
2.  Open a terminal or command prompt.
3.  Navigate to the directory where the `rc` executable is located.
4.  Run the application using one of the formats below.

## Command-Line Flags

- `[command]` (Positional Argument)
  - The command string to execute. This should be the last argument provided to the application.
  - If the command contains spaces, it must be enclosed in quotes.

- `-period [minutes]`
  - The time to wait between command executions, in minutes. (Default: 5)

- `-p`, `-precision`
  - A switch to enable "Precision Mode".

- `-s`, `-silent`
  - A switch to enable "Silent Mode".

- `-c`, `-clear`
  - A switch to enable "Clear Mode".
  - When enabled, clears the screen before executing the command in each iteration.

- `-skip <number>`
  - The number of initial executions to skip before starting to run the command.
  - If `-skip 0` is specified, it defaults to 1 (skips the first execution).
  - If `-skip` is not specified at all, no executions are skipped (default is 0).
  - For example, `-skip 2` will skip the first and second executions, then start executing from the third iteration onwards.

## Examples

### Example 1: Run a command every 10 minutes
```sh
./rc -period 10 "gw Portland"
```

### Example 2: Run a script with high precision every minute
```sh
./rc -p -period 1 ".\my-data-logger.ps1"
```

### Example 3: Run in silent mode
```sh
./rc -s -period 1 "date"
```

### Example 4: Combined precision and silent modes
```sh
./rc -p -s -period 5 "my-monitor.sh"
```

### Example 5: Run with clear mode
```sh
./rc -c -period 1 "date"
```
Runs 'date' every minute with the screen cleared before each execution, providing a clean output display for monitoring.

### Example 6: Run in interactive mode
```sh
./rc
```

### Example 7: Skip Mode
```sh
./rc -skip 2 -period 5 "Get-Process"
```
Runs 'Get-Process' every 5 minutes, but skips the first 2 executions. Execution will begin on the 3rd iteration. The timing schedule is maintained during skipped executions.

### Example 8: Skip with Default (Skip 1)
```sh
./rc -skip 0 -period 1 "date"
```
Runs 'date' every minute, but skips the first execution. Since `-skip 0` was specified, it defaults to 1. Execution will begin on the 2nd iteration.