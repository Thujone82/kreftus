package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// ANSI color codes for plain non-TTY fallback mode.
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

func runPlain(reader *bufio.Reader, setCode string) {
	fmt.Print("\033]0;Mastermind - Crack the code!\007")
	showStartScreenPlain(reader)

	var secret []byte
	if setCode != "" {
		var err error
		secret, err = parseSetCode(setCode)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		secret = generateSecret()
	}
	printGameInstructionsPlain()

	startTime := time.Now()
	for turn := 1; turn <= maxTurns; turn++ {
		guess, err := readGuessLine(reader, turn)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", err)
			os.Exit(1)
		}

		rightPlace, rightColor := score(secret, guess)
		fmt.Print("  Feedback: ")
		printFeedbackPlain(rightPlace, rightColor)
		fmt.Println()

		if rightPlace == codeLength {
			fmt.Printf("\nYou win! You cracked the code in %s.\n", formatPlaytime(time.Since(startTime)))
			_, _ = reader.ReadString('\n')
			return
		}

		if turn == maxTurns {
			fmt.Print("\nOut of turns. The secret was: ")
			printColoredPegsPlain(secret)
			fmt.Printf(" (%s)\n", formatPlaytime(time.Since(startTime)))
			_, _ = reader.ReadString('\n')
			return
		}
	}
}

func showStartScreenPlain(reader *bufio.Reader) {
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
	fmt.Print("        Press " + ansiGreen + "ENTER" + ansiReset + " to START ")
	_, _ = reader.ReadString('\n')
	fmt.Println()
}

func printGameInstructionsPlain() {
	fmt.Println("Enter a 4-peg guess each turn:")
	fmt.Print("Colors:  ")
	printColoredColorLettersPlain()
	fmt.Println()
	fmt.Print("Numbers: ")
	printColoredNumbersPlain()
	fmt.Println()
	fmt.Println()
}

func printColoredColorLettersPlain() {
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

func printColoredNumbersPlain() {
	for i := 0; i < len(colors); i++ {
		if i > 0 {
			fmt.Print(" ")
		}
		c := colors[i]
		ac := ansiByColor[c]
		fmt.Print(ac + string(rune('1'+i)) + ansiReset)
	}
}

func printColoredPegsPlain(code []byte) {
	fmt.Print(coloredPegsStringPlain(code))
}

func coloredPegsStringPlain(code []byte) string {
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

func printFeedbackPlain(rightPlace, rightColor int) {
	for i := 0; i < rightPlace; i++ {
		fmt.Print(ansiGreen + peg + ansiReset)
	}
	for i := 0; i < rightColor; i++ {
		fmt.Print(ansiYellow + peg + ansiReset)
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
