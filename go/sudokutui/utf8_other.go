//go:build !windows

package main

func enableUTF8Console() {
	// Most Unix terminals already speak UTF-8.
}
