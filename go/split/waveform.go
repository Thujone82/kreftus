package main

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os/exec"
)

// AudioWaveform stores downsampled perceptual peak and RMS sound levels along the video timeline.
type AudioWaveform struct {
	Ready       bool    `json:"ready"`
	HasAudio    bool    `json:"hasAudio"`
	Bins        int     `json:"bins"`
	BinDuration float64 `json:"binDuration"`
	PeaksB64    string  `json:"peaksB64"`
	RMSB64      string  `json:"rmsB64"`
}

// ExtractAudioWaveform decodes the video's audio track as 8 kHz mono s16le PCM via ffmpeg
// and streams it into compact peak/RMS histogram bins suitable for canvas rendering.
func ExtractAudioWaveform(ctx context.Context, ffmpegPath, filePath string, durationSec float64, hasAudio bool) (*AudioWaveform, error) {
	if !hasAudio || durationSec <= 0 || ffmpegPath == "" {
		return &AudioWaveform{Ready: true, HasAudio: false}, nil
	}

	const sampleRate = 8000
	targetBins := int(math.Round(durationSec * 20.0))
	if targetBins < 200 {
		targetBins = 200
	}
	if targetBins > 24000 {
		targetBins = 24000
	}

	samplesPerBin := int(math.Round((durationSec * float64(sampleRate)) / float64(targetBins)))
	if samplesPerBin < 1 {
		samplesPerBin = 1
	}

	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-hide_banner", "-loglevel", "error",
		"-i", filePath,
		"-vn",
		"-ac", "1",
		"-ar", "8000",
		"-f", "s16le",
		"pipe:1",
	)
	hideSubprocessWindow(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start audio waveform extraction: %w", err)
	}

	rawPeaks := make([]float64, 0, targetBins)
	rawRMS := make([]float64, 0, targetBins)

	buf := make([]byte, 64*1024)
	var leftover byte
	hasLeftover := false

	var binPeak float64
	var binSumSq float64
	var binCount int

	flushBin := func() {
		if binCount == 0 {
			return
		}
		rawPeaks = append(rawPeaks, binPeak)
		rawRMS = append(rawRMS, math.Sqrt(binSumSq/float64(binCount)))
		binPeak = 0
		binSumSq = 0
		binCount = 0
	}

	var readErr error
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			data := buf[:n]
			idx := 0
			if hasLeftover {
				sample := int16(binary.LittleEndian.Uint16([]byte{leftover, data[0]}))
				v := math.Abs(float64(sample)) / 32768.0
				if v > binPeak {
					binPeak = v
				}
				binSumSq += v * v
				binCount++
				if binCount >= samplesPerBin {
					flushBin()
				}
				hasLeftover = false
				idx = 1
			}
			for idx+1 < len(data) {
				sample := int16(binary.LittleEndian.Uint16(data[idx : idx+2]))
				v := math.Abs(float64(sample)) / 32768.0
				if v > binPeak {
					binPeak = v
				}
				binSumSq += v * v
				binCount++
				if binCount >= samplesPerBin {
					flushBin()
				}
				idx += 2
			}
			if idx < len(data) {
				leftover = data[idx]
				hasLeftover = true
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			readErr = err
			break
		}
	}
	flushBin()

	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if readErr != nil {
		return nil, readErr
	}
	if waitErr != nil && len(rawPeaks) == 0 {
		return nil, fmt.Errorf("ffmpeg audio waveform decode failed: %w", waitErr)
	}
	if len(rawPeaks) == 0 {
		return &AudioWaveform{Ready: true, HasAudio: false}, nil
	}

	// Normalize length to targetBins if audio stream ended slightly early or late
	if len(rawPeaks) > targetBins {
		rawPeaks = rawPeaks[:targetBins]
		rawRMS = rawRMS[:targetBins]
	} else {
		for len(rawPeaks) < targetBins {
			rawPeaks = append(rawPeaks, 0)
			rawRMS = append(rawRMS, 0)
		}
	}

	maxPeak := 0.0
	for _, p := range rawPeaks {
		if p > maxPeak {
			maxPeak = p
		}
	}
	ref := math.Max(0.02, maxPeak)

	peaksBytes := make([]byte, len(rawPeaks))
	rmsBytes := make([]byte, len(rawRMS))
	for i := range rawPeaks {
		peaksBytes[i] = perceptualLevelByte(rawPeaks[i], ref)
		rmsBytes[i] = perceptualLevelByte(rawRMS[i], ref)
	}

	return &AudioWaveform{
		Ready:       true,
		HasAudio:    true,
		Bins:        len(peaksBytes),
		BinDuration: durationSec / float64(len(peaksBytes)),
		PeaksB64:    base64.StdEncoding.EncodeToString(peaksBytes),
		RMSB64:      base64.StdEncoding.EncodeToString(rmsBytes),
	}, nil
}

func perceptualLevelByte(val, ref float64) byte {
	if val < 0.0015 || ref <= 0 {
		return 0
	}
	norm := val / ref
	if norm < 0 {
		norm = 0
	} else if norm > 1 {
		norm = 1
	}
	curved := math.Pow(norm, 0.58)
	b := math.Round(curved * 255.0)
	if b < 0 {
		return 0
	}
	if b > 255 {
		return 255
	}
	return byte(b)
}
