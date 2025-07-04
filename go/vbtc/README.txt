vBTC - Virtual Bitcoin Trading Simulator

A command-line application for simulating Bitcoin trading. It uses a live API to fetch real-time Bitcoin prices and allows users to manage a virtual portfolio.

--- Features ---

- Live Bitcoin price updates.
- Buy and sell functionality using a virtual portfolio.
- Portfolio tracking with profit/loss calculation.
- Transaction ledger to record all trades.
- Configuration file for API key and portfolio management.

--- First-Time Setup ---

On the first run, the application will prompt you to enter a LiveCoinWatch API key. 
You can obtain a free key from their website: https://www.livecoinwatch.com/tools/api

The application will create two files in the same directory:
- vbtc.ini: Stores your API key and portfolio data.
- ledger.csv: Records all your transactions.

--- How to Use ---

1. Run the executable file (Double-click on vbtc.exe, or run via CMD line).
2. On the first run, you will be prompted to enter your API key.
3. Once the main screen is displayed, you can enter commands to interact with the application.

--- Commands ---

- buy [amount]: Purchase a specific USD amount of Bitcoin.
- sell [amount]: Sell a specific amount of BTC (e.g., 0.5) or satoshis (e.g., 50000s).
- ledger: View a history of all your transactions.
- refresh: Manually update the market data.
- config: Access the configuration menu to update the API key or reset the portfolio.
- help: Show the help screen.
- exit: Exit the application and view a summary of your portfolio.

--- Configuration (vbtc.ini) ---

This file stores the application's settings and your portfolio data.

[Settings]
ApiKey = YOUR_API_KEY

[Portfolio]
PlayerUSD = 1000.00
PlayerBTC = 0.0
PlayerInvested = 0.0

--- Ledger (ledger.csv) ---

This file contains a history of all your transactions in CSV format.

TX,USD,BTC,BTC(USD),User BTC,Time
Buy,100.00,0.00123456,81000.00,0.00123456,010124@1200

--- Dependencies ---

This project uses the following third-party Go packages:
- github.com/fatih/color: For colorized console output.
- gopkg.in/ini.v1: For managing INI file configurations.
