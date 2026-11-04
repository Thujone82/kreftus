# Trimps Spire Assault Planner & Combat Simulator

A high-performance, self-contained single-page application ([`trimps/sa.html`](../sa.html)) for optimizing **Trimps Spire Assault (AutoBattle)** progression, evaluating community builds, and executing real-time combat simulations.

Included alongside the planner is our custom, installable Progressive Web App (PWA) fork of the full Trimps game ([`trimps/index.html`](../index.html)).

---

## ✨ Features & User Guide

### 1. Next Upgrade Advisor & Milestone Tracker
- **Smart Build Progression**: Matches your current loadout against over 3,500 community builds across 157 levels sourced from the player-curated [Community Spire Assault Spreadsheet](https://docs.google.com/spreadsheets/d/17Z3dwnkeAmY2La-LWreybTs4Sm7BAck79EzVF_gkZzs).
- **Step-by-Step Guidance**: Details required upgrades, dust/shard costs, and time-to-acquire based on your live dust bank and income.
- **Plan Locking**: Lock your active plan to prevent mid-progression recommendation jumping.
- **Milestones**: Tracks one-timer milestones (7–53 items) and Ring slot unlocks.
- **Light & Dark Theming**: Built-in dark (default) and light themes with persistent preference storage.

---

### 2. Intelligent Upgrade Step Order & Recommendation Engine
Rather than presenting a monolithic, unordered shopping list, the Next Upgrade Advisor features an intelligent recommendation engine (`sortSmartSteps`) that decomposes target builds into an optimized, step-by-step roadmap:

- **Strict Dependency Resolution**: Enforces mandatory game progression prerequisites automatically:
  - Contracts are always scheduled before level upgrades for that item.
  - The Ring unlock milestone is scheduled before Ring tier upgrades.
  - Extra Limbs purchases are prioritized before unequipped items that depend on that slot to be worn.
  - Multi-level upgrades on the same item are ordered sequentially (e.g. Level 12 → 13 → 14).
- **Active-Build Prioritization**: Detects whether an item in the target plan is already equipped in your current active farming setup. Upgrades for currently equipped gear are scheduled first, providing immediate stat and dust earning improvements *while* you work toward the full build, rather than stockpiling dust for inactive items.
- **Dynamic Cost & Wait-Time Minimization**: Evaluates your live dust bank, blood shard reserves, and real-time generation rate (`activeDustPs`) to calculate the exact effective time-to-acquire (`effWait`) for each step. Cheaper, faster upgrades that accelerate progress are scheduled ahead of long accumulation bottlenecks.
- **Atomic Single-Step Guidance**: The top banner highlights the immediate next action with clear live state indicators (`Ready now` green badge or dynamic countdown timer pill), showing exact dust/shard requirements and remaining time.
- **Custom Drag-and-Drop Reordering**: Allows full manual customization of step sequences directly in the step checklist. Custom order preferences are preserved per-plan in `localStorage` (`saPlanner.stepOrder`), with a 1-click **Reset Order** button to restore optimal algorithm ordering.
- **Plan Locking & Completion Tracking**: Locking a target build (`Lock Plan`) pins your active progression path so recommendations do not fluctuate mid-progression. Tracks total completed steps (e.g. `Step 4 of 9 — 5 remaining`) and seamlessly transitions to `Locked (Done)` with gear application reminders once ready.

---

### 3. High-Precision Sequential Time Estimation (Compounding Step Simulation & Bank Decrementing)
Standard planners calculate roadmap duration assuming a static earning rate throughout the entire plan (Total Cost / Current Rate). In reality, as you purchase intermediary upgrades—such as leveling currently equipped items, advancing The Ring tier, or activating dust-boosting contracts—your actual earning velocity increases *while* you work through the plan.

The Next Upgrade Advisor models this dynamic reality with a high-precision sequential simulation and time-decrementing engine:

- **Sequential Step-by-Step Simulation (`calculateStepWaitTimes`)**:
  - Rather than treating the roadmap as a static sum, the engine evaluates each step sequentially.
  - For any step that improves combat stats in your active farming loadout (equipped gear upgrades, Ring levels, Ring unlocks, and contracts), the planner simulates **9,001 headless Monte Carlo fights** using your live save state.
  - The newly simulated dust/second rate (`_stepRate`) is assigned to that step, and that compounded velocity is used to calculate the accumulation duration for the *next* step.
- **Parallel Currency Modeling (Dust vs. Blood Shards)**:
  - Tracks dust and blood shard generation independently.
  - Accumulation times are calculated in parallel as $\max(\text{dustWait}, \text{shardWait})$, accurately identifying whether dust reserves or blood shard production represents the critical path bottleneck.
- **CPU-Optimized Invariant Caching (`planRatesCache` & `stepSimCache`)**:
  - A 9,001-fight combat simulation determines the physical earning capacity after an upgrade, which is invariant over time for a given save configuration.
  - Results are cached persistently by step signature (`planRatesCache`). Once a roadmap has been simulated, it executes in **0.0 ms** on subsequent evaluations with **zero background simulation runs**.
- **Pure Arithmetic Time Decrementing**:
  - Second-by-second live updates (`updateLiveBank`) run zero combat simulations.
  - As time elapses and your live bank accumulates dust and shards, remaining wait times are decremented purely by arithmetic ($\text{wait} = (\text{cost} - \text{current\_bank}) / \text{step\_rate}$), requiring negligible CPU.
- **Strict Invalidation Rules**:
  - Simulations re-run **only** when a new save is pasted/loaded, or when the user manually changes the step order via drag-and-drop or order reset.
- **Synchronized Across All Views**:
  - Total build times are fully unified across the **Target Plan** header card (`Build Time`), the **Next Upgrade Advisor** atomic step pill (`Build:`), the **Roadmap Popover** (`Est. wait`), and the **Better Builds** table (`Earn / Return` column).
  - Tooltips display both compact decimal days (e.g. `7.8d`) and precise day/hour breakdowns (e.g. `7d 19h`) with explicit `Sim 9001` badges.

---

### 4. Community Build Sourcing & Database
The advisor's progression roadmap and runnable builds are sourced and synchronized directly from the player-maintained [Community Spire Assault Spreadsheet](https://docs.google.com/spreadsheets/d/17Z3dwnkeAmY2La-LWreybTs4Sm7BAck79EzVF_gkZzs):

- **Comprehensive Level Coverage**: Catalogs 3,500+ battle-tested configurations submitted and vetted by the Trimps community across all 157 Spire Assault zones.
- **Farming vs. Pushing Archetypes**: Categorizes builds into **Farming** setups (optimized for maximum steady dust/second yield) and **Pushing** setups (fine-tuned with defensive mitigation, lifesteal, and burst damage to overcome progression roadblocks).
- **Full Gear Specifications**: Evaluates exact item tiers, required equipment upgrade levels, Extra Limbs slots, Ring level thresholds, active Ring mod allocations (Attack, Dust, Health, Defense, Lifesteal), and contract requirements.
- **Automated Pipeline Extraction**: Sourced directly via [`build_data.py`](./build_data.py), which downloads the community Google Sheets workbook (`_sa.xlsx`), normalizes item IDs to match Trimps source definitions in [`objects.js`](../objects.js), calculates dust costs, and outputs the optimized [`data.json`](./data.json) catalog. Run `\refresh.ps1 -Fresh` at any time to sync with the latest community additions.
- **Live Combat Calibration**: Replaces static sheet estimates with your true in-game performance by running builds through [`sim-engine.js`](./sim-engine.js) using your real save stats.

---

### 5. In-Browser Combat Simulator & Sandbox Modal
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

### 6. Auto-Calibrated Dust Rates (9,001 Simulated Battles)
- **Zero-Input Accuracy**: When you import a Trimps save string or change enemy levels, the planner automatically executes an accelerated **9,001-fight headless Monte Carlo simulation** using your live gear and enemy level.
- **Exact Game Parity**: Replaces generic sheet baselines with the exact dust/second earned in real gameplay (e.g. `7.22B/s`).
- **Streamlined Custom Dust Override**:
  - The dust pill displays `Sim: X/s (SA Y)` alongside a subtle gear icon (`⚙`).
  - Clicking `⚙` exposes an inline numerical input for manual dust rate overrides.
  - When overridden, a red cancel pill (`✕`) appears. Clicking `✕`, pasting a new save, or switching SA levels instantly clears the override and restores the simulated rate.

---

### 7. Monte Carlo Push Test
Located in Tab 2 of the Simulator modal:

- **Push Viability Heatmap**: Tests your current sandbox build across higher Spire Assault levels to identify push boundaries.
- **Accelerated Scan**: Runs a fast 10-fight smoke test to detect the maximum winning level, then conducts a deep 500-fight per level scan across a 10-level window.
- **Metrics**: Visual win rate bars, win/loss breakdown, average fight duration, and estimated clear times (`~1.6d` or `Unclearable`).
- **One-Click Level Set**: Identifies the **Highest Winning Level** (≥1 win) and **Max Stable Level** (≥50% win rate) with a button to immediately set the simulator to that level.
- **Smart Result Retention**: Heatmap results are cached by build loadout signature (`getSimBuildSignature`). Switching between simulator tabs or changing the SA level dropdown re-renders cached results instantly with zero delay. Re-scans only occur when sandbox gear is modified or when clicking **🎲 Re-scan Push Heatmap**.

---

### 8. Ring Mod Optimizer
Located in Tab 3 of the Simulator modal:

- **Exhaustive Permutation Search**: Simulates every valid combination of Ring mods (Attack, Dust, Health, Defense, Lifesteal) for your current Ring level.
- **Relative Delta Comparison**: Accurately benchmarks combinations against your active build, displaying `Δ VS CURRENT` (e.g. `= Current`, `+7.41B/s`, etc.), simulated dust/s, win rate, and TTK. The active `Current` pill dynamically highlights green if optimal, or red if superior mod configurations exist.
- **Apply to Sandbox**: Instantly equips selected ring mods into the sandbox loadout with one click.

---

### 9. Upgrade Value (Upgrade Sensitivity & Payback ROI)
Located in Tab 4 of the Simulator modal:

- **Immediate Progression ROI**: Tests the impact of upgrading each equipped item by +1 level or upgrading The Ring by +1 level.
- **Accurate Payback Metric**: Calculates the exact payback duration (`Upgrade Cost / Simulated Δ Dust Gain`), showing how quickly the upgrade pays for itself.
- **Ranked Advice**: Automatically runs upon opening the tab or changing SA levels and sorts upgrades by highest efficiency.
- **Apply to Sandbox**: Directly applies the upgrade to the sandbox loadout to preview further progression.

---

## 🎮 Custom PWA Fork of Trimps (`trimps/index.html`)

Located at [`trimps/index.html`](../index.html), this fork provides a modernized desktop and mobile experience for playing Trimps 5.10.1:

- **SA Planner Dark Re-Theme ([`css/kreftus-local.css`](../css/kreftus-local.css) & [`js/kreftus-local.js`](../js/kreftus-local.js))**:
  - Overhauls the entire Trimps UI to match the dark design system of [`trimps/sa.html`](../sa.html): `#08090c` ambient canvas, elevated `#14171f` surface cards, semantic resource accent borders, segmented pill tabs (with dark green/red/purple status tints on the `SA` tab), emerald-tinted affordable purchase cards, royal-blue `.efficientYes` best-equipment highlights, dark tooltips/modals, and an expanded Spire Assault popup.
  - **Auto-Scaling Resource Readouts**: Combines CSS container query units (`cqw`) with dynamic JavaScript width fitting so long resource counts and `+X/sec` rates never overflow their cards.
- **Responsive Mobile & Narrow-Screen Reflow (`<= 1100px`)**:
  - Unlocks vertical scrolling, enlarges typography and touch targets for mobile legibility, stacks multi-column panels cleanly, and gives the 10×10 World/Map combat grid full screen width.
  - Injects a sticky **Mobile Quick-Switch Bar** (`📱 All`, `⚔️ Combat`, `🏗️ Buy`, `📦 Resources`, `📜 Log`) for 1-tap section filtering on phones and tablets.
- **Settings Drawer Theme Toggle (`SA Theme: On / Off`)**:
  - Adds a persistent **`SA Theme: On` / `SA Theme: Off`** toggle button directly to the right of the Settings search box (`#searchSettings`).
  - Defaults to **On** and remembers your preference in `localStorage` (`kreftus.trimps.themeEnabled`). Turning it **Off** completely disables `kreftus-local.css` and restores 100% stock Trimps styling while leaving the toggle accessible.
- **Installable Progressive Web App (PWA) & Offline Support**: Built with [`manifest.json`](../manifest.json) and [`sw.js`](../sw.js) for native standalone window launching and offline play.
- **Upstream Synchronization**: Maintained via [`updtrimps.ps1`](../../updtrimps.ps1), which updates the base game from upstream `Trimps.github.io` while safeguarding all local overlays and planner files.

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
