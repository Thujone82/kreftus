package main

import (
	"math/rand/v2"
	"testing"
)

func testRNG() *rand.Rand { return rand.New(rand.NewPCG(1, 2)) }

func TestColumnScoreTable(t *testing.T) {
	cases := []struct {
		col  [gridRows]int
		want int
	}{
		{[gridRows]int{6}, 6},
		{[gridRows]int{6, 2}, 8},
		{[gridRows]int{6, 6}, 24},
		{[gridRows]int{6, 6, 2}, 26},
		{[gridRows]int{4, 1, 4}, 17},
		{[gridRows]int{6, 6, 6}, 54},
		{[gridRows]int{3, 3, 3}, 27},
		{[gridRows]int{1, 1, 1}, 9},
		{[gridRows]int{}, 0},
	}
	for _, c := range cases {
		if got := columnScore(c.col); got != c.want {
			t.Errorf("columnScore(%v) = %d, want %d", c.col, got, c.want)
		}
	}
}

func TestMaxGridScore(t *testing.T) {
	var g Grid
	for c := 0; c < gridCols; c++ {
		for i := 0; i < gridRows; i++ {
			g.add(c, 6)
		}
	}
	if g.Score() != 162 || !g.Full() {
		t.Fatalf("score %d full %v, want 162 true", g.Score(), g.Full())
	}
}

func TestDestroyFacingColumnOnly(t *testing.T) {
	m := newMatch(Rules{}, 0)
	m.Grids[1][0] = [gridRows]int{5, 5, 0}
	m.Grids[1][2] = [gridRows]int{5, 0, 0}
	m.Dice = [2]int{5, 0}
	pl, err := m.Place(Move{Col: 0})
	if err != nil {
		t.Fatal(err)
	}
	if pl.Destroyed != 2 || pl.Lost != 20 || pl.Gain != 5 {
		t.Fatalf("placement %+v, want destroyed 2 lost 20 gain 5", pl)
	}
	if m.Grids[1].Count(0) != 0 || m.Grids[1].Count(2) != 1 {
		t.Fatalf("opponent grid %v", m.Grids[1])
	}
	if m.Turn != 1 {
		t.Fatalf("turn %d, want 1", m.Turn)
	}
}

func TestDestroySlidesRemainingDiceToMiddle(t *testing.T) {
	var g Grid
	g[1] = [gridRows]int{3, 4, 3}
	if n := g.destroy(1, 3); n != 2 {
		t.Fatalf("destroyed %d, want 2", n)
	}
	if g[1] != [gridRows]int{4, 0, 0} {
		t.Fatalf("column %v, want [4 0 0]", g[1])
	}
}

func TestFullColumnRejected(t *testing.T) {
	m := newMatch(Rules{}, 0)
	m.Grids[0][1] = [gridRows]int{1, 2, 3}
	m.Dice = [2]int{4, 0}
	if _, err := m.Place(Move{Col: 1}); err == nil {
		t.Fatal("placing in a full column should fail")
	}
	if _, err := m.Place(Move{Col: 0, Die: 1}); err == nil {
		t.Fatal("placing the missing second die should fail")
	}
}

func TestGameEndsWhenEitherGridFull(t *testing.T) {
	m := newMatch(Rules{}, 0)
	m.Grids[0] = Grid{{6, 6, 6}, {5, 5, 5}, {4, 4, 0}}
	m.Grids[1] = Grid{{1, 0, 0}}
	m.Dice = [2]int{2, 0}
	if _, err := m.Place(Move{Col: 2}); err != nil {
		t.Fatal(err)
	}
	if !m.Over || m.Winner() != 0 {
		t.Fatalf("over %v winner %d", m.Over, m.Winner())
	}
	if _, err := m.Place(Move{Col: 0}); err != errGameOver {
		t.Fatalf("err %v, want errGameOver", err)
	}
}

func TestTieWinner(t *testing.T) {
	m := newMatch(Rules{}, 0)
	m.Grids[0][0] = [gridRows]int{3}
	m.Grids[1][1] = [gridRows]int{3}
	if m.Winner() != -1 {
		t.Fatalf("winner %d, want tie", m.Winner())
	}
}

func TestNoDestroyVariant(t *testing.T) {
	m := newMatch(Rules{NoDestroy: true}, 0)
	m.Grids[1][0] = [gridRows]int{5, 5}
	m.Dice = [2]int{5, 0}
	pl, _ := m.Place(Move{Col: 0})
	if pl.Destroyed != 0 || m.Grids[1].Count(0) != 2 {
		t.Fatalf("no-destroy removed dice: %+v", pl)
	}
}

func TestTwoDiceVariant(t *testing.T) {
	m := newMatch(Rules{TwoDice: true}, 0)
	m.Roll(testRNG())
	if m.Dice[0] == 0 || m.Dice[1] == 0 {
		t.Fatalf("dice %v, want two", m.Dice)
	}
	m.Dice = [2]int{2, 6}
	if got := len(m.LegalMoves()); got != 6 {
		t.Fatalf("%d legal moves, want 6", got)
	}
	pl, err := m.Place(Move{Col: 1, Die: 1})
	if err != nil || pl.Die != 6 {
		t.Fatalf("placed %+v err %v", pl, err)
	}
	m.Dice = [2]int{3, 3}
	if got := len(m.LegalMoves()); got != 3 {
		t.Fatalf("%d legal moves for a double, want 3", got)
	}
}

func TestFillBothVariant(t *testing.T) {
	m := newMatch(Rules{FillBoth: true}, 0)
	m.Grids[0] = Grid{{6, 6, 6}, {5, 5, 5}, {4, 4, 0}}
	m.Dice = [2]int{1, 0}
	m.Place(Move{Col: 2})
	if m.Over {
		t.Fatal("fill-both ended with one grid full")
	}
	if m.Turn != 1 {
		t.Fatalf("turn %d, want 1", m.Turn)
	}
	m.Dice = [2]int{2, 0}
	m.Place(Move{Col: 0})
	if m.Turn != 1 {
		t.Fatalf("turn %d, want 1 to keep playing while the other grid is full", m.Turn)
	}
}

func TestSeriesBestOfThree(t *testing.T) {
	rng := testRNG()
	s := newSession(modeAI, [2]string{"A", "B"}, diffEasy, 3, Rules{}, rng)
	playRound := func() {
		for s.Phase == phasePlay {
			moves := s.Match.LegalMoves()
			if err := s.place(s.Match.Turn, moves[0], rng); err != nil {
				t.Fatal(err)
			}
		}
	}
	for i := 0; i < 20 && s.Phase != phaseGameOver; i++ {
		playRound()
		if s.Phase == phaseRoundOver {
			if !s.vote(0, voteContinue, rng) {
				t.Fatal("continue vote did not apply offline")
			}
		}
	}
	if s.Phase != phaseGameOver {
		t.Fatal("series never finished")
	}
	if w := s.seriesWinner(); w < 0 || s.Wins[w] != 2 {
		t.Fatalf("winner %d wins %v", w, s.Wins)
	}
	if !s.vote(0, voteFree, rng) || !s.freePlay() || s.Phase != phasePlay {
		t.Fatal("free play did not start")
	}
	if s.seriesDone() {
		t.Fatal("free play should never finish")
	}
}

func TestOnlineVotesNeedBoth(t *testing.T) {
	rng := testRNG()
	s := newSession(modeOnline, [2]string{"A", "B"}, "", 1, Rules{}, rng)
	s.forfeitBy(1, reasonResign)
	if s.vote(0, voteRematch, rng) {
		t.Fatal("one vote applied online")
	}
	if !s.vote(1, voteRematch, rng) || s.Phase != phasePlay || s.Series != 2 {
		t.Fatal("matching votes did not start a rematch")
	}
}

func TestTimeoutsForfeitOnThird(t *testing.T) {
	rng := testRNG()
	s := newSession(modeOnline, [2]string{"A", "B"}, "", 1, Rules{}, rng)
	p := s.Match.Turn
	for i := 0; i < 3 && s.Phase == phasePlay; i++ {
		s.Match.Turn = p
		s.timeout(rng)
	}
	if s.Forfeit == nil || s.Forfeit.Loser != p || s.Forfeit.Reason != reasonTimeout {
		t.Fatalf("forfeit %+v", s.Forfeit)
	}
}
