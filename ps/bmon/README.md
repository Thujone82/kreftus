# bmon — Lightweight Bitcoin Monitor

**Version:** 1.5 · **Author:** Kreft&Gemini[Gemini 2.5 Pro (preview)] · **Date:** 2025-08-07

## Description

`bmon.ps1` is a spinoff of the vBTC trading simulator, designed to be a fast and lightweight, real-time Bitcoin price monitor. It operates directly from the command line.

On first run, if no API key is found, it will guide the user through a one-time setup process. The script checks for an API key in this order:

1. `bmon.ini` (in the same directory)
2. `vbtc.ini` (as a fallback for users of the vBTC application)

The script has two primary modes of operation: real-time monitoring and on-demand currency conversion.

## Features

- **Onboarding:** If no API key is found, the script will prompt the user to enter one, which is then saved to `bmon.ini` for future use.
- **Interactive Mode:** A full-screen display that shows the current price. Users can start and pause a 5-minute monitoring session with the space bar. The `r` key (or Right arrow) resets the session baseline. Press `G` to jump directly into Go mode from the landing screen. Left arrow extends the session timer (same as E).
- **Go Mode:** A non-interactive mode (`-go` switch) that displays a single, updating line of price data for 15 minutes before automatically exiting. Ideal for quick glances or integration into other displays.
- **Long Go Mode:** A variation of Go Mode (`-golong` switch) for extended, low-intensity monitoring over 24 hours with a 20-second update interval.
- **K Mode:** Use the `-k` switch for 30-minute monitoring with 4-second updates, the history sparkline enabled, and window coloring (range-colored spinner) enabled automatically. Press `K` or Up arrow (in go/golong) to switch into K mode.
- **Currency Conversion:** Perform quick conversions directly from the command line. The script outputs only the resulting value, making it easy to use in other scripts.
- **Live Price Tracking:** During a monitoring session, the script tracks the price change from the moment monitoring began.
- **Dynamic Mode Toggling:** While in `-go` or `-golong` mode, press `m` or **Down arrow** to toggle between the two modes, resetting the duration timer. Press `K` or **Up arrow** to switch to K mode (sparkline + range coloring). Press `I` to switch back to interactive mode.
- **Audible Alerts:** Optionally enable sound with the `s` key to get high/low tones for every price change of at least $0.01.
- **Sparkline History:** Toggle a mini-chart with the `h` key to visualize the last 14 price samples.
- **Window Coloring:** In go/golong/k single-line modes, when the sparkline is active, the spinner color reflects sparkline volatility (max − min of recent prices). Enable at launch with `-range` / `-r`, auto-enabled with `-k`, or toggle during monitoring with `w`.
- **Command-line Toggles:** Start with sound (`-s`), sparkline history (`-h`), or range-colored spinner (`-range` / `-r`) enabled from the command line.
- **Configuration Menu:** Use the `-config` switch to open the configuration menu. If settings already exist, the current config file path and a masked API key are displayed. You can enter a new LiveCoinWatch API key (validated and saved to `bmon.ini`) or press Enter to keep the current setting and exit.
- **vBTC Integration:** Can seamlessly use the API key configured in `vbtc.ini` if `bmon.ini` is not present, requiring no extra setup for existing vBTC users.
- **Compact Retry Indicator:** In go/golong/k modes, temporary API/network failures no longer print long warnings. Instead, the spinner position shows a single digit for each retry: yellow `1`, `2`, `3`, `4`, and a red `5` on the final attempt. On a successful fetch, the spinner returns to normal immediately, keeping the display clean.

## Color Coding

The application uses colors to provide quick visual feedback during a monitoring session.

### Price line (session change from baseline)

- **Green:** Price has risen by at least $0.01 since the session started.
- **Red:** Price has fallen by at least $0.01 since the session started.
- **White:** Price has not changed significantly or when monitoring is paused.

### UI and fetch

- **Yellow:** Titles and prompts.
- **Cyan:** UI key hints and API fetch in progress (spinner turns cyan during fetch, overriding window coloring).

### Spinner window coloring (go/golong/k modes, sparkline active)

Based on sparkline range (max − min of the last 14 prices):

| Range (USD) | Spinner color |
| ----------- | ------------- |
| Under $10 | White |
| $10 – $49.99 | Green |
| $50 – $99.99 | Yellow |
| $100 – $249.99 | Red |
| $250 or more | Magenta |

Window coloring requires both the sparkline and window coloring to be enabled. Press `w` to toggle window coloring; press `h` to toggle the sparkline. Cyan always wins during an API fetch.

## Requirements

- PowerShell
- An internet connection
- A LiveCoinWatch API key (the script will prompt for one on first run if not found)

## How to Run

1. Open a PowerShell terminal.
2. Navigate to the directory where `bmon.ps1` is located.
3. Run the script using one of the formats below.

### Help

```powershell
.\bmon.ps1 -Help
# or
bmon.exe -Help
```

Show usage and exit.

### Configuration

**Configuration menu:**

```powershell
.\bmon.ps1 -config
# or
bmon.exe -config
```

Opens the configuration menu. If an API key is already configured, the current config file and a masked API key are shown. Enter a new API key to save to `bmon.ini`, or press Enter to exit without changes.

### Monitoring Modes

**Interactive mode:**

```powershell
.\bmon.ps1 [-s] [-h] [-range]
# or
bmon.exe [-s] [-h] [-range]
```

- Press **Spacebar** to start/pause monitoring.
- Press **G** to start Go mode immediately.
- Press **r** or **Right arrow** to reset the session baseline.
- Press **e** or **Left arrow** to extend the session timer (same as E).
- Press **h** to toggle the price history sparkline.
- Press **w** to toggle window coloring (stored for go/golong/k; no spinner in interactive view).
- Press **s** to toggle sound alerts on/off.
- Press **Ctrl+C** or **Esc** to exit.

**Go / GoLong / K modes (non-interactive):**

```powershell
.\bmon.ps1 -go [-s] [-h] [-range]    # 15-minute session, 5-second updates
.\bmon.ps1 -g [-s] [-h] [-range]     # alias

.\bmon.ps1 -golong [-s] [-h] [-range]   # 24-hour session, 20-second updates
.\bmon.ps1 -gl [-s] [-h] [-range]       # alias

.\bmon.ps1 -k [-s]                   # 30-minute session; sparkline + range coloring
.\bmon.ps1 -go -h -range             # 15-minute with sparkline and window coloring
```

The script will run for the specified duration and then exit.

- Press **r** or **Right arrow** to reset the session baseline to the current price.
- Press **e** or **Left arrow** to extend the current session timeout (15 min, 24 hr, or 30 min depending on mode) without changing the comparison baseline.
- Press **m** or **Down arrow** to toggle between `-go` and `-golong` modes.
- Press **k** or **Up arrow** to switch to K mode (30 min, sparkline + range coloring).
- Press **I** to switch back to interactive mode.
- Press **h** to toggle the price history sparkline.
- Press **w** to toggle window coloring (range-colored spinner).
- Press **s** to toggle sound alerts on/off.
- Press **Ctrl+C** or **Esc** to exit early.

### Conversion Mode

Use the following parameters to perform a conversion. The script will output only the result.

| Conversion | Command | Example |
| ---------- | ------- | ------- |
| Bitcoin to USD | `.\bmon.ps1 -bu <amount_in_btc>` | `.\bmon.ps1 -bu 0.5` |
| USD to Bitcoin | `.\bmon.ps1 -ub <amount_in_usd>` | `.\bmon.ps1 -ub 100` |
| USD to Satoshis | `.\bmon.ps1 -us <amount_in_usd>` | `.\bmon.ps1 -us 50` |
| Satoshis to USD | `.\bmon.ps1 -su <amount_in_sats>` | `.\bmon.ps1 -su 250000` |

## Files

| File | Purpose |
| ---- | ------- |
| `bmon.ps1` | Main script |
| `bmon.ini` | API key configuration (created on first run) |
| `vbtc.ini` | Optional fallback API key from vBTC |
| `README.md` | User documentation (source) |
| `README.html` | In-browser markdown viewer |
| `GEMINI.md` | Internal project reference for AI assistants |
