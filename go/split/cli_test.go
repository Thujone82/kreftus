package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShouldLaunchGUI(t *testing.T) {
	if useGUI, _ := shouldLaunchGUI(nil); !useGUI {
		t.Fatalf("expected 0 args to launch GUI")
	}
	if useGUI, preload := shouldLaunchGUI([]string{`C:\clip.mp4`}); !useGUI || preload != `C:\clip.mp4` {
		t.Fatalf("expected single file arg to launch GUI with preload, got (%v, %q)", useGUI, preload)
	}
	if useGUI, _ := shouldLaunchGUI([]string{`C:\clip.mp4`, `-2`}); useGUI {
		t.Fatalf("expected -2 flag to run CLI mode")
	}
	if useGUI, _ := shouldLaunchGUI([]string{`-help`}); useGUI {
		t.Fatalf("expected -help to run CLI mode")
	}
}

func TestCLIHelpAndActions(t *testing.T) {
	var buf bytes.Buffer
	if code := runCLI([]string{"-help"}, &buf); code != 0 {
		t.Fatalf("help exit code = %d", code)
	}
	if !strings.Contains(buf.String(), "SPLIT") {
		t.Fatalf("unexpected help output: %s", buf.String())
	}

	tools := FindMediaTools("", "")
	if !tools.Ready {
		t.Skip("ffmpeg/ffprobe not available")
	}

	srcPath := createSyntheticTestVideo(t, tools, "cli_src.mp4")

	// Balanced -2 split
	buf.Reset()
	if code := runCLI([]string{srcPath, "-2"}, &buf); code != 0 {
		t.Fatalf("-2 exit code = %d, out=%s", code, buf.String())
	}
	dir, base, ext := SplitBaseExt(srcPath)
	for _, name := range []string{base + "_1of2" + ext, base + "_2of2" + ext} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing balanced output %s: %v", name, err)
		}
	}

	// Trim mode
	buf.Reset()
	if code := runCLI([]string{srcPath, "-trim", "-s", "1s", "-e", "1s"}, &buf); code != 0 {
		t.Fatalf("-trim exit code = %d, out=%s", code, buf.String())
	}
	if _, err := os.Stat(filepath.Join(dir, base+"_trim"+ext)); err != nil {
		t.Fatalf("missing trim output: %v", err)
	}

	// Crop mode
	buf.Reset()
	if code := runCLI([]string{srcPath, "-crop", "1s", "4s"}, &buf); code != 0 {
		t.Fatalf("-crop exit code = %d, out=%s", code, buf.String())
	}
	if _, err := os.Stat(filepath.Join(dir, base+"_crop"+ext)); err != nil {
		t.Fatalf("missing crop output: %v", err)
	}

	// Multi-segment -keep + -join -> _edit
	buf.Reset()
	if code := runCLI([]string{srcPath, "-keep", "0s-2s,4s-end", "-join"}, &buf); code != 0 {
		t.Fatalf("-keep -join exit code = %d, out=%s", code, buf.String())
	}
	if _, err := os.Stat(filepath.Join(dir, base+"_edit"+ext)); err != nil {
		t.Fatalf("missing join output: %v", err)
	}

	// Multi-segment -cuts + -drop + -split -> _01, _02
	buf.Reset()
	if code := runCLI([]string{srcPath, "-cuts", "2s,4s", "-drop", "2", "-split"}, &buf); code != 0 {
		t.Fatalf("-cuts -split exit code = %d, out=%s", code, buf.String())
	}
	for _, name := range []string{base + "_01" + ext, base + "_02" + ext} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing split output %s: %v", name, err)
		}
	}
}
