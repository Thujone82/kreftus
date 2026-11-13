package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func dialWebSocket(t *testing.T, addr string) (*bufio.Reader, net.Conn, func([]byte) error) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	var keyBytes [16]byte
	_, _ = rand.Read(keyBytes[:])
	wsKey := base64.StdEncoding.EncodeToString(keyBytes[:])

	req := fmt.Sprintf(
		"GET / HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n",
		addr, wsKey,
	)
	if _, err := io.WriteString(conn, req); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	br := bufio.NewReaderSize(conn, 4096)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodGet})
	if err != nil {
		conn.Close()
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols || resp.Header.Get("Sec-WebSocket-Accept") != computeWSAccept(wsKey) {
		conn.Close()
		t.Fatalf("unexpected WS response: %s (%q)", resp.Status, resp.Header.Get("Sec-WebSocket-Accept"))
	}

	writeMaskedText := func(payload []byte) error {
		var hdr [14]byte
		hdr[0] = 0x81 // FIN + Text
		off := 2
		n := len(payload)
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
		if _, err := conn.Write(hdr[:off]); err != nil {
			return err
		}
		masked := make([]byte, len(payload))
		for i, b := range payload {
			masked[i] = b ^ mask[i%4]
		}
		_, err := conn.Write(masked)
		return err
	}

	return br, conn, writeMaskedText
}

func readWSFrameTextErr(r io.Reader) ([]byte, error) {
	var hdr [2]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	opcode := hdr[0] & 0x0f
	if opcode != 0x1 {
		return nil, fmt.Errorf("expected text frame opcode 1, got %d", opcode)
	}
	length := uint64(hdr[1] & 0x7f)
	switch length {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(r, ext[:]); err != nil {
			return nil, err
		}
		length = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(r, ext[:]); err != nil {
			return nil, err
		}
		length = binary.BigEndian.Uint64(ext[:])
	}
	buf := make([]byte, int(length))
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func TestWebSocketGuestHandshakeAndGameplay(t *testing.T) {
	h, err := startHost("Host", 0, hostConfig{BestOf: 1, RoomCode: "KREFT99"}, testRNG())
	if err != nil {
		t.Fatal(err)
	}
	defer h.leave()

	addr := fmt.Sprintf("127.0.0.1:%d", h.port)
	br, conn, sendWS := dialWebSocket(t, addr)
	defer conn.Close()

	wsMsgs := make(chan netMsg, 16)
	wsErrs := make(chan error, 1)
	go func() {
		for {
			raw, err := readWSFrameTextErr(br)
			if err != nil {
				wsErrs <- err
				return
			}
			var m netMsg
			if err := json.Unmarshal(raw, &m); err != nil {
				wsErrs <- err
				return
			}
			wsMsgs <- m
		}
	}()

	nextWSMsg := func() netMsg {
		t.Helper()
		deadline := time.After(3 * time.Second)
		for {
			select {
			case m := <-wsMsgs:
				return m
			case err := <-wsErrs:
				t.Fatalf("ws read error: %v", err)
			case ev := <-h.events:
				h.handle(ev, time.Now())
			case <-deadline:
				t.Fatal("timed out waiting for ws frame")
			case <-time.After(10 * time.Millisecond):
				h.tick(time.Now())
			}
		}
	}

	// 1. Read challenge sent by host in unmasked WS text frame.
	chal := nextWSMsg()
	if chal.Type != msgChallenge || !chal.AuthReq || chal.Nonce == "" || chal.Commit == "" {
		t.Fatalf("unexpected challenge: %+v", chal)
	}

	roomKey, err := deriveRoomKey("KREFT99")
	if err != nil {
		t.Fatal(err)
	}
	wk, err := deriveWireKey(roomKey, chal.Nonce)
	if err != nil {
		t.Fatal(err)
	}

	guestSeed := newRandomHex(32)
	token := newToken()
	hello := netMsg{
		Type:    msgHello,
		Version: protocolVersion,
		Name:    "WSGuest",
		Token:   token,
		Seed:    guestSeed,
		Auth:    helloAuthMAC(roomKey, chal.Nonce, "WSGuest", token, guestSeed),
	}
	hb, _ := json.Marshal(hello)
	if err := sendWS(hb); err != nil {
		t.Fatal(err)
	}

	// 2. Read initial state frame and verify MAC.
	stMsg := nextWSMsg()
	if stMsg.Type != msgState || stMsg.State == nil {
		t.Fatalf("unexpected state msg: %+v", stMsg)
	}
	if !verifyWireMsg(wk, 1, stMsg) {
		t.Fatalf("verifyWireMsg failed on state message from host")
	}
	if stMsg.State.Names != [2]string{"Host", "WSGuest"} {
		t.Fatalf("unexpected names: %v", stMsg.State.Names)
	}

	// 3. Play a signed move (letting host move first if roundCoin picked seat 0).
	if stMsg.State.Match.Turn == 0 {
		h.place(Move{Col: 0, Die: 0})
		for stMsg.State.Match.Turn == 0 && stMsg.State.Phase == phasePlay {
			m := nextWSMsg()
			if m.Type == msgState && m.State != nil {
				stMsg = m
			}
		}
	}
	if stMsg.State.Phase == phasePlay && stMsg.State.Match.Turn == 1 {
		mv := netMsg{Type: msgMove, Col: 1, Die: 0}
		signWireMsg(wk, 1, &mv)
		mb, _ := json.Marshal(mv)
		if err := sendWS(mb); err != nil {
			t.Fatal(err)
		}
		for stMsg.State.Match.Turn == 1 && stMsg.State.Phase == phasePlay {
			m := nextWSMsg()
			if m.Type == msgState && m.State != nil {
				if !verifyWireMsg(wk, 1, m) {
					t.Fatalf("verifyWireMsg failed after guest move")
				}
				stMsg = m
			}
		}
		if stMsg.State.Last == nil || stMsg.State.Last.Player != 1 || stMsg.State.Last.Col != 1 {
			t.Fatalf("expected guest move reflected in state, got %+v", stMsg.State.Last)
		}
	}
}
