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
    toolHealth: $("toolHealth"),
    loadingOverlay: $("loadingOverlay"),
    loadingText: $("loadingText"),
    video: $("videoEl"),
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
    iconVolOn: $("iconVolOn"),
    iconVolOff: $("iconVolOff"),
    rngVolume: $("rngVolume"),
    btnZoomOut: $("btnZoomOut"),
    rngZoom: $("rngZoom"),
    lblZoom: $("lblZoom"),
    btnZoomIn: $("btnZoomIn"),
    btnZoomFit: $("btnZoomFit"),
    btnKeepAll: $("btnKeepAll"),
    btnDropAll: $("btnDropAll"),
    btnInvert: $("btnInvert"),
    btnClearCuts: $("btnClearCuts"),
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
  let ghostSnapSec = null;
  let dragMarker = null; // { idx, previewSec }
  let scrubbingRuler = false;
  let toastTimer = null;
  let dragCounter = 0;

  // Timeline zoom & pan state
  let zoom = 1;
  let viewStart = 0;

  // Decoded audio histogram buffers
  let wavePeaks = null;
  let waveRMS = null;

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
    if (useRemux) {
      return Math.min(getTotalDuration(), remuxOffset + (el.video.currentTime || 0));
    }
    return el.video.currentTime || 0;
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

  function seekTo(sec) {
    const dur = getTotalDuration();
    if (dur <= 0) return;
    const clamped = Math.max(0, Math.min(dur, sec));
    if (useRemux) {
      const wasPlaying = !el.video.paused;
      startRemuxAt(clamped, wasPlaying);
    } else {
      el.video.currentTime = clamped;
    }
    updateHUD();
    drawTimeline();
    drawOverview();
  }

  function startRemuxAt(sec, autoPlay) {
    if (!state || !state.remuxUrl) return;
    useRemux = true;
    remuxOffset = Math.max(0, sec);
    el.remuxBadge.classList.remove("hidden");
    const tc = remuxTranscode ? "&transcode=1" : "";
    el.video.src = `${state.remuxUrl}&start=${remuxOffset.toFixed(3)}${tc}`;
    el.video.playbackRate = parseFloat(el.selSpeed.value) || 1;
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
      return;
    }
    wavePeaks = decodeBase64Bytes(wf.peaksB64);
    waveRMS = decodeBase64Bytes(wf.rmsB64);
  }

  function applyTheme(theme) {
    const isLight = theme === "light";
    document.documentElement.setAttribute("data-theme", isLight ? "light" : "dark");
    el.iconSun.classList.toggle("hidden", !isLight);
    el.iconMoon.classList.toggle("hidden", isLight);
    drawTimeline();
    drawOverview();
  }

  function renderState(nextState) {
    state = nextState;
    if (!state) return;

    applyTheme(state.theme);

    el.btnModeJoin.classList.toggle("active", state.mode === "join");
    el.btnModeSplit.classList.toggle("active", state.mode === "split");
    el.btnSkipDropped.classList.toggle("active", !!state.skipDropped);

    if (state.tools && state.tools.ready) {
      el.toolHealth.textContent = `FFmpeg ${state.tools.ffmpegVersion || "ready"} • Lossless stream copy`;
      el.toolHealth.className = "tool-health ok";
    } else if (state.tools) {
      el.toolHealth.textContent = state.tools.error || "FFmpeg not found on PATH";
      el.toolHealth.className = "tool-health err";
    }

    if (state.loadError) {
      showToast(state.loadError, true);
    }

    const hasMedia = !!(state.media && state.media.durationSec > 0);
    el.emptyView.classList.toggle("hidden", hasMedia || state.loading);
    el.editorView.classList.toggle("hidden", !hasMedia && !state.loading);
    el.loadingOverlay.classList.toggle("hidden", !state.loading);
    if (state.loading) {
      el.loadingText.textContent = `Scanning keyframes in ${state.loadingFile || "video"}...`;
    }

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
        zoom = 1;
        viewStart = 0;
        wavePeaks = null;
        waveRMS = null;
        el.remuxBadge.classList.add("hidden");
        el.video.src = state.videoUrl;
        el.video.playbackRate = parseFloat(el.selSpeed.value) || 1;
      }

      if (state.waveform) {
        ingestWaveform(state.waveform);
      }
      updateZoomUI();
    } else {
      el.fileBadge.classList.add("hidden");
      wavePeaks = null;
      waveRMS = null;
    }

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

    dragMarker = { idx: markerIdx, previewSec: markers[markerIdx] };

    const onMove = (ev) => {
      const raw = clientXToTime(ev.clientX);
      const snapped = nearestKeyframe(raw, minBound, maxBound);
      dragMarker.previewSec = snapped;
      ghostSnapSec = snapped;
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
      el.hoverTooltip.classList.add("hidden");
      const res = await window.api.moveMarker(markerIdx, targetSec);
      renderState(res.snapshot);
      if (res.changed) seekTo(res.snappedSec);
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

  function drawTimeline() {
    const canvas = el.timelineCanvas;
    const wrap = el.timelineWrap;
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

    const isLight = document.documentElement.getAttribute("data-theme") === "light";
    const rulerH = 26;
    const audioH = 58;
    const audioBottom = rulerH + audioH; // 84
    const segH = h - audioBottom;        // 64

    // Zone 1: Ruler background
    ctx.fillStyle = isLight ? "#e2e8f0" : "#151c29";
    ctx.fillRect(0, 0, w, rulerH);

    // Zone 2: Audio Histogram lane background
    ctx.fillStyle = isLight ? "#eef2f6" : "#0e141f";
    ctx.fillRect(0, rulerH, w, audioH);

    ctx.strokeStyle = isLight ? "rgba(15,23,42,0.12)" : "rgba(255,255,255,0.08)";
    ctx.beginPath();
    ctx.moveTo(0, rulerH);
    ctx.lineTo(w, rulerH);
    ctx.moveTo(0, audioBottom);
    ctx.lineTo(w, audioBottom);
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

    // Zone 2: Audio Sound-Level Histogram
    if (wavePeaks && wavePeaks.length > 0) {
      const bins = wavePeaks.length;
      const colStep = 3;
      const barW = 2;
      const maxBarH = audioH - 10;
      const segs = state.segments || [];
      let segCursor = 0;

      for (let x = 0; x < w; x += colStep) {
        const t0 = viewStart + (x / w) * span;
        const t1 = viewStart + ((x + colStep) / w) * span;
        let b0 = Math.floor((t0 / total) * bins);
        let b1 = Math.floor((t1 / total) * bins);
        if (b0 < 0) b0 = 0;
        if (b1 >= bins) b1 = bins - 1;
        if (b1 < b0) b1 = b0;

        let peakMax = 0;
        let rmsSum = 0;
        for (let b = b0; b <= b1; b++) {
          if (wavePeaks[b] > peakMax) peakMax = wavePeaks[b];
          if (waveRMS) rmsSum += waveRMS[b];
        }
        const rmsAvg = waveRMS ? rmsSum / (b1 - b0 + 1) : 0;

        const tMid = 0.5 * (t0 + t1);
        while (segCursor < segs.length && segs[segCursor].end < tMid) {
          segCursor++;
        }
        const kept = segCursor < segs.length ? segs[segCursor].keep : isTimeKept(tMid);

        if (peakMax <= 1) {
          ctx.fillStyle = kept
            ? (isLight ? "rgba(2, 132, 199, 0.16)" : "rgba(56, 189, 248, 0.14)")
            : (isLight ? "rgba(220, 38, 38, 0.10)" : "rgba(239, 68, 68, 0.10)");
          ctx.fillRect(x, audioBottom - 2, barW, 2);
          continue;
        }

        const pH = Math.max(2, (peakMax / 255) * maxBarH);
        const rH = Math.max(1, (rmsAvg / 255) * maxBarH);

        ctx.fillStyle = kept
          ? (isLight ? "rgba(14, 165, 233, 0.42)" : "rgba(56, 189, 248, 0.36)")
          : (isLight ? "rgba(220, 38, 38, 0.24)" : "rgba(248, 113, 113, 0.26)");
        ctx.fillRect(x, audioBottom - pH, barW, pH);

        if (rH > 1) {
          ctx.fillStyle = kept
            ? (isLight ? "#0284c7" : "#22d3ee")
            : (isLight ? "rgba(185, 28, 28, 0.48)" : "rgba(248, 113, 113, 0.48)");
          ctx.fillRect(x, audioBottom - rH, barW, rH);
        }
      }

      ctx.fillStyle = isLight ? "rgba(15,23,42,0.45)" : "rgba(148,163,184,0.6)";
      ctx.font = "700 9.5px 'Segoe UI', sans-serif";
      ctx.fillText("SOUND", 7, rulerH + 10);
    } else {
      ctx.fillStyle = isLight ? "#64748b" : "#64748b";
      ctx.font = "600 11px 'Segoe UI', sans-serif";
      const msg = state.waveformLoading
        ? "Analyzing audio sound levels…"
        : (state.media.hasAudio ? "SOUND" : "No audio track");
      ctx.fillText(msg, 10, rulerH + audioH / 2);
    }

    // Zone 3: Segments
    for (const seg of state.segments || []) {
      if (seg.end <= viewStart || seg.start >= viewEnd) continue;
      const x1 = Math.max(0, timeToX(seg.start));
      const x2 = Math.min(w, timeToX(seg.end));
      const sw = Math.max(1, x2 - x1);

      if (seg.keep) {
        ctx.fillStyle = isLight ? "rgba(16, 185, 129, 0.24)" : "rgba(16, 185, 129, 0.28)";
        ctx.fillRect(x1, audioBottom, sw, segH);
        ctx.fillStyle = "#10b981";
        ctx.fillRect(x1, audioBottom, sw, 3);
      } else {
        ctx.fillStyle = isLight ? "rgba(220, 38, 38, 0.12)" : "rgba(239, 68, 68, 0.14)";
        ctx.fillRect(x1, audioBottom, sw, segH);
        ctx.fillStyle = "rgba(239, 68, 68, 0.55)";
        ctx.fillRect(x1, audioBottom, sw, 2);
      }

      if (sw > 58) {
        ctx.fillStyle = seg.keep
          ? (isLight ? "#065f46" : "#a7f3d0")
          : (isLight ? "#991b1b" : "#fca5a5");
        ctx.font = "700 11px 'Segoe UI', sans-serif";
        ctx.fillText(`#${seg.index} ${seg.keep ? "KEEP" : "DROP"}`, x1 + 8, audioBottom + segH / 2 - 6);
        if (sw > 92) {
          ctx.font = "500 10.5px 'Cascadia Mono', monospace";
          ctx.fillText(fmtTime(seg.duration), x1 + 8, audioBottom + segH / 2 + 10);
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

    // Cut lines across BOTH Sound Histogram and Segments
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

    // Full-duration Keep/Drop bars
    for (const seg of state.segments || []) {
      const x1 = (seg.start / total) * w;
      const x2 = (seg.end / total) * w;
      ctx.fillStyle = seg.keep
        ? (isLight ? "rgba(16, 185, 129, 0.35)" : "rgba(16, 185, 129, 0.35)")
        : (isLight ? "rgba(220, 38, 38, 0.18)" : "rgba(239, 68, 68, 0.22)");
      ctx.fillRect(x1, 0, Math.max(1, x2 - x1), h);
    }

    // Cut markers
    ctx.fillStyle = "#f59e0b";
    for (const mSec of state.markers || []) {
      const mx = (mSec / total) * w;
      ctx.fillRect(mx - 1, 0, 2, h);
    }

    // Playhead
    const cur = getCurrentTime();
    const px = (cur / total) * w;
    ctx.fillStyle = isLight ? "#0f172a" : "#ffffff";
    ctx.fillRect(px - 0.75, 0, 1.5, h);

    // Viewport window box
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
    if (state && state.media) {
      enforceSkipDropped();

      // Auto-scroll zoomed viewport during playback
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

  // --- Wire Controls ---
  el.btnOpen.addEventListener("click", () => window.api.openFileDialog());
  el.btnEmptyBrowse.addEventListener("click", () => window.api.openFileDialog());

  el.btnModeJoin.addEventListener("click", async () => renderState(await window.api.setMode("join")));
  el.btnModeSplit.addEventListener("click", async () => renderState(await window.api.setMode("split")));

  el.btnTheme.addEventListener("click", async () => {
    const cur = document.documentElement.getAttribute("data-theme") || "dark";
    const next = cur === "dark" ? "light" : "dark";
    renderState(await window.api.setTheme(next));
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
    el.iconPlay.classList.add("hidden");
    el.iconPause.classList.remove("hidden");
  });
  el.video.addEventListener("pause", () => {
    el.iconPlay.classList.remove("hidden");
    el.iconPause.classList.add("hidden");
  });

  el.video.addEventListener("error", () => {
    if (!state || !state.media) return;
    if (!useRemux) {
      remuxTranscode = false;
      startRemuxAt(0, false);
    } else if (!remuxTranscode) {
      remuxTranscode = true;
      startRemuxAt(remuxOffset, false);
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

  el.selSpeed.addEventListener("change", () => {
    el.video.playbackRate = parseFloat(el.selSpeed.value) || 1;
  });

  el.btnMute.addEventListener("click", () => {
    el.video.muted = !el.video.muted;
    el.iconVolOn.classList.toggle("hidden", el.video.muted);
    el.iconVolOff.classList.toggle("hidden", !el.video.muted);
  });

  el.rngVolume.addEventListener("input", () => {
    el.video.volume = parseFloat(el.rngVolume.value);
    if (el.video.volume > 0 && el.video.muted) {
      el.video.muted = false;
      el.iconVolOn.classList.remove("hidden");
      el.iconVolOff.classList.add("hidden");
    }
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

  // Ruler & Audio Histogram scrubbing (Shift+Click places cut)
  el.rulerHit.addEventListener("mousedown", (e) => {
    if (e.button !== 0 || !state || !state.media) return;
    const t = clientXToTime(e.clientX);
    if (e.shiftKey) {
      placeCutAt(t);
      return;
    }
    scrubbingRuler = true;
    seekTo(t);
    const onMove = (ev) => {
      if (!scrubbingRuler) return;
      const nt = clientXToTime(ev.clientX);
      seekTo(nt);
      showHoverTime(ev.clientX, fmtTime(nt));
    };
    const onUp = () => {
      scrubbingRuler = false;
      el.hoverTooltip.classList.add("hidden");
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
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
  window.addEventListener("keydown", (e) => {
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
      window.chrome.webview.postMessageWithAdditionalObjects("__file_drop__", files);
    }
  });

  window.addEventListener("resize", () => {
    drawTimeline();
    drawOverview();
  });

  // Backend push hooks
  window.__onSplitState = (snap) => renderState(snap);

  window.__onSplitWaveform = (payload) => {
    if (!state || !payload || payload.seq !== state.loadSeq) return;
    state.waveformLoading = false;
    state.waveform = payload.waveform;
    ingestWaveform(payload.waveform);
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
    if (window.api && window.api.getState) {
      renderState(await window.api.getState());
    }
    requestAnimationFrame(animLoop);
  });
})();
