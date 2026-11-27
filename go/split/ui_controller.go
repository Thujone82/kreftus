//go:build windows || darwin || linux

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// UISnapshot is the complete state sent to the WebView frontend.
type UISnapshot struct {
	Platform         string          `json:"platform"`
	Theme            string          `json:"theme"`
	Mode             string          `json:"mode"`
	SkipDropped      bool            `json:"skipDropped"`
	ShowAudio        bool            `json:"showAudio"`
	ShowVisual       bool            `json:"showVisual"`
	TimelineScale    float64         `json:"timelineScale"`
	Tools            ToolStatus      `json:"tools"`
	Loading          bool            `json:"loading"`
	LoadingFile      string          `json:"loadingFile"`
	LoadingPct       float64         `json:"loadingPct"`
	LoadingPhase     string          `json:"loadingPhase"`
	LoadError        string          `json:"loadError"`
	LoadSeq          int64           `json:"loadSeq"`
	FilePath         string          `json:"filePath"`
	VideoURL         string          `json:"videoUrl"`
	RemuxURL         string          `json:"remuxUrl"`
	ScrubURL         string          `json:"scrubUrl"`
	Media            *MediaInfo      `json:"media"`
	Waveform         *AudioWaveform  `json:"waveform,omitempty"`
	WaveformLoading  bool            `json:"waveformLoading"`
	Visual           *VisualDelta    `json:"visual,omitempty"`
	VisualLoading    bool            `json:"visualLoading"`
	KeyframesLoading bool            `json:"keyframesLoading"`
	Markers          []float64       `json:"markers"`
	Segments         []Segment       `json:"segments"`
	OutputPreview    []string        `json:"outputPreview"`
	Exporting        bool            `json:"exporting"`
	ExportProgress   *ExportProgress `json:"exportProgress,omitempty"`
	LastExport       *ExportResult   `json:"lastExport,omitempty"`
	LastExportSec    float64         `json:"lastExportSec,omitempty"`
}

type MarkerActionResult struct {
	SnappedSec float64    `json:"snappedSec"`
	Changed    bool       `json:"changed"`
	Snapshot   UISnapshot `json:"snapshot"`
}

type rpcRequest struct {
	ID     int               `json:"id"`
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
}

type rpcResponse struct {
	ID     int    `json:"id"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

type uiHost interface {
	DispatchUI(fn func())
	EvalJS(script string)
	SetWindowTitle(title string)
	ApplyTheme(theme string)
	OpenVideoDialog(initialDir string) (string, bool)
	OpenExecutableDialog(title, initialDir string) (string, bool)
	SaveVideoDialog(title, initialDir, defaultName, ext string, promptOverwrite bool) (string, bool)
	ConfirmOverwriteDialog(title, message string) bool
	RevealInFileManager(path string)
}

type splitApp struct {
	host     uiHost
	srv      *mediaServer
	settings *SettingsStore

	mu               sync.Mutex
	tools            ToolStatus
	proj             *ProjectState
	waveform         *AudioWaveform
	waveformLoading  bool
	visual           *VisualDelta
	visualLoading    bool
	keyframesLoading bool
	loading          bool
	loadingFile      string
	loadingPct       float64
	loadingPhase     string
	loadError        string
	loadSeq          int64
	probeCancel      context.CancelFunc
	loadCtx          context.Context
	loadCancel       context.CancelFunc
	audioCancel      context.CancelFunc
	audioRunID       int64
	visualCancel     context.CancelFunc
	visualRunID      int64
	keyframesCancel  context.CancelFunc
	keyframesRunID   int64
	exporting        bool
	exportProg       *ExportProgress
	exportCancel     context.CancelFunc
	lastExport       *ExportResult
	lastExportSec    float64
	bindings         map[string]any
}

func newSplitApp(srv *mediaServer, settings *SettingsStore, tools ToolStatus, host uiHost) *splitApp {
	app := &splitApp{
		host:     host,
		srv:      srv,
		settings: settings,
		tools:    tools,
		bindings: make(map[string]any),
	}
	app.registerBindings()
	if srv != nil {
		srv.SetRPCHandler(app.handleHTTPRPC)
	}
	return app
}

func (a *splitApp) runOnUIThreadSync(fn func()) {
	done := make(chan struct{})
	a.host.DispatchUI(func() {
		defer close(done)
		fn()
	})
	<-done
}

func (a *splitApp) evalJS(script string) {
	if a.host != nil {
		a.host.EvalJS(script)
	}
}

func (a *splitApp) pushState() {
	snap := a.snapshot()
	raw, err := json.Marshal(snap)
	if err != nil {
		return
	}
	a.evalJS("window.__onSplitState && window.__onSplitState(" + string(raw) + ");")
}

func (a *splitApp) pushProgress(p ExportProgress) {
	raw, err := json.Marshal(p)
	if err != nil {
		return
	}
	a.evalJS("window.__onExportProgress && window.__onExportProgress(" + string(raw) + ");")
}

func (a *splitApp) pushWaveform(seq int64, wf *AudioWaveform) {
	payload := struct {
		Seq      int64          `json:"seq"`
		Waveform *AudioWaveform `json:"waveform"`
	}{Seq: seq, Waveform: wf}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	a.evalJS("window.__onSplitWaveform && window.__onSplitWaveform(" + string(raw) + ");")
}

func (a *splitApp) pushVisual(seq int64, vd *VisualDelta) {
	payload := struct {
		Seq    int64        `json:"seq"`
		Visual *VisualDelta `json:"visual"`
	}{Seq: seq, Visual: vd}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	a.evalJS("window.__onSplitVisual && window.__onSplitVisual(" + string(raw) + ");")
}

func (a *splitApp) pushLoadProgress(pct float64, kfs int, phase string) {
	payload := struct {
		Pct   float64 `json:"pct"`
		KFs   int     `json:"kfs"`
		Phase string  `json:"phase"`
	}{Pct: pct, KFs: kfs, Phase: phase}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	a.evalJS("window.__onLoadProgress && window.__onLoadProgress(" + string(raw) + ");")
}

func (a *splitApp) pushLoadBackdrop(seq int64, dataURI string, kfSec float64) {
	payload := struct {
		Seq     int64   `json:"seq"`
		DataURI string  `json:"dataUri"`
		KFSec   float64 `json:"kfSec"`
	}{Seq: seq, DataURI: dataURI, KFSec: kfSec}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	a.evalJS("window.__onLoadBackdrop && window.__onLoadBackdrop(" + string(raw) + ");")
}

func (a *splitApp) pushAnalysisProgress(seq int64, kind string, pct float64) {
	payload := struct {
		Seq  int64   `json:"seq"`
		Kind string  `json:"kind"`
		Pct  float64 `json:"pct"`
	}{Seq: seq, Kind: kind, Pct: pct}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	a.evalJS("window.__onAnalysisProgress && window.__onAnalysisProgress(" + string(raw) + ");")
}

func (a *splitApp) cancelAnalysisWorkersLocked() {
	if a.keyframesCancel != nil {
		a.keyframesCancel()
		a.keyframesCancel = nil
	}
	if a.audioCancel != nil {
		a.audioCancel()
		a.audioCancel = nil
	}
	if a.visualCancel != nil {
		a.visualCancel()
		a.visualCancel = nil
	}
	a.keyframesLoading = false
	a.waveformLoading = false
	a.visualLoading = false
}

func (a *splitApp) cancelLoadAndExport() {
	a.mu.Lock()
	if a.probeCancel != nil {
		a.probeCancel()
		a.probeCancel = nil
	}
	a.cancelAnalysisWorkersLocked()
	if a.loadCancel != nil {
		a.loadCancel()
		a.loadCancel = nil
	}
	if a.exportCancel != nil {
		a.exportCancel()
		a.exportCancel = nil
	}
	a.mu.Unlock()
}

func (a *splitApp) snapshot() UISnapshot {
	cfg := a.settings.Get()
	a.mu.Lock()
	defer a.mu.Unlock()

	snap := UISnapshot{
		Platform:         runtime.GOOS,
		Theme:            cfg.Theme,
		Mode:             cfg.ExportMode,
		SkipDropped:      cfg.SkipDropped,
		ShowAudio:        cfg.ShowAudio,
		ShowVisual:       cfg.ShowVisual,
		TimelineScale:    cfg.TimelineScale,
		Tools:            a.tools,
		Loading:          a.loading,
		LoadingFile:      a.loadingFile,
		LoadingPct:       a.loadingPct,
		LoadingPhase:     a.loadingPhase,
		LoadError:        a.loadError,
		LoadSeq:          a.loadSeq,
		Waveform:         a.waveform,
		WaveformLoading:  a.waveformLoading,
		Visual:           a.visual,
		VisualLoading:    a.visualLoading,
		KeyframesLoading: a.keyframesLoading,
		Exporting:        a.exporting,
		ExportProgress:   a.exportProg,
		LastExport:       a.lastExport,
		LastExportSec:    a.lastExportSec,
	}

	if a.proj != nil && a.proj.Media != nil {
		snap.Mode = a.proj.Mode
		snap.FilePath = a.proj.FilePath
		snap.Media = a.proj.Media
		snap.Markers = append([]float64(nil), a.proj.Markers...)
		snap.Segments = a.proj.Segments()
		seqStr := strconv.FormatInt(a.loadSeq, 10)
		snap.VideoURL = a.srv.baseURL + "/video?v=" + seqStr
		snap.RemuxURL = a.srv.baseURL + "/remux?v=" + seqStr
		snap.ScrubURL = a.srv.baseURL + "/scrub?v=" + seqStr

		kept := a.proj.KeptSegments()
		if len(kept) > 0 {
			if a.proj.Mode == "join" {
				snap.OutputPreview = []string{filepath.Base(SuggestedJoinPath(a.proj.FilePath))}
			} else {
				paths := DeriveSplitPaths(SuggestedSplitDialogPath(a.proj.FilePath), len(kept))
				for _, p := range paths {
					snap.OutputPreview = append(snap.OutputPreview, filepath.Base(p))
				}
			}
		}
	}
	return snap
}

func (a *splitApp) unloadMediaAndUnlock() {
	a.mu.Lock()
	if a.probeCancel != nil {
		a.probeCancel()
		a.probeCancel = nil
	}
	a.cancelAnalysisWorkersLocked()
	if a.loadCancel != nil {
		a.loadCancel()
		a.loadCancel = nil
	}
	a.proj = nil
	a.waveform = nil
	a.visual = nil
	a.lastExport = nil
	a.lastExportSec = 0
	a.loadSeq++
	a.mu.Unlock()

	a.srv.ClearActiveVideo()
	a.host.SetWindowTitle("Split")
}

func (a *splitApp) loadFileAsync(rawPath string) {
	absPath, err := filepath.Abs(strings.TrimSpace(rawPath))
	if err != nil {
		absPath = rawPath
	}

	// Release any previously open file handles, background analysis, and active HTTP streams immediately.
	a.srv.ClearActiveVideo()

	a.mu.Lock()
	if a.probeCancel != nil {
		a.probeCancel()
	}
	a.cancelAnalysisWorkersLocked()
	if a.loadCancel != nil {
		a.loadCancel()
		a.loadCancel = nil
	}
	probeCtx, probeCancel := context.WithCancel(context.Background())
	a.probeCancel = probeCancel
	a.proj = nil
	a.waveform = nil
	a.visual = nil
	a.loadSeq++
	probeSeq := a.loadSeq
	a.loading = true
	a.loadingFile = filepath.Base(absPath)
	a.loadingPct = 2
	a.loadingPhase = "Opening video…"
	a.loadError = ""
	a.lastExport = nil
	tools := FindMediaTools(a.settings.Get().FFmpegPath, a.settings.Get().FFprobePath)
	a.tools = tools
	a.mu.Unlock()

	a.host.SetWindowTitle("Split")
	a.srv.SetFFmpegPath(tools.FFmpegPath)
	a.pushState()

	go func() {
		defer probeCancel()
		if !tools.Ready {
			a.mu.Lock()
			a.loading = false
			a.loadingPct = 0
			a.loadingPhase = ""
			a.loadError = tools.Error
			a.mu.Unlock()
			a.pushState()
			return
		}

		var backdropBusy atomic.Bool
		var backdropMu sync.Mutex
		var lastBackdropAt time.Time
		lastBackdropKF := -1.0

		maybeEmitBackdrop := func(kfSec float64) {
			if probeCtx.Err() != nil {
				return
			}
			backdropMu.Lock()
			if !lastBackdropAt.IsZero() && time.Since(lastBackdropAt) < 1800*time.Millisecond {
				backdropMu.Unlock()
				return
			}
			if lastBackdropKF >= 0 && math.Abs(kfSec-lastBackdropKF) < 0.05 {
				backdropMu.Unlock()
				return
			}
			if !backdropBusy.CompareAndSwap(false, true) {
				backdropMu.Unlock()
				return
			}
			lastBackdropAt = time.Now()
			lastBackdropKF = kfSec
			backdropMu.Unlock()

			go func(targetKF float64) {
				defer backdropBusy.Store(false)
				uri, bErr := ExtractProbeBackdropDataURI(probeCtx, tools.FFmpegPath, absPath, targetKF)
				if bErr != nil || uri == "" || probeCtx.Err() != nil {
					return
				}
				a.mu.Lock()
				active := a.loading && a.loadSeq == probeSeq
				a.mu.Unlock()
				if active {
					a.pushLoadBackdrop(probeSeq, uri, targetKF)
				}
			}(kfSec)
		}

		maybeEmitBackdrop(0)

		media, err := ProbeMediaWithProgress(probeCtx, tools.FFprobePath, absPath, func(pct float64, kfs int, latestKFSec float64, phase string) {
			if probeCtx.Err() != nil {
				return
			}
			a.mu.Lock()
			stillLoading := a.loading
			if stillLoading {
				a.loadingPct = pct
				a.loadingPhase = phase
			}
			a.mu.Unlock()
			if stillLoading {
				a.pushLoadProgress(pct, kfs, phase)
				if kfs > 0 {
					maybeEmitBackdrop(latestKFSec)
				}
			}
		})
		if probeCtx.Err() != nil {
			return
		}
		if err != nil {
			a.mu.Lock()
			a.loading = false
			a.loadingPct = 0
			a.loadingPhase = ""
			a.loadError = err.Error()
			a.mu.Unlock()
			a.pushState()
			return
		}

		cfg := a.settings.Get()
		a.settings.Update(func(s *Settings) {
			s.LastDir = filepath.Dir(absPath)
		})
		a.srv.SetActiveVideo(absPath, media.VideoCodec)

		a.mu.Lock()
		a.cancelAnalysisWorkersLocked()
		if a.loadCancel != nil {
			a.loadCancel()
		}
		loadCtx, loadCancel := context.WithCancel(context.Background())
		a.loadCtx = loadCtx
		a.loadCancel = loadCancel
		a.probeCancel = nil
		a.loadSeq++
		a.loading = false
		a.loadingPct = 100
		a.loadingPhase = ""
		a.loadError = ""
		a.proj = NewProject(absPath, media, cfg.ExportMode)
		a.waveform = nil
		a.visual = nil

		var startAudio func()
		if !media.HasAudio {
			a.waveform = &AudioWaveform{Ready: true, HasAudio: false}
		} else if cfg.ShowAudio {
			startAudio = a.prepareAudioJobLocked()
		}
		var startVisual func()
		if cfg.ShowVisual {
			startVisual = a.prepareVisualJobLocked()
		}
		a.mu.Unlock()

		a.host.SetWindowTitle("Split — " + media.FileName)
		a.pushState()

		if startAudio != nil {
			go startAudio()
		}
		if startVisual != nil {
			go startVisual()
		}
	}()
}

// prepareAudioJobLocked cancels any prior audio analysis worker and prepares a fresh job. Caller holds a.mu.
func (a *splitApp) prepareAudioJobLocked() func() {
	if a.proj == nil || a.proj.Media == nil || !a.proj.Media.HasAudio || !a.tools.Ready || a.loadCtx == nil {
		return nil
	}
	if a.audioCancel != nil {
		a.audioCancel()
	}
	ctx, cancel := context.WithCancel(a.loadCtx)
	a.audioCancel = cancel
	a.audioRunID++
	runID := a.audioRunID
	seq := a.loadSeq
	tools := a.tools
	absPath := a.proj.FilePath
	media := a.proj.Media
	a.waveform = nil
	a.waveformLoading = true
	return func() {
		defer cancel()
		a.startAudioAnalysis(ctx, seq, runID, tools, absPath, media)
	}
}

// prepareVisualJobLocked cancels any prior visual delta worker and prepares a fresh job. Caller holds a.mu.
func (a *splitApp) prepareVisualJobLocked() func() {
	if a.proj == nil || a.proj.Media == nil || !a.tools.Ready || a.loadCtx == nil {
		return nil
	}
	if a.visualCancel != nil {
		a.visualCancel()
	}
	ctx, cancel := context.WithCancel(a.loadCtx)
	a.visualCancel = cancel
	a.visualRunID++
	runID := a.visualRunID
	seq := a.loadSeq
	tools := a.tools
	media := a.proj.Media
	a.visual = nil
	a.visualLoading = true
	return func() {
		defer cancel()
		a.startVisualAnalysis(ctx, seq, runID, tools, media)
	}
}

// prepareKeyframesJobLocked cancels any prior keyframe refresh worker and prepares a non-destructive re-probe. Caller holds a.mu.
func (a *splitApp) prepareKeyframesJobLocked() func() {
	if a.proj == nil || a.proj.Media == nil || !a.tools.Ready || a.loadCtx == nil {
		return nil
	}
	if a.keyframesCancel != nil {
		a.keyframesCancel()
	}
	ctx, cancel := context.WithCancel(a.loadCtx)
	a.keyframesCancel = cancel
	a.keyframesRunID++
	runID := a.keyframesRunID
	seq := a.loadSeq
	tools := a.tools
	absPath := a.proj.FilePath
	a.keyframesLoading = true
	return func() {
		defer cancel()
		a.startKeyframesRefresh(ctx, seq, runID, tools, absPath)
	}
}

func (a *splitApp) startAudioAnalysis(ctx context.Context, seq, runID int64, tools ToolStatus, absPath string, media *MediaInfo) {
	wf, wfErr := ExtractAudioWaveformWithProgress(ctx, tools.FFmpegPath, absPath, media.DurationSec, media.HasAudio, func(pct float64) {
		if ctx.Err() != nil {
			return
		}
		a.mu.Lock()
		ok := a.loadSeq == seq && a.audioRunID == runID && a.waveformLoading
		a.mu.Unlock()
		if ok {
			a.pushAnalysisProgress(seq, "audio", pct)
		}
	})
	if ctx.Err() != nil {
		return
	}
	if wfErr != nil {
		wf = &AudioWaveform{Ready: true, HasAudio: false}
	}
	a.mu.Lock()
	stillCurrent := a.loadSeq == seq && a.audioRunID == runID
	if stillCurrent {
		a.waveform = wf
		a.waveformLoading = false
	}
	a.mu.Unlock()
	if stillCurrent {
		a.pushWaveform(seq, wf)
	}
}

func (a *splitApp) startVisualAnalysis(ctx context.Context, seq, runID int64, tools ToolStatus, media *MediaInfo) {
	vd, vdErr := ExtractVisualDeltaWithProgress(ctx, tools.FFmpegPath, media, func(pct float64) {
		if ctx.Err() != nil {
			return
		}
		a.mu.Lock()
		ok := a.loadSeq == seq && a.visualRunID == runID && a.visualLoading
		a.mu.Unlock()
		if ok {
			a.pushAnalysisProgress(seq, "visual", pct)
		}
	})
	if ctx.Err() != nil {
		return
	}
	if vdErr != nil {
		vd = &VisualDelta{Ready: true}
	}
	a.mu.Lock()
	stillCurrent := a.loadSeq == seq && a.visualRunID == runID
	if stillCurrent {
		a.visual = vd
		a.visualLoading = false
	}
	a.mu.Unlock()
	if stillCurrent {
		a.pushVisual(seq, vd)
	}
}

func (a *splitApp) startKeyframesRefresh(ctx context.Context, seq, runID int64, tools ToolStatus, absPath string) {
	media, err := ProbeMediaWithProgress(ctx, tools.FFprobePath, absPath, func(pct float64, kfs int, latestKFSec float64, phase string) {
		if ctx.Err() != nil {
			return
		}
		a.mu.Lock()
		ok := a.loadSeq == seq && a.keyframesRunID == runID && a.keyframesLoading
		a.mu.Unlock()
		if ok {
			a.pushAnalysisProgress(seq, "keyframes", pct)
		}
	})
	if ctx.Err() != nil {
		return
	}
	a.mu.Lock()
	stillCurrent := a.loadSeq == seq && a.keyframesRunID == runID
	if stillCurrent {
		a.keyframesLoading = false
		if err == nil && media != nil && a.proj != nil {
			a.proj.UpdateMediaPreservingCuts(media)
		} else if err != nil {
			a.loadError = err.Error()
		}
	}
	a.mu.Unlock()
	if stillCurrent {
		a.pushState()
		if err != nil {
			a.mu.Lock()
			a.loadError = ""
			a.mu.Unlock()
		}
	}
}

// --- RPC Bridge ---

func defaultToolBrowseDir() string {
	switch runtime.GOOS {
	case "windows":
		return `C:\`
	case "darwin":
		if st, err := os.Stat("/opt/homebrew/bin"); err == nil && st.IsDir() {
			return "/opt/homebrew/bin"
		}
		return "/usr/local/bin"
	default:
		if st, err := os.Stat("/usr/local/bin"); err == nil && st.IsDir() {
			return "/usr/local/bin"
		}
		return "/usr/bin"
	}
}

func (a *splitApp) registerBindings() {
	a.bind("getState", func() (UISnapshot, error) {
		return a.snapshot(), nil
	})
	a.bind("openFileDialog", func() (bool, error) {
		cfg := a.settings.Get()
		var chosen string
		var ok bool
		a.runOnUIThreadSync(func() {
			chosen, ok = a.host.OpenVideoDialog(cfg.LastDir)
		})
		if !ok || chosen == "" {
			return false, nil
		}
		a.loadFileAsync(chosen)
		return true, nil
	})
	a.bind("loadFile", func(path string) (bool, error) {
		if strings.TrimSpace(path) == "" {
			return false, nil
		}
		a.loadFileAsync(path)
		return true, nil
	})
	a.bind("cancelLoad", func() (UISnapshot, error) {
		a.mu.Lock()
		if a.probeCancel != nil {
			a.probeCancel()
			a.probeCancel = nil
		}
		a.loading = false
		a.loadingPct = 0
		a.loadingPhase = ""
		a.loadError = ""
		a.mu.Unlock()
		return a.snapshot(), nil
	})
	a.bind("closeFile", func() (UISnapshot, error) {
		a.unloadMediaAndUnlock()
		a.mu.Lock()
		a.loading = false
		a.loadingPct = 0
		a.loadingPhase = ""
		a.loadError = ""
		a.mu.Unlock()
		return a.snapshot(), nil
	})
	a.bind("recheckTools", func() (UISnapshot, error) {
		cfg := a.settings.Get()
		tools := FindMediaTools(cfg.FFmpegPath, cfg.FFprobePath)
		a.mu.Lock()
		a.tools = tools
		a.mu.Unlock()
		a.srv.SetFFmpegPath(tools.FFmpegPath)
		return a.snapshot(), nil
	})
	a.bind("locateFFmpeg", func() (UISnapshot, error) {
		dialogTitle := "Locate ffmpeg (or ffprobe)"
		ffSibling := "ffmpeg"
		fpSibling := "ffprobe"
		if runtime.GOOS == "windows" {
			dialogTitle = "Locate ffmpeg.exe (or ffprobe.exe)"
			ffSibling = "ffmpeg.exe"
			fpSibling = "ffprobe.exe"
		}
		var chosen string
		var ok bool
		a.runOnUIThreadSync(func() {
			chosen, ok = a.host.OpenExecutableDialog(dialogTitle, defaultToolBrowseDir())
		})
		if !ok || chosen == "" {
			return a.snapshot(), nil
		}
		dir := filepath.Dir(chosen)
		base := strings.ToLower(filepath.Base(chosen))
		a.settings.Update(func(s *Settings) {
			if strings.Contains(base, "ffprobe") {
				s.FFprobePath = chosen
				sib := filepath.Join(dir, ffSibling)
				if st, err := os.Stat(sib); err == nil && !st.IsDir() {
					s.FFmpegPath = sib
				}
			} else {
				s.FFmpegPath = chosen
				sib := filepath.Join(dir, fpSibling)
				if st, err := os.Stat(sib); err == nil && !st.IsDir() {
					s.FFprobePath = sib
				}
			}
		})
		cfg := a.settings.Get()
		tools := FindMediaTools(cfg.FFmpegPath, cfg.FFprobePath)
		a.mu.Lock()
		a.tools = tools
		a.mu.Unlock()
		a.srv.SetFFmpegPath(tools.FFmpegPath)
		return a.snapshot(), nil
	})
	a.bind("addMarker", func(sec float64) (MarkerActionResult, error) {
		a.mu.Lock()
		if a.proj == nil {
			a.mu.Unlock()
			return MarkerActionResult{}, errors.New("no video loaded")
		}
		snapped, added := a.proj.AddMarkerAt(sec)
		a.mu.Unlock()
		return MarkerActionResult{
			SnappedSec: snapped,
			Changed:    added,
			Snapshot:   a.snapshot(),
		}, nil
	})
	a.bind("moveMarker", func(idx int, sec float64) (MarkerActionResult, error) {
		a.mu.Lock()
		if a.proj == nil {
			a.mu.Unlock()
			return MarkerActionResult{}, errors.New("no video loaded")
		}
		snapped, moved := a.proj.MoveMarker(idx, sec)
		a.mu.Unlock()
		return MarkerActionResult{
			SnappedSec: snapped,
			Changed:    moved,
			Snapshot:   a.snapshot(),
		}, nil
	})
	a.bind("removeMarker", func(idx int) (UISnapshot, error) {
		a.mu.Lock()
		if a.proj != nil {
			a.proj.RemoveMarker(idx)
		}
		a.mu.Unlock()
		return a.snapshot(), nil
	})
	a.bind("clearMarkers", func() (UISnapshot, error) {
		a.mu.Lock()
		if a.proj != nil {
			a.proj.ClearMarkers()
		}
		a.mu.Unlock()
		return a.snapshot(), nil
	})
	a.bind("toggleSegment", func(segIdx int) (UISnapshot, error) {
		a.mu.Lock()
		if a.proj != nil {
			a.proj.ToggleSegment(segIdx)
		}
		a.mu.Unlock()
		return a.snapshot(), nil
	})
	a.bind("setAllSegments", func(keep bool) (UISnapshot, error) {
		a.mu.Lock()
		if a.proj != nil {
			a.proj.SetAllKeep(keep)
		}
		a.mu.Unlock()
		return a.snapshot(), nil
	})
	a.bind("invertSegments", func() (UISnapshot, error) {
		a.mu.Lock()
		if a.proj != nil {
			a.proj.InvertKeep()
		}
		a.mu.Unlock()
		return a.snapshot(), nil
	})
	a.bind("setMode", func(mode string) (UISnapshot, error) {
		if mode != "split" {
			mode = "join"
		}
		a.settings.Update(func(s *Settings) { s.ExportMode = mode })
		a.mu.Lock()
		if a.proj != nil {
			a.proj.SetMode(mode)
		}
		a.mu.Unlock()
		return a.snapshot(), nil
	})
	a.bind("setTheme", func(theme string) (UISnapshot, error) {
		if theme != "light" {
			theme = "dark"
		}
		a.settings.Update(func(s *Settings) { s.Theme = theme })
		a.runOnUIThreadSync(func() {
			a.host.ApplyTheme(theme)
		})
		return a.snapshot(), nil
	})
	a.bind("setSkipDropped", func(enabled bool) (UISnapshot, error) {
		a.settings.Update(func(s *Settings) { s.SkipDropped = enabled })
		return a.snapshot(), nil
	})
	a.bind("setShowAudio", func(enabled bool) (UISnapshot, error) {
		a.settings.Update(func(s *Settings) { s.ShowAudio = enabled })
		if enabled {
			a.mu.Lock()
			var startAudio func()
			if a.proj != nil && a.proj.Media != nil && a.proj.Media.HasAudio && a.waveform == nil && !a.waveformLoading {
				startAudio = a.prepareAudioJobLocked()
			}
			a.mu.Unlock()
			if startAudio != nil {
				go startAudio()
			}
		}
		return a.snapshot(), nil
	})
	a.bind("setShowVisual", func(enabled bool) (UISnapshot, error) {
		a.settings.Update(func(s *Settings) { s.ShowVisual = enabled })
		if enabled {
			a.mu.Lock()
			var startVisual func()
			if a.proj != nil && a.proj.Media != nil && a.visual == nil && !a.visualLoading {
				startVisual = a.prepareVisualJobLocked()
			}
			a.mu.Unlock()
			if startVisual != nil {
				go startVisual()
			}
		}
		return a.snapshot(), nil
	})
	a.bind("refreshAnalysis", func(target string) (UISnapshot, error) {
		cfg := a.settings.Get()
		if !a.tools.Ready {
			tools := FindMediaTools(cfg.FFmpegPath, cfg.FFprobePath)
			a.mu.Lock()
			a.tools = tools
			a.mu.Unlock()
		}

		wantKF := target == "keyframes" || target == "all"
		wantAudio := target == "audio" || target == "all"
		wantVisual := target == "visual" || target == "all"

		if wantAudio && !cfg.ShowAudio {
			a.settings.Update(func(s *Settings) { s.ShowAudio = true })
		}
		if wantVisual && !cfg.ShowVisual {
			a.settings.Update(func(s *Settings) { s.ShowVisual = true })
		}

		a.mu.Lock()
		var jobs []func()
		if wantKF {
			if j := a.prepareKeyframesJobLocked(); j != nil {
				jobs = append(jobs, j)
			}
		}
		if wantAudio {
			if j := a.prepareAudioJobLocked(); j != nil {
				jobs = append(jobs, j)
			}
		}
		if wantVisual {
			if j := a.prepareVisualJobLocked(); j != nil {
				jobs = append(jobs, j)
			}
		}
		a.mu.Unlock()

		for _, j := range jobs {
			go j()
		}
		return a.snapshot(), nil
	})
	a.bind("setTimelineScale", func(scale float64) (UISnapshot, error) {
		clamped := ClampTimelineScale(scale)
		a.settings.Update(func(s *Settings) { s.TimelineScale = clamped })
		return a.snapshot(), nil
	})
	a.bind("exportVideo", func() (bool, error) {
		return a.startExportFlow()
	})
	a.bind("cancelExport", func() (bool, error) {
		a.mu.Lock()
		cancel := a.exportCancel
		a.mu.Unlock()
		if cancel != nil {
			cancel()
			return true, nil
		}
		return false, nil
	})
	a.bind("revealInExplorer", func(path string) (bool, error) {
		if path == "" {
			return false, nil
		}
		a.host.RevealInFileManager(path)
		return true, nil
	})
}

func (a *splitApp) bind(name string, fn any) {
	a.bindings[name] = fn
}

func (a *splitApp) handleHTTPRPC(raw []byte) ([]byte, error) {
	var req rpcRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}
	fn, ok := a.bindings[req.Method]
	if !ok {
		return json.Marshal(rpcResponse{ID: req.ID, Error: fmt.Sprintf("unknown method %q", req.Method)})
	}
	res, err := invokeBinding(fn, req.Params)
	if err != nil {
		return json.Marshal(rpcResponse{ID: req.ID, Error: err.Error()})
	}
	return json.Marshal(rpcResponse{ID: req.ID, Result: res})
}

func (a *splitApp) onWebMessage(raw string) {
	if raw == "__file_drop__" {
		return
	}
	var req rpcRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		return
	}
	fn, ok := a.bindings[req.Method]
	if !ok {
		a.resolveRPC(req.ID, nil, fmt.Errorf("unknown method %q", req.Method))
		return
	}

	go func() {
		res, err := invokeBinding(fn, req.Params)
		a.resolveRPC(req.ID, res, err)
	}()
}

func (a *splitApp) resolveRPC(id int, res any, err error) {
	idStr := strconv.Itoa(id)
	if err != nil {
		msgJSON, _ := json.Marshal(err.Error())
		a.evalJS("window._rpc[" + idStr + "] && window._rpc[" + idStr + "].reject(" + string(msgJSON) + "); delete window._rpc[" + idStr + "];")
		return
	}
	b, mErr := json.Marshal(res)
	if mErr != nil {
		msgJSON, _ := json.Marshal(mErr.Error())
		a.evalJS("window._rpc[" + idStr + "] && window._rpc[" + idStr + "].reject(" + string(msgJSON) + "); delete window._rpc[" + idStr + "];")
		return
	}
	a.evalJS("window._rpc[" + idStr + "] && window._rpc[" + idStr + "].resolve(" + string(b) + "); delete window._rpc[" + idStr + "];")
}

func invokeBinding(fn any, rawParams []json.RawMessage) (any, error) {
	v := reflect.ValueOf(fn)
	t := v.Type()
	if len(rawParams) != t.NumIn() {
		return nil, fmt.Errorf("expected %d arguments, got %d", t.NumIn(), len(rawParams))
	}
	args := make([]reflect.Value, len(rawParams))
	for i, p := range rawParams {
		argPtr := reflect.New(t.In(i))
		if err := json.Unmarshal(p, argPtr.Interface()); err != nil {
			return nil, err
		}
		args[i] = argPtr.Elem()
	}
	out := v.Call(args)
	if len(out) == 2 {
		if !out[1].IsNil() {
			return nil, out[1].Interface().(error)
		}
		return out[0].Interface(), nil
	}
	if len(out) == 1 {
		return out[0].Interface(), nil
	}
	return nil, nil
}

func (a *splitApp) startExportFlow() (bool, error) {
	a.mu.Lock()
	if a.exporting {
		a.mu.Unlock()
		return false, errors.New("an export is already in progress")
	}
	if a.proj == nil || a.proj.Media == nil {
		a.mu.Unlock()
		return false, errors.New("load a video file first")
	}
	if !a.tools.Ready {
		errMsg := a.tools.Error
		a.mu.Unlock()
		return false, errors.New(errMsg)
	}
	kept := a.proj.KeptSegments()
	if len(kept) == 0 {
		a.mu.Unlock()
		return false, errors.New("select at least one segment as Keep before exporting")
	}

	// Snapshot project copy so UI edits during export cannot corrupt in-flight export
	projCopy := &ProjectState{
		FilePath:  a.proj.FilePath,
		Media:     a.proj.Media,
		Markers:   append([]float64(nil), a.proj.Markers...),
		KeepFlags: append([]bool(nil), a.proj.KeepFlags...),
		Mode:      a.proj.Mode,
	}
	tools := a.tools
	a.mu.Unlock()

	srcDir, _, ext := SplitBaseExt(projCopy.FilePath)
	var chosen string
	var ok bool
	var splitTargets []string

	a.runOnUIThreadSync(func() {
		if projCopy.Mode == "join" {
			defName := filepath.Base(SuggestedJoinPath(projCopy.FilePath))
			chosen, ok = a.host.SaveVideoDialog("Export Joined Video", srcDir, defName, ext, true)
		} else {
			defName := filepath.Base(SuggestedSplitDialogPath(projCopy.FilePath))
			chosen, ok = a.host.SaveVideoDialog(fmt.Sprintf("Export %d Split Segments (Base Name)", len(kept)), srcDir, defName, ext, false)
			if ok && chosen != "" {
				splitTargets = DeriveSplitPaths(chosen, len(kept))
				var existing []string
				for _, tp := range splitTargets {
					if _, err := os.Stat(tp); err == nil {
						existing = append(existing, filepath.Base(tp))
					}
				}
				if len(existing) > 0 {
					preview := strings.Join(existing, ", ")
					if len(existing) > 5 {
						preview = strings.Join(existing[:5], ", ") + fmt.Sprintf(" (+%d more)", len(existing)-5)
					}
					if !a.host.ConfirmOverwriteDialog("Overwrite Existing Files?", fmt.Sprintf("The following output file(s) already exist:\n\n%s\n\nOverwrite them?", preview)) {
						ok = false
					}
				}
			}
		}
	})

	if !ok || chosen == "" {
		return false, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.exporting = true
	a.exportCancel = cancel
	a.lastExport = nil
	a.exportProg = &ExportProgress{
		Current: 0,
		Total:   len(kept),
		Label:   filepath.Base(chosen),
		Phase:   "Starting lossless export...",
	}
	a.mu.Unlock()
	a.pushState()

	go func() {
		defer cancel()
		sw := time.Now()
		onProg := func(p ExportProgress) {
			a.mu.Lock()
			pc := p
			a.exportProg = &pc
			a.mu.Unlock()
			a.pushProgress(p)
		}

		var res *ExportResult
		var err error
		if projCopy.Mode == "join" {
			res, err = ExportJoin(ctx, tools, projCopy, chosen, onProg)
		} else {
			res, err = ExportSplit(ctx, tools, projCopy, splitTargets, onProg)
		}

		elapsed := time.Since(sw).Seconds()
		a.mu.Lock()
		a.exporting = false
		a.exportCancel = nil
		a.exportProg = nil
		if err == nil {
			a.lastExport = res
			a.lastExportSec = elapsed
		}
		a.mu.Unlock()

		if err != nil {
			if errors.Is(err, context.Canceled) {
				a.evalJS("window.__onExportDone && window.__onExportDone({cancelled:true});")
			} else {
				errJSON, _ := json.Marshal(err.Error())
				a.evalJS("window.__onExportDone && window.__onExportDone({error:" + string(errJSON) + "});")
			}
			a.pushState()
			return
		}

		a.pushState()
		a.evalJS("window.__onExportDone && window.__onExportDone({ok:true});")
	}()

	return true, nil
}
