package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRenderAllThemes(t *testing.T) {
	now := time.Now()
	for _, id := range themeOrder {
		th := themeByID(id)
		for _, labels := range []bool{false, true} {
			for _, numbers := range []bool{false, true} {
				st := defaultSettings()
				st.Theme = id
				st.Labels = labels
				st.LabelNumbers = numbers
				st.Stats = stats{Played: 8, Won: 5, Streak: 2, BestStreak: 3, BestTurns: 4, BestTimeSec: 27}

				g := newGameState([]byte("MCGB"))
				g.turns = []turnRecord{
					{Guess: []byte("RRGG"), Exact: 1, Partial: 0},
					{Guess: []byte("MCGB"), Exact: 4, Partial: 0},
				}
				g.current = []byte("RY")

				c := renderBoard(g, st, th, now)
				if c.w != boardW || c.h > 22 {
					t.Errorf("theme=%s labels=%v numbers=%v board size = %dx%d, want width=%d height<=22", id, labels, numbers, c.w, c.h, boardW)
				}

				out := serializeCanvas(c, 80, 24, th.BG, true)
				wantBG := fmt.Sprintf("\033[48;2;%d;%d;%dm", th.BG.R, th.BG.G, th.BG.B)
				if !strings.Contains(out, wantBG) {
					t.Errorf("theme=%s labels=%v numbers=%v flushed output missing BG sequence %q", id, labels, numbers, wantBG)
				}

				// Overlay variant.
				g.won = true
				g.over = true
				g.revealedPegs = codeLength
				g.overlayVisible = true
				cOver := renderBoard(g, st, th, now)
				if cOver.w != boardW || cOver.h > 22 {
					t.Errorf("theme=%s overlay size = %dx%d", id, cOver.w, cOver.h)
				}

				// Title screen.
				cTitle := renderTitleScreen(st, th, 3)
				if cTitle.w != boardW || cTitle.h > 22 {
					t.Errorf("theme=%s title size = %dx%d", id, cTitle.w, cTitle.h)
				}

				// Too-small screen.
				cSmall := renderTooSmall(40, 12, th)
				if cSmall.w != 40 || cSmall.h != 12 {
					t.Errorf("theme=%s too-small size = %dx%d", id, cSmall.w, cSmall.h)
				}
			}
		}
	}
}

func TestLabelNumbersRender(t *testing.T) {
	th := themeByID("dark")
	now := time.Now()
	g := newGameState([]byte("MCGB"))
	g.turns = []turnRecord{
		{Guess: []byte("RGBC"), Exact: 1, Partial: 1},
	}

	stLetters := defaultSettings()
	stLetters.Labels = true
	stLetters.LabelNumbers = false
	cL := renderBoard(g, stLetters, th, now)
	wantLetters := []rune{'R', 'G', 'B', 'C'}
	for i, w := range wantLetters {
		if got := cL.cells[5][8+i*4].ch; got != w {
			t.Errorf("letters mode col %d = %q, want %q", 8+i*4, got, w)
		}
	}

	stNums := defaultSettings()
	stNums.Labels = true
	stNums.LabelNumbers = true
	cN := renderBoard(g, stNums, th, now)
	wantNums := []rune{'1', '2', '3', '4'}
	for i, w := range wantNums {
		if got := cN.cells[5][8+i*4].ch; got != w {
			t.Errorf("numbers mode col %d = %q, want %q", 8+i*4, got, w)
		}
	}

	// Verify footer width fits within 66 columns even for longest theme ("High Contrast").
	contrast := themeByID("contrast")
	stContrast := defaultSettings()
	stContrast.Theme = "contrast"
	stContrast.Sound = false
	stContrast.LabelNumbers = true
	cFooter := renderBoard(g, stContrast, contrast, now)
	if cFooter.w != boardW {
		t.Errorf("contrast board width = %d, want %d", cFooter.w, boardW)
	}
	var footerRunes []rune
	for _, cl := range cFooter.cells[20] {
		footerRunes = append(footerRunes, cl.ch)
	}
	footerText := strings.TrimSpace(string(footerRunes))
	if !strings.Contains(footerText, "L/Tab Labels 123") || !strings.Contains(footerText, "Esc Quit") {
		t.Errorf("contrast footer truncated or unexpected: %q", footerText)
	}
}
