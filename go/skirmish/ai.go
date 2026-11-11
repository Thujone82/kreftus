package main

import (
	"math"
)

// RunAITurn executes automated actions for an AI player during their turn.
func RunAITurn(ge *GameEngine) {
	if ge.Phase == PhaseDeployPlace {
		runAIDeployment(ge)
		return
	}

	if ge.Phase != PhaseTurnAction {
		return
	}

	aiPlayer := ge.ActivePlayerIdx
	if !ge.Players[aiPlayer].IsAI {
		return
	}

	enemies := ge.GetAliveUnits(1 - aiPlayer)
	if len(enemies) == 0 || ge.ActivationPoints < 1 {
		ge.EndTurn()
		return
	}

	aliveUnits := ge.GetAliveUnits(aiPlayer)
	var readyUnits []*Unit
	movedThisTurn := 0
	shotsThisTurn := 0
	for _, u := range aliveUnits {
		if u.MovesMade > 0 {
			movedThisTurn++
		}
		if u.HasShot {
			shotsThisTurn++
		}
		if !u.IsWounded && !u.HasShot && (u.MovesMade < MaxMovementSteps || u.IsShooter) {
			readyUnits = append(readyUnits, u)
		}
	}

	if len(readyUnits) == 0 {
		ge.EndTurn()
		return
	}

	var bestUnit *Unit
	var bestTarget *Unit
	var bestAction string // "shoot", "charge", "advance"
	var bestDestX, bestDestY float64

	// 1. Instant-kill melee charges on wounded ("1" state) enemies
	for _, u := range readyUnits {
		if u.MovesMade >= MaxMovementSteps {
			continue
		}
		for _, e := range enemies {
			if e.IsWounded {
				dist := Distance(u.X, u.Y, e.X, e.Y)
				if dist <= StepInches*float64(MaxMovementSteps-u.MovesMade) {
					crosses := ge.Board.PathCrossesObstacle(u.X, u.Y, e.X, e.Y)
					cost, _, err := CalculateMoveCost(dist, u.MovesMade, crosses)
					if err == nil && cost <= ge.ActivationPoints {
						bestUnit = u
						bestTarget = e
						bestAction = "charge"
						break
					}
				}
			}
		}
		if bestAction == "charge" {
			break
		}
	}

	// 2. Frontline melee charges (Attacker rolls 2 dice vs Defender 1 die!)
	if bestUnit == nil {
		for _, u := range readyUnits {
			if u.MovesMade >= MaxMovementSteps {
				continue
			}
			for _, e := range enemies {
				dist := Distance(u.X, u.Y, e.X, e.Y)
				if dist <= StepInches {
					crosses := ge.Board.PathCrossesObstacle(u.X, u.Y, e.X, e.Y)
					cost, _, err := CalculateMoveCost(dist, u.MovesMade, crosses)
					if err == nil && cost <= ge.ActivationPoints {
						if u.Info().Sides >= e.Info().Sides || u.Info().Sides >= 6 {
							bestUnit = u
							bestTarget = e
							bestAction = "charge"
							break
						}
					}
				}
			}
			if bestAction == "charge" {
				break
			}
		}
	}

	// 3. Helper closures to find best advance & best shot
	findBestAdvance := func(preferMelee bool) (*Unit, *Unit, float64, float64) {
		var advU, advT *Unit
		var advX, advY float64
		bestScore := -9999.0

		for _, u := range readyUnits {
			if u.MovesMade >= MaxMovementSteps {
				continue
			}
			if preferMelee && u.IsShooter {
				continue
			}
			for _, e := range enemies {
				d := Distance(u.X, u.Y, e.X, e.Y)
				if d <= 1.2 {
					continue
				}
				dx, dy, cost, ok := findBestAdvanceStep(ge, u, e)
				if ok {
					// Favor 1-AP unmoved units advancing smoothly toward nearby enemies
					score := 100.0 - d - float64(cost*10) - float64(u.MovesMade*25)
					if !u.IsShooter {
						score += 20.0
					}
					if score > bestScore {
						bestScore = score
						advU = u
						advT = e
						advX = dx
						advY = dy
					}
				}
			}
		}
		return advU, advT, advX, advY
	}

	findBestShot := func() (*Unit, *Unit, int) {
		var sU, sT *Unit
		bestScore := -999
		for _, u := range readyUnits {
			if !u.IsShooter || u.HasShot {
				continue
			}
			for _, e := range enemies {
				hasLOS, cover := ge.Board.CheckLineOfSight(u.X, u.Y, e.X, e.Y)
				if hasLOS {
					// Score target matchup: attacker die size vs cover + enemy die size
					score := (u.Info().Sides - e.Info().Sides) - cover*3
					if e.IsWounded {
						score += 12
					}
					if score > bestScore {
						bestScore = score
						sU = u
						sT = e
					}
				}
			}
		}
		return sU, sT, bestScore
	}

	// 4. Balance advancing units & shooting!
	// Advance frontline melee units first if fewer units have moved than shots fired
	if bestUnit == nil && movedThisTurn <= shotsThisTurn {
		if u, t, dx, dy := findBestAdvance(true); u != nil {
			bestUnit = u
			bestTarget = t
			bestDestX = dx
			bestDestY = dy
			bestAction = "advance"
		}
	}

	// 5. Shoot high-value or clear targets
	if bestUnit == nil && ge.ActivationPoints >= 1 {
		if sU, sT, score := findBestShot(); sU != nil && score > -4 {
			bestUnit = sU
			bestTarget = sT
			bestAction = "shoot"
		}
	}

	// 6. Advance any unit (melee or shooters seeking better ground / LOS)
	if bestUnit == nil {
		if u, t, dx, dy := findBestAdvance(false); u != nil {
			bestUnit = u
			bestTarget = t
			bestDestX = dx
			bestDestY = dy
			bestAction = "advance"
		}
	}

	// 7. Take any remaining shot if movement isn't possible
	if bestUnit == nil && ge.ActivationPoints >= 1 {
		if sU, sT, _ := findBestShot(); sU != nil {
			bestUnit = sU
			bestTarget = sT
			bestAction = "shoot"
		}
	}

	if bestUnit == nil {
		ge.EndTurn()
		return
	}

	// Execute action
	if err := ge.SelectUnit(bestUnit); err != nil {
		ge.EndTurn()
		return
	}

	switch bestAction {
	case "shoot":
		err := ge.ShootActiveUnit(bestTarget)
		ge.DoneWithUnit()
		if err != nil {
			ge.EndTurn()
		}

	case "charge":
		err := ge.MoveActiveUnit(bestTarget.X, bestTarget.Y)
		ge.DoneWithUnit()
		if err != nil {
			ge.EndTurn()
		}

	case "advance":
		err := ge.MoveActiveUnit(bestDestX, bestDestY)
		ge.DoneWithUnit()
		if err != nil {
			ge.EndTurn()
		}
	}

	if ge.Phase == PhaseTurnAction && ge.ActivePlayerIdx == aiPlayer && ge.ActivationPoints < 1 {
		ge.EndTurn()
	}
}

func findBestAdvanceStep(ge *GameEngine, u *Unit, target *Unit) (float64, float64, int, bool) {
	dx := target.X - u.X
	dy := target.Y - u.Y
	dist := math.Hypot(dx, dy)
	if dist <= 1.2 {
		return 0, 0, 0, false
	}

	baseAngle := math.Atan2(dy, dx)
	stepDist := math.Min(5.8, dist-1.0)
	if stepDist < 1.0 {
		return 0, 0, 0, false
	}

	angles := []float64{0, 0.45, -0.45, 0.85, -0.85}
	for _, offset := range angles {
		ang := baseAngle + offset
		cx := u.X + math.Cos(ang)*stepDist
		cy := u.Y + math.Sin(ang)*stepDist
		cx, cy = ge.Board.ClampToBoard(cx, cy, u.Info().Radius)

		actualDist := Distance(u.X, u.Y, cx, cy)
		if actualDist < 1.0 {
			continue
		}
		crosses := ge.Board.PathCrossesObstacle(u.X, u.Y, cx, cy)
		cost, _, err := CalculateMoveCost(actualDist, u.MovesMade, crosses)
		if err == nil && cost <= ge.ActivationPoints {
			return cx, cy, cost, true
		}
	}
	return 0, 0, 0, false
}

func runAIDeployment(ge *GameEngine) {
	pIdx := ge.DeployTurn
	var queue *[]*Unit
	if pIdx == 0 {
		queue = &ge.DeployRemaining0
	} else {
		queue = &ge.DeployRemaining1
	}

	if len(*queue) == 0 {
		return
	}

	u := (*queue)[0]
	p := ge.Players[pIdx]

	// Determine deployment edge Y
	var depY float64
	if p.TableEdge == 0 { // Top
		depY = 2.0 + float64(len(*queue)%4)*1.0
	} else { // Bottom
		depY = TableHeightInches - (2.0 + float64(len(*queue)%4)*1.0)
	}

	// Spread across X axis
	totalUnits := len(p.Units)
	stepX := (TableWidthInches - 6.0) / float64(totalUnits+1)
	slot := totalUnits - len(*queue) + 1
	depX := 3.0 + float64(slot)*stepX

	depX, depY = ge.Board.ClampToBoard(depX, depY, u.Info().Radius)
	_ = ge.DeployUnit(u, depX, depY)
}
