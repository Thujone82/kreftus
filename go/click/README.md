# Click

Windows app that counts down, then plays a mouse macro at the speed it was recorded.

Set a countdown, record a short sequence of clicks, then start. When the countdown reaches zero, Click hides and plays the sequence once. It does not retrace the pointer's path. It moves in a straight line to each click, taking as long as you took to get there, then clicks. A drag is a straight line while the button stays down.

## Requirements

- Windows
- Go 1.22 or later to build

## How to use

1. Run `click.exe`.
2. Enter a countdown. `5s` is the default. A bare number is seconds. You can also type `80m`, `1200s`, or `1h20m10s`. `0` plays as soon as you start. The maximum is 24 hours.
3. Press **Record** or **Alt+Shift+_** and perform the macro in any other window. Press **Stop** or **Alt+Shift+_** when you are done.
4. Press **Start** or **Alt+Shift+(**. The status line counts down, then the macro plays once.
5. Press **Alt+Shift+)** or **Abort** to cancel a countdown or playback. **Clear** discards the recording.

Clicks on the Click window stay out of the macro. Pressing Stop only controls Click. The recording stops on its own after 60 seconds or 100,000 events. It is kept in memory until you clear it, record again, or close the app.

Hotkeys, while Click is open:

- **Alt+Shift+_** — start or stop recording
- **Alt+Shift+(** — start the countdown
- **Alt+Shift+)** — stop the countdown or playback

On a US keyboard those are Alt+Shift together with the minus, 9, and 0 keys. The window stays on top during the countdown so you can see the timer. It hides while the macro plays, then comes back. Alt+Shift+) still stops playback while the window is hidden.

## Build

From `go/click`:

```powershell
./build.ps1
```

When `windres` is available, the build embeds `click.ico` into both executables. This writes GUI executables (no console window) to:

- `./bin/win/x86/click.exe`
- `./bin/win/x64/click.exe`

Pass `-upx` to compress those files when UPX is on your PATH.

To compile the current machine's version in place:

```powershell
go build -ldflags "-s -w -H=windowsgui" -o click.exe .
```

## Limits

Playback is simulated input. To drive a program that is running as administrator, run Click as administrator too. Record and play at the same display scale; coordinates are physical screen pixels, including monitors beside the primary one.
