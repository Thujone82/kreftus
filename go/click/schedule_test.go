package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestScheduleTimezoneLocalDisplayAndEvaluation(t *testing.T) {
	// Construct a schedule via JSON as if received from the frontend with UTC ISO format
	// 2026-11-04T10:00:00Z corresponds to 02:00:00 in PST (-0800).
	const rawJSON = `{
		"id": "sched_tz_1",
		"macroId": "m1",
		"name": "Macro 110326 01",
		"enabled": true,
		"start": "2026-11-04T10:00:00Z",
		"repeat": "none",
		"playsPerRun": 1,
		"missed": "skip",
		"overlap": "skip",
		"end": "never"
	}`

	var sc Schedule
	if err := json.Unmarshal([]byte(rawJSON), &sc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// sc.Start was unmarshaled with UTC location
	if sc.Start.Location() != time.UTC {
		t.Fatalf("expected UTC unmarshal, got %v", sc.Start.Location())
	}

	// Verify DescribeSchedule formats in time.Local
	expectedLocalStart := sc.Start.Local()
	expectedDesc := "Once at " + expectedLocalStart.Format("Jan 2, 3:04 PM")
	desc := DescribeSchedule(sc)
	if desc != expectedDesc {
		t.Fatalf("DescribeSchedule got %q, want %q", desc, expectedDesc)
	}

	// Verify ComputeNextRun returns a timestamp in time.Local
	now := expectedLocalStart.Add(-1 * time.Hour)
	next := ComputeNextRun(sc, now)
	if next == nil {
		t.Fatalf("ComputeNextRun returned nil")
	}
	if next.Location() != time.Local {
		t.Fatalf("ComputeNextRun expected time.Local location, got %v", next.Location())
	}
	if next.Hour() != expectedLocalStart.Hour() || next.Minute() != expectedLocalStart.Minute() {
		t.Fatalf("ComputeNextRun time mismatch: got %v, want %v", next, expectedLocalStart)
	}

	// Verify PreviewNextRuns returns local timestamps
	preview := PreviewNextRuns(sc, now, 5)
	if len(preview) != 1 {
		t.Fatalf("expected 1 run, got %d", len(preview))
	}
	if preview[0].Location() != time.Local {
		t.Fatalf("preview location: got %v, want time.Local", preview[0].Location())
	}
	if preview[0].Hour() != expectedLocalStart.Hour() {
		t.Fatalf("preview hour: got %d, want %d", preview[0].Hour(), expectedLocalStart.Hour())
	}
}

func TestScheduleDailyLocalEvaluation(t *testing.T) {
	// Schedule set to run daily at 02:00:00 local time
	localTime := time.Date(2026, 11, 4, 2, 0, 0, 0, time.Local)
	// Even if passed as UTC
	utcTime := localTime.UTC()

	sc := Schedule{
		ID:          "daily_sc",
		MacroID:     "m1",
		Enabled:     true,
		Start:       utcTime,
		Repeat:      RepeatDaily,
		PlaysPerRun: 1,
	}

	desc := DescribeSchedule(sc)
	if !strings.Contains(desc, "Daily at "+localTime.Format("3:04 PM")) {
		t.Fatalf("daily describe got %q, expected local time %s", desc, localTime.Format("3:04 PM"))
	}

	// Preview next 3 days
	prev := PreviewNextRuns(sc, localTime.Add(-1*time.Hour), 3)
	if len(prev) != 3 {
		t.Fatalf("expected 3 runs, got %d", len(prev))
	}
	for i, p := range prev {
		if p.Hour() != 2 || p.Minute() != 0 {
			t.Fatalf("run %d got %02d:%02d, want 02:00", i, p.Hour(), p.Minute())
		}
		if p.Location() != time.Local {
			t.Fatalf("run %d location: %v", i, p.Location())
		}
	}
}

func TestStoreNormalizesScheduleToLocal(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	utcTime := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	sc, err := store.UpsertSchedule(Schedule{
		MacroID: "m1",
		Name:    "Test TZ",
		Enabled: true,
		Start:   utcTime,
		Repeat:  RepeatNone,
	}, utcTime.Add(-time.Hour))
	if err != nil {
		t.Fatalf("UpsertSchedule: %v", err)
	}

	if sc.Start.Location() != time.Local {
		t.Fatalf("sc.Start not localized: %v", sc.Start.Location())
	}
	if sc.NextRun != nil && sc.NextRun.Location() != time.Local {
		t.Fatalf("sc.NextRun not localized: %v", sc.NextRun.Location())
	}

	// Reload store from disk
	store2, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore reload: %v", err)
	}
	_, scheds, _, _ := store2.Snapshot()
	if len(scheds) != 1 {
		t.Fatalf("expected 1 schedule on disk, got %d", len(scheds))
	}
	if scheds[0].Start.Location() != time.Local {
		t.Fatalf("reloaded schedule start not localized: %v", scheds[0].Start.Location())
	}
}
