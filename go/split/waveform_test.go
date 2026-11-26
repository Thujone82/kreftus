package main

import (
	"context"
	"encoding/base64"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExtractAudioWaveformSilenceWindow(t *testing.T) {
	tools := FindMediaTools("", "")
	if !tools.Ready {
		t.Skip("ffmpeg/ffprobe not available")
	}

	outPath := filepath.Join(t.TempDir(), "wave_test.mp4")
	cmd := exec.Command(tools.FFmpegPath,
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=duration=6:size=160x120:rate=15",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=6,volume='if(between(t,2,4),0,1)':eval=frame",
		"-c:v", "libx264", "-pix_fmt", "yuv420p",
		"-c:a", "aac",
		outPath,
	)
	hideSubprocessWindow(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to generate audio waveform test video: %v (%s)", err, string(out))
	}

	ctx := context.Background()
	wf, err := ExtractAudioWaveform(ctx, tools.FFmpegPath, outPath, 6.0, true)
	if err != nil {
		t.Fatalf("ExtractAudioWaveform err: %v", err)
	}
	if !wf.Ready || !wf.HasAudio || wf.Bins < 100 {
		t.Fatalf("unexpected waveform metadata: %+v", wf)
	}

	peaks, err := base64.StdEncoding.DecodeString(wf.PeaksB64)
	if err != nil {
		t.Fatalf("decode peaks: %v", err)
	}
	rms, err := base64.StdEncoding.DecodeString(wf.RMSB64)
	if err != nil {
		t.Fatalf("decode rms: %v", err)
	}
	if len(peaks) != wf.Bins || len(rms) != wf.Bins {
		t.Fatalf("length mismatch: peaks=%d rms=%d bins=%d", len(peaks), len(rms), wf.Bins)
	}

	meanPeak := func(startSec, endSec float64) float64 {
		i0 := int(startSec / wf.BinDuration)
		i1 := int(endSec / wf.BinDuration)
		if i0 < 0 {
			i0 = 0
		}
		if i1 > len(peaks) {
			i1 = len(peaks)
		}
		var sum float64
		for i := i0; i < i1; i++ {
			sum += float64(peaks[i])
		}
		return sum / float64(i1-i0)
	}

	pStart := meanPeak(0.3, 1.7)
	pSilent := meanPeak(2.3, 3.7)
	pEnd := meanPeak(4.3, 5.7)

	if pStart <= 150 {
		t.Fatalf("expected loud start peak > 150, got %.1f", pStart)
	}
	if pSilent > 5 {
		t.Fatalf("expected silent middle peak <= 5, got %.1f", pSilent)
	}
	if pEnd <= 150 {
		t.Fatalf("expected loud end peak > 150, got %.1f", pEnd)
	}
}
