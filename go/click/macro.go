package main

import (
	"strconv"
	"strings"
	"time"
)

// Kind is one recorded mouse or keyboard action.
type Kind int

const (
	Move Kind = iota
	LeftDown
	LeftUp
	RightDown
	RightUp
	MiddleDown
	MiddleUp
	Wheel
	KeyDown
	KeyUp
)

const (
	maxRecordDuration = 10 * time.Minute
	maxRecordEvents   = 100000
	maxCountdown      = 24 * time.Hour
)

var errDelay = errString("Enter a duration like 5s, 80m, or 1h20m10s, up to 24h.")

type errString string

func (e errString) Error() string { return string(e) }

// Event is one step of a macro. Offset is the time since the first event.
type Event struct {
	Offset time.Duration
	Kind   Kind
	X, Y   int32
	Wheel  int32
}

// RawEvent is a hook sample. Tick is the millisecond timestamp from the hook.
// VK is the virtual-key code for KeyDown and KeyUp samples.
type RawEvent struct {
	Tick  uint32
	Kind  Kind
	X, Y  int32
	Wheel int32
	VK    uint16
}

// parseDelay reads a countdown. A bare number is seconds. Otherwise the text
// is a sequence of counts and units, such as 1h20m10s, 80m, or 1200s.
func parseDelay(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, errDelay
	}
	if digitsOnly(s) {
		return secondsDelay(s)
	}
	var secs uint64
	saw := false
	for i := 0; i < len(s); {
		if s[i] == ' ' || s[i] == '\t' {
			i++
			continue
		}
		j := i
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		if j == i || j >= len(s) {
			return 0, errDelay
		}
		n, err := strconv.ParseUint(s[i:j], 10, 32)
		if err != nil || n > 100000 {
			return 0, errDelay
		}
		var mult uint64
		switch s[j] {
		case 'h':
			mult = 3600
		case 'm':
			mult = 60
		case 's':
			mult = 1
		default:
			return 0, errDelay
		}
		secs += n * mult
		if secs > uint64(maxCountdown/time.Second) {
			return 0, errDelay
		}
		saw = true
		i = j + 1
	}
	if !saw {
		return 0, errDelay
	}
	return time.Duration(secs) * time.Second, nil
}

func digitsOnly(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func secondsDelay(s string) (time.Duration, error) {
	n, err := strconv.ParseUint(s, 10, 32)
	if err != nil || n > uint64(maxCountdown/time.Second) {
		return 0, errDelay
	}
	return time.Duration(n) * time.Second, nil
}

// overRecordLimit reports whether a recording has hit the time or event cap.
func overRecordLimit(elapsed time.Duration, events int) bool {
	return elapsed >= maxRecordDuration || events >= maxRecordEvents
}

// Build keeps clicks and the wheel. The path the pointer took is discarded.
// Offset is the time since recording started, which is how long it took to
// reach that click. Playback draws a straight line across the gap.
func Build(raw []RawEvent) []Event {
	if len(raw) == 0 {
		return nil
	}
	out := make([]Event, 0, len(raw))
	for _, ev := range raw {
		if ev.Kind == Move || ev.Kind == KeyDown || ev.Kind == KeyUp {
			continue
		}
		out = append(out, Event{
			Offset: time.Duration(ev.Tick) * time.Millisecond,
			Kind:   ev.Kind,
			X:      ev.X,
			Y:      ev.Y,
			Wheel:  ev.Wheel,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// lerpCoord is the point `elapsed` of the way from `from` to `to` over `span`.
func lerpCoord(from, to int32, elapsed, span time.Duration) int32 {
	if span <= 0 || elapsed >= span {
		return to
	}
	if elapsed <= 0 {
		return from
	}
	return from + int32(int64(to-from)*int64(elapsed)/int64(span))
}
