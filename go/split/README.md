# Split

Fast, lightweight lossless keyframe video splitter, trimmer, and joiner for Windows, Linux, and macOS (GUI + CLI) written in Go (`CGO_ENABLED=0`).

## Features

- **Per-OS Native Efficiency**: Uses Go build tags so each OS binary links only its own native GUI host — Edge WebView2 on Windows (`-H=windowsgui` + parent console attach for CLI), GTK3 + WebKit2GTK on Linux via `purego`, and Cocoa + `WKWebView` on macOS via `purego`.
- **Drag & Drop or Browse**: Drop any video file anywhere onto the window, or use the native file picker (`Ctrl+O` / `Cmd+O`).
- **Audio Sound-Level & Voice/Noise Histogram**: Multithreaded PTS-locked audio analysis displays peak & RMS sound levels colored by Talking (amber-gold), Noise/Transients (pink), and ambient audio (cyan) directly above the timeline segments (toggleable & persisted).
- **Visual Frame-Delta & Scene-Cut Lane**: Multithreaded downscaled visual analysis displays frame-to-frame motion delta and color-histogram scene-cut spikes alongside the audio histogram (toggleable & persisted).
- **Resizable Split View**: Drag the grabber bar between the video player and transport/timeline dock to dynamically resize histogram & timeline height (double-click to reset).
- **Non-Blocking Progress, Load Cancel & Right-Click Refresh**: Live progress bars for initial video indexing (`Esc` to cancel) and background audio/visual analysis while playback stays unlocked, plus a right-click context menu on timeline lanes, overview strip, or file badge to refresh Keyframes, Audio, Visual Δ, or All independently.
- **Zoom & Pan Timeline**: Zoom in with the mouse wheel, `+`/`-` keys, or zoom slider (with an interactive overview minimap below) for fine marker placement.
- **Keyframe-Snapped Markers & Live Keyframe Scrubbing**: Cut points automatically snap to the nearest video keyframe for zero-reencode lossless stream copy (`ffmpeg -c copy`), and scrubbing the timeline or dragging cut markers previews the nearest keyframe smoothly without black screen flashes.
- **Keep / Drop Segment Toggling**: Click any timeline segment or pill to toggle **Keep** / **Drop**, drag cut markers along keyframes, and preview with **Skip Dropped** enabled.
- **Join & Split Export Modes**:
  - **Join (`_edit`)**: Combines all kept segments into a single output file.
  - **Split (`_01`, `_02`, ...)**: Exports each kept segment into its own numbered file.
- **Light & Dark Theme**: Instant theme toggle with synchronized Windows titlebar styling.
- **Transparent Remux Playback & Variable Speed**: Native HTTP 206 streaming for MP4/WebM/MOV, automatic live fragmented-MP4 remux fallback for MKV/AVI/AC3 containers, and playback speeds from `0.1×` to `10×`.
- **Full CLI Parity**: Replaces `splitvid.ps1` (`-2..-10`, `-half`, `-thirds`, `-quarters`, `-trim`, `-crop`) and adds `-keep`, `-cuts`, `-drop`, `-join`, `-split`, and `-info`.

## Prerequisites

### Windows Executable (`bin/win/x86/split.exe`, `bin/win/x64/split.exe`)
- **Windows 10 or Windows 11** + **Microsoft Edge WebView2 Runtime** (pre-installed on Windows 11 / modern Windows 10)
- **FFmpeg & FFprobe**: `winget install Gyan.FFmpeg` (or `scoop install ffmpeg`, or sibling beside `split.exe`)

### Linux Executable (`bin/linux/x64/split`, `bin/linux/arm64/split`)
- **GTK3 & WebKit2GTK** (`libgtk-3-0`, `libwebkit2gtk-4.1-0`) + **GStreamer** codecs for `<video>` preview (`gstreamer1.0-libav`)
- **FFmpeg & FFprobe**: `sudo apt install ffmpeg libwebkit2gtk-4.1-0 gstreamer1.0-libav` (or distro equivalent)

### macOS Executable (`bin/mac/arm64/split`, `bin/mac/x64/split`)
- Built-in **WebKit** (`WKWebView`) & **Cocoa** (no extra WebView runtime required; run `chmod +x split` and `xattr -d com.apple.quarantine split` if downloaded)
- **FFmpeg & FFprobe**: `brew install ffmpeg`

### Building from Source
- **Go 1.24+** (pure-Go cross-compilation with `CGO_ENABLED=0`; no C compiler or target SDK needed)
- **PowerShell 7 (`pwsh`)** (for `build.ps1`)
- Optional: **MinGW `windres`** (embeds `split.ico` resource on Windows) and **UPX** (when passing `-upx`)

## CLI Usage

```powershell
# Launch GUI (or pre-load a video)
.\split.exe
.\split.exe video.mp4

# Balanced byte-size keyframe split (2..10 parts -> _1ofN .. _NofN)
.\split.exe video.mp4 -2
.\split.exe video.mp4 -quarters

# Lossless trim or crop
.\split.exe video.mp4 -trim -s 1m3.2s -e 2m
.\split.exe video.mp4 -crop 1m3s 5m2.2s

# Multi-segment keep/drop with Join (_edit) or Split (_01, _02...)
.\split.exe video.mp4 -keep 0:10-1:25,2m-end -join
.\split.exe video.mp4 -cuts 30s,1m15s,2m -drop 2 -split

# Inspect stream metadata & keyframes
.\split.exe video.mp4 -info
```

## Keyboard Shortcuts (GUI)

- `Space`: Play / Pause
- `C` or `M`: Add cut marker at nearest keyframe
- `[` / `]`: Jump to previous / next keyframe
- `,` / `.`: Step back / forward 1 frame
- `+` / `-` / `0`: Zoom timeline in / out / fit (or mouse wheel)
- `←` / `→`: Seek `-5s` / `+5s`
- `Ctrl+O` / `Cmd+O`: Open video file
- `Ctrl+E` / `Cmd+E`: Export kept segments
- `Esc`: Cancel active video load

## Build

```powershell
.\build.ps1
```

Cross-compiles `./bin/win/x86/split.exe`, `./bin/win/x64/split.exe`, `./bin/linux/x64/split`, `./bin/linux/arm64/split`, `./bin/mac/arm64/split`, and `./bin/mac/x64/split` (pass `-upx` to compress Windows and Linux binaries with UPX).
