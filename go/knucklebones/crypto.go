package main

import (
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"unicode"
)

const maxRoomCodeLen = 12

// cleanRoomCode strips whitespace and control runes from a room code and caps its length.
func cleanRoomCode(s string) string {
	var b strings.Builder
	count := 0
	for _, r := range strings.TrimSpace(s) {
		if unicode.IsSpace(r) || !unicode.IsPrint(r) || unicode.IsControl(r) || isBidiControl(r) {
			continue
		}
		if count >= maxRoomCodeLen {
			break
		}
		b.WriteRune(r)
		count++
	}
	return b.String()
}

func newRandomHex(nBytes int) string {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func deriveRoomKey(roomCode string) ([]byte, error) {
	roomCode = cleanRoomCode(roomCode)
	if roomCode == "" {
		return nil, nil
	}
	return hkdf.Key(sha256.New, []byte(roomCode), []byte("knucklebones-v2-room"), "room-auth", 32)
}

func deriveWireKey(roomKey []byte, nonce string) ([]byte, error) {
	if len(roomKey) == 0 {
		return nil, nil
	}
	return hkdf.Key(sha256.New, roomKey, []byte(nonce), "wire-mac-v2", 32)
}

func helloAuthMAC(roomKey []byte, nonce, name, token, guestSeed string) string {
	if len(roomKey) == 0 {
		return ""
	}
	mac := hmac.New(sha256.New, roomKey)
	mac.Write([]byte("hello\x00" + nonce + "\x00" + name + "\x00" + token + "\x00" + guestSeed))
	return hex.EncodeToString(mac.Sum(nil))
}

func verifyHelloAuth(roomKey []byte, nonce, name, token, guestSeed, gotHex string) bool {
	if len(roomKey) == 0 {
		return false
	}
	want := helloAuthMAC(roomKey, nonce, name, token, guestSeed)
	wb, err1 := hex.DecodeString(want)
	gb, err2 := hex.DecodeString(gotHex)
	if err1 != nil || err2 != nil {
		return false
	}
	return hmac.Equal(wb, gb)
}

func canonicalMsgPayload(m netMsg) []byte {
	m.MAC = ""
	if m.State != nil {
		sCopy := *m.State
		if sCopy.Log == nil {
			sCopy.Log = []logEntry{}
		}
		if sCopy.History == nil {
			sCopy.History = []historyEntry{}
		}
		if sCopy.RoundRolls == nil {
			sCopy.RoundRolls = [][2]int{}
		}
		if sCopy.RoundMoves == nil {
			sCopy.RoundMoves = []replayMove{}
		}
		m.State = &sCopy
	}
	b, _ := json.Marshal(m)
	return b
}

func signWireMsg(wireKey []byte, seq uint64, m *netMsg) {
	if len(wireKey) == 0 {
		return
	}
	m.Seq = seq
	mac := hmac.New(sha256.New, wireKey)
	mac.Write(canonicalMsgPayload(*m))
	m.MAC = hex.EncodeToString(mac.Sum(nil))
}

func verifyWireMsg(wireKey []byte, expectedMinSeq uint64, m netMsg) bool {
	if len(wireKey) == 0 {
		return true
	}
	if m.Seq < expectedMinSeq {
		return false
	}
	gotHex := m.MAC
	gb, err := hex.DecodeString(gotHex)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, wireKey)
	mac.Write(canonicalMsgPayload(m))
	return hmac.Equal(mac.Sum(nil), gb)
}

func commitSeed(seedHex string) string {
	sum := sha256.Sum256([]byte("kb-seed-v2|" + seedHex))
	return hex.EncodeToString(sum[:])
}

func verifySeedCommit(seedHex, commitHex string) bool {
	if seedHex == "" || commitHex == "" {
		return false
	}
	want := commitSeed(seedHex)
	return len(want) == len(commitHex) && subtle.ConstantTimeCompare([]byte(want), []byte(commitHex)) == 1
}

func roundCoin(hostSeed, guestSeed string) int {
	mac := hmac.New(sha256.New, []byte(hostSeed))
	mac.Write([]byte("coin-v2|" + guestSeed))
	return int(mac.Sum(nil)[0] & 1)
}

// roundRoll computes uniform 1..6 dice values using rejection sampling over HMAC-SHA256.
func roundRoll(hostSeed, guestSeed string, moveIdx int, twoDice bool) [2]int {
	need := 1
	if twoDice {
		need = 2
	}
	var out [2]int
	idx := 0
	for ctr := 0; idx < need; ctr++ {
		mac := hmac.New(sha256.New, []byte(hostSeed))
		mac.Write([]byte("roll-v2|" + guestSeed + "|" + strconv.Itoa(moveIdx) + "|" + strconv.Itoa(ctr)))
		sum := mac.Sum(nil)
		for _, b := range sum {
			if b < 252 {
				out[idx] = int(b%6) + 1
				idx++
				if idx == need {
					break
				}
			}
		}
	}
	return out
}
