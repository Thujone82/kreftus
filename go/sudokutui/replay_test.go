package main

import (
	"reflect"
	"testing"
	"time"
)

func TestEncodeDecodeReplay(t *testing.T) {
	events := []replayEvent{
		{Cell: 2, Op: replayOpPlace, Digit: '5'},
		{Cell: 14, Op: replayOpMark, Digit: '3'},
		{Cell: 14, Op: replayOpMark, Digit: '3'},
		{Cell: 2, Op: replayOpClear},
		{Cell: 0, Op: replayOpClearMarks},
	}
	s := encodeReplay(events)
	if s != "2P5,14M3,14M3,2C,0X" {
		t.Fatalf("encode=%q", s)
	}
	got := decodeReplay(s)
	if !reflect.DeepEqual(got, events) {
		t.Fatalf("decode=%v want %v", got, events)
	}
	if encodeReplay(nil) != "" || decodeReplay("") != nil {
		t.Fatal("empty round trip")
	}
}

func TestApplyReplayMatchesLiveBoard(t *testing.T) {
	givens, sol := classicSolved()
	live := newBoard(givens, sol, givens)
	var events []replayEvent
	log := func(op, digit byte) {
		events = append(events, replayEvent{Cell: live.cursor, Op: op, Digit: digit})
	}

	live.cursor = 2
	if !live.markPencil('1') {
		t.Fatal("mark")
	}
	log(replayOpMark, '1')
	if !live.place(sol[2]) {
		t.Fatal("place")
	}
	log(replayOpPlace, sol[2])

	live.cursor = 3
	if !live.markPencil('2') || !live.markPencil('4') {
		t.Fatal("two marks")
	}
	log(replayOpMark, '2')
	log(replayOpMark, '4')
	if !live.clearPencil() {
		t.Fatal("clear marks")
	}
	log(replayOpClearMarks, 0)

	live.cursor = 5
	wrong := byte('9')
	if sol[5] == wrong {
		wrong = '1'
	}
	if !live.place(wrong) {
		t.Fatal("wrong place")
	}
	log(replayOpPlace, wrong)
	if !live.clear() {
		t.Fatal("clear")
	}
	log(replayOpClear, 0)

	replayed := newBoard(givens, sol, givens)
	applyReplay(&replayed, events)
	if live.grid != replayed.grid {
		t.Fatalf("grid mismatch\nlive=%s\nrep =%s", live.gridString(), replayed.gridString())
	}
	if live.pencil != replayed.pencil || live.pencilSlot != replayed.pencilSlot {
		t.Fatal("pencil mismatch")
	}
	if live.mistakes != replayed.mistakes {
		t.Fatalf("mistakes=%d want %d", replayed.mistakes, live.mistakes)
	}
}

func TestReplayAppliedCountTiming(t *testing.T) {
	n := 100
	if replayAppliedCount(n, 0) != 0 {
		t.Fatal("start should show givens")
	}
	if got := replayAppliedCount(n, 250); got != 5 {
		t.Fatalf("250ms=%d want 5", got)
	}
	if got := replayAppliedCount(n, 50); got != 1 {
		t.Fatalf("50ms=%d want 1", got)
	}
	if got := replayAppliedCount(n, 4999); got != 99 {
		t.Fatalf("4999ms=%d want 99", got)
	}
	for _, ms := range []int64{5000, 5001, 7999} {
		if got := replayAppliedCount(n, ms); got != n {
			t.Fatalf("%dms=%d want complete %d", ms, got, n)
		}
	}
	if got := replayAppliedCount(n, 8000); got != 0 {
		t.Fatalf("8000ms=%d want reset", got)
	}
	if replayAppliedCount(0, 1000) != 0 {
		t.Fatal("no events")
	}
}

func TestReplayCelebrateShift(t *testing.T) {
	if _, ok := replayCelebrateShift(4999); ok {
		t.Fatal("playback should not celebrate yet")
	}
	shift, ok := replayCelebrateShift(5000)
	if !ok || shift != 0 {
		t.Fatalf("celebrate start shift=%d ok=%v", shift, ok)
	}
	shift, ok = replayCelebrateShift(5200)
	if !ok || shift != 1 {
		t.Fatalf("5fps frame 1: shift=%d ok=%v", shift, ok)
	}
	shift, ok = replayCelebrateShift(7999)
	if !ok || shift != 14 {
		t.Fatalf("last celebrate frame: shift=%d ok=%v", shift, ok)
	}
	if _, ok := replayCelebrateShift(8000); ok {
		t.Fatal("loop should restart playback")
	}
}

func TestReplayBoardAtPrefix(t *testing.T) {
	givens, sol := classicSolved()
	events := []replayEvent{
		{Cell: 2, Op: replayOpPlace, Digit: sol[2]},
		{Cell: 3, Op: replayOpMark, Digit: '1'},
	}
	start := replayBoardAt(givens, sol, replayStart{}, events, 0)
	if start.gridString() != givens {
		t.Fatal("zero events should be givens")
	}
	one := replayBoardAt(givens, sol, replayStart{}, events, 1)
	if one.grid[2] != sol[2] {
		t.Fatal("first event not applied")
	}
	if one.hasPencil(3) {
		t.Fatal("second event should not be applied yet")
	}
	two := replayBoardAt(givens, sol, replayStart{}, events, 2)
	if !two.hasPencil(3) {
		t.Fatal("second event should mark cell 3")
	}
}

func TestReplayBoardAtStartSnapshot(t *testing.T) {
	givens, sol := classicSolved()
	mid := []byte(givens)
	mid[2] = sol[2]
	start := replayStart{Grid: string(mid)}
	events := []replayEvent{{Cell: 3, Op: replayOpPlace, Digit: sol[3]}}
	zero := replayBoardAt(givens, sol, start, events, 0)
	if zero.grid[2] != sol[2] {
		t.Fatal("frame 0 should jump to the snapshot")
	}
	if zero.grid[3] == sol[3] {
		t.Fatal("later events should not apply at frame 0")
	}
	if zero.gridString() == givens {
		t.Fatal("snapshot must not start from givens")
	}
	one := replayBoardAt(givens, sol, start, events, 1)
	if one.grid[2] != sol[2] || one.grid[3] != sol[3] {
		t.Fatal("events apply on top of the snapshot")
	}
}

func TestApplyDigitLogsReplay(t *testing.T) {
	chdirTemp(t)
	givens, sol := classicSolved()
	g := &game{
		puzzle: puzzleEntry{ID: testPuzzleID, Givens: givens, Solution: sol},
		board:  newBoard(givens, sol, givens),
		save:   newSaveData(),
		view:   viewPlay,
	}
	g.board.cursor = 2
	g.applyDigit(sol[2])
	if len(g.replayEvents) != 1 || g.replayEvents[0].Op != replayOpPlace || g.replayEvents[0].Cell != 2 {
		t.Fatalf("place log=%v", g.replayEvents)
	}
	g.pencil = true
	g.board.cursor = 3
	g.applyDigit('1')
	if len(g.replayEvents) != 2 || g.replayEvents[1].Op != replayOpMark {
		t.Fatalf("mark log=%v", g.replayEvents)
	}
	if !g.clearPlay() || g.replayEvents[2].Op != replayOpClearMarks {
		t.Fatalf("clear marks log=%v", g.replayEvents)
	}
	g.pencil = false
	g.board.cursor = 5
	wrong := byte('9')
	if sol[5] == wrong {
		wrong = '1'
	}
	g.applyDigit(wrong)
	if len(g.replayEvents) != 4 || g.replayEvents[3].Op != replayOpPlace {
		t.Fatalf("wrong place log=%v", g.replayEvents)
	}
	if !g.clearPlay() || g.replayEvents[4].Op != replayOpClear {
		t.Fatalf("clear log=%v", g.replayEvents)
	}
}

func TestFinishSuccessKeepsReplayForOverlay(t *testing.T) {
	chdirTemp(t)
	givens, sol := classicSolved()
	g := &game{
		save:           newSaveData(),
		difficulty:     diffEasy,
		puzzle:         puzzleEntry{ID: testPuzzleID, Givens: givens, Solution: sol},
		board:          newBoard(givens, sol, sol),
		view:           viewPlay,
		elapsed:        45 * time.Second,
		replayEvents:   []replayEvent{{Cell: 2, Op: replayOpPlace, Digit: sol[2]}},
		replayGivens:   givens,
		replaySolution: sol,
	}
	g.finishSuccess()
	if encodeReplay(g.replayEvents) == "" {
		t.Fatal("solved overlay still needs the session log")
	}
	if g.replayGivens != givens || g.replaySolution != sol {
		t.Fatal("playback snapshot missing")
	}
}

func menuGameWithReplay() *game {
	givens, _ := classicSolved()
	g := &game{
		save:       newSaveData(),
		difficulty: diffEasy,
		menuIndex:  menuNewGame,
	}
	g.save.statsFor(diffEasy).FastestReplay = &fastestReplay{
		ID:     testPuzzleID,
		Givens: givens,
		Events: "2P5",
	}
	return g
}

func TestVisibleMenuReplayOnlyWithRecording(t *testing.T) {
	g := &game{save: newSaveData(), difficulty: diffEasy}
	if g.canMenuReplay() {
		t.Fatal("empty stats should not offer replay")
	}
	if !reflect.DeepEqual(g.visibleMenu(), []int{menuNewGame, menuQuit}) {
		t.Fatalf("visible=%v", g.visibleMenu())
	}
	g.save.statsFor(diffEasy).FastestReplay = &fastestReplay{ID: "x", Givens: "short"}
	if g.canMenuReplay() {
		t.Fatal("short givens should not offer replay")
	}
	g = menuGameWithReplay()
	if !g.canMenuReplay() {
		t.Fatal("81-char fastestReplay should offer replay")
	}
	if !reflect.DeepEqual(g.menuItems(), []int{menuNewGame, menuQuit}) {
		t.Fatalf("items=%v", g.menuItems())
	}
	if !reflect.DeepEqual(g.visibleMenu(), []int{menuNewGame, menuQuit, menuReplay}) {
		t.Fatalf("visible=%v", g.visibleMenu())
	}
}

func TestMenuMoveDownFromQuitToReplay(t *testing.T) {
	g := menuGameWithReplay()
	g.menuIndex = menuQuit
	g.menuMove(1)
	if g.menuIndex != menuReplay {
		t.Fatalf("index=%d want replay", g.menuIndex)
	}
	if g.replayGivens != g.save.statsFor(diffEasy).FastestReplay.Givens {
		t.Fatal("landing on replay should load fastestReplay")
	}
	g.menuMove(-1)
	if g.menuIndex != menuQuit {
		t.Fatalf("up from replay=%d want quit", g.menuIndex)
	}
	g.menuIndex = menuReplay
	g.menuMove(1)
	if g.menuIndex != menuNewGame {
		t.Fatalf("down from replay=%d want wrap to new game", g.menuIndex)
	}
}

func TestClampMenuDropsReplayWithoutRecording(t *testing.T) {
	g := menuGameWithReplay()
	g.menuIndex = menuReplay
	g.difficulty = diffMedium
	g.clampMenu()
	if g.menuIndex != menuNewGame {
		t.Fatalf("index=%d want new game", g.menuIndex)
	}
}

func TestBeginMenuReplayLoadsFastest(t *testing.T) {
	g := menuGameWithReplay()
	g.beginMenuReplay()
	fr := g.save.statsFor(diffEasy).FastestReplay
	if g.replayGivens != fr.Givens {
		t.Fatalf("givens=%q", g.replayGivens)
	}
	if encodeReplay(g.replayEvents) != fr.Events {
		t.Fatalf("events=%q", encodeReplay(g.replayEvents))
	}
	if len(g.replaySolution) != 81 {
		t.Fatalf("solution len=%d", len(g.replaySolution))
	}
	if g.replayStarted.IsZero() {
		t.Fatal("replayStarted")
	}
}

func TestActivateMenuReplayDoesNotQuit(t *testing.T) {
	g := menuGameWithReplay()
	g.menuIndex = menuReplay
	if g.activateMenu() {
		t.Fatal("enter on replay should not quit")
	}
}
