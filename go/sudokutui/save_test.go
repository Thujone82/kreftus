package main

import (
	"bytes"
	"encoding/json"
	"math/rand/v2"
	"os"
	"testing"
	"time"
)

const testPuzzleID = "testhash12ab"

func classicSolved() (givens, sol string) {
	givens = "530070000600195000098000060800060003400803001700020006060000280000419005000080079"
	sol = solveSudoku(givens)
	return
}

func chdirTemp(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
}

func TestFinishSuccessWipesContinue(t *testing.T) {
	chdirTemp(t)
	givens, sol := classicSolved()
	g := &game{
		save:       newSaveData(),
		difficulty: diffEasy,
		puzzle:     puzzleEntry{ID: testPuzzleID, Givens: givens, Solution: sol},
		board:      newBoard(givens, sol, sol),
		view:       viewPlay,
		elapsed:    45 * time.Second,
	}
	g.save.setContinue(diffEasy, &continueGame{
		ID: testPuzzleID, Difficulty: diffEasy, Givens: givens, Solution: sol, Grid: sol,
	})
	g.save.setContinue(diffMedium, &continueGame{
		ID: "otherhash12ab", Difficulty: diffMedium, Givens: givens, Solution: sol, Grid: givens,
	})
	g.finishSuccess()
	if g.save.continueFor(diffEasy) != nil {
		t.Fatal("in-memory continue should be wiped")
	}
	if g.save.continueFor(diffMedium) == nil {
		t.Fatal("other difficulty continue should stay")
	}
	if g.puzzle.ID != "" {
		t.Fatal("puzzle identity should be cleared immediately")
	}
	if g.shouldPersistContinue() {
		t.Fatal("must not persist continue after success")
	}
	loaded := loadSave()
	if loaded.continueFor(diffEasy) != nil {
		t.Fatal("save file continue should be wiped")
	}
	if loaded.continueFor(diffMedium) == nil {
		t.Fatal("other difficulty continue should still be on disk")
	}
	if _, ok := loaded.completedSet(diffEasy)[testPuzzleID]; !ok {
		t.Fatal("completed id missing from save")
	}
	if loaded.statsFor(diffEasy).Successes != 1 {
		t.Fatalf("successes=%d", loaded.statsFor(diffEasy).Successes)
	}
	if loaded.statsFor(diffEasy).Perfect != 1 {
		t.Fatalf("perfect=%d", loaded.statsFor(diffEasy).Perfect)
	}
	if loaded.statsFor(diffEasy).errorRate() != "0.00" {
		t.Fatalf("error rate=%s", loaded.statsFor(diffEasy).errorRate())
	}
	if n, ok := loaded.completedMistakes(diffEasy, testPuzzleID); !ok || n != 0 {
		t.Fatalf("completed mistakes=%d ok=%v want 0", n, ok)
	}
	if loaded.averageCompletion(diffEasy) != "0:45" {
		t.Fatalf("average=%s want 0:45", loaded.averageCompletion(diffEasy))
	}
	rep := loaded.statsFor(diffEasy).FastestReplay
	if rep == nil || rep.ID != testPuzzleID || rep.Givens != givens {
		t.Fatalf("fastestReplay=%v", rep)
	}
	if !g.solvedNewRecord {
		t.Fatal("first success should be a new Best/Fastest")
	}
}

func TestFinishSuccessFlagsOnlyNewRecord(t *testing.T) {
	chdirTemp(t)
	givens, sol := classicSolved()
	g := &game{
		save:       newSaveData(),
		difficulty: diffEasy,
		puzzle:     puzzleEntry{ID: "secondhash12a", Givens: givens, Solution: sol},
		board:      newBoard(givens, sol, sol),
		view:       viewPlay,
		elapsed:    10 * time.Second,
	}
	g.board.mistakes = 1
	g.save.recordSuccess(diffEasy, 45000, 0)
	g.finishSuccess()
	if g.solvedNewRecord {
		t.Fatal("imperfect slower than a perfect must not be a new record")
	}
	if g.save.statsFor(diffEasy).bestLabel() != "Fastest" {
		t.Fatalf("label=%q", g.save.statsFor(diffEasy).bestLabel())
	}
}

func TestPersistPlaySkipsCompletedBoard(t *testing.T) {
	chdirTemp(t)
	givens, sol := classicSolved()
	g := &game{
		save:       newSaveData(),
		difficulty: diffEasy,
		puzzle:     puzzleEntry{ID: testPuzzleID, Givens: givens, Solution: sol},
		board:      newBoard(givens, sol, sol),
		view:       viewPlay,
	}
	g.persistPlay()
	if g.save.continueFor(diffEasy) != nil {
		t.Fatal("must not write continue for a completed board")
	}
	if _, err := os.Stat(saveFileName); !os.IsNotExist(err) {
		t.Fatal("must not write save file for a completed board")
	}
}

func TestScrubCompletedContinueOnLoad(t *testing.T) {
	chdirTemp(t)
	givens, sol := classicSolved()
	s := newSaveData()
	s.setContinue(diffEasy, &continueGame{
		ID:         testPuzzleID,
		Difficulty: diffEasy,
		Givens:     givens,
		Solution:   sol,
		Grid:       sol,
		ElapsedMs:  12000,
		Mistakes:   2,
	})
	if err := s.write(); err != nil {
		t.Fatal(err)
	}
	loaded := loadSave()
	if loaded.continueFor(diffEasy) != nil {
		t.Fatal("load should wipe a completed continue")
	}
	if loaded.statsFor(diffEasy).Successes != 1 {
		t.Fatalf("successes=%d", loaded.statsFor(diffEasy).Successes)
	}
	if _, ok := loaded.completedSet(diffEasy)[testPuzzleID]; !ok {
		t.Fatal("completed id missing after scrub")
	}
	if n, ok := loaded.completedMistakes(diffEasy, testPuzzleID); !ok || n != 2 {
		t.Fatalf("scrubbed continue mistakes=%d ok=%v want 2", n, ok)
	}
	again := loadSave()
	if again.statsFor(diffEasy).Successes != 1 {
		t.Fatalf("must not double-count success, got %d", again.statsFor(diffEasy).Successes)
	}
	if again.continueFor(diffEasy) != nil {
		t.Fatal("continue must stay wiped")
	}
}

func TestShouldPersistContinue(t *testing.T) {
	givens, sol := classicSolved()
	g := &game{
		puzzle: puzzleEntry{ID: testPuzzleID, Givens: givens, Solution: sol},
		board:  newBoard(givens, sol, givens),
		view:   viewPlay,
	}
	if !g.shouldPersistContinue() {
		t.Fatal("in-progress board should persist")
	}
	g.board = newBoard(givens, sol, sol)
	if g.shouldPersistContinue() {
		t.Fatal("completed board must not persist")
	}
	g.board = newBoard(givens, sol, givens)
	g.pendingSolved = true
	if g.shouldPersistContinue() {
		t.Fatal("pending solved must not persist")
	}
}

func TestRecordSuccessPerfectAndErrorRate(t *testing.T) {
	s := newSaveData()
	if s.statsFor(diffEasy).errorRate() != "—" {
		t.Fatal("no successes should show em dash")
	}
	s.recordSuccess(diffEasy, 1000, 0)
	s.recordSuccess(diffEasy, 2000, 2)
	s.recordSuccess(diffEasy, 1500, 1)
	st := s.statsFor(diffEasy)
	if st.Perfect != 1 || st.Successes != 3 {
		t.Fatalf("perfect=%d successes=%d", st.Perfect, st.Successes)
	}
	if st.errorRate() != "1.00" {
		t.Fatalf("error rate=%s want 1.00", st.errorRate())
	}
	s.recordSuccess(diffEasy, 3000, 1)
	if st.errorRate() != "1.00" {
		t.Fatalf("error rate=%s want 1.00 after fourth", st.errorRate())
	}
	s.recordSuccess(diffMedium, 4000, 3)
	s.recordSuccess(diffMedium, 5000, 2)
	if s.statsFor(diffMedium).errorRate() != "2.50" {
		t.Fatalf("medium error rate=%s want 2.50", s.statsFor(diffMedium).errorRate())
	}
	if s.statsFor(diffMedium).Perfect != 0 {
		t.Fatal("medium should have no perfects")
	}
	st.RatedSuccesses = 20
	st.MistakeSum = 21
	if st.errorRate() != "1.05" {
		t.Fatalf("error rate=%s want 1.05", st.errorRate())
	}
}

func TestMarkCompletedStoresMistakes(t *testing.T) {
	s := newSaveData()
	if s.averageCompletion(diffEasy) != "—" {
		t.Fatal("no completions should show em dash")
	}
	s.markCompleted(diffEasy, "abc", 4, 60000)
	if n, ok := s.completedMistakes(diffEasy, "abc"); !ok || n != 4 {
		t.Fatalf("mistakes=%d ok=%v want 4", n, ok)
	}
	s.markCompleted(diffEasy, "abc", 9, 90000)
	if n, _ := s.completedMistakes(diffEasy, "abc"); n != 4 {
		t.Fatalf("duplicate id should keep first mistakes, got %d", n)
	}
	s.markCompleted(diffEasy, "def", 1, 120000)
	if s.averageCompletion(diffEasy) != "1:30" {
		t.Fatalf("average=%s want 1:30", s.averageCompletion(diffEasy))
	}
	s.markCompleted(diffEasy, "ghi", 0, 30000)
	s.markCompleted(diffEasy, "jkl", 0, 90000)
	if s.averageCompletion(diffEasy) != "1:00" {
		t.Fatalf("perfect average=%s want 1:00 (only 0:30 and 1:30)", s.averageCompletion(diffEasy))
	}
}

func TestResumeContinueRestoresPencilMode(t *testing.T) {
	givens, sol := classicSolved()
	g := &game{save: newSaveData()}
	g.save.setContinue(diffEasy, &continueGame{
		ID: testPuzzleID, Difficulty: diffEasy, Givens: givens, Solution: sol, Grid: givens, Pencil: true,
	})
	g.difficulty = diffEasy
	g.resumeContinue()
	if !g.pencil {
		t.Fatal("continue with pencil true should open in pencil mode")
	}
	g.save.continueFor(diffEasy).Pencil = false
	g.resumeContinue()
	if g.pencil {
		t.Fatal("continue with pencil false should open in pen mode")
	}
}

func TestPersistPlayDebouncesDisk(t *testing.T) {
	chdirTemp(t)
	givens, sol := classicSolved()
	g := &game{
		save:       newSaveData(),
		difficulty: diffEasy,
		puzzle:     puzzleEntry{ID: testPuzzleID, Givens: givens, Solution: sol},
		board:      newBoard(givens, sol, givens),
		view:       viewPlay,
		saveFlush:  make(chan struct{}, 1),
	}
	g.persistPlay()
	if g.save.continueFor(diffEasy) == nil {
		t.Fatal("continue should be in memory immediately")
	}
	if _, err := os.Stat(saveFileName); !os.IsNotExist(err) {
		t.Fatal("debounced persist should not write immediately")
	}
	g.flushSave()
	if _, err := os.Stat(saveFileName); err != nil {
		t.Fatal("flush should write the save file")
	}
}

func TestWriteAtomicCompactJSON(t *testing.T) {
	chdirTemp(t)
	s := newSaveData()
	s.markCompleted(diffEasy, "abc", 1, 1000)
	if err := s.write(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(saveFileName)
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.TrimPrefix(data, utf8BOM)
	if bytes.Contains(data, []byte("\n  ")) {
		t.Fatal("save should be compact JSON, not indented")
	}
	loaded := loadSave()
	if n, ok := loaded.completedMistakes(diffEasy, "abc"); !ok || n != 1 {
		t.Fatalf("round trip mistakes=%d ok=%v", n, ok)
	}
	s.markCompleted(diffEasy, "def", 2, 2000)
	if err := s.write(); err != nil {
		t.Fatal(err)
	}
	loaded = loadSave()
	if _, ok := loaded.completedSet(diffEasy)["def"]; !ok {
		t.Fatal("second atomic write should replace the file")
	}
}

func TestLegacyContinueObjectLoadsPerDifficulty(t *testing.T) {
	chdirTemp(t)
	givens, sol := classicSolved()
	legacy := map[string]any{
		"version": "1.2",
		"continue": map[string]any{
			"id": testPuzzleID, "difficulty": diffMedium,
			"givens": givens, "solution": sol, "grid": givens,
			"elapsedMs": 1000, "mistakes": 0,
		},
	}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(saveFileName, data, 0644); err != nil {
		t.Fatal(err)
	}
	loaded := loadSave()
	if loaded.continueFor(diffEasy) != nil {
		t.Fatal("legacy continue should not land on easy")
	}
	c := loaded.continueFor(diffMedium)
	if c == nil || c.ID != testPuzzleID {
		t.Fatal("legacy continue should map to medium")
	}
}

func TestReturnToMenuKeepsContinue(t *testing.T) {
	chdirTemp(t)
	givens, sol := classicSolved()
	g := &game{
		save:       newSaveData(),
		difficulty: diffEasy,
		puzzle:     puzzleEntry{ID: testPuzzleID, Givens: givens, Solution: sol},
		board:      newBoard(givens, sol, givens),
		view:       viewPlay,
	}
	g.returnToMenu()
	if g.view != viewMenu {
		t.Fatalf("view=%d want menu", g.view)
	}
	if g.save.continueFor(diffEasy) == nil {
		t.Fatal("esc to menu should keep continue")
	}
	if g.save.statsFor(diffEasy).Failed != 0 {
		t.Fatal("esc to menu must not count as Failed")
	}
	if g.menuIndex != menuContinue {
		t.Fatal("menu should land on Continue")
	}
}

func TestNewGamePromptsOnlyForSameDifficulty(t *testing.T) {
	if err := loadPuzzleBank(); err != nil {
		t.Fatal(err)
	}
	givens, sol := classicSolved()
	g := &game{
		save:       newSaveData(),
		difficulty: diffEasy,
		menuIndex:  menuNewGame,
		rng:        rand.New(rand.NewPCG(1, 2)),
	}
	g.save.setContinue(diffMedium, &continueGame{
		ID: "medhash12ab", Difficulty: diffMedium, Givens: givens, Solution: sol, Grid: givens,
	})
	if g.activateMenu() {
		t.Fatal("new game should not quit")
	}
	if g.view != viewPlay {
		t.Fatalf("view=%d want play (no prompt for another difficulty's save)", g.view)
	}
	if g.save.continueFor(diffMedium) == nil {
		t.Fatal("medium continue should remain")
	}

	g2 := &game{
		save:       newSaveData(),
		difficulty: diffEasy,
		menuIndex:  menuNewGame,
		rng:        rand.New(rand.NewPCG(1, 2)),
	}
	g2.save.setContinue(diffEasy, &continueGame{
		ID: testPuzzleID, Difficulty: diffEasy, Givens: givens, Solution: sol, Grid: givens,
	})
	g2.activateMenu()
	if g2.view != viewConfirmNewGame {
		t.Fatalf("view=%d want confirm new game", g2.view)
	}
}

func TestRecordSuccessStoresBestReplay(t *testing.T) {
	s := newSaveData()
	st := s.statsFor(diffEasy)
	if !s.recordSuccess(diffEasy, 5000, 2) {
		t.Fatal("first success is best")
	}
	st.FastestReplay = &fastestReplay{ID: "first", Givens: "g", Events: "2P5"}
	if s.recordSuccess(diffEasy, 4000, 3) {
		t.Fatal("faster with more mistakes must not replace Best")
	}
	if st.FastestReplay == nil || st.FastestReplay.ID != "first" {
		t.Fatal("worse Best must keep the stored replay")
	}
	if *st.FastestMs != 5000 || *st.FastestMistakes != 2 {
		t.Fatalf("best=%d/%d", *st.FastestMs, *st.FastestMistakes)
	}
	if !s.recordSuccess(diffEasy, 8000, 1) {
		t.Fatal("fewer mistakes should replace Best even if slower")
	}
	if !s.recordSuccess(diffEasy, 7000, 1) {
		t.Fatal("same mistakes faster should replace Best")
	}
	if s.recordSuccess(diffEasy, 9000, 1) {
		t.Fatal("same mistakes slower must not replace")
	}
	if !s.recordSuccess(diffEasy, 20000, 0) {
		t.Fatal("first perfect should replace any imperfect Best")
	}
	if s.recordSuccess(diffEasy, 1000, 1) {
		t.Fatal("imperfect must not replace a perfect Fastest")
	}
	if s.recordSuccess(diffEasy, 21000, 0) {
		t.Fatal("slower perfect must not replace Fastest")
	}
	if !s.recordSuccess(diffEasy, 15000, 0) {
		t.Fatal("faster perfect should replace Fastest")
	}
	if *st.FastestMs != 15000 || *st.FastestMistakes != 0 {
		t.Fatalf("fastest perfect=%d/%d", *st.FastestMs, *st.FastestMistakes)
	}
}

func TestUnmarshalJSONKeepsFastestReplay(t *testing.T) {
	chdirTemp(t)
	givens, _ := classicSolved()
	s := newSaveData()
	ms := int64(1234)
	m := 2
	s.Stats[diffEasy] = &diffStats{
		Successes:       1,
		RatedSuccesses:  1,
		MistakeSum:      2,
		FastestMs:       &ms,
		FastestMistakes: &m,
		FastestReplay:   &fastestReplay{ID: testPuzzleID, Givens: givens, Events: "2P5,14M3"},
	}
	if err := s.write(); err != nil {
		t.Fatal(err)
	}
	loaded := loadSave()
	rep := loaded.statsFor(diffEasy).FastestReplay
	if rep == nil || rep.ID != testPuzzleID || rep.Givens != givens || rep.Events != "2P5,14M3" {
		t.Fatalf("fastestReplay dropped on load: %+v", rep)
	}
}

func TestContinuePersistsAndRestoresEvents(t *testing.T) {
	chdirTemp(t)
	givens, sol := classicSolved()
	events := []replayEvent{
		{Cell: 2, Op: replayOpPlace, Digit: sol[2]},
		{Cell: 3, Op: replayOpMark, Digit: '1'},
	}
	g := &game{
		save:         newSaveData(),
		difficulty:   diffEasy,
		puzzle:       puzzleEntry{ID: testPuzzleID, Givens: givens, Solution: sol},
		board:        newBoard(givens, sol, givens),
		view:         viewPlay,
		replayEvents: events,
	}
	g.persistPlayNow()
	c := g.save.continueFor(diffEasy)
	if c == nil || c.Events != encodeReplay(events) {
		t.Fatalf("continue events=%v", c)
	}
	loaded := loadSave()
	c2 := loaded.continueFor(diffEasy)
	if c2 == nil || c2.Events != encodeReplay(events) {
		t.Fatal("events missing after disk round trip")
	}
	g2 := &game{save: loaded, difficulty: diffEasy}
	g2.resumeContinue()
	if encodeReplay(g2.replayEvents) != encodeReplay(events) {
		t.Fatalf("restored events=%q", encodeReplay(g2.replayEvents))
	}
	if g2.replayGivens != givens || g2.replaySolution != sol {
		t.Fatal("replay givens/solution not restored")
	}
	if g2.replayStart.active(givens) {
		t.Fatal("native 1.5 continue with events should not onboard a start snapshot")
	}
}

func TestResumeContinueOnboardsPre15(t *testing.T) {
	chdirTemp(t)
	givens, sol := classicSolved()
	mid := []byte(givens)
	mid[2] = sol[2]
	grid := string(mid)
	g := &game{save: newSaveData(), difficulty: diffEasy}
	g.save.setContinue(diffEasy, &continueGame{
		ID: testPuzzleID, Difficulty: diffEasy, Givens: givens, Solution: sol, Grid: grid,
	})
	g.resumeContinue()
	if !g.replayStart.active(givens) || g.replayStart.Grid != grid {
		t.Fatalf("should snapshot the pre-1.5 board: %+v", g.replayStart)
	}
	if len(g.replayEvents) != 0 {
		t.Fatal("onboard must not invent pre-1.5 events")
	}
	c := g.save.continueFor(diffEasy)
	if c == nil || c.StartGrid != grid {
		t.Fatal("startGrid should be persisted on continue")
	}
	zero := replayBoardAt(g.replayGivens, g.replaySolution, g.replayStart, g.replayEvents, 0)
	if zero.gridString() != grid {
		t.Fatal("playback frame 0 should be the onboarded board")
	}

	g.board.cursor = 3
	g.applyDigit(sol[3])
	if encodeReplay(g.replayEvents) == "" {
		t.Fatal("new moves after onboard should log")
	}
	g.persistPlayNow()
	loaded := loadSave()
	g2 := &game{save: loaded, difficulty: diffEasy}
	g2.resumeContinue()
	if g2.replayStart.Grid != grid {
		t.Fatal("second continue should keep the snapshot origin")
	}
	if encodeReplay(g2.replayEvents) != encodeReplay(g.replayEvents) {
		t.Fatalf("post-onboard events=%q", encodeReplay(g2.replayEvents))
	}
	one := replayBoardAt(g2.replayGivens, g2.replaySolution, g2.replayStart, g2.replayEvents, 1)
	if one.grid[2] != sol[2] || one.grid[3] != sol[3] {
		t.Fatal("events should apply on top of the snapshot")
	}
}

func TestFinishSuccessStoresOnboardedReplay(t *testing.T) {
	chdirTemp(t)
	givens, sol := classicSolved()
	mid := []byte(givens)
	mid[2] = sol[2]
	g := &game{
		save:         newSaveData(),
		difficulty:   diffEasy,
		puzzle:       puzzleEntry{ID: testPuzzleID, Givens: givens, Solution: sol},
		board:        newBoard(givens, sol, sol),
		view:         viewPlay,
		elapsed:      45 * time.Second,
		replayEvents: []replayEvent{{Cell: 3, Op: replayOpPlace, Digit: sol[3]}},
		replayStart:  replayStart{Grid: string(mid)},
		replayGivens: givens,
	}
	g.finishSuccess()
	rep := g.save.statsFor(diffEasy).FastestReplay
	if rep == nil || rep.StartGrid != string(mid) || rep.Events == "" {
		t.Fatalf("fastestReplay should keep the onboard snapshot: %+v", rep)
	}
}

func TestScrubCompletedContinueKeepsReplay(t *testing.T) {
	chdirTemp(t)
	givens, sol := classicSolved()
	s := newSaveData()
	s.setContinue(diffEasy, &continueGame{
		ID:         testPuzzleID,
		Difficulty: diffEasy,
		Givens:     givens,
		Solution:   sol,
		Grid:       sol,
		ElapsedMs:  12000,
		Mistakes:   2,
		Events:     "2P5",
	})
	if err := s.write(); err != nil {
		t.Fatal(err)
	}
	loaded := loadSave()
	rep := loaded.statsFor(diffEasy).FastestReplay
	if rep == nil || rep.ID != testPuzzleID || rep.Events != "2P5" {
		t.Fatalf("scrub should store continue events as fastestReplay: %+v", rep)
	}
}

func TestLoadReconcilesBestFromCompleted(t *testing.T) {
	chdirTemp(t)
	givens, _ := classicSolved()
	s := newSaveData()
	fastWrong := int64(1000)
	wrong := 5
	s.Stats[diffEasy] = &diffStats{
		Successes:       2,
		Perfect:         1,
		RatedSuccesses:  2,
		MistakeSum:      5,
		FastestMs:       &fastWrong,
		FastestMistakes: &wrong,
		FastestReplay:   &fastestReplay{ID: "fast-wrong", Givens: givens, Events: "2P5"},
	}
	s.Completed[diffEasy] = []completedEntry{
		{ID: "fast-wrong", Mistakes: 5, ElapsedMs: 1000},
		{ID: "slow-perfect", Mistakes: 0, ElapsedMs: 90000},
	}
	if err := s.write(); err != nil {
		t.Fatal(err)
	}
	loaded := loadSave()
	st := loaded.statsFor(diffEasy)
	if st.FastestMs == nil || *st.FastestMs != 90000 || st.FastestMistakes == nil || *st.FastestMistakes != 0 {
		t.Fatalf("reconciled best=%v/%v want 90000/0", st.FastestMs, st.FastestMistakes)
	}
	if st.FastestReplay != nil {
		t.Fatalf("stale imperfect replay should drop: %+v", st.FastestReplay)
	}
	again := loadSave()
	if again.statsFor(diffEasy).FastestReplay != nil {
		t.Fatal("reconcile should stay written")
	}
}
