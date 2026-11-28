package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"golang.org/x/term"
)

type app struct {
	sess    *termSession
	cfgPath string
	st      settings
	mute    bool
	snd     *player
}

func (a *app) currentTheme() theme {
	return themeByID(a.st.Theme)
}

func (a *app) syncSound() {
	a.snd.muted = a.mute || !a.st.Sound
}

func (a *app) save() {
	_ = saveSettings(a.cfgPath, a.st)
}

func (a *app) quit() {
	if a.sess != nil {
		a.sess.restore()
	}
	os.Exit(0)
}

func (a *app) draw(build func(theme) *canvas) {
	th := a.currentTheme()
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 || h <= 0 {
		w, h = boardW, boardH
	}
	if w < boardW || h < boardH {
		flushCanvas(renderTooSmall(w, h, th), w, h, th.BG)
		return
	}
	flushCanvas(build(th), w, h, th.BG)
}

func (a *app) handleGlobalKey(ev keyEvent) (handled bool, quit bool) {
	switch ev.kind {
	case keyQuit:
		return true, true
	case keyTheme:
		a.st.Theme = nextTheme(a.st.Theme)
		a.save()
		a.snd.play(sndTheme)
		return true, false
	case keySound:
		a.mute = false // toggling sound manually overrides initial -mute flag
		a.st.Sound = !a.st.Sound
		a.syncSound()
		a.save()
		a.snd.play(sndClick)
		return true, false
	case keyLabels:
		a.st.Labels = !a.st.Labels
		a.save()
		a.snd.play(sndClick)
		return true, false
	case keyLabelMode:
		a.st.LabelNumbers = !a.st.LabelNumbers
		a.save()
		a.snd.play(sndClick)
		return true, false
	}
	return false, false
}

func main() {
	setCode := flag.String("set", "", "4-peg code for another player to guess (e.g. r22m)")
	themeFlag := flag.String("theme", "", "theme ID (dark, light, neon, ocean, wood, contrast)")
	muteFlag := flag.Bool("mute", false, "disable sound effects for this run")
	previewFlag := flag.Bool("preview", false, "print sample boards for all 6 themes and exit")
	flag.Parse()

	enableUTF8Console()

	if *previewFlag {
		fmt.Print(renderPreviewString())
		return
	}

	if *setCode != "" {
		if _, err := parseSetCode(*setCode); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		runPlain(bufio.NewReader(os.Stdin), *setCode)
		return
	}

	cfgPath := settingsPath()
	st := loadSettings(cfgPath)
	if *themeFlag != "" {
		if !validTheme(*themeFlag) {
			fmt.Fprintf(os.Stderr, "mind: unknown theme %q (valid: dark, light, neon, ocean, wood, contrast)\n", *themeFlag)
			os.Exit(1)
		}
		st.Theme = *themeFlag
		_ = saveSettings(cfgPath, st)
	}

	sess, err := openTermSession()
	if err != nil {
		runPlain(bufio.NewReader(os.Stdin), *setCode)
		return
	}
	defer sess.restore()

	a := &app{
		sess:    sess,
		cfgPath: cfgPath,
		st:      st,
		mute:    *muteFlag,
		snd:     &player{},
	}
	a.syncSound()

	defer func() {
		if r := recover(); r != nil {
			sess.restore()
			panic(r)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		a.quit()
	}()

	if !a.runTitleScreen() {
		return
	}

	firstSecret := []byte(nil)
	if *setCode != "" {
		firstSecret, _ = parseSetCode(*setCode)
	}

	for {
		var secret []byte
		if firstSecret != nil {
			secret = firstSecret
			firstSecret = nil
		} else {
			secret = generateSecret()
		}
		if !a.runGame(secret) {
			return
		}
	}
}

func (a *app) runTitleScreen() bool {
	animTick := 0
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()

	drawTitle := func() {
		a.draw(func(th theme) *canvas {
			return renderTitleScreen(a.st, th, animTick)
		})
	}
	drawTitle()

	for {
		select {
		case ev, ok := <-a.sess.keys:
			if !ok {
				return false
			}
			if handled, quit := a.handleGlobalKey(ev); handled {
				if quit {
					return false
				}
				drawTitle()
				continue
			}
			if ev.kind == keyEnter {
				a.snd.play(sndStart)
				return true
			}
		case <-ticker.C:
			animTick++
			drawTitle()
		}
	}
}

func (a *app) runGame(secret []byte) bool {
	g := newGameState(secret)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	drawBoard := func() {
		now := time.Now()
		a.draw(func(th theme) *canvas {
			return renderBoard(g, a.st, th, now)
		})
	}
	drawBoard()

	for !g.over {
		select {
		case ev, ok := <-a.sess.keys:
			if !ok {
				return false
			}
			if handled, quit := a.handleGlobalKey(ev); handled {
				if quit {
					return false
				}
				drawBoard()
				continue
			}
			switch ev.kind {
			case keyPeg:
				if len(g.current) < codeLength {
					g.current = append(g.current, ev.peg)
					g.statusMsg = ""
					a.snd.play(sndClick)
					drawBoard()
				}
			case keyBackspace:
				if len(g.current) > 0 {
					g.current = g.current[:len(g.current)-1]
					g.statusMsg = ""
					a.snd.play(sndDelete)
					drawBoard()
				}
			case keyEnter:
				if len(g.current) < codeLength {
					g.statusMsg = "Need 4 pegs!"
					g.statusIsErr = true
					g.statusUntil = time.Now().Add(1 * time.Second)
					a.snd.play(sndInvalid)
					drawBoard()
					continue
				}
				guess := append([]byte(nil), g.current...)
				g.current = g.current[:0]
				g.statusMsg = ""
				exact, partial := score(g.secret, guess)
				g.turns = append(g.turns, turnRecord{Guess: guess, Exact: exact, Partial: partial})
				a.snd.play(sndSubmit)

				if !a.animateFeedback(g, exact, partial, drawBoard) {
					return false
				}

				if exact == codeLength || len(g.turns) == maxTurns {
					g.over = true
					g.won = exact == codeLength
					g.endTime = time.Now()
				}
				drawBoard()
			}
		case <-ticker.C:
			drawBoard()
		}
	}

	if !a.animateEndGame(g, drawBoard) {
		return false
	}

	secs := int(g.elapsed().Round(time.Second) / time.Second)
	a.st.recordResult(g.won, len(g.turns), secs)
	a.save()

	g.overlayVisible = true
	drawBoard()

	for {
		select {
		case ev, ok := <-a.sess.keys:
			if !ok {
				return false
			}
			if handled, quit := a.handleGlobalKey(ev); handled {
				if quit {
					return false
				}
				drawBoard()
				continue
			}
			if ev.kind == keyNew || ev.kind == keyEnter {
				a.snd.play(sndStart)
				return true
			}
		case <-ticker.C:
			drawBoard()
		}
	}
}

// drainOrQuit checks pending keys during animations, returning false if the user quits.
func (a *app) drainOrQuit() bool {
	for {
		select {
		case ev, ok := <-a.sess.keys:
			if !ok {
				return false
			}
			if ev.kind == keyQuit {
				return false
			}
		default:
			return true
		}
	}
}

func (a *app) sleepAnim(d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	for {
		select {
		case ev, ok := <-a.sess.keys:
			if !ok {
				return false
			}
			if ev.kind == keyQuit {
				return false
			}
		case <-timer.C:
			return true
		}
	}
}

func (a *app) animateFeedback(g *gameState, exact, partial int, drawBoard func()) bool {
	g.animFeedback = true
	g.animShown = 0
	drawBoard()

	total := exact + partial
	if total == 0 {
		if !a.sleepAnim(130 * time.Millisecond) {
			return false
		}
		g.animShown = codeLength
		a.snd.play(sndMiss)
		drawBoard()
		g.animFeedback = false
		return a.drainOrQuit()
	}

	for i := 1; i <= total; i++ {
		if !a.sleepAnim(130 * time.Millisecond) {
			return false
		}
		g.animShown = i
		if i <= exact {
			a.snd.play(sndExact)
		} else {
			a.snd.play(sndPartial)
		}
		drawBoard()
	}
	g.animFeedback = false
	return a.drainOrQuit()
}

func (a *app) animateEndGame(g *gameState, drawBoard func()) bool {
	for i := 1; i <= codeLength; i++ {
		if !a.sleepAnim(150 * time.Millisecond) {
			return false
		}
		g.revealedPegs = i
		a.snd.play(sndReveal)
		drawBoard()
	}

	if g.won {
		a.snd.play(sndWin)
		for step := 0; step < 8; step++ {
			g.celebrateStep = step
			drawBoard()
			if !a.sleepAnim(150 * time.Millisecond) {
				return false
			}
		}
		g.celebrateStep = -1
	} else {
		a.snd.play(sndLose)
	}
	return a.drainOrQuit()
}
