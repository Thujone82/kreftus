package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	mrand "math/rand/v2"
	"net"
	"time"
)

const (
	protocolVersion = 1
	defaultPort     = 7777
	pingEvery       = 5 * time.Second
	readTimeout     = 15 * time.Second
	writeTimeout    = 3 * time.Second
	dialTimeout     = 5 * time.Second
	redialEvery     = 2 * time.Second

	msgHello  = "hello"
	msgState  = "state"
	msgMove   = "move"
	msgAim    = "aim"
	msgVote   = "vote"
	msgResign = "resign"
	msgBye    = "bye"
	msgPing   = "ping"
	msgError  = "error"

	errRoomFull    = "room-full"
	errOldProtocol = "unsupported-protocol"

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
	Name    string   `json:"name,omitempty"`
	Token   string   `json:"token,omitempty"`
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

func readLoop(id int, conn net.Conn, out chan<- netEvent, done <-chan struct{}) {
	r := bufio.NewReader(conn)
	push := func(ev netEvent) bool {
		select {
		case out <- ev:
			return true
		case <-done:
			return false
		}
	}
	for {
		_ = conn.SetReadDeadline(time.Now().Add(readTimeout))
		line, err := r.ReadBytes('\n')
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

	s      *session
	conn   net.Conn
	connID int
	nextID int
	token  string

	turnKey      [2]int
	turnDeadline time.Time
	reconnectBy  time.Time
	lastPing     time.Time
	opponentLeft bool
	since        time.Time
}

func startHost(name string, port int, cfg hostConfig, rng *mrand.Rand) (*netHost, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	h := &netHost{
		name:   name,
		cfg:    cfg,
		rng:    rng,
		ln:     ln,
		port:   ln.Addr().(*net.TCPAddr).Port,
		events: make(chan netEvent, 64),
		done:   make(chan struct{}),
		since:  time.Now(),
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
		select {
		case h.events <- netEvent{kind: evAccepted, conn: c}:
		case <-h.done:
			c.Close()
			return
		}
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

func (h *netHost) handle(ev netEvent, now time.Time) bool {
	switch ev.kind {
	case evAccepted:
		if h.conn != nil || h.opponentLeft || (h.s != nil && h.s.Phase == phaseGameOver && h.s.Forfeit != nil) {
			_ = sendMsg(ev.conn, netMsg{Type: msgError, Text: errRoomFull})
			ev.conn.Close()
			return false
		}
		h.nextID++
		h.conn, h.connID = ev.conn, h.nextID
		go readLoop(h.connID, ev.conn, h.events, h.done)
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

func (h *netHost) reject(text string) {
	if h.conn != nil {
		_ = sendMsg(h.conn, netMsg{Type: msgError, Text: text})
		h.conn.Close()
	}
	h.conn, h.connID = nil, 0
}

func (h *netHost) onMsg(m netMsg, now time.Time) bool {
	if m.Type == msgHello {
		switch {
		case m.Version != protocolVersion:
			h.reject(errOldProtocol)
			return false
		case h.s == nil:
			h.token = m.Token
			names := [2]string{h.name, cleanName(m.Name, "Guest")}
			h.s = newSession(modeOnline, names, "", h.cfg.BestOf, h.cfg.Rules, h.rng)
			h.s.TurnLimit = h.cfg.TurnSeconds
		case h.s.Connected[1] || m.Token != h.token:
			h.reject(errRoomFull)
			return false
		}
		h.s.Connected[1] = true
		h.reconnectBy = time.Time{}
		h.turnKey = [2]int{}
		h.broadcast(now)
		return true
	}
	if h.s == nil {
		return false
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
	case msgVote:
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
		h.conn.Close()
		h.conn, h.connID = nil, 0
		return true
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
	h.conn, h.connID = nil, 0
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
	h.syncTimer(now)
	h.s.TurnLeftMs = h.turnLeft(now).Milliseconds()
	if h.conn == nil {
		return
	}
	if err := sendMsg(h.conn, netMsg{Type: msgState, State: h.s}); err != nil {
		h.dropConn(now)
	}
}

func (h *netHost) tick(now time.Time) bool {
	if h.s == nil {
		return false
	}
	changed := false
	if h.conn != nil && now.Sub(h.lastPing) >= pingEvery {
		h.lastPing = now
		if sendMsg(h.conn, netMsg{Type: msgPing}) != nil {
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
	h.broadcast(time.Now())
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
		_ = sendMsg(h.conn, netMsg{Type: msgBye})
		h.conn.Close()
		h.conn = nil
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
	name   string
	addr   string
	token  string
	events chan netEvent
	done   chan struct{}

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
}

func startClient(name, addr string) *netClient {
	c := &netClient{
		name:   name,
		addr:   addr,
		token:  newToken(),
		events: make(chan netEvent, 64),
		done:   make(chan struct{}),
		since:  time.Now(),
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

func (c *netClient) retryConnect() {
	if c.s != nil || c.dialing || c.conn != nil {
		return
	}
	c.errText = ""
	c.fatal = false
	c.since = time.Now()
	c.dial()
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
				c.errText = ev.err.Error()
			}
			c.retryAt = now.Add(redialEvery)
			return true
		}
		c.conn, c.connID = ev.conn, ev.id
		go readLoop(c.connID, ev.conn, c.events, c.done)
		if sendMsg(c.conn, netMsg{Type: msgHello, Version: protocolVersion, Name: c.name, Token: c.token}) != nil {
			c.closeConn()
		}
		return true
	case evMsg:
		if ev.id != c.connID || c.conn == nil {
			return false
		}
		switch ev.msg.Type {
		case msgState:
			if ev.msg.State != nil {
				c.s = ev.msg.State
				c.gotAt = now
				c.lostAt = time.Time{}
				c.errText = ""
			}
		case msgError:
			c.errText = ev.msg.Text
			c.fatal = true
			c.closeConn()
		case msgBye:
			c.hostLeft = true
			c.closeConn()
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
}

func (c *netClient) tick(now time.Time) bool {
	if c.conn != nil && now.Sub(c.lastPing) >= pingEvery {
		c.lastPing = now
		if sendMsg(c.conn, netMsg{Type: msgPing}) != nil {
			c.closeConn()
		}
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
	if c.conn != nil && sendMsg(c.conn, m) != nil {
		c.closeConn()
	}
}

func (c *netClient) place(mv Move) { c.send(netMsg{Type: msgMove, Col: mv.Col, Die: mv.Die}) }

func (c *netClient) aim(col, die int) { c.send(netMsg{Type: msgAim, Col: col, Die: die}) }

func (c *netClient) vote(v string) {
	if c.s != nil {
		c.s.Votes[1] = v
	}
	c.send(netMsg{Type: msgVote, Vote: v})
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
