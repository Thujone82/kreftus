# vBTC — Virtual Bitcoin Trading Simulator (PowerShell Edition)

**Version:** 1.6 · **Author:** Kreft&Gemini[Gemini 2.5 Pro (preview)] · **Date:** 2025-07-08

## Description

vBTC is an interactive PowerShell-based Bitcoin trading application. Users can buy and sell Bitcoin using a simulated portfolio, track their trades in a ledger, and view real-time market data from the LiveCoinWatch API.

On first run, the script guides you through setting up your LiveCoinWatch API key and initializes your portfolio with a starting capital of **$1000.00**.

The main screen displays:

- Real-time Bitcoin market data (Price, 1H SMA, 24h Change, High, Low, Volatility [velocity], Volume, **Updated** timestamp)
- Your personal portfolio (Cash, BTC holdings, and total value)

## Features

- **Real-time Market Data:** Live Bitcoin prices from LiveCoinWatch, including 24h high, low, volatility (with velocity metric in brackets), and a 1-Hour Simple Moving Average (SMA), with a 15-minute cache for historical data to optimize API calls
- **Portfolio:** Tracks your cash (USD) and Bitcoin holdings
- **Transaction Ledger:** Records all buy and sell transactions in `ledger.csv`, with comprehensive statistics including portfolio summary, average prices, and transaction counts across all historical data
- **Configuration Options:** Update your API key, reset your portfolio, archive the main ledger, and merge old archives into a single master file
- **Command Shortcuts:** Partial commands (e.g. `b` for `buy`) for quick trading
- **Percentage-based Trading:** Use the `p` suffix to trade a percentage of your assets (e.g. `50p` for 50%, `100/3p` for 33.3%)

## Color Coding

The application uses colors to provide quick visual feedback:

- **Green:** Positive change (price up, profit) or a Buy transaction. Volatility is green if the market is more volatile in the last 12 hours than the previous 12
- **Red:** Negative change (price down, loss) or a Sell transaction. Volatility is red if the market is less volatile
- **Volatility line:** Displays as `Volatility: X.XX% [N]` where the bracketed value is velocity (see Tips)
- **White:** Neutral or unchanged value
- **Yellow / Cyan / Blue / DarkYellow:** UI elements like titles and command prompts
- **Updated:** Cyan timestamp showing when market data was last fetched
- **Market Rate (trade confirmation):** Green if current price is above 1H SMA, red if below, white if SMA is unavailable

## Requirements

- PowerShell
- An internet connection
- A free API key from [LiveCoinWatch](https://www.livecoinwatch.com/tools/api)

## How to Run

1. Open a PowerShell terminal
2. Navigate to the directory where `vbtc.ps1` is located
3. Run: `.\vbtc.ps1`
4. On first run, enter your LiveCoinWatch API key when prompted

## Help Options

- `.\vbtc.ps1 -help` or `-?` — display the help screen and exit
- `.\vbtc.ps1 -config` — open the configuration menu and exit (e.g. to fix or set your API key)
- `.\vbtc.ps1 -Verbose` — verbose output including velocity calculation details
- `help` command within the application — view available commands

If the application exits with **403 Encountered: Ensure API Key Configured and Enabled**, run `vbtc -config` (or `.\vbtc.ps1 -config`) to configure your API key.

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
- **Velocity:** Shown in brackets after Volatility (e.g. `Volatility: 3.99% [15]`). **Velocity color:** Magenta when velocity ≥ 50; Green when last-hour activity is above the 24h average; Red otherwise; White when multiplier data is missing

## Ledger Summary Features

The `ledger` command provides comprehensive trading statistics across all historical data.

### Statistics Displayed

- **Portfolio Summary:** Current value, profit/loss, BTC holdings, invested amount, cash
- **Transaction Count:** Total buy and sell transactions
- **Total Bought (USD & BTC):** Complete buy trading volume
- **Average Purchase / Average Sale:** Weighted average BTC prices

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
| `vbtc.ps1` | Main script |
| `vbtc.exe` | Compiled executable |
| `vbtc.ini` | API key and portfolio data |
| `ledger.csv` | Transaction log |
| `vBTC - Ledger_MMDDYY.csv` | Archived ledger files |
| `vBTC - Ledger_Merged.csv` | Combined ledger from merge |
| `README.md` | User documentation (source) |
| `README.html` | In-browser markdown viewer |
| `GEMINI.md` | Internal project reference for AI assistants |
