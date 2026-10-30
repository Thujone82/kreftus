package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// StepType identifies the kind of action a Step performs.
type StepType string

const (
	StepClick       StepType = "click"
	StepDoubleClick StepType = "double_click"
	StepDrag        StepType = "drag"
	StepWheel       StepType = "wheel"
	StepKey         StepType = "key"
	StepText        StepType = "text"
	StepWait        StepType = "wait"
)

// Button identifies a mouse button for click and drag steps.
type Button string

const (
	ButtonLeft   Button = "left"
	ButtonRight  Button = "right"
	ButtonMiddle Button = "middle"
)

// Mods is a bitmask of keyboard modifiers held during a key step.
type Mods uint8

const (
	ModCtrl Mods = 1 << iota
	ModAlt
	ModShift
	ModWin
)

// Has reports whether m contains all bits in flag.
func (m Mods) Has(flag Mods) bool {
	return m&flag == flag
}

// Step is one user-editable action in a Macro.
// DelayMs is the wait before this step runs (since the previous step finished).
// DelayEdited is set to true when the user manually overrides the delay in the UI;
// when false, the first step in a macro runs immediately (0 delay) while retaining
// its recorded DelayMs in case the user later reorders steps.
// HoldMs is how long a mouse button or key stays down during this step.
type Step struct {
	ID          string   `json:"id"`
	DelayMs     int64    `json:"delayMs"`
	DelayEdited bool     `json:"delayEdited,omitempty"`
	Type        StepType `json:"type"`
	Button      Button   `json:"button,omitempty"`
	X           int32    `json:"x,omitempty"`
	Y           int32    `json:"y,omitempty"`
	X2          int32    `json:"x2,omitempty"`
	Y2          int32    `json:"y2,omitempty"`
	Wheel       int32    `json:"wheel,omitempty"`
	Key         uint16   `json:"key,omitempty"`
	KeyName     string   `json:"keyName,omitempty"`
	Mods        Mods     `json:"mods,omitempty"`
	Text        string   `json:"text,omitempty"`
	HoldMs      int64    `json:"holdMs,omitempty"`
	Glide       bool     `json:"glide"`
}

// EffectiveDelayMs returns the delay in milliseconds that should be applied
// when this step is at position idx in a macro: 0 for the first step unless
// the user explicitly set a delay (or the step is a pure Wait step).
func (s Step) EffectiveDelayMs(idx int) int64 {
	if s.DelayMs <= 0 {
		return 0
	}
	if idx == 0 && !s.DelayEdited && s.Type != StepWait {
		return 0
	}
	return s.DelayMs
}

// Delay returns DelayMs as a time.Duration.
func (s Step) Delay() time.Duration {
	if s.DelayMs <= 0 {
		return 0
	}
	return time.Duration(s.DelayMs) * time.Millisecond
}

// Hold returns HoldMs as a time.Duration.
func (s Step) Hold() time.Duration {
	if s.HoldMs <= 0 {
		return 0
	}
	return time.Duration(s.HoldMs) * time.Millisecond
}

// Macro is a named, editable sequence of steps persisted to macros.json.
type Macro struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
	Steps    []Step    `json:"steps"`
	Speed    float64   `json:"speed,omitempty"`
	Notes    string    `json:"notes,omitempty"`
}

// DurationMs returns the total duration of the macro in milliseconds,
// accounting for its speed multiplier and suppressing unedited initial delay.
func (m Macro) DurationMs() int64 {
	var total int64
	for i, s := range m.Steps {
		if d := s.EffectiveDelayMs(i); d > 0 {
			total += d
		}
		if s.HoldMs > 0 && s.Type != StepWait {
			total += s.HoldMs
		}
	}
	spd := m.Speed
	if spd <= 0 {
		spd = 1.0
	}
	return int64(math.Round(float64(total) / spd))
}

// newID generates a short random hex identifier for steps, macros, and schedules.
func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("id-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// NextMacroName generates the default name 'Macro MMDDYY XX' where MMDDYY is
// the date of now and XX is the lowest free positive integer (starting at 01,
// two digits minimum, expanding to 100+ after 99).
func NextMacroName(existing []Macro, now time.Time) string {
	prefix := fmt.Sprintf("Macro %s ", now.Format("010206"))
	used := make(map[int]bool, len(existing))
	for _, m := range existing {
		name := strings.TrimSpace(m.Name)
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(name, prefix))
		if rest == "" || !digitsOnly(rest) {
			continue
		}
		n, err := strconv.Atoi(rest)
		if err == nil && n >= 1 {
			used[n] = true
		}
	}
	for i := 1; ; i++ {
		if !used[i] {
			return fmt.Sprintf("%s%02d", prefix, i)
		}
	}
}

// BuildOptions controls how raw hook samples are coalesced into steps.
type BuildOptions struct {
	MergeTyping     bool
	DragThresholdPx int32
	DoubleClickMs   int64
	TextMergeGapMs  int64
}

// DefaultBuildOptions returns standard step-coalescing thresholds.
func DefaultBuildOptions(mergeTyping bool) BuildOptions {
	return BuildOptions{
		MergeTyping:     mergeTyping,
		DragThresholdPx: 6,
		DoubleClickMs:   380,
		TextMergeGapMs:  450,
	}
}

// Standard Windows Virtual-Key codes used for modifier and printable key analysis.
const (
	vkBack       = 0x08
	vkTab        = 0x09
	vkReturn     = 0x0D
	vkShift      = 0x10
	vkControl    = 0x11
	vkMenu       = 0x12 // Alt
	vkPause      = 0x13
	vkCapital    = 0x14
	vkEscape     = 0x1B
	vkSpace      = 0x20
	vkPrior      = 0x21 // Page Up
	vkNext       = 0x22 // Page Down
	vkEnd        = 0x23
	vkHome       = 0x24
	vkLeft       = 0x25
	vkUp         = 0x26
	vkRight      = 0x27
	vkDown       = 0x28
	vkSnapshot   = 0x2C
	vkInsert     = 0x2D
	vkDelete     = 0x2E
	vkLWin       = 0x5B
	vkRWin       = 0x5C
	vkApps       = 0x5D
	vkNumpad0    = 0x60
	vkNumpad9    = 0x69
	vkMultiply   = 0x6A
	vkAdd        = 0x6B
	vkSubtract   = 0x6D
	vkDecimal    = 0x6E
	vkDivide     = 0x6F
	vkF1         = 0x70
	vkF12        = 0x7B
	vkF24        = 0x87
	vkNumlock    = 0x90
	vkScroll     = 0x91
	vkLShift     = 0xA0
	vkRShift     = 0xA1
	vkLControl   = 0xA2
	vkRControl   = 0xA3
	vkLMenu      = 0xA4
	vkRMenu      = 0xA5
	vkOEM1       = 0xBA // ;:
	vkOEMPlus    = 0xBB // =+
	vkOEMComma   = 0xBC // ,<
	vkOEMMinusVK = 0xBD // -_
	vkOEMPeriod  = 0xBE // .>
	vkOEM2       = 0xBF // /?
	vkOEM3       = 0xC0 // `~
	vkOEM4       = 0xDB // [{
	vkOEM5       = 0xDC // \|
	vkOEM6       = 0xDD // ]}
	vkOEM7       = 0xDE // '"
)

func isModifierVK(vk uint16) bool {
	switch vk {
	case vkShift, vkLShift, vkRShift,
		vkControl, vkLControl, vkRControl,
		vkMenu, vkLMenu, vkRMenu,
		vkLWin, vkRWin:
		return true
	default:
		return false
	}
}

func canonicalModVK(vk uint16) Mods {
	switch vk {
	case vkControl, vkLControl, vkRControl:
		return ModCtrl
	case vkMenu, vkLMenu, vkRMenu:
		return ModAlt
	case vkShift, vkLShift, vkRShift:
		return ModShift
	case vkLWin, vkRWin:
		return ModWin
	default:
		return 0
	}
}

// VKName returns a human-readable name for a Windows Virtual-Key code.
func VKName(vk uint16) string {
	switch {
	case vk >= 'A' && vk <= 'Z':
		return string(rune(vk))
	case vk >= '0' && vk <= '9':
		return string(rune(vk))
	case vk >= vkF1 && vk <= vkF24:
		return fmt.Sprintf("F%d", int(vk-vkF1)+1)
	case vk >= vkNumpad0 && vk <= vkNumpad9:
		return fmt.Sprintf("Num%d", int(vk-vkNumpad0))
	}
	switch vk {
	case vkBack:
		return "Backspace"
	case vkTab:
		return "Tab"
	case vkReturn:
		return "Enter"
	case vkEscape:
		return "Esc"
	case vkSpace:
		return "Space"
	case vkPrior:
		return "PageUp"
	case vkNext:
		return "PageDown"
	case vkEnd:
		return "End"
	case vkHome:
		return "Home"
	case vkLeft:
		return "Left"
	case vkUp:
		return "Up"
	case vkRight:
		return "Right"
	case vkDown:
		return "Down"
	case vkSnapshot:
		return "PrintScreen"
	case vkInsert:
		return "Insert"
	case vkDelete:
		return "Delete"
	case vkPause:
		return "Pause"
	case vkCapital:
		return "CapsLock"
	case vkNumlock:
		return "NumLock"
	case vkScroll:
		return "ScrollLock"
	case vkApps:
		return "Menu"
	case vkLWin, vkRWin:
		return "Win"
	case vkShift, vkLShift, vkRShift:
		return "Shift"
	case vkControl, vkLControl, vkRControl:
		return "Ctrl"
	case vkMenu, vkLMenu, vkRMenu:
		return "Alt"
	case vkMultiply:
		return "Num*"
	case vkAdd:
		return "Num+"
	case vkSubtract:
		return "Num-"
	case vkDecimal:
		return "Num."
	case vkDivide:
		return "Num/"
	case vkOEM1:
		return ";"
	case vkOEMPlus:
		return "="
	case vkOEMComma:
		return ","
	case vkOEMMinusVK:
		return "-"
	case vkOEMPeriod:
		return "."
	case vkOEM2:
		return "/"
	case vkOEM3:
		return "`"
	case vkOEM4:
		return "["
	case vkOEM5:
		return `\`
	case vkOEM6:
		return "]"
	case vkOEM7:
		return "'"
	default:
		return fmt.Sprintf("VK_0x%02X", vk)
	}
}

// NameToVK resolves a key name (such as "Enter", "A", "F5") to its Virtual-Key code.
func NameToVK(name string) (uint16, bool) {
	n := strings.TrimSpace(name)
	if len(n) == 1 {
		ch := n[0]
		if ch >= 'a' && ch <= 'z' {
			return uint16(ch - 'a' + 'A'), true
		}
		if (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			return uint16(ch), true
		}
	}
	for vk := uint16(0x08); vk <= 0xFE; vk++ {
		if strings.EqualFold(VKName(vk), n) {
			return vk, true
		}
	}
	return 0, false
}

// FormatChord formats a modifier mask and key into a display string like "Ctrl+Shift+S".
func FormatChord(mods Mods, vk uint16) string {
	var parts []string
	if mods.Has(ModCtrl) {
		parts = append(parts, "Ctrl")
	}
	if mods.Has(ModAlt) {
		parts = append(parts, "Alt")
	}
	if mods.Has(ModShift) {
		parts = append(parts, "Shift")
	}
	if mods.Has(ModWin) {
		parts = append(parts, "Win")
	}
	if vk != 0 && !isModifierVK(vk) {
		parts = append(parts, VKName(vk))
	} else if len(parts) == 0 && vk != 0 {
		parts = append(parts, VKName(vk))
	}
	return strings.Join(parts, "+")
}

// vkToPrintable converts a VK + Mods into a US-keyboard printable character if
// the key represents plain text typing (no Ctrl/Alt/Win).
func vkToPrintable(vk uint16, mods Mods) (byte, bool) {
	if mods&(ModCtrl|ModAlt|ModWin) != 0 {
		return 0, false
	}
	shift := mods.Has(ModShift)
	if vk >= 'A' && vk <= 'Z' {
		if shift {
			return byte(vk), true
		}
		return byte(vk - 'A' + 'a'), true
	}
	if vk >= '0' && vk <= '9' {
		if !shift {
			return byte(vk), true
		}
		shiftedDigits := ")!@#$%^&*("
		return shiftedDigits[vk-'0'], true
	}
	if vk == vkSpace {
		return ' ', true
	}
	type pair struct{ plain, shifted byte }
	oem := map[uint16]pair{
		vkOEM1:       {';', ':'},
		vkOEMPlus:    {'=', '+'},
		vkOEMComma:   {',', '<'},
		vkOEMMinusVK: {'-', '_'},
		vkOEMPeriod:  {'.', '>'},
		vkOEM2:       {'/', '?'},
		vkOEM3:       {'`', '~'},
		vkOEM4:       {'[', '{'},
		vkOEM5:       {'\\', '|'},
		vkOEM6:       {']', '}'},
		vkOEM7:       {'\'', '"'},
	}
	if p, ok := oem[vk]; ok {
		if shift {
			return p.shifted, true
		}
		return p.plain, true
	}
	return 0, false
}

// BuildSteps converts raw mouse and keyboard samples into high-level editable Steps.
func BuildSteps(raw []RawEvent, opts BuildOptions) []Step {
	if len(raw) == 0 {
		return nil
	}
	if opts.DragThresholdPx <= 0 {
		opts.DragThresholdPx = 6
	}
	if opts.TextMergeGapMs <= 0 {
		opts.TextMergeGapMs = 450
	}

	type pendingMouse struct {
		active bool
		btn    Button
		tick   uint32
		x, y   int32
	}
	type pendingKey struct {
		active bool
		vk     uint16
		mods   Mods
		tick   uint32
	}

	var (
		steps      []Step
		lastTick   uint32
		mouseDown  pendingMouse
		keysDown   = make(map[uint16]pendingKey)
		heldMods   Mods
		modUsed    bool // whether a non-modifier key was pressed while current modifiers were held
		soloModVK  uint16
		soloModT   uint32
	)

	emitStep := func(s Step, startTick, endTick uint32) {
		delay := int64(0)
		if startTick >= lastTick {
			delay = int64(startTick - lastTick)
		}
		s.ID = newID()
		s.DelayMs = delay
		if endTick > startTick {
			lastTick = endTick
		} else {
			lastTick = startTick
		}

		// Coalesce consecutive wheel steps in the same direction within 250ms
		if s.Type == StepWheel && len(steps) > 0 {
			prev := &steps[len(steps)-1]
			if prev.Type == StepWheel && s.DelayMs <= 250 &&
				((prev.Wheel > 0 && s.Wheel > 0) || (prev.Wheel < 0 && s.Wheel < 0)) {
				prev.Wheel += s.Wheel
				return
			}
		}

		// Coalesce printable key taps into a StepText when MergeTyping is on
		if opts.MergeTyping && s.Type == StepKey && s.HoldMs < 350 {
			if ch, ok := vkToPrintable(s.Key, s.Mods); ok {
				if len(steps) > 0 {
					prev := &steps[len(steps)-1]
					if prev.Type == StepText && s.DelayMs <= opts.TextMergeGapMs {
						prev.Text += string(ch)
						return
					}
				}
				s.Type = StepText
				s.Text = string(ch)
				s.Key = 0
				s.KeyName = ""
				s.Mods = 0
				s.HoldMs = 0
			}
		}

		steps = append(steps, s)
	}

	for _, ev := range raw {
		switch ev.Kind {
		case Move:
			continue

		case LeftDown, RightDown, MiddleDown:
			btn := ButtonLeft
			if ev.Kind == RightDown {
				btn = ButtonRight
			} else if ev.Kind == MiddleDown {
				btn = ButtonMiddle
			}
			// If a previous button-down never saw its button-up before this new press, flush it so no click is lost.
			if mouseDown.active {
				emitStep(Step{
					Type:   StepClick,
					Button: mouseDown.btn,
					X:      mouseDown.x,
					Y:      mouseDown.y,
					HoldMs: 40,
					Glide:  true,
				}, mouseDown.tick, mouseDown.tick)
			}
			mouseDown = pendingMouse{
				active: true,
				btn:    btn,
				tick:   ev.Tick,
				x:      ev.X,
				y:      ev.Y,
			}

		case LeftUp, RightUp, MiddleUp:
			btn := ButtonLeft
			if ev.Kind == RightUp {
				btn = ButtonRight
			} else if ev.Kind == MiddleUp {
				btn = ButtonMiddle
			}
			if !mouseDown.active || mouseDown.btn != btn {
				emitStep(Step{
					Type:   StepClick,
					Button: btn,
					X:      ev.X,
					Y:      ev.Y,
					HoldMs: 40,
					Glide:  true,
				}, ev.Tick, ev.Tick)
				mouseDown = pendingMouse{}
				continue
			}
			hold := int64(0)
			if ev.Tick > mouseDown.tick {
				hold = int64(ev.Tick - mouseDown.tick)
			}
			dx := abs32(ev.X - mouseDown.x)
			dy := abs32(ev.Y - mouseDown.y)
			if dx > opts.DragThresholdPx || dy > opts.DragThresholdPx {
				emitStep(Step{
					Type:   StepDrag,
					Button: btn,
					X:      mouseDown.x,
					Y:      mouseDown.y,
					X2:     ev.X,
					Y2:     ev.Y,
					HoldMs: hold,
					Glide:  true,
				}, mouseDown.tick, ev.Tick)
			} else {
				emitStep(Step{
					Type:   StepClick,
					Button: btn,
					X:      mouseDown.x,
					Y:      mouseDown.y,
					HoldMs: hold,
					Glide:  true,
				}, mouseDown.tick, ev.Tick)
			}
			mouseDown = pendingMouse{}

		case Wheel:
			emitStep(Step{
				Type:  StepWheel,
				X:     ev.X,
				Y:     ev.Y,
				Wheel: ev.Wheel,
				Glide: true,
			}, ev.Tick, ev.Tick)

		case KeyDown:
			if ev.VK == 0 {
				continue
			}
			if isModifierVK(ev.VK) {
				m := canonicalModVK(ev.VK)
				if heldMods == 0 {
					modUsed = false
					soloModVK = ev.VK
					soloModT = ev.Tick
				}
				heldMods |= m
				continue
			}
			modUsed = true
			if _, exists := keysDown[ev.VK]; !exists {
				keysDown[ev.VK] = pendingKey{
					active: true,
					vk:     ev.VK,
					mods:   heldMods,
					tick:   ev.Tick,
				}
			}

		case KeyUp:
			if ev.VK == 0 {
				continue
			}
			if isModifierVK(ev.VK) {
				m := canonicalModVK(ev.VK)
				wasHeld := heldMods
				heldMods &^= m
				// Solo tap of the Windows key (e.g. opening Start menu)
				if !modUsed && wasHeld == ModWin && heldMods == 0 && soloModVK != 0 {
					hold := int64(0)
					if ev.Tick > soloModT {
						hold = int64(ev.Tick - soloModT)
					}
					emitStep(Step{
						Type:    StepKey,
						Key:     soloModVK,
						KeyName: VKName(soloModVK),
						HoldMs:  hold,
					}, soloModT, ev.Tick)
				}
				if heldMods == 0 {
					soloModVK = 0
				}
				continue
			}
			pk, ok := keysDown[ev.VK]
			delete(keysDown, ev.VK)
			if !ok {
				emitStep(Step{
					Type:    StepKey,
					Key:     ev.VK,
					KeyName: FormatChord(heldMods, ev.VK),
					Mods:    heldMods,
				}, ev.Tick, ev.Tick)
				continue
			}
			hold := int64(0)
			if ev.Tick > pk.tick {
				hold = int64(ev.Tick - pk.tick)
			}
			emitStep(Step{
				Type:    StepKey,
				Key:     pk.vk,
				KeyName: FormatChord(pk.mods, pk.vk),
				Mods:    pk.mods,
				HoldMs:  hold,
			}, pk.tick, ev.Tick)
		}
	}

	// Flush any mouse button or key still held when recording stopped.
	if mouseDown.active {
		emitStep(Step{
			Type:   StepClick,
			Button: mouseDown.btn,
			X:      mouseDown.x,
			Y:      mouseDown.y,
			HoldMs: 40,
			Glide:  true,
		}, mouseDown.tick, mouseDown.tick)
	}
	for _, pk := range keysDown {
		emitStep(Step{
			Type:    StepKey,
			Key:     pk.vk,
			KeyName: FormatChord(pk.mods, pk.vk),
			Mods:    pk.mods,
		}, pk.tick, pk.tick)
	}

	return steps
}

// TrimTrailingHotkey strips any trailing key events that belong to the
// Alt+Shift+_ or Alt+Shift+) stop hotkey chord pressed to end recording.
func TrimTrailingHotkey(raw []RawEvent) []RawEvent {
	for len(raw) > 0 {
		last := raw[len(raw)-1]
		if last.Kind != KeyDown && last.Kind != KeyUp {
			break
		}
		if isModifierVK(last.VK) || last.VK == vkOEMMinusVK || last.VK == '0' || last.VK == '9' {
			raw = raw[:len(raw)-1]
			continue
		}
		break
	}
	return raw
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
