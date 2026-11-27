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

func TestAudioTalkingAndNoiseClassification(t *testing.T) {
	tools := FindMediaTools("", "")
	if !tools.Ready {
		t.Skip("ffmpeg/ffprobe not available")
	}

	outPath := filepath.Join(t.TempDir(), "voice_noise_test.mp4")
	// [0..2s] vocal-band formant tones (~450Hz + 1100Hz), [2..4s] silence, [4..6s] high-passed (>3500Hz) white noise
	cmd := exec.Command(tools.FFmpegPath,
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=duration=6:size=160x120:rate=15",
		"-f", "lavfi", "-i", "aevalsrc='0.35*sin(2*PI*450*t)+0.25*sin(2*PI*1100*t)':s=8000:d=2",
		"-f", "lavfi", "-i", "anullsrc=r=8000:cl=mono:d=2",
		"-f", "lavfi", "-i", "anoisesrc=d=2:c=white:r=8000:a=0.5,highpass=f=2400",
		"-filter_complex", "[1:a][2:a][3:a]concat=n=3:v=0:a=1[a]",
		"-map", "0:v", "-map", "[a]",
		"-c:v", "libx264", "-pix_fmt", "yuv420p",
		"-c:a", "aac",
		outPath,
	)
	hideSubprocessWindow(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to generate voice/noise test video: %v (%s)", err, string(out))
	}

	var gotProgress bool
	wf, err := ExtractAudioWaveformWithProgress(context.Background(), tools.FFmpegPath, outPath, 6.0, true, func(pct float64) {
		if pct > 0 {
			gotProgress = true
		}
	})
	if err != nil {
		t.Fatalf("ExtractAudioWaveformWithProgress err: %v", err)
	}
	if !gotProgress {
		t.Fatalf("expected progress callback to fire")
	}
	act, err := base64.StdEncoding.DecodeString(wf.ActivityB64)
	if err != nil {
		t.Fatalf("decode activityB64: %v", err)
	}
	if len(act) != wf.Bins {
		t.Fatalf("activity length = %d, want %d", len(act), wf.Bins)
	}

	majorityClass := func(startSec, endSec float64) byte {
		i0 := int(startSec / wf.BinDuration)
		i1 := int(endSec / wf.BinDuration)
		if i0 < 0 {
			i0 = 0
		}
		if i1 > len(act) {
			i1 = len(act)
		}
		var counts [3]int
		for i := i0; i < i1; i++ {
			if act[i] < 3 {
				counts[act[i]]++
			}
		}
		best := byte(0)
		for c := byte(1); c < 3; c++ {
			if counts[c] > counts[best] {
				best = c
			}
		}
		return best
	}

	if got := majorityClass(0.3, 1.7); got != 1 {
		t.Fatalf("expected vocal window [0.3..1.7s] majority class 1 (talking), got %d", got)
	}
	if got := majorityClass(2.3, 3.7); got != 0 {
		t.Fatalf("expected silence window [2.3..3.7s] majority class 0 (quiet), got %d", got)
	}
	if got := majorityClass(4.3, 5.7); got != 2 {
		t.Fatalf("expected highpass noise window [4.3..5.7s] majority class 2 (noise), got %d", got)
	}
}

func TestExtractAudioWaveformMultiChunk(t *testing.T) {
	tools := FindMediaTools("", "")
	if !tools.Ready {
		t.Skip("ffmpeg/ffprobe not available")
	}

	outPath := filepath.Join(t.TempDir(), "wave_multichunk.mp4")
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
		t.Fatalf("failed to generate multichunk audio test video: %v (%s)", err, string(out))
	}

	wf, err := extractAudioWaveformChunks(context.Background(), tools.FFmpegPath, outPath, 6.0, true, 4, nil)
	if err != nil {
		t.Fatalf("extractAudioWaveformChunks err: %v", err)
	}
	if !wf.Ready || !wf.HasAudio || wf.Bins < 100 {
		t.Fatalf("unexpected multichunk waveform metadata: %+v", wf)
	}

	peaks, err := base64.StdEncoding.DecodeString(wf.PeaksB64)
	if err != nil {
		t.Fatalf("decode peaks: %v", err)
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

	if p := meanPeak(0.3, 1.7); p <= 150 {
		t.Fatalf("expected loud start peak > 150, got %.1f", p)
	}
	if p := meanPeak(2.3, 3.7); p > 10 {
		t.Fatalf("expected silent middle peak <= 10, got %.1f", p)
	}
	if p := meanPeak(4.3, 5.7); p <= 150 {
		t.Fatalf("expected loud end peak > 150 across final chunk, got %.1f", p)
	}
}
