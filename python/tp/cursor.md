# Cursor Project File

## Project: tp

**Author:** Kreft&Cursor  
**Date:** 2026-07-15  
**Version:** 1.9.0

---

### Description

`tp` (**TemPy**) is a cross-platform Python TUI for monitoring ThermoPro TP35x (TP357/TP358/TP359) Bluetooth temperature/humidity sensors. It discovers devices, maintains a managed device list in `tp.ini`, polls readings every 5 minutes on clock-aligned boundaries, retries stale devices every minute within each chunk, displays color-coded sparklines (Less: 4H/24H/72H; More adds 8H/12H/36H/90M), preloads history from CSV, and optionally logs readings to CSV.

Built with **Textual** (UI) and **bleak** (BLE). Default **incremental poll mode** pulls minute-aligned history from the sensor buffer each cycle; **live poll mode** uses a single GATT notify read. **BLE history fetch** uses the TP357S/TP359 stream protocol (with legacy TP357 `0xA7` fallback), requesting up to **1 year** of minute records in **7-day** BLE chunks. Fetch via **H** (History Fetch) on Manage Devices or when adding a device; merges only the received timestamp span so older polled/log data is preserved.

---

### Key Functionality

- **Startup:** Main menu always on stack; push Monitoring if devices exist, else Manage Devices with auto-scan
- **Monitoring:** 5 rows per device; green/yellow device name by freshness; fetch arrows show BLE step (cyan connect / green sync read / yellow passive); sequential BLE fetch (one device at a time, 60 s timeout); optional multi-column layout (**C**); **T** / **Shift+T** cycle dashboard sparkline window forward / reverse (Less: 24H → 72H → 4H; More: 24H → 36H → 72H → 90M → 4H → 8H → 12H)
- **Scheduler:** 5-minute grid (`:00`, `:05`, …); minute retries for devices missing the current chunk
- **Startup fetch skip:** After log preload, fetch only devices stale for the current chunk (skip all if log is fresh)
- **Sparkline bootstrap:** When `LoggingEnabled=false`, pull 72H BLE history on monitoring mount for devices with sparse sparklines (before live polling)
- **Incremental polling:** Default `PollMode=incremental` — each cycle requests missing minute history since last stored reading (`read_recent_history`); logs multiple minute rows per cycle; falls back to `read_now` on failure
- **Live polling:** `PollMode=live` — one `read_now` snapshot per device per cycle (wall-clock timestamp)
- **Time detail:** Options **W** / `TimeDetail=less|more` — Less (default 4H/24H/72H) or More (adds 8H/12H/36H/90M) for dashboard **T**, device status, and `-x` snapshot; CLI `-more` overrides to More for one session without writing `tp.ini`
- **Sparklines:** 24-bin windows per active time-detail set; dashboard defaults to 24H (**T** cycles the set)
- **Log export to web:** Main menu **5** or Options **E** writes `tp_export.html` beside launcher; embedded CSV data; browser UI for device + timeframe (4H/24H/72H/7D/All) with ECharts dual-axis chart
- **CSV logging:** Optional append-only log; default `tp_log.csv`; 72h preload on mount/resume; renaming log file in Options renames on disk (overwrite prompt if target exists)
- **History fetch:** Manage Devices **H** — BLE minute history for selected device (up to 1 year); replaces only the received timestamp span in memory/log (older polled/log data outside that span is preserved); CSV rows for that MAC in the same span replaced only when `LoggingEnabled=true`
- **BLE recovery:** Prompt before enabling Bluetooth when the radio is off (`ble_radio.py` + `BluetoothPermissionModal`); auto power-cycle after entire fetch cycle fails; 90 s action cooldown, 5 min re-prompt cooldown after decline
- **BLE connect cache:** 120 s `BLEDevice` resolution cache, preferred WinRT connect strategy, inter-device prefetch (`ble.py`)
- **Build:** `build.ps1` → `tp.pyz` then `tp.exe` (or `-pyz` / `-exe` alone); optional `-upx` — see **Build** section
- **CLI:** `-debug`, `-x` snapshot, `-more` session More time detail, `-nopoll`/`-np`, `-f`/`-filter` device view filter, `--history-day MAC` — see **Command line**

---

### BLE Protocol

| UUID | Role |
|------|------|
| `00010203-0405-0607-0809-0a0b0c0d2b11` | Write characteristic |
| `00010203-0405-0607-0809-0a0b0c0d2b10` | Read/notify characteristic |

**`read_now`:** Two-step live read on the same GATT UUIDs:

1. **Fast path (TP357S/TP358/TP359):** `start_notify` → write datetime sync `0xA5` (same body as history) → wait for `0xC2` (`NOW_OPCODE` 194) within 10 s. Temp: signed LE bytes 3–4, tenths °C → °F; humidity byte 5.
2. **Passive fallback (legacy TP357):** `start_notify` only → wait for unsolicited `0xC2` within 30 s.

Connect path caches resolved `BLEDevice` for 120 s, prefetches the next device during the 2 s inter-device gap, and retries with extended scan (10 s) after a quick scan (5 s) miss. When Bluetooth is disabled, `is_bluetooth_radio_disabled()` detects the state and `ensure_bluetooth_enabled_for_polling()` shows `BluetoothPermissionModal` (Y/N) before `enable_bluetooth_radio()`. On `BleakBluetoothNotAvailableReason.POWERED_OFF` or scan/read errors, the same enable prompt is used. After a whole-fleet failure, `restart_bluetooth_radio()` power-cycles the adapter automatically (no prompt).

**`read_day_history`:** Two protocols on the same GATT UUIDs:

1. **TP357S / TP358 / TP359 (stream):** Write datetime sync `0xA5 YY MM DD HH MM SS DOW CS` (checksum = sum of body bytes mod 256), then three `0xCCCC…` commands (session init, offset placeholder, data request with 16-bit LE record count). Collect notify chunks from `cc cc` through trailing `66 66`; each record is 3 bytes (signed LE temp×10, humidity). Timestamps: most-recent = fetch minute, step back 1 minute per record. Reference: [pytp357s PROTOCOL.md](https://github.com/giovannipizzi/pytp357s/blob/main/PROTOCOL.md). Legacy `0xA7` on these models only echoes live `0xC2` readings.

2. **Original TP357 (legacy):** Write `[0xA7, 0x01, 0x00, 0x7A]` (fallback: 6-byte tpy357 variant). Collect packets while `data[0] == 0xA7`; five samples per packet; tpy357 timestamps. Used only when stream protocol returns no data.

Timeout scales with record count (`day_history_timeout`). Full history fetch requests up to **525600** records (365 days at 1 min/record) in **7-day** BLE chunks (`HISTORY_FETCH_CHUNK_RECORDS`) for faster first progress. Startup bootstrap still pulls **72H** when sparklines are empty. History fetch modal shows **waiting** status with poll/bootstrap detail when queued on `_ble_session_lock`. **`read_recent_history`:** same stream path with variable record count (1–1440 per poll); shorter timeout scaled to count. Uses `_ble_session_lock`. Minimum 100 valid samples before full-day merge. Merge window: **1 year** (`BLE_HISTORY_HOURS`).

**Scan filter:** Device name starts with `TP35`. Uses `BleakScanner.discover(return_adv=True)` (bleak 3.x) for RSSI and `local_name` from `AdvertisementData`.

**Temperature units:** GATT raw values are tenths of °C; converted to °F for display, sparklines, and CSV (`temp_f`).

**False sentinel filter (`history.py`):** Consecutive runs of exactly **32.0 °F** and **10%** humidity (two or more in a row) are treated as bogus sensor errors — dropped before memory merge and CSV logging. A single isolated 32/10 reading is kept.

---

### Configuration (`tp.ini`)

Beside the launcher (`application_dir()`): `tp.py`, `tp.exe`, or `tp.pyz` parent directory. `configparser` uses `delimiters=('=',)` only (not `:`) so MAC addresses are valid keys. `interpolation=None`; `optionxform = str` preserves MAC casing.

```ini
[Settings]
LoggingEnabled=false
LogDirectory=.
LogFileName=tp_log.csv
PollMode=incremental
TimeDetail=less

[Devices]
AA:BB:CC:DD:EE:FF=Living Room
```

`PollMode`: `incremental` (default) or `live`. Existing installs keep their saved `LogFileName` if already set.
`TimeDetail`: `less` (default: 4H/24H/72H) or `more` (adds 8H/12H/36H/90M for dashboard **T**, device status, and `-x`).

Resolved log path: `{absolute LogDirectory}/{LogFileName}`; relative `LogDirectory` is resolved from `application_dir()`. Changing `LogFileName` in Options renames the existing log file; if the target name already exists, a Y/N overwrite prompt is shown (directory writability is checked without creating the target file first).

---

### Log CSV

| Column | Header | Format |
|--------|--------|--------|
| 1 | `timestamp` | `YYYY-MM-DD HH:MM:SS` local time — minute-aligned in incremental poll mode |
| 2 | `device` | Friendly name |
| 3 | `temp_f` | One decimal °F |
| 4 | `humidity_pct` | Integer 0–100 |
| 5 | `mac` | Uppercase colon MAC |

Append after each fetch cycle (including partial retry cycles). Incremental mode may append multiple rows per device per cycle. UTF-8, `\n` line endings. Preload last 72h on mount; seeds `last_updated` and fetch status per device.

**Log export (`log_export.py` + `assets/log_export.html`):** Reads full CSV (managed devices only), applies false-sentinel filter, embeds JSON in standalone HTML. Output: `{application_dir()}/tp_export.html`. Browser controls: device select, timeframe 4H/24H/72H/7D/All. Chart: Apache ECharts (CDN) dual Y-axis.

---

### UI Screens

| Screen | Keys | Purpose |
|--------|------|---------|
| Main | 1–5, q | Route to sub-screens; **5** = export log to web; q exits |
| Monitoring | M/Esc, G, T / Shift+T, 1–9/0, C, q | Dashboard; G = full fetch; T / Shift+T = cycle sparkline window forward / reverse (set by TimeDetail); digit keys = device info; C = cycle columns when wide enough; header = status left, 🌡 TemPy center, clock right |
| Manage Devices | D, A, I, H, E, R, W, S, ↑/↓, M, q | Discover/add/status/history fetch/edit/remove/reorder |
| Options | L, P, W, E, B, D, F, M, q | Logging toggle, poll mode, time detail (Less/More), log export, debug log toggle, path edits (filename rename + overwrite prompt) |

**Monitoring layout (per device):**

1. Label row: device name — **green** if fresh (≤10 min), **yellow** if stale; while fetching, `▶` / `◀` show BLE step — **cyan** connecting, **green** sync live read, **yellow** passive fallback
2. Temp stats: `cur` / `min` / `max` (all color-banded; dim when stale)
3. Temp sparkline: 24 glyphs (default 24H window; **T** / **Shift+T** cycle Less or More set forward / reverse from Options **W**)
4. Humidity stats: `cur` / `min` / `max` (all color-banded; dim when stale)
5. Humidity sparkline: 24 glyphs

Blank line between single-column device blocks; multi-column rows are separated by a blank line. Header shows status (DEBUG, filter, next poll / polling off, fetch progress) left, **🌡 TemPy** (center when room), clock (right). Footer shows `m` / `g` / `1-x info` (first 10 visible devices; `0` = 10th) / `c` Columns (when terminal width supports 2+ columns).

**Device info from monitoring:** Keys **1**–**9** and **0** open `DeviceStatusModal` for the corresponding visible device (respects `-f` filter). **Q** dismisses the modal back to monitoring (same as Manage Devices **I** status).

**Multi-column layout (`helpers.py` + `monitoring.py`):** Default 1 column. When `area_width // (block_width + 4) ≥ 2`, footer offers **c Columns** and **C** cycles `1 … max`. Row-major order. `block_width` = max plain-text line width across visible devices; 2-char pad each side per column. View filter affects visible devices only; polling always targets all managed devices.

**Device status modal (I):** Log preload stats, last fetch (with timestamp), memory count, multi-window temp and humidity sparklines (Less: 4H/24H/72H; More adds 8H/12H/36H/90M).

**Add discovered device (A):** Name prompt, then optional **Y/N** prompt to load sensor history (opens the same progress modal as **H**).

**History fetch modal (H):** Progress modal while BLE history streams; shows phase, packet/sample counts, elapsed time. On success merges only the received timestamp span into memory and optionally rewrites matching CSV rows for that device when logging is enabled. **Q** cancels.

**Startup sparkline bootstrap (`monitoring.py` + `history_fetch.bootstrap_sparklines_from_ble`):** When `LoggingEnabled=false`, before the poll worker’s first live fetch, sequentially pull **72H** BLE history for each device with fewer than 8 populated hourly bins. Header shows **History** progress; uses same merge rules as **H**. Skipped when logging is on or bins are already filled (e.g. from log preload). Runs once per monitoring mount; also triggered on `-nopoll` mount via background worker.

---

### Sparklines and Colors

**Dashboard binning (`sparkline.py`):** 24 bins per row; window length set by **T** using Options `TimeDetail` — Less: 4H (10 min/bin), 24H (1 h/bin, default), 72H (3 h/bin); More also includes 8H / 12H / 36H / 90M.

**Status modal / `-x` windows:** Same 24 glyphs per row for each window in the active time-detail set.

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

**Poll modes (`tp/poll.py`, `tp/fetch.py`):**

| Mode | Setting | Behavior |
|------|---------|----------|
| Incremental (default) | `PollMode=incremental` | `read_recent_history(mac, count)` — `count` = minutes since last stored reading (1–1440); minute-aligned timestamps; multiple rows per cycle to CSV |
| Live | `PollMode=live` | `read_now` — one snapshot per device; wall-clock timestamp |

Incremental falls back to live read on failure. Options **P** toggles modes.

**Sequential collect:** One device at a time; global BLE session lock prevents overlapping GATT connections. Fetch cycles are serialized with `asyncio.Lock` so the poll worker, minute retries, and **G** cannot overlap.

**Full cycle:** All managed devices at each 5-minute boundary.

**Partial cycle:** Startup (stale only), minute retries (chunk-stale only), or manual subset.

**Chunk stale:** Device with no successful reading since `floor_to_boundary(now)` for the current 5-minute chunk.

**Measurement stale (UI):** No reading within `STALE_AFTER` (10 minutes) — yellow device name, dimmed stats/sparklines.

**Retry timing:** `next_retry_time()` — 60s after last retry or after cycle end, unless chunk boundary comes first.

**Header states:** DEBUG · filter · next poll / polling off · fetch / retry / **72H** (startup bootstrap) / saving spinner with active device name · progress bar.

**`-nopoll` / `-np` mode:** Poll/retry worker disabled; **G** still fetches; CSV log reloaded every 5 minutes (`POLL_INTERVAL`).

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
| `tp/ble.py` | bleak scan, `read_now`, `read_day_history`, `read_recent_history`, device cache, radio-recovery hooks |
| `tp/ble_radio.py` | Detect Bluetooth powered off; permission callback for enable; WinRT/Linux enable + restart |
| `tp/history.py` | In-memory readings, log preload, CSV append, fetch status, day-history merge, sparkline bootstrap gate |
| `tp/history_fetch.py` | Orchestrate BLE history fetch + history merge; startup `bootstrap_sparklines_from_ble` (72H) |
| `tp/poll.py` | Incremental history record count; poll mode helpers |
| `tp/scheduler.py` | 5-minute boundaries, stale/retry helpers |
| `tp/sparkline.py` | Multi-window binning, glyphs, Rich markup |
| `tp/colors.py` | Indoor temp/humidity band colors |
| `tp/fetch.py` | Sequential fetch cycle (incremental or live) |
| `tp/log_export.py` | CSV → embedded JSON → standalone `tp_export.html` |
| `tp/assets/log_export.html` | ECharts report template (CDN) |
| `tp/ui/app.py` | Textual App root, CSS, startup routing |
| `tp/ui/menus.py` | Main menu |
| `tp/ui/monitoring.py` | Dashboard + poll/retry worker |
| `tp/ui/devices.py` | Device management, add flow, history fetch modal |
| `tp/ui/history_fetch_status.py` | History fetch progress modal formatting |
| `tp/ui/device_status.py` | Status/sparkline formatting |
| `tp/ui/options.py` | Logging, poll mode, path edits, log rename, log export |
| `tp/ui/log_export_action.py` | Main menu / Options export trigger + browser open |
| `tp/ui/bluetooth_prompt.py` | Y/N modal before enabling Bluetooth |
| `tp/ui/helpers.py` | Label/stats formatting, multi-column layout, info hotkey helpers |
| `build.ps1` | Windows build script — see **Build** section below |
| `prepare_icon.py` | ICO prep/reapply helper used by `build.ps1` |

---

### Build (`build.ps1`)

Windows PowerShell build script in `python/tp/`. Produces two distributable artifacts from the same source tree. Run from `python/tp/`:

```powershell
./build.ps1           # tp.pyz, then tp.exe
./build.ps1 -pyz      # tp.pyz only
./build.ps1 -exe      # tp.exe only
./build.ps1 -exe -upx # optional UPX compression of tp.exe
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

1. **Cleanup** — Removes prior outputs for the target(s) being built:
   - `-pyz`: `tp.pyz`, `build/zipapp/`
   - `-exe`: `tp.exe`, `build/pyinstaller/`, `build/thermo-embedded.ico`
   - no flags (both): all of the above
   - Preserves `build/thermo.ico` and the artifact not being rebuilt

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

Both resolve `tp.ini` and relative log paths from the **launcher directory** (`application_dir()` in `tp/config.py`). Ship `tp.ini` / `tp_log.csv` beside the executable or zipapp.

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

Ignored (regenerated each build): `build/pyinstaller/`, `build/zipapp/`, `build/thermo-embedded.ico`, `__pycache__/`, local `tp.ini`, `tp.log`, `tp_log.csv`.

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
| `-x` | Load config + log preload, print Rich snapshot to terminal, exit (no Textual UI, no BLE). Time windows follow `TimeDetail`, or `-more` for this run. |
| `-more` | Session-only More time detail (90M/4/8/12/24/36/72H) for `-x` or interactive mode; does not write `tp.ini` |
| `-nopoll` / `-np` | Interactive Textual UI with poll/retry worker disabled; **G** manual fetch still works; header shows “Polling off”; CSV log reloaded every 5 minutes for multi-instance viewing |
| `-f` / `-filter` *TEXT* | View filter: monitoring dashboard and `-x` output show only devices whose display name contains *TEXT* (case-insensitive substring). Polling, retries, and **G** still target all managed devices. |

Examples: `python tp.py -x -f cab`, `python tp.py -x -more`, `tp.exe -more`, `tp.exe -np -filter guest`, `python tp.py -debug`.

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

Place `tp.ini` and optional `tp_log.csv` in the same folder as `tp.exe` or `tp.pyz`.

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
  tp_log.csv          # optional CSV log
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
    poll.py
    log_export.py
    assets/
      log_export.html
    ui/
      app.py
      menus.py
      monitoring.py
      devices.py
      history_fetch_status.py
      device_status.py
      options.py
      log_export_action.py
      helpers.py
```

---

### Changelog

- **v1.9.0** — Options **W** / `TimeDetail=less|more`: Less (default 4H/24H/72H) or More (adds 8H/12H/36H/90M) for monitoring **T**, device status, and `-x` snapshot.
- **v1.8.0** — **History fetch** renamed from 72H fetch (**H**); manual fetch up to **1 year** in **7-day** BLE chunks; **BLE queue** status when waiting on poll; immediate modal loading and byte/chunk progress; scaled `day_history_timeout`; startup bootstrap still **72H** (`SPARKLINE_BOOTSTRAP_HISTORY_HOURS`).
- **v1.7.0** — **Log export to web** (main menu **5**, Options **E**): self-contained **`tp_export.html`** with device/timeframe controls and ECharts dual-axis chart; `log_export.py` + `assets/log_export.html`.
- **v1.6.0** — **Incremental minute-history polling** (default `PollMode=incremental`; Options **P** toggles live mode); **`read_recent_history`** for gap-filled minute CSV rows; **72H** BLE fetch/bootstrap (expanded from 24H); dashboard **T** sparkline window rotation (24H → 72H → 4H) with window-accurate min/max; default log **`tp_log.csv`**; **log rename** on filename change with overwrite prompt; **72h log preload**; `build.ps1` **`-pyz` / `-exe`** selective build; unit tests for poll mode, log rename, multi-row append.
- **v1.5.0** — **Fast live read** (datetime sync `0xA5` then `0xC2`, passive fallback); **fetch step arrows** (cyan/green/yellow); **BLE connect cache** + inter-device prefetch; **startup history bootstrap** when logging off; **Bluetooth radio auto-restart** on powered-off errors (`ble_radio.py`); unit tests for radio detection and bootstrap gating.
- **v1.4.0** — **24H BLE history fetch** (**H** progress modal; optional **Y/N** when adding a device); TP357S/TP359 stream protocol (`0xA5` datetime sync + `0xCCCC` history commands) with legacy TP357 `0xA7` fallback; partial-span merge preserves polled/log data outside the received window; `--history-day` CLI; unit tests for stream/legacy parsers and log merge.
- **v1.3.0** — Multi-column monitoring dashboard (**C**); `-np` alias for `-nopoll`; device label freshness colors (no `updated` on row); header fetch device name fix; `build.ps1` builds `tp.pyz` before `tp.exe` with per-artifact completion messages.
- **v1.2.0** — Indoor temp color bands; cur/min/max stat coloring; CLI `-x` snapshot; `-nopoll`; `-f`/`-filter` device view filter; header layout (status / title / clock); 4H/24H/72H status sparklines; per-device 60s read timeout; BLE connect optimizations.
- **v1.1.0** — Parallel fetch; minute retries; stale UI (green/yellow); log preload skip on startup; device status 1H/8H/24H sparklines; launcher-relative paths; build.ps1; removed BLE day-history/bootstrap; startup routing and navigation fixes.
- **v1.0.0** — Initial release: Textual TUI, bleak BLE, 5-row monitoring layout, CSV logging, device management.
