package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
)

// Text shared by the terminal and window front ends.

type coloredLine struct {
	text  string
	color tcell.Color
}

var setupTitles = map[string]string{modeAI: "Play against an AI", modeLocal: "Play against someone", setupHost: "Host Game", setupOptions: "Options"}

var difficultyHints = map[string]string{
	diffEasy:   "Ratau drops dice anywhere.",
	diffMedium: "Flinky grabs the biggest swing each turn.",
	diffHard:   "Shrumy weighs every roll you could get back.",
}

const (
	creditLine1 = "Knucklebones is from Cult of the Lamb by Massive Monster."
	creditLine2 = "This fan-made version is not an official implementation."
)

func (g *game) menuLabels() [menuCount]string {
	return [menuCount]string{
		menuAI:      "Play against an AI",
		menuLocal:   "Play against someone (same keyboard)",
		menuHost:    "Host Game",
		menuJoin:    "Join Game",
		menuGuide:   "How to play",
		menuStats:   "Stats",
		menuName:    "Name: " + g.save.Name,
		menuOptions: "Options · " + themeLabel(g.save.Theme),
		menuQuit:    "Quit",
	}
}

func (g *game) headerText(s *session, now time.Time) (string, tcell.Color) {
	if p, left := g.ctl.disconnectLeft(now); p >= 0 {
		secs := int(left.Seconds() + 0.999)
		if g.isYou(s, 1-p) {
			return fmt.Sprintf("%s disconnected. They have %d seconds to reconnect.", s.Names[p], secs), g.pal.lose
		}
		return fmt.Sprintf("Connection lost. Reconnecting… %ds", secs), g.pal.lose
	}
	if s.Phase == phasePlay {
		p := s.Match.Turn
		if g.isYou(s, p) {
			return "Your turn", g.pal.accent
		}
		return s.Names[p] + "'s turn", g.pal.accent
	}
	if s.Phase == phaseGameOver {
		return "Game over", g.pal.fg
	}
	return "Round over", g.pal.fg
}

func (g *game) panelName(s *session, p int) string {
	name := s.Names[p]
	if g.isYou(s, p) {
		name += " (you)"
	}
	if !s.Connected[p] {
		name += " ⚠"
	}
	return name
}

func (g *game) winsText(s *session, p int) string {
	if !s.freePlay() && s.BestOf > 1 {
		need := s.BestOf/2 + 1
		return "Wins " + strings.Repeat("●", s.Wins[p]) + strings.Repeat("○", max(0, need-s.Wins[p]))
	}
	if s.Round > 1 || len(s.History) > 0 {
		return fmt.Sprintf("Wins %d", s.Wins[p])
	}
	return ""
}

func gainText(pl Placement) string {
	t := fmt.Sprintf("+%d", pl.Gain)
	if pl.Destroyed > 0 {
		t += fmt.Sprintf(" · destroys %d (−%d)", pl.Destroyed, pl.Lost)
	}
	return t
}

// resultStatus is the line under the score on the result panel.
func (g *game) resultStatus(s *session) (string, tcell.Color) {
	status, c := g.recordNote, g.pal.accent
	if s.Mode != modeOnline {
		return status, c
	}
	me := g.ctl.seat()
	opp := 1 - me
	switch {
	case g.ctl.opponentGone():
		return "Your opponent has left the room.", g.pal.lose
	case s.Votes[me] != "" && s.Votes[opp] != s.Votes[me]:
		return "Waiting for " + s.Names[opp] + "...", g.pal.dim
	case s.Votes[opp] == voteRematch:
		return s.Names[opp] + " wants to rematch!", c
	case s.Votes[opp] == voteContinue:
		return s.Names[opp] + " wants to continue!", c
	case s.Votes[opp] == voteFree:
		return s.Names[opp] + " wants to go into free play!", c
	}
	return status, c
}

func (g *game) resignWarning(s *session) string {
	if s.Mode == modeLocal {
		return s.Names[s.Match.Turn] + " gives the game to " + s.Names[1-s.Match.Turn] + "."
	}
	return "Resigning ends the game immediately and counts as a loss."
}

func leaveWarning(s *session) string {
	if s.Mode == modeOnline {
		return "Leaving counts as resigning."
	}
	return "This game is not saved."
}

func (g *game) logColor(e logEntry) tcell.Color {
	switch e.Kind {
	case logFirst:
		return g.pal.accent
	case logRound, logForfeit:
		return g.pal.win
	case logTimeout:
		return g.pal.lose
	}
	return g.pal.fg
}

func (g *game) historyLines(s *session) []coloredLine {
	var lines []coloredLine
	add := func(t string, c tcell.Color) { lines = append(lines, coloredLine{t, c}) }
	add(fmt.Sprintf("Round %d of %s", s.Round, s.roundsLabel()), g.pal.accent)
	add("", g.pal.fg)
	if len(s.History) == 0 {
		add("The first round isn't finished yet!", g.pal.dim)
	}
	for _, e := range s.History {
		add(fmt.Sprintf("Round %d   %s %d – %d %s", e.Round, s.Names[0], e.Scores[0], e.Scores[1], s.Names[1]), g.pal.fg)
		switch {
		case e.Winner < 0:
			add("    This is a tie!", g.pal.dim)
		case g.isYou(s, e.Winner):
			add("    You won!", g.pal.win)
		default:
			add("    "+s.Names[e.Winner]+" won!", g.pal.dim)
		}
		switch {
		case e.Lead < 0:
		case g.isYou(s, e.Lead):
			add("    You are taking the lead!", g.pal.accent)
		default:
			add("    "+s.Names[e.Lead]+" is taking the lead!", g.pal.accent)
		}
	}
	if s.Ties > 0 {
		add("", g.pal.fg)
		add(fmt.Sprintf("Ties: %d", s.Ties), g.pal.dim)
	}
	return lines
}

var statsHeader = [7]string{"", "Games", "Wins", "Losses", "Draws", "Best", "Series"}

var recordLabels = map[string]string{
	"ai-" + diffEasy:   "vs Ratau (Easy)",
	"ai-" + diffMedium: "vs Flinky (Medium)",
	"ai-" + diffHard:   "vs Shrumy (Hard)",
	recordOnline:       "Online",
	recordLocal:        "Same keyboard",
}

type statsRow struct {
	cells [7]string
	empty bool
}

func (g *game) statsRows() []statsRow {
	var rows []statsRow
	for _, k := range recordKeys {
		r := g.save.Records[k]
		if r == nil {
			r = &record{}
		}
		row := statsRow{empty: r.Games == 0}
		if k == recordLocal {
			row.cells = [7]string{recordLabels[k], fmt.Sprint(r.Games), "–", "–", fmt.Sprint(r.Draws), fmt.Sprint(r.Best), "–"}
		} else {
			series := "–"
			if r.SeriesWins+r.SeriesLosses > 0 {
				series = fmt.Sprintf("%d–%d", r.SeriesWins, r.SeriesLosses)
			}
			row.cells = [7]string{recordLabels[k], fmt.Sprint(r.Games), fmt.Sprint(r.Wins), fmt.Sprint(r.Losses), fmt.Sprint(r.Draws), fmt.Sprint(r.Best), series}
		}
		rows = append(rows, row)
	}
	return rows
}

// guideLines flattens guideText, wrapping body text with wrapFn.
func (g *game) guideLines(wrapFn func(string) []string) ([]string, []int) {
	var lines []string
	var kinds []int
	for i, gl := range guideText {
		switch gl.kind {
		case 1:
			if i > 0 {
				lines, kinds = append(lines, ""), append(kinds, 0)
			}
			lines, kinds = append(lines, gl.text), append(kinds, 1)
		case 2:
			lines, kinds = append(lines, gl.text), append(kinds, 2)
		default:
			for j, l := range wrapFn(gl.text) {
				if j > 0 && (strings.HasPrefix(gl.text, "•") || (len(gl.text) > 1 && gl.text[1] == '.')) {
					l = "  " + l
				}
				lines, kinds = append(lines, l), append(kinds, 0)
			}
		}
	}
	return lines, kinds
}
