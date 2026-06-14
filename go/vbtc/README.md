# vBTC — Virtual Bitcoin Trading Simulator (Go Edition)

**Version:** 1.6 · **Author:** Kreft&Gemini[Gemini 2.5 Pro (preview)] · **Date:** 2025-07-08

## Description

vBTC is a cross-platform command-line application for simulating Bitcoin trading. It uses a live API to fetch real-time Bitcoin prices and allows users to manage a virtual portfolio with a starting capital of **$1000.00**.

The main screen displays:

- Real-time Bitcoin market data (Price, 1H SMA, 24h Change, High, Low, Volatility [velocity], Volume, **Updated** timestamp)
- Your personal portfolio (Cash, BTC holdings, and total value)

## Features

- **Real-time Market Data:** Live Bitcoin prices from LiveCoinWatch, including 24h high, low, volatility (with velocity metric in brackets), and a 1-Hour Simple Moving Average (SMA), with a 15-minute cache for historical data to optimize API calls
- **Portfolio:** Tracks cash (USD), Bitcoin holdings, invested capital, and P/L
- **Transaction Ledger:** Records all buy and sell transactions in `ledger.csv`, with an in-app viewer, archive function, and comprehensive statistics
- **Configuration Options:** Update your API key, reset your portfolio, archive the main ledger, and merge old archives into a single master file
- **Command Shortcuts:** Partial commands (e.g. `b` for `buy`) for quick trading
- **Percentage-based Trading:** Use the `p` suffix to trade a percentage of your assets (e.g. `50p` for 50%, `100/3p` for 33.3%)
- **Cross-Platform:** Native executables for Windows, macOS, and Linux

## Color Coding

The application uses colors to provide quick visual feedback:

- **Green:** Positive change (price up, profit) or a Buy transaction. Volatility is green if the market is more volatile in the last 12 hours than the previous 12
- **Red:** Negative change (price down, loss) or a Sell transaction. Volatility is red if the market is less volatile
- **Volatility line:** Displays as `Volatility: X.XX% [N]` where the bracketed value is velocity (see Tips)
- **White:** Neutral or unchanged value
- **Yellow / Cyan / Blue / HiBlack:** UI elements like titles and command prompts
- **Updated:** Cyan timestamp showing when market data was last fetched
- **Market Rate (trade confirmation):** Green if current price is above 1H SMA, red if below, white if SMA is unavailable

## Requirements

- A terminal or command prompt
- An internet connection
- A free API key from [LiveCoinWatch](https://www.livecoinwatch.com/tools/api)

## How to Run

1. Open a terminal or command prompt
2. Navigate to the directory where `vbtc.exe` (or `vbtc`) is located
3. Run `.\vbtc.exe` on Windows, or `./vbtc` on Linux
4. **On macOS:** After unzipping, you can double-click `vbtc.app`. The first time, you may need to **right-click** the app and select **Open** to bypass security warnings
5. On first run, enter your LiveCoinWatch API key when prompted

## Help Options

- `-help`, `-h`, or `--help` — display the help screen and exit
- `-config` or `--config` — open the configuration menu and exit
- `-verbose` or `-v` — print velocity calculation details to stderr
- `help` command within the application — view available commands

If the application exits with a 403 API error (e.g. **403 Encountered: Ensure API Key Configured and Enabled**), run `vbtc -config` to configure your API key.

## Commands

| Command | Description |
| ------- | ----------- |
| `buy [amount]` | Purchase a specific USD amount of Bitcoin (prompts if amount omitted) |
| `sell [amount]` | Sell BTC (e.g. `0.5`) or satoshis (e.g. `50000s`) |
| `ledger` | View transaction history with detailed statistics |
| `refresh` | Manually update market data |
| `config` | Configuration menu (API key, portfolio reset, ledger archive/merge) |
| `help` | Show the help screen |
| `exit` | Exit with a comprehensive final summary |

## Keyboard Controls

### Trade Confirmation Screen

| Key | Action |
| --- | ------ |
| **Y** or **Up Arrow** | Accept the trade |
| **N**, **Esc**, **Down Arrow**, or **Left Arrow** | Cancel the trade |
| **R** or **Right Arrow** | Refresh the price and get a new offer (2s debounce) |
| **Enter** | Cancel (active offer) or return to main menu (expired offer) |

### Modal Navigation

- **Esc** — Return to main screen from Config, Help, or Ledger
- **Enter** — Confirm selection or return to previous screen
- **R** or **Right Arrow** — Refresh the ledger screen

## Tips

- **Command Shortcuts:** Use shortcuts when unique (e.g. `b 10` to buy $10 of BTC)
- **Percentage Trading:** `50p` for 50%; math expressions supported (e.g. `100/3p` for 33.3%)
- **Satoshi Trading:** When selling, use the `s` suffix (e.g. `100000s`)
- **1H SMA:** Average price over the last hour. Green if current price is above average, red if below. The buy/sell confirmation **Market Rate** uses the same comparison for its color
- **Velocity:** Shown in brackets after Volatility (e.g. `Volatility: 3.99% [15]`). **Velocity color:** Magenta when velocity ≥ 50; Green when last-hour activity is above the 24h average; Red otherwise; White when multiplier data is missing. Use `-verbose` or `-v` for calculation details

## Ledger Summary Features

The `ledger` command provides comprehensive trading statistics across all historical data.

### Statistics Displayed

- **Transaction Count:** Total buy and sell transactions
- **Total Bought/Sold (USD & BTC):** Complete trading volume
- **Average Purchase / Average Sale:** Weighted average BTC prices
- **Net BTC Position:** Current Bitcoin holdings (Total Bought − Total Sold)
- **Net Trading P/L (USD):** Overall trading profit/loss

### Archive Support

- **Current Ledger:** `ledger.csv`
- **Merged Ledger:** Prefers `vBTC - Ledger_Merged.csv` if available
- **Individual Archives:** `vBTC - Ledger_MMDDYY.csv` files
- **Deduplication:** Removes duplicate transactions by timestamp
- **Chronological Order:** All data sorted by transaction date

### Ledger Color Coding

- **Green:** Positive values, buy-related statistics, net gains
- **Red:** Negative values, sell-related statistics, net losses
- **White:** Neutral values
- **Yellow:** Section headers

## Files

| File | Purpose |
| ---- | ------- |
| `vbtc.exe` / `vbtc` | Main application executable |
| `vbtc.ini` | API key and portfolio data |
| `ledger.csv` | Transaction log |
| `vBTC - Ledger_MMDDYY.csv` | Archived ledger files |
| `vBTC - Ledger_Merged.csv` | Combined ledger from merge |
| `README.md` | User documentation (source) |
| `README.html` | In-browser markdown viewer |
| `GEMINI.md` | Internal project reference for AI assistants |

## Dependencies

This project uses the following third-party Go packages:

| Package | Purpose |
| ------- | ------- |
| `github.com/fatih/color` | Colorized console output |
| `github.com/Knetic/govaluate` | Math expression evaluation |
| `github.com/shirou/gopsutil/v3/process` | Smart pause-on-exit feature |
| `gopkg.in/ini.v1` | INI file configuration |
