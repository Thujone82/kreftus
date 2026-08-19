package main

import (
	"strconv"
	"strings"
)

const (
	replayOpPlace      byte = 'P'
	replayOpClear      byte = 'C'
	replayOpMark       byte = 'M'
	replayOpClearMarks byte = 'X'

	replayPlayMs           = 5000
	replayCelebrateMs      = 3000
	replayCelebrateFrameMs = 200 // 5 fps
	replayLoopMs           = replayPlayMs + replayCelebrateMs
)

type replayEvent struct {
	Cell  int
	Op    byte
	Digit byte
}

type fastestReplay struct {
	ID              string `json:"id"`
	Givens          string `json:"givens"`
	Events          string `json:"events,omitempty"`
	StartGrid       string `json:"startGrid,omitempty"`
	StartPencilTop  string `json:"startPencilTop,omitempty"`
	StartPencilBot  string `json:"startPencilBot,omitempty"`
	StartPencilSlot string `json:"startPencilSlot,omitempty"`
}

type replayStart struct {
	Grid string
	Top  string
	Bot  string
	Slot string
}

func (s replayStart) active(givens string) bool {
	if s.Grid != "" && s.Grid != givens {
		return true
	}
	return pencilStringsActive(s.Top, s.Bot)
}

func pencilStringsActive(top, bot string) bool {
	for i := 0; i < len(top); i++ {
		if top[i] >= '1' && top[i] <= '9' {
			return true
		}
	}
	for i := 0; i < len(bot); i++ {
		if bot[i] >= '1' && bot[i] <= '9' {
			return true
		}
	}
	return false
}

func boardProgressed(grid, givens, top, bot string) bool {
	if grid != "" && grid != givens {
		return true
	}
	return pencilStringsActive(top, bot)
}

func snapshotReplayStart(b *board) replayStart {
	top, bot, slot := b.pencilsString()
	return replayStart{Grid: b.gridString(), Top: top, Bot: bot, Slot: slot}
}

func (s replayStart) applyToReplay(fr *fastestReplay) {
	if fr == nil || !s.active(fr.Givens) {
		return
	}
	fr.StartGrid = s.Grid
	fr.StartPencilTop = s.Top
	fr.StartPencilBot = s.Bot
	fr.StartPencilSlot = s.Slot
}

func replayStartFromContinue(c *continueGame) replayStart {
	if c == nil {
		return replayStart{}
	}
	return replayStart{
		Grid: c.StartGrid,
		Top:  c.StartPencilTop,
		Bot:  c.StartPencilBot,
		Slot: c.StartPencilSlot,
	}
}

func replayStartFromFastest(fr *fastestReplay) replayStart {
	if fr == nil {
		return replayStart{}
	}
	return replayStart{
		Grid: fr.StartGrid,
		Top:  fr.StartPencilTop,
		Bot:  fr.StartPencilBot,
		Slot: fr.StartPencilSlot,
	}
}

func (s replayStart) applyToContinue(c *continueGame) {
	if c == nil || !s.active(c.Givens) {
		return
	}
	c.StartGrid = s.Grid
	c.StartPencilTop = s.Top
	c.StartPencilBot = s.Bot
	c.StartPencilSlot = s.Slot
}

func encodeReplay(events []replayEvent) string {
	if len(events) == 0 {
		return ""
	}
	var b strings.Builder
	for i, e := range events {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Itoa(e.Cell))
		b.WriteByte(e.Op)
		if e.Op == replayOpPlace || e.Op == replayOpMark {
			b.WriteByte(e.Digit)
		}
	}
	return b.String()
}

func decodeReplay(s string) []replayEvent {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]replayEvent, 0, len(parts))
	for _, p := range parts {
		e, ok := parseReplayToken(p)
		if ok {
			out = append(out, e)
		}
	}
	return out
}

func parseReplayToken(p string) (replayEvent, bool) {
	i := 0
	for i < len(p) && p[i] >= '0' && p[i] <= '9' {
		i++
	}
	if i == 0 || i >= len(p) {
		return replayEvent{}, false
	}
	cell, err := strconv.Atoi(p[:i])
	if err != nil || cell < 0 || cell > 80 {
		return replayEvent{}, false
	}
	op := p[i]
	switch op {
	case replayOpPlace, replayOpMark:
		if i+1 >= len(p) {
			return replayEvent{}, false
		}
		d := p[i+1]
		if d < '1' || d > '9' {
			return replayEvent{}, false
		}
		return replayEvent{Cell: cell, Op: op, Digit: d}, true
	case replayOpClear, replayOpClearMarks:
		return replayEvent{Cell: cell, Op: op}, true
	default:
		return replayEvent{}, false
	}
}

func applyReplay(b *board, events []replayEvent) {
	for _, e := range events {
		b.cursor = e.Cell
		switch e.Op {
		case replayOpPlace:
			b.place(e.Digit)
		case replayOpClear:
			b.clear()
		case replayOpMark:
			b.markPencil(e.Digit)
		case replayOpClearMarks:
			b.clearPencil()
		}
	}
}

func replayAppliedCount(n int, elapsedMs int64) int {
	if n <= 0 {
		return 0
	}
	if elapsedMs < 0 {
		elapsedMs = 0
	}
	t := elapsedMs % int64(replayLoopMs)
	if t >= replayPlayMs {
		return n
	}
	return int(t) * n / replayPlayMs
}

func replayCelebrateShift(elapsedMs int64) (shift int, celebrating bool) {
	if elapsedMs < 0 {
		elapsedMs = 0
	}
	t := elapsedMs % int64(replayLoopMs)
	if t < replayPlayMs {
		return 0, false
	}
	frame := int((t - replayPlayMs) / replayCelebrateFrameMs)
	return frame, true
}

func replayBoardAt(givens, solution string, start replayStart, events []replayEvent, n int) board {
	grid := givens
	if start.Grid != "" {
		grid = start.Grid
	}
	b := newBoard(givens, solution, grid)
	b.loadPencils(start.Top, start.Bot, start.Slot)
	if n > len(events) {
		n = len(events)
	}
	if n > 0 {
		applyReplay(&b, events[:n])
	}
	return b
}
