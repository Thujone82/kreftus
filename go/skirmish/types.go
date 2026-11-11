package main

import (
	"fmt"
	"image/color"
)

// Table dimensions in inches (standard 36" x 36" / 3ft x 3ft)
const (
	TableWidthInches  = 36.0
	TableHeightInches = 36.0
	DeployZoneInches  = 6.0
	MaxMovementSteps  = 3
	StepInches        = 6.0
	BounceBackInches  = 2.0
	StandardWarbandPoints = 70
)

// DieType represents the polyhedral die defining a warrior's rank, cost, and health.
type DieType int

const (
	DieD4 DieType = iota
	DieD6
	DieD8
	DieD10
)

type DieInfo struct {
	Type     DieType
	Name     string
	Rank     string
	Sides    int
	Cost     int
	MaxLimit int // 0 = unlimited
	Radius   float64 // physical footprint radius on board in inches
}

var DieCatalog = map[DieType]DieInfo{
	DieD10: {Type: DieD10, Name: "d10", Rank: "Commander", Sides: 10, Cost: 10, MaxLimit: 1, Radius: 1.28},
	DieD8:  {Type: DieD8,  Name: "d8",  Rank: "Captain",   Sides: 8,  Cost: 8,  MaxLimit: 1, Radius: 1.16},
	DieD6:  {Type: DieD6,  Name: "d6",  Rank: "Soldier",   Sides: 6,  Cost: 6,  MaxLimit: 0, Radius: 1.05},
	DieD4:  {Type: DieD4,  Name: "d4",  Rank: "Mercenary", Sides: 4,  Cost: 4,  MaxLimit: 0, Radius: 0.96},
}

// Unit represents an active warrior die on the battlefield.
type Unit struct {
	ID            int
	Player        int // 0 = Blue, 1 = Red
	Type          DieType
	IsShooter     bool
	X             float64 // in inches
	Y             float64 // in inches
	IsWounded     bool    // showing "1" state: cannot act, auto-dies in melee
	MovesMade     int     // 0, 1, 2, or 3 movements this turn
	HasShot       bool    // true if shot this turn (cannot move after shooting)
	IsCasualty    bool    // true if slain and removed from board
	Kills         int     // enemies eliminated in melee
	WoundsDealt   int     // enemies wounded via shooting
	ShotsFired    int     // total ranged shots taken
	MeleeFights   int     // total melee combats fought
	Recoveries    int     // casualty checks survived
	DistanceMoved float64 // total distance marched in inches
	LastEvent     string  // concise summary of latest battle event
}

func (u *Unit) Info() DieInfo {
	return DieCatalog[u.Type]
}

func (u *Unit) DisplayName() string {
	info := u.Info()
	shooter := ""
	if u.IsShooter {
		shooter = " (Shooter)"
	}
	if u.IsWounded {
		return fmt.Sprintf("%s [%s - Wounded '1']%s", info.Rank, info.Name, shooter)
	}
	return fmt.Sprintf("%s [%s]%s", info.Rank, info.Name, shooter)
}

// TerrainType classifies battlefield terrain pieces.
type TerrainType int

const (
	TerrainSoftCover TerrainType = iota // +1 cover bonus, does not block LOS or movement
	TerrainHardCover                    // +3 cover bonus, obstacle to move over (+1 move step)
	TerrainObstacleBlock                // Solid obstacle, blocks LOS and cannot be passed through
)

// Terrain represents an obstacle or cover piece on the board.
type Terrain struct {
	ID        int
	Name      string
	Type      TerrainType
	X         float64 // center X in inches
	Y         float64 // center Y in inches
	Width     float64 // width in inches
	Height    float64 // height in inches
	IsCircle  bool
	Radius    float64
}

// CoverBonus returns the defensive bonus provided by the terrain.
func (t Terrain) CoverBonus() int {
	switch t.Type {
	case TerrainSoftCover:
		return 1
	case TerrainHardCover:
		return 3
	default:
		return 0
	}
}

// PlayerState tracks score, casualties, and warband settings.
type PlayerState struct {
	ID          int
	Name        string
	Color       color.RGBA
	TableEdge   int // 0 = Top (North), 1 = Bottom (South), 2 = Left (West), 3 = Right (East)
	Casualties  int
	Units       []*Unit
	IsAI        bool
}

// ScreenState controls whether the player is on the Main Menu, Builder, or in Battle.
type ScreenState int

const (
	ScreenMainMenu ScreenState = iota
	ScreenBattle
	ScreenWarbandBuilder
)

// GamePhase tracks the exact phase within the Skirmish Dice round structure.
type GamePhase int

const (
	PhaseMenu GamePhase = iota
	PhaseBuilder
	PhaseDeployRoll    // d20 roll-off for table edge choice
	PhaseDeploySelect  // winner chooses edge
	PhaseDeployPlace   // alternating placement within 6"
	PhaseInitiativeRoll // d20 + casualties
	PhaseTurnStart     // roll 2d6 + casualties for AP pool
	PhaseTurnAction    // spend points to move/shoot, melee charges
	PhaseMorale        // round end: d20 > casualties
	PhaseRecovery      // wounded units roll for survival (even = live, odd = die)
	PhaseGameOver
)

// CombatLogEntry holds a roll or battle event description.
type CombatLogEntry struct {
	Text      string
	Player    int
	Highlight bool
}

type RollVisualKind int

const (
	RollVisualInitiative RollVisualKind = iota
	RollVisualTurnAP
	RollVisualDeployEdge
	RollVisualMorale
	RollVisualRecovery
)

// RollVisualEvent captures dice rolls and outcomes for visual roll showcases.
type RollVisualEvent struct {
	Kind     RollVisualKind
	Title    string
	Subtitle string
	Outcome  string
	WinnerID int // 0 = Blue, 1 = Red, -1 = Neutral
	SidesA   int
	RollA1   int
	RollA2   int  // optional 2nd die (for 2d6 AP)
	BonusA   int  // casualties modifier or target check
	PassedA  bool // pass/fail flag for morale/recovery
	LabelA   string
	SidesB   int
	RollB1   int
	BonusB   int
	PassedB  bool
	LabelB   string
	HasVs    bool // true for two-sided rolls (Initiative, Deploy, Morale)
}
