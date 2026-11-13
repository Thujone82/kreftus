package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const (
	maxTopScores     = 10
	maxRecentMatches = 25
)

type ScoreRecord struct {
	Name      string `json:"name"`
	Score     int    `json:"score"`
	Opponent  string `json:"opponent"`
	Against   int    `json:"against"`
	BestOf    int    `json:"bestOf"`
	Rules     Rules  `json:"rules"`
	RecordedAt string `json:"recordedAt"`
}

type MarginRecord struct {
	Winner     string `json:"winner"`
	Loser      string `json:"loser"`
	WinScore   int    `json:"winScore"`
	LoseScore  int    `json:"loseScore"`
	Margin     int    `json:"margin"`
	RecordedAt string `json:"recordedAt"`
}

type MatchRecord struct {
	Names      [2]string `json:"names"`
	Wins       [2]int    `json:"wins"`
	Ties       int       `json:"ties"`
	BestOf     int       `json:"bestOf"`
	Rules      Rules     `json:"rules"`
	Forfeit    string    `json:"forfeit,omitempty"`
	Winner     int       `json:"winner"` // 0, 1, or -1
	FinishedAt string    `json:"finishedAt"`
}

// StatsView is sent to clients in r_lobby / r_hosted and HTTP /stats.
type StatsView struct {
	LobbyWatchers      int               `json:"lobbyWatchers"`
	LiveMatches        int               `json:"liveMatches"`
	WaitingRooms       int               `json:"waitingRooms"`
	CompletedMatches   uint64            `json:"completedMatches"`
	CompletedRounds    uint64            `json:"completedRounds"`
	TotalDicePlayed    uint64            `json:"totalDicePlayed"`
	TotalDiceDestroyed uint64            `json:"totalDiceDestroyed"`
	TotalPointsScored  uint64            `json:"totalPointsScored"`
	DiceHistogram      [6]uint64         `json:"diceHistogram"`
	Forfeits           map[string]uint64 `json:"forfeits,omitempty"`
	HighestRound       *ScoreRecord      `json:"highestRound,omitempty"`
	UpdatedAt          int64             `json:"updatedAt"`
}

type PersistentStats struct {
	Version              int               `json:"version"`
	CompletedMatches     uint64            `json:"completedMatches"`
	CompletedRounds      uint64            `json:"completedRounds"`
	RoundDraws           uint64            `json:"roundDraws"`
	TotalDicePlayed      uint64            `json:"totalDicePlayed"`
	TotalDiceDestroyed   uint64            `json:"totalDiceDestroyed"`
	TotalPointsScored    uint64            `json:"totalPointsScored"`
	TotalPointsDestroyed uint64            `json:"totalPointsDestroyed"`
	DiceHistogram        [6]uint64         `json:"diceHistogram"`
	Forfeits             map[string]uint64 `json:"forfeits"`
	BestOfCounts         map[string]uint64 `json:"bestOfCounts"`
	StandardMatches      uint64            `json:"standardMatches"`
	TwoDiceMatches       uint64            `json:"twoDiceMatches"`
	NoDestroyMatches     uint64            `json:"noDestroyMatches"`
	FillBothMatches      uint64            `json:"fillBothMatches"`
	HighestRound         *ScoreRecord      `json:"highestRound,omitempty"`
	BiggestMargin        *MarginRecord     `json:"biggestMargin,omitempty"`
	MostDestroyedInMove  int               `json:"mostDestroyedInMove"`
	TopScores            []ScoreRecord     `json:"topScores"`
	RecentMatches        []MatchRecord     `json:"recentMatches"`
	UpdatedAt            int64             `json:"updatedAt"`
}

type roomProgress struct {
	countedMatch  bool
	anyMovePlayed bool
	lastMoveKey   [3]int // {series, round, moves}
	lastRoundKey  [2]int // {series, round}
	lastSeriesKey int
	ended         bool
}

type statsStore struct {
	mu       sync.Mutex
	path     string
	data     PersistentStats
	dirty    bool
	lastSave time.Time
}

func newStatsStore(path string) *statsStore {
	s := &statsStore{
		path: path,
		data: PersistentStats{
			Version:      1,
			Forfeits:     map[string]uint64{},
			BestOfCounts: map[string]uint64{},
		},
	}
	if path != "" {
		s.load()
	}
	return s
}

func (s *statsStore) load() {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var loaded PersistentStats
	if err := json.Unmarshal(b, &loaded); err != nil {
		return
	}
	if loaded.Forfeits == nil {
		loaded.Forfeits = map[string]uint64{}
	}
	if loaded.BestOfCounts == nil {
		loaded.BestOfCounts = map[string]uint64{}
	}
	loaded.Version = 1
	s.data = loaded
}

func (s *statsStore) flushLocked() {
	if s.path == "" || !s.dirty {
		return
	}
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return
	}
	dir := filepath.Dir(s.path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return
		}
	}
	tmp, err := os.CreateTemp(dir, ".stats-*.tmp")
	if err != nil {
		return
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		_ = os.Remove(tmpName)
		return
	}
	s.dirty = false
	s.lastSave = time.Now()
}

func (s *statsStore) Flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flushLocked()
}

func (s *statsStore) Snapshot(live, waiting, lobbyWatchers int) StatsView {
	s.mu.Lock()
	defer s.mu.Unlock()
	forfeits := make(map[string]uint64, len(s.data.Forfeits))
	for k, v := range s.data.Forfeits {
		forfeits[k] = v
	}
	var highCopy *ScoreRecord
	if s.data.HighestRound != nil {
		cp := *s.data.HighestRound
		highCopy = &cp
	}
	return StatsView{
		LobbyWatchers:      lobbyWatchers,
		LiveMatches:        live,
		WaitingRooms:       waiting,
		CompletedMatches:   s.data.CompletedMatches,
		CompletedRounds:    s.data.CompletedRounds,
		TotalDicePlayed:    s.data.TotalDicePlayed,
		TotalDiceDestroyed: s.data.TotalDiceDestroyed,
		TotalPointsScored:  s.data.TotalPointsScored,
		DiceHistogram:      s.data.DiceHistogram,
		Forfeits:           forfeits,
		HighestRound:       highCopy,
		UpdatedAt:          s.data.UpdatedAt,
	}
}

func (s *statsStore) FullDump(live, waiting, lobbyWatchers int) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]any{
		"lobbyWatchers": lobbyWatchers,
		"liveMatches":   live,
		"waitingRooms":  waiting,
		"stats":         s.data,
	}
}

func cmpTriplet(a, b [3]int) int {
	for i := 0; i < 3; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

func cmpPair(a, b [2]int) int {
	if a[0] != b[0] {
		if a[0] < b[0] {
			return -1
		}
		return 1
	}
	if a[1] < b[1] {
		return -1
	}
	if a[1] > b[1] {
		return 1
	}
	return 0
}

// ObserveState inspects a relayed game state packet to accumulate persistent stats.
// Returns (changed, matchJustCompleted, isEndedNow).
func (s *statsStore) ObserveState(prog *roomProgress, ws *wireSession) (changed bool, completedNow bool, isEndedNow bool) {
	if ws == nil || ws.Mode != "online" {
		return false, false, false
	}
	now := time.Now().UTC()
	dateStr := now.Format("2006-01-02")
	isoStr := now.Format(time.RFC3339)

	s.mu.Lock()
	defer s.mu.Unlock()

	if !prog.countedMatch {
		prog.countedMatch = true
		changed = true
		switch ws.BestOf {
		case 0:
			s.data.BestOfCounts["free"]++
		case 1:
			s.data.BestOfCounts["1"]++
		case 3:
			s.data.BestOfCounts["3"]++
		case 5:
			s.data.BestOfCounts["5"]++
		}
		if !ws.Rules.NoDestroy && !ws.Rules.TwoDice && !ws.Rules.FillBoth {
			s.data.StandardMatches++
		}
		if ws.Rules.TwoDice {
			s.data.TwoDiceMatches++
		}
		if ws.Rules.NoDestroy {
			s.data.NoDestroyMatches++
		}
		if ws.Rules.FillBoth {
			s.data.FillBothMatches++
		}
	}

	if ws.Phase == phasePlay {
		prog.ended = false
	}

	mvKey := [3]int{ws.Series, ws.Round, ws.Match.Moves}
	if ws.Last != nil && ws.Match.Moves > 0 && cmpTriplet(mvKey, prog.lastMoveKey) > 0 {
		prog.lastMoveKey = mvKey
		prog.anyMovePlayed = true
		changed = true
		s.data.TotalDicePlayed++
		if ws.Last.Die >= 1 && ws.Last.Die <= 6 {
			s.data.DiceHistogram[ws.Last.Die-1]++
		}
		if ws.Last.Destroyed > 0 {
			s.data.TotalDiceDestroyed += uint64(ws.Last.Destroyed)
			if ws.Last.Destroyed > s.data.MostDestroyedInMove {
				s.data.MostDestroyedInMove = ws.Last.Destroyed
			}
		}
		if ws.Last.Lost > 0 {
			s.data.TotalPointsDestroyed += uint64(ws.Last.Lost)
		}
	}

	roundKey := [2]int{ws.Series, ws.Round}
	if ws.Phase != phasePlay && ws.Forfeit == nil && cmpPair(roundKey, prog.lastRoundKey) > 0 {
		prog.lastRoundKey = roundKey
		changed = true
		s.data.CompletedRounds++

		scores := [2]int{ws.Match.Grids[0].score(), ws.Match.Grids[1].score()}
		if n := len(ws.History); n > 0 && ws.History[n-1].Round == ws.Round {
			scores = ws.History[n-1].Scores
		}
		s.data.TotalPointsScored += uint64(scores[0] + scores[1])
		if scores[0] == scores[1] {
			s.data.RoundDraws++
		}
		n0 := cleanDisplay(ws.Names[0], "Host", 16)
		n1 := cleanDisplay(ws.Names[1], "Guest", 16)
		names := [2]string{n0, n1}
		for p := 0; p < 2; p++ {
			rec := ScoreRecord{
				Name:       names[p],
				Score:      scores[p],
				Opponent:   names[1-p],
				Against:    scores[1-p],
				BestOf:     ws.BestOf,
				Rules:      ws.Rules,
				RecordedAt: dateStr,
			}
			if s.data.HighestRound == nil || rec.Score > s.data.HighestRound.Score {
				cp := rec
				s.data.HighestRound = &cp
			}
			s.insertTopScoreLocked(rec)
		}
		margin := scores[0] - scores[1]
		winIdx := 0
		if margin < 0 {
			margin = -margin
			winIdx = 1
		}
		if margin > 0 && (s.data.BiggestMargin == nil || margin > s.data.BiggestMargin.Margin) {
			s.data.BiggestMargin = &MarginRecord{
				Winner:     names[winIdx],
				Loser:      names[1-winIdx],
				WinScore:   scores[winIdx],
				LoseScore:  scores[1-winIdx],
				Margin:     margin,
				RecordedAt: dateStr,
			}
		}
	}

	if ws.Phase == phaseGameOver && ws.Series > prog.lastSeriesKey {
		prog.lastSeriesKey = ws.Series
		prog.ended = true
		changed = true
		if ws.Forfeit != nil {
			reason := ws.Forfeit.Reason
			if reason == "" {
				reason = "resignation"
			}
			s.data.Forfeits[reason]++
		}
		if ws.Forfeit == nil || prog.anyMovePlayed || len(ws.History) > 0 {
			s.data.CompletedMatches++
			completedNow = true
			winner := -1
			if ws.Forfeit != nil {
				if ws.Forfeit.Loser == 0 || ws.Forfeit.Loser == 1 {
					winner = 1 - ws.Forfeit.Loser
				}
			} else if ws.Wins[0] > ws.Wins[1] {
				winner = 0
			} else if ws.Wins[1] > ws.Wins[0] {
				winner = 1
			}
			mr := MatchRecord{
				Names:      [2]string{cleanDisplay(ws.Names[0], "Host", 16), cleanDisplay(ws.Names[1], "Guest", 16)},
				Wins:       ws.Wins,
				Ties:       ws.Ties,
				BestOf:     ws.BestOf,
				Rules:      ws.Rules,
				Winner:     winner,
				FinishedAt: isoStr,
			}
			if ws.Forfeit != nil {
				mr.Forfeit = ws.Forfeit.Reason
			}
			s.data.RecentMatches = append([]MatchRecord{mr}, s.data.RecentMatches...)
			if len(s.data.RecentMatches) > maxRecentMatches {
				s.data.RecentMatches = s.data.RecentMatches[:maxRecentMatches]
			}
		}
	}

	if changed {
		s.data.UpdatedAt = now.UnixMilli()
		s.dirty = true
		if completedNow || time.Since(s.lastSave) >= 5*time.Second {
			s.flushLocked()
		}
	}
	return changed, completedNow, prog.ended
}

func (s *statsStore) insertTopScoreLocked(rec ScoreRecord) {
	if rec.Score <= 0 {
		return
	}
	s.data.TopScores = append(s.data.TopScores, rec)
	sort.SliceStable(s.data.TopScores, func(i, j int) bool {
		return s.data.TopScores[i].Score > s.data.TopScores[j].Score
	})
	if len(s.data.TopScores) > maxTopScores {
		s.data.TopScores = s.data.TopScores[:maxTopScores]
	}
}
