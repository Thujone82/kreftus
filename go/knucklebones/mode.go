package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

type uiMode int

const (
	modeTUI uiMode = iota
	modeGUI
)

var errModeConflict = errors.New("-gui and -tui cannot be used together")

// chooseMode picks the front end. interactive is true when the program was
// started from a terminal or shell rather than by double-clicking it.
func chooseMode(gui, tui, interactive bool) (uiMode, error) {
	switch {
	case gui && tui:
		return modeTUI, errModeConflict
	case tui:
		return modeTUI, nil
	case gui:
		return modeGUI, nil
	case interactive:
		return modeTUI, nil
	}
	return modeGUI, nil
}

type launchFlags struct {
	gui, tui bool
}

func parseFlags(args []string, out io.Writer) (launchFlags, error) {
	var f launchFlags
	fs := flag.NewFlagSet("Knucklebones", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.BoolVar(&f.gui, "gui", false, "open the graphical window")
	fs.BoolVar(&f.tui, "tui", false, "play in the terminal, even when started by double-click")
	fs.Usage = func() {
		fmt.Fprintln(out, "Usage: Knucklebones [-tui | -gui]")
		fmt.Fprintln(out, "Started from a terminal it plays in the terminal; double-clicked it opens a window.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return f, err
	}
	if fs.NArg() > 0 {
		err := fmt.Errorf("unexpected argument %q", fs.Arg(0))
		fmt.Fprintln(out, err)
		fs.Usage()
		return f, err
	}
	return f, nil
}
