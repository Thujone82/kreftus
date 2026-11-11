package main

import (
	"errors"
	"fmt"
)

// WarbandRoster represents a constructed warband configuration before deployment.
type WarbandRoster struct {
	Name       string
	PointLimit int
	Units      []UnitConfig
}

type UnitConfig struct {
	Type      DieType
	IsShooter bool
}

// CalculateTotalPoints sums up the point cost of a roster.
func (r *WarbandRoster) CalculateTotalPoints() int {
	total := 0
	for _, u := range r.Units {
		total += DieCatalog[u.Type].Cost
	}
	return total
}

// CalculateShooterPoints sums the points of units equipped with ranged weapons.
func (r *WarbandRoster) CalculateShooterPoints() int {
	total := 0
	for _, u := range r.Units {
		if u.IsShooter {
			total += DieCatalog[u.Type].Cost
		}
	}
	return total
}

// Validate ensures the warband follows the official rules:
// - Max 1 d10 Commander
// - Max 1 d8 Captain
// - Total points <= PointLimit
func (r *WarbandRoster) Validate() error {
	d10Count := 0
	d8Count := 0
	totalPts := 0

	for _, u := range r.Units {
		info := DieCatalog[u.Type]
		totalPts += info.Cost
		if u.Type == DieD10 {
			d10Count++
		}
		if u.Type == DieD8 {
			d8Count++
		}
	}

	if d10Count > 1 {
		return errors.New("a warband may have at most 1 d10 Commander")
	}
	if d8Count > 1 {
		return errors.New("a warband may have at most 1 d8 Captain")
	}
	if totalPts > r.PointLimit {
		return fmt.Errorf("total cost (%d pts) exceeds agreed limit (%d pts)", totalPts, r.PointLimit)
	}
	if len(r.Units) == 0 {
		return errors.New("warband must have at least one unit")
	}
	return nil
}

// NewStandardRoster creates a default recommended 70-point balanced warband:
// 1 Commander (10), 1 Captain (8), 6 Soldiers (36), 4 Mercenaries (16) = 70 pts.
func NewStandardRoster(name string, shootersForSoldiers bool) *WarbandRoster {
	roster := &WarbandRoster{
		Name:       name,
		PointLimit: StandardWarbandPoints,
	}

	// 1 Commander (10 pts)
	roster.Units = append(roster.Units, UnitConfig{Type: DieD10, IsShooter: true})
	// 1 Captain (8 pts)
	roster.Units = append(roster.Units, UnitConfig{Type: DieD8, IsShooter: true})
	// 6 Soldiers (36 pts)
	for i := 0; i < 6; i++ {
		roster.Units = append(roster.Units, UnitConfig{Type: DieD6, IsShooter: shootersForSoldiers})
	}
	// 4 Mercenaries (16 pts)
	for i := 0; i < 4; i++ {
		roster.Units = append(roster.Units, UnitConfig{Type: DieD4, IsShooter: !shootersForSoldiers})
	}

	return roster
}

// NewMercenarySwarmRoster creates a horde warband:
// 1 Commander (10), 1 Captain (8), 2 Soldiers (12), 10 Mercenaries (40) = 70 pts.
func NewMercenarySwarmRoster(name string) *WarbandRoster {
	roster := &WarbandRoster{
		Name:       name,
		PointLimit: StandardWarbandPoints,
	}
	roster.Units = append(roster.Units, UnitConfig{Type: DieD10, IsShooter: true})
	roster.Units = append(roster.Units, UnitConfig{Type: DieD8, IsShooter: true})
	for i := 0; i < 2; i++ {
		roster.Units = append(roster.Units, UnitConfig{Type: DieD6, IsShooter: true})
	}
	for i := 0; i < 10; i++ {
		roster.Units = append(roster.Units, UnitConfig{Type: DieD4, IsShooter: false})
	}
	return roster
}
