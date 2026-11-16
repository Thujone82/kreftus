package main

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	mrand "math/rand/v2"
	"net"
	"strconv"
	"time"
)

const (
	challengeTurnSeconds = 30
	challengeAIThinkDelay = 300 * time.Millisecond
	challengeMaxLog       = 300
	challengeMaxReplay    = 256
	challengeMaxTimeouts  = 3
)

type challengeNetMsg struct {
	Type    string       `json:"type"`
	Version int          `json:"version,omitempty"`
	Seq     uint64       `json:"seq,omitempty"`
	MAC     string       `json:"mac,omitempty"`
	Nonce   string       `json:"nonce,omitempty"`
	AuthReq bool         `json:"authReq,omitempty"`
	Auth    string       `json:"auth,omitempty"`
	Commit  string       `json:"commit,omitempty"`
	Seed    string       `json:"seed,omitempty"`
	Name    string       `json:"name,omitempty"`
	Token   string       `json:"token,omitempty"`
	Aiming  bool         `json:"aiming,omitempty"`
	Col     int          `json:"col,omitempty"`
	Die     int          `json:"die,omitempty"`
	Vote    string       `json:"vote,omitempty"`
	Text    string       `json:"text,omitempty"`
	State   *wireSession `json:"state,omitempty"`
}

func challengeRandomHex(nBytes int) string {
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func challengeCommitSeed(seedHex string) string {
	sum := sha256.Sum256([]byte("kb-seed-v2|" + seedHex))
	return hex.EncodeToString(sum[:])
}

func challengeRoundCoin(hostSeed, guestSeed string) int {
	mac := hmac.New(sha256.New, []byte(hostSeed))
	mac.Write([]byte("coin-v2|" + guestSeed))
	return int(mac.Sum(nil)[0] & 1)
}

func challengeRoundRoll(hostSeed, guestSeed string, moveIdx int) [2]int {
	var out [2]int
	for ctr := 0; ; ctr++ {
		mac := hmac.New(sha256.New, []byte(hostSeed))
		mac.Write([]byte("roll-v2|" + guestSeed + "|" + strconv.Itoa(moveIdx) + "|" + strconv.Itoa(ctr)))
		sum := mac.Sum(nil)
		for _, b := range sum {
			if b < 252 {
				out[0] = int(b%6) + 1
				return out
			}
		}
	}
}

func challengeGridCount(g wireGrid, c int) int {
	n := 0
	for _, v := range g[c] {
		if v != 0 {
			n++
		}
	}
	return n
}

func challengeGridFull(g wireGrid) bool {
	for c := 0; c < gridCols; c++ {
		if challengeGridCount(g, c) < gridRows {
			return false
		}
	}
	return true
}

func challengeGridAdd(g *wireGrid, c, v int) int {
	for i := 0; i < gridRows; i++ {
		if g[c][i] == 0 {
			g[c][i] = v
			return i
		}
	}
	return -1
}

func challengeGridDestroy(g *wireGrid, c, v int) int {
	var kept [gridRows]int
	k, n := 0, 0
	for _, d := range g[c] {
		switch {
		case d == 0:
		case d == v:
			n++
		default:
			kept[k] = d
			k++
		}
	}
	g[c] = kept
	return n
}

type chMatch struct {
	Grids [2]wireGrid
	Turn  int
	Dice  [2]int
	Over  bool
	Moves int
}

func (m *chMatch) LegalCols() []int {
	if m.Over || m.Dice[0] == 0 {
		return nil
	}
	var cols []int
	for c := 0; c < gridCols; c++ {
		if challengeGridCount(m.Grids[m.Turn], c) < gridRows {
			cols = append(cols, c)
		}
	}
	return cols
}

func (m *chMatch) Apply(col int) (wirePlacement, bool) {
	if m.Over || m.Dice[0] == 0 || col < 0 || col >= gridCols || challengeGridCount(m.Grids[m.Turn], col) >= gridRows {
		return wirePlacement{}, false
	}
	p, o := m.Turn, 1-m.Turn
	v := m.Dice[0]
	before, oBefore := m.Grids[p].score(), m.Grids[o].score()
	pl := wirePlacement{Player: p, Col: col, Die: v}
	pl.Slot = challengeGridAdd(&m.Grids[p], col, v)
	pl.Destroyed = challengeGridDestroy(&m.Grids[o], col, v)
	pl.Gain = m.Grids[p].score() - before
	pl.Lost = oBefore - m.Grids[o].score()
	m.Dice = [2]int{}
	m.Moves++
	m.Over = challengeGridFull(m.Grids[0]) || challengeGridFull(m.Grids[1])
	if !m.Over {
		m.Turn = o
	}
	return pl, true
}

func (m *chMatch) Preview(col int) chMatch {
	next := *m
	next.Apply(col)
	return next
}

func (m *chMatch) Scores() [2]int {
	return [2]int{m.Grids[0].score(), m.Grids[1].score()}
}

func (m *chMatch) Winner() int {
	s := m.Scores()
	if s[0] > s[1] {
		return 0
	}
	if s[1] > s[0] {
		return 1
	}
	return -1
}

// Hard AI (Shrumy): 2-ply expectimax matching go/knucklebones/ai.go.
func chEvaluate(m *chMatch, p int) float64 {
	s := m.Scores()
	d := float64(s[p] - s[1-p])
	if m.Over {
		if d > 0 {
			return 1000.0 + d
		}
		if d < 0 {
			return -1000.0 + d
		}
	}
	return d
}

func chExpectimax(ctx context.Context, m *chMatch, p, plies int) (float64, bool) {
	if ctx.Err() != nil {
		return 0, false
	}
	if m.Over || plies == 0 {
		return chEvaluate(m, p), true
	}
	total := 0.0
	for a := 1; a <= 6; a++ {
		if ctx.Err() != nil {
			return 0, false
		}
		roll := *m
		roll.Dice = [2]int{a, 0}
		cols := roll.LegalCols()
		best := 0.0
		for i, col := range cols {
			after := roll.Preview(col)
			v, ok := chExpectimax(ctx, &after, p, plies-1)
			if !ok {
				return 0, false
			}
			if i == 0 || (roll.Turn == p && v > best) || (roll.Turn != p && v < best) {
				best = v
			}
		}
		total += best
	}
	return total / 6.0, true
}

func chooseHardChallengeCol(ctx context.Context, m *chMatch, rng *mrand.Rand) (int, bool) {
	cols := m.LegalCols()
	if len(cols) == 0 {
		return -1, true
	}
	p := m.Turn
	var picks []int
	best := 0.0
	for i, col := range cols {
		if ctx.Err() != nil {
			return -1, false
		}
		next := m.Preview(col)
		v, ok := chExpectimax(ctx, &next, p, 2)
		if !ok {
			return -1, false
		}
		if i == 0 || v > best+1e-9 {
			best = v
			picks = []int{col}
		} else if v > best-1e-9 {
			picks = append(picks, col)
		}
	}
	return picks[rng.IntN(len(picks))], true
}

type challengeEngine struct {
	hub        *hub
	conn       net.Conn
	playerName string
	rng        *mrand.Rand

	series        int
	round         int
	wins          [2]int
	ties          int
	phase         string
	match         chMatch
	log           []wireLogEntry
	history       []wireHistoryEntry
	timeouts      [2]int
	votes         [2]string
	forfeit       *wireForfeit
	last          *wirePlacement
	roundFirst    int
	roundRolls    [][2]int
	roundMoves    []wireReplayMove
	hostSeed      string
	hostCommit    string
	guestSeed     string
	hostReveal    string
	nextHostSeed  string
	nextHostComm  string
	nextGuestSeed string

	turnStartedAt time.Time
	firstDieAt    time.Time
	lastDieAt     time.Time
	delta         int
	rank          int
	attempted     bool
}

func newChallengeEngine(h *hub, conn net.Conn, playerName, hostSeed, guestSeed string) *challengeEngine {
	e := &challengeEngine{
		hub:          h,
		conn:         conn,
		playerName:   playerName,
		rng:          mrand.New(mrand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano()>>1))),
		nextHostSeed: hostSeed,
		nextGuestSeed: guestSeed,
	}
	e.startSeries()
	return e
}

func (e *challengeEngine) addLog(entry wireLogEntry) {
	entry.Round = e.round
	entry.Scores = e.match.Scores()
	e.log = append(e.log, entry)
	if len(e.log) > challengeMaxLog {
		e.log = append([]wireLogEntry(nil), e.log[len(e.log)-challengeMaxLog:]...)
	}
}

func (e *challengeEngine) startSeries() {
	e.series++
	e.round = 0
	e.wins = [2]int{}
	e.ties = 0
	e.history = nil
	e.timeouts = [2]int{}
	e.forfeit = nil
	e.firstDieAt = time.Time{}
	e.lastDieAt = time.Time{}
	e.delta = 0
	e.rank = 0
	if !e.attempted {
		e.attempted = true
		e.hub.stats.RecordChallengeAttempt()
	} else {
		e.hub.stats.RecordChallengeAttempt()
	}
	e.startRound()
}

func (e *challengeEngine) startRound() {
	e.round++
	if e.nextHostSeed == "" {
		e.nextHostSeed = challengeRandomHex(32)
	}
	e.hostSeed = e.nextHostSeed
	e.nextHostSeed = ""
	if e.nextGuestSeed != "" {
		e.guestSeed = e.nextGuestSeed
		e.nextGuestSeed = ""
	}
	e.hostCommit = challengeCommitSeed(e.hostSeed)
	e.hostReveal = ""
	e.nextHostComm = ""
	e.roundRolls = nil
	e.roundMoves = nil
	first := challengeRoundCoin(e.hostSeed, e.guestSeed)
	e.roundFirst = first
	e.match = chMatch{Turn: first}
	e.rollDie()
	e.phase = phasePlay
	e.votes = [2]string{}
	e.last = nil
	e.turnStartedAt = time.Now()
	e.addLog(wireLogEntry{Kind: "first", Player: first})
}

func (e *challengeEngine) rollDie() {
	e.match.Dice = challengeRoundRoll(e.hostSeed, e.guestSeed, e.match.Moves)
	if len(e.roundRolls) < 500 {
		e.roundRolls = append(e.roundRolls, e.match.Dice)
	}
}

func (e *challengeEngine) computeCompletedDelta() int {
	d := 0
	for _, h := range e.history {
		d += h.Scores[1] - h.Scores[0]
	}
	return d
}

func (e *challengeEngine) place(p, col int) bool {
	if e.phase != phasePlay || p != e.match.Turn {
		return false
	}
	prevDice := e.match.Dice
	pl, ok := e.match.Apply(col)
	if !ok {
		return false
	}
	now := time.Now()
	if e.firstDieAt.IsZero() {
		e.firstDieAt = now
	}
	e.lastDieAt = now
	e.hub.stats.RecordChallengeDie()

	if len(e.roundMoves) < challengeMaxReplay {
		e.roundMoves = append(e.roundMoves, wireReplayMove{
			Player: p,
			Col:    col,
			Die:    0,
			Dice:   prevDice,
		})
	}
	e.last = &pl
	e.addLog(wireLogEntry{
		Kind:      "place",
		Player:    p,
		Die:       pl.Die,
		Col:       pl.Col,
		Destroyed: pl.Destroyed,
		Gain:      pl.Gain,
		Lost:      pl.Lost,
	})
	if e.match.Over {
		e.finishRound()
	} else {
		e.rollDie()
		e.turnStartedAt = now
	}
	return true
}

func (e *challengeEngine) finishRound() {
	w := e.match.Winner()
	if w >= 0 {
		e.wins[w]++
	} else {
		e.ties++
	}
	lead := -1
	if e.wins[0] > e.wins[1] {
		lead = 0
	} else if e.wins[1] > e.wins[0] {
		lead = 1
	}
	scores := e.match.Scores()
	e.history = append(e.history, wireHistoryEntry{
		Round:  e.round,
		Scores: scores,
		Winner: w,
		Lead:   lead,
	})
	e.addLog(wireLogEntry{Kind: "round", Player: w})
	e.hostReveal = e.hostSeed
	if e.nextHostSeed == "" {
		e.nextHostSeed = challengeRandomHex(32)
	}
	e.nextHostComm = challengeCommitSeed(e.nextHostSeed)
	e.delta = e.computeCompletedDelta()

	// All 3 Challenge matches are always played regardless of early lead.
	if e.round >= 3 {
		e.phase = phaseGameOver
		durMs := int64(1)
		if !e.firstDieAt.IsZero() && !e.lastDieAt.IsZero() {
			if d := e.lastDieAt.Sub(e.firstDieAt).Milliseconds(); d > 1 {
				durMs = d
			}
		}
		var roundDeltas [3]int
		pTot, aiTot, pWins, aiWins, ties := 0, 0, 0, 0, 0
		for i, h := range e.history {
			if i < 3 {
				roundDeltas[i] = h.Scores[1] - h.Scores[0]
			}
			pTot += h.Scores[1]
			aiTot += h.Scores[0]
			switch {
			case h.Scores[1] > h.Scores[0]:
				pWins++
			case h.Scores[0] > h.Scores[1]:
				aiWins++
			default:
				ties++
			}
		}
		e.rank = e.hub.stats.RecordChallengeCompletion(ChallengeEntry{
			Name:        e.playerName,
			Delta:       e.delta,
			DurationMs:  durMs,
			RoundDeltas: roundDeltas,
			PlayerTotal: pTot,
			AITotal:     aiTot,
			Wins:        pWins,
			Losses:      aiWins,
			Ties:        ties,
		})
		e.hub.broadcastLobby(true)
	} else {
		e.phase = phaseRoundOver
	}
}

func (e *challengeEngine) forfeitBy(loser int, reason string) {
	if e.forfeit != nil || e.phase == phaseGameOver {
		return
	}
	e.forfeit = &wireForfeit{Reason: reason, Loser: loser}
	e.match.Over = true
	e.match.Dice = [2]int{}
	e.phase = phaseGameOver
	e.votes = [2]string{}
	e.addLog(wireLogEntry{Kind: "forfeit", Player: loser, Reason: reason})
	e.hostReveal = e.hostSeed
	if e.nextHostSeed == "" {
		e.nextHostSeed = challengeRandomHex(32)
	}
	e.nextHostComm = challengeCommitSeed(e.nextHostSeed)
	e.hub.stats.RecordChallengeForfeit()
}

func (e *challengeEngine) elapsedMs(now time.Time) int64 {
	if e.firstDieAt.IsZero() {
		return 0
	}
	if e.phase == phaseGameOver && !e.lastDieAt.IsZero() {
		if d := e.lastDieAt.Sub(e.firstDieAt).Milliseconds(); d > 0 {
			return d
		}
		return 1
	}
	if d := now.Sub(e.firstDieAt).Milliseconds(); d > 0 {
		return d
	}
	return 0
}

func (e *challengeEngine) wireState(now time.Time) *wireSession {
	var turnLeft int64
	if e.phase == phasePlay && challengeTurnSeconds > 0 {
		rem := time.Duration(challengeTurnSeconds)*time.Second - now.Sub(e.turnStartedAt)
		if rem < 0 {
			rem = 0
		}
		turnLeft = rem.Milliseconds()
	}
	logCopy := append([]wireLogEntry(nil), e.log...)
	if logCopy == nil {
		logCopy = []wireLogEntry{}
	}
	histCopy := append([]wireHistoryEntry(nil), e.history...)
	if histCopy == nil {
		histCopy = []wireHistoryEntry{}
	}
	rollsCopy := append([][2]int(nil), e.roundRolls...)
	if rollsCopy == nil {
		rollsCopy = [][2]int{}
	}
	movesCopy := append([]wireReplayMove(nil), e.roundMoves...)
	if movesCopy == nil {
		movesCopy = []wireReplayMove{}
	}
	return &wireSession{
		Mode:      "online",
		Names:     [2]string{"Shrumy", e.playerName},
		Difficulty: "hard",
		Challenge: true,
		BestOf:    3,
		FreePlay:  false,
		Rules:     Rules{},
		Match: wireMatch{
			Rules: Rules{},
			Grids: e.match.Grids,
			Turn:  e.match.Turn,
			Dice:  e.match.Dice,
			Over:  e.match.Over,
			Moves: e.match.Moves,
		},
		Series:             e.series,
		Round:              e.round,
		Wins:               e.wins,
		Ties:               e.ties,
		Phase:              e.phase,
		Log:                logCopy,
		History:            histCopy,
		Timeouts:           e.timeouts,
		Votes:              e.votes,
		Forfeit:            e.forfeit,
		Last:               e.last,
		Connected:          [2]bool{true, true},
		TurnLimit:          challengeTurnSeconds,
		TurnLeftMs:         turnLeft,
		HostCommit:         e.hostCommit,
		GuestSeed:          e.guestSeed,
		HostReveal:         e.hostReveal,
		NextHostCommit:     e.nextHostComm,
		RoundFirst:         e.roundFirst,
		RoundRolls:         rollsCopy,
		RoundMoves:         movesCopy,
		ChallengeElapsedMs: e.elapsedMs(now),
		ChallengeDelta:     e.computeCompletedDelta(),
		ChallengeRank:      e.rank,
	}
}

func (e *challengeEngine) sendState() error {
	b, err := json.Marshal(challengeNetMsg{
		Type:  "state",
		State: e.wireState(time.Now()),
	})
	if err != nil {
		return err
	}
	return writeRawLine(e.conn, b, writeTimeout)
}

func (e *challengeEngine) sendPong() error {
	b, err := json.Marshal(challengeNetMsg{Type: "pong"})
	if err != nil {
		return err
	}
	return writeRawLine(e.conn, b, writeTimeout)
}

func (h *hub) runChallenge(conn net.Conn, br *bufio.Reader, req envelope) {
	if _, nameErr := ValidateChallengeName(req.Name); nameErr != "" {
		msg := "Please choose a custom name for Challenge mode."
		if nameErr == errBadName {
			msg = "That name is not allowed on the Challenge leaderboard."
		}
		_ = writeEnvelope(conn, envelope{Type: rMsgError, Code: nameErr, Text: msg}, writeTimeout)
		return
	}

	h.mu.Lock()
	if h.activeChallenges >= maxRooms {
		h.mu.Unlock()
		_ = writeEnvelope(conn, envelope{Type: rMsgError, Code: errServerBusy, Text: errServerBusy}, writeTimeout)
		return
	}
	h.activeChallenges++
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		h.activeChallenges--
		h.mu.Unlock()
		h.broadcastLobby(true)
	}()

	chID := "ch-" + randomRoomID()
	if err := writeEnvelope(conn, envelope{
		Type:    rMsgPaired,
		RoomID:  chID,
		ConnSeq: 1,
	}, writeTimeout); err != nil {
		return
	}

	h.broadcastLobby(true)

	nonce := challengeRandomHex(32)
	hostSeed := challengeRandomHex(32)
	commit := challengeCommitSeed(hostSeed)

	chalBytes, _ := json.Marshal(challengeNetMsg{
		Type:    "challenge",
		Version: 2,
		Nonce:   nonce,
		AuthReq: false,
		Commit:  commit,
	})
	if err := writeRawLine(conn, chalBytes, writeTimeout); err != nil {
		return
	}

	// Read client hello.
	helloLine, err := readJSONLine(br, conn, 15*time.Second, maxWireMsgBytes)
	if err != nil || len(helloLine) == 0 {
		return
	}
	var hello challengeNetMsg
	if json.Unmarshal(helloLine, &hello) != nil || hello.Type != "hello" || hello.Version != 2 {
		b, _ := json.Marshal(challengeNetMsg{Type: "error", Text: "unsupported-protocol"})
		_ = writeRawLine(conn, b, writeTimeout)
		return
	}
	helloName, helloErr := ValidateChallengeName(hello.Name)
	if helloErr != "" {
		b, _ := json.Marshal(challengeNetMsg{Type: "error", Text: helloErr})
		_ = writeRawLine(conn, b, writeTimeout)
		return
	}
	guestSeed := hello.Seed
	if len(guestSeed) != 64 {
		guestSeed = challengeRandomHex(32)
	}

	eng := newChallengeEngine(h, conn, helloName, hostSeed, guestSeed)
	if err := eng.sendState(); err != nil {
		return
	}

	msgCh := make(chan challengeNetMsg, 16)
	readErrCh := make(chan error, 1)
	go func() {
		for {
			line, rerr := readJSONLine(br, conn, relayReadTimeout, maxWireMsgBytes)
			if rerr != nil {
				readErrCh <- rerr
				return
			}
			if len(line) == 0 {
				continue
			}
			var m challengeNetMsg
			if json.Unmarshal(line, &m) == nil {
				msgCh <- m
			}
		}
	}()

	type aiResult struct {
		series int
		round  int
		moves  int
		col    int
	}
	aiCh := make(chan aiResult, 1)
	aiRunning := false
	var aiCancel context.CancelFunc
	stopAI := func() {
		if aiCancel != nil {
			aiCancel()
			aiCancel = nil
		}
		aiRunning = false
	}
	defer stopAI()

	maybeLaunchAI := func() {
		if eng.phase != phasePlay || eng.match.Turn != 0 || eng.match.Over || aiRunning {
			return
		}
		aiRunning = true
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		aiCancel = cancel
		snap := eng.match
		s, r, mv := eng.series, eng.round, eng.match.Moves
		go func() {
			timer := time.NewTimer(challengeAIThinkDelay)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return
			}
			col, ok := chooseHardChallengeCol(ctx, &snap, mrand.New(mrand.NewPCG(uint64(time.Now().UnixNano()), uint64(mv+1))))
			if !ok || col < 0 {
				cols := snap.LegalCols()
				if len(cols) > 0 {
					col = cols[0]
				}
			}
			select {
			case aiCh <- aiResult{series: s, round: r, moves: mv, col: col}:
			case <-ctx.Done():
			}
		}()
	}

	maybeLaunchAI()

	tick := time.NewTicker(1 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-readErrCh:
			if eng.phase != phaseGameOver {
				eng.forfeitBy(1, "disconnect")
			}
			return
		case res := <-aiCh:
			aiRunning = false
			if aiCancel != nil {
				aiCancel()
				aiCancel = nil
			}
			if eng.phase == phasePlay && eng.match.Turn == 0 &&
				res.series == eng.series && res.round == eng.round && res.moves == eng.match.Moves {
				if eng.place(0, res.col) {
					if err := eng.sendState(); err != nil {
						return
					}
					maybeLaunchAI()
				}
			}
		case m := <-msgCh:
			switch m.Type {
			case "ping":
				if err := eng.sendPong(); err != nil {
					return
				}
			case "bye":
				if eng.phase != phaseGameOver {
					eng.forfeitBy(1, "resignation")
				}
				return
			case "resign":
				stopAI()
				eng.forfeitBy(1, "resignation")
				if err := eng.sendState(); err != nil {
					return
				}
			case "move":
				if eng.phase == phasePlay && eng.match.Turn == 1 {
					if eng.place(1, m.Col) {
						if err := eng.sendState(); err != nil {
							return
						}
						maybeLaunchAI()
					}
				}
			case "vote":
				if len(m.Seed) == 64 && eng.nextGuestSeed == "" {
					eng.nextGuestSeed = m.Seed
				}
				switch m.Vote {
				case "continue":
					if eng.phase == phaseRoundOver {
						stopAI()
						eng.startRound()
						if err := eng.sendState(); err != nil {
							return
						}
						maybeLaunchAI()
					}
				case "rematch":
					if eng.phase == phaseGameOver {
						stopAI()
						eng.startSeries()
						if err := eng.sendState(); err != nil {
							return
						}
						maybeLaunchAI()
					}
				}
			}
		case now := <-tick.C:
			if eng.phase == phasePlay && challengeTurnSeconds > 0 {
				if now.Sub(eng.turnStartedAt) >= time.Duration(challengeTurnSeconds)*time.Second {
					turn := eng.match.Turn
					eng.timeouts[turn]++
					eng.addLog(wireLogEntry{Kind: "timeout", Player: turn, Count: eng.timeouts[turn]})
					if eng.timeouts[turn] >= challengeMaxTimeouts {
						stopAI()
						eng.forfeitBy(turn, "timeout")
						if err := eng.sendState(); err != nil {
							return
						}
						continue
					}
					cols := eng.match.LegalCols()
					if len(cols) > 0 {
						stopAI()
						_ = eng.place(turn, cols[eng.rng.IntN(len(cols))])
						if err := eng.sendState(); err != nil {
							return
						}
						maybeLaunchAI()
					}
				} else if err := eng.sendState(); err != nil {
					return
				}
			}
		}
	}
}
