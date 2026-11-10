package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveRoundTripWithBOM(t *testing.T) {
	path := filepath.Join(t.TempDir(), saveFileName)
	s := newSaveData()
	s.Name = "Kaleb"
	s.Theme = themeLight
	s.Settings.BestOf = 3
	s.Settings.Rules = Rules{TwoDice: true}
	r := s.recordFor(recordKey(modeAI, diffHard))
	r.Games, r.Wins, r.Best = 4, 3, 88
	s.addHighScore(highScore{Name: "Kaleb", Score: 88, Opponent: "Shrumy", Mode: modeAI})
	if err := s.write(path); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !bytes.HasPrefix(data, utf8BOM) {
		t.Fatal("save file has no UTF-8 BOM")
	}
	got := loadSave(path)
	if got.Name != "Kaleb" || got.Theme != themeLight || got.Settings.BestOf != 3 || !got.Settings.Rules.TwoDice {
		t.Fatalf("loaded %+v", got)
	}
	if rr := got.Records["ai-hard"]; rr == nil || rr.Wins != 3 || rr.Best != 88 {
		t.Fatalf("record %+v", rr)
	}
	if len(got.HighScores) != 1 || got.HighScores[0].Score != 88 {
		t.Fatalf("high scores %+v", got.HighScores)
	}
}

func TestLoadMissingOrBrokenSave(t *testing.T) {
	dir := t.TempDir()
	if s := loadSave(filepath.Join(dir, "none.json")); s.Settings.Port != defaultPort {
		t.Fatalf("defaults %+v", s.Settings)
	}
	bad := filepath.Join(dir, "bad.json")
	os.WriteFile(bad, []byte("{not json"), 0o644)
	if s := loadSave(bad); s.Records == nil || s.Settings.BestOf != 1 {
		t.Fatalf("broken save not reset: %+v", s)
	}
	odd := filepath.Join(dir, "odd.json")
	os.WriteFile(odd, []byte(`{"settings":{"bestOf":7,"port":-1,"difficulty":"insane","turnSeconds":9}}`), 0o644)
	s := loadSave(odd)
	if s.Settings.BestOf != 1 || s.Settings.Port != defaultPort || s.Settings.Difficulty != diffMedium || s.Settings.TurnSeconds != 30 {
		t.Fatalf("settings not normalized: %+v", s.Settings)
	}
}

func TestHighScoresTopTen(t *testing.T) {
	s := newSaveData()
	for i := 1; i <= 12; i++ {
		s.addHighScore(highScore{Score: i * 5})
	}
	if len(s.HighScores) != maxHighScores || s.HighScores[0].Score != 60 || s.HighScores[9].Score != 15 {
		t.Fatalf("scores %+v", s.HighScores)
	}
	if rank := s.addHighScore(highScore{Score: 1}); rank != 0 {
		t.Fatalf("low score ranked %d", rank)
	}
	if rank := s.addHighScore(highScore{Score: 61}); rank != 1 {
		t.Fatalf("top score ranked %d", rank)
	}
}

func TestCleanName(t *testing.T) {
	if got := cleanName("  Klunko & Bop the Great Drummer ", "x"); got != "Klunko & Bop the" {
		t.Fatalf("got %q", got)
	}
	if got := cleanName("\t\n", "Lamb"); got != "Lamb" {
		t.Fatalf("got %q", got)
	}
}
