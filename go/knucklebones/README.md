# Knucklebones — Terminal Dice Duel

**Version:** 1.0 · **Author:** Kreft & Opus 5.5 · **Date:** 2026-11-10

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

On Windows the program stays a console app so the terminal UI works. A double-click briefly shows a console window, which closes as the game window opens. With `-tui` from a shortcut, a new console opens for the game.

The graphical window uses `die.png` for its icon and logo. It resizes freely and scales to fit, and every keyboard control below works in it too.

## Features

- **Start menu**: Play against an AI, Play against someone (same keyboard), Host Game, Join Game, How to play, Stats, Name, Theme, and Quit
- **AI opponents**: Ratau (Easy) places in a random open column. Flinky (Medium) takes the biggest immediate swing. Shrumy (Hard) averages your best reply over every roll, looks one step further, and knows when ending the game wins or loses it
- **Game settings**: AI difficulty, number of rounds (Best of 1, 3, 5, or Free play), and house rules. Your last choices are remembered
- **House rules** (from the dice.now guide): turn destruction off, roll two dice and place one, or play until both boards are full
- **Board**: your grid is at the bottom and your opponent's at the top, both filling from the middle outward. Ivory dice are singles, gold dice are pairs, and blue dice are triples. Column scores sit above and below the grids, and each player's total is in the left panel
- **Placement preview**: while you choose a column, a shaded die shows where yours lands, the opponent's dice it would destroy turn red, column scores show the change (`8→24`), and the panel shows `+gain · destroys N (−points)`
- **Online live aim**: your opponent sees the rolled die tumble, then watches your column and die selection update the same preview until you place. The board finalizes when the move lands
- **Rolling die**: each new die tumbles briefly in the roller's tray. Red pips mark the last die placed
- **Coin toss**: a random player starts each round
- **Series**: Best of 3 and 5 track rounds won with ●○ markers. After a round, **Continue game**. After the series, **Play another game** or **Go into free play**
- **Log**: the side panel shows the latest moves. **L** opens the full log of rolls, placements, destruction, timeouts, and results
- **History**: **H** lists each round's score and winner, plus who took the lead
- **Resign**: **R** with a confirmation. Vs AI and online it counts as a loss
- **Online play**: one player hosts on a TCP port (default 8647) and the lobby lists that computer's LAN addresses. The other joins with `host:port`
  - The host rolls every die and sends the full game state after each move
  - While a player aims, their column and die choice are shared so both screens show the placement preview before the move is confirmed
  - Optional turn timer (15, 30, or 60 seconds). A timed-out turn places the die in a random open column; three timeouts lose the match
  - A player who drops has 30 seconds to reconnect before forfeiting. Leaving mid-game counts as resigning
  - After a game, both players must pick the same option (play again, continue, free play). The result panel shows "Waiting for…" and "… wants to rematch!"
- **Stats**: games, wins, losses, draws, and best score for each AI, online, and same-keyboard play, plus best-of series won and lost. A Top 10 of final scores; making it is announced on the result panel
- **Themes**: dark and light (**T** on the menu)
- **How to play**: rules, the scoring table, house rules, strategy tips, and opponent notes

## Controls

| Context | Input | Action |
| ------- | ----- | ------ |
| Menu | ↑↓ or W/S | Move selection |
| Menu | Enter or Space | Choose |
| Menu | T (or ←→ on Theme) | Toggle dark / light theme |
| Menu | Esc | Quit |
| Game settings | ↑↓, Tab | Move between settings |
| Game settings | ←→ or Enter | Change the selected setting |
| Game settings | Typing, Backspace | Edit Player 2 name or Port |
| Game settings | Enter on Start game | Start |
| Game settings | Esc | Back to the menu |
| Name / Join | Typing, ←→, Home/End, click (GUI), Backspace/Delete, Ctrl+U | Edit the text |
| Name / Join | Enter | Confirm (Join connects) |
| Lobby | Enter | Retry after a failed connection |
| Lobby | Esc | Cancel |
| Your turn | ←→ or A/D | Choose a column |
| Your turn | 1–3 | Place in that column |
| Your turn | Enter or Space | Place in the chosen column |
| Your turn (two dice) | Tab, Q/E, or ↑↓ | Switch which die to place |
| In game | L | Log |
| In game | H | History |
| In game | R | Resign (asks first) |
| In game | Esc | Leave (asks first while a game is in progress) |
| Round over | ←→, Tab | Choose an option |
| Round over | Enter | Confirm the option |
| Round over | V | Hide or show the result panel to see the board |
| Log / History | ↑↓, PgUp/PgDn | Scroll |
| Log / History | Esc, Enter, or the same letter | Close |
| How to play | ↑↓, PgUp/PgDn, Space, Home | Scroll |
| Anywhere | Ctrl+C | Exit |

On the same keyboard, both players use the same keys on their own turn. Player 1 is at the bottom and player 2 at the top.

### Mouse (graphical window)

| Context | Input | Action |
| ------- | ----- | ------ |
| Menu | Hover, click | Select, choose |
| Game settings | Click ◀ ▶ or the row | Change the setting (click Player 2 or Port, then type) |
| Game settings | Click Start game or Back | Start, or back to the menu |
| Name / Join / Lobby | Click the buttons | Confirm, retry, cancel, or go back |
| Your turn | Hover a column | Preview the placement |
| Your turn | Click a column (either board) | Place the die there |
| Your turn (two dice) | Click a tray die | Choose which die to place |
| In game | Buttons under the board | Log, History, Resign, Leave, View board |
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

1. The host picks **Host Game**, sets the rounds, turn timer, port, and house rules, then **Start game**. The lobby lists this computer's addresses.
2. The other player picks **Join Game** and types one of those addresses, for example `192.168.1.20:8647`. A bare host name uses port 8647.
3. The game starts as soon as the guest connects. The host's copy runs the game and rolls the dice.

The game is played directly between the two computers, so both must reach each other (same network, VPN, or a forwarded port). Windows may ask to allow Knucklebones through the firewall the first time you host. A room holds two players; a third connection is turned away.

## Save file

`knucklebones.json` is created in the current working directory. It stores:

- Your name, theme, and last game settings (difficulty, rounds, house rules, turn timer, port, last join address, Player 2 name)
- A record for each AI difficulty, online, and same keyboard: games, wins, losses, draws, best score, and series won and lost
- The Top 10 final scores, with opponent, their score, and date

The file is UTF-8 with BOM and is written after every finished round and settings change. Games in progress are not saved.

## Console notes

- Prefer **Cascadia Mono** (or another modern Unicode font) so `●`, `╭─╮`, and related glyphs render cleanly. On Windows, Knucklebones selects Cascadia Mono automatically when present.
- On launch Knucklebones requests an **80×30** console. Windows Terminal and some hosts may ignore resize requests, so set the profile size manually if needed. The side panels hide on narrow windows.
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
go test -count=1 .
```

### Binary output

| Platform | Path |
| -------- | ---- |
| Windows x86 | `bin/win/x86/Knucklebones.exe` |
| Windows x64 | `bin/win/x64/Knucklebones.exe` |
| Linux x86 | `bin/linux/x86/Knucklebones` |
| Linux amd64 | `bin/linux/amd64/Knucklebones` |

The graphical window is built into Windows binaries. Linux binaries cross-compiled from Windows have no cgo, so they are terminal-only, and `-gui` reports that the window isn't available. To include the window on Linux, build there with cgo enabled and the X11/OpenGL development packages installed (for example `libx11-dev libxrandr-dev libxcursor-dev libxinerama-dev libxi-dev libxxf86vm-dev libgl1-mesa-dev`).

If `die.ico` and `knucklebones.rc` are present and `windres` is available, Windows builds embed the icon automatically. Pass `-upx` to compress binaries with [UPX](https://upx.github.io/).

## Credits

The original Knucklebones game in *Cult of the Lamb* was created by Massive Monster. This is a fan-made game and not an official implementation by Massive Monster.
