package main

import "testing"

func TestThemes(t *testing.T) {
	if len(themes) != 6 || len(themeOrder) != 6 {
		t.Fatalf("expected 6 themes, got len(themes)=%d len(themeOrder)=%d", len(themes), len(themeOrder))
	}

	seenIDs := make(map[string]bool)
	seenNames := make(map[string]bool)

	for _, id := range themeOrder {
		th, ok := themes[id]
		if !ok {
			t.Fatalf("themeOrder contains missing theme %q", id)
		}
		if th.ID != id {
			t.Errorf("theme %q has ID %q", id, th.ID)
		}
		if seenIDs[th.ID] {
			t.Errorf("duplicate theme ID %q", th.ID)
		}
		seenIDs[th.ID] = true

		if th.Name == "" {
			t.Errorf("theme %q has empty Name", id)
		}
		if seenNames[th.Name] {
			t.Errorf("duplicate theme Name %q", th.Name)
		}
		seenNames[th.Name] = true

		if th.BG == th.FG {
			t.Errorf("theme %q has identical BG and FG: %+v", id, th.BG)
		}

		// Non-zero box runeset.
		boxRunes := []rune{
			th.Box.TL, th.Box.TR, th.Box.BL, th.Box.BR, th.Box.H, th.Box.V,
			th.Box.LT, th.Box.RT, th.Box.TT, th.Box.BT, th.Box.X,
			th.Box.IH, th.Box.IV, th.Box.ILT, th.Box.IRT, th.Box.ITT, th.Box.IBT,
		}
		for i, r := range boxRunes {
			if r == 0 {
				t.Errorf("theme %q has zero box rune at index %d", id, i)
			}
		}

		// 6 distinct peg colors.
		pegSet := make(map[rgb]bool)
		for i, p := range th.Pegs {
			if pegSet[p] {
				t.Errorf("theme %q has duplicate peg color at %d: %+v", id, i, p)
			}
			pegSet[p] = true
		}
	}

	// nextTheme cycles through all 6 and wraps.
	cur := themeOrder[0]
	visited := make(map[string]bool)
	for i := 0; i < len(themeOrder); i++ {
		if visited[cur] {
			t.Fatalf("nextTheme repeated %q before cycling all 6", cur)
		}
		visited[cur] = true
		cur = nextTheme(cur)
	}
	if cur != themeOrder[0] {
		t.Errorf("nextTheme did not wrap back to %q, got %q", themeOrder[0], cur)
	}

	// Unknown ID falls back to dark.
	if got := themeByID("nonexistent").ID; got != defaultThemeID {
		t.Errorf("themeByID(\"nonexistent\") = %q, want %q", got, defaultThemeID)
	}
}
