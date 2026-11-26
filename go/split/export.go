package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ExportProgress reports live export status to GUI or CLI callers.
type ExportProgress struct {
	Current int    `json:"current"`
	Total   int    `json:"total"`
	Label   string `json:"label"`
	Phase   string `json:"phase"`
}

// ExportOutputFile describes a single generated output file.
type ExportOutputFile struct {
	Index  int     `json:"index"`
	Name   string  `json:"name"`
	Path   string  `json:"path"`
	Bytes  int64   `json:"bytes"`
	Dur    float64 `json:"dur"`
	Frames int     `json:"frames"`
}

// ExportResult summarizes a completed export run.
type ExportResult struct {
	Mode       string             `json:"mode"`
	Files      []ExportOutputFile `json:"files"`
	TotalBytes int64              `json:"totalBytes"`
}

type ProgressFunc func(ExportProgress)

func randomTag() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func isMP4Family(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".mp4" || ext == ".m4v" || ext == ".mov"
}

func formatFFmpegSec(sec float64) string {
	return strconv.FormatFloat(sec, 'f', 6, 64)
}

func moveReplace(src, dst string) error {
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(src, dst)
}

// sliceSegmentLossless copies [startSec, endSec] via ffmpeg -c copy with keyframe-safe seek bounds.
func sliceSegmentLossless(ctx context.Context, ffmpegPath string, media *MediaInfo, startSec, endSec float64, outPath string) error {
	args := []string{"-hide_banner", "-loglevel", "error", "-y"}

	actualStart := 0.0
	if startSec > 0.001 {
		nextKf := endSec
		for _, kf := range media.Keyframes {
			if kf > startSec+0.0005 {
				if kf < nextKf {
					nextKf = kf
				}
				break
			}
		}
		epsStart := math.Min(0.002, 0.2*math.Max(0.0, nextKf-startSec))
		seekAt := startSec + epsStart
		actualStart = startSec
		args = append(args, "-noaccurate_seek", "-ss", formatFFmpegSec(seekAt))
	}

	args = append(args, "-i", media.FilePath)

	if endSec < media.DurationSec-0.005 {
		prevKf := actualStart
		for _, kf := range media.Keyframes {
			if kf < endSec-0.0005 {
				prevKf = kf
			} else {
				break
			}
		}
		epsEnd := math.Min(0.002, 0.2*math.Max(0.0, endSec-prevKf))
		dur := (endSec - epsEnd) - actualStart
		if dur <= 0.001 {
			dur = math.Max(0.01, endSec-actualStart)
		}
		args = append(args, "-t", formatFFmpegSec(dur))
	}

	args = append(args,
		"-c", "copy",
		"-map", "0:v:0",
		"-map", "0:a?",
		"-avoid_negative_ts", "make_zero",
	)
	if isMP4Family(outPath) {
		args = append(args, "-movflags", "+faststart")
	}
	args = append(args, outPath)

	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	hideSubprocessWindow(cmd)
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		msg := strings.TrimSpace(stderrBuf.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("ffmpeg slice failed: %s", msg)
	}
	if st, err := os.Stat(outPath); err != nil || st.Size() == 0 {
		return fmt.Errorf("ffmpeg produced an empty output file")
	}
	return nil
}

// ExportSplit exports each kept segment sequentially to targetPaths (`_01`, `_02`, ...).
func ExportSplit(ctx context.Context, tools ToolStatus, proj *ProjectState, targetPaths []string, onProgress ProgressFunc) (*ExportResult, error) {
	if !tools.Ready {
		return nil, fmt.Errorf("%s", tools.Error)
	}
	if proj == nil || proj.Media == nil {
		return nil, fmt.Errorf("no video loaded")
	}
	kept := proj.KeptSegments()
	if len(kept) == 0 {
		return nil, fmt.Errorf("no kept segments selected for export")
	}
	if len(targetPaths) != len(kept) {
		return nil, fmt.Errorf("target path count (%d) does not match kept segment count (%d)", len(targetPaths), len(kept))
	}

	tag := randomTag()
	var tempFiles []string
	defer func() {
		for _, tf := range tempFiles {
			_ = os.Remove(tf)
		}
	}()

	res := &ExportResult{
		Mode:  "split",
		Files: make([]ExportOutputFile, 0, len(kept)),
	}

	for i, seg := range kept {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		dstPath := targetPaths[i]
		dstDir := filepath.Dir(dstPath)
		dstExt := filepath.Ext(dstPath)
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return nil, err
		}
		tmpPath := filepath.Join(dstDir, fmt.Sprintf("__split_tmp_%s_%02d%s", tag, i+1, dstExt))
		tempFiles = append(tempFiles, tmpPath)

		if onProgress != nil {
			onProgress(ExportProgress{
				Current: i + 1,
				Total:   len(kept),
				Label:   filepath.Base(dstPath),
				Phase:   fmt.Sprintf("Writing segment %d of %d...", i+1, len(kept)),
			})
		}

		if err := sliceSegmentLossless(ctx, tools.FFmpegPath, proj.Media, seg.Start, seg.End, tmpPath); err != nil {
			return nil, err
		}
		if err := moveReplace(tmpPath, dstPath); err != nil {
			return nil, fmt.Errorf("failed to write %s: %w", filepath.Base(dstPath), err)
		}

		probe := ProbeOutputSegment(ctx, tools.FFprobePath, dstPath, proj.Media.FPS)
		res.TotalBytes += probe.Bytes
		res.Files = append(res.Files, ExportOutputFile{
			Index:  i + 1,
			Name:   filepath.Base(dstPath),
			Path:   dstPath,
			Bytes:  probe.Bytes,
			Dur:    probe.Dur,
			Frames: probe.Frames,
		})
	}
	return res, nil
}

// ExportJoin exports all kept segments joined losslessly into one file at targetPath (`_edit`).
func ExportJoin(ctx context.Context, tools ToolStatus, proj *ProjectState, targetPath string, onProgress ProgressFunc) (*ExportResult, error) {
	if !tools.Ready {
		return nil, fmt.Errorf("%s", tools.Error)
	}
	if proj == nil || proj.Media == nil {
		return nil, fmt.Errorf("no video loaded")
	}
	kept := proj.KeptSegments()
	if len(kept) == 0 {
		return nil, fmt.Errorf("no kept segments selected for export")
	}

	dstDir := filepath.Dir(targetPath)
	dstExt := filepath.Ext(targetPath)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return nil, err
	}

	tag := randomTag()
	var tempFiles []string
	defer func() {
		for _, tf := range tempFiles {
			_ = os.Remove(tf)
		}
	}()

	tmpFinal := filepath.Join(dstDir, fmt.Sprintf("__join_tmp_%s_final%s", tag, dstExt))
	tempFiles = append(tempFiles, tmpFinal)

	if len(kept) == 1 {
		if onProgress != nil {
			onProgress(ExportProgress{
				Current: 1,
				Total:   1,
				Label:   filepath.Base(targetPath),
				Phase:   "Copying kept segment (-c copy)...",
			})
		}
		if err := sliceSegmentLossless(ctx, tools.FFmpegPath, proj.Media, kept[0].Start, kept[0].End, tmpFinal); err != nil {
			return nil, err
		}
	} else {
		totalSteps := len(kept) + 1
		partPaths := make([]string, 0, len(kept))
		for i, seg := range kept {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			partPath := filepath.Join(dstDir, fmt.Sprintf("__join_tmp_%s_part%02d%s", tag, i+1, dstExt))
			tempFiles = append(tempFiles, partPath)
			partPaths = append(partPaths, partPath)

			if onProgress != nil {
				onProgress(ExportProgress{
					Current: i + 1,
					Total:   totalSteps,
					Label:   filepath.Base(targetPath),
					Phase:   fmt.Sprintf("Extracting kept part %d of %d...", i+1, len(kept)),
				})
			}
			if err := sliceSegmentLossless(ctx, tools.FFmpegPath, proj.Media, seg.Start, seg.End, partPath); err != nil {
				return nil, err
			}
		}

		listPath := filepath.Join(dstDir, fmt.Sprintf("__join_tmp_%s_list.txt", tag))
		tempFiles = append(tempFiles, listPath)
		var listBuf strings.Builder
		listBuf.WriteString("ffconcat version 1.0\r\n")
		for _, pp := range partPaths {
			escaped := strings.ReplaceAll(pp, "'", `'\''`)
			listBuf.WriteString("file '")
			listBuf.WriteString(escaped)
			listBuf.WriteString("'\r\n")
		}
		if err := os.WriteFile(listPath, []byte(listBuf.String()), 0o644); err != nil {
			return nil, err
		}

		if onProgress != nil {
			onProgress(ExportProgress{
				Current: totalSteps,
				Total:   totalSteps,
				Label:   filepath.Base(targetPath),
				Phase:   "Joining segments losslessly (-c copy)...",
			})
		}

		concatArgs := []string{
			"-hide_banner", "-loglevel", "error", "-y",
			"-f", "concat", "-safe", "0",
			"-i", listPath,
			"-c", "copy",
			"-map", "0:v:0",
			"-map", "0:a?",
		}
		if isMP4Family(targetPath) {
			concatArgs = append(concatArgs, "-movflags", "+faststart")
		}
		concatArgs = append(concatArgs, tmpFinal)

		cmd := exec.CommandContext(ctx, tools.FFmpegPath, concatArgs...)
		hideSubprocessWindow(cmd)
		var stderrBuf bytes.Buffer
		cmd.Stderr = &stderrBuf
		if err := cmd.Run(); err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			msg := strings.TrimSpace(stderrBuf.String())
			if msg == "" {
				msg = err.Error()
			}
			return nil, fmt.Errorf("ffmpeg concat join failed: %s", msg)
		}
	}

	if err := moveReplace(tmpFinal, targetPath); err != nil {
		return nil, fmt.Errorf("failed to write %s: %w", filepath.Base(targetPath), err)
	}

	probe := ProbeOutputSegment(ctx, tools.FFprobePath, targetPath, proj.Media.FPS)
	return &ExportResult{
		Mode:       "join",
		TotalBytes: probe.Bytes,
		Files: []ExportOutputFile{
			{
				Index:  1,
				Name:   filepath.Base(targetPath),
				Path:   targetPath,
				Bytes:  probe.Bytes,
				Dur:    probe.Dur,
				Frames: probe.Frames,
			},
		},
	}, nil
}

// ExportBalancedParts splits a video into `parts` byte-balanced lossless files (`_1ofN` .. `_NofN`),
// matching splitvid.ps1's -2..-10 behavior.
func ExportBalancedParts(ctx context.Context, tools ToolStatus, media *MediaInfo, parts int, outDir string) (*ExportResult, []string, error) {
	if !tools.Ready {
		return nil, nil, fmt.Errorf("%s", tools.Error)
	}
	cutIndices, err := ComputeBalancedCutKeyframes(media.KFRecords, media.TotalBytes, parts)
	if err != nil {
		return nil, nil, err
	}

	cutTriggers := make([]string, 0, len(cutIndices))
	cutLabels := make([]string, 0, len(cutIndices))
	for _, idx := range cutIndices {
		kfSec := media.KFRecords[idx].RelSec
		prevSec := media.KFRecords[idx-1].RelSec
		gap := math.Max(0.004, kfSec-prevSec)
		margin := math.Min(0.15, gap*0.5)
		trigger := math.Max(0.001, math.Round((kfSec-margin)*1e6)/1e6)
		cutTriggers = append(cutTriggers, strconv.FormatFloat(trigger, 'f', -1, 64))
		cutLabels = append(cutLabels, fmt.Sprintf("%.2fs", kfSec))
	}

	srcDir, base, ext := SplitBaseExt(media.FilePath)
	if outDir == "" {
		outDir = srcDir
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, nil, err
	}

	tag := randomTag()
	tempPrefix := fmt.Sprintf("__split_tmp_%s_", tag)
	segPattern := filepath.Join(outDir, tempPrefix+"%d"+ext)
	defer func() {
		if matches, err := filepath.Glob(filepath.Join(outDir, tempPrefix+"*"+ext)); err == nil {
			for _, m := range matches {
				_ = os.Remove(m)
			}
		}
	}()

	cmd := exec.CommandContext(ctx, tools.FFmpegPath,
		"-hide_banner", "-loglevel", "error", "-y",
		"-i", media.FilePath,
		"-c", "copy",
		"-map", "0:v:0",
		"-map", "0:a?",
		"-f", "segment",
		"-segment_times", strings.Join(cutTriggers, ","),
		"-segment_start_number", "1",
		"-reset_timestamps", "1",
		segPattern,
	)
	hideSubprocessWindow(cmd)
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderrBuf.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, nil, fmt.Errorf("ffmpeg stream copy failed: %s", msg)
	}

	generated, err := filepath.Glob(filepath.Join(outDir, tempPrefix+"*"+ext))
	if err != nil || len(generated) == 0 {
		return nil, nil, fmt.Errorf("ffmpeg produced no output segments")
	}
	sort.Slice(generated, func(i, j int) bool {
		return extractTrailingNum(generated[i]) < extractTrailingNum(generated[j])
	})

	actualTotal := len(generated)
	res := &ExportResult{
		Mode:  "balanced",
		Files: make([]ExportOutputFile, 0, actualTotal),
	}
	for i, tmpPath := range generated {
		idx := i + 1
		finalName := fmt.Sprintf("%s_%dof%d%s", base, idx, actualTotal, ext)
		finalPath := filepath.Join(outDir, finalName)
		if err := moveReplace(tmpPath, finalPath); err != nil {
			return nil, nil, err
		}
		probe := ProbeOutputSegment(ctx, tools.FFprobePath, finalPath, media.FPS)
		res.TotalBytes += probe.Bytes
		res.Files = append(res.Files, ExportOutputFile{
			Index:  idx,
			Name:   finalName,
			Path:   finalPath,
			Bytes:  probe.Bytes,
			Dur:    probe.Dur,
			Frames: probe.Frames,
		})
	}
	return res, cutLabels, nil
}

func extractTrailingNum(p string) int {
	_, base, _ := SplitBaseExt(p)
	i := len(base)
	for i > 0 && base[i-1] >= '0' && base[i-1] <= '9' {
		i--
	}
	if i == len(base) {
		return 0
	}
	n, _ := strconv.Atoi(base[i:])
	return n
}

// ExportWindow performs splitvid.ps1-style lossless Trim or Crop to targetPath.
func ExportWindow(ctx context.Context, tools ToolStatus, media *MediaInfo, isCrop bool, startValSec, endValSec float64, targetPath string) (*ExportResult, float64, float64, error) {
	if !tools.Ready {
		return nil, 0, 0, fmt.Errorf("%s", tools.Error)
	}
	totalSec := media.DurationSec
	var targetEndSec float64
	if isCrop {
		if endValSec > totalSec {
			return nil, 0, 0, fmt.Errorf("requested crop -end timestamp (%.2fs) exceeds video duration (%.2fs)", endValSec, totalSec)
		}
		targetEndSec = endValSec
		if startValSec >= targetEndSec {
			return nil, 0, 0, fmt.Errorf("requested crop range [%.2fs .. %.2fs] has empty or negative duration", startValSec, targetEndSec)
		}
	} else {
		if startValSec+endValSec >= totalSec {
			return nil, 0, 0, fmt.Errorf("requested trim (%.2fs start + %.2fs end) removes entire %.2fs video", startValSec, endValSec, totalSec)
		}
		targetEndSec = totalSec - endValSec
	}

	actualStartSec := SnapStartKeyframe(media.Keyframes, startValSec, targetEndSec)
	if actualStartSec >= targetEndSec {
		return nil, 0, 0, fmt.Errorf("keyframed start (%.2fs) leaves no duration before target end (%.2fs)", actualStartSec, targetEndSec)
	}
	keepDur := targetEndSec - actualStartSec

	if targetPath == "" {
		tag := "trim"
		if isCrop {
			tag = "crop"
		}
		targetPath = SuggestedSuffixPath(media.FilePath, tag)
	}

	dstDir := filepath.Dir(targetPath)
	dstExt := filepath.Ext(targetPath)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return nil, 0, 0, err
	}
	tmpPath := filepath.Join(dstDir, fmt.Sprintf("__win_tmp_%s%s", randomTag(), dstExt))
	defer func() { _ = os.Remove(tmpPath) }()

	args := []string{"-hide_banner", "-loglevel", "error", "-y"}
	if actualStartSec > 0 {
		args = append(args, "-noaccurate_seek", "-ss", strconv.FormatFloat(actualStartSec, 'f', 3, 64))
	}
	args = append(args, "-i", media.FilePath)
	if isCrop || endValSec > 0 {
		args = append(args, "-t", strconv.FormatFloat(keepDur, 'f', 3, 64))
	}
	args = append(args,
		"-c", "copy",
		"-map", "0:v:0",
		"-map", "0:a?",
		"-avoid_negative_ts", "make_zero",
	)
	if isMP4Family(targetPath) {
		args = append(args, "-movflags", "+faststart")
	}
	args = append(args, tmpPath)

	cmd := exec.CommandContext(ctx, tools.FFmpegPath, args...)
	hideSubprocessWindow(cmd)
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderrBuf.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, 0, 0, fmt.Errorf("ffmpeg window copy failed: %s", msg)
	}

	if err := moveReplace(tmpPath, targetPath); err != nil {
		return nil, 0, 0, err
	}

	probe := ProbeOutputSegment(ctx, tools.FFprobePath, targetPath, media.FPS)
	modeName := "trim"
	if isCrop {
		modeName = "crop"
	}
	return &ExportResult{
		Mode:       modeName,
		TotalBytes: probe.Bytes,
		Files: []ExportOutputFile{
			{
				Index:  1,
				Name:   filepath.Base(targetPath),
				Path:   targetPath,
				Bytes:  probe.Bytes,
				Dur:    probe.Dur,
				Frames: probe.Frames,
			},
		},
	}, actualStartSec, targetEndSec, nil
}
