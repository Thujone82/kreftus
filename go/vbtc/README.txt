'# vBTC - Virtual Bitcoin Trading Simulator

## Version 1.5

## Author
Kreft&Gemini[Gemini 2.5 Pro (preview)]

## Date
2025-07-08

## Description
vBTC is a command-line application for simulating Bitcoin trading. It uses a live API to fetch real-time Bitcoin prices and allows users to manage a virtual portfolio with a starting capital of $1000.00.

The main screen displays:
- Real-time Bitcoin market data (Price, 24h Change, High, Low, Volatility, Volume).
- The user's personal portfolio (Cash, BTC holdings, and total value).

## Features
- **Real-time Market Data:** Fetches and displays live Bitcoin prices from LiveCoinWatch, including 24h high, low, volatility, and a 1-Hour Simple Moving Average (SMA), with a 15-minute cache for historical data to optimize API calls.
- **Portfolio:** Tracks your cash (USD) and Bitcoin holdings, including invested capital and P/L.
- **Transaction Ledger:** Records all buy and sell transactions in a `ledger.csv` file, with an in-app viewer, archive function, and comprehensive statistics including portfolio summary, average prices, and transaction counts across all historical data.
- **Configuration Options:** Allows you to update your API key, reset your portfolio, archive the main ledger, and merge old archives into a single master file.
- **Command Shortcuts:** Use partial commands (e.g., 'b' for 'buy') for quick trading.
- **Percentage-based Trading:** Use the 'p' suffix to trade a percentage of your assets (e.g., `50p` for 50%, `100/3p` for 33.3%).

## Color Coding

The application uses colors to provide quick visual feedback:
- **Green:** Indicates a positive change (price up, profit) or a "Buy" transaction. The Volatility metric is green if the market is more volatile in the last 12 hours than the previous 12.
- **Red:** Indicates a negative change (price down, loss) or a "Sell" transaction. The Volatility metric is red if the market is less volatile.
- **White:** Indicates a neutral or unchanged value.
- **Yellow / Cyan / Blue / Black (HiBlack):** Used for UI elements like titles and command prompts for better readability.

## Requirements
- A terminal or command prompt.
- An internet connection.
- A free API key from https://www.livecoinwatch.com/tools/api

## How to Run
1.  Open a terminal or command prompt.
2.  Navigate to the directory where `vbtc.exe` is located.
3.  Run the executable: `.\vbtc.exe` on Windows, or `./vbtc` on Linux.
    - **On macOS:** After unzipping, you can double-click `vbtc.app`. The first time, you may need to **right-click** the app and select **Open** to bypass security warnings.
4.  On the first run, you will be prompted to enter your LiveCoinWatch API key.

## Help Options
- Run with `-help`, `-h`, or `--help` to display the help screen and exit
- Use the `help` command within the application to view available commands

## Commands
- `buy [amount]`: Purchase a specific USD amount of Bitcoin. If no amount is provided, you will be prompted.
- `sell [amount]`: Sell a specific amount of BTC (e.g., `0.5`) or satoshis (e.g., `50000s`). If no amount is provided, you will be prompted.
- `ledger`: View a comprehensive history of all your transactions with detailed statistics including portfolio summary, average purchase/sale prices, and transaction counts across current and archived ledgers.
- `refresh`: Manually update the market data.
- `config`: Access the configuration menu. From here you can:
  - Update your API key.
  - Reset your portfolio to its starting state.
  - Archive the main `ledger.csv` to a timestamped file.
  - Merge all archived ledgers into a single `vBTC - Ledger_Merged.csv` file.
- `help`: Show the help screen.
- `exit`: Exit the application and display a comprehensive final summary including portfolio performance, session statistics (including transaction count for the current session), and complete trading history with all-time statistics.


## Keyboard Controls

### Trade Confirmation Screen
When confirming a buy or sell transaction, you can use the following keyboard shortcuts:
- **Y** or **Up Arrow** = Accept the trade
- **N**, **Esc**, **Down Arrow**, or **Left Arrow** = Cancel the trade
- **R** or **Right Arrow** = Refresh the price and get a new offer
- **Enter** = Cancel (on active offer) or Return to main menu (on expired offer)

### Modal Navigation
- **Esc** = Return to main screen from Config, Help, or Ledger screens
- **Enter** = Confirm selection or return to previous screen
- **R** or **Right Arrow** = Refresh the ledger screen (reloads transaction data)

## Tips

- **Command Shortcuts:** You can use shortcuts for commands (e.g. 'b 10' to buy $10 of BTC). As long as the shortcut is a unique match, it will work.
- **Percentage Trading:** Use the 'p' suffix to trade a percentage of your balance. This field also supports math expressions (e.g., '50p' for 50%, or '100/3p' for 33.3%).
- **Satoshi Trading:** When selling, use the 's' suffix to specify an amount in satoshis (e.g., '100000s').
- **1H SMA:** The 1-Hour Simple Moving Average shows the average price over the last hour. It's green if the current price is above the average (bullish) and red if below (bearish).

## Ledger Summary Features

The `ledger` command now provides comprehensive trading statistics across all your historical data:

### Statistics Displayed
- **Transaction Count**: Total number of buy and sell transactions
- **Total Bought/Sold (USD & BTC)**: Complete trading volume
- **Average Purchase**: Weighted average BTC price for all buy transactions
- **Average Sale**: Weighted average BTC price for all sell transactions  
- **Net BTC Position**: Current Bitcoin holdings (Total Bought - Total Sold)
- **Net Trading P/L (USD)**: Overall trading profit/loss

### Archive Support
- **Current Ledger**: Reads from `ledger.csv`
- **Merged Ledger**: Prefers `vBTC - Ledger_Merged.csv` if available
- **Individual Archives**: Falls back to `vBTC - Ledger_MMDDYY.csv` files
- **Deduplication**: Automatically removes duplicate transactions by timestamp
- **Chronological Order**: All data is sorted by transaction date

### Color Coding
- **Green**: Positive values, buy-related statistics, net gains
- **Red**: Negative values, sell-related statistics, net losses
- **White**: Neutral values (zero or informational counts)
- **Yellow**: Section headers

## Files
- `vbtc.exe` (or `vbtc` on macOS/Linux): The main application executable.
- `vbtc.ini`: Stores your API key and portfolio data.
- `ledger.csv`: Logs all buy and sell transactions.
- `vBTC - Ledger_MMDDYY.csv`: Archived ledger files (created when archiving).
- `vBTC - Ledger_Merged.csv`: Combined ledger file (created when merging archives).
- `README.txt`: This file.

## Dependencies
This project uses the following third-party Go packages:
- github.com/fatih/color: For colorized console output.
- github.com/Knetic/govaluate: For math expression evaluation.
- github.com/shirou/gopsutil/v3/process: For the smart "pause on exit" feature.
- gopkg.in/ini.v1: For managing INI file configurations.
