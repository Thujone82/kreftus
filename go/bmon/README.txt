# bmon - Bitcoin Monitor v1.5

## Version 1.5

## Author
Kreft&Gemini

## Date
2025-08-07

## Description
`bmon` is a cross-platform command-line Bitcoin price monitoring utility written in Go. It is a direct port of the `bmon.ps1` PowerShell script, designed to be a lightweight, dependency-free executable that provides real-time Bitcoin price monitoring with visual indicators and conversion tools.

The application can run in several modes: interactive monitoring with keyboard controls, timed monitoring sessions, and quick conversion tools for BTC/USD and satoshi calculations.

## Features
- **Real-time Price Monitoring:** Fetches live Bitcoin prices from LiveCoinWatch API
- **Multiple Monitoring Modes:**
  - **Interactive Mode:** Press Space to start/pause, R to reset, Ctrl+C to exit. Press G on the landing screen to jump directly into Go mode.
  - **Go Mode:** 15-minute monitoring with 5-second updates
  - **Long Go Mode:** 24-hour monitoring with 20-second updates
- **Visual Indicators:** Color-coded price changes (green for gains, red for losses)
- **Price Flash Alerts:** Visual flashing when significant price movements occur
- **Sound Alerts:** Optional audio notifications for price movements
- **Historical Sparkline:** Visual price trend display using Unicode characters
- **Conversion Tools:** BTC to USD, USD to BTC, USD to satoshis, satoshis to USD
- **API Key Management:** Automatic setup and configuration file handling
- **Cross-Platform:** Native executables for Windows and Linux
- **Color-coded Output:** Clear, colorized feedback for all operations

## Requirements
- Go (for building from source). The compiled executable has no external dependencies.
- LiveCoinWatch API key (free from https://www.livecoinwatch.com/tools/api)

## How to Run
1. (Optional) Use the `build.ps1` script to compile the executable for your platform.
2. Open a terminal or command prompt.
3. Navigate to the directory where the `bmon` executable is located.
4. Run the application using one of the formats below.

## Command-Line Flags

### Monitoring Modes
- `-go` - Monitor for 15 minutes with 5-second updates
- `-golong` - Monitor for 24 hours with 20-second updates
- `-s` - Enable sound alerts
- `-h` - Enable history sparkline

### Conversion Tools
- `-bu <amount>` - Convert Bitcoin amount to USD
- `-ub <amount>` - Convert USD amount to Bitcoin
- `-us <amount>` - Convert USD amount to satoshis
- `-su <amount>` - Convert satoshi amount to USD

### Controls (during monitoring)
- `R` - Reset baseline price and timer
- `M` - Switch between go/golong modes
- `S` - Toggle sound alerts
- `H` - Toggle history sparkline
 - `I` - Switch back to interactive mode (from go/golong)

## Examples

### Example 1: Interactive monitoring
```sh
./bmon
```

### Example 2: Monitor for 15 minutes with sound
```sh
./bmon -go -s
```

### Example 3: Convert 0.5 BTC to USD
```sh
./bmon -bu 0.5
```

### Example 4: Convert $50,000 to BTC
```sh
./bmon -ub 50000
```

### Example 5: Convert $100 to satoshis
```sh
./bmon -us 100
```

### Example 6: Convert 1M satoshis to USD
```sh
./bmon -su 1000000
```

## Configuration
The application automatically creates and manages configuration files:
- `bmon.ini` - Primary configuration file (created in executable directory)
- `vbtc.ini` - Fallback configuration file (if bmon.ini not found)

On first run, the application will guide you through API key setup.
