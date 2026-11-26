# Split

Fast, lightweight lossless keyframe video splitter, trimmer, and joiner for Windows (GUI + CLI) written in Go.

## Features

- **No Console Window on Launch**: Built with `-H=windowsgui` for silent desktop launch; automatically attaches to the parent console when invoked with CLI flags.
- **Drag & Drop or Browse**: Drop any video file anywhere onto the window (or onto `split.exe`), or use the native Windows file picker (`Ctrl+O`).
- **Audio Sound-Level Histogram**: Displays peak & RMS sound levels directly above the timeline segments so speech pauses and silent gaps are easy to spot.
- **Zoom & Pan Timeline**: Zoom in with the mouse wheel, `+`/`-` keys, or zoom slider (with an interactive overview minimap below) for fine marker placement.
- **Keyframe-Snapped Markers**: Cut points automatically snap to the nearest video keyframe for zero-reencode lossless stream copy (`ffmpeg -c copy`).
- **Keep / Drop Segment Toggling**: Click any timeline segment or pill to toggle **Keep** / **Drop**, drag cut markers along keyframes, and preview with **Skip Dropped** enabled.
- **Join & Split Export Modes**:
  - **Join (`_edit`)**: Combines all kept segments into a single output file.
  - **Split (`_01`, `_02`, ...)**: Exports each kept segment into its own numbered file.
- **Light & Dark Theme**: Instant theme toggle with synchronized Windows titlebar styling.
- **Transparent Remux Playback**: Native HTTP 206 streaming for MP4/WebM/MOV, plus automatic live fragmented-MP4 remux fallback for MKV/AVI/AC3 containers.
- **Full CLI Parity**: Replaces `splitvid.ps1` (`-2..-10`, `-half`, `-thirds`, `-quarters`, `-trim`, `-crop`) and adds `-keep`, `-cuts`, `-drop`, `-join`, `-split`, and `-info`.

## Requirements

- Windows 10 or Windows 11 (Microsoft Edge WebView2 Runtime)
- `ffmpeg` and `ffprobe` on `PATH` (`winget install Gyan.FFmpeg`)

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
- `Ctrl+O`: Open video file
- `Ctrl+E`: Export kept segments

## Build

```powershell
.\build.ps1
```

Outputs `./bin/win/x86/split.exe` and `./bin/win/x64/split.exe` (pass `-upx` to compress with UPX).
