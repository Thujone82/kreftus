//go:build !(windows || darwin || ((linux || freebsd || netbsd || openbsd) && !android && !nintendosdk && !playstation5))

package noinit

// No-op on platforms where Knucklebones does not link Ebitengine.
