package main

import (
	"context"
	"encoding/base64"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExtractVisualDeltaSceneCutsAndMotion(t *testing.T) {
	tools := FindMediaTools("", "")
	if !tools.Ready {
		t.Skip("ffmpeg/ffprobe not available")
	}

	outPath := filepath.Join(t.TempDir(), "visual_test.mp4")
	cmd := exec.Command(tools.FFmpegPath,
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "color=c=blue:s=160x120:r=15:d=2",
		"-f", "lavfi", "-i", "testsrc2=s=160x120:r=15:d=2",
		"-f", "lavfi", "-i", "color=c=red:s=160x120:r=15:d=2",
		"-filter_complex", "[0:v][1:v][2:v]concat=n=3:v=1:a=0[v]",
		"-map", "[v]",
		"-c:v", "libx264", "-pix_fmt", "yuv420p",
		outPath,
	)
	hideSubprocessWindow(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to generate visual test video: %v (%s)", err, string(out))
	}

	ctx := context.Background()
	media, err := ProbeMedia(ctx, tools.FFprobePath, outPath)
	if err != nil {
		t.Fatalf("ProbeMedia err: %v", err)
	}

	vd, err := ExtractVisualDelta(ctx, tools.FFmpegPath, media)
	if err != nil {
		t.Fatalf("ExtractVisualDelta err: %v", err)
	}
	if !vd.Ready || vd.Bins < 20 {
		t.Fatalf("unexpected VisualDelta metadata: %+v", vd)
	}

	deltas, err := base64.StdEncoding.DecodeString(vd.DeltaB64)
	if err != nil {
		t.Fatalf("decode delta: %v", err)
	}
	scenes, err := base64.StdEncoding.DecodeString(vd.SceneB64)
	if err != nil {
		t.Fatalf("decode scene: %v", err)
	}
	if len(deltas) != vd.Bins || len(scenes) != vd.Bins {
		t.Fatalf("length mismatch: deltas=%d scenes=%d bins=%d", len(deltas), len(scenes), vd.Bins)
	}

	meanRange := func(arr []byte, startSec, endSec float64) float64 {
		i0 := int(startSec / vd.BinDuration)
		i1 := int(endSec / vd.BinDuration)
		if i0 < 0 {
			i0 = 0
		}
		if i1 > len(arr) {
			i1 = len(arr)
		}
		var sum float64
		for i := i0; i < i1; i++ {
			sum += float64(arr[i])
		}
		return sum / float64(i1-i0)
	}

	maxRange := func(arr []byte, startSec, endSec float64) byte {
		i0 := int(startSec / vd.BinDuration)
		i1 := int(endSec / vd.BinDuration)
		if i0 < 0 {
			i0 = 0
		}
		if i1 > len(arr) {
			i1 = len(arr)
		}
		var mx byte
		for i := i0; i < i1; i++ {
			if arr[i] > mx {
				mx = arr[i]
			}
		}
		return mx
	}

	staticBlue := meanRange(deltas, 0.3, 1.7)
	movingMid := meanRange(deltas, 2.3, 3.7)
	staticRed := meanRange(deltas, 4.3, 5.7)

	if staticBlue > 10 {
		t.Fatalf("expected static blue mean delta <= 10, got %.1f", staticBlue)
	}
	if movingMid <= 25 {
		t.Fatalf("expected moving middle mean delta > 25, got %.1f", movingMid)
	}
	if staticRed > 10 {
		t.Fatalf("expected static red mean delta <= 10, got %.1f", staticRed)
	}

	cut1Delta := maxRange(deltas, 1.8, 2.3)
	cut1Scene := maxRange(scenes, 1.8, 2.3)
	cut2Delta := maxRange(deltas, 3.8, 4.3)
	cut2Scene := maxRange(scenes, 3.8, 4.3)

	if cut1Delta <= 170 || cut1Scene <= 170 {
		t.Fatalf("expected hard scene cut 1 spike > 170, got delta=%d scene=%d", cut1Delta, cut1Scene)
	}
	if cut2Delta <= 170 || cut2Scene <= 170 {
		t.Fatalf("expected hard scene cut 2 spike > 170, got delta=%d scene=%d", cut2Delta, cut2Scene)
	}
}

func TestExtractVisualDeltaMultiChunk(t *testing.T) {
	tools := FindMediaTools("", "")
	if !tools.Ready {
		t.Skip("ffmpeg/ffprobe not available")
	}

	outPath := filepath.Join(t.TempDir(), "visual_multichunk.mp4")
	cmd := exec.Command(tools.FFmpegPath,
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "color=c=blue:s=160x120:r=15:d=2",
		"-f", "lavfi", "-i", "testsrc2=s=160x120:r=15:d=2",
		"-f", "lavfi", "-i", "color=c=red:s=160x120:r=15:d=2",
		"-filter_complex", "[0:v][1:v][2:v]concat=n=3:v=1:a=0[v]",
		"-map", "[v]",
		"-c:v", "libx264", "-pix_fmt", "yuv420p",
		outPath,
	)
	hideSubprocessWindow(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to generate multichunk visual test video: %v (%s)", err, string(out))
	}

	ctx := context.Background()
	media, err := ProbeMedia(ctx, tools.FFprobePath, outPath)
	if err != nil {
		t.Fatalf("ProbeMedia err: %v", err)
	}

	vd, err := extractVisualDeltaChunks(ctx, tools.FFmpegPath, media, 3, nil)
	if err != nil {
		t.Fatalf("extractVisualDeltaChunks err: %v", err)
	}
	if !vd.Ready || vd.Bins < 20 {
		t.Fatalf("unexpected VisualDelta metadata: %+v", vd)
	}

	deltas, err := base64.StdEncoding.DecodeString(vd.DeltaB64)
	if err != nil {
		t.Fatalf("decode delta: %v", err)
	}
	scenes, err := base64.StdEncoding.DecodeString(vd.SceneB64)
	if err != nil {
		t.Fatalf("decode scene: %v", err)
	}

	meanRange := func(arr []byte, startSec, endSec float64) float64 {
		i0 := int(startSec / vd.BinDuration)
		i1 := int(endSec / vd.BinDuration)
		if i0 < 0 {
			i0 = 0
		}
		if i1 > len(arr) {
			i1 = len(arr)
		}
		var sum float64
		for i := i0; i < i1; i++ {
			sum += float64(arr[i])
		}
		return sum / float64(i1-i0)
	}

	maxRange := func(arr []byte, startSec, endSec float64) byte {
		i0 := int(startSec / vd.BinDuration)
		i1 := int(endSec / vd.BinDuration)
		if i0 < 0 {
			i0 = 0
		}
		if i1 > len(arr) {
			i1 = len(arr)
		}
		var mx byte
		for i := i0; i < i1; i++ {
			if arr[i] > mx {
				mx = arr[i]
			}
		}
		return mx
	}

	if sb := meanRange(deltas, 0.3, 1.7); sb > 10 {
		t.Fatalf("expected static blue mean delta <= 10, got %.1f", sb)
	}
	if mm := meanRange(deltas, 2.3, 3.7); mm <= 25 {
		t.Fatalf("expected moving middle mean delta > 25, got %.1f", mm)
	}
	if sr := meanRange(deltas, 4.3, 5.7); sr > 10 {
		t.Fatalf("expected static red mean delta <= 10, got %.1f", sr)
	}
	if c1 := maxRange(scenes, 1.75, 2.35); c1 <= 150 {
		t.Fatalf("expected scene cut 1 across chunk boundary > 150, got %d", c1)
	}
	if c2 := maxRange(scenes, 3.75, 4.35); c2 <= 150 {
		t.Fatalf("expected scene cut 2 across chunk boundary > 150, got %d", c2)
	}
}
