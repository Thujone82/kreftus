// Package hideconsole is a no-op stand-in for github.com/ebitengine/hideconsole.
// Ebiten blank-imports that module and FreeConsoles at init, which breaks
// Windows SSH/ConPTY sessions. Knucklebones detaches the console itself when
// opening the GUI from a double-click.
package hideconsole
