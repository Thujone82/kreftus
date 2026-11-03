package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strings"
	"testing"
)

func TestSamplePuzzle(t *testing.T) {
	img := loadSample(t)
	got, n := ocrSudokuImage(img)
	if !sameBoard(got, samplePuzzle) {
		t.Fatalf("recognized %d\n got:\n%s\nwant:\n%s", n, formatBoard(got), formatBoard(samplePuzzle))
	}
	if !validateInitialBoard(&got) {
		t.Fatal("result repeats a digit in a row, column, or box")
	}
}

func TestPaddedSample(t *testing.T) {
	img := loadSample(t)
	padded := padImage(img, 47, 80, 23, 16, color.RGBA{18, 16, 28, 255})
	got, _ := ocrSudokuImage(padded)
	if !sameBoard(got, samplePuzzle) {
		t.Fatalf("dark margin shifted the crop\n got:\n%s\nwant:\n%s", formatBoard(got), formatBoard(samplePuzzle))
	}
	whiteTop := padImage(img, 12, 36, 12, 8, color.RGBA{245, 245, 245, 255})
	got, _ = ocrSudokuImage(whiteTop)
	if !sameBoard(got, samplePuzzle) {
		t.Fatalf("white margin shifted the crop\n got:\n%s\nwant:\n%s", formatBoard(got), formatBoard(samplePuzzle))
	}
}

func TestInvertedSample(t *testing.T) {
	img := loadSample(t)
	got, _ := ocrSudokuImage(invertImage(img))
	if !sameBoard(got, samplePuzzle) {
		t.Fatalf("inverted colors\n got:\n%s\nwant:\n%s", formatBoard(got), formatBoard(samplePuzzle))
	}
}

func TestOCRReportsProgress(t *testing.T) {
	img := loadSample(t)
	var last float64
	var sawCell bool
	_, n, ok := ocrWithProgress(img, func(frac float64, stage string) {
		if frac+0.0001 < last {
			t.Errorf("progress went backwards %.3f -> %.3f (%s)", last, frac, stage)
		}
		last = frac
		if strings.Contains(stage, "cell") {
			sawCell = true
		}
	})
	if !ok || n == 0 {
		t.Fatal("expected a full grid")
	}
	if last < 0.98 {
		t.Fatalf("final progress %.3f", last)
	}
	if !sawCell {
		t.Fatal("expected per-cell progress")
	}
}

func TestPartialImageIsNotAGrid(t *testing.T) {
	f, err := os.Open("testdata/partial_grid.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	board, n, ok := ocrWithProgress(img, nil)
	if ok || n != 0 {
		t.Fatalf("partial image accepted: ok=%v n=%d\n%s", ok, n, formatBoard(board))
	}
}

func TestBlankImage(t *testing.T) {
	black := image.NewRGBA(image.Rect(0, 0, 180, 180))
	if _, n := ocrSudokuImage(black); n != 0 {
		t.Fatalf("black image recognized %d digits", n)
	}
	white := image.NewRGBA(image.Rect(0, 0, 180, 180))
	draw.Draw(white, white.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	if _, n := ocrSudokuImage(white); n != 0 {
		t.Fatalf("white image recognized %d digits", n)
	}
}

func TestPrototypeGlyphs(t *testing.T) {
	for d := 1; d <= 9; d++ {
		bits := protoBits[d][0]
		got, alt, conf, altConf := classifyBits(bits)
		if got != d {
			t.Errorf("prototype %d read as %d (alt %d conf %.2f/%.2f) holes=%d", d, got, alt, conf, altConf, len(findHoles(bits)))
		}
	}
	marker := bitsFrom([]string{
		"#########",
		"#########",
		"#########",
		"#########",
		"#########",
		"#########",
		"#########",
		"#########",
		"#########",
		"#########",
	})
	if d, _, _, _ := classifyBits(marker); d != 0 {
		t.Errorf("solid square read as %d", d)
	}
}

func TestConflictingReadingsDropped(t *testing.T) {
	hits := []ocrHit{
		{r: 0, c: 0, digit: 5, conf: 0.95},
		{r: 0, c: 1, digit: 5, alt: 3, conf: 0.80, altConf: 0.72},
		{r: 1, c: 0, digit: 5, conf: 0.70},
	}
	board, n := reconcile(hits)
	if board[0][0] != 5 || board[0][1] != 3 || board[1][0] != 0 || n != 2 {
		t.Fatalf("conflicts not resolved: %v n=%d", board, n)
	}
	if !validateInitialBoard(&board) {
		t.Fatal("reconciled board is illegal")
	}
}

var samplePuzzle = Board{
	{8, 2, 0, 6, 0, 0, 0, 9, 1},
	{0, 0, 1, 0, 0, 8, 2, 6, 3},
	{3, 0, 6, 2, 1, 0, 8, 0, 0},
	{6, 0, 0, 7, 0, 1, 9, 0, 4},
	{0, 0, 4, 0, 5, 0, 1, 0, 6},
	{1, 0, 9, 0, 0, 6, 0, 0, 0},
	{0, 1, 3, 0, 6, 2, 0, 0, 9},
	{4, 9, 8, 1, 0, 0, 6, 0, 0},
	{0, 6, 0, 0, 0, 0, 0, 1, 0},
}

func loadSample(t *testing.T) image.Image {
	t.Helper()
	f, err := os.Open("testdata/sample_grid.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func padImage(src image.Image, l, t, r, b int, bg color.RGBA) *image.RGBA {
	sb := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, sb.Dx()+l+r, sb.Dy()+t+b))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)
	draw.Draw(dst, image.Rect(l, t, l+sb.Dx(), t+sb.Dy()), src, sb.Min, draw.Src)
	return dst
}

func invertImage(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := color.RGBAModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.RGBA)
			dst.SetRGBA(x, y, color.RGBA{R: 255 - c.R, G: 255 - c.G, B: 255 - c.B, A: c.A})
		}
	}
	return dst
}

func sameBoard(a, b Board) bool {
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			if a[r][c] != b[r][c] {
				return false
			}
		}
	}
	return true
}

func formatBoard(b Board) string {
	s := ""
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			if b[r][c] == 0 {
				s += "."
			} else {
				s += string(rune('0' + b[r][c]))
			}
		}
		s += "\n"
	}
	return s
}
