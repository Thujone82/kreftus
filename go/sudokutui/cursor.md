# Cursor Project File

## Project: sudokutui (Sudoku)

**Author:** Kreft&Cursor  
**Date:** 2026-08-19  
**Version:** 1.5

---

### Description

`sudokutui` is a cross-platform terminal Sudoku game written in Go with [`tcell`](https://github.com/gdamore/tcell). The Windows/Linux executable is named **Sudoku**. UTF-8 box-drawing frames a 9×9 board; each digit 1–9 has a distinct color. Wrong entries keep the digit and paint the cell maroon. A play clock sits on the HUD; **Space** pauses the clock and hides the board.

Puzzles are a sample of the public-domain [Sudoku Exchange Puzzle Bank](https://github.com/grantm/sudoku-exchange-puzzle-bank) (Grant McLean / QQWing + Sukaku Explainer): **2500 Easy, 2000 Medium, 1000 Hard, 250 Diabolical**. `puzzles.json` stores id + givens only; the unique solution is computed at New Game / Continue with `solveSudoku`. `sudoku.json` (cwd, UTF-8 with BOM) stores continue-state (including the move log), per-difficulty stats (including the Best/Fastest playback), and completed puzzles (`id` + `mistakes`). Completed IDs are never offered again; New Game draws uniformly from the incomplete pool.

---

### Key Functionality

- **Menu:** Under the SUDOKU banner, **── Medium ──** (or Easy / Hard / Diabolical) then **Continue** (only when that difficulty has a continue blob), **New Game**, Quit. A single-line `┌─┐` **Stats** box sits below the items; Perfect, Successes, Error Rate, Failed, Best (or Fastest after a perfect), and Average appear after the first success at that difficulty. When that difficulty has a stored Best/Fastest recording, ↓ past Quit lands on the Stats header (not a third `▶` row). The title becomes **Replay** (accent highlight) and the box shrinks to wrap a 9×9 playback of that run, same 5s + 3s hue-cycle loop as the solved overlay. ↑ back to Quit restores the Stats listing; ↓ from Replay wraps to Continue (or New Game) like the rest of the menu. ←→ cycles difficulty (footer `←→ difficulty`) and clamps the highlight if Continue or Replay disappears. Dimmed New Game when the pool is empty (`New Game  (complete)`). ↑↓←→, play cursor moves, and starting a new game each rotate the accent wheel one step forward. In pencil mode, play moves and marks rotate **−1**.
- **Pencil marks:** Tab toggles ✒️ pen / ✏️ pencil (HUD). **Hold Shift** for momentary pencil (HUD, yellow grid, and 1–9 marks) without changing the saved Tab mode; US Shift+1..9 (`!@#$%^&*(`) also mark. NumLock+Shift on the keypad arrives as arrows/Home/End/Clear with a synthetic Shift-up — those still map to 1–9 if Shift was down within ~150ms. Dedicated arrow keys still move (extended keys vs keypad); WASD also moves. New Game always opens in pen mode; Continue restores saved `pencil`. Empty unlocked cells show up to two *different* candidates on `▀` (FG = top mark color, BG = bottom). First 1–9 fills top, second fills bottom, further presses overwrite top then bottom. Pressing a digit already on the cell **toggles it off** and that half is the next fill slot. A digit that is already complete (all nine placed) is ignored for both pen and pencil. A correct lock-in **removes** that digit from pencil marks in the same row, column, and box (`stripPencilPeers`). When the last of a digit is placed, that color is removed from **all** pencil marks (`stripPencilDigit`), not turned white. Any leftover mark in a cell moves to the **background** (bottom) and the top half becomes the next slot. Continue sanitizes with `stripImpossiblePencils` (completed digits plus peers of every locked cell, including givens). 0 / Backspace / Delete in pencil mode clears marks. Placing a pen digit clears that cell's marks. Saved in continue as `pencil` / `pencilTop` / `pencilBot` / `pencilSlot`.
- **Stats panel:** Below the menu, a single-line accent box with **Stats** centered on the top border. Remaining is always inside; full Perfect / Successes / Error Rate / Failed / Best or Fastest / Average only after at least one success at that difficulty. **Best** is the fewest-mistake win (then fastest among those); after a zero-error win the same row is **Fastest** (quickest perfect). Best shows `  (n incorrect)`; Fastest does not. **Average** is the mean of all solves until a perfect exists, then the mean of perfects only. Labels are right-aligned so `:` shares a column, then two spaces before the value (`Perfect:  0`). The block is left-aligned as a unit so columns stay lined up. The selected item is ` ▶ Label ◀ ` (spaces included) so the accent bar is wider than the label. Selecting the Stats header (when a recording exists) replaces this listing with the tight Replay box until the cursor returns to Continue / New Game / Quit.
- **Continue:** One in-progress game per difficulty (`continue[easy|medium|hard|diabolical]`). Restores grid, pencil marks, pen/pencil mode (`pencil`), elapsed ms, mistake count, and the compact move log (`events`). Clock resumes on load. A v1.2 single `continue` object is migrated into that map on load. A pre-1.5 continue (progressed `grid` / pencils, no `events`) is onboarded on Continue: the current board is stored as `startGrid` / start pencils and used as playback frame 0; only later moves are logged. Native 1.5 games omit `startGrid` and apply `events` to givens.
- **New Game while that difficulty has a continue:** confirm overlay (Cancel / Abandon). Abandon records a Failure for **that** difficulty, wipes only that continue, then picks a random incomplete puzzle. Continues at other difficulties are left alone. Esc in play/pause does **not** abandon.
- **Esc (play or pause):** persist that difficulty's continue and return to the menu (hint **Esc Menu**). No Abandon/Quit overlay. Quit the app from the menu (Esc / Quit) or Ctrl+C.
- **Success:** `grid == solution`. Record success + Best/Fastest (fewest mistakes, then fastest time), append `{id, mistakes, elapsedMs}` to `completed[difficulty]`, **immediately** clear **that difficulty's** continue and write `sudoku.json`, then show a joined double-line frame (rainbow SUDOKU banner, `╔═╦╗`): stats on the left, a 9-wide playback pane sharing the middle `║` with no extra left/right pad. Left: title bar is `accent2()` fill with **{Difficulty} Solved!** in `accent3()` (same as **Press Enter**); frame is `accent()`; **Press Enter** is `accent3()`. Stats are colon-aligned (`Time:  22:23` / `Errors:  2`); zero mistakes replaces Errors with **Perfect Finish!** in `accent2()`. A new Best/Fastest also shows **New Best Completion!** or **New Fastest Completion!** (same Best-until-perfect / Fastest-after-perfect label as the menu) in `accent2()` under that. Right: borderless 9×9 playback of the session log (same digit hues / maroon wrongs / `▀` marks, no cursor). Frames are timed so the whole run lasts 5s (`5000ms / eventCount`); then the solution is shown and digit hues rotate at 5 fps for 3s before looping from the origin (givens, or an onboarded `startGrid`). A new Best/Fastest also stores `{id, givens, events}` (plus `startGrid` when onboarded) as `stats[d].fastestReplay`; ↓ past Quit on the menu plays that recording in the Stats box. `persistPlay` / Ctrl+C must not rewrite a completed board as Continue. A leftover completed continue blob is scrubbed on load (`scrubCompletedContinue`). On load, `fastestMs` / `fastestMistakes` / `fastestReplay` are reconciled from `completed` so a slower perfect replaces a faster imperfect (stale replay dropped if the id no longer wins).
- **Incorrect entry:** compare to unique solution. Maroon cell; mistake +1 per wrong place. Grid border flashes **red** ~600ms. Accent wheel jumps **−8**. Givens and locked-correct cells cannot be changed.
- **Correct lock-in:** cell locks (bold). Grid border flashes **green** ~600ms. Accent wheel rotates **forward**. Completing the board holds the green flash then shows Solved.
- **Pause (Space):** freeze clock, hide the board. Same rainbow SUDOKU banner as the menu (S = accent+5, then +1 per letter). Under the banner: difficulty in caps (`MEDIUM`), a 20-cell `█`/`░` bar of filled vs originally empty cells plus percent, and `Pencil Marks: n  Errors: n` only when either count is > 0. PAUSED + elapsed time stay centered. Space resumes; Esc returns to the menu (continue saved).
- **Ctrl+C / SIGTERM:** persist continue if a puzzle is in progress and exit. A just-solved board is not persisted as Continue (Success already written). Cleanup resets colors, shows the cursor, `Fini`s tcell, then ANSI `\033[H\033[2J` (Clear-Host) so the console is blank.
- **Windows console:** CP65001, VT processing, Cascadia Mono preference; request 80×24.

---

### Screen flow

```
Menu ──Continue──► Play ──Space──► Paused ──Space──► Play
  │                  │
  │ New Game         └─Esc──► Menu (continue saved per difficulty)
  │   └─if that difficulty already has a continue──► Confirm (Cancel / Abandon)
  │         Abandon──► Failed + new puzzle at that difficulty
  └─Quit/Esc         Play ──complete──► Solved overlay ──Enter──► Menu (Success)
```

Views in `main.go`: `viewMenu`, `viewPlay`, `viewPaused`, `viewConfirmNewGame`, `viewSolved`. Menu Replay is a `menuIndex` on `viewMenu`, not a separate view. New Game confirm defaults to **Cancel**.

---

### Source layout

| File | Role |
|------|------|
| `main.go` | Entry, tcell loop (Larry-style event drain + 1s HUD clock tick + 50ms Shift / solved / menu-Replay tick + debounced save flush), input, persist, win/lose |
| `render.go` | Menu, HUD, 37×19 box-drawn board, Active digit strip, pause/confirm/solved overlays, digit colors, Stats / Replay boxes |
| `replay.go` | Compact session log (place/clear/mark/clear-marks), 5s playback + 3s hue-cycle clock, apply-prefix reconstruct, Best/Fastest menu Replay |
| `colors.go` | 16-color accent wheel, HUD/title/selection styles, 600ms grid-border flash |
| `board.go` | 81-cell grid, cursor, place/clear, pencil marks, `isWrong` / `isComplete` |
| `save.go` | `sudoku.json` load/save (BOM), stats, completed IDs, continue pencils + events, Best/Fastest replay |
| `puzzles.go` | `go:embed puzzles.json.gz`, remaining count, `pickIncomplete`, `ensureSolved` at play time |
| `solver.go` | Bitmask MRV backtracker (import bake + tests) |
| `importpuzzles.go` | `//go:build ignore` — sample Easy 2500 / Medium 2000 / Hard 1000 / Diabolical 250, verify each solves, write `puzzles.json` and `puzzles.json.gz` without solutions |
| `utf8_*.go` / `title_*.go` / `size_*.go` / `shift_*.go` | Windows UTF-8 font/CP, title, 80×24 resize; Shift-held pencil poll (`GetAsyncKeyState`); low-level hook so dedicated arrows stay movement while keypad 2/4/6/8 mark |
| `sudoku.rc` + `sudoku.ico` | Windows executable icon (`windres` → `sudoku.syso`) |
| `build.ps1` | win x86/x64 + linux x86/amd64 → `bin/.../Sudoku[.exe]`, optional `-upx` |
| `cursor.md` | This file |
| `README.md` / `README.html` | End-user docs |

---

### Puzzle bank

**Source:** https://github.com/grantm/sudoku-exchange-puzzle-bank (public domain). Bank line format: 12-char hash, 81-char givens (`0` empty), rating.

**Regenerate `puzzles.json` / `puzzles.json.gz`:**

```powershell
go run importpuzzles.go solver.go
```

Reservoir-samples Easy 2500 / Medium 2000 / Hard 1000 / Diabolical 250 from `easy.txt` / `medium.txt` / `hard.txt` / `diabolical.txt`, verifies each has a unique solution (not stored), writes UTF-8 BOM JSON plus a gzip copy for `go:embed`:

```json
{
  "source": "...",
  "easy": [{ "id": "12charhash", "givens": "81digits" }]
}
```

Difficulty buckets (Sukaku Explainer): Easy &lt; 1.5, Medium &lt; 2.5, Hard &lt; 5.0, Diabolical ≥ 5.0.

**Selection (`remainingCount` + `pickIncomplete`):** New Game uses IDs not listed in `save.Completed[difficulty]`. The menu Remaining line counts without copying the leftover pool. Empty pool → New Game disabled, menu label `New Game  (complete)`. No wrap/recycle.

---

### Save file (`sudoku.json`)

Written to the process **cwd** (same convention as Larry’s `larry.scores.json`). UTF-8 with BOM, compact JSON. In-memory continue updates on every place/clear/mark; disk writes are debounced (~300ms) and flushed immediately on pause, quit, abandon, and solve. Writes go to a temp file then rename (with a `.bak` swap on Windows) so a crash cannot leave a half-written save. A solved board is never stored as `continue`; the completing move records Success and wipes the in-progress blob in the same write. User actions (set number, clear number, place/toggle mark, clear marks) append to a compact `events` string on continue; auto pencil-strips are not logged and replay through `place` / `markPencil`.

```json
{
  "version": "1.5",
  "stats": {
    "easy": { "successes": 0, "failures": 0, "perfect": 0, "mistakeSum": 0, "ratedSuccesses": 0, "fastestMs": null, "fastestMistakes": null, "fastestReplay": { "id": "...", "givens": "...", "events": "2P5,14M3", "startGrid": "..." } }
  },
  "completed": { "easy": [{ "id": "abc123...", "mistakes": 3, "elapsedMs": 45000 }] },
  "continue": {
    "easy": {
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
      "pencilSlot": "000...",
      "events": "2P5,14M3",
      "startGrid": "..."
    }
  }
}
```

- Missing file → empty stats, no continue.
- Corrupt / invalid continue blob → continue dropped, stats kept if parseable.
- `fastestMs` / `fastestMistakes` store **Best** until a perfect exists, then the fastest perfect. Compare fewest mistakes first, then elapsed. That write also replaces `fastestReplay` (`id` + `givens` + `events`, and `startGrid` when a pre-1.5 continue was onboarded). Load recomputes the pair from `completed` and drops a replay whose id is no longer the winner. Solution is not stored — `solveSudoku(givens)` at playback. Pre-1.5 continues with no `events` snapshot the live board as `startGrid` on Continue so playback jumps there instead of reconstructing from givens.
- `perfect` counts successes with 0 incorrect entries. `mistakeSum` / `ratedSuccesses` is **Error Rate** (`%.2f`).
- `completed[difficulty]` is `{id, mistakes, elapsedMs}` per solved puzzle. IDs stay out of the New Game pool. Average is the mean of `elapsedMs` until a perfect exists, then the mean of perfects only.
- `failures` (shown as **Failed**) increments on confirmed Abandon when starting New Game while that difficulty already has a continue. Legacy `abandonments` is copied into `failures` on load if `failures` is 0.
- A v1.2 `continue` object (not keyed by difficulty) loads into `continue[difficulty]`.

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

Givens and locked-correct user entries: bold. Cursor: `accent2()` box border around the selected cell only (existing grid glyphs recolored; cell interior stays black, or maroon if wrong). In pencil mode gold and yellow (wheel 3–4) are skipped so the cursor does not sit on the light-yellow grid (`styleGridPencil`). Accent-colored **▶ ◀** sit on the selected row (outside the frame) and **▼ ▲** on the selected column (padding rows above and below the board). Empty cursor: space glyph, no fill. During a 600ms flash the whole grid border is lime (correct) or red (incorrect); the cell cursor is hidden for that interval; row/column marks stay. Play footer uses `Tab/Shift ✏️` / `Tab ✒️` (not the words Pencil/Pen).

---

### Accent color wheel (`colors.go`)

16 saturated hues. Random index at process start (`accentIndex`).

| Role | Wheel offset | Uses |
|------|----------------|------|
| Primary `accent()` | 0 | Selection bar, stats header, pause title, HUD bar, overlay titles, solved box frame, safe confirm highlight, row/column selection arrows |
| Secondary `accent2()` | +5 | Dialog panel fill, solved title bar / Perfect Finish / New Best or Fastest Completion, cell cursor border (pencil mode skips gold/yellow, wheel 3–4); menu/pause **S** in SUDOKU (U–U then +1 per letter) |
| Tertiary `accent3()` | −5 | Destructive confirm highlight (Abandon); solved **{Difficulty} Solved!** title text and **Press Enter** |

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

Playfield 37 columns × 19 rows, centered under a 1-row HUD (one column of margin on each side when the console is wide enough). Accent **▶** / **◀** track the selected row; **▼** / **▲** track the selected column on the padding rows. One blank/arrow row under the board, then a centered **Active:** strip of remaining (uncompleted) digits in their hues; completed numbers are omitted (e.g. `Active: 3 6 8`). Controls hint sits on the last row so 80×24 still fits.

- Double-line `╔═╗║╚╝╦╩╬╠╣` for 3×3 boxes and outer frame.
- Single-line `─│┼` inside a box; mixed `╤╧╟╢╫╪` at box/cell joins.
- Each cell is three columns: space, digit, space.
- Cursor movement is toroidal: left from column 0 wraps to column 8, and the same for the other edges.

Preferred console **80×24** (`size.go`). HUD: `SUDOKU {Difficulty}` + ✒️/✏️ mode; `×` tally for mistakes; elapsed `M:SS` or `H:MM:SS` right. Under the board: blank row, then centered `Active:` remaining digits; last row is the controls hint.

---

### Clock

`elapsed` accumulates paused-out play time. `clockRunning` + `clockAnchor` for the active segment. `currentElapsed()` = elapsed + since(anchor) if running. Pause/give-up-confirm call `stopClock()`. 1s ticker redraws only in `viewPlay` while the clock runs. The 50ms Shift ticker also redraws `viewSolved` and menu Replay so playback frames can be 50ms (100 events → 5s) and the 5 fps hue cycle can tick.

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
- `puzzles_test.go` — bundled counts (2500/2000/1000/250) unique IDs, no baked solutions, sample solvability, incomplete pool / remainingCount / pickIncomplete omit completed IDs, mistake/correct/complete board behavior, digit-complete (all nine of a number), Active strip omits completed digits, toroidal cursor wrap, selection-mark positions, pencil mark cycle/clear/toggle/save, strip completed pencil color to background, strip peer (row/col/box) marks on correct place, pen and pencil ignore completed digits.
- `save_test.go` — finishSuccess wipes that difficulty's continue (others stay); persistPlay skips a completed board; load scrubs a leftover solved continue without double-counting; Perfect, Error Rate, and Average Completion from stored completions (perfects only once a zero-error win exists); Best prefers fewer mistakes then time and a perfect replaces any imperfect; load reconciles Best from completed; Continue restores saved pencil mode; pre-1.5 continue without events is onboarded with `startGrid`; persistPlay debounce; compact atomic write; legacy single continue object maps per difficulty; Esc to menu keeps continue without Failed; New Game prompts only when that difficulty already has a save; finishSuccess flags `solvedNewRecord` only when the run is the new Best/Fastest.
- `replay_test.go` — encode/decode, apply matches live board, 5s playback + 3s hue-cycle frame index, playback prefix, start-snapshot origin; menu Replay is in `visibleMenu` only with an 81-char `fastestReplay`, ↓ from Quit / ↑ back / ↓ from Replay wraps to the first item, clamp drops Replay when the other difficulty has no recording, `beginMenuReplay` loads that blob, Enter on Replay is a no-op
- `render_test.go` — solved overlay title is **{Difficulty} Solved!**; uses **Perfect Finish!** at 0 mistakes and colon-aligns Time / Errors otherwise; **New Best Completion!** / **New Fastest Completion!** only when the run is the new record; Best vs Fastest label/value; Stats box top border centers the title and Average is the short stats label; Shift+digit symbols map to 1–9; NumLock+Shift keypad nav keys map to 1–9 and stay pencil-active through Windows' fake Shift-up; pause progress bar and Pencil Marks / Errors line; replay celebrate shows the solution and digit hue-shift wraps.
- `colors_test.go` — 16-step wheel wrap; −8 incorrect jump; pencil mode −1 step; held Shift is pencil-active; digit hues are not white; pencil cursor skips gold/yellow; SUDOKU banner S is accent+5.

---

### Agent notes

- Module name `sudokutui` (folder); binary name **Sudoku** (capital S).
- Do not recycle completed IDs. Abandoned puzzles may be drawn again.
- Space pauses/resumes in play (clears with 0 / Backspace / Delete). Cursor wraps toroidally. Tab toggles pencil mode; holding Shift is momentary pencil (not saved). Menu selection also wraps, including Replay when a Best/Fastest recording exists (↓ from Replay returns to Continue or New Game).
- Esc in play/pause returns to the menu and keeps that difficulty's continue. Abandon (Failed) only when starting New Game over an existing continue at the **same** difficulty. Completing a puzzle writes Success and clears that continue before the solved overlay. Any exit path (Quit, Esc on menu, Ctrl+C) Clear-Hosts after `Fini`.
- Confirm dialogs default to No (Larry give-up / destructive-action convention).
- Keep `puzzles.json` and `puzzles.json.gz` in git (gz is the embed); do not download the bank at runtime.
- If adding difficulties, update `difficultyOrder`, JSON keys, import list, and stats maps together.
