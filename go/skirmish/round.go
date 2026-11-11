package main

import (
	"fmt"
	"math"
)

// GameEngine orchestrates the full state and phase progression.
type GameEngine struct {
	Board           *Board
	Players         [2]*PlayerState
	Rosters         [2]*WarbandRoster
	Phase           GamePhase
	RoundNumber     int
	TurnNumber      int // 1, 2, 3, 4
	ActivePlayerIdx int // 0 or 1
	InitiativeWinner int // 0 or 1 for who won round initiative (Player A)

	// Turn State
	ActivationPoints int
	SelectedUnitID   int
	HasSelectedActed bool // true if active die has performed a move or shot

	// Deployment State
	DeployWinner     int // who won d20 edge roll
	DeployTurn       int // who is placing next unit
	DeployRemaining0 []*Unit
	DeployRemaining1 []*Unit

	// Combat / Log History
	Log          []CombatLogEntry
	RollQueue    []RollVisualEvent
	LastMelee    *MeleeResult
	LastShooting *ShootingResult
	LastMorale   [2]*MoraleResult
	LastRecovery []RecoveryResult

	// Game Over State
	WinnerPlayerID int // -1 = ongoing, 0 = Player 1, 1 = Player 2, -2 = Tie
	GameOverReason string

	RNG RNG
}

func NewGameEngine(roster1, roster2 *WarbandRoster, isAI1, isAI2 bool) *GameEngine {
	ge := &GameEngine{
		Board:       NewBoard(),
		Rosters:     [2]*WarbandRoster{roster1, roster2},
		RoundNumber: 1,
		TurnNumber:  1,
		RNG:         newDefaultRNG(),
		WinnerPlayerID: -1,
	}

	ge.Players[0] = &PlayerState{
		ID:        0,
		Name:      roster1.Name,
		Color:     ColorPlayerBlue,
		TableEdge: 1, // Bottom (South) by default
		IsAI:      isAI1,
	}
	ge.Players[1] = &PlayerState{
		ID:        1,
		Name:      roster2.Name,
		Color:     ColorPlayerRed,
		TableEdge: 0, // Top (North) by default
		IsAI:      isAI2,
	}

	// Create initial units
	unitID := 1
	for p := 0; p < 2; p++ {
		for _, cfg := range ge.Rosters[p].Units {
			u := &Unit{
				ID:        unitID,
				Player:    p,
				Type:      cfg.Type,
				IsShooter: cfg.IsShooter,
			}
			ge.Players[p].Units = append(ge.Players[p].Units, u)
			unitID++
		}
	}

	ge.StartDeploymentRoll()
	return ge
}

func (ge *GameEngine) AddLog(text string, player int, highlight bool) {
	ge.Log = append(ge.Log, CombatLogEntry{
		Text:      text,
		Player:    player,
		Highlight: highlight,
	})
	if len(ge.Log) > 100 {
		ge.Log = ge.Log[len(ge.Log)-100:]
	}
}

func (ge *GameEngine) PushRollVisual(ev RollVisualEvent) {
	ge.RollQueue = append(ge.RollQueue, ev)
	if len(ge.RollQueue) > 10 {
		ge.RollQueue = ge.RollQueue[len(ge.RollQueue)-10:]
	}
}

// StartDeploymentRoll initiates the d20 roll-off for choosing table edges.
func (ge *GameEngine) StartDeploymentRoll() {
	ge.Phase = PhaseDeployRoll
	r0 := RollDie(20, ge.RNG)
	r1 := RollDie(20, ge.RNG)
	for r0 == r1 {
		r0 = RollDie(20, ge.RNG)
		r1 = RollDie(20, ge.RNG)
	}

	if r0 > r1 {
		ge.DeployWinner = 0
		ge.AddLog(fmt.Sprintf("DEPLOYMENT: %s rolls %d vs %s rolls %d. %s wins choice of table edge!",
			ge.Players[0].Name, r0, ge.Players[1].Name, r1, ge.Players[0].Name), 0, true)
	} else {
		ge.DeployWinner = 1
		ge.AddLog(fmt.Sprintf("DEPLOYMENT: %s rolls %d vs %s rolls %d. %s wins choice of table edge!",
			ge.Players[1].Name, r1, ge.Players[0].Name, r0, ge.Players[1].Name), 1, true)
	}

	ge.PushRollVisual(RollVisualEvent{
		Kind:     RollVisualDeployEdge,
		Title:    "DEPLOYMENT EDGE ROLL-OFF",
		Subtitle: "Higher d20 chooses table side; opponent deploys first",
		Outcome:  fmt.Sprintf("%s wins table edge choice!", ge.Players[ge.DeployWinner].Name),
		WinnerID: ge.DeployWinner,
		SidesA:   20,
		RollA1:   r0,
		LabelA:   ge.Players[0].Name,
		SidesB:   20,
		RollB1:   r1,
		LabelB:   ge.Players[1].Name,
		HasVs:    true,
	})

	// Winner picks their edge (AI picks North, human defaults to South, or customizable)
	ge.Phase = PhaseDeploySelect
	if ge.DeployWinner == 1 {
		ge.SelectEdge(1, 0) // Red takes North
	} else {
		ge.SelectEdge(0, 1) // Blue takes South
	}
}

// SelectEdge sets table edges and transitions to unit placement.
// edge: 0 = Top (North, Y: 0-6"), 1 = Bottom (South, Y: 30-36")
func (ge *GameEngine) SelectEdge(winnerPlayer int, chosenEdge int) {
	ge.Players[winnerPlayer].TableEdge = chosenEdge
	ge.Players[1-winnerPlayer].TableEdge = 1 - chosenEdge

	edgeName := "South (Bottom)"
	if chosenEdge == 0 {
		edgeName = "North (Top)"
	}
	ge.AddLog(fmt.Sprintf("%s chose %s edge. Deployment starts with the other player!",
		ge.Players[winnerPlayer].Name, edgeName), winnerPlayer, false)

	// Non-winner places first die (per video rules)
	ge.DeployTurn = 1 - winnerPlayer
	ge.DeployRemaining0 = append([]*Unit(nil), ge.Players[0].Units...)
	ge.DeployRemaining1 = append([]*Unit(nil), ge.Players[1].Units...)

	ge.Phase = PhaseDeployPlace
}

// DeployUnit places a specific unit within the player's 6" deployment zone.
func (ge *GameEngine) DeployUnit(u *Unit, x, y float64) error {
	p := ge.Players[u.Player]
	// Validate deployment zone: within 6" of edge
	// Clamp to deployment zone
	if p.TableEdge == 0 { // Top (North): Y between 0 and 6
		if y < u.Info().Radius+0.2 {
			y = u.Info().Radius + 0.2
		}
		if y > DeployZoneInches-0.2 {
			y = DeployZoneInches - 0.2
		}
	} else { // Bottom (South): Y between 30 and 36
		minY := TableHeightInches - DeployZoneInches + 0.2
		maxY := TableHeightInches - u.Info().Radius - 0.2
		if y < minY {
			y = minY
		}
		if y > maxY {
			y = maxY
		}
	}
	if x < u.Info().Radius+0.5 {
		x = u.Info().Radius + 0.5
	}
	if x > TableWidthInches-u.Info().Radius-0.5 {
		x = TableWidthInches - u.Info().Radius - 0.5
	}

	// Adjust for overlap if needed
	for _, other := range ge.AllAliveUnits() {
		if other.ID != u.ID && other.X > 0 && other.Y > 0 {
			if Distance(x, y, other.X, other.Y) < (u.Info().Radius + other.Info().Radius + 0.1) {
				x += (u.Info().Radius + other.Info().Radius + 0.3)
				if x > TableWidthInches-2.0 {
					x = 3.0
					if p.TableEdge == 0 {
						y += 1.2
					} else {
						y -= 1.2
					}
				}
			}
		}
	}

	u.X = x
	u.Y = y

	// Remove from remaining queue
	if u.Player == 0 {
		for i, rem := range ge.DeployRemaining0 {
			if rem.ID == u.ID {
				ge.DeployRemaining0 = append(ge.DeployRemaining0[:i], ge.DeployRemaining0[i+1:]...)
				break
			}
		}
	} else {
		for i, rem := range ge.DeployRemaining1 {
			if rem.ID == u.ID {
				ge.DeployRemaining1 = append(ge.DeployRemaining1[:i], ge.DeployRemaining1[i+1:]...)
				break
			}
		}
	}

	ge.AddLog(fmt.Sprintf("%s placed %s at (%.1f\", %.1f\")", p.Name, u.DisplayName(), x, y), u.Player, false)

	// Check if deployment complete
	if len(ge.DeployRemaining0) == 0 && len(ge.DeployRemaining1) == 0 {
		ge.AddLog("All units deployed! Starting Round 1.", -1, true)
		ge.StartRoundInitiative()
		return nil
	}

	// Alternate deployment turns if other player has units left
	if ge.DeployTurn == 0 {
		if len(ge.DeployRemaining1) > 0 {
			ge.DeployTurn = 1
		}
	} else {
		if len(ge.DeployRemaining0) > 0 {
			ge.DeployTurn = 0
		}
	}

	return nil
}


// AutoDeployAll automatically places all remaining units for both armies and starts Round 1.
func (ge *GameEngine) AutoDeployAll() {
	ge.RollQueue = nil
	if len(ge.DeployRemaining0) == 0 && len(ge.DeployRemaining1) == 0 && ge.Phase != PhaseTurnAction {
		ge.SelectEdge(0, 1)
	}
	for len(ge.DeployRemaining0) > 0 || len(ge.DeployRemaining1) > 0 {
		pIdx := ge.DeployTurn
		var queue *[]*Unit
		if pIdx == 0 {
			queue = &ge.DeployRemaining0
		} else {
			queue = &ge.DeployRemaining1
		}
		if len(*queue) == 0 {
			ge.DeployTurn = 1 - pIdx
			continue
		}
		u := (*queue)[0]
		p := ge.Players[pIdx]

		var depY float64
		if p.TableEdge == 0 { // North
			depY = 1.5 + float64(len(*queue)%3)*1.5
		} else { // South
			depY = TableHeightInches - (1.5 + float64(len(*queue)%3)*1.5)
		}

		totalUnits := len(p.Units)
		stepX := (TableWidthInches - 6.0) / float64(totalUnits+1)
		slot := totalUnits - len(*queue) + 1
		depX := 3.0 + float64(slot)*stepX

		_ = ge.DeployUnit(u, depX, depY)
	}
}

// StartRoundInitiative rolls d20 + casualties for each player.
func (ge *GameEngine) StartRoundInitiative() {
	ge.Phase = PhaseInitiativeRoll
	ge.TurnNumber = 1
	ge.SelectedUnitID = 0
	ge.HasSelectedActed = false

	cas0 := ge.Players[0].Casualties
	cas1 := ge.Players[1].Casualties

	r0 := RollDie(20, ge.RNG)
	r1 := RollDie(20, ge.RNG)
	tot0 := r0 + cas0
	tot1 := r1 + cas1

	for tot0 == tot1 {
		r0 = RollDie(20, ge.RNG)
		r1 = RollDie(20, ge.RNG)
		tot0 = r0 + cas0
		tot1 = r1 + cas1
	}

	if tot0 > tot1 {
		ge.InitiativeWinner = 0
		ge.AddLog(fmt.Sprintf("INITIATIVE (Round %d): %s rolls %d + %d casualties = %d vs %s rolls %d + %d casualties = %d. %s takes Turn 1!",
			ge.RoundNumber, ge.Players[0].Name, r0, cas0, tot0, ge.Players[1].Name, r1, cas1, tot1, ge.Players[0].Name), 0, true)
	} else {
		ge.InitiativeWinner = 1
		ge.AddLog(fmt.Sprintf("INITIATIVE (Round %d): %s rolls %d + %d casualties = %d vs %s rolls %d + %d casualties = %d. %s takes Turn 1!",
			ge.RoundNumber, ge.Players[1].Name, r1, cas1, tot1, ge.Players[0].Name, r0, cas0, tot0, ge.Players[1].Name), 1, true)
	}

	ge.PushRollVisual(RollVisualEvent{
		Kind:     RollVisualInitiative,
		Title:    fmt.Sprintf("ROUND %d INITIATIVE ROLL", ge.RoundNumber),
		Subtitle: "Roll d20 + Casualties (catch-up bonus) • Highest goes first",
		Outcome:  fmt.Sprintf("★ %s Takes Turn 1! ★", ge.Players[ge.InitiativeWinner].Name),
		WinnerID: ge.InitiativeWinner,
		SidesA:   20,
		RollA1:   r0,
		BonusA:   cas0,
		LabelA:   ge.Players[0].Name,
		SidesB:   20,
		RollB1:   r1,
		BonusB:   cas1,
		LabelB:   ge.Players[1].Name,
		HasVs:    true,
	})

	ge.StartTurn(1)
}

// StartTurn rolls 2d6 + casualties for the active player's AP pool.
// Round turns alternate: A-B-A-B (where A is InitiativeWinner).
func (ge *GameEngine) StartTurn(turnNum int) {
	ge.TurnNumber = turnNum
	if turnNum%2 == 1 {
		ge.ActivePlayerIdx = ge.InitiativeWinner
	} else {
		ge.ActivePlayerIdx = 1 - ge.InitiativeWinner
	}

	// Reset per-turn move count and shooting restrictions for the active player's units
	for _, u := range ge.GetFriendlyUnits(ge.ActivePlayerIdx) {
		u.MovesMade = 0
		u.HasShot = false
	}

	p := ge.Players[ge.ActivePlayerIdx]
	d1 := RollDie(6, ge.RNG)
	d2 := RollDie(6, ge.RNG)
	ge.ActivationPoints = d1 + d2 + p.Casualties
	ge.SelectedUnitID = 0
	ge.HasSelectedActed = false

	ge.AddLog(fmt.Sprintf("TURN %d/4: %s rolls 2d6 [%d, %d] + %d casualties = %d ACTIVATION POINTS!",
		turnNum, p.Name, d1, d2, p.Casualties, ge.ActivationPoints), ge.ActivePlayerIdx, true)

	ge.PushRollVisual(RollVisualEvent{
		Kind:     RollVisualTurnAP,
		Title:    fmt.Sprintf("TURN %d/4 ACTIVATION POOL", turnNum),
		Subtitle: fmt.Sprintf("%s rolls 2d6 + %d casualties", p.Name, p.Casualties),
		Outcome:  fmt.Sprintf("%s gains %d Activation Points!", p.Name, ge.ActivationPoints),
		WinnerID: ge.ActivePlayerIdx,
		SidesA:   6,
		RollA1:   d1,
		RollA2:   d2,
		BonusA:   p.Casualties,
		LabelA:   p.Name,
		HasVs:    false,
	})

	ge.Phase = PhaseTurnAction
}

// SelectUnit selects a die to activate.
// Rules requirement: "You must fully resolve one die's actions before moving to the next."
func (ge *GameEngine) SelectUnit(u *Unit) error {
	if u.Player != ge.ActivePlayerIdx {
		return fmt.Errorf("not your unit")
	}
	if u.IsCasualty {
		return fmt.Errorf("unit has been eliminated")
	}
	if u.IsWounded {
		return fmt.Errorf("unit is showing '1' (wounded) and cannot act for the rest of the round")
	}
	if ge.HasSelectedActed && ge.SelectedUnitID != 0 && ge.SelectedUnitID != u.ID {
		return fmt.Errorf("must fully resolve current die before activating another")
	}
	ge.SelectedUnitID = u.ID
	return nil
}


// MoveActiveUnit moves the selected die to (destX, destY).
// Handles distance, obstacle penalty, AP cost deduction, and melee charge triggers.
func (ge *GameEngine) MoveActiveUnit(destX, destY float64) error {
	u := ge.GetUnit(ge.SelectedUnitID)
	if u == nil {
		return fmt.Errorf("no unit selected")
	}
	if u.HasShot {
		return fmt.Errorf("unit has already fired this turn and cannot move")
	}
	if u.MovesMade >= MaxMovementSteps {
		return fmt.Errorf("unit has reached maximum 3 movements (18 inches) this turn")
	}

	dist := Distance(u.X, u.Y, destX, destY)
	if dist < 0.1 {
		return nil
	}

	crossesObstacle := ge.Board.PathCrossesObstacle(u.X, u.Y, destX, destY)
	cost, steps, err := CalculateMoveCost(dist, u.MovesMade, crossesObstacle)
	if err != nil {
		return err
	}

	if cost > ge.ActivationPoints {
		return fmt.Errorf("not enough Activation Points (need %d, have %d)", cost, ge.ActivationPoints)
	}

	// Check if this move charges into base-to-base contact with an enemy
	var chargedEnemy *Unit
	for _, enemy := range ge.GetEnemyUnits(u.Player) {
		if !enemy.IsCasualty {
			contactDist := u.Info().Radius + enemy.Info().Radius + 0.15
			if Distance(destX, destY, enemy.X, enemy.Y) <= contactDist {
				chargedEnemy = enemy
				break
			}
		}
	}

	// Check collision with friendlies (cannot overlap)
	for _, ally := range ge.GetFriendlyUnits(u.Player) {
		if ally.ID != u.ID && !ally.IsCasualty {
			if Distance(destX, destY, ally.X, ally.Y) < (u.Info().Radius + ally.Info().Radius) {
				return fmt.Errorf("cannot end movement overlapping a friendly die")
			}
		}
	}

	// Execute move
	ge.ActivationPoints -= cost
	u.MovesMade += steps
	u.DistanceMoved += dist
	u.LastEvent = fmt.Sprintf("Marched %.1f\" (%d AP)", dist, cost)
	ge.HasSelectedActed = true
	u.X, u.Y = ge.Board.ClampToBoard(destX, destY, u.Info().Radius)

	obstacleNote := ""
	if crossesObstacle {
		obstacleNote = " (crossed obstacle: +1 step)"
	}
	ge.AddLog(fmt.Sprintf("%s moved %.1f\" (cost %d AP, %d left)%s",
		u.DisplayName(), dist, cost, ge.ActivationPoints, obstacleNote), u.Player, false)

	// If charging into melee, immediately resolve combat!
	if chargedEnemy != nil {
		ge.resolveMeleeOutcome(u, chargedEnemy)
	}

	if ge.Phase == PhaseTurnAction && ge.ActivationPoints <= 0 {
		ge.EndTurn()
	}

	return nil
}

func (ge *GameEngine) resolveMeleeOutcome(attacker, defender *Unit) {
	res := ResolveMeleeCombat(attacker, defender, ge.RNG)
	ge.LastMelee = &res
	ge.AddLog(res.LogMessage, attacker.Player, true)

	if res.DefenderWon {
		ge.Players[attacker.Player].Casualties++
		ge.SelectedUnitID = 0
		ge.HasSelectedActed = false
	} else if res.AttackerWon {
		ge.Players[defender.Player].Casualties++
	} else if res.IsTieBounceBack {
		bx, by := FindBounceBackPosition(attacker.X, attacker.Y, defender.X, defender.Y, ge.Board, attacker.Info().Radius)
		attacker.X, attacker.Y = bx, by
		ge.AddLog(fmt.Sprintf("%s bounced back 2 inches to (%.1f\", %.1f\")", attacker.DisplayName(), attacker.X, attacker.Y), attacker.Player, false)
	}

	ge.CheckWipeout()
}

// FindChargeApproach calculates a valid base-to-base charge position and AP cost against target.
func (ge *GameEngine) FindChargeApproach(u, target *Unit) (destX, destY, travelDist float64, cost int, steps int, err error) {
	if u == nil || target == nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("invalid unit or target")
	}
	if u.HasShot {
		return 0, 0, 0, 0, 0, fmt.Errorf("unit has already fired this turn and cannot move")
	}
	if u.MovesMade >= MaxMovementSteps {
		return 0, 0, 0, 0, 0, fmt.Errorf("unit has reached maximum 3 movements this turn")
	}

	dx := target.X - u.X
	dy := target.Y - u.Y
	centerDist := math.Hypot(dx, dy)
	contactDist := u.Info().Radius + target.Info().Radius + 0.05

	if centerDist <= contactDist+0.2 {
		c, st, calcErr := CalculateMoveCost(0.5, u.MovesMade, false)
		if calcErr != nil {
			return 0, 0, 0, 0, 0, calcErr
		}
		if c > ge.ActivationPoints {
			return 0, 0, 0, 0, 0, fmt.Errorf("not enough AP (need %d, have %d)", c, ge.ActivationPoints)
		}
		return u.X, u.Y, 0, c, st, nil
	}

	baseAngle := math.Atan2(u.Y-target.Y, u.X-target.X)
	angles := []float64{0, 0.35, -0.35, 0.70, -0.70, 1.05, -1.05}

	var firstErr error
	for _, offset := range angles {
		ang := baseAngle + offset
		cx := target.X + math.Cos(ang)*contactDist
		cy := target.Y + math.Sin(ang)*contactDist
		cx, cy = ge.Board.ClampToBoard(cx, cy, u.Info().Radius)

		if Distance(cx, cy, target.X, target.Y) > u.Info().Radius+target.Info().Radius+0.15 {
			continue
		}
		overlapsAlly := false
		for _, ally := range ge.GetFriendlyUnits(u.Player) {
			if ally.ID != u.ID && !ally.IsCasualty {
				if Distance(cx, cy, ally.X, ally.Y) < (u.Info().Radius + ally.Info().Radius) {
					overlapsAlly = true
					break
				}
			}
		}
		if overlapsAlly {
			continue
		}

		dist := Distance(u.X, u.Y, cx, cy)
		crosses := ge.Board.PathCrossesObstacle(u.X, u.Y, cx, cy)
		c, st, calcErr := CalculateMoveCost(dist, u.MovesMade, crosses)
		if calcErr != nil {
			if firstErr == nil {
				firstErr = calcErr
			}
			continue
		}
		if c > ge.ActivationPoints {
			if firstErr == nil {
				firstErr = fmt.Errorf("not enough AP (need %d, have %d)", c, ge.ActivationPoints)
			}
			continue
		}
		return cx, cy, dist, c, st, nil
	}

	if firstErr != nil {
		return 0, 0, 0, 0, 0, firstErr
	}
	return 0, 0, 0, 0, 0, fmt.Errorf("no valid charge approach path available")
}

// ChargeTarget moves the selected unit into base-to-base contact with target and executes melee combat.
func (ge *GameEngine) ChargeTarget(target *Unit) error {
	u := ge.GetUnit(ge.SelectedUnitID)
	if u == nil {
		return fmt.Errorf("no unit selected")
	}
	if target == nil || target.IsCasualty || target.Player == u.Player {
		return fmt.Errorf("invalid charge target")
	}

	destX, destY, travelDist, cost, steps, err := ge.FindChargeApproach(u, target)
	if err != nil {
		return err
	}

	ge.ActivationPoints -= cost
	u.MovesMade += steps
	u.DistanceMoved += travelDist
	ge.HasSelectedActed = true
	u.X, u.Y = destX, destY

	if travelDist > 0.1 {
		ge.AddLog(fmt.Sprintf("%s charged %.1f\" into %s (cost %d AP, %d left)",
			u.DisplayName(), travelDist, target.DisplayName(), cost, ge.ActivationPoints), u.Player, false)
	}

	ge.resolveMeleeOutcome(u, target)

	if ge.Phase == PhaseTurnAction && ge.ActivationPoints <= 0 {
		ge.EndTurn()
	}
	return nil
}


// ShootActiveUnit fires the selected unit at targetUnit.
// Costs 1 AP (author errata). Requires line of sight.
func (ge *GameEngine) ShootActiveUnit(target *Unit) error {
	u := ge.GetUnit(ge.SelectedUnitID)
	if u == nil {
		return fmt.Errorf("no unit selected")
	}
	if !u.IsShooter {
		return fmt.Errorf("unit is not equipped with a ranged weapon")
	}
	if u.HasShot {
		return fmt.Errorf("unit has already fired this turn")
	}
	if target.Player == u.Player {
		return fmt.Errorf("cannot shoot friendly unit")
	}
	if target.IsCasualty {
		return fmt.Errorf("target is already eliminated")
	}
	if ge.ActivationPoints < 1 {
		return fmt.Errorf("firing a ranged weapon costs 1 Activation Point (have 0)")
	}

	hasLOS, coverBonus := ge.Board.CheckLineOfSight(u.X, u.Y, target.X, target.Y)
	if !hasLOS {
		return fmt.Errorf("target is blocked by solid obstacle (no Line of Sight)")
	}

	ge.ActivationPoints -= 1
	ge.HasSelectedActed = true

	res := ResolveShootingCombat(u, target, coverBonus, ge.RNG)
	ge.LastShooting = &res
	ge.AddLog(res.LogMessage, u.Player, true)

	// Per official rules: "once you shoot, that die cannot move again in that turn."
	// Since it also cannot shoot again, its activation for this turn is complete.
	ge.DoneWithUnit()

	if ge.Phase == PhaseTurnAction && ge.ActivationPoints <= 0 {
		ge.EndTurn()
	}

	return nil
}


// DoneWithUnit finishes activation for the currently selected die, allowing another die to act.
func (ge *GameEngine) DoneWithUnit() {
	ge.SelectedUnitID = 0
	ge.HasSelectedActed = false
}

// EndTurn completes the current turn and advances to next turn or Morale check.
func (ge *GameEngine) EndTurn() {
	if ge.Phase != PhaseTurnAction {
		return
	}
	ge.DoneWithUnit()
	if ge.TurnNumber < 4 {
		ge.StartTurn(ge.TurnNumber + 1)
	} else {
		ge.StartMoralePhase()
	}
}

// StartMoralePhase runs Morale checks for both players at the end of Turn 4.
func (ge *GameEngine) StartMoralePhase() {
	ge.Phase = PhaseMorale
	ge.AddLog(fmt.Sprintf("--- END OF ROUND %d: MORALE TEST ---", ge.RoundNumber), -1, true)

	m0 := RollMoraleTest(0, ge.Players[0].Casualties, ge.RNG)
	m1 := RollMoraleTest(1, ge.Players[1].Casualties, ge.RNG)
	ge.LastMorale[0] = &m0
	ge.LastMorale[1] = &m1
	ge.AddLog(m0.LogMessage, 0, !m0.Passed)
	ge.AddLog(m1.LogMessage, 1, !m1.Passed)

	moraleOutcome := "Both warbands pass Morale and advance to Recovery!"
	moraleWinner := -1
	if !m0.Passed && m1.Passed {
		moraleOutcome = fmt.Sprintf("DEFEAT: %s flees! %s wins the battle!", ge.Players[0].Name, ge.Players[1].Name)
		moraleWinner = 1
	} else if !m1.Passed && m0.Passed {
		moraleOutcome = fmt.Sprintf("DEFEAT: %s flees! %s wins the battle!", ge.Players[1].Name, ge.Players[0].Name)
		moraleWinner = 0
	} else if !m0.Passed && !m1.Passed {
		if ge.Players[0].Casualties < ge.Players[1].Casualties {
			moraleOutcome = fmt.Sprintf("Both broke! %s wins (fewer casualties: %d vs %d)", ge.Players[0].Name, ge.Players[0].Casualties, ge.Players[1].Casualties)
			moraleWinner = 0
		} else if ge.Players[1].Casualties < ge.Players[0].Casualties {
			moraleOutcome = fmt.Sprintf("Both broke! %s wins (fewer casualties: %d vs %d)", ge.Players[1].Name, ge.Players[1].Casualties, ge.Players[0].Casualties)
			moraleWinner = 1
		} else {
			moraleOutcome = "Both warbands broke Morale with equal casualties! Draw!"
		}
	}
	ge.PushRollVisual(RollVisualEvent{
		Kind:     RollVisualMorale,
		Title:    fmt.Sprintf("END OF ROUND %d: MORALE TEST", ge.RoundNumber),
		Subtitle: "Rule: Must roll d20 > Casualties • Fail = Warband flees & loses!",
		Outcome:  moraleOutcome,
		WinnerID: moraleWinner,
		SidesA:   20,
		RollA1:   m0.Roll,
		BonusA:   m0.Casualties,
		PassedA:  m0.Passed,
		LabelA:   ge.Players[0].Name,
		SidesB:   20,
		RollB1:   m1.Roll,
		BonusB:   m1.Casualties,
		PassedB:  m1.Passed,
		LabelB:   ge.Players[1].Name,
		HasVs:    true,
	})

	if !m0.Passed && m1.Passed {
		ge.GameOver(1, fmt.Sprintf("%s failed Morale Test and fled the field! %s wins!", ge.Players[0].Name, ge.Players[1].Name))
		return
	}
	if !m1.Passed && m0.Passed {
		ge.GameOver(0, fmt.Sprintf("%s failed Morale Test and fled the field! %s wins!", ge.Players[1].Name, ge.Players[0].Name))
		return
	}
	if !m0.Passed && !m1.Passed {
		// Both failed: fewest casualties wins
		if ge.Players[0].Casualties < ge.Players[1].Casualties {
			ge.GameOver(0, fmt.Sprintf("Both warbands broke morale! %s wins with fewer casualties (%d vs %d)!",
				ge.Players[0].Name, ge.Players[0].Casualties, ge.Players[1].Casualties))
		} else if ge.Players[1].Casualties < ge.Players[0].Casualties {
			ge.GameOver(1, fmt.Sprintf("Both warbands broke morale! %s wins with fewer casualties (%d vs %d)!",
				ge.Players[1].Name, ge.Players[1].Casualties, ge.Players[0].Casualties))
		} else {
			ge.GameOver(-2, "Both warbands broke morale with equal casualties! The battle ends in a draw!")
		}
		return
	}

	// Both passed morale! Proceed to Casualty Recovery
	ge.StartRecoveryPhase()
}

// StartRecoveryPhase checks survival for all wounded dice showing "1".
func (ge *GameEngine) StartRecoveryPhase() {
	ge.Phase = PhaseRecovery
	ge.AddLog("--- CASUALTY CHECK (RECOVERY) ---", -1, true)
	ge.LastRecovery = nil

	woundedUnits := []*Unit{}
	for _, u := range ge.AllAliveUnits() {
		if u.IsWounded {
			woundedUnits = append(woundedUnits, u)
		}
	}

	if len(woundedUnits) == 0 {
		ge.AddLog("No wounded units to check for recovery.", -1, false)
	} else {
		for _, u := range woundedUnits {
			res := RollCasualtyRecovery(u, ge.RNG)
			ge.LastRecovery = append(ge.LastRecovery, res)
			ge.AddLog(res.LogMessage, u.Player, true)
			recOutcome := fmt.Sprintf("EVEN (%d) — %s recovers for next round!", res.Roll, u.Info().Rank)
			if !res.Recovered {
				ge.Players[u.Player].Casualties++
				recOutcome = fmt.Sprintf("ODD (%d) — %s succumbed (+1 Casualty)!", res.Roll, u.Info().Rank)
			}
			ge.PushRollVisual(RollVisualEvent{
				Kind:     RollVisualRecovery,
				Title:    fmt.Sprintf("CASUALTY RECOVERY: %s [%s]", u.Info().Rank, u.Info().Name),
				Subtitle: "Rule: Wounded '1' rolls own die • EVEN = Recovers • ODD = Dies",
				Outcome:  recOutcome,
				WinnerID: u.Player,
				SidesA:   u.Info().Sides,
				RollA1:   res.Roll,
				PassedA:  res.Recovered,
				LabelA:   ge.Players[u.Player].Name,
				HasVs:    false,
			})
		}
	}

	if ge.CheckWipeout() {
		return
	}

	// Prepare for next round
	ge.RoundNumber++
	for _, u := range ge.AllAliveUnits() {
		u.MovesMade = 0
		u.HasShot = false
	}

	ge.AddLog(fmt.Sprintf("Round %d starting with new initiative roll!", ge.RoundNumber), -1, true)
	ge.StartRoundInitiative()
}

// CheckWipeout ends the game if either player has 0 surviving units.
func (ge *GameEngine) CheckWipeout() bool {
	alive0 := len(ge.GetAliveUnits(0))
	alive1 := len(ge.GetAliveUnits(1))

	if alive0 == 0 && alive1 == 0 {
		ge.GameOver(-2, "Both warbands have been completely wiped out! A bloody draw!")
		return true
	}
	if alive0 == 0 {
		ge.GameOver(1, fmt.Sprintf("%s has been completely wiped out! %s is victorious!", ge.Players[0].Name, ge.Players[1].Name))
		return true
	}
	if alive1 == 0 {
		ge.GameOver(0, fmt.Sprintf("%s has been completely wiped out! %s is victorious!", ge.Players[1].Name, ge.Players[0].Name))
		return true
	}
	return false
}

func (ge *GameEngine) GameOver(winnerID int, reason string) {
	ge.Phase = PhaseGameOver
	ge.WinnerPlayerID = winnerID
	ge.GameOverReason = reason
	ge.AddLog(fmt.Sprintf("=== GAME OVER: %s ===", reason), winnerID, true)
}

// Helper accessors
func (ge *GameEngine) GetUnit(id int) *Unit {
	for p := 0; p < 2; p++ {
		for _, u := range ge.Players[p].Units {
			if u.ID == id {
				return u
			}
		}
	}
	return nil
}


func (ge *GameEngine) GetFriendlyUnits(player int) []*Unit {
	return ge.Players[player].Units
}

func (ge *GameEngine) GetEnemyUnits(player int) []*Unit {
	return ge.Players[1-player].Units
}

func (ge *GameEngine) GetAliveUnits(player int) []*Unit {
	var res []*Unit
	for _, u := range ge.Players[player].Units {
		if !u.IsCasualty {
			res = append(res, u)
		}
	}
	return res
}

func (ge *GameEngine) AllAliveUnits() []*Unit {
	var res []*Unit
	for p := 0; p < 2; p++ {
		for _, u := range ge.Players[p].Units {
			if !u.IsCasualty {
				res = append(res, u)
			}
		}
	}
	return res
}

// Player colors
var (
	ColorPlayerBlue = struct{ R, G, B, A uint8 }{40, 110, 220, 255}
	ColorPlayerRed  = struct{ R, G, B, A uint8 }{210, 45, 45, 255}
)
