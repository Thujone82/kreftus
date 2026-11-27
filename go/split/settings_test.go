package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsDefaultsAndPersistence(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "settings.json")

	// Legacy JSON without showAudio / showVisual should default both to true.
	legacyJSON := `{"theme":"light","exportMode":"split","skipDropped":false}`
	if err := os.WriteFile(cfgPath, []byte(legacyJSON), 0o644); err != nil {
		t.Fatalf("write legacy settings: %v", err)
	}

	store := NewSettingsStoreAt(cfgPath)
	got := store.Get()
	if got.Theme != "light" || got.ExportMode != "split" || got.SkipDropped {
		t.Fatalf("unexpected legacy load: %+v", got)
	}
	if !got.ShowAudio || !got.ShowVisual {
		t.Fatalf("expected ShowAudio and ShowVisual to default true on legacy settings, got %+v", got)
	}
	if got.TimelineScale != 1.0 {
		t.Fatalf("expected TimelineScale to default to 1.0 on legacy settings, got %v", got.TimelineScale)
	}

	// Explicitly disable both, set a custom scale, and verify persistence across reload.
	store.Update(func(s *Settings) {
		s.ShowAudio = false
		s.ShowVisual = false
		s.TimelineScale = ClampTimelineScale(1.65)
	})

	reloaded := NewSettingsStoreAt(cfgPath).Get()
	if reloaded.ShowAudio || reloaded.ShowVisual {
		t.Fatalf("expected disabled lane preferences to persist, got %+v", reloaded)
	}
	if reloaded.TimelineScale != 1.65 {
		t.Fatalf("expected TimelineScale 1.65 to persist, got %v", reloaded.TimelineScale)
	}
	if ClampTimelineScale(0.1) != 0.55 || ClampTimelineScale(9.0) != 3.2 || ClampTimelineScale(0) != 1.0 {
		t.Fatalf("unexpected ClampTimelineScale behavior")
	}
}
