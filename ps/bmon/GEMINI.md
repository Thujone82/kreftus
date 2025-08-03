# Gemini Project File

## Project: bmon

**Author:** Kreft&Gemini[Gemini 2.5 Pro (preview)]
**Date:** 2025-08-02
**Version:** 1.4

---

### Description

`bmon` is a lightweight, command-line based Bitcoin price monitor written in PowerShell. It is a spinoff of the `vBTC` trading simulator, focusing on providing fast, real-time price data with minimal overhead. The script features an automatic one-time setup for API key configuration and can fall back to using the `vbtc.ini` file if available, making it seamless for existing `vBTC` users.

The tool operates in several modes:
1.  **Interactive Mode:** A full-screen interface where users can manually start and stop monitoring sessions.
2.  **Go Mode (`-go`):** A non-interactive, "fire-and-forget" mode that displays a single updating line of price data for 5 minutes.
3.  **Long Go Mode (`-golong`):** An extended version of Go Mode for 24-hour, low-intensity monitoring.
4.  **Conversion Mode:** A set of command-line switches for quick currency conversions between BTC, USD, and Satoshis.

### Key Functionality

- **API Key Management:** Automatically prompts for a LiveCoinWatch API key on first run and saves it to `bmon.ini`. It can also read the key from `vbtc.ini` if present.
- **Multiple Monitoring Modes:** Supports interactive, short-term (`-go`), and long-term (`-golong`) monitoring.
- **Dynamic Controls:** In all modes, users can reset the price baseline (`r`), toggle sound alerts (`s`), and toggle a price history sparkline (`h`). In `-go`/`-golong` modes, users can also switch between them (`m`).
- **Visual & Audible Alerts:** Uses color-coding (green for price up, red for down) and optional beeps to indicate price changes.
- **Currency Conversion:** Provides direct command-line arguments for converting between USD, BTC, and Satoshis (e.g., `-bu`, `-ub`, `-us`, `-su`).
- **Error Handling:** Includes retry logic with exponential backoff for API calls to handle network instability.

### How to Run

The script is executed from a PowerShell terminal.

**Monitoring:**
- **Interactive:** `.\bmon.ps1`
- **Go Mode (5 mins):** `.\bmon.ps1 -go`
- **Long Go Mode (24 hrs):** `.\bmon.ps1 -golong`

**Conversion:**
- **BTC to USD:** `.\bmon.ps1 -bu <amount>`
- **USD to BTC:** `.\bmon.ps1 -ub <amount>`
- **USD to Sats:** `.\bmon.ps1 -us <amount>`
- **Sats to USD:** `.\bmon.ps1 -su <amount>`

### Dependencies

- Windows PowerShell
- An active internet connection
- A free API key from [LiveCoinWatch](https://www.livecoinwatch.com/tools/api)

### File Structure

- `bmon.ps1`: The main executable script.
- `bmon.exe`: A compiled executable of the script.
- `bmon.ini`: Configuration file for storing the API key (auto-generated).
- `README.txt`: Detailed user documentation.
