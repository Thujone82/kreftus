package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Settings persists user preferences across sessions in %APPDATA%\Split\settings.json.
type Settings struct {
	Theme         string  `json:"theme"`         // "dark" | "light"
	ExportMode    string  `json:"exportMode"`    // "join" | "split"
	SkipDropped   bool    `json:"skipDropped"`   // skip dropped segments during playback preview
	ShowAudio     bool    `json:"showAudio"`     // show audio sound-level histogram lane
	ShowVisual    bool    `json:"showVisual"`    // show visual frame-delta & scene-cut lane
	TimelineScale float64 `json:"timelineScale"` // vertical timeline lane height scale (0.55..3.2)
	LastDir       string  `json:"lastDir"`       // last opened video folder
	FFmpegPath    string  `json:"ffmpegPath"`    // optional override
	FFprobePath   string  `json:"ffprobePath"`   // optional override
}

type SettingsStore struct {
	mu   sync.Mutex
	path string
	data Settings
}

func NewSettingsStore() *SettingsStore {
	dir := os.Getenv("APPDATA")
	if dir == "" {
		if cfgDir, err := os.UserConfigDir(); err == nil {
			dir = cfgDir
		} else {
			dir = os.TempDir()
		}
	}
	storeDir := filepath.Join(dir, "Split")
	_ = os.MkdirAll(storeDir, 0o755)
	return NewSettingsStoreAt(filepath.Join(storeDir, "settings.json"))
}

// NewSettingsStoreAt creates a SettingsStore backed by a specific JSON path.
func NewSettingsStoreAt(path string) *SettingsStore {
	s := &SettingsStore{
		path: path,
		data: Settings{
			Theme:         "dark",
			ExportMode:    "join",
			SkipDropped:   true,
			ShowAudio:     true,
			ShowVisual:    true,
			TimelineScale: 1.0,
		},
	}
	s.load()
	return s
}

// ClampTimelineScale constrains the persisted timeline vertical scale factor.
func ClampTimelineScale(v float64) float64 {
	if v <= 0 {
		return 1.0
	}
	if v < 0.55 {
		return 0.55
	}
	if v > 3.2 {
		return 3.2
	}
	return v
}

func (s *SettingsStore) load() {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	// Pre-populate defaults so legacy settings files without new keys keep them enabled.
	loaded := s.data
	if err := json.Unmarshal(raw, &loaded); err != nil {
		return
	}
	if loaded.Theme == "light" || loaded.Theme == "dark" {
		s.data.Theme = loaded.Theme
	}
	if loaded.ExportMode == "join" || loaded.ExportMode == "split" {
		s.data.ExportMode = loaded.ExportMode
	}
	s.data.SkipDropped = loaded.SkipDropped
	s.data.ShowAudio = loaded.ShowAudio
	s.data.ShowVisual = loaded.ShowVisual
	s.data.TimelineScale = ClampTimelineScale(loaded.TimelineScale)
	s.data.LastDir = loaded.LastDir
	s.data.FFmpegPath = loaded.FFmpegPath
	s.data.FFprobePath = loaded.FFprobePath
}

func (s *SettingsStore) Get() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data
}

func (s *SettingsStore) Update(fn func(*Settings)) {
	s.mu.Lock()
	fn(&s.data)
	copyData := s.data
	s.mu.Unlock()

	raw, err := json.MarshalIndent(copyData, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.path, append(raw, '\r', '\n'), 0o644)
}
