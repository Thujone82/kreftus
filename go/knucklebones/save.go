package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
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
	maxSaveBytes        = 1 << 20
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
	BindIP      string `json:"bindIP,omitempty"`
	JoinAddr    string `json:"joinAddr"`
	Player2     string `json:"player2"`
	Sound       bool   `json:"sound"`
	Colorblind  bool   `json:"colorblind,omitempty"`
	WinMeter    bool   `json:"winMeter,omitempty"`
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
			Sound:       true,
		},
		Records:    map[string]*record{},
		HighScores: []highScore{},
	}
}

func defaultSavePath() string {
	dir, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(dir) == "" {
		return saveFileName
	}
	return filepath.Join(dir, "knucklebones", saveFileName)
}

func resolveSavePath() string {
	return resolveSavePathWith(saveFileName, defaultSavePath())
}

// resolveSavePathWith migrates a legacy CWD save file to the user config
// directory on first launch when no config save exists yet.
func resolveSavePathWith(legacyPath, targetPath string) string {
	if targetPath == "" || targetPath == legacyPath {
		return legacyPath
	}
	if _, err := os.Stat(targetPath); errors.Is(err, os.ErrNotExist) {
		if info, err := os.Stat(legacyPath); err == nil && info.Mode().IsRegular() {
			s := loadSave(legacyPath)
			_ = s.write(targetPath)
		}
	}
	return targetPath
}

func loadSave(path string) *saveData {
	s := newSaveData()
	f, err := os.Open(path)
	if err != nil {
		return s
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxSaveBytes+1))
	if err != nil || len(data) > maxSaveBytes {
		return s
	}
	data = bytes.TrimPrefix(data, utf8BOM)
	if err := json.Unmarshal(data, s); err != nil {
		return newSaveData()
	}
	if s.normalize() {
		_ = s.write(path)
	}
	return s
}

// normalize repairs settings and reports whether the old default port was migrated.
func (s *saveData) normalize() bool {
	d := newSaveData()
	s.Version = knucklebonesVersion
	s.Name = cleanName(s.Name, d.Name)
	if !validTheme(s.Theme) {
		s.Theme = themeDark
	}
	s.Settings.BindIP = cleanBindIP(s.Settings.BindIP)
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
	migrated := false
	if s.Settings.Port <= 0 || s.Settings.Port > 65535 {
		s.Settings.Port = defaultPort
	}
	// Older builds used 7777 as the default host/join port.
	if s.Settings.Port == 7777 {
		s.Settings.Port = defaultPort
		migrated = true
	}
	if strings.TrimSpace(s.Settings.JoinAddr) == "" {
		s.Settings.JoinAddr = d.Settings.JoinAddr
	}
	if host, port, err := net.SplitHostPort(s.Settings.JoinAddr); err == nil && port == "7777" {
		s.Settings.JoinAddr = net.JoinHostPort(host, strconv.Itoa(defaultPort))
		migrated = true
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
	return migrated
}

// write saves atomically: a temp file in the same folder, then a rename.
func (s *saveData) write(path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(append([]byte{}, utf8BOM...), data...)
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	tmp, err := os.CreateTemp(dir, ".knucklebones-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
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
		if !unicode.IsPrint(r) || unicode.IsControl(r) || isBidiControl(r) || count >= maxNameLen {
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

func isBidiControl(r rune) bool {
	return (r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069) || r == 0x200E || r == 0x200F
}

// cleanText strips non-printable and bidirectional control runes and caps length.
func cleanText(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	var b strings.Builder
	count := 0
	for _, r := range strings.TrimSpace(s) {
		if !unicode.IsPrint(r) || unicode.IsControl(r) || isBidiControl(r) {
			continue
		}
		if count >= maxLen {
			break
		}
		b.WriteRune(r)
		count++
	}
	return strings.TrimSpace(b.String())
}

func cleanBindIP(s string) string {
	s = cleanText(s, 64)
	if s == "" || net.ParseIP(s) == nil {
		return ""
	}
	return s
}

func bindIPChoices() []string {
	out := []string{"", "127.0.0.1"}
	for _, ip := range localAddrs() {
		if ip != "" && ip != "127.0.0.1" {
			out = append(out, ip)
		}
	}
	return out
}

func bindIPLabel(ip string) string {
	if ip == "" {
		return "All interfaces"
	}
	if ip == "127.0.0.1" {
		return "127.0.0.1 (local only)"
	}
	return ip
}
