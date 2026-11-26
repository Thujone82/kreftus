package main

import (
	"context"
	"math"
	"os/exec"
	"path/filepath"
	"testing"
)

func createSyntheticTestVideo(t *testing.T, tools ToolStatus, name string) string {
	t.Helper()
	outPath := filepath.Join(t.TempDir(), name)
	cmd := exec.Command(tools.FFmpegPath,
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=duration=6:size=320x240:rate=30",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=6",
		"-g", "30", "-keyint_min", "30", "-sc_threshold", "0",
		"-c:v", "libx264", "-pix_fmt", "yuv420p",
		"-c:a", "aac",
		outPath,
	)
	hideSubprocessWindow(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to generate synthetic test video: %v (%s)", err, string(out))
	}
	return outPath
}

func TestProbeAndLosslessExportSplitAndJoin(t *testing.T) {
	tools := FindMediaTools("", "")
	if !tools.Ready {
		t.Skip("ffmpeg/ffprobe not available")
	}

	ctx := context.Background()
	srcPath := createSyntheticTestVideo(t, tools, "sample.mp4")

	media, err := ProbeMedia(ctx, tools.FFprobePath, srcPath)
	if err != nil {
		t.Fatalf("ProbeMedia err: %v", err)
	}
	if math.Abs(media.DurationSec-6.0) > 0.25 {
		t.Fatalf("DurationSec = %v, want ~6.0", media.DurationSec)
	}
	if len(media.Keyframes) < 5 {
		t.Fatalf("expected >= 5 keyframes, got %v", media.Keyframes)
	}

	// Build project: keep [0..2], drop [2..4], keep [4..6]
	proj := NewProject(srcPath, media, "split")
	proj.AddMarkerAt(2.1) // snaps to 2.0
	proj.AddMarkerAt(3.9) // snaps to 4.0
	proj.ToggleSegment(1) // drop middle [2..4]

	kept := proj.KeptSegments()
	if len(kept) != 2 {
		t.Fatalf("kept count = %d, want 2", len(kept))
	}

	// Test Split export -> sample_01.mp4, sample_02.mp4
	splitTargets := DeriveSplitPaths(SuggestedSplitDialogPath(srcPath), len(kept))
	splitRes, err := ExportSplit(ctx, tools, proj, splitTargets, nil)
	if err != nil {
		t.Fatalf("ExportSplit err: %v", err)
	}
	if len(splitRes.Files) != 2 {
		t.Fatalf("split file count = %d, want 2", len(splitRes.Files))
	}
	for _, f := range splitRes.Files {
		if f.Dur < 1.7 || f.Dur > 2.3 {
			t.Fatalf("split clip %s duration = %.2fs, expected ~2.0s", f.Name, f.Dur)
		}
	}

	// Test Join export -> sample_edit.mp4 (~4.0s)
	joinTarget := SuggestedJoinPath(srcPath)
	joinRes, err := ExportJoin(ctx, tools, proj, joinTarget, nil)
	if err != nil {
		t.Fatalf("ExportJoin err: %v", err)
	}
	if len(joinRes.Files) != 1 {
		t.Fatalf("join file count = %d, want 1", len(joinRes.Files))
	}
	if joinRes.Files[0].Dur < 3.6 || joinRes.Files[0].Dur > 4.4 {
		t.Fatalf("joined clip duration = %.2fs, expected ~4.0s", joinRes.Files[0].Dur)
	}
}
