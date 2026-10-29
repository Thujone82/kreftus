# Click

Windows macro recorder, step editor, and scheduler for mouse and keyboard automation.

Click records mouse actions (left/right/middle clicks, double-clicks, drags, scroll wheel) and keyboard input (keystrokes, modifier chords such as `Ctrl+Shift+S`, and typed text), converts them into editable steps, and lets you replay them on demand or on a recurring schedule.

## Requirements

- Windows 10 or Windows 11 (uses the built-in Microsoft Edge WebView2 runtime)
- Go 1.22 or later to build

## Features

1. **Mouse & Keystroke Recording**
   - Polls mouse buttons, wheel, and keyboard virtual keys without stalling the Windows message loop.
   - Coalesces modifier chords (`Ctrl`, `Alt`, `Shift`, `Win`) and optionally merges consecutive typed characters into single **Type Text** steps.
   - Automatically ignores clicks inside the Click window, ignores keys typed while Click is focused, and strips the trailing stop-recording hotkey chord.
   - Straight-line cursor glides between click targets at the recorded speed, with guaranteed modifier/button release on abort.

2. **Macro Panel & Step Editor**
   - New recordings default to `Macro MMDDYY XX` (where `MMDDYY` is the current date and `XX` is the lowest unused two-digit number starting at `01`, expanding to `100+` after `99`).
   - Inline renaming, search/filter, duplicate, delete, and JSON import/export.
   - **Step Editor:** edit pre-step delays, switch clicks between **Left**, **Right**, and **Middle**, edit coordinates or pick a point directly from the screen, re-capture key chords, edit typed text, insert new steps (`Click`, `Double Click`, `Drag`, `Key`, `Type Text`, `Scroll Wheel`, `Wait`), append recordings, adjust macro playback speed (`0.25×`–`4.0×`), drag-and-drop reorder steps (or `Alt+↑`/`Alt+↓`), undo/redo (`Ctrl+Z`/`Ctrl+Y`), and test playback from any selected step.

3. **Macro Scheduler**
   - Schedule any macro to run at a specific date and time, or click **Schedule** on a macro card to pre-fill the scheduler.
   - **Advanced settings:** repeat every interval (`30s`, `15m`, `2h`), daily, or weekly on specific days; restrict execution to an active time-of-day window; stop after $N$ runs, after a duration, or at an end date/time; run multiple back-to-back plays per trigger; configure missed-run policy (*Skip* vs. *Run once ASAP*) and overlap policy (*Skip* vs. *Queue*).
   - Live preview of the next 5 scheduled run times and an **Activity** log of completed, aborted, skipped, and missed runs.
   - Minimizes to the system tray on close when schedules are active so background schedules continue firing.

4. **Portable JSON Storage**
   - Saves `macros.json`, `schedules.json`, `settings.json`, and `history.json` next to `click.exe` (falling back to `%APPDATA%\Click\` if the executable folder is read-only).

## Hotkeys

- **Alt+Shift+_** — start or stop recording
- **Alt+Shift+(** — play the selected macro after the countdown
- **Alt+Shift+)** — abort the active countdown or playback

## Build

From `go/click`:

```powershell
./build.ps1
```

When `windres` is available, the build embeds `click.ico` into both executables and writes GUI binaries to:

- `./bin/win/x86/click.exe`
- `./bin/win/x64/click.exe`

Pass `-upx` to compress those files when UPX is on your `PATH`.

To compile the current machine's architecture in place:

```powershell
go build -ldflags "-s -w -H=windowsgui" -o click.exe .
```
