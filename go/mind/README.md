# mind

Terminal-based **Mastermind** code-breaking game written in Go. Guess the secret 4-peg code in 12 turns. Pegs are shown as colored **⬤**; feedback uses green ⬤ (right color, right slot) and yellow ⬤ (right color, wrong slot).

## Requirements

- Go 1.21 or later (see `go.mod`).

## Game rules

- **Secret**: The program picks a code of 4 pegs. Each peg is one of 6 colors: **R**ed, **G**reen, **B**lue, **C**yan, **M**agenta, **Y**ellow (order RGBCMY). Colors may repeat.
- **Turns**: You have up to **12** turns to guess the code.
- **Feedback** (after each guess, shown as colored pegs):
  - **Green ⬤**: Correct color in the correct position.
  - **Yellow ⬤**: Correct color in the wrong position (each peg in the secret is counted at most once).
- **Win**: Your guess matches the secret exactly (4 green pegs).
- **Lose**: You run out of turns; the secret is revealed as colored pegs.

## Start screen

When you run the game you see:

- **MASTERMIND** title (ASCII art)
- Brief instructions (colors, input format, turn limit)
- Explanation of feedback (green ⬤ vs yellow ⬤)
- **Press ENTER to START** — the game begins after you press Enter.

## How to run

From the `go/mind` directory:

```bash
go run .
```

Or build and run the binary:

```bash
go build -o mind .
./mind        # Linux/macOS
mind.exe      # Windows
```

## Input format

- Each turn shows **Turn 01/12:** through **Turn 12/12:** (turn number zero-padded for alignment).
- Type **4 pegs** key-by-key: each key shows a colored **⬤** immediately (no letters echoed).
- **Keys**: **R** **G** **B** **C** **M** **Y** (case-insensitive) or number aliases **1** **2** **3** **4** **5** **6** (1=Red, 2=Green, 3=Blue, 4=Cyan, 5=Magenta, 6=Yellow).
- **Backspace**: removes the last peg.
- **Enter**: submits the guess only after 4 valid pegs have been entered.
- During gameplay, the **Colors** line shows **R G B C M Y** with each letter in its color.

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

| File        | Description                          |
| ----------- | ------------------------------------ |
| `main.go`   | Game logic, I/O, scoring, main loop  |
| `go.mod`    | Go module definition                 |
| `build.ps1` | Cross-build script (Windows/Linux)   |
| `README.md` | This documentation                   |

## License

See the repository [LICENSE](../../LICENSE).
