package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"sort"
)

// ocrSudokuImage locates the puzzle grid, reads one digit per cell, and drops
// any reading that would duplicate a number in the same row, column, or box.
func ocrSudokuImage(img image.Image) (Board, int) {
	board, n, ok := ocrWithProgress(img, nil)
	if !ok {
		return Board{}, 0
	}
	return board, n
}

// fullSudokuGrid reports whether the detected lines are a complete 9x9 lattice.
// A cropped fragment has a different line count and must not be stretched into a puzzle.
func fullSudokuGrid(xs, ys []float64) bool {
	if len(xs) != 10 || len(ys) != 10 || !uniform(xs, 1.45) || !uniform(ys, 1.45) {
		return false
	}
	cellW := (xs[9] - xs[0]) / 9
	cellH := (ys[9] - ys[0]) / 9
	if cellW < 8 || cellH < 8 {
		return false
	}
	ratio := cellW / cellH
	return ratio >= 0.72 && ratio <= 1.4
}

// ocrWithProgress is ocrSudokuImage with reports in the 0..1 range so the UI
// can show how far the read has gotten. report may be nil.
func ocrWithProgress(img image.Image, report func(frac float64, stage string)) (Board, int, bool) {
	var board Board
	step := func(frac float64, stage string) {
		if report == nil {
			return
		}
		if frac < 0 {
			frac = 0
		}
		if frac > 1 {
			frac = 1
		}
		report(frac, stage)
	}
	if img == nil {
		return board, 0, false
	}
	step(0.02, "Preparing image")
	px := trimFlatMargins(toRGBA(img))
	w, h := px.Bounds().Dx(), px.Bounds().Dy()
	if w < 40 || h < 40 {
		return board, 0, false
	}

	scanned := 0.0
	tick := func() {
		scanned += 0.012
		if scanned > 0.32 {
			scanned = 0.32
		}
		step(0.06+scanned, "Locating the grid")
	}
	step(0.06, "Locating the grid")
	xs := detectAxisTick(px, false, tick)
	ys := detectAxisTick(px, true, tick)
	if !fullSudokuGrid(xs, ys) {
		step(1, "No Sudoku grid")
		return board, 0, false
	}

	step(0.42, "Reading cells")
	hits := make([]ocrHit, 0, 40)
	cell := 0
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			cell++
			if cell == 1 || cell%3 == 0 || cell == 81 {
				step(0.42+0.54*float64(cell)/81, fmt.Sprintf("Reading cell %d of 81", cell))
			}
			x0, y0, x1, y1 := insetRect(xs[c], ys[r], xs[c+1], ys[r+1])
			glyph := extractGlyph(px, x0, y0, x1, y1)
			if glyph == nil {
				continue
			}
			digit, alt, conf, altConf := classifyBits(glyph)
			if digit == 0 {
				continue
			}
			hits = append(hits, ocrHit{r: r, c: c, digit: digit, alt: alt, conf: conf, altConf: altConf})
		}
	}
	step(0.98, "Checking for duplicates")
	board, n := reconcile(hits)
	step(1, "Done")
	return board, n, true
}

type ocrHit struct {
	r, c          int
	digit, alt    int
	conf, altConf float64
}

func reconcile(hits []ocrHit) (Board, int) {
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].conf == hits[j].conf {
			return hits[i].r < hits[j].r || (hits[i].r == hits[j].r && hits[i].c < hits[j].c)
		}
		return hits[i].conf > hits[j].conf
	})
	var board Board
	for _, h := range hits {
		placed := false
		if h.digit >= 1 && h.digit <= 9 && isValid(&board, h.r, h.c, h.digit) {
			board[h.r][h.c] = h.digit
			placed = true
		}
		if !placed && h.alt >= 1 && h.alt <= 9 && h.altConf >= 0.68 && h.altConf+0.15 >= h.conf && isValid(&board, h.r, h.c, h.alt) {
			board[h.r][h.c] = h.alt
		}
	}
	count := 0
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			if board[r][c] != 0 {
				count++
			}
		}
	}
	return board, count
}

// trimFlatMargins drops a solid page margin. A real grid line is only a few
// pixels thick, so a uniform band has to be wider than that before it is cut.
func trimFlatMargins(px *image.RGBA) *image.RGBA {
	w, h := px.Bounds().Dx(), px.Bounds().Dy()
	top := flatSpan(px, true, 0, 1, h)
	bottom := flatSpan(px, true, h-1, -1, h)
	left := flatSpan(px, false, 0, 1, w)
	right := flatSpan(px, false, w-1, -1, w)
	if top+bottom >= h-40 || left+right >= w-40 {
		return px
	}
	if top == 0 && bottom == 0 && left == 0 && right == 0 {
		return px
	}
	return cropRGBA(px, left, top, w-right, h-bottom)
}

func flatSpan(px *image.RGBA, horizontal bool, start, step, limit int) int {
	n := 0
	half := limit / 2
	for i := start; n < half; i += step {
		if i < 0 || i >= limit || !stripFlat(px, horizontal, i) {
			break
		}
		n++
	}
	if n >= 8 {
		return n
	}
	return 0
}

func stripFlat(px *image.RGBA, horizontal bool, i int) bool {
	w, h := px.Bounds().Dx(), px.Bounds().Dy()
	n := w
	if !horizontal {
		n = h
	}
	if n == 0 {
		return false
	}
	minL, maxL := 255, 0
	for j := 0; j < n; j++ {
		var c color.RGBA
		if horizontal {
			c = px.RGBAAt(j, i)
		} else {
			c = px.RGBAAt(i, j)
		}
		l := lumOf(c)
		if l < minL {
			minL = l
		}
		if l > maxL {
			maxL = l
		}
	}
	return maxL-minL <= 28
}

func cropRGBA(px *image.RGBA, x0, y0, x1, y1 int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, x1-x0, y1-y0))
	draw.Draw(dst, dst.Bounds(), px, image.Point{x0, y0}, draw.Src)
	return dst
}

func toRGBA(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}

func lumOf(c color.RGBA) int {
	return (299*int(c.R) + 587*int(c.G) + 114*int(c.B)) / 1000
}

func medianLum(px *image.RGBA) int {
	w, h := px.Bounds().Dx(), px.Bounds().Dy()
	if w == 0 || h == 0 {
		return 0
	}
	stepX := w / 48
	stepY := h / 48
	if stepX < 1 {
		stepX = 1
	}
	if stepY < 1 {
		stepY = 1
	}
	vals := make([]int, 0, 64)
	for y := 0; y < h; y += stepY {
		for x := 0; x < w; x += stepX {
			vals = append(vals, lumOf(px.RGBAAt(x, y)))
		}
	}
	return medianInt(vals)
}

func dist2(c color.RGBA, r, g, b int) int {
	dr := int(c.R) - r
	dg := int(c.G) - g
	db := int(c.B) - b
	return dr*dr + dg*dg + db*db
}

// --- grid lines ---

type lineSeg struct {
	pos   float64
	thick int
	run   int
}

func detectAxis(px *image.RGBA, horizontal bool) []float64 {
	return detectAxisTick(px, horizontal, nil)
}

func detectAxisTick(px *image.RGBA, horizontal bool, tick func()) []float64 {
	light := toGrid(segmentsTick(px, horizontal, true, tick))
	dark := toGrid(segmentsTick(px, horizontal, false, tick))
	if axisQuality(light) >= axisQuality(dark) {
		return light
	}
	return dark
}

func axisQuality(lines []float64) float64 {
	if len(lines) != 10 {
		return float64(len(lines))
	}
	if !uniform(lines, 1.55) {
		return 30
	}
	return 1000 + (lines[9] - lines[0])
}

func segments(px *image.RGBA, horizontal, light bool) []lineSeg {
	return segmentsTick(px, horizontal, light, nil)
}

func segmentsTick(px *image.RGBA, horizontal, light bool, tick func()) []lineSeg {
	return filterSegs(clusterRuns(lineRuns(px, horizontal, light, tick)))
}

func lineRuns(px *image.RGBA, horizontal, light bool, tick func()) []int {
	w, h := px.Bounds().Dx(), px.Bounds().Dy()
	n := h
	length := w
	if !horizontal {
		n = w
		length = h
	}
	bg := medianLum(px)
	lightCut := bg + 22
	if lightCut < 46 {
		lightCut = 46
	}
	darkCut := bg - 26
	if darkCut > 150 {
		darkCut = 150
	}
	runs := make([]int, n)
	on := make([]bool, length)
	for i := 0; i < n; i++ {
		if tick != nil && i%40 == 0 {
			tick()
		}
		for j := 0; j < length; j++ {
			on[j] = linePixel(px, i, j, horizontal, light, lightCut, darkCut)
		}
		runs[i] = longestOn(on)
	}
	return runs
}

// linePixel is true for a thin stroke. A solid margin is the same color on
// both sides, so only its edges — and real grid lines — qualify.
func linePixel(px *image.RGBA, i, j int, horizontal, light bool, lightCut, darkCut int) bool {
	w, h := px.Bounds().Dx(), px.Bounds().Dy()
	x, y := j, i
	if !horizontal {
		x, y = i, j
	}
	if !isLineColor(px, x, y, w, h, light, lightCut, darkCut) {
		return false
	}
	const d = 4
	var a, b bool
	if horizontal {
		a = isLineColor(px, x, y-d, w, h, light, lightCut, darkCut)
		b = isLineColor(px, x, y+d, w, h, light, lightCut, darkCut)
	} else {
		a = isLineColor(px, x-d, y, w, h, light, lightCut, darkCut)
		b = isLineColor(px, x+d, y, w, h, light, lightCut, darkCut)
	}
	return !a || !b
}

func isLineColor(px *image.RGBA, x, y, w, h int, light bool, lightCut, darkCut int) bool {
	if x < 0 || y < 0 || x >= w || y >= h {
		return false
	}
	l := lumOf(px.RGBAAt(x, y))
	if light {
		return l >= lightCut
	}
	return l <= darkCut
}

// longestOn is the longest run of line pixels, allowing a 2px break so a
// slightly broken grid line still counts as one line.
func longestOn(on []bool) int {
	n := len(on)
	best := 0
	i := 0
	for i < n {
		if !on[i] {
			i++
			continue
		}
		j := i
		gap := 0
		last := i
		for j < n {
			if on[j] {
				gap = 0
				last = j
				j++
				continue
			}
			gap++
			if gap > 2 {
				break
			}
			j++
		}
		if last-i+1 > best {
			best = last - i + 1
		}
		i = last + 1
	}
	return best
}

func clusterRuns(runs []int) []lineSeg {
	maxRun := 0
	for _, r := range runs {
		if r > maxRun {
			maxRun = r
		}
	}
	if maxRun < 24 {
		return nil
	}
	n := len(runs)
	isLine := make([]bool, n)
	for i, r := range runs {
		if r >= 16 {
			isLine[i] = true
		}
	}
	idxs := make([]int, 0)
	for i, v := range isLine {
		if v {
			idxs = append(idxs, i)
		}
	}
	if len(idxs) == 0 {
		return nil
	}
	include := make([]bool, n)
	for _, i := range idxs {
		include[i] = true
	}
	for i := 0; i < len(idxs)-1; i++ {
		gap := idxs[i+1] - idxs[i] - 1
		if gap > 0 && gap <= 3 {
			for g := idxs[i] + 1; g < idxs[i+1]; g++ {
				include[g] = true
			}
		}
	}
	var segs []lineSeg
	i := 0
	for i < n {
		if !include[i] {
			i++
			continue
		}
		j := i
		run := 0
		for j < n && include[j] {
			if runs[j] > run {
				run = runs[j]
			}
			j++
		}
		segs = append(segs, lineSeg{pos: float64(i+j-1) / 2, thick: j - i, run: run})
		i = j
	}
	return segs
}

func filterSegs(segs []lineSeg) []lineSeg {
	if len(segs) == 0 {
		return nil
	}
	med := medianGapSeg(segs)
	thickLimit := 12.0
	if med > 0 {
		thickLimit = math.Max(12, med*0.40)
	}
	thin := make([]lineSeg, 0, len(segs))
	for i, s := range segs {
		if float64(s.thick) <= thickLimit {
			thin = append(thin, s)
			continue
		}
		// A page margin fused with the outer border is much thicker than a grid
		// line. Keep the edge that faces the puzzle instead of dropping the border.
		if i == 0 {
			thin = append(thin, lineSeg{pos: s.pos + float64(s.thick)/2, thick: 2, run: s.run})
		} else if i == len(segs)-1 {
			thin = append(thin, lineSeg{pos: s.pos - float64(s.thick)/2, thick: 2, run: s.run})
		}
	}
	segs = thin
	if len(segs) == 0 {
		return nil
	}
	maxRun := 0
	for _, s := range segs {
		if s.run > maxRun {
			maxRun = s.run
		}
	}
	kept := make([]lineSeg, 0, len(segs))
	for _, s := range segs {
		if float64(s.run) >= 0.62*float64(maxRun) {
			kept = append(kept, s)
		}
	}
	segs = kept
	if len(segs) == 0 {
		return nil
	}
	med = medianGapSeg(segs)
	merged := []lineSeg{segs[0]}
	for _, s := range segs[1:] {
		prev := &merged[len(merged)-1]
		if med > 0 && s.pos-prev.pos < med*0.38 {
			if s.run > prev.run {
				*prev = s
			}
			continue
		}
		merged = append(merged, s)
	}
	return merged
}

func medianGapSeg(segs []lineSeg) float64 {
	if len(segs) < 2 {
		return 0
	}
	gaps := make([]float64, 0, len(segs)-1)
	for i := 1; i < len(segs); i++ {
		gaps = append(gaps, segs[i].pos-segs[i-1].pos)
	}
	return medianFloat(gaps)
}

func medianFloat(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64{}, v...)
	sort.Float64s(s)
	return s[len(s)/2]
}

func toGrid(segs []lineSeg) []float64 {
	if len(segs) == 0 {
		return nil
	}
	pos := make([]float64, len(segs))
	for i, s := range segs {
		pos[i] = s.pos
	}
	if lines := acceptGrid(pos); lines != nil {
		return lines
	}
	if lines := acceptGrid(bestRegular(pos)); lines != nil {
		return lines
	}
	return nil
}

func acceptGrid(pos []float64) []float64 {
	if len(pos) == 10 && uniform(pos, 1.45) {
		return append([]float64{}, pos...)
	}
	// Four box borders, evenly spaced, are enough to recover the nine cells.
	if len(pos) == 4 && uniform(pos, 1.35) {
		return subdivide4(pos)
	}
	return nil
}

func uniform(pos []float64, ratio float64) bool {
	if len(pos) < 2 {
		return false
	}
	minG, maxG := math.MaxFloat64, 0.0
	for i := 1; i < len(pos); i++ {
		g := pos[i] - pos[i-1]
		if g < minG {
			minG = g
		}
		if g > maxG {
			maxG = g
		}
	}
	if minG <= 0.5 {
		return false
	}
	return maxG/minG <= ratio
}

func subdivide4(pos []float64) []float64 {
	out := []float64{pos[0]}
	for i := 0; i < 3; i++ {
		a, b := pos[i], pos[i+1]
		out = append(out, a+(b-a)/3, a+2*(b-a)/3, b)
	}
	return out
}

func splitRange(a, b float64, cells int) []float64 {
	out := make([]float64, cells+1)
	for i := 0; i <= cells; i++ {
		out[i] = a + (b-a)*float64(i)/float64(cells)
	}
	return out
}

func bestRegular(pos []float64) []float64 {
	if len(pos) <= 2 {
		return pos
	}
	pitch := medianFloat(gapsOf(pos))
	if pitch < 4 {
		return pos
	}
	best := []float64{}
	for i := range pos {
		run := []float64{pos[i]}
		for j := i + 1; j < len(pos); j++ {
			gap := pos[j] - run[len(run)-1]
			switch {
			case gap < pitch*0.72:
				continue
			case gap < pitch*1.38:
				run = append(run, pos[j])
			case gap > pitch*1.65 && gap < pitch*2.45:
				run = append(run, run[len(run)-1]+pitch, pos[j])
			default:
				j = len(pos)
			}
		}
		if betterRun(run, best) {
			best = append([]float64{}, run...)
		}
	}
	return best
}

func gapsOf(pos []float64) []float64 {
	g := make([]float64, 0, len(pos))
	for i := 1; i < len(pos); i++ {
		g = append(g, pos[i]-pos[i-1])
	}
	return g
}

func betterRun(a, b []float64) bool {
	if len(b) == 0 {
		return true
	}
	da := absInt(len(a) - 10)
	db := absInt(len(b) - 10)
	if da != db {
		return da < db
	}
	return spanOf(a) > spanOf(b)
}

func spanOf(p []float64) float64 {
	if len(p) < 2 {
		return 0
	}
	return p[len(p)-1] - p[0]
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func medianInt(v []int) int {
	s := append([]int{}, v...)
	sort.Ints(s)
	return s[len(s)/2]
}

func insetRect(x0, y0, x1, y1 float64) (int, int, int, int) {
	w := x1 - x0
	h := y1 - y0
	ix := math.Max(2, w*0.20)
	iy := math.Max(2, h*0.20)
	return int(math.Round(x0 + ix)), int(math.Round(y0 + iy)), int(math.Round(x1 - ix)), int(math.Round(y1 - iy))
}

// --- glyph extraction ---

func extractGlyph(px *image.RGBA, x0, y0, x1, y1 int) [][]bool {
	w, h := px.Bounds().Dx(), px.Bounds().Dy()
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > w {
		x1 = w
	}
	if y1 > h {
		y1 = h
	}
	cw, ch := x1-x0, y1-y0
	if cw < 6 || ch < 6 {
		return nil
	}
	rs := make([]int, 0, (cw+ch)*2)
	gs := make([]int, 0, (cw+ch)*2)
	bs := make([]int, 0, (cw+ch)*2)
	add := func(x, y int) {
		c := px.RGBAAt(x, y)
		rs = append(rs, int(c.R))
		gs = append(gs, int(c.G))
		bs = append(bs, int(c.B))
	}
	for x := x0; x < x1; x++ {
		add(x, y0)
		add(x, y1-1)
	}
	for y := y0; y < y1; y++ {
		add(x0, y)
		add(x1-1, y)
	}
	bgR, bgG, bgB := medianInt(rs), medianInt(gs), medianInt(bs)

	mask := make([][]bool, ch)
	for y := 0; y < ch; y++ {
		mask[y] = make([]bool, cw)
		for x := 0; x < cw; x++ {
			if dist2(px.RGBAAt(x0+x, y0+y), bgR, bgG, bgB) >= 42*42 {
				mask[y][x] = true
			}
		}
	}
	label := components(mask)
	id, box := pickComponent(label, cw, ch)
	if id == 0 {
		return nil
	}
	glyph := cropLabel(label, id, box)
	if looksLikeMarker(glyph) {
		return nil
	}
	return glyph
}

type compBox struct {
	minX, minY, maxX, maxY int
	area                   int
	touch                  bool
}

func components(mask [][]bool) [][]int {
	h := len(mask)
	w := len(mask[0])
	label := make([][]int, h)
	for y := 0; y < h; y++ {
		label[y] = make([]int, w)
	}
	type pt struct{ x, y int }
	next := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !mask[y][x] || label[y][x] != 0 {
				continue
			}
			next++
			q := []pt{{x, y}}
			label[y][x] = next
			for len(q) > 0 {
				p := q[len(q)-1]
				q = q[:len(q)-1]
				for _, d := range []pt{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					nx, ny := p.x+d.x, p.y+d.y
					if nx < 0 || ny < 0 || nx >= w || ny >= h || !mask[ny][nx] || label[ny][nx] != 0 {
						continue
					}
					label[ny][nx] = next
					q = append(q, pt{nx, ny})
				}
			}
		}
	}
	return label
}

func pickComponent(label [][]int, cw, ch int) (int, compBox) {
	h := len(label)
	w := len(label[0])
	boxes := map[int]*compBox{}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			id := label[y][x]
			if id == 0 {
				continue
			}
			b := boxes[id]
			if b == nil {
				b = &compBox{minX: x, minY: y, maxX: x, maxY: y}
				boxes[id] = b
			}
			if x < b.minX {
				b.minX = x
			}
			if y < b.minY {
				b.minY = y
			}
			if x > b.maxX {
				b.maxX = x
			}
			if y > b.maxY {
				b.maxY = y
			}
			b.area++
			if x == 0 || y == 0 || x == w-1 || y == h-1 {
				b.touch = true
			}
		}
	}
	bestID := 0
	bestScore := -1.0
	var best compBox
	for id, b := range boxes {
		bw := b.maxX - b.minX + 1
		bh := b.maxY - b.minY + 1
		if b.area < 16 || bh < ch*30/100 || bw < 2 {
			continue
		}
		fill := float64(b.area) / float64(bw*bh)
		if fill > 0.90 {
			continue
		}
		// Bare grid-line scraps: a real "1" is wide enough to include its base.
		if bw <= 3 && bh > bw*3 {
			continue
		}
		if bh <= 3 && bw > bh*3 {
			continue
		}
		if b.touch && bh > ch*65/100 && bw < cw*30/100 {
			continue
		}
		if b.touch && bw > cw*65/100 && bh < ch*32/100 {
			continue
		}
		cx := float64(b.minX+b.maxX)/2 - float64(cw)/2
		cy := float64(b.minY+b.maxY)/2 - float64(ch)/2
		score := float64(b.area) - 0.2*(cx*cx+cy*cy)
		if score > bestScore {
			bestScore = score
			bestID = id
			best = *b
		}
	}
	return bestID, best
}

func cropLabel(label [][]int, id int, b compBox) [][]bool {
	h := b.maxY - b.minY + 1
	w := b.maxX - b.minX + 1
	out := make([][]bool, h)
	for y := 0; y < h; y++ {
		out[y] = make([]bool, w)
		for x := 0; x < w; x++ {
			if label[b.minY+y][b.minX+x] == id {
				out[y][x] = true
			}
		}
	}
	return out
}

func looksLikeMarker(bits [][]bool) bool {
	if len(bits) == 0 || len(bits[0]) == 0 {
		return true
	}
	h, w := len(bits), len(bits[0])
	full := 0
	ink := 0
	for y := 0; y < h; y++ {
		row := 0
		for x := 0; x < w; x++ {
			if bits[y][x] {
				row++
				ink++
			}
		}
		if float64(row) >= 0.90*float64(w) {
			full++
		}
	}
	if h >= 4 && full >= h-1 {
		return true
	}
	fill := float64(ink) / float64(h*w)
	aspect := float64(h) / float64(w)
	return fill > 0.92 && aspect > 0.65 && aspect < 2.8
}

// --- classification ---

func classifyBits(bits [][]bool) (digit, alt int, conf, altConf float64) {
	if len(bits) < 5 || len(bits[0]) < 2 || looksLikeMarker(bits) {
		return 0, 0, 0, 0
	}
	var scores [10]float64
	rb := raster(bits)
	for d := 1; d <= 9; d++ {
		best := 0.0
		for _, proto := range protoRaster[d] {
			s := fuzzyDice(rb, proto)
			if s > best {
				best = s
			}
		}
		scores[d] = best
	}
	// A near-exact font match already separates the digit. Hole bonuses are for
	// the cases where the templates are close, and must not overturn this one.
	if d, a, c, ac, ok := decisiveTemplate(scores); ok {
		return d, a, c, ac
	}
	applyHoles(&scores, bits)
	raise(&scores, 1, scoreOne(bits))
	raise(&scores, 7, scoreSeven(bits))
	raise(&scores, 4, scoreFour(bits))
	raise(&scores, 2, scoreTwo(bits))
	raise(&scores, 3, scoreThree(bits))
	raise(&scores, 5, scoreFive(bits))
	raise(&scores, 6, scoreBowl(bits, false))
	raise(&scores, 9, scoreBowl(bits, true))

	bestD, secondD := 0, 0
	bestS, secondS := 0.0, 0.0
	for d := 1; d <= 9; d++ {
		if scores[d] > bestS {
			secondS, secondD = bestS, bestD
			bestS, bestD = scores[d], d
		} else if scores[d] > secondS {
			secondS, secondD = scores[d], d
		}
	}
	if bestS < 0.70 {
		return 0, 0, 0, 0
	}
	need := 0.05
	if bestS >= 0.96 {
		need = 0.035
	}
	if bestS-secondS < need {
		return 0, 0, 0, 0
	}
	if secondS >= 0.62 {
		alt, altConf = secondD, secondS
	}
	return bestD, alt, bestS, altConf
}

func decisiveTemplate(scores [10]float64) (digit, alt int, conf, altConf float64, ok bool) {
	bestD, secondD := 0, 0
	bestS, secondS := 0.0, 0.0
	for d := 1; d <= 9; d++ {
		if scores[d] > bestS {
			secondS, secondD = bestS, bestD
			bestS, bestD = scores[d], d
		} else if scores[d] > secondS {
			secondS, secondD = scores[d], d
		}
	}
	if bestS < 0.94 || bestS-secondS < 0.06 {
		return 0, 0, 0, 0, false
	}
	if secondS >= 0.62 {
		alt, altConf = secondD, secondS
	}
	return bestD, alt, bestS, altConf, true
}

func raise(scores *[10]float64, digit int, s float64) {
	if s > scores[digit] {
		scores[digit] = s
	}
}

type hole struct {
	cy   float64
	area int
}

func findHoles(bits [][]bool) []hole {
	h := len(bits)
	w := len(bits[0])
	H, W := h+2, w+2
	ink := func(x, y int) bool {
		if x <= 0 || y <= 0 || x >= W-1 || y >= H-1 {
			return false
		}
		return bits[y-1][x-1]
	}
	seen := make([][]bool, H)
	for y := 0; y < H; y++ {
		seen[y] = make([]bool, W)
	}
	type pt struct{ x, y int }
	q := make([]pt, 0, W*2)
	push := func(x, y int) {
		if x < 0 || y < 0 || x >= W || y >= H || seen[y][x] || ink(x, y) {
			return
		}
		seen[y][x] = true
		q = append(q, pt{x, y})
	}
	for x := 0; x < W; x++ {
		push(x, 0)
		push(x, H-1)
	}
	for y := 0; y < H; y++ {
		push(0, y)
		push(W-1, y)
	}
	for head := 0; head < len(q); head++ {
		p := q[head]
		push(p.x+1, p.y)
		push(p.x-1, p.y)
		push(p.x, p.y+1)
		push(p.x, p.y-1)
	}
	var holes []hole
	for y := 1; y <= h; y++ {
		for x := 1; x <= w; x++ {
			if seen[y][x] || ink(x, y) {
				continue
			}
			area := 0
			sumY := 0
			hq := []pt{{x, y}}
			seen[y][x] = true
			for len(hq) > 0 {
				p := hq[len(hq)-1]
				hq = hq[:len(hq)-1]
				area++
				sumY += p.y - 1
				for _, d := range []pt{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					nx, ny := p.x+d.x, p.y+d.y
					if nx < 0 || ny < 0 || nx >= W || ny >= H || seen[ny][nx] || ink(nx, ny) {
						continue
					}
					seen[ny][nx] = true
					hq = append(hq, pt{nx, ny})
				}
			}
			if area >= 2 {
				holes = append(holes, hole{cy: float64(sumY) / float64(area*h), area: area})
			}
		}
	}
	return holes
}

func applyHoles(scores *[10]float64, bits [][]bool) {
	holes := findHoles(bits)
	switch {
	case len(holes) >= 2:
		scores[8] = math.Min(1, scores[8]+0.12)
		for d := 1; d <= 9; d++ {
			if d != 8 {
				scores[d] -= 0.18
			}
		}
	case len(holes) == 1:
		if holes[0].cy < 0.50 {
			scores[9] = math.Min(1, scores[9]+0.10)
			scores[6] -= 0.18
		} else {
			scores[6] = math.Min(1, scores[6]+0.10)
			scores[9] -= 0.18
		}
		scores[8] -= 0.10
		scores[1] -= 0.25
		scores[7] -= 0.20
		scores[4] -= 0.06
		// 2, 3, and 5 often close a bowl. A single counter is not enough to
		// throw away a strong template match for those digits.
	default:
		scores[8] -= 0.22
		scores[6] -= 0.08
		scores[9] -= 0.08
	}
	for d := 1; d <= 9; d++ {
		if scores[d] < 0 {
			scores[d] = 0
		}
	}
}

func density(bits [][]bool, x0, x1, y0, y1 float64) float64 {
	h, w := len(bits), len(bits[0])
	xa := clamp(int(math.Floor(x0*float64(w))), 0, w)
	xb := clamp(int(math.Ceil(x1*float64(w))), 0, w)
	ya := clamp(int(math.Floor(y0*float64(h))), 0, h)
	yb := clamp(int(math.Ceil(y1*float64(h))), 0, h)
	if xb <= xa {
		xb = clamp(xa+1, 0, w)
	}
	if yb <= ya {
		yb = clamp(ya+1, 0, h)
	}
	ink, tot := 0, 0
	for y := ya; y < yb; y++ {
		for x := xa; x < xb; x++ {
			tot++
			if bits[y][x] {
				ink++
			}
		}
	}
	if tot == 0 {
		return 0
	}
	return float64(ink) / float64(tot)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func rowFill(bits [][]bool, y int) float64 {
	if y < 0 || y >= len(bits) {
		return 0
	}
	ink := 0
	for x := 0; x < len(bits[y]); x++ {
		if bits[y][x] {
			ink++
		}
	}
	return float64(ink) / float64(len(bits[y]))
}

func comX(bits [][]bool, y0, y1 float64) float64 {
	h, w := len(bits), len(bits[0])
	ya := clamp(int(math.Floor(y0*float64(h))), 0, h)
	yb := clamp(int(math.Ceil(y1*float64(h))), 0, h)
	sum, n := 0.0, 0.0
	for y := ya; y < yb; y++ {
		for x := 0; x < w; x++ {
			if bits[y][x] {
				sum += float64(x) + 0.5
				n++
			}
		}
	}
	if n == 0 {
		return 0.5
	}
	return sum / n / float64(w)
}

func scoreOne(bits [][]bool) float64 {
	h, w := len(bits), len(bits[0])
	y0, y1 := int(0.28*float64(h)), int(0.75*float64(h))
	if y1 <= y0 {
		return 0
	}
	maxMid := 0
	for y := y0; y < y1; y++ {
		c := 0
		for x := 0; x < w; x++ {
			if bits[y][x] {
				c++
			}
		}
		if c > maxMid {
			maxMid = c
		}
	}
	if float64(maxMid) > 0.42*float64(w) {
		return 0
	}
	bestCol := 0
	for x := 0; x < w; x++ {
		c := 0
		for y := 0; y < h; y++ {
			if bits[y][x] {
				c++
			}
		}
		if c > bestCol {
			bestCol = c
		}
	}
	if float64(bestCol) < 0.72*float64(h) {
		return 0
	}
	return 0.84
}

func scoreSeven(bits [][]bool) float64 {
	top := density(bits, 0, 1, 0, 0.18)
	bot := density(bits, 0, 1, 0.72, 1)
	if top < 0.82 || bot > 0.42 {
		return 0
	}
	if comX(bits, 0, 0.35)-comX(bits, 0.62, 1) < 0.10 {
		return 0
	}
	if comX(bits, 0.62, 1) > 0.58 {
		return 0
	}
	return 0.84
}

func scoreFour(bits [][]bool) float64 {
	h := len(bits)
	yA := int(0.50 * float64(h))
	yB := int(0.84 * float64(h))
	if yB <= yA {
		return 0
	}
	bar := yA
	barFill := -1.0
	for y := yA; y < yB; y++ {
		f := rowFill(bits, y)
		if f > barFill {
			barFill = f
			bar = y
		}
	}
	if barFill < 0.85 {
		return 0
	}
	belowL := density(bits, 0, 0.48, float64(bar+1)/float64(h), 1)
	belowR := density(bits, 0.52, 1, float64(bar+1)/float64(h), 1)
	aboveL := density(bits, 0, 0.48, 0.05, float64(bar)/float64(h))
	aboveR := density(bits, 0.52, 1, 0.05, float64(bar)/float64(h))
	if belowR < 0.22 || belowL > 0.16 {
		return 0
	}
	if aboveL < 0.15 || aboveR < 0.15 {
		return 0
	}
	return 0.84
}

func scoreTwo(bits [][]bool) float64 {
	bot := density(bits, 0, 1, 0.82, 1)
	if bot < 0.70 {
		return 0
	}
	ur := density(bits, 0.50, 1, 0.18, 0.50)
	ul := density(bits, 0, 0.48, 0.18, 0.50)
	ll := density(bits, 0, 0.50, 0.55, 0.82)
	lr := density(bits, 0.55, 1, 0.55, 0.82)
	if ur < ul+0.12 || ll < lr {
		return 0
	}
	if len(findHoles(bits)) > 0 {
		return 0
	}
	return 0.80
}

func scoreThree(bits [][]bool) float64 {
	top := density(bits, 0, 1, 0, 0.20)
	bot := density(bits, 0, 1, 0.78, 1)
	right := density(bits, 0.55, 1, 0.15, 0.85)
	leftMid := density(bits, 0, 0.42, 0.30, 0.70)
	if top < 0.60 || bot < 0.55 || right < 0.45 || leftMid > 0.28 {
		return 0
	}
	if len(findHoles(bits)) > 0 {
		return 0
	}
	return 0.80
}

func scoreFive(bits [][]bool) float64 {
	top := density(bits, 0, 1, 0, 0.20)
	ul := density(bits, 0, 0.50, 0.12, 0.48)
	ur := density(bits, 0.62, 1, 0.22, 0.45)
	bot := density(bits, 0, 1, 0.62, 1)
	if top < 0.70 || ul < 0.40 || ur > 0.35 || bot < 0.45 {
		return 0
	}
	if len(findHoles(bits)) > 0 {
		return 0
	}
	return 0.80
}

// scoreBowl recognizes a 6 (hole low) or 9 (hole high) when the counter is
// almost closed but the stroke is broken by a pixel.
func scoreBowl(bits [][]bool, upper bool) float64 {
	holes := findHoles(bits)
	if len(holes) == 1 {
		if upper && holes[0].cy < 0.48 {
			return 0.88
		}
		if !upper && holes[0].cy >= 0.48 {
			return 0.88
		}
	}
	return 0
}

func raster(bits [][]bool) [][]bool {
	const canvasW, canvasH = 24, 32
	h := len(bits)
	w := len(bits[0])
	scale := float64(canvasH) / float64(h)
	tw := int(math.Round(float64(w) * scale))
	th := canvasH
	if tw > canvasW {
		scale = float64(canvasW) / float64(w)
		tw = canvasW
		th = int(math.Round(float64(h) * scale))
	}
	if tw < 1 {
		tw = 1
	}
	if th < 1 {
		th = 1
	}
	dst := make([][]bool, canvasH)
	for y := 0; y < canvasH; y++ {
		dst[y] = make([]bool, canvasW)
	}
	x0 := (canvasW - tw) / 2
	y0 := (canvasH - th) / 2
	for y := 0; y < th; y++ {
		sy0 := y * h / th
		sy1 := (y + 1) * h / th
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for x := 0; x < tw; x++ {
			sx0 := x * w / tw
			sx1 := (x + 1) * w / tw
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			ink, tot := 0, 0
			for yy := sy0; yy < sy1 && yy < h; yy++ {
				for xx := sx0; xx < sx1 && xx < w; xx++ {
					tot++
					if bits[yy][xx] {
						ink++
					}
				}
			}
			if tot > 0 && ink*2 >= tot {
				dst[y0+y][x0+x] = true
			}
		}
	}
	return dst
}

func fuzzyDice(a, b [][]bool) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) || len(a[0]) != len(b[0]) {
		return 0
	}
	h, w := len(a), len(a[0])
	near := func(bits [][]bool, x, y int) bool {
		for dy := -1; dy <= 1; dy++ {
			yy := y + dy
			if yy < 0 || yy >= h {
				continue
			}
			for dx := -1; dx <= 1; dx++ {
				xx := x + dx
				if xx >= 0 && xx < w && bits[yy][xx] {
					return true
				}
			}
		}
		return false
	}
	ga, gb, hitA, hitB := 0, 0, 0, 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if a[y][x] {
				ga++
				if near(b, x, y) {
					hitA++
				}
			}
			if b[y][x] {
				gb++
				if near(a, x, y) {
					hitB++
				}
			}
		}
	}
	if ga == 0 || gb == 0 {
		return 0
	}
	prec := float64(hitA) / float64(ga)
	rec := float64(hitB) / float64(gb)
	if prec+rec == 0 {
		return 0
	}
	return 2 * prec * rec / (prec + rec)
}

func bitsFrom(rows []string) [][]bool {
	h := len(rows)
	w := len(rows[0])
	b := make([][]bool, h)
	for y, row := range rows {
		b[y] = make([]bool, w)
		for x := 0; x < len(row) && x < w; x++ {
			if row[x] == '#' {
				b[y][x] = true
			}
		}
	}
	return b
}

// protoSrc holds one bold shape per digit, taken from a real on-screen
// sudoku face, plus a plainer geometric shape so a thinner font still lands
// nearer the right digit than the others.
var protoSrc = map[int][][]string{
	1: {
		{
			"#####...",
			"#####...",
			"#####...",
			"...##...",
			"...##...",
			"...##...",
			"...##...",
			"...##...",
			"...##...",
			"########",
			"########",
		},
		{
			"..#..",
			"..#..",
			"..#..",
			"..#..",
			"..#..",
			"..#..",
			"..#..",
			"..#..",
			"#####",
		},
	},
	2: {
		{
			"..#####..",
			"########.",
			".##..###.",
			"......##.",
			".....###.",
			".....###.",
			"....###..",
			"...###...",
			".####....",
			".########",
			".########",
		},
		{
			".####.",
			"#....#",
			".....#",
			"....#.",
			"...#..",
			"..#...",
			".#....",
			"#.....",
			"######",
		},
	},
	3: {
		{
			"#######.",
			"#######.",
			"###.###.",
			"....###.",
			"..####..",
			"..####..",
			"..#####.",
			".....###",
			"#...####",
			"#######.",
			"######..",
		},
		{
			".####.",
			"#....#",
			".....#",
			"..###.",
			".....#",
			".....#",
			"#....#",
			".####.",
		},
	},
	4: {
		{
			"..##.###.",
			".###.###.",
			".###.###.",
			".###.###.",
			".###.###.",
			".##..###.",
			".##..###.",
			"#########",
			"#########",
			".....###.",
			".....###.",
		},
		{
			"...#..",
			"..##..",
			".#.#..",
			"#..#..",
			"#..#..",
			"######",
			"...#..",
			"...#..",
			"...#..",
		},
	},
	5: {
		{
			"#######.",
			"#######.",
			"###.....",
			"###.#...",
			"#######.",
			"########",
			".#...###",
			"##...###",
			"###..###",
			"#######.",
			".#####..",
		},
		{
			"######",
			"#.....",
			"#.....",
			"#####.",
			".....#",
			".....#",
			"#....#",
			".####.",
		},
	},
	6: {
		{
			"....####.",
			"..######.",
			".######..",
			".###.....",
			".#######.",
			".########",
			"####..###",
			".##...###",
			".###.####",
			".########",
			"..######.",
		},
		{
			"..###.",
			".#....",
			"#.....",
			"#.....",
			"#.###.",
			"##...#",
			"#....#",
			"#....#",
			".####.",
		},
	},
	7: {
		{
			"#########",
			"#########",
			"###...###",
			"###...##.",
			".....###.",
			".....###.",
			".....##..",
			"....###..",
			"....###..",
			"....##...",
			"...###...",
		},
		{
			"######",
			".....#",
			"....#.",
			"...#..",
			"...#..",
			"..#...",
			"..#...",
			".#....",
		},
	},
	8: {
		{
			"..######.",
			".#######.",
			".###.####",
			".###..##.",
			".#######.",
			"..######.",
			".########",
			"###...###",
			"####..###",
			".########",
			"..######.",
		},
		{
			".####.",
			"#....#",
			"#....#",
			".####.",
			"#....#",
			"#....#",
			"#....#",
			".####.",
		},
	},
	9: {
		{
			"..#####..",
			".#######.",
			".###..###",
			"###...###",
			".###..###",
			".########",
			".########",
			".....###.",
			"...#####.",
			".######..",
			".#####...",
		},
		{
			".####.",
			"#....#",
			"#....#",
			"#....#",
			".#####",
			".....#",
			".....#",
			".###..",
		},
	},
}

var protoBits [10][][][]bool
var protoRaster [10][][][]bool

func init() {
	for d := 1; d <= 9; d++ {
		for _, rows := range protoSrc[d] {
			bits := bitsFrom(rows)
			protoBits[d] = append(protoBits[d], bits)
			protoRaster[d] = append(protoRaster[d], raster(bits))
		}
	}
}
