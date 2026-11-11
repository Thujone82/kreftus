package main

import (
	"math"
)

// Board contains terrain and spatial calculation helpers for the 36" x 36" battlefield.
type Board struct {
	Width    float64
	Height   float64
	Terrains []Terrain
}

// NewBoard creates a default 36" x 36" tabletop with balanced tactical terrain (ruins, stone walls, hedges).
func NewBoard() *Board {
	b := &Board{
		Width:  TableWidthInches,
		Height: TableHeightInches,
	}
	b.PopulateDefaultTerrain()
	return b
}

func (b *Board) PopulateDefaultTerrain() {
	b.Terrains = []Terrain{
		// Center Stone Ruin (Hard cover + Obstacle)
		{ID: 1, Name: "Center Ruins North", Type: TerrainHardCover, X: 18, Y: 14, Width: 8, Height: 1.2},
		{ID: 2, Name: "Center Ruins South", Type: TerrainHardCover, X: 18, Y: 22, Width: 8, Height: 1.2},
		{ID: 3, Name: "Center Pillar West", Type: TerrainObstacleBlock, X: 13, Y: 18, Width: 2, Height: 2},
		{ID: 4, Name: "Center Pillar East", Type: TerrainObstacleBlock, X: 23, Y: 18, Width: 2, Height: 2},

		// Flank Stone Walls (Hard cover + Obstacle)
		{ID: 5, Name: "West Wall", Type: TerrainHardCover, X: 8, Y: 18, Width: 1.2, Height: 6},
		{ID: 6, Name: "East Wall", Type: TerrainHardCover, X: 28, Y: 18, Width: 1.2, Height: 6},

		// Hedges / Bushes (Soft cover: +1 defense, does not block movement or LOS)
		{ID: 7, Name: "Northwest Hedge", Type: TerrainSoftCover, X: 10, Y: 10, Width: 5, Height: 1.5},
		{ID: 8, Name: "Southeast Hedge", Type: TerrainSoftCover, X: 26, Y: 26, Width: 5, Height: 1.5},
		{ID: 9, Name: "Northeast Hedge", Type: TerrainSoftCover, X: 26, Y: 10, Width: 5, Height: 1.5},
		{ID: 10, Name: "Southwest Hedge", Type: TerrainSoftCover, X: 10, Y: 26, Width: 5, Height: 1.5},

		// Rocks / Barrels (Circular hard cover)
		{ID: 11, Name: "North Boulders", Type: TerrainHardCover, X: 18, Y: 9, Width: 2.5, Height: 2.5, IsCircle: true, Radius: 1.25},
		{ID: 12, Name: "South Boulders", Type: TerrainHardCover, X: 18, Y: 27, Width: 2.5, Height: 2.5, IsCircle: true, Radius: 1.25},
	}
}

// Distance returns Euclidean distance in inches between two coordinates.
func Distance(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return math.Hypot(dx, dy)
}

// LineIntersectsRect tests if a line segment (x1, y1)-(x2, y2) intersects an axis-aligned rectangle.
func LineIntersectsRect(x1, y1, x2, y2, rx, ry, rw, rh float64) bool {
	// Center-based rect to bounding box
	minX := rx - rw/2
	maxX := rx + rw/2
	minY := ry - rh/2
	maxY := ry + rh/2

	// Check if either endpoint is inside rect
	if (x1 >= minX && x1 <= maxX && y1 >= minY && y1 <= maxY) ||
		(x2 >= minX && x2 <= maxX && y2 >= minY && y2 <= maxY) {
		return true
	}

	// Cohen-Sutherland style / segment vs 4 edges
	edges := [4][4]float64{
		{minX, minY, maxX, minY}, // Top
		{maxX, minY, maxX, maxY}, // Right
		{maxX, maxY, minX, maxY}, // Bottom
		{minX, maxY, minX, minY}, // Left
	}

	for _, e := range edges {
		if LineSegmentsIntersect(x1, y1, x2, y2, e[0], e[1], e[2], e[3]) {
			return true
		}
	}
	return false
}

// LineIntersectsCircle tests if a line segment intersects a circle.
func LineIntersectsCircle(x1, y1, x2, y2, cx, cy, radius float64) bool {
	dx := x2 - x1
	dy := y2 - y1
	lengthSq := dx*dx + dy*dy
	if lengthSq == 0 {
		return Distance(x1, y1, cx, cy) <= radius
	}

	// Project circle center onto line segment, clamped between 0 and 1
	t := ((cx-x1)*dx + (cy-y1)*dy) / lengthSq
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}

	closestX := x1 + t*dx
	closestY := y1 + t*dy
	return Distance(cx, cy, closestX, closestY) <= radius
}

// LineSegmentsIntersect checks if segment (p0_x, p0_y)-(p1_x, p1_y) intersects (p2_x, p2_y)-(p3_x, p3_y).
func LineSegmentsIntersect(p0x, p0y, p1x, p1y, p2x, p2y, p3x, p3y float64) bool {
	s10x := p1x - p0x
	s10y := p1y - p0y
	s32x := p3x - p2x
	s32y := p3y - p2y

	denom := s10x*s32y - s10y*s32x
	if math.Abs(denom) < 1e-9 {
		return false // Collinear or parallel
	}

	denomPositive := denom > 0
	s02x := p0x - p2x
	s02y := p0y - p2y

	sNumer := s10x*s02y - s10y*s02x
	if (sNumer < 0) == denomPositive {
		return false
	}

	tNumer := s32x*s02y - s32y*s02x
	if (tNumer < 0) == denomPositive {
		return false
	}

	if (sNumer > denom) == denomPositive || (tNumer > denom) == denomPositive {
		return false
	}

	return true
}

// CheckLineOfSight determines if line of sight exists between attacker and target,
// and calculates cover bonus (+0, +1 for soft, +3 for hard).
// Mid-field hard obstacles and stone walls block line of sight completely unless a unit
// is taking cover directly behind it.
func (b *Board) CheckLineOfSight(x1, y1, x2, y2 float64) (hasLOS bool, coverBonus int) {
	coverBonus = 0
	hasLOS = true

	// Check each terrain
	for _, t := range b.Terrains {
		hit := false
		if t.IsCircle {
			hit = LineIntersectsCircle(x1, y1, x2, y2, t.X, t.Y, t.Radius)
		} else {
			hit = LineIntersectsRect(x1, y1, x2, y2, t.X, t.Y, t.Width, t.Height)
		}

		if hit {
			if t.Type == TerrainObstacleBlock {
				// Solid pillars block line of sight completely
				return false, 0
			}
			if t.Type == TerrainHardCover {
				// Stone ruins and walls block line of sight unless attacker or target is adjacent to it (taking cover)
				reach := math.Max(t.Width, t.Height)/2 + 2.0
				attackerNear := Distance(x1, y1, t.X, t.Y) <= reach
				targetNear := Distance(x2, y2, t.X, t.Y) <= reach

				if attackerNear || targetNear {
					if 3 > coverBonus {
						coverBonus = 3
					}
				} else {
					// Hard obstacle in the middle of field blocks line of sight
					return false, 0
				}
			} else if t.Type == TerrainSoftCover {
				if 1 > coverBonus {
					coverBonus = 1
				}
			}
		}
	}

	return hasLOS, coverBonus
}

// PathCrossesObstacle checks if a move from (x1, y1) to (x2, y2) crosses an obstacle higher than the die.
// Over obstacles higher than the die, rules state it costs +1 movement step.
func (b *Board) PathCrossesObstacle(x1, y1, x2, y2 float64) bool {
	for _, t := range b.Terrains {
		if t.Type == TerrainHardCover || t.Type == TerrainObstacleBlock {
			if t.IsCircle {
				if LineIntersectsCircle(x1, y1, x2, y2, t.X, t.Y, t.Radius) {
					return true
				}
			} else {
				if LineIntersectsRect(x1, y1, x2, y2, t.X, t.Y, t.Width, t.Height) {
					return true
				}
			}
		}
	}
	return false
}

// GetNearbyCover returns the terrain name and defense cover bonus (+0, +1, +3) for a unit's position.
func (b *Board) GetNearbyCover(x, y float64) (string, int) {
	bestBonus := 0
	bestName := "Open Ground"

	for _, t := range b.Terrains {
		near := false
		if t.IsCircle {
			near = Distance(x, y, t.X, t.Y) <= t.Radius+1.8
		} else {
			reachX := t.Width/2 + 1.8
			reachY := t.Height/2 + 1.8
			near = math.Abs(x-t.X) <= reachX && math.Abs(y-t.Y) <= reachY
		}

		if near {
			bonus := t.CoverBonus()
			if bonus >= bestBonus {
				bestBonus = bonus
				bestName = t.Name
			}
		}
	}
	return bestName, bestBonus
}

// ClampToBoard keeps a point inside the 36" x 36" table boundaries.
func (b *Board) ClampToBoard(x, y, radius float64) (float64, float64) {
	if x < radius {
		x = radius
	}
	if x > b.Width-radius {
		x = b.Width - radius
	}
	if y < radius {
		y = radius
	}
	if y > b.Height-radius {
		y = b.Height - radius
	}
	return x, y
}
