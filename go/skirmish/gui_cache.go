package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const tableBorderPad = 8.0

const toastDurationFrames = 120

type menuChrome struct {
	bannerX float64
	bannerY float64
	bannerW float64
	bannerH float64
	btnX    float64
	btnW    float64
	btnH    float64
	startY  float64
	step    float64
}

type chargeApproach struct {
	ok    bool
	destX float64
	destY float64
	cost  int
}

type losResult struct {
	hasLOS bool
	cover  int
}

type actionSnap struct {
	key     string
	charges map[int]chargeApproach
	los     map[int]losResult
}

type cachedLogLine struct {
	text      string
	player    int
	highlight bool
}

type logSnap struct {
	key   string
	lines []cachedLogLine
}

type uiRect struct {
	x, y, w, h float64
}

func (r uiRect) contains(x, y float64) bool {
	return x >= r.x && x <= r.x+r.w && y >= r.y && y <= r.y+r.h
}

func (app *GUIApp) flash(err error) {
	if err == nil {
		return
	}
	app.toastText = err.Error()
	app.toastFrames = toastDurationFrames
}

func (app *GUIApp) menuChrome() menuChrome {
	w := float64(app.w)
	by := 40.0
	bannerH := 160.0
	bannerW := 520.0
	if app.headerImg != nil {
		bw := float64(app.headerImg.Bounds().Dx())
		bh := float64(app.headerImg.Bounds().Dy())
		if bh > 0 {
			aspect := bw / bh
			bannerW = bannerH * aspect
			if bannerW > w-80.0 {
				bannerW = w - 80.0
				if aspect > 0 {
					bannerH = bannerW / aspect
				}
			}
		}
	}
	btnW := 390.0
	return menuChrome{
		bannerX: (w - bannerW) / 2,
		bannerY: by,
		bannerW: bannerW,
		bannerH: bannerH,
		btnX:    (w - btnW) / 2,
		btnW:    btnW,
		btnH:    46,
		startY:  by + bannerH + 54,
		step:    56,
	}
}

func (app *GUIApp) ensureMenuBackdrop() *ebiten.Image {
	if app.w < 1 || app.h < 1 {
		return nil
	}
	if app.menuBackdrop != nil && app.menuBackdropW == app.w && app.menuBackdropH == app.h {
		return app.menuBackdrop
	}
	if app.menuBackdrop != nil {
		app.menuBackdrop.Dispose()
	}
	img := ebiten.NewImage(app.w, app.h)
	img.Fill(color.RGBA{R: 18, G: 22, B: 30, A: 255})
	gridCol := color.RGBA{28, 35, 48, 150}
	wf := float32(app.w)
	hf := float32(app.h)
	for x := float32(0); x < wf; x += 40 {
		vector.StrokeLine(img, x, 0, x, hf, 1, gridCol, false)
	}
	for y := float32(0); y < hf; y += 40 {
		vector.StrokeLine(img, 0, y, wf, y, 1, gridCol, false)
	}
	app.menuBackdrop = img
	app.menuBackdropW = app.w
	app.menuBackdropH = app.h
	return img
}

func (app *GUIApp) ensureTableImage(tablePx float64) *ebiten.Image {
	if tablePx < 1 {
		return nil
	}
	if app.tableImg != nil && math.Abs(app.tableImgPx-tablePx) < 0.5 {
		return app.tableImg
	}
	if app.tableImg != nil {
		app.tableImg.Dispose()
	}
	app.tableImg = buildTableImage(tablePx)
	app.tableImgPx = tablePx
	return app.tableImg
}

func buildTableImage(tablePx float64) *ebiten.Image {
	felt := int(math.Ceil(tablePx))
	pad := int(tableBorderPad)
	size := felt + pad*2
	if size < 1 {
		size = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	rim := color.RGBA{60, 40, 24, 255}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dst.SetRGBA(x, y, rim)
		}
	}
	rng := rand.New(rand.NewSource(1))
	for y := pad; y < pad+felt && y < size; y++ {
		for x := pad; x < pad+felt && x < size; x++ {
			j := rng.Intn(17) - 8
			dst.SetRGBA(x, y, color.RGBA{
				R: clamp8(42 + j/2),
				G: clamp8(90 + j),
				B: clamp8(52 + j/3),
				A: 255,
			})
		}
	}
	img := ebiten.NewImageFromImage(dst)

	gridCol := color.RGBA{52, 108, 64, 160}
	y0 := float32(tableBorderPad)
	y1 := float32(tableBorderPad + tablePx)
	for i := 1; i < 6; i++ {
		p := float32(tableBorderPad + float64(i)*6.0/TableWidthInches*tablePx)
		vector.StrokeLine(img, p, y0, p, y1, 1, gridCol, true)
		vector.StrokeLine(img, y0, p, y1, p, 1, gridCol, true)
	}

	face := GetFace(11, FontBold)
	labelCol := color.RGBA{230, 242, 220, 230}
	for i := 0; i <= 6; i++ {
		inches := i * 6
		label := fmt.Sprintf("%d", inches)
		tw, th := text.Measure(label, face, 0)
		px := tableBorderPad + float64(inches)/TableWidthInches*tablePx

		lx := px - tw/2
		if lx < tableBorderPad+2 {
			lx = tableBorderPad + 2
		}
		maxX := tableBorderPad + tablePx - tw - 2
		if lx > maxX {
			lx = maxX
		}
		topOp := &text.DrawOptions{}
		topOp.GeoM.Translate(lx, tableBorderPad+3)
		topOp.ColorScale.ScaleWithColor(labelCol)
		text.Draw(img, label, face, topOp)

		ly := px - th/2
		minY := tableBorderPad + th + 4
		if ly < minY {
			ly = minY
		}
		maxY := tableBorderPad + tablePx - th - 2
		if ly > maxY {
			ly = maxY
		}
		leftOp := &text.DrawOptions{}
		leftOp.GeoM.Translate(tableBorderPad+3, ly)
		leftOp.ColorScale.ScaleWithColor(labelCol)
		text.Draw(img, label, face, leftOp)
	}
	return img
}

func clamp8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func (app *GUIApp) actionStateKey() string {
	if app.ge == nil {
		return ""
	}
	sel := app.ge.GetUnit(app.ge.SelectedUnitID)
	key := fmt.Sprintf("s=%d;ap=%d;", app.ge.SelectedUnitID, app.ge.ActivationPoints)
	if sel != nil {
		key += fmt.Sprintf("sx=%.2f;sy=%.2f;mv=%d;sh=%t;", sel.X, sel.Y, sel.MovesMade, sel.HasShot)
	}
	for _, u := range app.ge.AllAliveUnits() {
		key += fmt.Sprintf("%d:%.2f,%.2f;", u.ID, u.X, u.Y)
	}
	return key
}

func (app *GUIApp) ensureActionCache() {
	if app.ge == nil {
		app.actionCache = actionSnap{}
		return
	}
	key := app.actionStateKey()
	if app.actionCache.charges != nil && app.actionCache.key == key {
		return
	}
	charges := map[int]chargeApproach{}
	los := map[int]losResult{}
	if sel := app.ge.GetUnit(app.ge.SelectedUnitID); sel != nil {
		for _, enemy := range app.ge.GetEnemyUnits(sel.Player) {
			if enemy.IsCasualty {
				continue
			}
			destX, destY, _, cost, _, err := app.ge.FindChargeApproach(sel, enemy)
			if err == nil {
				charges[enemy.ID] = chargeApproach{ok: true, destX: destX, destY: destY, cost: cost}
			}
			hasLOS, cover := app.ge.Board.CheckLineOfSight(sel.X, sel.Y, enemy.X, enemy.Y)
			los[enemy.ID] = losResult{hasLOS: hasLOS, cover: cover}
		}
	}
	app.actionCache = actionSnap{key: key, charges: charges, los: los}
}

func (app *GUIApp) chargeFor(u, target *Unit) (chargeApproach, bool) {
	if app.ge == nil || u == nil || target == nil {
		return chargeApproach{}, false
	}
	app.ensureActionCache()
	if u.ID == app.ge.SelectedUnitID {
		ch, ok := app.actionCache.charges[target.ID]
		return ch, ok && ch.ok
	}
	destX, destY, _, cost, _, err := app.ge.FindChargeApproach(u, target)
	if err != nil {
		return chargeApproach{}, false
	}
	return chargeApproach{ok: true, destX: destX, destY: destY, cost: cost}, true
}

func (app *GUIApp) sightFor(from, to *Unit) (bool, int) {
	if app.ge == nil || app.ge.Board == nil || from == nil || to == nil {
		return false, 0
	}
	app.ensureActionCache()
	if from.ID == app.ge.SelectedUnitID {
		if los, ok := app.actionCache.los[to.ID]; ok {
			return los.hasLOS, los.cover
		}
	}
	return app.ge.Board.CheckLineOfSight(from.X, from.Y, to.X, to.Y)
}

func (app *GUIApp) cachedLogLines(maxWidth float64) []cachedLogLine {
	if app.ge == nil {
		return nil
	}
	last := ""
	if n := len(app.ge.Log); n > 0 {
		last = app.ge.Log[n-1].Text
	}
	key := fmt.Sprintf("%d|%s|%.1f", len(app.ge.Log), last, maxWidth)
	if key == app.logCache.key {
		return app.logCache.lines
	}
	face := GetFace(10, FontMono)
	lines := make([]cachedLogLine, 0, len(app.ge.Log))
	for _, entry := range app.ge.Log {
		for _, line := range wrapLogText(entry.Text, face, maxWidth) {
			lines = append(lines, cachedLogLine{
				text:      line,
				player:    entry.Player,
				highlight: entry.Highlight,
			})
		}
	}
	app.logCache = logSnap{key: key, lines: lines}
	return lines
}

func (app *GUIApp) isPassAndPlay() bool {
	return app.ge != nil && !app.ge.Players[0].IsAI && !app.ge.Players[1].IsAI
}

func (app *GUIApp) handoffKey() string {
	if app.ge == nil {
		return ""
	}
	return fmt.Sprintf("%d:%d:%d", app.ge.RoundNumber, app.ge.TurnNumber, app.ge.ActivePlayerIdx)
}

// needsHandoff is true for a pass-and-play human turn that has not been
// acknowledged, once earlier combat showcases have finished and the next
// visual is that turn's initiative or activation roll (or nothing is queued).
func (app *GUIApp) needsHandoff() bool {
	if app.screen != ScreenBattle || !app.isPassAndPlay() {
		return false
	}
	if app.ge.Phase != PhaseTurnAction {
		return false
	}
	if app.ge.Players[app.ge.ActivePlayerIdx].IsAI {
		return false
	}
	if app.handoffKey() == app.ackedHandoffKey {
		return false
	}
	if app.activeRoll != nil {
		return false
	}
	if len(app.ge.RollQueue) == 0 {
		return true
	}
	next := app.ge.RollQueue[0].Kind
	return next == RollVisualInitiative || next == RollVisualTurnAP
}

func (app *GUIApp) maybeAutoSelect() {
	if app.screen != ScreenBattle || app.ge == nil || app.ge.Phase != PhaseTurnAction {
		return
	}
	if app.ge.Players[app.ge.ActivePlayerIdx].IsAI {
		return
	}
	if app.showHandoff || app.showExitConfirm || app.showRulesModal || app.activeRoll != nil || len(app.ge.RollQueue) > 0 {
		return
	}
	key := app.handoffKey()
	if app.autoSelectKey == key {
		return
	}
	app.autoSelectKey = key
	if app.ge.SelectedUnitID != 0 || app.ge.HasSelectedActed {
		return
	}
	friendlies := app.ge.GetAliveUnits(app.ge.ActivePlayerIdx)
	var candidates []*Unit
	for _, u := range friendlies {
		if !u.IsWounded && (!u.HasShot || u.MovesMade < MaxMovementSteps) {
			candidates = append(candidates, u)
		}
	}
	if len(candidates) == 0 {
		candidates = friendlies
	}
	for _, u := range candidates {
		if err := app.ge.SelectUnit(u); err == nil {
			return
		}
	}
}

func (app *GUIApp) updateCursor() {
	shape := ebiten.CursorShapeDefault
	switch {
	case app.showHandoff || (app.activeRoll != nil && app.screen == ScreenBattle):
		shape = ebiten.CursorShapePointer
	case app.pointerOverButton():
		shape = ebiten.CursorShapePointer
	case app.crosshairHover():
		shape = ebiten.CursorShapeCrosshair
	}
	ebiten.SetCursorShape(shape)
}

func (app *GUIApp) pointerOverButton() bool {
	mx, my := ebiten.CursorPosition()
	x, y := float64(mx), float64(my)
	sw, sh := float64(app.w), float64(app.h)

	if app.showExitConfirm {
		modalW, modalH := 520.0, 230.0
		mx0 := (sw - modalW) / 2
		my0 := (sh - modalH) / 2
		return uiRect{mx0 + 35, my0 + 148, 215, 42}.contains(x, y) ||
			uiRect{mx0 + 270, my0 + 148, 215, 42}.contains(x, y)
	}
	if app.showRulesModal {
		modalW, modalH := 700.0, 540.0
		mx0 := (sw - modalW) / 2
		my0 := (sh - modalH) / 2
		return uiRect{mx0 + modalW/2 - 80, my0 + modalH - 50, 160, 36}.contains(x, y)
	}
	if app.screen == ScreenMainMenu {
		chrome := app.menuChrome()
		for i := 0; i < 5; i++ {
			if (uiRect{chrome.btnX, chrome.startY + float64(i)*chrome.step, chrome.btnW, chrome.btnH}).contains(x, y) {
				return true
			}
		}
		return false
	}
	if app.ge != nil && app.ge.Phase == PhaseGameOver && app.activeRoll == nil && len(app.ge.RollQueue) == 0 {
		modalW, modalH := 540.0, 240.0
		mx0 := (sw - modalW) / 2
		my0 := (sh - modalH) / 2
		return uiRect{mx0 + 40, my0 + 150, 140, 44}.contains(x, y) ||
			uiRect{mx0 + 195, my0 + 150, 140, 44}.contains(x, y) ||
			uiRect{mx0 + 350, my0 + 150, 140, 44}.contains(x, y)
	}
	return app.sidebarButtonAt(x, y)
}

func (app *GUIApp) sidebarButtonAt(x, y float64) bool {
	if app.screen != ScreenBattle || app.ge == nil || app.ge.Phase == PhaseGameOver {
		return false
	}
	vp := app.calcViewport()
	sbX := vp.SbX
	sbW := vp.SbW
	cardW := (sbW - 20.0 - 8.0) / 2.0
	p1X := sbX + 10.0 + cardW + 8.0
	btnY := vp.SbY + 76.0

	if app.ge.Phase == PhaseDeployPlace {
		return uiRect{sbX + 10, btnY, sbW - 20, 34}.contains(x, y)
	}

	human := !app.ge.Players[app.ge.ActivePlayerIdx].IsAI
	if human {
		if (uiRect{sbX + 10, btnY, cardW, 32}).contains(x, y) || (uiRect{p1X, btnY, cardW, 32}).contains(x, y) {
			return true
		}
	}
	btnY += 36

	if app.ge.SelectedUnitID != 0 {
		if sel := app.ge.GetUnit(app.ge.SelectedUnitID); sel != nil {
			canShoot := sel.IsShooter && !sel.HasShot && app.ge.ActivationPoints >= 1
			canCharge := len(app.GetChargeableEnemies(sel)) > 0
			if canShoot && canCharge {
				if (uiRect{sbX + 10, btnY, cardW, 30}).contains(x, y) || (uiRect{p1X, btnY, cardW, 30}).contains(x, y) {
					return true
				}
			} else if canShoot || canCharge {
				if (uiRect{sbX + 10, btnY, sbW - 20, 30}).contains(x, y) {
					return true
				}
			}
		}
	}

	// Rules and Main Menu sit under the optional action row. Match drawSidebar's advance.
	if app.ge.SelectedUnitID != 0 {
		if sel := app.ge.GetUnit(app.ge.SelectedUnitID); sel != nil {
			canShoot := sel.IsShooter && !sel.HasShot && app.ge.ActivationPoints >= 1
			canCharge := len(app.GetChargeableEnemies(sel)) > 0
			if canShoot || canCharge {
				btnY += 34
			}
		}
	}
	return (uiRect{sbX + 10, btnY, cardW, 26}).contains(x, y) || (uiRect{p1X, btnY, cardW, 26}).contains(x, y)
}

func (app *GUIApp) crosshairHover() bool {
	if app.screen != ScreenBattle || app.ge == nil {
		return false
	}
	if app.showHandoff || app.showRulesModal || app.showExitConfirm || app.activeRoll != nil {
		return false
	}
	if app.ge.Phase != PhaseTurnAction || app.ge.Players[app.ge.ActivePlayerIdx].IsAI {
		return false
	}
	u := app.ge.GetUnit(app.ge.SelectedUnitID)
	if u == nil {
		return false
	}
	mx, my := ebiten.CursorPosition()
	vp := app.calcViewport()
	wx, wy := vp.ScreenToWorld(float64(mx), float64(my))
	if wx < 0 || wy < 0 || wx > TableWidthInches || wy > TableHeightInches {
		return false
	}
	hov := app.getHoveredUnit(vp)
	if hov != nil && hov.Player != u.Player && !hov.IsCasualty {
		if _, ok := app.chargeFor(u, hov); ok {
			return true
		}
		if u.IsShooter && !u.HasShot && app.ge.ActivationPoints >= 1 {
			if hasLOS, _ := app.sightFor(u, hov); hasLOS {
				return true
			}
		}
		return false
	}
	if hov != nil {
		return false
	}
	if u.HasShot || u.MovesMade >= MaxMovementSteps {
		return false
	}
	dist := Distance(u.X, u.Y, wx, wy)
	crosses := app.ge.Board.PathCrossesObstacle(u.X, u.Y, wx, wy)
	cost, _, err := CalculateMoveCost(dist, u.MovesMade, crosses)
	return err == nil && cost <= app.ge.ActivationPoints
}
