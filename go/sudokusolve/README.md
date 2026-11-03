# Sudoku Solver & OCR

**Version:** 1.0 · **Author:** Kreft & Cursor · **Date:** 2026-11-03

## Description

`Sudoku Solver & OCR` is a Windows desktop Sudoku app written in Go. Fill the 9×9 grid so every row, column, and 3×3 box contains the digits 1–9, or paste a screenshot of a puzzle and let the app read the givens.

A pasted image is accepted only when it contains a full Sudoku grid. A cropped fragment is left off the board, and the status line says so.

## Features

- **Board** — click a cell or move with the arrow keys. Typed digits and the on-screen keypad write the selected cell. Gold digits are what you entered or what OCR read. Green digits are filled in by Solve
- **Solve Puzzle** — completes a legal board. Conflicting givens are rejected instead of being solved
- **Clear Grid** — empties every cell
- **Paste Screenshot** — reads an image from the clipboard (`Ctrl+V` or the button). A progress card shows the read while it runs, and a second paste waits until the first one finishes
- **Full grid only** — OCR requires a complete 9×9. A partial crop does not fill the board
- **Clipboard text** — if the clipboard has no image, nine lines of digits (use `.`, `0`, `_`, or `-` for blanks) are loaded when at least five digits are present
- **Erase Selected Cell** — clears the highlighted cell. `0`, Backspace, and Delete do the same
- **Die** — sits in the header, green by default. Left-click loads a random puzzle of the current difficulty. Right-click advances the difficulty and the die color: green Easy, yellow Medium, red Hard, purple Diabolical, then back to Easy. Each difficulty has 250 puzzles sampled from the public-domain [Sudoku Exchange Puzzle Bank](https://github.com/grantm/sudoku-exchange-puzzle-bank)

## Controls

| Context | Input | Action |
| ------- | ----- | ------ |
| Board | Click | Select a cell |
| Board | Arrow keys | Move the selection (wraps around the board) |
| Board | 1–9 or numpad 1–9 | Enter a digit in the selected cell |
| Board | 0, Backspace, or Delete | Clear the selected cell |
| Anywhere | Ctrl+V | Paste a screenshot or a text grid |
| Buttons | Solve Puzzle | Fill the rest of a legal puzzle |
| Buttons | Clear Grid | Reset the board |
| Buttons | Paste Screenshot | Same as Ctrl+V |
| Keypad | 1–9 | Enter that digit |
| Keypad | Erase Selected Cell | Clear the selected cell |
| Die | Left click | Load a random puzzle of the current difficulty |
| Die | Right click | Next difficulty (Easy, Medium, Hard, Diabolical) and matching color |

## Paste and OCR

1. Copy a puzzle image (`Win+Shift+S`, then copy the region).
2. Focus this window and press `Ctrl+V`, or click **Paste Screenshot**.
3. The status line and the card on the grid show progress. When the read finishes, recognized givens are placed in gold.

The image needs a full 9×9 with visible grid lines. A fragment, a blank image, or a picture that is not a puzzle leaves the board as it was and reports **No Sudoku grid found in that image.**

A text grid on the clipboard looks like this:

```text
82.6...91
..1..8263
3.621.8..
6..7.19.4
..4.5.1.6
1.9..6...
.13.62..9
4981..6..
.6.....1.
```

## Build

From the `app` folder:

**All platforms (recommended)**

```powershell
pwsh -NoLogo -NoProfile -File ./build.ps1
```

Pass `-upx` to compress the binaries with [UPX](https://upx.github.io/).

**Quick local build**

```powershell
go build -ldflags "-H windowsgui" -o sudoku.exe .
```

Windows builds use `-H windowsgui` so a console window does not open with the app. If `sudoku.ico`, `sudoku.rc`, and `windres` are present, `build.ps1` embeds the icon.

**Tests**

```powershell
go test -count=1 -timeout 90s .
```

### Binary output

| Platform | Path |
| -------- | ---- |
| Windows x86 | `bin/win/x86/Sudoku.exe` |
| Windows x64 | `bin/win/x64/Sudoku.exe` |
| Linux x86 | `bin/linux/x86/Sudoku` |
| Linux amd64 | `bin/linux/amd64/Sudoku` |

## Files

| File | Role |
| ---- | ---- |
| `puzzles.txt` | 250 Easy, Medium, Hard, and Diabolical puzzles |
| `build.ps1` | Windows and Linux build |
| `bin/win/x64/Sudoku.exe` | Windows x64 app |
| `main.go` | Window, input, solver, clipboard paste |
| `ocr.go` | Grid detection and digit reading |
| `sudoku.ico` | Window icon |
| `README.md` | This document |
| `README.html` | In-browser viewer for this document |
