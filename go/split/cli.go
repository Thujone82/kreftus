package main

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const cliBoxWidth = 76

type cliOptions struct {
	videoPath string
	parts     int
	hasParts  bool
	trim      bool
	crop      bool
	startSpec string
	endSpec   string
	keepSpec  string
	cutsSpec  string
	dropSpec  string
	joinMode  bool
	splitMode bool
	outPath   string
	overwrite bool
	infoOnly  bool
	help      bool
	gui       bool
}

// shouldLaunchGUI determines whether the current CLI args should open the GUI window.
func shouldLaunchGUI(args []string) (useGUI bool, preloadPath string) {
	if len(args) == 0 {
		return true, ""
	}
	if len(args) == 1 {
		a := strings.TrimSpace(args[0])
		if a == "-gui" || a == "--gui" {
			return true, ""
		}
		if a == "/?" || a == "help" || strings.HasPrefix(a, "-") {
			return false, ""
		}
		return true, a
	}
	for _, raw := range args {
		low := strings.ToLower(strings.TrimSpace(raw))
		if low == "-gui" || low == "--gui" {
			// Find first non-flag argument as video path
			for _, p := range args {
				if !strings.HasPrefix(p, "-") {
					return true, p
				}
			}
			return true, ""
		}
	}
	return false, ""
}

func runCLI(args []string, out io.Writer) int {
	opts, err := parseCLIArgs(args)
	if err != nil {
		printCLIError(out, "Command Parsing", err.Error())
		return 1
	}
	if opts.help {
		printCLIHelp(out)
		return 0
	}

	tools := FindMediaTools("", "")
	if opts.videoPath == "" {
		printCLIError(out, "Missing Input", "Missing required video path. Run with -help (-h) for usage.")
		return 1
	}
	if !tools.Ready {
		printCLIError(out, "External Tools", tools.Error)
		return 1
	}

	absVideo, err := filepath.Abs(opts.videoPath)
	if err != nil {
		absVideo = opts.videoPath
	}

	ctx := context.Background()
	sw := time.Now()

	if opts.infoOnly {
		return runCLIInfo(ctx, out, tools, absVideo)
	}

	// Determine active CLI mode
	isMultiSeg := opts.keepSpec != "" || opts.cutsSpec != "" || opts.joinMode || opts.splitMode
	isWindow := opts.trim || opts.crop

	if isMultiSeg && (isWindow || opts.hasParts) {
		printCLIError(out, "Argument Conflict", "-keep/-cuts/-join/-split cannot be combined with -trim, -crop, or -2..-10 split flags.")
		return 1
	}
	if isWindow && opts.hasParts {
		printCLIError(out, "Argument Conflict", "-trim and -crop cannot be combined with -2..-10 split flags.")
		return 1
	}
	if opts.trim && opts.crop {
		printCLIError(out, "Argument Conflict", "-trim and -crop cannot be combined.")
		return 1
	}

	media, err := ProbeMedia(ctx, tools.FFprobePath, absVideo)
	if err != nil {
		printCLIError(out, "Media Probe", err.Error())
		return 1
	}

	if isMultiSeg {
		return runCLIMultiSegment(ctx, out, tools, media, opts, sw)
	}
	if isWindow {
		return runCLIWindow(ctx, out, tools, media, opts, sw)
	}

	// Default balanced N-part split (-2..-10, -half, -thirds, -quarters)
	parts := 2
	if opts.hasParts {
		parts = opts.parts
	}
	return runCLIBalancedSplit(ctx, out, tools, media, parts, opts, sw)
}

func parseCLIArgs(args []string) (cliOptions, error) {
	var opts cliOptions
	var cropPosTimes []string
	var pathArgs []string
	var namedParts []int
	var numericParts []int

	normalizeFlag := func(s string) string {
		s = strings.ToLower(strings.TrimSpace(s))
		if strings.HasPrefix(s, "--") {
			s = "-" + s[2:]
		}
		return s
	}

	for i := 0; i < len(args); i++ {
		raw := args[i]
		low := normalizeFlag(raw)

		if low == "/?" || low == "-h" || low == "-help" {
			opts.help = true
			return opts, nil
		}
		if low == "-gui" {
			opts.gui = true
			continue
		}

		if strings.HasPrefix(low, "-") && len(low) > 1 {
			// Check numeric split flags -2..-10
			if n, err := strconv.Atoi(low[1:]); err == nil {
				if n < 2 || n > 10 {
					return opts, fmt.Errorf("unsupported numeric split flag %q (supported: -2 through -10)", raw)
				}
				numericParts = append(numericParts, n)
				continue
			}

			switch low {
			case "-half", "-two":
				namedParts = append(namedParts, 2)
			case "-thirds", "-three", "-third":
				namedParts = append(namedParts, 3)
			case "-quarters", "-quarter", "-fourth", "-fourths", "-four":
				namedParts = append(namedParts, 4)
			case "-parts":
				if i+1 >= len(args) {
					return opts, fmt.Errorf("-parts requires an integer count (2..10)")
				}
				i++
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 2 || n > 10 {
					return opts, fmt.Errorf("invalid -parts value %q (must be 2..10)", args[i])
				}
				numericParts = append(numericParts, n)
			case "-trim":
				opts.trim = true
			case "-crop":
				opts.crop = true
			case "-start", "-s":
				if i+1 >= len(args) {
					return opts, fmt.Errorf("%s requires a time value", raw)
				}
				i++
				opts.startSpec = args[i]
			case "-end", "-e":
				if i+1 >= len(args) {
					return opts, fmt.Errorf("%s requires a time value", raw)
				}
				i++
				opts.endSpec = args[i]
			case "-keep":
				if i+1 >= len(args) {
					return opts, fmt.Errorf("-keep requires ranges like 0:10-1:25,2m-end")
				}
				i++
				opts.keepSpec = args[i]
			case "-cuts":
				if i+1 >= len(args) {
					return opts, fmt.Errorf("-cuts requires comma-separated timestamps like 30s,1m15s")
				}
				i++
				opts.cutsSpec = args[i]
			case "-drop":
				if i+1 >= len(args) {
					return opts, fmt.Errorf("-drop requires comma-separated 1-based segment indices like 2,4")
				}
				i++
				opts.dropSpec = args[i]
			case "-join":
				opts.joinMode = true
			case "-split":
				opts.splitMode = true
			case "-o", "-out", "-output":
				if i+1 >= len(args) {
					return opts, fmt.Errorf("%s requires an output file or directory path", raw)
				}
				i++
				opts.outPath = args[i]
			case "-y", "-yes", "-overwrite":
				opts.overwrite = true
			case "-info":
				opts.infoOnly = true
			default:
				return opts, fmt.Errorf("unknown option %q (run with -help for usage)", raw)
			}
			continue
		}

		// Positional arg: could be a -crop timestamp if not an existing file path
		if opts.crop {
			if _, statErr := os.Stat(raw); statErr != nil {
				if _, durErr := ParseDurationSpec(raw); durErr == nil {
					cropPosTimes = append(cropPosTimes, raw)
					continue
				}
			}
		}
		pathArgs = append(pathArgs, raw)
	}

	if len(pathArgs) > 1 {
		return opts, fmt.Errorf("unexpected positional arguments: %s", strings.Join(pathArgs, " "))
	}
	if len(pathArgs) == 1 {
		opts.videoPath = pathArgs[0]
	}

	if len(namedParts)+len(numericParts) > 1 {
		return opts, fmt.Errorf("specify only one split count option (-2..-10, -half, -thirds, -quarters)")
	}
	if len(numericParts) == 1 {
		opts.parts = numericParts[0]
		opts.hasParts = true
	} else if len(namedParts) == 1 {
		opts.parts = namedParts[0]
		opts.hasParts = true
	}

	if opts.crop && len(cropPosTimes) > 0 {
		if opts.startSpec != "" || opts.endSpec != "" {
			return opts, fmt.Errorf("do not mix positional timestamps with -start (-s) or -end (-e) in -crop mode")
		}
		if len(cropPosTimes) != 2 {
			return opts, fmt.Errorf("-crop requires two timestamps: -crop <start> <end>")
		}
		opts.startSpec = cropPosTimes[0]
		opts.endSpec = cropPosTimes[1]
	}

	if opts.joinMode && opts.splitMode {
		return opts, fmt.Errorf("-join and -split cannot be combined")
	}

	return opts, nil
}

func runCLIInfo(ctx context.Context, out io.Writer, tools ToolStatus, videoPath string) int {
	media, err := ProbeMedia(ctx, tools.FFprobePath, videoPath)
	if err != nil {
		printCLIError(out, "Media Probe", err.Error())
		return 1
	}

	bar := strings.Repeat("═", cliBoxWidth-2)
	fmt.Fprintln(out)
	fmt.Fprintf(out, "╔%s╗\n", bar)
	writeBoxHeader(out, "SPLIT — MEDIA & KEYFRAME INSPECTOR")
	fmt.Fprintf(out, "╠%s╣\n", bar)
	writeBoxRow(out, "File     :", fmt.Sprintf(" %s (%s)", media.FileName, FormatBytes(media.FileSize)))
	writeBoxRow(out, "Folder   :", " "+filepath.Dir(media.FilePath))
	writeBoxRow(out, "Duration :", fmt.Sprintf(" %.3fs (%s)", media.DurationSec, FormatTimecode(media.DurationSec)))
	writeBoxRow(out, "Video    :", fmt.Sprintf(" %s %dx%d @ %.2f fps", media.VideoCodec, media.Width, media.Height, media.FPS))
	if media.HasAudio {
		writeBoxRow(out, "Audio    :", " "+media.AudioCodec)
	} else {
		writeBoxRow(out, "Audio    :", " none")
	}
	writeBoxRow(out, "Keyframes:", fmt.Sprintf(" %d keyframes", len(media.Keyframes)))
	if len(media.Keyframes) > 0 {
		var sample []string
		limit := len(media.Keyframes)
		if limit > 8 {
			limit = 8
		}
		for i := 0; i < limit; i++ {
			sample = append(sample, fmt.Sprintf("%.2fs", media.Keyframes[i]))
		}
		if len(media.Keyframes) > 8 {
			sample = append(sample, "...")
		}
		writeBoxRow(out, "First KFs:", " "+strings.Join(sample, ", "))
	}
	fmt.Fprintf(out, "╚%s╝\n\n", bar)
	return 0
}

func runCLIBalancedSplit(ctx context.Context, out io.Writer, tools ToolStatus, media *MediaInfo, parts int, opts cliOptions, sw time.Time) int {
	_, base, ext := SplitBaseExt(media.FilePath)
	outDir := filepath.Dir(media.FilePath)
	if opts.outPath != "" {
		outDir = opts.outPath
	}

	bar := strings.Repeat("═", cliBoxWidth-2)
	fmt.Fprintln(out)
	fmt.Fprintf(out, "╔%s╗\n", bar)
	writeBoxHeader(out, "LOSSLESS VIDEO SPLITTER (STREAM COPY)")
	fmt.Fprintf(out, "╠%s╣\n", bar)
	writeBoxRow(out, "Action :", fmt.Sprintf(" Split into %d equal-size parts (-c copy)", parts))
	writeBoxRow(out, "Source :", fmt.Sprintf(" %s (%s)", media.FileName, FormatBytes(media.FileSize)))
	writeBoxRow(out, "Folder :", " "+outDir)
	writeBoxRow(out, "Output :", fmt.Sprintf(" %s_1of%d%s .. %s_%dof%d%s", base, parts, ext, base, parts, parts, ext))
	fmt.Fprintf(out, "╚%s╝\n\n", bar)

	fmt.Fprintf(out, "│ [1/3] Probe  : %.2fs · %s %dx%d @ %.2ffps · %d keyframes\n",
		media.DurationSec, media.VideoCodec, media.Width, media.Height, media.FPS, len(media.Keyframes))

	res, cutLabels, err := ExportBalancedParts(ctx, tools, media, parts, outDir)
	if err != nil {
		printCLIError(out, "Lossless Split", err.Error())
		return 1
	}

	fmt.Fprintf(out, "│ [2/3] Plan   : Byte-balanced cuts at %s\n", strings.Join(cutLabels, ", "))
	fmt.Fprintf(out, "│ [3/3] Split  : OK (%d files written via lossless -c copy)\n\n", len(res.Files))

	elapsed := time.Since(sw).Seconds()
	delta := res.TotalBytes - media.FileSize

	fmt.Fprintf(out, "╔%s╗\n", bar)
	writeBoxHeader(out, "✔  LOSSLESS SPLIT COMPLETE")
	fmt.Fprintf(out, "╠%s╣\n", bar)
	writeBoxRow(out, "Status :", fmt.Sprintf(" Created %d balanced lossless parts in %.2fs (0 re-encodes)", len(res.Files), elapsed))
	writeBoxRow(out, "Input  :", fmt.Sprintf(" %s · %.2fs", FormatBytes(media.FileSize), media.DurationSec))
	writeBoxRow(out, "Total  :", fmt.Sprintf(" %s (%s vs source)", FormatBytes(res.TotalBytes), FormatSignedDelta(delta, "B")))
	writeBoxSection(out, "OUTPUT SEGMENTS")
	for _, f := range res.Files {
		label := fmt.Sprintf("%-7s:", fmt.Sprintf("%dof%d", f.Index, len(res.Files)))
		val := fmt.Sprintf(" %s · %.2fs (%d◫) · %s", f.Name, f.Dur, f.Frames, FormatBytes(f.Bytes))
		writeBoxRow(out, label, val)
	}
	fmt.Fprintf(out, "╚%s╝\n\n", bar)
	return 0
}

func runCLIWindow(ctx context.Context, out io.Writer, tools ToolStatus, media *MediaInfo, opts cliOptions, sw time.Time) int {
	var startSec, endSec float64
	var err error
	if opts.crop {
		if opts.startSpec == "" || opts.endSpec == "" {
			printCLIError(out, "Crop Options", "-crop requires TWO timestamps: -crop <start> <end> or -start <ts> -end <ts>")
			return 1
		}
		if startSec, err = ParseDurationSpec(opts.startSpec); err != nil {
			printCLIError(out, "Crop Start", err.Error())
			return 1
		}
		if endSec, err = ParseDurationSpec(opts.endSpec); err != nil {
			printCLIError(out, "Crop End", err.Error())
			return 1
		}
	} else {
		if opts.startSpec == "" && opts.endSpec == "" {
			printCLIError(out, "Trim Options", "-trim requires at least one of -start <dur> (-s) or -end <dur> (-e)")
			return 1
		}
		if opts.startSpec != "" {
			if startSec, err = ParseDurationSpec(opts.startSpec); err != nil {
				printCLIError(out, "Trim Start", err.Error())
				return 1
			}
		}
		if opts.endSpec != "" {
			if endSec, err = ParseDurationSpec(opts.endSpec); err != nil {
				printCLIError(out, "Trim End", err.Error())
				return 1
			}
		}
		if startSec+endSec <= 0 {
			printCLIError(out, "Trim Options", "At least one of -start or -end must be > 0s.")
			return 1
		}
	}

	tag := "trim"
	verb := "Trim"
	past := "Trimmed"
	if opts.crop {
		tag = "crop"
		verb = "Crop"
		past = "Cropped"
	}

	targetPath := opts.outPath
	if targetPath == "" {
		targetPath = SuggestedSuffixPath(media.FilePath, tag)
	}

	res, actualStart, actualEnd, err := ExportWindow(ctx, tools, media, opts.crop, startSec, endSec, targetPath)
	if err != nil {
		printCLIError(out, "Lossless "+verb, err.Error())
		return 1
	}

	elapsed := time.Since(sw).Seconds()
	outFile := res.Files[0]
	delta := outFile.Bytes - media.FileSize

	bar := strings.Repeat("═", cliBoxWidth-2)
	fmt.Fprintln(out)
	fmt.Fprintf(out, "╔%s╗\n", bar)
	writeBoxHeader(out, fmt.Sprintf("✔  LOSSLESS %s COMPLETE", strings.ToUpper(verb)))
	fmt.Fprintf(out, "╠%s╣\n", bar)
	writeBoxRow(out, "Status :", fmt.Sprintf(" %s [%.2fs..%.2fs] -> %.2fs in %.2fs", past, actualStart, actualEnd, outFile.Dur, elapsed))
	writeBoxRow(out, "Input  :", fmt.Sprintf(" %s · %.2fs", FormatBytes(media.FileSize), media.DurationSec))
	writeBoxRow(out, "Saved  :", fmt.Sprintf(" %s (%s vs source)", FormatBytes(outFile.Bytes), FormatSignedDelta(delta, "B")))
	writeBoxRow(out, "Output :", fmt.Sprintf(" %s (%d◫)", outFile.Name, outFile.Frames))
	fmt.Fprintf(out, "╚%s╝\n\n", bar)
	return 0
}

func runCLIMultiSegment(ctx context.Context, out io.Writer, tools ToolStatus, media *MediaInfo, opts cliOptions, sw time.Time) int {
	mode := "join"
	if opts.splitMode {
		mode = "split"
	}
	proj := NewProject(media.FilePath, media, mode)

	if opts.keepSpec != "" {
		ranges, err := ParseRangeList(opts.keepSpec, media.DurationSec)
		if err != nil {
			printCLIError(out, "Keep Ranges", err.Error())
			return 1
		}
		if err := proj.ApplyKeepRanges(ranges); err != nil {
			printCLIError(out, "Keep Ranges", err.Error())
			return 1
		}
	} else if opts.cutsSpec != "" {
		for _, tok := range strings.Split(opts.cutsSpec, ",") {
			tok = strings.TrimSpace(tok)
			if tok == "" {
				continue
			}
			sec, err := ParseDurationSpec(tok)
			if err != nil {
				printCLIError(out, "Cut Timestamp", err.Error())
				return 1
			}
			proj.AddMarkerAt(sec)
		}
	}

	if opts.dropSpec != "" {
		for _, tok := range strings.Split(opts.dropSpec, ",") {
			tok = strings.TrimSpace(tok)
			if tok == "" {
				continue
			}
			idx, err := strconv.Atoi(tok)
			if err != nil || idx < 1 || idx > len(proj.KeepFlags) {
				printCLIError(out, "Drop Index", fmt.Sprintf("invalid segment index %q (valid: 1..%d)", tok, len(proj.KeepFlags)))
				return 1
			}
			proj.KeepFlags[idx-1] = false
		}
	}

	kept := proj.KeptSegments()
	if len(kept) == 0 {
		printCLIError(out, "Segment Selection", "No kept segments remain after applying -drop / -keep.")
		return 1
	}

	var res *ExportResult
	var err error
	if mode == "join" {
		target := opts.outPath
		if target == "" {
			target = SuggestedJoinPath(media.FilePath)
		}
		res, err = ExportJoin(ctx, tools, proj, target, func(p ExportProgress) {
			fmt.Fprintf(out, "│ [%d/%d] %s\n", p.Current, p.Total, p.Phase)
		})
	} else {
		baseTarget := opts.outPath
		if baseTarget == "" {
			baseTarget = SuggestedSplitDialogPath(media.FilePath)
		} else if st, statErr := os.Stat(baseTarget); statErr == nil && st.IsDir() {
			_, base, ext := SplitBaseExt(media.FilePath)
			baseTarget = filepath.Join(baseTarget, base+"_01"+ext)
		}
		targets := DeriveSplitPaths(baseTarget, len(kept))
		res, err = ExportSplit(ctx, tools, proj, targets, func(p ExportProgress) {
			fmt.Fprintf(out, "│ [%d/%d] %s -> %s\n", p.Current, p.Total, p.Phase, p.Label)
		})
	}
	if err != nil {
		printCLIError(out, "Export "+strings.ToUpper(mode), err.Error())
		return 1
	}

	elapsed := time.Since(sw).Seconds()
	delta := res.TotalBytes - media.FileSize
	bar := strings.Repeat("═", cliBoxWidth-2)
	fmt.Fprintln(out)
	fmt.Fprintf(out, "╔%s╗\n", bar)
	writeBoxHeader(out, fmt.Sprintf("✔  LOSSLESS %s COMPLETE", strings.ToUpper(mode)))
	fmt.Fprintf(out, "╠%s╣\n", bar)
	writeBoxRow(out, "Status :", fmt.Sprintf(" Exported %d kept file(s) in %.2fs (0 re-encodes)", len(res.Files), elapsed))
	writeBoxRow(out, "Input  :", fmt.Sprintf(" %s · %.2fs", FormatBytes(media.FileSize), media.DurationSec))
	writeBoxRow(out, "Total  :", fmt.Sprintf(" %s (%s vs source)", FormatBytes(res.TotalBytes), FormatSignedDelta(delta, "B")))
	writeBoxSection(out, "OUTPUT FILES")
	for _, f := range res.Files {
		label := fmt.Sprintf("#%02d    :", f.Index)
		val := fmt.Sprintf(" %s · %.2fs (%d◫) · %s", f.Name, f.Dur, f.Frames, FormatBytes(f.Bytes))
		writeBoxRow(out, label, val)
	}
	fmt.Fprintf(out, "╚%s╝\n\n", bar)
	return 0
}

func printCLIHelp(out io.Writer) {
	tools := FindMediaTools("", "")
	bar := strings.Repeat("═", cliBoxWidth-2)
	fmt.Fprintln(out)
	fmt.Fprintf(out, "╔%s╗\n", bar)
	writeBoxHeader(out, "SPLIT — LOSSLESS VIDEO SPLITTER, JOINER & TRIMMER")
	fmt.Fprintf(out, "╠%s╣\n", bar)
	writeBoxRow(out, "GUI Mode :", " split.exe  (or split.exe <video> to open pre-loaded)")
	writeBoxRow(out, "Split    :", " split.exe <video> [-2..-10|-half|-thirds|-quarters]")
	writeBoxRow(out, "Trim     :", " split.exe <video> -trim [-start <dur>] [-end <dur>]")
	writeBoxRow(out, "Crop     :", " split.exe <video> -crop <start> <end>  (or -start/-end)")
	writeBoxRow(out, "Segments :", " split.exe <video> -keep <r1,r2> [-join|-split] [-o <path>]")
	writeBoxRow(out, "Cuts     :", " split.exe <video> -cuts <t1,t2> [-drop <idx>] [-join|-split]")
	writeBoxRow(out, "Inspect  :", " split.exe <video> -info")

	writeBoxSection(out, "TIME & RANGE FORMATS")
	writeBoxRow(out, "Times    :", " 1m3.2s, 1h2m3s, 45s, 2m, 1:03.2, 0:01:03.2, or 63.2")
	writeBoxRow(out, "Ranges   :", " 0:10-1:25,2m-end  or  start..45s,1m30s..2m")
	writeBoxRow(out, "Keyframe :", " All cut/start points snap cleanly to nearest keyframe (-c copy)")

	writeBoxSection(out, "EXTERNAL TOOLS")
	if tools.FFmpegPath != "" {
		writeBoxRow(out, "ffmpeg   :", fmt.Sprintf(" OK [%s] %s", tools.FFmpegVersion, tools.FFmpegPath))
	} else {
		writeBoxRow(out, "ffmpeg   :", " MISSING (winget install Gyan.FFmpeg)")
	}
	if tools.FFprobePath != "" {
		writeBoxRow(out, "ffprobe  :", fmt.Sprintf(" OK [%s] %s", tools.FFprobeVersion, tools.FFprobePath))
	} else {
		writeBoxRow(out, "ffprobe  :", " MISSING (winget install Gyan.FFmpeg)")
	}
	fmt.Fprintf(out, "╚%s╝\n\n", bar)
}

func printCLIError(out io.Writer, action, msg string) {
	bar := strings.Repeat("═", cliBoxWidth-2)
	fmt.Fprintln(out)
	fmt.Fprintf(out, "╔%s╗\n", bar)
	writeBoxHeader(out, "✖  ERROR: "+action)
	fmt.Fprintf(out, "╠%s╣\n", bar)
	for _, line := range wrapWords(msg, cliBoxWidth-14) {
		writeBoxRow(out, "Details:", " "+line)
	}
	fmt.Fprintf(out, "╚%s╝\n\n", bar)
}

func writeBoxHeader(out io.Writer, title string) {
	inner := cliBoxWidth - 2
	n := utf8.RuneCountInString(title)
	if n >= inner {
		title = truncateRunes(title, inner)
		n = utf8.RuneCountInString(title)
	}
	left := (inner - n) / 2
	right := inner - n - left
	fmt.Fprintf(out, "║%s%s%s║\n", strings.Repeat(" ", left), title, strings.Repeat(" ", right))
}

func writeBoxSection(out io.Writer, title string) {
	tag := "─── [ " + title + " ] "
	rem := (cliBoxWidth - 2) - utf8.RuneCountInString(tag)
	if rem < 0 {
		rem = 0
	}
	fmt.Fprintf(out, "╟%s%s╢\n", tag, strings.Repeat("─", rem))
}

func writeBoxRow(out io.Writer, label, val string) {
	contentW := cliBoxWidth - 4
	lw := utf8.RuneCountInString(label)
	vw := utf8.RuneCountInString(val)
	if lw+vw > contentW {
		avail := contentW - lw
		if avail > 3 {
			val = truncateRunes(val, avail-3) + "..."
		} else if avail >= 0 {
			val = truncateRunes(val, avail)
		}
		vw = utf8.RuneCountInString(val)
	}
	pad := contentW - lw - vw
	if pad < 0 {
		pad = 0
	}
	fmt.Fprintf(out, "║ %s%s%s ║\n", label, val, strings.Repeat(" ", pad))
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	var b strings.Builder
	count := 0
	for _, r := range s {
		if count >= max {
			break
		}
		b.WriteRune(r)
		count++
	}
	return b.String()
}

func wrapWords(s string, width int) []string {
	if width <= 10 || len(s) <= width {
		return []string{s}
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		if len(cur)+1+len(w) <= width {
			cur += " " + w
		} else {
			lines = append(lines, cur)
			cur = w
		}
	}
	lines = append(lines, cur)
	return lines
}

// Keep unused math import guard clean if future rounding tweaks are added
var _ = math.Abs
var _ = sort.Float64s
