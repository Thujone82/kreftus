package main

import (
	"errors"
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
	maxReplay   = 256

	fairNone = iota
	fairVerified
	fairFailed
)

type replayMove struct {
	Player int    `json:"p"`
	Col    int    `json:"c"`
	Die    int    `json:"d"`
	Dice   [2]int `json:"dice"`
}

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
	// Aiming is the player-to-move's live column/die choice so the
	// opponent can watch the placement preview before the move lands.
	Aiming bool `json:"aiming,omitempty"`
	AimCol int  `json:"aimCol,omitempty"`
	AimDie int  `json:"aimDie,omitempty"`

	// Fair-play commit-reveal state for online matches. Unexported seeds are
	// never serialized over JSON until round end (HostReveal).
	HostCommit     string   `json:"hostCommit,omitempty"`
	GuestSeed      string   `json:"guestSeed,omitempty"`
	HostReveal     string       `json:"hostReveal,omitempty"`
	NextHostCommit string       `json:"nextHostCommit,omitempty"`
	RoundFirst     int          `json:"roundFirst,omitempty"`
	RoundRolls     [][2]int     `json:"roundRolls,omitempty"`
	RoundMoves     []replayMove `json:"roundMoves,omitempty"`

	hostSeed      string
	nextHostSeed  string
	nextGuestSeed string
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

func newOnlineSession(names [2]string, bestOf int, rules Rules, rng *rand.Rand, hostSeed, guestSeed string) *session {
	s := &session{
		Mode:          modeOnline,
		Names:         names,
		BestOf:        bestOf,
		Rules:         rules,
		Connected:     [2]bool{true, true},
		nextHostSeed:  hostSeed,
		nextGuestSeed: guestSeed,
	}
	s.startSeries(rng)
	return s
}

func (s *session) rollDice(rng *rand.Rand) {
	if s.Mode == modeOnline && s.hostSeed != "" && s.GuestSeed != "" {
		s.Match.Dice = roundRoll(s.hostSeed, s.GuestSeed, s.Match.Moves, s.Rules.TwoDice)
	} else {
		s.Match.Roll(rng)
	}
	if len(s.RoundRolls) < 500 {
		s.RoundRolls = append(s.RoundRolls, s.Match.Dice)
	}
}

func (s *session) revealRound() {
	if s.Mode != modeOnline || s.hostSeed == "" {
		return
	}
	s.HostReveal = s.hostSeed
	if s.nextHostSeed == "" {
		s.nextHostSeed = newRandomHex(32)
	}
	s.NextHostCommit = commitSeed(s.nextHostSeed)
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
	if s.nextHostSeed != "" {
		s.hostSeed = s.nextHostSeed
		s.nextHostSeed = ""
		s.GuestSeed = s.nextGuestSeed
		s.nextGuestSeed = ""
	}
	s.HostReveal = ""
	s.NextHostCommit = ""
	s.RoundRolls = nil
	s.RoundMoves = nil
	first := 0
	if s.Mode == modeOnline && s.hostSeed != "" && s.GuestSeed != "" {
		s.HostCommit = commitSeed(s.hostSeed)
		first = roundCoin(s.hostSeed, s.GuestSeed)
	} else {
		s.HostCommit = ""
		first = rng.IntN(2)
	}
	s.RoundFirst = first
	s.Match = newMatch(s.Rules, first)
	s.rollDice(rng)
	s.Phase = phasePlay
	s.Votes = [2]string{}
	s.Last = nil
	s.clearAim()
	s.addLog(logEntry{Kind: logFirst, Player: first})
}

func (s *session) clearAim() {
	s.Aiming = false
	s.AimCol = 0
	s.AimDie = 0
}

func (s *session) setAim(col, die int) bool {
	if s.Phase != phasePlay {
		return false
	}
	if col < 0 || col >= gridCols || die < 0 || die > 1 {
		return false
	}
	if s.Match.Dice[die] == 0 {
		return false
	}
	if s.Aiming && s.AimCol == col && s.AimDie == die {
		return false
	}
	s.Aiming = true
	s.AimCol = col
	s.AimDie = die
	return true
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
	prevDice := s.Match.Dice
	pl, err := s.Match.Place(mv)
	if err != nil {
		return err
	}
	if len(s.RoundMoves) < maxReplay {
		s.RoundMoves = append(s.RoundMoves, replayMove{Player: p, Col: mv.Col, Die: mv.Die, Dice: prevDice})
	}
	s.clearAim()
	s.Last = &pl
	s.addLog(logEntry{Kind: logPlace, Player: p, Die: pl.Die, Col: pl.Col, Destroyed: pl.Destroyed, Gain: pl.Gain, Lost: pl.Lost})
	if s.Match.Over {
		s.finishRound()
	} else {
		s.rollDice(rng)
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
	s.revealRound()
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
	s.clearAim()
	s.addLog(logEntry{Kind: logForfeit, Player: loser, Reason: reason})
	s.revealRound()
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

func validVote(v string) bool {
	return v == "" || v == voteContinue || v == voteRematch || v == voteFree
}

func validReason(r string) bool {
	return r == reasonResign || r == reasonDisconnect || r == reasonTimeout
}

func validHex(s string, maxLen int) bool {
	if len(s) > maxLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// validate checks untrusted session state received over the network and
// sanitizes display strings to protect client renderers.
func (s *session) validate() error {
	if s == nil {
		return errors.New("nil session")
	}
	switch s.Mode {
	case modeAI, modeLocal, modeOnline:
	default:
		return errors.New("invalid mode")
	}
	switch s.Phase {
	case phasePlay, phaseRoundOver, phaseGameOver:
	default:
		return errors.New("invalid phase")
	}
	if s.Difficulty != "" && difficultyLabel[s.Difficulty] == "" {
		return errors.New("invalid difficulty")
	}
	if s.BestOf < 0 || s.BestOf > 99 || s.Series < 0 || s.Series > 10000 || s.Round < 0 || s.Round > 10000 {
		return errors.New("invalid series counters")
	}
	if s.Wins[0] < 0 || s.Wins[0] > 10000 || s.Wins[1] < 0 || s.Wins[1] > 10000 || s.Ties < 0 || s.Ties > 10000 {
		return errors.New("invalid win counters")
	}
	if s.Timeouts[0] < 0 || s.Timeouts[0] > maxTimeouts || s.Timeouts[1] < 0 || s.Timeouts[1] > maxTimeouts {
		return errors.New("invalid timeout counters")
	}
	if s.TurnLimit < 0 || s.TurnLimit > 3600 || s.TurnLeftMs < 0 || s.TurnLeftMs > 3600*1000 || s.WaitLeftMs < 0 || s.WaitLeftMs > 3600*1000 {
		return errors.New("invalid timers")
	}
	s.Names[0] = cleanName(s.Names[0], "Host")
	s.Names[1] = cleanName(s.Names[1], "Guest")

	m := &s.Match
	if m.Turn < 0 || m.Turn > 1 || m.Moves < 0 || m.Moves > 1000 {
		return errors.New("invalid match turn or moves")
	}
	for p := 0; p < 2; p++ {
		for c := 0; c < gridCols; c++ {
			seenZero := false
			for k := 0; k < gridRows; k++ {
				v := m.Grids[p][c][k]
				if v < 0 || v > 6 {
					return errors.New("die value out of range")
				}
				if v == 0 {
					seenZero = true
				} else if seenZero {
					return errors.New("non-packed column")
				}
			}
		}
	}
	for i := 0; i < 2; i++ {
		if m.Dice[i] < 0 || m.Dice[i] > 6 {
			return errors.New("rolled die out of range")
		}
	}
	if !m.Rules.TwoDice && m.Dice[1] != 0 {
		return errors.New("unexpected second die")
	}
	if s.Phase == phasePlay && !m.Over {
		if m.Dice[0] == 0 {
			return errors.New("missing active die")
		}
		if m.Rules.TwoDice && m.Dice[1] == 0 {
			return errors.New("missing second active die")
		}
	}
	if s.AimCol < 0 || s.AimCol >= gridCols || s.AimDie < 0 || s.AimDie > 1 {
		return errors.New("aim out of range")
	}
	if s.Aiming && s.Phase == phasePlay && m.Dice[s.AimDie] == 0 {
		return errors.New("aiming with missing die")
	}
	if !validVote(s.Votes[0]) || !validVote(s.Votes[1]) {
		return errors.New("invalid vote")
	}
	if s.Last != nil {
		if s.Last.Player < 0 || s.Last.Player > 1 || s.Last.Col < 0 || s.Last.Col >= gridCols || s.Last.Slot < -1 || s.Last.Slot >= gridRows || s.Last.Die < 1 || s.Last.Die > 6 {
			return errors.New("invalid last placement")
		}
	}
	if s.Forfeit != nil {
		if s.Forfeit.Loser < 0 || s.Forfeit.Loser > 1 || !validReason(s.Forfeit.Reason) || s.Phase != phaseGameOver {
			return errors.New("invalid forfeit")
		}
	}
	if len(s.Log) > maxLog {
		return errors.New("oversized log")
	}
	for _, e := range s.Log {
		if e.Round < 0 || e.Round > 10000 || e.Col < 0 || e.Col >= gridCols || e.Die < 0 || e.Die > 6 || e.Destroyed < 0 || e.Destroyed > gridRows || e.Count < 0 || e.Count > maxTimeouts {
			return errors.New("invalid log entry")
		}
		switch e.Kind {
		case logFirst, logTimeout:
			if e.Player < 0 || e.Player > 1 {
				return errors.New("invalid log player")
			}
		case logPlace:
			if e.Player < 0 || e.Player > 1 || e.Die < 1 || e.Die > 6 {
				return errors.New("invalid place log entry")
			}
		case logRound:
			if e.Player < -1 || e.Player > 1 {
				return errors.New("invalid round log player")
			}
		case logForfeit:
			if e.Player < 0 || e.Player > 1 || !validReason(e.Reason) {
				return errors.New("invalid forfeit log entry")
			}
		default:
			return errors.New("unknown log entry kind")
		}
	}
	if len(s.History) > 1000 {
		return errors.New("oversized history")
	}
	for _, h := range s.History {
		if h.Round < 0 || h.Round > 10000 || h.Winner < -1 || h.Winner > 1 || h.Lead < -1 || h.Lead > 1 {
			return errors.New("invalid history entry")
		}
	}
	if !validHex(s.HostCommit, 128) || !validHex(s.GuestSeed, 128) || !validHex(s.HostReveal, 128) || !validHex(s.NextHostCommit, 128) {
		return errors.New("invalid commit-reveal hex")
	}
	if s.RoundFirst < 0 || s.RoundFirst > 1 || len(s.RoundRolls) > 500 {
		return errors.New("invalid round fairness metadata")
	}
	for _, r := range s.RoundRolls {
		if r[0] < 0 || r[0] > 6 || r[1] < 0 || r[1] > 6 {
			return errors.New("invalid round roll entry")
		}
	}
	if len(s.RoundMoves) > maxReplay {
		return errors.New("oversized replay")
	}
	for _, rm := range s.RoundMoves {
		if rm.Player < 0 || rm.Player > 1 || rm.Col < 0 || rm.Col >= gridCols || rm.Die < 0 || rm.Die > 1 || rm.Dice[0] < 0 || rm.Dice[0] > 6 || rm.Dice[1] < 0 || rm.Dice[1] > 6 {
			return errors.New("invalid replay move")
		}
	}
	return nil
}
