# bmon - Lightweight Bitcoin Monitor

## Version 1.0

## Author
Kreft&Gemini[Gemini 2.5 Pro (preview)]

## Date
2025-07-06

## Description
`bmon.ps1` is a spinoff of the vBTC trading simulator, designed to be a fast and lightweight, real-time Bitcoin price monitor. It operates directly from the command line and leverages the existing configuration from `vbtc.ps1`.

The script has two primary modes of operation: an interactive session and a non-interactive "go" mode for quick, unattended checks.

## Features
- **Interactive Mode:** A full-screen display that shows the current price. Users can start and pause a 5-minute monitoring session with the space bar.
- **Go Mode:** A non-interactive mode (`-go` switch) that displays a single, updating line of price data for 5 minutes before automatically exiting. Ideal for quick glances or integration into other displays.
- **Live Price Tracking:** During a monitoring session, the script tracks the price change from the moment monitoring began.
- **Dependency on vBTC:** Seamlessly uses the API key configured in `vbtc.ini`, requiring vBTC to be set up first.

## Color Coding
The application uses colors to provide quick visual feedback during a monitoring session:

- **Green:** Indicates the price has risen by at least $0.01 since the session started.
- **Red:** Indicates the price has fallen by at least $0.01 since the session started.
- **White:** Indicates the price has not changed significantly or when monitoring is paused.
- **Yellow / Cyan:** Used for UI elements like titles and prompts for better readability.

## Requirements
- PowerShell
- An internet connection
- A configured `vbtc.ini` file from the vBTC application with a valid LiveCoinWatch API key.

## How to Run
1.  Open a PowerShell terminal.
2.  Navigate to the directory where `bmon.ps1` is located.
3.  Run the script in one of two ways:

    -   **Interactive Mode:**
        `.\bmon.ps1`
        Press the space bar to start/pause monitoring. Press Ctrl+C to exit.

    -   **Go Mode (Non-Interactive):**
        `.\bmon.ps1 -go`
        The script will run for 5 minutes and then exit automatically. Press Ctrl+C to exit early.

## Files
-   `bmon.ps1`: The main script file.
-   `vbtc.ini`: The configuration file shared with `vbtc.ps1`, which must contain the API key.