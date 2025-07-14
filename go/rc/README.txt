# rc (Go Version) - Run Continuously

## Version 1.2

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

## Examples

### Example 1: Run a command every 10 minutes
```sh
./rc -period 10 "gw Portland"
```

### Example 2: Run a script with high precision every minute
```sh
./rc -p -period 1 ".\my-data-logger.ps1"
```

### Example 3: Run in interactive mode
```sh
./rc
```