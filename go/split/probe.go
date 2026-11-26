package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// KeyframeRecord captures a video keyframe timestamp alongside cumulative packet bytes.
type KeyframeRecord struct {
	RelSec   float64 `json:"relSec"`
	CumBytes int64   `json:"cumBytes"`
}

// MediaInfo holds probed container, stream, and keyframe metadata for a video file.
type MediaInfo struct {
	FilePath     string           `json:"filePath"`
	FileName     string           `json:"fileName"`
	FileSize     int64            `json:"fileSize"`
	DurationSec  float64          `json:"durationSec"`
	StartTimeSec float64          `json:"startTimeSec"`
	VideoCodec   string           `json:"videoCodec"`
	AudioCodec   string           `json:"audioCodec"`
	Width        int              `json:"width"`
	Height       int              `json:"height"`
	FPS          float64          `json:"fps"`
	HasAudio     bool             `json:"hasAudio"`
	TotalBytes   int64            `json:"totalBytes"`
	Keyframes    []float64        `json:"keyframes"`
	KFRecords    []KeyframeRecord `json:"-"`
}

// ToolStatus summarizes discovered ffmpeg/ffprobe binaries.
type ToolStatus struct {
	FFmpegPath     string `json:"ffmpegPath"`
	FFprobePath    string `json:"ffprobePath"`
	FFmpegVersion  string `json:"ffmpegVersion"`
	FFprobeVersion string `json:"ffprobeVersion"`
	Ready          bool   `json:"ready"`
	Error          string `json:"error"`
}

// FindMediaTools locates ffmpeg and ffprobe from custom overrides, PATH, or common Windows install locations.
func FindMediaTools(customFFmpeg, customFFprobe string) ToolStatus {
	ff := resolveToolBinary(customFFmpeg, "ffmpeg")
	fp := resolveToolBinary(customFFprobe, "ffprobe")

	status := ToolStatus{
		FFmpegPath:  ff,
		FFprobePath: fp,
		Ready:       ff != "" && fp != "",
	}
	if ff != "" {
		status.FFmpegVersion = readToolVersion(ff)
	}
	if fp != "" {
		status.FFprobeVersion = readToolVersion(fp)
	}
	if !status.Ready {
		var missing []string
		if ff == "" {
			missing = append(missing, "ffmpeg")
		}
		if fp == "" {
			missing = append(missing, "ffprobe")
		}
		status.Error = fmt.Sprintf("missing %s on PATH (install on Windows via: winget install Gyan.FFmpeg)", strings.Join(missing, " and "))
	}
	return status
}

func resolveToolBinary(customPath, name string) string {
	if customPath != "" {
		if info, err := os.Stat(customPath); err == nil && !info.IsDir() {
			return customPath
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	if runtime.GOOS == "windows" {
		exeName := name + ".exe"
		candidates := []string{
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "WinGet", "Links", exeName),
			filepath.Join(os.Getenv("USERPROFILE"), "scoop", "shims", exeName),
			filepath.Join(`C:\ProgramData\chocolatey\bin`, exeName),
			filepath.Join(`C:\ffmpeg\bin`, exeName),
			filepath.Join(os.Getenv("ProgramFiles"), "ffmpeg", "bin", exeName),
		}
		for _, c := range candidates {
			if c == exeName {
				continue
			}
			if info, err := os.Stat(c); err == nil && !info.IsDir() {
				return c
			}
		}
	}
	return ""
}

func readToolVersion(bin string) string {
	cmd := exec.Command(bin, "-version")
	hideSubprocessWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	line := strings.SplitN(string(out), "\n", 2)[0]
	fields := strings.Fields(line)
	for i := 0; i < len(fields)-1; i++ {
		if fields[i] == "version" {
			v := fields[i+1]
			if idx := strings.IndexByte(v, '-'); idx > 0 {
				v = v[:idx]
			}
			return v
		}
	}
	return ""
}

type ffprobeJSON struct {
	Format struct {
		StartTime string `json:"start_time"`
		Duration  string `json:"duration"`
		Size      string `json:"size"`
	} `json:"format"`
	Streams []struct {
		Index      int    `json:"index"`
		CodecType  string `json:"codec_type"`
		CodecName  string `json:"codec_name"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		RFrameRate string `json:"r_frame_rate"`
		Duration   string `json:"duration"`
		StartTime  string `json:"start_time"`
	} `json:"streams"`
}

// ProbeMedia inspects stream metadata and scans packet keyframes via ffprobe.
func ProbeMedia(ctx context.Context, ffprobePath, filePath string) (*MediaInfo, error) {
	if ffprobePath == "" {
		return nil, fmt.Errorf("ffprobe is required on PATH")
	}
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}
	stat, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("cannot access %q: %w", filePath, err)
	}
	if stat.IsDir() {
		return nil, fmt.Errorf("%q is a directory, not a video file", filePath)
	}

	// Step 1: JSON metadata probe
	metaCmd := exec.CommandContext(ctx, ffprobePath,
		"-v", "error",
		"-show_entries", "format=start_time,duration,size",
		"-show_entries", "stream=index,codec_type,codec_name,width,height,r_frame_rate,duration,start_time",
		"-of", "json",
		absPath,
	)
	hideSubprocessWindow(metaCmd)
	var metaErr bytes.Buffer
	metaCmd.Stderr = &metaErr
	metaOut, err := metaCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe metadata failed: %s", strings.TrimSpace(metaErr.String()))
	}

	var parsed ffprobeJSON
	if err := json.Unmarshal(metaOut, &parsed); err != nil {
		return nil, fmt.Errorf("invalid ffprobe JSON: %w", err)
	}

	info := &MediaInfo{
		FilePath: absPath,
		FileName: filepath.Base(absPath),
		FileSize: stat.Size(),
	}

	var hasVideo bool
	var vDur float64
	for _, st := range parsed.Streams {
		switch st.CodecType {
		case "video":
			if !hasVideo {
				hasVideo = true
				info.VideoCodec = st.CodecName
				info.Width = st.Width
				info.Height = st.Height
				info.FPS = parseRateRatio(st.RFrameRate)
				if d, ok := parseProbeFloat(st.Duration); ok && d > 0 {
					vDur = d
				}
			}
		case "audio":
			if !info.HasAudio {
				info.HasAudio = true
				info.AudioCodec = st.CodecName
			}
		}
	}
	if !hasVideo {
		return nil, fmt.Errorf("no video stream found in %q", info.FileName)
	}

	info.DurationSec = vDur
	if info.DurationSec <= 0 {
		if d, ok := parseProbeFloat(parsed.Format.Duration); ok && d > 0 {
			info.DurationSec = d
		}
	}
	if st, ok := parseProbeFloat(parsed.Format.StartTime); ok {
		info.StartTimeSec = st
	}

	// Step 2: Fast zero-decode packet scan for keyframes + cumulative packet byte sizes
	records, cumBytes, err := scanPacketKeyframes(ctx, ffprobePath, absPath, info.StartTimeSec)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		// Fallback for containers that omit packet keyframe flags
		fallbackKFs, fbErr := scanFrameKeyframesFallback(ctx, ffprobePath, absPath, info.StartTimeSec)
		if fbErr == nil {
			for _, kf := range fallbackKFs {
				records = append(records, KeyframeRecord{RelSec: kf, CumBytes: 0})
			}
		}
	}

	// If format/stream metadata lacked duration, infer from packet scan
	if info.DurationSec <= 0 && len(records) > 0 {
		last := records[len(records)-1].RelSec
		if last > 0 {
			info.DurationSec = last + 0.1
		}
	}
	if info.DurationSec <= 0 {
		return nil, fmt.Errorf("invalid or zero video duration in %q", info.FileName)
	}

	// Normalize keyframe timestamps: sorted, deduplicated, anchored at 0.0 if needed
	sort.Slice(records, func(i, j int) bool {
		return records[i].RelSec < records[j].RelSec
	})
	dedup := make([]KeyframeRecord, 0, len(records)+1)
	for _, r := range records {
		if r.RelSec > info.DurationSec {
			continue
		}
		if len(dedup) == 0 || r.RelSec > dedup[len(dedup)-1].RelSec+0.0005 {
			dedup = append(dedup, r)
		}
	}
	if len(dedup) == 0 || dedup[0].RelSec > 0.05 {
		dedup = append([]KeyframeRecord{{RelSec: 0, CumBytes: 0}}, dedup...)
	}

	kfs := make([]float64, len(dedup))
	for i, r := range dedup {
		kfs[i] = r.RelSec
	}

	info.KFRecords = dedup
	info.Keyframes = kfs
	info.TotalBytes = cumBytes
	if info.TotalBytes <= 0 {
		info.TotalBytes = info.FileSize
	}
	return info, nil
}

func scanPacketKeyframes(ctx context.Context, ffprobePath, absPath string, startTimeSec float64) ([]KeyframeRecord, int64, error) {
	cmd := exec.CommandContext(ctx, ffprobePath,
		"-v", "error",
		"-show_entries", "packet=codec_type,pts_time,dts_time,size,flags",
		"-of", "csv=p=0",
		absPath,
	)
	hideSubprocessWindow(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, 0, err
	}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, 0, err
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var records []KeyframeRecord
	var cumBytes int64
	lastRelSec := -1.0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		cols := strings.Split(line, ",")
		if len(cols) < 5 {
			continue
		}
		// Columns in FFmpeg AVPacket field order:
		// cols[0]=codec_type, cols[1]=pts_time, cols[2]=dts_time, cols[3]=size, cols[4]=flags
		pktSize, err := strconv.ParseInt(strings.TrimSpace(cols[3]), 10, 64)
		if err != nil {
			continue
		}
		if cols[0] == "video" && strings.HasPrefix(cols[4], "K") {
			ts, ok := parseProbeFloat(cols[1])
			if !ok {
				ts, ok = parseProbeFloat(cols[2])
			}
			if ok {
				relSec := math.Max(0.0, ts-startTimeSec)
				if len(records) == 0 || relSec > lastRelSec+0.0001 {
					records = append(records, KeyframeRecord{
						RelSec:   relSec,
						CumBytes: cumBytes,
					})
					lastRelSec = relSec
				}
			}
		}
		cumBytes += pktSize
	}

	scanErr := scanner.Err()
	waitErr := cmd.Wait()
	if scanErr != nil {
		return nil, 0, scanErr
	}
	if waitErr != nil {
		return nil, 0, fmt.Errorf("ffprobe packet scan failed: %s", strings.TrimSpace(stderrBuf.String()))
	}
	return records, cumBytes, nil
}

func scanFrameKeyframesFallback(ctx context.Context, ffprobePath, absPath string, startTimeSec float64) ([]float64, error) {
	cmd := exec.CommandContext(ctx, ffprobePath,
		"-v", "error",
		"-select_streams", "v:0",
		"-skip_frame", "nokey",
		"-show_entries", "frame=pts_time",
		"-of", "csv=p=0",
		absPath,
	)
	hideSubprocessWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var res []float64
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		s := strings.TrimRight(strings.TrimSpace(scanner.Text()), ",")
		if pts, ok := parseProbeFloat(s); ok {
			res = append(res, math.Max(0.0, pts-startTimeSec))
		}
	}
	return res, nil
}

// SegmentProbeInfo holds output file verification metrics (bytes, duration, video frame count).
type SegmentProbeInfo struct {
	Bytes  int64   `json:"bytes"`
	Dur    float64 `json:"dur"`
	Frames int     `json:"frames"`
}

// ProbeOutputSegment inspects a newly written output file for duration, bytes, and frame count.
func ProbeOutputSegment(ctx context.Context, ffprobePath, filePath string, fpsHint float64) SegmentProbeInfo {
	var info SegmentProbeInfo
	if st, err := os.Stat(filePath); err == nil {
		info.Bytes = st.Size()
	}
	if ffprobePath == "" {
		return info
	}

	cmd := exec.CommandContext(ctx, ffprobePath,
		"-v", "error",
		"-select_streams", "v:0",
		"-count_packets",
		"-show_entries", "stream=duration,nb_read_packets",
		"-of", "csv=p=0",
		filePath,
	)
	hideSubprocessWindow(cmd)
	if out, err := cmd.Output(); err == nil {
		line := strings.TrimSpace(string(out))
		cols := strings.Split(line, ",")
		if len(cols) >= 2 {
			if d, ok := parseProbeFloat(cols[0]); ok {
				info.Dur = d
			}
			if n, err := strconv.Atoi(strings.TrimSpace(cols[1])); err == nil {
				info.Frames = n
			}
		}
	}

	if fpsHint > 0 && info.Frames > 0 {
		frameDur := float64(info.Frames) / fpsHint
		if info.Dur <= 0 || info.Dur > frameDur*2.5 {
			info.Dur = frameDur
		}
	}
	if info.Dur <= 0 {
		cmd2 := exec.CommandContext(ctx, ffprobePath,
			"-v", "error",
			"-show_entries", "format=duration",
			"-of", "csv=p=0",
			filePath,
		)
		hideSubprocessWindow(cmd2)
		if out, err := cmd2.Output(); err == nil {
			if d, ok := parseProbeFloat(string(out)); ok {
				info.Dur = d
			}
		}
	}
	return info
}

func parseProbeFloat(raw string) (float64, bool) {
	s := strings.TrimSpace(raw)
	if s == "" || strings.EqualFold(s, "N/A") {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

func parseRateRatio(r string) float64 {
	parts := strings.Split(strings.TrimSpace(r), "/")
	if len(parts) != 2 {
		return 0
	}
	num, err1 := strconv.ParseFloat(parts[0], 64)
	den, err2 := strconv.ParseFloat(parts[1], 64)
	if err1 != nil || err2 != nil || den <= 0 {
		return 0
	}
	return num / den
}
