# Gemini Project File

## Project: vBTC (Virtual Bitcoin Trading Simulator) - Go Version

**Author:** Kreft&Gemini[Gemini 2.5 Pro (preview)]
**Date:** 2025-08-02
**Version:** 1.0

---

### Description

This project is a Go implementation of `vBTC`, an interactive, command-line based Bitcoin trading simulator originally written in PowerShell. It allows users to practice trading strategies by buying and selling virtual Bitcoin using a simulated portfolio, all while tracking performance against real-time market data from the LiveCoinWatch API.

As a Go application, it is designed to be a compiled, cross-platform executable, offering the same core features as its PowerShell counterpart with the performance and portability benefits of Go.

### Key Functionality

- **Cross-Platform:** Written in Go, it can be compiled and run on Windows, macOS, and Linux.
- **Real-time Data Simulation:** Fetches live Bitcoin market data, including a 1-Hour Simple Moving Average (SMA) and 24-hour volatility metrics. Historical data is cached for 15 minutes to optimize API usage.
- **Portfolio Management:** Initializes users with a starting capital of $1000 and tracks their cash (USD), Bitcoin (BTC) holdings, and total portfolio value in `vbtc.ini`.
- **Transaction Ledger:** All buy and sell activities are recorded in `ledger.csv`, providing a complete history of trades.
- **Configuration & Maintenance:** A `config` menu allows users to update their API key, reset their portfolio, archive the main ledger, and merge multiple archives into a master file.
- **Flexible Trading:** Supports trading by specific amounts, percentages of the user's balance (e.g., `50p`), and selling amounts specified in satoshis (e.g., `50000s`).
- **User-Friendly Interface:** Employs command shortcuts (e.g., `b` for `buy`), color-coded feedback for market and portfolio changes, and a trade confirmation screen with a 2-minute timeout to ensure prices are current.
- **Safe Trading Logic:** Implements a read-before-write mechanism to prevent race conditions, ensuring that the user's balance is always accurate before a trade is finalized.
- **Onboarding:** A guided first-time setup process helps users configure their required API key.
- **Smart Exit:** Detects if it's being run in a non-persistent shell (e.g., by double-clicking) and pauses for user input before closing.

### How to Run

1.  **Compile:** Open a terminal in the project directory and run:
    ```sh
    go build main.go
    ```
    This will create `main.exe` (Windows) or `main` (Linux/macOS). You can rename this to `vbtc.exe` or `vbtc`.

2.  **Execute:** Run the compiled binary from your terminal.
    - **Windows:** `.\vbtc.exe`
    - **Linux/macOS:** `./vbtc`

### Commands

-   `buy [amount]`: Purchase Bitcoin with a specified USD amount.
-   `sell [amount]`: Sell a specified amount of BTC or satoshis.
-   `ledger`: View the transaction history.
-   `refresh`: Manually force an update of market data.
-   `config`: Access the configuration menu.
-   `help`: Display the help screen.
-   `exit`: Close the application and view a final summary.

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
