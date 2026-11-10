package main

import (
	"errors"
	"math/rand/v2"
)

const (
	gridCols  = 3
	gridRows  = 3
	gridSlots = gridCols * gridRows
)

var (
	errGameOver    = errors.New("the game is over")
	errBadColumn   = errors.New("that column is full")
	errNoDie       = errors.New("no die to place")
	errNotPlaying  = errors.New("no round in progress")
	errNotYourTurn = errors.New("it is not your turn")
)

// Grid is one player's 3×3 board, indexed [column][slot]. Slot 0 is the
// slot nearest the middle of the table; 0 means empty. Dice in a column are
// kept packed toward slot 0 so a column always fills from the middle outward.
type Grid [gridCols][gridRows]int

func (g *Grid) Count(c int) int {
	n := 0
	for _, v := range g[c] {
		if v != 0 {
			n++
		}
	}
	return n
}

func (g *Grid) Open(c int) bool { return g.Count(c) < gridRows }

func (g *Grid) Filled() int {
	n := 0
	for c := range g {
		n += g.Count(c)
	}
	return n
}

func (g *Grid) Full() bool { return g.Filled() == gridSlots }

// Matches is how many dice showing v sit in column c.
func (g *Grid) Matches(c, v int) int {
	n := 0
	for _, d := range g[c] {
		if d == v {
			n++
		}
	}
	return n
}

func (g *Grid) add(c, v int) int {
	for i := range g[c] {
		if g[c][i] == 0 {
			g[c][i] = v
			return i
		}
	}
	return -1
}

// destroy removes every die showing v from column c and slides the rest
// toward the middle. It returns how many dice were removed.
func (g *Grid) destroy(c, v int) int {
	var kept [gridRows]int
	k, n := 0, 0
	for _, d := range g[c] {
		switch {
		case d == 0:
		case d == v:
			n++
		default:
			kept[k] = d
			k++
		}
	}
	g[c] = kept
	return n
}

// columnScore adds n²·v for each value v that appears n times.
func columnScore(col [gridRows]int) int {
	var counts [7]int
	for _, v := range col {
		if v > 0 {
			counts[v]++
		}
	}
	s := 0
	for v := 1; v <= 6; v++ {
		s += counts[v] * counts[v] * v
	}
	return s
}

func (g *Grid) ColumnScore(c int) int { return columnScore(g[c]) }

func (g *Grid) Score() int {
	s := 0
	for c := range g {
		s += g.ColumnScore(c)
	}
	return s
}

// Rules holds the optional house rules from the dice.now guide.
type Rules struct {
	NoDestroy bool `json:"noDestroy,omitempty"` // placing never removes opponent dice
	TwoDice   bool `json:"twoDice,omitempty"`   // roll two dice, place one
	FillBoth  bool `json:"fillBoth,omitempty"`  // play until both grids are full
}

type Move struct {
	Col int `json:"col"`
	Die int `json:"die"`
}

type Placement struct {
	Player    int `json:"player"`
	Col       int `json:"col"`
	Slot      int `json:"slot"`
	Die       int `json:"die"`
	Destroyed int `json:"destroyed"`
	Gain      int `json:"gain"`
	Lost      int `json:"lost"`
}

type Match struct {
	Rules Rules   `json:"rules"`
	Grids [2]Grid `json:"grids"`
	Turn  int     `json:"turn"`
	Dice  [2]int  `json:"dice"`
	Over  bool    `json:"over"`
	Moves int     `json:"moves"`
}

func newMatch(r Rules, first int) Match {
	return Match{Rules: r, Turn: first}
}

func (m *Match) Roll(rng *rand.Rand) {
	m.Dice[0] = rng.IntN(6) + 1
	m.Dice[1] = 0
	if m.Rules.TwoDice {
		m.Dice[1] = rng.IntN(6) + 1
	}
}

func (m *Match) Legal(mv Move) bool {
	if m.Over || mv.Col < 0 || mv.Col >= gridCols || mv.Die < 0 || mv.Die > 1 {
		return false
	}
	return m.Dice[mv.Die] != 0 && m.Grids[m.Turn].Open(mv.Col)
}

func (m *Match) LegalMoves() []Move {
	var out []Move
	for d := 0; d < 2; d++ {
		if d == 1 && m.Dice[1] == m.Dice[0] {
			continue
		}
		for c := 0; c < gridCols; c++ {
			if mv := (Move{Col: c, Die: d}); m.Legal(mv) {
				out = append(out, mv)
			}
		}
	}
	return out
}

// apply places value v for the player to move without checking dice, then
// destroys, checks for the end, and passes the turn.
func (m *Match) apply(col, v int) Placement {
	p, o := m.Turn, 1-m.Turn
	before, oBefore := m.Grids[p].Score(), m.Grids[o].Score()
	pl := Placement{Player: p, Col: col, Die: v}
	pl.Slot = m.Grids[p].add(col, v)
	if !m.Rules.NoDestroy {
		pl.Destroyed = m.Grids[o].destroy(col, v)
	}
	pl.Gain = m.Grids[p].Score() - before
	pl.Lost = oBefore - m.Grids[o].Score()
	m.Dice = [2]int{}
	m.Moves++

	if m.Rules.FillBoth {
		m.Over = m.Grids[0].Full() && m.Grids[1].Full()
	} else {
		m.Over = m.Grids[0].Full() || m.Grids[1].Full()
	}
	if !m.Over {
		m.Turn = o
		if m.Rules.FillBoth && m.Grids[o].Full() {
			m.Turn = p
		}
	}
	return pl
}

func (m *Match) Place(mv Move) (Placement, error) {
	if m.Over {
		return Placement{}, errGameOver
	}
	if mv.Die < 0 || mv.Die > 1 || m.Dice[mv.Die] == 0 {
		return Placement{}, errNoDie
	}
	if !m.Legal(mv) {
		return Placement{}, errBadColumn
	}
	return m.apply(mv.Col, m.Dice[mv.Die]), nil
}

// Preview reports what placing the move would do without changing m.
func (m *Match) Preview(mv Move) (Placement, Match) {
	next := *m
	if !next.Legal(mv) {
		return Placement{}, next
	}
	return next.apply(mv.Col, next.Dice[mv.Die]), next
}

func (m *Match) Scores() [2]int {
	return [2]int{m.Grids[0].Score(), m.Grids[1].Score()}
}

// Winner is 0 or 1, or -1 for a tie.
func (m *Match) Winner() int {
	s := m.Scores()
	switch {
	case s[0] > s[1]:
		return 0
	case s[1] > s[0]:
		return 1
	}
	return -1
}
