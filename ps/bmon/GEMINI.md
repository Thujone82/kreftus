# Gemini Project File

## Project: bmon

**Author:** Kreft&Gemini[Gemini 2.5 Pro (preview)]
**Date:** 2025-08-07
**Version:** 1.6

---

### Description

`bmon` is a lightweight, command-line based Bitcoin price monitor written in PowerShell. It is a spinoff of the `vBTC` trading simulator, focusing on providing fast, real-time price data with minimal overhead. The script features an automatic one-time setup for API key configuration and can fall back to using the `vbtc.ini` file if available, making it seamless for existing `vBTC` users.

The tool operates in several modes:

1. **Interactive Mode:** A full-screen interface where users can manually start and stop 5-minute monitoring sessions.
2. **Go Mode (`-go` / `-g`):** A non-interactive single-line mode that monitors for 15 minutes with 5-second updates.
3. **Long Go Mode (`-golong` / `-gl`):** Extended single-line monitoring for 24 hours with 20-second updates.
4. **K Mode (`-k`):** 30-minute single-line monitoring with 4-second updates, sparkline enabled, and volatility coloring (volatility-colored spinner) enabled by default.
5. **K Long Run (`-kl`):** Same as K mode for 30 minutes, then automatically continues in golong for 24 hours.
6. **Conversion Mode:** Command-line switches for quick currency conversions between BTC, USD, and Satoshis.

### Key Functionality

- **API Key Management:** Automatically prompts for a LiveCoinWatch API key on first run and saves it to `bmon.ini`. It can also read the key from `vbtc.ini` if present. Use `-config` to open the configuration menu.
- **Multiple Monitoring Modes:** Interactive, Go, GoLong, and K modes with runtime switching via keyboard.
- **Dynamic Controls:** Reset baseline (`r` / Right arrow), extend timer (`e` / Left arrow), toggle go/golong (`m` / Down arrow), switch to K mode (`k` / Up arrow), return to interactive (`i`), toggle sound (`s`), toggle sparkline (`h`), toggle volatility coloring (`v`).
- **Volatility Coloring:** When the sparkline is active in go/golong/k single-line modes, the spinner color reflects sparkline volatility (`max − min` of the last 14 prices). Enable at launch with `-volatility` / `-vl`, auto-enabled with `-k`, or toggle at runtime with `v`.
- **Visual & Audible Alerts:** Color-coded price changes (green/red), optional beeps, and inverted flash on significant moves.
- **Sparkline History:** Unicode mini-chart of the last 14 price samples; toggled with `h` or enabled at launch with `-h`.
- **Currency Conversion:** Direct arguments `-bu`, `-ub`, `-us`, `-su`.
- **Compact Retry Indicator:** During API failures in go/golong/k modes, the spinner is replaced briefly by a colored retry digit (yellow 1–4, red 5).

### Volatility Coloring (Spinner)

Applies only when `$volatilitySpinnerEnabled` is true **and** the sparkline is visible **and** at least 2 history points exist. During an API fetch the spinner inverts: cyan background with the volatility tier as foreground (white foreground when volatility coloring is off). Retry digits keep yellow (or red on final failure) foreground; volatility tier is the background when enabled, default background when off.

| Sparkline volatility (USD) | Spinner color |
| -------------------------- | ------------- |
| < 10                       | White         |
| 10 – 49.99                 | Green         |
| 50 – 99.99                 | Yellow        |
| 100 – 249.99               | Red           |
| ≥ 250                      | Magenta       |

Key helpers in `bmon.ps1`: `Get-SparklineRange`, `Get-VolatilitySpinnerColor`, `Get-SpinnerColors`, `Write-SpinnerChar`.

### How to Run

**Configuration:**
- **Config Menu:** `.\bmon.ps1 -config`

**Monitoring:**
- **Interactive:** `.\bmon.ps1 [-s] [-h] [-volatility]`
- **Go Mode (15 min):** `.\bmon.ps1 -go` or `.\bmon.ps1 -g`
- **Long Go Mode (24 hr):** `.\bmon.ps1 -golong` or `.\bmon.ps1 -gl`
- **K Mode (30 min):** `.\bmon.ps1 -k`
- **K Long Run (30 min K + 24 hr golong):** `.\bmon.ps1 -kl`
- **With options:** `.\bmon.ps1 -go -s -h -volatility` (sound, sparkline, volatility coloring)

**Conversion:**
- **BTC to USD:** `.\bmon.ps1 -bu <amount>`
- **USD to BTC:** `.\bmon.ps1 -ub <amount>`
- **USD to Sats:** `.\bmon.ps1 -us <amount>`
- **Sats to USD:** `.\bmon.ps1 -su <amount>`

**Help:** `.\bmon.ps1 -Help`

### Dependencies

- Windows PowerShell
- An active internet connection
- A free API key from [LiveCoinWatch](https://www.livecoinwatch.com/tools/api)

### File Structure

- `bmon.ps1`: The main executable script.
- `bmon.exe`: A compiled executable of the script.
- `bmon.ini`: Configuration file for storing the API key (auto-generated).
- `README.md`: Detailed user documentation.
- `README.html`: In-browser markdown viewer.
- `GEMINI.md`: Internal project reference for AI assistants.
