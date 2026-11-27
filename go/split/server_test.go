package main

import (
	"bytes"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestServerScrubAndMime(t *testing.T) {
	tools := FindMediaTools("", "")
	if !tools.Ready {
		t.Skip("ffmpeg/ffprobe not available")
	}

	outPath := filepath.Join(t.TempDir(), "scrub_colors.mp4")
	// 6s video @ 15fps with keyframes every 30 frames (0s blue, 2s green, 4s red)
	cmd := exec.Command(tools.FFmpegPath,
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "color=c=blue:s=320x180:r=15:d=2",
		"-f", "lavfi", "-i", "color=c=green:s=320x180:r=15:d=2",
		"-f", "lavfi", "-i", "color=c=red:s=320x180:r=15:d=2",
		"-filter_complex", "[0:v][1:v][2:v]concat=n=3:v=1:a=0[v]",
		"-map", "[v]",
		"-c:v", "libx264", "-g", "30", "-keyint_min", "30", "-sc_threshold", "0", "-pix_fmt", "yuv420p",
		outPath,
	)
	hideSubprocessWindow(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to generate scrub test video: %v (%s)", err, string(out))
	}

	srv, err := startMediaServer(tools.FFmpegPath)
	if err != nil {
		t.Fatalf("startMediaServer err: %v", err)
	}
	defer srv.Close()
	srv.SetActiveVideo(outPath, "h264")

	// Verify explicit video/mp4 Content-Type on /video
	vResp, err := http.Get(srv.baseURL + "/video")
	if err != nil {
		t.Fatalf("GET /video err: %v", err)
	}
	_ = vResp.Body.Close()
	if vResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /video status = %d", vResp.StatusCode)
	}
	if ct := vResp.Header.Get("Content-Type"); ct != "video/mp4" {
		t.Fatalf("GET /video Content-Type = %q, want video/mp4", ct)
	}

	fetchCenterRGB := func(query string) (uint32, uint32, uint32, []byte) {
		resp, err := http.Get(srv.baseURL + "/scrub" + query)
		if err != nil {
			t.Fatalf("GET /scrub%s err: %v", query, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET /scrub%s status = %d", query, resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); ct != "image/jpeg" {
			t.Fatalf("GET /scrub%s Content-Type = %q, want image/jpeg", query, ct)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read scrub body: %v", err)
		}
		img, err := jpeg.Decode(bytes.NewReader(body))
		if err != nil {
			t.Fatalf("jpeg decode %s: %v", query, err)
		}
		b := img.Bounds()
		r, g, bl, _ := img.At(b.Dx()/2, b.Dy()/2).RGBA()
		return r >> 8, g >> 8, bl >> 8, body
	}

	r0, g0, b0, _ := fetchCenterRGB("?t=0")
	if b0 <= r0+50 || b0 <= g0+50 {
		t.Fatalf("expected blue keyframe at t=0, got R=%d G=%d B=%d", r0, g0, b0)
	}

	r2, g2, b2, raw2 := fetchCenterRGB("?t=2.0")
	if g2 <= r2+50 || g2 <= b2+50 {
		t.Fatalf("expected green keyframe at t=2.0, got R=%d G=%d B=%d", r2, g2, b2)
	}

	r4, g4, b4, _ := fetchCenterRGB("?t=4.0")
	if r4 <= g4+50 || r4 <= b4+50 {
		t.Fatalf("expected red keyframe at t=4.0, got R=%d G=%d B=%d", r4, g4, b4)
	}

	// Verify cache hit returns identical bytes and exact=1 returns valid JPEG
	_, _, _, raw2Cached := fetchCenterRGB("?t=2.0")
	if !bytes.Equal(raw2, raw2Cached) {
		t.Fatalf("expected cache hit for t=2.0 to return identical JPEG bytes")
	}
	re, ge, be, _ := fetchCenterRGB("?t=5.0&exact=1")
	if re <= ge+50 || re <= be+50 {
		t.Fatalf("expected red exact frame at t=5.0, got R=%d G=%d B=%d", re, ge, be)
	}
}

func TestActiveVideoAllowsRenameAndDeleteOnUnload(t *testing.T) {
	tools := FindMediaTools("", "")
	if !tools.Ready {
		t.Skip("ffmpeg/ffprobe not available")
	}

	dir := t.TempDir()
	vidPath := filepath.Join(dir, "unlock_test.mp4")
	cmd := exec.Command(tools.FFmpegPath,
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=size=160x120:rate=15:d=3",
		"-c:v", "libx264", "-pix_fmt", "yuv420p",
		vidPath,
	)
	hideSubprocessWindow(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to generate unlock test video: %v (%s)", err, string(out))
	}

	srv, err := startMediaServer(tools.FFmpegPath)
	if err != nil {
		t.Fatalf("startMediaServer err: %v", err)
	}
	defer srv.Close()
	srv.SetActiveVideo(vidPath, "h264")

	// Simulate an active /video range stream and a /scrub fetch, then immediately unload
	req, err := http.NewRequest(http.MethodGet, srv.baseURL+"/video", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Range", "bytes=0-1023")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /video err: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	sResp, err := http.Get(srv.baseURL + "/scrub?t=0")
	if err != nil {
		t.Fatalf("GET /scrub err: %v", err)
	}
	_ = sResp.Body.Close()

	// Unloading clears active video and allows immediate rename & deletion
	srv.ClearActiveVideo()

	renamedPath := filepath.Join(dir, "unlock_renamed.mp4")
	if err := os.Rename(vidPath, renamedPath); err != nil {
		t.Fatalf("expected unloaded video to be renameable immediately, got %v", err)
	}
	if err := os.Remove(renamedPath); err != nil {
		t.Fatalf("expected unloaded video to be deletable immediately, got %v", err)
	}
}
