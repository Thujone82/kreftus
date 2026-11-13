package main

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func startTestServer(t *testing.T, withTLS bool) (*hub, string, *x509.CertPool) {
	t.Helper()
	statsPath := filepath.Join(t.TempDir(), "stats.json")
	h := newHub(statsPath)

	var cr *certReloader
	var pool *x509.CertPool
	if withTLS {
		certFile, keyFile, cp := generateSelfSignedCert(t)
		cr = newCertReloader(certFile, keyFile)
		pool = cp
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go h.serveConn(c, cr.config)
		}
	}()

	return h, ln.Addr().String(), pool
}

func generateSelfSignedCert(t *testing.T) (string, string, *x509.CertPool) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyBytes, _ := x509.MarshalECPrivateKey(priv)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	if err := writeTestFile(certFile, certPEM); err != nil {
		t.Fatal(err)
	}
	if err := writeTestFile(keyFile, keyPEM); err != nil {
		t.Fatal(err)
	}
	cp := x509.NewCertPool()
	cp.AppendCertsFromPEM(certPEM)
	return certFile, keyFile, cp
}

func writeTestFile(path string, b []byte) error {
	return os.WriteFile(path, b, 0o600)
}

func dialWSClient(t *testing.T, addr string, tlsCfg *tls.Config) (net.Conn, *bufio.Reader) {
	t.Helper()
	var conn net.Conn
	var err error
	if tlsCfg != nil {
		conn, err = tls.Dial("tcp", addr, tlsCfg)
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		t.Fatal(err)
	}
	var kb [16]byte
	_, _ = rand.Read(kb[:])
	wsKey := base64.StdEncoding.EncodeToString(kb[:])
	req := fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n", addr, wsKey)
	if _, err := io.WriteString(conn, req); err != nil {
		_ = conn.Close()
		t.Fatal(err)
	}
	br := bufio.NewReaderSize(conn, 4096)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodGet})
	if err != nil {
		_ = conn.Close()
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		_ = conn.Close()
		t.Fatalf("expected 101, got %s", resp.Status)
	}
	wsc := &clientWSConn{Conn: conn, br: br}
	return wsc, bufio.NewReaderSize(wsc, 4096)
}

type clientWSConn struct {
	net.Conn
	br      *bufio.Reader
	pending []byte
}

func (c *clientWSConn) Write(p []byte) (int, error) {
	var hdr [14]byte
	hdr[0] = 0x81
	off := 2
	n := len(p)
	switch {
	case n < 126:
		hdr[1] = 0x80 | byte(n)
	case n <= 0xffff:
		hdr[1] = 0x80 | 126
		binary.BigEndian.PutUint16(hdr[2:4], uint16(n))
		off = 4
	default:
		hdr[1] = 0x80 | 127
		binary.BigEndian.PutUint64(hdr[2:10], uint64(n))
		off = 10
	}
	var mask [4]byte
	_, _ = rand.Read(mask[:])
	copy(hdr[off:off+4], mask[:])
	off += 4
	if _, err := c.Conn.Write(hdr[:off]); err != nil {
		return 0, err
	}
	masked := make([]byte, n)
	for i := range p {
		masked[i] = p[i] ^ mask[i%4]
	}
	if _, err := c.Conn.Write(masked); err != nil {
		return 0, err
	}
	return n, nil
}

func (c *clientWSConn) Read(p []byte) (int, error) {
	if len(c.pending) == 0 {
		var hdr [2]byte
		if _, err := io.ReadFull(c.br, hdr[:]); err != nil {
			return 0, err
		}
		length := uint64(hdr[1] & 0x7f)
		if length == 126 {
			var ext [2]byte
			if _, err := io.ReadFull(c.br, ext[:]); err != nil {
				return 0, err
			}
			length = uint64(binary.BigEndian.Uint16(ext[:]))
		} else if length == 127 {
			var ext [8]byte
			if _, err := io.ReadFull(c.br, ext[:]); err != nil {
				return 0, err
			}
			length = binary.BigEndian.Uint64(ext[:])
		}
		buf := make([]byte, int(length))
		if _, err := io.ReadFull(c.br, buf); err != nil {
			return 0, err
		}
		c.pending = ensureTrailingNewline(buf)
	}
	n := copy(p, c.pending)
	c.pending = c.pending[n:]
	return n, nil
}

func readEnv(t *testing.T, br *bufio.Reader, conn net.Conn) envelope {
	t.Helper()
	line, err := readJSONLine(br, conn, 3*time.Second, maxServerMsgBytes)
	if err != nil {
		t.Fatalf("readJSONLine: %v", err)
	}
	var env envelope
	if err := json.Unmarshal(line, &env); err != nil {
		t.Fatalf("unmarshal %q: %v", string(line), err)
	}
	return env
}

func TestRendezvousLobbyHostJoinStatsAndWSS(t *testing.T) {
	h, addr, cp := startTestServer(t, true)
	tlsCfg := &tls.Config{RootCAs: cp, ServerName: "localhost"}

	// 1. Watcher over WSS receives empty lobby + stats.
	watchConn, watchBR := dialWSClient(t, addr, tlsCfg)
	defer watchConn.Close()
	if err := writeEnvelope(watchConn, envelope{Type: rMsgWatch}, time.Second); err != nil {
		t.Fatal(err)
	}
	firstLobby := readEnv(t, watchBR, watchConn)
	if firstLobby.Type != rMsgLobby || len(firstLobby.Rooms) != 0 || firstLobby.Stats == nil {
		t.Fatalf("unexpected initial lobby: %+v", firstLobby)
	}

	// 2. Host over TLS TCP registers a locked room.
	hostConn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		t.Fatal(err)
	}
	defer hostConn.Close()
	hostBR := bufio.NewReaderSize(hostConn, 4096)
	if err := writeEnvelope(hostConn, envelope{
		Type:        rMsgHost,
		Name:        "HostA",
		BestOf:      3,
		Rules:       Rules{TwoDice: true},
		TurnSeconds: 30,
		Locked:      true,
		Token:       "hosttok",
	}, time.Second); err != nil {
		t.Fatal(err)
	}
	hosted := readEnv(t, hostBR, hostConn)
	if hosted.Type != rMsgHosted || hosted.RoomID == "" {
		t.Fatalf("unexpected hosted ack: %+v", hosted)
	}

	// Watcher sees room with Locked=true and WaitingRooms=1.
	lob := readEnv(t, watchBR, watchConn)
	if len(lob.Rooms) != 1 || lob.Rooms[0].Host != "HostA" || !lob.Rooms[0].Locked || lob.Stats.WaitingRooms != 1 {
		t.Fatalf("unexpected lobby update: %+v", lob)
	}

	// 3. Guest over WSS joins room.
	guestConn, guestBR := dialWSClient(t, addr, tlsCfg)
	defer guestConn.Close()
	if err := writeEnvelope(guestConn, envelope{
		Type:   rMsgJoin,
		RoomID: hosted.RoomID,
		Token:  "guesttok1",
		Name:   "GuestB",
	}, time.Second); err != nil {
		t.Fatal(err)
	}
	paired := readEnv(t, guestBR, guestConn)
	if paired.Type != rMsgPaired || paired.RoomID != hosted.RoomID {
		t.Fatalf("unexpected paired msg: %+v", paired)
	}
	joined := readEnv(t, hostBR, hostConn)
	if joined.Type != rMsgGuestJoined || joined.ConnSeq != paired.ConnSeq {
		t.Fatalf("unexpected guest joined msg: %+v", joined)
	}

	// 4. Host relays initial state + 1 move + game completion -> verify telemetry & persistence.
	stPlay1 := wireSession{
		Mode:   "online",
		Names:  [2]string{"HostA", "GuestB"},
		BestOf: 1,
		Series: 1,
		Round:  1,
		Phase:  phasePlay,
		Match:  wireMatch{Moves: 0, Turn: 0},
	}
	stPlay1Bytes, _ := json.Marshal(map[string]any{"type": "state", "state": stPlay1})
	_ = writeEnvelope(hostConn, envelope{Type: rMsgRelay, ConnSeq: joined.ConnSeq, Data: string(stPlay1Bytes)}, time.Second)
	rawGuestLine, err := readJSONLine(guestBR, guestConn, 2*time.Second, maxWireMsgBytes)
	if err != nil || len(rawGuestLine) == 0 {
		t.Fatalf("guest failed to read relayed state: %v", err)
	}

	// Move 1 (die=6, destroyed=1) + game over state.
	var g1, g2 wireGrid
	g1[0] = [3]int{6, 6, 6}
	g2[1] = [3]int{4, 0, 0}
	stDone := wireSession{
		Mode:   "online",
		Names:  [2]string{"HostA", "GuestB"},
		BestOf: 1,
		Series: 1,
		Round:  1,
		Wins:   [2]int{1, 0},
		Phase:  phaseGameOver,
		Last:   &wirePlacement{Player: 0, Col: 0, Slot: 2, Die: 6, Destroyed: 1, Gain: 30, Lost: 6},
		History: []wireHistoryEntry{{
			Round:  1,
			Scores: [2]int{54, 4},
			Winner: 0,
			Lead:   0,
		}},
		Match: wireMatch{
			Grids: [2]wireGrid{g1, g2},
			Moves: 1,
		},
	}
	stDoneBytes, _ := json.Marshal(map[string]any{"type": "state", "state": stDone})
	_ = writeEnvelope(hostConn, envelope{Type: rMsgRelay, ConnSeq: joined.ConnSeq, Data: string(stDoneBytes)}, time.Second)
	_, err = readJSONLine(guestBR, guestConn, 2*time.Second, maxWireMsgBytes)
	if err != nil {
		t.Fatal(err)
	}

	// Read watcher notifications until we see CompletedMatches == 1.
	deadline := time.Now().Add(3 * time.Second)
	var finalStats *StatsView
	for time.Now().Before(deadline) {
		ev := readEnv(t, watchBR, watchConn)
		if ev.Type == rMsgLobby && ev.Stats != nil && ev.Stats.CompletedMatches == 1 {
			finalStats = ev.Stats
			break
		}
	}
	if finalStats == nil {
		t.Fatalf("timed out waiting for completed match stat update")
	}
	if finalStats.TotalDicePlayed != 1 || finalStats.TotalDiceDestroyed != 1 || finalStats.TotalPointsScored != 58 || finalStats.HighestRound == nil || finalStats.HighestRound.Score != 54 {
		t.Fatalf("unexpected stats: %+v", finalStats)
	}

	// Reload persisted stats from disk to verify persistence.
	reloaded := newStatsStore(h.stats.path)
	snap := reloaded.Snapshot(0, 0)
	if snap.CompletedMatches != 1 || snap.TotalDicePlayed != 1 {
		t.Fatalf("reloaded stats mismatch: %+v", snap)
	}
}

func TestServerBadRoomCodeAndReconnect(t *testing.T) {
	_, addr, pool := startTestServer(t, true)
	tlsCfg := &tls.Config{RootCAs: pool, ServerName: "127.0.0.1"}

	// Host opens a locked room over TLS TCP.
	hostConn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		t.Fatal(err)
	}
	defer hostConn.Close()
	hostBR := bufio.NewReaderSize(hostConn, 8192)
	if err := writeEnvelope(hostConn, envelope{
		Type:        rMsgHost,
		Name:        "HostWait",
		BestOf:      3,
		Rules:       Rules{},
		TurnSeconds: 30,
		Locked:      true,
		Token:       "hostwait1",
	}, time.Second); err != nil {
		t.Fatal(err)
	}
	hosted := readEnv(t, hostBR, hostConn)
	if hosted.Type != rMsgHosted || hosted.RoomID == "" {
		t.Fatalf("unexpected hosted ack: %+v", hosted)
	}

	// Watcher verifies WaitingRooms == 1.
	watchConn, watchBR := dialWSClient(t, addr, tlsCfg)
	defer watchConn.Close()
	_ = writeEnvelope(watchConn, envelope{Type: rMsgWatch}, time.Second)
	lob := readEnv(t, watchBR, watchConn)
	if lob.Stats == nil || lob.Stats.WaitingRooms != 1 {
		t.Fatalf("expected 1 waiting room, got %+v", lob.Stats)
	}

	// First guest joins, receives r_guest_joined at host, then host rejects / closes guest conn (wrong room code).
	g1Conn, g1BR := dialWSClient(t, addr, tlsCfg)
	_ = writeEnvelope(g1Conn, envelope{Type: rMsgJoin, RoomID: hosted.RoomID, Token: "g1", Name: "Wrong"}, time.Second)
	p1 := readEnv(t, g1BR, g1Conn)
	j1 := readEnv(t, hostBR, hostConn)
	if p1.Type != rMsgPaired || j1.Type != rMsgGuestJoined || j1.ConnSeq != p1.ConnSeq {
		t.Fatalf("unexpected pairing: p1=%+v j1=%+v", p1, j1)
	}
	// Host sends error relay and tells server to drop guest 1 pre-game.
	errBytes, _ := json.Marshal(map[string]any{"type": "error", "error": "bad-room-code"})
	_ = writeEnvelope(hostConn, envelope{Type: rMsgRelay, ConnSeq: j1.ConnSeq, Data: string(errBytes)}, time.Second)
	_ = writeEnvelope(hostConn, envelope{Type: rMsgGuestDrop, ConnSeq: j1.ConnSeq}, time.Second)
	rawErr, err := readJSONLine(g1BR, g1Conn, 2*time.Second, maxWireMsgBytes)
	if err != nil || len(rawErr) == 0 {
		t.Fatalf("expected bad-room-code relay to g1: %v", err)
	}
	_ = g1Conn.Close()

	// Room must STILL be waiting in lobby and accept a second guest attempt!
	time.Sleep(50 * time.Millisecond)
	g2Conn, g2BR := dialWSClient(t, addr, tlsCfg)
	defer g2Conn.Close()
	_ = writeEnvelope(g2Conn, envelope{Type: rMsgJoin, RoomID: hosted.RoomID, Token: "g2", Name: "Right"}, time.Second)
	p2 := readEnv(t, g2BR, g2Conn)
	j2 := readEnv(t, hostBR, hostConn)
	if p2.Type != rMsgPaired || j2.Type != rMsgGuestJoined || j2.ConnSeq <= j1.ConnSeq {
		t.Fatalf("second guest join failed: p2=%+v j2=%+v", p2, j2)
	}

	// Host marks game started by sending phase=play state -> WaitingRooms becomes 0, LiveMatches becomes 1.
	stPlay, _ := json.Marshal(map[string]any{"type": "state", "state": wireSession{
		Mode: "online", Names: [2]string{"HostWait", "Right"}, BestOf: 3, Series: 1, Round: 1, Phase: phasePlay,
	}})
	_ = writeEnvelope(hostConn, envelope{Type: rMsgRelay, ConnSeq: j2.ConnSeq, Data: string(stPlay)}, time.Second)
	_, _ = readJSONLine(g2BR, g2Conn, 2*time.Second, maxWireMsgBytes)

	// Guest 2 drops and reconnects mid-game using the same guest connection flow.
	_ = g2Conn.Close()
	dropEnv := readEnv(t, hostBR, hostConn)
	if dropEnv.Type != rMsgGuestDrop || dropEnv.ConnSeq != j2.ConnSeq {
		t.Fatalf("expected host r_guest_drop for g2, got %+v", dropEnv)
	}
	g2ReConn, g2ReBR := dialWSClient(t, addr, tlsCfg)
	defer g2ReConn.Close()
	_ = writeEnvelope(g2ReConn, envelope{Type: rMsgJoin, RoomID: hosted.RoomID, Token: "g2", Name: "Right"}, time.Second)
	p3 := readEnv(t, g2ReBR, g2ReConn)
	j3 := readEnv(t, hostBR, hostConn)
	if p3.Type != rMsgPaired || j3.Type != rMsgGuestJoined || j3.ConnSeq <= j2.ConnSeq {
		t.Fatalf("reconnect join failed: p3=%+v j3=%+v", p3, j3)
	}
}
