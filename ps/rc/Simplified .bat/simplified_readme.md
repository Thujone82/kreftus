# r.bat — Run Continuously (Simplified Batch Edition)

## Version

See build stamp in `r.bat` (e.g. `RC_BUILD=20260529-debug3`).

## Author

Kreft&Cursor

## Description

`r.bat` is a minimal Windows batch script that runs a command in a loop until you press **Ctrl+C** or a run limit is reached. It is a lightweight alternative to [`rc.ps1`](../README.md) in the same folder: **no PowerShell required**, suitable for `cmd.exe`-only environments or quick double-click use.

Download `r.bat` and place it alongside `rc.ps1` in your `ps\rc` directory (for example `C:\path\to\kreftus\ps\rc\`). You do not need a separate subfolder—just the single batch file.

Timing is simple: after each command finishes, the script waits for the configured interval, then runs again. There is no precision grid scheduling and no expect/fail runtime tracking (use full [`rc.ps1`](../README.md) or [`go/rc`](../../go/rc/README.md) for those features).

## Features

- **Continuous execution** — Runs any command string via `cmd /d /c` until stopped.
- **Configurable interval** — Period accepts `s`, `m`, or `h` suffixes; a bare number means minutes (e.g. `5`, `15s`, `5m`, `1h`). Default: 5 minutes.
- **Interactive mode** — Run `r.bat` with no arguments to be prompted for command, period, and optional flags.
- **Clear mode (`-c`)** — Clears the screen before each run.
- **Silent mode (`-q` / `-quiet`)** — Suppresses status lines; command output still appears.
- **Limit mode (`-limit N`)** — Stops after N completed runs (`0` = unlimited).
- **Debug mode (`-debug`)** — Prints `[DEBUG]` lines and appends to `%TEMP%\rc-r-debug.log` (period parsing, run count, errors).

## Requirements

- Windows Command Prompt (`cmd.exe`)
- No PowerShell or Go runtime required

## How to Run

1. Open Command Prompt.
2. Change to the directory where you installed `r.bat`:

   ```bat
   cd /d C:\path\to\kreftus\ps\rc
   ```

3. Run `r.bat` or `r` (if `.BAT` is associated with cmd).
4. Press **Ctrl+C** to stop (unless `-limit` exits first).

## Command-Line Usage

```bat
r.bat "command" [period] [flags...]
```

### Positional arguments

| Argument | Required | Description |
|----------|----------|-------------|
| **command** | Yes | Command to run each iteration. Use quotes if it contains spaces. |
| **period** | No | Interval between runs. Default: `5` (minutes). |

### Flags (any order after period)

| Flag | Description |
|------|-------------|
| **`-c`** | Clear screen before each run. |
| **`-q`** / **`-quiet`** | Silent mode (minimal status output). |
| **`-limit N`** | Exit after N runs. `0` or omitted = unlimited. |
| **`-debug`** | Enable debug output and log file. |

Unknown flags print a warning and are ignored.

## Examples

### Run every 10 seconds, five times

```bat
r.bat "echo hello" 10s -limit 5
```

### Monitor GetForecast every 5 minutes with a clean screen

```bat
r.bat "gf -x Portland" 5m -c
```

### Quick test with debug logging

```bat
r.bat "echo yes" 10s -limit 1 -debug
```

### Interactive (prompts for command, period, flags)

```bat
r.bat
```

```
Command: echo yes
Period (5, 15s, 5m, 1h) [default: 5]: 10s
Flags [-c -q -limit N -debug] (optional): -debug
```

## Debug log

When `-debug` is set:

- Console shows lines prefixed with `[DEBUG]`.
- Full trace is appended to: `%TEMP%\rc-r-debug.log`

Use debug mode if period parsing or intervals look wrong; the log includes raw period text, suffix, computed seconds, and each run.

## Edition comparison

All three tools run a command on a repeating interval. **`r.bat`** is the smallest subset for Windows `cmd` only. **`rc.ps1`** and **`go/rc`** add precision timing, skip/limit semantics with skipped runs, expected-runtime metrics, marker replace, and success/failure exit limits.

| Feature | **r.bat** (Simplified Batch) | **rc.ps1** (PowerShell) | **go/rc** (Go binary) |
|---------|:----------------------------:|:-----------------------:|:---------------------:|
| Location | `ps/rc/r.bat` | `ps/rc/rc.ps1` | `go/rc/` (build → `rc.exe` / `rc`) |
| Version (doc) | Build stamp in script | v1.4 | v1.4 |
| **Continuous execution** | Yes | Yes | Yes |
| **Interval: `s` / `m` / `h` / bare minutes** | Yes | Yes | Yes |
| **Interactive mode** | Yes | Yes | Yes |
| **Clear mode (`-c`)** | Yes | Yes | Yes |
| **Silent mode (`-q`)** | Yes | Yes | Yes |
| **Limit runs (`-limit`)** | Yes (completed runs) | Yes (skipped runs don’t count) | Yes (skipped runs don’t count) |
| **Debug mode (`-debug`)** | Yes | — | — |
| **Standard timing** (wait after command finishes) | Yes | Yes | Yes |
| **Precision scheduling (`-p`)** | — | Yes | Yes |
| **Skip initial runs (`-Skip` / `-skip`)** | — | Yes | Yes |
| **Expected runtime (`-Expect` / `-e`)** | — | Yes | Yes |
| **Success metrics / last-success display** | — | Yes | Yes |
| **Marker replace (`^*` + `-Replace` / `-r`)** | — | Yes | Yes |
| **Failure limits (`-Fail`, `-FailTime`)** | — | Yes | Yes |
| **Success limits (`-Success`, `-SuccessTime`)** | — | Yes | Yes |
| **Shell / runtime** | `cmd.exe` only | PowerShell | Native binary (shell per OS) |
| **Cross-platform** | Windows | Windows | Windows, Linux (built binaries) |
| **Color status output** | Minimal | Yes | Yes |

### When to use which edition

| Use case | Recommended tool |
|----------|------------------|
| Double-click or batch-only PC, no PowerShell | **r.bat** |
| Quick loop with debug log for period parsing | **r.bat** (`-debug`) |
| PowerShell commands, precision grid, or `-Expect` metrics | **rc.ps1** |
| Scheduled monitoring with `-Skip`, `-Fail`, or `-Success` limits | **rc.ps1** or **go/rc** |
| Linux/macOS or a standalone `.exe` with no PowerShell | **go/rc** |
| Substitute location in command via `^*` marker | **rc.ps1** or **go/rc** |

## Notes

- Commands run through `cmd /d /c "your command"` — use syntax valid for **cmd**, not only PowerShell. To loop a PowerShell one-liner from `r.bat`, wrap it explicitly, e.g. `r.bat "powershell -NoProfile -Command \"Get-Date\"" 1m`.
- Invalid periods fall back to 5 minutes with a warning.
- Invalid `-limit` values fall back to unlimited with a warning.
- Place `r.bat` in `ps\rc` (or add that directory to your PATH) so you can run it from anywhere.

## See also

- [`../README.md`](../README.md) — full PowerShell `rc.ps1` documentation
- [`../../go/rc/README.md`](../../go/rc/README.md) — Go `rc` port documentation
