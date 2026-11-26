package main

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Segment describes one contiguous slice of the source timeline.
type Segment struct {
	Index    int     `json:"index"` // 1-based display index
	Start    float64 `json:"start"`
	End      float64 `json:"end"`
	Duration float64 `json:"duration"`
	Keep     bool    `json:"keep"`
}

// ProjectState manages markers, keep/drop toggles, and output mode for a loaded video.
type ProjectState struct {
	FilePath  string     `json:"filePath"`
	Media     *MediaInfo `json:"media"`
	Markers   []float64  `json:"markers"`   // sorted interior keyframe cut timestamps
	KeepFlags []bool     `json:"keepFlags"` // len == len(Markers)+1
	Mode      string     `json:"mode"`      // "join" | "split"
}

const (
	markerDedupEps = 0.002
	edgeMarginSec  = 0.02
)

// NewProject initializes a single-segment kept project for the probed video.
func NewProject(filePath string, media *MediaInfo, mode string) *ProjectState {
	if mode != "split" {
		mode = "join"
	}
	return &ProjectState{
		FilePath:  filePath,
		Media:     media,
		Markers:   nil,
		KeepFlags: []bool{true},
		Mode:      mode,
	}
}

// SetMode changes export mode between "join" and "split".
func (p *ProjectState) SetMode(mode string) {
	if mode == "split" {
		p.Mode = "split"
	} else {
		p.Mode = "join"
	}
}

// Segments materializes all timeline segments from Markers and KeepFlags.
func (p *ProjectState) Segments() []Segment {
	if p == nil || p.Media == nil || p.Media.DurationSec <= 0 {
		return nil
	}
	total := p.Media.DurationSec
	bounds := make([]float64, 0, len(p.Markers)+2)
	bounds = append(bounds, 0)
	bounds = append(bounds, p.Markers...)
	bounds = append(bounds, total)

	segs := make([]Segment, 0, len(bounds)-1)
	for i := 0; i < len(bounds)-1; i++ {
		keep := true
		if i < len(p.KeepFlags) {
			keep = p.KeepFlags[i]
		}
		start := bounds[i]
		end := bounds[i+1]
		dur := end - start
		if dur < 0 {
			dur = 0
		}
		segs = append(segs, Segment{
			Index:    i + 1,
			Start:    start,
			End:      end,
			Duration: dur,
			Keep:     keep,
		})
	}
	return segs
}

// KeptSegments returns only segments marked Keep == true.
func (p *ProjectState) KeptSegments() []Segment {
	all := p.Segments()
	kept := make([]Segment, 0, len(all))
	for _, s := range all {
		if s.Keep && s.Duration > 0.001 {
			kept = append(kept, s)
		}
	}
	return kept
}

// AddMarkerAt snaps rawSec to the nearest interior keyframe and inserts a cut point.
func (p *ProjectState) AddMarkerAt(rawSec float64) (snapped float64, added bool) {
	if p == nil || p.Media == nil {
		return 0, false
	}
	kf, _, ok := SnapToKeyframe(p.Media.Keyframes, rawSec, p.Media.DurationSec)
	if !ok {
		return 0, false
	}
	for _, m := range p.Markers {
		if math.Abs(m-kf) <= markerDedupEps {
			return m, false
		}
	}

	idx := sort.SearchFloat64s(p.Markers, kf)
	newMarkers := make([]float64, len(p.Markers)+1)
	copy(newMarkers[:idx], p.Markers[:idx])
	newMarkers[idx] = kf
	copy(newMarkers[idx+1:], p.Markers[idx:])

	keepVal := true
	if idx < len(p.KeepFlags) {
		keepVal = p.KeepFlags[idx]
	}
	newKeep := make([]bool, len(p.KeepFlags)+1)
	copy(newKeep[:idx+1], p.KeepFlags[:idx+1])
	newKeep[idx+1] = keepVal
	if idx+1 < len(p.KeepFlags) {
		copy(newKeep[idx+2:], p.KeepFlags[idx+1:])
	}

	p.Markers = newMarkers
	p.KeepFlags = newKeep
	return kf, true
}

// MoveMarker drags markerIdx to the nearest interior keyframe strictly between neighbors.
func (p *ProjectState) MoveMarker(markerIdx int, rawSec float64) (snapped float64, moved bool) {
	if p == nil || p.Media == nil || markerIdx < 0 || markerIdx >= len(p.Markers) {
		return 0, false
	}
	minBound := edgeMarginSec
	if markerIdx > 0 {
		minBound = p.Markers[markerIdx-1] + markerDedupEps
	}
	maxBound := p.Media.DurationSec - edgeMarginSec
	if markerIdx+1 < len(p.Markers) {
		maxBound = p.Markers[markerIdx+1] - markerDedupEps
	}
	if maxBound <= minBound {
		return p.Markers[markerIdx], false
	}

	bestKf := -1.0
	bestDist := math.MaxFloat64
	for _, kf := range p.Media.Keyframes {
		if kf <= minBound || kf >= maxBound {
			continue
		}
		d := math.Abs(kf - rawSec)
		if d < bestDist {
			bestDist = d
			bestKf = kf
		}
	}
	if bestKf < 0 {
		return p.Markers[markerIdx], false
	}
	if math.Abs(bestKf-p.Markers[markerIdx]) <= 0.0005 {
		return p.Markers[markerIdx], false
	}
	p.Markers[markerIdx] = bestKf
	return bestKf, true
}

// RemoveMarker deletes markerIdx and merges the two adjacent segments.
func (p *ProjectState) RemoveMarker(markerIdx int) bool {
	if p == nil || markerIdx < 0 || markerIdx >= len(p.Markers) {
		return false
	}
	mergedKeep := p.KeepFlags[markerIdx] || p.KeepFlags[markerIdx+1]
	p.Markers = append(p.Markers[:markerIdx], p.Markers[markerIdx+1:]...)

	newKeep := make([]bool, 0, len(p.KeepFlags)-1)
	newKeep = append(newKeep, p.KeepFlags[:markerIdx]...)
	newKeep = append(newKeep, mergedKeep)
	if markerIdx+2 < len(p.KeepFlags) {
		newKeep = append(newKeep, p.KeepFlags[markerIdx+2:]...)
	}
	p.KeepFlags = newKeep
	return true
}

// ClearMarkers removes all interior cuts and resets to a single kept segment.
func (p *ProjectState) ClearMarkers() {
	if p == nil {
		return
	}
	p.Markers = nil
	p.KeepFlags = []bool{true}
}

// ToggleSegment flips Keep/Drop for 0-based segment index segIdx.
func (p *ProjectState) ToggleSegment(segIdx int) bool {
	if p == nil || segIdx < 0 || segIdx >= len(p.KeepFlags) {
		return false
	}
	p.KeepFlags[segIdx] = !p.KeepFlags[segIdx]
	return true
}

// SetAllKeep sets all segments to Keep=keep.
func (p *ProjectState) SetAllKeep(keep bool) {
	if p == nil {
		return
	}
	for i := range p.KeepFlags {
		p.KeepFlags[i] = keep
	}
}

// InvertKeep flips Keep/Drop for every segment.
func (p *ProjectState) InvertKeep() {
	if p == nil {
		return
	}
	for i := range p.KeepFlags {
		p.KeepFlags[i] = !p.KeepFlags[i]
	}
}

// ApplyKeepRanges configures markers and keep flags from explicit [start, end] time ranges.
// All boundaries snap cleanly to keyframes.
func (p *ProjectState) ApplyKeepRanges(ranges []TimeRange) error {
	if p == nil || p.Media == nil {
		return fmt.Errorf("no video loaded")
	}
	total := p.Media.DurationSec
	if len(ranges) == 0 {
		return fmt.Errorf("at least one keep range is required")
	}

	type snappedRange struct {
		start float64
		end   float64
	}
	snapped := make([]snappedRange, 0, len(ranges))
	for _, r := range ranges {
		s := SnapStartKeyframe(p.Media.Keyframes, r.Start, r.End-0.02)
		var e float64
		if r.End >= total-edgeMarginSec {
			e = total
		} else if kf, _, ok := SnapToKeyframe(p.Media.Keyframes, r.End, total); ok {
			e = kf
		} else {
			e = total
		}
		if e <= s+0.01 {
			return fmt.Errorf("keyframe-snapped range [%.2fs .. %.2fs] has zero duration", s, e)
		}
		snapped = append(snapped, snappedRange{start: s, end: e})
	}

	sort.Slice(snapped, func(i, j int) bool {
		if snapped[i].start == snapped[j].start {
			return snapped[i].end < snapped[j].end
		}
		return snapped[i].start < snapped[j].start
	})

	// Merge overlapping or adjacent snapped ranges
	merged := make([]snappedRange, 0, len(snapped))
	for _, sr := range snapped {
		if len(merged) == 0 || sr.start > merged[len(merged)-1].end+markerDedupEps {
			merged = append(merged, sr)
		} else if sr.end > merged[len(merged)-1].end {
			merged[len(merged)-1].end = sr.end
		}
	}

	var markers []float64
	addBoundary := func(t float64) {
		if t <= edgeMarginSec || t >= total-edgeMarginSec {
			return
		}
		for _, m := range markers {
			if math.Abs(m-t) <= markerDedupEps {
				return
			}
		}
		markers = append(markers, t)
	}
	for _, mr := range merged {
		addBoundary(mr.start)
		addBoundary(mr.end)
	}
	sort.Float64s(markers)

	p.Markers = markers
	p.KeepFlags = make([]bool, len(markers)+1)
	segs := p.Segments()
	for i, seg := range segs {
		mid := 0.5 * (seg.Start + seg.End)
		keep := false
		for _, mr := range merged {
			if mid >= mr.start && mid <= mr.end {
				keep = true
				break
			}
		}
		p.KeepFlags[i] = keep
	}
	return nil
}

// ComputeBalancedCutKeyframes selects keyframe indices in kfRecords (strictly increasing)
// that equalize output segment byte sizes across `parts` segments.
// Ported from splitvid.ps1's greedy + 6-pass coordinate descent balancer.
func ComputeBalancedCutKeyframes(kfRecords []KeyframeRecord, totalBytes int64, parts int) ([]int, error) {
	kfCount := len(kfRecords)
	if kfCount <= 1 {
		return nil, fmt.Errorf("video has only %d keyframe; lossless stream copy cannot split mid-stream without transcoding", kfCount)
	}
	if parts < 2 {
		return nil, fmt.Errorf("parts must be >= 2")
	}

	internalCount := kfCount - 1
	cutCount := parts - 1
	if internalCount < cutCount {
		cutCount = internalCount
	}

	cutIndices := make([]int, cutCount)
	prevKfIdx := 0
	for i := 0; i < cutCount; i++ {
		targetBytes := float64(totalBytes) * (float64(i+1) / float64(cutCount+1))
		minIdx := prevKfIdx + 1
		maxIdx := kfCount - (cutCount - i)

		bestIdx := minIdx
		bestDiff := math.MaxFloat64
		for k := minIdx; k <= maxIdx; k++ {
			b := float64(kfRecords[k].CumBytes)
			diff := math.Abs(b - targetBytes)
			if diff < bestDiff {
				bestDiff = diff
				bestIdx = k
			} else if b > targetBytes {
				break
			}
		}
		cutIndices[i] = bestIdx
		prevKfIdx = bestIdx
	}

	idealSegBytes := float64(totalBytes) / float64(cutCount+1)
	for pass := 0; pass < 6; pass++ {
		moved := false
		for i := 0; i < cutCount; i++ {
			leftKfIdx := 0
			if i > 0 {
				leftKfIdx = cutIndices[i-1]
			}
			rightKfIdx := -1
			if i < cutCount-1 {
				rightKfIdx = cutIndices[i+1]
			}

			leftBytes := float64(kfRecords[leftKfIdx].CumBytes)
			rightBytes := float64(totalBytes)
			if rightKfIdx >= 0 {
				rightBytes = float64(kfRecords[rightKfIdx].CumBytes)
			}

			lo := leftKfIdx + 1
			hi := kfCount - 1
			if rightKfIdx >= 0 {
				hi = rightKfIdx - 1
			}
			if lo >= hi {
				continue
			}

			bestK := cutIndices[i]
			bestCost := math.MaxFloat64
			for k := lo; k <= hi; k++ {
				b := float64(kfRecords[k].CumBytes)
				s1 := b - leftBytes
				s2 := rightBytes - b
				cost := math.Pow(s1-idealSegBytes, 2) + math.Pow(s2-idealSegBytes, 2) + 0.15*math.Pow(s1-s2, 2)
				if cost < bestCost {
					bestCost = cost
					bestK = k
				}
			}
			if bestK != cutIndices[i] {
				cutIndices[i] = bestK
				moved = true
			}
		}
		if !moved {
			break
		}
	}

	return cutIndices, nil
}

// SnapToKeyframe returns the nearest interior keyframe in (edgeMarginSec, totalSec-edgeMarginSec).
func SnapToKeyframe(keyframes []float64, targetSec, totalSec float64) (snapped float64, idx int, ok bool) {
	if len(keyframes) == 0 || totalSec <= 2*edgeMarginSec {
		return 0, -1, false
	}
	lo := edgeMarginSec
	hi := totalSec - edgeMarginSec
	if targetSec < lo {
		targetSec = lo
	}
	if targetSec > hi {
		targetSec = hi
	}

	pos := sort.SearchFloat64s(keyframes, targetSec)
	bestIdx := -1
	bestDist := math.MaxFloat64
	for _, cand := range []int{pos - 1, pos, pos + 1} {
		if cand < 0 || cand >= len(keyframes) {
			continue
		}
		kf := keyframes[cand]
		if kf <= lo || kf >= hi {
			continue
		}
		d := math.Abs(kf - targetSec)
		if d < bestDist {
			bestDist = d
			bestIdx = cand
		}
	}
	if bestIdx >= 0 {
		return keyframes[bestIdx], bestIdx, true
	}

	// Fallback linear scan inside interior bounds
	for i, kf := range keyframes {
		if kf <= lo || kf >= hi {
			continue
		}
		d := math.Abs(kf - targetSec)
		if d < bestDist {
			bestDist = d
			bestIdx = i
		}
	}
	if bestIdx >= 0 {
		return keyframes[bestIdx], bestIdx, true
	}
	return 0, -1, false
}

// SnapStartKeyframe snaps a segment start timestamp to the nearest keyframe strictly before maxEndSec.
func SnapStartKeyframe(keyframes []float64, targetSec, maxEndSec float64) float64 {
	if targetSec <= 0.001 || len(keyframes) == 0 {
		return 0
	}
	limit := maxEndSec - 0.05
	if limit <= 0 {
		return 0
	}
	best := 0.0
	bestDist := math.MaxFloat64
	for _, kf := range keyframes {
		if kf >= limit {
			break
		}
		d := math.Abs(kf - targetSec)
		if d < bestDist {
			bestDist = d
			best = kf
		}
	}
	return best
}

// SplitBaseExt splits a file path into (dir, baseWithoutExt, ext).
func SplitBaseExt(path string) (dir, base, ext string) {
	dir = filepath.Dir(path)
	filename := filepath.Base(path)
	ext = filepath.Ext(filename)
	base = strings.TrimSuffix(filename, ext)
	return dir, base, ext
}

// SuggestedJoinPath returns `<dir>\<base>_edit<ext>`.
func SuggestedJoinPath(srcPath string) string {
	dir, base, ext := SplitBaseExt(srcPath)
	return filepath.Join(dir, base+"_edit"+ext)
}

// SuggestedSplitDialogPath returns `<dir>\<base>_01<ext>`.
func SuggestedSplitDialogPath(srcPath string) string {
	dir, base, ext := SplitBaseExt(srcPath)
	return filepath.Join(dir, base+"_01"+ext)
}

var trailingSplitIndexRe = regexp.MustCompile(`_\d{2,}$`)

// DeriveSplitPaths converts a chosen Save-dialog path (e.g. `clip_01.mp4` or `custom.mp4`)
// into sequential output paths `<dir>\<cleanBase>_01<ext>`, `<dir>\<cleanBase>_02<ext>`, etc.
func DeriveSplitPaths(chosenPath string, count int) []string {
	if count <= 0 {
		return nil
	}
	dir, base, ext := SplitBaseExt(chosenPath)
	cleanBase := trailingSplitIndexRe.ReplaceAllString(base, "")
	if cleanBase == "" {
		cleanBase = base
	}
	paths := make([]string, count)
	for i := 0; i < count; i++ {
		paths[i] = filepath.Join(dir, fmt.Sprintf("%s_%02d%s", cleanBase, i+1, ext))
	}
	return paths
}

// SuggestedSuffixPath returns `<dir>\<base>_<suffix><ext>`.
func SuggestedSuffixPath(srcPath, suffix string) string {
	dir, base, ext := SplitBaseExt(srcPath)
	return filepath.Join(dir, base+"_"+suffix+ext)
}
