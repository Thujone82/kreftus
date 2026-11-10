package main

import "github.com/gdamore/tcell/v2"

// guiAction is a clickable button that stands in for a key press, so the
// window drives the same handlers as the terminal.
type guiAction struct {
	label string
	key   *tcell.EventKey
}

func keyEv(k tcell.Key) *tcell.EventKey { return tcell.NewEventKey(k, 0, tcell.ModNone) }
func runeEv(r rune) *tcell.EventKey     { return tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone) }

// playActions are the buttons under the board.
func (g *game) playActions(s *session) []guiAction {
	var out []guiAction
	if s.Phase != phasePlay {
		label := "View board"
		if g.hideResult {
			label = "Show result"
		}
		out = append(out, guiAction{label, runeEv('v')})
	}
	out = append(out, guiAction{"Log", runeEv('l')}, guiAction{"History", runeEv('h')})
	if g.canResign(s) {
		out = append(out, guiAction{"Resign", runeEv('r')})
	}
	leave := "Leave"
	if s.Phase == phaseGameOver || g.ctl.opponentGone() {
		leave = "Menu"
	}
	return append(out, guiAction{leave, keyEv(tcell.KeyEscape)})
}

// settingClick is the key a click on a settings row sends.
func settingClick(row int) *tcell.EventKey {
	switch row {
	case rowStart:
		return keyEv(tcell.KeyEnter)
	case rowPlayer2, rowPort:
		return nil
	}
	return keyEv(tcell.KeyRight)
}
