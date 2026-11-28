package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsPersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mind.json")

	// Missing file returns defaults.
	def := loadSettings(path)
	if def != defaultSettings() {
		t.Fatalf("missing file returned %+v, want %+v", def, defaultSettings())
	}

	// Corrupt file returns defaults.
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := loadSettings(path); got != defaultSettings() {
		t.Fatalf("corrupt file returned %+v, want %+v", got, defaultSettings())
	}

	// Save writes UTF-8 BOM and round-trips cleanly.
	want := settings{
		Version:      1,
		Theme:        "neon",
		Sound:        false,
		Labels:       true,
		LabelNumbers: true,
		Stats: stats{
			Played:      10,
			Won:         7,
			Streak:      3,
			BestStreak:  5,
			BestTurns:   4,
			BestTimeSec: 27,
		},
	}
	if err := saveSettings(path, want); err != nil {
		t.Fatalf("saveSettings failed: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(raw, utf8BOM) {
		t.Fatalf("saved settings missing UTF-8 BOM: %v", raw[:min(4, len(raw))])
	}
	got := loadSettings(path)
	if got != want {
		t.Errorf("round-trip settings = %+v, want %+v", got, want)
	}

	// Record result updates BestTurns / BestTimeSec only on wins.
	s := defaultSettings()
	s.recordResult(false, 12, 50)
	if s.Stats.Played != 1 || s.Stats.Won != 0 || s.Stats.BestTurns != 0 || s.Stats.BestTimeSec != 0 {
		t.Errorf("after loss, stats = %+v", s.Stats)
	}
	s.recordResult(true, 6, 40)
	s.recordResult(true, 4, 55)
	if s.Stats.Won != 2 || s.Stats.Streak != 2 || s.Stats.BestTurns != 4 || s.Stats.BestTimeSec != 40 {
		t.Errorf("after wins, stats = %+v", s.Stats)
	}
}
