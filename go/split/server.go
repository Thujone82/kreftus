package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"fmt"
	"io"
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

type mediaServer struct {
	token    string
	baseURL  string
	listener net.Listener
	srv      *http.Server

	mu         sync.RWMutex
	videoPath  string
	videoCodec string
	ffmpegPath string
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

	ms := &mediaServer{
		token:      token,
		baseURL:    fmt.Sprintf("http://%s/s/%s", ln.Addr().String(), token),
		listener:   ln,
		ffmpegPath: ffmpegPath,
	}

	mux := http.NewServeMux()
	prefix := "/s/" + token
	mux.HandleFunc(prefix+"/", ms.handleAsset)
	mux.HandleFunc(prefix+"/app.css", ms.handleAsset)
	mux.HandleFunc(prefix+"/app.js", ms.handleAsset)
	mux.HandleFunc(prefix+"/video", ms.handleVideo)
	mux.HandleFunc(prefix+"/remux", ms.handleRemux)

	ms.srv = &http.Server{Handler: mux}
	go func() {
		_ = ms.srv.Serve(ln)
	}()
	return ms, nil
}

func (ms *mediaServer) Close() {
	if ms.srv != nil {
		_ = ms.srv.Close()
	}
}

func (ms *mediaServer) SetActiveVideo(path, videoCodec string) {
	ms.mu.Lock()
	ms.videoPath = path
	ms.videoCodec = strings.ToLower(strings.TrimSpace(videoCodec))
	ms.mu.Unlock()
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
	ms.mu.RUnlock()
	if p == "" {
		http.NotFound(w, r)
		return
	}

	f, err := os.Open(p)
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

	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(p))); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, st.Name(), st.ModTime(), f)
}

func (ms *mediaServer) handleRemux(w http.ResponseWriter, r *http.Request) {
	ms.mu.RLock()
	p := ms.videoPath
	vcodec := ms.videoCodec
	ffmpegPath := ms.ffmpegPath
	ms.mu.RUnlock()

	if p == "" || ffmpegPath == "" {
		http.Error(w, "remux unavailable", http.StatusServiceUnavailable)
		return
	}

	startSec := 0.0
	if rawStart := r.URL.Query().Get("start"); rawStart != "" {
		if v, err := strconv.ParseFloat(rawStart, 64); err == nil && v > 0 {
			startSec = v
		}
	}
	forceTranscode := r.URL.Query().Get("transcode") == "1"

	args := []string{"-hide_banner", "-loglevel", "error"}
	if startSec > 0.01 {
		args = append(args, "-ss", strconv.FormatFloat(startSec, 'f', 3, 64))
	}
	args = append(args, "-i", p, "-map", "0:v:0", "-map", "0:a?")

	canCopyVideo := !forceTranscode && (vcodec == "h264" || vcodec == "vp9" || vcodec == "av1")
	if canCopyVideo {
		args = append(args, "-c:v", "copy")
	} else {
		args = append(args, "-c:v", "libx264", "-preset", "ultrafast", "-crf", "23", "-pix_fmt", "yuv420p")
	}
	args = append(args,
		"-c:a", "aac", "-ac", "2", "-b:a", "160k",
		"-movflags", "frag_keyframe+empty_moov+default_base_moof",
		"-f", "mp4",
		"pipe:1",
	)

	cmd := exec.CommandContext(r.Context(), ffmpegPath, args...)
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
