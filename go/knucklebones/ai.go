package main

import "math/rand/v2"

const (
	diffEasy   = "easy"
	diffMedium = "medium"
	diffHard   = "hard"
)

var difficultyOrder = []string{diffEasy, diffMedium, diffHard}

var difficultyLabel = map[string]string{
	diffEasy:   "Easy",
	diffMedium: "Medium",
	diffHard:   "Hard",
}

// Opponents from the Lonely Shack, easiest to hardest.
var aiName = map[string]string{
	diffEasy:   "Ratau",
	diffMedium: "Flinky",
	diffHard:   "Shrumy",
}

// A finished game outweighs any score margin.
const terminalBonus = 1000.0

func evaluate(m *Match, p int) float64 {
	s := m.Scores()
	d := float64(s[p] - s[1-p])
	if m.Over {
		switch {
		case d > 0:
			return terminalBonus + d
		case d < 0:
			return -terminalBonus + d
		}
	}
	return d
}

// chooseMove picks a move for the player to move.
//   - Easy places in a random open column.
//   - Medium takes the biggest immediate swing.
//   - Hard takes the swing minus the opponent's best reply averaged over
//     every roll, and treats ending the game as a win or loss.
func chooseMove(m *Match, difficulty string, rng *rand.Rand) Move {
	moves := m.LegalMoves()
	if len(moves) == 0 {
		return Move{Col: -1}
	}
	if difficulty == diffEasy {
		return moves[rng.IntN(len(moves))]
	}
	p := m.Turn
	var picks []Move
	best := 0.0
	for i, mv := range moves {
		_, next := m.Preview(mv)
		var v float64
		if difficulty == diffMedium {
			s := next.Scores()
			v = float64(s[p] - s[1-p])
		} else {
			v = expectedAfterReply(&next, p)
		}
		switch {
		case i == 0 || v > best+1e-9:
			best = v
			picks = []Move{mv}
		case v > best-1e-9:
			picks = append(picks, mv)
		}
	}
	return picks[rng.IntN(len(picks))]
}

// hardPlies is how many placements Hard looks past its own: the
// opponent's reply, then its own answer to that. Two-dice rolls have 36
// outcomes, so that variant stops after the reply to stay responsive.
const hardPlies = 2

func expectedAfterReply(m *Match, p int) float64 {
	plies := hardPlies
	if m.Rules.TwoDice {
		plies = 1
	}
	return expectimax(m, p, plies)
}

// expectimax averages over every roll for the player to move, who then
// picks their best placement (max for p, min for the opponent).
func expectimax(m *Match, p, plies int) float64 {
	if m.Over || plies == 0 {
		return evaluate(m, p)
	}
	bs := []int{0}
	if m.Rules.TwoDice {
		bs = []int{1, 2, 3, 4, 5, 6}
	}
	total, n := 0.0, 0
	for a := 1; a <= 6; a++ {
		for _, b := range bs {
			roll := *m
			roll.Dice = [2]int{a, b}
			best := 0.0
			for i, mv := range roll.LegalMoves() {
				_, after := roll.Preview(mv)
				v := expectimax(&after, p, plies-1)
				if i == 0 || (roll.Turn == p && v > best) || (roll.Turn != p && v < best) {
					best = v
				}
			}
			total += best
			n++
		}
	}
	return total / float64(n)
}
