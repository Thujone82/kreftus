package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	mrand "math/rand/v2"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultRendezvousAddr = "kb.kreft.us:8647"
	errRoomNotFound       = "room-not-found"
	rendezvousPingEvery   = 18 * time.Second
	rendezvousReadWait    = 50 * time.Second
	rendezvousWriteWait   = 5 * time.Second
	rendezvousMaxLine     = 64 * 1024
)

type fatalDialError struct {
	code string
}

func (e *fatalDialError) Error() string {
	return e.code
}

type lobbyStatsView struct {
	LobbyWatchers      int    `json:"lobbyWatchers"`
	LiveMatches        int    `json:"liveMatches"`
	WaitingRooms       int    `json:"waitingRooms"`
	CompletedMatches   uint64 `json:"completedMatches"`
	CompletedRounds    uint64 `json:"completedRounds"`
	TotalDicePlayed    uint64 `json:"totalDicePlayed"`
	TotalDiceDestroyed uint64 `json:"totalDiceDestroyed"`
	TotalPointsScored  uint64 `json:"totalPointsScored"`
	UpdatedAt          int64  `json:"updatedAt"`
}

type lobbyRoomInfo struct {
	ID          string `json:"id,omitempty"`
	RoomID      string `json:"roomId,omitempty"`
	Host        string `json:"host"`
	BestOf      int    `json:"bestOf"`
	Rules       Rules  `json:"rules"`
	TurnSeconds int    `json:"turnSeconds"`
	Locked      bool   `json:"locked"`
	CreatedAt   int64  `json:"createdAt"`
}

type lobbySnapshot struct {
	Rooms      []lobbyRoomInfo
	Stats      lobbyStatsView
	Connected  bool
	Connecting bool
	Err        string
	UpdatedAt  time.Time
}

type rendezvousEnvelope struct {
	Type        string          `json:"type"`
	RoomID      string          `json:"roomId,omitempty"`
	Token       string          `json:"token,omitempty"`
	Name        string          `json:"name,omitempty"`
	BestOf      int             `json:"bestOf,omitempty"`
	Rules       Rules           `json:"rules,omitempty"`
	TurnSeconds int             `json:"turnSeconds,omitempty"`
	Locked      bool            `json:"locked,omitempty"`
	ConnSeq     int             `json:"connSeq,omitempty"`
	Data        string          `json:"data,omitempty"`
	Code        string          `json:"code,omitempty"`
	Text        string          `json:"text,omitempty"`
	Rooms       []lobbyRoomInfo `json:"rooms,omitempty"`
	Stats       *lobbyStatsView `json:"stats,omitempty"`
}

func dialRendezvousConn(addr string, tlsCfg *tls.Config, plainTCP bool) (net.Conn, error) {
	if addr == "" {
		addr = defaultRendezvousAddr
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		addr = net.JoinHostPort(addr, fmt.Sprintf("%d", defaultPort))
		host, _, _ = net.SplitHostPort(addr)
	}
	d := &net.Dialer{Timeout: dialTimeout}
	raw, err := d.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	if plainTCP {
		return raw, nil
	}
	var cfg *tls.Config
	if tlsCfg != nil {
		cfg = tlsCfg.Clone()
		if cfg.ServerName == "" {
			cfg.ServerName = host
		}
	} else {
		cfg = &tls.Config{
			ServerName: host,
			MinVersion: tls.VersionTLS12,
		}
	}
	tc := tls.Client(raw, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	if err := tc.HandshakeContext(ctx); err != nil {
		_ = raw.Close()
		return nil, err
	}
	return tc, nil
}

func writeRendezvousEnv(conn net.Conn, env rendezvousEnvelope) error {
	b, err := json.Marshal(env)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_ = conn.SetWriteDeadline(time.Now().Add(rendezvousWriteWait))
	_, err = conn.Write(b)
	_ = conn.SetWriteDeadline(time.Time{})
	return err
}

func readRendezvousLine(br *bufio.Reader, conn net.Conn, timeout time.Duration) ([]byte, error) {
	if timeout > 0 {
		_ = conn.SetReadDeadline(time.Now().Add(timeout))
	} else {
		_ = conn.SetReadDeadline(time.Time{})
	}
	line, err := br.ReadBytes('\n')
	if err != nil && len(line) == 0 {
		return nil, err
	}
	if len(line) > rendezvousMaxLine {
		return nil, errors.New("server message too large")
	}
	return bytes.TrimSpace(line), nil
}

// bufferedConn ensures any bytes prefetched into bufio.Reader during the
// handshake are returned before reading directly from the underlying conn.
type bufferedConn struct {
	net.Conn
	br *bufio.Reader
}

func (b *bufferedConn) Read(p []byte) (int, error) {
	if b.br.Buffered() > 0 {
		return b.br.Read(p)
	}
	return b.Conn.Read(p)
}

// rWatcher maintains a background watch connection to the rendezvous server
// for live room listings and global stats.
type rWatcher struct {
	addr     string
	tlsCfg   *tls.Config
	plainTCP bool

	mu   sync.Mutex
	snap lobbySnapshot

	notifyCh chan struct{}
	retryCh  chan struct{}
	done     chan struct{}
	once     sync.Once
}

func startRendezvousWatcher(addr string, tlsCfg *tls.Config, plainTCP bool) *rWatcher {
	if addr == "" {
		addr = defaultRendezvousAddr
	}
	w := &rWatcher{
		addr:     addr,
		tlsCfg:   tlsCfg,
		plainTCP: plainTCP,
		snap:     lobbySnapshot{Connecting: true},
		notifyCh: make(chan struct{}, 1),
		retryCh:  make(chan struct{}, 1),
		done:     make(chan struct{}),
	}
	go w.loop()
	return w
}

func (w *rWatcher) Close() {
	w.once.Do(func() { close(w.done) })
}

func (w *rWatcher) Notify() <-chan struct{} {
	return w.notifyCh
}

func (w *rWatcher) Reconnect() {
	select {
	case w.retryCh <- struct{}{}:
	default:
	}
}

func (w *rWatcher) Snapshot() lobbySnapshot {
	w.mu.Lock()
	defer w.mu.Unlock()
	cp := w.snap
	if len(w.snap.Rooms) > 0 {
		cp.Rooms = make([]lobbyRoomInfo, len(w.snap.Rooms))
		copy(cp.Rooms, w.snap.Rooms)
	}
	return cp
}

func (w *rWatcher) signal() {
	select {
	case w.notifyCh <- struct{}{}:
	default:
	}
}

func (w *rWatcher) loop() {
	for {
		select {
		case <-w.done:
			return
		default:
		}
		w.mu.Lock()
		w.snap.Connecting = true
		w.mu.Unlock()
		w.signal()

		err := w.runOne()
		select {
		case <-w.done:
			return
		default:
		}
		w.mu.Lock()
		w.snap.Connected = false
		w.snap.Connecting = false
		if err != nil {
			w.snap.Err = cleanText(err.Error(), 96)
		}
		w.mu.Unlock()
		w.signal()

		t := time.NewTimer(3 * time.Second)
		select {
		case <-w.done:
			t.Stop()
			return
		case <-w.retryCh:
			t.Stop()
		case <-t.C:
		}
	}
}

func (w *rWatcher) runOne() error {
	conn, err := dialRendezvousConn(w.addr, w.tlsCfg, w.plainTCP)
	if err != nil {
		return err
	}
	defer conn.Close()

	var wmu sync.Mutex
	sendEnv := func(env rendezvousEnvelope) error {
		wmu.Lock()
		defer wmu.Unlock()
		return writeRendezvousEnv(conn, env)
	}
	if err := sendEnv(rendezvousEnvelope{Type: "r_watch"}); err != nil {
		return err
	}

	stopPing := make(chan struct{})
	go func() {
		defer conn.Close()
		_ = sendEnv(rendezvousEnvelope{Type: "ping"})
		ticker := time.NewTicker(rendezvousPingEvery)
		defer ticker.Stop()
		for {
			select {
			case <-w.done:
				return
			case <-stopPing:
				return
			case <-ticker.C:
				if sendEnv(rendezvousEnvelope{Type: "ping"}) != nil {
					return
				}
			}
		}
	}()
	defer close(stopPing)

	br := bufio.NewReaderSize(conn, 8192)
	for {
		line, err := readRendezvousLine(br, conn, rendezvousReadWait)
		if err != nil {
			return err
		}
		if len(line) == 0 {
			continue
		}
		var env rendezvousEnvelope
		if json.Unmarshal(line, &env) != nil {
			continue
		}
		switch env.Type {
		case "r_lobby":
			for i := range env.Rooms {
				if env.Rooms[i].RoomID == "" {
					env.Rooms[i].RoomID = env.Rooms[i].ID
				}
			}
			w.mu.Lock()
			w.snap.Connected = true
			w.snap.Connecting = false
			w.snap.Err = ""
			w.snap.Rooms = env.Rooms
			if env.Stats != nil {
				w.snap.Stats = *env.Stats
			}
			w.snap.UpdatedAt = time.Now()
			w.mu.Unlock()
			w.signal()
		case "r_error":
			msg := env.Text
			if msg == "" {
				msg = env.Code
			}
			return errors.New(msg)
		}
	}
}

type rendezvousAddr struct {
	network string
	text    string
}

func (a rendezvousAddr) Network() string { return a.network }
func (a rendezvousAddr) String() string  { return a.text }

// rHostListener bridges netHost to a rendezvous server room so all existing
// netHost session, HMAC, commit-reveal, and timer code works unchanged.
type rHostListener struct {
	conn   net.Conn
	addr   string
	roomID string

	wmu      sync.Mutex
	mu       sync.Mutex
	conns    map[int]*rHostConn
	acceptCh chan net.Conn
	done     chan struct{}
	once     sync.Once
}

func startOnlineHost(name, addr string, cfg hostConfig, rng *mrand.Rand, tlsCfg *tls.Config, plainTCP bool) (*netHost, error) {
	if addr == "" {
		addr = defaultRendezvousAddr
	}
	cfg.RoomCode = cleanRoomCode(cfg.RoomCode)
	conn, err := dialRendezvousConn(addr, tlsCfg, plainTCP)
	if err != nil {
		return nil, err
	}
	hostName := cleanName(name, "Host")
	hostToken := newToken()
	if err := writeRendezvousEnv(conn, rendezvousEnvelope{
		Type:        "r_host",
		Name:        hostName,
		BestOf:      cfg.BestOf,
		Rules:       cfg.Rules,
		TurnSeconds: cfg.TurnSeconds,
		Locked:      cfg.RoomCode != "",
		Token:       hostToken,
	}); err != nil {
		_ = conn.Close()
		return nil, err
	}
	br := bufio.NewReaderSize(conn, 8192)
	line, err := readRendezvousLine(br, conn, dialTimeout)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	var env rendezvousEnvelope
	if err := json.Unmarshal(line, &env); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if env.Type == "r_error" {
		_ = conn.Close()
		if env.Text != "" {
			return nil, errors.New(env.Text)
		}
		return nil, errors.New(env.Code)
	}
	if env.Type != "r_hosted" || env.RoomID == "" {
		_ = conn.Close()
		return nil, errors.New("unexpected rendezvous response")
	}

	ln := &rHostListener{
		conn:     conn,
		addr:     addr,
		roomID:   env.RoomID,
		conns:    make(map[int]*rHostConn),
		acceptCh: make(chan net.Conn, 8),
		done:     make(chan struct{}),
	}
	go ln.readLoop(br)
	go ln.pingLoop()

	h, err := startHostWithListener(hostName, ln, cfg, rng)
	if err != nil {
		_ = ln.Close()
		return nil, err
	}
	h.onlineRoomID = env.RoomID
	h.onlineAddr = addr
	return h, nil
}

func (l *rHostListener) writeEnv(env rendezvousEnvelope) error {
	l.wmu.Lock()
	defer l.wmu.Unlock()
	return writeRendezvousEnv(l.conn, env)
}

func (l *rHostListener) pingLoop() {
	ticker := time.NewTicker(rendezvousPingEvery)
	defer ticker.Stop()
	for {
		select {
		case <-l.done:
			return
		case <-ticker.C:
			if l.writeEnv(rendezvousEnvelope{Type: "ping"}) != nil {
				_ = l.Close()
				return
			}
		}
	}
}

func (l *rHostListener) readLoop(br *bufio.Reader) {
	defer l.Close()
	for {
		line, err := readRendezvousLine(br, l.conn, rendezvousReadWait)
		if err != nil {
			return
		}
		if len(line) == 0 {
			continue
		}
		var env rendezvousEnvelope
		if json.Unmarshal(line, &env) != nil {
			continue
		}
		switch env.Type {
		case "r_guest_joined":
			c := newRHostConn(l, env.ConnSeq)
			l.mu.Lock()
			l.conns[env.ConnSeq] = c
			l.mu.Unlock()
			select {
			case l.acceptCh <- c:
			case <-l.done:
				c.closeLocal()
				return
			}
		case "r_relay":
			l.mu.Lock()
			c := l.conns[env.ConnSeq]
			l.mu.Unlock()
			if c != nil {
				c.pushLine(env.Data)
			}
		case "r_guest_drop":
			l.mu.Lock()
			c := l.conns[env.ConnSeq]
			delete(l.conns, env.ConnSeq)
			l.mu.Unlock()
			if c != nil {
				c.closeLocal()
			}
		case "r_error":
			return
		}
	}
}

func (l *rHostListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.acceptCh:
		return c, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *rHostListener) Close() error {
	l.once.Do(func() {
		close(l.done)
		_ = l.conn.Close()
		l.mu.Lock()
		for seq, c := range l.conns {
			delete(l.conns, seq)
			c.closeLocal()
		}
		l.mu.Unlock()
	})
	return nil
}

func (l *rHostListener) Addr() net.Addr {
	return rendezvousAddr{network: "rendezvous", text: l.addr + "/" + l.roomID}
}

type rHostConn struct {
	ln   *rHostListener
	seq  int
	cond *sync.Cond

	mu         sync.Mutex
	rx         bytes.Buffer
	tx         bytes.Buffer
	closed     bool
	readDead   time.Time
	deadlineTm *time.Timer
}

func newRHostConn(ln *rHostListener, seq int) *rHostConn {
	c := &rHostConn{ln: ln, seq: seq}
	c.cond = sync.NewCond(&c.mu)
	return c
}

func (c *rHostConn) pushLine(data string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.rx.WriteString(data)
	c.rx.WriteByte('\n')
	c.cond.Broadcast()
}

func (c *rHostConn) closeLocal() {
	c.mu.Lock()
	if !c.closed {
		c.closed = true
		if c.deadlineTm != nil {
			c.deadlineTm.Stop()
		}
		c.cond.Broadcast()
	}
	c.mu.Unlock()
}

func (c *rHostConn) Read(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for c.rx.Len() == 0 {
		if c.closed {
			return 0, io.EOF
		}
		if !c.readDead.IsZero() && !time.Now().Before(c.readDead) {
			return 0, os.ErrDeadlineExceeded
		}
		c.cond.Wait()
	}
	return c.rx.Read(p)
}

func (c *rHostConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return 0, net.ErrClosed
	}
	c.tx.Write(p)
	var lines []string
	for {
		raw := c.tx.Bytes()
		idx := bytes.IndexByte(raw, '\n')
		if idx < 0 {
			break
		}
		line := strings.TrimSpace(string(raw[:idx]))
		c.tx.Next(idx + 1)
		if line != "" {
			lines = append(lines, line)
		}
	}
	c.mu.Unlock()

	for _, line := range lines {
		if err := c.ln.writeEnv(rendezvousEnvelope{
			Type:    "r_relay",
			ConnSeq: c.seq,
			Data:    line,
		}); err != nil {
			c.closeLocal()
			return 0, err
		}
	}
	return len(p), nil
}

func (c *rHostConn) Close() error {
	c.ln.mu.Lock()
	_, active := c.ln.conns[c.seq]
	if active {
		delete(c.ln.conns, c.seq)
	}
	c.ln.mu.Unlock()
	c.closeLocal()
	if active {
		go func() {
			_ = c.ln.writeEnv(rendezvousEnvelope{Type: "r_guest_drop", ConnSeq: c.seq})
		}()
	}
	return nil
}

func (c *rHostConn) LocalAddr() net.Addr  { return rendezvousAddr{"rendezvous", "host"} }
func (c *rHostConn) RemoteAddr() net.Addr { return rendezvousAddr{"rendezvous", fmt.Sprintf("guest#%d", c.seq)} }

func (c *rHostConn) SetDeadline(t time.Time) error {
	return c.SetReadDeadline(t)
}

func (c *rHostConn) SetReadDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readDead = t
	if c.deadlineTm != nil {
		c.deadlineTm.Stop()
		c.deadlineTm = nil
	}
	if !t.IsZero() {
		d := time.Until(t)
		if d <= 0 {
			c.cond.Broadcast()
		} else {
			c.deadlineTm = time.AfterFunc(d, func() {
				c.mu.Lock()
				c.cond.Broadcast()
				c.mu.Unlock()
			})
		}
	}
	return nil
}

func (c *rHostConn) SetWriteDeadline(time.Time) error { return nil }

func startOnlineClient(name, addr, roomID, roomCode string, tlsCfg *tls.Config, plainTCP bool) *netClient {
	if addr == "" {
		addr = defaultRendezvousAddr
	}
	guestName := cleanName(name, "Guest")
	guestTok := newToken()
	c := startClientWithDialer(guestName, addr, roomCode, func(target string) (net.Conn, error) {
		conn, err := dialRendezvousConn(target, tlsCfg, plainTCP)
		if err != nil {
			return nil, err
		}
		if err := writeRendezvousEnv(conn, rendezvousEnvelope{
			Type:   "r_join",
			RoomID: roomID,
			Token:  guestTok,
			Name:   guestName,
		}); err != nil {
			_ = conn.Close()
			return nil, err
		}
		br := bufio.NewReaderSize(conn, 8192)
		line, err := readRendezvousLine(br, conn, dialTimeout)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		var env rendezvousEnvelope
		if err := json.Unmarshal(line, &env); err != nil {
			_ = conn.Close()
			return nil, err
		}
		if env.Type == "r_error" {
			_ = conn.Close()
			code := env.Code
			if code == "" {
				code = env.Text
			}
			return nil, &fatalDialError{code: code}
		}
		if env.Type != "r_paired" {
			_ = conn.Close()
			return nil, errors.New("unexpected rendezvous response")
		}
		return &bufferedConn{Conn: conn, br: br}, nil
	})
	return c
}
