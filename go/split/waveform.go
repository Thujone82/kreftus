package main

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os/exec"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// AnalysisProgressFunc reports 0..100 completion percentage during background audio/visual analysis.
type AnalysisProgressFunc func(pct float64)

// AudioWaveform stores downsampled perceptual peak, RMS sound levels, and acoustic activity
// classification (0 = quiet/ambient, 1 = talking/speech, 2 = noise/transient) along the timeline.
type AudioWaveform struct {
	Ready       bool    `json:"ready"`
	HasAudio    bool    `json:"hasAudio"`
	Bins        int     `json:"bins"`
	BinDuration float64 `json:"binDuration"`
	PeaksB64    string  `json:"peaksB64"`
	RMSB64      string  `json:"rmsB64"`
	ActivityB64 string  `json:"activityB64"`
}

// ExtractAudioWaveform decodes the video's audio track as 8 kHz mono s16le PCM via ffmpeg
// and streams it into compact peak/RMS/activity histogram bins.
func ExtractAudioWaveform(ctx context.Context, ffmpegPath, filePath string, durationSec float64, hasAudio bool) (*AudioWaveform, error) {
	return ExtractAudioWaveformWithProgress(ctx, ffmpegPath, filePath, durationSec, hasAudio, nil)
}

// ExtractAudioWaveformWithProgress is like ExtractAudioWaveform but reports streaming percentage updates
// while decoding audio in parallel PTS-locked chunks across multiple CPU workers.
func ExtractAudioWaveformWithProgress(ctx context.Context, ffmpegPath, filePath string, durationSec float64, hasAudio bool, onProgress AnalysisProgressFunc) (*AudioWaveform, error) {
	return extractAudioWaveformChunks(ctx, ffmpegPath, filePath, durationSec, hasAudio, 0, onProgress)
}

func extractAudioWaveformChunks(ctx context.Context, ffmpegPath, filePath string, durationSec float64, hasAudio bool, forceChunks int, onProgress AnalysisProgressFunc) (*AudioWaveform, error) {
	if !hasAudio || durationSec <= 0 || ffmpegPath == "" {
		return &AudioWaveform{Ready: true, HasAudio: false}, nil
	}

	const sampleRate = 8000
	targetBins := int(math.Round(durationSec * 8.0))
	if targetBins < 160 {
		targetBins = 160
	}
	if targetBins > 4800 {
		targetBins = 4800
	}

	maxWorkers := runtime.NumCPU() / 2
	if maxWorkers < 2 {
		maxWorkers = 2
	}
	if maxWorkers > 6 {
		maxWorkers = 6
	}

	numChunks := 1
	if forceChunks > 0 {
		numChunks = forceChunks
	} else if durationSec >= 20 {
		numChunks = int(math.Ceil(durationSec / 45.0))
		if numChunks < maxWorkers {
			numChunks = maxWorkers
		}
		if numChunks > 16 {
			numChunks = 16
		}
	}
	if numChunks > targetBins {
		numChunks = targetBins
	}
	if numChunks < 1 {
		numChunks = 1
	}

	if onProgress != nil {
		onProgress(2)
	}

	rawPeaks := make([]float64, targetBins)
	rawRMS := make([]float64, targetBins)
	rawBandRMS := make([]float64, targetBins)
	rawZCR := make([]float64, targetBins)

	binDur := durationSec / float64(targetBins)

	var doneBins atomic.Int64
	var totalSamples atomic.Int64
	var progressMu sync.Mutex
	lastEmit := time.Now()

	reportBin := func() {
		done := doneBins.Add(1)
		if onProgress == nil {
			return
		}
		progressMu.Lock()
		if time.Since(lastEmit) >= 65*time.Millisecond {
			pct := math.Min(96.0, 2.0+(float64(done)/float64(targetBins))*94.0)
			onProgress(pct)
			lastEmit = time.Now()
		}
		progressMu.Unlock()
	}

	type chunkJob struct {
		chunkIdx int
		startBin int
		endBin   int
	}
	chunkRanges := make([]chunkJob, 0, numChunks)
	jobs := make(chan chunkJob, numChunks)
	for c := 0; c < numChunks; c++ {
		sBin := (c * targetBins) / numChunks
		eBin := ((c + 1) * targetBins) / numChunks
		if eBin > sBin {
			j := chunkJob{chunkIdx: len(chunkRanges), startBin: sBin, endBin: eBin}
			chunkRanges = append(chunkRanges, j)
			jobs <- j
		}
	}
	close(jobs)

	chunkFilled := make([]bool, len(chunkRanges))

	var errMu sync.Mutex
	var firstErr error
	setErr := func(err error) {
		if err == nil {
			return
		}
		errMu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		errMu.Unlock()
	}

	workers := maxWorkers
	if len(chunkRanges) < workers {
		workers = len(chunkRanges)
	}

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 64*1024)
			for job := range jobs {
				if ctx.Err() != nil {
					return
				}
				chunkBins := job.endBin - job.startBin
				tStart := float64(job.startBin) * binDur
				chunkDur := float64(chunkBins) * binDur

				peaks, rms, bandRMS, zcr, chunkSamples, err := decodeAudioRange(
					ctx, ffmpegPath, filePath, tStart, chunkDur, chunkBins, buf, reportBin,
				)
				if ctx.Err() != nil {
					return
				}
				if err != nil || chunkSamples == 0 {
					select {
					case <-ctx.Done():
						return
					case <-time.After(40 * time.Millisecond):
					}
					p2, r2, b2, z2, s2, err2 := decodeAudioRange(
						ctx, ffmpegPath, filePath, tStart, chunkDur, chunkBins, buf, reportBin,
					)
					if ctx.Err() != nil {
						return
					}
					if s2 > chunkSamples {
						peaks, rms, bandRMS, zcr, chunkSamples, err = p2, r2, b2, z2, s2, err2
					} else if err2 != nil && err == nil {
						err = err2
					}
				}
				if err != nil && chunkSamples == 0 {
					setErr(err)
					continue
				}
				if chunkSamples > 0 {
					for lb := 0; lb < chunkBins; lb++ {
						gIdx := job.startBin + lb
						rawPeaks[gIdx] = peaks[lb]
						rawRMS[gIdx] = rms[lb]
						rawBandRMS[gIdx] = bandRMS[lb]
						rawZCR[gIdx] = zcr[lb]
					}
					chunkFilled[job.chunkIdx] = true
					totalSamples.Add(chunkSamples)
				}
			}
		}()
	}

	wg.Wait()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// If some chunks succeeded while others returned 0 samples (e.g., broken container seek index / PTS reset),
	// perform one sequential pass to fill all remaining timeline bins completely before displaying.
	if totalSamples.Load() > 0 {
		hasMissing := false
		for _, ok := range chunkFilled {
			if !ok {
				hasMissing = true
				break
			}
		}
		if hasMissing {
			buf := make([]byte, 64*1024)
			sp, sr, sb, sz, seqSamples, seqErr := decodeAudioRange(
				ctx, ffmpegPath, filePath, 0, durationSec, targetBins, buf, reportBin,
			)
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if seqErr == nil && seqSamples > 0 {
				for _, cr := range chunkRanges {
					if !chunkFilled[cr.chunkIdx] {
						for b := cr.startBin; b < cr.endBin; b++ {
							rawPeaks[b] = sp[b]
							rawRMS[b] = sr[b]
							rawBandRMS[b] = sb[b]
							rawZCR[b] = sz[b]
						}
						chunkFilled[cr.chunkIdx] = true
					}
				}
				totalSamples.Add(seqSamples)
				firstErr = nil
			}
		}
	}

	if totalSamples.Load() == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return &AudioWaveform{Ready: true, HasAudio: false}, nil
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
	activityBytes := classifyAudioActivity(rawPeaks, rawRMS, rawBandRMS, rawZCR)

	if onProgress != nil {
		onProgress(100)
	}

	return &AudioWaveform{
		Ready:       true,
		HasAudio:    true,
		Bins:        len(peaksBytes),
		BinDuration: durationSec / float64(len(peaksBytes)),
		PeaksB64:    base64.StdEncoding.EncodeToString(peaksBytes),
		RMSB64:      base64.StdEncoding.EncodeToString(rmsBytes),
		ActivityB64: base64.StdEncoding.EncodeToString(activityBytes),
	}, nil
}

func decodeAudioRange(
	ctx context.Context,
	ffmpegPath, filePath string,
	tStart, chunkDur float64,
	chunkBins int,
	buf []byte,
	onBin func(),
) ([]float64, []float64, []float64, []float64, int64, error) {
	const sampleRate = 8000

	args := []string{
		"-hide_banner", "-loglevel", "error",
		"-err_detect", "ignore_err",
	}
	if tStart > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.6f", tStart))
	}
	args = append(args,
		"-t", fmt.Sprintf("%.6f", chunkDur),
		"-i", filePath,
		"-vn",
		"-ac", "1",
		"-ar", "8000",
		"-af", "aresample=8000:async=1",
		"-f", "s16le",
		"pipe:1",
	)

	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	hideSubprocessWindow(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, nil, 0, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, nil, nil, 0, fmt.Errorf("failed to start audio waveform extraction: %w", err)
	}

	rawPeaks := make([]float64, chunkBins)
	rawRMS := make([]float64, chunkBins)
	rawBandRMS := make([]float64, chunkBins)
	rawZCR := make([]float64, chunkBins)

	expectedSamples := int(math.Round(chunkDur * float64(sampleRate)))
	if expectedSamples < chunkBins {
		expectedSamples = chunkBins
	}

	var leftover byte
	hasLeftover := false

	localBin := 0
	nextBinSampleEnd := expectedSamples / chunkBins
	sampleIdx := 0

	var binPeak float64
	var binSumSq float64
	var binBandSumSq float64
	var binCrossings int
	var binCount int

	// 1st-order vocal bandpass (~250 Hz HP -> ~3300 Hz LP at 8 kHz sample rate)
	const (
		hpAlpha = 0.835
		lpAlpha = 0.720
	)
	var prevRaw float64
	var hpY float64
	var lpY float64
	var prevSign int8

	flushLocalBin := func() {
		if localBin >= chunkBins {
			return
		}
		if binCount > 0 {
			n := float64(binCount)
			rawPeaks[localBin] = binPeak
			rawRMS[localBin] = math.Sqrt(binSumSq / n)
			rawBandRMS[localBin] = math.Sqrt(binBandSumSq / n)
			rawZCR[localBin] = float64(binCrossings) / n
		}
		binPeak = 0
		binSumSq = 0
		binBandSumSq = 0
		binCrossings = 0
		binCount = 0
		localBin++
		nextBinSampleEnd = ((localBin + 1) * expectedSamples) / chunkBins
		if onBin != nil {
			onBin()
		}
	}

	processSample := func(sample int16) {
		v := float64(sample) / 32768.0
		av := math.Abs(v)
		if av > binPeak {
			binPeak = av
		}
		binSumSq += v * v

		hpY = hpAlpha * (hpY + v - prevRaw)
		prevRaw = v
		lpY = lpY + lpAlpha*(hpY-lpY)
		binBandSumSq += lpY * lpY

		if av > 0.002 {
			var s int8 = 1
			if v < 0 {
				s = -1
			}
			if prevSign != 0 && s != prevSign {
				binCrossings++
			}
			prevSign = s
		}

		binCount++
		sampleIdx++
		if sampleIdx >= nextBinSampleEnd && localBin < chunkBins-1 {
			flushLocalBin()
		}
	}

	var readErr error
	var chunkSamples int64
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			data := buf[:n]
			idx := 0
			if hasLeftover {
				sample := int16(binary.LittleEndian.Uint16([]byte{leftover, data[0]}))
				processSample(sample)
				chunkSamples++
				hasLeftover = false
				idx = 1
			}
			for idx+1 < len(data) {
				sample := int16(binary.LittleEndian.Uint16(data[idx : idx+2]))
				processSample(sample)
				chunkSamples++
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
	if localBin < chunkBins && binCount > 0 {
		flushLocalBin()
	}

	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return nil, nil, nil, nil, chunkSamples, ctx.Err()
	}
	if readErr != nil {
		return rawPeaks, rawRMS, rawBandRMS, rawZCR, chunkSamples, readErr
	}
	if waitErr != nil && chunkSamples == 0 {
		return rawPeaks, rawRMS, rawBandRMS, rawZCR, 0, fmt.Errorf("ffmpeg audio waveform decode failed: %w", waitErr)
	}
	return rawPeaks, rawRMS, rawBandRMS, rawZCR, chunkSamples, nil
}

// classifyAudioActivity labels each bin as 0=quiet/ambient, 1=talking/speech, or 2=noise/transient.
func classifyAudioActivity(rawPeaks, rawRMS, rawBandRMS, rawZCR []float64) []byte {
	n := len(rawRMS)
	act := make([]byte, n)
	if n == 0 {
		return act
	}

	sorted := append([]float64(nil), rawRMS...)
	sort.Float64s(sorted)
	idx := int(math.Floor(0.20 * float64(len(sorted)-1)))
	if idx < 0 {
		idx = 0
	}
	noiseFloor := sorted[idx]
	if noiseFloor < 0.001 {
		noiseFloor = 0.001
	} else if noiseFloor > 0.05 {
		noiseFloor = 0.05
	}

	rmsGate := math.Max(0.004, noiseFloor*2.2)
	peakGate := math.Max(0.016, noiseFloor*4.2)

	for i := 0; i < n; i++ {
		rms := rawRMS[i]
		peak := rawPeaks[i]
		if rms < rmsGate && peak < peakGate {
			act[i] = 0
			continue
		}
		denom := math.Max(1e-6, rms)
		bandRatio := rawBandRMS[i] / denom
		crest := peak / denom
		zcr := rawZCR[i]

		if bandRatio >= 0.42 && zcr >= 0.025 && zcr <= 0.48 && crest <= 5.5 {
			act[i] = 1
		} else {
			act[i] = 2
		}
	}

	// Pass A: Bridge brief <50ms consonant gaps inside continuous speech
	if n >= 3 {
		bridged := append([]byte(nil), act...)
		for i := 1; i < n-1; i++ {
			if act[i] != 1 && act[i-1] == 1 && act[i+1] == 1 && rawRMS[i] > noiseFloor*1.3 {
				bridged[i] = 1
			}
		}
		act = bridged

		// Pass B: Demote isolated 1-bin (<50ms) speech candidates to transient noise
		smoothed := append([]byte(nil), act...)
		for i := 1; i < n-1; i++ {
			if act[i] == 1 && act[i-1] != 1 && act[i+1] != 1 {
				smoothed[i] = 2
			}
		}
		act = smoothed
	}

	return act
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
