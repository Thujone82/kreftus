package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const maxHistoryEntries = 200

// HotkeyBinding represents a configurable global shortcut.
type HotkeyBinding struct {
	Mods uint32 `json:"mods"` // modAlt=1, modCtrl=2, modShift=4, modWin=8
	VK   uint16 `json:"vk"`
	Text string `json:"text"`
}

// Settings holds persistent user preferences stored in settings.json.
type Settings struct {
	Theme              string        `json:"theme"`              // "dark" | "light" | "system"
	DefaultCountdown   string        `json:"defaultCountdown"`   // e.g. "5s"
	MergeTyping        bool          `json:"mergeTyping"`        // coalesce typed keys into Text steps
	HideWhilePlaying   bool          `json:"hideWhilePlaying"`   // hide main window during playback
	ScheduleWarnSecs   int           `json:"scheduleWarnSecs"`   // pre-run countdown before scheduled runs
	KeepAwake          bool          `json:"keepAwake"`          // prevent system sleep while schedules active
	StartWithWindows   bool          `json:"startWithWindows"`   // HKCU Run registry toggle
	RecordMouse        bool          `json:"recordMouse"`        // capture mouse clicks/wheel
	RecordKeyboard     bool          `json:"recordKeyboard"`     // capture keystrokes
	HotkeyRecord       HotkeyBinding `json:"hotkeyRecord"`
	HotkeyPlay         HotkeyBinding `json:"hotkeyPlay"`
	HotkeyAbort        HotkeyBinding `json:"hotkeyAbort"`
}

// DefaultSettings returns the out-of-the-box Settings.
func DefaultSettings() Settings {
	return Settings{
		Theme:            "dark",
		DefaultCountdown: "5s",
		MergeTyping:      true,
		HideWhilePlaying: true,
		ScheduleWarnSecs: 5,
		KeepAwake:        false,
		StartWithWindows: false,
		RecordMouse:      true,
		RecordKeyboard:   true,
		HotkeyRecord:     HotkeyBinding{Mods: 0x0001 | 0x0004, VK: 0xBD, Text: "Alt+Shift+_"},
		HotkeyPlay:       HotkeyBinding{Mods: 0x0001 | 0x0004, VK: 0x39, Text: "Alt+Shift+("},
		HotkeyAbort:      HotkeyBinding{Mods: 0x0001 | 0x0004, VK: 0x30, Text: "Alt+Shift+)"},
	}
}

// Store manages persistence of macros, schedules, settings, and run history as JSON files.
type Store struct {
	mu        sync.Mutex
	dir       string
	macros    []Macro
	schedules []Schedule
	settings  Settings
	history   []HistoryEntry
}

// NewStore resolves the storage directory (preferring the executable's directory
// for portable storage, falling back to %APPDATA%\Click if not writable) and loads state.
func NewStore(customDir string) (*Store, error) {
	dir := customDir
	if dir == "" {
		dir = resolvePortableOrAppDataDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create storage directory: %w", err)
	}
	st := &Store{
		dir:      dir,
		settings: DefaultSettings(),
	}
	st.loadAll()
	return st, nil
}

func resolvePortableOrAppDataDir() string {
	exe, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exe)
		if isDirWritable(exeDir) {
			return exeDir
		}
	}
	if appData := os.Getenv("APPDATA"); appData != "" {
		return filepath.Join(appData, "Click")
	}
	return "."
}

func isDirWritable(dir string) bool {
	probe := filepath.Join(dir, fmt.Sprintf(".click-write-test-%d", time.Now().UnixNano()))
	f, err := os.OpenFile(probe, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return false
	}
	_ = f.Close()
	_ = os.Remove(probe)
	return true
}

// Dir returns the active directory where JSON files are saved.
func (s *Store) Dir() string {
	return s.dir
}

func (s *Store) macrosPath() string    { return filepath.Join(s.dir, "macros.json") }
func (s *Store) schedulesPath() string { return filepath.Join(s.dir, "schedules.json") }
func (s *Store) settingsPath() string  { return filepath.Join(s.dir, "settings.json") }
func (s *Store) historyPath() string   { return filepath.Join(s.dir, "history.json") }

func (s *Store) loadAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = readJSONWithBackup(s.macrosPath(), &s.macros)
	_ = readJSONWithBackup(s.schedulesPath(), &s.schedules)
	_ = readJSONWithBackup(s.settingsPath(), &s.settings)
	_ = readJSONWithBackup(s.historyPath(), &s.history)
	if s.settings.Theme == "" {
		s.settings = DefaultSettings()
	}
	if s.settings.DefaultCountdown == "" {
		s.settings.DefaultCountdown = "5s"
	}
	if s.settings.HotkeyRecord.VK == 0 {
		def := DefaultSettings()
		s.settings.HotkeyRecord = def.HotkeyRecord
		s.settings.HotkeyPlay = def.HotkeyPlay
		s.settings.HotkeyAbort = def.HotkeyAbort
	}
}

func readJSONWithBackup(path string, dst interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dst); err != nil {
		// Keep a .bak copy of the corrupt file so user data is never silently lost
		_ = os.WriteFile(path+".bak", data, 0o644)
		return err
	}
	return nil
}

func writeJSONAtomic(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(path)
		if err2 := os.Rename(tmp, path); err2 != nil {
			_ = os.Remove(tmp)
			return err2
		}
	}
	return nil
}

// Snapshot returns copies of the current macros, schedules, settings, and history.
func (s *Store) Snapshot() ([]Macro, []Schedule, Settings, []HistoryEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	macros := cloneMacros(s.macros)
	scheds := make([]Schedule, len(s.schedules))
	copy(scheds, s.schedules)
	hist := make([]HistoryEntry, len(s.history))
	copy(hist, s.history)
	return macros, scheds, s.settings, hist
}

func cloneMacros(in []Macro) []Macro {
	out := make([]Macro, len(in))
	for i, m := range in {
		steps := make([]Step, len(m.Steps))
		copy(steps, m.Steps)
		m.Steps = steps
		out[i] = m
	}
	return out
}

// GetMacro returns a macro by ID.
func (s *Store) GetMacro(id string) (Macro, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.macros {
		if m.ID == id {
			steps := make([]Step, len(m.Steps))
			copy(steps, m.Steps)
			m.Steps = steps
			return m, true
		}
	}
	return Macro{}, false
}

// UpsertMacro saves a new or updated Macro and writes macros.json.
func (s *Store) UpsertMacro(m Macro) (Macro, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if m.ID == "" {
		m.ID = newID()
	}
	if m.Name == "" {
		m.Name = NextMacroName(s.macros, now)
	}
	if m.Created.IsZero() {
		m.Created = now
	}
	if m.Speed <= 0 {
		m.Speed = 1.0
	}
	for i := range m.Steps {
		if m.Steps[i].ID == "" {
			m.Steps[i].ID = newID()
		}
	}
	m.Modified = now

	replaced := false
	for i := range s.macros {
		if s.macros[i].ID == m.ID {
			s.macros[i] = m
			replaced = true
			break
		}
	}
	if !replaced {
		s.macros = append(s.macros, m)
	}
	return m, writeJSONAtomic(s.macrosPath(), s.macros)
}

// DeleteMacro removes a macro by ID and disables any schedules referencing it.
func (s *Store) DeleteMacro(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.macros[:0]
	for _, m := range s.macros {
		if m.ID != id {
			out = append(out, m)
		}
	}
	s.macros = out
	schedChanged := false
	for i := range s.schedules {
		if s.schedules[i].MacroID == id && s.schedules[i].Enabled {
			s.schedules[i].Enabled = false
			s.schedules[i].NextRun = nil
			schedChanged = true
		}
	}
	if schedChanged {
		_ = writeJSONAtomic(s.schedulesPath(), s.schedules)
	}
	return writeJSONAtomic(s.macrosPath(), s.macros)
}

// UpsertSchedule creates or updates a Schedule, recomputing its NextRun.
func (s *Store) UpsertSchedule(sc Schedule, now time.Time) (Schedule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sc.ID == "" {
		sc.ID = newID()
	}
	baseName := strings.TrimSpace(sc.Name)
	if baseName == "" {
		for _, m := range s.macros {
			if m.ID == sc.MacroID {
				baseName = m.Name
				break
			}
		}
	}
	sc.Name = NextScheduleName(s.schedules, baseName, sc.ID)
	if sc.PlaysPerRun <= 0 {
		sc.PlaysPerRun = 1
	}
	if sc.Missed == "" {
		sc.Missed = MissedSkip
	}
	if sc.Overlap == "" {
		sc.Overlap = OverlapSkip
	}
	if sc.End == "" {
		sc.End = EndNever
	}
	if sc.Repeat == "" {
		sc.Repeat = RepeatNone
	}
	sc.NextRun = ComputeNextRun(sc, now)

	replaced := false
	for i := range s.schedules {
		if s.schedules[i].ID == sc.ID {
			s.schedules[i] = sc
			replaced = true
			break
		}
	}
	if !replaced {
		s.schedules = append(s.schedules, sc)
	}
	return sc, writeJSONAtomic(s.schedulesPath(), s.schedules)
}

// UpdateSchedules persists bulk schedule runtime updates (such as RunCount, LastRun, NextRun).
func (s *Store) UpdateSchedules(fn func(schedules []Schedule) []Schedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.schedules = fn(s.schedules)
	return writeJSONAtomic(s.schedulesPath(), s.schedules)
}

// DeleteSchedule deletes a schedule by ID.
func (s *Store) DeleteSchedule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.schedules[:0]
	for _, sc := range s.schedules {
		if sc.ID != id {
			out = append(out, sc)
		}
	}
	s.schedules = out
	return writeJSONAtomic(s.schedulesPath(), s.schedules)
}

// HasActiveSchedules reports whether at least one schedule is currently enabled.
func (s *Store) HasActiveSchedules() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sc := range s.schedules {
		if sc.Enabled {
			return true
		}
	}
	return false
}

// SaveSettings updates user settings and writes settings.json.
func (s *Store) SaveSettings(cfg Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = cfg
	return writeJSONAtomic(s.settingsPath(), s.settings)
}

// GetSettings returns the current Settings.
func (s *Store) GetSettings() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings
}

// AppendHistory records a HistoryEntry, capping the list to maxHistoryEntries.
func (s *Store) AppendHistory(e HistoryEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.ID == "" {
		e.ID = newID()
	}
	if e.Time.IsZero() {
		e.Time = time.Now()
	}
	s.history = append([]HistoryEntry{e}, s.history...)
	if len(s.history) > maxHistoryEntries {
		s.history = s.history[:maxHistoryEntries]
	}
	return writeJSONAtomic(s.historyPath(), s.history)
}

// ClearHistory empties the activity log.
func (s *Store) ClearHistory() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = nil
	return writeJSONAtomic(s.historyPath(), s.history)
}
