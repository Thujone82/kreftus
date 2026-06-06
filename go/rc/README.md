# rc — Run Continuously (Go Edition)

## Version 1.4

**Author:** Kreft&Gemini

## Description

`rc` is a cross-platform command-line utility written in Go that executes a given command string in a loop at a specified interval. It is a direct port of the `rc.ps1` PowerShell script—a lightweight, dependency-free executable.

Two timing modes are available: standard (wait the full period after each run) and precision (maintain a strict schedule and prevent drift over long periods).

## Features

- **Command execution** — Runs any command the system shell can execute (`cmd.exe` on Windows, `sh` on Linux/macOS).
- **Configurable interval** — Period suffixes: `s`, `m` (optional), `h`; bare integers default to minutes.
- **Standard mode (default)** — Waits the full period after the command finishes.
- **Precision mode (`-p`)** — Accounts for execution time so each run starts on a fixed schedule.
- **Silent mode (`-q` / `-quiet`)** — Suppresses status lines; command output and errors remain.
- **Clear mode (`-c`)** — Clears the screen before each run.
- **Skip mode (`-skip`)** — Skip initial iterations before running the command. `-skip 0` defaults to skipping one.
- **Limit mode (`-limit`)** — Stop after a set number of executions. Skipped iterations do not count.
- **Expected runtime (`-e` / `-expect`)** — Minimum duration for success; prints metrics after each run.
- **Command marker replace (`-r` / `-replace`)** — Substitutes a value for every literal `^*` in the command.
- **Failure limits (`-f` / `-fail`, `-ft` / `-failtime`)** — Exit on failed-run count or cumulative failure time. Requires `-expect`.
- **Success limits (`-s` / `-success`, `-st` / `-successtime`)** — Exit on success count or accumulated successful runtime. Requires `-expect`.
- **Interactive mode** — Prompts for command, period, and options when run with no arguments.
- **Cross-platform** — `build.ps1` compiles native Windows and Linux binaries.
- **Color-coded output** — Status and timing feedback in the terminal.

## Requirements

- Go (to build from source). The compiled executable has no external dependencies.

## How to Run

1. (Optional) Run `build.ps1` to compile for your platform.
2. Open a terminal or command prompt.
3. Navigate to the `rc` executable directory.
4. Run using one of the examples below.

## Command-line flags

| Flag | Description |
|------|-------------|
| `[command]` | Command string to execute (usually last argument). Quote if it contains spaces. |
| `[period]` or `-period <value>` | Interval between runs. Default: `5` (minutes). Examples: `5`, `15s`, `5m`, `1h`. |
| `-p`, `-precision` | Precision grid scheduling. |
| `-q`, `-quiet` | Silent mode. |
| `-c`, `-clear` | Clear screen before each run. |
| `-skip <n>` | Skip initial iterations. `-skip 0` → skip 1. Default: 0. |
| `-limit <n>` | Max executions; skipped runs don't count. `0` = unlimited. |
| `-e`, `-expect <period>` | Minimum successful runtime; enables success summary. |
| `-r`, `-replace <string>` | Replace every `^*` marker. Warns if marker missing. |
| `-f`, `-fail <n>` | Exit after N failed runs. Requires `-expect`. |
| `-ft`, `-failtime <period>` | Exit when failure cost (failures × period) reaches cap. Requires `-expect`. |
| `-s`, `-success <n>` | Exit after N successful runs. Requires `-expect`. |
| `-st`, `-successtime <period>` | Exit when accumulated successful runtime reaches cap. Requires `-expect`. |

When both count and time limits are set for failure or success, rc exits when **either** limit is reached first.

## Examples

### Every 10 minutes

```sh
./rc -period 10 "gw Portland"
```

### Precision every minute

```sh
./rc -p -period 1 ".\my-data-logger.ps1"
```

### Silent mode

```sh
./rc -q -period 1 "date"
```

### Precision and silent

```sh
./rc -p -q -period 5 "my-monitor.sh"
```

### Clear mode

```sh
./rc -c -period 1 "date"
```

### Interactive mode

```sh
./rc
```

### Skip mode

```sh
./rc -skip 2 -period 5 "Get-Process"
./rc -skip 0 -period 1 "date"
```

### Period suffixes

```sh
./rc "Get-Process" 15s
./rc ".\backup.sh" 1h
```

### Limit mode

```sh
./rc "Get-Process" 5 -limit 3
./rc "date" 30s -skip 2 -limit 5
```

### Command marker replace

```sh
./rc "gf -x ^*" 5 -r pdx
```

### Expected runtime

```sh
./rc "echo test" 3s -e 1s -limit 2
```

### Failure and success limits

```sh
./rc "date" 5m -e 30s -fail 3
./rc "date" 5s -e 1s -failtime 30s
./rc "date" 5m -e 30s -success 2
./rc "date" 5s -e 1s -successtime 30s
```

## Notes

- Press **Ctrl+C** to stop at any time.
- Run `./rc -help` for the full CLI reference.
