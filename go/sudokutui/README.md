# Sudoku — Terminal Puzzle Game

**Version:** 1.5 · **Author:** Kreft & Cursor · **Date:** 2026-08-19

## Description

`Sudoku` is a cross-platform terminal Sudoku game written in Go. Fill the 9×9 grid so every row, column, and 3×3 box contains the digits 1–9. Given clues are locked. Your entries use a distinct color per digit; a correct entry locks the same way so it cannot be changed by accident. A wrong digit stays in the cell and the box turns red so you can correct it.

Puzzles come from the public-domain [Sudoku Exchange Puzzle Bank](https://github.com/grantm/sudoku-exchange-puzzle-bank) (Grant McLean), graded into Easy, Medium, Hard, and Diabolical. The bundled set has 2500 Easy, 2000 Medium, 1000 Hard, and 250 Diabolical puzzles. Solved puzzles are never offered again.

On Windows, Sudoku switches the console to UTF-8 and prefers **Cascadia Mono** when installed so box-drawing characters render cleanly.

## Features

- **Start menu** — difficulty header (`── Medium ──`, etc.), then **Continue** (when that difficulty has a save), **New Game**, Quit. A boxed **Stats** panel below (Remaining always; Perfect through Average Completion after the first win). When that difficulty has a Best/Fastest recording, ↓ past Quit highlights the Stats header as **Replay** and plays that run in a tight 9×9 box; ↑ restores Stats; ↓ from Replay wraps to Continue or New Game. ←→ still changes difficulty
- **Accent color** — random hue each launch; shifts as you move in the menu or on the board, start a game, or lock a correct digit (wrong digits jump eight steps the other way). In **pencil** mode, movement and marks rotate the other direction
- **Pencil marks** — Tab toggles ✒️ pen / ✏️ pencil (footer shows `Tab/Shift ✏️` / `Tab ✒️`). **Hold Shift** to mark without toggling the saved mode. New Game always starts in pen mode; Continue restores the saved mode. Empty cells can hold two *different* candidate colors on a `▀` glyph (top = first mark, bottom = second). Further marks overwrite top, then bottom. Pressing a digit already on the cell removes it and that half is filled next. A digit that is already complete (all nine placed) is ignored in pen and pencil. A correct lock-in removes that color from pencil marks in the same row, column, and box; completing a number removes it everywhere. A leftover mark drops to the bottom half so the top is next. 0 clears marks. The grid border turns light yellow in pencil mode
- **Grid flash** — green ~0.6s on a correct lock-in, red on a mistake
- **Difficulty stats** — Perfect (zero-error wins), Successes, Error Rate (average incorrect entries per success), Failed, Best (fewest mistakes, then fastest) until a perfect exists then Fastest, Average Completion (all solves, then perfects only once you have one), and remaining puzzles update as you change difficulty
- **Continue Game** — one in-progress save per difficulty; progress (including the compact move log) is saved after every move
- **Colored digits** — 1–9 each have a distinct hue around the color wheel; a digit turns **white** when all nine of that number are correctly placed. Under the board, **Active:** lists the numbers that are not finished yet, in those same hues. Accent-colored ▶ ◀ ▼ ▲ mark the selected row and column
- **Mistake tally** — the HUD shows a `×` for each incorrect entry (no count label)
- **Red cell** — incorrect entry against the solution
- **Play clock** — elapsed time in the top-right; pause freezes it
- **Pause** — Space hides the board and freezes the clock; the rainbow SUDOKU banner stays on screen, with difficulty, a fill progress bar, and Pencil Marks / Errors when those counts are above zero
- **Solved** — joined double-line frame (accent, accent+5 title bar, accent−5 hint). Left: Time and **Errors**, or **Perfect Finish!** when there were none; **New Best Completion!** or **New Fastest Completion!** when this run is the new record (Best until a perfect exists, then Fastest). Right: a borderless 9×9 playback of the run compressed into 5 seconds, then the solved digits cycle their hues at 5 fps for 3 seconds before looping

## Controls

| Context | Input | Action |
| ------- | ----- | ------ |
| Menu | ↑↓ or W/S | Move selection (↓ past Quit opens Replay when a Best/Fastest recording exists; ↓ from Replay wraps to the first item) |
| Menu | ←→ or A/D | Change difficulty |
| Menu | Enter | Continue or start New Game |
| Menu | Esc | Quit |
| In game | Arrows or WASD | Move cursor (wraps around the board) |
| In game | Tab | Toggle ✒️ pen / ✏️ pencil |
| In game | Hold Shift | Momentary pencil (HUD + 1–9 marks; does not change Tab mode). Keypad 1–9 mark; arrows and WASD still move |
| In game | 1–9 | Enter a digit (pen) or add/toggle a pencil mark; completed numbers are ignored |
| In game | 0, Backspace, or Delete | Clear the cell (pen) or clear pencil marks |
| In game | Space | Pause / resume |
| In game | Esc | Return to the menu (game is saved) |
| Pause | Space | Resume |
| Pause | Esc | Return to the menu (game is saved) |
| New Game confirm | ←→ | Cancel / Abandon (defaults to Cancel) |
| New Game confirm | Enter, A, or C | Confirm selection (A abandon, C cancel) |
| New Game confirm | Esc | Cancel |
| Solved | Enter or Esc | Return to menu |
| Anywhere | Ctrl+C | Exit (in-progress games are saved for Continue) |

## Difficulties

| Level | Bundled | Typical techniques |
| ----- | ------- | ------------------ |
| Easy | 2500 | Singles and basic elimination |
| Medium | 2000 | Intermediate |
| Hard | 1000 | Advanced patterns |
| Diabolical | 250 | The hardest graded puzzles in the bundled set |

New Game picks a **random unsolved** puzzle at the selected difficulty. If that difficulty already has an in-progress game, you are asked to **Cancel** or **Abandon** (Abandon counts as **Failed** and only wipes that difficulty's save). **Esc** during play or pause returns to the menu and keeps the save. Finishing immediately records a **Success**, writes the save, and wipes that difficulty's Continue so the menu will not offer that game again. **Best** for that difficulty is the win with the fewest incorrect entries (ties prefer the faster time). After a perfect, that row becomes **Fastest** and only a quicker perfect replaces it. That run's playback is stored with the stats; ↓ past Quit on the menu plays it.

When every puzzle at a difficulty is solved, New Game is disabled for that level.

## Save file

`sudoku.json` is created in the current working directory. It stores:

- Per-difficulty Perfect, Successes, Error Rate, Failed, Best (or Fastest after a perfect), and Average Completion
- The Best/Fastest run's playback (`id`, givens, and move log) stored with those stats
- Completed puzzles (`id`, mistake count, and elapsed time) so they are not shown again
- One in-progress board per difficulty (pencil marks, clock, mistake count, and move log) for **Continue Game**. A continue started before 1.5 (no move log) is kept: playback jumps to that saved board, then records only the rest of the run.

The file is UTF-8 with BOM. Progress is kept in memory on each move and flushed to disk shortly after, or immediately on pause, return to menu, quit, or finish.

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
