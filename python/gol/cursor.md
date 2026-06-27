# Cursor Project File

## Project: gol (GoLPy)

**Author:** Kreft&Cursor  
**Date:** 2026-06-27  
**Version:** 1.0.0

---

### Description

`gol` (**GoLPy**) is a pygame desktop port of the Kreft.us [Game of Life](https://kreft.us/gol/) web app. It implements Conway's B3/S23 rules in **wrapped** (toroidal grid sized to the window, 50 cells on the shorter axis) and **infinite** (sparse unbounded) modes, with 57 built-in patterns, play/pause/step controls, pan/zoom, cell aging HSL colors, and in-memory M+/MR snapshots.

**Not ported:** `pattern.html` RLE designer (use web tool), footer branding, PWA/service worker.

---

### Key Functionality

- **Engine:** Pure-Python `GameOfLife` in `gol/engine.py` — testable without pygame
- **Wrapped mode:** Dense toroidal scan sized to the play area — square cells, 50 on the shorter axis, columns/rows follow window aspect ratio (`wrapped_grid_layout()` in `patterns.py`)
- **Infinite mode:** Candidate set from live cells ±1
- **Patterns:** `gol/patterns.json` bundled via `importlib.resources`; `extract_patterns.py` regenerates from `gol/index.html`
- **UI:** `gol/ui/app.py` main loop; `gol/ui/controls.py` toolbar, sliders, pattern overlay
- **CLI:** `--mode`, `--pattern`, `--speed`, `-debug`
- **Build:** `build.ps1` → `gol.pyz` + `gol.exe`; `.exe` icon from `build/icon-32.ico`, window icon from `build/32x32.png`

---

### Module map

| Module | Role |
|--------|------|
| `gol.py` | Entry point, argparse, path bootstrap |
| `gol/engine.py` | Grid state, dynamic `grid_cols`/`grid_rows`, `step()`, snapshot/restore |
| `gol/patterns.py` | Load JSON, `center_pattern()`, `wrapped_grid_layout()` |
| `gol/patterns.json` | 57 patterns with labels |
| `gol/colors.py` | HSL cell aging |
| `gol/config.py` | `application_dir()` |
| `gol/ui/app.py` | pygame window, render, input |
| `gol/ui/controls.py` | Buttons, sliders, pattern picker |
| `extract_patterns.py` | Dev helper: HTML → JSON |

---

### Pattern JSON schema

```json
{
  "glider": {
    "label": "Glider",
    "cells": [[1, 0], [2, 1], ...]
  }
}
```

Coordinates are relative to pattern bounding box top-left, matching web `P` object.

---

### UI event flow

1. Toolbar buttons → play/step/reset/M+/MR/mode/pattern
2. Pattern overlay → `load_pattern(name)` centers on viewport
3. Canvas click (no drag) → `toggle_cell`
4. Drag on canvas → pan (wrapped only when paused)
5. Wheel / zoom slider → `handle_zoom` toward cursor (wrapped only when paused)
6. Sim loop: delay = `16ms × (200/speed)`; wrapped+running locks zoom 1×, disables window resize, freezes grid dimensions

---

### Build

Run from `python/gol/`:

```powershell
./build.ps1           # both
./build.ps1 -pyz
./build.ps1 -exe
./build.ps1 -exe -upx
```

- **zipapp:** `gol.py` → `build/zipapp/__main__.py` + `gol/` package
- **Icon:** `python prepare_icon.py build/icon-32.ico build/gol-embedded.ico`
- **PyInstaller:** `--onefile --noupx --add-data gol/patterns.json;gol --hidden-import pygame --collect-submodules pygame`

---

### Tests

```bash
cd python/gol
python -m unittest discover -s tests -v
```

- `test_engine.py` — B3/S23, blinker, glider, wrap, snapshot
- `test_patterns.py` — JSON validity, centering

---

### File structure

```
python/gol/
  gol.py
  gol/
    __init__.py
    engine.py
    patterns.py
    patterns.json
    colors.py
    config.py
    icon.py
    assets/
      32x32.png
    ui/
      app.py
      controls.py
  tests/
  extract_patterns.py
  requirements.txt
  build.ps1
  prepare_icon.py
  README.md
  cursor.md
  README.html

  build/
    icon-32.ico           # committed source icon for gol.exe
    32x32.png             # window title-bar icon (pygame); synced to gol/assets/

golpy/index.html          # project landing page at repo root (not on main index yet)
```

---

### Dependencies

- `pygame-ce` — imports as `pygame`; used for window, render, input
