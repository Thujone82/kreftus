# RC - Run Continuously (Go Edition)

## Overview

Cross-platform `rc` binary that runs a shell command string on a repeating interval until **Ctrl+C**, **`-limit`**, or an expect-based **failure limit** exits the process. Behavior matches the PowerShell reference implementation in `ps/rc/rc.ps1`.

**Source:** `go/rc/main.go`  
**Build:** `go/rc/build.ps1` → `go/rc/bin/` (Windows x86/x64, Linux x86/amd64)  
**Help:** `rc -help` (aliases `-h`)

**Argument parsing:** Manual loop over `os.Args` (not the `flag` package) so switches may appear **before or after** the command and period. Non-flag tokens: first = command; additional tokens scanned for the first valid period string.

**Execution:** `cmd /C` on Windows, `sh -c` elsewhere. Stdout/stderr piped through. Non-zero exit prints yellow warning; loop continues.

---

## Flags and positional arguments

| Flag | Aliases | Value | Default | Description |
|------|---------|-------|---------|-------------|
| (positional) | — | string | (prompt) | Command string. Quote if it contains spaces. |
| (positional) | — | string | `5` | Period (see format below). |
| `-precision` | `-p` | — | off | Grid-aligned scheduling from process start. |
| `-silent` | `-s` | — | off | Suppress status lines; command output and warnings remain. |
| `-clear` | `-c` | — | off | Clear screen at startup (if set) and before each run. |
| `-skip` | `-Skip` | int | `0` | Loop iterations that skip command execution but still wait. If flag **present** and value `0`, treated as `1`. |
| `-limit` | `-Limit` | int | `0` | Max actual command runs (skips do not count). `0` = unlimited. |
| `-expect` | `-e`, `-Expect` | period | — | Minimum runtime for a successful run. |
| `-replace` | `-r`, `-Replace` | string | — | Replace every literal `^*` in the command before each run. |
| `-fail` | `-f`, `-Fail` | int | `0` | Exit after N failed runs (&lt; expect). Requires `-expect`. |
| `-failtime` | `-ft`, `-FailTime` | period | — | Exit when cumulative failure cost reaches cap. Requires `-expect`. |
| `-help` | `-h` | — | — | Print usage (`printUsage`) and exit. |

**Constants:** `replaceMarker = "^*"` (package-level in `main.go`)

---

## Features

### Standard scheduling (default)
After each loop iteration (including skip-only iterations), `time.Sleep(periodDuration)` for the full configured period.

### Precision mode (`-p` / `-precision`)
- `scriptStartTime` set when precision is enabled (at loop entry, after banners).
- Next boundary:  
  `nextTarget = scriptStart + (floor(elapsedMinutes / periodMinutes) + 1) * periodDuration`
- Positive sleep → status line `Runtime: … Waiting: … Next Run: …` plus expect summary.
- Non-positive sleep → yellow overrun warning, immediate next iteration.

### Silent mode (`-silent` / `-s`)
No banners, execute line, wait lines, expect summary, or limit messages. `executeCommand` output and `color.Yellow` warnings still appear.

### Clear mode (`-clear` / `-c`)
- Windows: `cmd /C cls`
- Unix: ANSI `\033[2J\033[H`

Clears before startup output (once) and before each command. Expect/fail config is repeated on the `Executing command...` line because startup text is cleared each run.

### Skip mode (`-skip`)
- `executionCount` increments every loop; command runs when `executionCount > skip`.
- Skipped iterations still run precision/standard sleep.
- Yellow skip status unless silent.

### Limit mode (`-limit`)
- `actualExecutionCount` counts only real runs.
- Green exit message when limit reached.

### Expect mode (`-expect` / `-e`)
- `expectState` holds threshold, display string, success counters, runtimes, last completion time.
- **Success:** `commandDuration >= expect.threshold`
- **Failure:** below threshold when expect is set.
- **Summary** (standard + precision waits, and before fail-limit exit):

  ```
  Last Success: HH:mm:ss (successes/total)
  Total Runtime: HH:mm:ss.cs (last success runtime or N/A)
  ```

- Timestamps: same day → `15:04:05`; else `010206@15:04:05`.

### Replace mode (`-replace` / `-r`)
- `applyReplace()` at startup: `strings.ReplaceAll(commandStr, "^*", replaceValue)`
- Marker works in double-quoted commands (no shell `$` issues on Unix; no PS `$` expansion on Windows when using `cmd /C` with a quoted string).
- Soft warning if `-replace` set but marker missing (unless silent).

### Failure limits (`-fail` / `-f`, `-failtime` / `-ft`)
**Require `-expect`.** Without expect: yellow warning, limits ignored.

| Limit | Exit when |
|-------|-----------|
| `-fail N` | `failedExecutionCount >= N` |
| `-failtime` | `failedRetryTime >=` parsed duration |

- Each failure adds **`periodDuration`** (configured interval), not precision sleep.
- Both set → **either** limit ends the loop first.
- Exit path: `printExpectSummary` then red failure message.

### Period format (`parsePeriod`)
| Input | Result |
|-------|--------|
| empty | 5 minutes |
| `15s`, `5m`, `1h` | suffixed duration + display string |
| `5` (no suffix) | minutes |
| parse error | 5 minutes (fallback) |

Used for period, `-expect`, and `-failtime`.

---

## Types and functions

| Symbol | Purpose |
|--------|---------|
| `expectState` | Threshold, display, success/fail accounting fields |
| `parsePeriod` | Parse `s`/`m`/`h` → `time.Duration` + display |
| `formatCompactDuration` | Precision status line durations |
| `formatDateAwareTimestamp` | Next run / last success timestamps |
| `formatSuccessRuntime` | `HH:mm:ss.cs` for summary lines |
| `formatExpectConfigDetails` | `Expect: … \| Fail: … \| FailTime: …` |
| `printExpectSummary` | Success summary when expect set and `executionCount > skip` |
| `applyReplace` | `^*` substitution + warning |
| `clearScreen` | Platform-specific clear |
| `executeCommand` | `cmd /C` or `sh -c` |
| `printUsage` | Colored help text |

---

## Main loop structure

```
for {
  executionCount++
  if executionCount <= skip {
    // skip message
  } else {
    actualExecutionCount++
    clearScreen if -clear
    "Executing command..." [+ expect config]
    executeCommand
    measure duration; update expect success/fail
    check -limit → break
    check -fail → summary + break
    check -failtime → summary + break
  }
  if precision { grid sleep + status + summary }
  else { wait message + summary; sleep(period) }
}
```

**Failure accounting:** `failedExecutionCount`, `failedRetryTime` — only when `expect != nil`.

---

## Startup output (non-silent)

1. `Running "<command>" every <period>. Press Ctrl+C to stop.`
2. Magenta `expectConfigDetails` line (if any)
3. Yellow skip banner (if `skip > 0`)
4. Cyan limit banner (if `limit > 0`)
5. Cyan precision grid start (if `-p`)

---

## Color scheme (`github.com/fatih/color`)

| Color | Usage |
|-------|--------|
| Yellow | Interactive title, skip, command-failure warnings, soft warnings |
| Cyan | Limit banner, precision mode |
| Magenta | Expect / fail config line |
| Green | Execution limit reached |
| Red | Failure limit / failure time exit |
| White | Execute line, runtime/wait status, Ctrl+C hint |

---

## Usage examples

```sh
# Basic
./rc "date" 1m

# Flags anywhere
./rc -p -s "my-monitor.sh" 5m

# Period suffix, skip, limit
./rc "date" 30s -skip 2 -limit 5

# Expect
./rc "echo test" 3s -e 1s -limit 2

# Replace (^* marker)
./rc "gf -x ^*" 5m -r pdx

# Failure limits
./rc "date" 5m -e 30s -fail 3
./rc "date" 5s -e 1s -failtime 30s

# Help
./rc -help
```

---

## Interactive mode

When no command in argv after parsing:

1. Command  
2. Period (default 5)  
3. Precision y/n  
4. Clear y/n  
5. Skip (empty = 0 no skip; `0` entered → defaults to 1)  
6. Limit (0 = none)

Does **not** prompt for `-expect`, `-replace`, `-fail`, or `-failtime` — pass on command line.

---

## Building and distribution

```powershell
cd go/rc
.\build.ps1          # requires Go + windres (MinGW) for Windows icon embed
.\build.ps1 -upx     # optional UPX compression
```

Outputs under `go/rc/bin/`:
- `win/x86/rc.exe`, `win/x64/rc.exe`
- `linux/x86/rc`, `linux/amd64/rc`

**Dependencies (build time):** Go module `github.com/fatih/color`, `golang.org/x/sys` (transitive). Compiled binaries have no runtime deps beyond libc/OS.

**Resources:** `rc.rc`, `cli_rc_icon.ico` embedded on Windows via `windres`.

---

## Platform notes

| OS | Command | Clear |
|----|---------|-------|
| Windows | `cmd /C <command>` | `cmd /C cls` |
| Linux / macOS | `sh -c <command>` | ANSI escape |

macOS is supported by the Unix code paths; release binaries are built for Windows and Linux per `build.ps1`.

---

## Parity with PowerShell edition

| Area | Notes |
|------|--------|
| Period / expect / failtime parsing | Equivalent rules; Go uses `time.Duration`, PS uses minutes → `TimeSpan` |
| Replace marker | `^*` both editions |
| Fail limits | Same require-expect, either-limit-wins, period-based failtime |
| Config display | Startup line + execute-line bracket |
| Interactive | Go prompts skip; PS does not prompt skip in interactive mode |
| Command execution | PS `Invoke-Expression`; Go shell wrapper |
| Help | PS inline here-string + `Get-Help`; Go `printUsage()` |

Reference: `ps/rc/GEMINI.md`, `ps/rc/rc.ps1`

---

## Related files

| Path | Role |
|------|------|
| `go/rc/main.go` | Implementation |
| `go/rc/README.txt` | User README |
| `go/rc/build.ps1` | Cross-compile + strip |
| `ps/rc/rc.ps1` | PowerShell reference |
| `rc/index.html` | Project landing page |

---

## Version history

| Version | Changes |
|---------|---------|
| **Current** | `-expect`, `-replace` (`^*`), `-fail`, `-failtime`, `-help`, expect config on execute line, summary on fail-limit exit, consolidated startup config line |
| **v1.4** | `-limit`, period suffixes, `-skip` |
| **v1.3** | `-clear` |
| **v1.0** | Core loop, precision, silent, cross-platform build |

---

## Requirements

- Go 1.16+ to build from source  
- Compiled binary: no Go runtime required  
- Stopping: **Ctrl+C** (no expect summary on interrupt — not implemented)
