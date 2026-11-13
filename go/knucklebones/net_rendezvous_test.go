package main

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"math/big"
	mrand "math/rand/v2"
	"net"
	"sync"
	"testing"
	"time"
)

func selfSignedTLSForLoopback(t *testing.T) (*tls.Config, *tls.Config) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(certPEM)
	srv := &tls.Config{Certificates: []tls.Certificate{cert}}
	cli := &tls.Config{RootCAs: pool, ServerName: "127.0.0.1"}
	return srv, cli
}

type miniRoom struct {
	id       string
	host     string
	bestOf   int
	rules    Rules
	locked   bool
	hostConn net.Conn
	hostMu   sync.Mutex
	guest    net.Conn
	seq      int
}

func startMiniRendezvous(t *testing.T, srvTLS *tls.Config) string {
	t.Helper()
	ln, err := tls.Listen("tcp", "127.0.0.1:0", srvTLS)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	var mu sync.Mutex
	rooms := make(map[string]*miniRoom)
	watchers := make(map[net.Conn]*sync.Mutex)

	pushLobby := func() {
		mu.Lock()
		var list []lobbyRoomInfo
		for _, r := range rooms {
			if r.guest == nil {
				list = append(list, lobbyRoomInfo{
					RoomID:    r.id,
					Host:      r.host,
					BestOf:    r.bestOf,
					Rules:     r.rules,
					Locked:    r.locked,
					CreatedAt: time.Now().Unix(),
				})
			}
		}
		env := rendezvousEnvelope{
			Type:  "r_lobby",
			Rooms: list,
			Stats: &lobbyStatsView{WaitingRooms: len(list)},
		}
		targets := make(map[net.Conn]*sync.Mutex, len(watchers))
		for c, wmu := range watchers {
			targets[c] = wmu
		}
		mu.Unlock()
		for c, wmu := range targets {
			wmu.Lock()
			_ = writeRendezvousEnv(c, env)
			wmu.Unlock()
		}
	}

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				br := bufio.NewReaderSize(conn, 8192)
				line, err := readRendezvousLine(br, conn, 5*time.Second)
				if err != nil || len(line) == 0 {
					return
				}
				var first rendezvousEnvelope
				if json.Unmarshal(line, &first) != nil {
					return
				}
				switch first.Type {
				case "r_watch":
					var wmu sync.Mutex
					mu.Lock()
					watchers[conn] = &wmu
					mu.Unlock()
					pushLobby()
					for {
						if _, err := readRendezvousLine(br, conn, 15*time.Second); err != nil {
							break
						}
					}
					mu.Lock()
					delete(watchers, conn)
					mu.Unlock()
				case "r_host":
					r := &miniRoom{
						id:       "room1",
						host:     first.Name,
						bestOf:   first.BestOf,
						rules:    first.Rules,
						locked:   first.Locked,
						hostConn: conn,
					}
					mu.Lock()
					rooms[r.id] = r
					mu.Unlock()
					_ = writeRendezvousEnv(conn, rendezvousEnvelope{Type: "r_hosted", RoomID: r.id})
					pushLobby()
					for {
						l, err := readRendezvousLine(br, conn, 15*time.Second)
						if err != nil {
							break
						}
						var env rendezvousEnvelope
						if json.Unmarshal(l, &env) != nil {
							continue
						}
						switch env.Type {
						case "r_relay":
							mu.Lock()
							g := r.guest
							seq := r.seq
							mu.Unlock()
							if g != nil && env.ConnSeq == seq {
								_, _ = g.Write(append([]byte(env.Data), '\n'))
							}
						case "r_guest_drop":
							mu.Lock()
							if r.guest != nil && env.ConnSeq == r.seq {
								_ = r.guest.Close()
								r.guest = nil
							}
							mu.Unlock()
							pushLobby()
						}
					}
					mu.Lock()
					delete(rooms, r.id)
					if r.guest != nil {
						_ = r.guest.Close()
					}
					mu.Unlock()
					pushLobby()
				case "r_join":
					mu.Lock()
					r := rooms[first.RoomID]
					if r == nil || r.guest != nil {
						mu.Unlock()
						_ = writeRendezvousEnv(conn, rendezvousEnvelope{Type: "r_error", Code: errRoomNotFound})
						return
					}
					r.seq++
					seq := r.seq
					r.guest = conn
					mu.Unlock()

					_ = writeRendezvousEnv(conn, rendezvousEnvelope{Type: "r_paired", RoomID: r.id, ConnSeq: seq})
					r.hostMu.Lock()
					_ = writeRendezvousEnv(r.hostConn, rendezvousEnvelope{Type: "r_guest_joined", ConnSeq: seq})
					r.hostMu.Unlock()
					pushLobby()

					for {
						l, err := readRendezvousLine(br, conn, 15*time.Second)
						if err != nil {
							break
						}
						r.hostMu.Lock()
						_ = writeRendezvousEnv(r.hostConn, rendezvousEnvelope{Type: "r_relay", ConnSeq: seq, Data: string(l)})
						r.hostMu.Unlock()
					}
					mu.Lock()
					if r.guest == conn {
						r.guest = nil
					}
					mu.Unlock()
					r.hostMu.Lock()
					_ = writeRendezvousEnv(r.hostConn, rendezvousEnvelope{Type: "r_guest_drop", ConnSeq: seq})
					r.hostMu.Unlock()
					pushLobby()
				}
			}(c)
		}
	}()
	return ln.Addr().String()
}

func TestGoClientOnlineRendezvousLockedMatchFlow(t *testing.T) {
	srvTLS, cliTLS := selfSignedTLSForLoopback(t)
	addr := startMiniRendezvous(t, srvTLS)

	// Start UI watcher and verify it connects cleanly.
	w := startRendezvousWatcher(addr, cliTLS, false)
	defer w.Close()

	// Start Online Host with a Room Code over TLS.
	host, err := startOnlineHost("Alice", addr, hostConfig{
		BestOf:   1,
		RoomCode: "lock99",
	}, mrand.New(mrand.NewPCG(1, 2)), cliTLS, false)
	if err != nil {
		t.Fatalf("startOnlineHost: %v", err)
	}
	defer host.leave()
	if host.hostRoomID() == "" || host.hostPort() != 0 {
		t.Fatalf("expected non-empty roomID and port=0, got roomID=%q port=%d", host.hostRoomID(), host.hostPort())
	}

	// Watcher must see Alice's locked room.
	deadline := time.Now().Add(3 * time.Second)
	var foundRoom lobbyRoomInfo
	for time.Now().Before(deadline) {
		snap := w.Snapshot()
		if snap.Connected && len(snap.Rooms) == 1 {
			foundRoom = snap.Rooms[0]
			break
		}
		time.Sleep(15 * time.Millisecond)
	}
	if foundRoom.RoomID == "" || foundRoom.Host != "Alice" || !foundRoom.Locked {
		t.Fatalf("watcher did not receive locked room: %+v", foundRoom)
	}

	// Wrong room code attempt returns errBadAuth cleanly while Host remains open!
	badGuest := startOnlineClient("Bob", addr, foundRoom.RoomID, "wrong1", cliTLS, false)
	pumpControllers := func(gctl controller, done func() bool) bool {
		end := time.Now().Add(3 * time.Second)
		for time.Now().Before(end) {
			now := time.Now()
			for more := true; more; {
				more = false
				select {
				case ev := <-host.netEvents():
					host.handle(ev, now)
					more = true
				default:
				}
				select {
				case ev := <-gctl.netEvents():
					gctl.handle(ev, now)
					more = true
				default:
				}
			}
			host.tick(now)
			gctl.tick(now)
			if done() {
				return true
			}
			time.Sleep(5 * time.Millisecond)
		}
		return false
	}
	if !pumpControllers(badGuest, func() bool { return badGuest.lobbyError() == errBadAuth }) {
		t.Fatalf("expected errBadAuth on wrong code, got %q", badGuest.lobbyError())
	}
	badGuest.leave()

	// Valid room code joins and enters phasePlay cleanly!
	goodGuest := startOnlineClient("Bob", addr, foundRoom.RoomID, "lock99", cliTLS, false)
	defer goodGuest.leave()
	if !pumpControllers(goodGuest, func() bool {
		return host.state() != nil && goodGuest.state() != nil && goodGuest.state().Phase == phasePlay
	}) {
		t.Fatalf("failed to enter online play with valid room code; err=%q", goodGuest.lobbyError())
	}
	if goodGuest.state().Names != [2]string{"Alice", "Bob"} {
		t.Fatalf("unexpected names: %v", goodGuest.state().Names)
	}
}
