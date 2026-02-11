# Gemini Project File

## Project: vBTC (Virtual Bitcoin Trading Simulator) - Go Version

**Author:** Kreft&Gemini[Gemini 2.5 Pro (preview)]
**Date:** 2026-01-29
**Version:** 1.6

---

### Description

This project is a Go implementation of `vBTC`, an interactive, command-line based Bitcoin trading simulator originally written in PowerShell. It allows users to practice trading strategies by buying and selling virtual Bitcoin using a simulated portfolio, all while tracking performance against real-time market data from the LiveCoinWatch API.

As a Go application, it is designed to be a compiled, cross-platform executable, offering the same core features as its PowerShell counterpart with the performance and portability benefits of Go.

### Key Functionality

- **Cross-Platform:** Written in Go, it can be compiled and run on Windows, macOS, and Linux.
- **Real-time Data Simulation:** Fetches live Bitcoin market data, including a 1-Hour Simple Moving Average (SMA), 24-hour volatility metrics, and a velocity telemetry (displayed in brackets after Volatility). Historical data is cached for 15 minutes to optimize API usage.
- **Portfolio Management:** Initializes users with a starting capital of $1000 and tracks their cash (USD), Bitcoin (BTC) holdings, and total portfolio value in `vbtc.ini`.
- **Transaction Ledger:** All buy and sell activities are recorded in `ledger.csv`, providing a complete history of trades with comprehensive statistics including portfolio summary, average prices, and transaction counts across all historical data.
- **Configuration & Maintenance:** A `config` menu allows users to update their API key, reset their portfolio, archive the main ledger, and merge multiple archives into a master file.
- **Flexible Trading:** Supports trading by specific amounts, percentages of the user's balance (e.g., `50p`), and selling amounts specified in satoshis (e.g., `50000s`).
- **User-Friendly Interface:** Employs command shortcuts (e.g., `b` for `buy`), color-coded feedback for market and portfolio changes, and a trade confirmation screen with a 2-minute timeout to ensure prices are current. Arrow keys can be used as shortcuts during trade confirmation (Up = Accept, Down/Left = Cancel, Right = Refresh). Esc key can be used to exit from Config, Help, and Ledger screens.
- **Safe Trading Logic:** Implements a read-before-write mechanism to prevent race conditions, ensuring that the user's balance is always accurate before a trade is finalized.
- **Onboarding:** A guided first-time setup process helps users configure their required API key.
- **Smart Exit:** Detects if it's being run in a non-persistent shell (e.g., by double-clicking) and pauses for user input before closing.
- **Session Statistics:** The exit summary includes a "Transactions:" count showing the total number of buy and sell transactions made during the current session, displayed as the first item in the Session Summary section.

### How to Run

1.  **Compile:** Open a terminal in the project directory and run:
    ```sh
    go build main.go
    ```
    This will create `main.exe` (Windows) or `main` (Linux/macOS). You can rename this to `vbtc.exe` or `vbtc`.

2.  **Execute:** Run the compiled binary from your terminal.
    - **Windows:** `.\vbtc.exe`
    - **Linux/macOS:** `./vbtc`

### Help Options

- Run with `-help`, `-h`, or `--help` to display the help screen and exit
- Run with `-config` or `--config` to open the configuration menu and exit (e.g. to fix or set your API key when it is broken or missing)
- Use the `help` command within the application to view available commands

If the application exits with a 403 API error (e.g. "403 Encountered: Ensure API Key Configured and Enabled"), run `vbtc -config` to configure your API key.

### Commands

-   `buy [amount]`: Purchase Bitcoin with a specified USD amount.
-   `sell [amount]`: Sell a specified amount of BTC or satoshis.
-   `ledger`: View comprehensive transaction history with detailed statistics including portfolio summary, average purchase/sale prices, and transaction counts across current and archived ledgers.
-   `refresh`: Manually force an update of market data.
-   `config`: Access the configuration menu.
-   `help`: Display the help screen.
-   `exit`: Close the application and view a comprehensive final summary including portfolio performance, session statistics (including transaction count for the current session), and complete trading history with all-time statistics.

### Dependencies

-   Go programming language
-   External Go Modules:
    -   `github.com/fatih/color`
    -   `github.com/shirou/gopsutil/v3/process`
    -   `gopkg.in/ini.v1`
    -   `golang.org/x/term`
    -   `github.com/Knetic/govaluate`

### File Structure

-   `main.go`: The main Go source code for the application.
-   `go.mod` / `go.sum`: Go module files defining dependencies.
-   `vbtc.exe` (or `vbtc`): The compiled executable.
-   `vbtc.ini`: Stores the API key and user's portfolio data (auto-generated).
-   `ledger.csv`: Logs all buy and sell transactions (auto-generated).
-   `vBTC - Ledger_*.csv`: Archived ledger files created via the config menu.
-   `vBTC - Ledger_Merged.csv`: Combined ledger file created when merging archives.

### Ledger Summary Features

The application provides comprehensive trading statistics across all historical data. In the **Ledger** screen (Ledger modal), all-time values are shown with current-session values in brackets where applicable (e.g. `Average Purchase: $92,262.37 [$80,234.10]`). The **Exit** screen shows a Trading History Summary with all-time statistics only (no session brackets).

#### Statistics Displayed
- **Portfolio Summary**: Current portfolio value, total profit/loss, Bitcoin holdings, invested amount, and cash
- **Transaction Count**: Total number of buy and sell transactions (Ledger modal: session count in brackets)
- **Total Bought (USD & BTC)**: Complete buy trading volume (Ledger modal: session volume in brackets)
- **Average Purchase**: Weighted average BTC price for all buy transactions (Ledger modal: session average in brackets)
- **Average Sale**: Weighted average BTC price for all sell transactions (Ledger modal: session average in brackets)
- **Tx Range**: Minimum and maximum Bitcoin price (USD per BTC) at the time of any transaction—not total transaction value (Ledger modal only)
- **Session Tx Range**: Same as Tx Range but for the current session; shown on a separate line in the Ledger modal when session has transactions
- **Time**: Span from first ledger entry to latest. Format: minutes (`M`) under 1 hour, hours (`H`) under 24h, integer days (`D`) for 24h+. Ledger modal shows total with session span in brackets (e.g. `Time: 204D [3H]`); Exit modal shows total only

#### Archive Support
- **Current Ledger**: Reads from `ledger.csv`
- **Merged Ledger**: Prefers `vBTC - Ledger_Merged.csv` if available
- **Individual Archives**: Falls back to `vBTC - Ledger_MMDDYY.csv` files
- **Deduplication**: Automatically removes duplicate transactions by timestamp
- **Chronological Order**: All data is sorted by transaction date

#### Color Coding
- **Green**: Positive values, buy-related statistics, net gains
- **Red**: Negative values, sell-related statistics, net losses
- **White**: Neutral values (zero or informational counts)
- **Yellow**: Section headers

### Velocity telemetry (technical)

The main screen shows **Volatility** with an optional bracketed integer (e.g. `Volatility: 3.99% [15]`). The bracketed value is **velocity**.

#### Formula
- **Velocity** = `(TotalChange / (24H High - 24H Low)) * Volatility * (1HourDeltaTotal / (24DeltaTotal/24))`
- **TotalChange** (24DeltaTotal): Sum of absolute deltas between consecutive historical price points over the 24h window. For API history points sorted by date, `TotalChange = |rate[1]-rate[0]| + |rate[2]-rate[1]| + ...` (total price path length in USD).
- **1HourDeltaTotal**: Same sum for the **last hour only**—include segment `|rate[i]-rate[i-1]|` when `point[i].date >= cutoffMs` (cutoff = end of 24h window minus 1 hour). The multiplier compares last-hour activity to the 24h average: above-average increases velocity, below-average decreases it.
- **24H High / 24H Low**: Min and max BTC rate from the same 24h history (same as the values shown on the main screen).
- **Volatility**: The displayed 24h volatility percentage used as a **whole number** (e.g. 3.99% → multiply by 3.99, not 0.0399).
- Result is **rounded to the nearest integer** for display. Edge cases: if `24DeltaTotal/24` is 0, use `multiplier = 1`; if `1HourDeltaTotal` is 0, velocity becomes 0.

#### Velocity color
- The **velocity bracket** (e.g. `[15]`) is colored separately from the volatility percentage:
  - **Green:** `1HourDeltaTotal > (24DeltaTotal/24)` — last hour above average
  - **Red:** otherwise (last hour at or below average)
  - **White:** when multiplier data is missing (no `1HourDeltaTotal` or `hourlyAvg`)

#### Data flow
- **Historical fetch** (in `updateApiData`): After sorting `history.History` by date, computes `totalChange` and `totalChange1h` (sum of deltas in last hour). Sets `newData.Rate24hTotalChange` and `newData.Rate24hTotalChange1h`. On fallback (no history or no old data), both set to 0. `copyHistoricalData` copies both from source to dest.
- **showMainScreen**: When displaying the Volatility line, if `Rate24hTotalChange > 0` and `(Rate24hHigh - Rate24hLow) > 0`, velocity is computed (raw * multiplier) and appended as `[N]`. Otherwise only `X.XX%` is shown.

#### Verbose output
- Run with `-verbose` or `-v` (checked at startup; can appear anywhere in `os.Args[1:]`). When set, writes to **stderr**:
  - After computing TotalChange in historical block: `TotalChange (sum of absolute deltas over 24h history): <value> from <count> points; 1HourDeltaTotal: <value>`
  - When velocity is displayed on main screen: `Velocity calculation: TotalChange=..., 1HourDeltaTotal=..., 24H High=..., 24H Low=..., range=..., Volatility=...% (as whole number), hourlyAvg=..., multiplier=..., velocity=...`
