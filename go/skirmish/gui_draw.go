package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Viewport transforms 36" x 36" tabletop inches to screen pixels.
type Viewport struct {
	ScreenW float64
	ScreenH float64
	TableX  float64
	TableY  float64
	TablePx float64
	Scale   float64
	SbX     float64
	SbY     float64
	SbW     float64
	SbH     float64
}

func (vp *Viewport) InchToPx(inches float64) float64 {
	return inches * vp.Scale
}

func (vp *Viewport) WorldToScreen(x, y float64) (float64, float64) {
	return vp.TableX + x*vp.Scale, vp.TableY + y*vp.Scale
}

func (vp *Viewport) ScreenToWorld(sx, sy float64) (float64, float64) {
	return (sx - vp.TableX) / vp.Scale, (sy - vp.TableY) / vp.Scale
}

// DrawMainMenu renders the opening title and game setup screen.
func (app *GUIApp) DrawMainMenu(screen *ebiten.Image) {
	w, h := float64(app.w), float64(app.h)
	if bg := app.ensureMenuBackdrop(); bg != nil {
		screen.DrawImage(bg, nil)
	} else {
		screen.Fill(color.RGBA{R: 18, G: 22, B: 30, A: 255})
	}

	chrome := app.menuChrome()
	by := chrome.bannerY
	bannerH := chrome.bannerH
	bannerW := chrome.bannerW
	bx := chrome.bannerX

	// 1. Draw skirmish.png Header Logo with STRICT NATURAL ASPECT RATIO
	if app.headerImg != nil {
		bw := float64(app.headerImg.Bounds().Dx())
		bh := float64(app.headerImg.Bounds().Dy())
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(bannerW/bw, bannerH/bh)
		op.GeoM.Translate(bx, by)
		screen.DrawImage(app.headerImg, op)
	} else {
		vector.DrawFilledRect(screen, float32(bx), float32(by), float32(bannerW), float32(bannerH), color.RGBA{30, 36, 52, 255}, false)
		title := "SKIRMISH DICE"
		titleFace := GetFace(36, FontBold)
		tw, _ := text.Measure(title, titleFace, 0)
		tOp := &text.DrawOptions{}
		tOp.GeoM.Translate(bx+(bannerW-tw)/2, by+55)
		tOp.ColorScale.ScaleWithColor(color.RGBA{240, 215, 100, 255})
		text.Draw(screen, title, titleFace, tOp)
	}

	// Subtitle
	sub := "Tactical Tabletop Wargame • Based on Skirmish Dice by Gray Army Gaming"
	subFace := GetFace(13, FontRegular)
	sw, _ := text.Measure(sub, subFace, 0)
	subOp := &text.DrawOptions{}
	subOp.GeoM.Translate((w-sw)/2, by+bannerH+16)
	subOp.ColorScale.ScaleWithColor(color.RGBA{180, 195, 220, 255})
	text.Draw(screen, sub, subFace, subOp)

	app.drawButton(screen, chrome.btnX, chrome.startY, chrome.btnW, chrome.btnH, "Quick Battle vs AI (Auto-Deployed)", color.RGBA{45, 110, 185, 255})
	app.drawButton(screen, chrome.btnX, chrome.startY+chrome.step, chrome.btnW, chrome.btnH, "Standard Battle vs AI (Manual Deploy)", color.RGBA{40, 95, 155, 255})
	app.drawButton(screen, chrome.btnX, chrome.startY+chrome.step*2, chrome.btnW, chrome.btnH, "Pass & Play (Local 2-Player)", color.RGBA{50, 135, 95, 255})
	app.drawButton(screen, chrome.btnX, chrome.startY+chrome.step*3, chrome.btnW, chrome.btnH, "How to Play / Official Rules", color.RGBA{60, 75, 105, 255})
	app.drawButton(screen, chrome.btnX, chrome.startY+chrome.step*4, chrome.btnW, chrome.btnH, "Quit Game", color.RGBA{140, 50, 45, 255})

	// Footer
	foot := "Version 1.1 • Built with Go & Ebitengine"
	footFace := GetFace(11, FontRegular)
	fw, _ := text.Measure(foot, footFace, 0)
	footOp := &text.DrawOptions{}
	footOp.GeoM.Translate((w-fw)/2, h-26)
	footOp.ColorScale.ScaleWithColor(color.RGBA{100, 115, 140, 255})
	text.Draw(screen, foot, footFace, footOp)

	if app.showRulesModal {
		vp := Viewport{ScreenW: w, ScreenH: h}
		app.drawRulesModal(screen, vp)
	}
}

// DrawGame renders the battlefield and HUD.
func (app *GUIApp) DrawGame(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 20, G: 24, B: 32, A: 255})

	vp := app.calcViewport()

	// 1. Top Header Bar
	app.drawHeader(screen, vp)

	// 2. Tabletop Battlefield
	app.drawBoard(screen, vp)

	// 3. Terrain Objects
	app.drawTerrain(screen, vp)

	// 4. Movement & Targeting Overlays (Ground layer)
	app.drawOverlays(screen, vp)

	// 5. Units (Dice)
	app.drawUnits(screen, vp)

	// 6. Floating Tooltips & Targeting HUD Cards (Drawn on top of units)
	app.drawTooltips(screen, vp)

	// 7. Right-hand HUD / Combat Log
	app.drawSidebar(screen, vp)

	// 8. Animated Dice Roll Showcase
	app.drawRollShowcase(screen, vp)

	// 9. Action error toast (under modals)
	app.drawToast(screen, vp)

	// 10. Modals
	app.drawModals(screen, vp)
}

func (app *GUIApp) calcViewport() Viewport {
	w, h := float64(app.w), float64(app.h)
	headerH := 82.0

	// Determine table size to fit vertically
	availH := h - headerH - 24.0
	tablePx := availH
	if tablePx < 300 {
		tablePx = 300
	}

	// Ensure sidebar gets plenty of dedicated space (at least 420px, up to 480px on 1200w screen)
	minSidebarW := 420.0
	if w-(tablePx+32.0) < minSidebarW {
		tablePx = w - minSidebarW - 32.0
	}
	if tablePx < 300 {
		tablePx = 300
	}

	tableX := 15.0
	tableY := headerH + 12.0
	scale := tablePx / TableWidthInches

	sbX := tableX + tablePx + 16.0
	sbY := 8.0
	sbW := w - sbX - 12.0
	sbH := h - 16.0

	return Viewport{
		ScreenW: w,
		ScreenH: h,
		TableX:  tableX,
		TableY:  tableY,
		TablePx: tablePx,
		Scale:   scale,
		SbX:     sbX,
		SbY:     sbY,
		SbW:     sbW,
		SbH:     sbH,
	}
}

func (app *GUIApp) drawHeader(screen *ebiten.Image, vp Viewport) {
	headerW := vp.TablePx

	// Header background panel aligned with tabletop
	vector.DrawFilledRect(screen, float32(vp.TableX), 8, float32(headerW), 78, color.RGBA{26, 32, 44, 255}, false)
	vector.StrokeRect(screen, float32(vp.TableX), 8, float32(headerW), 78, 1.5, color.RGBA{45, 55, 75, 255}, false)

	// 1. Draw Header Logo with STRICT DYNAMIC NATURAL ASPECT RATIO
	logoH := 64.0
	var logoW float64 = 220.0
	if app.headerImg != nil {
		bw := float64(app.headerImg.Bounds().Dx())
		bh := float64(app.headerImg.Bounds().Dy())
		aspect := bw / bh
		logoW = logoH * aspect
		if logoW > 260.0 {
			logoW = 260.0
			logoH = logoW / aspect
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(logoW/bw, logoH/bh)
		op.GeoM.Translate(vp.TableX+12.0, 8+(78-logoH)/2)
		screen.DrawImage(app.headerImg, op)
	} else {
		textOp := &text.DrawOptions{}
		textOp.GeoM.Translate(vp.TableX+16.0, 26)
		textOp.ColorScale.ScaleWithColor(color.RGBA{240, 215, 120, 255})
		text.Draw(screen, "SKIRMISH DICE", GetFace(22, FontBold), textOp)
	}

	// 2. Round & Turn Phase Status Pill (Center)
	if app.ge != nil {
		statusText := fmt.Sprintf("ROUND %d  •  TURN %d/4", app.ge.RoundNumber, app.ge.TurnNumber)
		if app.ge.Phase == PhaseDeployPlace {
			statusText = "DEPLOYMENT: Place Warriors"
		} else if app.ge.Phase == PhaseMorale {
			statusText = "MORALE TEST PHASE"
		} else if app.ge.Phase == PhaseRecovery {
			statusText = "CASUALTY RECOVERY PHASE"
		} else if app.ge.Phase == PhaseGameOver {
			statusText = "BATTLE CONCLUDED"
		}

		infoX := vp.TableX + 12.0 + logoW + 16.0
		pBannerX := vp.TableX + headerW - 195.0

		textOp3 := &text.DrawOptions{}
		textOp3.GeoM.Translate(infoX, 18)
		textOp3.ColorScale.ScaleWithColor(color.RGBA{130, 225, 185, 255})
		text.Draw(screen, statusText, GetFace(15, FontBold), textOp3)

		// Contextual Action Instructions (Wrapped cleanly before Turn/AP banner)
		subInst := "Click unit to select (or TAB). Move or Shoot."
		if app.ge.Phase == PhaseDeployPlace {
			curP := app.ge.Players[app.ge.DeployTurn]
			subInst = fmt.Sprintf("%s: Click zone to place die (or Auto-Deploy)", curP.Name)
		} else if app.ge.SelectedUnitID != 0 {
			sel := app.ge.GetUnit(app.ge.SelectedUnitID)
			if sel != nil {
				hovEnemy, hovLOS, _ := app.getHoveredEnemy(vp, sel)
				if hovEnemy != nil {
					if hovLOS {
						subInst = fmt.Sprintf("AIMING: %s | Click or [S] Shoot (1 AP)", hovEnemy.DisplayName())
					} else {
						subInst = fmt.Sprintf("BLOCKED: %s | Wall blocks Line of Sight", hovEnemy.DisplayName())
					}
				} else if sel.IsShooter && !sel.HasShot {
					subInst = fmt.Sprintf("Selected: %s | [Click] Move/Shoot (1 AP) • [Right-Click/C] Charge", sel.DisplayName())
				} else {
					subInst = fmt.Sprintf("Selected: %s | Click ground to Move • Click enemy to CHARGE (2d vs 1d)", sel.DisplayName())
				}
			}
		}

		maxSubW := pBannerX - infoX - 12.0
		if maxSubW > 80.0 {
			subFace := GetFace(11, FontRegular)
			subLines := wrapLogText(subInst, subFace, maxSubW)
			subY := 42.0
			for i, l := range subLines {
				if i >= 2 {
					break
				}
				instOp := &text.DrawOptions{}
				instOp.GeoM.Translate(infoX, subY)
				instOp.ColorScale.ScaleWithColor(color.RGBA{185, 205, 225, 255})
				text.Draw(screen, l, subFace, instOp)
				subY += 15.0
			}
		}

		// 3. Active Player & AP Pool Card (Right)
		activeP := app.ge.Players[app.ge.ActivePlayerIdx]
		vector.DrawFilledRect(screen, float32(pBannerX), 12, 185, 70, color.RGBA{15, 18, 26, 230}, false)
		activeCol := color.RGBA{activeP.Color.R, activeP.Color.G, activeP.Color.B, 255}
		vector.StrokeRect(screen, float32(pBannerX), 12, 185, 70, 2, activeCol, false)

		pTextOp := &text.DrawOptions{}
		pTextOp.GeoM.Translate(pBannerX+10, 20)
		pTextOp.ColorScale.ScaleWithColor(activeCol)
		text.Draw(screen, fmt.Sprintf("Turn: %s", activeP.Name), GetFace(13, FontBold), pTextOp)

		apTextOp := &text.DrawOptions{}
		apTextOp.GeoM.Translate(pBannerX+10, 46)
		apCol := color.RGBA{255, 225, 80, 255}
		apText := fmt.Sprintf("Pool: %d AP", app.ge.ActivationPoints)
		if app.ge.Phase == PhaseDeployPlace {
			apText = "Deploying..."
			apCol = color.RGBA{180, 190, 210, 255}
		}
		apTextOp.ColorScale.ScaleWithColor(apCol)
		text.Draw(screen, apText, GetFace(14, FontBold), apTextOp)
	}
}

func (app *GUIApp) drawBoard(screen *ebiten.Image, vp Viewport) {
	if img := app.ensureTableImage(vp.TablePx); img != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(vp.TableX-tableBorderPad, vp.TableY-tableBorderPad)
		screen.DrawImage(img, op)
	}

	// 6" Deployment Zones
	depH := vp.InchToPx(DeployZoneInches)

	// North Zone (Top)
	northCol := color.RGBA{210, 50, 50, 40}
	if app.ge != nil && app.ge.Phase == PhaseDeployPlace && app.ge.Players[app.ge.DeployTurn].TableEdge == 0 {
		northCol = color.RGBA{255, 60, 60, 85}
	}
	vector.DrawFilledRect(screen, float32(vp.TableX), float32(vp.TableY), float32(vp.TablePx), float32(depH), northCol, false)
	vector.StrokeLine(screen, float32(vp.TableX), float32(vp.TableY+depH), float32(vp.TableX+vp.TablePx), float32(vp.TableY+depH), 2, color.RGBA{220, 60, 60, 180}, true)

	// South Zone (Bottom)
	southCol := color.RGBA{45, 100, 220, 40}
	if app.ge != nil && app.ge.Phase == PhaseDeployPlace && app.ge.Players[app.ge.DeployTurn].TableEdge == 1 {
		southCol = color.RGBA{60, 130, 255, 85}
	}
	vector.DrawFilledRect(screen, float32(vp.TableX), float32(vp.TableY+vp.TablePx-depH), float32(vp.TablePx), float32(depH), southCol, false)
	vector.StrokeLine(screen, float32(vp.TableX), float32(vp.TableY+vp.TablePx-depH), float32(vp.TableX+vp.TablePx), float32(vp.TableY+vp.TablePx-depH), 2, color.RGBA{60, 130, 240, 180}, true)
}

func (app *GUIApp) drawTerrain(screen *ebiten.Image, vp Viewport) {
	if app.ge == nil || app.ge.Board == nil {
		return
	}

	for _, t := range app.ge.Board.Terrains {
		sx, sy := vp.WorldToScreen(t.X, t.Y)
		if t.IsCircle {
			r := float32(vp.InchToPx(t.Radius))
			// Drop shadow
			vector.DrawFilledCircle(screen, float32(sx)+3, float32(sy)+4, r, color.RGBA{20, 25, 20, 140}, true)
			// Rock / Boulder
			vector.DrawFilledCircle(screen, float32(sx), float32(sy), r, color.RGBA{100, 100, 112, 255}, true)
			vector.StrokeCircle(screen, float32(sx), float32(sy), r, 2, color.RGBA{65, 65, 75, 255}, true)
		} else {
			tw := float32(vp.InchToPx(t.Width))
			th := float32(vp.InchToPx(t.Height))
			rx := float32(sx) - tw/2
			ry := float32(sy) - th/2

			// Drop shadow
			vector.DrawFilledRect(screen, rx+3, ry+4, tw, th, color.RGBA{20, 25, 20, 140}, false)

			switch t.Type {
			case TerrainHardCover:
				// Stone Wall / Ruin with top bevel
				vector.DrawFilledRect(screen, rx, ry, tw, th, color.RGBA{125, 125, 138, 255}, false)
				vector.DrawFilledRect(screen, rx, ry, tw, 3, color.RGBA{160, 160, 175, 255}, false)
				vector.StrokeRect(screen, rx, ry, tw, th, 2, color.RGBA{70, 70, 80, 255}, false)
			case TerrainObstacleBlock:
				// Solid Pillar
				vector.DrawFilledRect(screen, rx, ry, tw, th, color.RGBA{78, 78, 88, 255}, false)
				vector.StrokeRect(screen, rx, ry, tw, th, 2, color.RGBA{45, 45, 52, 255}, false)
			case TerrainSoftCover:
				// Bush / Hedge
				vector.DrawFilledRect(screen, rx, ry, tw, th, color.RGBA{38, 130, 48, 210}, false)
				vector.StrokeRect(screen, rx, ry, tw, th, 1.5, color.RGBA{25, 95, 34, 230}, false)
			}
		}
	}
}

func (app *GUIApp) getHoveredUnit(vp Viewport) *Unit {
	if app.ge == nil || app.ge.Phase == PhaseDeployPlace {
		return nil
	}
	mx, my := ebiten.CursorPosition()
	wx, wy := vp.ScreenToWorld(float64(mx), float64(my))
	if wx < 0 || wx > TableWidthInches || wy < 0 || wy > TableHeightInches {
		return nil
	}
	var closest *Unit
	bestDist := 999.0
	for _, u := range app.ge.AllAliveUnits() {
		if u.X <= 0 && u.Y <= 0 {
			continue
		}
		d := Distance(wx, wy, u.X, u.Y)
		if d <= (u.Info().Radius+0.7) && d < bestDist {
			bestDist = d
			closest = u
		}
	}
	return closest
}

func (app *GUIApp) getHoveredEnemy(vp Viewport, shooter *Unit) (*Unit, bool, int) {
	if shooter == nil || !shooter.IsShooter || shooter.HasShot || app.ge.ActivationPoints < 1 {
		return nil, false, 0
	}
	hov := app.getHoveredUnit(vp)
	if hov != nil && hov.Player != shooter.Player && !hov.IsCasualty {
		hasLOS, cover := app.sightFor(shooter, hov)
		return hov, hasLOS, cover
	}
	return nil, false, 0
}

func (app *GUIApp) drawOverlays(screen *ebiten.Image, vp Viewport) {
	if app.ge == nil {
		return
	}

	mx, my := ebiten.CursorPosition()
	wx, wy := vp.ScreenToWorld(float64(mx), float64(my))

	// 1. Ghost Die Preview during Deployment
	if app.ge.Phase == PhaseDeployPlace {
		pIdx := app.ge.DeployTurn
		var queue []*Unit
		if pIdx == 0 {
			queue = app.ge.DeployRemaining0
		} else {
			queue = app.ge.DeployRemaining1
		}

		if len(queue) > 0 {
			nextUnit := queue[0]
			p := app.ge.Players[pIdx]
			col := color.RGBA{p.Color.R, p.Color.G, p.Color.B, 180}

			inZone := false
			if p.TableEdge == 0 && wy >= 0 && wy <= DeployZoneInches {
				inZone = true
			} else if p.TableEdge == 1 && wy >= (TableHeightInches-DeployZoneInches) && wy <= TableHeightInches {
				inZone = true
			}

			haloCol := color.RGBA{240, 60, 60, 200}
			if inZone {
				haloCol = color.RGBA{60, 240, 100, 220}
			}

			radius := float32(vp.InchToPx(nextUnit.Info().Radius))
			vector.StrokeCircle(screen, float32(mx), float32(my), radius+6, 2, haloCol, true)
			app.drawPolyhedralDie(screen, float32(mx), float32(my), radius, nextUnit.Type, col, false)
		}
		return
	}

	// 2. Battle Overlays
	if app.ge.SelectedUnitID == 0 {
		return
	}

	u := app.ge.GetUnit(app.ge.SelectedUnitID)
	if u == nil {
		return
	}

	ux, uy := vp.WorldToScreen(u.X, u.Y)
	app.ensureActionCache()

	// Movement Rings: 6", 12", 18" (Anti-aliased!)
	movesLeft := MaxMovementSteps - u.MovesMade
	if movesLeft > 0 && !u.HasShot {
		for step := 1; step <= movesLeft; step++ {
			radius := vp.InchToPx(float64(step) * StepInches)
			ringCol := color.RGBA{100, 225, 100, 150}
			if step == 2 {
				ringCol = color.RGBA{245, 215, 60, 150}
			} else if step == 3 {
				ringCol = color.RGBA{245, 125, 50, 150}
			}
			vector.StrokeCircle(screen, float32(ux), float32(uy), float32(radius), 1.6, ringCol, true)
		}
	}

	hovUnit := app.getHoveredUnit(vp)
	hovEnemy, _, _ := app.getHoveredEnemy(vp, u)

	// Highlight any hovered unit subtly
	if hovUnit != nil && hovUnit.ID != u.ID {
		hx, hy := vp.WorldToScreen(hovUnit.X, hovUnit.Y)
		hr := float32(vp.InchToPx(hovUnit.Info().Radius) + 6)
		vector.StrokeCircle(screen, float32(hx), float32(hy), hr, 1.8, color.RGBA{255, 255, 255, 190}, true)
	}

	// Mouse Cursor Measurement & Move Line (ONLY when hovering open ground)
	if hovUnit == nil && wx >= 0 && wx <= TableWidthInches && wy >= 0 && wy <= TableHeightInches && movesLeft > 0 && !u.HasShot {
		dist := Distance(u.X, u.Y, wx, wy)
		crosses := app.ge.Board.PathCrossesObstacle(u.X, u.Y, wx, wy)
		cost, _, err := CalculateMoveCost(dist, u.MovesMade, crosses)

		lineCol := color.RGBA{255, 255, 255, 190}
		if err != nil || cost > app.ge.ActivationPoints {
			lineCol = color.RGBA{240, 60, 60, 210}
		}

		vector.StrokeLine(screen, float32(ux), float32(uy), float32(mx), float32(my), 2, lineCol, true)
	}

	// 3. Line of Sight & Shooting Target Reticles (When Shooter is Selected)
	if u.IsShooter && !u.HasShot && app.ge.ActivationPoints >= 1 {
		for _, enemy := range app.ge.GetEnemyUnits(u.Player) {
			if !enemy.IsCasualty {
				ex, ey := vp.WorldToScreen(enemy.X, enemy.Y)
				hasLOS := app.actionCache.los[enemy.ID].hasLOS
				isHovered := (hovEnemy != nil && hovEnemy.ID == enemy.ID)

				if hasLOS {
					eRadius := float32(vp.InchToPx(enemy.Info().Radius) + 7)
					if isHovered {
						// Bright targeting reticle with crosshairs for hovered target
						reticleCol := color.RGBA{255, 235, 50, 255}
						vector.StrokeCircle(screen, float32(ex), float32(ey), eRadius, 2.5, reticleCol, true)
						vector.StrokeLine(screen, float32(ex)-eRadius-5, float32(ey), float32(ex)+eRadius+5, float32(ey), 1.5, reticleCol, true)
						vector.StrokeLine(screen, float32(ex), float32(ey)-eRadius-5, float32(ex), float32(ey)+eRadius+5, 1.5, reticleCol, true)

						// Vibrant targeting laser beam to hovered target
						vector.StrokeLine(screen, float32(ux), float32(uy), float32(ex), float32(ey), 4, color.RGBA{255, 230, 80, 80}, true)
						vector.StrokeLine(screen, float32(ux), float32(uy), float32(ex), float32(ey), 2, color.RGBA{255, 240, 100, 240}, true)
					} else {
						// Subtle unobtrusive reticle for other valid targets in LOS
						vector.StrokeCircle(screen, float32(ex), float32(ey), eRadius-1, 1.5, color.RGBA{225, 75, 75, 140}, true)
					}
				} else if isHovered {
					// Hovering an enemy that is blocked by an obstacle
					eRadius := float32(vp.InchToPx(enemy.Info().Radius) + 7)
					vector.StrokeCircle(screen, float32(ex), float32(ey), eRadius, 2, color.RGBA{235, 60, 60, 200}, true)
					vector.StrokeLine(screen, float32(ux), float32(uy), float32(ex), float32(ey), 1.5, color.RGBA{220, 50, 50, 160}, true)
				}
			}
		}
	}

	// 4. Melee Charge Rings & Trajectory Line (Any unit with movement remaining)
	if !u.HasShot && movesLeft > 0 && app.ge.ActivationPoints >= 1 {
		for _, enemy := range app.ge.GetEnemyUnits(u.Player) {
			if enemy.IsCasualty {
				continue
			}
			ch, ok := app.actionCache.charges[enemy.ID]
			if ok && ch.ok {
				ex, ey := vp.WorldToScreen(enemy.X, enemy.Y)
				cRadius := float32(vp.InchToPx(enemy.Info().Radius) + 10)
				isHov := (hovUnit != nil && hovUnit.ID == enemy.ID)
				if isHov {
					vector.StrokeCircle(screen, float32(ex), float32(ey), cRadius, 2.5, color.RGBA{255, 150, 40, 255}, true)
					cx, cy := vp.WorldToScreen(ch.destX, ch.destY)
					vector.StrokeLine(screen, float32(ux), float32(uy), float32(cx), float32(cy), 3, color.RGBA{255, 150, 40, 235}, true)
					vector.StrokeCircle(screen, float32(cx), float32(cy), float32(vp.InchToPx(u.Info().Radius)), 1.8, color.RGBA{255, 180, 70, 200}, true)
				} else {
					vector.StrokeCircle(screen, float32(ex), float32(ey), cRadius-1, 1.5, color.RGBA{255, 145, 45, 155}, true)
				}
			}
		}
	}
}

func (app *GUIApp) drawTooltips(screen *ebiten.Image, vp Viewport) {
	if app.ge == nil {
		return
	}

	mx, my := ebiten.CursorPosition()
	wx, wy := vp.ScreenToWorld(float64(mx), float64(my))

	// 1. Deployment Tooltip Card
	if app.ge.Phase == PhaseDeployPlace {
		pIdx := app.ge.DeployTurn
		var queue []*Unit
		if pIdx == 0 {
			queue = app.ge.DeployRemaining0
		} else {
			queue = app.ge.DeployRemaining1
		}

		if len(queue) > 0 {
			nextUnit := queue[0]
			p := app.ge.Players[pIdx]

			inZone := false
			if p.TableEdge == 0 && wy >= 0 && wy <= DeployZoneInches {
				inZone = true
			} else if p.TableEdge == 1 && wy >= (TableHeightInches-DeployZoneInches) && wy <= TableHeightInches {
				inZone = true
			}

			haloCol := color.RGBA{240, 70, 70, 220}
			if inZone {
				haloCol = color.RGBA{60, 240, 110, 230}
			}

			tipText := fmt.Sprintf("Placing: %s (%s)", nextUnit.DisplayName(), p.Name)
			f := GetFace(12, FontBold)
			tw, _ := text.Measure(tipText, f, 0)
			bw, bh := tw+22.0, 28.0
			bx := float64(mx) + 16.0
			by := float64(my) - 14.0
			if bx+bw > vp.TableX+vp.TablePx-6 {
				bx = float64(mx) - bw - 14.0
			}

			vector.DrawFilledRect(screen, float32(bx)+2, float32(by)+2, float32(bw), float32(bh), color.RGBA{0, 0, 0, 170}, false)
			vector.DrawFilledRect(screen, float32(bx), float32(by), float32(bw), float32(bh), color.RGBA{14, 18, 26, 245}, false)
			vector.StrokeRect(screen, float32(bx), float32(by), float32(bw), float32(bh), 1.5, haloCol, false)

			tipOp := &text.DrawOptions{}
			tipOp.GeoM.Translate(bx+11, by+6)
			tipOp.ColorScale.ScaleWithColor(haloCol)
			text.Draw(screen, tipText, f, tipOp)
		}
		return
	}

	// 2. Battle Floating Tooltips & Unit Hover Inspector
	var selected *Unit
	if app.ge.SelectedUnitID != 0 {
		selected = app.ge.GetUnit(app.ge.SelectedUnitID)
	}

	// Priority A: Hovering ANY unit on the board shows the rich Unit Hover Inspector Card
	// (including embedded shooting/charge combat matchup banners when targeting an enemy!)
	hovUnit := app.getHoveredUnit(vp)
	if hovUnit != nil {
		app.drawUnitInfoPanel(screen, vp, hovUnit, selected)
		return
	}

	// Priority B: Hovering Open Ground while a unit is selected -> Movement Cost Pill
	if selected == nil {
		return
	}

	movesLeft := MaxMovementSteps - selected.MovesMade
	if wx >= 0 && wx <= TableWidthInches && wy >= 0 && wy <= TableHeightInches && movesLeft > 0 && !selected.HasShot {
		dist := Distance(selected.X, selected.Y, wx, wy)
		crosses := app.ge.Board.PathCrossesObstacle(selected.X, selected.Y, wx, wy)
		cost, _, err := CalculateMoveCost(dist, selected.MovesMade, crosses)

		lineCol := color.RGBA{255, 255, 255, 240}
		borderCol := color.RGBA{100, 225, 100, 200}
		if err != nil || cost > app.ge.ActivationPoints {
			lineCol = color.RGBA{245, 80, 80, 240}
			borderCol = color.RGBA{240, 70, 70, 200}
		}

		costDesc := fmt.Sprintf("%.1f\" • %d AP", dist, cost)
		if crosses {
			costDesc += " (+1 Obstacle)"
		}
		if err != nil {
			costDesc = "Exceeds 18\" max limit"
		}

		bFace := GetFace(11, FontBold)
		bw, _ := text.Measure(costDesc, bFace, 0)
		badgeW := bw + 18.0
		badgeH := 24.0

		bx := float64(mx) + 14.0
		by := float64(my) - 12.0
		if bx+badgeW > vp.TableX+vp.TablePx-8 {
			bx = float64(mx) - badgeW - 14.0
		}

		vector.DrawFilledRect(screen, float32(bx)+2, float32(by)+2, float32(badgeW), float32(badgeH), color.RGBA{0, 0, 0, 160}, false)
		vector.DrawFilledRect(screen, float32(bx), float32(by), float32(badgeW), float32(badgeH), color.RGBA{14, 18, 26, 235}, false)
		vector.StrokeRect(screen, float32(bx), float32(by), float32(badgeW), float32(badgeH), 1, borderCol, false)

		tOp := &text.DrawOptions{}
		tOp.GeoM.Translate(bx+9, by+4)
		tOp.ColorScale.ScaleWithColor(lineCol)
		text.Draw(screen, costDesc, bFace, tOp)
	}
}

func (app *GUIApp) drawUnitInfoPanel(screen *ebiten.Image, vp Viewport, hov *Unit, selected *Unit) {
	info := hov.Info()
	p := app.ge.Players[hov.Player]

	// Faction colors
	headerFill := color.RGBA{24, 52, 98, 248}
	borderCol := color.RGBA{85, 160, 250, 255}
	factionShort := "Blue"
	if hov.Player == 1 {
		headerFill = color.RGBA{98, 28, 32, 248}
		borderCol = color.RGBA{245, 85, 85, 255}
		factionShort = "Red"
	}

	// Header info
	titleText := fmt.Sprintf("%s [%s]", info.Rank, info.Name)
	rightTag := fmt.Sprintf("%s • %d pts", factionShort, info.Cost)

	// Role & Current Status
	roleText := "Melee Warrior (2d Charge)"
	if hov.IsShooter {
		roleText = "Ranged Shooter (Brass Base)"
	}

	var statusText string
	var statusCol color.RGBA
	if hov.IsWounded {
		statusText = "WOUNDED ('1' State) - Auto-dies in melee!"
		statusCol = color.RGBA{255, 110, 95, 255}
	} else if hov.HasShot {
		statusText = "Fired this turn (Cannot move/shoot)"
		statusCol = color.RGBA{255, 165, 85, 255}
	} else if hov.MovesMade >= MaxMovementSteps {
		statusText = "3/3 Moves used (18\" max reached)"
		statusCol = color.RGBA{255, 175, 85, 255}
	} else if hov.MovesMade > 0 {
		nextCost := StepCosts[hov.MovesMade+1]
		statusText = fmt.Sprintf("Moved %d/3 steps • Next move: %d AP", hov.MovesMade, nextCost)
		statusCol = color.RGBA{90, 230, 215, 255}
	} else {
		statusText = "Ready • 0/3 moves (Next move: 1 AP)"
		statusCol = color.RGBA{105, 235, 130, 255}
	}

	// Terrain Cover & Position
	coverName, coverBonus := app.ge.Board.GetNearbyCover(hov.X, hov.Y)
	coverText := fmt.Sprintf("Terrain: %s (+%d Def)  •  Pos: (%.1f\", %.1f\")", coverName, coverBonus, hov.X, hov.Y)

	// Battle Record Stats
	stats1 := fmt.Sprintf("Kills: %d   Wounds Dealt: %d   Melee Fights: %d", hov.Kills, hov.WoundsDealt, hov.MeleeFights)
	stats2 := fmt.Sprintf("Shots: %d   Recoveries: %d     Marched: %.1f\"", hov.ShotsFired, hov.Recoveries, hov.DistanceMoved)

	// Latest Battle Highlight
	lastEv := hov.LastEvent
	if lastEv == "" {
		lastEv = "Deployed and awaiting orders."
	}
	eventText := fmt.Sprintf("Latest: %s", lastEv)

	// Optional Contextual Action / Combat Matchup Banners
	var actionBanner, chargeBanner string
	var actionBg, actionCol, chargeBg, chargeCol color.RGBA
	isAI := app.ge.Players[app.ge.ActivePlayerIdx].IsAI

	if selected != nil && selected.ID != hov.ID && app.ge.Phase == PhaseTurnAction && !isAI {
		if hov.Player != selected.Player {
			// Check melee charge eligibility
			ch, canCharge := app.chargeFor(selected, hov)
			chCost := ch.cost
			chargeMatch := fmt.Sprintf("2d%d (best) vs 1d%d", selected.Info().Sides, info.Sides)
			if hov.IsWounded {
				chargeMatch = "INSTANT KILL ON '1'!"
			}

			// Check ranged shot eligibility
			if selected.IsShooter && !selected.HasShot && app.ge.ActivationPoints >= 1 {
				hasLOS, shotCover := app.sightFor(selected, hov)
				if hasLOS {
					cDesc := "Open"
					if shotCover == 1 {
						cDesc = "+1 Soft Cover"
					} else if shotCover == 3 {
						cDesc = "+3 Hard Cover"
					}
					actionBanner = fmt.Sprintf("[Click / S] SHOOT: 1d%d vs 1d%d (%s) • 1 AP", selected.Info().Sides, info.Sides, cDesc)
					actionBg = color.RGBA{70, 58, 12, 240}
					actionCol = color.RGBA{255, 235, 85, 255}
					borderCol = color.RGBA{255, 220, 65, 255}

					if canCharge {
						chargeBanner = fmt.Sprintf("[Right-Click / C] CHARGE: %s • %d AP", chargeMatch, chCost)
						chargeBg = color.RGBA{80, 42, 12, 245}
						chargeCol = color.RGBA{255, 195, 85, 255}
					}
				} else if canCharge {
					actionBanner = fmt.Sprintf("[Click / Right-Click / C] CHARGE: %s • %d AP", chargeMatch, chCost)
					actionBg = color.RGBA{80, 42, 12, 245}
					actionCol = color.RGBA{255, 200, 85, 255}
					borderCol = color.RGBA{255, 155, 50, 255}
				} else {
					actionBanner = "LOS BLOCKED: Mid-field obstacle blocks shot"
					actionBg = color.RGBA{75, 20, 20, 240}
					actionCol = color.RGBA{255, 115, 115, 255}
					borderCol = color.RGBA{240, 70, 70, 255}
				}
			} else if canCharge {
				actionBanner = fmt.Sprintf("[Click / Right-Click / C] CHARGE: %s • %d AP", chargeMatch, chCost)
				actionBg = color.RGBA{80, 42, 12, 245}
				actionCol = color.RGBA{255, 200, 85, 255}
				borderCol = color.RGBA{255, 155, 50, 255}
			}
		} else if !hov.IsWounded {
			actionBanner = "Click unit to switch active selection"
			actionBg = color.RGBA{20, 48, 78, 235}
			actionCol = color.RGBA{145, 205, 255, 255}
		}
	} else if selected == nil && app.ge.Phase == PhaseTurnAction && !isAI && hov.Player == app.ge.ActivePlayerIdx && !hov.IsWounded {
		actionBanner = fmt.Sprintf("Click to activate %s (%s)", info.Rank, p.Name)
		actionBg = color.RGBA{20, 48, 78, 235}
		actionCol = color.RGBA{145, 205, 255, 255}
	}

	// Measure fonts to compute clean dynamic card width
	fTitle := GetFace(12, FontBold)
	fSub := GetFace(10, FontBold)
	fBody := GetFace(10, FontRegular)
	fMono := GetFace(10, FontMono)

	wTitle, _ := text.Measure(titleText+"   "+rightTag, fTitle, 0)
	wStatus, _ := text.Measure(statusText, fSub, 0)
	wCover, _ := text.Measure(coverText, fBody, 0)
	wS1, _ := text.Measure(stats1, fMono, 0)
	wEv, _ := text.Measure(eventText, fBody, 0)
	wAct, wChg := 0.0, 0.0
	if actionBanner != "" {
		wAct, _ = text.Measure(actionBanner, fSub, 0)
	}
	if chargeBanner != "" {
		wChg, _ = text.Measure(chargeBanner, fSub, 0)
	}

	cardW := math.Max(290.0, math.Max(wTitle, math.Max(wStatus, math.Max(wCover, math.Max(wS1, math.Max(wEv, math.Max(wAct, wChg))))))) + 26.0
	cardH := 136.0
	if actionBanner != "" {
		cardH += 24.0
	}
	if chargeBanner != "" {
		cardH += 23.0
	}

	// Position beside hovered unit (opposite side of board center) & clamp inside table bounds
	ux, uy := vp.WorldToScreen(hov.X, hov.Y)
	r := float64(vp.InchToPx(info.Radius))

	var cardX float64
	if ux < vp.TableX+vp.TablePx*0.5 {
		cardX = ux + r + 18.0
	} else {
		cardX = ux - r - 18.0 - cardW
	}
	minX := vp.TableX + 8.0
	maxX := vp.TableX + vp.TablePx - 8.0 - cardW
	if cardX < minX {
		cardX = minX
	}
	if cardX > maxX {
		cardX = maxX
	}

	cardY := uy - cardH/2
	minY := vp.TableY + 8.0
	maxY := vp.TableY + vp.TablePx - 8.0 - cardH
	if cardY < minY {
		cardY = minY
	}
	if cardY > maxY {
		cardY = maxY
	}

	// 1. Drop shadow & main panel body
	vector.DrawFilledRect(screen, float32(cardX)+4, float32(cardY)+5, float32(cardW), float32(cardH), color.RGBA{0, 0, 0, 195}, false)
	vector.DrawFilledRect(screen, float32(cardX), float32(cardY), float32(cardW), float32(cardH), color.RGBA{13, 17, 25, 250}, false)

	// 2. Colored Faction Header Bar
	vector.DrawFilledRect(screen, float32(cardX), float32(cardY), float32(cardW), 26, headerFill, false)
	vector.StrokeRect(screen, float32(cardX), float32(cardY), float32(cardW), float32(cardH), 1.5, borderCol, false)

	// Header Left: Unit Rank [Die]
	opTitle := &text.DrawOptions{}
	opTitle.GeoM.Translate(cardX+10, cardY+6)
	opTitle.ColorScale.ScaleWithColor(color.RGBA{255, 255, 255, 255})
	text.Draw(screen, titleText, fTitle, opTitle)

	// Header Right: Faction + Points
	rw, _ := text.Measure(rightTag, fSub, 0)
	opRight := &text.DrawOptions{}
	opRight.GeoM.Translate(cardX+cardW-rw-10, cardY+7)
	opRight.ColorScale.ScaleWithColor(color.RGBA{225, 230, 245, 235})
	text.Draw(screen, rightTag, fSub, opRight)

	// Role & Status
	curY := cardY + 32.0
	opRole := &text.DrawOptions{}
	opRole.GeoM.Translate(cardX+10, curY)
	opRole.ColorScale.ScaleWithColor(color.RGBA{210, 220, 235, 255})
	text.Draw(screen, roleText, fBody, opRole)
	curY += 16.0

	opStatus := &text.DrawOptions{}
	opStatus.GeoM.Translate(cardX+10, curY)
	opStatus.ColorScale.ScaleWithColor(statusCol)
	text.Draw(screen, statusText, fSub, opStatus)
	curY += 17.0

	// Cover & Coordinates
	opCover := &text.DrawOptions{}
	opCover.GeoM.Translate(cardX+10, curY)
	opCover.ColorScale.ScaleWithColor(color.RGBA{175, 195, 220, 255})
	text.Draw(screen, coverText, fBody, opCover)
	curY += 17.0

	// Subtle horizontal divider
	vector.StrokeLine(screen, float32(cardX+10), float32(curY), float32(cardX+cardW-10), float32(curY), 1, color.RGBA{45, 56, 78, 220}, false)
	curY += 6.0

	// Battle Stats Grid
	opS1 := &text.DrawOptions{}
	opS1.GeoM.Translate(cardX+10, curY)
	opS1.ColorScale.ScaleWithColor(color.RGBA{235, 220, 145, 255})
	text.Draw(screen, stats1, fMono, opS1)
	curY += 15.0

	opS2 := &text.DrawOptions{}
	opS2.GeoM.Translate(cardX+10, curY)
	opS2.ColorScale.ScaleWithColor(color.RGBA{190, 205, 225, 255})
	text.Draw(screen, stats2, fMono, opS2)
	curY += 16.0

	// Latest Battle Highlight
	opEv := &text.DrawOptions{}
	opEv.GeoM.Translate(cardX+10, curY)
	opEv.ColorScale.ScaleWithColor(color.RGBA{155, 210, 255, 255})
	text.Draw(screen, eventText, fBody, opEv)

	// Optional Action / Matchup Footer Strips
	if actionBanner != "" {
		stripY := cardY + cardH - 23.0
		if chargeBanner != "" {
			stripY -= 23.0
		}
		vector.DrawFilledRect(screen, float32(cardX+1), float32(stripY), float32(cardW-2), 22, actionBg, false)
		opAct := &text.DrawOptions{}
		opAct.GeoM.Translate(cardX+10, stripY+4)
		opAct.ColorScale.ScaleWithColor(actionCol)
		text.Draw(screen, actionBanner, fSub, opAct)
	}
	if chargeBanner != "" {
		stripY := cardY + cardH - 23.0
		vector.DrawFilledRect(screen, float32(cardX+1), float32(stripY), float32(cardW-2), 22, chargeBg, false)
		opChg := &text.DrawOptions{}
		opChg.GeoM.Translate(cardX+10, stripY+4)
		opChg.ColorScale.ScaleWithColor(chargeCol)
		text.Draw(screen, chargeBanner, fSub, opChg)
	}
}

func (app *GUIApp) drawUnits(screen *ebiten.Image, vp Viewport) {
	if app.ge == nil {
		return
	}

	for _, u := range app.ge.AllAliveUnits() {
		if u.X <= 0 && u.Y <= 0 {
			continue
		}

		sx, sy := vp.WorldToScreen(u.X, u.Y)
		r := float32(vp.InchToPx(u.Info().Radius))
		isSelected := app.ge.SelectedUnitID == u.ID

		pCol := color.RGBA{app.ge.Players[u.Player].Color.R, app.ge.Players[u.Player].Color.G, app.ge.Players[u.Player].Color.B, 255}

		// 1. Heavy Metal Coin Base for Shooters (milled brass ring with shadow)
		coinPad := float32(6.5)
		if u.IsShooter {
			coinR := r + coinPad
			// Drop shadow
			vector.DrawFilledCircle(screen, float32(sx)+3, float32(sy)+3.5, coinR, color.RGBA{15, 20, 20, 165}, true)
			var fillC, strokeC, innerC color.RGBA
			if u.HasShot {
				// Weathered bronze when shooter has already fired this turn
				fillC = color.RGBA{100, 82, 44, 235}
				strokeC = color.RGBA{150, 125, 70, 235}
				innerC = color.RGBA{70, 56, 30, 190}
			} else {
				// Bright milled brass base when ready to shoot
				fillC = color.RGBA{195, 150, 40, 245}
				strokeC = color.RGBA{250, 222, 95, 255}
				innerC = color.RGBA{135, 100, 25, 210}
			}
			vector.DrawFilledCircle(screen, float32(sx), float32(sy), coinR, fillC, true)
			vector.StrokeCircle(screen, float32(sx), float32(sy), coinR, 2.2, strokeC, true)
			vector.StrokeCircle(screen, float32(sx), float32(sy), coinR-3.0, 1.1, innerC, true)
		} else {
			// Subtle drop shadow under normal unit
			vector.DrawFilledCircle(screen, float32(sx)+2.5, float32(sy)+3, r+1.5, color.RGBA{15, 20, 20, 155}, true)
		}

		// 2. Selection Ring or Moved Indicator Ring
		outerRef := r
		if u.IsShooter {
			outerRef = r + coinPad
		}
		if isSelected {
			vector.StrokeCircle(screen, float32(sx), float32(sy), outerRef+4.5, 3, color.RGBA{255, 240, 80, 255}, true)
		} else if u.MovesMade > 0 {
			// Clean cyan/teal ring indicating unit has moved this turn
			vector.StrokeCircle(screen, float32(sx), float32(sy), outerRef+2.5, 1.8, color.RGBA{75, 225, 210, 195}, true)
		}

		// 3. Polyhedral Die Shape with 3D Shading & Facets
		if u.IsWounded {
			pCol = color.RGBA{250, 235, 70, 255}
		}
		app.drawPolyhedralDie(screen, float32(sx), float32(sy), r, u.Type, pCol, u.IsWounded)

		// 4. Scaled & Centered Engraved Numeral with Drop Shadow
		numeral := fmt.Sprintf("%d", u.Info().Sides)
		if u.IsWounded {
			numeral = "1"
		}
		numCol := color.RGBA{255, 255, 255, 255}
		if u.IsWounded {
			numCol = color.RGBA{210, 20, 20, 255}
		}

		numSize := math.Max(15.0, float64(r)*0.76)
		numFace := GetFace(numSize, FontBold)
		nw, nh := text.Measure(numeral, numFace, 0)
		numX := sx - nw/2
		numY := sy - nh*0.72
		if u.Type == DieD4 {
			numY = sy - nh*0.44
		}

		shadOp := &text.DrawOptions{}
		shadOp.GeoM.Translate(numX+1.2, numY+1.2)
		shadOp.ColorScale.ScaleWithColor(color.RGBA{15, 18, 25, 230})
		text.Draw(screen, numeral, numFace, shadOp)

		numOp := &text.DrawOptions{}
		numOp.GeoM.Translate(numX, numY)
		numOp.ColorScale.ScaleWithColor(numCol)
		text.Draw(screen, numeral, numFace, numOp)

		// 5. Scaled & Centered Rank Badge Tag Underneath Numeral (CMD, CPT, SOL, MERC, WND)
		rankTag := "SOL"
		switch u.Type {
		case DieD10:
			rankTag = "CMD"
		case DieD8:
			rankTag = "CPT"
		case DieD6:
			rankTag = "SOL"
		case DieD4:
			rankTag = "MERC"
		}
		tagCol := color.RGBA{230, 238, 255, 225}
		if u.IsWounded {
			rankTag = "WND"
			tagCol = color.RGBA{190, 25, 25, 255}
		}

		tagSize := math.Max(9.5, float64(r)*0.44)
		tagFace := GetFace(tagSize, FontBold)
		tw, _ := text.Measure(rankTag, tagFace, 0)
		tagX := sx - tw/2
		tagY := sy + float64(r)*0.15
		if u.Type == DieD4 {
			tagY = sy + float64(r)*0.24
		}
		tagOp := &text.DrawOptions{}
		tagOp.GeoM.Translate(tagX, tagY)
		tagOp.ColorScale.ScaleWithColor(tagCol)
		text.Draw(screen, rankTag, tagFace, tagOp)

		// 6. Movement Step Pips (Visual feedback for units that moved this turn)
		if u.MovesMade > 0 {
			pipY := float32(sy) + outerRef + 4.5
			capW := float32(28.0)
			capH := float32(10.0)
			capX := float32(sx) - capW/2
			capY := pipY - capH/2

			vector.DrawFilledRect(screen, capX, capY, capW, capH, color.RGBA{12, 16, 24, 245}, false)
			vector.StrokeRect(screen, capX, capY, capW, capH, 1.2, color.RGBA{75, 200, 195, 230}, false)

			for step := 1; step <= MaxMovementSteps; step++ {
				dotX := float32(sx) + float32(step-2)*7.5
				dotCol := color.RGBA{45, 52, 65, 255}
				if u.MovesMade >= step {
					switch step {
					case 1:
						dotCol = color.RGBA{75, 235, 105, 255} // 1st move: Green
					case 2:
						dotCol = color.RGBA{255, 215, 60, 255} // 2nd move: Gold
					case 3:
						dotCol = color.RGBA{255, 105, 50, 255} // 3rd move (Max): Orange-Red
					}
				}
				vector.DrawFilledCircle(screen, dotX, pipY, 2.6, dotCol, true)
			}
		}

		// 7. Spent Shooter Badge ("SHOT")
		if u.HasShot {
			shotW := float32(34.0)
			shotH := float32(13.0)
			shotX := float32(sx) - shotW/2
			shotY := float32(sy) - outerRef - 9.0
			vector.DrawFilledRect(screen, shotX, shotY, shotW, shotH, color.RGBA{145, 30, 30, 245}, false)
			vector.StrokeRect(screen, shotX, shotY, shotW, shotH, 1.2, color.RGBA{245, 105, 95, 245}, false)
			shotFace := GetFace(9, FontBold)
			sw, _ := text.Measure("SHOT", shotFace, 0)
			sOp := &text.DrawOptions{}
			sOp.GeoM.Translate(sx-sw/2, float64(shotY+1.0))
			sOp.ColorScale.ScaleWithColor(color.RGBA{255, 240, 240, 255})
			text.Draw(screen, "SHOT", shotFace, sOp)
		}
	}
}

// drawPolyhedralDie renders chunky, realistic 3D polyhedral dice with light and dark facets
func (app *GUIApp) drawPolyhedralDie(screen *ebiten.Image, cx, cy, r float32, dType DieType, baseCol color.RGBA, isWounded bool) {
	lightCol := adjustColor(baseCol, +35)
	midCol := baseCol
	darkCol := adjustColor(baseCol, -40)
	edgeCol := color.RGBA{25, 28, 35, 255}

	switch dType {
	case DieD4:
		// 3D Tetrahedron (3 triangular facets converging to elevated apex)
		topX, topY := cx, cy-r*1.12
		botLeftX, botLeftY := cx-r*1.06, cy+r*0.78
		botRightX, botRightY := cx+r*1.06, cy+r*0.78
		apexX, apexY := cx, cy-r*0.08

		// Left Facet (Light)
		var f1 vector.Path
		f1.MoveTo(topX, topY)
		f1.LineTo(botLeftX, botLeftY)
		f1.LineTo(apexX, apexY)
		f1.Close()
		drawPathFilled(screen, &f1, lightCol)

		// Right Facet (Dark)
		var f2 vector.Path
		f2.MoveTo(topX, topY)
		f2.LineTo(botRightX, botRightY)
		f2.LineTo(apexX, apexY)
		f2.Close()
		drawPathFilled(screen, &f2, darkCol)

		// Bottom Facet (Mid)
		var f3 vector.Path
		f3.MoveTo(botLeftX, botLeftY)
		f3.LineTo(botRightX, botRightY)
		f3.LineTo(apexX, apexY)
		f3.Close()
		drawPathFilled(screen, &f3, midCol)

		// Outer Edge Stroke
		var outer vector.Path
		outer.MoveTo(topX, topY)
		outer.LineTo(botLeftX, botLeftY)
		outer.LineTo(botRightX, botRightY)
		outer.Close()
		drawPathStroke(screen, &outer, 2.5, edgeCol)

	case DieD6:
		// 3D Rounded Cube with Chamfered Edges
		// Top Face (Light)
		vector.DrawFilledRect(screen, cx-r, cy-r, r*2, r*2, midCol, false)
		// Highlight top & left bevel
		vector.DrawFilledRect(screen, cx-r, cy-r, r*2, 3, lightCol, false)
		vector.DrawFilledRect(screen, cx-r, cy-r, 3, r*2, lightCol, false)
		// Shadow bottom & right bevel
		vector.DrawFilledRect(screen, cx-r, cy+r-3, r*2, 3, darkCol, false)
		vector.DrawFilledRect(screen, cx+r-3, cy-r, 3, r*2, darkCol, false)
		// Border
		vector.StrokeRect(screen, cx-r, cy-r, r*2, r*2, 2.5, edgeCol, false)

	case DieD8:
		// 3D Octahedron Diamond (4 facets meeting at elevated apex)
		topX, topY := cx, cy-r*1.15
		botX, botY := cx, cy+r*1.15
		leftX, leftY := cx-r*1.15, cy
		rightX, rightY := cx+r*1.15, cy
		apexX, apexY := cx, cy

		// Top-Left (Brightest)
		var f1 vector.Path
		f1.MoveTo(topX, topY)
		f1.LineTo(leftX, leftY)
		f1.LineTo(apexX, apexY)
		f1.Close()
		drawPathFilled(screen, &f1, lightCol)

		// Top-Right (Mid-Light)
		var f2 vector.Path
		f2.MoveTo(topX, topY)
		f2.LineTo(rightX, rightY)
		f2.LineTo(apexX, apexY)
		f2.Close()
		drawPathFilled(screen, &f2, midCol)

		// Bottom-Left (Mid-Dark)
		var f3 vector.Path
		f3.MoveTo(botX, botY)
		f3.LineTo(leftX, leftY)
		f3.LineTo(apexX, apexY)
		f3.Close()
		drawPathFilled(screen, &f3, midCol)

		// Bottom-Right (Darkest)
		var f4 vector.Path
		f4.MoveTo(botX, botY)
		f4.LineTo(rightX, rightY)
		f4.LineTo(apexX, apexY)
		f4.Close()
		drawPathFilled(screen, &f4, darkCol)

		// Outer Edge Stroke
		var outer vector.Path
		outer.MoveTo(topX, topY)
		outer.LineTo(rightX, rightY)
		outer.LineTo(botX, botY)
		outer.LineTo(leftX, leftY)
		outer.Close()
		drawPathStroke(screen, &outer, 2.5, edgeCol)

	case DieD10:
		// 3D Decahedron Kite Shield
		topX, topY := cx, cy-r*1.25
		upRightX, upRightY := cx+r*1.05, cy-r*0.2
		dnRightX, dnRightY := cx+r*0.65, cy+r*1.15
		dnLeftX, dnLeftY := cx-r*0.65, cy+r*1.15
		upLeftX, upLeftY := cx-r*1.05, cy-r*0.2
		apexX, apexY := cx, cy-r*0.05

		// Top Left Facet (Light)
		var f1 vector.Path
		f1.MoveTo(topX, topY)
		f1.LineTo(upLeftX, upLeftY)
		f1.LineTo(apexX, apexY)
		f1.Close()
		drawPathFilled(screen, &f1, lightCol)

		// Top Right Facet (Mid)
		var f2 vector.Path
		f2.MoveTo(topX, topY)
		f2.LineTo(upRightX, upRightY)
		f2.LineTo(apexX, apexY)
		f2.Close()
		drawPathFilled(screen, &f2, midCol)

		// Bottom Right Facet (Dark)
		var f3 vector.Path
		f3.MoveTo(upRightX, upRightY)
		f3.LineTo(dnRightX, dnRightY)
		f3.LineTo(apexX, apexY)
		f3.Close()
		drawPathFilled(screen, &f3, darkCol)

		// Bottom Left Facet (Mid-Dark)
		var f4 vector.Path
		f4.MoveTo(dnLeftX, dnLeftY)
		f4.LineTo(dnRightX, dnRightY)
		f4.LineTo(apexX, apexY)
		f4.Close()
		drawPathFilled(screen, &f4, darkCol)

		var f5 vector.Path
		f5.MoveTo(upLeftX, upLeftY)
		f5.LineTo(dnLeftX, dnLeftY)
		f5.LineTo(apexX, apexY)
		f5.Close()
		drawPathFilled(screen, &f5, midCol)

		// Outer Edge Stroke
		var outer vector.Path
		outer.MoveTo(topX, topY)
		outer.LineTo(upRightX, upRightY)
		outer.LineTo(dnRightX, dnRightY)
		outer.LineTo(dnLeftX, dnLeftY)
		outer.LineTo(upLeftX, upLeftY)
		outer.Close()
		drawPathStroke(screen, &outer, 3.0, edgeCol)
	}
}

func adjustColor(c color.RGBA, amount int) color.RGBA {
	clamp := func(v int) uint8 {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint8(v)
	}
	return color.RGBA{
		R: clamp(int(c.R) + amount),
		G: clamp(int(c.G) + amount),
		B: clamp(int(c.B) + amount),
		A: c.A,
	}
}

func drawPathFilled(screen *ebiten.Image, path *vector.Path, col color.RGBA) {
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].ColorR = float32(col.R) / 255
		vs[i].ColorG = float32(col.G) / 255
		vs[i].ColorB = float32(col.B) / 255
		vs[i].ColorA = float32(col.A) / 255
	}
	op := &ebiten.DrawTrianglesOptions{AntiAlias: true}
	screen.DrawTriangles(vs, is, emptySubImage, op)
}

func drawPathStroke(screen *ebiten.Image, path *vector.Path, width float32, col color.RGBA) {
	opStroke := &vector.StrokeOptions{
		Width:    width,
		LineJoin: vector.LineJoinRound,
		LineCap:  vector.LineCapRound,
	}
	vs, is := path.AppendVerticesAndIndicesForStroke(nil, nil, opStroke)
	for i := range vs {
		vs[i].ColorR = float32(col.R) / 255
		vs[i].ColorG = float32(col.G) / 255
		vs[i].ColorB = float32(col.B) / 255
		vs[i].ColorA = float32(col.A) / 255
	}
	op := &ebiten.DrawTrianglesOptions{AntiAlias: true}
	screen.DrawTriangles(vs, is, emptySubImage, op)
}

var emptySubImage = func() *ebiten.Image {
	img := ebiten.NewImage(3, 3)
	img.Fill(color.White)
	return img.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
}()

func (app *GUIApp) drawSidebar(screen *ebiten.Image, vp Viewport) {
	sbX := float32(vp.SbX)
	sbY := float32(vp.SbY)
	sbW := float32(vp.SbW)
	sbH := float32(vp.SbH)

	// Sidebar Panel Background
	vector.DrawFilledRect(screen, sbX, sbY, sbW, sbH, color.RGBA{18, 22, 30, 240}, false)
	vector.StrokeRect(screen, sbX, sbY, sbW, sbH, 1.5, color.RGBA{45, 52, 70, 255}, false)

	if app.ge == nil {
		return
	}

	p0 := app.ge.Players[0]
	p1 := app.ge.Players[1]

	// Side-by-side Player Status Cards (with live Morale rule & outcome indicator)
	cardW := (sbW - 20 - 8) / 2
	cardH := float32(62.0)

	formatMoraleLine := func(idx, cas int) (string, color.RGBA) {
		if m := app.ge.LastMorale[idx]; m != nil {
			if m.Passed {
				return fmt.Sprintf("Morale: %d > %d (PASS) • Next >%d", m.Roll, m.Casualties, cas), color.RGBA{120, 230, 145, 255}
			}
			return fmt.Sprintf("Morale: %d <= %d (FLED!)", m.Roll, m.Casualties), color.RGBA{255, 115, 110, 255}
		}
		if cas == 0 {
			return "Morale: Need d20 > 0 (Safe)", color.RGBA{155, 185, 215, 255}
		}
		return fmt.Sprintf("Morale: Need d20 > %d (Fail=Flee)", cas), color.RGBA{255, 210, 100, 255}
	}

	// Player 0 (Blue) Status Card
	vector.DrawFilledRect(screen, sbX+10, sbY+8, cardW, cardH, color.RGBA{22, 42, 76, 220}, false)
	vector.StrokeRect(screen, sbX+10, sbY+8, cardW, cardH, 1.5, color.RGBA{65, 135, 235, 255}, false)
	opP0 := &text.DrawOptions{}
	opP0.GeoM.Translate(float64(sbX+16), float64(sbY+12))
	opP0.ColorScale.ScaleWithColor(color.RGBA{140, 190, 255, 255})
	text.Draw(screen, p0.Name, GetFace(12, FontBold), opP0)

	opP0Stats := &text.DrawOptions{}
	opP0Stats.GeoM.Translate(float64(sbX+16), float64(sbY+30))
	opP0Stats.ColorScale.ScaleWithColor(color.RGBA{200, 215, 240, 255})
	text.Draw(screen, fmt.Sprintf("Cas: %d | Alive: %d | Queue: %d", p0.Casualties, len(app.ge.GetAliveUnits(0)), len(app.ge.DeployRemaining0)), GetFace(10, FontRegular), opP0Stats)

	m0Str, m0Col := formatMoraleLine(0, p0.Casualties)
	opP0Mor := &text.DrawOptions{}
	opP0Mor.GeoM.Translate(float64(sbX+16), float64(sbY+46))
	opP0Mor.ColorScale.ScaleWithColor(m0Col)
	text.Draw(screen, m0Str, GetFace(9, FontBold), opP0Mor)

	// Player 1 (Red) Status Card
	p1X := sbX + 10 + cardW + 8
	vector.DrawFilledRect(screen, p1X, sbY+8, cardW, cardH, color.RGBA{76, 24, 24, 220}, false)
	vector.StrokeRect(screen, p1X, sbY+8, cardW, cardH, 1.5, color.RGBA{235, 65, 65, 255}, false)
	opP1 := &text.DrawOptions{}
	opP1.GeoM.Translate(float64(p1X+6), float64(sbY+12))
	opP1.ColorScale.ScaleWithColor(color.RGBA{255, 140, 140, 255})
	text.Draw(screen, p1.Name, GetFace(12, FontBold), opP1)

	opP1Stats := &text.DrawOptions{}
	opP1Stats.GeoM.Translate(float64(p1X+6), float64(sbY+30))
	opP1Stats.ColorScale.ScaleWithColor(color.RGBA{240, 210, 210, 255})
	text.Draw(screen, fmt.Sprintf("Cas: %d | Alive: %d | Queue: %d", p1.Casualties, len(app.ge.GetAliveUnits(1)), len(app.ge.DeployRemaining1)), GetFace(10, FontRegular), opP1Stats)

	m1Str, m1Col := formatMoraleLine(1, p1.Casualties)
	opP1Mor := &text.DrawOptions{}
	opP1Mor.GeoM.Translate(float64(p1X+6), float64(sbY+46))
	opP1Mor.ColorScale.ScaleWithColor(m1Col)
	text.Draw(screen, m1Str, GetFace(9, FontBold), opP1Mor)

	btnY := float64(sbY + 76)

	// Action Buttons (2-Column Grid)
	if app.ge.Phase == PhaseDeployPlace {
		app.drawButton(screen, float64(sbX+10), btnY, float64(sbW-20), 34, "Auto-Deploy Remaining", color.RGBA{45, 145, 95, 255})
		btnY += 40
	} else {
		// Row 1: Next Die & End Turn side-by-side
		app.drawButton(screen, float64(sbX+10), btnY, float64(cardW), 32, "Next Die (TAB)", color.RGBA{40, 100, 160, 255})
		app.drawButton(screen, float64(p1X), btnY, float64(cardW), 32, "End Turn (SPACE)", color.RGBA{180, 80, 40, 255})
		btnY += 36

		// Row 2: Action buttons ("Shoot (S)" and/or "Charge (C)")
		if app.ge.SelectedUnitID != 0 {
			sel := app.ge.GetUnit(app.ge.SelectedUnitID)
			if sel != nil {
				canShoot := sel.IsShooter && !sel.HasShot && app.ge.ActivationPoints >= 1
				canCharge := len(app.GetChargeableEnemies(sel)) > 0

				if canShoot && canCharge {
					app.drawButton(screen, float64(sbX+10), btnY, float64(cardW), 30, "Shoot (S)", color.RGBA{185, 50, 45, 255})
					app.drawButton(screen, float64(p1X), btnY, float64(cardW), 30, "Charge (C)", color.RGBA{200, 105, 30, 255})
					btnY += 34
				} else if canShoot {
					app.drawButton(screen, float64(sbX+10), btnY, float64(sbW-20), 30, "Shoot Target (S)", color.RGBA{190, 50, 45, 255})
					btnY += 34
				} else if canCharge {
					app.drawButton(screen, float64(sbX+10), btnY, float64(sbW-20), 30, "Charge Enemy (C)", color.RGBA{200, 105, 30, 255})
					btnY += 34
				}
			}
		}

		// Row 3: Rules & Main Menu side-by-side
		app.drawButton(screen, float64(sbX+10), btnY, float64(cardW), 26, "View Rules (H)", color.RGBA{45, 60, 85, 255})
		app.drawButton(screen, float64(p1X), btnY, float64(cardW), 26, "Main Menu (M)", color.RGBA{95, 45, 55, 255})
		btnY += 30
	}

	// Hotkey reminder bar
	hotkeyOp := &text.DrawOptions{}
	hotkeyOp.GeoM.Translate(float64(sbX+12), btnY)
	hotkeyOp.ColorScale.ScaleWithColor(color.RGBA{130, 150, 175, 255})
	text.Draw(screen, "Keys: [TAB] Next • [S] Shoot • [C] Charge • [SPACE] End • [M] Menu", GetFace(9, FontRegular), hotkeyOp)
	btnY += 15

	// Combat Roll Log Panel
	logHeaderOp := &text.DrawOptions{}
	logHeaderOp.GeoM.Translate(float64(sbX+15), btnY)
	logHeaderOp.ColorScale.ScaleWithColor(color.RGBA{220, 220, 140, 255})
	text.Draw(screen, "COMBAT & EVENT LOG", GetFace(12, FontBold), logHeaderOp)
	btnY += 18

	logBoxY := float32(btnY)
	logBoxH := sbH - float32(btnY) - 10.0
	if logBoxH > 50 {
		vector.DrawFilledRect(screen, sbX+10, logBoxY, sbW-20, logBoxH, color.RGBA{12, 14, 20, 255}, false)
		vector.StrokeRect(screen, sbX+10, logBoxY, sbW-20, logBoxH, 1, color.RGBA{40, 48, 65, 255}, false)

		logFace := GetFace(10, FontMono)
		maxTextWidth := float64(sbW - 20 - 14) // Available width inside log box padding
		lineHeight := 14.0
		maxLines := int((logBoxH - 12) / float32(lineHeight))
		allWrapped := app.cachedLogLines(maxTextWidth)

		startLineIdx := len(allWrapped) - maxLines
		if startLineIdx < 0 {
			startLineIdx = 0
		}

		lineY := float64(logBoxY + 6)
		for i := startLineIdx; i < len(allWrapped); i++ {
			wl := allWrapped[i]
			var col color.RGBA
			switch wl.player {
			case 0: // Player 1 (Blue)
				if wl.highlight {
					col = color.RGBA{100, 220, 255, 255} // Bright cyan-blue
				} else {
					col = color.RGBA{145, 195, 255, 255} // Soft sky blue
				}
			case 1: // Player 2 (Red)
				if wl.highlight {
					col = color.RGBA{255, 115, 110, 255} // Bright coral-red
				} else {
					col = color.RGBA{255, 165, 155, 255} // Soft salmon-red
				}
			default: // System / General (-1)
				if wl.highlight {
					col = color.RGBA{255, 225, 90, 255} // Vibrant gold
				} else {
					col = color.RGBA{185, 195, 210, 255} // Clean silver
				}
			}
			logOp := &text.DrawOptions{}
			logOp.GeoM.Translate(float64(sbX+14), lineY)
			logOp.ColorScale.ScaleWithColor(col)
			text.Draw(screen, wl.text, logFace, logOp)
			lineY += lineHeight
		}
	}
}

func wrapLogText(str string, face text.Face, maxWidth float64) []string {
	if str == "" {
		return []string{""}
	}
	words := strings.Fields(str)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	currentLine := ""

	for _, word := range words {
		testLine := word
		if currentLine != "" {
			testLine = currentLine + " " + word
		} else if len(lines) > 0 {
			testLine = "  " + word
		}
		w, _ := text.Measure(testLine, face, 0)
		if w <= maxWidth {
			currentLine = testLine
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = ""
			}
			prefix := ""
			if len(lines) > 0 {
				prefix = "  "
			}
			wordW, _ := text.Measure(prefix+word, face, 0)
			if wordW <= maxWidth {
				currentLine = prefix + word
			} else {
				// Break long unbroken word character by character if wider than maxWidth
				var chunk strings.Builder
				chunk.WriteString(prefix)
				for _, r := range word {
					testChunk := chunk.String() + string(r)
					cw, _ := text.Measure(testChunk, face, 0)
					if cw > maxWidth && chunk.Len() > len(prefix) {
						lines = append(lines, chunk.String())
						chunk.Reset()
						prefix = "  "
						chunk.WriteString(prefix)
					}
					chunk.WriteRune(r)
				}
				currentLine = chunk.String()
			}
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	return lines
}

func (app *GUIApp) drawButton(screen *ebiten.Image, x, y, w, h float64, label string, baseCol color.RGBA) {
	mx, my := ebiten.CursorPosition()
	isHover := float64(mx) >= x && float64(mx) <= x+w && float64(my) >= y && float64(my) <= y+h

	drawCol := baseCol
	if isHover {
		drawCol = color.RGBA{
			uint8(math.Min(255, float64(baseCol.R)+35)),
			uint8(math.Min(255, float64(baseCol.G)+35)),
			uint8(math.Min(255, float64(baseCol.B)+35)),
			255,
		}
	}

	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), drawCol, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 1.5, color.RGBA{220, 225, 240, 180}, false)

	face := GetFace(12, FontBold)
	_, th := text.Measure(label, face, 0)
	lblOp := &text.DrawOptions{}
	lblOp.GeoM.Translate(x+14, y+(h-th)/2)
	lblOp.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, label, face, lblOp)
}

func (app *GUIApp) drawToast(screen *ebiten.Image, vp Viewport) {
	if app.toastFrames <= 0 || app.toastText == "" {
		return
	}
	face := GetFace(13, FontBold)
	tw, th := text.Measure(app.toastText, face, 0)
	padX, padY := 16.0, 10.0
	bw := tw + padX*2
	bh := th + padY*2
	bx := vp.TableX + (vp.TablePx-bw)/2
	by := vp.TableY + vp.TablePx - bh - 18
	if bx < vp.TableX+8 {
		bx = vp.TableX + 8
	}
	vector.DrawFilledRect(screen, float32(bx)+2, float32(by)+3, float32(bw), float32(bh), color.RGBA{0, 0, 0, 170}, false)
	vector.DrawFilledRect(screen, float32(bx), float32(by), float32(bw), float32(bh), color.RGBA{48, 22, 22, 245}, false)
	vector.StrokeRect(screen, float32(bx), float32(by), float32(bw), float32(bh), 1.5, color.RGBA{240, 120, 90, 255}, false)
	op := &text.DrawOptions{}
	op.GeoM.Translate(bx+padX, by+padY)
	op.ColorScale.ScaleWithColor(color.RGBA{255, 228, 210, 255})
	text.Draw(screen, app.toastText, face, op)
}

func (app *GUIApp) drawHandoff(screen *ebiten.Image, vp Viewport) {
	vector.DrawFilledRect(screen, 0, 0, float32(vp.ScreenW), float32(vp.ScreenH), color.RGBA{12, 14, 20, 255}, false)
	if app.ge == nil {
		return
	}
	p := app.ge.Players[app.ge.ActivePlayerIdx]
	title := fmt.Sprintf("%s's turn", p.Name)
	hint := "Click when ready"
	cardW := 520.0
	cardH := 180.0
	mx := (vp.ScreenW - cardW) / 2
	my := (vp.ScreenH - cardH) / 2
	border := color.RGBA{p.Color.R, p.Color.G, p.Color.B, 255}
	vector.DrawFilledRect(screen, float32(mx), float32(my), float32(cardW), float32(cardH), color.RGBA{24, 28, 38, 255}, false)
	vector.StrokeRect(screen, float32(mx), float32(my), float32(cardW), float32(cardH), 2.5, border, false)

	titleFace := GetFace(22, FontBold)
	tw, _ := text.Measure(title, titleFace, 0)
	tOp := &text.DrawOptions{}
	tOp.GeoM.Translate(mx+(cardW-tw)/2, my+48)
	tOp.ColorScale.ScaleWithColor(border)
	text.Draw(screen, title, titleFace, tOp)

	hintFace := GetFace(16, FontRegular)
	hw, _ := text.Measure(hint, hintFace, 0)
	hOp := &text.DrawOptions{}
	hOp.GeoM.Translate(mx+(cardW-hw)/2, my+96)
	hOp.ColorScale.ScaleWithColor(color.RGBA{220, 228, 240, 255})
	text.Draw(screen, hint, hintFace, hOp)
}

func (app *GUIApp) drawModals(screen *ebiten.Image, vp Viewport) {
	if app.showRulesModal {
		app.drawRulesModal(screen, vp)
		return
	}

	if app.showExitConfirm {
		app.drawExitConfirmModal(screen, vp)
		return
	}

	if app.showHandoff {
		app.drawHandoff(screen, vp)
		return
	}

	if app.ge != nil && app.ge.Phase == PhaseGameOver && app.activeRoll == nil && len(app.ge.RollQueue) == 0 {
		app.drawGameOverModal(screen, vp)
	}
}

func (app *GUIApp) drawExitConfirmModal(screen *ebiten.Image, vp Viewport) {
	// Full-screen backdrop
	vector.DrawFilledRect(screen, 0, 0, float32(vp.ScreenW), float32(vp.ScreenH), color.RGBA{0, 0, 0, 195}, false)

	modalW := 520.0
	modalH := 230.0
	mx := (vp.ScreenW - modalW) / 2
	my := (vp.ScreenH - modalH) / 2

	// Modal background
	vector.DrawFilledRect(screen, float32(mx), float32(my), float32(modalW), float32(modalH), color.RGBA{24, 28, 38, 255}, false)
	vector.StrokeRect(screen, float32(mx), float32(my), float32(modalW), float32(modalH), 2.5, color.RGBA{225, 175, 55, 255}, false)

	// Title
	tOp := &text.DrawOptions{}
	tOp.GeoM.Translate(mx+30, my+28)
	tOp.ColorScale.ScaleWithColor(color.RGBA{255, 220, 80, 255})
	text.Draw(screen, "LEAVE CURRENT BATTLE?", GetFace(18, FontBold), tOp)

	// Body text
	mOp1 := &text.DrawOptions{}
	mOp1.GeoM.Translate(mx+30, my+68)
	mOp1.ColorScale.ScaleWithColor(color.RGBA{220, 230, 245, 255})
	text.Draw(screen, "Are you sure you want to abandon this battle and return to the", GetFace(13, FontRegular), mOp1)

	mOp2 := &text.DrawOptions{}
	mOp2.GeoM.Translate(mx+30, my+92)
	mOp2.ColorScale.ScaleWithColor(color.RGBA{220, 230, 245, 255})
	text.Draw(screen, "Main Menu? Any current battle progress will be lost.", GetFace(13, FontRegular), mOp2)

	// Action buttons: "Stay & Play (ESC)" and "Leave Battle"
	app.drawButton(screen, mx+35, my+148, 215, 42, "Stay & Play (ESC)", color.RGBA{45, 125, 85, 255})
	app.drawButton(screen, mx+270, my+148, 215, 42, "Leave Battle", color.RGBA{185, 50, 45, 255})
}

func (app *GUIApp) drawRulesModal(screen *ebiten.Image, vp Viewport) {
	vector.DrawFilledRect(screen, 0, 0, float32(vp.ScreenW), float32(vp.ScreenH), color.RGBA{0, 0, 0, 190}, false)

	modalW := 700.0
	modalH := 540.0
	mx := (vp.ScreenW - modalW) / 2
	my := (vp.ScreenH - modalH) / 2

	vector.DrawFilledRect(screen, float32(mx), float32(my), float32(modalW), float32(modalH), color.RGBA{22, 26, 36, 255}, false)
	vector.StrokeRect(screen, float32(mx), float32(my), float32(modalW), float32(modalH), 2, color.RGBA{210, 175, 60, 255}, false)

	tOp := &text.DrawOptions{}
	tOp.GeoM.Translate(mx+24, my+20)
	tOp.ColorScale.ScaleWithColor(color.RGBA{240, 215, 100, 255})
	text.Draw(screen, "SKIRMISH DICE: OFFICIAL RULES SUMMARY", GetFace(16, FontBold), tOp)

	rules := []string{
		"• WARBAND: 70 pts budget. d10 Commander (10pts, max 1), d8 Captain (8pts, max 1),",
		"  d6 Soldiers (6pts, unlim), d4 Mercenaries (4pts, unlim). Metal bases mark shooters.",
		"• INITIATIVE: Each player rolls 1d20 + Casualties. Highest total takes Turn 1.",
		"• ACTIVATION (4 Turns: A-B-A-B): Active player rolls 2d6 + Casualties for point pool.",
		"  Resolve one die fully before moving to next. Movement: 1st 6\" = 1 pt; 2nd 6\" = 3 pts",
		"  (+3, total 4); 3rd 6\" = 5 pts (+5, total 9). Obstacles higher than die cost +1 step.",
		"• SHOOTING: Costs 1 AP (Author Errata). Requires clear Line of Sight (walls block LOS!).",
		"  Attacker rolls die vs Defender die + Cover (+1 Soft, +3 Hard). High roll wounds target",
		"  (turn to '1' face). A die may move then shoot, but cannot move after shooting.",
		"• MELEE CHARGE: Moving into contact costs no extra AP. Attacker gets Advantage (rolls",
		"  two dice, picks highest) vs Defender 1 die. High roll wins; loser removed. Ties bounce",
		"  back 2\". Targets showing '1' (wounded) are automatically slain in melee!",
		"• MORALE: End of Turn 4, both roll 1d20. Must beat Casualties (roll > casualties) or flee!",
		"• CASUALTY RECOVERY: Dice on '1' roll die: Even = recovers; Odd = succumbed & dies.",
		"",
		"• CONTROLS: [Click] Move/Shoot/Charge • [Right-Click] Charge Enemy / Finish Unit",
		"  [TAB] Next Unit • [S] Shoot • [C] Charge • [SPACE] End Turn • [M] Menu",
	}

	y := my + 56
	for _, line := range rules {
		op := &text.DrawOptions{}
		op.GeoM.Translate(mx+24, y)
		op.ColorScale.ScaleWithColor(color.RGBA{210, 220, 235, 255})
		text.Draw(screen, line, GetFace(12, FontRegular), op)
		y += 23
	}

	app.drawButton(screen, mx+modalW/2-80, my+modalH-50, 160, 36, "Close Rules (ESC)", color.RGBA{180, 60, 50, 255})
}

func (app *GUIApp) drawGameOverModal(screen *ebiten.Image, vp Viewport) {
	vector.DrawFilledRect(screen, 0, 0, float32(vp.ScreenW), float32(vp.ScreenH), color.RGBA{0, 0, 0, 160}, false)

	modalW := 540.0
	modalH := 240.0
	mx := (vp.ScreenW - modalW) / 2
	my := (vp.ScreenH - modalH) / 2

	vector.DrawFilledRect(screen, float32(mx), float32(my), float32(modalW), float32(modalH), color.RGBA{20, 25, 36, 255}, false)
	vector.StrokeRect(screen, float32(mx), float32(my), float32(modalW), float32(modalH), 2.5, color.RGBA{220, 190, 60, 255}, false)

	tOp := &text.DrawOptions{}
	tOp.GeoM.Translate(mx+30, my+30)
	tOp.ColorScale.ScaleWithColor(color.RGBA{255, 220, 80, 255})
	text.Draw(screen, "BATTLE COMPLETED", GetFace(20, FontBold), tOp)

	rOp := &text.DrawOptions{}
	rOp.GeoM.Translate(mx+30, my+70)
	rOp.ColorScale.ScaleWithColor(color.RGBA{220, 230, 245, 255})
	text.Draw(screen, app.ge.GameOverReason, GetFace(14, FontRegular), rOp)

	app.drawButton(screen, mx+40, my+150, 140, 44, "Play Again (AI)", color.RGBA{40, 110, 180, 255})
	app.drawButton(screen, mx+195, my+150, 140, 44, "Pass & Play", color.RGBA{50, 140, 90, 255})
	app.drawButton(screen, mx+350, my+150, 140, 44, "Main Menu", color.RGBA{95, 45, 55, 255})
}

func (app *GUIApp) drawRollShowcase(screen *ebiten.Image, vp Viewport) {
	ev := app.activeRoll
	if ev == nil {
		return
	}

	isSpinning := app.rollFrame < 36

	// Subtle dim tint over battlefield so dice pop
	vector.DrawFilledRect(screen, float32(vp.TableX), float32(vp.TableY), float32(vp.TablePx), float32(vp.TablePx), color.RGBA{8, 10, 16, 125}, false)

	cardW := math.Min(480.0, vp.TablePx-32.0)
	cardH := 208.0
	if ev.Kind == RollVisualMorale {
		cardH = 236.0
	} else if ev.Kind == RollVisualRecovery {
		cardH = 222.0
	}
	cx := vp.TableX + (vp.TablePx-cardW)/2
	cy := vp.TableY + (vp.TablePx-cardH)*0.45

	borderCol := color.RGBA{255, 215, 70, 255}
	if !isSpinning {
		if ev.Kind == RollVisualRecovery {
			if ev.PassedA {
				borderCol = color.RGBA{75, 220, 120, 255}
			} else {
				borderCol = color.RGBA{255, 85, 85, 255}
			}
		} else if ev.Kind == RollVisualMorale && (!ev.PassedA || !ev.PassedB) {
			borderCol = color.RGBA{255, 85, 85, 255}
		} else if ev.WinnerID == 0 {
			borderCol = color.RGBA{80, 165, 255, 255}
		} else if ev.WinnerID == 1 {
			borderCol = color.RGBA{255, 95, 90, 255}
		}
	}

	// Shadow, body, glowing border
	vector.DrawFilledRect(screen, float32(cx)+5, float32(cy)+6, float32(cardW), float32(cardH), color.RGBA{0, 0, 0, 200}, false)
	vector.DrawFilledRect(screen, float32(cx), float32(cy), float32(cardW), float32(cardH), color.RGBA{15, 20, 30, 250}, false)
	vector.StrokeRect(screen, float32(cx), float32(cy), float32(cardW), float32(cardH), 2.0, borderCol, true)

	// Top header strip
	vector.DrawFilledRect(screen, float32(cx+1), float32(cy+1), float32(cardW-2), 46, color.RGBA{26, 34, 52, 245}, false)

	titleFace := GetFace(15, FontBold)
	tw, _ := text.Measure(ev.Title, titleFace, 0)
	if tw > cardW-16 {
		titleFace = GetFace(13, FontBold)
		tw, _ = text.Measure(ev.Title, titleFace, 0)
	}
	tOp := &text.DrawOptions{}
	tOp.GeoM.Translate(cx+(cardW-tw)/2, cy+8)
	tOp.ColorScale.ScaleWithColor(color.RGBA{255, 225, 95, 255})
	text.Draw(screen, ev.Title, titleFace, tOp)

	subFace := GetFace(11, FontRegular)
	sw, _ := text.Measure(ev.Subtitle, subFace, 0)
	if sw > cardW-16 {
		subFace = GetFace(10, FontRegular)
		sw, _ = text.Measure(ev.Subtitle, subFace, 0)
	}
	sOp := &text.DrawOptions{}
	sOp.GeoM.Translate(cx+(cardW-sw)/2, cy+28)
	sOp.ColorScale.ScaleWithColor(color.RGBA{185, 205, 225, 255})
	text.Draw(screen, ev.Subtitle, subFace, sOp)

	// Determine display numbers (tumbling during spin, true value when locked)
	spinTick := app.rollFrame / 3
	dispA1 := ev.RollA1
	dispA2 := ev.RollA2
	dispB1 := ev.RollB1
	wobbleY := 0.0
	if isSpinning {
		dispA1 = ((spinTick*7 + 3) % ev.SidesA) + 1
		if ev.RollA2 > 0 {
			dispA2 = ((spinTick*11 + 5) % ev.SidesA) + 1
		}
		if ev.HasVs {
			dispB1 = ((spinTick*13 + 7) % ev.SidesB) + 1
		}
		if (app.rollFrame/2)%2 == 0 {
			wobbleY = -3.0
		} else {
			wobbleY = 3.0
		}
	}

	if ev.HasVs {
		// Head-to-Head Roll (Initiative, Deploy Edge, Morale)
		leftX := cx + cardW*0.25
		rightX := cx + cardW*0.75
		dieY := cy + 98.0 + wobbleY

		// Player 1 Label
		lblFace := GetFace(11, FontBold)
		lwa, _ := text.Measure(ev.LabelA, lblFace, 0)
		laOp := &text.DrawOptions{}
		laOp.GeoM.Translate(leftX-lwa/2, cy+54)
		laOp.ColorScale.ScaleWithColor(color.RGBA{115, 185, 255, 255})
		text.Draw(screen, ev.LabelA, lblFace, laOp)

		// Player 1 Die
		blueCol := color.RGBA{45, 115, 230, 255}
		if !isSpinning {
			if ev.Kind == RollVisualMorale {
				ringCol := color.RGBA{255, 75, 75, 255}
				if ev.PassedA {
					ringCol = color.RGBA{75, 220, 120, 255}
				}
				vector.StrokeCircle(screen, float32(leftX), float32(dieY), 33, 3, ringCol, true)
			} else if ev.WinnerID == 0 {
				vector.StrokeCircle(screen, float32(leftX), float32(dieY), 33, 3, color.RGBA{255, 230, 75, 255}, true)
			}
		}
		app.drawShowcaseDie(screen, float32(leftX), float32(dieY), 26, ev.SidesA, dispA1, blueCol)

		// Player 1 Math
		var mathA string
		if ev.Kind == RollVisualMorale {
			mathA = fmt.Sprintf("Rolled %d vs > %d Cas", dispA1, ev.BonusA)
		} else if ev.BonusA > 0 {
			mathA = fmt.Sprintf("%d + %d Cas = %d", dispA1, ev.BonusA, dispA1+ev.BonusA)
		} else {
			mathA = fmt.Sprintf("Rolled: %d", dispA1)
		}
		if isSpinning {
			mathA = fmt.Sprintf("Rolling d%d...", ev.SidesA)
		}
		mFace := GetFace(11, FontBold)
		mwa, _ := text.Measure(mathA, mFace, 0)
		maOp := &text.DrawOptions{}
		maOp.GeoM.Translate(leftX-mwa/2, cy+134)
		maOp.ColorScale.ScaleWithColor(color.RGBA{220, 230, 245, 255})
		text.Draw(screen, mathA, mFace, maOp)

		// Center Divider / VS
		if ev.Kind == RollVisualMorale {
			midX := float32(cx + cardW/2)
			vector.StrokeLine(screen, midX, float32(cy+54), midX, float32(cy+cardH-50), 1.5, color.RGBA{70, 82, 105, 180}, false)
			midLbl := "d20 > Cas"
			midFace := GetFace(10, FontBold)
			mlw, _ := text.Measure(midLbl, midFace, 0)
			pillW := float32(mlw + 14)
			vector.DrawFilledRect(screen, midX-pillW/2, float32(cy+88), pillW, 20, color.RGBA{22, 28, 42, 250}, false)
			vector.StrokeRect(screen, midX-pillW/2, float32(cy+88), pillW, 20, 1, color.RGBA{95, 110, 140, 220}, true)
			mOp := &text.DrawOptions{}
			mOp.GeoM.Translate(float64(midX)-mlw/2, cy+91)
			mOp.ColorScale.ScaleWithColor(color.RGBA{240, 205, 85, 255})
			text.Draw(screen, midLbl, midFace, mOp)
		} else {
			vsFace := GetFace(18, FontBold)
			vsw, _ := text.Measure("VS", vsFace, 0)
			vsOp := &text.DrawOptions{}
			vsOp.GeoM.Translate(cx+cardW/2-vsw/2, cy+86)
			vsOp.ColorScale.ScaleWithColor(color.RGBA{240, 200, 75, 230})
			text.Draw(screen, "VS", vsFace, vsOp)
		}

		// Player 2 Label
		lwb, _ := text.Measure(ev.LabelB, lblFace, 0)
		lbOp := &text.DrawOptions{}
		lbOp.GeoM.Translate(rightX-lwb/2, cy+54)
		lbOp.ColorScale.ScaleWithColor(color.RGBA{255, 130, 125, 255})
		text.Draw(screen, ev.LabelB, lblFace, lbOp)

		// Player 2 Die
		redCol := color.RGBA{215, 55, 55, 255}
		if !isSpinning {
			if ev.Kind == RollVisualMorale {
				ringCol := color.RGBA{255, 75, 75, 255}
				if ev.PassedB {
					ringCol = color.RGBA{75, 220, 120, 255}
				}
				vector.StrokeCircle(screen, float32(rightX), float32(dieY), 33, 3, ringCol, true)
			} else if ev.WinnerID == 1 {
				vector.StrokeCircle(screen, float32(rightX), float32(dieY), 33, 3, color.RGBA{255, 230, 75, 255}, true)
			}
		}
		app.drawShowcaseDie(screen, float32(rightX), float32(dieY), 26, ev.SidesB, dispB1, redCol)

		// Player 2 Math
		var mathB string
		if ev.Kind == RollVisualMorale {
			mathB = fmt.Sprintf("Rolled %d vs > %d Cas", dispB1, ev.BonusB)
		} else if ev.BonusB > 0 {
			mathB = fmt.Sprintf("%d + %d Cas = %d", dispB1, ev.BonusB, dispB1+ev.BonusB)
		} else {
			mathB = fmt.Sprintf("Rolled: %d", dispB1)
		}
		if isSpinning {
			mathB = fmt.Sprintf("Rolling d%d...", ev.SidesB)
		}
		mwb, _ := text.Measure(mathB, mFace, 0)
		mbOp := &text.DrawOptions{}
		mbOp.GeoM.Translate(rightX-mwb/2, cy+134)
		mbOp.ColorScale.ScaleWithColor(color.RGBA{240, 220, 220, 255})
		text.Draw(screen, mathB, mFace, mbOp)

		// Morale Pass/Fail status badges
		if ev.Kind == RollVisualMorale {
			drawBadge := func(centerX float64, passed bool) {
				badgeText := "Rolling..."
				bgCol := color.RGBA{34, 42, 60, 240}
				bdCol := color.RGBA{100, 115, 145, 220}
				txCol := color.RGBA{190, 205, 225, 255}
				if !isSpinning {
					if passed {
						badgeText = "PASSED (Holds)"
						bgCol = color.RGBA{22, 68, 42, 245}
						bdCol = color.RGBA{75, 220, 120, 255}
						txCol = color.RGBA{165, 255, 195, 255}
					} else {
						badgeText = "FAILED (Flees!)"
						bgCol = color.RGBA{85, 24, 28, 245}
						bdCol = color.RGBA{255, 85, 85, 255}
						txCol = color.RGBA{255, 175, 175, 255}
					}
				}
				bw := 134.0
				bh := 22.0
				bx := centerX - bw/2
				by := cy + 156.0
				vector.DrawFilledRect(screen, float32(bx), float32(by), float32(bw), float32(bh), bgCol, false)
				vector.StrokeRect(screen, float32(bx), float32(by), float32(bw), float32(bh), 1.2, bdCol, true)
				bFace := GetFace(11, FontBold)
				btw, _ := text.Measure(badgeText, bFace, 0)
				bOp := &text.DrawOptions{}
				bOp.GeoM.Translate(bx+(bw-btw)/2, by+4)
				bOp.ColorScale.ScaleWithColor(txCol)
				text.Draw(screen, badgeText, bFace, bOp)
			}
			drawBadge(leftX, ev.PassedA)
			drawBadge(rightX, ev.PassedB)
		}
	} else if ev.Kind == RollVisualRecovery {
		// Single Wounded Unit Recovery Die
		dieX := cx + cardW*0.5
		dieY := cy + 96.0 + wobbleY

		lblFace := GetFace(11, FontBold)
		lwa, _ := text.Measure(ev.LabelA, lblFace, 0)
		laOp := &text.DrawOptions{}
		laOp.GeoM.Translate(dieX-lwa/2, cy+52)
		lblCol := color.RGBA{115, 185, 255, 255}
		dieCol := color.RGBA{45, 115, 230, 255}
		if ev.WinnerID == 1 {
			lblCol = color.RGBA{255, 130, 125, 255}
			dieCol = color.RGBA{215, 55, 55, 255}
		}
		laOp.ColorScale.ScaleWithColor(lblCol)
		text.Draw(screen, ev.LabelA, lblFace, laOp)

		if !isSpinning {
			ringCol := color.RGBA{255, 85, 85, 255}
			if ev.PassedA {
				ringCol = color.RGBA{75, 220, 120, 255}
			}
			vector.StrokeCircle(screen, float32(dieX), float32(dieY), 34, 3, ringCol, true)
		}
		app.drawShowcaseDie(screen, float32(dieX), float32(dieY), 27, ev.SidesA, dispA1, dieCol)

		statusStr := fmt.Sprintf("Rolling d%d...", ev.SidesA)
		statusCol := color.RGBA{225, 235, 245, 255}
		if !isSpinning {
			if ev.PassedA {
				statusStr = fmt.Sprintf("Rolled %d (EVEN) — Unit Recovers!", dispA1)
				statusCol = color.RGBA{150, 250, 185, 255}
			} else {
				statusStr = fmt.Sprintf("Rolled %d (ODD) — Succumbed (+1 Casualty)", dispA1)
				statusCol = color.RGBA{255, 160, 160, 255}
			}
		}
		sumFace := GetFace(12, FontBold)
		smw, _ := text.Measure(statusStr, sumFace, 0)
		smOp := &text.DrawOptions{}
		smOp.GeoM.Translate(cx+(cardW-smw)/2, cy+142)
		smOp.ColorScale.ScaleWithColor(statusCol)
		text.Draw(screen, statusStr, sumFace, smOp)
	} else {
		// Single-Player Multi-Die Roll (2d6 + Casualties Activation Pool)
		d1X := cx + cardW*0.36
		d2X := cx + cardW*0.64
		dieY := cy + 94.0 + wobbleY

		dieCol := color.RGBA{45, 115, 230, 255}
		if ev.WinnerID == 1 {
			dieCol = color.RGBA{215, 55, 55, 255}
		}
		app.drawShowcaseDie(screen, float32(d1X), float32(dieY), 24, ev.SidesA, dispA1, dieCol)
		app.drawShowcaseDie(screen, float32(d2X), float32(dieY), 24, ev.SidesA, dispA2, dieCol)

		plusFace := GetFace(20, FontBold)
		pw, _ := text.Measure("+", plusFace, 0)
		pOp := &text.DrawOptions{}
		pOp.GeoM.Translate(cx+cardW/2-pw/2, cy+82)
		pOp.ColorScale.ScaleWithColor(color.RGBA{225, 230, 240, 230})
		text.Draw(screen, "+", plusFace, pOp)

		sumStr := "Rolling 2d6..."
		if !isSpinning {
			sumStr = fmt.Sprintf("[%d] + [%d] + %d Casualties = %d AP", dispA1, dispA2, ev.BonusA, dispA1+dispA2+ev.BonusA)
		}
		sumFace := GetFace(12, FontBold)
		smw, _ := text.Measure(sumStr, sumFace, 0)
		smOp := &text.DrawOptions{}
		smOp.GeoM.Translate(cx+(cardW-smw)/2, cy+132)
		smOp.ColorScale.ScaleWithColor(color.RGBA{225, 235, 245, 255})
		text.Draw(screen, sumStr, sumFace, smOp)
	}

	// Bottom Outcome Strip
	stripY := cy + cardH - 42.0
	vector.DrawFilledRect(screen, float32(cx+1), float32(stripY), float32(cardW-2), 41, color.RGBA{20, 26, 38, 250}, false)

	outText := "Rolling..."
	outCol := color.RGBA{175, 190, 215, 255}
	if !isSpinning {
		outText = ev.Outcome
		outCol = color.RGBA{255, 230, 80, 255}
		if ev.Kind == RollVisualRecovery {
			if ev.PassedA {
				outCol = color.RGBA{130, 245, 175, 255}
			} else {
				outCol = color.RGBA{255, 140, 140, 255}
			}
		} else if ev.WinnerID == 0 {
			outCol = color.RGBA{115, 200, 255, 255}
		} else if ev.WinnerID == 1 {
			outCol = color.RGBA{255, 130, 125, 255}
		}
	}
	outFace := GetFace(13, FontBold)
	ow, _ := text.Measure(outText, outFace, 0)
	if ow > cardW-18 {
		outFace = GetFace(11, FontBold)
		ow, _ = text.Measure(outText, outFace, 0)
	}
	oOp := &text.DrawOptions{}
	oOp.GeoM.Translate(cx+(cardW-ow)/2, stripY+7)
	oOp.ColorScale.ScaleWithColor(outCol)
	text.Draw(screen, outText, outFace, oOp)

	hintFace := GetFace(9, FontRegular)
	hint := "Click or [SPACE] to continue"
	hw, _ := text.Measure(hint, hintFace, 0)
	hOp := &text.DrawOptions{}
	hOp.GeoM.Translate(cx+cardW-hw-10, stripY+26)
	hOp.ColorScale.ScaleWithColor(color.RGBA{125, 140, 165, 220})
	text.Draw(screen, hint, hintFace, hOp)
}

func (app *GUIApp) drawShowcaseDie(screen *ebiten.Image, cx, cy, r float32, sides, val int, col color.RGBA) {
	dType := DieD10
	switch sides {
	case 4:
		dType = DieD4
	case 6:
		dType = DieD6
	case 8:
		dType = DieD8
	}
	// Shadow
	vector.DrawFilledCircle(screen, cx+3, cy+4, r+1, color.RGBA{0, 0, 0, 175}, true)
	app.drawPolyhedralDie(screen, cx, cy, r, dType, col, false)

	numStr := fmt.Sprintf("%d", val)
	numFace := GetFace(20, FontBold)
	nw, nh := text.Measure(numStr, numFace, 0)
	nx := float64(cx) - nw/2
	ny := float64(cy) - nh*0.58

	shadOp := &text.DrawOptions{}
	shadOp.GeoM.Translate(nx+1.5, ny+1.5)
	shadOp.ColorScale.ScaleWithColor(color.RGBA{10, 14, 22, 230})
	text.Draw(screen, numStr, numFace, shadOp)

	numOp := &text.DrawOptions{}
	numOp.GeoM.Translate(nx, ny)
	numOp.ColorScale.ScaleWithColor(color.RGBA{255, 255, 255, 255})
	text.Draw(screen, numStr, numFace, numOp)
}
