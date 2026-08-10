# Larry — Terminal Frogger-like Game

**Version:** 1.4 · **Author:** Kreft & GPT-5 · **Date:** 2026-08-10

## Description

`larry` is a fast, cross-platform terminal game written in Go with [`tcell`](https://github.com/gdamore/tcell). Guide **Larry** (`⬢`) from the bottom safe shoulder to the top while dodging traffic. Reach the goal to clear the level, gain a life, and roll a new color theme.

Difficulty rises with lane density and speed. Later levels add hazards and pickups on safe gaps. The UI uses UTF-8 box-drawing and gameplay glyphs; on Windows, Larry switches the console to UTF-8 (CP65001) and prefers **Cascadia Mono** when installed.

## Features

- **Start menu** — hop Larry between Start, High Scores, and Quit (aligned labels)
- **High Scores** — top-10 list from the menu; persistent `larry.scores.json` (MMDDYY dates)
- **Traffic backdrop** — animated lanes behind the title screen
- **Safe playfield** — top/bottom shoulders plus random safe gaps between road packs
- **Goal line** — yellow `▚▞` checker on the top shoulder
- **Themes** — palette changes each level (levels continue past 9)
- **Debris (L6+)** — impassable `☙` on mid safe gaps (1% at L6, +1%/level through L10, then +0.5%/level, cap 10%)
- **Heart (L8+)** — flashing mid-lane `♥`; hop for +1 life (HUD: `+1 Life`)
- **Diamond (L10+)** — flashing top mid-lane `♦`; hop for +1,000 points (HUD: `+1,000 Points`)
- **Scoring** — per-row climb bonus, clear bonus, session Top, and Best from history
- **Vehicles** — distinct classes per lane (see below)
- **Console** — requests **80×42** on launch; resize-aware rendering

### Vehicles

| Class | Length | Speeds | Glyphs |
| ----- | ------ | ------ | ------ |
| Motorcycle / compact | 2 | 3–5 | `═` or `▩` with `▶` / `◀` |
| Car | 3 | 2–4 | `█` or `◙` with `◀…▶` |
| Truck / semi | 5 | 1–3 | `█` or `▓` boxes with `▶` / `◀` |

## Controls

| Context | Input | Action |
| ------- | ----- | ------ |
| Start menu | ↑↓ or W/S | Hop Larry |
| Start menu | Enter / Space | Confirm selection |
| Start menu | Esc | Exit the game |
| High Scores | Esc or Enter | Return to start menu |
| In game | Arrows or WASD | Move Larry |
| In game | Space | Pause / resume |
| In game | Esc | Confirm return to menu (Larry on **No**; ←/→ hop, Enter selects, Esc cancels) |
| Anywhere | Ctrl+C | Exit immediately |

## Scoring

| Event | Points / reward |
| ----- | --------------- |
| New upward row within a level | +10 per row |
| Reach top safe shoulder | +100 × current level |
| Clear a level | +1 life |
| Hop mid-lane `♥` (L8+) | +1 life · inverted HUD `+1 Life` (~1s) |
| Hop top mid-lane `♦` (L10+) | +1,000 · inverted HUD `+1,000 Points` (~1s) |

Session **Top** and historical **Best** appear on the right side of the HUD. Score decays by 1 per second after your first move on a level.

## Console notes

- Prefer **Cascadia Mono** (or another modern Unicode font) so `▶` `◀` `⬢` `☙` `♥` `♦` render cleanly. Larry selects Cascadia Mono automatically when present.
- On launch Larry requests an **80×42** console so the HUD fits. Windows Terminal and some hosts may ignore resize requests — set the profile size manually if needed.

## Testing

`-testlvl INT` skips the start menu and opens immediately at that level. Prior levels are treated as cleared:

- Theme, density, and speed match the requested level
- Score simulates climb + clear bonuses for levels `1..(INT-1)`, minus **10** points per prior level (time spent)
- Lives include the usual +1 life per cleared level

Name-entry UI still runs in test mode, but **`larry.scores.json` is never written**, so the real scoresheet stays untouched.

```powershell
.\bin\larry.exe -testlvl 10
go run . -testlvl 8
```

## Build

From the `go/larry` folder:

**All platforms (recommended)**

```powershell
pwsh -NoLogo -NoProfile -File ./build.ps1
```

**Quick local build**

```powershell
go build -o bin/larry.exe
```

**Run without a binary**

```powershell
go run .
```

### Binary output

| Platform | Path |
| -------- | ---- |
| Windows x86 | `bin/win/x86/larry.exe` |
| Windows x64 | `bin/win/x64/larry.exe` |
| Linux x86 | `bin/linux/x86/larry` |
| Linux amd64 | `bin/linux/amd64/larry` |

If `larry.ico` and `larry.rc` are present and `windres` is available, Windows builds embed the icon automatically.
