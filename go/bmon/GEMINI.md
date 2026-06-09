# Gemini Project File

## Project: bmon (Go Edition)

**Author:** Kreft&Gemini
**Date:** 2025-08-07
**Version:** 1.5

---

### Description

This project is the Go port of `bmon.ps1`, a lightweight command-line Bitcoin price monitor. It is implemented as a Bubble Tea TUI with cross-platform executables for Windows and Linux. It fetches live prices from the LiveCoinWatch API and supports interactive monitoring, timed single-line sessions, currency conversion, and configuration management.

### Key Functionality

- **Cross-Platform:** Compiled Go binary; no runtime dependencies beyond the executable.
- **Multiple Monitoring Modes:** Landing/interactive, Go (15 min), GoLong (24 hr), and K (30 min) via `-go`, `-golong`, `-k`, or keyboard.
- **Bubble Tea TUI:** Single-line spinner display for go/golong/k; multi-line interactive view; spinner animation via Charm bubbles.
- **Window Coloring:** Range-colored spinner encodes sparkline volatility (`max − min` of up to 14 history points). Flag `-range` / `-r`, auto-on with `-k`, runtime toggle `w` / `W`. Logic in `getSparklineRange`, `rangeSpinnerColorCode`, `spinnerColorCode`.
- **Dynamic Controls:** Same keyboard map as the PowerShell edition (R, E, M, K, I, S, H, W, arrow aliases).
- **Visual & Audible Alerts:** Lipgloss color styling, flash on price moves, optional beeps.
- **Compact Retry Indicator:** Shared retry state replaces spinner with colored digits during API retries.
- **Configuration:** `bmon.ini` primary, `vbtc.ini` fallback; `-config` menu.

### Window Coloring (Spinner)

Gated by `tuiModel.rangeSpinnerEnabled && sparklineEnabled`. Precedence: retry digit → cyan (fetching) → range color → white.

| Sparkline range (USD) | ANSI color code | Appearance |
| --------------------- | --------------- | ---------- |
| < 10                  | 15              | White      |
| 10 – 49.99            | 2               | Green      |
| 50 – 99.99            | 11              | Yellow     |
| 100 – 249.99          | 1               | Red        |
| ≥ 250                 | 5               | Magenta    |

Visible in go/golong/k single-line `View()` only (interactive mode has no spinner).

### How to Run

1. **Build:** `.\build.ps1` or `go build -o bmon .`
2. **Execute:**
   - Interactive: `./bmon`
   - Go: `./bmon -go -s -h -range`
   - K mode: `./bmon -k`
   - Help: `./bmon -help`
   - Config: `./bmon -config`

### Source Layout

- `main.go`: CLI parsing, API, TUI model, sparkline, window coloring, help text.
- `console_windows.go` / `console_other.go`: Terminal UTF-8 and ANSI setup.
- `README.md`: User documentation.
- `README.html`: In-browser markdown viewer.
- `GEMINI.md`: Internal project reference for AI assistants.

### Dependencies

- Go 1.x (build only)
- LiveCoinWatch API key
