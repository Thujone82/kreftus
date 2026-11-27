package main

import (
	"math"
	"path/filepath"
	"testing"
)

func TestSnapAndMarkers(t *testing.T) {
	media := &MediaInfo{
		FilePath:    `C:\videos\demo.mp4`,
		DurationSec: 10.0,
		Keyframes:   []float64{0.0, 2.0, 4.0, 6.0, 8.0},
	}
	proj := NewProject(media.FilePath, media, "join")

	// Snap 3.2 -> 4.0
	snapped, added := proj.AddMarkerAt(3.2)
	if !added || math.Abs(snapped-4.0) > 1e-6 {
		t.Fatalf("AddMarkerAt(3.2) = (%v, %v), want (4.0, true)", snapped, added)
	}
	// Duplicate on same keyframe should not add
	if _, added2 := proj.AddMarkerAt(4.1); added2 {
		t.Fatalf("expected duplicate marker at 4.0 to be rejected")
	}

	// Add second marker at 7.6 -> 8.0
	snapped2, added2 := proj.AddMarkerAt(7.6)
	if !added2 || math.Abs(snapped2-8.0) > 1e-6 {
		t.Fatalf("AddMarkerAt(7.6) = (%v, %v), want (8.0, true)", snapped2, added2)
	}

	segs := proj.Segments()
	if len(segs) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(segs))
	}
	// Toggle middle segment to Drop
	proj.ToggleSegment(1)
	kept := proj.KeptSegments()
	if len(kept) != 2 || kept[0].End != 4.0 || kept[1].Start != 8.0 {
		t.Fatalf("unexpected kept segments: %+v", kept)
	}

	// Move marker 0 from 4.0 to 2.1 -> 2.0
	movedTo, ok := proj.MoveMarker(0, 2.1)
	if !ok || math.Abs(movedTo-2.0) > 1e-6 {
		t.Fatalf("MoveMarker(0, 2.1) = (%v, %v), want (2.0, true)", movedTo, ok)
	}

	// Remove marker 0 -> merges seg 0 (Keep=true) and seg 1 (Keep=false) -> Keep=true
	if !proj.RemoveMarker(0) {
		t.Fatalf("RemoveMarker(0) failed")
	}
	if len(proj.Segments()) != 2 || !proj.KeepFlags[0] {
		t.Fatalf("unexpected segments after merge: %+v", proj.Segments())
	}
}

func TestUpdateMediaPreservingCuts(t *testing.T) {
	media1 := &MediaInfo{
		FilePath:    `C:\videos\demo.mp4`,
		DurationSec: 12.0,
		Keyframes:   []float64{0.0, 2.0, 4.0, 6.0, 8.0, 10.0},
	}
	proj := NewProject(media1.FilePath, media1, "split")
	proj.AddMarkerAt(4.0)
	proj.AddMarkerAt(8.0)
	proj.ToggleSegment(1) // Drop 4..8s, Keep 0..4s & 8..12s

	// Simulate keyframe refresh returning a denser set of keyframes (including 4.0 and 8.0)
	media2 := &MediaInfo{
		FilePath:    media1.FilePath,
		DurationSec: 12.0,
		Keyframes:   []float64{0.0, 1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0, 11.0},
	}
	proj.UpdateMediaPreservingCuts(media2)

	if len(proj.Media.Keyframes) != 12 {
		t.Fatalf("expected updated keyframes len 12, got %d", len(proj.Media.Keyframes))
	}
	if len(proj.Markers) != 2 || math.Abs(proj.Markers[0]-4.0) > 1e-6 || math.Abs(proj.Markers[1]-8.0) > 1e-6 {
		t.Fatalf("expected markers [4.0, 8.0] preserved, got %v", proj.Markers)
	}
	segs := proj.Segments()
	if len(segs) != 3 || !segs[0].Keep || segs[1].Keep || !segs[2].Keep {
		t.Fatalf("expected [Keep, Drop, Keep] preserved, got %+v", segs)
	}
}

func TestOutputNaming(t *testing.T) {
	src := filepath.Join(`C:\clips`, `holiday.mp4`)
	if got := SuggestedJoinPath(src); got != filepath.Join(`C:\clips`, `holiday_edit.mp4`) {
		t.Fatalf("SuggestedJoinPath = %q", got)
	}
	defSplit := SuggestedSplitDialogPath(src)
	if defSplit != filepath.Join(`C:\clips`, `holiday_01.mp4`) {
		t.Fatalf("SuggestedSplitDialogPath = %q", defSplit)
	}

	paths := DeriveSplitPaths(defSplit, 3)
	want := []string{
		filepath.Join(`C:\clips`, `holiday_01.mp4`),
		filepath.Join(`C:\clips`, `holiday_02.mp4`),
		filepath.Join(`C:\clips`, `holiday_03.mp4`),
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("paths[%d] = %q, want %q", i, paths[i], want[i])
		}
	}
}

func TestComputeBalancedCutKeyframes(t *testing.T) {
	var records []KeyframeRecord
	for i := 0; i < 10; i++ {
		records = append(records, KeyframeRecord{
			RelSec:   float64(i),
			CumBytes: int64(i * 100000),
		})
	}
	cuts, err := ComputeBalancedCutKeyframes(records, 1000000, 2)
	if err != nil {
		t.Fatalf("ComputeBalancedCutKeyframes err: %v", err)
	}
	if len(cuts) != 1 || cuts[0] != 5 {
		t.Fatalf("cuts = %v, want [5]", cuts)
	}
}
