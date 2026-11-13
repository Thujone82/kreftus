package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
)

func runTUI() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Print("\033[0m")
			fmt.Print("\033[?25h")
			panic(r)
		}
	}()

	enableUTF8Console()
	resizeToPreferred()

	s, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "knucklebones: %v\n", err)
		os.Exit(1)
	}
	if err := s.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "knucklebones: %v\n", err)
		os.Exit(1)
	}
	enableUTF8Console()
	resizeToPreferred()
	s.Sync()
	s.DisableMouse()
	s.DisablePaste()
	s.HideCursor()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	g := newGame()
	g.screen = s
	g.width, g.height = s.Size()

	cleanup := func() {
		g.leaveGame()
		fmt.Print("\033[0m")
		fmt.Print("\033[?25h")
		s.Fini()
		fmt.Print("\033[H\033[2J")
	}
	defer cleanup()

	setTerminalTitle("Knucklebones")

	events := make(chan tcell.Event, 64)
	go func() {
		for {
			ev := s.PollEvent()
			if ev == nil {
				return
			}
			switch ev.(type) {
			case *tcell.EventKey, *tcell.EventResize:
				events <- ev
			}
		}
	}()

	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()

	g.render()
	lastSig, _ := tuiSnapshot(g, time.Now())
	var lastBeep time.Time
	playTUI := func(now time.Time) {
		for _, id := range g.drainSounds() {
			if playPCMSound(id) {
				continue
			}
			if id.tuiAudible() && now.Sub(lastBeep) >= 200*time.Millisecond {
				_ = s.Beep()
				lastBeep = now
			}
		}
	}
	for {
		var netCh <-chan netEvent
		if g.ctl != nil {
			netCh = g.ctl.netEvents()
		}
		var watchCh <-chan struct{}
		if g.watcher != nil {
			watchCh = g.watcher.Notify()
		}
		select {
		case <-watchCh:
			now := time.Now()
			sig, _ := tuiSnapshot(g, now)
			if sig != lastSig {
				g.afterChange()
				playTUI(now)
				lastSig, _ = tuiSnapshot(g, now)
			}
		case ev := <-events:
			switch e := ev.(type) {
			case *tcell.EventResize:
				g.width, g.height = s.Size()
				s.Sync()
			case *tcell.EventKey:
				if g.handleKey(e) {
					return
				}
			}
			now := time.Now()
			g.afterChange()
			playTUI(now)
			lastSig, _ = tuiSnapshot(g, now)
		case ev := <-netCh:
			changed := false
			now := time.Now()
			if g.ctl != nil {
				changed = g.ctl.handle(ev, now)
			}
			sig, anim := tuiSnapshot(g, now)
			if changed || anim || sig != lastSig {
				g.afterChange()
				playTUI(now)
				lastSig, _ = tuiSnapshot(g, time.Now())
			}
		case now := <-tick.C:
			changed := false
			if g.ctl != nil {
				changed = g.ctl.tick(now)
			}
			sig, anim := tuiSnapshot(g, now)
			if changed || anim || sig != lastSig {
				g.afterChange()
				playTUI(now)
				lastSig, _ = tuiSnapshot(g, time.Now())
			}
		case <-sigChan:
			return
		}
	}
}

type tuiSig struct {
	w, h          int
	view          int
	theme         string
	sound         bool
	colorblind    bool
	winMeter      bool
	replayActive  bool
	replayStep    int
	fairness      int
	lobbyClock     int64
	lobbyErr       string
	lobbyBackFocus bool
	netOnline      bool
	netFocus       int
	netRoomIdx     int
	netUpdatedNs   int64
	netRooms       int
	netStatsKey    uint64
	turnClock     int64
	sessPhase     string
	sessMoves     int
	sessTurn      int
	sessRound     int
	sessSeries    int
	sessConnected [2]bool
	sessVotes     [2]string
	sessForfeit   bool
	turnSec       int
	discSec       int
	aiming        bool
	aimCol        int
	aimDie        int
	oppGone       bool
}

func tuiSnapshot(g *game, now time.Time) (tuiSig, bool) {
	sig := tuiSig{
		w:            g.width,
		h:            g.height,
		view:         g.view,
		theme:        g.save.Theme,
		sound:        g.save.Settings.Sound,
		colorblind:   g.save.Settings.Colorblind,
		winMeter:     g.save.Settings.WinMeter,
		replayActive: g.replayActive,
		replayStep:   g.replayStep,
	}
	anim := false
	switch g.view {
	case viewNetwork:
		sig.netOnline = g.netOnline
		sig.netFocus = g.netFocus
		sig.netRoomIdx = g.netRoomIdx
		if g.netOnline {
			snap := g.networkSnapshot()
			sig.netUpdatedNs = snap.UpdatedAt.UnixNano()
			sig.netRooms = len(snap.Rooms)
			sig.netStatsKey = uint64(snap.Stats.LiveMatches)*1000003 + snap.Stats.CompletedMatches*1009 + snap.Stats.TotalDicePlayed
			if snap.Err != "" {
				sig.lobbyErr = snap.Err
			}
			if len(snap.Rooms) > 0 {
				sig.lobbyClock = now.Unix()
			}
		}
	case viewLobby:
		if l, ok := g.ctl.(lobby); ok {
			sig.lobbyErr = l.lobbyError()
			sig.lobbyBackFocus = g.lobbyBackFocus
			if l.hostPort() > 0 || l.hostRoomID() != "" {
				sig.lobbyClock = int64(time.Since(l.waitingSince()).Seconds())
			} else if sig.lobbyErr == "" {
				sig.lobbyClock = now.UnixMilli() / 400
			}
		}
	case viewPlay:
		if g.ctl != nil {
			sig.oppGone = g.ctl.opponentGone()
			sig.fairness = g.ctl.fairness()
			if s := g.ctl.state(); s != nil {
				sig.sessPhase = s.Phase
				sig.sessMoves = s.Match.Moves
				sig.sessTurn = s.Match.Turn
				sig.sessRound = s.Round
				sig.sessSeries = s.Series
				sig.sessConnected = s.Connected
				sig.sessVotes = s.Votes
				sig.sessForfeit = s.Forfeit != nil
				sig.aiming = s.Aiming
				sig.aimCol = s.AimCol
				sig.aimDie = s.AimDie
				if left := g.ctl.turnLeft(now); left > 0 {
					sig.turnSec = int(left.Seconds() + 0.999)
				}
				if p, d := g.ctl.disconnectLeft(now); p >= 0 {
					sig.discSec = p*10000 + int(d.Seconds()+0.999)
				}
				if s.Phase == phasePlay && !g.myTurn(s) && !g.rolling() && !g.replayActive {
					sig.turnClock = now.UnixMilli() / 350
				}
			}
			if g.rolling() || effectProgress(g.effectAt, now) >= 0 {
				anim = true
			}
		}
	}
	return sig, anim
}
