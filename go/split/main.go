package main

import (
	"fmt"
	"os"
)

func main() {
	useGUI, preloadPath := shouldLaunchGUI(os.Args[1:])
	if useGUI {
		if err := runGUI(preloadPath); err != nil {
			showFatalError("Split", err.Error())
			os.Exit(1)
		}
		return
	}

	initCLIConsole()
	code := runCLI(os.Args[1:], os.Stdout)
	_ = os.Stdout.Sync()
	_ = os.Stderr.Sync()
	if code != 0 {
		os.Exit(code)
	}
}

func showFatalError(title, msg string) {
	if !showMessageBoxError(title, msg) {
		fmt.Fprintf(os.Stderr, "%s: %s\n", title, msg)
	}
}
