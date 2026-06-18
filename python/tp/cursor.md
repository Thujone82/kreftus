# Cursor Project File

## Project: tp

**Author:** Kreft&Cursor  
**Date:** 2026-06-15  
**Version:** 1.5.0

---

### Description

`tp` (**TemPy**) is a cross-platform Python TUI for monitoring ThermoPro TP35x (TP357/TP358/TP359) Bluetooth temperature/humidity sensors. It discovers devices, maintains a managed device list in `tp.ini`, polls readings every 5 minutes on clock-aligned boundaries, retries stale devices every minute within each chunk, displays 24-hour color-coded sparklines, preloads history from CSV, and optionally logs readings to CSV.

Built with **Textual** (UI) and **bleak** (BLE). Live reads use the TP35x GATT notify path; **24H BLE day-history** uses the TP357S/TP359 stream protocol (with legacy TP357 `0xA7` fallback). Fetch via **H** on Manage Devices or when adding a device; merges only the received timestamp span so older polled/log data is preserved.

---

### Key Functionality

- **Startup:** Main menu always on stack; push Monitoring if devices exist, else Manage Devices with auto-scan
- **Monitoring:** 5 rows per device; green/yellow device name by freshness; fetch arrows show BLE step (cyan connect / green sync read / yellow passive); sequential BLE fetch (one device at a time, 60 s timeout); optional multi-column layout (**C**)
- **Scheduler:** 5-minute grid (`:00`, `:05`, …); minute retries for devices missing the current chunk
- **Startup fetch skip:** After log preload, fetch only devices stale for the current chunk (skip all if log is fresh)
- **Sparkline bootstrap:** When `LoggingEnabled=false`, pull 24H BLE history on monitoring mount for devices with sparse sparklines (before live polling)
- **Sparklines:** 24-bin windows — 4H/24H/72H on device status modal; 24H on dashboard (1 hour per glyph)
- **CSV logging:** Optional append-only log; 24h preload on mount/resume
- **24H fetch:** Manage Devices **H** — BLE minute history for selected device; replaces only the received timestamp span in memory/log (older polled/log data outside that span is preserved); CSV last-24h rows for that MAC in the same span replaced only when `LoggingEnabled=true`
- **BLE recovery:** Auto power-cycle Bluetooth radio when bleak reports `POWERED_OFF` (`ble_radio.py`); 90 s cooldown
- **BLE connect cache:** 120 s `BLEDevice` resolution cache, preferred WinRT connect strategy, inter-device prefetch (`ble.py`)
- **Build:** `build.ps1` → `tp.pyz` then `tp.exe`; optional `-upx` — see **Build** section
- **CLI:** `-debug`, `-x` snapshot, `-nopoll`/`-np`, `-f`/`-filter` device view filter, `--history-day MAC` — see **Command line**

---

### BLE Protocol

| UUID | Role |
|------|------|
| `00010203-0405-0607-0809-0a0b0c0d2b11` | Write characteristic |
| `00010203-0405-0607-0809-0a0b0c0d2b10` | Read/notify characteristic |

**`read_now`:** Two-step live read on the same GATT UUIDs:

1. **Fast path (TP357S/TP358/TP359):** `start_notify` → write datetime sync `0xA5` (same body as history) → wait for `0xC2` (`NOW_OPCODE` 194) within 10 s. Temp: signed LE bytes 3–4, tenths °C → °F; humidity byte 5.
2. **Passive fallback (legacy TP357):** `start_notify` only → wait for unsolicited `0xC2` within 30 s.

Connect path caches resolved `BLEDevice` for 120 s, prefetches the next device during the 2 s inter-device gap, and retries with extended scan (10 s) after a quick scan (5 s) miss. On `BleakBluetoothNotAvailableReason.POWERED_OFF`, `ble_radio.restart_bluetooth_radio()` toggles the system radio off/on (WinRT on Windows; `rfkill` / `bluetoothctl` on Linux) once per 90 s, then retries.

**`read_day_history`:** Two protocols on the same GATT UUIDs:

1. **TP357S / TP358 / TP359 (stream):** Write datetime sync `0xA5 YY MM DD HH MM SS DOW CS` (checksum = sum of body bytes mod 256), then three `0xCCCC…` commands (session init, offset placeholder, data request with 16-bit LE record count). Collect notify chunks from `cc cc` through trailing `66 66`; each record is 3 bytes (signed LE temp×10, humidity). Timestamps: most-recent = fetch minute, step back 1 minute per record. Reference: [pytp357s PROTOCOL.md](https://github.com/giovannipizzi/pytp357s/blob/main/PROTOCOL.md). Legacy `0xA7` on these models only echoes live `0xC2` readings.

2. **Original TP357 (legacy):** Write `[0xA7, 0x01, 0x00, 0x7A]` (fallback: 6-byte tpy357 variant). Collect packets while `data[0] == 0xA7`; five samples per packet; tpy357 timestamps. Used only when stream protocol returns no data.

Timeout ~180s. Uses `_ble_session_lock`. Minimum 100 valid samples before merge.

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
| Monitoring | M/Esc, G, 1–9/0, C, q | Dashboard; G = full fetch; digit keys = device info; C = cycle columns when wide enough; header = status left, 🌡 TemPy center, clock right |
| Devices | D, A, I, H, E, R, W, S, ↑/↓, M, q | Discover/add/status/24H fetch/edit/remove/reorder |
| Options | L, B, D, F, M, q | Logging toggle, debug log toggle, path edits |

**Monitoring layout (per device):**

1. Label row: device name — **green** if fresh (≤5 min), **yellow** if stale; while fetching, `▶` / `◀` show BLE step — **cyan** connecting, **green** sync live read, **yellow** passive fallback
2. Temp stats: `cur` / `min` / `max` (all color-banded; dim when stale)
3. Temp sparkline: 24 glyphs, 1 hour per bin (24H window)
4. Humidity stats: `cur` / `min` / `max` (all color-banded; dim when stale)
5. Humidity sparkline: 24 glyphs

Blank line between single-column device blocks; multi-column rows are separated by a blank line. Header shows status (DEBUG, filter, next poll / polling off, fetch progress) left, **🌡 TemPy** (center when room), clock (right). Footer shows `m` / `g` / `1-x info` (first 10 visible devices; `0` = 10th) / `c` Columns (when terminal width supports 2+ columns).

**Device info from monitoring:** Keys **1**–**9** and **0** open `DeviceStatusModal` for the corresponding visible device (respects `-f` filter). **Q** dismisses the modal back to monitoring (same as Manage Devices **I** status).

**Multi-column layout (`helpers.py` + `monitoring.py`):** Default 1 column. When `area_width // (block_width + 4) ≥ 2`, footer offers **c Columns** and **C** cycles `1 … max`. Row-major order. `block_width` = max plain-text line width across visible devices; 2-char pad each side per column. View filter affects visible devices only; polling always targets all managed devices.

**Device status modal (I):** Log preload stats, last fetch (with timestamp), memory count, 4H/24H/72H temp and humidity sparklines.

**Add discovered device (A):** Name prompt, then optional **Y/N** prompt to load 24H BLE history (opens the same progress modal as **H**).

**24H history fetch modal (H):** Progress modal while BLE day-history streams; shows phase, packet/sample counts, elapsed time. On success merges only the received timestamp span into memory and optionally rewrites matching CSV rows for that device when logging is enabled. **Q** blocked until complete.

**Startup sparkline bootstrap (`monitoring.py` + `history_fetch.bootstrap_sparklines_from_ble`):** When `LoggingEnabled=false`, before the poll worker’s first live fetch, sequentially pull 24H BLE history for each device with fewer than 8 populated hourly bins. Header shows **24H** progress; uses same merge rules as **H**. Skipped when logging is on or bins are already filled (e.g. from log preload). Runs once per monitoring mount; also triggered on `-nopoll` mount via background worker.

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

**Sequential collect:** One device at a time via `read_now` (60 s `DEVICE_READ_TIMEOUT` per device); global BLE session lock prevents overlapping GATT connections. Fetch cycles are serialized with `asyncio.Lock` so the poll worker, minute retries, and **G** cannot overlap.

**Full cycle:** All managed devices at each 5-minute boundary.

**Partial cycle:** Startup (stale only), minute retries (chunk-stale only), or manual subset.

**Chunk stale:** Device with no successful reading since `floor_to_boundary(now)` for the current 5-minute chunk.

**Measurement stale (UI):** No reading within `STALE_AFTER` (5 minutes) — yellow device name, dimmed stats/sparklines.

**Retry timing:** `next_retry_time()` — 60s after last retry or after cycle end, unless chunk boundary comes first.

**Header states:** DEBUG · filter · next poll / polling off · fetch / retry / **24H** (startup bootstrap) / saving spinner with active device name · progress bar.

**`-nopoll` / `-np` mode:** Poll/retry worker disabled; **G** still fetches; `tp.log` reloaded every 5 minutes (`POLL_INTERVAL`).

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
| `tp/ble.py` | bleak scan, `read_now`, `read_day_history`, device cache, radio-recovery hooks |
| `tp/ble_radio.py` | Detect Bluetooth powered off; WinRT/Linux radio power-cycle |
| `tp/history.py` | In-memory readings, log preload, CSV append, fetch status, day-history merge, sparkline bootstrap gate |
| `tp/history_fetch.py` | Orchestrate 24H BLE fetch + history merge; startup `bootstrap_sparklines_from_ble` |
| `tp/scheduler.py` | 5-minute boundaries, stale/retry helpers |
| `tp/sparkline.py` | Multi-window binning, glyphs, Rich markup |
| `tp/colors.py` | Indoor temp/humidity band colors |
| `tp/fetch.py` | Sequential fetch cycle orchestration |
| `tp/ui/app.py` | Textual App root, CSS, startup routing |
| `tp/ui/menus.py` | Main menu |
| `tp/ui/monitoring.py` | Dashboard + poll/retry worker |
| `tp/ui/devices.py` | Device management, add flow, history fetch modal |
| `tp/ui/history_fetch_status.py` | 24H fetch progress modal formatting |
| `tp/ui/device_status.py` | Status/sparkline formatting |
| `tp/ui/options.py` | Logging options |
| `tp/ui/helpers.py` | Label/stats formatting, multi-column layout, info hotkey helpers |
| `build.ps1` | Windows build script — see **Build** section below |
| `prepare_icon.py` | ICO prep/reapply helper used by `build.ps1` |

---

### Build (`build.ps1`)

Windows PowerShell build script in `python/tp/`. Produces two distributable artifacts from the same source tree. Run from `python/tp/`:

```powershell
./build.ps1        # tp.pyz, then tp.exe
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

3. **Output 1: `tp.pyz`** (compressed Python zipapp)
   - Stage `build/zipapp/`: copy `tp.py` → `__main__.py`, copy `tp/` package
   - Strip `__pycache__` from staging tree
   - `python -m zipapp build/zipapp -o tp.pyz -p . -c`
   - **Does not bundle** bleak/textual — target machine needs `pip install -r requirements.txt`
   - Prints `tp.pyz build complete.`

4. **PyInstaller check** — Install PyInstaller if missing

5. **Output 2: `tp.exe`** (standalone Windows executable)
   - Run `prepare_icon.py build/thermo.ico build/thermo-embedded.ico` — strips PNG-compressed ICO entries so Explorer shows the custom icon (see below)
   - PyInstaller `--onefile` with absolute path to prepared icon
   - `--noupx` on PyInstaller (UPX is opt-in via script flag, not automatic)
   - Bundles `bleak`, `textual`, and winrt backends via `--hidden-import` / `--collect-submodules`
   - Writes `tp.exe` to `python/tp/`; work files under `build/pyinstaller/`
   - Prints `tp.exe build complete.`

6. **Optional UPX** (only with `-upx`)
   - `upx --best --lzma tp.exe`
   - Re-applies icon via `prepare_icon.py reapply tp.exe build/thermo-embedded.ico` (UPX can strip resources)
   - Skipped with a message if UPX is not in `PATH`

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
| `-nopoll` / `-np` | Interactive Textual UI with poll/retry worker disabled; **G** manual fetch still works; header shows “Polling off”; CSV log reloaded every 5 minutes for multi-instance viewing |
| `-f` / `-filter` *TEXT* | View filter: monitoring dashboard and `-x` output show only devices whose display name contains *TEXT* (case-insensitive substring). Polling, retries, and **G** still target all managed devices. |

Examples: `python tp.py -x -f cab`, `tp.exe -np -filter guest`, `python tp.py -debug`.

`-x` with a filter that matches no devices prints `No devices match filter '…'.` and exits.

---

### How to Run

**From source:**

```bash
cd python/tp
pip install -r requirements.txt
python tp.py
python tp.py -x -f office    # one-shot snapshot, filtered
python tp.py -np              # UI only, no scheduled polls
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
    history_fetch.py
    ble_radio.py
    scheduler.py
    fetch.py
    ui/
      app.py
      menus.py
      monitoring.py
      devices.py
      history_fetch_status.py
      device_status.py
      options.py
      helpers.py
```

---

### Changelog

- **v1.5.0** — **Fast live read** (datetime sync `0xA5` then `0xC2`, passive fallback); **fetch step arrows** (cyan/green/yellow); **BLE connect cache** + inter-device prefetch; **startup 24H bootstrap** when logging off; **Bluetooth radio auto-restart** on powered-off errors (`ble_radio.py`); unit tests for radio detection and bootstrap gating.
- **v1.4.0** — **24H BLE history fetch** (**H** progress modal; optional **Y/N** when adding a device); TP357S/TP359 stream protocol (`0xA5` datetime sync + `0xCCCC` history commands) with legacy TP357 `0xA7` fallback; partial-span merge preserves polled/log data outside the received window; `--history-day` CLI; unit tests for stream/legacy parsers and log merge.
- **v1.3.0** — Multi-column monitoring dashboard (**C**); `-np` alias for `-nopoll`; device label freshness colors (no `updated` on row); header fetch device name fix; `build.ps1` builds `tp.pyz` before `tp.exe` with per-artifact completion messages.
- **v1.2.0** — Indoor temp color bands; cur/min/max stat coloring; CLI `-x` snapshot; `-nopoll`; `-f`/`-filter` device view filter; header layout (status / title / clock); 4H/24H/72H status sparklines; per-device 60s read timeout; BLE connect optimizations.
- **v1.1.0** — Parallel fetch; minute retries; stale UI (green/yellow); log preload skip on startup; device status 1H/8H/24H sparklines; launcher-relative paths; build.ps1; removed BLE day-history/bootstrap; startup routing and navigation fixes.
- **v1.0.0** — Initial release: Textual TUI, bleak BLE, 5-row monitoring layout, CSV logging, device management.
