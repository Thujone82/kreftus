package main

// Layout for the window front end, in a fixed design space of
// baseW × baseH that is scaled uniformly to fit the window.

const (
	baseW = 1100.0
	baseH = 760.0

	guiDie      = 72.0
	guiSlotGap  = 10.0
	guiColGap   = 18.0
	guiGridW    = gridCols*guiDie + (gridCols-1)*guiColGap
	guiGridH    = gridRows*guiDie + (gridRows-1)*guiSlotGap
	guiGridX    = (baseW - guiGridW) / 2
	guiTopScore = 74.0
	guiTopGrid  = 92.0
	guiMidY     = guiTopGrid + guiGridH
	guiBotGrid  = guiMidY + 34
	guiBotScore = guiBotGrid + guiGridH + 20
	guiTrayDie  = 64.0
	guiPanelX   = 34.0
	guiPanelW   = guiGridX - guiPanelX - 30
	guiSideX    = guiGridX + guiGridW + 34
	guiSideW    = baseW - guiSideX - 30

	guiLineH     = 26.0
	guiGuideTop  = 84.0
	guiGuideRoom = 23
	guiGuideW    = 820.0
	guiOverRoom  = 19
)

type rect struct{ X, Y, W, H float64 }

func (r rect) has(x, y float64) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }
func (r rect) cx() float64           { return r.X + r.W/2 }
func (r rect) cy() float64           { return r.Y + r.H/2 }

func centered(cx, y, w, h float64) rect { return rect{cx - w/2, y, w, h} }

// viewport maps the design space onto a window of w × h pixels.
type viewport struct{ scale, offX, offY float64 }

func fitViewport(w, h float64) viewport {
	s := w / baseW
	if hs := h / baseH; hs < s {
		s = hs
	}
	if s <= 0 {
		s = 1
	}
	return viewport{s, (w - baseW*s) / 2, (h - baseH*s) / 2}
}

func (v viewport) toBase(x, y float64) (float64, float64) {
	return (x - v.offX) / v.scale, (y - v.offY) / v.scale
}

func (v viewport) toScreen(r rect) rect {
	return rect{v.offX + r.X*v.scale, v.offY + r.Y*v.scale, r.W * v.scale, r.H * v.scale}
}

// ---- menu and setup screens ----

func menuLogoRect() rect { return centered(baseW/2, 18, 108, 108) }

func menuItemRect(i int) rect { return centered(baseW/2, 204+float64(i)*46, 440, 39) }

func settingsRowRect(i int) rect { return centered(baseW/2, 156+float64(i)*54, 600, 46) }

func settingsStartRect(i int) rect { return centered(baseW/2, 156+float64(i)*54+4, 260, 44) }

// settingsArrows are the ◀ ▶ buttons around a setting's value.
func settingsArrows(row rect) (left, right rect) {
	left = rect{row.X + row.W - 290, row.Y + 6, 34, 34}
	right = rect{row.X + row.W - 40, row.Y + 6, 34, 34}
	return left, right
}

func promptCard() rect { return centered(baseW/2, 236, 580, 240) }

func promptField() rect {
	c := promptCard()
	return rect{c.X + 30, c.Y + 104, c.W - 60, 46}
}

func promptButtons() (back, ok rect) {
	c := promptCard()
	return rect{c.cx() - 170, c.Y + 172, 160, 42}, rect{c.cx() + 10, c.Y + 172, 160, 42}
}

func lobbyCard() rect { return centered(baseW/2, 150, 660, 420) }

func lobbyButtons(retry bool) (back, again rect) {
	c := lobbyCard()
	if !retry {
		return centered(c.cx(), c.Y+c.H-62, 180, 42), rect{}
	}
	return rect{c.cx() - 190, c.Y + c.H - 62, 180, 42}, rect{c.cx() + 10, c.Y + c.H - 62, 180, 42}
}

func backButton() rect { return rect{24, baseH - 54, 120, 38} }

// ---- play screen ----

// playCell is slot k of column c; slot 0 sits next to the middle.
func playCell(isTop bool, c, k int) rect {
	x := guiGridX + float64(c)*(guiDie+guiColGap)
	if isTop {
		return rect{x, guiTopGrid + float64(gridRows-1-k)*(guiDie+guiSlotGap), guiDie, guiDie}
	}
	return rect{x, guiBotGrid + float64(k)*(guiDie+guiSlotGap), guiDie, guiDie}
}

// columnHit covers a column through both grids so either board can be clicked.
func columnHit(c int) rect {
	x := guiGridX + float64(c)*(guiDie+guiColGap)
	return rect{x - guiColGap/2, guiTopGrid, guiDie + guiColGap, guiBotGrid + guiGridH - guiTopGrid}
}

func columnAt(x, y float64) int {
	for c := 0; c < gridCols; c++ {
		if columnHit(c).has(x, y) {
			return c
		}
	}
	return -1
}

func trayDieRect(isTop bool, i int) rect {
	y := guiBotGrid
	if isTop {
		y = guiTopGrid + 106
	}
	return rect{guiPanelX + float64(i)*(guiTrayDie+16), y, guiTrayDie, guiTrayDie}
}

// panelNameY is the first text line of a player's panel.
func panelNameY(isTop bool) float64 {
	if isTop {
		return guiTopGrid + 10
	}
	return guiBotGrid + guiGridH - 90
}

func rowButtons(n int, y, w, h, gap float64) []rect {
	total := float64(n)*w + float64(n-1)*gap
	x := baseW/2 - total/2
	out := make([]rect, n)
	for i := range out {
		out[i] = rect{x + float64(i)*(w+gap), y, w, h}
	}
	return out
}

func playButtons(n int) []rect { return rowButtons(n, baseH-92, 150, 38, 12) }

func resultCard(lines int) rect {
	h := 250 + float64(lines)*30
	return rect{baseW/2 - 360, guiMidY + 17 - h/2, 720, h}
}

func resultButtons(card rect, n int) []rect { return rowButtons(n, card.Y+card.H-62, 210, 42, 14) }

func confirmCard(lines int) rect {
	h := 190 + float64(lines)*28
	return centered(baseW/2, baseH/2-h/2, 600, h)
}

func confirmButtons(card rect) []rect { return rowButtons(2, card.Y+card.H-64, 240, 44, 20) }

func overlayCard() rect { return centered(baseW/2, 40, 820, 640) }

func overlayClose() rect {
	c := overlayCard()
	return centered(c.cx(), c.Y+c.H-56, 160, 40)
}
