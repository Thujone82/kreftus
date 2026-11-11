package main

import (
	"testing"
)

func TestLineOfSightAndCover(t *testing.T) {
	board := NewBoard()

	// 1. Ray between (10, 10) and (26, 10) passes through Center Ruins / Hedge
	hasLOS, cover := board.CheckLineOfSight(5, 10, 20, 10)
	if !hasLOS {
		t.Fatalf("expected LOS to be present")
	}
	if cover < 1 {
		t.Fatalf("expected at least soft cover (+1) passing near/through hedge")
	}

	// 2. Direct ray through center pillar (13, 18, 2x2 solid block)
	// Ray from (10, 18) to (16, 18)
	hasLOSBlocked, _ := board.CheckLineOfSight(10, 18, 16, 18)
	if hasLOSBlocked {
		t.Fatalf("expected pillar to block line of sight completely")
	}

	// 3. Clear line along side of board
	hasLOSClear, coverClear := board.CheckLineOfSight(2, 2, 2, 34)
	if !hasLOSClear || coverClear != 0 {
		t.Fatalf("expected clear LOS with 0 cover on empty board edge, got los=%v, cover=%d", hasLOSClear, coverClear)
	}
}

func TestObstacleCrossing(t *testing.T) {
	board := NewBoard()

	// Path crossing center ruin wall (18, 14, 8x1.2 hard cover)
	// From (18, 10) to (18, 16)
	crosses := board.PathCrossesObstacle(18, 10, 18, 16)
	if !crosses {
		t.Fatalf("expected path to cross center ruins wall obstacle")
	}

	// Path in clear open ground
	crossesClear := board.PathCrossesObstacle(2, 10, 2, 20)
	if crossesClear {
		t.Fatalf("expected clear path to not cross obstacle")
	}
}
