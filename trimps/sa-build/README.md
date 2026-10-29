# Trimps Spire Assault Planner Build Pipeline

This directory contains the authoring source files and automation pipeline for compiling the self-contained Spire Assault planner ([`trimps/sa.html`](../sa.html)).

## 📁 Architecture & File Layout

- **`shell.html`**: HTML shell, CSS custom properties, and UI component layout.
- **`app.js`**: Client-side application logic:
  - Save string decoding & decompression (`decodeSave`)
  - Live dust and shard bank accumulation timers
  - Community sheet build matching & ROI optimization algorithms
  - Step progression roadmap, atomic cost calculations, and custom drag-and-drop reordering
  - In-game preset inspection (`ab.presets`) and build match indicators
  - One-timer milestone progression tracking (7–53 items)
  - The Ring inherent stat calculations & customizable slot milestone forecasting
  - Combat session diagnostics and AFK stability metrics
  - In-game safety audits (Practice mode, Load Ring setting)
- **`build_data.py`**: Python script that extracts:
  - Community SA builds from Google Sheets workbook (`_sa.xlsx`)
  - SA item catalog, pricing tiers, and dust types from Trimps `objects.js`
  - Output: `data.json` (~1.3 MB dataset with 3,500+ builds across 157 levels)
- **`assemble.py`**: Python script that inlines `shell.html`, `lz-string.js`, `data.json`, and `app.js` into the final standalone `trimps/sa.html` with strict CRLF line endings.
- **`refresh.ps1`**: PowerShell automation script that coordinates workbook extraction, syntax validation via `node --check`, and bundle assembly.

## 🚀 Build Pipeline (`refresh.ps1`)

Run from PowerShell:

```powershell
# Fast build (skips workbook download/re-parse, validates app.js, and compiles sa.html)
.\refresh.ps1 -AssembleOnly

# Full build using cached _sa.xlsx
.\refresh.ps1

# Fresh build downloading the latest community workbook from Google Sheets
.\refresh.ps1 -Fresh

# Fresh build and clean temporary _sa.xlsx cache after completion
.\refresh.ps1 -Fresh -CleanCache
```

### Script Parameters

| Parameter | Type | Description |
|---|---|---|
| `-AssembleOnly` | Switch | Skips `build_data.py` and rapidly validates & compiles `sa.html` from existing `data.json`. |
| `-Fresh` | Switch | Forces a fresh download of the community Google Sheets workbook before rebuilding. |
| `-CleanCache` | Switch | Removes the cached `_sa.xlsx` file after a successful build. |
| `-VerboseSheets` | Switch | Streams detailed per-level build counts during extraction. |
| `-Help` | Switch | Displays full parameter documentation and usage examples. |

## 🛡️ Protection from `updtrimps.ps1`

`updtrimps.ps1` synchronizes local `trimps/` files with the upstream `Trimps.github.io` repository while preserving locally owned components. Both `sa.html` and the entire `sa-build/` directory are explicitly excluded from upstream overwrites.
