# vBTC - Virtual Bitcoin Trading Simulator

## Version 4.0

## Author
Kreft&Gemini[Gemini 2.5 Pro (preview)]

## Date
2025-07-02

## Description
vBTC is an interactive PowerShell-based Bitcoin trading application. Users can buy and sell Bitcoin using a simulated portfolio, track their trades in a ledger, and view real-time market data from the LiveCoinWatch API.

This script provides a command-line interface for a simulated Bitcoin trading experience. On first run, it guides the user through setting up their LiveCoinWatch API key and initializes their portfolio with a starting capital of $1000.00.

The main screen displays:
- Real-time Bitcoin market data (Price, 24h Change, Volume).
- The user's personal portfolio (Cash, BTC holdings, and total value).

## Features
- **Real-time Market Data:** Fetches and displays live Bitcoin prices from LiveCoinWatch.
- **Portfolio Management:** Tracks your cash (USD) and Bitcoin holdings.
- **Transaction Ledger:** Records all buy and sell transactions in a `ledger.csv` file.
- **Configuration Options:** Allows you to update your API key and reset your portfolio.
- **Command Shortcuts:** Use partial commands (e.g., 'b' for 'buy') for quick trading.

## Requirements
- PowerShell
- An internet connection
- A free API key from [LiveCoinWatch](https://www.livecoinwatch.com/developers/api-keys)

## How to Run
1.  Open a PowerShell terminal.
2.  Navigate to the directory where `vbtc.ps1` is located.
3.  Run the script with the command: `.\vbtc.ps1`
4.  On the first run, you will be prompted to enter your LiveCoinWatch API key.

## Commands
The application uses a simple command-line interface. The following commands are available:

-   `buy [amount]`: Purchase a specific USD amount of Bitcoin. If no amount is provided, you will be prompted to enter one.
-   `sell [amount]`: Sell a specific amount of Bitcoin (e.g., `0.5`) or satoshis (e.g., `50000s`). If no amount is provided, you will be prompted to enter one.
-   `ledger`: View a history of all your transactions.
-   `refresh`: Manually update the market data.
-   `config`: Access the configuration menu to update the API key or reset your portfolio.
-   `help`: Show the help screen with a list of commands.
-   `exit`: Exit the application and display a final portfolio summary.

**Tip:** You can use shortcuts for commands. For example, `b` for `buy`, `s` for `sell`, `l` for `ledger`, etc. As long as the shortcut is a unique match for a command, it will work.

## Files
-   `vbtc.ps1`: The main script file.
-   `vbtc.ini`: Stores the user's API key and portfolio data.
-   `ledger.csv`: Logs all buy and sell transactions.
-   `README.txt`: This file.
