# GoLPy — Conway's Game of Life

**Version 1.0**

Desktop Python port of the [web Game of Life](https://kreft.us/gol/) with pygame.

**Author:** Kreft&Cursor

## Requirements

- Python 3.11+
- Dependencies: `pip install -r requirements.txt` (`pygame-ce`)

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
| `-debug` | Log population and scope every 100 generations to stderr |

Examples:

```bash
python gol.py --mode infinite --pattern gosper
python gol.py --pattern pulsar --speed 150
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

## Modes

- **Wrapped** — Toroidal grid sized to the window aspect ratio with square cells (50 on the shorter axis; e.g. a 2:1 window is 100×50). Resizing before play recomputes the grid; window resize and zoom are disabled while running.
- **Infinite** — Sparse unbounded grid; pan and zoom freely at any time.

## Patterns

57 built-in patterns migrated from the web edition (gliders, oscillators, guns, methuselahs, spacefillers, and more). Select from the pattern picker or pass `--pattern`.

## Build (Windows)

From `python/gol/`:

```powershell
./build.ps1           # gol.pyz, then gol.exe
./build.ps1 -pyz      # gol.pyz only
./build.ps1 -exe      # gol.exe only
./build.ps1 -exe -upx # optional UPX compression of gol.exe
```

| Output | Usage |
|--------|--------|
| `gol.exe` | Standalone executable (PyInstaller) |
| `gol.pyz` | `python gol.pyz` — requires pygame-ce installed |

Icon sources: `build/icon-32.ico` (Windows `.exe` shell icon) and `build/32x32.png` (pygame window title-bar icon).

## Changelog

### 1.0.0

- Initial Python port with pygame UI
- Wrapped and infinite modes, 57 patterns
- Play/pause/step, speed and zoom, pan, cell aging colors
- M+/MR memory save and restore
- zipapp and PyInstaller builds

## See also

- Web edition: [gol/](https://kreft.us/gol/)
- Technical reference for AI assistants: [cursor.md](cursor.md)
