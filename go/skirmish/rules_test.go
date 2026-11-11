package main

import (
	"testing"
)

// mockRNG produces predetermined integers for deterministic unit tests.
type mockRNG struct {
	values []int
	idx    int
}

func (m *mockRNG) Intn(n int) int {
	if m.idx >= len(m.values) {
		return 0
	}
	val := m.values[m.idx]
	m.idx++
	return val % n
}

func TestMovementCosts(t *testing.T) {
	// 1st move (0-6")
	cost, steps, err := CalculateMoveCost(5.0, 0, false)
	if err != nil || cost != 1 || steps != 1 {
		t.Fatalf("expected 1 pt, 1 step; got cost=%d, steps=%d, err=%v", cost, steps, err)
	}

	// 2nd move (6-12" total)
	cost, steps, err = CalculateMoveCost(6.0, 1, false)
	if err != nil || cost != 3 || steps != 1 {
		t.Fatalf("expected 3 pts (for step 2), 1 step; got cost=%d, steps=%d, err=%v", cost, steps, err)
	}

	// 3rd move (12-18" total)
	cost, steps, err = CalculateMoveCost(5.5, 2, false)
	if err != nil || cost != 5 || steps != 1 {
		t.Fatalf("expected 5 pts (for step 3), 1 step; got cost=%d, steps=%d, err=%v", cost, steps, err)
	}

	// Moving 12" in one activation
	cost, steps, err = CalculateMoveCost(12.0, 0, false)
	if err != nil || cost != 4 || steps != 2 {
		t.Fatalf("expected 4 pts (1+3), 2 steps; got cost=%d, steps=%d, err=%v", cost, steps, err)
	}

	// Moving over an obstacle (+1 step penalty)
	cost, steps, err = CalculateMoveCost(5.0, 0, true)
	if err != nil || cost != 4 || steps != 2 {
		t.Fatalf("expected 4 pts (1+3 for 2 steps), 2 steps; got cost=%d, steps=%d, err=%v", cost, steps, err)
	}

	// Moving over max 3 steps should fail
	_, _, err = CalculateMoveCost(10.0, 2, false)
	if err == nil {
		t.Fatalf("expected error exceeding 3 steps, got nil")
	}
}

func TestMeleeCombatAttackerWins(t *testing.T) {
	attacker := &Unit{ID: 1, Type: DieD10} // Commander
	defender := &Unit{ID: 2, Type: DieD6}  // Soldier

	// mock rolls: attacker rolls 8 and 3 (best 9 and 4 from 0-based Intn -> +1 = 9, 4)
	// Intn(10) returns 8 (+1=9) and 3 (+1=4)
	// Defender Intn(6) returns 2 (+1=3)
	rng := &mockRNG{values: []int{8, 3, 2}}

	res := ResolveMeleeCombat(attacker, defender, rng)
	if !res.AttackerWon {
		t.Fatalf("expected attacker to win, got %+v", res)
	}
	if !defender.IsCasualty {
		t.Fatalf("expected defender to be removed as casualty")
	}
}

func TestMeleeCombatInstantKillOnOne(t *testing.T) {
	attacker := &Unit{ID: 1, Type: DieD4}
	defender := &Unit{ID: 2, Type: DieD10, IsWounded: true} // Commander on "1"

	rng := &mockRNG{values: []int{}}
	res := ResolveMeleeCombat(attacker, defender, rng)

	if !res.InstantKillOnOne || !res.AttackerWon {
		t.Fatalf("expected instant kill on wounded unit, got %+v", res)
	}
	if !defender.IsCasualty {
		t.Fatalf("expected wounded defender to be eliminated")
	}
}

func TestMeleeCombatTieBounceBack(t *testing.T) {
	attacker := &Unit{ID: 1, Type: DieD6}
	defender := &Unit{ID: 2, Type: DieD6}

	// Attacker Intn(6) returns 3, 1 -> rolls are 4, 2 (best 4)
	// Defender Intn(6) returns 3 -> roll is 4
	rng := &mockRNG{values: []int{3, 1, 3}}
	res := ResolveMeleeCombat(attacker, defender, rng)

	if !res.IsTieBounceBack {
		t.Fatalf("expected tie bounce-back, got %+v", res)
	}
	if attacker.IsCasualty || defender.IsCasualty {
		t.Fatalf("neither unit should die on tie bounce-back")
	}
}

func TestShootingCombatWithCover(t *testing.T) {
	attacker := &Unit{ID: 1, Type: DieD8, IsShooter: true}
	defender := &Unit{ID: 2, Type: DieD6}

	// Attacker roll Intn(8) -> 6 (+1 = 7)
	// Defender roll Intn(6) -> 2 (+1 = 3)
	// Hard Cover: +3 -> Total defense = 3 + 3 = 6
	// Attack (7) > Defense (6) -> HIT!
	rng := &mockRNG{values: []int{6, 2}}
	res := ResolveShootingCombat(attacker, defender, 3, rng)

	if !res.Hit {
		t.Fatalf("expected shooting hit, got %+v", res)
	}
	if !defender.IsWounded {
		t.Fatalf("expected defender to be wounded ('1' state)")
	}
	if !attacker.HasShot {
		t.Fatalf("expected attacker to be marked as hasShot")
	}
}

func TestMoraleTest(t *testing.T) {
	// 4 casualties: Roll Intn(20) returns 14 (+1 = 15). 15 > 4: Pass!
	rngPass := &mockRNG{values: []int{14}}
	resPass := RollMoraleTest(0, 4, rngPass)
	if !resPass.Passed {
		t.Fatalf("expected morale pass with roll 15 against 4 casualties")
	}

	// 5 casualties: Roll Intn(20) returns 3 (+1 = 4). 4 <= 5: Fail!
	rngFail := &mockRNG{values: []int{3}}
	resFail := RollMoraleTest(0, 5, rngFail)
	if resFail.Passed {
		t.Fatalf("expected morale fail with roll 4 against 5 casualties")
	}
}

func TestCasualtyRecovery(t *testing.T) {
	// Even roll recovers
	uEven := &Unit{ID: 1, Type: DieD6, IsWounded: true}
	rngEven := &mockRNG{values: []int{3}} // Intn(6)=3 -> roll=4 (Even)
	resEven := RollCasualtyRecovery(uEven, rngEven)
	if !resEven.Recovered || uEven.IsWounded || uEven.IsCasualty {
		t.Fatalf("expected recovery on even roll, got %+v", resEven)
	}

	// Odd roll succumbs
	uOdd := &Unit{ID: 2, Type: DieD6, IsWounded: true}
	rngOdd := &mockRNG{values: []int{2}} // Intn(6)=2 -> roll=3 (Odd)
	resOdd := RollCasualtyRecovery(uOdd, rngOdd)
	if resOdd.Recovered || !uOdd.IsCasualty {
		t.Fatalf("expected succumbed on odd roll, got %+v", resOdd)
	}
}
