package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Settings persists user preferences across sessions in %APPDATA%\Split\settings.json.
type Settings struct {
	Theme         string `json:"theme"`         // "dark" | "light"
	ExportMode    string `json:"exportMode"`    // "join" | "split"
	SkipDropped   bool   `json:"skipDropped"`   // skip dropped segments during playback preview
	LastDir       string `json:"lastDir"`       // last opened video folder
	FFmpegPath    string `json:"ffmpegPath"`    // optional override
	FFprobePath   string `json:"ffprobePath"`   // optional override
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
	s := &SettingsStore{
		path: filepath.Join(storeDir, "settings.json"),
		data: Settings{
			Theme:       "dark",
			ExportMode:  "join",
			SkipDropped: true,
		},
	}
	s.load()
	return s
}

func (s *SettingsStore) load() {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var loaded Settings
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
