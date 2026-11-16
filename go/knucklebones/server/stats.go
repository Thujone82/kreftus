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

type ChallengeEntry struct {
	Name        string `json:"name"`
	Delta       int    `json:"delta"`
	DurationMs  int64  `json:"durationMs"`
	RoundDeltas [3]int `json:"roundDeltas"`
	PlayerTotal int    `json:"playerTotal"`
	AITotal     int    `json:"aiTotal"`
	Wins        int    `json:"wins"`
	Losses      int    `json:"losses"`
	Ties        int    `json:"ties"`
	RecordedAt  string `json:"recordedAt"`
}

type ChallengeStats struct {
	Attempts        uint64           `json:"attempts"`
	Completions     uint64           `json:"completions"`
	PlayerWins      uint64           `json:"playerWins"`
	AIWins          uint64           `json:"aiWins"`
	Ties            uint64           `json:"ties"`
	Forfeits        uint64           `json:"forfeits"`
	ActiveMatches   int              `json:"activeMatches,omitempty"`
	TotalDicePlayed uint64           `json:"totalDicePlayed"`
	BestDelta       int              `json:"bestDelta"`
	HasBestDelta    bool             `json:"hasBestDelta,omitempty"`
	BestTimeMs      int64            `json:"bestTimeMs,omitempty"`
	Top10           []ChallengeEntry `json:"top10"`
	Recent          []ChallengeEntry `json:"recent"`
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
	Challenge          *ChallengeStats   `json:"challenge,omitempty"`
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
	Challenge            ChallengeStats    `json:"challenge"`
	UpdatedAt            int64             `json:"updatedAt"`
}

type LiveMatchInfo struct {
	ID     string `json:"id"`
	Host   string `json:"host"`
	Guest  string `json:"guest"`
	Round  int    `json:"round"`
	BestOf int    `json:"bestOf"`
	Wins   [2]int `json:"wins"`
	Scores [2]int `json:"scores"`
}

type roomProgress struct {
	countedMatch  bool
	anyMovePlayed bool
	lastMoveKey   [3]int // {series, round, moves}
	lastRoundKey  [2]int // {series, round}
	lastSeriesKey int
	ended         bool
	liveRound     int
	liveNames     [2]string
	liveWins      [2]int
	liveScores    [2]int
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
	if len(loaded.Challenge.Top10) > 0 {
		cleanTop := loaded.Challenge.Top10[:0]
		for _, e := range loaded.Challenge.Top10 {
			if _, bad := ValidateChallengeName(e.Name); bad == "" {
				cleanTop = append(cleanTop, e)
			}
		}
		loaded.Challenge.Top10 = cleanTop
		sortChallengeEntries(loaded.Challenge.Top10)
		if len(loaded.Challenge.Top10) > maxTopScores {
			loaded.Challenge.Top10 = loaded.Challenge.Top10[:maxTopScores]
		}
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
	return s.SnapshotWithChallenges(live, waiting, lobbyWatchers, 0)
}

func (s *statsStore) SnapshotWithChallenges(live, waiting, lobbyWatchers, activeChallenges int) StatsView {
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
	chCopy := s.data.Challenge
	chCopy.ActiveMatches = activeChallenges
	if len(s.data.Challenge.Top10) > 0 {
		chCopy.Top10 = append([]ChallengeEntry(nil), s.data.Challenge.Top10...)
	} else {
		chCopy.Top10 = []ChallengeEntry{}
	}
	chCopy.Recent = nil
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
		Challenge:          &chCopy,
		UpdatedAt:          s.data.UpdatedAt,
	}
}

func (s *statsStore) FullDump(live, waitingCount, lobbyWatchers int, waiting []RoomInfo, matches []LiveMatchInfo) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	if waiting == nil {
		waiting = []RoomInfo{}
	}
	if matches == nil {
		matches = []LiveMatchInfo{}
	}
	return map[string]any{
		"lobbyWatchers": lobbyWatchers,
		"liveMatches":   live,
		"waitingRooms":  waitingCount,
		"rooms":         waiting,
		"matches":       matches,
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
	prog.liveRound = ws.Round
	prog.liveNames = [2]string{
		cleanDisplay(ws.Names[0], "Host", 16),
		cleanDisplay(ws.Names[1], "Guest", 16),
	}
	prog.liveWins = ws.Wins
	prog.liveScores = [2]int{ws.Match.Grids[0].score(), ws.Match.Grids[1].score()}

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

func sortChallengeEntries(list []ChallengeEntry) {
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Delta != list[j].Delta {
			return list[i].Delta > list[j].Delta
		}
		if list[i].DurationMs != list[j].DurationMs {
			return list[i].DurationMs < list[j].DurationMs
		}
		return list[i].RecordedAt < list[j].RecordedAt
	})
}

func (s *statsStore) FullDumpWithChallenges(live, waitingCount, lobbyWatchers, activeChallenges int, waiting []RoomInfo, matches []LiveMatchInfo) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	if waiting == nil {
		waiting = []RoomInfo{}
	}
	if matches == nil {
		matches = []LiveMatchInfo{}
	}
	dataCopy := s.data
	dataCopy.Challenge.ActiveMatches = activeChallenges
	return map[string]any{
		"lobbyWatchers": lobbyWatchers,
		"liveMatches":   live,
		"waitingRooms":  waitingCount,
		"rooms":         waiting,
		"matches":       matches,
		"stats":         dataCopy,
	}
}

func (s *statsStore) RecordChallengeAttempt() {
	now := time.Now().UTC()
	s.mu.Lock()
	s.data.Challenge.Attempts++
	s.data.UpdatedAt = now.UnixMilli()
	s.dirty = true
	if time.Since(s.lastSave) >= 5*time.Second {
		s.flushLocked()
	}
	s.mu.Unlock()
}

func (s *statsStore) RecordChallengeDie() {
	now := time.Now().UTC()
	s.mu.Lock()
	s.data.Challenge.TotalDicePlayed++
	s.data.UpdatedAt = now.UnixMilli()
	s.dirty = true
	if time.Since(s.lastSave) >= 5*time.Second {
		s.flushLocked()
	}
	s.mu.Unlock()
}

func (s *statsStore) RecordChallengeForfeit() {
	now := time.Now().UTC()
	s.mu.Lock()
	s.data.Challenge.Forfeits++
	s.data.UpdatedAt = now.UnixMilli()
	s.dirty = true
	s.flushLocked()
	s.mu.Unlock()
}

// RecordChallengeCompletion records a finished 3-match Challenge run, inserts
// eligible entries (non-Lamb, non-offensive) into the Top 10 ranked by delta DESC
// then durationMs ASC, and returns the 1-based leaderboard rank (or 0 if outside Top 10).
func (s *statsStore) RecordChallengeCompletion(entry ChallengeEntry) int {
	now := time.Now().UTC()
	if entry.RecordedAt == "" {
		entry.RecordedAt = now.Format(time.RFC3339)
	}
	if entry.DurationMs < 1 {
		entry.DurationMs = 1
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.Challenge.Completions++
	switch {
	case entry.Delta > 0:
		s.data.Challenge.PlayerWins++
	case entry.Delta < 0:
		s.data.Challenge.AIWins++
	default:
		s.data.Challenge.Ties++
	}
	if !s.data.Challenge.HasBestDelta ||
		entry.Delta > s.data.Challenge.BestDelta ||
		(entry.Delta == s.data.Challenge.BestDelta && (s.data.Challenge.BestTimeMs == 0 || entry.DurationMs < s.data.Challenge.BestTimeMs)) {
		s.data.Challenge.HasBestDelta = true
		s.data.Challenge.BestDelta = entry.Delta
		s.data.Challenge.BestTimeMs = entry.DurationMs
	}

	rank := 0
	if clean, bad := ValidateChallengeName(entry.Name); bad == "" {
		entry.Name = clean
		s.data.Challenge.Recent = append([]ChallengeEntry{entry}, s.data.Challenge.Recent...)
		if len(s.data.Challenge.Recent) > maxRecentMatches {
			s.data.Challenge.Recent = s.data.Challenge.Recent[:maxRecentMatches]
		}
		s.data.Challenge.Top10 = append(s.data.Challenge.Top10, entry)
		sortChallengeEntries(s.data.Challenge.Top10)
		for i := range s.data.Challenge.Top10 {
			if i >= maxTopScores {
				break
			}
			if s.data.Challenge.Top10[i].Name == entry.Name &&
				s.data.Challenge.Top10[i].Delta == entry.Delta &&
				s.data.Challenge.Top10[i].DurationMs == entry.DurationMs &&
				s.data.Challenge.Top10[i].RecordedAt == entry.RecordedAt {
				rank = i + 1
				break
			}
		}
		if len(s.data.Challenge.Top10) > maxTopScores {
			s.data.Challenge.Top10 = s.data.Challenge.Top10[:maxTopScores]
		}
	}

	s.data.UpdatedAt = now.UnixMilli()
	s.dirty = true
	s.flushLocked()
	return rank
}
