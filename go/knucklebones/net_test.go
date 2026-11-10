package main

import (
	"fmt"
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
	close(c.done)
	c.conn.Close()
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
		}
	}
	if other.errText != errRoomFull {
		t.Fatalf("third player got %q", other.errText)
	}
}
