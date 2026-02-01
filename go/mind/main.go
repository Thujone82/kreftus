package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"

	"golang.org/x/term"
)

const (
	codeLength = 4
	numColors  = 6
	maxTurns   = 12
)

// Colors: R=Red, G=Green, B=Blue, C=Cyan, M=Magenta, Y=Yellow (order RGBCMY)
const colors = "RGBCMY"

const peg = "⬤"

// ANSI color codes
const (
	ansiReset   = "\033[0m"
	ansiRed     = "\033[31m"
	ansiGreen   = "\033[32m"
	ansiYellow  = "\033[33m"
	ansiBlue    = "\033[34m"
	ansiMagenta = "\033[35m"
	ansiCyan    = "\033[36m"
)

var ansiByColor = map[byte]string{
	'R': ansiRed,
	'G': ansiGreen,
	'B': ansiBlue,
	'C': ansiCyan,
	'M': ansiMagenta,
	'Y': ansiYellow,
}

// termRestoreOnce and termRestoreFunc allow Ctrl+C and ESC to restore the terminal before exiting.
var (
	termRestoreOnce sync.Once
	termRestoreFunc func()
)

func main() {
	// Allow Ctrl+C to exit cleanly (restore terminal if in raw mode)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		if termRestoreFunc != nil {
			termRestoreOnce.Do(termRestoreFunc)
		}
		os.Exit(0)
	}()

	reader := bufio.NewReader(os.Stdin)
	showStartScreen(reader)

	secret := generateSecret()
	printGameInstructions()

	for turn := 1; turn <= maxTurns; turn++ {
		guess, err := readGuess(reader, turn)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", err)
			os.Exit(1)
		}

		fmt.Println() // newline after "Turn NN/12: ⬤⬤⬤⬤"
		rightPlace, rightColor := score(secret, guess)
		fmt.Print("  Feedback: ")
		printFeedback(rightPlace, rightColor)
		fmt.Println()

		if rightPlace == codeLength {
			fmt.Println("\nYou win! You cracked the code.")
			return
		}

		if turn == maxTurns {
			fmt.Print("\nOut of turns. The secret was: ")
			printColoredPegs(secret)
			fmt.Println()
			return
		}
	}
}

func showStartScreen(reader *bufio.Reader) {
	fmt.Println()
	fmt.Println("  ╔═══════════════════════════════╗")
	fmt.Println("  ║      M A S T E R M I N D      ║")
	fmt.Println("  ╚═══════════════════════════════╝")
	fmt.Println()
	fmt.Println("  Guess the secret code of 4 pegs.")
	fmt.Println("  Colors: R=" + ansiRed + "Red" + ansiReset + ", G=" + ansiGreen + "Green" + ansiReset + ", B=" + ansiBlue + "Blue" + ansiReset)
	fmt.Println("          C=" + ansiCyan + "Cyan" + ansiReset + ", M=" + ansiMagenta + "Magenta" + ansiReset + ", Y=" + ansiYellow + "Yellow" + ansiReset)
	fmt.Println("  Enter 4 letters (e.g. RGBC). You have 12 turns.")
	fmt.Println()
	fmt.Println("  Feedback: " + ansiGreen + peg + ansiReset + " = right color, right slot")
	fmt.Println("            " + ansiYellow + peg + ansiReset + " = right color, wrong slot")
	fmt.Println()
	fmt.Print("  Press ENTER to START ")
	_, _ = reader.ReadString('\n')
	fmt.Println()
}

func printGameInstructions() {
	fmt.Println("Enter a 4-peg guess each turn:")
	fmt.Print("Colors:  ")
	printColoredColorLetters()
	fmt.Println()
	fmt.Print("Numbers: ")
	printColoredNumbers()
	fmt.Println()
	fmt.Println()
}

// printColoredColorLetters prints "R G B C M Y" with each letter in its color.
func printColoredColorLetters() {
	for i := 0; i < len(colors); i++ {
		if i > 0 {
			fmt.Print(" ")
		}
		c := colors[i]
		if ac, ok := ansiByColor[c]; ok {
			fmt.Print(ac + string(c) + ansiReset)
		}
	}
}

// printColoredNumbers prints "1 2 3 4 5 6" with each number in the color that matches R G B C M Y (1=red, 5=magenta, 6=yellow).
func printColoredNumbers() {
	for i := 0; i < len(colors); i++ {
		if i > 0 {
			fmt.Print(" ")
		}
		c := colors[i]
		ac := ansiByColor[c]
		fmt.Print(ac + string(rune('1'+i)) + ansiReset)
	}
}



func printColoredPegs(code []byte) {
	fmt.Print(coloredPegsString(code))
}

// coloredPegsString returns a string of colored pegs for the given code (for redrawing the input line).
func coloredPegsString(code []byte) string {
	var b strings.Builder
	for _, c := range code {
		if ac, ok := ansiByColor[c]; ok {
			b.WriteString(ac)
			b.WriteString(peg)
			b.WriteString(ansiReset)
		}
	}
	return b.String()
}

func printFeedback(rightPlace, rightColor int) {
	for i := 0; i < rightPlace; i++ {
		fmt.Print(ansiGreen + peg + ansiReset)
	}
	for i := 0; i < rightColor; i++ {
		fmt.Print(ansiYellow + peg + ansiReset)
	}
}

func generateSecret() []byte {
	secret := make([]byte, codeLength)
	for i := 0; i < codeLength; i++ {
		secret[i] = colors[rand.Intn(numColors)]
	}
	return secret
}

// keyToColor maps input runes to color bytes: r,g,b,c,m,y (case-insensitive) and 1–6 (1=R, 2=G, 3=B, 4=C, 5=M, 6=Y).
func keyToColor(r rune) (byte, bool) {
	switch r {
	case 'r', 'R':
		return 'R', true
	case 'g', 'G':
		return 'G', true
	case 'b', 'B':
		return 'B', true
	case 'c', 'C':
		return 'C', true
	case 'm', 'M':
		return 'M', true
	case 'y', 'Y':
		return 'Y', true
	case '1':
		return 'R', true
	case '2':
		return 'G', true
	case '3':
		return 'B', true
	case '4':
		return 'C', true
	case '5':
		return 'M', true
	case '6':
		return 'Y', true
	}
	return 0, false
}

func readGuess(reader *bufio.Reader, turn int) ([]byte, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return readGuessLine(reader, turn)
	}
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return readGuessLine(reader, turn)
	}
	termRestoreFunc = func() { _ = term.Restore(fd, oldState) }
	defer func() { _ = term.Restore(fd, oldState) }()

	turnStr := fmt.Sprintf("%02d", turn)
	prompt := fmt.Sprintf("Turn %s/%d: ", turnStr, maxTurns)

	redrawLine := func(buf []byte) {
		fmt.Print("\r\033[K" + prompt + coloredPegsString(buf))
	}

	buf := make([]byte, 0, codeLength)
	redrawLine(buf)
	for {
		r, _, err := reader.ReadRune()
		if err != nil {
			return nil, err
		}
		if c, ok := keyToColor(r); ok {
			if len(buf) < codeLength {
				buf = append(buf, c)
				redrawLine(buf)
			}
			continue
		}
		if r == '\b' || r == 127 { // Backspace — remove one peg, allow backspace down to empty buffer
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				redrawLine(buf)
			}
			continue
		}
		if r == '\n' || r == '\r' {
			if len(buf) == codeLength {
				return buf, nil
			}
			continue
		}
		if r == 27 { // ESC - exit
			termRestoreOnce.Do(termRestoreFunc)
			os.Exit(0)
		}
	}
}

// readGuessLine is the fallback when raw mode is not available (e.g. not a TTY).
func readGuessLine(reader *bufio.Reader, turn int) ([]byte, error) {
	for {
		turnStr := fmt.Sprintf("%02d", turn)
		fmt.Printf("Turn %s/%d: ", turnStr, maxTurns)
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(strings.ToUpper(line))
		// Allow number aliases in line mode
		var decoded strings.Builder
		for _, r := range line {
			if c, ok := keyToColor(r); ok {
				decoded.WriteByte(c)
			}
		}
		line = decoded.String()
		if len(line) != codeLength {
			fmt.Printf("  (enter 4 pegs: R G B C M Y or 1–6)\n")
			continue
		}
		return []byte(line), nil
	}
}

// score returns (rightPlace, rightColor): rightPlace = correct color and position, rightColor = correct color wrong position.
func score(secret, guess []byte) (rightPlace, rightColor int) {
	usedSecret := make([]bool, codeLength)
	usedGuess := make([]bool, codeLength)

	for i := 0; i < codeLength; i++ {
		if secret[i] == guess[i] {
			rightPlace++
			usedSecret[i] = true
			usedGuess[i] = true
		}
	}

	secretCount := make(map[byte]int)
	guessCount := make(map[byte]int)
	for i := 0; i < codeLength; i++ {
		if !usedSecret[i] {
			secretCount[secret[i]]++
		}
		if !usedGuess[i] {
			guessCount[guess[i]]++
		}
	}
	for c, n := range secretCount {
		g := guessCount[c]
		if g < n {
			rightColor += g
		} else {
			rightColor += n
		}
	}
	return rightPlace, rightColor
}
