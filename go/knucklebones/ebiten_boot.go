//go:build windows || darwin || (linux && cgo)

package main

// Skip Ebitengine's GLFW package-init on SSH / -tui so the unified exe can
// run the terminal UI without CreateWindowEx failing in Session 0.
import _ "github.com/bstkhq/go-ebiten-noinit"
