# a2b — ASCII to Binary Converter

PowerShell utility that converts strings between ASCII and 8-bit binary representation, with automatic direction detection.

## Modes

1. **Direct** — pass a string as an argument for a one-off conversion.
2. **Pipeline** — pipe one or more strings into the script.
3. **Interactive** — run with no arguments; convert repeatedly until you quit.

## Features

- Automatic ASCII ↔ binary detection
- Forced direction with `-ToBinary` / `-ToAscii`
- Binary output in space-separated 8-bit chunks (e.g. `01001000 01101001`)
- Spaces in binary input are ignored
- Rejects non-ASCII characters, empty/whitespace-only input, and binary lengths that are not a multiple of 8
- Direct and pipeline modes exit with code `1` on failure

### Auto-detect caveat

In auto mode, any string made only of `0`, `1`, and whitespace is treated as **binary**. To encode those characters as ASCII (e.g. `"01"`), use `-ToBinary`.

## Requirements

- PowerShell 5.1+ (Windows PowerShell or PowerShell 7+)

## Parameters

| Parameter | Description |
| --- | --- |
| `InputString` | String to convert (positional). Omit for interactive mode. |
| `-ToBinary` | Force ASCII → binary. |
| `-ToAscii` | Force binary → ASCII. |
| `-Help` / `-h` | Show built-in help and exit. |

`-ToBinary` and `-ToAscii` are mutually exclusive. Default is auto-detect.

```powershell
.\a2b.ps1 -Help
```

## Usage

### Direct mode

**ASCII → binary:**

```powershell
.\a2b.ps1 "Hello World"
```

```
01001000 01100101 01101100 01101100 01101111 00100000 01010111 01101111 01110010 01101100 01100100
```

**Binary → ASCII:**

```powershell
.\a2b.ps1 "01001000 01101001"
```

```
Hi
```

**Force ASCII → binary for digit-only input:**

```powershell
.\a2b.ps1 "01" -ToBinary
```

```
00110000 00110001
```

### Pipeline mode

```powershell
"Hi" | .\a2b.ps1
"Hello", "World" | .\a2b.ps1
```

### Interactive mode

```powershell
.\a2b.ps1
```

Enter strings at the prompt. Quit with an empty line, or type `q`, `quit`, or `exit`.

## Validation and exit codes

| Situation | Behavior |
| --- | --- |
| Valid conversion | Writes the result; exit code `0` |
| Invalid input (direct/pipeline) | Prints a short red message; exit code `1` |
| Invalid input (interactive) | Prints a short red message; loop continues |

## Files

- `a2b.ps1` — main script
- `README.md` — this file
