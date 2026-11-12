package main

import (
	"context"
	"math/rand/v2"
)

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
	mv, _ := chooseMoveContext(context.Background(), m, difficulty, rng)
	return mv
}

func chooseMoveContext(ctx context.Context, m *Match, difficulty string, rng *rand.Rand) (Move, bool) {
	if ctx.Err() != nil {
		return Move{Col: -1}, false
	}
	moves := m.LegalMoves()
	if len(moves) == 0 {
		return Move{Col: -1}, true
	}
	if difficulty == diffEasy {
		return moves[rng.IntN(len(moves))], true
	}
	p := m.Turn
	var picks []Move
	best := 0.0
	for i, mv := range moves {
		if ctx.Err() != nil {
			return Move{Col: -1}, false
		}
		_, next := m.Preview(mv)
		var v float64
		if difficulty == diffMedium {
			s := next.Scores()
			v = float64(s[p] - s[1-p])
		} else {
			var ok bool
			v, ok = expectedAfterReply(ctx, &next, p)
			if !ok {
				return Move{Col: -1}, false
			}
		}
		switch {
		case i == 0 || v > best+1e-9:
			best = v
			picks = []Move{mv}
		case v > best-1e-9:
			picks = append(picks, mv)
		}
	}
	return picks[rng.IntN(len(picks))], true
}

// hardPlies is how many placements Hard looks past its own: the
// opponent's reply, then its own answer to that. Two-dice rolls have 36
// outcomes, so that variant stops after the reply to stay responsive.
const hardPlies = 2

func expectedAfterReply(ctx context.Context, m *Match, p int) (float64, bool) {
	plies := hardPlies
	if m.Rules.TwoDice {
		plies = 1
	}
	return expectimax(ctx, m, p, plies)
}

// expectimax averages over every roll for the player to move, who then
// picks their best placement (max for p, min for the opponent).
func expectimax(ctx context.Context, m *Match, p, plies int) (float64, bool) {
	if ctx.Err() != nil {
		return 0, false
	}
	if m.Over || plies == 0 {
		return evaluate(m, p), true
	}
	bs := []int{0}
	if m.Rules.TwoDice {
		bs = []int{1, 2, 3, 4, 5, 6}
	}
	total, n := 0.0, 0
	for a := 1; a <= 6; a++ {
		if ctx.Err() != nil {
			return 0, false
		}
		for _, b := range bs {
			roll := *m
			roll.Dice = [2]int{a, b}
			best := 0.0
			for i, mv := range roll.LegalMoves() {
				_, after := roll.Preview(mv)
				v, ok := expectimax(ctx, &after, p, plies-1)
				if !ok {
					return 0, false
				}
				if i == 0 || (roll.Turn == p && v > best) || (roll.Turn != p && v < best) {
					best = v
				}
			}
			total += best
			n++
		}
	}
	return total / float64(n), true
}

// estimateWinProb estimates the probability [0..1] that perspective p wins
// the current match using fast deterministic greedy-biased Monte Carlo rollouts.
func estimateWinProb(m *Match, p, samples int) float64 {
	if m == nil {
		return 0.5
	}
	if p < 0 || p > 1 {
		p = 0
	}
	if m.Over {
		w := m.Winner()
		switch w {
		case p:
			return 1
		case -1:
			return 0.5
		default:
			return 0
		}
	}
	if samples <= 0 {
		samples = 96
	}
	// Deterministic seed from board geometry so the meter doesn't flicker between redraws.
	var h1, h2 uint64 = 0x9e3779b97f4a7c15, 0xbf58476d1ce4e5b9
	mix := func(v int) {
		h1 = h1*1315423911 ^ uint64(v+17)
		h2 = h2*2654435761 ^ uint64(v*31+3)
	}
	mix(m.Turn)
	mix(m.Moves)
	mix(m.Dice[0])
	mix(m.Dice[1])
	for gp := 0; gp < 2; gp++ {
		for c := 0; c < gridCols; c++ {
			for k := 0; k < gridRows; k++ {
				mix(m.Grids[gp][c][k])
			}
		}
	}
	rng := rand.New(rand.NewPCG(h1, h2))

	wins := 0.0
	for i := 0; i < samples; i++ {
		sim := *m
		for !sim.Over {
			moves := sim.LegalMoves()
			if len(moves) == 0 {
				break
			}
			mv := moves[0]
			if len(moves) > 1 {
				if rng.IntN(100) < 15 {
					mv = moves[rng.IntN(len(moves))]
				} else {
					bestSwing := -1 << 30
					var tied []Move
					turn := sim.Turn
					for _, cand := range moves {
						_, nxt := sim.Preview(cand)
						sc := nxt.Scores()
						swing := sc[turn] - sc[1-turn]
						if nxt.Over {
							if w := nxt.Winner(); w == turn {
								swing += 500
							} else if w == 1-turn {
								swing -= 500
							}
						}
						if len(tied) == 0 || swing > bestSwing {
							bestSwing = swing
							tied = []Move{cand}
						} else if swing == bestSwing {
							tied = append(tied, cand)
						}
					}
					mv = tied[rng.IntN(len(tied))]
				}
			}
			_, nxt := sim.Preview(mv)
			sim = nxt
			if !sim.Over {
				sim.Roll(rng)
			}
		}
		switch sim.Winner() {
		case p:
			wins += 1.0
		case -1:
			wins += 0.5
		}
	}
	return wins / float64(samples)
}
