'# vBTC - Virtual Bitcoin Trading Simulator

## Version 1.3

## Author
Kreft&Gemini[Gemini 2.5 Pro (preview)]

## Date
2025-07-05

## Description
vBTC is a command-line application for simulating Bitcoin trading. It uses a live API to fetch real-time Bitcoin prices and allows users to manage a virtual portfolio with a starting capital of $1000.00.

The main screen displays:
- Real-time Bitcoin market data (Price, 24h Change, High, Low, Volatility, Volume).
- The user's personal portfolio (Cash, BTC holdings, and total value).

## Features
- **Real-time Market Data:** Fetches and displays live Bitcoin prices from LiveCoinWatch, including 24h high, low, volatility, and a 1-Hour Simple Moving Average (SMA), with a 15-minute cache for historical data to optimize API calls.
- **Portfolio:** Tracks your cash (USD) and Bitcoin holdings, including invested capital and P/L.
- **Transaction Ledger:** Records all buy and sell transactions in a `ledger.csv` file, with an in-app viewer and archive function.
- **Configuration Options:** Allows you to update your API key, reset your portfolio, and archive the ledger.
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
3.  Run the executable: `.\vbtc.exe` on Windows, or `./vbtc` on macOS/Linux.
4.  On the first run, you will be prompted to enter your LiveCoinWatch API key.

## Commands
- `buy [amount]`: Purchase a specific USD amount of Bitcoin. If no amount is provided, you will be prompted.
- `sell [amount]`: Sell a specific amount of BTC (e.g., `0.5`) or satoshis (e.g., `50000s`). If no amount is provided, you will be prompted.
- `ledger`: View a history of all your transactions.
- `refresh`: Manually update the market data.
- `config`: Access the configuration menu.
- `help`: Show the help screen.
- `exit`: Exit the application and display a final portfolio summary.


## Tips

- **Command Shortcuts:** You can use shortcuts for commands (e.g. 'b 10' to buy $10 of BTC). As long as the shortcut is a unique match, it will work.
- **Percentage Trading:** Use the 'p' suffix to trade a percentage of your balance. This field also supports math expressions (e.g., '50p' for 50%, or '100/3p' for 33.3%).
- **Satoshi Trading:** When selling, use the 's' suffix to specify an amount in satoshis (e.g., '100000s').
- **1H SMA:** The 1-Hour Simple Moving Average shows the average price over the last hour. It's green if the current price is above the average (bullish) and red if below (bearish).

## Files
- `vbtc.exe` (or `vbtc` on macOS/Linux): The main application executable.
- `vbtc.ini`: Stores your API key and portfolio data.
- `ledger.csv`: Logs all buy and sell transactions.
- `README.txt`: This file.

## Dependencies
This project uses the following third-party Go packages:
- github.com/fatih/color: For colorized console output.
- github.com/Knetic/govaluate: For math expression evaluation.
- github.com/shirou/gopsutil/v3/process: For the smart "pause on exit" feature.
- gopkg.in/ini.v1: For managing INI file configurations.
