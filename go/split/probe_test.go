package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractKeyframesIgnoresSpikeAndUnwrapsReset(t *testing.T) {
	csvData := strings.Join([]string{
		"video,0,0.000000,0.000000,1000,K__",
		"audio,1,0.500000,0.500000,200,K__",
		"video,2,1.000000,1.000000,900,K__", // secondary stream (e.g. cover art) should be ignored
		"video,0,2.500000,2.500000,1200,K__",
		"video,0,5.000000,5.000000,1100,___",
		"video,0,99999.000000,99999.000000,500,K__", // corrupt PTS spike must not truncate following KFs
		"video,0,6.000000,6.000000,1200,K__",
		"video,0,10.000000,10.000000,1200,K__",
		"video,0,100.000000,100.000000,1200,K__", // advance to 100s
		"video,0,0.000000,0.000000,1300,K__",     // backward PTS reset to 0 (>30s drop)
		"video,0,0.033333,0.033333,400,___",      // second packet confirms new timeline
		"video,0,5.000000,5.000000,1200,K__",     // 105s unwrapped
		"video,0,15.000000,15.000000,1200,K__",   // 115s unwrapped
	}, "\n")

	var gotProgress bool
	var lastKFReported float64
	recs, cumBytes, err := extractKeyframeRecordsFromCSV(strings.NewReader(csvData), 0, 0, 120.0, 30.0, 20000, func(pct float64, kfs int, latestKF float64, phase string) {
		gotProgress = true
		if latestKF > lastKFReported {
			lastKFReported = latestKF
		}
	})
	_ = gotProgress
	_ = lastKFReported
	if err != nil {
		t.Fatalf("extractKeyframeRecordsFromCSV err: %v", err)
	}
	if cumBytes <= 0 {
		t.Fatalf("expected positive cumBytes, got %d", cumBytes)
	}

	hasNear := func(target float64) bool {
		for _, r := range recs {
			diff := r.RelSec - target
			if diff < 0 {
				diff = -diff
			}
			if diff <= 0.15 {
				return true
			}
		}
		return false
	}

	for _, want := range []float64{0.0, 2.5, 6.0, 10.0, 100.0, 105.0, 115.0} {
		if !hasNear(want) {
			t.Fatalf("expected keyframe near %.1fs in %+v", want, recs)
		}
	}
	for _, r := range recs {
		if r.RelSec > 130 {
			t.Fatalf("unexpected outlier keyframe leaked through: %+v", r)
		}
	}
}

func TestProbeMediaConcatTSWithPTSReset(t *testing.T) {
	tools := FindMediaTools("", "")
	if !tools.Ready {
		t.Skip("ffmpeg/ffprobe not available")
	}

	dir := t.TempDir()
	seg1 := filepath.Join(dir, "seg1.ts")
	seg2 := filepath.Join(dir, "seg2.ts")
	joined := filepath.Join(dir, "joined.ts")

	for i, out := range []string{seg1, seg2} {
		offset := "0"
		if i == 0 {
			offset = "120" // 120s first segment -> hard backward PTS jump to 0s at seg2
		}
		cmd := exec.Command(tools.FFmpegPath,
			"-hide_banner", "-loglevel", "error", "-y",
			"-f", "lavfi", "-i", "testsrc=size=160x120:rate=15:duration=2",
			"-output_ts_offset", offset,
			"-c:v", "libx264", "-g", "15", "-pix_fmt", "yuv420p",
			out,
		)
		hideSubprocessWindow(cmd)
		if res, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("failed to generate ts seg %d: %v (%s)", i, err, string(res))
		}
	}

	b1, err := os.ReadFile(seg1)
	if err != nil {
		t.Fatalf("read seg1: %v", err)
	}
	b2, err := os.ReadFile(seg2)
	if err != nil {
		t.Fatalf("read seg2: %v", err)
	}
	if err := os.WriteFile(joined, append(b1, b2...), 0o644); err != nil {
		t.Fatalf("write joined.ts: %v", err)
	}

	media, err := ProbeMedia(context.Background(), tools.FFprobePath, joined)
	if err != nil {
		t.Fatalf("ProbeMedia(joined.ts) err: %v", err)
	}
	if len(media.Keyframes) < 4 {
		t.Fatalf("expected keyframes from both joined TS segments (>=4), got %v", media.Keyframes)
	}
	lastKF := media.Keyframes[len(media.Keyframes)-1]
	if lastKF < 2.5 {
		t.Fatalf("expected unwrapped keyframes extending past 2.5s into second segment (dur=%.2fs), got %v", media.DurationSec, media.Keyframes)
	}
}

func TestExtractProbeBackdropDataURI(t *testing.T) {
	tools := FindMediaTools("", "")
	if !tools.Ready {
		t.Skip("ffmpeg/ffprobe not available")
	}

	outPath := filepath.Join(t.TempDir(), "backdrop_test.mp4")
	cmd := exec.Command(tools.FFmpegPath,
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "color=c=blue:s=320x180:r=15:d=2",
		"-f", "lavfi", "-i", "color=c=green:s=320x180:r=15:d=2",
		"-filter_complex", "[0:v][1:v]concat=n=2:v=1:a=0[v]",
		"-map", "[v]",
		"-c:v", "libx264", "-g", "30", "-keyint_min", "30", "-sc_threshold", "0", "-pix_fmt", "yuv420p",
		outPath,
	)
	hideSubprocessWindow(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to generate backdrop test video: %v (%s)", err, string(out))
	}

	checkColor := func(kfSec float64, expectGreen bool) {
		uri, err := ExtractProbeBackdropDataURI(context.Background(), tools.FFmpegPath, outPath, kfSec)
		if err != nil {
			t.Fatalf("ExtractProbeBackdropDataURI(%.1fs) err: %v", kfSec, err)
		}
		const prefix = "data:image/jpeg;base64,"
		if !strings.HasPrefix(uri, prefix) {
			t.Fatalf("expected data:image/jpeg;base64 prefix, got %q", uri[:min(len(uri), 40)])
		}
		raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(uri, prefix))
		if err != nil {
			t.Fatalf("decode backdrop base64: %v", err)
		}
		img, err := jpeg.Decode(bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("decode backdrop jpeg: %v", err)
		}
		b := img.Bounds()
		r, g, bl, _ := img.At(b.Dx()/2, b.Dy()/2).RGBA()
		r8, g8, b8 := r>>8, g>>8, bl>>8
		if expectGreen {
			if g8 <= r8+50 || g8 <= b8+50 {
				t.Fatalf("expected green frame at %.1fs, got R=%d G=%d B=%d", kfSec, r8, g8, b8)
			}
		} else {
			if b8 <= r8+50 || b8 <= g8+50 {
				t.Fatalf("expected blue frame at %.1fs, got R=%d G=%d B=%d", kfSec, r8, g8, b8)
			}
		}
	}

	checkColor(0.0, false)
	checkColor(2.0, true)
}
