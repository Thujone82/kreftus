package main

import (
	"bufio"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	wsGUID            = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	detectTimeout     = 1500 * time.Millisecond
	tlsHandshakeWait  = 5 * time.Second
	wsUpgradeWait     = 5 * time.Second
	maxWireMsgBytes   = 64 * 1024
	maxServerMsgBytes = 128 * 1024
)

var (
	errWSNotUpgrade  = errors.New("not a websocket upgrade request")
	errWSBadFrame    = errors.New("invalid websocket frame")
	errWSMsgTooLarge = errors.New("websocket message exceeds limit")
	errLineTooLong   = errors.New("wire message exceeds limit")
)

type peekConn struct {
	net.Conn
	r *bufio.Reader
}

func (p *peekConn) Read(b []byte) (int, error) {
	return p.r.Read(b)
}

type acceptedConn struct {
	net.Conn
	isWS       bool
	isTLS      bool
	isHTTPOnly bool
	httpPath   string
}

// acceptAnyConn multiplexes TLS (0x16), WebSocket HTTP GET ('G'), and raw JSON-line TCP
// on a single listening port.
func acceptAnyConn(raw net.Conn, getTLSConfig func() *tls.Config) (*acceptedConn, error) {
	br := bufio.NewReaderSize(raw, 4096)
	_ = raw.SetReadDeadline(time.Now().Add(detectTimeout))
	first, err := br.Peek(1)
	_ = raw.SetReadDeadline(time.Time{})
	if err != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() && br.Buffered() == 0 {
			return &acceptedConn{Conn: raw}, nil
		}
		return nil, err
	}

	var conn net.Conn = &peekConn{Conn: raw, r: br}
	isTLS := false

	if first[0] == 0x16 && getTLSConfig != nil {
		if cfg := getTLSConfig(); cfg != nil {
			tlsConn := tls.Server(conn, cfg)
			_ = tlsConn.SetDeadline(time.Now().Add(tlsHandshakeWait))
			if err := tlsConn.Handshake(); err != nil {
				_ = tlsConn.Close()
				return nil, err
			}
			_ = tlsConn.SetDeadline(time.Time{})
			isTLS = true
			br = bufio.NewReaderSize(tlsConn, 4096)
			_ = tlsConn.SetReadDeadline(time.Now().Add(detectTimeout))
			next, err := br.Peek(1)
			_ = tlsConn.SetReadDeadline(time.Time{})
			if err != nil {
				var ne net.Error
				if errors.As(err, &ne) && ne.Timeout() && br.Buffered() == 0 {
					return &acceptedConn{Conn: tlsConn, isTLS: true}, nil
				}
				_ = tlsConn.Close()
				return nil, err
			}
			conn = &peekConn{Conn: tlsConn, r: br}
			first = next
		}
	}

	if first[0] == 'G' {
		_ = conn.SetDeadline(time.Now().Add(wsUpgradeWait))
		req, err := http.ReadRequest(br)
		if err != nil {
			return nil, err
		}
		if err := writeWSUpgrade(conn, req); err != nil {
			_ = conn.SetDeadline(time.Time{})
			return &acceptedConn{
				Conn:       conn,
				isTLS:      isTLS,
				isHTTPOnly: true,
				httpPath:   req.URL.Path,
			}, nil
		}
		_ = conn.SetDeadline(time.Time{})
		return &acceptedConn{
			Conn:  newWSConn(conn, br, true, maxServerMsgBytes),
			isWS:  true,
			isTLS: isTLS,
		}, nil
	}

	return &acceptedConn{Conn: conn, isTLS: isTLS}, nil
}

func headerHasToken(h http.Header, name, token string) bool {
	for _, v := range h.Values(name) {
		for _, part := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}

func computeWSAccept(key string) string {
	sum := sha1.Sum([]byte(key + wsGUID))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func writeWSUpgrade(w io.Writer, req *http.Request) error {
	if req.Method != http.MethodGet {
		return errWSNotUpgrade
	}
	if !headerHasToken(req.Header, "Connection", "upgrade") ||
		!strings.EqualFold(strings.TrimSpace(req.Header.Get("Upgrade")), "websocket") {
		return errWSNotUpgrade
	}
	key := strings.TrimSpace(req.Header.Get("Sec-WebSocket-Key"))
	if key == "" {
		return errWSNotUpgrade
	}
	if ver := strings.TrimSpace(req.Header.Get("Sec-WebSocket-Version")); ver != "" && ver != "13" {
		return errWSNotUpgrade
	}
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + computeWSAccept(key) + "\r\n\r\n"
	_, err := io.WriteString(w, resp)
	return err
}

type wsConn struct {
	net.Conn
	br           *bufio.Reader
	expectMasked bool
	maxMsgBytes  int
	pending      []byte
	wmu          sync.Mutex
	closeOnce    sync.Once
}

func newWSConn(conn net.Conn, br *bufio.Reader, expectMasked bool, maxMsg int) *wsConn {
	if br == nil {
		br = bufio.NewReaderSize(conn, 4096)
	}
	if maxMsg <= 0 {
		maxMsg = maxWireMsgBytes
	}
	return &wsConn{
		Conn:         conn,
		br:           br,
		expectMasked: expectMasked,
		maxMsgBytes:  maxMsg,
	}
}

func (w *wsConn) Read(p []byte) (int, error) {
	if len(w.pending) == 0 {
		msg, err := w.readNextMessage()
		if err != nil {
			return 0, err
		}
		w.pending = msg
	}
	n := copy(p, w.pending)
	w.pending = w.pending[n:]
	return n, nil
}

func (w *wsConn) Write(p []byte) (int, error) {
	if err := w.writeFrame(0x1, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *wsConn) Close() error {
	var err error
	w.closeOnce.Do(func() {
		_ = w.SetWriteDeadline(time.Now().Add(200 * time.Millisecond))
		_ = w.writeFrame(0x8, nil)
		err = w.Conn.Close()
	})
	return err
}

func (w *wsConn) writeFrame(opcode byte, payload []byte) error {
	w.wmu.Lock()
	defer w.wmu.Unlock()

	var hdr [10]byte
	hdr[0] = 0x80 | (opcode & 0x0f)
	hdrLen := 2
	n := len(payload)
	switch {
	case n < 126:
		hdr[1] = byte(n)
	case n <= 0xffff:
		hdr[1] = 126
		binary.BigEndian.PutUint16(hdr[2:4], uint16(n))
		hdrLen = 4
	default:
		hdr[1] = 127
		binary.BigEndian.PutUint64(hdr[2:10], uint64(n))
		hdrLen = 10
	}
	if _, err := w.Conn.Write(hdr[:hdrLen]); err != nil {
		return err
	}
	if n > 0 {
		if _, err := w.Conn.Write(payload); err != nil {
			return err
		}
	}
	return nil
}

func (w *wsConn) readNextMessage() ([]byte, error) {
	var out []byte
	inFrag := false
	for {
		fin, opcode, payload, err := w.readOneFrame(w.maxMsgBytes - len(out))
		if err != nil {
			return nil, err
		}
		switch opcode {
		case 0x8:
			_ = w.Close()
			return nil, io.EOF
		case 0x9:
			if len(payload) > 125 {
				return nil, errWSBadFrame
			}
			_ = w.writeFrame(0xA, payload)
			continue
		case 0xA:
			continue
		case 0x1, 0x2:
			if inFrag {
				return nil, errWSBadFrame
			}
			out = append(out, payload...)
			if fin {
				return ensureTrailingNewline(out), nil
			}
			inFrag = true
		case 0x0:
			if !inFrag {
				return nil, errWSBadFrame
			}
			out = append(out, payload...)
			if fin {
				return ensureTrailingNewline(out), nil
			}
		default:
			return nil, errWSBadFrame
		}
	}
}

func ensureTrailingNewline(b []byte) []byte {
	if len(b) == 0 || b[len(b)-1] != '\n' {
		return append(b, '\n')
	}
	return b
}

func (w *wsConn) readOneFrame(maxPayload int) (bool, byte, []byte, error) {
	var hdr [2]byte
	if _, err := io.ReadFull(w.br, hdr[:]); err != nil {
		return false, 0, nil, err
	}
	fin := hdr[0]&0x80 != 0
	opcode := hdr[0] & 0x0f
	masked := hdr[1]&0x80 != 0
	if w.expectMasked && !masked {
		return false, 0, nil, errWSBadFrame
	}
	length := uint64(hdr[1] & 0x7f)
	switch length {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(w.br, ext[:]); err != nil {
			return false, 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(w.br, ext[:]); err != nil {
			return false, 0, nil, err
		}
		length = binary.BigEndian.Uint64(ext[:])
	}
	if opcode >= 0x8 && (!fin || length > 125) {
		return false, 0, nil, errWSBadFrame
	}
	if int(length) > maxPayload || length > uint64(w.maxMsgBytes) {
		return false, 0, nil, errWSMsgTooLarge
	}
	var maskKey [4]byte
	if masked {
		if _, err := io.ReadFull(w.br, maskKey[:]); err != nil {
			return false, 0, nil, err
		}
	}
	payload := make([]byte, int(length))
	if length > 0 {
		if _, err := io.ReadFull(w.br, payload); err != nil {
			return false, 0, nil, err
		}
		if masked {
			for i := range payload {
				payload[i] ^= maskKey[i%4]
			}
		}
	}
	return fin, opcode, payload, nil
}
