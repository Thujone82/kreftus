# Knucklebones — Terminal Dice Duel

**Version:** 1.0 · **Author:** Kreft & Opus 5.5 · **Date:** 2026-11-12

## Description

`Knucklebones` is a cross-platform terminal version of the dice game from *Cult of the Lamb*, written in Go with tcell. The terminal UI is the main mode; a matching graphical window (built with [Ebitengine](https://ebitengine.org/)) opens when you double-click the program or pass `-gui`. Two players face each other across 3×3 grids. Each turn you roll one die and drop it into one of your three columns. Matching dice in a column multiply, and placing a die destroys every die of the same value in your opponent's facing column. The game ends the moment either grid is full, and the higher total wins.

Play against three AI opponents, against someone on the same keyboard, or against a friend over the network. It follows the features of [knucklebones.io](https://knucklebones.io/) and the rules in the [dice.now Knucklebones guide](https://dice.now/knucklebones#guide).

On Windows, Knucklebones switches the console to UTF-8 and prefers **Cascadia Mono** when installed so dice and box-drawing characters render cleanly.

## Launch modes

| How it starts | What opens |
| ------------- | ---------- |
| From a terminal or shell (`Knucklebones`) | Terminal UI |
| Double-clicked in Explorer or a file manager | Graphical window |
| `Knucklebones -gui` | Graphical window, even from a terminal |
| `Knucklebones -tui` | Terminal UI, even from a shortcut or double-click |

`-gui` and `-tui` cannot be combined. Both modes share the same save file, settings, AI, and online play, so a terminal player can host or join a window player.

On Windows the program stays a console app so the terminal UI works. A double-click briefly shows a console window, which closes as the game window opens. With `-tui` from a shortcut, a new console opens for the game. Over SSH or with `-tui` / `KNUCKLEBONES_TUI=1` it skips Ebitengine's window setup entirely (that init runs before `main` and would otherwise fail in Session 0) and plays in the terminal only.

The graphical window uses `die.png` for its icon and logo. It resizes freely and scales to fit, and every keyboard control below works in it too.

## Features

- **Start menu**: Play against an AI, Play against someone (same keyboard), Host Game, Join Game, How to play, Stats, Name, Options, and Quit
- **AI opponents**: Ratau (Easy) places in a random open column. Flinky (Medium) takes the biggest immediate swing. Shrumy (Hard) averages your best reply over every roll, looks one step further, and knows when ending the game wins or loses it
- **Game settings**: AI difficulty, number of rounds (Best of 1, 3, 5, or Free play), and house rules. Hosting adds turn timer, port, bind interface (LAN / localhost / specific IP), and an optional ephemeral room code. Your last choices are remembered (room codes are never written to disk)
- **House rules** (from the dice.now guide): turn destruction off, roll two dice and place one, or play until both boards are full
- **Board & visual feedback**: your grid is at the bottom and your opponent's at the top, both filling from the middle outward. Ivory dice are singles, gold dice are pairs, and blue dice are triples; column scores sit above and below the grids, an active border marks whose board is moving (`┃` in TUI, accent frame in GUI), column impact flashes + floating `+gain` / `-loss` badges highlight placements and captures, and an animated `Thinking...` indicator appears during AI or opponent turns
- **Colorblind marks**: press **C** in game (or toggle in **Options**) to mark matching and doomed dice without relying on color alone — corner runes (`=`, `#`, `x`) in the terminal UI and top-edge marks (`=`, `#`, `X`) on GUI dice
- **Placement preview**: while you choose a column, a shaded die shows where yours lands, the opponent's dice it would destroy turn red, column scores show the change (`8→24`), and the panel shows `+gain · destroys N (−points)`
- **Win probability meter**: press **P** in game, toggle in **Options**, or click `Prob [P]` in the GUI side panel to show live Monte Carlo rollout odds for the active board
- **Round replay**: after any finished round, press **X** (or click **Replay**) to step through every move with `←`/`→` or `< Prev` / `Next >`
- **Online live aim**: your opponent sees the rolled die tumble, then watches your column and die selection update the same preview until you place. The board finalizes when the move lands
- **Rolling die**: each new die tumbles briefly in the roller's tray. Red pips mark the last die placed
- **Coin toss**: a random player starts each round
- **Series**: Best of 3 and 5 track rounds won with ●○ markers. After a round, **Continue game**. After the series, **Play another game** or **Go into free play**
- **Log**: the side panel shows the latest moves. **L** opens the full log of rolls, placements, destruction, timeouts, and results
- **History**: **H** lists each round's score and winner, plus who took the lead
- **Resign**: **R** with a confirmation. Vs AI and online it counts as a loss
- **Online play (Protocol v2)**: one player hosts on a desktop binary's dual-protocol TCP + WebSocket port (default `8647`) with an optional bind interface and optional room code; another desktop client or the [Knucklebones Web App (PWA)](../../knucklebones/) joins via the 2-field Join screen (`host:port` + optional room code, last host remembered)
  - When a host sets a room code, both peers authenticate with challenge-response and protect wire messages with HKDF-SHA256 / HMAC-SHA256 and monotonic sequence numbers. Leaving the room code blank keeps the room open for trusted LAN play (unexpected codes are rejected)
  - **Commit-reveal fair play**: the host commits to its dice PRNG seed before play begins and reveals it at round end so the guest verifies all observed rolls (`Fair play commit locked` / `Fair play verified` / `(!) Host dice unverifiable`)
  - While a player aims, their column and die choice are shared (rate-limited) so both screens show the placement preview before the move is confirmed
  - Optional turn timer (15, 30, or 60 seconds). A timed-out turn places the die in a random open column; three timeouts lose the match
  - A player who drops has 30 seconds to reconnect before forfeiting. Leaving mid-game counts as resigning
  - After a game, both players must pick the same option (play again, continue, free play). The result panel shows "Waiting for…" and "… wants to rematch!"
- **Web App (PWA) companion**: play offline in any modern browser or install via **Options** (including iOS Share → Add to Home Screen instructions) with matching GUI & TUI views (**M** or Options), 4 themes (**T** or Options), colorblind marks (**C** or Options), Web Audio sound synthesis (**S**), Monte Carlo win odds (**P**), round replay (**X**), and WebSocket **Join Game** to desktop Go hosts
- **Stats**: games, wins, losses, draws, and best score for each AI, online, and same-keyboard play, plus best-of series won and lost. A Top 10 of final scores; making it is announced on the result panel
- **Themes & Options**: four built-in themes (**Dark**, **Light**, **High Contrast**, **Parchment** — press **T** on the menu or `←`/`→` on **Options**) plus **Options** for **Theme**, **Sound effects**, **Colorblind marks**, and **Win probability**
- **Sound effects**: enabled by default and toggled in **Options → Sound effects** or in game by pressing **S** / clicking `Sound [S]` in the GUI side panel (persisted in `knucklebones.json`). Both the graphical window and local desktop TUI play synthesized PCM sound effects via `ebiten/v2/audio` (`click`, `place`, `destroy`, `turn`, `win`, `lose`). PCM audio is bypassed in headless/SSH sessions (or when `KNUCKLEBONES_HEADLESS=1` is set) and in `CGO_ENABLED=0` builds, where the TUI falls back to throttled terminal bells (`Beep()`) on captures and round outcomes
- **How to play**: rules, the scoring table, house rules, strategy tips, and opponent notes

## Controls

| Context | Input | Action |
| ------- | ----- | ------ |
| Menu | ↑↓ or W/S | Move selection |
| Menu | Enter or Space | Choose |
| Menu | T (or ←→ on Options) | Cycle theme (Dark / Light / High Contrast / Parchment) |
| Menu | Esc | Quit |
| Settings / Options | ↑↓, Tab (or W/S on non-text rows) | Move between settings |
| Settings / Options | ←→, A/D, or Enter | Change the selected setting / toggle option |
| Settings / Options | Typing, ←→, Home/End, click (GUI), Backspace/Delete, Ctrl+U | Edit Player 2 name, Port, or Room code |
| Settings / Options | Enter on Start game / Done | Start game, or return to menu |
| Settings / Options | Esc | Save and return to the menu |
| Name / Join | Tab or ↑↓ (Join) | Switch between Host address and optional Room code |
| Name / Join | Typing, ←→, Home/End, click (GUI), Backspace/Delete, Ctrl+U | Edit text (including literal `w`/`a`/`s`/`d`) |
| Name / Join | Enter | Confirm (`Join` connects) |
| Name / Join | Esc | Cancel and return to menu |
| Lobby | ←→, A/D, or Tab | Switch between `Back` and `Retry` on retryable errors |
| Lobby | Enter or Space | Confirm `Back` or `Retry` |
| Lobby | Esc | Return to Join Game (client) or cancel hosting (host) |
| Your turn | ←→ or A/D | Choose a column |
| Your turn | 1–3 | Place in that column |
| Your turn | Enter or Space | Place in the chosen column |
| Your turn (two dice) | Tab, Q/E, W, or ↑↓ | Switch which die to place |
| In game | C | Toggle colorblind marks |
| In game | P | Toggle Monte Carlo win probability meter |
| In game | S | Toggle sound effects (muted / on) |
| In game | L | Log |
| In game | H | History |
| In game | R | Resign (asks first) |
| In game | Esc | Leave (asks first while a game is in progress) |
| Round over | ←→, Tab | Choose an option |
| Round over | Enter | Confirm the option |
| Round over | V | Hide or show the result panel to see the board |
| Round over | X | Enter round replay |
| Round replay | ←→ or A/D | Step backward / forward one move |
| Round replay | Home / End | Jump to start or end of the round |
| Round replay | Esc, Enter, or X | Exit replay |
| Log / History | ↑↓ or W/S, PgUp/PgDn | Scroll |
| Log / History | Esc, Enter, or the same letter | Close |
| How to play | ↑↓ or W/S, PgUp/PgDn, Space, Home | Scroll |
| How to play / Stats | Esc, Enter, or Q | Return to menu |
| Anywhere | Ctrl+C | Exit |

On the same keyboard, both players use the same keys on their own turn. Player 1 is at the bottom and player 2 at the top.

### Mouse (graphical window)

| Context | Input | Action |
| ------- | ----- | ------ |
| Menu | Hover, click | Select, choose |
| Settings / Options | Click ◀ ▶ or the row | Change the setting (click Player 2, Port, or Room code, then type) |
| Settings / Options | Click Start game, Done, or Back | Start, or back to the menu |
| Join Game | Click field | Focus Host address or Room code box |
| Name / Join / Lobby | Click the buttons | Confirm, retry, cancel, or go back |
| Your turn | Hover a column | Preview the placement |
| Your turn | Click a column (either board) | Place the die there |
| Your turn (two dice) | Click a tray die | Choose which die to place |
| In game | Click `Prob [P]` / `Sound [S]` pill | Toggle win probability meter or sound effects |
| In game | Buttons under the board | Log, History, Resign, Leave / Menu, View board / Show result, Replay (`< Prev`, `Next >`, `Done`) |
| Round over | Click an option | Confirm it |
| Log / History / How to play | Mouse wheel | Scroll |
| Anywhere | Close the window | Leave the game and exit |

## Rules

1. A die is rolled for the player to move.
2. They place it in any of their columns with an empty slot.
3. Every die of the same value in the opponent's facing column is destroyed. The remaining dice slide toward the middle, and the freed slots can be filled again.
4. Scores update and the turn passes.

**Column scoring**: each die is worth its face value times the number of dice of that value in its column. For each value *v* appearing *n* times, the column scores *n* × *n* × *v*.

| Column | Calculation | Score |
| ------ | ----------- | ----- |
| 6 | 6 | 6 |
| 6, 2 | 6 + 2 | 8 |
| 6, 6 | (6×2) + (6×2) | 24 |
| 6, 6, 2 | 24 + 2 | 26 |
| 4, 1, 4 | (4×2) + 1 + (4×2) | 17 |
| 6, 6, 6 | (6×3) × 3 | 54 |
| 3, 3, 3 | (3×3) × 3 | 27 |
| 1, 1, 1 | (1×3) × 3 | 9 |

The best column is 54, so the best board is 162. The game stops as soon as any board is full (or, with the house rule, when both are). A tie is a draw. In a best-of series, ties do not count toward the majority, so a tied round is replayed.

## Online play

1. The host starts **Host Game** in a desktop Knucklebones executable (TUI or GUI), sets rounds, turn timer, port, bind interface, an optional room code, and house rules, then **Start game**. The lobby lists the bound address(es) (`host:port` and `localhost:8647` on the host machine) and active room code indicator (if set).
2. The other player picks **Join Game** in either the desktop app or the [Knucklebones PWA](../../knucklebones/), enters the host address (for example `192.168.1.20:8647` or `localhost:8647`; a bare host name uses port `8647`), and optionally enters the matching room code in the second field. Both desktop and browser PWA remember the last host address used (room codes are never saved).
3. The game starts as soon as the guest completes the Protocol v2 handshake (`unsupported-protocol` rejects incompatible v1 peers). On the host listener (`net.go` / `ws.go`), native TCP and browser RFC 6455 WebSocket upgrades share the same port seamlessly, while the host commits to its dice seed for fair-play verification at round end.

The game is played directly between the host and guest, so both must reach each other (same network, VPN, or a forwarded port; from HTTPS browser origins, same-machine connections use `localhost:8647` / `127.0.0.1:8647`, while remote LAN IPs require a `wss://` TLS reverse proxy or running from a local origin). Windows may ask to allow Knucklebones through the firewall the first time you host. A room holds two players; a third connection is turned away.

## Save file

`knucklebones.json` is stored in your user config directory (`%AppData%\knucklebones\knucklebones.json` on Windows, `~/.config/knucklebones/knucklebones.json` on Linux, `~/Library/Application Support/knucklebones/knucklebones.json` on macOS). On first launch, a legacy `./knucklebones.json` in the working directory is automatically migrated there. It stores:

- Your name, selected theme, sound/colorblind/win-meter preferences, and last game settings (difficulty, rounds, house rules, turn timer, port, bind interface, last join address, Player 2 name)
- A record for each AI difficulty, online, and same keyboard: games, wins, losses, draws, best score, and series won and lost
- The Top 10 final scores, with opponent, their score, and date

Room codes are kept only in memory for the active session and are never persisted to `knucklebones.json`. The file is UTF-8 with BOM and is written safely via atomic replace after every finished round and settings/options change. Games in progress are not saved.

## Console notes

- Prefer **Cascadia Mono** (or another modern Unicode font) so `●`, `┃`, `╭─╮`, and related glyphs render cleanly. On Windows, Knucklebones selects Cascadia Mono automatically when present.
- On launch Knucklebones requests an **80×30** console. Windows Terminal and some hosts may ignore resize requests, so set the profile size manually if needed. Compact terminals (down to **52×20** / **60×22**, removing vertical slot gaps below height 27 and shifting the grid cleanly) remain playable while hiding wide side panels when space is tight.
- Colors use 24-bit RGB. Terminals limited to 256 colors show the nearest match.

## Build

From the `go/knucklebones` folder:

**All platforms (recommended)**

```powershell
pwsh -NoLogo -NoProfile -File ./build.ps1
```

**Quick local build**

```powershell
go build -o Knucklebones.exe
```

**Run without a binary**

```powershell
go run .
go run . -gui
```

**Tests**

```powershell
go test -v -count=1 -race .
```

### Binary output

| Platform | Path |
| -------- | ---- |
| Windows x86 | `bin/win/x86/Knucklebones.exe` |
| Windows x64 | `bin/win/x64/Knucklebones.exe` |
| Linux x86 | `bin/linux/x86/Knucklebones` |
| Linux amd64 | `bin/linux/amd64/Knucklebones` |

The graphical window and synthesized PCM sound (`ebiten/v2/audio`, used in both GUI and local desktop TUI) are built into Windows binaries; headless and SSH sessions bypass PCM initialization and fall back to terminal bell alerts (`Beep()`). Linux binaries cross-compiled from Windows use `CGO_ENABLED=0`, so they are terminal-only (`sound_stub.go` / `gui_stub.go`, using terminal bell for TUI sound alerts), and `-gui` reports that the window isn't available. To include the window and PCM audio on Linux, build there with cgo enabled and the X11/OpenGL/audio development packages installed (for example `libx11-dev libxrandr-dev libxcursor-dev libxinerama-dev libxi-dev libxxf86vm-dev libgl1-mesa-dev libasound2-dev`).

If `die.ico` and `knucklebones.rc` are present and `windres` is available, Windows builds embed the icon automatically. Pass `-upx` to compress binaries with [UPX](https://upx.github.io/).

## Credits

The original Knucklebones game in *Cult of the Lamb* was created by Massive Monster. This is a fan-made game and not an official implementation by Massive Monster.
