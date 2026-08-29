# ⏱️ Modern Radial Countdown Timer (PWA)

A minimalist, high-precision Progressive Web App (PWA) countdown timer featuring a signature **dual-layer conic-gradient inverted color sweep**, synthesized Web Audio chimes, Screen Wake Lock support, persistent settings memory, multi-theme customization, quick presets, and complete keyboard accessibility.

---

## ✨ Features

- **📱 Installable Progressive Web App (PWA)**: Installable directly to your desktop, home screen, or dock on Chrome, Edge, Safari, iOS, and Android with custom app icon (`timer.png`).
- **💾 Persistent Settings Memory**: Automatically remembers your last used duration (presets or custom time), active theme, and sound toggle preferences so it reopens exactly how you left it.
- **🌀 Dual-Layer Conic Sweep**: Beautiful 12 o'clock clockwise radial mask that inverts the background and typography colors as time progresses.
- **⚡ High-Precision Timing Engine**: Driven by `performance.now()` timestamps rather than drifting intervals, ensuring millisecond accuracy even across throttled background tabs.
- **🔄 Tab Visibility & Offline Sync**: Uses `visibilitychange` for instant sync upon tab return, and a Service Worker (`sw.js`) for 100% offline functionality.
- **🎵 Built-in Web Audio Alert Tones**: Synthesizes a crisp, ascending melodic chime across multiple harmonic oscillators upon timer expiration—no external audio files or permission dialogs required.
- **⚡ 2 Hz Screen Inversion Flash**: High-visibility 2 Hz screen inversion flash effect upon countdown completion to immediately grab visual attention.
- **📱 Screen Wake Lock API**: Automatically keeps phone, tablet, and laptop displays awake while the timer is actively counting down.
- **🎨 6 Color Themes**: Seamlessly cycle between *Monochrome*, *Midnight Neon*, *Amber CRT*, *Crimson Stealth*, *Emerald Forest*, and *Nord Frost*.
- **📊 Dynamic Tab Title & Live Favicon**: Displays the remaining time in the browser tab title (e.g. `(01:45) Timer`) and generates real-time SVG pie-chart favicons indicating progress.
- **⏱️ Quick Presets & Custom Duration**: Jump between 30s, 1m, 2m, 3m, 5m, 10m, and 25m (Pomodoro), or enter custom minutes and seconds via modal.
- **🛡️ Running Timer Protection & State Recovery**: Opening the custom menu during a countdown never pauses or loses running timers (custom time inputs lock automatically while color and font customizations remain live). If the browser crashes or the page is refreshed during an active countdown, the timer automatically detects its previous running state and restores the countdown seamlessly or triggers completion.
- **🔗 URL Parameter & Hash Support**: Launch with custom configurations directly via URL query parameters or hash fragments.
- **⌨️ Keyboard & Touch Optimized**: Full hotkey support, auto-hiding controls during countdowns, zero-latency stage tap to toggle, and dedicated reset controls.

---

## ⌨️ Keyboard Shortcuts

| Shortcut | Action |
| :--- | :--- |
| <kbd>Space</kbd> / <kbd>K</kbd> | Play / Pause countdown |
| <kbd>R</kbd> | Reset timer to initial duration |
| <kbd>+</kbd> / <kbd>↑</kbd> | Add 30 seconds |
| <kbd>-</kbd> / <kbd>↓</kbd> | Subtract 30 seconds |
| <kbd>1</kbd> – <kbd>7</kbd> | Switch presets (`30s`, `1m`, `2m`, `3m`, `5m`, `10m`, `25m`) |
| <kbd>C</kbd> | Open Custom Time dialog |
| <kbd>T</kbd> | Cycle Color Themes |
| <kbd>M</kbd> | Toggle Sound Mute / Unmute |
| <kbd>F</kbd> | Toggle Fullscreen mode |
| <kbd>?</kbd> / <kbd>H</kbd> | Open Keyboard Shortcuts help overlay |
| <kbd>Esc</kbd> | Close dialogs / Exit fullscreen |

---

## 🔗 URL Parameters & Query Options

You can deep-link or bookmark specific durations and settings using URL parameters or hashes:

| Parameter | Example | Description |
| :--- | :--- | :--- |
| `?t=` / `?time=` | `index.html?t=90` or `index.html?t=30s` | Set timer in seconds (90s / 30s) |
| `?t=Xm` | `index.html?t=5m` | Set timer in minutes (5m) |
| `?t=XmYs` | `index.html?t=2m30s` | Set timer in minutes & seconds (2m 30s) |
| `?t=Xh` / `?t=XhYm` | `index.html?t=1h` or `index.html?t=1h30m` | Set timer in hours or hours & minutes |
| `?t=MM:SS` / `HH:MM:SS` | `index.html?t=1:45` or `index.html?t=1:30:00` | Set timer via clock notation |
| `#...` | `index.html#3m` or `index.html#30s` | Set duration using URL hash |
| `?theme=` | `index.html?theme=amber` | Load specific theme (`monochrome`, `midnight`, `amber`, `crimson`, `emerald`, `nord`, `custom`) |
| `?bg=` & `?fg=` | `index.html?bg=1a0933&fg=fcee0a` | Set custom two-color theme (Base Background & Sweep Accent) |
| `?font=` | `index.html?font=mono` | Set typography style (`system`, `mono`, `rounded`, `serif`, `condensed`) |
| `?autostart=1` | `index.html?t=2m&autostart=1` | Automatically begin countdown on load |

---

## 🎨 Color Themes

| Theme Name | Base Palette | Inverted Overlay | Aesthetic |
| :--- | :--- | :--- | :--- |
| **Monochrome** *(Default)* | Pure White / Black | Solid Black / White | Minimalist Swiss |
| **Midnight** | Deep Navy (`#0a0e17`) | Cyan Glow (`#00f2fe`) | Cyberpunk / Neon |
| **Amber** | Dark Charcoal (`#121212`) | Warm Amber (`#ffb000`) | Retro CRT Terminal |
| **Crimson** | Pitch Black (`#0f0f0f`) | Neon Crimson (`#ff3366`) | Stealth / High Energy |
| **Emerald** | Forest Dark (`#061a14`) | Mint Emerald (`#10b981`) | Clean Bio / Productivity |
| **Nord** | Polar Night (`#2e3440`) | Frost Blue (`#88c0d0`) | Arctic Scandinavian |
| **Custom** *(User-defined)* | Custom Base Color | Custom Inverted Sweep | Configurable via Custom menu (<kbd>C</kbd>) or URL params |

---

## 🔤 Typography Styles

| Style Option | Font Stack | Visual Aesthetic |
| :--- | :--- | :--- |
| **System Sans** *(Default)* | `-apple-system`, `BlinkMacSystemFont`, `Segoe UI`, `Roboto` | Clean, modern, native OS feel |
| **Monospace** | `ui-monospace`, `SF Mono`, `Menlo`, `Consolas`, `monospace` | Retro terminal / digital code clock |
| **Rounded** | `ui-rounded`, `SF Pro Rounded`, `Nunito`, `system-ui` | Soft, friendly, modern aesthetic |
| **Classic Serif** | `Georgia`, `Cambria`, `Times New Roman`, `serif` | Editorial, luxury timepiece |
| **Impact Condensed** | `Impact`, `Arial Narrow`, `Haettenschweiler`, `sans-serif` | Bold poster / scoreboard display |

---

## 🚀 Installation & Getting Started

### Install as PWA (Desktop & Mobile)
When served over HTTP/HTTPS or local dev server:
1. Open [`index.html`](file:///C:/kreftus/timer/index.html) in your browser (Chrome, Edge, Safari, Opera).
2. Click the **Install** icon in the top toolbar or use the browser address bar prompt ("Install Timer").
3. Launch directly from your desktop, taskbar, dock, or home screen like a native app.

### Offline & Local Use
Double-click [`index.html`](file:///C:/kreftus/timer/index.html) to run locally, or host with any static web server:

```bash
# Python local server
python -m http.server 8000

# Node.js serve
npx serve .
```

---

## 📂 Project Structure

```text
kreftus/timer/
├── index.html       # Application interface and PWA entry point
├── manifest.json    # Web App Manifest configuration
├── sw.js            # Service Worker for offline asset caching
├── timer.png        # PWA app icon (256x256)
├── timer.ico        # Favicon and Windows icon
└── README.md        # Comprehensive documentation
```

---

## 🌐 Browser Compatibility

- **Google Chrome / Chromium**: Full PWA installability, Service Worker, Web Audio, Wake Lock, CSS Masks.
- **Microsoft Edge**: Full PWA & desktop installation support, Web Audio, Wake Lock.
- **Apple Safari (macOS & iOS)**: Full PWA "Add to Home Screen" support, Web Audio chimes, WebKit Masks, Wake Lock.
- **Mozilla Firefox**: Full offline Service Worker, Web Audio, CSS Conic Masks.
- **Android Browsers**: Full PWA home screen installation, Web Audio, Wake Lock.

---

## 📄 License

MIT License — Free for personal and commercial use.
