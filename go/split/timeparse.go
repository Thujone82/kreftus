package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ParseDurationSpec parses duration or timestamp strings such as:
// "1m3.2s", "45s", "2m", "1h2m3.4s", "1:03.2", "0:01:03.2", or "63.2".
func ParseDurationSpec(spec string) (float64, error) {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return 0, fmt.Errorf("time specification cannot be empty")
	}
	if strings.HasPrefix(trimmed, "-") {
		return 0, fmt.Errorf("negative time %q is not supported", spec)
	}

	clean := strings.ToLower(strings.Join(strings.Fields(trimmed), ""))
	if strings.HasPrefix(clean, "-") {
		return 0, fmt.Errorf("negative time %q is not supported", spec)
	}

	// Colon format: [HH:]MM:SS(.fff)
	if strings.Contains(clean, ":") {
		parts := strings.Split(clean, ":")
		if len(parts) < 2 || len(parts) > 3 {
			return 0, fmt.Errorf("invalid time format %q", spec)
		}
		var h, m, s float64
		var err error
		if len(parts) == 3 {
			if h, err = strconv.ParseFloat(parts[0], 64); err != nil || h < 0 || math.IsNaN(h) || math.IsInf(h, 0) {
				return 0, fmt.Errorf("invalid hours in %q", spec)
			}
			if m, err = strconv.ParseFloat(parts[1], 64); err != nil || m < 0 || math.IsNaN(m) || math.IsInf(m, 0) {
				return 0, fmt.Errorf("invalid minutes in %q", spec)
			}
			if s, err = strconv.ParseFloat(parts[2], 64); err != nil || s < 0 || math.IsNaN(s) || math.IsInf(s, 0) {
				return 0, fmt.Errorf("invalid seconds in %q", spec)
			}
		} else {
			if m, err = strconv.ParseFloat(parts[0], 64); err != nil || m < 0 || math.IsNaN(m) || math.IsInf(m, 0) {
				return 0, fmt.Errorf("invalid minutes in %q", spec)
			}
			if s, err = strconv.ParseFloat(parts[1], 64); err != nil || s < 0 || math.IsNaN(s) || math.IsInf(s, 0) {
				return 0, fmt.Errorf("invalid seconds in %q", spec)
			}
		}
		return h*3600 + m*60 + s, nil
	}

	// Unit suffix format: [Xh][Ym][Zs]
	if strings.ContainsAny(clean, "hms") {
		total, ok := parseHMS(clean)
		if !ok {
			return 0, fmt.Errorf("invalid time format %q (expected e.g. 1m3.2s, 45s, 1h5m10s, 1:03.2, or 63.2)", spec)
		}
		return total, nil
	}

	// Plain numeric seconds
	val, err := strconv.ParseFloat(clean, 64)
	if err != nil || val < 0 || math.IsNaN(val) || math.IsInf(val, 0) {
		return 0, fmt.Errorf("invalid time format %q (expected e.g. 1m3.2s, 45s, 1:03.2, or 63.2)", spec)
	}
	return val, nil
}

func parseHMS(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}
	i := 0
	lastOrder := -1
	total := 0.0
	matchedAny := false

	for i < len(s) {
		start := i
		dots := 0
		for i < len(s) && ((s[i] >= '0' && s[i] <= '9') || s[i] == '.') {
			if s[i] == '.' {
				dots++
				if dots > 1 {
					return 0, false
				}
			}
			i++
		}
		if i == start || i >= len(s) {
			return 0, false
		}
		numStr := s[start:i]
		if numStr == "." {
			return 0, false
		}
		num, err := strconv.ParseFloat(numStr, 64)
		if err != nil || num < 0 || math.IsNaN(num) || math.IsInf(num, 0) {
			return 0, false
		}

		var order int
		var mult float64
		switch s[i] {
		case 'h':
			order, mult = 0, 3600
		case 'm':
			order, mult = 1, 60
		case 's':
			order, mult = 2, 1
		default:
			return 0, false
		}
		if order <= lastOrder {
			return 0, false
		}
		lastOrder = order
		total += num * mult
		matchedAny = true
		i++
	}
	return total, matchedAny
}

// TimeRange represents a half-open / closed playback interval [Start, End] in seconds.
type TimeRange struct {
	Start float64
	End   float64
}

// ParseRangeList parses comma-separated ranges such as:
// "0:10-1:25,2m-end" or "start..45s,1m30s..2m".
func ParseRangeList(spec string, totalDur float64) ([]TimeRange, error) {
	parts := strings.Split(spec, ",")
	var out []TimeRange
	for _, raw := range parts {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		var left, right string
		if idx := strings.Index(item, ".."); idx >= 0 {
			left = strings.TrimSpace(item[:idx])
			right = strings.TrimSpace(item[idx+2:])
		} else if idx := strings.Index(item, "-"); idx >= 0 {
			left = strings.TrimSpace(item[:idx])
			right = strings.TrimSpace(item[idx+1:])
		} else {
			return nil, fmt.Errorf("invalid range %q (expected start-end or start..end)", item)
		}

		startSec, err := parseBound(left, totalDur, true)
		if err != nil {
			return nil, fmt.Errorf("invalid range start in %q: %w", item, err)
		}
		endSec, err := parseBound(right, totalDur, false)
		if err != nil {
			return nil, fmt.Errorf("invalid range end in %q: %w", item, err)
		}
		if endSec <= startSec {
			return nil, fmt.Errorf("range %q end (%.2fs) must be greater than start (%.2fs)", item, endSec, startSec)
		}
		out = append(out, TimeRange{Start: startSec, End: endSec})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no valid ranges found in %q", spec)
	}
	return out, nil
}

func parseBound(tok string, totalDur float64, isStart bool) (float64, error) {
	low := strings.ToLower(strings.TrimSpace(tok))
	switch low {
	case "", "start":
		if isStart {
			return 0, nil
		}
		return 0, fmt.Errorf("end bound cannot be %q", tok)
	case "end", "eof":
		if totalDur <= 0 {
			return 0, fmt.Errorf("cannot resolve %q without known video duration", tok)
		}
		return totalDur, nil
	default:
		v, err := ParseDurationSpec(tok)
		if err != nil {
			return 0, err
		}
		if totalDur > 0 && v > totalDur {
			return totalDur, nil
		}
		return v, nil
	}
}

// FormatTimecode formats seconds as MM:SS.cc or HH:MM:SS.cc.
func FormatTimecode(sec float64) string {
	if sec < 0 || math.IsNaN(sec) || math.IsInf(sec, 0) {
		sec = 0
	}
	totalCentis := int64(math.Round(sec * 100))
	cs := totalCentis % 100
	totalSecs := totalCentis / 100
	s := totalSecs % 60
	totalMins := totalSecs / 60
	m := totalMins % 60
	h := totalMins / 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d.%02d", h, m, s, cs)
	}
	return fmt.Sprintf("%02d:%02d.%02d", m, s, cs)
}

// FormatBytes formats a byte count in human-readable units.
func FormatBytes(bytes int64) string {
	abs := bytes
	if abs < 0 {
		abs = -abs
	}
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case abs >= gb:
		return fmt.Sprintf("%.2f GB (%s B)", float64(bytes)/float64(gb), formatCommaInt(bytes))
	case abs >= mb:
		return fmt.Sprintf("%.2f MB (%s B)", float64(bytes)/float64(mb), formatCommaInt(bytes))
	case abs >= kb:
		return fmt.Sprintf("%.1f KB (%s B)", float64(bytes)/float64(kb), formatCommaInt(bytes))
	default:
		return fmt.Sprintf("%s B", formatCommaInt(bytes))
	}
}

// FormatSignedDelta formats a signed byte delta with leading +/-.
func FormatSignedDelta(delta int64, unit string) string {
	if delta > 0 {
		return fmt.Sprintf("+%s %s", formatCommaInt(delta), unit)
	}
	if delta < 0 {
		return fmt.Sprintf("%s %s", formatCommaInt(delta), unit)
	}
	return "0 " + unit
}

func formatCommaInt(v int64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	s := strconv.FormatInt(v, 10)
	if len(s) <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	rem := len(s) % 3
	if rem > 0 {
		b.WriteString(s[:rem])
		if len(s) > rem {
			b.WriteByte(',')
		}
	}
	for i := rem; i < len(s); i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < len(s) {
			b.WriteByte(',')
		}
	}
	return b.String()
}
