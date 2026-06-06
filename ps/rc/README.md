# rc — Run Continuously (PowerShell Edition)

## Version 1.4

**Author:** Kreft&Gemini

## Description

`rc.ps1` (Run Continuously) is a flexible PowerShell utility designed to execute a given command repeatedly at a specified interval. It is ideal for simple, recurring tasks, monitoring, or any scenario where a command needs to be run in a loop without the complexity of setting up a formal scheduled task.

The script offers two modes for scheduling: a simple delay mode and a high-precision grid-based mode for tasks that require exact timing.

## Features

- **Continuous execution** — Runs any valid PowerShell command string in an infinite loop until manually stopped.
- **Configurable interval** — Set the time between runs with suffixes: `s` (seconds), `m` (minutes, optional), `h` (hours). Bare integers default to minutes.
- **Interactive mode** — Prompts for command and interval when run without parameters.
- **Standard mode (default)** — Waits for the full period after the command finishes. Simple, but timing can drift if run duration varies.
- **Precision mode (`-p`)** — Fixed-interval grid from start time; accounts for execution time so runs align to predictable moments (e.g. every 10 minutes at :00, :10, :20). Overlong runs trigger an immediate next iteration to catch up.
- **Silent mode (`-q`)** — Suppresses status lines; command output and errors still show.
- **Clear mode (`-c`)** — Clears the screen before each run.
- **Skip mode (`-Skip`)** — Skip initial loop iterations before running the command. `-Skip 0` defaults to skipping one execution.
- **Limit mode (`-Limit`)** — Stop after a set number of executions. Skipped iterations do not count.
- **Expected runtime (`-Expect` / `-e`)** — Minimum duration for a successful run; tracks and reports success metrics.
- **Command marker replace (`-Replace` / `-r`)** — Replaces every literal `^*` in the command before execution (e.g. `gf -x ^*` with `-r pdx` → `gf -x pdx`).
- **Failure limits (`-Fail` / `-f`, `-FailTime` / `-ft`)** — Exit after failed-run count or cumulative failure time. Requires `-Expect`.
- **Success limits (`-Success` / `-s`, `-SuccessTime` / `-st`)** — Exit after success count or accumulated successful runtime. Requires `-Expect`. Green exit messages.

## Requirements

- PowerShell

## How to Run

1. Open a PowerShell terminal.
2. Navigate to the directory where `rc.ps1` is located.
3. Run the script using one of the examples below.
4. Press **Ctrl+C** to stop.

## Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `Command` | string (positional 0) | PowerShell command each iteration. Quote if it contains spaces. Prompted if omitted. |
| `Period` | string (positional 1) | Wait between runs: `s`, `m`, `h`, or bare minutes. Default: `5`. Examples: `5`, `15s`, `5m`, `1h`. |
| `-Precision` / `-p` | switch | Enable precision grid scheduling. |
| `-Silent` / `-q` | switch | Suppress status output. |
| `-Clear` / `-c` | switch | Clear screen before each run. |
| `-Skip` | int | Skip this many initial iterations before running the command. `-Skip 0` → skip 1. Default: 0 (skip none). |
| `-Limit` | int | Max executions; skipped runs don't count. `0` = unlimited. |
| `-Expect` / `-e` | string | Minimum successful runtime (period format). Enables success/fail metrics after each run. |
| `-Replace` / `-r` | string | Substitute value for every `^*` marker in the command. Warns if no marker present. |
| `-Fail` / `-f` | int | Exit after this many failed runs. Requires `-Expect`. |
| `-FailTime` / `-ft` | string | Exit when cumulative failure time (failures × period) reaches cap. Requires `-Expect`. |
| `-Success` / `-s` | int | Exit after this many successful runs. Requires `-Expect`. |
| `-SuccessTime` / `-st` | string | Exit when accumulated successful runtime reaches cap. Requires `-Expect`. |

When `-Expect` is set, each run prints last success time, `successes/total`, total successful runtime, and last successful runtime. Before the first success, timestamps show `N/A`.

## Examples

### Simple task

```powershell
.\rc.ps1 "Get-Process -Name 'chrome' | Stop-Process -Force" 1
```

Stops Chrome every minute; the timer starts after each run finishes.

### Running another script

```powershell
.\rc.ps1 "gw Portland" 10
```

Runs `gw.ps1` with parameter `Portland` every 10 minutes.

### High-precision logging

```powershell
.\rc.ps1 ".\my-data-logger.ps1" 10 -Precision
```

Fixed 10-minute grid; if a run at 10:00:00 takes 20 seconds, the next run starts at 10:10:00.

### Silent mode

```powershell
.\rc.ps1 "Get-Date" 1 -Silent
```

### Combined precision and silent

```powershell
.\rc.ps1 ".\my-monitor.ps1" 5 -Precision -Silent
```

### Clear mode

```powershell
.\rc.ps1 "Get-Date" 1 -Clear
```

### Skip mode

```powershell
.\rc.ps1 "Get-Process" 5 -Skip 2
```

Skips the first two iterations, then runs every 5 minutes.

### Skip default (skip one)

```powershell
.\rc.ps1 "Get-Date" 1 -Skip 0
```

`-Skip 0` defaults to skipping one execution.

### Period suffixes

```powershell
.\rc.ps1 "Get-Process" 15s
.\rc.ps1 ".\backup.ps1" 1h
```

### Limit mode

```powershell
.\rc.ps1 "Get-Process" 5 -Limit 3
```

### Skip and limit together

```powershell
.\rc.ps1 "Get-Date" 30s -Skip 2 -Limit 5
```

### Expected runtime threshold

```powershell
.\rc.ps1 "Invoke-WebRequest https://kreft.us" 5s -Expect 1s
.\rc.ps1 "Get-Date" 1m -e 1s
```

### Command marker replace

```powershell
.\rc.ps1 "gf -x ^*" 5 -r pdx
```

Runs `gf -x pdx` every 5 minutes.

### Failure limits

```powershell
.\rc.ps1 "Get-Date" 5m -e 30s -fail 3
.\rc.ps1 "Get-Date" 5s -e 1s -failtime 30s
```

### Success limits

```powershell
.\rc.ps1 "Get-Date" 5m -e 30s -success 2
.\rc.ps1 "Get-Date" 5s -e 1s -successtime 30s
```

## Notes

- Press **Ctrl+C** at any time to stop.
- If `-Fail` / `-FailTime` / `-Success` / `-SuccessTime` are set without `-Expect`, rc warns and ignores those limits.
