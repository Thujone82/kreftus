# Trimps Spire Assault Planner & Combat Simulator

A high-performance, self-contained single-page application ([`trimps/sa.html`](../sa.html)) for optimizing **Trimps Spire Assault (AutoBattle)** progression, evaluating community builds, and executing real-time combat simulations.

Included alongside the planner is our custom, installable Progressive Web App (PWA) fork of the full Trimps game ([`trimps/index.html`](../index.html)).

---

## 🎮 Custom PWA Fork of Trimps (`trimps/index.html`)

Located at [`trimps/index.html`](../index.html), this fork provides a standalone desktop and mobile experience for playing Trimps 5.10.1:

- **Installable Progressive Web App (PWA)**: Built with [`manifest.json`](../manifest.json) for native standalone window launching with desktop and home screen icons.
- **Offline Capable**: Runs completely locally without requiring continuous internet access.
- **Custom Local Styling**: Features custom styling and responsive UI enhancements via [`css/kreftus-local.css`](../css/kreftus-local.css).
- **Upstream Synchronization**: Maintained via [`updtrimps.ps1`](../updtrimps.ps1), which updates the base game from upstream `Trimps.github.io` while safeguarding local customizations and planner files.

---

## ✨ Features & User Guide

### 1. In-Browser Combat Simulator & Sandbox Modal
The Spire Assault Simulator accurately reproduces the official Trimps AutoBattle combat engine ([`AutoBattle.js`](../AutoBattle.js)) inside [`sim-engine.js`](./sim-engine.js).

- **Full Affix & Mechanic Simulation**: Faithfully models all enemy properties (Berserking, Poisoning, Shocking, Bleed, Lifestealing, Fast x2/x3, Defensive, Slowing Aura, Enrage, etc.) and Huffy stats (Attack, Defense, Lifesteal, Dust Mult, Resists, Limbs, The Ring).
- **Speed Controls**: Supports real-time playback at `1x`, `2x`, `5x`, `10x`, `50x`, plus frame-by-frame `Step`, `Pause/Resume`, and `Reset`.
- **Batch 1K Execution**: Instantly runs 1,000 fights in headless mode to calculate high-precision win rates, average kill times (TTK), and dust income rates.
- **Testing Sandbox Loadout**:
  - Displays active equipped items with quick `−` and `+` level adjustments.
  - Ring level stepper and dynamic mod toggles with live slot enforcement.
  - **Reset to Save**: Instantly reverts sandbox gear back to your live save loadout.
  - **Lock to Plan**: Locks custom sandbox builds directly into the Next Upgrade Advisor roadmap. Prompts for a custom build name or offers a smart default.
- **Maximized Vertical Space**: The simulator dialog expands to `min(960px, calc(100vh - 36px))`, using up to 96% of available screen height to display tables, sandboxes, and arena combat without unnecessary vertical scrolling on 1080p displays.

---

### 2. Auto-Calibrated Dust Rates (9,001 Simulated Battles)
- **Zero-Input Accuracy**: When you import a Trimps save string or change enemy levels, the planner automatically executes an accelerated **9,001-fight headless Monte Carlo simulation** using your live gear and enemy level.
- **Exact Game Parity**: Replaces generic sheet baselines with the exact dust/second earned in real gameplay (e.g. `7.22B/s`).
- **Streamlined Custom Dust Override**:
  - The dust pill displays `Sim: X/s (SA Y)` alongside a subtle gear icon (`⚙`).
  - Clicking `⚙` exposes an inline numerical input for manual dust rate overrides.
  - When overridden, a red cancel pill (`✕`) appears. Clicking `✕`, pasting a new save, or switching SA levels instantly clears the override and restores the simulated rate.

---

### 3. Monte Carlo Push Test
Located in Tab 2 of the Simulator modal:

- **Push Viability Heatmap**: Tests your current sandbox build across higher Spire Assault levels to identify push boundaries.
- **Accelerated Scan**: Runs a fast 10-fight smoke test to detect the maximum winning level, then conducts a deep 500-fight per level scan across a 10-level window.
- **Metrics**: Visual win rate bars, win/loss breakdown, average fight duration, and estimated clear times (`~1.6d` or `Unclearable`).
- **One-Click Level Set**: Identifies the **Highest Winning Level** (≥1 win) and **Max Stable Level** (≥50% win rate) with a button to immediately set the simulator to that level.
- **Smart Result Retention**: Heatmap results are cached by build loadout signature (`getSimBuildSignature`). Switching between simulator tabs or changing the SA level dropdown re-renders cached results instantly with zero delay. Re-scans only occur when sandbox gear is modified or when clicking **🎲 Re-scan Push Heatmap**.

---

### 4. Ring Mod Optimizer
Located in Tab 3 of the Simulator modal:

- **Exhaustive Permutation Search**: Simulates every valid combination of Ring mods (Attack, Dust, Health, Defense, Lifesteal) for your current Ring level.
- **Relative Delta Comparison**: Accurately benchmarks combinations against your active build, displaying `Δ VS CURRENT` (e.g. `= Current`, `+7.41B/s`, etc.), simulated dust/s, win rate, and TTK. The active `Current` pill dynamically highlights green if optimal, or red if superior mod configurations exist.
- **Apply to Sandbox**: Instantly equips selected ring mods into the sandbox loadout with one click.

---

### 5. Upgrade Value (Upgrade Sensitivity & Payback ROI)
Located in Tab 4 of the Simulator modal:

- **Immediate Progression ROI**: Tests the impact of upgrading each equipped item by +1 level or upgrading The Ring by +1 level.
- **Accurate Payback Metric**: Calculates the exact payback duration (`Upgrade Cost / Simulated Δ Dust Gain`), showing how quickly the upgrade pays for itself.
- **Ranked Advice**: Automatically runs upon opening the tab or changing SA levels and sorts upgrades by highest efficiency.
- **Apply to Sandbox**: Directly applies the upgrade to the sandbox loadout to preview further progression.

---

### 6. Next Upgrade Advisor & Milestone Tracker
- **Smart Build Progression**: Matches your current loadout against over 3,500 community builds across 157 levels from Google Sheets.
- **Step-by-Step Guidance**: Details required upgrades, dust/shard costs, and time-to-acquire based on your live dust bank and income.
- **Plan Locking**: Lock your active plan to prevent mid-progression recommendation jumping.
- **Milestones**: Tracks one-timer milestones (7–53 items) and Ring slot unlocks.
- **Light & Dark Theming**: Built-in dark (default) and light themes with persistent preference storage.

---

## 📁 Architecture & File Layout

| File | Purpose |
|---|---|
| [`shell.html`](./shell.html) | HTML shell, CSS custom properties, responsive layout, modal templates, and UI components. |
| [`app.js`](./app.js) | Client application logic: save decompression, live bank accumulation, community build matcher, simulator controllers, sandbox state, and UI binding. |
| [`sim-engine.js`](./sim-engine.js) | Full-featured Trimps AutoBattle simulator engine: fighter modeling, buff/debuff management, tick loops, 9,001-fight calibration, push heatmap scanner, ring optimizer, and upgrade sensitivity. |
| [`data.json`](./data.json) | Compiled dataset (~1.3 MB) containing 3,500+ community builds, item definitions, dust tiers, and level parameters. |
| [`build_data.py`](./build_data.py) | Python extractor that pulls builds from the community Google Sheets workbook (`_sa.xlsx`) and item stats from `objects.js`. |
| [`assemble.py`](./assemble.py) | Python script that bundles `shell.html`, `lz-string.js`, `sim-engine.js`, `data.json`, and `app.js` into the self-contained `trimps/sa.html` with CRLF line endings. |
| [`refresh.ps1`](./refresh.ps1) | PowerShell build pipeline coordinating syntax validation (`node --check`), dataset compilation, and bundle assembly. |
| [`README.html`](./README.html) | Interactive HTML documentation viewer with live markdown parsing, theme toggle (Dark/Light), and offline support. |

---

## 🚀 Build Pipeline (`refresh.ps1`)

Run from PowerShell inside `trimps/sa-build/`:

```powershell
# Fast build: Validates JavaScript syntax and compiles trimps/sa.html (skips workbook download)
.\refresh.ps1 -AssembleOnly

# Standard build: Re-parses local cached _sa.xlsx and compiles sa.html
.\refresh.ps1

# Fresh build: Downloads the latest community Google Sheets workbook, re-parses data, and compiles sa.html
.\refresh.ps1 -Fresh

# Fresh build with cache cleanup: Downloads latest workbook, builds, and removes temporary _sa.xlsx
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

---

## 🛡️ Upstream Protection (`updtrimps.ps1`)

`updtrimps.ps1` synchronizes local `trimps/` files with the upstream `Trimps.github.io` repository while preserving locally owned components. Both `sa.html` and the entire `sa-build/` directory are explicitly excluded from upstream overwrites.
