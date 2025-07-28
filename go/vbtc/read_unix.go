//go:build !windows

package main

import (
	"errors"
	"os"
	"time"
)

// cancellableRead attempts to read a single byte from stdin in a non-blocking,
// cancellable way that is safe for Unix-like systems. It uses SetReadDeadline
// to poll for input, allowing it to also check the 'done' channel.
func cancellableRead(done chan struct{}) (byte, error) {
	for {
		select {
		case <-done:
			return 0, os.ErrClosed
		default:
			os.Stdin.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			buffer := make([]byte, 1)
			n, err := os.Stdin.Read(buffer)

			if err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					continue // Timeout is expected, just loop again.
				}
				return 0, err
			}
			if n > 0 {
				return buffer[0], nil
			}
		}
	}
}
