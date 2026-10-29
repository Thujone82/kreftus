//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AppSnapshot is the full UI payload pushed to the WebView frontend.
type AppSnapshot struct {
	State          string         `json:"state"` // "idle" | "recording" | "countdown" | "playing"
	StatusText     string         `json:"statusText"`
	Note           string         `json:"note"`
	ActiveMacroID  string         `json:"activeMacroId"`
	PlayingStepIdx int            `json:"playingStepIdx"`
	RecEvents      int            `json:"recEvents"`
	RecElapsedMs   int64          `json:"recElapsedMs"`
	CountdownRemMs int64          `json:"countdownRemMs"`
	Macros         []Macro        `json:"macros"`
	Schedules      []ScheduleView `json:"schedules"`
	Settings       Settings       `json:"settings"`
	History        []HistoryEntry `json:"history"`
	StorageDir     string         `json:"storageDir"`
}

// ScheduleView enriches a Schedule with its human-readable summary and macro name for the UI.
type ScheduleView struct {
	Schedule
	MacroName string   `json:"macroName"`
	Summary   string   `json:"summary"`
	Preview   []string `json:"preview"`
}

// PickPointResult is returned when the user picks a screen coordinate for a step.
type PickPointResult struct {
	X  int32 `json:"x"`
	Y  int32 `json:"y"`
	OK bool  `json:"ok"`
}

func (a *application) buildSnapshot() AppSnapshot {
	macros, schedules, settings, history := a.store.Snapshot()
	macroNames := make(map[string]string, len(macros))
	for _, m := range macros {
		macroNames[m.ID] = m.Name
	}

	now := time.Now()
	schedViews := make([]ScheduleView, len(schedules))
	for i, sc := range schedules {
		mName := macroNames[sc.MacroID]
		if mName == "" {
			mName = "(deleted macro)"
		}
		var prevStrs []string
		for _, t := range PreviewNextRuns(sc, now, 5) {
			prevStrs = append(prevStrs, t.Format("Mon Jan 2, 3:04:05 PM"))
		}
		schedViews[i] = ScheduleView{
			Schedule:  sc,
			MacroName: mName,
			Summary:   DescribeSchedule(sc),
			Preview:   prevStrs,
		}
	}

	stateStr := "idle"
	var recEv int
	var recEl int64
	var cdRem int64

	switch a.state {
	case stateRecording:
		stateStr = "recording"
		n, _, elapsed, _ := a.rec.snapshot()
		recEv = n
		recEl = elapsed.Milliseconds()
	case stateCountdown:
		stateStr = "countdown"
		rem := time.Until(a.countdownDeadline)
		if rem < 0 {
			rem = 0
		}
		cdRem = rem.Milliseconds()
	case statePlaying:
		stateStr = "playing"
	}

	return AppSnapshot{
		State:          stateStr,
		StatusText:     a.statusText(),
		Note:           a.note,
		ActiveMacroID:  a.activeMacroID,
		PlayingStepIdx: int(a.playingStepIdx.Load()),
		RecEvents:      recEv,
		RecElapsedMs:   recEl,
		CountdownRemMs: cdRem,
		Macros:         macros,
		Schedules:      schedViews,
		Settings:       settings,
		History:        history,
		StorageDir:     a.store.Dir(),
	}
}

func (a *application) registerBridge() error {
	w := a.wv
	if w == nil {
		return nil
	}

	_ = w.Bind("apiGetState", func() (AppSnapshot, error) {
		return a.buildSnapshot(), nil
	})

	_ = w.Bind("apiSelectMacro", func(id string) (AppSnapshot, error) {
		a.activeMacroID = id
		return a.buildSnapshot(), nil
	})

	_ = w.Bind("apiCreateMacro", func(name string) (Macro, error) {
		macros, _, _, _ := a.store.Snapshot()
		if strings.TrimSpace(name) == "" {
			name = NextMacroName(macros, time.Now())
		}
		m, err := a.store.UpsertMacro(Macro{
			Name:  strings.TrimSpace(name),
			Steps: []Step{},
			Speed: 1.0,
		})
		if err != nil {
			return Macro{}, err
		}
		a.activeMacroID = m.ID
		a.pushState()
		return m, nil
	})

	_ = w.Bind("apiSaveMacro", func(m Macro) (Macro, error) {
		m.Name = strings.TrimSpace(m.Name)
		if m.Name == "" {
			return Macro{}, fmt.Errorf("macro name cannot be empty")
		}
		saved, err := a.store.UpsertMacro(m)
		if err != nil {
			return Macro{}, err
		}
		a.activeMacroID = saved.ID
		a.pushState()
		return saved, nil
	})

	_ = w.Bind("apiDuplicateMacro", func(id string) (Macro, error) {
		orig, ok := a.store.GetMacro(id)
		if !ok {
			return Macro{}, fmt.Errorf("macro not found")
		}
		steps := make([]Step, len(orig.Steps))
		for i, s := range orig.Steps {
			s.ID = newID()
			steps[i] = s
		}
		dup, err := a.store.UpsertMacro(Macro{
			Name:  orig.Name + " (Copy)",
			Steps: steps,
			Speed: orig.Speed,
			Notes: orig.Notes,
		})
		if err != nil {
			return Macro{}, err
		}
		a.activeMacroID = dup.ID
		a.pushState()
		return dup, nil
	})

	_ = w.Bind("apiDeleteMacro", func(id string) (AppSnapshot, error) {
		if err := a.store.DeleteMacro(id); err != nil {
			return AppSnapshot{}, err
		}
		if a.activeMacroID == id {
			macros, _, _, _ := a.store.Snapshot()
			if len(macros) > 0 {
				a.activeMacroID = macros[0].ID
			} else {
				a.activeMacroID = ""
			}
		}
		a.wakeScheduler()
		return a.buildSnapshot(), nil
	})

	_ = w.Bind("apiExportMacroJSON", func(id string) (string, error) {
		m, ok := a.store.GetMacro(id)
		if !ok {
			return "", fmt.Errorf("macro not found")
		}
		b, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			return "", err
		}
		return string(b), nil
	})

	_ = w.Bind("apiImportMacroJSON", func(raw string) (Macro, error) {
		var m Macro
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			return Macro{}, fmt.Errorf("invalid macro JSON: %w", err)
		}
		m.ID = newID()
		m.Created = time.Now()
		for i := range m.Steps {
			m.Steps[i].ID = newID()
		}
		saved, err := a.store.UpsertMacro(m)
		if err != nil {
			return Macro{}, err
		}
		a.activeMacroID = saved.ID
		a.pushState()
		return saved, nil
	})

	_ = w.Bind("apiToggleRecord", func(appendToMacroID string) (AppSnapshot, error) {
		a.dispatchUI(func() {
			a.appendToMacroID = appendToMacroID
			a.toggleRecord()
		})
		return a.buildSnapshot(), nil
	})

	_ = w.Bind("apiStartPlay", func(macroID string, delayStr string, fromStep int) (AppSnapshot, error) {
		if macroID != "" {
			a.activeMacroID = macroID
		}
		a.dispatchUI(func() {
			a.startCountdownWith(a.activeMacroID, delayStr, fromStep)
		})
		return a.buildSnapshot(), nil
	})

	_ = w.Bind("apiAbort", func() (AppSnapshot, error) {
		a.dispatchUI(func() {
			a.abort()
		})
		return a.buildSnapshot(), nil
	})

	_ = w.Bind("apiPickPoint", func() (PickPointResult, error) {
		abort := make(chan struct{})
		a.dispatchUI(func() {
			procShowWindow.Call(a.hwnd, swHide)
		})
		x, y, ok := PickNextClick(abort, 15*time.Second)
		a.dispatchUI(func() {
			a.showMainWindow()
		})
		return PickPointResult{X: x, Y: y, OK: ok}, nil
	})

	_ = w.Bind("apiSaveSchedule", func(sc Schedule) (ScheduleView, error) {
		saved, err := a.store.UpsertSchedule(sc, time.Now())
		if err != nil {
			return ScheduleView{}, err
		}
		a.wakeScheduler()
		a.pushState()
		macros, _, _, _ := a.store.Snapshot()
		mName := ""
		for _, m := range macros {
			if m.ID == saved.MacroID {
				mName = m.Name
				break
			}
		}
		var prev []string
		for _, t := range PreviewNextRuns(saved, time.Now(), 5) {
			prev = append(prev, t.Format("Mon Jan 2, 3:04:05 PM"))
		}
		return ScheduleView{
			Schedule:  saved,
			MacroName: mName,
			Summary:   DescribeSchedule(saved),
			Preview:   prev,
		}, nil
	})

	_ = w.Bind("apiToggleSchedule", func(id string, enabled bool) (AppSnapshot, error) {
		_, schedules, _, _ := a.store.Snapshot()
		for _, sc := range schedules {
			if sc.ID == id {
				sc.Enabled = enabled
				_, _ = a.store.UpsertSchedule(sc, time.Now())
				break
			}
		}
		a.wakeScheduler()
		return a.buildSnapshot(), nil
	})

	_ = w.Bind("apiDeleteSchedule", func(id string) (AppSnapshot, error) {
		if err := a.store.DeleteSchedule(id); err != nil {
			return AppSnapshot{}, err
		}
		a.wakeScheduler()
		return a.buildSnapshot(), nil
	})

	_ = w.Bind("apiRunScheduleNow", func(id string) (AppSnapshot, error) {
		_, schedules, _, _ := a.store.Snapshot()
		for _, sc := range schedules {
			if sc.ID == id {
				scCopy := sc
				a.dispatchUI(func() {
					a.triggerScheduledMacro(scCopy)
				})
				break
			}
		}
		return a.buildSnapshot(), nil
	})

	_ = w.Bind("apiPreviewSchedule", func(sc Schedule) ([]string, error) {
		runs := PreviewNextRuns(sc, time.Now(), 5)
		out := make([]string, len(runs))
		for i, t := range runs {
			out[i] = t.Format("Mon Jan 2, 2006 · 3:04:05 PM")
		}
		return out, nil
	})

	_ = w.Bind("apiSaveSettings", func(cfg Settings) (AppSnapshot, error) {
		if _, err := parseDelay(cfg.DefaultCountdown); err != nil {
			return AppSnapshot{}, err
		}
		if err := a.store.SaveSettings(cfg); err != nil {
			return AppSnapshot{}, err
		}
		if exe, err := os.Executable(); err == nil {
			if absExe, err2 := filepath.Abs(exe); err2 == nil {
				_ = setStartWithWindows(cfg.StartWithWindows, absExe)
			}
		}
		a.dispatchUI(func() {
			a.rebindHotkeys()
			a.applyKeepAwake()
		})
		return a.buildSnapshot(), nil
	})

	_ = w.Bind("apiClearHistory", func() (AppSnapshot, error) {
		_ = a.store.ClearHistory()
		return a.buildSnapshot(), nil
	})

	return nil
}
