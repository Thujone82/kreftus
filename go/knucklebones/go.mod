module knucklebones

go 1.24.0

toolchain go1.24.6

require (
	github.com/gdamore/tcell/v2 v2.7.4
	github.com/hajimehoshi/ebiten/v2 v2.9.9
	github.com/mattn/go-runewidth v0.0.15
	golang.org/x/image v0.31.0
	golang.org/x/term v0.17.0
)

require github.com/ebitengine/oto/v3 v3.4.0 // indirect

// Ebiten's hideconsole FreeConsoles at init; the GUI helper manages the console itself.
replace github.com/ebitengine/hideconsole => ./internal/hideconsolestub

// Conditional GLFW skip for SSH/-tui in the unified Windows exe (see internal/ebitennoinit).
replace github.com/bstkhq/go-ebiten-noinit => ./internal/ebitennoinit

require (
	github.com/bstkhq/go-ebiten-noinit v0.1.1
	github.com/ebitengine/gomobile v0.0.0-20250923094054-ea854a63cce1 // indirect
	github.com/ebitengine/hideconsole v1.0.0 // indirect
	github.com/ebitengine/purego v0.9.0 // indirect
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/go-text/typesetting v0.3.0 // indirect
	github.com/jezek/xgb v1.1.1 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)
