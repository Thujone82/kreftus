# larry - Terminal Frogger-like Game

## Version
1.1

## Author
Kreft & GPT-5

## Date
2025-08-10

## Description
`larry` is a fast, cross-platform terminal game written in Go using `tcell`. You guide Larry (a green `@`) from the bottom safe shoulder to the top safe shoulder while dodging traffic. Each time you reach the top, you advance to the next level, gain a life, and the color theme changes. Difficulty scales gradually by increasing lane density and speed.

## Features
- Real-time input (Arrows and WASD)
- Safe shoulders (top and bottom) and safe gaps between roads
- Level progression with changing themes
- Lives, per-line progression score, and session Top score
- Distinct vehicle classes per lane:
  - Compact car: length 2, speeds 3–5, glyphs: `=>` (right) / `<=` (left)
  - Regular car: length 3, speeds 2–4, glyph: `<#>`
  - Semi trailer: length 5, speeds 1–3, glyphs: `####>` (right) / `<####` (left)
- Resize-aware rendering

## Controls
- Move: Arrow keys or WASD
- Pause: Space
- Quit: Esc

## Scoring
- +10 for each new upward row reached within a level
- +100 × level on reaching the top safe shoulder
- An extra life is awarded each time you clear a level
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


