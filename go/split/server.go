package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

//go:embed web/index.html web/app.css web/app.js
var webAssets embed.FS

type scrubCacheEntry struct {
	videoPath string
	ms        int64
	exact     bool
	jpeg      []byte
}

const maxScrubCacheEntries = 96

type mediaServer struct {
	token    string
	baseURL  string
	listener net.Listener
	srv      *http.Server

	mu           sync.RWMutex
	videoPath    string
	videoCodec   string
	ffmpegPath   string
	streamCtx    context.Context
	streamCancel context.CancelFunc

	scrubMu    sync.Mutex
	scrubCache []scrubCacheEntry
}

func mergeCancelContexts(a, b context.Context) (context.Context, context.CancelFunc) {
	if b == nil {
		return context.WithCancel(a)
	}
	ctx, cancel := context.WithCancel(a)
	stop := context.AfterFunc(b, cancel)
	return ctx, func() {
		stop()
		cancel()
	}
}

type ctxReadSeeker struct {
	f   *os.File
	ctx context.Context
}

func (c ctxReadSeeker) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.f.Read(p)
}

func (c ctxReadSeeker) Seek(offset int64, whence int) (int64, error) {
	return c.f.Seek(offset, whence)
}

func startMediaServer(ffmpegPath string) (*mediaServer, error) {
	var tokBytes [16]byte
	if _, err := rand.Read(tokBytes[:]); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(tokBytes[:])

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	streamCtx, streamCancel := context.WithCancel(context.Background())
	ms := &mediaServer{
		token:        token,
		baseURL:      fmt.Sprintf("http://%s/s/%s", ln.Addr().String(), token),
		listener:     ln,
		ffmpegPath:   ffmpegPath,
		streamCtx:    streamCtx,
		streamCancel: streamCancel,
	}

	mux := http.NewServeMux()
	prefix := "/s/" + token
	mux.HandleFunc(prefix+"/", ms.handleAsset)
	mux.HandleFunc(prefix+"/app.css", ms.handleAsset)
	mux.HandleFunc(prefix+"/app.js", ms.handleAsset)
	mux.HandleFunc(prefix+"/video", ms.handleVideo)
	mux.HandleFunc(prefix+"/remux", ms.handleRemux)
	mux.HandleFunc(prefix+"/scrub", ms.handleScrub)

	ms.srv = &http.Server{Handler: mux}
	go func() {
		_ = ms.srv.Serve(ln)
	}()
	return ms, nil
}

func (ms *mediaServer) Close() {
	ms.mu.Lock()
	if ms.streamCancel != nil {
		ms.streamCancel()
		ms.streamCancel = nil
	}
	ms.mu.Unlock()
	if ms.srv != nil {
		_ = ms.srv.Close()
	}
}

func (ms *mediaServer) ClearActiveVideo() {
	ms.mu.Lock()
	if ms.streamCancel != nil {
		ms.streamCancel()
	}
	ms.streamCtx, ms.streamCancel = context.WithCancel(context.Background())
	ms.videoPath = ""
	ms.videoCodec = ""
	ms.mu.Unlock()

	ms.scrubMu.Lock()
	ms.scrubCache = nil
	ms.scrubMu.Unlock()
}

func (ms *mediaServer) SetActiveVideo(path, videoCodec string) {
	ms.mu.Lock()
	if ms.streamCancel != nil {
		ms.streamCancel()
	}
	ms.streamCtx, ms.streamCancel = context.WithCancel(context.Background())
	ms.videoPath = path
	ms.videoCodec = strings.ToLower(strings.TrimSpace(videoCodec))
	ms.mu.Unlock()

	ms.scrubMu.Lock()
	ms.scrubCache = nil
	ms.scrubMu.Unlock()
}

func (ms *mediaServer) SetFFmpegPath(p string) {
	ms.mu.Lock()
	ms.ffmpegPath = p
	ms.mu.Unlock()
}

func (ms *mediaServer) IndexURL() string {
	return ms.baseURL + "/"
}

func (ms *mediaServer) handleAsset(w http.ResponseWriter, r *http.Request) {
	prefix := "/s/" + ms.token + "/"
	rel := strings.TrimPrefix(r.URL.Path, prefix)
	if rel == "" {
		rel = "index.html"
	}
	switch rel {
	case "index.html", "app.css", "app.js":
	default:
		http.NotFound(w, r)
		return
	}

	data, err := webAssets.ReadFile("web/" + rel)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	ct := "application/octet-stream"
	switch filepath.Ext(rel) {
	case ".html":
		ct = "text/html; charset=utf-8"
	case ".css":
		ct = "text/css; charset=utf-8"
	case ".js":
		ct = "application/javascript; charset=utf-8"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}

func (ms *mediaServer) handleVideo(w http.ResponseWriter, r *http.Request) {
	ms.mu.RLock()
	p := ms.videoPath
	sCtx := ms.streamCtx
	ms.mu.RUnlock()
	if p == "" {
		http.NotFound(w, r)
		return
	}

	f, err := openSharedReadFile(p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ext := strings.ToLower(filepath.Ext(p))
	var ct string
	switch ext {
	case ".mp4", ".m4v":
		ct = "video/mp4"
	case ".webm":
		ct = "video/webm"
	case ".mov":
		ct = "video/quicktime"
	case ".mkv":
		ct = "video/x-matroska"
	case ".avi":
		ct = "video/x-msvideo"
	case ".ts", ".m2ts":
		ct = "video/mp2t"
	default:
		ct = mime.TypeByExtension(ext)
	}
	if ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Cache-Control", "no-store")
	reqCtx, cancel := mergeCancelContexts(r.Context(), sCtx)
	defer cancel()
	http.ServeContent(w, r.WithContext(reqCtx), st.Name(), st.ModTime(), ctxReadSeeker{f: f, ctx: reqCtx})
}

func (ms *mediaServer) handleScrub(w http.ResponseWriter, r *http.Request) {
	ms.mu.RLock()
	p := ms.videoPath
	ffmpegPath := ms.ffmpegPath
	sCtx := ms.streamCtx
	ms.mu.RUnlock()

	if p == "" || ffmpegPath == "" {
		http.Error(w, "scrub unavailable", http.StatusServiceUnavailable)
		return
	}

	reqCtx, cancel := mergeCancelContexts(r.Context(), sCtx)
	defer cancel()

	tSec := 0.0
	if rawT := r.URL.Query().Get("t"); rawT != "" {
		if v, err := strconv.ParseFloat(rawT, 64); err == nil && v > 0 {
			tSec = v
		}
	}
	exact := r.URL.Query().Get("exact") == "1"
	msKey := int64(math.Round(tSec * 1000))

	ms.scrubMu.Lock()
	for i := len(ms.scrubCache) - 1; i >= 0; i-- {
		e := ms.scrubCache[i]
		if e.videoPath == p && e.ms == msKey && e.exact == exact {
			cached := e.jpeg
			ms.scrubMu.Unlock()
			w.Header().Set("Content-Type", "image/jpeg")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(cached)
			return
		}
	}
	ms.scrubMu.Unlock()

	extractFrame := func(exactMode bool) ([]byte, error) {
		args := []string{
			"-hide_banner", "-loglevel", "error",
			"-threads", "2",
			"-err_detect", "ignore_err",
		}
		if !exactMode {
			args = append(args, "-noaccurate_seek")
			if tSec > 0.001 {
				args = append(args, "-ss", strconv.FormatFloat(tSec+0.005, 'f', 6, 64))
			}
			args = append(args, "-skip_frame", "nokey")
		} else if tSec > 0.001 {
			args = append(args, "-ss", strconv.FormatFloat(tSec, 'f', 6, 64))
		}
		args = append(args,
			"-i", p,
			"-an",
			"-frames:v", "1",
			"-vf", "scale='min(960,iw)':-2:flags=fast_bilinear",
			"-f", "image2",
			"-c:v", "mjpeg",
			"-q:v", "5",
			"pipe:1",
		)

		cmd := exec.CommandContext(reqCtx, ffmpegPath, args...)
		hideSubprocessWindow(cmd)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		err := cmd.Run()
		data := stdout.Bytes()
		if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("empty or invalid jpeg scrub frame")
		}
		return data, nil
	}

	data, err := extractFrame(exact)
	if err != nil && !exact && reqCtx.Err() == nil {
		data, err = extractFrame(true)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ms.scrubMu.Lock()
	if ms.videoPath == p {
		ms.scrubCache = append(ms.scrubCache, scrubCacheEntry{
			videoPath: p,
			ms:        msKey,
			exact:     exact,
			jpeg:      data,
		})
		if len(ms.scrubCache) > maxScrubCacheEntries {
			copy(ms.scrubCache, ms.scrubCache[1:])
			ms.scrubCache = ms.scrubCache[:maxScrubCacheEntries]
		}
	}
	ms.scrubMu.Unlock()

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}

func (ms *mediaServer) handleRemux(w http.ResponseWriter, r *http.Request) {
	ms.mu.RLock()
	p := ms.videoPath
	vcodec := ms.videoCodec
	ffmpegPath := ms.ffmpegPath
	sCtx := ms.streamCtx
	ms.mu.RUnlock()

	if p == "" || ffmpegPath == "" {
		http.Error(w, "remux unavailable", http.StatusServiceUnavailable)
		return
	}

	reqCtx, cancel := mergeCancelContexts(r.Context(), sCtx)
	defer cancel()

	startSec := 0.0
	if rawStart := r.URL.Query().Get("start"); rawStart != "" {
		if v, err := strconv.ParseFloat(rawStart, 64); err == nil && v > 0 {
			startSec = v
		}
	}
	forceTranscode := r.URL.Query().Get("transcode") == "1"
	kfAligned := r.URL.Query().Get("kf") == "1"
	canCopyVideo := !forceTranscode && (vcodec == "h264" || vcodec == "vp9" || vcodec == "av1")

	args := []string{"-hide_banner", "-loglevel", "error"}
	if canCopyVideo && kfAligned && startSec > 0.001 {
		args = append(args, "-noaccurate_seek", "-ss", strconv.FormatFloat(startSec+0.005, 'f', 6, 64))
	} else if startSec > 0.01 {
		args = append(args, "-ss", strconv.FormatFloat(startSec, 'f', 3, 64))
	}
	args = append(args, "-i", p, "-map", "0:v:0", "-map", "0:a?")

	if canCopyVideo {
		args = append(args, "-c:v", "copy")
	} else {
		args = append(args, "-c:v", "libx264", "-preset", "ultrafast", "-crf", "23", "-pix_fmt", "yuv420p")
	}
	if startSec > 0.001 {
		args = append(args, "-avoid_negative_ts", "make_zero")
	}
	args = append(args,
		"-c:a", "aac", "-ac", "2", "-b:a", "160k",
		"-movflags", "frag_keyframe+empty_moov+default_base_moof",
		"-f", "mp4",
		"pipe:1",
	)

	cmd := exec.CommandContext(reqCtx, ffmpegPath, args...)
	hideSubprocessWindow(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := cmd.Start(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()

	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.Copy(w, stdout)
}
