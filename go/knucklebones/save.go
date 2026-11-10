package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	saveFileName        = "knucklebones.json"
	knucklebonesVersion = "1.0"
	maxHighScores       = 10
	maxNameLen          = 16
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

const (
	recordLocal  = "local"
	recordOnline = "online"
)

// recordKeys in Stats screen order.
var recordKeys = []string{"ai-" + diffEasy, "ai-" + diffMedium, "ai-" + diffHard, recordOnline, recordLocal}

type record struct {
	Games        int `json:"games"`
	Wins         int `json:"wins"`
	Losses       int `json:"losses"`
	Draws        int `json:"draws"`
	Best         int `json:"best"`
	SeriesWins   int `json:"seriesWins,omitempty"`
	SeriesLosses int `json:"seriesLosses,omitempty"`
}

type highScore struct {
	Name     string `json:"name"`
	Score    int    `json:"score"`
	Opponent string `json:"opponent"`
	Against  int    `json:"against"`
	Mode     string `json:"mode"`
	Date     string `json:"date"`
}

type settings struct {
	Difficulty  string `json:"difficulty"`
	BestOf      int    `json:"bestOf"`
	Rules       Rules  `json:"rules"`
	TurnSeconds int    `json:"turnSeconds"`
	Port        int    `json:"port"`
	JoinAddr    string `json:"joinAddr"`
	Player2     string `json:"player2"`
}

type saveData struct {
	Version    string             `json:"version"`
	Name       string             `json:"name"`
	Theme      string             `json:"theme"`
	Settings   settings           `json:"settings"`
	Records    map[string]*record `json:"records"`
	HighScores []highScore        `json:"highScores"`
}

func defaultName() string {
	for _, k := range []string{"USERNAME", "USER"} {
		if n := cleanName(os.Getenv(k), ""); n != "" {
			return n
		}
	}
	return "Lamb"
}

func newSaveData() *saveData {
	return &saveData{
		Version: knucklebonesVersion,
		Name:    defaultName(),
		Theme:   themeDark,
		Settings: settings{
			Difficulty:  diffMedium,
			BestOf:      1,
			TurnSeconds: 30,
			Port:        defaultPort,
			JoinAddr:    "localhost:" + strconv.Itoa(defaultPort),
			Player2:     "Player 2",
		},
		Records:    map[string]*record{},
		HighScores: []highScore{},
	}
}

func loadSave(path string) *saveData {
	s := newSaveData()
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	data = bytes.TrimPrefix(data, utf8BOM)
	if err := json.Unmarshal(data, s); err != nil {
		return newSaveData()
	}
	s.normalize()
	return s
}

func (s *saveData) normalize() {
	d := newSaveData()
	s.Version = knucklebonesVersion
	s.Name = cleanName(s.Name, d.Name)
	if s.Theme != themeLight {
		s.Theme = themeDark
	}
	if difficultyLabel[s.Settings.Difficulty] == "" {
		s.Settings.Difficulty = d.Settings.Difficulty
	}
	switch s.Settings.BestOf {
	case 0, 1, 3, 5:
	default:
		s.Settings.BestOf = 1
	}
	switch s.Settings.TurnSeconds {
	case 0, 15, 30, 60:
	default:
		s.Settings.TurnSeconds = 30
	}
	if s.Settings.Port <= 0 || s.Settings.Port > 65535 {
		s.Settings.Port = defaultPort
	}
	if strings.TrimSpace(s.Settings.JoinAddr) == "" {
		s.Settings.JoinAddr = d.Settings.JoinAddr
	}
	s.Settings.Player2 = cleanName(s.Settings.Player2, d.Settings.Player2)
	if s.Records == nil {
		s.Records = map[string]*record{}
	}
	for k, r := range s.Records {
		if r == nil {
			delete(s.Records, k)
		}
	}
	if s.HighScores == nil {
		s.HighScores = []highScore{}
	}
	sortHighScores(s.HighScores)
	if len(s.HighScores) > maxHighScores {
		s.HighScores = s.HighScores[:maxHighScores]
	}
}

// write saves atomically: a temp file in the same folder, then a rename.
func (s *saveData) write(path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(append([]byte{}, utf8BOM...), data...)
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".knucklebones-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

func (s *saveData) recordFor(key string) *record {
	r := s.Records[key]
	if r == nil {
		r = &record{}
		s.Records[key] = r
	}
	return r
}

func recordKey(mode, difficulty string) string {
	switch mode {
	case modeAI:
		return "ai-" + difficulty
	case modeOnline:
		return recordOnline
	}
	return recordLocal
}

func sortHighScores(h []highScore) {
	sort.SliceStable(h, func(i, j int) bool { return h[i].Score > h[j].Score })
}

// addHighScore inserts e and returns its 1-based rank, or 0 when it did
// not make the top ten.
func (s *saveData) addHighScore(e highScore) int {
	if e.Date == "" {
		e.Date = time.Now().Format("2006-01-02")
	}
	pos := len(s.HighScores)
	for i, h := range s.HighScores {
		if e.Score > h.Score {
			pos = i
			break
		}
	}
	if pos >= maxHighScores {
		return 0
	}
	s.HighScores = append(s.HighScores, highScore{})
	copy(s.HighScores[pos+1:], s.HighScores[pos:])
	s.HighScores[pos] = e
	if len(s.HighScores) > maxHighScores {
		s.HighScores = s.HighScores[:maxHighScores]
	}
	return pos + 1
}

// cleanName keeps printable characters, trims, and caps the length.
func cleanName(n, fallback string) string {
	var b strings.Builder
	count := 0
	for _, r := range strings.TrimSpace(n) {
		if !unicode.IsPrint(r) || count >= maxNameLen {
			continue
		}
		b.WriteRune(r)
		count++
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return fallback
	}
	return out
}
