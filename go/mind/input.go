package main

import (
	"bufio"
	"os"
	"sync"

	"golang.org/x/term"
)

type keyKind int

const (
	keyNone keyKind = iota
	keyPeg
	keyBackspace
	keyEnter
	keyTheme
	keySound
	keyLabels
	keyLabelMode
	keyNew
	keyQuit
)

type keyEvent struct {
	kind keyKind
	peg  byte
}

type termSession struct {
	fd          int
	oldState    *term.State
	restoreOnce sync.Once
	keys        chan keyEvent
}

func openTermSession() (*termSession, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	_, _ = os.Stdout.WriteString("\033[?1049h\033[?25l\033]0;Mastermind - Crack the code!\007")

	s := &termSession{
		fd:       fd,
		oldState: oldState,
		keys:     make(chan keyEvent, 32),
	}
	go s.readLoop()
	return s, nil
}

func (s *termSession) restore() {
	if s == nil {
		return
	}
	s.restoreOnce.Do(func() {
		_, _ = os.Stdout.WriteString("\033[0m\033[?25h\033[?1049l")
		if s.oldState != nil {
			_ = term.Restore(s.fd, s.oldState)
		}
	})
}

func (s *termSession) readLoop() {
	reader := bufio.NewReader(os.Stdin)
	for {
		r, _, err := reader.ReadRune()
		if err != nil {
			close(s.keys)
			return
		}
		ev := parseKey(r, reader)
		if ev.kind != keyNone {
			s.keys <- ev
		}
	}
}

func parseKey(r rune, reader *bufio.Reader) keyEvent {
	if c, ok := keyToColor(r); ok {
		return keyEvent{kind: keyPeg, peg: c}
	}
	switch r {
	case '\b', 127:
		return keyEvent{kind: keyBackspace}
	case '\r', '\n':
		return keyEvent{kind: keyEnter}
	case 3: // Ctrl+C in raw mode
		return keyEvent{kind: keyQuit}
	case 27: // ESC or escape sequence
		if reader.Buffered() > 0 {
			consumeEscapeSequence(reader)
			return keyEvent{kind: keyNone}
		}
		return keyEvent{kind: keyQuit}
	case 't', 'T':
		return keyEvent{kind: keyTheme}
	case 's', 'S':
		return keyEvent{kind: keySound}
	case 'l', 'L':
		return keyEvent{kind: keyLabels}
	case '\t':
		return keyEvent{kind: keyLabelMode}
	case 'n', 'N':
		return keyEvent{kind: keyNew}
	case 'q', 'Q':
		return keyEvent{kind: keyQuit}
	}
	return keyEvent{kind: keyNone}
}

func consumeEscapeSequence(reader *bufio.Reader) {
	b, err := reader.ReadByte()
	if err != nil {
		return
	}
	if b != '[' && b != 'O' {
		return
	}
	for {
		b, err = reader.ReadByte()
		if err != nil {
			return
		}
		if b >= 0x40 && b <= 0x7E {
			return
		}
	}
}
