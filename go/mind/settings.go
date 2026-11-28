package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const settingsFileName = "mind.json"

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

type stats struct {
	Played      int `json:"played"`
	Won         int `json:"won"`
	Streak      int `json:"streak"`
	BestStreak  int `json:"bestStreak"`
	BestTurns   int `json:"bestTurns"`
	BestTimeSec int `json:"bestTimeSec"`
}

type settings struct {
	Version      int    `json:"version"`
	Theme        string `json:"theme"`
	Sound        bool   `json:"sound"`
	Labels       bool   `json:"labels"`
	LabelNumbers bool   `json:"labelNumbers"`
	Stats        stats  `json:"stats"`
}

func defaultSettings() settings {
	return settings{
		Version: 1,
		Theme:   defaultThemeID,
		Sound:   true,
		Labels:  false,
	}
}

func settingsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(dir) == "" {
		return settingsFileName
	}
	return filepath.Join(dir, "mind", settingsFileName)
}

func loadSettings(path string) settings {
	s := defaultSettings()
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	data = bytes.TrimPrefix(data, utf8BOM)
	var loaded settings
	if err := json.Unmarshal(data, &loaded); err != nil {
		return s
	}
	if loaded.Version == 0 {
		loaded.Version = 1
	}
	if !validTheme(loaded.Theme) {
		loaded.Theme = defaultThemeID
	}
	return loaded
}

func saveSettings(path string, s settings) error {
	if !validTheme(s.Theme) {
		s.Theme = defaultThemeID
	}
	if s.Version == 0 {
		s.Version = 1
	}
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	raw = bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
	raw = bytes.ReplaceAll(raw, []byte("\n"), []byte("\r\n"))

	var buf bytes.Buffer
	buf.Write(utf8BOM)
	buf.Write(raw)
	buf.WriteString("\r\n")

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (s *settings) recordResult(won bool, turns int, secs int) {
	s.Stats.Played++
	if !won {
		s.Stats.Streak = 0
		return
	}
	s.Stats.Won++
	s.Stats.Streak++
	if s.Stats.Streak > s.Stats.BestStreak {
		s.Stats.BestStreak = s.Stats.Streak
	}
	if turns > 0 && (s.Stats.BestTurns == 0 || turns < s.Stats.BestTurns) {
		s.Stats.BestTurns = turns
	}
	if secs < 0 {
		secs = 0
	}
	if s.Stats.BestTimeSec == 0 || secs < s.Stats.BestTimeSec {
		s.Stats.BestTimeSec = secs
	}
}
