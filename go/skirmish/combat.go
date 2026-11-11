package main

import (
	"fmt"
	"math/rand"
	"time"
)

type RNG interface {
	Intn(n int) int
}

type defaultRNG struct {
	r *rand.Rand
}

func newDefaultRNG() RNG {
	return &defaultRNG{r: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

func (d *defaultRNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	return d.r.Intn(n)
}

// RollDie rolls a 1-based die with N sides.
func RollDie(sides int, rng RNG) int {
	if sides <= 1 {
		return 1
	}
	return rng.Intn(sides) + 1
}

// MeleeResult details the outcome of close combat.
type MeleeResult struct {
	AttackerID       int
	DefenderID       int
	AttackerRoll1    int
	AttackerRoll2    int
	AttackerBest     int
	DefenderRoll     int
	AttackerWon      bool
	DefenderWon      bool
	IsTieBounceBack  bool
	InstantKillOnOne bool
	LogMessage       string
}

// ResolveMeleeCombat executes melee combat per the official rules.
func ResolveMeleeCombat(attacker, defender *Unit, rng RNG) MeleeResult {
	res := MeleeResult{
		AttackerID: attacker.ID,
		DefenderID: defender.ID,
	}

	attacker.MeleeFights++
	defender.MeleeFights++

	// Automatic elimination if defender is on "1" (wounded)
	if defender.IsWounded {
		res.InstantKillOnOne = true
		res.AttackerWon = true
		defender.IsCasualty = true
		attacker.Kills++
		attacker.LastEvent = fmt.Sprintf("Slain wounded %s in melee!", defender.Info().Rank)
		defender.LastEvent = fmt.Sprintf("Slain while wounded by %s", attacker.Info().Rank)
		res.LogMessage = fmt.Sprintf("MELEE: %s charges wounded %s ('1' state) - INSTANTLY SLAIN!",
			attacker.DisplayName(), defender.DisplayName())
		return res
	}

	// Attacker advantage: rolls two dice of their type, picks highest
	aSides := attacker.Info().Sides
	dSides := defender.Info().Sides

	res.AttackerRoll1 = RollDie(aSides, rng)
	res.AttackerRoll2 = RollDie(aSides, rng)
	if res.AttackerRoll1 >= res.AttackerRoll2 {
		res.AttackerBest = res.AttackerRoll1
	} else {
		res.AttackerBest = res.AttackerRoll2
	}

	// Defender rolls one die of their type
	res.DefenderRoll = RollDie(dSides, rng)

	if res.AttackerBest > res.DefenderRoll {
		// Attacker wins! Defender removed
		res.AttackerWon = true
		defender.IsCasualty = true
		attacker.Kills++
		attacker.LastEvent = fmt.Sprintf("Slain %s in melee [%d vs %d]", defender.Info().Rank, res.AttackerBest, res.DefenderRoll)
		defender.LastEvent = fmt.Sprintf("Slain by %s in melee [%d vs %d]", attacker.Info().Rank, res.AttackerBest, res.DefenderRoll)
		res.LogMessage = fmt.Sprintf("MELEE: %s charges %s! Attacker [%d, %d -> %d] vs Defender [%d]. %s SLAYS %s!",
			attacker.Info().Rank, defender.Info().Rank,
			res.AttackerRoll1, res.AttackerRoll2, res.AttackerBest, res.DefenderRoll,
			attacker.Info().Rank, defender.Info().Rank)
	} else if res.DefenderRoll > res.AttackerBest {
		// Defender wins! Attacker removed
		res.DefenderWon = true
		attacker.IsCasualty = true
		defender.Kills++
		defender.LastEvent = fmt.Sprintf("Counter-slain %s [%d vs %d]", attacker.Info().Rank, res.DefenderRoll, res.AttackerBest)
		attacker.LastEvent = fmt.Sprintf("Slain charging %s [%d vs %d]", defender.Info().Rank, res.AttackerBest, res.DefenderRoll)
		res.LogMessage = fmt.Sprintf("MELEE: %s charges %s! Attacker [%d, %d -> %d] vs Defender [%d]. Defender strikes back and SLAYS %s!",
			attacker.Info().Rank, defender.Info().Rank,
			res.AttackerRoll1, res.AttackerRoll2, res.AttackerBest, res.DefenderRoll,
			attacker.Info().Rank)
	} else {
		// Tie: Attacker bounces back 2 inches. No casualties.
		res.IsTieBounceBack = true
		attacker.LastEvent = fmt.Sprintf("Tied %s [%d=%d], bounced 2\"", defender.Info().Rank, res.AttackerBest, res.DefenderRoll)
		defender.LastEvent = fmt.Sprintf("Held vs %s [%d=%d]", attacker.Info().Rank, res.AttackerBest, res.DefenderRoll)
		res.LogMessage = fmt.Sprintf("MELEE: %s charges %s! Attacker [%d, %d -> %d] ties Defender [%d]! Attacker bounces back 2 inches!",
			attacker.Info().Rank, defender.Info().Rank,
			res.AttackerRoll1, res.AttackerRoll2, res.AttackerBest, res.DefenderRoll)
	}

	return res
}

// ShootingResult details the outcome of ranged combat.
type ShootingResult struct {
	AttackerID   int
	DefenderID   int
	AttackerRoll int
	DefenderBase int
	CoverBonus   int
	DefenderTotal int
	Hit          bool
	LogMessage   string
}

// ResolveShootingCombat executes a ranged attack.
// Costs 1 AP (author errata). Attacker die vs Defender die + Cover (+1 soft, +3 hard).
func ResolveShootingCombat(attacker, defender *Unit, coverBonus int, rng RNG) ShootingResult {
	res := ShootingResult{
		AttackerID: attacker.ID,
		DefenderID: defender.ID,
		CoverBonus: coverBonus,
	}

	attacker.HasShot = true
	attacker.ShotsFired++

	aSides := attacker.Info().Sides
	dSides := defender.Info().Sides

	res.AttackerRoll = RollDie(aSides, rng)
	res.DefenderBase = RollDie(dSides, rng)
	res.DefenderTotal = res.DefenderBase + coverBonus

	if res.AttackerRoll > res.DefenderTotal {
		// Hit! Defender turned to "1" face (wounded)
		res.Hit = true
		defender.IsWounded = true
		attacker.WoundsDealt++
		attacker.LastEvent = fmt.Sprintf("Wounded %s [%d vs %d]", defender.Info().Rank, res.AttackerRoll, res.DefenderTotal)
		defender.LastEvent = fmt.Sprintf("Wounded by %s's shot [%d vs %d]", attacker.Info().Rank, res.AttackerRoll, res.DefenderTotal)
		coverDesc := ""
		if coverBonus > 0 {
			coverDesc = fmt.Sprintf(" (+%d cover)", coverBonus)
		}
		res.LogMessage = fmt.Sprintf("SHOOTING: %s shoots %s! Attack [%d] vs Defense [%d%s = %d]. TARGET WOUNDED (set to '1' face)!",
			attacker.Info().Rank, defender.Info().Rank,
			res.AttackerRoll, res.DefenderBase, coverDesc, res.DefenderTotal)
	} else {
		// Miss / Evaded
		res.Hit = false
		attacker.LastEvent = fmt.Sprintf("Missed %s [%d vs %d]", defender.Info().Rank, res.AttackerRoll, res.DefenderTotal)
		defender.LastEvent = fmt.Sprintf("Evaded %s's shot [%d vs %d]", attacker.Info().Rank, res.AttackerRoll, res.DefenderTotal)
		coverDesc := ""
		if coverBonus > 0 {
			coverDesc = fmt.Sprintf(" (+%d cover)", coverBonus)
		}
		res.LogMessage = fmt.Sprintf("SHOOTING: %s shoots %s! Attack [%d] vs Defense [%d%s = %d]. Shot evaded / deflected!",
			attacker.Info().Rank, defender.Info().Rank,
			res.AttackerRoll, res.DefenderBase, coverDesc, res.DefenderTotal)
	}

	return res
}

// MoraleResult details a player's round-end morale test.
type MoraleResult struct {
	PlayerID   int
	Casualties int
	Roll       int
	Passed     bool
	LogMessage string
}

// RollMoraleTest rolls a d20 against current casualties. Must roll higher than casualties.
func RollMoraleTest(playerID int, casualties int, rng RNG) MoraleResult {
	roll := RollDie(20, rng)
	passed := roll > casualties
	msg := ""
	if passed {
		msg = fmt.Sprintf("MORALE: Player %d rolled %d against %d casualties: PASSED! The warband holds firm.",
			playerID+1, roll, casualties)
	} else {
		msg = fmt.Sprintf("MORALE: Player %d rolled %d against %d casualties: FAILED! The warband breaks and flees!",
			playerID+1, roll, casualties)
	}
	return MoraleResult{
		PlayerID:   playerID,
		Casualties: casualties,
		Roll:       roll,
		Passed:     passed,
		LogMessage: msg,
	}
}

// RecoveryResult details survival check for a wounded unit.
type RecoveryResult struct {
	UnitID     int
	Roll       int
	Recovered  bool
	LogMessage string
}

// RollCasualtyRecovery rolls a unit's die for a wounded unit showing "1".
// Even = recovers; Odd = dies and removed as casualty.
func RollCasualtyRecovery(u *Unit, rng RNG) RecoveryResult {
	roll := RollDie(u.Info().Sides, rng)
	even := (roll % 2) == 0
	msg := ""
	if even {
		u.IsWounded = false
		u.Recoveries++
		u.LastEvent = fmt.Sprintf("Recovered from wound (rolled %d)", roll)
		msg = fmt.Sprintf("RECOVERY: Wounded %s rolled %d (EVEN) - RECOVERED for next round!", u.DisplayName(), roll)
	} else {
		u.IsCasualty = true
		u.LastEvent = fmt.Sprintf("Succumbed to wound (rolled %d)", roll)
		msg = fmt.Sprintf("RECOVERY: Wounded %s rolled %d (ODD) - SUCCUMBED to wounds and eliminated!", u.DisplayName(), roll)
	}
	return RecoveryResult{
		UnitID:     u.ID,
		Roll:       roll,
		Recovered:  even,
		LogMessage: msg,
	}
}
