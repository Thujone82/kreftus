# Cursor Project File

## Project: tp

**Author:** Kreft&Cursor  
**Date:** 2026-06-15  
**Version:** 1.2.0

---

### Description

`tp` (**TemPy**) is a cross-platform Python TUI for monitoring ThermoPro TP35x (TP357/TP358/TP359) Bluetooth temperature/humidity sensors. It discovers devices, maintains a managed device list in `tp.ini`, polls readings every 5 minutes on clock-aligned boundaries, retries stale devices every minute within each chunk, displays 24-hour color-coded sparklines, preloads history from CSV, and optionally logs readings to CSV.

Built with **Textual** (UI) and **bleak** (BLE). Live-read protocol logic ported from [pasky/tp357 tp357tool.py](https://github.com/pasky/tp357). **No BLE day-history or bootstrap** — history comes from live polls plus log preload only.

---

### Key Functionality

- **Startup:** Main menu always on stack; push Monitoring if devices exist, else Manage Devices with auto-scan
- **Monitoring:** 5 rows per device; green/yellow `updated HH:MM`; sequential BLE fetch (one device at a time)
- **Scheduler:** 5-minute grid (`:00`, `:05`, …); minute retries for devices missing the current chunk
- **Startup fetch skip:** After log preload, fetch only devices stale for the current chunk (skip all if log is fresh)
- **Sparklines:** 24-bin windows — 4H/24H/72H on device status modal; 24H on dashboard (1 hour per glyph)
- **CSV logging:** Optional append-only log; 24h preload on mount/resume
- **Build:** `build.ps1` → `tp.exe` + `tp.pyz`; optional `-upx` — see **Build** section
- **CLI:** `-debug`, `-x` snapshot, `-nopoll`, `-f`/`-filter` device view filter — see **Command line**

---

### BLE Protocol

| UUID | Role |
|------|------|
| `00010203-0405-0607-0809-0a0b0c0d2b11` | Write characteristic |
| `00010203-0405-0607-0809-0a0b0c0d2b10` | Read/notify characteristic |

**`read_now` only:** Start notify on read char; wait for packet with `data[0] == 194`. Parse temp from bytes 3–4 (raw/10 → °C → °F), humidity from byte 5.

**Scan filter:** Device name starts with `TP35`. Uses `BleakScanner.discover(return_adv=True)` (bleak 3.x) for RSSI and `local_name` from `AdvertisementData`.

**Temperature units:** GATT raw values are tenths of °C; converted to °F for display, sparklines, and CSV (`temp_f`).

---

### Configuration (`tp.ini`)

Beside the launcher (`application_dir()`): `tp.py`, `tp.exe`, or `tp.pyz` parent directory. `configparser` uses `delimiters=('=',)` only (not `:`) so MAC addresses are valid keys. `interpolation=None`; `optionxform = str` preserves MAC casing.

```ini
[Settings]
LoggingEnabled=false
LogDirectory=.
LogFileName=tp.log

[Devices]
AA:BB:CC:DD:EE:FF=Living Room
```

Resolved log path: `{absolute LogDirectory}/{LogFileName}`; relative `LogDirectory` is resolved from `application_dir()`.

---

### Log CSV

| Column | Header | Format |
|--------|--------|--------|
| 1 | `timestamp` | `YYYY-MM-DD HH:MM:SS` local time at read completion |
| 2 | `device` | Friendly name |
| 3 | `temp_f` | One decimal °F |
| 4 | `humidity_pct` | Integer 0–100 |
| 5 | `mac` | Uppercase colon MAC |

Append after each fetch cycle (including partial retry cycles). UTF-8, `\n` line endings. Preload last 24h on mount; seeds `last_updated` and fetch status per device.

---

### UI Screens

| Screen | Keys | Purpose |
|--------|------|---------|
| Main | 1–4, q | Route to sub-screens; q exits |
| Monitoring | M/Esc, G, q | Dashboard; G = full fetch; header = status left, 🌡 TemPy center, clock right |
| Devices | D, A, I, E, R, W, S, ↑/↓, M, q | Discover/add/status/edit/remove/reorder |
| Options | L, D, F, M, q | Logging toggle, path edits |

**Monitoring layout (per device):**

1. Label row: name (yellow, left); `updated HH:MM` right-aligned in fixed column — **green** if fresh (≤5 min), **yellow** if stale; cyan `▶` / `◀` when actively fetching
2. Temp stats: `cur` / `min` / `max` (all color-banded; dim when stale)
3. Temp sparkline: 24 glyphs, 1 hour per bin (24H window)
4. Humidity stats: `cur` / `min` / `max` (all color-banded; dim when stale)
5. Humidity sparkline: 24 glyphs

Blank line between devices. Header shows status (DEBUG, filter, next poll / polling off, fetch progress) left, **🌡 TemPy** (center when room), clock (right). Footer = keybindings only.

**Device status modal (I):** Log preload stats, last fetch, memory count, 4H/24H/72H temp and humidity sparklines.

---

### Sparklines and Colors

**Dashboard binning (`sparkline.py`):** 24 bins × 1 hour = 24H window ending at `datetime.now()`.

**Status modal windows:** Same 24 glyphs per row; bin width scales — 4H (10 min/bin), 24H (1 h/bin), 72H (3 h/bin).

**Colors (`colors.py`):**

| Metric | Condition | Color |
|--------|-----------|-------|
| Temp °F | &lt; 55 | cyan |
| Temp °F | 55–64 | green |
| Temp °F | 65–71 | white |
| Temp °F | 72–77 | yellow |
| Temp °F | 78–81 | red |
| Temp °F | ≥ 82 | magenta |
| Humidity % | &lt; 30 | cyan |
| Humidity % | 30–60 | white |
| Humidity % | 61–70 | yellow |
| Humidity % | &gt; 70 | red |

Sparkline glyph **color** = band at bin average. Glyph **height** = normalized trend within the window.

---

### Fetch Cycle and Scheduling

**Sequential collect:** One device at a time via `read_now`; global BLE session lock prevents overlapping GATT connections. Fetch cycles are serialized with `asyncio.Lock` so the poll worker, minute retries, and **G** cannot overlap.

**Full cycle:** All managed devices at each 5-minute boundary.

**Partial cycle:** Startup (stale only), minute retries (chunk-stale only), or manual subset.

**Chunk stale:** Device with no successful reading since `floor_to_boundary(now)` for the current 5-minute chunk.

**Measurement stale (UI):** No reading within `STALE_AFTER` (5 minutes) — yellow timestamp, dimmed stats.

**Retry timing:** `next_retry_time()` — 60s after last retry or after cycle end, unless chunk boundary comes first.

**Footer states:** Last fetch · next poll · retry countdown · fetch spinner/progress · logging on/off.

---

### Navigation (`tp/ui/app.py`)

- Screen stack: always push `main` first, then `monitoring` or `devices`
- `pop_or_main_menu()`: pop if depth &gt; 1, else switch to main
- **Q** (`action_quit_or_back`): dismiss modal → pop sub-screen → exit from main

---

### Module Map

| Module | Role |
|--------|------|
| `tp.py` | Entry point, `-x` snapshot renderer, CLI argument parsing |
| `tp/config.py` | INI load/save, `application_dir()`, log path resolution, `filter_devices()` |
| `tp/ble.py` | bleak scan, `read_now` |
| `tp/history.py` | In-memory readings, log preload, CSV append, fetch status |
| `tp/scheduler.py` | 5-minute boundaries, stale/retry helpers |
| `tp/sparkline.py` | Multi-window binning, glyphs, Rich markup |
| `tp/colors.py` | GF band colors |
| `tp/fetch.py` | Parallel fetch cycle orchestration |
| `tp/ui/app.py` | Textual App root, CSS, startup routing |
| `tp/ui/menus.py` | Main menu |
| `tp/ui/monitoring.py` | Dashboard + poll/retry worker |
| `tp/ui/devices.py` | Device management + status modal |
| `tp/ui/device_status.py` | Status/sparkline formatting |
| `tp/ui/options.py` | Logging options |
| `tp/ui/helpers.py` | Label/stats formatting, aligned device rows |
| `build.ps1` | Windows build script — see **Build** section below |
| `prepare_icon.py` | ICO prep/reapply helper used by `build.ps1` |

---

### Build (`build.ps1`)

Windows PowerShell build script in `python/tp/`. Produces two distributable artifacts from the same source tree. Run from `python/tp/`:

```powershell
./build.ps1        # tp.exe + tp.pyz
./build.ps1 -upx   # optional UPX compression of tp.exe
```

#### Prerequisites

| Requirement | Notes |
|-------------|--------|
| Python 3.11+ | Must be on `PATH` as `python` |
| `tp.py` | Script must be run from `python/tp/` |
| `build/thermo.ico` | Source icon for the executable (committed) |
| UPX (optional) | Only when `-upx` is passed; must be on `PATH` |

PyInstaller is **not** pre-installed — the script runs `pip install pyinstaller` when import fails. Runtime deps come from `pip install -r requirements.txt` at the start of every build.

#### Build flow

1. **Cleanup** — Removes prior outputs before building:
   - `tp.exe`, `tp.pyz`
   - `build/pyinstaller/` (PyInstaller cache/spec/work)
   - `build/zipapp/` (zipapp staging)
   - `build/thermo-embedded.ico` (generated icon)
   - Preserves `build/thermo.ico`

2. **Dependencies** — `pip install -r requirements.txt`

3. **PyInstaller check** — Install PyInstaller if missing

4. **Output 1: `tp.exe`** (standalone Windows executable)
   - Run `prepare_icon.py build/thermo.ico build/thermo-embedded.ico` — strips PNG-compressed ICO entries so Explorer shows the custom icon (see below)
   - PyInstaller `--onefile` with absolute path to prepared icon
   - `--noupx` on PyInstaller (UPX is opt-in via script flag, not automatic)
   - Bundles `bleak`, `textual`, and winrt backends via `--hidden-import` / `--collect-submodules`
   - Writes `tp.exe` to `python/tp/`; work files under `build/pyinstaller/`

5. **Optional UPX** (only with `-upx`)
   - `upx --best --lzma tp.exe`
   - Re-applies icon via `prepare_icon.py reapply tp.exe build/thermo-embedded.ico` (UPX can strip resources)
   - Skipped with a message if UPX is not in `PATH`

6. **Output 2: `tp.pyz`** (compressed Python zipapp)
   - Stage `build/zipapp/`: copy `tp.py` → `__main__.py`, copy `tp/` package
   - Strip `__pycache__` from staging tree
   - `python -m zipapp build/zipapp -o tp.pyz -p . -c`
   - **Does not bundle** bleak/textual — target machine needs `pip install -r requirements.txt`

#### Outputs

| File | Type | Run | Dependencies bundled |
|------|------|-----|----------------------|
| `tp.exe` | PyInstaller one-file | `./tp.exe` | Python runtime + bleak + textual + winrt |
| `tp.pyz` | zipapp archive | `python tp.pyz` | App source only (~22 KB) |

Both resolve `tp.ini` and relative log paths from the **launcher directory** (`application_dir()` in `tp/config.py`). Ship `tp.ini` / `tp.log` beside the executable or zipapp.

#### Icon helper (`prepare_icon.py`)

| Command | Purpose |
|---------|---------|
| `python prepare_icon.py build/thermo.ico build/thermo-embedded.ico` | Rewrite ICO with BMP-only entries for reliable Windows embedding |
| `python prepare_icon.py reapply tp.exe build/thermo-embedded.ico` | Re-embed icon after UPX compression |

Vista-style ICO files often store 256×256 as PNG payloads. PyInstaller embeds them as-is, but Explorer may show the default PyInstaller icon. The prepare step drops PNG entries and keeps BMP sizes (128×128 max).

#### PyInstaller flags (reference)

```
--onefile --name tp --icon <abs path to thermo-embedded.ico>
--clean --noconfirm --noupx
--distpath . --workpath build/pyinstaller --specpath build/pyinstaller
--hidden-import bleak --hidden-import textual
--collect-submodules textual --collect-submodules bleak
tp.py
```

#### Git ignore (`.gitignore`)

Ignored (regenerated each build): `build/pyinstaller/`, `build/zipapp/`, `build/thermo-embedded.ico`, `__pycache__/`, local `tp.ini`, `tp.log`.

**Not** ignored: `tp.exe`, `tp.pyz` (build outputs may be committed), `build/thermo.ico`.

#### Troubleshooting

| Issue | Cause / fix |
|-------|-------------|
| `Icon not found: build\thermo.ico` | Add icon file under `build/` |
| `FileNotFoundError` icon path during PyInstaller | Fixed by absolute icon path + `--noupx`; rebuild with current script |
| Explorer shows Python/PyInstaller icon | Rebuild; ensure `prepare_icon.py` runs before PyInstaller |
| `python tp.pyz` → `FileExistsError` on `tp.ini` | Run from a directory where `tp.pyz` is a file, not a folder; fixed in `config.application_dir()` |
| UPX fails on Python 3.14+ | CFG-enabled PE files may reject UPX; omit `-upx` or use `--force` manually |

---

### Dependencies

- `bleak>=0.21` — cross-platform BLE (WinRT / BlueZ)
- `textual>=0.40` — TUI

Python 3.11+. PyInstaller installed on demand by `build.ps1`.

---

### Command line (`tp.py`)

Parsed in `tp.py` `parse_args()`; passed to `run_app()` or `_render_snapshot()`.

| Flag | Effect |
|------|--------|
| `-debug` / `--debug` | Session `debug.log` beside resolved log directory; Options **B** toggles at runtime |
| `-x` | Load config + log preload, print Rich snapshot to terminal, exit (no Textual UI, no BLE) |
| `-nopoll` | Interactive Textual UI with poll/retry worker disabled; **G** manual fetch still works; header shows “Polling off”; CSV log reloaded every 5 minutes for multi-instance viewing |
| `-f` / `-filter` *TEXT* | View filter: monitoring dashboard and `-x` output show only devices whose display name contains *TEXT* (case-insensitive substring). Polling, retries, and **G** still target all managed devices. |

Examples: `python tp.py -x -f cab`, `tp.exe -nopoll -filter guest`, `python tp.py -debug`.

`-x` with a filter that matches no devices prints `No devices match filter '…'.` and exits.

---

### How to Run

**From source:**

```bash
cd python/tp
pip install -r requirements.txt
python tp.py
python tp.py -x -f office    # one-shot snapshot, filtered
python tp.py -nopoll         # UI only, no scheduled polls
```

**From build outputs** (after `./build.ps1`):

```powershell
./tp.exe
python tp.pyz   # requires bleak/textual installed on target machine
```

Place `tp.ini` and optional `tp.log` in the same folder as `tp.exe` or `tp.pyz`.

---

### File Structure

```
python/tp/
  tp.py
  tp.exe / tp.pyz     # build outputs
  requirements.txt
  build.ps1
  prepare_icon.py
  build/thermo.ico
  tp.ini              # auto-created beside launcher
  tp.log              # optional CSV log
  .gitignore
  README.md
  cursor.md
  tp/
    __init__.py
    ble.py
    config.py
    colors.py
    sparkline.py
    history.py
    scheduler.py
    fetch.py
    ui/
      app.py
      menus.py
      monitoring.py
      devices.py
      device_status.py
      options.py
      helpers.py
```

---

### Changelog

- **v1.2.0** — Indoor temp color bands; cur/min/max stat coloring; CLI `-x` snapshot; `-nopoll`; `-f`/`-filter` device view filter; header layout (status / title / clock); 4H/24H/72H status sparklines; per-device 60s read timeout; BLE connect optimizations.
- **v1.1.0** — Parallel fetch; minute retries; stale UI (green/yellow); log preload skip on startup; device status 1H/8H/24H sparklines; launcher-relative paths; build.ps1; removed BLE day-history/bootstrap; startup routing and navigation fixes.
- **v1.0.0** — Initial release: Textual TUI, bleak BLE, 5-row monitoring layout, CSV logging, device management.
