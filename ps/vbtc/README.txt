'# vBTC - Virtual Bitcoin Trading Simulator

## Version 1.5

## Author
Kreft&Gemini[Gemini 2.5 Pro (preview)]

## Date
2025-07-08

## Description
vBTC is an interactive PowerShell-based Bitcoin trading application. Users can buy and sell Bitcoin using a simulated portfolio, track their trades in a ledger, and view real-time market data from the LiveCoinWatch API.

This script provides a command-line interface for a simulated Bitcoin trading experience. On first run, it guides the user through setting up their LiveCoinWatch API key and initializes their portfolio with a starting capital of $1000.00.

The main screen displays:
- Real-time Bitcoin market data (Price, 1H SMA, 24h Change, High, Low, Volatility, Volume).
- The user's personal portfolio (Cash, BTC holdings, and total value).

## Features
- **Real-time Market Data:** Fetches and displays live Bitcoin prices from LiveCoinWatch, including 24h high, low, volatility, and a 1-Hour Simple Moving Average (SMA), with a 15-minute cache for historical data to optimize API calls.
- **Portfolio:** Tracks your cash (USD) and Bitcoin holdings.
- **Transaction Ledger:** Records all buy and sell transactions in a `ledger.csv` file.
- **Configuration Options:** Allows you to update your API key, reset your portfolio, archive the main ledger, and merge old archives into a single master file.
- **Command Shortcuts:** Use partial commands (e.g., 'b' for 'buy') for quick trading.
- **Percentage-based Trading:** Use the 'p' suffix to trade a percentage of your assets (e.g., `50p` for 50%, `100/3p` for 33.3%).

## Color Coding
The application uses colors to provide quick visual feedback:

- **Green:** Indicates a positive change (price up, profit) or a "Buy" transaction. The Volatility metric is green if the market is more volatile in the last 12 hours than the previous 12.
- **Red:** Indicates a negative change (price down, loss) or a "Sell" transaction. The Volatility metric is red if the market is less volatile.
- **White:** Indicates a neutral or unchanged value.
- **Yellow / Cyan / Blue / DarkYellow:** Used for UI elements like titles and command prompts for better readability.

## Requirements
- PowerShell
- An internet connection
- A free API key from [https://www.livecoinwatch.com/tools/api](https://www.livecoinwatch.com/tools/api)

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
-   `config`: Access the configuration menu. From here you can:
    - Update your API key.
    - Reset your portfolio to its starting state.
    - Archive the main `ledger.csv` to a timestamped file.
    - Merge all archived ledgers into a single `vBTC - Ledger_Merged.csv` file.
-   `help`: Show the help screen with a list of commands.
-   `exit`: Exit the application and display a final portfolio summary.

## Tips
- **Command Shortcuts:** You can use shortcuts for commands (e.g. 'b 10' to buy $10 of BTC). As long as the shortcut is a unique match for a command, it will work.
- **Percentage Trading:** Use the 'p' suffix to trade a percentage of your balance (e.g., '50p' for 50%). Math expressions are also supported (e.g., '100/3p' for 33.3%).
- **Satoshi Trading:** When selling, use the 's' suffix to specify an amount in satoshis (e.g., '100000s').
- **1H SMA:** The 1-Hour Simple Moving Average shows the average price over the last hour. It's green if the current price is above the average (bullish) and red if below (bearish).

## Files
-   `vbtc.ps1`: The main script file.
-   `vbtc.ini`: Stores the user's API key and portfolio data.
-   `ledger.csv`: Logs all buy and sell transactions.
-   `README.txt`: This file.
