package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
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

func TestMigrateOldDefaultPort(t *testing.T) {
	path := filepath.Join(t.TempDir(), saveFileName)
	os.WriteFile(path, []byte(`{"settings":{"bestOf":3,"port":7777,"joinAddr":"192.168.1.20:7777","difficulty":"medium","turnSeconds":30}}`), 0o644)
	s := loadSave(path)
	if s.Settings.Port != defaultPort {
		t.Fatalf("port %d, want %d", s.Settings.Port, defaultPort)
	}
	if s.Settings.JoinAddr != "192.168.1.20:"+strconv.Itoa(defaultPort) {
		t.Fatalf("join %q", s.Settings.JoinAddr)
	}
	got := loadSave(path)
	if got.Settings.Port != defaultPort || got.Settings.JoinAddr != s.Settings.JoinAddr {
		t.Fatalf("migrated save not written: %+v", got.Settings)
	}
}

func TestCleanName(t *testing.T) {
	if got := cleanName("  Klunko & Bop the Great Drummer ", "x"); got != "Klunko & Bop the" {
		t.Fatalf("got %q", got)
	}
	if got := cleanName("\t\n", "Lamb"); got != "Lamb" {
		t.Fatalf("got %q", got)
	}
	if got := cleanName("A\x1b[31mB\u202eC", "x"); got != "A[31mBC" {
		t.Fatalf("got %q", got)
	}
}

func TestCleanTextAndSaveBounds(t *testing.T) {
	if got := cleanText("  bad\x1b[2J\u202etext\nline  ", 20); got != "bad[2Jtextline" {
		t.Fatalf("cleanText=%q", got)
	}
	oversized := filepath.Join(t.TempDir(), saveFileName)
	buf := make([]byte, maxSaveBytes+32)
	for i := range buf {
		buf[i] = 'a'
	}
	if err := os.WriteFile(oversized, buf, 0o600); err != nil {
		t.Fatal(err)
	}
	s := loadSave(oversized)
	if s.Settings.Port != defaultPort {
		t.Fatalf("oversized save should return defaults, got port %d", s.Settings.Port)
	}
}

func TestResolveSavePathMigration(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "legacy.json")
	target := filepath.Join(dir, "cfg", "knucklebones", saveFileName)

	orig := newSaveData()
	orig.Name = "Wanderer"
	orig.Settings.BestOf = 5
	if err := orig.write(legacy); err != nil {
		t.Fatal(err)
	}

	gotPath := resolveSavePathWith(legacy, target)
	if gotPath != target {
		t.Fatalf("path=%q, want %q", gotPath, target)
	}
	migrated := loadSave(target)
	if migrated.Name != "Wanderer" || migrated.Settings.BestOf != 5 {
		t.Fatalf("migrated data mismatch: %+v", migrated)
	}
}
