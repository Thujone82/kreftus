# Trimps Surky Perk Optimizer (`trimps/surky.html`)

A self-contained, offline-ready single-page application ([`trimps/surky.html`](../surky.html)) for optimizing **Trimps Universe 2 (Radon) perks**, evaluating preset weight profiles, and calculating Golden Upgrade, Runetrinket, and zone-push recommendations.

Built on top of **Surky** by **[nsheetz](https://github.com/nsheetz/perks)** ([upstream live calculator](https://nsheetz.github.io/perks/), original UI adapted from **justme**), wrapped in a non-destructive build pipeline (`trimps/surky-build/`) that preserves upstream perk math while adding a **Trimps 5.10.1 compatibility layer** (covering v5.8.0–5.10.1 mechanics omitted upstream), a modern dark/light theme overhaul, one-click clipboard import/export with save validation, live status pills, and keyboard shortcuts.

---

## ✨ Features & User Guide

### 1. Universe 2 Radon Perk Optimization Engine
- **Marginal Efficiency Ranking**: Evaluates every unlocked Universe 2 perk (`Resilience`, `Masterfulness`, `Criticality`, `Prismal`, `Tenacity`, `Greed`, `Frenzy`, `Meta-Leaching`, `Championism`, `Observation`, `Carpentry`, `Expansion`, `Smithology`, `Hunger`, `Artisanistry`, `Equality`, `Motivation`, `Looting`, `Power`, `Toughness`, `Range`, `Agility`, `Bait`, `Trumps`, `Pheromones`) and highlights rows by relative marginal efficiency ("the greener, the better").
- **Best Next Perk Indicator**: Automatically surfaces the single highest-efficiency next perk in the top header bar (`Best: <Perk>`) and highlights its row in the perk list.
- **Autobuy Modes**:
  - **Autobuy (respec available)** (`A` / `Shift+A`): Clears unlocked, non-level-locked perks and purchases the optimal Radon allocation from scratch (`Shift+A` also copies the exported perk string directly to your clipboard).
  - **Autobuy (no respec)** (`B`): Retains your current perk levels and spends only remaining unallocated Radon on optimal upgrades.
- **Granular Level Locks**: Toggle **Show level locks** to pin individual perks at fixed minimum levels while optimizing the rest of your Radon budget.
- **Strategy Callouts**: Automatically computes Golden Upgrade buying order (`Suggested GU strategy`), Runetrinket farming projections, and next trinket-cache zone push value.

---

### 2. Built-In Challenge & Playstyle Presets
Select from 14 curated preset profiles in the right-hand panel:

- **Basic Presets (with 1-Key Macro Hotkeys `1` / `2` / `3`)**:
  - **`[1]` Easy Radon Challenge**: Pure Radon/trinket farming when completing your Radon challenge requires minimal combat push. Pressing `1` selects this preset, pastes your save, runs Autobuy, and copies the perk string to your clipboard in one keypress.
  - **`[2]` Difficult Radon Challenge**: Balanced Attack, Health/Equality, and Radon weights for standard progression runs. Pressing `2` runs the full Paste + Autobuy + Copy workflow with this preset.
  - **`[3]` Push / C^3 / Mayhem**: Maximizes combat and survivability stats with zero weight on Radon growth. Pressing `3` runs the full Paste + Autobuy + Copy workflow with this preset.
- **Special Challenge Presets**:
  - **Alchemy**: Optimizes for Runetrinket drop rate alongside Finding Potions.
  - **Trappa** & **Max Carpentry**: Tailored for Trappapalooza trapping hours or pre-portal Carpentry maxing.
  - **Downsize**, **Berserk/^3**, **Smithless/^3**, **Duel/^3**: Applies challenge-specific mechanical rules (housing limits, Frenzy disable, Smithology disable, sub-100% crit chance).
  - **Equipment farming**, **Combat Respec**, **Resource farming (+maps / -maps)**: Specialized mid-run or pre-respec optimization modes.

---

### 3. Enhanced Copy & Paste Workflow
- **One-Click Paste Save (`V`)**:
  - Click **Paste Save** in the sticky header, click **Paste from Clipboard** below the save box, or press `V` anywhere on the page.
  - **Pre-Flight Save Validation**: Decompresses and inspects the clipboard payload *before* touching your active session. Distinguishes valid Trimps save strings from perk export strings or unrelated clipboard text and displays a clear toast notification instead of overwriting your save box with an error message.
  - **Auto-Collapsing Save Panel (`U`)**: Once a valid save is imported, the save acquisition panel automatically collapses to maximize screen space for perks and run inputs. Click **Update Save** (`U`) in the header at any time to expand the panel, inspect or retrieve your raw save string, or paste a new save (`Esc` collapses it back).
- **One-Click Copy Perks (`C`)**:
  - Click **Copy Perks** in the header, click **Copy** next to the export field, or press `C`.
  - Automatically runs `exportPerkString()`, copies the LZString-compressed perk allocation to your clipboard via the async Clipboard API (with `execCommand` fallback), and displays a confirmation toast ready for **Trimps → Portal → Import Perks**.
- **Shared Browser Save Integration (`L`)**:
  - When hosted on the same origin as other Trimps tools that store a save in `localStorage` (`saPlanner.saveString`), a **Use Saved Save** button automatically appears in the header and save toolbar for 1-click loading.
- **Restore Previous Save (`R`)**:
  - Instantly reloads the last Trimps save imported into Surky from `localStorage` (`trimpsSave`).

---

### 4. Keyboard Shortcuts

| Key | Action |
|---|---|
| `1` | **Easy Radon Macro**: Select *Easy Radon challenge*, paste save, Autobuy perks, and copy perk string to clipboard |
| `2` | **Difficult Radon Macro**: Select *Difficult Radon challenge*, paste save, Autobuy perks, and copy perk string to clipboard |
| `3` | **Push Macro**: Select *Push/C^3/Mayhem*, paste save, Autobuy perks, and copy perk string to clipboard |
| `V` | Paste and validate Trimps save string from clipboard |
| `U` | Expand / collapse the save acquisition panel to edit or retrieve your save string |
| `C` | Export and copy compressed perk string to clipboard |
| `A` | Clear & Autobuy optimal perks (respec available) |
| `Shift+A` | Clear & Autobuy optimal perks and automatically copy perk string to clipboard |
| `B` | Autobuy optimal perks keeping current levels (no respec) |
| `R` | Restore previous save stored in Surky |
| `L` | Load shared save string from browser `localStorage` (when available) |
| `T` | Toggle between Dark and Light themes |
| `?` | Open the Shortcuts, Workflow & Credits dialog |

---

### 5. Trimps 5.10.1 Compatibility Layer (`overlay.js`)

Upstream `nsheetz/perks` (`c57fd98`, 2024-04-20) has not been updated for Trimps `5.10.0` / `5.10.1` and never parsed Universe 2 Mutators (`game.global.u2MutationData`) introduced in `5.8.0`–`5.9.0`. `overlay.js` non-destructively extends Surky's evaluation pipeline at runtime so [`upstream/index.html`](./upstream/index.html) remains 100% unmodified while delivering full **Trimps 5.10.1** accuracy:

| Trimps Patch | Game Mechanic | Upstream Surky Behavior | `surky.html` 5.10.1 Behavior |
|---|---|---|---|
| **v5.10.0** | **Glass Reward Radon Cap at Z400** (`glassDone` stops doubling VM Radon at Z400+) | Treated `vmZone >= 400` as still receiving 2x Radon per zone (`Greed` over-valued at Z400+) | Caps `props.glassRadon` strictly below Z400 (`(vmZone > 174 && vmZone < 200) \|\| (vmZone > 200 && vmZone < 400)`). |
| **v5.9.0** | **`Runed` Mutator** (+50% `Observation` Runetrinket cap: `1,000` -> `1,500` per level) | Hardcoded `1,000` per `Observation` level (`max 51,000` at L50) in `getTrinketValues`, `getObservationGains`, and `iterateValueFeedback` | Dynamically uses `1,500` per `Observation` level (`max 76,500` at L50) when `u2MutationData.Runed` is purchased and updates strategy callouts. |
| **v5.8.0** | **`Trimps` ("Small Trimps") Mutator** (`1.5x` max housing) | Ignored in `Combat-only respec` (`populatePerks()`), causing 4–5 wasted levels of `Carpentry` to reach current population | Multiplies `props.maxTrimps` by `1.5` when `u2MutationData.Trimps` is purchased. |
| **v5.8.0–5.9.0** | **`GeneAttack` / `GeneHealth` & Imp-ort Mutators** (`50x` slower breeding each; `Randimp1`/`2` & `Tauntimps` extra cell spawns) | Ignored in `potency` and `imperzone` (under-bought `Pheromones` during `Trappapalooza` / `Alchemy`) | Divides `props.potency` by `50` per active Gene mutator and adds `+0.6`/`+1.2` (`Randimp1`/`2`) and `+0.2` (`Tauntimps`) to `props.imperzone`. |
| **v5.8.0+** | **Z201+ Mutated Enemy Stat Scaling** (`1.01x` Attack & `1.02x` Health per zone at Z201+) | Only modeled `Z300+` `1.15x` scaling | Adds `Math.log(1.01) + Math.log(1.02)` to `props.logEnemyScaling` when `targetZone >= 201`. |
| **UI Bug** | **`#input-smithology` & `#input-expansion` HTML `max="30"`** | Copy-pasted `max="30"` attribute from `Observation` triggered HTML5 invalid warnings above level 30 | Strips the erroneous `max` attribute from both unlimited perks at startup. |

When a loaded save has purchased Universe 2 Mutators (`u2MutationData`), the header status pill (`#surkyStatusPill`) displays `Mut <count>` and lists the active perk-affecting mutators in its tooltip.

---

## 🎨 Design System & Theme Overhaul (`theme.css`)

`theme.css` is injected after the upstream stylesheet to harmonize every visual surface with our dark/light design system while leaving Surky's DOM IDs and calculation hooks untouched:

| Upstream Element / Class | Overhauled Design Token / Treatment |
|---|---|
| `body`, `.container-fluid` | Ambient `#08090c` dark canvas (`#f3f4f7` light) with subtle orange/blue radial gradients and centered `1480px` max-width container. |
| `#perks-col`, `#aux-col`, `.form-group` | Elevated `--surface` cards (`#14171f` / `#ffffff`) with `14px` border radius, subtle 1px borders, and ambient shadows. |
| `.bg-success` / `.bg-success-dark` | Semantic emerald surface (`--buy-soft`) with a 3px solid `--buy` (`#22c55e`) left accent bar for required user inputs. |
| `.bg-primary` / `.bg-primary-dark` | Semantic royal-blue surface (`--info-soft`) with a 3px solid `--info` (`#60a5fa`) left accent bar for fields loaded from any save. |
| `.bg-danger` / `.bg-danger-dark` | Semantic rose surface (`--sell-soft`) with a 3px solid `--sell` (`#f43f5e`) left accent bar for end-of-run U2 save fields. |
| `.bg-warning` / `.bg-warning-dark` | Semantic amber surface (`--warn-soft`) with a 3px solid `--warn` (`#fbbf24`) left accent bar for challenge-specific fields and input warnings. |
| `.form-control`, `#preset-select` | Monospace tabular-numeral inputs (`--mono`) on `--surface-2` with orange accent focus rings (`--accent`). |
| `#input-dark-mode` | Replaced by the sticky header Dark/Light toggle (`#btnTopTheme`), which synchronizes both `data-theme` and Surky's internal `props.darkMode` opacity math. |

---

## 🧩 Non-Destructive Overlay Architecture

Rather than hand-editing a fork of `nsheetz/perks`, `surky-build/` keeps the pristine upstream `index.html` in `upstream/index.html` and applies our theme and UX layer at build time via `assemble.py`.

### Structural Injection Anchors
`assemble.py` verifies that all 7 structural regex anchors match **exactly once** before building. If an upstream update alters the page structure in a way that breaks any anchor, the build aborts with a diagnostic error and leaves your existing `surky.html` untouched:

| Anchor | Pattern | Purpose |
|---|---|---|
| `html_open` | `<html>` | Prepends `<!DOCTYPE html>`, build provenance comment, and `<html lang="en" data-theme="dark">`. |
| `head_open` | `<head>` | Injects [`head-inject.html`](./head-inject.html) (strict CSP, metadata, inline SVG favicon, pre-paint theme script). |
| `title_tag` | `<title>.*?</title>` | Sets clean page title (`Surky Perk Optimizer — Trimps`). |
| `style_head_close` | `</style>\s*</head>` | Appends [`theme.css`](./theme.css) inside `<style id="surky-sa-theme">` right after upstream CSS. |
| `body_open` | `<body>` | Injects [`body-inject.html`](./body-inject.html) (sticky header, status pills, toast host, help dialog). |
| `main_script` | `<script type="text/javascript">...</script>` | Extracts upstream JS for `node --check` validation and appends [`overlay.js`](./overlay.js). |
| `body_close` | `</body>` | Validates document termination. |

### Stripped External Dependencies (100% Offline & CSP-Safe)
Upstream `index.html` references 6 Google Fonts `@font-face` blocks (`fonts.gstatic.com`) and 3 unused CDN `<script>` tags (`code.jquery.com`, `cdnjs.cloudflare.com/ajax/libs/popper.js`, `maxcdn.bootstrapcdn.com/bootstrap`). `assemble.py` strips these automatically and runs a post-assembly **self-containment check** confirming zero external scripts, stylesheets, fonts, or parent-directory (`../`) asset dependencies remain.

---

## 📦 Standalone Self-Hosting & Portability

- **To host the calculator**: Copy [`trimps/surky.html`](../surky.html) to any static web server, GitHub Pages site, or open it directly via `file://`. It has **zero external file or network dependencies**.
- **To rebuild or pull upstream updates**: Copy the `surky-build/` folder anywhere. It requires only **Python 3** (standard library only; no `pip` packages) and **PowerShell** (`node` and `git` are optional enhancements for syntax checking and working-tree diff display).

---

## 📁 Architecture & File Layout

| File | Purpose |
|---|---|
| [`upstream/index.html`](./upstream/index.html) | Pristine, unmodified snapshot of `index.html` from [`github.com/nsheetz/perks`](https://github.com/nsheetz/perks). |
| [`upstream.json`](./upstream.json) | Upstream commit SHA, commit date, commit message, fetch timestamp, and SHA-256 checksum. |
| [`head-inject.html`](./head-inject.html) | Content-Security-Policy, metadata, inline SVG data-URI icons, and pre-paint theme initializer. |
| [`body-inject.html`](./body-inject.html) | Sticky top header bar, live status pills, animated action buttons, toast container, and help `<dialog>`. |
| [`theme.css`](./theme.css) | Dark & light design system variables, card surfaces, semantic field highlights, and responsive grid overrides. |
| [`overlay.js`](./overlay.js) | Non-invasive IIFE handling Trimps 5.10.1 mechanics (`Runed`, `Glass` Z400 cap, U2 mutators, Z201+ scaling), clipboard paste/copy, save validation, theme sync, status pills, and keyboard shortcuts. |
| [`fetch_upstream.py`](./fetch_upstream.py) | Python downloader that pulls the latest `nsheetz/perks` commit metadata and `index.html` snapshot. |
| [`assemble.py`](./assemble.py) | Anchor-checked Python bundler that strips CDN tags, injects overlays, verifies self-containment, and writes `surky.html` (CRLF). |
| [`refresh.ps1`](./refresh.ps1) | PowerShell build pipeline coordinating upstream fetch, `node --check` syntax validation, diff reporting, and assembly. |
| [`README.html`](./README.html) | Interactive HTML documentation viewer with live Markdown rendering, Dark/Light theme toggle, and `file://` fallback. |

---

## 🚀 Build Pipeline (`refresh.ps1`)

Run from PowerShell inside `trimps/surky-build/`:

```powershell
# Fast build: Validates JavaScript syntax and compiles surky.html from local files
.\refresh.ps1 -AssembleOnly

# Standard offline build: Validates snapshot + overlays and compiles ..\surky.html
.\refresh.ps1

# Fresh upstream sync: Downloads latest nsheetz/perks index.html, reports perk/preset diffs, and rebuilds surky.html
.\refresh.ps1 -Fresh

# Pin to a specific upstream commit or branch and output to a custom path
.\refresh.ps1 -Fresh -Ref c57fd98 -OutFile C:\path\to\surky.html
```

### Script Parameters

| Parameter | Type | Description |
|---|---|---|
| `-AssembleOnly` | Switch | Skips upstream snapshot fetch and rapidly validates & compiles `surky.html` from local files. |
| `-Fresh` | Switch | Downloads the latest `index.html` and commit metadata from `github.com/nsheetz/perks` before rebuilding. |
| `-Ref` | String | Git branch or commit SHA to fetch when `-Fresh` is specified (default: `master`). |
| `-OutFile` | String | Custom output file path (default: `..\surky.html` relative to `surky-build/`). |
| `-Help` | Switch | Displays full parameter documentation and usage examples. |

---

## 🛡️ Upstream Protection (`updtrimps.ps1`)

When hosted inside the main Trimps repository, `updtrimps.ps1` synchronizes `trimps/` with upstream `Trimps.github.io` while explicitly preserving both `trimps/surky.html` and the entire `trimps/surky-build/` directory alongside `trimps/sw.js` offline precaching.
