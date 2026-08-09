# larry - Terminal Frogger-like Game

## Version
1.3

## Author
Kreft & GPT-5

## Date
2026-08-09

## Description
`larry` is a fast, cross-platform terminal game written in Go using `tcell`. You guide Larry (a bright highlight ⬢) from the bottom safe shoulder to the top safe shoulder while dodging traffic. Each time you reach the top, you advance to the next level, gain a life, and the color theme changes. Difficulty scales gradually by increasing lane density and speed; from level 6 onward, impassable debris appears on mid safe gaps; from level 8, a flashing ♥ life pickup appears on a mid safe lane. The UI uses UTF-8 box-drawing and gameplay glyphs (Windows consoles switch to CP65001 and prefer Cascadia Mono on launch).

## Features
- Start menu with Larry as the hop cursor (aligned Start / High Scores / Quit)
- High Scores view from the start menu (top-10 list)
- Animated traffic backdrop behind the start menu
- Real-time input (Arrows and WASD)
- Safe shoulders (top and bottom) and safe gaps between roads
- Goal shoulder marked with a yellow `▚▞` checker pattern
- Level progression with changing themes (levels continue past 9)
- From level 6: impassable `☙` debris on mid safe gaps (1% at L6, +1%/level through L10, then +0.5%/level, cap 10%)
- From level 8: one flashing `♥` on the center of a mid safe lane — hop on it for an extra life (HUD flashes `+1 Life`)
- Lives, per-line progression score, and session Top score
- Distinct vehicle classes per lane:
  - Motorcycle/compact: length 2, speeds 3–5, bodies `═` or `▩` with `▶`/`◀`
  - Car: length 3, speeds 2–4, body `█` or `◙` with `◀…▶`
  - Truck/semi: length 5, speeds 1–3, boxes `█` or `▓` with `▶`/`◀`
- UTF-8 menus, overlays, and HUD separators
- Auto console size request **80×42** on launch; resize-aware rendering
- Top-10 scoreboard with name entry and saved history (MMDDYY date)

## Notes
For best vehicle/Larry glyph rendering (`▶` `◀` `⬢` `☙` `♥`) on Windows, use a modern console font such as **Cascadia Mono** (Windows Terminal default). Larry selects it automatically when installed.

On launch, Larry requests an **80×42** console size so the full HUD header fits. Windows Terminal and some hosts may ignore resize requests; set the profile size manually if needed.

## Controls
- Start menu: ↑↓ or W/S to hop Larry, Enter/Space to confirm
  - Start — begin a new game
  - High Scores — view the top-10 list (Esc or Enter returns to the menu)
  - Quit — exit the game
- Move: Arrow keys or WASD
- Pause: Space
- Esc during play: confirm return to main menu (Larry on No; ←/→ hop, Enter selects, Esc cancels)
- Esc on main menu: exit the game
- Esc on high scores: return to the start menu
- Ctrl+C: exit immediately

## Scoring
- +10 for each new upward row reached within a level
- +100 × level on reaching the top safe shoulder
- An extra life is awarded each time you clear a level
- From level 8: hopping the mid-lane `♥` grants +1 life (inverted HUD shows `+1 Life` for ~1s)
- Session Top score is shown on the right of the status bar

## Build
From the `go/larry` folder:

Windows and Linux builds (recommended):
```powershell
pwsh -NoLogo -NoProfile -File ./build.ps1
```

Quick local build:
```powershell
go build -o bin/larry.exe
```

Run directly (no binary):
```powershell
go run .
```

## Binary Output
- Windows: `bin/win/x86/larry.exe`, `bin/win/x64/larry.exe`
- Linux: `bin/linux/x86/larry`, `bin/linux/amd64/larry`

If `larry.ico` and `larry.rc` are present and `windres` is available, Windows builds will embed the icon automatically.
