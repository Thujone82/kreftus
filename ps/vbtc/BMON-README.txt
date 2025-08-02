# bmon - Lightweight Bitcoin Monitor

## Version 1.4

## Author
Kreft&Gemini[Gemini 2.5 Pro (preview)]

## Date
2025-08-02

## Description
`bmon.ps1` is a spinoff of the vBTC trading simulator, designed to be a fast and lightweight, real-time Bitcoin price monitor. It operates directly from the command line and leverages the existing configuration from `vbtc.ps1`.

The script has two primary modes of operation: real-time monitoring and on-demand currency conversion.

## Features
- **Interactive Mode:** A full-screen display that shows the current price. Users can start and pause a 5-minute monitoring session with the space bar. The 'r' key can be used to reset the session baseline.
- **Go Mode:** A non-interactive mode (`-go` switch) that displays a single, updating line of price data for 5 minutes before automatically exiting. Ideal for quick glances or integration into other displays.
- **Long Go Mode:** A variation of Go Mode (`-golong` switch) for extended, low-intensity monitoring over 24 hours with a 20-second update interval.
- **Currency Conversion:** Perform quick conversions directly from the command line. The script outputs only the resulting value, making it easy to use in other scripts.
- **Live Price Tracking:** During a monitoring session, the script tracks the price change from the moment monitoring began.
- **Dynamic Mode Toggling:** While in `-go` or `-golong` mode, press 'm' to toggle between the two modes, resetting the duration timer for the newly selected mode.
- **Audible Alerts:** Optionally enable sound with the 's' key to get high/low tones for every price change of at least $0.01.
- **Sparkline History:** Toggle a mini-chart with the 'h' key to visualize the last 8 price ticks.
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
    
### Monitoring Modes
---
-   **Interactive Mode:**
    `.\bmon.ps1` or `bmon.exe`
    - Press `Spacebar` to start/pause monitoring.
    - Press `r` to reset the session baseline.
    - Press `h` to toggle the price history sparkline.
    - Press `s` to toggle sound alerts on/off.
    - Press `Ctrl+C` to exit.
-   **Go / GoLong Modes (Non-Interactive):**
    `.\bmon.ps1 -go` (5-minute session, 5-second updates)
    `.\bmon.ps1 -golong` (24-hour session, 20-second updates)
    The script will run for the specified duration and then exit.
    - Press `r` to reset the session baseline to the current price.
    - Press `m` to toggle between -go and -golong modes.
    - Press `h` to toggle the price history sparkline.
    - Press `s` to toggle sound alerts on/off.
    - Press `Ctrl+C` to exit early.
### Conversion Mode
---
Use the following parameters to perform a conversion. The script will output only the result.
-   **Bitcoin to USD:**
    `.\bmon.ps1 -bu <amount_in_btc>`
    Example: `.\bmon.ps1 -bu 0.5`
-   **USD to Bitcoin:**
    `.\bmon.ps1 -ub <amount_in_usd>`
    Example: `.\bmon.ps1 -ub 100`
-   **USD to Satoshis:**
    `.\bmon.ps1 -us <amount_in_usd>`
    Example: `.\bmon.ps1 -us 50`
-   **Satoshis to USD:**
    `.\bmon.ps1 -su <amount_in_sats>`
    Example: `.\bmon.ps1 -su 250000`

## Files
-   `bmon.ps1`: The main script file.
-   `vbtc.ini`: The configuration file shared with `vbtc.ps1`, which must contain the API key.