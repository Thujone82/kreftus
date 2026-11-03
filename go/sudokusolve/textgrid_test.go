package main

import "testing"

func TestDecodeTextGrid_User81CharString(t *testing.T) {
	s := "600050004050704020000690000090802450802000901034907060000028000060309080200070009"
	b, ok := decodeTextGrid(s)
	if !ok {
		t.Fatalf("expected 81-char string to decode successfully")
	}
	if b[0][0] != 6 || b[0][4] != 5 || b[0][8] != 4 {
		t.Errorf("row 0 mismatch: %v", b[0])
	}
	if b[8][0] != 2 || b[8][4] != 7 || b[8][8] != 9 {
		t.Errorf("row 8 mismatch: %v", b[8])
	}
	if !validateInitialBoard(&b) {
		t.Errorf("expected board to be valid")
	}
}

func TestDecodeTextGrid_QuotedAndEmbedded81(t *testing.T) {
	s := `  "600050004050704020000690000090802450802000901034907060000028000060309080200070009"  `
	b, ok := decodeTextGrid(s)
	if !ok || b[0][0] != 6 {
		t.Fatalf("expected quoted 81-char string to decode")
	}

	embedded := `Puzzle 42: https://sudoku.example.com/#600050004050704020000690000090802450802000901034907060000028000060309080200070009 (Hard)`
	b2, ok2 := decodeTextGrid(embedded)
	if !ok2 || b2[0][0] != 6 {
		t.Fatalf("expected embedded 81-char string to decode")
	}
}

func TestDecodeTextGrid_DotBlanks81(t *testing.T) {
	s := "6...5...4.5.7.4.2....69.....9.8.245.8.2...9.1.349.7.6.....28....6.3.9.8.2...7...9"
	b, ok := decodeTextGrid(s)
	if !ok {
		t.Fatalf("expected dot-based 81-char string to decode")
	}
	if b[0][0] != 6 || b[0][1] != 0 || b[0][4] != 5 {
		t.Errorf("row 0 mismatch: %v", b[0])
	}
}

func TestDecodeTextGrid_NineLines(t *testing.T) {
	s := `82.6...91
..1..8263
3.621.8..
6..7.19.4
..4.5.1.6
1.9..6...
.13.62..9
4981..6..
.6.....1.`
	b, ok := decodeTextGrid(s)
	if !ok {
		t.Fatalf("expected 9 lines format to decode")
	}
	if b[0][0] != 8 || b[0][1] != 2 || b[0][2] != 0 || b[0][3] != 6 {
		t.Errorf("row 0 mismatch: %v", b[0])
	}
}

func TestDecodeTextGrid_NineLinesWithSpaces(t *testing.T) {
	s := `8 2 . 6 . . . 9 1
. . 1 . . 8 2 6 3
3 . 6 2 1 . 8 . .
6 . . 7 . 1 9 . 4
. . 4 . 5 . 1 . 6
1 . 9 . . 6 . . .
. 1 3 . 6 2 . . 9
4 9 8 1 . . 6 . .
. 6 . . . . . 1 .`
	b, ok := decodeTextGrid(s)
	if !ok {
		t.Fatalf("expected spaced 9 lines format to decode")
	}
	if b[0][0] != 8 || b[0][1] != 2 {
		t.Errorf("row 0 mismatch: %v", b[0])
	}
}

func TestDecodeTextGrid_ASCIIBox(t *testing.T) {
	s := `+-------+-------+-------+
| 8 2 . | 6 . . | . 9 1 |
| . . 1 | . . 8 | 2 6 3 |
| 3 . 6 | 2 1 . | 8 . . |
+-------+-------+-------+
| 6 . . | 7 . 1 | 9 . 4 |
| . . 4 | . 5 . | 1 . 6 |
| 1 . 9 | . . 6 | . . . |
+-------+-------+-------+
| . 1 3 | . 6 2 | . . 9 |
| 4 9 8 | 1 . . | 6 . . |
| . 6 . | . . . | . 1 . |
+-------+-------+-------+`
	b, ok := decodeTextGrid(s)
	if !ok {
		t.Fatalf("expected ASCII box format to decode")
	}
	if b[0][0] != 8 || b[0][1] != 2 || b[8][7] != 1 {
		t.Errorf("cell mismatch: %v", b)
	}
}

func TestDecodeTextGrid_CSV(t *testing.T) {
	s := `8,2,0,6,0,0,0,9,1
0,0,1,0,0,8,2,6,3
3,0,6,2,1,0,8,0,0
6,0,0,7,0,1,9,0,4
0,0,4,0,5,0,1,0,6
1,0,9,0,0,6,0,0,0
0,1,3,0,6,2,0,0,9
4,9,8,1,0,0,6,0,0
0,6,0,0,0,0,0,1,0`
	b, ok := decodeTextGrid(s)
	if !ok {
		t.Fatalf("expected CSV format to decode")
	}
	if b[0][0] != 8 || b[0][1] != 2 {
		t.Errorf("cell mismatch: %v", b)
	}
}

func TestDecodeTextGrid_JSONArray2D(t *testing.T) {
	s := `[
  [8, 2, 0, 6, 0, 0, 0, 9, 1],
  [0, 0, 1, 0, 0, 8, 2, 6, 3],
  [3, 0, 6, 2, 1, 0, 8, 0, 0],
  [6, 0, 0, 7, 0, 1, 9, 0, 4],
  [0, 0, 4, 0, 5, 0, 1, 0, 6],
  [1, 0, 9, 0, 0, 6, 0, 0, 0],
  [0, 1, 3, 0, 6, 2, 0, 0, 9],
  [4, 9, 8, 1, 0, 0, 6, 0, 0],
  [0, 6, 0, 0, 0, 0, 0, 1, 0]
]`
	b, ok := decodeTextGrid(s)
	if !ok {
		t.Fatalf("expected 2D JSON array format to decode")
	}
	if b[0][0] != 8 || b[0][1] != 2 {
		t.Errorf("cell mismatch: %v", b)
	}
}

func TestDecodeTextGrid_JSONArray1D(t *testing.T) {
	s := `[8, 2, 0, 6, 0, 0, 0, 9, 1, 0, 0, 1, 0, 0, 8, 2, 6, 3, 3, 0, 6, 2, 1, 0, 8, 0, 0, 6, 0, 0, 7, 0, 1, 9, 0, 4, 0, 0, 4, 0, 5, 0, 1, 0, 6, 1, 0, 9, 0, 0, 6, 0, 0, 0, 0, 1, 3, 0, 6, 2, 0, 0, 9, 4, 9, 8, 1, 0, 0, 6, 0, 0, 0, 6, 0, 0, 0, 0, 0, 1, 0]`
	b, ok := decodeTextGrid(s)
	if !ok {
		t.Fatalf("expected 1D JSON array format to decode")
	}
	if b[0][0] != 8 || b[8][7] != 1 {
		t.Errorf("cell mismatch: %v", b)
	}
}

func TestDecodeTextGrid_RowPrefixed(t *testing.T) {
	s := `Row 1: 8 2 . 6 . . . 9 1
Row 2: . . 1 . . 8 2 6 3
Row 3: 3 . 6 2 1 . 8 . .
Row 4: 6 . . 7 . 1 9 . 4
Row 5: . . 4 . 5 . 1 . 6
Row 6: 1 . 9 . . 6 . . .
Row 7: . 1 3 . 6 2 . . 9
Row 8: 4 9 8 1 . . 6 . .
Row 9: . 6 . . . . . 1 .`
	b, ok := decodeTextGrid(s)
	if !ok {
		t.Fatalf("expected row-prefixed format to decode")
	}
	if b[0][0] != 8 || b[0][1] != 2 {
		t.Errorf("cell mismatch: %v", b)
	}
}

func TestDecodeTextGrid_InvalidText(t *testing.T) {
	invalid := []string{
		"",
		"hello world",
		"1 2 3 4 5",
		"12345678",
	}
	for _, text := range invalid {
		if _, ok := decodeTextGrid(text); ok {
			t.Errorf("expected %q to fail decoding", text)
		}
	}
}