# bmon — Bitcoin Monitor (Go Edition)

**Version:** 1.5 · **Author:** Kreft&Gemini · **Date:** 2025-08-07

## Description

`bmon` is a cross-platform command-line Bitcoin price monitoring utility written in Go. It is a direct port of the `bmon.ps1` PowerShell script, designed to be a lightweight, dependency-free executable that provides real-time Bitcoin price monitoring with visual indicators and conversion tools.

The application can run in several modes: interactive monitoring with keyboard controls, timed single-line monitoring sessions, and quick conversion tools for BTC/USD and satoshi calculations.

## Features

- **Real-time Price Monitoring:** Fetches live Bitcoin prices from LiveCoinWatch API
- **Multiple Monitoring Modes:**
  - **Interactive Mode:** Press Space to start/pause, R to reset, Ctrl+C or Esc to exit. Press G on the landing screen to jump directly into Go mode.
  - **Go Mode:** 15-minute monitoring with 5-second updates
  - **Long Go Mode:** 24-hour monitoring with 20-second updates
  - **K Mode (`-k`):** 30-minute monitoring with 4-second updates; sparkline and range coloring enabled by default
  - **K Long Run (`-kl`):** K mode for 30 minutes, then continues in golong for 24 hours
- **Visual Indicators:** Color-coded price changes (green for gains, red for losses)
- **Price Flash Alerts:** Visual flashing when significant price movements occur
- **Sound Alerts:** Optional audio notifications for price movements
- **Historical Sparkline:** Visual price trend display using Unicode characters (last 14 samples)
- **Window Coloring:** In go/golong/k single-line modes, the spinner color reflects sparkline volatility (max − min). Enable with `-range` / `-r`, auto-on with `-k`, toggle with `W` during monitoring
- **Conversion Tools:** BTC to USD, USD to BTC, USD to satoshis, satoshis to USD
- **API Key Management:** Automatic setup and configuration file handling
- **Configuration Menu:** Use the `-config` flag to open the configuration menu. If settings already exist, the current config file path and a masked API key are displayed. You can enter a new API key (validated and saved to `bmon.ini`) or press Enter to keep the current setting and exit.
- **Cross-Platform:** Native executables for Windows and Linux
- **Color-coded Output:** Clear, colorized feedback for all operations
- **Compact Retry Indicator:** During temporary network/API hiccups in go/golong/k modes, the spinner is briefly replaced with a single digit to indicate retries: yellow `1`, `2`, `3`, `4`, and a red `5` on the final attempt. On the next successful fetch the indicator disappears and the normal spinner resumes.

## Color Coding

### Price line (session change from baseline)

- **Green:** Price has risen by at least $0.01 since the session started.
- **Red:** Price has fallen by at least $0.01 since the session started.
- **White:** No significant change or paused state.

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

Window coloring requires both the sparkline and window coloring to be enabled. Press `W` to toggle window coloring; press `H` to toggle the sparkline.

## Requirements

- Go (for building from source). The compiled executable has no external dependencies.
- LiveCoinWatch API key (free from [livecoinwatch.com/tools/api](https://www.livecoinwatch.com/tools/api))

## How to Run

1. (Optional) Use the `build.ps1` script to compile the executable for your platform.
2. Open a terminal or command prompt.
3. Navigate to the directory where the `bmon` executable is located.
4. Run the application using one of the formats below.

## Command-Line Flags

### Monitoring Modes

| Flag | Description |
| ---- | ----------- |
| `-go` or `-g` | Monitor for 15 minutes with 5-second updates |
| `-golong` or `-gl` | Monitor for 24 hours with 20-second updates |
| `-k` | K mode: 30-minute monitoring; sparkline and range coloring enabled |
| `-kl` | K long run: 30-minute K, then 24-hour golong |
| `-range` or `-r` | Enable range-colored spinner (window coloring) |
| `-s` | Enable sound alerts |
| `-h` | Enable history sparkline |

### Configuration

- `-config` — Open the configuration menu. If an API key is already configured, the current config file and a masked API key are shown. Enter a new API key to save to `bmon.ini`, or press Enter to exit without changes.

### Other

- `-help` — Show usage and exit

### Conversion Tools

| Flag | Description |
| ---- | ----------- |
| `-bu <amount>` | Convert Bitcoin amount to USD |
| `-ub <amount>` | Convert USD amount to Bitcoin |
| `-us <amount>` | Convert USD amount to satoshis |
| `-su <amount>` | Convert satoshi amount to USD |

### Controls (during monitoring)

Letter keys and arrow-key aliases:

| Key | Action |
| --- | ------ |
| `R` or **Right arrow** | Reset baseline price and timer |
| `E` or **Left arrow** | Extend session timeout without changing comparison baseline |
| `M` or **Down arrow** | Switch between go/golong modes |
| `K` or **Up arrow** | Switch to K mode (30 min, sparkline + range coloring) |
| `I` | Switch back to interactive mode (from go/golong/k) |
| `S` | Toggle sound alerts |
| `H` | Toggle history sparkline |
| `W` | Toggle window coloring (go/golong/k single-line modes) |
| `Esc` or `Ctrl+C` | Quit |

## Examples

### Interactive monitoring

```sh
./bmon
```

### Monitor for 15 minutes with sound

```sh
./bmon -go -s
# or
./bmon -g -s
```

### K mode with default sparkline and window coloring

```sh
./bmon -k
```

### K long run (K then golong)

```sh
./bmon -kl
```

### Go mode with sparkline and window coloring

```sh
./bmon -go -h -range
```

### Convert 0.5 BTC to USD

```sh
./bmon -bu 0.5
```

### Convert $50,000 to BTC

```sh
./bmon -ub 50000
```

### Convert $100 to satoshis

```sh
./bmon -us 100
```

### Convert 1M satoshis to USD

```sh
./bmon -su 1000000
```

## Configuration

The application automatically creates and manages configuration files:

- `bmon.ini` — Primary configuration file (created in executable directory)
- `vbtc.ini` — Fallback configuration file (if bmon.ini not found)

On first run, the application will guide you through API key setup. Use `-config` at any time to open the configuration menu, view the current config file and masked API key (if set), and optionally enter a new API key to save.

## Files

| File | Purpose |
| ---- | ------- |
| `main.go` | Application source |
| `README.md` | User documentation (source) |
| `README.html` | In-browser markdown viewer |
| `GEMINI.md` | Internal project reference for AI assistants |
