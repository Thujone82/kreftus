package main

import (
	"math/rand/v2"
	"testing"
)

// You hold 4 in column 2; the opponent holds 4, 4 across. A 4 there
// scores 12 and destroys 16: the 28-point swing.
func TestAIScoreAndDestroy(t *testing.T) {
	for _, d := range []string{diffMedium, diffHard} {
		m := newMatch(Rules{}, 0)
		m.Grids[0][1] = [gridRows]int{4}
		m.Grids[1][1] = [gridRows]int{4, 4}
		m.Grids[1][0] = [gridRows]int{2}
		m.Dice = [2]int{4, 0}
		if mv := chooseMove(&m, d, testRNG()); mv.Col != 1 {
			t.Errorf("%s chose column %d, want 1", d, mv.Col)
		}
	}
}

// Destroying the opponent's 6, 6 beats stacking a pair of 6s.
func TestAIDestroyBeatsStack(t *testing.T) {
	for _, d := range []string{diffMedium, diffHard} {
		m := newMatch(Rules{}, 0)
		m.Grids[0][1] = [gridRows]int{6}
		m.Grids[1][2] = [gridRows]int{6, 6}
		m.Dice = [2]int{6, 0}
		if mv := chooseMove(&m, d, testRNG()); mv.Col != 2 {
			t.Errorf("%s chose column %d, want 2", d, mv.Col)
		}
	}
}

// Ahead with one slot left: Hard closes the board instead of waiting.
func TestHardClosesBoardWhenAhead(t *testing.T) {
	m := newMatch(Rules{}, 0)
	m.Grids[0] = Grid{{6, 6, 6}, {5, 5, 5}, {1, 2, 0}}
	m.Grids[1] = Grid{{2, 3}, {4}, {1, 2}}
	m.Dice = [2]int{3, 0}
	mv := chooseMove(&m, diffHard, testRNG())
	_, next := m.Preview(mv)
	if !next.Over {
		t.Fatalf("hard chose %+v and left the game open", mv)
	}
}

// Behind, Hard keeps its last open column for a die that matters and
// avoids handing the opponent an easy destruction.
func TestHardAvoidsLosingClose(t *testing.T) {
	m := newMatch(Rules{}, 0)
	m.Grids[0] = Grid{{1, 2, 0}, {1, 2, 3}, {2, 3, 4}}
	m.Grids[1] = Grid{{6, 6, 6}, {5, 5}, {}}
	m.Dice = [2]int{1, 0}
	mv := chooseMove(&m, diffHard, testRNG())
	if mv.Col != 0 {
		t.Fatalf("only column 0 is open, got %+v", mv)
	}
	_, next := m.Preview(mv)
	if !next.Over || evaluate(&next, 0) > 0 {
		t.Fatal("expected the forced close to lose")
	}
}

func TestAIAlwaysLegal(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 9))
	for _, rules := range []Rules{{}, {TwoDice: true}, {FillBoth: true}, {NoDestroy: true}} {
		for _, d := range difficultyOrder {
			for game := 0; game < 20; game++ {
				m := newMatch(rules, rng.IntN(2))
				for steps := 0; !m.Over; steps++ {
					if steps > 500 {
						t.Fatalf("%s %+v: game never ended", d, rules)
					}
					m.Roll(rng)
					mv := chooseMove(&m, d, rng)
					if !m.Legal(mv) {
						t.Fatalf("%s %+v: illegal move %+v", d, rules, mv)
					}
					m.Place(mv)
				}
			}
		}
	}
}

func playAI(a, b string, games int, rng *rand.Rand) (winsA, winsB int) {
	diff := [2]string{a, b}
	for i := 0; i < games; i++ {
		m := newMatch(Rules{}, i%2)
		for !m.Over {
			m.Roll(rng)
			m.Place(chooseMove(&m, diff[m.Turn], rng))
		}
		switch m.Winner() {
		case 0:
			winsA++
		case 1:
			winsB++
		}
	}
	return
}

func TestDifficultyOrdering(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	hard, easy := playAI(diffHard, diffEasy, 400, rng)
	if float64(hard) < 0.65*float64(hard+easy) {
		t.Errorf("hard beat easy %d to %d, want at least 65%%", hard, easy)
	}
	med, easy2 := playAI(diffMedium, diffEasy, 400, rng)
	if med <= easy2 {
		t.Errorf("medium beat easy %d to %d", med, easy2)
	}
	hard2, med2 := playAI(diffHard, diffMedium, 400, rng)
	if hard2 <= med2 {
		t.Errorf("hard beat medium %d to %d", hard2, med2)
	}
	t.Logf("hard-easy %d-%d, medium-easy %d-%d, hard-medium %d-%d", hard, easy, med, easy2, hard2, med2)
}
