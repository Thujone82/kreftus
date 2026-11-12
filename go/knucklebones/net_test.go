package main

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func startPair(t *testing.T, cfg hostConfig) (*netHost, *netClient) {
	t.Helper()
	h, err := startHost("Host", 0, cfg, testRNG())
	if err != nil {
		t.Fatal(err)
	}
	c := startClient("Guest", fmt.Sprintf("127.0.0.1:%d", h.port))
	t.Cleanup(func() {
		h.leave()
		c.leave()
	})
	pump(t, h, c, func() bool { return c.s != nil && h.s != nil })
	return h, c
}

// pump feeds network events to both ends until done reports true.
func pump(t *testing.T, h *netHost, c *netClient, done func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !done() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for the network")
		}
		select {
		case ev := <-h.events:
			h.handle(ev, time.Now())
		case ev := <-c.events:
			c.handle(ev, time.Now())
		case <-time.After(10 * time.Millisecond):
			now := time.Now()
			h.tick(now)
			c.tick(now)
		}
	}
}

func TestOnlineGameToTheEnd(t *testing.T) {
	h, c := startPair(t, hostConfig{BestOf: 1})
	if h.s.Names != [2]string{"Host", "Guest"} || c.s.Names != h.s.Names {
		t.Fatalf("names host %v guest %v", h.s.Names, c.s.Names)
	}
	for h.s.Phase == phasePlay {
		moves := h.s.Match.Moves
		if h.s.Match.Turn == 0 {
			h.place(h.s.Match.LegalMoves()[0])
		} else {
			c.place(c.s.Match.LegalMoves()[0])
		}
		pump(t, h, c, func() bool { return c.s.Match.Moves > moves || c.s.Phase != phasePlay })
	}
	pump(t, h, c, func() bool { return c.s.Phase == phaseGameOver })
	if c.s.Match.Scores() != h.s.Match.Scores() || c.s.Wins != h.s.Wins {
		t.Fatalf("host %v %v, guest %v %v", h.s.Match.Scores(), h.s.Wins, c.s.Match.Scores(), c.s.Wins)
	}

	h.vote(voteRematch)
	pump(t, h, c, func() bool { return c.s.Votes[0] == voteRematch })
	if h.s.Phase != phaseGameOver {
		t.Fatal("rematch started with one vote")
	}
	c.vote(voteRematch)
	pump(t, h, c, func() bool { return c.s.Series == 2 && c.s.Phase == phasePlay })
}

func TestOnlineAimPreview(t *testing.T) {
	h, c := startPair(t, hostConfig{BestOf: 1})
	// Put the guest on move so client aims travel host → guest state.
	for h.s.Match.Turn != 1 {
		h.place(h.s.Match.LegalMoves()[0])
		pump(t, h, c, func() bool { return c.s.Match.Moves == h.s.Match.Moves })
		if h.s.Phase != phasePlay {
			t.Fatal("game ended before guest's turn")
		}
	}
	open := -1
	for col := 0; col < gridCols; col++ {
		if h.s.Match.Grids[1].Open(col) {
			open = col
			break
		}
	}
	if open < 0 {
		t.Fatal("no open column for guest")
	}
	c.aim(open, 0)
	pump(t, h, c, func() bool { return h.s.Aiming && c.s.Aiming })
	if !h.s.Aiming || h.s.AimCol != open || h.s.AimDie != 0 {
		t.Fatalf("host aim %+v col=%d die=%d", h.s.Aiming, h.s.AimCol, h.s.AimDie)
	}
	if !c.s.Aiming || c.s.AimCol != open {
		t.Fatalf("guest aim %+v col=%d", c.s.Aiming, c.s.AimCol)
	}
	if !h.s.Match.Legal(Move{Col: open, Die: 0}) {
		t.Fatal("aimed move should be legal")
	}

	beforeMoves := h.s.Match.Moves
	c.place(Move{Col: open, Die: 0})
	pump(t, h, c, func() bool {
		return (h.s.Match.Moves > beforeMoves || h.s.Phase != phasePlay) &&
			c.s.Match.Moves == h.s.Match.Moves && !h.s.Aiming && !c.s.Aiming
	})
	if h.s.Aiming || c.s.Aiming {
		t.Fatalf("aim should clear after place: host=%v guest=%v", h.s.Aiming, c.s.Aiming)
	}

	// Host aim is mirrored to the guest while the host is to move.
	if h.s.Phase != phasePlay {
		return
	}
	for h.s.Match.Turn != 0 {
		if h.s.Match.Turn == 1 {
			c.place(c.s.Match.LegalMoves()[0])
		} else {
			h.place(h.s.Match.LegalMoves()[0])
		}
		pump(t, h, c, func() bool { return c.s.Match.Moves == h.s.Match.Moves || h.s.Phase != phasePlay })
		if h.s.Phase != phasePlay {
			return
		}
	}
	col := h.s.Match.LegalMoves()[0].Col
	h.aim(col, 0)
	pump(t, h, c, func() bool { return c.s.Aiming && c.s.AimCol == col })
	if !c.s.Aiming || c.s.AimCol != col {
		t.Fatalf("guest missing host aim: %+v col=%d", c.s.Aiming, c.s.AimCol)
	}
}

func TestOnlineResign(t *testing.T) {
	h, c := startPair(t, hostConfig{BestOf: 3})
	c.resign()
	pump(t, h, c, func() bool { return c.s.Forfeit != nil })
	if h.s.Forfeit.Loser != 1 || h.s.Forfeit.Reason != reasonResign || h.s.seriesWinner() != 0 {
		t.Fatalf("forfeit %+v", h.s.Forfeit)
	}
}

func TestOnlineLeaveCountsAsResign(t *testing.T) {
	h, c := startPair(t, hostConfig{BestOf: 1})
	c.leave()
	pump(t, h, c, func() bool { return h.s.Forfeit != nil })
	if h.s.Forfeit.Loser != 1 || !h.opponentGone() {
		t.Fatalf("forfeit %+v gone %v", h.s.Forfeit, h.opponentGone())
	}
}

func TestOnlineDisconnectForfeit(t *testing.T) {
	old := reconnectWindow
	reconnectWindow = 200 * time.Millisecond
	defer func() { reconnectWindow = old }()

	h, c := startPair(t, hostConfig{BestOf: 1})
	// Drop the guest without a clean bye, and stop it from dialing back so
	// the host's reconnect window can expire into a disconnect forfeit.
	c.fatal = true
	h.ln.Close()
	c.closeConn()
	pump(t, h, c, func() bool { return h.s.Forfeit != nil })
	if h.s.Forfeit.Reason != reasonDisconnect || h.s.Forfeit.Loser != 1 {
		t.Fatalf("forfeit %+v", h.s.Forfeit)
	}
}

func TestOnlineReconnect(t *testing.T) {
	h, c := startPair(t, hostConfig{BestOf: 1})
	c.conn.Close()
	pump(t, h, c, func() bool { return !h.s.Connected[1] })
	pump(t, h, c, func() bool { return h.s.Connected[1] && c.conn != nil && c.lostAt.IsZero() })
	if h.s.Forfeit != nil {
		t.Fatalf("reconnect forfeited: %+v", h.s.Forfeit)
	}
}

func TestOnlineTimeouts(t *testing.T) {
	old := turnUnit
	turnUnit = 20 * time.Millisecond
	defer func() { turnUnit = old }()

	h, c := startPair(t, hostConfig{BestOf: 1, TurnSeconds: 1})
	pump(t, h, c, func() bool { return h.s.Forfeit != nil || h.s.Phase != phasePlay })
	if h.s.Forfeit == nil || h.s.Forfeit.Reason != reasonTimeout || h.s.Timeouts[h.s.Forfeit.Loser] != maxTimeouts {
		t.Fatalf("forfeit %+v timeouts %v", h.s.Forfeit, h.s.Timeouts)
	}
	pump(t, h, c, func() bool { return c.s.Forfeit != nil })
}

func TestRoomFull(t *testing.T) {
	h, c := startPair(t, hostConfig{BestOf: 1})
	other := startClient("Third", fmt.Sprintf("127.0.0.1:%d", h.port))
	defer other.leave()
	deadline := time.Now().Add(5 * time.Second)
	for other.errText == "" && time.Now().Before(deadline) {
		select {
		case ev := <-h.events:
			h.handle(ev, time.Now())
		case ev := <-c.events:
			c.handle(ev, time.Now())
		case ev := <-other.events:
			other.handle(ev, time.Now())
		case <-time.After(10 * time.Millisecond):
			now := time.Now()
			h.tick(now)
			c.tick(now)
			other.tick(now)
		}
	}
	if other.errText != errRoomFull {
		t.Fatalf("third player got %q", other.errText)
	}
}

func TestReadLoopRejectsOversizedLine(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	events := make(chan netEvent, 4)
	done := make(chan struct{})
	defer close(done)
	go readLoop(1, c1, events, done)

	go func() {
		_, _ = c2.Write([]byte(strings.Repeat("x", maxWireMsgBytes+128) + "\n"))
	}()

	select {
	case ev := <-events:
		if ev.kind != evClosed {
			t.Fatalf("got event %q, want %q", ev.kind, evClosed)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for oversized line rejection")
	}
}

func TestClientRejectsInvalidStateAndArbitraryErrorText(t *testing.T) {
	c := &netClient{connID: 1, events: make(chan netEvent, 4), done: make(chan struct{})}
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	c.conn = a

	bad := newSession(modeOnline, [2]string{"Host", "Guest"}, "", 1, Rules{}, testRNG())
	bad.Match.Dice[0] = 9 // invalid die value
	c.handle(netEvent{kind: evMsg, id: 1, msg: netMsg{Type: msgState, State: bad}}, time.Now())
	if c.errText != errBadState || !c.fatal || c.s != nil {
		t.Fatalf("expected errBadState fatal rejection, got err=%q fatal=%v s=%v", c.errText, c.fatal, c.s)
	}

	c2 := &netClient{connID: 1}
	c2.conn = b
	c2.handle(netEvent{kind: evMsg, id: 1, msg: netMsg{Type: msgError, Text: "\x1b[2Jspoofed text"}}, time.Now())
	if c2.errText != errRejected {
		t.Fatalf("unexpected error mapping: %q", c2.errText)
	}
}

func TestHostThrottlesAimBroadcast(t *testing.T) {
	h, c := startPair(t, hostConfig{BestOf: 1})
	// Move into host's turn.
	for h.s.Match.Turn != 0 {
		c.place(c.s.Match.LegalMoves()[0])
		pump(t, h, c, func() bool { return h.s.Match.Turn == 0 || h.s.Phase != phasePlay })
		if h.s.Phase != phasePlay {
			return
		}
	}
	now := time.Now()
	h.lastAimAt = now
	h.s.Aiming = false
	h.onMsg(netMsg{Type: msgAim, Col: 0, Die: 0}, now)
	// Above wouldn't accept Turn != 1; verify direct aim coalesces within aimMinInterval.
	h.aim(1, 0)
	if !h.aimPending {
		t.Fatal("expected aim to coalesce inside aimMinInterval")
	}
	h.tick(now.Add(aimMinInterval + time.Millisecond))
	if h.aimPending {
		t.Fatal("expected tick to flush coalesced aim")
	}
}

func pumpAll(t *testing.T, h *netHost, clients []*netClient, done func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !done() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for clients")
		}
		progressed := false
		for {
			select {
			case ev := <-h.events:
				h.handle(ev, time.Now())
				progressed = true
			default:
				goto drainedHost
			}
		}
	drainedHost:
		for _, cl := range clients {
			for {
				select {
				case ev := <-cl.events:
					cl.handle(ev, time.Now())
					progressed = true
				default:
					goto drainedClient
				}
			}
		drainedClient:
		}
		if !progressed {
			time.Sleep(10 * time.Millisecond)
			now := time.Now()
			h.tick(now)
			for _, cl := range clients {
				cl.tick(now)
			}
		}
	}
}

func TestRoomCodeAuthAndHMAC(t *testing.T) {
	h, err := startHost("Host", 0, hostConfig{BestOf: 1, BindIP: "127.0.0.1", RoomCode: "LAMB-42"}, testRNG())
	if err != nil {
		t.Fatal(err)
	}
	defer h.leave()
	addr := h.ln.Addr().String()
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Fatalf("expected 127.0.0.1 bind, got %s", addr)
	}

	// Missing room code -> errBadAuth.
	missing := startClient("NoCode", addr)
	defer missing.leave()
	pumpAll(t, h, []*netClient{missing}, func() bool { return missing.errText != "" })
	if missing.errText != errBadAuth {
		t.Fatalf("missing room code got %q, want %q", missing.errText, errBadAuth)
	}

	// Wrong room code -> errBadAuth.
	wrong := startClientWithCode("Wrong", addr, "WRONG")
	defer wrong.leave()
	pumpAll(t, h, []*netClient{wrong}, func() bool { return wrong.errText != "" })
	if wrong.errText != errBadAuth {
		t.Fatalf("wrong room code got %q, want %q", wrong.errText, errBadAuth)
	}

	// Correct room code connects cleanly.
	good := startClientWithCode("Guest", addr, "LAMB-42")
	defer good.leave()
	pump(t, h, good, func() bool { return good.s != nil && h.s != nil })

	// Play to completion with HMAC-signed messages.
	for h.s.Phase == phasePlay {
		moves := h.s.Match.Moves
		if h.s.Match.Turn == 0 {
			h.place(h.s.Match.LegalMoves()[0])
		} else {
			good.place(good.s.Match.LegalMoves()[0])
		}
		pump(t, h, good, func() bool { return good.s.Match.Moves > moves || good.s.Phase != phasePlay })
	}
	pump(t, h, good, func() bool { return good.s.Phase == phaseGameOver })
	if good.fairness() != fairVerified {
		t.Fatalf("expected fairVerified after completed round, got %d", good.fairness())
	}

	// Tampered wire MAC is rejected on host.
	badMsg := netMsg{Type: msgPing, Seq: h.rxSeq + 1, MAC: strings.Repeat("00", 32)}
	if h.onMsg(badMsg, time.Now()) {
		t.Fatal("expected tampered HMAC message to be rejected")
	}
	if h.authed {
		t.Fatal("expected host to drop authentication on invalid wire MAC")
	}
}

func TestOpenRoomRejectsUnexpectedRoomCode(t *testing.T) {
	h, err := startHost("Host", 0, hostConfig{BestOf: 1}, testRNG())
	if err != nil {
		t.Fatal(err)
	}
	defer h.leave()

	withCode := startClientWithCode("Guest", fmt.Sprintf("127.0.0.1:%d", h.port), "SECRET")
	defer withCode.leave()
	pumpAll(t, h, []*netClient{withCode}, func() bool { return withCode.errText != "" })
	if withCode.errText != errBadAuth {
		t.Fatalf("unexpected-code client got %q, want %q", withCode.errText, errBadAuth)
	}
}

func TestProtocolV1Rejected(t *testing.T) {
	h, err := startHost("Host", 0, hostConfig{BestOf: 1}, testRNG())
	if err != nil {
		t.Fatal(err)
	}
	defer h.leave()

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", h.port), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	done := make(chan struct{})
	defer close(done)
	events := make(chan netEvent, 8)
	go readLoop(1, conn, events, done)

	// Wait for host challenge while pumping host.
	deadline := time.Now().Add(3 * time.Second)
	gotChallenge := false
	for !gotChallenge && time.Now().Before(deadline) {
		select {
		case ev := <-h.events:
			h.handle(ev, time.Now())
		case ev := <-events:
			if ev.kind == evMsg && ev.msg.Type == msgChallenge {
				if ev.msg.Version != protocolVersion {
					t.Fatalf("challenge version = %d, want %d", ev.msg.Version, protocolVersion)
				}
				gotChallenge = true
			}
		case <-time.After(5 * time.Millisecond):
		}
	}
	if !gotChallenge {
		t.Fatal("did not receive msgChallenge")
	}

	if err := sendMsg(conn, netMsg{
		Type:    msgHello,
		Version: 1,
		Name:    "OldClient",
		Token:   newToken(),
		Seed:    newRandomHex(32),
	}); err != nil {
		t.Fatal(err)
	}

	var gotErr string
	for gotErr == "" && time.Now().Before(deadline) {
		select {
		case ev := <-h.events:
			h.handle(ev, time.Now())
		case ev := <-events:
			if ev.kind == evMsg && ev.msg.Type == msgError {
				gotErr = ev.msg.Text
			}
		case <-time.After(5 * time.Millisecond):
		}
	}
	if gotErr != errOldProtocol {
		t.Fatalf("v1 hello got %q, want %q", gotErr, errOldProtocol)
	}
}

func TestFairPlayMultiRoundAndCheatDetection(t *testing.T) {
	h, c := startPair(t, hostConfig{BestOf: 3})

	playRound := func() {
		for h.s.Phase == phasePlay {
			moves := h.s.Match.Moves
			if h.s.Match.Turn == 0 {
				h.place(h.s.Match.LegalMoves()[0])
			} else {
				c.place(c.s.Match.LegalMoves()[0])
			}
			pump(t, h, c, func() bool { return c.s.Match.Moves > moves || c.s.Phase != phasePlay })
		}
	}

	playRound()
	if c.fairness() != fairVerified {
		t.Fatalf("round 1 fairness = %d, want fairVerified", c.fairness())
	}

	// Continue to round 2 (or rematch if needed) and verify commit rotation.
	if h.s.Phase == phaseRoundOver {
		h.vote(voteContinue)
		c.vote(voteContinue)
		pump(t, h, c, func() bool { return c.s.Round == 2 && c.s.Phase == phasePlay })
		playRound()
		if c.fairness() != fairVerified {
			t.Fatalf("round 2 fairness = %d, want fairVerified", c.fairness())
		}
	}

	// Cheat detection on synthetic forged reveal / forged rolls.
	h2, c2 := startPair(t, hostConfig{BestOf: 1})
	badState := *c2.s
	badState.Phase = phaseRoundOver
	badState.HostReveal = newRandomHex(32) // does not match expectedCommit
	badState.RoundRolls = [][2]int{{1, 0}}
	c2.checkFairPlay(&badState)
	if c2.fairness() != fairFailed {
		t.Fatalf("expected fairFailed on bad reveal, got %d", c2.fairness())
	}
	_ = h2
}
