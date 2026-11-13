package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	mrand "math/rand/v2"
	"net"
	"strconv"
	"time"
)

const (
	protocolVersion = 2
	defaultPort     = 8647
	pingEvery       = 5 * time.Second
	readTimeout     = 15 * time.Second
	writeTimeout    = 3 * time.Second
	dialTimeout     = 5 * time.Second
	redialEvery     = 2 * time.Second
	maxWireMsgBytes = 64 * 1024
	aimMinInterval  = 50 * time.Millisecond

	msgChallenge = "challenge"
	msgHello     = "hello"
	msgState     = "state"
	msgMove      = "move"
	msgAim       = "aim"
	msgVote      = "vote"
	msgResign    = "resign"
	msgBye       = "bye"
	msgPing      = "ping"
	msgError     = "error"

	errRoomFull    = "room-full"
	errOldProtocol = "unsupported-protocol"
	errBadAuth     = "bad-room-code"
	errBadState    = "invalid-state"
	errRejected    = "connection-rejected"

	evAccepted = "accepted"
	evMsg      = "msg"
	evClosed   = "closed"
	evDialed   = "dialed"
)

// Variables so tests can shorten them.
var (
	reconnectWindow = 30 * time.Second
	turnUnit        = time.Second
)

// netMsg is one JSON line on the wire.
type netMsg struct {
	Type    string   `json:"type"`
	Version int      `json:"version,omitempty"`
	Seq     uint64   `json:"seq,omitempty"`
	MAC     string   `json:"mac,omitempty"`
	Nonce   string   `json:"nonce,omitempty"`
	AuthReq bool     `json:"authReq,omitempty"`
	Auth    string   `json:"auth,omitempty"`
	Commit  string   `json:"commit,omitempty"`
	Seed    string   `json:"seed,omitempty"`
	Name    string   `json:"name,omitempty"`
	Token   string   `json:"token,omitempty"`
	Aiming  bool     `json:"aiming,omitempty"`
	Col     int      `json:"col,omitempty"`
	Die     int      `json:"die,omitempty"`
	Vote    string   `json:"vote,omitempty"`
	Text    string   `json:"text,omitempty"`
	State   *session `json:"state,omitempty"`
}

// netEvent carries network activity into the UI goroutine, which is the
// only goroutine that touches a session.
type netEvent struct {
	kind string
	id   int
	conn net.Conn
	msg  netMsg
	err  error
}

func sendMsg(conn net.Conn, m netMsg) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	_, err = conn.Write(append(data, '\n'))
	return err
}

var errWireLineTooLong = errors.New("wire message exceeds limit")

func readLineLimited(r *bufio.Reader, conn net.Conn, limit int) ([]byte, error) {
	var line []byte
	for {
		_ = conn.SetReadDeadline(time.Now().Add(readTimeout))
		frag, err := r.ReadSlice('\n')
		if len(line)+len(frag) > limit {
			return nil, errWireLineTooLong
		}
		if len(frag) > 0 {
			line = append(line, frag...)
		}
		if err == nil {
			return line, nil
		}
		if err != bufio.ErrBufferFull {
			return line, err
		}
	}
}

func readLoop(id int, conn net.Conn, out chan<- netEvent, done <-chan struct{}) {
	r := bufio.NewReaderSize(conn, 4096)
	push := func(ev netEvent) bool {
		select {
		case out <- ev:
			return true
		case <-done:
			return false
		}
	}
	for {
		line, err := readLineLimited(r, conn, maxWireMsgBytes)
		if len(line) > 0 && err == nil {
			var m netMsg
			if json.Unmarshal(line, &m) == nil && !push(netEvent{kind: evMsg, id: id, msg: m}) {
				return
			}
		}
		if err != nil {
			push(netEvent{kind: evClosed, id: id, err: err})
			return
		}
	}
}

func safeErrorCode(code string) string {
	switch code {
	case errRoomFull, errOldProtocol, errBadAuth, errBadState, errRejected:
		return code
	}
	return errRejected
}

func newToken() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// localAddrs lists this machine's IPv4 LAN addresses for the lobby.
func localAddrs() []string {
	var out []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}
	for _, a := range addrs {
		if ipn, ok := a.(*net.IPNet); ok {
			if ip := ipn.IP.To4(); ip != nil && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
				out = append(out, ip.String())
			}
		}
	}
	return out
}

type hostConfig struct {
	BestOf      int
	Rules       Rules
	TurnSeconds int
	BindIP      string
	RoomCode    string
}

// netHost owns the session. It is seat 0; the guest is seat 1.
type netHost struct {
	name   string
	cfg    hostConfig
	rng    *mrand.Rand
	ln     net.Listener
	port   int
	events chan netEvent
	done   chan struct{}

	roomKey       []byte
	wireKey       []byte
	connNonce     string
	challengeSeed string
	authed        bool
	txSeq         uint64
	rxSeq         uint64

	s      *session
	conn   net.Conn
	connID int
	nextID int
	token  string

	turnKey      [2]int
	turnDeadline time.Time
	reconnectBy  time.Time
	lastPing     time.Time
	lastAimAt    time.Time
	aimPending   bool
	opponentLeft bool
	since        time.Time
}

func startHost(name string, port int, cfg hostConfig, rng *mrand.Rand) (*netHost, error) {
	cfg.RoomCode = cleanRoomCode(cfg.RoomCode)
	roomKey, err := deriveRoomKey(cfg.RoomCode)
	if err != nil {
		return nil, err
	}
	listenAddr := fmt.Sprintf(":%d", port)
	if bindIP := cleanText(cfg.BindIP, 64); bindIP != "" {
		listenAddr = net.JoinHostPort(bindIP, strconv.Itoa(port))
	}
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, err
	}
	h := &netHost{
		name:          cleanName(name, "Host"),
		cfg:           cfg,
		rng:           rng,
		ln:            ln,
		port:          ln.Addr().(*net.TCPAddr).Port,
		events:        make(chan netEvent, 64),
		done:          make(chan struct{}),
		roomKey:       roomKey,
		challengeSeed: newRandomHex(32),
		since:         time.Now(),
	}
	go h.acceptLoop()
	return h, nil
}

func (h *netHost) acceptLoop() {
	for {
		c, err := h.ln.Accept()
		if err != nil {
			return
		}
		go h.acceptOne(c)
	}
}

func (h *netHost) acceptOne(raw net.Conn) {
	c, err := acceptNetOrWSConn(raw)
	if err != nil {
		raw.Close()
		return
	}
	select {
	case h.events <- netEvent{kind: evAccepted, conn: c}:
	case <-h.done:
		c.Close()
	}
}

func (h *netHost) state() *session            { return h.s }
func (h *netHost) seat() int                  { return 0 }
func (h *netHost) netEvents() <-chan netEvent { return h.events }
func (h *netHost) opponentGone() bool         { return h.opponentLeft }
func (h *netHost) waitingSince() time.Time    { return h.since }
func (h *netHost) lobbyError() string         { return "" }
func (h *netHost) retryConnect()              {}
func (h *netHost) peerConnected() bool        { return h.s != nil && h.s.Connected[1] }
func (h *netHost) hostPort() int              { return h.port }
func (h *netHost) hostRoomCode() string       { return h.cfg.RoomCode }

func (h *netHost) handle(ev netEvent, now time.Time) bool {
	switch ev.kind {
	case evAccepted:
		if h.conn != nil || h.opponentLeft || (h.s != nil && h.s.Phase == phaseGameOver && h.s.Forfeit != nil) {
			rejectConn(ev.conn, errRoomFull)
			return false
		}
		connNonce := newRandomHex(16)
		wk, err := deriveWireKey(h.roomKey, connNonce)
		if err != nil {
			ev.conn.Close()
			return false
		}
		commit := commitSeed(h.challengeSeed)
		if h.s != nil && h.s.HostCommit != "" {
			commit = h.s.HostCommit
		}
		h.nextID++
		h.conn, h.connID = ev.conn, h.nextID
		h.connNonce = connNonce
		h.wireKey = wk
		h.authed = false
		h.txSeq = 0
		h.rxSeq = 0
		go readLoop(h.connID, ev.conn, h.events, h.done)
		if err := sendMsg(h.conn, netMsg{
			Type:    msgChallenge,
			Version: protocolVersion,
			Nonce:   h.connNonce,
			AuthReq: len(h.roomKey) > 0,
			Commit:  commit,
		}); err != nil {
			h.dropConn(now)
		}
		return false
	case evMsg:
		if ev.id != h.connID || h.conn == nil {
			return false
		}
		return h.onMsg(ev.msg, now)
	case evClosed:
		if ev.id != h.connID {
			return false
		}
		h.dropConn(now)
		return true
	}
	return false
}

func rejectConn(conn net.Conn, text string) {
	if conn == nil {
		return
	}
	// Finish on a goroutine so the peer can attach its reader and consume the
	// error line. Closing immediately races that attach and Windows aborts the
	// socket before room-full (or similar) is read.
	go func() {
		defer conn.Close()
		_ = sendMsg(conn, netMsg{Type: msgError, Text: text})
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		buf := make([]byte, 256)
		for {
			if _, err := conn.Read(buf); err != nil {
				return
			}
		}
	}()
}

func (h *netHost) clearConnState() {
	h.conn, h.connID = nil, 0
	h.wireKey = nil
	h.connNonce = ""
	h.authed = false
	h.txSeq = 0
	h.rxSeq = 0
}

func (h *netHost) reject(text string) {
	if h.conn != nil {
		rejectConn(h.conn, text)
	}
	h.clearConnState()
}

func (h *netHost) send(m netMsg) error {
	if h.conn == nil {
		return errors.New("not connected")
	}
	if len(h.wireKey) > 0 {
		h.txSeq++
		signWireMsg(h.wireKey, h.txSeq, &m)
	}
	return sendMsg(h.conn, m)
}

func (h *netHost) fairness() int { return fairNone }

func validToken(t string) bool {
	return len(t) >= 8 && len(t) <= 64 && t == cleanText(t, 64)
}

func (h *netHost) onMsg(m netMsg, now time.Time) bool {
	if m.Type == msgHello {
		if h.authed {
			return false
		}
		guestName := cleanName(m.Name, "Guest")
		switch {
		case m.Version != protocolVersion:
			h.reject(errOldProtocol)
			return false
		case !validToken(m.Token) || len(m.Seed) != 64 || !validHex(m.Seed, 64):
			h.reject(errRejected)
			return false
		case len(h.roomKey) > 0 && !verifyHelloAuth(h.roomKey, h.connNonce, m.Name, m.Token, m.Seed, m.Auth):
			h.reject(errBadAuth)
			return false
		case h.s == nil:
			h.token = m.Token
			names := [2]string{h.name, guestName}
			h.s = newOnlineSession(names, h.cfg.BestOf, h.cfg.Rules, h.rng, h.challengeSeed, m.Seed)
			h.s.TurnLimit = h.cfg.TurnSeconds
		case h.s.Connected[1] || m.Token != h.token:
			h.reject(errRoomFull)
			return false
		}
		h.authed = true
		h.s.Connected[1] = true
		h.reconnectBy = time.Time{}
		h.turnKey = [2]int{}
		h.lastPing = now
		h.broadcast(now)
		return true
	}
	if !h.authed || h.s == nil {
		if !h.authed {
			h.reject(errRejected)
		}
		return false
	}
	if len(h.wireKey) > 0 {
		if !verifyWireMsg(h.wireKey, h.rxSeq+1, m) {
			h.reject(errBadAuth)
			return false
		}
		h.rxSeq = m.Seq
	}
	switch m.Type {
	case msgMove:
		if h.s.place(1, Move{Col: m.Col, Die: m.Die}, h.rng) != nil {
			return false
		}
	case msgAim:
		if h.s.Match.Turn != 1 || !h.s.setAim(m.Col, m.Die) {
			return false
		}
		if now.Sub(h.lastAimAt) >= aimMinInterval {
			h.lastAimAt = now
			h.aimPending = false
			h.broadcastAim(now)
		} else {
			h.aimPending = true
		}
		return true
	case msgVote:
		if len(m.Seed) == 64 && validHex(m.Seed, 64) {
			h.s.nextGuestSeed = m.Seed
		}
		h.s.vote(1, m.Vote, h.rng)
	case msgResign:
		if h.s.Phase == phaseGameOver {
			return false
		}
		h.s.forfeitBy(1, reasonResign)
	case msgBye:
		h.s.forfeitBy(1, reasonResign)
		h.opponentLeft = true
		h.s.Connected[1] = false
		if h.conn != nil {
			h.conn.Close()
		}
		h.clearConnState()
		return true
	case msgPing:
		return false
	default:
		return false
	}
	h.broadcast(now)
	return true
}

func (h *netHost) dropConn(now time.Time) {
	if h.conn != nil {
		h.conn.Close()
	}
	h.clearConnState()
	if h.s == nil || !h.s.Connected[1] {
		return
	}
	h.s.Connected[1] = false
	if h.s.Phase == phaseGameOver {
		h.opponentLeft = true
		return
	}
	h.reconnectBy = now.Add(reconnectWindow)
}

// syncTimer starts a fresh turn clock whenever the turn changes.
func (h *netHost) syncTimer(now time.Time) {
	if h.s == nil || h.cfg.TurnSeconds <= 0 || h.s.Phase != phasePlay || !h.s.Connected[1] {
		return
	}
	key := [2]int{h.s.Series*1000 + h.s.Round, h.s.Match.Moves + 1}
	if key != h.turnKey {
		h.turnKey = key
		h.turnDeadline = now.Add(time.Duration(h.cfg.TurnSeconds) * turnUnit)
	}
}

func (h *netHost) turnLeft(now time.Time) time.Duration {
	if h.s == nil || h.cfg.TurnSeconds <= 0 || h.s.Phase != phasePlay || !h.s.Connected[1] || h.turnKey == ([2]int{}) {
		return 0
	}
	if d := h.turnDeadline.Sub(now); d > 0 {
		return d
	}
	return time.Millisecond
}

func (h *netHost) disconnectLeft(now time.Time) (int, time.Duration) {
	if h.s == nil || h.s.Connected[1] || h.reconnectBy.IsZero() || h.s.Phase == phaseGameOver {
		return -1, 0
	}
	d := h.reconnectBy.Sub(now)
	if d < 0 {
		d = 0
	}
	return 1, d
}

func (h *netHost) broadcast(now time.Time) {
	if h.s == nil {
		return
	}
	h.aimPending = false
	h.syncTimer(now)
	h.s.TurnLeftMs = h.turnLeft(now).Milliseconds()
	if h.conn == nil || !h.authed {
		return
	}
	if err := h.send(netMsg{Type: msgState, State: h.s}); err != nil {
		h.dropConn(now)
	}
}

func (h *netHost) broadcastAim(now time.Time) {
	if h.s == nil || h.conn == nil || !h.authed {
		return
	}
	if err := h.send(netMsg{Type: msgAim, Aiming: h.s.Aiming, Col: h.s.AimCol, Die: h.s.AimDie}); err != nil {
		h.dropConn(now)
	}
}

func (h *netHost) tick(now time.Time) bool {
	if h.s == nil {
		return false
	}
	changed := false
	if h.aimPending && now.Sub(h.lastAimAt) >= aimMinInterval {
		h.lastAimAt = now
		h.aimPending = false
		h.broadcastAim(now)
	}
	if h.conn != nil && h.authed && now.Sub(h.lastPing) >= pingEvery {
		h.lastPing = now
		if h.send(netMsg{Type: msgPing}) != nil {
			h.dropConn(now)
			changed = true
		}
	}
	if p, left := h.disconnectLeft(now); p >= 0 && left <= 0 {
		h.s.forfeitBy(1, reasonDisconnect)
		h.opponentLeft = true
		changed = true
	}
	h.syncTimer(now)
	if h.turnLeft(now) > 0 && !now.Before(h.turnDeadline) {
		h.s.timeout(h.rng)
		changed = true
	}
	if changed {
		h.broadcast(now)
	}
	return changed
}

func (h *netHost) place(mv Move) {
	if h.s != nil && h.s.place(0, mv, h.rng) == nil {
		h.broadcast(time.Now())
	}
}

func (h *netHost) aim(col, die int) {
	if h.s == nil || h.s.Match.Turn != 0 || !h.s.setAim(col, die) {
		return
	}
	now := time.Now()
	if now.Sub(h.lastAimAt) >= aimMinInterval {
		h.lastAimAt = now
		h.broadcastAim(now)
	} else {
		h.aimPending = true
	}
}

func (h *netHost) vote(v string) {
	if h.s == nil {
		return
	}
	h.s.vote(0, v, h.rng)
	h.broadcast(time.Now())
}

func (h *netHost) resign() {
	if h.s == nil || h.s.Phase == phaseGameOver {
		return
	}
	h.s.forfeitBy(0, reasonResign)
	h.broadcast(time.Now())
}

func (h *netHost) leave() {
	if h.s != nil && h.s.Phase != phaseGameOver {
		h.s.forfeitBy(0, reasonResign)
		h.broadcast(time.Now())
	}
	if h.conn != nil {
		if h.authed {
			_ = h.send(netMsg{Type: msgBye})
		} else {
			_ = sendMsg(h.conn, netMsg{Type: msgBye})
		}
		h.conn.Close()
		h.clearConnState()
	}
	select {
	case <-h.done:
	default:
		close(h.done)
	}
	h.ln.Close()
}

// netClient mirrors the host's session. It is seat 1.
type netClient struct {
	name     string
	addr     string
	roomCode string
	token    string
	events   chan netEvent
	done     chan struct{}

	roomKey      []byte
	wireKey      []byte
	gotChallenge bool
	handshakeOK  bool
	txSeq        uint64
	rxSeq        uint64

	conn     net.Conn
	connID   int
	nextID   int
	dialing  bool
	lastPing time.Time

	s        *session
	gotAt    time.Time
	lostAt   time.Time
	retryAt  time.Time
	hostLeft bool
	fatal    bool
	errText  string
	since    time.Time

	expectedCommit     string
	nextExpectedCommit string
	activeGuestSeed    string
	nextGuestSeed      string
	obsRoundKey        [2]int
	obsRolls           map[int][2]int
	fairStatus         int
	verifiedKey        [2]int
}

func startClient(name, addr string) *netClient {
	return startClientWithCode(name, addr, "")
}

func startClientWithCode(name, addr, roomCode string) *netClient {
	code := cleanRoomCode(roomCode)
	rk, err := deriveRoomKey(code)
	c := &netClient{
		name:     cleanName(name, "Guest"),
		addr:     addr,
		roomCode: code,
		roomKey:  rk,
		token:    newToken(),
		events:   make(chan netEvent, 64),
		done:     make(chan struct{}),
		since:    time.Now(),
		obsRolls: make(map[int][2]int),
	}
	if err != nil {
		c.errText = errBadAuth
		c.fatal = true
		return c
	}
	c.dial()
	return c
}

func (c *netClient) dial() {
	c.dialing = true
	c.nextID++
	id, addr, out, done := c.nextID, c.addr, c.events, c.done
	go func() {
		conn, err := net.DialTimeout("tcp", addr, dialTimeout)
		// Prefer done over a send so leave()/shutdown cannot lose the race
		// and rejoin a host that is waiting out a disconnect forfeit.
		select {
		case <-done:
			if conn != nil {
				conn.Close()
			}
			return
		default:
		}
		select {
		case out <- netEvent{kind: evDialed, id: id, conn: conn, err: err}:
		case <-done:
			if conn != nil {
				conn.Close()
			}
		}
	}()
}

func (c *netClient) state() *session            { return c.s }
func (c *netClient) seat() int                  { return 1 }
func (c *netClient) netEvents() <-chan netEvent { return c.events }
func (c *netClient) opponentGone() bool         { return c.hostLeft }
func (c *netClient) waitingSince() time.Time    { return c.since }
func (c *netClient) lobbyError() string         { return c.errText }
func (c *netClient) peerConnected() bool        { return c.conn != nil }
func (c *netClient) hostPort() int              { return 0 }
func (c *netClient) hostRoomCode() string       { return c.roomCode }
func (c *netClient) fairness() int              { return c.fairStatus }

func (c *netClient) retryConnect() {
	if c.s != nil || c.dialing || c.conn != nil {
		return
	}
	c.errText = ""
	c.fatal = false
	c.wireKey = nil
	c.gotChallenge = false
	c.handshakeOK = false
	c.txSeq = 0
	c.rxSeq = 0
	c.since = time.Now()
	c.dial()
}

func (c *netClient) checkFairPlay(st *session) {
	if st == nil || st.Mode != modeOnline || c.expectedCommit == "" || c.fairStatus == fairFailed {
		return
	}
	if c.obsRolls == nil {
		c.obsRolls = make(map[int][2]int)
	}
	rk := [2]int{st.Series, st.Round}
	if c.obsRoundKey != rk {
		if c.obsRoundKey == [2]int{} {
			if st.HostCommit != c.expectedCommit || st.GuestSeed != c.activeGuestSeed {
				c.fairStatus = fairFailed
				return
			}
		} else {
			if c.nextExpectedCommit == "" || c.nextGuestSeed == "" ||
				st.HostCommit != c.nextExpectedCommit || st.GuestSeed != c.nextGuestSeed {
				c.fairStatus = fairFailed
				return
			}
			c.expectedCommit = st.HostCommit
			c.activeGuestSeed = st.GuestSeed
			c.nextExpectedCommit = ""
			c.nextGuestSeed = ""
		}
		c.obsRoundKey = rk
		c.obsRolls = make(map[int][2]int)
	} else if st.HostCommit != c.expectedCommit || st.GuestSeed != c.activeGuestSeed {
		c.fairStatus = fairFailed
		return
	}

	if st.Phase == phasePlay {
		if !st.Match.Over {
			if prev, ok := c.obsRolls[st.Match.Moves]; ok {
				if prev != st.Match.Dice {
					c.fairStatus = fairFailed
					return
				}
			} else {
				c.obsRolls[st.Match.Moves] = st.Match.Dice
			}
		}
		return
	}

	if st.NextHostCommit != "" {
		if c.nextExpectedCommit == "" {
			c.nextExpectedCommit = st.NextHostCommit
		} else if st.NextHostCommit != c.nextExpectedCommit {
			c.fairStatus = fairFailed
			return
		}
	}
	if c.verifiedKey == rk {
		return
	}
	if !verifySeedCommit(st.HostReveal, c.expectedCommit) {
		c.fairStatus = fairFailed
		return
	}
	if st.RoundFirst != roundCoin(st.HostReveal, c.activeGuestSeed) || len(st.RoundRolls) == 0 {
		c.fairStatus = fairFailed
		return
	}
	for i, got := range st.RoundRolls {
		if got != roundRoll(st.HostReveal, c.activeGuestSeed, i, st.Rules.TwoDice) {
			c.fairStatus = fairFailed
			return
		}
	}
	for moveIdx, obs := range c.obsRolls {
		if moveIdx < 0 || moveIdx >= len(st.RoundRolls) || st.RoundRolls[moveIdx] != obs {
			c.fairStatus = fairFailed
			return
		}
	}
	c.fairStatus = fairVerified
	c.verifiedKey = rk
}

func (c *netClient) handle(ev netEvent, now time.Time) bool {
	switch ev.kind {
	case evDialed:
		if ev.id != c.nextID {
			if ev.conn != nil {
				ev.conn.Close()
			}
			return false
		}
		c.dialing = false
		if ev.err != nil {
			if c.s == nil && c.lostAt.IsZero() {
				c.errText = cleanText(ev.err.Error(), 120)
			}
			c.retryAt = now.Add(redialEvery)
			return true
		}
		c.conn, c.connID = ev.conn, ev.id
		c.wireKey = nil
		c.gotChallenge = false
		c.handshakeOK = false
		c.txSeq = 0
		c.rxSeq = 0
		go readLoop(c.connID, ev.conn, c.events, c.done)
		return true
	case evMsg:
		if ev.id != c.connID || c.conn == nil {
			return false
		}
		switch ev.msg.Type {
		case msgChallenge:
			if c.gotChallenge {
				return false
			}
			if ev.msg.Version != protocolVersion {
				c.errText = errOldProtocol
				c.fatal = true
				c.closeConn()
				return true
			}
			if ev.msg.AuthReq != (len(c.roomKey) > 0) {
				c.errText = errBadAuth
				c.fatal = true
				c.closeConn()
				return true
			}
			if ev.msg.Nonce == "" || !validHex(ev.msg.Nonce, 64) || len(ev.msg.Commit) != 64 || !validHex(ev.msg.Commit, 64) {
				c.errText = errBadState
				c.fatal = true
				c.closeConn()
				return true
			}
			wk, err := deriveWireKey(c.roomKey, ev.msg.Nonce)
			if err != nil {
				c.errText = errBadAuth
				c.fatal = true
				c.closeConn()
				return true
			}
			c.wireKey = wk
			c.gotChallenge = true
			if c.expectedCommit == "" {
				c.expectedCommit = ev.msg.Commit
			} else if ev.msg.Commit != c.expectedCommit && (c.nextExpectedCommit == "" || ev.msg.Commit != c.nextExpectedCommit) {
				c.fairStatus = fairFailed
			}
			if c.activeGuestSeed == "" {
				c.activeGuestSeed = newRandomHex(32)
			}
			auth := helloAuthMAC(c.roomKey, ev.msg.Nonce, c.name, c.token, c.activeGuestSeed)
			c.lastPing = now
			if sendMsg(c.conn, netMsg{
				Type:    msgHello,
				Version: protocolVersion,
				Name:    c.name,
				Token:   c.token,
				Seed:    c.activeGuestSeed,
				Auth:    auth,
			}) != nil {
				c.closeConn()
			}
			return true
		case msgError:
			if c.handshakeOK && len(c.wireKey) > 0 && !verifyWireMsg(c.wireKey, c.rxSeq+1, ev.msg) {
				return false
			}
			c.errText = safeErrorCode(ev.msg.Text)
			c.fatal = true
			c.closeConn()
			return true
		}

		if len(c.roomKey) > 0 && !c.gotChallenge {
			c.errText = errBadAuth
			c.fatal = true
			c.closeConn()
			return true
		}
		if len(c.wireKey) > 0 {
			if !verifyWireMsg(c.wireKey, c.rxSeq+1, ev.msg) {
				c.errText = errBadAuth
				c.fatal = true
				c.closeConn()
				return true
			}
			c.rxSeq = ev.msg.Seq
		}

		switch ev.msg.Type {
		case msgState:
			if ev.msg.State != nil {
				if ev.msg.State.validate() != nil {
					c.errText = errBadState
					c.fatal = true
					c.closeConn()
					return true
				}
				c.handshakeOK = true
				c.checkFairPlay(ev.msg.State)
				c.s = ev.msg.State
				c.gotAt = now
				c.lostAt = time.Time{}
				c.errText = ""
			}
		case msgAim:
			if c.s != nil && c.s.Phase == phasePlay {
				if ev.msg.Aiming {
					if ev.msg.Col >= 0 && ev.msg.Col < gridCols && ev.msg.Die >= 0 && ev.msg.Die <= 1 && c.s.Match.Dice[ev.msg.Die] != 0 {
						c.s.Aiming = true
						c.s.AimCol = ev.msg.Col
						c.s.AimDie = ev.msg.Die
					}
				} else {
					c.s.clearAim()
				}
			}
		case msgBye:
			c.hostLeft = true
			c.closeConn()
		case msgPing:
			return false
		default:
			return false
		}
		return true
	case evClosed:
		if ev.id != c.connID {
			return false
		}
		c.closeConn()
		switch {
		case c.fatal:
		case c.s == nil:
			if c.errText == "" {
				c.errText = "the host closed the connection"
			}
		case c.s.Phase != phaseGameOver && !c.hostLeft:
			c.lostAt = now
			c.retryAt = now
		default:
			c.hostLeft = true
		}
		return true
	}
	return false
}

func (c *netClient) closeConn() {
	if c.conn != nil {
		c.conn.Close()
	}
	c.conn, c.connID = nil, 0
	c.wireKey = nil
	c.gotChallenge = false
	c.handshakeOK = false
	c.txSeq = 0
	c.rxSeq = 0
}

func (c *netClient) tick(now time.Time) bool {
	if c.conn != nil && c.gotChallenge && now.Sub(c.lastPing) >= pingEvery {
		c.lastPing = now
		c.send(netMsg{Type: msgPing})
	}
	if c.lostAt.IsZero() {
		return false
	}
	if now.Sub(c.lostAt) >= reconnectWindow {
		c.lostAt = time.Time{}
		c.hostLeft = true
		if c.s != nil {
			c.s.Connected[0] = false
			c.s.forfeitBy(0, reasonDisconnect)
		}
		return true
	}
	if !c.dialing && c.conn == nil && !now.Before(c.retryAt) {
		c.dial()
	}
	return false
}

func (c *netClient) turnLeft(now time.Time) time.Duration {
	if c.s == nil || c.s.TurnLeftMs <= 0 || c.s.Phase != phasePlay || !c.lostAt.IsZero() {
		return 0
	}
	d := time.Duration(c.s.TurnLeftMs)*time.Millisecond - now.Sub(c.gotAt)
	if d <= 0 {
		return time.Millisecond
	}
	return d
}

func (c *netClient) disconnectLeft(now time.Time) (int, time.Duration) {
	if c.lostAt.IsZero() {
		return -1, 0
	}
	d := reconnectWindow - now.Sub(c.lostAt)
	if d < 0 {
		d = 0
	}
	return 0, d
}

func (c *netClient) send(m netMsg) {
	if c.conn == nil || (len(c.roomKey) > 0 && !c.gotChallenge) {
		return
	}
	if len(c.wireKey) > 0 {
		c.txSeq++
		signWireMsg(c.wireKey, c.txSeq, &m)
	}
	if sendMsg(c.conn, m) != nil {
		c.closeConn()
	}
}

func (c *netClient) place(mv Move) { c.send(netMsg{Type: msgMove, Col: mv.Col, Die: mv.Die}) }

func (c *netClient) aim(col, die int) { c.send(netMsg{Type: msgAim, Col: col, Die: die}) }

func (c *netClient) vote(v string) {
	if c.s != nil {
		c.s.Votes[1] = v
	}
	if c.nextGuestSeed == "" {
		c.nextGuestSeed = newRandomHex(32)
	}
	c.send(netMsg{Type: msgVote, Vote: v, Seed: c.nextGuestSeed})
}

func (c *netClient) resign() { c.send(netMsg{Type: msgResign}) }

func (c *netClient) leave() {
	c.send(netMsg{Type: msgBye})
	c.closeConn()
	select {
	case <-c.done:
	default:
		close(c.done)
	}
}
