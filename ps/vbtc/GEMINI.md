# Gemini Project File

## Project: vBTC (Virtual Bitcoin Trading Simulator)

**Author:** Kreft&Gemini[Gemini 2.5 Pro (preview)]
**Date:** 2025-07-28
**Version:** 1.5

---

### Description

`vBTC` is an interactive, command-line based Bitcoin trading simulator built with PowerShell. It allows users to practice trading strategies by buying and selling virtual Bitcoin using a simulated portfolio, all while tracking performance against real-time market data from the LiveCoinWatch API.

The application provides a comprehensive main screen displaying live market statistics (price, 1H SMA, 24h change, volume, etc.) alongside the user's personal portfolio (cash, BTC holdings, total value). It includes robust features for managing the simulation, such as a transaction ledger, portfolio reset, and API key configuration.

### Key Functionality

- **Real-time Data Simulation:** Fetches live Bitcoin market data, including a 1-Hour Simple Moving Average (SMA) and 24-hour volatility metrics. Historical data is cached for 15 minutes to optimize API usage.
- **Portfolio Management:** Initializes users with a starting capital of $1000 and tracks their cash (USD), Bitcoin (BTC) holdings, and total portfolio value.
- **Transaction Ledger:** All buy and sell activities are recorded in `ledger.csv`, providing a complete history of trades.
- **Configuration & Maintenance:** A `config` menu allows users to update their API key, reset their portfolio, archive the main ledger, and merge multiple archives into a master file.
- **Flexible Trading:** Supports trading by specific amounts, percentages of the user's balance (e.g., `50p`), and selling amounts specified in satoshis (e.g., `50000s`).
- **User-Friendly Interface:** Employs command shortcuts (e.g., `b` for `buy`), color-coded feedback for market and portfolio changes, and a trade confirmation screen with a timeout to ensure prices are current.
- **Safe Trading Logic:** Implements a read-before-write mechanism to prevent race conditions, ensuring that the user's balance is always accurate before a trade is finalized.
- **Onboarding:** A guided first-time setup process helps users configure their required API key.

### How to Run

The script is executed from a PowerShell terminal.

1.  Navigate to the directory containing the script.
2.  Run the command: `.\vbtc.ps1` or `vbtc.exe`
3.  On the first run, the script will prompt for a LiveCoinWatch API key.

### Commands

-   `buy [amount]`: Purchase Bitcoin with a specified USD amount.
-   `sell [amount]`: Sell a specified amount of BTC or satoshis.
-   `ledger`: View the transaction history.
-   `refresh`: Manually force an update of market data.
-   `config`: Access the configuration menu.
-   `help`: Display the help screen.
-   `exit`: Close the application and view a final summary.

### Dependencies

-   Windows PowerShell
-   An active internet connection
-   A free API key from [LiveCoinWatch](https://www.livecoinwatch.com/tools/api)

### File Structure

-   `vbtc.ps1`: The main executable script.
-   `vbtc.exe`: A compiled executable of the script.
-   `vbtc.ini`: Stores the API key and user's portfolio data.
-   `ledger.csv`: Logs all buy and sell transactions.
-   `vBTC - Ledger_*.csv`: Archived ledger files.
-   `README.txt`: Detailed user documentation.
-   `build.ps1`: Script to compile the PowerShell script into an executable.
