package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// CycleNextUnit selects the next friendly unit that still has actions remaining.
func (app *GUIApp) CycleNextUnit(reverse bool) {
	if app.ge == nil || app.ge.Phase != PhaseTurnAction {
		return
	}
	friendlies := app.ge.GetAliveUnits(app.ge.ActivePlayerIdx)
	if len(friendlies) == 0 {
		return
	}

	var candidates []*Unit
	for _, u := range friendlies {
		if !u.IsWounded && (!u.HasShot || u.MovesMade < MaxMovementSteps) {
			candidates = append(candidates, u)
		}
	}
	if len(candidates) == 0 {
		candidates = friendlies
	}

	curIdx := -1
	for i, u := range candidates {
		if u.ID == app.ge.SelectedUnitID {
			curIdx = i
			break
		}
	}

	nextIdx := 0
	if reverse {
		if curIdx <= 0 {
			nextIdx = len(candidates) - 1
		} else {
			nextIdx = curIdx - 1
		}
	} else {
		if curIdx >= 0 && curIdx < len(candidates)-1 {
			nextIdx = curIdx + 1
		} else {
			nextIdx = 0
		}
	}

	app.ge.DoneWithUnit()
	app.flash(app.ge.SelectUnit(candidates[nextIdx]))
}

// AutoTargetNearestEnemy finds the closest visible enemy in LOS and shoots them.
func (app *GUIApp) AutoTargetNearestEnemy() {
	if app.ge == nil || app.ge.Phase != PhaseTurnAction || app.ge.SelectedUnitID == 0 {
		return
	}
	u := app.ge.GetUnit(app.ge.SelectedUnitID)
	if u == nil || !u.IsShooter || u.HasShot || app.ge.ActivationPoints < 1 {
		return
	}

	var closestEnemy *Unit
	minDist := 999.0
	for _, enemy := range app.ge.GetEnemyUnits(u.Player) {
		if !enemy.IsCasualty {
			hasLOS, _ := app.ge.Board.CheckLineOfSight(u.X, u.Y, enemy.X, enemy.Y)
			if hasLOS {
				d := Distance(u.X, u.Y, enemy.X, enemy.Y)
				if d < minDist {
					minDist = d
					closestEnemy = enemy
				}
			}
		}
	}

	if closestEnemy != nil {
		app.flash(app.ge.ShootActiveUnit(closestEnemy))
	}
}

// GetChargeableEnemies returns all alive enemy units the given unit can currently afford to charge.
func (app *GUIApp) GetChargeableEnemies(u *Unit) []*Unit {
	if app.ge == nil || u == nil || u.HasShot || u.MovesMade >= MaxMovementSteps {
		return nil
	}
	app.ensureActionCache()
	var out []*Unit
	for _, enemy := range app.ge.GetEnemyUnits(u.Player) {
		if enemy.IsCasualty {
			continue
		}
		if u.ID == app.ge.SelectedUnitID {
			if ch, ok := app.actionCache.charges[enemy.ID]; ok && ch.ok {
				out = append(out, enemy)
			}
			continue
		}
		if _, ok := app.chargeFor(u, enemy); ok {
			out = append(out, enemy)
		}
	}
	return out
}

// AutoChargeTarget charges the currently hovered enemy if reachable, or the best reachable target (prioritizing wounded '1' units).
func (app *GUIApp) AutoChargeTarget(vp Viewport) {
	if app.ge == nil || app.ge.Phase != PhaseTurnAction || app.ge.SelectedUnitID == 0 {
		return
	}
	u := app.ge.GetUnit(app.ge.SelectedUnitID)
	if u == nil {
		return
	}

	// If cursor is hovering a reachable enemy, charge that exact target
	var hoverErr error
	if hov := app.getHoveredUnit(vp); hov != nil && hov.Player != u.Player && !hov.IsCasualty {
		hoverErr = app.ge.ChargeTarget(hov)
		if hoverErr == nil {
			return
		}
	}

	candidates := app.GetChargeableEnemies(u)
	if len(candidates) == 0 {
		app.flash(hoverErr)
		return
	}

	var bestTarget *Unit
	bestScore := 9999.0
	for _, e := range candidates {
		d := Distance(u.X, u.Y, e.X, e.Y)
		score := d
		if e.IsWounded {
			score -= 50.0 // Strongly prioritize guaranteed instant kills!
		}
		if score < bestScore {
			bestScore = score
			bestTarget = e
		}
	}

	if bestTarget != nil {
		app.flash(app.ge.ChargeTarget(bestTarget))
	}
}

// HandleInput processes mouse and keyboard events.
func (app *GUIApp) HandleInput() {
	cx, cy := ebiten.CursorPosition()
	curX, curY := float64(cx), float64(cy)

	// 1. Exit Confirmation Modal Interactivity
	if app.showExitConfirm {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			app.showExitConfirm = false
			return
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyY) {
			app.ReturnToMenu()
			return
		}
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			vp := app.calcViewport()
			modalW, modalH := 520.0, 230.0
			mx := (vp.ScreenW - modalW) / 2
			my := (vp.ScreenH - modalH) / 2

			// Button 1: Stay & Play (ESC) -> mx+35, my+148, w: 215, h: 42
			if curX >= mx+35 && curX <= mx+250 && curY >= my+148 && curY <= my+190 {
				app.showExitConfirm = false
				return
			}
			// Button 2: Leave Battle -> mx+270, my+148, w: 215, h: 42
			if curX >= mx+270 && curX <= mx+485 && curY >= my+148 && curY <= my+190 {
				app.ReturnToMenu()
				return
			}
		}
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if app.showRulesModal {
			app.showRulesModal = false
			return
		}
		if app.screen == ScreenBattle {
			app.showExitConfirm = true
			return
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		app.showRulesModal = !app.showRulesModal
		return
	}

	// 2. Rules Modal Interactivity
	if app.showRulesModal {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			vp := app.calcViewport()
			modalW, modalH := 700.0, 540.0
			mx := (vp.ScreenW - modalW) / 2
			my := (vp.ScreenH - modalH) / 2
			btnX, btnY := mx+modalW/2-80, my+modalH-50
			if curX >= btnX && curX <= btnX+160 && curY >= btnY && curY <= btnY+36 {
				app.showRulesModal = false
			}
		}
		return
	}

	// 2. Main Menu Interactivity
	if app.screen == ScreenMainMenu {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			chrome := app.menuChrome()
			hit := func(i int) bool {
				y := chrome.startY + float64(i)*chrome.step
				return curX >= chrome.btnX && curX <= chrome.btnX+chrome.btnW && curY >= y && curY <= y+chrome.btnH
			}
			if hit(0) {
				app.StartNewGame(true, true)
				return
			}
			if hit(1) {
				app.StartNewGame(true, false)
				return
			}
			if hit(2) {
				app.StartNewGame(false, false)
				return
			}
			if hit(3) {
				app.showRulesModal = true
				return
			}
			if hit(4) {
				app.shouldQuit = true
				return
			}
		}
		return
	}

	if app.ge == nil {
		return
	}

	vp := app.calcViewport()

	if app.needsHandoff() {
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			app.ackedHandoffKey = app.handoffKey()
			app.handoffHoldDraw = true
		}
		return
	}

	// 3. Animated Roll Showcase Playback & Dismissal (Runs even before Game Over so Morale failures are always visible!)
	if app.activeRoll == nil && len(app.ge.RollQueue) > 0 {
		ev := app.ge.RollQueue[0]
		app.ge.RollQueue = app.ge.RollQueue[1:]
		app.activeRoll = &ev
		app.rollFrame = 0
	}
	if app.activeRoll != nil {
		app.rollFrame++
		spinFrames := 36   // ~0.60s (+50%)
		totalFrames := 132 // ~2.20s (+50%)
		if app.activeRoll.Kind == RollVisualMorale {
			totalFrames = 165 // ~2.75s for reading both warbands' Morale status
		}
		skipPressed := inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter)
		clicked := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
		if skipPressed || clicked {
			if app.rollFrame < spinFrames {
				app.rollFrame = spinFrames // Snap immediately to final rolled outcome
			} else {
				app.activeRoll = nil
				app.rollFrame = 0
			}
			return
		}
		if app.rollFrame >= totalFrames {
			app.activeRoll = nil
			app.rollFrame = 0
		}
		return // Wait for showcase to finish before Game Over modal, AI, or board clicks
	}

	// 3.5 Game Over Modal (After any queued Morale/Recovery roll showcases finish)
	if app.ge.Phase == PhaseGameOver {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			modalW, modalH := 540.0, 240.0
			mx := (vp.ScreenW - modalW) / 2
			my := (vp.ScreenH - modalH) / 2

			// Play Again (AI)
			if curX >= mx+40 && curX <= mx+180 && curY >= my+150 && curY <= my+194 {
				app.StartNewGame(true, true)
				return
			}
			// Pass & Play
			if curX >= mx+195 && curX <= mx+335 && curY >= my+150 && curY <= my+194 {
				app.StartNewGame(false, true)
				return
			}
			// Main Menu
			if curX >= mx+350 && curX <= mx+490 && curY >= my+150 && curY <= my+194 {
				app.ReturnToMenu()
				return
			}
		}
		return
	}

	// 4. Sidebar Button Clicks (Always responsive)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		sbX := vp.SbX
		sbY := vp.SbY
		sbW := vp.SbW
		cardW := (sbW - 20.0 - 8.0) / 2.0
		p1X := sbX + 10.0 + cardW + 8.0
		btnY := sbY + 76.0

		if app.ge.Phase == PhaseDeployPlace {
			// "Auto-Deploy Remaining" button
			if curX >= sbX+10 && curX <= sbX+sbW-10 && curY >= btnY && curY <= btnY+34 {
				app.ge.AutoDeployAll()
				return
			}
			btnY += 40.0
		} else {
			// Row 1: Next Die (Left) and End Turn (Right)
			if !app.ge.Players[app.ge.ActivePlayerIdx].IsAI {
				if curX >= sbX+10 && curX <= sbX+10+cardW && curY >= btnY && curY <= btnY+32 {
					app.CycleNextUnit(false)
					return
				}
				if curX >= p1X && curX <= p1X+cardW && curY >= btnY && curY <= btnY+32 {
					app.ge.EndTurn()
					return
				}
			}
			btnY += 36.0

			// Row 2: Action buttons ("Shoot (S)" and/or "Charge (C)")
			if app.ge.SelectedUnitID != 0 {
				sel := app.ge.GetUnit(app.ge.SelectedUnitID)
				if sel != nil {
					canShoot := sel.IsShooter && !sel.HasShot && app.ge.ActivationPoints >= 1
					canCharge := len(app.GetChargeableEnemies(sel)) > 0

					if canShoot && canCharge {
						if curX >= sbX+10 && curX <= sbX+10+cardW && curY >= btnY && curY <= btnY+30 {
							app.AutoTargetNearestEnemy()
							return
						}
						if curX >= p1X && curX <= p1X+cardW && curY >= btnY && curY <= btnY+30 {
							app.AutoChargeTarget(vp)
							return
						}
						btnY += 34.0
					} else if canShoot {
						if curX >= sbX+10 && curX <= sbX+sbW-10 && curY >= btnY && curY <= btnY+30 {
							app.AutoTargetNearestEnemy()
							return
						}
						btnY += 34.0
					} else if canCharge {
						if curX >= sbX+10 && curX <= sbX+sbW-10 && curY >= btnY && curY <= btnY+30 {
							app.AutoChargeTarget(vp)
							return
						}
						btnY += 34.0
					}
				}
			}

			// Row 3: Rules (Left) and Main Menu (Right)
			if curX >= sbX+10 && curX <= sbX+10+cardW && curY >= btnY && curY <= btnY+26 {
				app.showRulesModal = true
				return
			}
			if curX >= p1X && curX <= p1X+cardW && curY >= btnY && curY <= btnY+26 {
				app.showExitConfirm = true
				return
			}
		}
	}

	// 5. Hotkeys During Human Turn
	if app.ge.Phase == PhaseTurnAction && !app.ge.Players[app.ge.ActivePlayerIdx].IsAI {
		// TAB to cycle next unit (Shift+TAB for previous)
		if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
			shift := ebiten.IsKeyPressed(ebiten.KeyShift)
			app.CycleNextUnit(shift)
			return
		}
		// S to shoot nearest enemy
		if inpututil.IsKeyJustPressed(ebiten.KeyS) {
			app.AutoTargetNearestEnemy()
			return
		}
		// C to melee charge hovered or nearest reachable enemy
		if inpututil.IsKeyJustPressed(ebiten.KeyC) {
			app.AutoChargeTarget(vp)
			return
		}
		// SPACE to end turn
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			app.ge.EndTurn()
			return
		}
		// ENTER or D for Done with unit / next
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyD) {
			app.CycleNextUnit(false)
			return
		}
		// M to prompt exit to Main Menu
		if inpututil.IsKeyJustPressed(ebiten.KeyM) {
			app.showExitConfirm = true
			return
		}
	}

	// 6. AI Automated Actions
	if app.ge.Phase == PhaseTurnAction && app.ge.Players[app.ge.ActivePlayerIdx].IsAI {
		app.aiTimer++
		if app.aiTimer >= 35 {
			app.aiTimer = 0
			RunAITurn(app.ge)
		}
		return
	}

	if app.ge.Phase == PhaseDeployPlace && app.ge.Players[app.ge.DeployTurn].IsAI {
		app.aiTimer++
		if app.aiTimer >= 20 {
			app.aiTimer = 0
			RunAITurn(app.ge)
		}
		return
	}

	// 7. Battlefield Mouse Interaction
	wx, wy := vp.ScreenToWorld(curX, curY)
	clickedOnBoard := wx >= 0 && wx <= TableWidthInches && wy >= 0 && wy <= TableHeightInches

	// Deployment Clicks
	if app.ge.Phase == PhaseDeployPlace {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && clickedOnBoard {
			pIdx := app.ge.DeployTurn
			var queue *[]*Unit
			if pIdx == 0 {
				queue = &app.ge.DeployRemaining0
			} else {
				queue = &app.ge.DeployRemaining1
			}
			if len(*queue) > 0 {
				u := (*queue)[0]
				app.flash(app.ge.DeployUnit(u, wx, wy))
			}
		}
		return
	}

	// Battle Turn Mouse Actions
	if app.ge.Phase == PhaseTurnAction && !app.ge.Players[app.ge.ActivePlayerIdx].IsAI {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && clickedOnBoard {
			// A. If a friendly unit is selected and clicked an enemy:
			//    - Shooter with clear LOS shoots
			//    - Melee unit (or Shooter whose shot is blocked) charges into melee!
			if app.ge.SelectedUnitID != 0 {
				selected := app.ge.GetUnit(app.ge.SelectedUnitID)
				if selected != nil {
					for _, enemy := range app.ge.GetEnemyUnits(selected.Player) {
						if !enemy.IsCasualty && Distance(wx, wy, enemy.X, enemy.Y) <= (enemy.Info().Radius+0.7) {
							if selected.IsShooter && !selected.HasShot && app.ge.ActivationPoints >= 1 {
								if hasLOS, _ := app.sightFor(selected, enemy); hasLOS {
									app.flash(app.ge.ShootActiveUnit(enemy))
									return
								}
							}
							// Melee charge when unit is melee warrior or shot is blocked
							app.flash(app.ge.ChargeTarget(enemy))
							return
						}
					}
				}
			}

			// B. Check if clicked a friendly unit to select
			var clickedFriendly *Unit
			for _, ally := range app.ge.GetFriendlyUnits(app.ge.ActivePlayerIdx) {
				if !ally.IsCasualty && Distance(wx, wy, ally.X, ally.Y) <= (ally.Info().Radius+0.7) {
					clickedFriendly = ally
					break
				}
			}

			if clickedFriendly != nil {
				if !clickedFriendly.IsWounded {
					app.ge.DoneWithUnit()
					app.flash(app.ge.SelectUnit(clickedFriendly))
				}
				return
			}

			// C. Move selected unit to click location
			if app.ge.SelectedUnitID != 0 {
				app.flash(app.ge.MoveActiveUnit(wx, wy))
				return
			}
		}

		// Right Click:
		//  - On an enemy within charge range: Melee Charge! (Works for both Melee & Shooters!)
		//  - On open ground: Done with current die
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
			if clickedOnBoard && app.ge.SelectedUnitID != 0 {
				selected := app.ge.GetUnit(app.ge.SelectedUnitID)
				if selected != nil {
					for _, enemy := range app.ge.GetEnemyUnits(selected.Player) {
						if !enemy.IsCasualty && Distance(wx, wy, enemy.X, enemy.Y) <= (enemy.Info().Radius+0.7) {
							app.flash(app.ge.ChargeTarget(enemy))
							return
						}
					}
				}
			}
			app.ge.DoneWithUnit()
		}
	}
}
