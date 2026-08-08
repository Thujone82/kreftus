//go:build !windows

package main

import "fmt"

// resizeToPreferred asks the terminal to adopt preferredCols × preferredRows
// via the xterm window-resize sequence (honored by many Unix terminals).
func resizeToPreferred() {
	fmt.Printf("\x1b[8;%d;%dt", preferredRows, preferredCols)
}
