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
	statsVersion     = 2
)

type ScoreRecord struct {
	Name       string `json:"name"`
	Score      int    `json:"score"`
	Opponent   string `json:"opponent"`
	Against    int    `json:"against"`
	BestOf     int    `json:"bestOf"`
	Rules      Rules  `json:"rules"`
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
	Challenge  bool      `json:"challenge,omitempty"`
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
	Attempts              uint64            `json:"attempts"`
	Completions           uint64            `json:"completions"`
	PlayerWins            uint64            `json:"playerWins"`
	AIWins                uint64            `json:"aiWins"`
	Ties                  uint64            `json:"ties"`
	Forfeits              uint64            `json:"forfeits"`
	ActiveMatches         int               `json:"activeMatches,omitempty"`
	PeakActiveMatches     int               `json:"peakActiveMatches,omitempty"`
	CompletedRounds       uint64            `json:"completedRounds"`
	PlayerRoundWins       uint64            `json:"playerRoundWins"`
	AIRoundWins           uint64            `json:"aiRoundWins"`
	RoundTies             uint64            `json:"roundTies"`
	FirstMoverRoundWins   uint64            `json:"firstMoverRoundWins"`
	SecondMoverRoundWins  uint64            `json:"secondMoverRoundWins"`
	Sweeps3_0             uint64            `json:"sweeps3_0"`
	AISweeps3_0           uint64            `json:"aiSweeps3_0"`
	TotalDiceRolled       uint64            `json:"totalDiceRolled"`
	TotalDicePlayed       uint64            `json:"totalDicePlayed"`
	PlayerDicePlayed      uint64            `json:"playerDicePlayed"`
	AIDicePlayed          uint64            `json:"aiDicePlayed"`
	DiceHistogram         [6]uint64         `json:"diceHistogram"`
	RollHistogram         [6]uint64         `json:"rollHistogram"`
	TotalDiceDestroyed    uint64            `json:"totalDiceDestroyed"`
	PlayerDiceDestroyed   uint64            `json:"playerDiceDestroyed"`
	AIDiceDestroyed       uint64            `json:"aiDiceDestroyed"`
	TotalPointsScored     uint64            `json:"totalPointsScored"`
	PlayerPointsScored    uint64            `json:"playerPointsScored"`
	AIPointsScored        uint64            `json:"aiPointsScored"`
	TotalPointsDestroyed  uint64            `json:"totalPointsDestroyed"`
	PlayerPointsDestroyed uint64            `json:"playerPointsDestroyed"`
	AIPointsDestroyed     uint64            `json:"aiPointsDestroyed"`
	BestPlayerRound       int               `json:"bestPlayerRound"`
	BestAIRound           int               `json:"bestAiRound"`
	MostDestroyedInMove   int               `json:"mostDestroyedInMove"`
	HighestSingleMoveGain int               `json:"highestSingleMoveGain"`
	BestDelta             int               `json:"bestDelta"`
	HasBestDelta          bool              `json:"hasBestDelta,omitempty"`
	WorstDelta            int               `json:"worstDelta,omitempty"`
	HasWorstDelta         bool              `json:"hasWorstDelta,omitempty"`
	TotalNetDelta         int64             `json:"totalNetDelta"`
	BestTimeMs            int64             `json:"bestTimeMs,omitempty"`
	FastestWinMs          int64             `json:"fastestWinMs,omitempty"`
	TotalDurationMs       uint64            `json:"totalDurationMs"`
	Timeouts              uint64            `json:"timeouts"`
	PlayerTimeouts        uint64            `json:"playerTimeouts"`
	AITimeouts            uint64            `json:"aiTimeouts"`
	ForfeitsByReason      map[string]uint64 `json:"forfeitsByReason,omitempty"`
	Top10                 []ChallengeEntry  `json:"top10"`
	Recent                []ChallengeEntry  `json:"recent"`
	Leaderboard           []ChallengeEntry  `json:"leaderboard,omitempty"`
}

// StatsView is sent to clients in r_lobby / r_hosted and HTTP /stats.
type StatsView struct {
	LobbyWatchers         int               `json:"lobbyWatchers"`
	LiveMatches           int               `json:"liveMatches"`
	WaitingRooms          int               `json:"waitingRooms"`
	CompletedMatches      uint64            `json:"completedMatches"`
	OnlineMatches         uint64            `json:"onlineMatches,omitempty"`
	CompletedRounds       uint64            `json:"completedRounds"`
	RoundDraws            uint64            `json:"roundDraws,omitempty"`
	Shutouts              uint64            `json:"shutouts,omitempty"`
	TotalDiceRolled       uint64            `json:"totalDiceRolled,omitempty"`
	TotalDicePlayed       uint64            `json:"totalDicePlayed"`
	TotalDiceDestroyed    uint64            `json:"totalDiceDestroyed"`
	TotalPointsScored     uint64            `json:"totalPointsScored"`
	TotalPointsDestroyed  uint64            `json:"totalPointsDestroyed,omitempty"`
	TotalTimeouts         uint64            `json:"totalTimeouts,omitempty"`
	DiceHistogram         [6]uint64         `json:"diceHistogram"`
	Forfeits              map[string]uint64 `json:"forfeits,omitempty"`
	HighestRound          *ScoreRecord      `json:"highestRound,omitempty"`
	HighestCombinedRound  int               `json:"highestCombinedRound,omitempty"`
	HighestSingleMoveGain int               `json:"highestSingleMoveGain,omitempty"`
	MostDestroyedInMove   int               `json:"mostDestroyedInMove,omitempty"`
	BiggestMargin         *MarginRecord     `json:"biggestMargin,omitempty"`
	FirstMoverWins        uint64            `json:"firstMoverWins,omitempty"`
	SecondMoverWins       uint64            `json:"secondMoverWins,omitempty"`
	PeakLobbyWatchers     int               `json:"peakLobbyWatchers,omitempty"`
	PeakLiveMatches       int               `json:"peakLiveMatches,omitempty"`
	ServerStartedAt       int64             `json:"serverStartedAt,omitempty"`
	Challenge             *ChallengeStats   `json:"challenge,omitempty"`
	UpdatedAt             int64             `json:"updatedAt"`
}

type PersistentStats struct {
	Version               int               `json:"version"`
	CompletedMatches      uint64            `json:"completedMatches"`
	OnlineMatches         uint64            `json:"onlineMatches"`
	CompletedRounds       uint64            `json:"completedRounds"`
	RoundDraws            uint64            `json:"roundDraws"`
	SeriesDraws           uint64            `json:"seriesDraws"`
	Shutouts              uint64            `json:"shutouts"`
	TotalDiceRolled       uint64            `json:"totalDiceRolled"`
	TotalDicePlayed       uint64            `json:"totalDicePlayed"`
	TotalDiceDestroyed    uint64            `json:"totalDiceDestroyed"`
	TotalPointsScored     uint64            `json:"totalPointsScored"`
	TotalPointsDestroyed  uint64            `json:"totalPointsDestroyed"`
	TotalTimeouts         uint64            `json:"totalTimeouts"`
	TotalMatchDurationMs  uint64            `json:"totalMatchDurationMs"`
	DiceHistogram         [6]uint64         `json:"diceHistogram"`
	Forfeits              map[string]uint64 `json:"forfeits"`
	BestOfCounts          map[string]uint64 `json:"bestOfCounts"`
	StandardMatches       uint64            `json:"standardMatches"`
	TwoDiceMatches        uint64            `json:"twoDiceMatches"`
	NoDestroyMatches      uint64            `json:"noDestroyMatches"`
	FillBothMatches       uint64            `json:"fillBothMatches"`
	HighestRound          *ScoreRecord      `json:"highestRound,omitempty"`
	HighestCombinedRound  int               `json:"highestCombinedRound"`
	BiggestMargin         *MarginRecord     `json:"biggestMargin,omitempty"`
	MostDestroyedInMove   int               `json:"mostDestroyedInMove"`
	HighestSingleMoveGain int               `json:"highestSingleMoveGain"`
	BiggestSingleMoveLoss int               `json:"biggestSingleMoveLoss"`
	FirstMoverWins        uint64            `json:"firstMoverWins"`
	SecondMoverWins       uint64            `json:"secondMoverWins"`
	TotalRoomsCreated     uint64            `json:"totalRoomsCreated"`
	TotalWatchersServed   uint64            `json:"totalWatchersServed"`
	PeakLobbyWatchers     int               `json:"peakLobbyWatchers"`
	PeakLiveMatches       int               `json:"peakLiveMatches"`
	PeakActiveChallenges  int               `json:"peakActiveChallenges"`
	ServerStartedAt       int64             `json:"serverStartedAt"`
	TopScores             []ScoreRecord     `json:"topScores"`
	RecentMatches         []MatchRecord     `json:"recentMatches"`
	Challenge             ChallengeStats    `json:"challenge"`
	UpdatedAt             int64             `json:"updatedAt"`
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
	startedAt     time.Time
	lastRollKey   [3]int // {series, round, moves}
	lastMoveKey   [3]int // {series, round, moves}
	lastRoundKey  [2]int // {series, round}
	lastSeriesKey int
	lastTimeouts  [2]int
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
			Version:      statsVersion,
			Forfeits:     map[string]uint64{},
			BestOfCounts: map[string]uint64{},
			Challenge: ChallengeStats{
				ForfeitsByReason: map[string]uint64{},
			},
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
	if loaded.Challenge.ForfeitsByReason == nil {
		loaded.Challenge.ForfeitsByReason = map[string]uint64{}
	}
	hadSeed := len(loaded.Challenge.Leaderboard) == 0 && (len(loaded.Challenge.Top10) > 0 || len(loaded.Challenge.Recent) > 0)
	loaded.Challenge.Leaderboard, loaded.Challenge.Top10 = mergeChallengeLeaderboard(
		loaded.Challenge.Leaderboard,
		loaded.Challenge.Top10,
		loaded.Challenge.Recent,
	)
	migrated := hadSeed
	if loaded.Version < statsVersion {
		migrated = true
		loaded.OnlineMatches = loaded.CompletedMatches
		loaded.TotalDicePlayed += loaded.Challenge.TotalDicePlayed
		if loaded.Challenge.Completions > 0 {
			loaded.CompletedMatches += loaded.Challenge.Completions
			loaded.BestOfCounts["3"] += loaded.Challenge.Completions
			loaded.StandardMatches += loaded.Challenge.Completions
			rounds := loaded.Challenge.Completions * 3
			loaded.CompletedRounds += rounds
			loaded.Challenge.CompletedRounds += rounds
			for _, e := range loaded.Challenge.Recent {
				loaded.Challenge.PlayerRoundWins += uint64(e.Wins)
				loaded.Challenge.AIRoundWins += uint64(e.Losses)
				loaded.Challenge.RoundTies += uint64(e.Ties)
				loaded.RoundDraws += uint64(e.Ties)
				pts := uint64(e.PlayerTotal + e.AITotal)
				loaded.Challenge.PlayerPointsScored += uint64(e.PlayerTotal)
				loaded.Challenge.AIPointsScored += uint64(e.AITotal)
				loaded.Challenge.TotalPointsScored += pts
				loaded.TotalPointsScored += pts
				loaded.Challenge.TotalDurationMs += uint64(e.DurationMs)
				loaded.TotalMatchDurationMs += uint64(e.DurationMs)
				loaded.Challenge.TotalNetDelta += int64(e.Delta)
				if e.Wins == 3 && e.Losses == 0 && e.Ties == 0 {
					loaded.Challenge.Sweeps3_0++
				}
				if e.Losses == 3 && e.Wins == 0 && e.Ties == 0 {
					loaded.Challenge.AISweeps3_0++
				}
				if e.Delta > 0 && (loaded.Challenge.FastestWinMs == 0 || e.DurationMs < loaded.Challenge.FastestWinMs) {
					loaded.Challenge.FastestWinMs = e.DurationMs
				}
				if !loaded.Challenge.HasWorstDelta || e.Delta < loaded.Challenge.WorstDelta {
					loaded.Challenge.HasWorstDelta = true
					loaded.Challenge.WorstDelta = e.Delta
				}
			}
		}
	}
	loaded.Version = statsVersion
	s.data = loaded
	if migrated {
		s.dirty = true
		s.flushLocked()
	}
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

func copyUint64Map(m map[string]uint64) map[string]uint64 {
	if len(m) == 0 {
		return map[string]uint64{}
	}
	out := make(map[string]uint64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (s *statsStore) SnapshotWithChallenges(live, waiting, lobbyWatchers, activeChallenges int) StatsView {
	s.mu.Lock()
	defer s.mu.Unlock()
	forfeits := copyUint64Map(s.data.Forfeits)
	var highCopy *ScoreRecord
	if s.data.HighestRound != nil {
		cp := *s.data.HighestRound
		highCopy = &cp
	}
	var marginCopy *MarginRecord
	if s.data.BiggestMargin != nil {
		cp := *s.data.BiggestMargin
		marginCopy = &cp
	}
	chCopy := s.data.Challenge
	chCopy.ActiveMatches = activeChallenges
	chCopy.ForfeitsByReason = copyUint64Map(s.data.Challenge.ForfeitsByReason)
	if len(s.data.Challenge.Top10) > 0 {
		chCopy.Top10 = append([]ChallengeEntry(nil), s.data.Challenge.Top10...)
	} else {
		chCopy.Top10 = []ChallengeEntry{}
	}
	chCopy.Recent = nil
	chCopy.Leaderboard = nil
	return StatsView{
		LobbyWatchers:         lobbyWatchers,
		LiveMatches:           live,
		WaitingRooms:          waiting,
		CompletedMatches:      s.data.CompletedMatches,
		OnlineMatches:         s.data.OnlineMatches,
		CompletedRounds:       s.data.CompletedRounds,
		RoundDraws:            s.data.RoundDraws,
		Shutouts:              s.data.Shutouts,
		TotalDiceRolled:       s.data.TotalDiceRolled,
		TotalDicePlayed:       s.data.TotalDicePlayed,
		TotalDiceDestroyed:    s.data.TotalDiceDestroyed,
		TotalPointsScored:     s.data.TotalPointsScored,
		TotalPointsDestroyed:  s.data.TotalPointsDestroyed,
		TotalTimeouts:         s.data.TotalTimeouts,
		DiceHistogram:         s.data.DiceHistogram,
		Forfeits:              forfeits,
		HighestRound:          highCopy,
		HighestCombinedRound:  s.data.HighestCombinedRound,
		HighestSingleMoveGain: s.data.HighestSingleMoveGain,
		MostDestroyedInMove:   s.data.MostDestroyedInMove,
		BiggestMargin:         marginCopy,
		FirstMoverWins:        s.data.FirstMoverWins,
		SecondMoverWins:       s.data.SecondMoverWins,
		PeakLobbyWatchers:     s.data.PeakLobbyWatchers,
		PeakLiveMatches:       s.data.PeakLiveMatches,
		ServerStartedAt:       s.data.ServerStartedAt,
		Challenge:             &chCopy,
		UpdatedAt:             s.data.UpdatedAt,
	}
}

func (s *statsStore) FullDump(live, waitingCount, lobbyWatchers int, waiting []RoomInfo, matches []LiveMatchInfo) map[string]any {
	return s.FullDumpWithChallenges(live, waitingCount, lobbyWatchers, 0, waiting, matches)
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

func (s *statsStore) recordMoveOutcomeLocked(pl wirePlacement) {
	s.data.TotalDicePlayed++
	if pl.Die >= 1 && pl.Die <= 6 {
		s.data.DiceHistogram[pl.Die-1]++
	}
	if pl.Destroyed > 0 {
		s.data.TotalDiceDestroyed += uint64(pl.Destroyed)
		if pl.Destroyed > s.data.MostDestroyedInMove {
			s.data.MostDestroyedInMove = pl.Destroyed
		}
	}
	if pl.Lost > 0 {
		s.data.TotalPointsDestroyed += uint64(pl.Lost)
		if pl.Lost > s.data.BiggestSingleMoveLoss {
			s.data.BiggestSingleMoveLoss = pl.Lost
		}
	}
	if pl.Gain > s.data.HighestSingleMoveGain {
		s.data.HighestSingleMoveGain = pl.Gain
	}
}

func (s *statsStore) recordRoundOutcomeLocked(names [2]string, scores [2]int, roundFirst, bestOf int, rules Rules, dateStr string) {
	s.data.CompletedRounds++
	combined := scores[0] + scores[1]
	s.data.TotalPointsScored += uint64(combined)
	if combined > s.data.HighestCombinedRound {
		s.data.HighestCombinedRound = combined
	}
	if scores[0] == scores[1] {
		s.data.RoundDraws++
	} else {
		if scores[0] == 0 || scores[1] == 0 {
			s.data.Shutouts++
		}
		winner := 0
		if scores[1] > scores[0] {
			winner = 1
		}
		if roundFirst == 0 || roundFirst == 1 {
			if winner == roundFirst {
				s.data.FirstMoverWins++
			} else {
				s.data.SecondMoverWins++
			}
		}
	}
	for p := 0; p < 2; p++ {
		rec := ScoreRecord{
			Name:       names[p],
			Score:      scores[p],
			Opponent:   names[1-p],
			Against:    scores[1-p],
			BestOf:     bestOf,
			Rules:      rules,
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

func (s *statsStore) prependRecentMatchLocked(mr MatchRecord) {
	s.data.RecentMatches = append([]MatchRecord{mr}, s.data.RecentMatches...)
	if len(s.data.RecentMatches) > maxRecentMatches {
		s.data.RecentMatches = s.data.RecentMatches[:maxRecentMatches]
	}
}

func (s *statsStore) markDirtyLocked(now time.Time, forceFlush bool) {
	s.data.UpdatedAt = now.UnixMilli()
	s.dirty = true
	if forceFlush || time.Since(s.lastSave) >= 5*time.Second {
		s.flushLocked()
	}
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

	for p := 0; p < 2; p++ {
		if ws.Timeouts[p] > prog.lastTimeouts[p] {
			s.data.TotalTimeouts += uint64(ws.Timeouts[p] - prog.lastTimeouts[p])
			prog.lastTimeouts[p] = ws.Timeouts[p]
			changed = true
		} else if ws.Timeouts[p] < prog.lastTimeouts[p] {
			prog.lastTimeouts[p] = ws.Timeouts[p]
		}
	}

	mvKey := [3]int{ws.Series, ws.Round, ws.Match.Moves}
	if ws.Phase == phasePlay && ws.Match.Dice[0] > 0 && cmpTriplet(mvKey, prog.lastRollKey) > 0 {
		prog.lastRollKey = mvKey
		rolls := uint64(1)
		if ws.Match.Dice[1] > 0 {
			rolls = 2
		}
		s.data.TotalDiceRolled += rolls
		changed = true
	}

	if ws.Last != nil && ws.Match.Moves > 0 && cmpTriplet(mvKey, prog.lastMoveKey) > 0 {
		prog.lastMoveKey = mvKey
		prog.anyMovePlayed = true
		if prog.startedAt.IsZero() {
			prog.startedAt = now
		}
		changed = true
		s.recordMoveOutcomeLocked(*ws.Last)
	}

	roundKey := [2]int{ws.Series, ws.Round}
	if ws.Phase != phasePlay && ws.Forfeit == nil && cmpPair(roundKey, prog.lastRoundKey) > 0 {
		prog.lastRoundKey = roundKey
		changed = true

		scores := [2]int{ws.Match.Grids[0].score(), ws.Match.Grids[1].score()}
		if n := len(ws.History); n > 0 && ws.History[n-1].Round == ws.Round {
			scores = ws.History[n-1].Scores
		}
		n0 := cleanDisplay(ws.Names[0], "Host", 16)
		n1 := cleanDisplay(ws.Names[1], "Guest", 16)
		s.recordRoundOutcomeLocked([2]string{n0, n1}, scores, ws.RoundFirst, ws.BestOf, ws.Rules, dateStr)
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
			s.data.OnlineMatches++
			completedNow = true
			if !prog.startedAt.IsZero() {
				if d := now.Sub(prog.startedAt).Milliseconds(); d > 0 {
					s.data.TotalMatchDurationMs += uint64(d)
				}
				prog.startedAt = time.Time{}
			}
			winner := -1
			if ws.Forfeit != nil {
				if ws.Forfeit.Loser == 0 || ws.Forfeit.Loser == 1 {
					winner = 1 - ws.Forfeit.Loser
				}
			} else if ws.Wins[0] > ws.Wins[1] {
				winner = 0
			} else if ws.Wins[1] > ws.Wins[0] {
				winner = 1
			} else {
				s.data.SeriesDraws++
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
			s.prependRecentMatchLocked(mr)
		}
	}

	if changed {
		s.markDirtyLocked(now, completedNow)
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

func mergeChallengeLeaderboard(lists ...[]ChallengeEntry) ([]ChallengeEntry, []ChallengeEntry) {
	type dedupKey struct {
		name       string
		delta      int
		durationMs int64
		recordedAt string
		rounds     [3]int
	}
	seen := make(map[dedupKey]struct{})
	var all []ChallengeEntry
	for _, list := range lists {
		for _, e := range list {
			clean, bad := ValidateChallengeName(e.Name)
			if bad != "" {
				continue
			}
			e.Name = clean
			k := dedupKey{
				name:       e.Name,
				delta:      e.Delta,
				durationMs: e.DurationMs,
				recordedAt: e.RecordedAt,
				rounds:     e.RoundDeltas,
			}
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			all = append(all, e)
		}
	}
	sortChallengeEntries(all)
	topN := len(all)
	if topN > maxTopScores {
		topN = maxTopScores
	}
	top10 := append([]ChallengeEntry(nil), all[:topN]...)
	return all, top10
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
	dataCopy.Forfeits = copyUint64Map(s.data.Forfeits)
	dataCopy.BestOfCounts = copyUint64Map(s.data.BestOfCounts)
	dataCopy.TopScores = append([]ScoreRecord(nil), s.data.TopScores...)
	dataCopy.RecentMatches = append([]MatchRecord(nil), s.data.RecentMatches...)
	dataCopy.Challenge.ActiveMatches = activeChallenges
	dataCopy.Challenge.ForfeitsByReason = copyUint64Map(s.data.Challenge.ForfeitsByReason)
	dataCopy.Challenge.Top10 = append([]ChallengeEntry(nil), s.data.Challenge.Top10...)
	dataCopy.Challenge.Recent = append([]ChallengeEntry(nil), s.data.Challenge.Recent...)
	dataCopy.Challenge.Leaderboard = append([]ChallengeEntry(nil), s.data.Challenge.Leaderboard...)
	return map[string]any{
		"lobbyWatchers":    lobbyWatchers,
		"liveMatches":      live,
		"waitingRooms":     waitingCount,
		"activeChallenges": activeChallenges,
		"rooms":            waiting,
		"matches":          matches,
		"stats":            dataCopy,
	}
}

func (s *statsStore) MarkServerStart(startedAtMs int64) {
	s.mu.Lock()
	s.data.ServerStartedAt = startedAtMs
	s.mu.Unlock()
}

func (s *statsStore) RecordWatcherConnected(watchers int) {
	now := time.Now().UTC()
	s.mu.Lock()
	s.data.TotalWatchersServed++
	if watchers > s.data.PeakLobbyWatchers {
		s.data.PeakLobbyWatchers = watchers
	}
	s.markDirtyLocked(now, false)
	s.mu.Unlock()
}

func (s *statsStore) RecordRoomCreated() {
	now := time.Now().UTC()
	s.mu.Lock()
	s.data.TotalRoomsCreated++
	s.markDirtyLocked(now, false)
	s.mu.Unlock()
}

func (s *statsStore) ObserveLivePeak(live int) {
	s.mu.Lock()
	if live > s.data.PeakLiveMatches {
		s.data.PeakLiveMatches = live
		s.markDirtyLocked(time.Now().UTC(), false)
	}
	s.mu.Unlock()
}

func (s *statsStore) ObserveChallengePeak(active int) {
	s.mu.Lock()
	changed := false
	if active > s.data.PeakActiveChallenges {
		s.data.PeakActiveChallenges = active
		changed = true
	}
	if active > s.data.Challenge.PeakActiveMatches {
		s.data.Challenge.PeakActiveMatches = active
		changed = true
	}
	if changed {
		s.markDirtyLocked(time.Now().UTC(), false)
	}
	s.mu.Unlock()
}

func (s *statsStore) RecordChallengeAttempt() {
	now := time.Now().UTC()
	s.mu.Lock()
	s.data.Challenge.Attempts++
	s.markDirtyLocked(now, false)
	s.mu.Unlock()
}

func (s *statsStore) RecordChallengeRoll(die int) {
	now := time.Now().UTC()
	s.mu.Lock()
	s.data.TotalDiceRolled++
	s.data.Challenge.TotalDiceRolled++
	if die >= 1 && die <= 6 {
		s.data.Challenge.RollHistogram[die-1]++
	}
	s.markDirtyLocked(now, false)
	s.mu.Unlock()
}

func (s *statsStore) RecordChallengeMove(pl wirePlacement) {
	now := time.Now().UTC()
	s.mu.Lock()
	s.recordMoveOutcomeLocked(pl)
	s.data.Challenge.TotalDicePlayed++
	if pl.Player == 1 {
		s.data.Challenge.PlayerDicePlayed++
	} else {
		s.data.Challenge.AIDicePlayed++
	}
	if pl.Die >= 1 && pl.Die <= 6 {
		s.data.Challenge.DiceHistogram[pl.Die-1]++
	}
	if pl.Destroyed > 0 {
		s.data.Challenge.TotalDiceDestroyed += uint64(pl.Destroyed)
		if pl.Player == 1 {
			s.data.Challenge.PlayerDiceDestroyed += uint64(pl.Destroyed)
		} else {
			s.data.Challenge.AIDiceDestroyed += uint64(pl.Destroyed)
		}
		if pl.Destroyed > s.data.Challenge.MostDestroyedInMove {
			s.data.Challenge.MostDestroyedInMove = pl.Destroyed
		}
	}
	if pl.Lost > 0 {
		s.data.Challenge.TotalPointsDestroyed += uint64(pl.Lost)
		if pl.Player == 1 {
			s.data.Challenge.PlayerPointsDestroyed += uint64(pl.Lost)
		} else {
			s.data.Challenge.AIPointsDestroyed += uint64(pl.Lost)
		}
	}
	if pl.Gain > s.data.Challenge.HighestSingleMoveGain {
		s.data.Challenge.HighestSingleMoveGain = pl.Gain
	}
	s.markDirtyLocked(now, false)
	s.mu.Unlock()
}

func (s *statsStore) RecordChallengeTimeout(turn int) {
	now := time.Now().UTC()
	s.mu.Lock()
	s.data.TotalTimeouts++
	s.data.Challenge.Timeouts++
	if turn == 1 {
		s.data.Challenge.PlayerTimeouts++
	} else {
		s.data.Challenge.AITimeouts++
	}
	s.markDirtyLocked(now, false)
	s.mu.Unlock()
}

func (s *statsStore) RecordChallengeRound(playerName string, scores [2]int, roundFirst, winner int) {
	now := time.Now().UTC()
	dateStr := now.Format("2006-01-02")
	cleanPlayer := cleanDisplay(playerName, "Challenger", 16)

	s.mu.Lock()
	s.recordRoundOutcomeLocked([2]string{"Shrumy", cleanPlayer}, scores, roundFirst, 3, Rules{}, dateStr)
	s.data.Challenge.CompletedRounds++
	s.data.Challenge.TotalPointsScored += uint64(scores[0] + scores[1])
	s.data.Challenge.AIPointsScored += uint64(scores[0])
	s.data.Challenge.PlayerPointsScored += uint64(scores[1])
	if scores[1] > scores[0] {
		s.data.Challenge.PlayerRoundWins++
	} else if scores[0] > scores[1] {
		s.data.Challenge.AIRoundWins++
	} else {
		s.data.Challenge.RoundTies++
	}
	if roundFirst == 0 || roundFirst == 1 {
		if winner == roundFirst {
			s.data.Challenge.FirstMoverRoundWins++
		} else if winner == 1-roundFirst {
			s.data.Challenge.SecondMoverRoundWins++
		}
	}
	if scores[1] > s.data.Challenge.BestPlayerRound {
		s.data.Challenge.BestPlayerRound = scores[1]
	}
	if scores[0] > s.data.Challenge.BestAIRound {
		s.data.Challenge.BestAIRound = scores[0]
	}
	s.markDirtyLocked(now, false)
	s.mu.Unlock()
}

func (s *statsStore) RecordChallengeForfeit(playerName, reason string, loser int, wins [2]int, ties int, hadMoves bool) {
	now := time.Now().UTC()
	if reason == "" {
		reason = "resignation"
	}
	s.mu.Lock()
	s.data.Challenge.Forfeits++
	if s.data.Challenge.ForfeitsByReason == nil {
		s.data.Challenge.ForfeitsByReason = map[string]uint64{}
	}
	s.data.Challenge.ForfeitsByReason[reason]++
	s.data.Forfeits[reason]++
	if hadMoves {
		s.data.CompletedMatches++
		s.data.BestOfCounts["3"]++
		s.data.StandardMatches++
		winner := -1
		if loser == 0 || loser == 1 {
			winner = 1 - loser
		}
		s.prependRecentMatchLocked(MatchRecord{
			Names:      [2]string{"Shrumy", cleanDisplay(playerName, "Challenger", 16)},
			Wins:       wins,
			Ties:       ties,
			BestOf:     3,
			Challenge:  true,
			Forfeit:    reason,
			Winner:     winner,
			FinishedAt: now.Format(time.RFC3339),
		})
	}
	s.markDirtyLocked(now, true)
	s.mu.Unlock()
}

// RecordChallengeCompletion records a finished 3-match Challenge run, inserts
// eligible entries (non-Lamb, non-offensive) into the full Leaderboard and Top 10
// ranked by delta DESC then durationMs ASC, and returns the 1-based overall rank
// (or 0 if the name is ineligible).
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

	s.data.CompletedMatches++
	s.data.BestOfCounts["3"]++
	s.data.StandardMatches++
	s.data.TotalMatchDurationMs += uint64(entry.DurationMs)

	s.data.Challenge.Completions++
	s.data.Challenge.TotalDurationMs += uint64(entry.DurationMs)
	s.data.Challenge.TotalNetDelta += int64(entry.Delta)
	winner := -1
	switch {
	case entry.Delta > 0:
		s.data.Challenge.PlayerWins++
		winner = 1
		if s.data.Challenge.FastestWinMs == 0 || entry.DurationMs < s.data.Challenge.FastestWinMs {
			s.data.Challenge.FastestWinMs = entry.DurationMs
		}
	case entry.Delta < 0:
		s.data.Challenge.AIWins++
		winner = 0
	default:
		s.data.Challenge.Ties++
		s.data.SeriesDraws++
	}
	if entry.Wins == 3 && entry.Losses == 0 && entry.Ties == 0 {
		s.data.Challenge.Sweeps3_0++
	}
	if entry.Losses == 3 && entry.Wins == 0 && entry.Ties == 0 {
		s.data.Challenge.AISweeps3_0++
	}
	if !s.data.Challenge.HasBestDelta ||
		entry.Delta > s.data.Challenge.BestDelta ||
		(entry.Delta == s.data.Challenge.BestDelta && (s.data.Challenge.BestTimeMs == 0 || entry.DurationMs < s.data.Challenge.BestTimeMs)) {
		s.data.Challenge.HasBestDelta = true
		s.data.Challenge.BestDelta = entry.Delta
		s.data.Challenge.BestTimeMs = entry.DurationMs
	}
	if !s.data.Challenge.HasWorstDelta || entry.Delta < s.data.Challenge.WorstDelta {
		s.data.Challenge.HasWorstDelta = true
		s.data.Challenge.WorstDelta = entry.Delta
	}

	rank := 0
	if clean, bad := ValidateChallengeName(entry.Name); bad == "" {
		entry.Name = clean
		s.data.Challenge.Recent = append([]ChallengeEntry{entry}, s.data.Challenge.Recent...)
		if len(s.data.Challenge.Recent) > maxRecentMatches {
			s.data.Challenge.Recent = s.data.Challenge.Recent[:maxRecentMatches]
		}
		if len(s.data.Challenge.Leaderboard) < len(s.data.Challenge.Top10) {
			s.data.Challenge.Leaderboard, _ = mergeChallengeLeaderboard(
				s.data.Challenge.Leaderboard,
				s.data.Challenge.Top10,
				s.data.Challenge.Recent,
			)
		}
		s.data.Challenge.Leaderboard = append(s.data.Challenge.Leaderboard, entry)
		sortChallengeEntries(s.data.Challenge.Leaderboard)
		for i := range s.data.Challenge.Leaderboard {
			e := s.data.Challenge.Leaderboard[i]
			if e.Name == entry.Name &&
				e.Delta == entry.Delta &&
				e.DurationMs == entry.DurationMs &&
				e.RecordedAt == entry.RecordedAt &&
				e.RoundDeltas == entry.RoundDeltas {
				rank = i + 1
				break
			}
		}
		topN := len(s.data.Challenge.Leaderboard)
		if topN > maxTopScores {
			topN = maxTopScores
		}
		s.data.Challenge.Top10 = append([]ChallengeEntry(nil), s.data.Challenge.Leaderboard[:topN]...)
	}

	s.prependRecentMatchLocked(MatchRecord{
		Names:      [2]string{"Shrumy", cleanDisplay(entry.Name, "Challenger", 16)},
		Wins:       [2]int{entry.Losses, entry.Wins},
		Ties:       entry.Ties,
		BestOf:     3,
		Challenge:  true,
		Winner:     winner,
		FinishedAt: entry.RecordedAt,
	})

	s.markDirtyLocked(now, true)
	return rank
}
