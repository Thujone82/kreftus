// Package noinit optionally skips Ebitengine's GLFW UI initializer.
//
// Knucklebones links Ebitengine for the local GUI, but Windows OpenSSH runs in
// Session 0 where CreateWindowEx fails with ERROR_REQUIRES_INTERACTIVE_WINDOWSTATION
// during Ebitengine's package init — before main can honor -tui. Importing this
// package replaces that initializer with a non-graphical stub when the process
// is clearly headless (-tui, SSH, or a non-interactive window station). On a
// normal desktop session the real initializer still runs so -gui works.
//
// Based on github.com/bstkhq/go-ebiten-noinit (MIT).
package noinit
