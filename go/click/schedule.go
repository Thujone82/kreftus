package main

import (
	"fmt"
	"strings"
	"time"
)

// RepeatKind specifies how a Schedule repeats after its initial Start time.
type RepeatKind string

const (
	RepeatNone          RepeatKind = "none"
	RepeatEveryInterval RepeatKind = "interval"
	RepeatDaily         RepeatKind = "daily"
	RepeatWeekly        RepeatKind = "weekly"
)

// EndKind specifies when a repeating Schedule stops firing.
type EndKind string

const (
	EndNever       EndKind = "never"
	EndAfterRuns   EndKind = "after_runs"
	EndUntilTime   EndKind = "until_time"
	EndForDuration EndKind = "for_duration"
)

// MissedPolicy controls what happens if a scheduled run time passed while the PC was asleep or Click was closed.
type MissedPolicy string

const (
	MissedSkip        MissedPolicy = "skip"
	MissedRunOnceASAP MissedPolicy = "run_once_asap"
)

// OverlapPolicy controls what happens if a schedule triggers while another macro is already playing.
type OverlapPolicy string

const (
	OverlapQueue OverlapPolicy = "queue"
	OverlapSkip  OverlapPolicy = "skip"
)

// TimeOfDay represents a wall-clock time of day in minutes since midnight (0..1439), e.g. "09:00" -> 540.
type TimeOfDay struct {
	Hour   int `json:"hour"`
	Minute int `json:"minute"`
}

// Minutes returns the minutes since midnight (0..1439).
func (t TimeOfDay) Minutes() int {
	return t.Hour*60 + t.Minute
}

// String formats TimeOfDay as "HH:MM".
func (t TimeOfDay) String() string {
	return fmt.Sprintf("%02d:%02d", t.Hour, t.Minute)
}

// ParseTimeOfDay parses an "HH:MM" 24-hour string.
func ParseTimeOfDay(s string) (*TimeOfDay, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var h, m int
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
		return nil, fmt.Errorf("invalid time of day %q", s)
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return nil, fmt.Errorf("time of day out of range %q", s)
	}
	return &TimeOfDay{Hour: h, Minute: m}, nil
}

// Schedule defines when and how a Macro is automatically played.
type Schedule struct {
	ID            string        `json:"id"`
	MacroID       string        `json:"macroId"`
	Name          string        `json:"name"`
	Enabled       bool          `json:"enabled"`
	Start         time.Time     `json:"start"`
	Repeat        RepeatKind    `json:"repeat"`
	IntervalMs    int64         `json:"intervalMs,omitempty"`
	Weekdays      uint8         `json:"weekdays,omitempty"` // bit 0 = Sunday .. bit 6 = Saturday
	WindowStart   *TimeOfDay    `json:"windowStart,omitempty"`
	WindowEnd     *TimeOfDay    `json:"windowEnd,omitempty"`
	End           EndKind       `json:"end"`
	EndRuns       int           `json:"endRuns,omitempty"`
	EndAt         *time.Time    `json:"endAt,omitempty"`
	EndDurationMs int64         `json:"endDurationMs,omitempty"`
	PlaysPerRun   int           `json:"playsPerRun"`
	PlayGapMs     int64         `json:"playGapMs,omitempty"`
	Missed        MissedPolicy  `json:"missed"`
	Overlap       OverlapPolicy `json:"overlap"`

	// Runtime state (persisted)
	LastRun    *time.Time `json:"lastRun,omitempty"`
	NextRun    *time.Time `json:"nextRun,omitempty"`
	RunCount   int        `json:"runCount"`
	LastResult string     `json:"lastResult,omitempty"`
}

// HistoryEntry records one completed, aborted, skipped, or missed macro run.
type HistoryEntry struct {
	ID         string    `json:"id"`
	Time       time.Time `json:"time"`
	MacroID    string    `json:"macroId"`
	MacroName  string    `json:"macroName"`
	ScheduleID string    `json:"scheduleId,omitempty"`
	SchedName  string    `json:"scheduleName,omitempty"`
	Trigger    string    `json:"trigger"` // "manual" | "schedule"
	Result     string    `json:"result"`  // "done" | "aborted" | "skipped" | "missed" | "error"
	Detail     string    `json:"detail,omitempty"`
}

// ComputeNextRun calculates the next run time strictly after `after` (or at
// `s.Start` if it has not yet run and `s.Start` >= `after`).
// Returns nil if the schedule has finished or cannot produce a valid future time.
func ComputeNextRun(s Schedule, after time.Time) *time.Time {
	if !s.Enabled {
		return nil
	}
	if s.End == EndAfterRuns && s.EndRuns > 0 && s.RunCount >= s.EndRuns {
		return nil
	}

	loc := s.Start.Location()
	after = after.In(loc)
	start := s.Start

	// Check end time boundary
	var cutoff *time.Time
	switch s.End {
	case EndUntilTime:
		if s.EndAt != nil {
			t := s.EndAt.In(loc)
			cutoff = &t
		}
	case EndForDuration:
		if s.EndDurationMs > 0 {
			t := start.Add(time.Duration(s.EndDurationMs) * time.Millisecond)
			cutoff = &t
		}
	}

	var candidate time.Time

	switch s.Repeat {
	case RepeatNone, "":
		if s.RunCount > 0 {
			return nil
		}
		if !start.Before(after) {
			candidate = start
		} else if s.Missed == MissedRunOnceASAP && s.LastRun == nil {
			candidate = after
		} else {
			return nil
		}

	case RepeatEveryInterval:
		iv := time.Duration(s.IntervalMs) * time.Millisecond
		if iv < time.Second {
			iv = time.Minute
		}
		if !start.Before(after) {
			candidate = start
		} else {
			if s.Missed == MissedRunOnceASAP && s.LastRun == nil {
				candidate = after
			} else if s.Missed == MissedRunOnceASAP && s.LastRun != nil && after.Sub(*s.LastRun) >= iv {
				candidate = after
			} else {
				elapsed := after.Sub(start)
				steps := elapsed/iv + 1
				candidate = start.Add(steps * iv)
			}
		}
		candidate = adjustForWindowAndDays(candidate, s, iv)

	case RepeatDaily:
		if !start.Before(after) {
			candidate = start
		} else if s.Missed == MissedRunOnceASAP && shouldRunMissedDaily(s, after) {
			candidate = after
		} else {
			candidate = timeOnDay(after, start.Hour(), start.Minute(), start.Second())
			if !candidate.After(after) {
				candidate = timeOnDay(after.AddDate(0, 0, 1), start.Hour(), start.Minute(), start.Second())
			}
		}
		candidate = adjustForWindowAndDays(candidate, s, 24*time.Hour)

	case RepeatWeekly:
		mask := s.Weekdays
		if mask == 0 {
			mask = 1 << uint(start.Weekday())
		}
		if !start.Before(after) && (mask&(1<<uint(start.Weekday()))) != 0 {
			candidate = start
		} else {
			found := false
			for d := 0; d < 8; d++ {
				day := after.AddDate(0, 0, d)
				if (mask & (1 << uint(day.Weekday()))) == 0 {
					continue
				}
				t := timeOnDay(day, start.Hour(), start.Minute(), start.Second())
				if t.Before(start) {
					continue
				}
				if t.After(after) {
					candidate = t
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		}
		candidate = adjustForWindowAndDays(candidate, s, 24*time.Hour)
	}

	if candidate.IsZero() {
		return nil
	}
	if cutoff != nil && candidate.After(*cutoff) {
		return nil
	}
	return &candidate
}

func shouldRunMissedDaily(s Schedule, now time.Time) bool {
	todayTarget := timeOnDay(now, s.Start.Hour(), s.Start.Minute(), s.Start.Second())
	if now.Before(todayTarget) || todayTarget.Before(s.Start) {
		return false
	}
	if s.LastRun == nil {
		return true
	}
	return s.LastRun.Before(todayTarget)
}

func timeOnDay(day time.Time, hr, min, sec int) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), hr, min, sec, 0, day.Location())
}

// adjustForWindowAndDays advances `t` until it falls within the allowed active
// time-of-day window and weekday mask (if configured).
func adjustForWindowAndDays(t time.Time, s Schedule, step time.Duration) time.Time {
	if step <= 0 {
		step = time.Minute
	}
	for i := 0; i < 366*2; i++ {
		if s.Weekdays != 0 && (s.Weekdays&(1<<uint(t.Weekday()))) == 0 {
			nextDay := timeOnDay(t.AddDate(0, 0, 1), s.Start.Hour(), s.Start.Minute(), s.Start.Second())
			if s.WindowStart != nil {
				nextDay = timeOnDay(t.AddDate(0, 0, 1), s.WindowStart.Hour, s.WindowStart.Minute, 0)
			}
			t = nextDay
			continue
		}
		if s.WindowStart != nil && s.WindowEnd != nil {
			curMin := t.Hour()*60 + t.Minute()
			wStart := s.WindowStart.Minutes()
			wEnd := s.WindowEnd.Minutes()
			if wStart <= wEnd {
				if curMin < wStart {
					t = timeOnDay(t, s.WindowStart.Hour, s.WindowStart.Minute, 0)
					continue
				}
				if curMin > wEnd {
					t = timeOnDay(t.AddDate(0, 0, 1), s.WindowStart.Hour, s.WindowStart.Minute, 0)
					continue
				}
			}
		}
		return t
	}
	return time.Time{}
}

// PreviewNextRuns returns up to `count` upcoming run timestamps starting from `now`.
func PreviewNextRuns(s Schedule, now time.Time, count int) []time.Time {
	if count <= 0 {
		count = 5
	}
	sim := s
	sim.Enabled = true
	var out []time.Time
	cursor := now
	for i := 0; i < count; i++ {
		next := ComputeNextRun(sim, cursor)
		if next == nil {
			break
		}
		out = append(out, *next)
		sim.RunCount++
		t := *next
		sim.LastRun = &t
		if sim.Repeat == RepeatNone || sim.Repeat == "" {
			break
		}
		cursor = next.Add(time.Second)
	}
	return out
}

// DescribeSchedule produces a human-readable summary of a Schedule.
func DescribeSchedule(s Schedule) string {
	var parts []string
	switch s.Repeat {
	case RepeatNone, "":
		parts = append(parts, "Once at "+s.Start.Format("Jan 2, 3:04 PM"))
	case RepeatEveryInterval:
		iv := time.Duration(s.IntervalMs) * time.Millisecond
		parts = append(parts, "Every "+formatShortDuration(iv))
	case RepeatDaily:
		parts = append(parts, "Daily at "+s.Start.Format("3:04 PM"))
	case RepeatWeekly:
		days := formatWeekdays(s.Weekdays)
		if days == "" {
			days = s.Start.Weekday().String()[:3]
		}
		parts = append(parts, fmt.Sprintf("Weekly (%s) at %s", days, s.Start.Format("3:04 PM")))
	}
	if s.WindowStart != nil && s.WindowEnd != nil {
		parts = append(parts, fmt.Sprintf("%s–%s", s.WindowStart.String(), s.WindowEnd.String()))
	}
	switch s.End {
	case EndAfterRuns:
		if s.EndRuns > 0 {
			parts = append(parts, fmt.Sprintf("%d/%d runs", s.RunCount, s.EndRuns))
		}
	case EndUntilTime:
		if s.EndAt != nil {
			parts = append(parts, "until "+s.EndAt.Format("Jan 2, 3:04 PM"))
		}
	case EndForDuration:
		if s.EndDurationMs > 0 {
			parts = append(parts, "for "+formatShortDuration(time.Duration(s.EndDurationMs)*time.Millisecond))
		}
	}
	if s.PlaysPerRun > 1 {
		parts = append(parts, fmt.Sprintf("×%d per run", s.PlaysPerRun))
	}
	return strings.Join(parts, " · ")
}

func formatWeekdays(mask uint8) string {
	names := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	if mask == 0b0111110 {
		return "Mon–Fri"
	}
	if mask == 0b1111111 {
		return "Every day"
	}
	var out []string
	for i, n := range names {
		if (mask & (1 << uint(i))) != 0 {
			out = append(out, n)
		}
	}
	return strings.Join(out, ", ")
}

func formatShortDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	var b strings.Builder
	if h > 0 {
		fmt.Fprintf(&b, "%dh", h)
	}
	if m > 0 {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%dm", m)
	}
	if s > 0 || b.Len() == 0 {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%ds", s)
	}
	return b.String()
}

// NextScheduleName returns a collision-free schedule name based on baseName.
// If baseName is unused (ignoring the schedule with excludeID), it returns baseName.
// Otherwise it strips any trailing " - NN" suffix and returns "<root> - 02", "<root> - 03",
// etc., picking the lowest unused number >= 2.
func NextScheduleName(existing []Schedule, baseName string, excludeID string) string {
	base := strings.TrimSpace(baseName)
	if base == "" {
		base = "Scheduled Macro"
	}
	taken := func(candidate string) bool {
		for _, sc := range existing {
			if excludeID != "" && sc.ID == excludeID {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(sc.Name), candidate) {
				return true
			}
		}
		return false
	}
	if !taken(base) {
		return base
	}
	root := stripNumericSuffix(base)
	for n := 2; ; n++ {
		cand := fmt.Sprintf("%s - %02d", root, n)
		if !taken(cand) {
			return cand
		}
	}
}

func stripNumericSuffix(s string) string {
	idx := strings.LastIndex(s, " - ")
	if idx <= 0 {
		return s
	}
	tail := strings.TrimSpace(s[idx+3:])
	if len(tail) >= 2 && digitsOnly(tail) {
		return strings.TrimSpace(s[:idx])
	}
	return s
}
