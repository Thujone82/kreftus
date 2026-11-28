# mind

Terminal-based **Mastermind** code-breaking game written in Go with a full-screen true-color TUI, 6 selectable themes, UTF-8 box drawing, animated feedback & win celebrations, persistent stats, and synthesized PCM sound effects.

## Requirements

- Go 1.24 or later (see `go.mod`).
- A UTF-8 terminal with 24-bit (true-color) ANSI support (Windows Terminal, modern `conhost`, macOS Terminal/iTerm2, or Linux terminals). On Windows, `mind` automatically enables UTF-8 (CP 65001) and Virtual Terminal processing.

## Game rules

- **Secret**: The program picks a random code of 4 pegs, or you can set it with `-set` (e.g. `mind -set r22m`) for another player to guess. Each peg is one of 6 colors: **R**ed, **G**reen, **B**lue, **C**yan, **M**agenta, **Y**ellow (order RGBCMY). Colors may repeat.
- **Turns**: You have up to **12** turns to guess the code.
- **Feedback** (revealed peg-by-peg after each guess):
  - **Exact (`●`)**: Correct color in the correct position.
  - **Partial (`●` or `○` depending on theme)**: Correct color in the wrong position (each peg in the secret is counted at most once).
- **Win**: Your guess matches the secret exactly (4 exact pegs).
- **Lose**: You run out of turns; the secret is revealed at the top of the board.

## Themes

Press **`T`** at any time (on the title screen, mid-game, or on the result overlay) to cycle through the 6 built-in themes, or launch with `-theme <id>`. Your selection is saved automatically.

| ID | Name | Look |
| --- | --- | --- |
| `dark` | Dark | Deep charcoal background, double-line box (`╔═╗`), vibrant colored pegs |
| `light` | Light | Warm paper background, rounded box (`╭─╮`), crisp dark ink text |
| `neon` | Neon | Midnight synthwave violet, heavy box (`┏━┓`), electric neon pegs |
| `ocean` | Ocean | Deep sea navy, rounded box (`╭─╮`), coral and aqua accents |
| `wood` | Wood Board | Classic tabletop mahogany, double-line box (`╔═╗`), white/hollow key pegs |
| `contrast` | High Contrast | Pure black, heavy box (`┏━┓`), labeled color tiles (` R `) for maximum legibility |

## Controls & keys

- **Pegs**: `R` `G` `B` `C` `M` `Y` (case-insensitive) or number aliases `1` `2` `3` `4` `5` `6` (`1`=Red, `2`=Green, `3`=Blue, `4`=Cyan, `5`=Magenta, `6`=Yellow).
- **Backspace**: Removes the last entered peg.
- **Enter**: Starts the game from the title screen, submits a 4-peg guess, or starts a new game from the result overlay.
- **`T`**: Cycle theme live.
- **`S`**: Toggle sound effects on/off.
- **`L`**: Toggle peg labels on/off (` R ` tile mode vs `⬤` dot mode).
- **`Tab`**: Toggle label style between color letters (`R G B C M Y`) and color numbers (`1`–`6`); in the side panel, the highlighted column switches accordingly. Both label settings are saved.
- **`N`**: Start a new game after winning or losing.
- **`Esc`** (or **`Q`** / `Ctrl+C`): Quit and restore the terminal.

## Sound

- **Windows (local)**: Synthesizes 22,050 Hz mono 16-bit PCM WAV cues in memory (peg clicks, deletes, invalid buzz, submit, feedback reveals, win fanfare, lose jingle, theme switch) and plays them asynchronously via `winmm.dll` (`PlaySoundW`).
- **Linux/macOS & SSH**: Falls back to the terminal bell (`\a`) for major events only (win, lose, invalid submit).
- Pass `-mute` to silence sound for a single run without changing your saved preference, or press **`S`** to toggle and save it.

## How to run & flags

From the `go/mind` directory:

```bash
go run .
```

Available flags:

- `-set <code >` — Set a 4-peg secret for the first game (e.g. `mind -set r22m`).
- `-theme <id>` — Select and save a starting theme (`dark`, `light`, `neon`, `ocean`, `wood`, `contrast`).
- `-mute` — Disable sound for this session only (does not overwrite saved settings).
- `-preview` — Print sample mid-game and won boards for all 6 themes to stdout and exit.

When stdin is not an interactive terminal (e.g. piped input), `mind` automatically falls back to plain line-oriented mode.

## Settings & stats

Preferences (theme, sound, labels) and lifetime stats (games played, wins, current/best streak, best turns, fastest solve) are stored as UTF-8 BOM JSON in your user config directory:

- Windows: `%AppData%\mind\mind.json`
- Linux/macOS: `~/.config/mind/mind.json` (or `$XDG_CONFIG_HOME/mind/mind.json`)

## How to build (cross-compilation)

From the `go/mind` directory, run the PowerShell build script:

```powershell
./build.ps1
```

Or:

```powershell
pwsh -File build.ps1
```

This will:

1. Clean the `./bin` directory (if it exists).
2. Run `go mod tidy`.
3. Build for Windows (x86 and x64) and Linux (x86 and amd64).
4. Place binaries under `./bin/`:
   - `./bin/win/x86/mind.exe`
   - `./bin/win/x64/mind.exe`
   - `./bin/linux/x86/mind`
   - `./bin/linux/amd64/mind`

Optional: compress binaries with [UPX](https://upx.github.io/) by passing `-upx`:

```powershell
./build.ps1 -upx
```

## File layout

| File | Description |
| --- | --- |
| `main.go` | CLI flags, app lifecycle, title/game/animation loops |
| `game.go` | Core constants, secret generation, code parsing, scoring, game state |
| `theme.go` | 6 true-color themes and box-drawing styles |
| `render.go` | Cell canvas, ANSI true-color serializer, screens, overlays, `-preview` |
| `input.go` | Raw-mode terminal session, escape-sequence parser, clean restore |
| `plain.go` | Non-TTY line-oriented fallback |
| `sound.go` | Sound specs, in-memory WAV synthesizer, mute/SSH routing |
| `sound_windows.go` | Windows `winmm.dll` async `PlaySoundW` playback |
| `sound_other.go` | Non-Windows PCM stub (uses terminal bell fallback) |
| `console_common.go` | SSH session detection |
| `console_windows.go` | Windows UTF-8 (CP 65001), VT processing, Unicode console font setup |
| `console_other.go` | Non-Windows console setup no-op |
| `settings.go` | UTF-8 BOM JSON settings & stats persistence |
| `*_test.go` | Unit tests for game logic, themes, WAV synth, settings, rendering |
| `build.ps1` | Cross-build script (Windows/Linux) |
| `README.md` | This documentation |

## License

See the repository [LICENSE](../../LICENSE).
