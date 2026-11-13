package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net"
	"strings"
	"time"
	"unicode"
)

const (
	gridCols = 3
	gridRows = 3

	phasePlay      = "play"
	phaseRoundOver = "round"
	phaseGameOver  = "game"

	rMsgWatch       = "r_watch"
	rMsgLobby       = "r_lobby"
	rMsgHost        = "r_host"
	rMsgHosted      = "r_hosted"
	rMsgJoin        = "r_join"
	rMsgPaired      = "r_paired"
	rMsgGuestJoined = "r_guest_joined"
	rMsgGuestDrop   = "r_guest_drop"
	rMsgRelay       = "r_relay"
	rMsgError       = "r_error"
	rMsgPing        = "ping"
	rMsgPong        = "r_pong"

	errRoomNotFound = "room-not-found"
	errRoomFull     = "room-full"
	errServerBusy   = "server-busy"
	errBadRequest   = "bad-request"
)

type Rules struct {
	NoDestroy bool `json:"noDestroy,omitempty"`
	TwoDice   bool `json:"twoDice,omitempty"`
	FillBoth  bool `json:"fillBoth,omitempty"`
}

type RoomInfo struct {
	ID          string `json:"id"`
	Host        string `json:"host"`
	BestOf      int    `json:"bestOf"`
	Rules       Rules  `json:"rules"`
	TurnSeconds int    `json:"turnSeconds"`
	Locked      bool   `json:"locked"`
	CreatedAt   int64  `json:"createdAt"`
}

type envelope struct {
	Type        string     `json:"type"`
	Name        string     `json:"name,omitempty"`
	BestOf      int        `json:"bestOf,omitempty"`
	Rules       Rules      `json:"rules,omitempty"`
	TurnSeconds int        `json:"turnSeconds,omitempty"`
	Locked      bool       `json:"locked,omitempty"`
	Token       string     `json:"token,omitempty"`
	RoomID      string     `json:"roomId,omitempty"`
	ConnSeq     int        `json:"connSeq,omitempty"`
	Data        string     `json:"data,omitempty"`
	Code        string     `json:"code,omitempty"`
	Text        string     `json:"text,omitempty"`
	Rooms       []RoomInfo `json:"rooms,omitempty"`
	Stats       *StatsView `json:"stats,omitempty"`
}

// Mirror only the fields of a relayed game message needed for telemetry.
type wireGameEnvelope struct {
	Type  string        `json:"type"`
	State *wireSession  `json:"state,omitempty"`
}

type wirePlacement struct {
	Player    int `json:"player"`
	Col       int `json:"col"`
	Slot      int `json:"slot"`
	Die       int `json:"die"`
	Destroyed int `json:"destroyed"`
	Gain      int `json:"gain"`
	Lost      int `json:"lost"`
}

type wireHistoryEntry struct {
	Round  int    `json:"round"`
	Scores [2]int `json:"scores"`
	Winner int    `json:"winner"`
	Lead   int    `json:"lead"`
}

type wireForfeit struct {
	Reason string `json:"reason"`
	Loser  int    `json:"loser"`
}

type wireGrid [gridCols][gridRows]int

func (g wireGrid) score() int {
	total := 0
	for c := 0; c < gridCols; c++ {
		var counts [7]int
		for k := 0; k < gridRows; k++ {
			v := g[c][k]
			if v >= 1 && v <= 6 {
				counts[v]++
			}
		}
		for v := 1; v <= 6; v++ {
			total += counts[v] * counts[v] * v
		}
	}
	return total
}

type wireMatch struct {
	Rules Rules       `json:"rules"`
	Grids [2]wireGrid `json:"grids"`
	Turn  int         `json:"turn"`
	Moves int         `json:"moves"`
}

type wireSession struct {
	Mode      string             `json:"mode"`
	Names     [2]string          `json:"names"`
	BestOf    int                `json:"bestOf"`
	FreePlay  bool               `json:"freePlay"`
	Rules     Rules              `json:"rules"`
	Match     wireMatch          `json:"match"`
	Series    int                `json:"series"`
	Round     int                `json:"round"`
	Wins      [2]int             `json:"wins"`
	Ties      int                `json:"ties"`
	Phase     string             `json:"phase"`
	History   []wireHistoryEntry `json:"history"`
	Forfeit   *wireForfeit       `json:"forfeit,omitempty"`
	Last      *wirePlacement     `json:"last,omitempty"`
}

func readJSONLine(r *bufio.Reader, conn net.Conn, timeout time.Duration, limit int) ([]byte, error) {
	var line []byte
	for {
		if timeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(timeout))
		}
		frag, err := r.ReadSlice('\n')
		if len(line)+len(frag) > limit {
			return nil, errLineTooLong
		}
		if len(frag) > 0 {
			line = append(line, frag...)
		}
		if err == nil {
			return bytes.TrimRight(line, "\r\n"), nil
		}
		if err != bufio.ErrBufferFull {
			return bytes.TrimRight(line, "\r\n"), err
		}
	}
}

func writeRawLine(conn net.Conn, data []byte, timeout time.Duration) error {
	if len(data) == 0 || data[len(data)-1] != '\n' {
		buf := make([]byte, len(data)+1)
		copy(buf, data)
		buf[len(data)] = '\n'
		data = buf
	}
	if timeout > 0 {
		_ = conn.SetWriteDeadline(time.Now().Add(timeout))
	}
	_, err := conn.Write(data)
	return err
}

func writeEnvelope(conn net.Conn, env envelope, timeout time.Duration) error {
	b, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return writeRawLine(conn, b, timeout)
}

func isBidiControl(r rune) bool {
	return (r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069) || r == 0x200E || r == 0x200F
}

func cleanDisplay(s, fallback string, maxLen int) string {
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
	out := strings.TrimSpace(b.String())
	if out == "" {
		return fallback
	}
	return out
}

func validBestOf(b int) int {
	switch b {
	case 0, 1, 3, 5:
		return b
	}
	return 1
}

func validTurnSec(s int) int {
	switch s {
	case 0, 15, 30, 60:
		return s
	}
	return 30
}
