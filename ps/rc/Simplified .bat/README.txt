# r.bat - Run Continuously (Simplified Batch Edition)

## Version
See build stamp in r.bat (e.g. RC_BUILD=20260529-debug3)

## Author
Kreft&Cursor

## Description
`r.bat` is a minimal Windows batch script that runs a command in a loop until you press Ctrl+C or a run limit is reached. It is a lightweight alternative to `rc.ps1` in the parent `ps/rc` folder: no PowerShell required, suitable for `cmd.exe` only environments or quick double-click use.

Timing is simple: after each command finishes, the script waits for the configured interval, then runs again. There is no precision grid scheduling and no expect/fail runtime tracking (use full `rc.ps1` or `go/rc` for those features).

## Features
- **Continuous execution:** Runs any command string via `cmd /d /c` until stopped.
- **Configurable interval:** Period accepts `s`, `m`, or `h` suffixes; a bare number means minutes (e.g. `5`, `15s`, `5m`, `1h`). Default: 5 minutes.
- **Interactive mode:** Run `r.bat` with no arguments to be prompted for command, period, and optional flags.
- **Clear mode (`-c`):** Clears the screen before each run.
- **Silent mode (`-s`):** Suppresses status lines; command output still appears.
- **Limit mode (`-limit N`):** Stops after N completed runs (0 = unlimited).
- **Debug mode (`-debug`):** Prints `[DEBUG]` lines and appends to `%TEMP%\rc-r-debug.log` (period parsing, run count, errors).

## Requirements
- Windows Command Prompt (`cmd.exe`)
- No PowerShell or Go runtime required

## How to Run
1. Open Command Prompt.
2. Change to this directory (quotes recommended because of the space in the folder name):
   cd /d "C:\path\to\kreftus\ps\rc\Simplified .bat"
3. Run `r.bat` or `r` (if `.BAT` is associated with cmd).
4. Press **Ctrl+C** to stop (unless `-limit` exits first).

## Command-Line Usage

  r.bat "command" [period] [flags...]

### Positional arguments
- **command** (required) — Command to run each iteration. Use quotes if it contains spaces.
- **period** (optional) — Interval between runs. Default: `5` (minutes).

### Flags (any order after period)
- **-c** — Clear screen before each run.
- **-s** — Silent mode (minimal status output).
- **-limit N** — Exit after N runs. `0` or omitted = unlimited.
- **-debug** — Enable debug output and log file.

Unknown flags print a warning and are ignored.

## Examples

### Run every 10 seconds, five times
  r.bat "echo hello" 10s -limit 5

### Monitor a script every 5 minutes with a clean screen
  r.bat "gf -x Portland" 5m -c

### Quick test with debug logging
  r.bat "echo yes" 10s -limit 1 -debug

### Interactive (prompts for command, period, flags)
  r.bat

  Command: echo yes
  Period (5, 15s, 5m, 1h) [default: 5]: 10s
  Flags [-c -s -limit N -debug] (optional): -debug

## Debug log
When `-debug` is set:
- Console shows lines prefixed with `[DEBUG]`.
- Full trace is appended to: `%TEMP%\rc-r-debug.log`

Use debug mode if period parsing or intervals look wrong; the log includes raw period text, suffix, computed seconds, and each run.

## Not included (use full rc.ps1 / go rc)
The simplified batch edition does **not** implement:
- Precision scheduling (`-p` / fixed grid)
- Skip initial runs (`-Skip`)
- Expected runtime / success metrics (`-Expect`, `-e`)
- Command marker replace (`^*` with `-Replace`, `-r`)
- Failure limits (`-Fail`, `-FailTime`)
- Period parsing beyond s/m/h and bare minutes

See `..\README.txt` and `..\..\go\rc\README.txt` for the full RC toolset.

## Notes
- Commands run through `cmd /d /c "your command"` — use syntax valid for `cmd`, not only PowerShell.
- Invalid periods fall back to 5 minutes with a warning.
- Invalid `-limit` values fall back to unlimited with a warning.
- Place this folder on your PATH or create a shortcut to `r.bat` if you want to run it from anywhere.
