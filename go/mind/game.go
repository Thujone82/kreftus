package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const (
	codeLength = 4
	numColors  = 6
	maxTurns   = 12
)

// Colors: R=Red, G=Green, B=Blue, C=Cyan, M=Magenta, Y=Yellow (order RGBCMY)
const colors = "RGBCMY"

const peg = "⬤"

var colorNames = [numColors]string{"Red", "Green", "Blue", "Cyan", "Magenta", "Yellow"}

func colorIndex(c byte) int {
	for i := 0; i < len(colors); i++ {
		if colors[i] == c {
			return i
		}
	}
	return -1
}

type turnRecord struct {
	Guess   []byte
	Exact   int
	Partial int
}

type gameState struct {
	secret         []byte
	turns          []turnRecord
	current        []byte
	startTime      time.Time
	endTime        time.Time
	won            bool
	over           bool
	revealedPegs   int
	animFeedback   bool
	animShown      int
	statusMsg      string
	statusIsErr    bool
	statusUntil    time.Time
	celebrateStep  int
	overlayVisible bool
}

func newGameState(secret []byte) *gameState {
	return &gameState{
		secret:        secret,
		turns:         make([]turnRecord, 0, maxTurns),
		current:       make([]byte, 0, codeLength),
		startTime:     time.Now(),
		celebrateStep: -1,
	}
}

func (g *gameState) elapsed() time.Duration {
	if g.over && !g.endTime.IsZero() {
		return g.endTime.Sub(g.startTime)
	}
	return time.Since(g.startTime)
}

func generateSecret() []byte {
	secret := make([]byte, codeLength)
	for i := 0; i < codeLength; i++ {
		secret[i] = colors[rand.Intn(numColors)]
	}
	return secret
}

// formatPlaytime returns a short human-readable duration (e.g. "45s", "1m 23s").
func formatPlaytime(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", d/time.Second)
	}
	m := d / time.Minute
	s := (d % time.Minute) / time.Second
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm %ds", m, s)
}

// formatClock returns M:SS or MM:SS duration formatting for the TUI side panel.
func formatClock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d.Round(time.Second) / time.Second)
	m := total / 60
	s := total % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

// parseSetCode parses a 4-character string (R G B C M Y or 1–6, case-insensitive) into the secret code.
// Used with -set for one person to set the code for another to guess.
func parseSetCode(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if len(s) != codeLength {
		return nil, fmt.Errorf("mind: -set requires exactly %d characters (e.g. -set r22m), got %d", codeLength, len(s))
	}
	secret := make([]byte, codeLength)
	for i, r := range s {
		c, ok := keyToColor(r)
		if !ok {
			return nil, fmt.Errorf("mind: invalid character %q in -set (use R G B C M Y or 1–6)", r)
		}
		secret[i] = c
	}
	return secret, nil
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
