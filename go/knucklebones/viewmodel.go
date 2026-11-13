package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
)

const (
	effectDuration = 650 * time.Millisecond

	markNone   = 0
	markPair   = 2
	markTriple = 3
	markDoom   = 4
)

type winProbCache struct {
	key  [2]int
	turn int
	dice [2]int
	seat int
	prob float64
	ok   bool
}

type replayFrame struct {
	match Match
	last  *Placement
	step  int
	total int
	desc  string
}

func (g *game) cellMark(matches int, doom bool) int {
	if !g.save.Settings.Colorblind {
		return markNone
	}
	if doom {
		return markDoom
	}
	if matches >= 3 {
		return markTriple
	}
	if matches == 2 {
		return markPair
	}
	return markNone
}

func tuiMarkRune(mark int) rune {
	switch mark {
	case markDoom:
		return 'x'
	case markTriple:
		return '#'
	case markPair:
		return '='
	}
	return 0
}

func onOff(b bool) string {
	if b {
		return "On"
	}
	return "Off"
}

func (g *game) settingText(row int) (string, string) {
	st := g.save.Settings
	switch row {
	case rowDifficulty:
		return "AI difficulty", difficultyLabel[st.Difficulty] + " · " + aiName[st.Difficulty]
	case rowPlayer2:
		return "Player 2 name", st.Player2
	case rowRounds:
		return "Number of rounds", roundsLabel(st.BestOf)
	case rowTimer:
		if st.TurnSeconds == 0 {
			return "Turn timer", "Off"
		}
		return "Turn timer", fmt.Sprintf("%d seconds", st.TurnSeconds)
	case rowPort:
		if st.Port == 0 {
			return "Port", fmt.Sprint(defaultPort)
		}
		return "Port", fmt.Sprint(st.Port)
	case rowBindIP:
		return "Bind interface", bindIPLabel(st.BindIP)
	case rowRoomCode:
		if g.hostRoomCode == "" {
			return "Room code", "(optional)"
		}
		return "Room code", g.hostRoomCode
	case rowDestroy:
		return "Destroy matching dice", onOff(!st.Rules.NoDestroy)
	case rowDice:
		if st.Rules.TwoDice {
			return "Dice per turn", "Two (place one)"
		}
		return "Dice per turn", "One"
	case rowEnd:
		if st.Rules.FillBoth {
			return "Game ends when", "Both boards are full"
		}
		return "Game ends when", "One board is full"
	case rowTheme:
		return "Theme", themeLabel(g.save.Theme)
	case rowSound:
		return "Sound effects", onOff(st.Sound)
	case rowColorblind:
		return "Colorblind marks", onOff(st.Colorblind)
	case rowWinMeter:
		return "Win probability", onOff(st.WinMeter)
	case rowDone:
		return "Done", ""
	}
	return "Start game", ""
}

func lobbyErrorText(e string) string {
	switch e {
	case errRoomFull:
		return "That game already has two players."
	case errRoomNotFound:
		return "That match is no longer waiting in the lobby."
	case errOldProtocol:
		return "This game version is no longer compatible with the host."
	case errBadAuth:
		return "Incorrect room code."
	case errBadState:
		return "The host sent invalid game data."
	case errRejected:
		return "The host rejected the connection."
	}
	clean := cleanText(e, 120)
	if clean == "" {
		clean = "connection failed"
	}
	return "Could not connect: " + clean
}

// lobbyCanRetry reports whether retrying a failed connection with the same
// address and room code could succeed.
func lobbyCanRetry(errCode string) bool {
	return errCode != "" && errCode != errBadAuth && errCode != errOldProtocol && errCode != errRoomNotFound
}

func lobbyHostAddresses(bindIP string, port int) []string {
	if bindIP != "" {
		return []string{fmt.Sprintf("%s:%d", bindIP, port)}
	}
	var out []string
	for _, a := range localAddrs() {
		out = append(out, fmt.Sprintf("%s:%d", a, port))
	}
	return out
}

func (g *game) fairnessText(s *session) (string, tcell.Color) {
	if s == nil || s.Mode != modeOnline || g.ctl == nil || g.ctl.seat() != 1 {
		return "", g.pal.dim
	}
	switch g.ctl.fairness() {
	case fairVerified:
		return "Fair play verified", g.pal.win
	case fairFailed:
		return "(!) Host dice unverifiable", g.pal.lose
	default:
		if s.HostCommit != "" {
			return "Fair play commit locked", g.pal.dim
		}
	}
	return "", g.pal.dim
}

func thinkingText(now time.Time) string {
	dots := 1 + int(now.UnixMilli()/350%3)
	if dots < 1 {
		dots = 1
	}
	return "Thinking" + strings.Repeat(".", dots)
}

func effectProgress(at, now time.Time) float64 {
	if at.IsZero() {
		return -1
	}
	d := now.Sub(at)
	if d < 0 || d >= effectDuration {
		return -1
	}
	return float64(d) / float64(effectDuration)
}

func (g *game) bottomSeat() int {
	if g.ctl == nil {
		return 0
	}
	s := g.ctl.seat()
	if s < 0 {
		return 0
	}
	return s
}

func (g *game) winProb(s *session) float64 {
	if s == nil {
		return 0.5
	}
	seat := g.bottomSeat()
	key := [2]int{s.Series*1000 + s.Round, s.Match.Moves + 1}
	c := &g.winCache
	if !c.ok || c.key != key || c.turn != s.Match.Turn || c.dice != s.Match.Dice || c.seat != seat {
		c.key = key
		c.turn = s.Match.Turn
		c.dice = s.Match.Dice
		c.seat = seat
		c.prob = estimateWinProb(&s.Match, seat, 96)
		c.ok = true
	}
	return c.prob
}

func (g *game) winProbSummary(s *session) (int, string) {
	p := g.winProb(s)
	pct := int(p*100 + 0.5)
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	name := s.Names[g.bottomSeat()]
	if g.ctl != nil && g.isYou(s, g.bottomSeat()) {
		name = "You"
	}
	return pct, fmt.Sprintf("%s %d%%", name, pct)
}

func canReplay(s *session) bool {
	return s != nil && s.Phase != phasePlay && len(s.RoundMoves) > 0
}

func buildReplay(s *session) []replayFrame {
	if s == nil || len(s.RoundMoves) == 0 {
		return nil
	}
	total := len(s.RoundMoves)
	frames := make([]replayFrame, 0, total+1)
	m := newMatch(s.Rules, s.RoundFirst)
	m.Dice = s.RoundMoves[0].Dice
	frames = append(frames, replayFrame{
		match: m,
		step:  0,
		total: total,
		desc:  fmt.Sprintf("Round %d start · %s goes first", s.Round, s.Names[s.RoundFirst]),
	})
	for i, rm := range s.RoundMoves {
		m.Turn = rm.Player
		m.Dice = rm.Dice
		pl, err := m.Place(Move{Col: rm.Col, Die: rm.Die})
		if err != nil {
			break
		}
		if i+1 < total {
			m.Dice = s.RoundMoves[i+1].Dice
		} else {
			m.Dice = [2]int{}
		}
		desc := fmt.Sprintf("%s placed %d in col %d (+%d)", s.Names[rm.Player], pl.Die, rm.Col+1, pl.Gain)
		if pl.Destroyed > 0 {
			desc += fmt.Sprintf(", −%d", pl.Lost)
		}
		plCopy := pl
		frames = append(frames, replayFrame{
			match: m,
			last:  &plCopy,
			step:  i + 1,
			total: total,
			desc:  desc,
		})
	}
	return frames
}
