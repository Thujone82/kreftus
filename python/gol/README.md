# GoLPy — Conway's Game of Life

**Version 1.1**

Desktop Python port of the [web Game of Life](https://kreft.us/gol/) with pygame.

**Author:** Kreft&Cursor

## Requirements

- Python 3.11+
- Dependencies: `pip install -r requirements.txt` (`pygame-ce` for GUI, `textual` for `-tui`)

## Quick start

```bash
cd python/gol
pip install -r requirements.txt
python gol.py
```

## Command-line options

| Flag | Description |
|------|-------------|
| `--mode wrapped\|infinite` | Grid mode (default: `wrapped`) |
| `--pattern` *NAME* | Load a built-in pattern on startup (e.g. `glider`, `gosper`) |
| `--speed` *N* | Simulation speed 10–200 (default: 100) |
| `--density low\|high` | TUI cell density: low (1 char/cell) or high (2 logical rows per char via `▄`) |
| `-tui` | Terminal UI (Textual); no pygame window |
| `-debug` | Log population and scope every 100 generations to stderr |

Examples:

```bash
python gol.py --mode infinite --pattern gosper
python gol.py --pattern pulsar --speed 150
python gol.py -tui
python gol.py -tui --mode infinite --pattern glider
python gol.py -tui --density high --mode infinite --pattern gosper
```

## Controls

| Action | Input |
|--------|-------|
| Play / Pause | **Space** or toolbar button |
| Step | Toolbar **Step** or **N** |
| Reset | Toolbar **Reset** or **R** |
| Toggle cell | Left-click on the grid |
| Pan | Drag on the grid (infinite always; wrapped when paused) |
| Zoom | Mouse wheel, toolbar slider, or **+** / **-** (wrapped: when paused only) |
| Pattern | Toolbar **Pattern..** → scrollable list |
| Mode | Toolbar **Wrapped** / **Infinite** toggle |
| Save / Restore | **M+** / **MR** (in-memory snapshot) |
| Speed | Toolbar speed slider |
| HUD (Pop / Step) | **H** toggle corner overlay (on by default) |

### Terminal UI (`-tui`)

Launches a Textual setup screen (mode, pattern, speed), then uses the full terminal as the cell grid.

| Action | Input |
|--------|-------|
| Setup: mode | **W** wrapped / **I** infinite |
| Setup: pattern | **↑** / **↓** |
| Setup: speed | **←** / **→** |
| Setup: density | **D** toggle Low (1 char/cell) / High (2 rows/char via `▄`) |
| Setup: start | **Enter** or **S** |
| Setup: controls help | **C** |
| Play / Pause | **Space** |
| Step | **N** |
| Reset | **R** |
| Quit simulation | **Q** (returns to setup) |
| Pan (infinite) | Arrows or **WASD** (paused, or while running if follow off) |
| Toggle follow (infinite) | **F** |
| Speed (simulation) | **+** / **-** |
| HUD (Pop / Step) | **H** toggle corner overlay (on by default) |
| Edit / selection mode | **E** toggle (pauses; **T** toggles cell under cursor) |
| Save / restore layout | **,** save (M+) / **.** restore (MR) in-memory snapshot |
| Edit: move | Infinite — arrows/**WASD** pans field under fixed center cursor; wrapped — moves cursor on torus |
| Edit: coordinates | Stats bar shows `@ x,y` (logical cell) in edit mode |
| Controls overview | **C** (setup or simulation) |

Wrapped mode uses the terminal size as the toroidal grid. **High** density doubles logical rows (`▄`: background = upper cell, foreground = lower). Infinite follow pans one cell per 0.5s while the centroid stays on-screen, and snaps back if it leaves; press **F** to toggle (off by default).

Run `python gol.py --help` for pygame and TUI key reference, or `python gol_tui.py --help` / `gol-tui.exe --help` for terminal-only help.

## Modes

- **Wrapped** — Toroidal grid sized to the window aspect ratio with square cells (50 on the shorter axis; e.g. a 2:1 window is 100×50). Resizing before play recomputes the grid; window resize and zoom are disabled while running.
- **Infinite** — Sparse unbounded grid; pan and zoom freely at any time.

## Patterns

57 built-in patterns migrated from the web edition (gliders, oscillators, guns, methuselahs, spacefillers, and more). Select from the pattern picker or pass `--pattern`.

## Build (Windows)

From `python/gol/`:

```powershell
./build.ps1              # gol.pyz, gol.exe, then gol-tui.exe
./build.ps1 -pyz         # gol.pyz only
./build.ps1 -exe         # gol.exe only (pygame GUI)
./build.ps1 -tui         # gol-tui.exe only (terminal, no pygame)
./build.ps1 -exe -upx    # optional UPX compression of gol.exe
```

| Output | Usage |
|--------|--------|
| `gol.exe` | Standalone GUI executable (PyInstaller + pygame) |
| `gol-tui.exe` | Standalone terminal executable (Textual only; launches TUI by default) |
| `gol.pyz` | `python gol.pyz` — requires pygame-ce and textual installed |

Icon sources: `build/icon-32.ico` (Windows `.exe` shell icon) and `build/32x32.png` (pygame window title-bar icon).

## Changelog

### 1.1.0

- Terminal UI (Textual): setup screen, full-terminal simulation, `gol-tui.exe` build
- Controls help modal (**C**); infinite-mode follow toggle (**F**) and improved pan rules
- Dynamic wrapped grid (square cells, 50 on shorter axis); click/grid alignment fixes

### 1.0.0

- Initial Python port with pygame UI
- Wrapped and infinite modes, 57 patterns
- Play/pause/step, speed and zoom, pan, cell aging colors
- M+/MR memory save and restore
- zipapp and PyInstaller builds

## See also

- Web edition: [gol/](https://kreft.us/gol/)
- Technical reference for AI assistants: [cursor.md](cursor.md)
