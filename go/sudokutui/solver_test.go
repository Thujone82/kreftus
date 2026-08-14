package main

import "testing"

func TestSolveWikipedia(t *testing.T) {
	givens := "530070000600195000098000060800060003400803001700020006060000280000419005000080079"
	want := "534678912672195348198342567859761423426853791713924856961537284287419635345286179"
	got := solveSudoku(givens)
	if got != want {
		t.Fatalf("solveSudoku = %q\nwant %q", got, want)
	}
}

func TestSolveRejectsBadLength(t *testing.T) {
	if solveSudoku("123") != "" {
		t.Fatal("expected empty for short input")
	}
}

func TestSolveEmptyHasSolution(t *testing.T) {
	empty := "000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	got := solveSudoku(empty)
	if len(got) != 81 {
		t.Fatalf("empty grid should still solve, got len %d", len(got))
	}
	if solveSudoku(got) != got {
		t.Fatal("solved empty grid is not self-consistent")
	}
}
