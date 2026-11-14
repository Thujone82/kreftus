package main

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	idleControlTimeout = 60 * time.Second
	relayReadTimeout   = 25 * time.Second
	writeTimeout       = 5 * time.Second
	maxConns           = 512
	maxRooms           = 200
)

type watcher struct {
	id   uint64
	conn net.Conn
	wmu  sync.Mutex
}

type room struct {
	id         string
	hostConn   net.Conn
	hostWmu    sync.Mutex
	hostToken  string
	info       RoomInfo
	guestConn  net.Conn
	guestToken string
	connSeq    int
	live       bool
	progress   roomProgress
	closed     atomic.Bool
}

type hub struct {
	mu           sync.Mutex
	rooms        map[string]*room
	watchers     map[uint64]*watcher
	nextWatchID  uint64
	stats        *statsStore
	lastStatPush time.Time
	activeConns  int
}

func newHub(statsPath string) *hub {
	return &hub{
		rooms:    make(map[string]*room),
		watchers: make(map[uint64]*watcher),
		stats:    newStatsStore(statsPath),
	}
}

func randomRoomID() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func (r *room) writeHost(env envelope) error {
	b, err := json.Marshal(env)
	if err != nil {
		return err
	}
	r.hostWmu.Lock()
	defer r.hostWmu.Unlock()
	if r.closed.Load() {
		return net.ErrClosed
	}
	return writeRawLine(r.hostConn, b, writeTimeout)
}

func (h *hub) countsLocked() (live int, waiting []RoomInfo) {
	for _, r := range h.rooms {
		if r.closed.Load() {
			continue
		}
		if r.live {
			if !r.progress.ended {
				live++
			}
			continue
		}
		if r.guestConn == nil {
			waiting = append(waiting, r.info)
		}
	}
	sort.Slice(waiting, func(i, j int) bool {
		if waiting[i].CreatedAt != waiting[j].CreatedAt {
			return waiting[i].CreatedAt > waiting[j].CreatedAt
		}
		return waiting[i].ID < waiting[j].ID
	})
	return live, waiting
}

func (h *hub) liveMatchesLocked() []LiveMatchInfo {
	var out []LiveMatchInfo
	for _, r := range h.rooms {
		if r.closed.Load() || !r.live || r.progress.ended {
			continue
		}
		host := r.progress.liveNames[0]
		if host == "" {
			host = r.info.Host
		}
		guest := r.progress.liveNames[1]
		if guest == "" {
			guest = "Guest"
		}
		out = append(out, LiveMatchInfo{
			ID:     r.id,
			Host:   host,
			Guest:  guest,
			Round:  r.progress.liveRound,
			BestOf: r.info.BestOf,
			Wins:   r.progress.liveWins,
			Scores: r.progress.liveScores,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func (h *hub) snapshotLocked() ([]RoomInfo, StatsView) {
	live, waiting := h.countsLocked()
	sv := h.stats.Snapshot(live, len(waiting), len(h.watchers))
	return waiting, sv
}

func (h *hub) broadcastLobby(force bool) {
	h.broadcastLobbyExcept(force, 0)
}

func (h *hub) broadcastLobbyExcept(force bool, exceptID uint64) {
	h.mu.Lock()
	if !force && time.Since(h.lastStatPush) < 400*time.Millisecond {
		h.mu.Unlock()
		return
	}
	h.lastStatPush = time.Now()
	rooms, sv := h.snapshotLocked()
	targets := make([]*watcher, 0, len(h.watchers))
	for _, w := range h.watchers {
		if exceptID != 0 && w.id == exceptID {
			continue
		}
		targets = append(targets, w)
	}
	h.mu.Unlock()

	if len(targets) == 0 {
		return
	}
	b, err := json.Marshal(envelope{
		Type:  rMsgLobby,
		Rooms: rooms,
		Stats: &sv,
	})
	if err != nil {
		return
	}
	for _, w := range targets {
		go func(wt *watcher) {
			wt.wmu.Lock()
			err := writeRawLine(wt.conn, b, writeTimeout)
			wt.wmu.Unlock()
			if err != nil {
				_ = wt.conn.Close()
			}
		}(w)
	}
}

func (h *hub) addWatcher(conn net.Conn) *watcher {
	h.mu.Lock()
	h.nextWatchID++
	w := &watcher{id: h.nextWatchID, conn: conn}
	h.watchers[w.id] = w
	rooms, sv := h.snapshotLocked()
	h.mu.Unlock()

	w.wmu.Lock()
	_ = writeEnvelope(conn, envelope{Type: rMsgLobby, Rooms: rooms, Stats: &sv}, writeTimeout)
	w.wmu.Unlock()
	go h.broadcastLobbyExcept(true, w.id)
	return w
}

func (h *hub) removeWatcher(w *watcher) {
	h.mu.Lock()
	delete(h.watchers, w.id)
	h.mu.Unlock()
	go h.broadcastLobby(true)
}

func (h *hub) createRoom(conn net.Conn, req envelope) (*room, error) {
	h.mu.Lock()
	if len(h.rooms) >= maxRooms {
		h.mu.Unlock()
		return nil, errors.New(errServerBusy)
	}
	id := randomRoomID()
	for h.rooms[id] != nil {
		id = randomRoomID()
	}
	r := &room{
		id:        id,
		hostConn:  conn,
		hostToken: req.Token,
		info: RoomInfo{
			ID:          id,
			Host:        cleanDisplay(req.Name, "Host", 16),
			BestOf:      validBestOf(req.BestOf),
			Rules:       req.Rules,
			TurnSeconds: validTurnSec(req.TurnSeconds),
			Locked:      req.Locked,
			CreatedAt:   time.Now().UnixMilli(),
		},
	}
	h.rooms[id] = r
	_, sv := h.snapshotLocked()
	h.mu.Unlock()

	if err := r.writeHost(envelope{
		Type:   rMsgHosted,
		RoomID: id,
		Stats:  &sv,
	}); err != nil {
		h.removeRoom(r)
		return nil, err
	}
	h.broadcastLobby(true)
	return r, nil
}

func (h *hub) removeRoom(r *room) {
	h.mu.Lock()
	if r.closed.Swap(true) {
		h.mu.Unlock()
		return
	}
	guest := r.guestConn
	r.guestConn = nil
	delete(h.rooms, r.id)
	h.mu.Unlock()

	_ = r.hostConn.Close()
	if guest != nil {
		_ = guest.Close()
	}
	h.broadcastLobby(true)
}

func (h *hub) pairGuest(conn net.Conn, req envelope) (*room, int, error) {
	h.mu.Lock()
	r := h.rooms[req.RoomID]
	if r == nil || r.closed.Load() {
		h.mu.Unlock()
		return nil, 0, errors.New(errRoomNotFound)
	}
	if r.guestConn != nil {
		h.mu.Unlock()
		return nil, 0, errors.New(errRoomFull)
	}
	if r.live {
		if req.Token == "" || req.Token != r.guestToken {
			h.mu.Unlock()
			return nil, 0, errors.New(errRoomFull)
		}
	} else {
		r.guestToken = req.Token
	}
	r.connSeq++
	seq := r.connSeq
	r.guestConn = conn
	h.mu.Unlock()

	if err := writeEnvelope(conn, envelope{
		Type:    rMsgPaired,
		RoomID:  r.id,
		ConnSeq: seq,
	}, writeTimeout); err != nil {
		h.dropGuest(r, seq, false)
		return nil, 0, err
	}
	if err := r.writeHost(envelope{
		Type:    rMsgGuestJoined,
		RoomID:  r.id,
		ConnSeq: seq,
		Name:    cleanDisplay(req.Name, "Guest", 16),
	}); err != nil {
		h.dropGuest(r, seq, false)
		return nil, 0, err
	}
	h.broadcastLobby(true)
	return r, seq, nil
}

func (h *hub) dropGuest(r *room, seq int, notifyHost bool) {
	h.mu.Lock()
	if r.closed.Load() || r.connSeq != seq || r.guestConn == nil {
		h.mu.Unlock()
		return
	}
	guest := r.guestConn
	r.guestConn = nil
	h.mu.Unlock()

	_ = guest.Close()
	if notifyHost {
		_ = r.writeHost(envelope{
			Type:    rMsgGuestDrop,
			RoomID:  r.id,
			ConnSeq: seq,
		})
	}
	h.broadcastLobby(true)
}

func (h *hub) serveConn(raw net.Conn, getTLS func() *tls.Config) {
	h.mu.Lock()
	if h.activeConns >= maxConns {
		h.mu.Unlock()
		_ = raw.Close()
		return
	}
	h.activeConns++
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		h.activeConns--
		h.mu.Unlock()
	}()

	ac, err := acceptAnyConn(raw, getTLS)
	if err != nil {
		_ = raw.Close()
		return
	}
	defer ac.Close()

	if ac.isHTTPOnly {
		h.handleHTTPGet(ac)
		return
	}

	br := bufio.NewReaderSize(ac.Conn, 4096)
	line, err := readJSONLine(br, ac.Conn, 15*time.Second, maxServerMsgBytes)
	if err != nil || len(line) == 0 {
		return
	}
	var first envelope
	if err := json.Unmarshal(line, &first); err != nil {
		_ = writeEnvelope(ac.Conn, envelope{Type: rMsgError, Code: errBadRequest, Text: "invalid request"}, writeTimeout)
		return
	}

	switch first.Type {
	case rMsgWatch:
		h.runWatcher(ac.Conn, br)
	case rMsgHost:
		h.runHost(ac.Conn, br, first)
	case rMsgJoin:
		h.runGuest(ac.Conn, br, first)
	default:
		_ = writeEnvelope(ac.Conn, envelope{Type: rMsgError, Code: errBadRequest, Text: "unknown command"}, writeTimeout)
	}
}

func (h *hub) handleHTTPGet(ac *acceptedConn) {
	h.mu.Lock()
	live, waiting := h.countsLocked()
	watchers := len(h.watchers)
	sv := h.stats.Snapshot(live, len(waiting), watchers)
	matches := h.liveMatchesLocked()
	full := h.stats.FullDump(live, len(waiting), watchers, waiting, matches)
	h.mu.Unlock()

	var body []byte
	if ac.httpPath == "/stats" || ac.httpPath == "/stats.json" {
		body, _ = json.MarshalIndent(full, "", "  ")
	} else {
		body, _ = json.Marshal(map[string]any{
			"service": "knucklebones-rendezvous",
			"status":  "ok",
			"stats":   sv,
		})
	}
	resp := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nAccess-Control-Allow-Origin: *\r\nContent-Length: %d\r\nConnection: close\r\n\r\n", len(body))
	_ = ac.SetWriteDeadline(time.Now().Add(writeTimeout))
	_, _ = ac.Write([]byte(resp))
	_, _ = ac.Write(body)
}

func (h *hub) runWatcher(conn net.Conn, br *bufio.Reader) {
	w := h.addWatcher(conn)
	defer h.removeWatcher(w)
	for {
		line, err := readJSONLine(br, conn, idleControlTimeout, maxServerMsgBytes)
		if err != nil {
			return
		}
		if len(line) == 0 {
			continue
		}
		var env envelope
		if json.Unmarshal(line, &env) != nil {
			continue
		}
		if env.Type == rMsgPing {
			w.wmu.Lock()
			err = writeEnvelope(conn, envelope{Type: rMsgPong}, writeTimeout)
			w.wmu.Unlock()
			if err != nil {
				return
			}
		} else if env.Type == rMsgWatch {
			h.mu.Lock()
			rooms, sv := h.snapshotLocked()
			h.mu.Unlock()
			w.wmu.Lock()
			err = writeEnvelope(conn, envelope{Type: rMsgLobby, Rooms: rooms, Stats: &sv}, writeTimeout)
			w.wmu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

func (h *hub) runHost(conn net.Conn, br *bufio.Reader, req envelope) {
	r, err := h.createRoom(conn, req)
	if err != nil {
		_ = writeEnvelope(conn, envelope{Type: rMsgError, Code: err.Error(), Text: err.Error()}, writeTimeout)
		return
	}
	defer h.removeRoom(r)

	for {
		line, err := readJSONLine(br, conn, idleControlTimeout, maxServerMsgBytes)
		if err != nil {
			return
		}
		if len(line) == 0 {
			continue
		}
		var env envelope
		if json.Unmarshal(line, &env) != nil {
			continue
		}
		switch env.Type {
		case rMsgPing:
			_ = r.writeHost(envelope{Type: rMsgPong})
		case rMsgGuestDrop:
			h.dropGuest(r, env.ConnSeq, false)
		case rMsgRelay:
			if len(env.Data) > maxWireMsgBytes {
				return
			}
			h.mu.Lock()
			active := !r.closed.Load() && env.ConnSeq == r.connSeq
			guest := r.guestConn
			h.mu.Unlock()

			var inner wireGameEnvelope
			if json.Unmarshal([]byte(env.Data), &inner) == nil && inner.Type == "state" && inner.State != nil {
				becameLive := false
				h.mu.Lock()
				if !r.live {
					r.live = true
					becameLive = true
				}
				changed, completedNow, _ := h.stats.ObserveState(&r.progress, inner.State)
				h.mu.Unlock()
				if becameLive || completedNow {
					h.broadcastLobby(true)
				} else if changed {
					h.broadcastLobby(false)
				}
			}

			if active && guest != nil {
				if writeRawLine(guest, []byte(env.Data), writeTimeout) != nil {
					h.dropGuest(r, env.ConnSeq, true)
				}
			}
		}
	}
}

func (h *hub) runGuest(conn net.Conn, br *bufio.Reader, req envelope) {
	r, seq, err := h.pairGuest(conn, req)
	if err != nil {
		code := err.Error()
		_ = writeEnvelope(conn, envelope{Type: rMsgError, Code: code, Text: code}, writeTimeout)
		return
	}
	defer h.dropGuest(r, seq, true)

	for {
		line, err := readJSONLine(br, conn, relayReadTimeout, maxWireMsgBytes)
		if err != nil {
			return
		}
		if len(line) == 0 {
			continue
		}
		if err := r.writeHost(envelope{
			Type:    rMsgRelay,
			ConnSeq: seq,
			Data:    string(line),
		}); err != nil {
			h.removeRoom(r)
			return
		}
	}
}

type certReloader struct {
	certPath string
	keyPath  string
	mu       sync.RWMutex
	cert     *tls.Certificate
	cfg      *tls.Config
	mtime    time.Time
}

func newCertReloader(certPath, keyPath string) *certReloader {
	if certPath == "" || keyPath == "" {
		return nil
	}
	cr := &certReloader{certPath: certPath, keyPath: keyPath}
	if err := cr.reload(); err != nil {
		log.Fatalf("failed to load TLS certificate: %v", err)
	}
	return cr
}

func (c *certReloader) reload() error {
	info, err := os.Stat(c.certPath)
	if err != nil {
		return err
	}
	cert, err := tls.LoadX509KeyPair(c.certPath, c.keyPath)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.cert = &cert
	c.mtime = info.ModTime()
	c.cfg = &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			c.mu.RLock()
			defer c.mu.RUnlock()
			return c.cert, nil
		},
	}
	c.mu.Unlock()
	return nil
}

func (c *certReloader) config() *tls.Config {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	cfg := c.cfg
	c.mu.RUnlock()
	return cfg
}

func main() {
	addr := flag.String("addr", ":8647", "TCP/WS/WSS listen address")
	statePath := flag.String("state", "stats.json", "Path to persistent JSON statistics file")
	certPath := flag.String("cert", "", "Path to TLS fullchain PEM (enables WSS/TLS on the same port)")
	keyPath := flag.String("key", "", "Path to TLS private key PEM")
	flag.Parse()

	cr := newCertReloader(*certPath, *keyPath)
	h := newHub(*statePath)

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen %s: %v", *addr, err)
	}
	defer ln.Close()

	log.Printf("knucklebones-server listening on %s (tls=%v, stats=%s)", ln.Addr(), cr != nil, *statePath)

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		for sig := range sigCh {
			if sig == syscall.SIGHUP {
				if cr != nil {
					if err := cr.reload(); err != nil {
						log.Printf("TLS cert reload failed: %v", err)
					} else {
						log.Printf("TLS cert reloaded")
					}
				}
				continue
			}
			log.Printf("shutting down (%v), flushing stats...", sig)
			h.stats.Flush()
			_ = ln.Close()
			os.Exit(0)
		}
	}()

	for {
		c, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}
		go h.serveConn(c, cr.config)
	}
}
