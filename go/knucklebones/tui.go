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
	for {
		var netCh <-chan netEvent
		if g.ctl != nil {
			netCh = g.ctl.netEvents()
		}
		select {
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
			g.afterChange()
		case ev := <-netCh:
			if g.ctl != nil {
				g.ctl.handle(ev, time.Now())
			}
			g.afterChange()
		case now := <-tick.C:
			if g.ctl != nil {
				g.ctl.tick(now)
				g.afterChange()
			}
		case <-sigChan:
			return
		}
	}
}
