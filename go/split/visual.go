package main

import (
	"context"
	"encoding/base64"
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

// VisualDelta holds downsampled frame-to-frame pixel delta and scene-cut spike bins across the timeline.
type VisualDelta struct {
	Ready       bool    `json:"ready"`
	Bins        int     `json:"bins"`
	BinDuration float64 `json:"binDuration"`
	DeltaB64    string  `json:"deltaB64"`
	SceneB64    string  `json:"sceneB64"`
}

const (
	thumbW         = 24
	thumbH         = 14
	thumbPixels    = thumbW * thumbH
	thumbFrameSize = thumbPixels * 3 // rgb24
)

// ExtractVisualDelta computes compact visual motion delta and color-histogram scene-cut intensity
// along the video timeline using fast downscaled FFmpeg decoding.
func ExtractVisualDelta(ctx context.Context, ffmpegPath string, media *MediaInfo) (*VisualDelta, error) {
	return ExtractVisualDeltaWithProgress(ctx, ffmpegPath, media, nil)
}

// ExtractVisualDeltaWithProgress is like ExtractVisualDelta but reports streaming percentage updates
// while decoding thumbnails in parallel chunks across multiple workers.
func ExtractVisualDeltaWithProgress(ctx context.Context, ffmpegPath string, media *MediaInfo, onProgress AnalysisProgressFunc) (*VisualDelta, error) {
	return extractVisualDeltaChunks(ctx, ffmpegPath, media, 0, onProgress)
}

func extractVisualDeltaChunks(ctx context.Context, ffmpegPath string, media *MediaInfo, forceChunks int, onProgress AnalysisProgressFunc) (*VisualDelta, error) {
	if media == nil || media.DurationSec <= 0 || ffmpegPath == "" {
		return &VisualDelta{Ready: true}, nil
	}

	var targetBins int
	switch {
	case media.DurationSec <= 120:
		targetBins = int(math.Round(media.DurationSec * 4.0))
	case media.DurationSec <= 600:
		targetBins = int(math.Round(media.DurationSec * 2.0))
	default:
		targetBins = int(math.Round(media.DurationSec / 2.0))
		if targetBins < 600 {
			targetBins = 600
		}
		if targetBins > 1800 {
			targetBins = 1800
		}
	}
	if targetBins < 80 {
		targetBins = 80
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
	} else if media.DurationSec >= 20 {
		numChunks = int(math.Ceil(media.DurationSec / 45.0))
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

	sampleFPS := float64(targetBins) / media.DurationSec
	if media.DurationSec <= 120 {
		if sampleFPS < 1.0 {
			sampleFPS = 1.0
		} else if sampleFPS > 14.0 {
			sampleFPS = 14.0
		}
	} else if media.DurationSec <= 600 {
		if sampleFPS < 1.0 {
			sampleFPS = 1.0
		} else if sampleFPS > 2.5 {
			sampleFPS = 2.5
		}
	} else {
		if sampleFPS < 0.05 {
			sampleFPS = 0.05
		} else if sampleFPS > 2.0 {
			sampleFPS = 2.0
		}
	}
	if media.FPS > 0 && sampleFPS > media.FPS {
		sampleFPS = media.FPS
	}
	expectedFramesTotal := math.Max(1.0, media.DurationSec*sampleFPS)

	if onProgress != nil {
		onProgress(2)
	}

	rawDelta := make([]float64, targetBins)
	rawScene := make([]float64, targetBins)

	var doneFrames atomic.Int64
	var totalFrames atomic.Int64
	var progressMu sync.Mutex
	lastEmit := time.Now()

	reportFrame := func() {
		done := doneFrames.Add(1)
		if onProgress == nil {
			return
		}
		progressMu.Lock()
		if time.Since(lastEmit) >= 65*time.Millisecond {
			pct := math.Min(96.0, 2.0+(float64(done)/expectedFramesTotal)*94.0)
			onProgress(pct)
			lastEmit = time.Now()
		}
		progressMu.Unlock()
	}

	type visualJob struct {
		chunkIdx int
		startIdx int
		endIdx   int
	}
	chunkRanges := make([]visualJob, 0, numChunks)
	jobs := make(chan visualJob, numChunks)
	for c := 0; c < numChunks; c++ {
		sIdx := (c * targetBins) / numChunks
		eIdx := ((c + 1) * targetBins) / numChunks
		if eIdx > sIdx {
			j := visualJob{chunkIdx: len(chunkRanges), startIdx: sIdx, endIdx: eIdx}
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

	binDur := media.DurationSec / float64(targetBins)
	preferNoKey := media.DurationSec > 600 && binDur >= 1.5

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			curFrame := make([]byte, thumbFrameSize)
			prevFrame := make([]byte, thumbFrameSize)

			for job := range jobs {
				if ctx.Err() != nil {
					return
				}

				localBins := job.endIdx - job.startIdx
				tStart := float64(job.startIdx) * binDur
				chunkDur := float64(localBins) * binDur
				leadSec := 0.0
				if job.startIdx > 0 {
					leadSec = 1.0 / sampleFPS
					if leadSec > tStart {
						leadSec = tStart
					}
				}
				seekTime := tStart - leadSec
				actualLeadSec := tStart - seekTime
				readDur := chunkDur + actualLeadSec

				skipMode := "noref"
				minAcceptable := localBins / 2
				if minAcceptable < 1 {
					minAcceptable = 1
				}
				if preferNoKey {
					skipMode = "nokey"
					minAcceptable = (localBins * 3) / 4
					if minAcceptable < 1 {
						minAcceptable = 1
					}
				}

				localDelta, localScene, frameIdx, err := decodeVisualRange(
					ctx, ffmpegPath, media.FilePath, skipMode,
					seekTime, readDur, actualLeadSec, chunkDur, sampleFPS, localBins,
					curFrame, prevFrame, reportFrame,
				)
				if ctx.Err() != nil {
					return
				}

				if err != nil || frameIdx < minAcceptable {
					// Fallback to noref (plus brief backoff if already noref) so open-GOP / I/O spikes complete cleanly
					if skipMode != "nokey" {
						select {
						case <-ctx.Done():
							return
						case <-time.After(40 * time.Millisecond):
						}
					}
					d2, s2, f2, err2 := decodeVisualRange(
						ctx, ffmpegPath, media.FilePath, "noref",
						seekTime, readDur, actualLeadSec, chunkDur, sampleFPS, localBins,
						curFrame, prevFrame, reportFrame,
					)
					if ctx.Err() != nil {
						return
					}
					if f2 > frameIdx {
						localDelta, localScene, frameIdx, err = d2, s2, f2, err2
					} else if err2 != nil && err == nil {
						err = err2
					}
				}

				if err != nil && frameIdx == 0 {
					setErr(err)
					continue
				}
				if frameIdx > 0 {
					for lb := 0; lb < localBins; lb++ {
						rawDelta[job.startIdx+lb] = localDelta[lb]
						rawScene[job.startIdx+lb] = localScene[lb]
					}
					chunkFilled[job.chunkIdx] = true
					totalFrames.Add(int64(frameIdx))
				}
			}
		}()
	}

	wg.Wait()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Verify 100% timeline chunk coverage; if any chunk missed frames (e.g. seek past wrapped PTS), sweep sequentially
	hasMissingChunk := false
	for _, ok := range chunkFilled {
		if !ok {
			hasMissingChunk = true
			break
		}
	}
	if hasMissingChunk {
		curFrame := make([]byte, thumbFrameSize)
		prevFrame := make([]byte, thumbFrameSize)
		seqDelta, seqScene, seqFrames, seqErr := decodeVisualRange(
			ctx, ffmpegPath, media.FilePath, "noref",
			0, media.DurationSec, 0, media.DurationSec, sampleFPS, targetBins,
			curFrame, prevFrame, reportFrame,
		)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if seqErr != nil && totalFrames.Load() == 0 && seqFrames == 0 {
			return nil, seqErr
		}
		if seqFrames > 0 {
			for _, cr := range chunkRanges {
				if !chunkFilled[cr.chunkIdx] {
					for b := cr.startIdx; b < cr.endIdx; b++ {
						rawDelta[b] = seqDelta[b]
						rawScene[b] = seqScene[b]
					}
					chunkFilled[cr.chunkIdx] = true
				}
			}
			totalFrames.Add(int64(seqFrames))
			firstErr = nil
		}
	}

	if totalFrames.Load() == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return &VisualDelta{Ready: true}, nil
	}

	p98Delta := percentile98(rawDelta)
	maxScene := 0.0
	for i := 0; i < targetBins; i++ {
		if rawScene[i] > maxScene {
			maxScene = rawScene[i]
		}
	}

	refDelta := math.Max(0.05, p98Delta)
	refScene := math.Max(0.25, maxScene)

	deltaBytes := make([]byte, targetBins)
	sceneBytes := make([]byte, targetBins)
	for i := 0; i < targetBins; i++ {
		deltaBytes[i] = visualScaleByte(rawDelta[i], refDelta, 0.004, 0.62)
		sceneBytes[i] = visualScaleByte(rawScene[i], refScene, 0.08, 0.75)
	}

	if onProgress != nil {
		onProgress(100)
	}

	return &VisualDelta{
		Ready:       true,
		Bins:        targetBins,
		BinDuration: media.DurationSec / float64(targetBins),
		DeltaB64:    base64.StdEncoding.EncodeToString(deltaBytes),
		SceneB64:    base64.StdEncoding.EncodeToString(sceneBytes),
	}, nil
}

func decodeVisualRange(
	ctx context.Context,
	ffmpegPath, filePath, skipMode string,
	seekTime, readDur, actualLeadSec, chunkDur, sampleFPS float64,
	localBins int,
	curFrame, prevFrame []byte,
	onFrame func(),
) ([]float64, []float64, int, error) {
	args := []string{
		"-hide_banner", "-loglevel", "error",
		"-threads", "2",
		"-err_detect", "ignore_err",
	}
	if seekTime > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.6f", seekTime))
	}
	if readDur > 0 {
		args = append(args, "-t", fmt.Sprintf("%.6f", readDur))
	}
	if skipMode != "" {
		args = append(args, "-skip_frame", skipMode)
	}
	args = append(args,
		"-i", filePath,
		"-an",
		"-vf", fmt.Sprintf("fps=%.6f,scale=%d:%d:flags=neighbor,format=rgb24", sampleFPS, thumbW, thumbH),
		"-f", "rawvideo",
		"pipe:1",
	)

	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	hideSubprocessWindow(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, 0, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, 0, fmt.Errorf("failed to start visual delta extraction: %w", err)
	}

	localDelta := make([]float64, localBins)
	localScene := make([]float64, localBins)
	var curHist [64]int
	var prevHist [64]int

	frameIdx := 0
	var readErr error
	for {
		_, err := io.ReadFull(stdout, curFrame)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			readErr = err
			break
		}

		computeRGBHist64(curFrame, &curHist)
		var mad, hdiff float64
		if frameIdx > 0 {
			mad = computeFrameMAD(prevFrame, curFrame)
			hdiff = computeHistDiff(&prevHist, &curHist)
		}

		tRel := (float64(frameIdx) / sampleFPS) - actualLeadSec
		if tRel >= -0.001 && chunkDur > 0 {
			if tRel < 0 {
				tRel = 0
			}
			lb := int(math.Floor((tRel / chunkDur) * float64(localBins)))
			if lb >= localBins {
				lb = localBins - 1
			}
			if lb >= 0 {
				if mad > localDelta[lb] {
					localDelta[lb] = mad
				}
				if hdiff > localScene[lb] {
					localScene[lb] = hdiff
				}
			}
		}

		copy(prevFrame, curFrame)
		prevHist = curHist
		frameIdx++
		if onFrame != nil {
			onFrame()
		}
	}

	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return nil, nil, frameIdx, ctx.Err()
	}
	if readErr != nil {
		return localDelta, localScene, frameIdx, readErr
	}
	if waitErr != nil && frameIdx == 0 {
		return localDelta, localScene, 0, fmt.Errorf("ffmpeg visual delta decode failed: %w", waitErr)
	}
	return localDelta, localScene, frameIdx, nil
}

func computeRGBHist64(frame []byte, hist *[64]int) {
	*hist = [64]int{}
	for i := 0; i+2 < len(frame); i += 3 {
		bin := ((int(frame[i]) >> 6) << 4) | ((int(frame[i+1]) >> 6) << 2) | (int(frame[i+2]) >> 6)
		hist[bin]++
	}
}

func computeFrameMAD(a, b []byte) float64 {
	var sum int64
	for i := 0; i < len(a); i++ {
		d := int(a[i]) - int(b[i])
		if d < 0 {
			d = -d
		}
		sum += int64(d)
	}
	return float64(sum) / (float64(len(a)) * 255.0)
}

func computeHistDiff(a, b *[64]int) float64 {
	var sum int
	for i := 0; i < 64; i++ {
		d := a[i] - b[i]
		if d < 0 {
			d = -d
		}
		sum += d
	}
	return float64(sum) / (2.0 * float64(thumbPixels))
}

func percentile98(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := append([]float64(nil), vals...)
	sort.Float64s(sorted)
	idx := int(math.Floor(0.98 * float64(len(sorted)-1)))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func visualScaleByte(val, ref, gate, gamma float64) byte {
	if val < gate || ref <= 0 {
		return 0
	}
	norm := val / ref
	if norm < 0 {
		norm = 0
	} else if norm > 1 {
		norm = 1
	}
	curved := math.Pow(norm, gamma)
	b := math.Round(curved * 255.0)
	if b < 0 {
		return 0
	}
	if b > 255 {
		return 255
	}
	return byte(b)
}
