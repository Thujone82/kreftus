# Cursor Project File

## Project: sudokutui (Sudoku)

**Author:** Kreft&Cursor  
**Date:** 2026-08-14  
**Version:** 1.2

---

### Description

`sudokutui` is a cross-platform terminal Sudoku game written in Go with [`tcell`](https://github.com/gdamore/tcell). The Windows/Linux executable is named **Sudoku**. UTF-8 box-drawing frames a 9×9 board; each digit 1–9 has a distinct color. Wrong entries keep the digit and paint the cell maroon. A play clock sits on the HUD; **Space** pauses the clock and hides the board.

Puzzles are a sample of the public-domain [Sudoku Exchange Puzzle Bank](https://github.com/grantm/sudoku-exchange-puzzle-bank) (Grant McLean / QQWing + Sukaku Explainer): **2500 Easy, 2000 Medium, 1000 Hard, 250 Diabolical**. `puzzles.json` stores id + givens only; the unique solution is computed at New Game / Continue with `solveSudoku`. `sudoku.json` (cwd, UTF-8 with BOM) stores continue-state, per-difficulty stats, and completed puzzle IDs. Completed IDs are never offered again; New Game draws uniformly from the incomplete pool.

---

### Key Functionality

- **Menu:** Continue Game (only listed when a `continue` blob exists), **New Game: ◀ {Difficulty} ▶** (←→ cycles difficulty and starts a game on Enter; dimmed when the pool is empty), Quit. ←→ always changes difficulty. ↑↓←→, play cursor moves, and starting a new game each rotate the accent wheel one step forward. In pencil mode, play moves and marks rotate **−1**.
- **Pencil marks:** Tab toggles ✒️ pen / ✏️ pencil (HUD). Play always opens in pen mode (New Game and Continue). Empty unlocked cells show up to two *different* candidates on `▀` (FG = top mark color, BG = bottom). First 1–9 fills top, second fills bottom, further presses overwrite top then bottom. Duplicate of the other half is ignored. A digit that is already complete (all nine placed) is ignored for both pen and pencil. A correct lock-in **removes** that digit from pencil marks in the same row, column, and box (`stripPencilPeers`). When the last of a digit is placed, that color is removed from **all** pencil marks (`stripPencilDigit`), not turned white. Any leftover mark in a cell moves to the **background** (bottom) and the top half becomes the next slot. Continue sanitizes with `stripImpossiblePencils` (completed digits plus peers of every locked cell, including givens). 0 / Backspace / Delete in pencil mode clears marks. Placing a pen digit clears that cell's marks. Saved in continue as `pencilTop` / `pencilBot` / `pencilSlot`.
- **Stats panel:** Successes, Failed, Fastest (time + incorrect entries on that run), Remaining `n / total` (total is the bundled count for that difficulty).
- **Continue:** Restores grid, pencil marks, mode, elapsed ms, mistake count, difficulty. Clock resumes on load.
- **New Game while a continue exists:** confirm overlay (Cancel / Abandon); Abandon records a Failure for the *in-progress* difficulty, wipes `continue`, then picks a random incomplete puzzle at the *selected* difficulty.
- **Exit (Esc in play or pause):** overlay **Abandon** or **Quit Sudoku** (Quit selected by default). **Quit** writes continue and exits so the next launch can Continue. **Abandon** clears `continue`, increments Failed, returns to the menu (puzzle ID stays eligible). Esc on the overlay cancels back to play.
- **Success:** `grid == solution`. Record success + fastest (tie-break: fewer mistakes), append ID to `completed[difficulty]`, **immediately** clear `continue` and write `sudoku.json`, then show overlay (time + incorrect entries). `persistPlay` / Quit / Ctrl+C must not rewrite a completed board as Continue. A leftover completed `continue` blob is scrubbed on load (`scrubCompletedContinue`).
- **Incorrect entry:** compare to unique solution. Maroon cell; mistake +1 per wrong place. Grid border flashes **red** ~600ms. Accent wheel jumps **−8**. Givens and locked-correct cells cannot be changed.
- **Correct lock-in:** cell locks (bold). Grid border flashes **green** ~600ms. Accent wheel rotates **forward**. Completing the board holds the green flash then shows Solved.
- **Pause (Space):** freeze clock, fill screen with PAUSED (board hidden). Space resumes; Esc opens the Exit overlay.
- **Ctrl+C / SIGTERM:** persist continue if a puzzle is in progress (same as Quit) and exit. A just-solved board is not persisted as Continue (Success already written). Cleanup resets colors, shows the cursor, `Fini`s tcell, then ANSI `\033[H\033[2J` (Clear-Host) so the console is blank.
- **Windows console:** CP65001, VT processing, Cascadia Mono preference; request 80×24.

---

### Screen flow

```
Menu ──Continue──► Play ──Space──► Paused ──Space──► Play
  │                  │
  │ New Game         ├─Esc──► Exit overlay
  │                  │         ├─Quit──► exit (continue saved)
  │                  │         └─Abandon──► Menu (+Failed, continue wiped)
  └─Quit/Esc         └─complete──► Solved overlay ──Enter──► Menu (Success)
```

Views in `main.go`: `viewMenu`, `viewPlay`, `viewPaused`, `viewConfirmExit`, `viewConfirmNewGame`, `viewSolved`. Exit overlay defaults to **Quit**; New Game confirm defaults to **Cancel**.

---

### Source layout

| File | Role |
|------|------|
| `main.go` | Entry, tcell loop (Larry-style event drain + 1s clock tick), input, persist, win/lose |
| `render.go` | Menu, HUD, 37×19 box-drawn board, Active digit strip, pause/confirm/solved overlays, digit colors |
| `colors.go` | 16-color accent wheel, HUD/title/selection styles, 600ms grid-border flash |
| `board.go` | 81-cell grid, cursor, place/clear, pencil marks, `isWrong` / `isComplete` |
| `save.go` | `sudoku.json` load/save (BOM), stats, completed IDs, continue pencils |
| `puzzles.go` | `go:embed puzzles.json`, incomplete pool, random pick, `ensureSolved` at play time |
| `solver.go` | Bitmask MRV backtracker (import bake + tests) |
| `importpuzzles.go` | `//go:build ignore` — sample Easy 2500 / Medium 2000 / Hard 1000 / Diabolical 250, verify each solves, write `puzzles.json` without solutions |
| `utf8_*.go` / `title_*.go` / `size_*.go` | Windows UTF-8 font/CP, title, 80×24 resize |
| `sudoku.rc` + `sudoku.ico` | Windows executable icon (`windres` → `sudoku.syso`) |
| `build.ps1` | win x86/x64 + linux x86/amd64 → `bin/.../Sudoku[.exe]`, optional `-upx` |
| `cursor.md` | This file |
| `README.md` / `README.html` | End-user docs |

---

### Puzzle bank

**Source:** https://github.com/grantm/sudoku-exchange-puzzle-bank (public domain). Bank line format: 12-char hash, 81-char givens (`0` empty), rating.

**Regenerate `puzzles.json`:**

```powershell
go run importpuzzles.go solver.go
```

Reservoir-samples Easy 2500 / Medium 2000 / Hard 1000 / Diabolical 250 from `easy.txt` / `medium.txt` / `hard.txt` / `diabolical.txt`, verifies each has a unique solution (not stored), writes UTF-8 BOM JSON:

```json
{
  "source": "...",
  "easy": [{ "id": "12charhash", "givens": "81digits" }]
}
```

Difficulty buckets (Sukaku Explainer): Easy &lt; 1.5, Medium &lt; 2.5, Hard &lt; 5.0, Diabolical ≥ 5.0.

**Selection (`incompletePool` + `pickRandom`):** New Game uses IDs not listed in `save.Completed[difficulty]`. Empty pool → New Game disabled, menu label `all {Difficulty} complete`. No wrap/recycle.

---

### Save file (`sudoku.json`)

Written to the process **cwd** (same convention as Larry’s `larry.scores.json`). UTF-8 with BOM. Written after every place/clear, pause, resume, abandon, and solve. A solved board is never stored as `continue`; the completing move records Success and wipes the in-progress blob in the same write.

```json
{
  "version": "1.2",
  "stats": {
    "easy": { "successes": 0, "failures": 0, "fastestMs": null, "fastestMistakes": null }
  },
  "completed": { "easy": ["abc123..."] },
  "continue": {
    "id": "...",
    "difficulty": "easy",
    "givens": "...",
    "solution": "...",
    "grid": "...",
    "elapsedMs": 45000,
    "mistakes": 2,
    "pencil": false,
    "pencilTop": "000...",
    "pencilBot": "000...",
    "pencilSlot": "000..."
  }
}
```

- Missing file → empty stats, no continue.
- Corrupt / invalid continue blob → continue dropped, stats kept if parseable.
- `fastestMs` updated when `elapsed < fastest` OR same time with fewer mistakes.
- `failures` (shown as **Failed**) increments on confirmed Abandon (Esc overlay or New Game while a continue exists). Legacy `abandonments` is copied into `failures` on load if `failures` is 0.

---

### Digit colors (`render.go` `digitColor`)

Nine hues equally spaced around the wheel. **White is reserved** for placed digits when all nine of that number are on the board (`digitCompleteColor`). Pencil marks for a completed digit are stripped (`stripPencilDigit`); they do not turn white. Error feedback is **background maroon**, not a digit hue.

| Digit | Color |
|-------|--------|
| 1 | Red RGB(232,64,64) |
| 2 | Orange RGB(255,140,32) |
| 3 | Gold RGB(240,210,32) |
| 4 | Green RGB(48,196,64) |
| 5 | Teal RGB(32,204,168) |
| 6 | Azure RGB(32,168,232) |
| 7 | Blue RGB(72,104,255) |
| 8 | Violet RGB(168,80,255) |
| 9 | Magenta RGB(232,64,168) |
| complete | White |

HUD mistakes: one `×` (same glyph as README `9×9`) per incorrect entry, no label, clipped so it does not overwrite the clock.

Givens and locked-correct user entries: bold. Cursor: lime-green box border around the selected cell only (existing grid glyphs recolored; cell interior stays black, or maroon if wrong). Empty cursor: space glyph, no fill. During a 600ms flash the whole grid border is lime (correct) or red (incorrect); the cell cursor is hidden for that interval. In pencil mode the idle grid border is light yellow (`styleGridPencil`). Play footer uses `Tab ✏️` / `Tab ✒️` (not the words Pencil/Pen).

---

### Accent color wheel (`colors.go`)

16 saturated hues. Random index at process start (`accentIndex`).

| Role | Wheel offset | Uses |
|------|----------------|------|
| Primary `accent()` | 0 | Menu title, selection bar, stats header, pause title, HUD bar, overlay titles, safe confirm highlight |
| Secondary `accent2()` | +5 | Dialog/solved panel fill, cell cursor border |
| Tertiary `accent3()` | −5 | Destructive confirm highlight (Abandon) |

Black text on primary/secondary/tertiary fills. Grid flash stays semantic lime/red (not chrome). Digit hues stay independent of the accent wheel; completed digits go white.

| Event | Wheel |
|-------|--------|
| Menu ↑↓←→ (and WASD aliases) | +1 |
| Play cursor move (pen) | +1 |
| Play cursor move (pencil) | −1 |
| Pencil mark | −1 |
| `startNewGame` | +1 |
| Correct digit locked | +1 |
| Incorrect digit placed | −8 |

Not persisted. Continue Game / Quit do not rotate except via the arrow keys used to reach them.

---

### Board geometry

Playfield 37 columns × 19 rows, centered under a 1-row HUD. One blank row under the board, then a centered **Active:** strip of remaining (uncompleted) digits in their hues; completed numbers are omitted (e.g. `Active: 3 6 8`). Controls hint sits on the last row so 80×24 still fits.

- Double-line `╔═╗║╚╝╦╩╬╠╣` for 3×3 boxes and outer frame.
- Single-line `─│┼` inside a box; mixed `╤╧╟╢╫╪` at box/cell joins.
- Each cell is three columns: space, digit, space.
- Cursor movement is toroidal: left from column 0 wraps to column 8, and the same for the other edges.

Preferred console **80×24** (`size.go`). HUD: `SUDOKU {Difficulty}` + ✒️/✏️ mode; `×` tally for mistakes; elapsed `M:SS` or `H:MM:SS` right. Under the board: blank row, then centered `Active:` remaining digits; last row is the controls hint.

---

### Clock

`elapsed` accumulates paused-out play time. `clockRunning` + `clockAnchor` for the active segment. `currentElapsed()` = elapsed + since(anchor) if running. Pause/give-up-confirm call `stopClock()`. 1s ticker redraws only in `viewPlay` while the clock runs.

---

### Encoding

Go sources, `puzzles.json`, `sudoku.json`, markdown, and HTML are **UTF-8 with BOM** so Windows editors keep box-drawing literals. `loadSave` / `loadPuzzleBank` strip a leading BOM. Windows still calls `enableUTF8Console()` (CP65001 + VT + Cascadia Mono).

Do not save `.go` files as UTF-16. `go.mod` / `go.sum` stay UTF-8 without BOM.

---

### Build

From `go/sudokutui`:

```powershell
pwsh -NoLogo -NoProfile -File ./build.ps1
```

Outputs `bin/win/x86/Sudoku.exe`, `bin/win/x64/Sudoku.exe`, `bin/linux/x86/Sudoku`, `bin/linux/amd64/Sudoku`. Icon embed requires `windres` + `sudoku.rc` + `sudoku.ico`. Pass `-upx` to compress. Do not run `build.ps1` from an agent session unless verifying a compile; the user owns Larry-style rebuilds.

Quick check: `go test .`  ·  `go build -o Sudoku.exe .`

`go/build.ps1` auto-discovers this folder via `build.ps1`.

---

### Tests

- `solver_test.go` — Wikipedia puzzle, reject short input, empty grid solves.
- `puzzles_test.go` — bundled counts (2500/2000/1000/250) unique IDs, no baked solutions, sample solvability, incomplete pool omits completed IDs, mistake/correct/complete board behavior, digit-complete (all nine of a number), Active strip omits completed digits, toroidal cursor wrap, pencil mark cycle/clear/save, strip completed pencil color to background, strip peer (row/col/box) marks on correct place, pen and pencil ignore completed digits.
- `save_test.go` — finishSuccess wipes continue and records completion immediately; persistPlay skips a completed board; load scrubs a leftover solved continue without double-counting.
- `colors_test.go` — 16-step wheel wrap; −8 incorrect jump; pencil mode −1 step; digit hues are not white.

---

### Agent notes

- Module name `sudokutui` (folder); binary name **Sudoku** (capital S).
- Do not recycle completed IDs. Abandoned puzzles may be drawn again.
- Space pauses/resumes in play (clears with 0 / Backspace / Delete). Cursor wraps toroidally. Tab toggles pencil mode.
- Esc in play/pause opens Exit: **Quit** (default) persists continue and leaves; **Abandon** wipes continue and increments Failed. Continue is omitted from the menu when there is no in-progress game. Completing a puzzle writes Success and clears continue before the solved overlay, so the next launch cannot Continue that board. Any exit path (Quit, Esc on menu, Ctrl+C) Clear-Hosts after `Fini`.
- Confirm dialogs default to No (Larry give-up / destructive-action convention).
- Keep `puzzles.json` in git; do not download the bank at runtime.
- If adding difficulties, update `difficultyOrder`, JSON keys, import list, and stats maps together.
