# Gemini Project File

## Project: gw (Get Weather) - Go Version

**Author:** Kreft&Gemini[Gemini 2.5 Pro (preview)]
**Date:** 2025-08-02
**Version:** 1.0

---

### Description

This project is a Go implementation of the `gw` (Get Weather) command-line utility, originally written in PowerShell. It provides detailed, real-time weather information for a specified location by fetching data from the OpenWeatherMap One Call API 3.0.

The application is designed to be a cross-platform equivalent of its PowerShell counterpart, offering the same core functionality. It accepts US zip codes or "City, State" strings, handles first-time API key setup, and presents the weather data in a color-coded, easy-to-read format directly in the terminal.

### Key Functionality

- **Cross-Platform:** Written in Go, it can be compiled and run on Windows, macOS, and Linux.
- **API Key Management:** On the first run, it interactively prompts the user for an OpenWeatherMap API key, validates it, and saves it to a `gw.ini` file in the appropriate user configuration directory for the host OS.
- **Flexible Location Input:** Geocodes locations from either a 5-digit US zip code or a "City, State" formatted string.
- **Concurrent API Calls:** Uses goroutines to fetch detailed weather data and the descriptive weather overview concurrently, improving performance.
- **Comprehensive Data Display:** Outputs current temperature, high/low forecast, humidity, UV Index, wind speed/gusts, sunrise/sunset times, moon phase, and a detailed text report.
- **Color-Coded Output:** Important metrics like temperature, wind speed, and UV index are colored to quickly draw attention to notable or potentially hazardous conditions.
- **Weather Alerts:** Automatically displays any active weather alerts for the given location.
- **Terse Mode (`-t`):** A command-line flag to show a simplified, less verbose output.
- **Interactive & Scriptable:** Can be run with command-line arguments for scripting or without arguments for an interactive prompt.
- **Smart Exit:** Detects if it's being run in a non-persistent shell (e.g., by double-clicking the executable on Windows) and pauses for user input before closing the window.

### How to Run

1.  **Compile:** Open a terminal in the project directory and run:
    ```sh
    go build gw.go
    ```
2.  **Execute:** Run the compiled binary with a location.

**Examples:**
- **By Zip Code (Windows):** `.\gw.exe 97219`
- **By Zip Code (Linux/macOS):** `./gw 97219`
- **By City, State:** `./gw "Portland, OR"`
- **Terse Mode:** `./gw -t "Portland, OR"`
- **Help:** `./gw -h`

### Dependencies

- Go programming language
- External Go Modules:
  - `github.com/fatih/color` (for colored console output)
  - `github.com/shirou/gopsutil/v3/process` (to detect parent process for smart exit)
  - `gopkg.in/ini.v1` (for managing the `gw.ini` configuration file)

### File Structure

- `gw.go`: The main Go source code for the application.
- `go.mod` / `go.sum`: Go module files defining dependencies.
- `gw.exe` (or `gw`): The compiled executable (after running `go build`).
- `README.txt`: User documentation (shared with the PowerShell version).
