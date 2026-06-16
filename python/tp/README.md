# TemPy — ThermoPro TP35x Monitor

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

## Build (Windows)

From `python/tp/`:

```powershell
./build.ps1        # tp.exe + tp.pyz
./build.ps1 -upx   # optional UPX compression of tp.exe
```

| Output | Usage |
|--------|--------|
| `tp.exe` | Standalone executable (PyInstaller) |
| `tp.pyz` | `python tp.pyz` — requires bleak/textual installed |

Place `tp.ini` and `tp.log` in the same folder as the launcher. The build script uses `build/thermo.ico` for the executable icon.

## Navigation

| Key | Context | Action |
|-----|---------|--------|
| **Q** | Anywhere | Back one level; exit from main menu |
| **M** / Esc | Sub-screens | Main menu |
| **G** | Monitoring | Fetch all devices now |

## Main menu

| Key | Action |
|-----|--------|
| 1 | Monitoring — live dashboard with 24h sparklines |
| 2 | Manage Devices — discover, add, rename, remove sensors |
| 3 | Options — logging toggle and log file path |
| 4 / q | Exit |

## Monitoring

Each tracked device shows 5 rows:

1. Device name (left) and `updated HH:MM` (right-aligned)
2. Temperature cur / min / max (°F)
3. 24-character temperature sparkline (1 hour per glyph)
4. Humidity cur / min / max (%)
5. 24-character humidity sparkline

**Updated time colors:** green = fresh (within 5 minutes), yellow = stale. Stale rows also dim stats and sparklines.

**Header:** DEBUG / next poll / fetch progress (left); **🌡 TemPy** centered when room; clock (right).

**Footer:** Keybinding shortcuts only.

### Polling and retries

- **Scheduled polls** run on 5-minute clock boundaries (`:00`, `:05`, `:10`, …). Devices are fetched one at a time per cycle.
- **Startup:** Log preload runs first. If every device already has a fresh reading for the current 5-minute chunk, the initial BLE fetch is skipped.
- **Minute retries:** Devices that miss a poll in the current chunk are retried every 60 seconds until the next boundary.
- Press **G** to force an immediate full fetch.

## Manage Devices

| Key | Action |
|-----|--------|
| D | Discover nearby TP35x devices (10 s scan) |
| A | Add selected discovered device (name prompt starts empty) |
| I | Status — log preload, last fetch, 4H/24H/72H sparklines |
| E | Rename selected managed device |
| R | Remove selected managed device |
| W | Move selected managed device up |
| S | Move selected managed device down |
| ↑/↓ | Change selection |
| M / Esc | Main menu |

## Options

| Key | Action |
|-----|--------|
| L | Toggle CSV logging |
| D | Edit log directory |
| F | Edit log filename |
| M / Esc | Main menu |

Default log path: `tp.log` beside the launcher (when `LogDirectory=.`).

Relative log directories (e.g. `.` or `logs`) resolve from the **launcher directory**, not the current working directory. The Options screen shows the resolved full path — verify it before enabling logging.

## Configuration (`tp.ini`)

Created beside the launcher on first run:

```ini
[Settings]
LoggingEnabled=false
LogDirectory=.
LogFileName=tp.log

[Devices]
AA:BB:CC:DD:EE:FF=Living Room
```

## Log file (CSV)

```csv
timestamp,device,temp_f,humidity_pct,mac
2026-06-15 14:05:00,Living Room,72.4,48,AA:BB:CC:DD:EE:FF
```

Rows are appended after each fetch cycle when logging is enabled. On startup, readings from the last 24 hours are preloaded into memory for sparklines and freshness checks.

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

Temperature and humidity sparklines and current values use the same comfort bands as [ps/gf](../../ps/gf/README.md):

- **Temp °F:** Blue &lt;33, White 33–89, Red &gt;89
- **Humidity %:** Cyan &lt;30, White 30–60, Yellow 61–70, Red &gt;70

## Notes

- Sensors must be powered and in range; GATT live reads do not require phone-app pairing
- Sparkline height reflects trend within the window; glyph color reflects the band at each bin average
- Distant sensors may miss scheduled polls; minute retries and manual **G** fetch help recover them
- Technical reference for AI assistants: [cursor.md](cursor.md)
