package main

import (
	"math/rand/v2"
	"strconv"
)

const (
	modeAI     = "ai"
	modeLocal  = "local"
	modeOnline = "online"

	phasePlay      = "play"
	phaseRoundOver = "round"
	phaseGameOver  = "game"

	voteContinue = "continue"
	voteRematch  = "rematch"
	voteFree     = "free"

	reasonResign     = "resignation"
	reasonDisconnect = "disconnect"
	reasonTimeout    = "timeout"

	logFirst   = "first"
	logPlace   = "place"
	logTimeout = "timeout"
	logRound   = "round"
	logForfeit = "forfeit"

	maxTimeouts = 3
	maxLog      = 300
)

type logEntry struct {
	Round     int    `json:"round"`
	Kind      string `json:"kind"`
	Player    int    `json:"player"`
	Die       int    `json:"die,omitempty"`
	Col       int    `json:"col,omitempty"`
	Destroyed int    `json:"destroyed,omitempty"`
	Gain      int    `json:"gain,omitempty"`
	Lost      int    `json:"lost,omitempty"`
	Count     int    `json:"count,omitempty"`
	Scores    [2]int `json:"scores"`
	Reason    string `json:"reason,omitempty"`
}

type historyEntry struct {
	Round  int    `json:"round"`
	Scores [2]int `json:"scores"`
	Winner int    `json:"winner"` // -1 tie
	Lead   int    `json:"lead"`   // player who took the series lead with this round, or -1
}

type forfeit struct {
	Reason string `json:"reason"`
	Loser  int    `json:"loser"`
}

// session is everything both screens need to draw a game. Offline modes
// own it directly; online the host owns it and sends copies to the guest.
type session struct {
	Mode       string         `json:"mode"`
	Names      [2]string      `json:"names"`
	Difficulty string         `json:"difficulty,omitempty"`
	BestOf     int            `json:"bestOf"` // 0 = free play
	FreePlay   bool           `json:"freePlay,omitempty"`
	Rules      Rules          `json:"rules"`
	Match      Match          `json:"match"`
	Series     int            `json:"series"`
	Round      int            `json:"round"`
	Wins       [2]int         `json:"wins"`
	Ties       int            `json:"ties"`
	Phase      string         `json:"phase"`
	Log        []logEntry     `json:"log"`
	History    []historyEntry `json:"history"`
	Timeouts   [2]int         `json:"timeouts"`
	Votes      [2]string      `json:"votes"`
	Forfeit    *forfeit       `json:"forfeit,omitempty"`
	Last       *Placement     `json:"last,omitempty"`
	Connected  [2]bool        `json:"connected"`
	TurnLimit  int            `json:"turnLimit,omitempty"` // seconds, online only
	TurnLeftMs int64          `json:"turnLeftMs,omitempty"`
	WaitLeftMs int64          `json:"waitLeftMs,omitempty"` // reconnect window
}

func newSession(mode string, names [2]string, difficulty string, bestOf int, rules Rules, rng *rand.Rand) *session {
	s := &session{
		Mode:       mode,
		Names:      names,
		Difficulty: difficulty,
		BestOf:     bestOf,
		Rules:      rules,
		Connected:  [2]bool{true, true},
	}
	s.startSeries(rng)
	return s
}

func (s *session) startSeries(rng *rand.Rand) {
	s.Series++
	s.Round = 0
	s.Wins = [2]int{}
	s.Ties = 0
	s.History = nil
	s.Timeouts = [2]int{}
	s.Forfeit = nil
	s.FreePlay = false
	s.startRound(rng)
}

func (s *session) startRound(rng *rand.Rand) {
	s.Round++
	first := rng.IntN(2)
	s.Match = newMatch(s.Rules, first)
	s.Match.Roll(rng)
	s.Phase = phasePlay
	s.Votes = [2]string{}
	s.Last = nil
	s.addLog(logEntry{Kind: logFirst, Player: first})
}

func (s *session) addLog(e logEntry) {
	e.Round = s.Round
	e.Scores = s.Match.Scores()
	s.Log = append(s.Log, e)
	if len(s.Log) > maxLog {
		s.Log = append([]logEntry(nil), s.Log[len(s.Log)-maxLog:]...)
	}
}

func (s *session) place(p int, mv Move, rng *rand.Rand) error {
	if s.Phase != phasePlay {
		return errNotPlaying
	}
	if p != s.Match.Turn {
		return errNotYourTurn
	}
	pl, err := s.Match.Place(mv)
	if err != nil {
		return err
	}
	s.Last = &pl
	s.addLog(logEntry{Kind: logPlace, Player: p, Die: pl.Die, Col: pl.Col, Destroyed: pl.Destroyed, Gain: pl.Gain, Lost: pl.Lost})
	if s.Match.Over {
		s.finishRound()
	} else {
		s.Match.Roll(rng)
	}
	return nil
}

func (s *session) finishRound() {
	prev := s.leader()
	w := s.Match.Winner()
	if w >= 0 {
		s.Wins[w]++
	} else {
		s.Ties++
	}
	lead := -1
	if l := s.leader(); l >= 0 && l != prev {
		lead = l
	}
	s.History = append(s.History, historyEntry{Round: s.Round, Scores: s.Match.Scores(), Winner: w, Lead: lead})
	s.addLog(logEntry{Kind: logRound, Player: w})
	if s.seriesDone() {
		s.Phase = phaseGameOver
	} else {
		s.Phase = phaseRoundOver
	}
}

// leader is the player with more round wins, or -1 when level.
func (s *session) leader() int {
	switch {
	case s.Wins[0] > s.Wins[1]:
		return 0
	case s.Wins[1] > s.Wins[0]:
		return 1
	}
	return -1
}

func (s *session) freePlay() bool { return s.BestOf <= 0 || s.FreePlay }

func (s *session) seriesDone() bool {
	if s.freePlay() {
		return false
	}
	need := s.BestOf/2 + 1
	return s.Wins[0] >= need || s.Wins[1] >= need
}

// seriesWinner is the winner of a finished series (or forfeit), -1 for none.
func (s *session) seriesWinner() int {
	if s.Forfeit != nil {
		return 1 - s.Forfeit.Loser
	}
	if s.Phase != phaseGameOver {
		return -1
	}
	return s.leader()
}

func (s *session) forfeitBy(loser int, reason string) {
	if s.Forfeit != nil || s.Phase == phaseGameOver {
		return
	}
	s.Forfeit = &forfeit{Reason: reason, Loser: loser}
	s.Match.Over = true
	s.Match.Dice = [2]int{}
	s.Phase = phaseGameOver
	s.Votes = [2]string{}
	s.addLog(logEntry{Kind: logForfeit, Player: loser, Reason: reason})
}

// options lists the votes allowed in the current phase.
func (s *session) options() []string {
	switch s.Phase {
	case phaseRoundOver:
		return []string{voteContinue}
	case phaseGameOver:
		if s.Forfeit != nil && s.Forfeit.Reason == reasonDisconnect {
			return nil
		}
		opts := []string{voteRematch}
		if s.Forfeit == nil && !s.freePlay() {
			opts = append(opts, voteFree)
		}
		return opts
	}
	return nil
}

// vote records p's choice. Online both players must pick the same option;
// offline the vote applies at once. It reports whether the vote applied.
func (s *session) vote(p int, v string, rng *rand.Rand) bool {
	allowed := false
	for _, o := range s.options() {
		if o == v {
			allowed = true
		}
	}
	if !allowed {
		return false
	}
	s.Votes[p] = v
	if s.Mode == modeOnline && s.Votes[1-p] != v {
		return false
	}
	switch v {
	case voteContinue:
		s.startRound(rng)
	case voteRematch:
		s.startSeries(rng)
	case voteFree:
		s.FreePlay = true
		s.startRound(rng)
	}
	return true
}

// timeout charges the player to move with a timeout. The third forfeits;
// otherwise the die goes into a random open column.
func (s *session) timeout(rng *rand.Rand) {
	if s.Phase != phasePlay {
		return
	}
	p := s.Match.Turn
	s.Timeouts[p]++
	s.addLog(logEntry{Kind: logTimeout, Player: p, Count: s.Timeouts[p]})
	if s.Timeouts[p] >= maxTimeouts {
		s.forfeitBy(p, reasonTimeout)
		return
	}
	moves := s.Match.LegalMoves()
	if len(moves) == 0 {
		return
	}
	_ = s.place(p, moves[rng.IntN(len(moves))], rng)
}

func roundsLabel(bestOf int) string {
	switch bestOf {
	case 0:
		return "Free play"
	case 1:
		return "Best of 1"
	case 3:
		return "Best of 3"
	case 5:
		return "Best of 5"
	}
	return "Best of " + strconv.Itoa(bestOf)
}

func (s *session) roundsLabel() string {
	if s.freePlay() {
		return "Free play"
	}
	return roundsLabel(s.BestOf)
}
