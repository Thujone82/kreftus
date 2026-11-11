package main

import (
	"errors"
	"math"
)

// MovementCostTable: Cost in points for each movement increment (1-based index)
// Step 1: 1 pt (Total 1)
// Step 2: 3 pts (Total 4)
// Step 3: 5 pts (Total 9)
var StepCosts = []int{0, 1, 3, 5}

// CalculateMoveCost determines the AP cost for a unit to move a specified distance.
// unitMovesMade is how many moves the unit has already executed this turn (0, 1, or 2).
// crossesObstacle adds 1 movement step penalty per the official rules.
func CalculateMoveCost(dist float64, unitMovesMade int, crossesObstacle bool) (cost int, stepsUsed int, err error) {
	if dist <= 0 {
		return 0, 0, nil
	}

	// Calculate base steps based on distance
	stepsNeeded := int(math.Ceil(dist / StepInches))
	if stepsNeeded <= 0 {
		stepsNeeded = 1
	}

	// Obstacle penalty: costs one extra "step" of movement
	if crossesObstacle {
		stepsNeeded++
	}

	totalStepsAfter := unitMovesMade + stepsNeeded
	if totalStepsAfter > MaxMovementSteps {
		return 0, 0, errors.New("movement exceeds the 3-increment limit (18 inches) per turn")
	}

	// Accumulate cost of newly used steps
	for s := unitMovesMade + 1; s <= totalStepsAfter; s++ {
		cost += StepCosts[s]
	}

	return cost, stepsNeeded, nil
}

// CheckMeleeContact returns true if attacker and target are in base-to-base contact.
func CheckMeleeContact(u1, u2 *Unit) bool {
	contactDistance := u1.Info().Radius + u2.Info().Radius + 0.15 // small threshold for tabletop contact
	return Distance(u1.X, u1.Y, u2.X, u2.Y) <= contactDistance
}

// FindBounceBackPosition computes the position 2 inches back along the charge vector.
func FindBounceBackPosition(attackerX, attackerY, defenderX, defenderY float64, board *Board, radius float64) (float64, float64) {
	dx := attackerX - defenderX
	dy := attackerY - defenderY
	length := math.Hypot(dx, dy)
	if length == 0 {
		// Random push if exactly coincident
		dx, dy, length = 1, 0, 1
	}
	normX := dx / length
	normY := dy / length

	// 2 inches away from the defender base
	newX := defenderX + normX*(radius*2+BounceBackInches)
	newY := defenderY + normY*(radius*2+BounceBackInches)

	return board.ClampToBoard(newX, newY, radius)
}
