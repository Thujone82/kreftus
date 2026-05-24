# RC.PS1 - Run Continuously (PowerShell Edition)

## Overview

`rc.ps1` executes a PowerShell command string on a repeating interval until stopped with **Ctrl+C**, a **`-Limit`** is reached, or an expect-based **failure limit** triggers exit. It is the reference implementation; a cross-platform Go port lives in `go/rc/` with matching behavior.

**Entry points:**
- `.\rc.ps1 [[-Command] string] [[-Period] string] [switches…]`
- `.\rc.ps1 -Help` (aliases `-h`, `-?`) — inline CLI reference, then exit
- No `-Command` — interactive prompts for command, period, precision, clear, and limit

**Execution model:** `Invoke-Expression` on the command string each iteration (after optional `-Replace` substitution). Errors are caught per iteration; the loop continues unless a limit exits the script.

---

## Parameters

| Parameter | Alias | Type | Default | Description |
|-----------|-------|------|---------|-------------|
| `Command` | (positional 0) | string | (prompt) | Expression run each iteration. Quote if it contains spaces. |
| `Period` | (positional 1) | string | `5` | Wait between iterations. Period format (see below). |
| `Precision` | `p` | switch | off | Grid-aligned scheduling from script start. |
| `Silent` | `s` | switch | off | Suppress status lines; command output and warnings still show. |
| `Clear` | `c`, `cl` | switch | off | `Clear-Host` at startup (if set) and before each run. |
| `Skip` | — | int | `0` | Loop iterations that skip `Invoke-Expression` but still wait. If `-Skip` is **bound** and value is `0`, treated as `1`. |
| `Limit` | — | int | `0` | Max **actual** command runs (skipped iterations do not count). `0` = unlimited. |
| `Expect` | `e` | string | — | Minimum runtime for a **successful** run (period format). Enables success tracking and fail limits. |
| `Replace` | `r` | string | — | Replace every literal `^*` in `-Command` with this value before each run. |
| `Fail` | `f` | int | `0` | Exit after this many **failed** runs (duration &lt; `-Expect`). Requires `-Expect`. `0` = off. |
| `FailTime` | `ft` | string | — | Exit when cumulative failure cost reaches cap (period format). Requires `-Expect`. |
| `Help` | `h`, `?` | switch | — | Print full CLI reference and exit. |

Comment-based help: `Get-Help .\rc.ps1 -Full`

---

## Features

### Standard scheduling (default)
After each loop iteration (including skip-only iterations), sleeps for the full **Period** (`Start-Sleep` using `$PeriodMinutes * 60`). Simple; timing drifts if command duration varies.

### Precision mode (`-Precision` / `-p`)
- Records `$scriptStartTime` at loop entry.
- After each iteration, computes the next grid boundary:  
  `nextTarget = scriptStart + (floor(elapsedMinutes / periodMinutes) + 1) * periodMinutes`
- Sleeps until `nextTarget` if positive; otherwise warns and runs immediately (overrun recovery).
- Status line: `Runtime: … Waiting: … Next Run: …` (compact duration format).

### Silent mode (`-Silent` / `-s`)
Suppresses banners, timestamps, wait lines, expect summary, and limit messages. Command output and `Write-Warning` (command errors, soft warnings) still appear.

### Clear mode (`-Clear` / `-c` / `-cl`)
`Clear-Host` once before startup output (if not silent) and again before each command execution. **Note:** startup banners (including expect/fail config) are cleared on each run; active expect/fail settings are repeated on the `Executing command...` line.

### Skip mode (`-Skip`)
- `$executionCount` increments every loop; command runs only when `$executionCount > $Skip`.
- Skipped iterations still execute the wait/precision sleep path (schedule preserved).
- Yellow skip status unless silent.

### Limit mode (`-Limit`)
- `$actualExecutionCount` increments only on real runs.
- Green exit message when `$actualExecutionCount -ge $Limit`.

### Expect mode (`-Expect` / `-e`)
- Parses threshold via `Convert-Period` → `[TimeSpan]::FromMinutes(...)`.
- **Success:** `commandDuration >= expectThreshold` (measured loop start → end, including failed `Invoke-Expression` that still returns).
- **Failure:** duration below threshold (when `-Expect` is set).
- Tracks: `$successfulExecutionCount`, `$actualExecutionCount` (for ratio), `$totalSuccessfulRuntime`, `$lastSuccessfulRuntime`, `$lastSuccessfulCompletionTime`.
- **Summary** (after each wait in standard/precision mode, and before fail-limit exit):

  ```
  Last Success: HH:mm:ss (successes/total)
  Total Runtime: HH:mm:ss.cs (last success runtime or N/A)
  ```

- Timestamps: today → `HH:mm:ss`; other days → `MMddyy@HH:mm:ss`.

### Replace mode (`-Replace` / `-r`)
- Script constant: `$ReplaceMarker = '^*'`
- Applied once at startup: `$Command = $Command.Replace($ReplaceMarker, $Replace)`
- Marker is safe inside **double-quoted** commands (no `$` expansion issue).
- Soft warning if `-Replace` set but command does not contain `^*` (unless silent).

### Failure limits (`-Fail` / `-f`, `-FailTime` / `-ft`)
**Require `-Expect`.** If set without `-Expect`, soft warning and limits ignored.

| Limit | Behavior |
|-------|----------|
| `-Fail N` | Exit when `$failedExecutionCount >= N` |
| `-FailTime` | Exit when `$failedRetryTime >=` parsed threshold |

- Each failure adds **one configured Period** (`$periodInterval`) to `$failedRetryTime`, not precision-mode computed sleep.
- If **both** are set, **either** limit ends the loop (whichever is hit first).
- On exit: prints expect summary, then red message (`Reached failure limit…` / `Reached failure time limit…`).

### Period format (`Convert-Period`)
| Input | Meaning |
|-------|---------|
| (none) / blank | 5 minutes |
| `15s` | seconds → internal minutes |
| `5` or `5m` | minutes |
| `1h` | hours |
| invalid | 5 minutes |

Returns hashtable: `@{ Minutes = [double]; Display = "human string" }`

Used for `-Period`, `-Expect`, and `-FailTime`.

---

## Helper functions

| Function | Purpose |
|----------|---------|
| `Convert-Period` | Parse period strings with `s`/`m`/`h` suffixes |
| `Format-CompactDuration` | `HH:mm:ss`, `mm:ss`, or fractional seconds for precision status |
| `Format-DateAwareTimestamp` | Today vs other-day timestamp for Next Run / Last Success |
| `Format-ExpectConfigDetails` | Build `Expect: … \| Fail: … \| FailTime: …` for banners and execute line |
| `Write-ExpectSummaryIfNeeded` | Print success summary when `-Expect` set and `$executionCount > $Skip` |

---

## Main loop structure

```
while ($true) {
  $executionCount++
  if ($executionCount <= $Skip) {
    # skip message (optional)
  } else {
    $actualExecutionCount++
    Clear-Host if -Clear
    "Executing command..." [+ expect config bracket]
    try { Invoke-Expression } catch { warning; still record duration }
    update success/fail counters if -Expect
    check -Limit → break
    check -Fail → summary + break
    check -FailTime → summary + break
  }
  if (-Precision) { grid sleep + runtime line + expect summary }
  else { "Waiting …" + expect summary; full period sleep }
}
```

**Counters:**
- `$executionCount` — every loop iteration (including skips)
- `$actualExecutionCount` — command invocations only
- `$failedExecutionCount` / `$failedRetryTime` — only when `-Expect` active

---

## Startup output (non-silent)

1. `Running "<command>" every <period>. Press Ctrl+C to stop.`
2. Magenta expect config line (if any of Expect / Fail / FailTime active)
3. Yellow skip banner (if `$Skip > 0`)
4. Cyan limit banner (if `$Limit > 0`)
5. Cyan precision grid start time (if `-Precision`)

---

## Color scheme

| Color | Usage |
|-------|--------|
| Yellow | Interactive title, skip messages |
| Cyan | Limit banner, precision mode |
| Magenta | Expect / fail config line |
| Green | Execution limit reached |
| Red | Failure limit / failure time limit exit |
| Default | Execute line, wait messages |
| Warning | Command errors, soft warnings (replace marker, fail without expect) |

---

## Usage examples

```powershell
# Basic
.\rc.ps1 "Get-Date" 1m

# Precision + silent
.\rc.ps1 ".\my-monitor.ps1" 10m -p -s

# Skip, limit, period suffix
.\rc.ps1 "Get-Date" 30s -Skip 2 -Limit 5

# Expect tracking
.\rc.ps1 "Invoke-WebRequest https://example.com" 5s -e 1s

# Replace marker (double-quoted)
.\rc.ps1 "gf -x ^*" 5m -r pdx

# Failure limits (require -e)
.\rc.ps1 ".\task.ps1" 5m -e 30s -fail 3
.\rc.ps1 ".\task.ps1" 5s -e 1s -failtime 30s

# Help
.\rc.ps1 -Help
```

---

## Interactive mode

When `-Command` is omitted after parameter parsing:

1. Command (Read-Host)
2. Period (default 5)
3. Precision y/n
4. Clear y/n
5. Limit (0 = none)

Does **not** prompt for Expect, Replace, Fail, FailTime, or Skip — pass those on the command line.

---

## Error handling

- `Invoke-Expression` wrapped in `try/catch`; failures become warnings with duration still recorded.
- Invalid period strings fall back to 5 minutes.
- Script does not exit on command failure unless a limit triggers.

---

## Parity and related files

| Path | Role |
|------|------|
| `ps/rc/rc.ps1` | This script |
| `ps/rc/README.txt` | User-facing README |
| `go/rc/main.go` | Go port (flags anywhere in argv; `cmd`/`sh` execution) |
| `go/rc/build.ps1` | Cross-compile Windows/Linux binaries to `go/rc/bin/` |
| `rc/index.html` | Project landing page |

Go edition uses the same `^*` replace marker, expect summary, fail limits, and config display patterns. Flag names are lowercase with leading `-` (e.g. `-fail`, `-failtime`).

---

## Version history

| Version | Changes |
|---------|---------|
| **Current** | `-Expect`, `-Replace` (`^*` marker), `-Fail`, `-FailTime`, `-Help`, expect config on execute line, expect summary on fail-limit exit, consolidated expect/fail startup line |
| **v1.4** | `-Limit`, period suffixes (`s`/`m`/`h`), `-Skip` |
| **v1.3** | `-Clear` |
| **v1.0** | Core loop, `-Precision`, `-Silent` |

---

## Requirements

- PowerShell 5.1+
- Windows (primary target; script uses `Clear-Host`, `Invoke-Expression`)
- No external modules

## Stopping

- **Ctrl+C** — immediate process stop; no expect summary on interrupt (not implemented).
- **`-Limit`** — green message, no expect summary unless `-Expect` also set (summary only printed during wait / fail-limit paths today).
