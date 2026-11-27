(() => {
  "use strict";

  const $ = (id) => document.getElementById(id);

  const el = {
    emptyView: $("emptyView"),
    dropCard: $("dropCard"),
    globalDropOverlay: $("globalDropOverlay"),
    editorView: $("editorView"),
    fileBadge: $("fileBadge"),
    fileName: $("fileName"),
    fileMeta: $("fileMeta"),
    btnCloseFile: $("btnCloseFile"),
    toolHealth: $("toolHealth"),
    prereqCard: $("prereqCard"),
    prereqFFmpeg: $("prereqFFmpeg"),
    prereqFFprobe: $("prereqFFprobe"),
    btnCopyWinget: $("btnCopyWinget"),
    btnLocateFFmpeg: $("btnLocateFFmpeg"),
    btnRecheckTools: $("btnRecheckTools"),
    loadingOverlay: $("loadingOverlay"),
    loadBgA: $("loadBgA"),
    loadBgB: $("loadBgB"),
    loadingTitle: $("loadingTitle"),
    loadPctText: $("loadPctText"),
    loadProgressBar: $("loadProgressBar"),
    loadingText: $("loadingText"),
    btnCancelLoad: $("btnCancelLoad"),
    video: $("videoEl"),
    scrubCanvas: $("scrubCanvas"),
    hudOverlay: $("hudOverlay"),
    hudTime: $("hudTime"),
    hudDuration: $("hudDuration"),
    kfBadge: $("kfBadge"),
    remuxBadge: $("remuxBadge"),
    btnOpen: $("btnOpen"),
    btnEmptyBrowse: $("btnEmptyBrowse"),
    btnModeJoin: $("btnModeJoin"),
    btnModeSplit: $("btnModeSplit"),
    btnTheme: $("btnTheme"),
    iconSun: $("iconSun"),
    iconMoon: $("iconMoon"),
    btnExport: $("btnExport"),
    lblExport: $("lblExport"),
    btnPlayPause: $("btnPlayPause"),
    iconPlay: $("iconPlay"),
    iconPause: $("iconPause"),
    btnBack5: $("btnBack5"),
    btnFwd5: $("btnFwd5"),
    btnPrevKF: $("btnPrevKF"),
    btnNextKF: $("btnNextKF"),
    btnAddCut: $("btnAddCut"),
    btnSkipDropped: $("btnSkipDropped"),
    selSpeed: $("selSpeed"),
    btnMute: $("btnMute"),
    volWaveLow: $("volWaveLow"),
    volWaveHigh: $("volWaveHigh"),
    volMuteCross: $("volMuteCross"),
    rngVolume: $("rngVolume"),
    btnToggleAudio: $("btnToggleAudio"),
    btnToggleVisual: $("btnToggleVisual"),
    btnZoomOut: $("btnZoomOut"),
    rngZoom: $("rngZoom"),
    lblZoom: $("lblZoom"),
    btnZoomIn: $("btnZoomIn"),
    btnZoomFit: $("btnZoomFit"),
    btnKeepAll: $("btnKeepAll"),
    btnDropAll: $("btnDropAll"),
    btnInvert: $("btnInvert"),
    btnClearCuts: $("btnClearCuts"),
    verticalSplitter: $("verticalSplitter"),
    editorShelf: $("editorShelf"),
    timelineWrap: $("timelineWrap"),
    timelineCanvas: $("timelineCanvas"),
    overviewWrap: $("overviewWrap"),
    overviewCanvas: $("overviewCanvas"),
    rulerHit: $("rulerHit"),
    segmentsHit: $("segmentsHit"),
    markerLayer: $("markerLayer"),
    hoverTooltip: $("hoverTooltip"),
    segmentPills: $("segmentPills"),
    outputSummary: $("outputSummary"),
    toast: $("toast"),
    exportModal: $("exportModal"),
    modalTitle: $("modalTitle"),
    btnModalClose: $("btnModalClose"),
    exportProgressArea: $("exportProgressArea"),
    exportProgressBar: $("exportProgressBar"),
    exportPhaseText: $("exportPhaseText"),
    exportResultArea: $("exportResultArea"),
    btnCancelExport: $("btnCancelExport"),
    btnRevealFolder: $("btnRevealFolder"),
    btnDoneModal: $("btnDoneModal"),
  };

  let state = null;
  let lastLoadedSeq = 0;
  let useRemux = false;
  let remuxTranscode = false;
  let remuxOffset = 0;
  let remuxSubOffset = 0;
  let remuxPinnedSec = null;
  let remuxSeq = 0;
  let ghostSnapSec = null;
  let dragMarker = null; // { idx, previewSec, playheadBefore }
  let scrubbingRuler = false;
  let scrubPreviewSec = null;
  let scrubInFlight = false;
  let pendingScrubReq = null;
  let displayedScrubKey = "";
  let nativeScrubSeeking = false;
  let settleHideTimer = null;
  let seekVersion = 0;
  let timelineScale = 1.0;
  let activeLoadBgSlot = 0;
  let lastBackdropSeq = 0;
  let splitterDragging = false;
  let toastTimer = null;
  let dragCounter = 0;

  // Timeline zoom & pan state
  let zoom = 1;
  let viewStart = 0;

  // Decoded audio & visual histogram buffers
  let wavePeaks = null;
  let waveRMS = null;
  let waveActivity = null; // 0=quiet/ambient, 1=talking/speech, 2=noise/transient
  let visualDeltas = null;
  let visualScenes = null;

  // Smooth 60fps progress animation state
  let loadTargetPct = 0;
  let loadDisplayPct = 0;
  let audioTargetPct = 0;
  let audioDisplayPct = 0;
  let visualTargetPct = 0;
  let visualDisplayPct = 0;

  // Cached DOM nodes for zero-thrashing zoom/pan updates
  let segDomItems = [];    // { el, start, end }
  let markerDomItems = []; // { el, time }

  function fmtTime(sec) {
    if (!Number.isFinite(sec) || sec < 0) sec = 0;
    const totalCs = Math.round(sec * 100);
    const cs = totalCs % 100;
    const totalS = Math.floor(totalCs / 100);
    const s = totalS % 60;
    const totalM = Math.floor(totalS / 60);
    const m = totalM % 60;
    const h = Math.floor(totalM / 60);
    const p2 = (n) => String(n).padStart(2, "0");
    if (h > 0) return `${h}:${p2(m)}:${p2(s)}.${p2(cs)}`;
    return `${p2(m)}:${p2(s)}.${p2(cs)}`;
  }

  function fmtBytes(bytes) {
    if (bytes >= 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
    if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
    if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${bytes} B`;
  }

  function showToast(msg, isError = false) {
    el.toast.textContent = msg;
    el.toast.classList.toggle("error", isError);
    el.toast.classList.remove("hidden");
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => el.toast.classList.add("hidden"), 2600);
  }

  function getTotalDuration() {
    if (state && state.media && state.media.durationSec > 0) {
      return state.media.durationSec;
    }
    return el.video.duration || 0;
  }

  function getCurrentTime() {
    if (scrubPreviewSec !== null) {
      return Math.max(0, Math.min(getTotalDuration(), scrubPreviewSec));
    }
    if (dragMarker && typeof dragMarker.playheadBefore === "number") {
      return dragMarker.playheadBefore;
    }
    if (useRemux) {
      if (remuxPinnedSec !== null) {
        return Math.max(0, Math.min(getTotalDuration(), remuxPinnedSec));
      }
      return Math.min(getTotalDuration(), remuxOffset + (el.video.currentTime || 0));
    }
    return el.video.currentTime || 0;
  }

  function getBaseScalableHeight() {
    const showAudio = !state || state.showAudio !== false;
    const showVisual = !state || state.showVisual !== false;
    return (showAudio ? 52 : 0) + (showVisual ? 52 : 0) + 62;
  }

  function getMaxAllowedTimelineScale() {
    const winH = window.innerHeight || 720;
    const availForTimeline = Math.max(120, winH - 130 - 216);
    const scalable = getBaseScalableHeight();
    if (scalable <= 0) return 3.2;
    return Math.max(0.55, Math.min(3.2, (availForTimeline - 26) / scalable));
  }

  function getTimelineLayout() {
    const showAudio = !state || state.showAudio !== false;
    const showVisual = !state || state.showVisual !== false;
    const scale = Math.max(0.55, Math.min(getMaxAllowedTimelineScale(), timelineScale || 1.0));
    const rulerH = 26;
    const audioH = showAudio ? Math.max(28, Math.round(52 * scale)) : 0;
    const visualH = showVisual ? Math.max(28, Math.round(52 * scale)) : 0;
    const segH = Math.max(40, Math.round(62 * scale));
    const audioBottom = rulerH + audioH;
    const scrubBottom = audioBottom + visualH;
    const totalH = scrubBottom + segH;
    return { rulerH, audioH, visualH, segH, audioBottom, scrubBottom, totalH };
  }

  function applyTimelineLayout() {
    const lay = getTimelineLayout();
    el.timelineWrap.style.height = `${lay.totalH}px`;
    el.rulerHit.style.height = `${lay.scrubBottom}px`;
    el.segmentsHit.style.top = `${lay.scrubBottom}px`;
    el.timelineWrap.style.setProperty("--seg-top", `${lay.scrubBottom + 4}px`);
  }

  function getMaxZoom() {
    const dur = getTotalDuration();
    if (dur <= 0) return 16;
    return Math.max(16, Math.min(240, Math.ceil(dur / 2.5)));
  }

  function getViewSpan() {
    const dur = getTotalDuration();
    if (dur <= 0) return 0;
    return dur / Math.max(1, zoom);
  }

  function clampViewStart(vs) {
    const dur = getTotalDuration();
    const span = getViewSpan();
    const maxStart = Math.max(0, dur - span);
    return Math.max(0, Math.min(maxStart, vs));
  }

  function updateZoomUI() {
    const maxZ = getMaxZoom();
    const ratio = maxZ > 1 ? Math.log(zoom) / Math.log(maxZ) : 0;
    el.rngZoom.value = String(Math.round(Math.max(0, Math.min(1, ratio)) * 1000));
    el.lblZoom.textContent = zoom >= 10 ? `${zoom.toFixed(0)}×` : `${zoom.toFixed(1)}×`;
  }

  function setZoomAt(nextZoom, anchorTime, anchorRatio) {
    const dur = getTotalDuration();
    if (dur <= 0) return;
    const maxZ = getMaxZoom();
    zoom = Math.max(1, Math.min(maxZ, nextZoom));
    if (zoom <= 1.001) {
      zoom = 1;
      viewStart = 0;
    } else {
      const span = dur / zoom;
      const ratio = Number.isFinite(anchorRatio) ? anchorRatio : 0.5;
      const anchor = Number.isFinite(anchorTime) ? anchorTime : getCurrentTime();
      viewStart = clampViewStart(anchor - ratio * span);
    }
    updateZoomUI();
    syncTimelineDOMPositions();
    drawTimeline();
    drawOverview();
  }

  function clearScrubQueue() {
    pendingScrubReq = null;
    scrubInFlight = false;
    nativeScrubSeeking = false;
  }

  function hideScrubCanvas() {
    clearTimeout(settleHideTimer);
    settleHideTimer = null;
    if (el.scrubCanvas) {
      el.scrubCanvas.classList.add("hidden");
    }
    displayedScrubKey = "";
  }

  function paintToScrubCanvas(drawable, srcW, srcH) {
    const wrap = el.video && el.video.parentElement;
    if (!wrap || !el.scrubCanvas || !srcW || !srcH) return false;
    const w = wrap.clientWidth;
    const h = wrap.clientHeight;
    if (w <= 0 || h <= 0) return false;

    const dpr = window.devicePixelRatio || 1;
    const cw = Math.round(w * dpr);
    const ch = Math.round(h * dpr);
    if (el.scrubCanvas.width !== cw || el.scrubCanvas.height !== ch) {
      el.scrubCanvas.width = cw;
      el.scrubCanvas.height = ch;
    }

    const ctx = el.scrubCanvas.getContext("2d");
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.fillStyle = "#000000";
    ctx.fillRect(0, 0, w, h);

    const scale = Math.min(w / srcW, h / srcH);
    const dw = srcW * scale;
    const dh = srcH * scale;
    const dx = (w - dw) * 0.5;
    const dy = (h - dh) * 0.5;
    ctx.drawImage(drawable, dx, dy, dw, dh);
    el.scrubCanvas.classList.remove("hidden");
    return true;
  }

  function freezeCurrentFrameToCanvas() {
    if (!el.scrubCanvas) return false;
    if (!el.scrubCanvas.classList.contains("hidden")) {
      return true;
    }
    if (el.video.readyState >= 2 && el.video.videoWidth > 0 && el.video.videoHeight > 0) {
      return paintToScrubCanvas(el.video, el.video.videoWidth, el.video.videoHeight);
    }
    return false;
  }

  function repaintScrubCanvasIfVisible() {
    if (!el.scrubCanvas || el.scrubCanvas.classList.contains("hidden")) return;
    if (el.video.readyState >= 2 && el.video.videoWidth > 0) {
      paintToScrubCanvas(el.video, el.video.videoWidth, el.video.videoHeight);
    }
  }

  function requestScrubFrame(sec, exact = false) {
    if (!state || !state.media) return;
    const seq = state.loadSeq;
    const dur = getTotalDuration();
    const clamped = Math.max(0, Math.min(dur > 0 ? dur : sec, sec));
    const key = `${seq}:${exact ? "E" : "K"}:${Math.round(clamped * 1000)}`;
    if (key === displayedScrubKey && el.scrubCanvas && !el.scrubCanvas.classList.contains("hidden")) {
      return;
    }
    pendingScrubReq = { sec: clamped, exact, key, seq };
    pumpScrubQueue();
  }

  function fetchServerScrubFrame(req) {
    if (!state || !state.scrubUrl || state.loadSeq !== req.seq) {
      scrubInFlight = false;
      pumpScrubQueue();
      return;
    }
    const img = new Image();
    img.onload = () => {
      if (state && state.loadSeq === req.seq && img.naturalWidth > 0) {
        if (paintToScrubCanvas(img, img.naturalWidth, img.naturalHeight)) {
          displayedScrubKey = req.key;
        }
      }
      scrubInFlight = false;
      pumpScrubQueue();
    };
    img.onerror = () => {
      scrubInFlight = false;
      pumpScrubQueue();
    };
    img.src = `${state.scrubUrl}&t=${req.sec.toFixed(6)}${req.exact ? "&exact=1" : ""}`;
  }

  function pumpScrubQueue() {
    if (scrubInFlight || !pendingScrubReq || !state || !state.media) return;
    const req = pendingScrubReq;
    pendingScrubReq = null;

    if (!useRemux && !req.exact && el.video.readyState >= 1) {
      scrubInFlight = true;
      nativeScrubSeeking = true;
      let finished = false;
      let timer = null;

      const finish = (paint) => {
        if (finished) return;
        finished = true;
        clearTimeout(timer);
        el.video.removeEventListener("seeked", onSeeked);
        nativeScrubSeeking = false;
        if (paint && state && state.loadSeq === req.seq && el.video.readyState >= 2 && el.video.videoWidth > 0) {
          if (paintToScrubCanvas(el.video, el.video.videoWidth, el.video.videoHeight)) {
            displayedScrubKey = req.key;
          }
          scrubInFlight = false;
          pumpScrubQueue();
          return;
        }
        // Fallback to server-side FFmpeg keyframe extraction if browser seek stalled
        if (state && state.loadSeq === req.seq && state.scrubUrl) {
          fetchServerScrubFrame(req);
          return;
        }
        scrubInFlight = false;
        pumpScrubQueue();
      };

      const onSeeked = () => finish(true);
      el.video.addEventListener("seeked", onSeeked, { once: true });
      const seekTimeoutMs = (state.waveformLoading || state.visualLoading) ? 900 : 380;
      timer = setTimeout(() => finish(false), seekTimeoutMs);

      try {
        el.video.currentTime = req.sec;
        if (!el.video.seeking && Math.abs((el.video.currentTime || 0) - req.sec) <= 0.015) {
          finish(true);
        }
      } catch {
        finish(false);
      }
      return;
    }

    scrubInFlight = true;
    fetchServerScrubFrame(req);
  }

  function prevOrAtKeyframe(sec) {
    const kfs = state && state.media ? state.media.keyframes : null;
    if (!kfs || kfs.length === 0) return sec;
    const idx = lowerBoundKF(kfs, sec + 0.005);
    if (idx <= 0) return kfs[0];
    return kfs[idx - 1];
  }

  function seekTo(sec, opts = {}) {
    const dur = getTotalDuration();
    if (dur <= 0) return;
    const clamped = Math.max(0, Math.min(dur, sec));
    const targetPlay = opts.autoPlay !== undefined ? opts.autoPlay : !el.video.paused;

    if (useRemux) {
      startRemuxAt(clamped, targetPlay);
    } else {
      remuxPinnedSec = null;
      const mySeek = ++seekVersion;
      clearTimeout(settleHideTimer);
      settleHideTimer = null;

      const cur = el.video.currentTime || 0;
      const canvasVisible = el.scrubCanvas && !el.scrubCanvas.classList.contains("hidden");
      if (canvasVisible || Math.abs(clamped - cur) > 0.20) {
        freezeCurrentFrameToCanvas();
      }

      const finishNativeSeek = () => {
        if (mySeek !== seekVersion) return;
        el.video.removeEventListener("seeked", finishNativeSeek);
        clearTimeout(settleHideTimer);
        settleHideTimer = null;
        if (!scrubbingRuler && !dragMarker && el.video.readyState >= 2 && el.video.videoWidth > 0) {
          hideScrubCanvas();
        }
        if (targetPlay && el.video.paused) {
          el.video.play().catch(() => {});
        }
        updateHUD();
        drawTimeline();
        drawOverview();
      };

      el.video.addEventListener("seeked", finishNativeSeek, { once: true });
      settleHideTimer = setTimeout(() => {
        if (mySeek === seekVersion && !scrubbingRuler && !dragMarker && el.video.readyState >= 2) {
          finishNativeSeek();
        }
      }, 450);

      el.video.currentTime = clamped;
      if (!el.video.seeking && Math.abs((el.video.currentTime || 0) - clamped) <= 0.02) {
        finishNativeSeek();
      }
    }
    updateHUD();
    drawTimeline();
    drawOverview();
  }

  function applyPlaybackRate() {
    const rate = Math.max(0.1, Math.min(10, parseFloat(el.selSpeed.value) || 1));
    if ("preservesPitch" in el.video) {
      el.video.preservesPitch = rate >= 0.5 && rate <= 2.5;
    }
    try {
      el.video.playbackRate = rate;
    } catch {}
  }

  function startRemuxAt(sec, autoPlay) {
    if (!state || !state.remuxUrl) return;
    const dur = getTotalDuration();
    const clamped = Math.max(0, Math.min(dur > 0 ? dur : sec, sec));
    freezeCurrentFrameToCanvas();
    useRemux = true;
    const kfAligned = !remuxTranscode;
    const startBase = kfAligned ? prevOrAtKeyframe(clamped) : clamped;
    remuxOffset = Math.max(0, startBase);
    remuxSubOffset = Math.max(0, clamped - remuxOffset);
    remuxPinnedSec = autoPlay ? null : clamped;
    el.remuxBadge.classList.remove("hidden");

    const myReq = ++remuxSeq;
    const tc = remuxTranscode ? "&transcode=1" : "&kf=1";
    el.video.src = `${state.remuxUrl}&start=${remuxOffset.toFixed(6)}${tc}`;
    applyPlaybackRate();

    if (!autoPlay) {
      requestScrubFrame(clamped, !kfAligned || remuxSubOffset > 0.04);
    }

    el.video.addEventListener("loadeddata", () => {
      if (myReq !== remuxSeq) return;
      if (remuxSubOffset > 0.03) {
        try {
          el.video.currentTime = remuxSubOffset;
        } catch {}
      }
      if (autoPlay) {
        el.video.play().catch(() => {});
      }
    }, { once: true });

    if (autoPlay) {
      el.video.play().catch(() => {});
    }
  }

  function nearestKeyframe(targetSec, minBound, maxBound) {
    const kfs = state && state.media ? state.media.keyframes : null;
    if (!kfs || kfs.length === 0) return targetSec;
    let best = kfs[0];
    let bestDist = Infinity;
    for (let i = 0; i < kfs.length; i++) {
      const kf = kfs[i];
      if (minBound !== undefined && kf <= minBound) continue;
      if (maxBound !== undefined && kf >= maxBound) continue;
      const d = Math.abs(kf - targetSec);
      if (d < bestDist) {
        bestDist = d;
        best = kf;
      } else if (kf > targetSec) {
        break;
      }
    }
    return best;
  }

  function lowerBoundKF(kfs, target) {
    let lo = 0;
    let hi = kfs.length;
    while (lo < hi) {
      const mid = (lo + hi) >> 1;
      if (kfs[mid] < target) lo = mid + 1;
      else hi = mid;
    }
    return lo;
  }

  function isNearKeyframe(sec) {
    const kfs = state && state.media ? state.media.keyframes : null;
    if (!kfs || kfs.length === 0) return false;
    let lo = 0, hi = kfs.length - 1;
    while (lo <= hi) {
      const mid = (lo + hi) >> 1;
      if (Math.abs(kfs[mid] - sec) <= 0.025) return true;
      if (kfs[mid] < sec) lo = mid + 1;
      else hi = mid - 1;
    }
    return false;
  }

  function decodeBase64Bytes(b64) {
    if (!b64) return null;
    const bin = atob(b64);
    const arr = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) {
      arr[i] = bin.charCodeAt(i);
    }
    return arr;
  }

  function ingestWaveform(wf) {
    if (!wf || !wf.ready || !wf.hasAudio || !wf.peaksB64) {
      wavePeaks = null;
      waveRMS = null;
      waveActivity = null;
      return;
    }
    wavePeaks = decodeBase64Bytes(wf.peaksB64);
    waveRMS = decodeBase64Bytes(wf.rmsB64);
    waveActivity = decodeBase64Bytes(wf.activityB64);
  }

  function ingestVisual(vd) {
    if (!vd || !vd.ready || !vd.deltaB64) {
      visualDeltas = null;
      visualScenes = null;
      return;
    }
    visualDeltas = decodeBase64Bytes(vd.deltaB64);
    visualScenes = decodeBase64Bytes(vd.sceneB64);
  }

  function applyTheme(theme) {
    const isLight = theme === "light";
    document.documentElement.setAttribute("data-theme", isLight ? "light" : "dark");
    el.iconSun.classList.toggle("hidden", !isLight);
    el.iconMoon.classList.toggle("hidden", isLight);
    drawTimeline();
    drawOverview();
  }

  function renderLoadOverlay() {
    const loading = !!(state && state.loading);
    el.loadingOverlay.classList.toggle("hidden", !loading);
    if (loading) {
      el.loadingTitle.textContent = `Loading ${state.loadingFile || "video"}…`;
      el.loadingText.textContent = state.loadingPhase || "Reading stream headers…";
      const pct = Math.max(0, Math.min(100, Math.round(loadDisplayPct)));
      el.loadPctText.textContent = `${pct}%`;
      el.loadProgressBar.style.width = `${pct}%`;
    }
  }

  function clearLoadBackdrop() {
    [el.loadBgA, el.loadBgB].forEach((img) => {
      if (!img) return;
      img.onload = null;
      img.classList.remove("visible");
      img.removeAttribute("src");
    });
    activeLoadBgSlot = 0;
  }

  function releaseVideoElement() {
    remuxSeq++;
    seekVersion++;
    useRemux = false;
    remuxTranscode = false;
    remuxOffset = 0;
    remuxSubOffset = 0;
    remuxPinnedSec = null;
    lastLoadedSeq = 0;
    el.remuxBadge.classList.add("hidden");
    el.btnPlayPause.classList.remove("is-playing");
    el.iconPlay.classList.remove("hidden");
    el.iconPause.classList.add("hidden");
    try {
      el.video.pause();
    } catch {}
    if (el.video.getAttribute("src") || el.video.currentSrc) {
      el.video.removeAttribute("src");
      el.video.load();
    }
  }

  function renderState(nextState) {
    state = nextState;
    if (!state) return;

    applyTheme(state.theme);
    if (!splitterDragging && typeof state.timelineScale === "number" && state.timelineScale > 0) {
      timelineScale = state.timelineScale;
    }

    el.btnModeJoin.classList.toggle("active", state.mode === "join");
    el.btnModeSplit.classList.toggle("active", state.mode === "split");
    el.btnSkipDropped.classList.toggle("active", !!state.skipDropped);
    el.btnToggleAudio.classList.toggle("active", state.showAudio !== false);
    el.btnToggleVisual.classList.toggle("active", state.showVisual !== false);

    // FFmpeg / FFprobe prerequisite status
    const toolsReady = !!(state.tools && state.tools.ready);
    if (toolsReady) {
      el.toolHealth.textContent = `FFmpeg ${state.tools.ffmpegVersion || "ready"} • Lossless stream copy`;
      el.toolHealth.className = "tool-health ok";
      el.prereqCard.classList.add("hidden");
    } else if (state.tools) {
      el.toolHealth.textContent = state.tools.error || "FFmpeg not found on PATH";
      el.toolHealth.className = "tool-health err";
      el.prereqCard.classList.remove("hidden");
      const hasFF = !!state.tools.ffmpegPath;
      const hasFP = !!state.tools.ffprobePath;
      el.prereqFFmpeg.textContent = `${hasFF ? "✔" : "✖"} ffmpeg`;
      el.prereqFFmpeg.classList.toggle("ok", hasFF);
      el.prereqFFprobe.textContent = `${hasFP ? "✔" : "✖"} ffprobe`;
      el.prereqFFprobe.classList.toggle("ok", hasFP);
    }

    if (state.loadError) {
      showToast(state.loadError, true);
    }

    const hasMedia = !!(state.media && state.media.durationSec > 0);
    el.emptyView.classList.toggle("hidden", hasMedia || state.loading);
    el.editorView.classList.toggle("hidden", !hasMedia && !state.loading);
    if (el.hudOverlay) {
      el.hudOverlay.classList.toggle("hidden", !hasMedia);
    }
    if (el.editorShelf) {
      if (!hasMedia) {
        el.editorShelf.classList.add("collapsed");
        el.editorShelf.classList.remove("shelf-open");
      } else if (el.editorShelf.classList.contains("collapsed")) {
        const reduced = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
        if (reduced) {
          el.editorShelf.classList.remove("collapsed");
          el.editorShelf.classList.add("shelf-open");
        } else {
          requestAnimationFrame(() => {
            requestAnimationFrame(() => {
              if (state && state.media && state.media.durationSec > 0 && el.editorShelf) {
                el.editorShelf.classList.remove("collapsed");
              }
            });
          });
        }
      }
    }

    if (state.loading) {
      loadTargetPct = Math.max(loadTargetPct, state.loadingPct || 2);
      if (loadDisplayPct === 0) loadDisplayPct = loadTargetPct;
      if (state.loadSeq !== lastBackdropSeq) {
        lastBackdropSeq = state.loadSeq;
        clearLoadBackdrop();
      }
    } else {
      loadTargetPct = 0;
      loadDisplayPct = 0;
      clearLoadBackdrop();
    }
    renderLoadOverlay();

    if (hasMedia) {
      el.fileBadge.classList.remove("hidden");
      el.fileName.textContent = state.media.fileName;
      const fpsText = state.media.fps > 0 ? `${state.media.fps.toFixed(2)}fps` : "";
      el.fileMeta.textContent = `${state.media.width}×${state.media.height} • ${state.media.videoCodec}${fpsText ? " • " + fpsText : ""} • ${state.media.keyframes.length} KFs`;
      el.hudDuration.textContent = fmtTime(state.media.durationSec);

      if (state.loadSeq !== lastLoadedSeq) {
        lastLoadedSeq = state.loadSeq;
        useRemux = false;
        remuxTranscode = false;
        remuxOffset = 0;
        remuxSubOffset = 0;
        remuxPinnedSec = null;
        remuxSeq++;
        seekVersion++;
        scrubbingRuler = false;
        scrubPreviewSec = null;
        dragMarker = null;
        ghostSnapSec = null;
        clearScrubQueue();
        hideScrubCanvas();
        zoom = 1;
        viewStart = 0;
        wavePeaks = null;
        waveRMS = null;
        waveActivity = null;
        visualDeltas = null;
        visualScenes = null;
        audioTargetPct = state.waveformLoading ? 3 : 0;
        audioDisplayPct = audioTargetPct;
        visualTargetPct = state.visualLoading ? 3 : 0;
        visualDisplayPct = visualTargetPct;
        el.remuxBadge.classList.add("hidden");
        el.btnPlayPause.classList.remove("is-playing");
        el.video.src = state.videoUrl;
        applyPlaybackRate();
      }

      if (state.waveform) {
        ingestWaveform(state.waveform);
      }
      if (state.visual) {
        ingestVisual(state.visual);
      }
      updateZoomUI();
    } else {
      el.fileBadge.classList.add("hidden");
      scrubPreviewSec = null;
      dragMarker = null;
      ghostSnapSec = null;
      releaseVideoElement();
      clearScrubQueue();
      hideScrubCanvas();
      wavePeaks = null;
      waveRMS = null;
      waveActivity = null;
      visualDeltas = null;
      visualScenes = null;
    }

    applyTimelineLayout();
    renderSegments();
    updateExportButton();
    updateHUD();
    drawTimeline();
    drawOverview();
  }

  function updateExportButton() {
    if (!state || !state.segments || state.segments.length === 0 || state.exporting) {
      el.btnExport.disabled = true;
      el.lblExport.textContent = "Export";
      return;
    }
    const kept = state.segments.filter((s) => s.keep);
    el.btnExport.disabled = kept.length === 0;
    if (kept.length === 0) {
      el.lblExport.textContent = "Nothing Kept";
    } else if (state.mode === "join") {
      el.lblExport.textContent = "Export _edit";
    } else {
      el.lblExport.textContent = `Export ${kept.length} Clip${kept.length > 1 ? "s" : ""}`;
    }
  }

  function syncTimelineDOMPositions() {
    const span = getViewSpan();
    if (span <= 0) return;
    const viewEnd = viewStart + span;

    for (const item of segDomItems) {
      if (item.end <= viewStart || item.start >= viewEnd) {
        item.el.style.display = "none";
        continue;
      }
      const visStart = Math.max(item.start, viewStart);
      const visEnd = Math.min(item.end, viewEnd);
      const leftPct = ((visStart - viewStart) / span) * 100;
      const widthPct = ((visEnd - visStart) / span) * 100;
      item.el.style.display = "";
      item.el.style.left = `${leftPct}%`;
      item.el.style.width = `${widthPct}%`;
    }

    for (let i = 0; i < markerDomItems.length; i++) {
      const m = markerDomItems[i];
      const sec = dragMarker && dragMarker.idx === i ? dragMarker.previewSec : m.time;
      if (sec < viewStart || sec > viewEnd) {
        m.el.style.display = "none";
      } else {
        m.el.style.display = "flex";
        m.el.style.left = `${((sec - viewStart) / span) * 100}%`;
      }
    }
  }

  function renderSegments() {
    el.segmentsHit.innerHTML = "";
    el.markerLayer.innerHTML = "";
    el.segmentPills.innerHTML = "";
    segDomItems = [];
    markerDomItems = [];

    if (!state || !state.media || !state.segments) {
      el.outputSummary.textContent = "";
      return;
    }

    const total = state.media.durationSec;
    if (total <= 0) return;

    state.segments.forEach((seg, idx) => {
      const box = document.createElement("div");
      box.className = "seg-hit-box";
      box.title = `#${seg.index} (${fmtTime(seg.start)}–${fmtTime(seg.end)}) • Click to ${seg.keep ? "DROP" : "KEEP"}`;
      box.addEventListener("click", async () => {
        const snap = await window.api.toggleSegment(idx);
        renderState(snap);
      });
      el.segmentsHit.appendChild(box);
      segDomItems.push({ el: box, start: seg.start, end: seg.end });

      const pill = document.createElement("button");
      pill.className = `seg-pill${seg.keep ? "" : " drop"}`;
      pill.title = "Click to toggle Keep/Drop • Double-click to seek to start";
      pill.textContent = `#${String(seg.index).padStart(2, "0")} · ${fmtTime(seg.start)}–${fmtTime(seg.end)} · ${seg.keep ? "KEEP" : "DROP"}`;
      pill.addEventListener("click", async () => {
        const snap = await window.api.toggleSegment(idx);
        renderState(snap);
      });
      pill.addEventListener("dblclick", () => {
        seekTo(seg.start);
      });
      el.segmentPills.appendChild(pill);
    });

    (state.markers || []).forEach((mSec, idx) => {
      const handle = document.createElement("div");
      handle.className = "cut-handle";
      const displaySec = dragMarker && dragMarker.idx === idx ? dragMarker.previewSec : mSec;
      handle.title = `Cut @ ${fmtTime(displaySec)} (drag to move, right-click or × to delete)`;

      const knob = document.createElement("div");
      knob.className = "cut-knob";
      handle.appendChild(knob);

      const rmBtn = document.createElement("button");
      rmBtn.className = "cut-remove";
      rmBtn.textContent = "×";
      rmBtn.title = "Remove cut point";
      rmBtn.addEventListener("mousedown", (e) => e.stopPropagation());
      rmBtn.addEventListener("click", async (e) => {
        e.stopPropagation();
        const snap = await window.api.removeMarker(idx);
        renderState(snap);
      });
      handle.appendChild(rmBtn);

      handle.addEventListener("contextmenu", async (e) => {
        e.preventDefault();
        const snap = await window.api.removeMarker(idx);
        renderState(snap);
      });

      handle.addEventListener("mousedown", (e) => {
        if (e.button !== 0) return;
        e.stopPropagation();
        startMarkerDrag(idx);
      });

      el.markerLayer.appendChild(handle);
      markerDomItems.push({ el: handle, time: mSec });
    });

    syncTimelineDOMPositions();

    const kept = state.segments.filter((s) => s.keep);
    const keptDur = kept.reduce((acc, s) => acc + s.duration, 0);
    const previewNames = state.outputPreview || [];
    if (kept.length === 0) {
      el.outputSummary.textContent = "0 segments kept — click a segment to keep it";
    } else if (state.mode === "join") {
      el.outputSummary.textContent = `Keeping ${kept.length}/${state.segments.length} (${fmtTime(keptDur)}) → ${previewNames[0] || "output_edit"}`;
    } else {
      const listStr = previewNames.length <= 3 ? previewNames.join(", ") : `${previewNames[0]} … ${previewNames[previewNames.length - 1]}`;
      el.outputSummary.textContent = `Keeping ${kept.length}/${state.segments.length} (${fmtTime(keptDur)}) → ${listStr}`;
    }
  }

  function clientXToTime(clientX) {
    const rect = el.timelineWrap.getBoundingClientRect();
    const ratio = Math.max(0, Math.min(1, (clientX - rect.left) / Math.max(1, rect.width)));
    return viewStart + ratio * getViewSpan();
  }

  function startMarkerDrag(markerIdx) {
    const total = getTotalDuration();
    const markers = state.markers || [];
    const minBound = markerIdx > 0 ? markers[markerIdx - 1] + 0.002 : 0.02;
    const maxBound = markerIdx + 1 < markers.length ? markers[markerIdx + 1] - 0.002 : total - 0.02;
    const playheadBefore = getCurrentTime();

    if (!el.video.paused) {
      el.video.pause();
    }
    freezeCurrentFrameToCanvas();
    dragMarker = { idx: markerIdx, previewSec: markers[markerIdx], playheadBefore };
    requestScrubFrame(markers[markerIdx], false);

    const onMove = (ev) => {
      const raw = clientXToTime(ev.clientX);
      const snapped = nearestKeyframe(raw, minBound, maxBound);
      dragMarker.previewSec = snapped;
      ghostSnapSec = snapped;
      requestScrubFrame(snapped, false);
      syncTimelineDOMPositions();
      showHoverTime(ev.clientX, `KF ${fmtTime(snapped)}`);
      drawTimeline();
      drawOverview();
    };

    const onUp = async () => {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
      const targetSec = dragMarker.previewSec;
      dragMarker = null;
      ghostSnapSec = null;
      clearScrubQueue();
      el.hoverTooltip.classList.add("hidden");
      const res = await window.api.moveMarker(markerIdx, targetSec);
      renderState(res.snapshot);
      seekTo(res.changed ? res.snappedSec : targetSec, { autoPlay: false });
    };

    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
  }

  function showHoverTime(clientX, label) {
    const rect = el.timelineWrap.getBoundingClientRect();
    const x = Math.max(36, Math.min(rect.width - 36, clientX - rect.left));
    el.hoverTooltip.style.left = `${x}px`;
    el.hoverTooltip.textContent = label;
    el.hoverTooltip.classList.remove("hidden");
  }

  function isTimeKept(t) {
    const segs = state && state.segments ? state.segments : null;
    if (!segs || segs.length === 0) return true;
    for (let i = 0; i < segs.length; i++) {
      if (t >= segs[i].start && t <= segs[i].end) {
        return segs[i].keep;
      }
    }
    return true;
  }

  function drawLaneProgressBar(ctx, w, laneTop, laneH, label, pct, c1, c2, isLight) {
    const padX = 14;
    const barW = Math.min(340, Math.max(120, w - padX * 2));
    const barH = 7;
    const barX = padX;
    const barY = laneTop + Math.round(laneH * 0.56);
    const clampedPct = Math.max(2, Math.min(100, pct));

    ctx.fillStyle = isLight ? "#475569" : "#94a3b8";
    ctx.font = "600 11px 'Segoe UI', sans-serif";
    ctx.textBaseline = "middle";
    ctx.fillText(`${label}  ${Math.round(clampedPct)}%`, barX, laneTop + Math.round(laneH * 0.30));

    // Track
    ctx.fillStyle = isLight ? "rgba(15, 23, 42, 0.12)" : "rgba(255, 255, 255, 0.08)";
    ctx.beginPath();
    ctx.roundRect(barX, barY, barW, barH, 4);
    ctx.fill();

    // Gradient Fill
    const fillW = Math.max(8, (clampedPct / 100) * barW);
    const grad = ctx.createLinearGradient(barX, 0, barX + barW, 0);
    grad.addColorStop(0, c1);
    grad.addColorStop(1, c2);
    ctx.fillStyle = grad;
    ctx.beginPath();
    ctx.roundRect(barX, barY, fillW, barH, 4);
    ctx.fill();

    // Animated shimmer sweep
    const phase = ((performance.now() * 0.0012) % 1.4) - 0.2;
    const shimX = barX + phase * fillW;
    if (fillW > 24 && shimX >= barX - 24 && shimX <= barX + fillW) {
      ctx.save();
      ctx.beginPath();
      ctx.roundRect(barX, barY, fillW, barH, 4);
      ctx.clip();
      const sGrad = ctx.createLinearGradient(shimX - 24, 0, shimX + 24, 0);
      sGrad.addColorStop(0, "rgba(255,255,255,0)");
      sGrad.addColorStop(0.5, "rgba(255,255,255,0.45)");
      sGrad.addColorStop(1, "rgba(255,255,255,0)");
      ctx.fillStyle = sGrad;
      ctx.fillRect(shimX - 24, barY, 48, barH);
      ctx.restore();
    }
  }

  function drawTimeline() {
    const canvas = el.timelineCanvas;
    const wrap = el.timelineWrap;
    if (!canvas || !wrap) return;

    const lay = getTimelineLayout();
    const dpr = window.devicePixelRatio || 1;
    const w = wrap.clientWidth;
    const h = wrap.clientHeight || lay.totalH;
    if (w <= 0 || h <= 0) return;

    if (canvas.width !== Math.round(w * dpr) || canvas.height !== Math.round(h * dpr)) {
      canvas.width = Math.round(w * dpr);
      canvas.height = Math.round(h * dpr);
    }

    const ctx = canvas.getContext("2d");
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, w, h);

    const isLight = document.documentElement.getAttribute("data-theme") === "light";
    const { rulerH, audioH, visualH, segH, audioBottom, scrubBottom } = lay;

    // Zone 1: Ruler background
    ctx.fillStyle = isLight ? "#e2e8f0" : "#151c29";
    ctx.fillRect(0, 0, w, rulerH);

    // Zone 2: Audio Histogram lane background
    if (audioH > 0) {
      ctx.fillStyle = isLight ? "#eef2f6" : "#0e141f";
      ctx.fillRect(0, rulerH, w, audioH);
    }

    // Zone 3: Visual Delta lane background
    if (visualH > 0) {
      ctx.fillStyle = isLight ? "#e8edf5" : "#101624";
      ctx.fillRect(0, audioBottom, w, visualH);
    }

    ctx.strokeStyle = isLight ? "rgba(15,23,42,0.12)" : "rgba(255,255,255,0.08)";
    ctx.beginPath();
    ctx.moveTo(0, rulerH);
    ctx.lineTo(w, rulerH);
    if (audioH > 0 && visualH > 0) {
      ctx.moveTo(0, audioBottom);
      ctx.lineTo(w, audioBottom);
    }
    ctx.moveTo(0, scrubBottom);
    ctx.lineTo(w, scrubBottom);
    ctx.stroke();

    const total = getTotalDuration();
    if (!state || !state.media || total <= 0) return;

    const span = getViewSpan();
    if (span <= 0) return;
    const viewEnd = viewStart + span;
    const timeToX = (t) => ((t - viewStart) / span) * w;

    // Fine zoom-aware ruler ticks
    const stepOptions = [0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10, 15, 30, 60, 120, 300, 600, 1800];
    const targetTicks = Math.max(4, Math.floor(w / 92));
    let tickStep = stepOptions[stepOptions.length - 1];
    for (const s of stepOptions) {
      if (span / s <= targetTicks) {
        tickStep = s;
        break;
      }
    }

    const firstTick = Math.floor(viewStart / tickStep) * tickStep;
    ctx.fillStyle = isLight ? "#475569" : "#94a3b8";
    ctx.font = "600 10.5px 'Segoe UI', sans-serif";
    ctx.textBaseline = "middle";
    for (let t = firstTick; t <= viewEnd + tickStep * 0.001; t += tickStep) {
      if (t < viewStart - tickStep * 0.25) continue;
      const x = timeToX(t);
      if (x < -2 || x > w + 2) continue;
      ctx.strokeStyle = isLight ? "rgba(15,23,42,0.18)" : "rgba(255,255,255,0.14)";
      ctx.beginPath();
      ctx.moveTo(x, rulerH - 7);
      ctx.lineTo(x, rulerH);
      ctx.stroke();
      if (x >= 0 && x < w - 38) {
        const lbl = tickStep < 1 ? fmtTime(t) : fmtTime(t).replace(/\.00$/, "");
        ctx.fillText(lbl, x + 4, 13);
      }
    }

    const segs = state.segments || [];

    const forEachBarSlot = (bins, cb) => {
      const pxPerBin = (w * (total / span)) / bins;
      if (pxPerBin > 3.2) {
        const bStart = Math.max(0, Math.floor((viewStart / total) * bins));
        const bEnd = Math.min(bins - 1, Math.ceil((viewEnd / total) * bins));
        for (let b = bStart; b <= bEnd; b++) {
          const t0 = (b / bins) * total;
          const t1 = ((b + 1) / bins) * total;
          const x0 = timeToX(t0);
          const x1 = timeToX(t1);
          if (x1 <= 0 || x0 >= w) continue;
          const rawW = x1 - x0;
          const gap = Math.min(2, Math.max(1, rawW * 0.14));
          const drawX = Math.max(0, x0);
          const drawW = Math.max(2, Math.min(w - drawX, rawW - gap));
          cb(drawX, drawW, b, b, 0.5 * (t0 + t1));
        }
        return;
      }
      const colStep = 3;
      const barW = 2;
      for (let x = 0; x < w; x += colStep) {
        const t0 = viewStart + (x / w) * span;
        const t1 = viewStart + ((x + colStep) / w) * span;
        let b0 = Math.floor((t0 / total) * bins);
        let b1 = Math.floor((t1 / total) * bins);
        if (b0 < 0) b0 = 0;
        if (b1 >= bins) b1 = bins - 1;
        if (b1 < b0) b1 = b0;
        cb(x, barW, b0, b1, 0.5 * (t0 + t1));
      }
    };

    // Audio Sound-Level + Talking/Noise Histogram
    if (audioH > 0) {
      if (wavePeaks && wavePeaks.length > 0) {
        const bins = wavePeaks.length;
        const maxBarH = audioH - 12;
        let segCursor = 0;

        forEachBarSlot(bins, (x, barW, b0, b1, tMid) => {
          let peakMax = 0;
          let rmsSum = 0;
          let hasTalking = false;
          let hasNoise = false;
          for (let b = b0; b <= b1; b++) {
            if (wavePeaks[b] > peakMax) peakMax = wavePeaks[b];
            if (waveRMS) rmsSum += waveRMS[b];
            if (waveActivity) {
              if (waveActivity[b] === 1) hasTalking = true;
              else if (waveActivity[b] === 2) hasNoise = true;
            }
          }
          const rmsAvg = waveRMS ? rmsSum / (b1 - b0 + 1) : 0;
          const act = hasTalking ? 1 : (hasNoise ? 2 : 0);

          while (segCursor < segs.length && segs[segCursor].end < tMid) {
            segCursor++;
          }
          const kept = segCursor < segs.length ? segs[segCursor].keep : isTimeKept(tMid);

          if (peakMax <= 1) {
            ctx.fillStyle = kept
              ? (isLight ? "rgba(2, 132, 199, 0.16)" : "rgba(56, 189, 248, 0.14)")
              : (isLight ? "rgba(220, 38, 38, 0.10)" : "rgba(239, 68, 68, 0.10)");
            ctx.fillRect(x, audioBottom - 2, barW, 2);
            return;
          }

          const pH = Math.max(2, (peakMax / 255) * maxBarH);
          const rH = Math.max(1, (rmsAvg / 255) * maxBarH);

          let peakCol;
          let rmsCol;
          if (!kept) {
            peakCol = isLight ? "rgba(220, 38, 38, 0.22)" : "rgba(248, 113, 113, 0.24)";
            rmsCol = isLight ? "rgba(185, 28, 28, 0.45)" : "rgba(248, 113, 113, 0.45)";
          } else if (act === 1) {
            // Talking / Speech: Amber-Gold
            peakCol = isLight ? "rgba(217, 119, 6, 0.44)" : "rgba(245, 158, 11, 0.45)";
            rmsCol = isLight ? "#d97706" : "#fbbf24";
          } else if (act === 2) {
            // Noise / Transient: Rose-Magenta
            peakCol = isLight ? "rgba(219, 39, 119, 0.40)" : "rgba(236, 72, 153, 0.42)";
            rmsCol = isLight ? "#db2777" : "#f472b6";
          } else {
            // Ambient / Baseline: Cyan
            peakCol = isLight ? "rgba(14, 165, 233, 0.36)" : "rgba(56, 189, 248, 0.34)";
            rmsCol = isLight ? "#0284c7" : "#22d3ee";
          }

          ctx.fillStyle = peakCol;
          ctx.fillRect(x, audioBottom - pH, barW, pH);
          if (rH > 1) {
            ctx.fillStyle = rmsCol;
            ctx.fillRect(x, audioBottom - rH, barW, rH);
          }
        });

        // Micro-legend in top-left of SOUND lane
        ctx.font = "700 9.5px 'Segoe UI', sans-serif";
        ctx.textBaseline = "middle";
        const ly = rulerH + 9;
        ctx.fillStyle = isLight ? "rgba(15,23,42,0.52)" : "rgba(148,163,184,0.68)";
        ctx.fillText("SOUND", 7, ly);
        if (waveActivity) {
          ctx.fillStyle = isLight ? "#d97706" : "#fbbf24";
          ctx.beginPath();
          ctx.arc(54, ly, 3, 0, Math.PI * 2);
          ctx.fill();
          ctx.fillStyle = isLight ? "rgba(15,23,42,0.58)" : "rgba(226,232,240,0.72)";
          ctx.font = "600 9.5px 'Segoe UI', sans-serif";
          ctx.fillText("Talking", 60, ly);

          ctx.fillStyle = isLight ? "#db2777" : "#f472b6";
          ctx.beginPath();
          ctx.arc(104, ly, 3, 0, Math.PI * 2);
          ctx.fill();
          ctx.fillStyle = isLight ? "rgba(15,23,42,0.58)" : "rgba(226,232,240,0.72)";
          ctx.fillText("Noise", 110, ly);
        }
      } else if (state.waveformLoading) {
        drawLaneProgressBar(ctx, w, rulerH, audioH, "Analyzing audio & voice…", audioDisplayPct, "#06b6d4", "#22d3ee", isLight);
      } else {
        ctx.fillStyle = "#64748b";
        ctx.font = "600 11px 'Segoe UI', sans-serif";
        ctx.fillText(state.media.hasAudio ? "SOUND" : "No audio track", 10, rulerH + audioH / 2);
      }
    }

    // Visual Frame-Delta & Scene-Cut Histogram
    if (visualH > 0) {
      if (visualDeltas && visualDeltas.length > 0) {
        const bins = visualDeltas.length;
        const maxBarH = visualH - 12;
        let segCursor = 0;

        forEachBarSlot(bins, (x, barW, b0, b1, tMid) => {
          let deltaMax = 0;
          let sceneMax = 0;
          for (let b = b0; b <= b1; b++) {
            if (visualDeltas[b] > deltaMax) deltaMax = visualDeltas[b];
            if (visualScenes && visualScenes[b] > sceneMax) sceneMax = visualScenes[b];
          }

          while (segCursor < segs.length && segs[segCursor].end < tMid) {
            segCursor++;
          }
          const kept = segCursor < segs.length ? segs[segCursor].keep : isTimeKept(tMid);

          if (deltaMax <= 1 && sceneMax <= 1) {
            ctx.fillStyle = kept
              ? (isLight ? "rgba(109, 40, 217, 0.16)" : "rgba(139, 92, 246, 0.16)")
              : (isLight ? "rgba(220, 38, 38, 0.10)" : "rgba(239, 68, 68, 0.10)");
            ctx.fillRect(x, scrubBottom - 2, barW, 2);
            return;
          }

          const dH = Math.max(2, (deltaMax / 255) * maxBarH);
          const isSceneCut = sceneMax >= 155;

          if (isSceneCut) {
            const sH = Math.max(dH, (sceneMax / 255) * maxBarH);
            ctx.fillStyle = kept
              ? (isLight ? "#d97706" : "#f59e0b")
              : "rgba(245, 158, 11, 0.45)";
            ctx.fillRect(x, scrubBottom - sH, barW, sH);
          } else {
            ctx.fillStyle = kept
              ? (isLight ? "rgba(109, 40, 217, 0.45)" : "rgba(139, 92, 246, 0.50)")
              : (isLight ? "rgba(220, 38, 38, 0.22)" : "rgba(248, 113, 113, 0.24)");
            ctx.fillRect(x, scrubBottom - dH, barW, dH);

            if (dH > 4) {
              ctx.fillStyle = kept
                ? (isLight ? "#7c3aed" : "#a78bfa")
                : (isLight ? "rgba(185, 28, 28, 0.42)" : "rgba(248, 113, 113, 0.42)");
              ctx.fillRect(x, scrubBottom - dH, barW, Math.max(2, dH * 0.35));
            }
          }
        });

        ctx.fillStyle = isLight ? "rgba(15,23,42,0.48)" : "rgba(148,163,184,0.68)";
        ctx.font = "700 9.5px 'Segoe UI', sans-serif";
        ctx.fillText("VISUAL Δ", 7, audioBottom + 9);
      } else if (state.visualLoading) {
        drawLaneProgressBar(ctx, w, audioBottom, visualH, "Analyzing visual frame deltas…", visualDisplayPct, "#8b5cf6", "#f59e0b", isLight);
      } else {
        ctx.fillStyle = "#64748b";
        ctx.font = "600 11px 'Segoe UI', sans-serif";
        ctx.fillText("VISUAL Δ", 10, audioBottom + visualH / 2);
      }
    }

    // Segments lane
    for (const seg of segs) {
      if (seg.end <= viewStart || seg.start >= viewEnd) continue;
      const x1 = Math.max(0, timeToX(seg.start));
      const x2 = Math.min(w, timeToX(seg.end));
      const sw = Math.max(1, x2 - x1);

      if (seg.keep) {
        ctx.fillStyle = isLight ? "rgba(16, 185, 129, 0.24)" : "rgba(16, 185, 129, 0.28)";
        ctx.fillRect(x1, scrubBottom, sw, segH);
        ctx.fillStyle = "#10b981";
        ctx.fillRect(x1, scrubBottom, sw, 3);
      } else {
        ctx.fillStyle = isLight ? "rgba(220, 38, 38, 0.12)" : "rgba(239, 68, 68, 0.14)";
        ctx.fillRect(x1, scrubBottom, sw, segH);
        ctx.fillStyle = "rgba(239, 68, 68, 0.55)";
        ctx.fillRect(x1, scrubBottom, sw, 2);
      }

      if (sw > 58) {
        ctx.fillStyle = seg.keep
          ? (isLight ? "#065f46" : "#a7f3d0")
          : (isLight ? "#991b1b" : "#fca5a5");
        ctx.font = "700 11px 'Segoe UI', sans-serif";
        ctx.fillText(`#${seg.index} ${seg.keep ? "KEEP" : "DROP"}`, x1 + 8, scrubBottom + segH / 2 - 6);
        if (sw > 92) {
          ctx.font = "500 10.5px 'Cascadia Mono', monospace";
          ctx.fillText(fmtTime(seg.duration), x1 + 8, scrubBottom + segH / 2 + 10);
        }
      }
    }

    // Keyframe ticks + diamond nodes when zoomed in
    const kfs = state.media.keyframes || [];
    if (kfs.length > 0) {
      const i0 = lowerBoundKF(kfs, viewStart);
      let i1 = lowerBoundKF(kfs, viewEnd);
      if (i1 >= kfs.length) i1 = kfs.length - 1;
      const visibleCount = Math.max(1, i1 - i0 + 1);
      const avgSpacing = w / visibleCount;
      const showDiamonds = avgSpacing >= 12;
      const stride = Math.max(1, Math.floor(visibleCount / Math.max(1, w / 3)));

      ctx.fillStyle = isLight ? "rgba(2, 132, 199, 0.68)" : "rgba(56, 189, 248, 0.65)";
      for (let i = i0; i <= i1; i += stride) {
        const kx = timeToX(kfs[i]);
        if (kx < -2 || kx > w + 2) continue;
        if (showDiamonds) {
          const cy = h - 7;
          ctx.beginPath();
          ctx.moveTo(kx, cy - 3.5);
          ctx.lineTo(kx + 3.5, cy);
          ctx.lineTo(kx, cy + 3.5);
          ctx.lineTo(kx - 3.5, cy);
          ctx.closePath();
          ctx.fill();
        } else {
          ctx.fillRect(kx, h - 6, 1.5, 6);
        }
      }
    }

    // Cut lines across all active lanes
    (state.markers || []).forEach((mSec, idx) => {
      const sec = dragMarker && dragMarker.idx === idx ? dragMarker.previewSec : mSec;
      if (sec < viewStart || sec > viewEnd) return;
      const mx = timeToX(sec);
      ctx.strokeStyle = "#f59e0b";
      ctx.lineWidth = 2;
      ctx.beginPath();
      ctx.moveTo(mx, rulerH);
      ctx.lineTo(mx, h);
      ctx.stroke();
    });

    // Ghost snap line during marker drag
    if (ghostSnapSec !== null && ghostSnapSec >= viewStart && ghostSnapSec <= viewEnd) {
      const gx = timeToX(ghostSnapSec);
      ctx.strokeStyle = "#38bdf8";
      ctx.lineWidth = 1.5;
      ctx.setLineDash([4, 3]);
      ctx.beginPath();
      ctx.moveTo(gx, 0);
      ctx.lineTo(gx, h);
      ctx.stroke();
      ctx.setLineDash([]);
    }

    // Playhead needle
    const cur = getCurrentTime();
    if (cur >= viewStart && cur <= viewEnd) {
      const px = Math.max(0, Math.min(w, timeToX(cur)));
      ctx.strokeStyle = isLight ? "#0f172a" : "#ffffff";
      ctx.lineWidth = 2;
      ctx.beginPath();
      ctx.moveTo(px, 0);
      ctx.lineTo(px, h);
      ctx.stroke();

      ctx.fillStyle = "#38bdf8";
      ctx.beginPath();
      ctx.moveTo(px - 5, 0);
      ctx.lineTo(px + 5, 0);
      ctx.lineTo(px, 7);
      ctx.closePath();
      ctx.fill();
    }
  }

  function drawOverview() {
    const canvas = el.overviewCanvas;
    const wrap = el.overviewWrap;
    if (!canvas || !wrap) return;

    const dpr = window.devicePixelRatio || 1;
    const w = wrap.clientWidth;
    const h = wrap.clientHeight;
    if (w <= 0 || h <= 0) return;

    if (canvas.width !== Math.round(w * dpr) || canvas.height !== Math.round(h * dpr)) {
      canvas.width = Math.round(w * dpr);
      canvas.height = Math.round(h * dpr);
    }

    const ctx = canvas.getContext("2d");
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, w, h);

    const total = getTotalDuration();
    if (!state || !state.media || total <= 0) return;
    const isLight = document.documentElement.getAttribute("data-theme") === "light";

    for (const seg of state.segments || []) {
      const x1 = (seg.start / total) * w;
      const x2 = (seg.end / total) * w;
      ctx.fillStyle = seg.keep
        ? "rgba(16, 185, 129, 0.35)"
        : (isLight ? "rgba(220, 38, 38, 0.18)" : "rgba(239, 68, 68, 0.22)");
      ctx.fillRect(x1, 0, Math.max(1, x2 - x1), h);
    }

    ctx.fillStyle = "#f59e0b";
    for (const mSec of state.markers || []) {
      const mx = (mSec / total) * w;
      ctx.fillRect(mx - 1, 0, 2, h);
    }

    const cur = getCurrentTime();
    const px = (cur / total) * w;
    ctx.fillStyle = isLight ? "#0f172a" : "#ffffff";
    ctx.fillRect(px - 0.75, 0, 1.5, h);

    const span = getViewSpan();
    const vx1 = (viewStart / total) * w;
    const vw = Math.max(10, (span / total) * w);
    ctx.fillStyle = isLight ? "rgba(2, 132, 199, 0.16)" : "rgba(56, 189, 248, 0.20)";
    ctx.fillRect(vx1, 0, vw, h);
    ctx.strokeStyle = "#38bdf8";
    ctx.lineWidth = 1.5;
    ctx.strokeRect(vx1 + 0.75, 0.75, Math.max(4, vw - 1.5), h - 1.5);
  }

  function updateHUD() {
    const cur = getCurrentTime();
    el.hudTime.textContent = fmtTime(cur);
    el.kfBadge.classList.toggle("on", isNearKeyframe(cur));
  }

  function enforceSkipDropped() {
    if (!state || !state.skipDropped || !state.segments || el.video.paused) return;
    const cur = getCurrentTime();
    for (let i = 0; i < state.segments.length; i++) {
      const seg = state.segments[i];
      if (cur >= seg.start && cur < seg.end - 0.03 && !seg.keep) {
        const nextKept = state.segments.slice(i + 1).find((s) => s.keep);
        if (nextKept) {
          seekTo(nextKept.start);
        } else {
          el.video.pause();
        }
        break;
      }
    }
  }

  function animLoop() {
    if (state && state.loading) {
      loadDisplayPct += (loadTargetPct - loadDisplayPct) * 0.22;
      renderLoadOverlay();
    }

    if (state && state.media) {
      enforceSkipDropped();

      if (state.waveformLoading) {
        audioDisplayPct += (Math.max(audioTargetPct, 4) - audioDisplayPct) * 0.20;
      }
      if (state.visualLoading) {
        visualDisplayPct += (Math.max(visualTargetPct, 4) - visualDisplayPct) * 0.20;
      }

      if (!el.video.paused && !scrubbingRuler && !dragMarker && zoom > 1.001) {
        const cur = getCurrentTime();
        const span = getViewSpan();
        const viewEnd = viewStart + span;
        if (cur >= viewStart && cur <= viewEnd && (cur - viewStart) > 0.92 * span) {
          viewStart = clampViewStart(cur - 0.15 * span);
          syncTimelineDOMPositions();
        }
      }

      updateHUD();
      drawTimeline();
      drawOverview();
    }
    requestAnimationFrame(animLoop);
  }

  function pulseButton(btn) {
    if (!btn || btn.disabled) return;
    btn.classList.remove("is-pulsing");
    void btn.offsetWidth;
    btn.classList.add("is-pulsing");
  }

  document.addEventListener("pointermove", (e) => {
    const target = e.target && e.target.closest ? e.target.closest(".btn, .btn-lane, .btn-mini, .mode-btn") : null;
    if (!target) return;
    const rect = target.getBoundingClientRect();
    target.style.setProperty("--mx", `${(e.clientX - rect.left).toFixed(1)}px`);
    target.style.setProperty("--my", `${(e.clientY - rect.top).toFixed(1)}px`);
  }, { passive: true });

  document.addEventListener("click", (e) => {
    const target = e.target && e.target.closest ? e.target.closest(".btn, .btn-lane, .btn-mini, .mode-btn") : null;
    if (target) {
      pulseButton(target);
    }
  });

  document.addEventListener("animationend", (e) => {
    if (e.animationName === "btnPulseRing" && e.target) {
      e.target.classList.remove("is-pulsing");
    }
  });

  // --- Wire Controls ---
  el.btnOpen.addEventListener("click", () => window.api.openFileDialog());
  el.btnEmptyBrowse.addEventListener("click", () => window.api.openFileDialog());

  el.btnCancelLoad.addEventListener("click", async () => {
    const snap = await window.api.cancelLoad();
    renderState(snap);
    showToast("Video load cancelled");
  });

  if (el.btnCloseFile) {
    el.btnCloseFile.addEventListener("click", async () => {
      releaseVideoElement();
      scrubPreviewSec = null;
      dragMarker = null;
      ghostSnapSec = null;
      clearScrubQueue();
      hideScrubCanvas();
      const snap = await window.api.closeFile();
      renderState(snap);
    });
  }

  el.btnCopyWinget.addEventListener("click", async () => {
    try {
      await navigator.clipboard.writeText("winget install Gyan.FFmpeg");
      showToast("Copied: winget install Gyan.FFmpeg");
    } catch {
      showToast("Run in PowerShell: winget install Gyan.FFmpeg");
    }
  });
  el.btnLocateFFmpeg.addEventListener("click", async () => renderState(await window.api.locateFFmpeg()));
  el.btnRecheckTools.addEventListener("click", async () => renderState(await window.api.recheckTools()));

  el.btnModeJoin.addEventListener("click", async () => renderState(await window.api.setMode("join")));
  el.btnModeSplit.addEventListener("click", async () => renderState(await window.api.setMode("split")));

  el.btnTheme.addEventListener("click", async () => {
    const cur = document.documentElement.getAttribute("data-theme") || "dark";
    const next = cur === "dark" ? "light" : "dark";
    renderState(await window.api.setTheme(next));
  });

  el.btnToggleAudio.addEventListener("click", async () => {
    const next = !(state && state.showAudio !== false);
    if (next) {
      audioTargetPct = 3;
      audioDisplayPct = 3;
    }
    renderState(await window.api.setShowAudio(next));
  });

  el.btnToggleVisual.addEventListener("click", async () => {
    const next = !(state && state.showVisual !== false);
    if (next) {
      visualTargetPct = 3;
      visualDisplayPct = 3;
    }
    renderState(await window.api.setShowVisual(next));
  });

  el.btnPlayPause.addEventListener("click", () => {
    if (el.video.paused) el.video.play().catch(() => {});
    else el.video.pause();
  });

  el.video.addEventListener("click", () => {
    if (el.video.paused) el.video.play().catch(() => {});
    else el.video.pause();
  });

  el.video.addEventListener("play", () => {
    remuxPinnedSec = null;
    el.btnPlayPause.classList.add("is-playing");
    el.iconPlay.classList.add("hidden");
    el.iconPause.classList.remove("hidden");
  });
  el.video.addEventListener("playing", () => {
    remuxPinnedSec = null;
    if (!scrubbingRuler && !dragMarker && el.video.readyState >= 2) {
      hideScrubCanvas();
    }
  });
  el.video.addEventListener("pause", () => {
    el.btnPlayPause.classList.remove("is-playing");
    el.iconPlay.classList.remove("hidden");
    el.iconPause.classList.add("hidden");
  });

  el.video.addEventListener("error", () => {
    if (!state || !state.media) return;
    const mediaErr = el.video.error;
    if (!mediaErr || mediaErr.code === 1) {
      // MEDIA_ERR_ABORTED during rapid scrubbing or source reload is not a codec failure.
      return;
    }
    if (!useRemux) {
      remuxTranscode = false;
      startRemuxAt(getCurrentTime(), !el.video.paused);
    } else if (!remuxTranscode) {
      remuxTranscode = true;
      startRemuxAt(remuxOffset, !el.video.paused);
    } else {
      showToast("Could not decode video stream for preview (export still works)", true);
    }
  });

  el.btnBack5.addEventListener("click", () => seekTo(getCurrentTime() - 5));
  el.btnFwd5.addEventListener("click", () => seekTo(getCurrentTime() + 5));

  el.btnPrevKF.addEventListener("click", () => {
    const kfs = state && state.media ? state.media.keyframes : null;
    if (!kfs || kfs.length === 0) return;
    const cur = getCurrentTime();
    for (let i = kfs.length - 1; i >= 0; i--) {
      if (kfs[i] < cur - 0.05) {
        seekTo(kfs[i]);
        return;
      }
    }
    seekTo(kfs[0]);
  });

  el.btnNextKF.addEventListener("click", () => {
    const kfs = state && state.media ? state.media.keyframes : null;
    if (!kfs || kfs.length === 0) return;
    const cur = getCurrentTime();
    for (let i = 0; i < kfs.length; i++) {
      if (kfs[i] > cur + 0.05) {
        seekTo(kfs[i]);
        return;
      }
    }
  });

  async function placeCutAt(rawSec) {
    if (!state || !state.media) return;
    const res = await window.api.addMarker(rawSec);
    renderState(res.snapshot);
    if (res.changed) {
      seekTo(res.snappedSec);
      const delta = res.snappedSec - rawSec;
      const sign = delta >= 0 ? "+" : "";
      showToast(`Cut snapped to keyframe ${fmtTime(res.snappedSec)} (${sign}${delta.toFixed(2)}s)`);
    } else {
      showToast("A cut marker already exists on this keyframe");
    }
  }

  el.btnAddCut.addEventListener("click", () => placeCutAt(getCurrentTime()));

  el.btnSkipDropped.addEventListener("click", async () => {
    const next = !(state && state.skipDropped);
    renderState(await window.api.setSkipDropped(next));
  });

  el.video.addEventListener("loadedmetadata", () => {
    applyPlaybackRate();
  });

  el.selSpeed.addEventListener("change", () => {
    applyPlaybackRate();
  });

  function syncVolumeUI() {
    const eff = el.video.muted ? 0 : el.video.volume;
    const muted = eff <= 0.001;
    el.btnMute.classList.toggle("is-muted", muted);
    if (el.volMuteCross) {
      el.volMuteCross.classList.toggle("hidden", !muted);
    }
    if (el.volWaveLow) {
      el.volWaveLow.classList.toggle("off", muted);
      el.volWaveLow.style.opacity = muted ? "0" : String(Math.min(1, 0.42 + eff * 0.85).toFixed(2));
    }
    if (el.volWaveHigh) {
      const showHigh = eff >= 0.48;
      el.volWaveHigh.classList.toggle("off", !showHigh);
      el.volWaveHigh.style.opacity = showHigh ? String(Math.min(1, 0.35 + (eff - 0.45) * 1.2).toFixed(2)) : "0";
    }
    el.btnMute.title = muted ? "Muted (click to unmute)" : `Volume ${Math.round(eff * 100)}% (click to mute)`;
  }

  el.btnMute.addEventListener("click", () => {
    el.video.muted = !el.video.muted;
    if (!el.video.muted && el.video.volume <= 0.001) {
      el.video.volume = 0.5;
      el.rngVolume.value = "0.5";
    }
    syncVolumeUI();
  });

  el.rngVolume.addEventListener("input", () => {
    const nextVol = parseFloat(el.rngVolume.value);
    el.video.volume = Number.isFinite(nextVol) ? nextVol : 1;
    if (el.video.volume > 0 && el.video.muted) {
      el.video.muted = false;
    }
    syncVolumeUI();
  });

  el.video.addEventListener("volumechange", () => {
    syncVolumeUI();
  });

  el.btnKeepAll.addEventListener("click", async () => renderState(await window.api.setAllSegments(true)));
  el.btnDropAll.addEventListener("click", async () => renderState(await window.api.setAllSegments(false)));
  el.btnInvert.addEventListener("click", async () => renderState(await window.api.invertSegments()));
  el.btnClearCuts.addEventListener("click", async () => renderState(await window.api.clearMarkers()));

  // Zoom controls
  el.btnZoomIn.addEventListener("click", () => setZoomAt(zoom * 1.4, getCurrentTime(), 0.5));
  el.btnZoomOut.addEventListener("click", () => setZoomAt(zoom / 1.4, getCurrentTime(), 0.5));
  el.btnZoomFit.addEventListener("click", () => setZoomAt(1, 0, 0));
  el.rngZoom.addEventListener("input", () => {
    const ratio = parseFloat(el.rngZoom.value) / 1000;
    const nextZ = Math.pow(getMaxZoom(), ratio);
    const cur = getCurrentTime();
    const span = getViewSpan();
    const curRatio = (span > 0 && cur >= viewStart && cur <= viewStart + span)
      ? (cur - viewStart) / span
      : 0.5;
    setZoomAt(nextZ, cur, curRatio);
  });

  // Wheel zoom & pan on timeline
  el.timelineWrap.addEventListener("wheel", (e) => {
    if (!state || !state.media) return;
    e.preventDefault();
    const rect = el.timelineWrap.getBoundingClientRect();
    const w = Math.max(1, rect.width);

    if (e.shiftKey || Math.abs(e.deltaX) > Math.abs(e.deltaY)) {
      const deltaPx = Math.abs(e.deltaX) > Math.abs(e.deltaY) ? e.deltaX : e.deltaY;
      viewStart = clampViewStart(viewStart + (deltaPx / w) * getViewSpan());
      syncTimelineDOMPositions();
      drawTimeline();
      drawOverview();
      return;
    }

    const cursorRatio = Math.max(0, Math.min(1, (e.clientX - rect.left) / w));
    const anchorTime = viewStart + cursorRatio * getViewSpan();
    const factor = e.deltaY < 0 ? 1.25 : 1 / 1.25;
    setZoomAt(zoom * factor, anchorTime, cursorRatio);
  }, { passive: false });

  // Middle-mouse drag pan
  el.timelineWrap.addEventListener("mousedown", (e) => {
    if (e.button !== 1 || !state || !state.media) return;
    e.preventDefault();
    const startX = e.clientX;
    const origStart = viewStart;
    const w = Math.max(1, el.timelineWrap.clientWidth);
    const span = getViewSpan();

    const onMove = (ev) => {
      const dx = ev.clientX - startX;
      viewStart = clampViewStart(origStart - (dx / w) * span);
      syncTimelineDOMPositions();
      drawTimeline();
      drawOverview();
    };
    const onUp = () => {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
    };
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
  });

  // Overview minimap click/drag panning
  el.overviewWrap.addEventListener("mousedown", (e) => {
    if (e.button !== 0 || !state || !state.media) return;
    const panToClientX = (cx) => {
      const rect = el.overviewWrap.getBoundingClientRect();
      const ratio = Math.max(0, Math.min(1, (cx - rect.left) / Math.max(1, rect.width)));
      const targetCenter = ratio * getTotalDuration();
      viewStart = clampViewStart(targetCenter - getViewSpan() * 0.5);
      syncTimelineDOMPositions();
      drawTimeline();
      drawOverview();
    };
    panToClientX(e.clientX);
    const onMove = (ev) => panToClientX(ev.clientX);
    const onUp = () => {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
    };
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
  });

  // Ruler & Histogram scrubbing (Shift+Click places cut)
  el.rulerHit.addEventListener("mousedown", (e) => {
    if (e.button !== 0 || !state || !state.media) return;
    const t = clientXToTime(e.clientX);
    if (e.shiftKey) {
      placeCutAt(t);
      return;
    }

    const wasPlaying = !el.video.paused;
    if (wasPlaying) {
      el.video.pause();
    }
    scrubbingRuler = true;
    scrubPreviewSec = t;
    freezeCurrentFrameToCanvas();

    const previewKF = (sec, clientX) => {
      const kf = nearestKeyframe(sec);
      ghostSnapSec = kf;
      requestScrubFrame(kf, false);
      const label = Math.abs(sec - kf) <= 0.025 ? `KF ${fmtTime(kf)}` : `${fmtTime(sec)} · KF ${fmtTime(kf)}`;
      showHoverTime(clientX, label);
    };

    previewKF(t, e.clientX);
    updateHUD();
    drawTimeline();
    drawOverview();

    const onMove = (ev) => {
      if (!scrubbingRuler) return;
      const nt = clientXToTime(ev.clientX);
      scrubPreviewSec = nt;
      previewKF(nt, ev.clientX);
      updateHUD();
      drawTimeline();
      drawOverview();
    };

    const onUp = () => {
      const finalSec = scrubPreviewSec !== null ? scrubPreviewSec : t;
      scrubbingRuler = false;
      scrubPreviewSec = null;
      ghostSnapSec = null;
      clearScrubQueue();
      el.hoverTooltip.classList.add("hidden");
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
      seekTo(finalSec, { autoPlay: wasPlaying });
    };

    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
  });

  // Export Modal & Actions
  function openExportModalProgress() {
    el.modalTitle.textContent = state && state.mode === "split" ? "Exporting Split Clips" : "Exporting Joined Video";
    el.exportProgressArea.classList.remove("hidden");
    el.exportResultArea.classList.add("hidden");
    el.exportProgressBar.style.width = "10%";
    el.exportPhaseText.textContent = "Starting lossless stream copy...";
    el.btnCancelExport.classList.remove("hidden");
    el.btnRevealFolder.classList.add("hidden");
    el.btnDoneModal.classList.add("hidden");
    el.btnModalClose.classList.add("hidden");
    el.exportModal.classList.remove("hidden");
  }

  el.btnExport.addEventListener("click", async () => {
    try {
      const started = await window.api.exportVideo();
      if (started) {
        openExportModalProgress();
      }
    } catch (err) {
      showToast(String(err), true);
    }
  });

  el.btnCancelExport.addEventListener("click", () => window.api.cancelExport());
  el.btnDoneModal.addEventListener("click", () => el.exportModal.classList.add("hidden"));
  el.btnModalClose.addEventListener("click", () => el.exportModal.classList.add("hidden"));
  el.btnRevealFolder.addEventListener("click", () => {
    if (state && state.lastExport && state.lastExport.files && state.lastExport.files.length > 0) {
      window.api.revealInExplorer(state.lastExport.files[0].path);
    }
  });

  // Keyboard shortcuts
  window.addEventListener("keydown", async (e) => {
    if (e.key === "Escape" && state && state.loading) {
      e.preventDefault();
      const snap = await window.api.cancelLoad();
      renderState(snap);
      showToast("Video load cancelled");
      return;
    }
    if (e.ctrlKey && e.key.toLowerCase() === "o") {
      e.preventDefault();
      window.api.openFileDialog();
      return;
    }
    if (e.ctrlKey && e.key.toLowerCase() === "e") {
      e.preventDefault();
      if (!el.btnExport.disabled) el.btnExport.click();
      return;
    }
    if (!state || !state.media) return;
    const fps = state.media.fps > 0 ? state.media.fps : 30;
    const frameStep = 1 / fps;

    if (e.code === "Space") {
      e.preventDefault();
      el.btnPlayPause.click();
    } else if (e.key.toLowerCase() === "c" || e.key.toLowerCase() === "m") {
      e.preventDefault();
      pulseButton(el.btnAddCut);
      placeCutAt(getCurrentTime());
    } else if (e.key === "[") {
      e.preventDefault();
      el.btnPrevKF.click();
    } else if (e.key === "]") {
      e.preventDefault();
      el.btnNextKF.click();
    } else if (e.key === ",") {
      e.preventDefault();
      seekTo(getCurrentTime() - frameStep);
    } else if (e.key === ".") {
      e.preventDefault();
      seekTo(getCurrentTime() + frameStep);
    } else if (e.key === "=" || e.key === "+") {
      e.preventDefault();
      el.btnZoomIn.click();
    } else if (e.key === "-") {
      e.preventDefault();
      el.btnZoomOut.click();
    } else if (e.key === "0") {
      e.preventDefault();
      el.btnZoomFit.click();
    } else if (e.key === "ArrowLeft") {
      e.preventDefault();
      el.btnBack5.click();
    } else if (e.key === "ArrowRight") {
      e.preventDefault();
      el.btnFwd5.click();
    }
  });

  // Native WebView2 File Drag-and-Drop via postMessageWithAdditionalObjects
  function isFileDrag(e) {
    return e.dataTransfer && Array.from(e.dataTransfer.types || []).includes("Files");
  }

  window.addEventListener("dragenter", (e) => {
    if (!isFileDrag(e)) return;
    e.preventDefault();
    dragCounter++;
    el.dropCard.classList.add("drag-over");
    if (state && state.media) {
      el.globalDropOverlay.classList.remove("hidden");
    }
  });

  window.addEventListener("dragover", (e) => {
    if (!isFileDrag(e)) return;
    e.preventDefault();
    if (e.dataTransfer) {
      e.dataTransfer.dropEffect = "copy";
    }
  });

  window.addEventListener("dragleave", (e) => {
    if (!isFileDrag(e)) return;
    dragCounter = Math.max(0, dragCounter - 1);
    if (dragCounter === 0) {
      el.dropCard.classList.remove("drag-over");
      el.globalDropOverlay.classList.add("hidden");
    }
  });

  window.addEventListener("drop", (e) => {
    e.preventDefault();
    dragCounter = 0;
    el.dropCard.classList.remove("drag-over");
    el.globalDropOverlay.classList.add("hidden");
    const files = e.dataTransfer && e.dataTransfer.files ? Array.from(e.dataTransfer.files) : [];
    if (files.length > 0 && window.chrome && window.chrome.webview && typeof window.chrome.webview.postMessageWithAdditionalObjects === "function") {
      releaseVideoElement();
      clearScrubQueue();
      hideScrubCanvas();
      window.chrome.webview.postMessageWithAdditionalObjects("__file_drop__", files);
    }
  });

  if (el.editorShelf) {
    el.editorShelf.addEventListener("transitionend", (e) => {
      if (e.target !== el.editorShelf || e.propertyName !== "grid-template-rows") return;
      if (!el.editorShelf.classList.contains("collapsed")) {
        el.editorShelf.classList.add("shelf-open");
        applyTimelineLayout();
        syncTimelineDOMPositions();
        repaintScrubCanvasIfVisible();
        drawTimeline();
        drawOverview();
      }
    });
  }

  if (el.verticalSplitter) {
    el.verticalSplitter.addEventListener("mousedown", (e) => {
      if (e.button !== 0) return;
      e.preventDefault();
      splitterDragging = true;
      el.verticalSplitter.classList.add("dragging");
      const startY = e.clientY;
      const startTimelineH = el.timelineWrap.getBoundingClientRect().height || getTimelineLayout().totalH;
      const scalable = getBaseScalableHeight();

      const onMove = (ev) => {
        const delta = startY - ev.clientY;
        const targetTotal = startTimelineH + delta;
        const maxScale = getMaxAllowedTimelineScale();
        const rawScale = scalable > 0 ? (targetTotal - 26) / scalable : 1.0;
        timelineScale = Math.max(0.55, Math.min(maxScale, rawScale));
        applyTimelineLayout();
        syncTimelineDOMPositions();
        drawTimeline();
      };

      const onUp = async () => {
        window.removeEventListener("mousemove", onMove);
        window.removeEventListener("mouseup", onUp);
        splitterDragging = false;
        el.verticalSplitter.classList.remove("dragging");
        if (window.api && window.api.setTimelineScale) {
          await window.api.setTimelineScale(timelineScale);
        }
      };

      window.addEventListener("mousemove", onMove);
      window.addEventListener("mouseup", onUp);
    });

    el.verticalSplitter.addEventListener("dblclick", async () => {
      timelineScale = 1.0;
      applyTimelineLayout();
      syncTimelineDOMPositions();
      drawTimeline();
      if (window.api && window.api.setTimelineScale) {
        await window.api.setTimelineScale(1.0);
      }
    });
  }

  window.addEventListener("resize", () => {
    applyTimelineLayout();
    syncTimelineDOMPositions();
    repaintScrubCanvasIfVisible();
    drawTimeline();
    drawOverview();
  });

  // Backend push hooks
  window.__onSplitState = (snap) => renderState(snap);

  window.__onLoadProgress = (p) => {
    if (!state || !state.loading || !p) return;
    loadTargetPct = Math.max(loadTargetPct, p.pct || 0);
    if (p.phase) {
      state.loadingPhase = p.phase;
    }
    renderLoadOverlay();
  };

  window.__onLoadBackdrop = (p) => {
    if (!state || !state.loading || !p || !p.dataUri) return;
    if (p.seq && state.loadSeq && p.seq !== state.loadSeq) return;
    const imgA = el.loadBgA;
    const imgB = el.loadBgB;
    if (!imgA || !imgB) return;

    const useB = activeLoadBgSlot === 0;
    activeLoadBgSlot = useB ? 1 : 0;
    const nextImg = useB ? imgB : imgA;
    const prevImg = useB ? imgA : imgB;
    const expectedSeq = state.loadSeq;

    nextImg.onload = () => {
      if (!state || !state.loading || state.loadSeq !== expectedSeq) return;
      nextImg.classList.add("visible");
      prevImg.classList.remove("visible");
    };
    nextImg.src = p.dataUri;
  };

  window.__onAnalysisProgress = (p) => {
    if (!state || !p || p.seq !== state.loadSeq) return;
    if (p.kind === "audio") {
      audioTargetPct = Math.max(audioTargetPct, p.pct || 0);
    } else if (p.kind === "visual") {
      visualTargetPct = Math.max(visualTargetPct, p.pct || 0);
    }
  };

  window.__onSplitWaveform = (payload) => {
    if (!state || !payload || payload.seq !== state.loadSeq) return;
    state.waveformLoading = false;
    audioTargetPct = 100;
    audioDisplayPct = 100;
    state.waveform = payload.waveform;
    ingestWaveform(payload.waveform);
    drawTimeline();
  };

  window.__onSplitVisual = (payload) => {
    if (!state || !payload || payload.seq !== state.loadSeq) return;
    state.visualLoading = false;
    visualTargetPct = 100;
    visualDisplayPct = 100;
    state.visual = payload.visual;
    ingestVisual(payload.visual);
    drawTimeline();
  };

  window.__onExportProgress = (p) => {
    if (!p) return;
    const pct = p.total > 0 ? Math.max(8, Math.round((p.current / p.total) * 100)) : 35;
    el.exportProgressBar.style.width = `${pct}%`;
    el.exportPhaseText.textContent = p.phase || "Copying streams...";
  };

  window.__onExportDone = (info) => {
    if (info && info.cancelled) {
      el.exportModal.classList.add("hidden");
      showToast("Export cancelled");
      return;
    }
    if (info && info.error) {
      el.exportModal.classList.add("hidden");
      showToast(info.error, true);
      return;
    }
    el.exportProgressArea.classList.add("hidden");
    el.exportResultArea.classList.remove("hidden");
    el.btnCancelExport.classList.add("hidden");
    el.btnDoneModal.classList.remove("hidden");
    el.btnModalClose.classList.remove("hidden");
    el.btnRevealFolder.classList.remove("hidden");
    el.modalTitle.textContent = "Lossless Export Complete";

    const res = state && state.lastExport;
    if (res && res.files) {
      const secStr = state.lastExportSec ? ` in ${state.lastExportSec.toFixed(2)}s` : "";
      let html = `<div class="result-summary">✔ Wrote ${res.files.length} file${res.files.length > 1 ? "s" : ""}${secStr} (${fmtBytes(res.totalBytes)}, 0 re-encodes)</div>`;
      html += `<div class="result-list">`;
      for (const f of res.files) {
        html += `<div class="result-item"><span class="result-item-name">${escapeHtml(f.name)}</span><span class="result-item-meta">${fmtTime(f.dur)} • ${fmtBytes(f.bytes)}</span></div>`;
      }
      html += `</div>`;
      el.exportResultArea.innerHTML = html;
    }
  };

  function escapeHtml(s) {
    return String(s).replace(/[&<>"']/g, (ch) => ({
      "&": "&amp;", "<": "&lt;", ">": "&gt;", "\"": "&quot;", "'": "&#39;",
    }[ch]));
  }

  window.addEventListener("DOMContentLoaded", async () => {
    syncVolumeUI();
    if (window.api && window.api.getState) {
      renderState(await window.api.getState());
    }
    requestAnimationFrame(animLoop);
  });
})();
