# Sudoku — Terminal Puzzle Game

**Version:** 1.2 · **Author:** Kreft & Cursor · **Date:** 2026-08-14

## Description

`Sudoku` is a cross-platform terminal Sudoku game written in Go. Fill the 9×9 grid so every row, column, and 3×3 box contains the digits 1–9. Given clues are locked. Your entries use a distinct color per digit; a correct entry locks the same way so it cannot be changed by accident. A wrong digit stays in the cell and the box turns red so you can correct it.

Puzzles come from the public-domain [Sudoku Exchange Puzzle Bank](https://github.com/grantm/sudoku-exchange-puzzle-bank) (Grant McLean), graded into Easy, Medium, Hard, and Diabolical. The bundled set has 2500 Easy, 2000 Medium, 1000 Hard, and 250 Diabolical puzzles. Solved puzzles are never offered again.

On Windows, Sudoku switches the console to UTF-8 and prefers **Cascadia Mono** when installed so box-drawing characters render cleanly.

## Features

- **Start menu** — Continue Game (only when a game is in progress), **New Game: ◀ difficulty ▶**, Quit
- **Accent color** — random hue each launch; shifts as you move in the menu or on the board, start a game, or lock a correct digit (wrong digits jump eight steps the other way). In **pencil** mode, movement and marks rotate the other direction
- **Pencil marks** — Tab toggles ✒️ pen / ✏️ pencil (footer shows `Tab ✏️` / `Tab ✒️`). Play always starts in pen mode. Empty cells can hold two *different* candidate colors on a `▀` glyph (top = first mark, bottom = second). Further marks overwrite top, then bottom. A digit that is already complete (all nine placed) is ignored in pen and pencil. A correct lock-in removes that color from pencil marks in the same row, column, and box; completing a number removes it everywhere. A leftover mark drops to the bottom half so the top is next. 0 clears marks. The grid border turns light yellow in pencil mode
- **Grid flash** — green ~0.6s on a correct lock-in, red on a mistake
- **Difficulty stats** — Perfect (zero-error wins), Successes, Error Rate (average incorrect entries per success), Failed, Fastest Completion, and remaining puzzles update as you change difficulty
- **Continue Game** — progress is saved after every move
- **Colored digits** — 1–9 each have a distinct hue around the color wheel; a digit turns **white** when all nine of that number are correctly placed. Under the board, **Active:** lists the numbers that are not finished yet, in those same hues. Accent-colored ▶ ◀ ▼ ▲ mark the selected row and column
- **Mistake tally** — the HUD shows a `×` for each incorrect entry (no count label)
- **Red cell** — incorrect entry against the solution
- **Play clock** — elapsed time in the top-right; pause freezes it
- **Pause** — Space hides the board and freezes the clock

## Controls

| Context | Input | Action |
| ------- | ----- | ------ |
| Menu | ↑↓ or W/S | Move selection |
| Menu | ←→ or A/D | Change difficulty |
| Menu | Enter | Continue or start New Game |
| Menu | Esc | Quit |
| In game | Arrows or WASD | Move cursor (wraps around the board) |
| In game | Tab | Toggle ✒️ pen / ✏️ pencil |
| In game | 1–9 | Enter a digit (pen) or add a pencil mark; completed numbers are ignored |
| In game | 0, Backspace, or Delete | Clear the cell (pen) or clear pencil marks |
| In game | Space | Pause / resume |
| In game | Esc | Exit — Abandon or Quit Sudoku |
| Pause | Space | Resume |
| Pause | Esc | Exit — Abandon or Quit Sudoku |
| Exit overlay | ←→ | Abandon / Quit (defaults to Quit) |
| Exit overlay | Enter, A, or Q | Confirm selection (A abandon, Q quit) |
| Exit overlay | Esc | Back to the puzzle |
| Solved | Enter or Esc | Return to menu |
| Anywhere | Ctrl+C | Exit (in-progress game is saved for Continue) |

## Difficulties

| Level | Bundled | Typical techniques |
| ----- | ------- | ------------------ |
| Easy | 2500 | Singles and basic elimination |
| Medium | 2000 | Intermediate |
| Hard | 1000 | Advanced patterns |
| Diabolical | 250 | The hardest graded puzzles in the bundled set |

New Game picks a **random unsolved** puzzle at the selected difficulty. **Abandon** (from Esc while playing, or starting New Game while one is in progress) wipes the in-progress save and counts as **Failed**. **Quit** from the Esc overlay exits and keeps Continue for the next launch. Finishing immediately records a **Success**, writes the save, and wipes Continue so the menu will not offer that game again. Fastest time for that difficulty is kept (ties prefer fewer incorrect entries).

When every puzzle at a difficulty is solved, New Game is disabled for that level.

## Save file

`sudoku.json` is created in the current working directory. It stores:

- Per-difficulty Perfect, Successes, Error Rate, Failed, and Fastest Completion
- IDs of completed puzzles (so they are not shown again)
- The in-progress board, pencil marks, clock, and mistake count for **Continue Game**

The file is UTF-8 with BOM and is rewritten after each move.

## Console notes

- Prefer **Cascadia Mono** (or another modern Unicode font) so `╔═╤╗║` and related glyphs render cleanly. On Windows, Sudoku selects Cascadia Mono automatically when present.
- On launch Sudoku requests an **80×24** console. Windows Terminal and some hosts may ignore resize requests — set the profile size manually if needed.

## Build

From the `go/sudokutui` folder:

**All platforms (recommended)**

```powershell
pwsh -NoLogo -NoProfile -File ./build.ps1
```

**Quick local build**

```powershell
go build -o Sudoku.exe
```

**Run without a binary**

```powershell
go run .
```

### Binary output

| Platform | Path |
| -------- | ---- |
| Windows x86 | `bin/win/x86/Sudoku.exe` |
| Windows x64 | `bin/win/x64/Sudoku.exe` |
| Linux x86 | `bin/linux/x86/Sudoku` |
| Linux amd64 | `bin/linux/amd64/Sudoku` |

If `sudoku.ico` and `sudoku.rc` are present and `windres` is available, Windows builds embed the icon automatically. Pass `-upx` to compress binaries with [UPX](https://upx.github.io/).
