# TemPy — ThermoPro TP35x Monitor

**Version 1.9**

Cross-platform Python TUI (**TemPy**) for ThermoPro TP357/TP358/TP359 Bluetooth hygrometer/thermometer units.

**Author:** Kreft&Cursor

## Requirements

- Python 3.11+
- Bluetooth adapter (Windows 10+ or Linux with BlueZ)
- Dependencies: `pip install -r requirements.txt`

## Quick start

```bash
cd python/tp
pip install -r requirements.txt
python tp.py
```

On first run, `tp.ini` is created beside the launcher (`tp.py`, `tp.exe`, or `tp.pyz`).

**Startup:** If devices are configured, the app opens the monitoring dashboard. If none are configured, it opens Manage Devices and starts a discovery scan.

## Command-line options

| Flag | Description |
|------|-------------|
| `-debug` | Enable session `debug.log` in the configured log directory (Options **B** toggles during a session) |
| `-x` | Print one snapshot from saved log/history data and exit (no UI, no BLE). Windows follow Options `TimeDetail` (Less/More), or `-more` for this run. |
| `-more` | Session-only **More** time detail (90M/4H/8H/12H/24H/36H/72H) for `-x` or interactive mode — does not change `tp.ini` |
| `-nopoll` / `-np` | Interactive mode without automatic poll scheduling; **G** still fetches manually; reloads the CSV log every 5 minutes for multi-instance viewing |
| `-f` / `-filter` *TEXT* | View filter — only show devices whose name contains *TEXT* (case-insensitive). Works with interactive mode and `-x`. Polling and manual fetch still run for all managed devices; only the dashboard display is filtered. |
| `--history-day` *MAC* | Fetch up to 1 year of BLE history for *MAC* and exit (dev/test; no UI) |

Examples:

```bash
python tp.py -x                    # snapshot of all devices from log
python tp.py -x -more              # snapshot with 90M/4/8/12/24/36/72H windows (ini unchanged)
python tp.py -x -f cab             # snapshot: only names matching "cab" (e.g. Plant Cabinet)
python tp.py -more                 # interactive session with More windows (ini unchanged)
python tp.py -np -filter office
./tp.exe -debug -f guest
```

## Build (Windows)

From `python/tp/`:

```powershell
./build.ps1           # tp.pyz, then tp.exe
./build.ps1 -pyz      # tp.pyz only
./build.ps1 -exe      # tp.exe only
./build.ps1 -exe -upx # optional UPX compression of tp.exe
```

| Output | Usage |
|--------|--------|
| `tp.exe` | Standalone executable (PyInstaller) |
| `tp.pyz` | `python tp.pyz` — requires bleak/textual installed |

Place `tp.ini` and `tp_log.csv` in the same folder as the launcher. The build script uses `build/thermo.ico` for the executable icon.

## Navigation

| Key | Context | Action |
|-----|---------|--------|
| **Q** | Anywhere | Back one level (sub-screens and modals); exit from main menu |
| **M** | Sub-screens | Main menu |
| **G** | Monitoring | Fetch stale devices only; full poll if none are stale |
| **T** | Monitoring | Cycle dashboard sparkline window (Less: 24H → 72H → 4H; More: 24H → 36H → 72H → 90M → 4H → 8H → 12H) |
| **1**–**9**, **0** | Monitoring | Open device info for visible device 1–10 (**Q** closes back to monitoring) |
| **C** | Monitoring | Cycle column layout (shown only when the terminal is wide enough for 2+ columns) |

## Main menu

| Key | Action |
|-----|--------|
| 1 | Monitoring — live dashboard with sparklines (press **T** to cycle windows) |
| 2 | Manage Devices — discover, add, rename, remove sensors |
| 3 | Options — logging, poll mode, time detail, log file path, web export |
| 4 / q | Exit |
| 5 | Export log to web — writes `tp_export.html` and opens it in your browser |

## Monitoring

Each tracked device shows 5 rows:

1. Device name — **green** if fresh (within 10 minutes), **yellow** if stale. While that device is being fetched, `▶` / `◀` show the BLE step: **cyan** connecting, **green** fast live read (datetime sync), **yellow** passive fallback (legacy sensors)
2. Temperature cur / min / max (°F) — cur is the latest reading; min/max are the true lows and highs within the active sparkline window
3. 24-character temperature sparkline (default 24H window; **T** cycles windows from Options time detail — Less: 24H → 72H → 4H; More: 24H → 36H → 72H → 90M → 4H → 8H → 12H)
4. Humidity cur / min / max (%) — each value color-banded to match sparkline glyphs
5. 24-character humidity sparkline

**Updated time colors:** removed from the label row (see device status **I**). Stale rows still dim stats and sparklines.

**Header:** DEBUG / filter / next poll or “Polling off” / fetch progress (left); **🌡 TemPy** centered when room; clock (right).

**Footer:** `m` Menu, `g` Fetch now (hidden while a fetch is active), `1`–`x` info (first 10 visible devices; `x` is `0` when all ten slots are mapped), and `c` Columns when multi-column layout is available.

### Multi-column layout

When the terminal is wide enough to fit two or more device blocks side by side (content width plus 2-character padding on each side), **c Columns** appears in the footer. Press **C** to cycle `1 → 2 → … → max → 1`. Press **T** to cycle the sparkline time window (Less: `24H → 72H → 4H`; More: `24H → 36H → 72H → 90M → 4H → 8H → 12H`). Devices fill row-major (`1 2` / `3 4` / …). Default is a single column. Narrowing the window clamps the active column count automatically.

The view filter (`-f`) only affects which devices are shown; column layout applies to the filtered set.

### Polling and retries

- **Scheduled polls** run on 5-minute clock boundaries (`:00`, `:05`, `:10`, …). Devices are fetched one at a time per cycle. Use `-nopoll` / `-np` to disable BLE scheduling; the dashboard still reloads the CSV log every 5 minutes so a second viewer can follow a polling instance.
- **Incremental poll mode (default):** Each cycle requests missing **minute-aligned** history from the sensor since the last stored reading (up to 24 h per fetch). The CSV log gets one row per minute when logging is enabled. Falls back to a single live read if history fetch fails.
- **Live poll mode:** Options **P** switches to the legacy behavior — one live snapshot per device per cycle (wall-clock timestamp, may include seconds).
- **Startup:** Log preload runs first. If every device already has a fresh reading for the current 5-minute chunk, the initial BLE fetch is skipped.
- **Logging off:** On startup, TemPy automatically pulls 72H history from each sensor over BLE (when sparklines are still empty) so the dashboard fills in without a CSV log. Header shows **History** while this runs.
- **Minute retries:** Devices that miss a poll in the current chunk are retried every 60 seconds until the next boundary.
- Press **G** to fetch stale devices for the current chunk, or run a full poll when all devices are fresh.

## Manage Devices

| Key | Action |
|-----|--------|
| D | Discover nearby TP35x devices (10 s scan) |
| A | Add selected discovered device (name prompt, then optional history load) |
| I | Status — log preload, last fetch, multi-window sparklines (Less: 4H/24H/72H; More adds 8H/12H/36H/90M) |
| H | History fetch — pull up to 1 year of minute history over BLE; merges only the received timestamp span (preserves older polled/log data outside that range); log rows in the same span replaced only when logging is enabled |
| E | Rename selected managed device |
| R | Remove selected managed device |
| W | Move selected managed device up |
| S | Move selected managed device down |
| ↑/↓ | Change selection |
| **Q** | Back one level |
| **M** | Main menu |

## BLE history fetch

TemPy can pull minute-resolution history stored on the sensor over BLE (up to **1 year** per fetch, in **7-day** BLE chunks).

| How | Action |
|-----|--------|
| Manage Devices **H** | History fetch for the selected managed device (progress modal) |
| Add device **A** | After naming, **Y** loads history immediately; **N** or **Q** skips |
| CLI `--history-day` *MAC* | Headless fetch for testing (no UI) |

**Merge behavior:** Only timestamps covered by the received BLE data are replaced in memory (and in the CSV log when logging is enabled). Older polled or log data outside that span is kept — useful when the sensor has less history on board after a reboot.

**Protocols:** TP358/TP359 and TP357S use the stream protocol (datetime sync + history request). Original TP357 uses the legacy `0xA7` packet stream when the stream protocol returns no data.

## Options

| Key | Action |
|-----|--------|
| L | Toggle CSV logging |
| P | Toggle poll mode — incremental (minute history) or live (single snapshot) |
| W | Toggle time detail — Less (4H/24H/72H) or More (adds 8H/12H/36H/90M) for dashboard **T**, device status, and `-x` snapshot |
| E | Export log to web — interactive HTML chart (`tp_export.html`) |
| B | Toggle session debug log |
| D | Edit log directory |
| F | Edit log filename (renames existing log; prompts before overwriting an existing target file) |
| **Q** | Back one level |
| **M** | Main menu |

Default log path: `tp_log.csv` beside the launcher (when `LogDirectory=.`). Existing `tp.ini` files keep their configured filename.

Relative log directories (e.g. `.` or `logs`) resolve from the **launcher directory**, not the current working directory. The Options screen shows the resolved full path — verify it before enabling logging.

## Configuration (`tp.ini`)

Created beside the launcher on first run:

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

`PollMode` is `incremental` (default) or `live`.
`TimeDetail` is `less` (default: 4H/24H/72H) or `more` (adds 8H/12H/36H/90M).

## Log file (CSV)

```csv
timestamp,device,temp_f,humidity_pct,mac
2026-06-15 14:03:00,Living Room,72.1,48,AA:BB:CC:DD:EE:FF
2026-06-15 14:04:00,Living Room,72.3,48,AA:BB:CC:DD:EE:FF
2026-06-15 14:05:00,Living Room,72.4,48,AA:BB:CC:DD:EE:FF
```

In incremental poll mode, rows are minute-aligned (`:00` seconds). Multiple rows per device may be appended after each 5-minute cycle. On startup, readings from the last 72 hours are preloaded into memory for sparklines and freshness checks.

## Log export to web

Export the CSV log to a self-contained **`tp_export.html`** beside the launcher:

| How | Action |
|-----|--------|
| Main menu **5** | Export log to web |
| Options **E** | Same export |

The report opens in your default browser. Pick a **device** and **timeframe** (4H, 24H, 72H, 7D, All) to filter an interactive dual-axis chart (temperature and humidity). Data is embedded in the file — no server required after export. Consecutive bogus 32 °F / 10% readings are excluded (same as the live app).

## Platform setup

### Windows

- Enable Bluetooth in Settings
- Run from a terminal that supports Unicode block characters (Windows Terminal recommended)
- bleak uses the WinRT Bluetooth backend

### Linux

Install BlueZ and ensure your user can access Bluetooth:

```bash
sudo apt-get install bluetooth bluez bluez-tools
sudo usermod -aG bluetooth $USER
```

Log out and back in after adding the `bluetooth` group. See also [childs.be TP350S guide](https://www.childs.be/articles/post/how-to-log-temperature-and-humidity-from-a-thermopro-tp350s-in-linux).

## Color bands

Temperature and humidity sparklines and cur/min/max values use indoor comfort bands:

- **Temp °F:** Cyan &lt;55, Green 55–64, White 65–71, Yellow 72–77, Red 78–81, Magenta ≥82
- **Humidity %:** Cyan &lt;30, White 30–60, Yellow 61–70, Red &gt;70

## Notes

- Sensors must be powered and in range; GATT live reads do not require phone-app pairing
- Consecutive **32.0 °F / 10%** readings (a known sensor error pattern) are discarded and not logged when two or more arrive in a row
- Sparkline height reflects trend within the window; glyph color reflects the band at each bin average
- Distant sensors may miss scheduled polls; minute retries and manual **G** fetch help recover them
- If Bluetooth is off at the OS level, TemPy prompts **Y/N** before enabling it (e.g. after sleep). Stale bleak errors after you already turned Bluetooth back on do not re-prompt; a whole-fleet poll failure still auto-restarts the adapter without asking
- Last-updated time is on the device status screen (**I**), not on the dashboard label row
- Technical reference for AI assistants: [cursor.md](cursor.md)

## Credits

TemPy’s BLE protocol work builds on [pasky/tp357](https://github.com/pasky/tp357) (`tp357tool.py`), which was a starting point for this project.

## Changelog

- **v1.9** — Options **W** time-detail toggle: **Less** (4H/24H/72H, default) or **More** (adds 8H/12H/36H/90M) for monitoring **T**, device status, and `-x` snapshot.
- **v1.8** — **History fetch** renamed (**H** → History Fetch); manual fetch up to **1 year** in 7-day BLE chunks; BLE queue notice when poll holds the radio; improved fetch progress UI.
- **v1.7.0** — **Log export to web** (main menu **5**, Options **E**): self-contained `tp_export.html` with device/timeframe controls and ECharts dual-axis chart.
- **v1.6.0** — Incremental minute-history polling (default); Options **P** poll-mode toggle; 72H BLE fetch/bootstrap; dashboard **T** sparkline window rotation; default log `tp_log.csv`; log rename on filename change with overwrite prompt; 72h log preload.
- **v1.5.0** — Faster live reads; colored fetch-step arrows; automatic history bootstrap on startup when logging is off; Bluetooth auto-recovery when the radio is off; quicker reconnects between polls.
- **v1.4.0** — 24H BLE history fetch (**H**, optional on add); TP357S/TP359 stream protocol; partial-span merge; `--history-day` CLI.
- **v1.3.0** — Multi-column dashboard (**C**); `-np` alias; device freshness label colors; build script improvements.
- **v1.2.0** — Indoor color bands; `-x` snapshot; `-nopoll`; `-f` filter; 4H/24H/72H status sparklines.
- **v1.1.0** — Minute retries; stale UI; log preload; launcher-relative paths; `build.ps1`.
- **v1.0.0** — Initial Textual TUI release.
