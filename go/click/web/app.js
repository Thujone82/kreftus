(function () {
  "use strict";

  let appState = {
    state: "idle",
    statusText: "Ready",
    activeMacroId: "",
    playingStepIdx: -1,
    macros: [],
    schedules: [],
    settings: {},
    history: [],
    storageDir: ""
  };

  let selectedStepIndex = -1;
  let activeScheduleEditId = "";
  let dragFromIndex = -1;
  let saveDebounceTimer = null;
  let undoStack = [];
  let redoStack = [];
  let lastUndoMacroId = "";

  const $ = (sel) => document.querySelector(sel);
  const $$ = (sel) => Array.from(document.querySelectorAll(sel));

  function showToast(kind, msg) {
    const container = $("#toastContainer");
    if (!container) return;
    const el = document.createElement("div");
    el.className = "toast " + (kind || "info");
    el.textContent = msg;
    container.appendChild(el);
    setTimeout(() => {
      el.remove();
    }, 3400);
  }

  window.__onClickToast = showToast;

  window.__onClickState = function (snap) {
    if (!snap) return;
    const prevMacroId = appState.activeMacroId;
    appState = snap;
    if (prevMacroId !== appState.activeMacroId) {
      selectedStepIndex = -1;
      undoStack = [];
      redoStack = [];
      lastUndoMacroId = appState.activeMacroId;
    }
    applyTheme(appState.settings.theme);
    renderTopBar();
    renderMacroPanel();
    renderEditorPane();
    renderSchedulerTab();
    renderActivityTab();
    renderSettingsTab();
  };

  function applyTheme(theme) {
    const t = theme === "light" ? "light" : "dark";
    document.documentElement.setAttribute("data-theme", t);
  }

  function currentMacro() {
    return (appState.macros || []).find((m) => m.id === appState.activeMacroId) || null;
  }

  function pushUndoSnapshot(m) {
    if (!m) return;
    if (lastUndoMacroId !== m.id) {
      undoStack = [];
      redoStack = [];
      lastUndoMacroId = m.id;
    }
    undoStack.push(JSON.stringify(m));
    if (undoStack.length > 40) undoStack.shift();
    redoStack = [];
    updateUndoRedoButtons();
  }

  function updateUndoRedoButtons() {
    $("#btnUndo").disabled = undoStack.length === 0;
    $("#btnRedo").disabled = redoStack.length === 0;
  }

  function scheduleSaveMacro(m) {
    if (!m) return;
    const sub = $("#editorSubtitle");
    if (sub) sub.textContent = formatEditorSubtitle(m, false);
    clearTimeout(saveDebounceTimer);
    saveDebounceTimer = setTimeout(async () => {
      try {
        await window.apiSaveMacro(m);
      } catch (e) {
        showToast("error", String(e));
      }
    }, 220);
  }

  function formatEditorSubtitle(m, saved) {
    const count = (m.steps || []).length;
    let totalMs = 0;
    for (const s of m.steps || []) {
      if (s.delayMs > 0) totalMs += s.delayMs;
      if (s.holdMs > 0 && s.type !== "wait") totalMs += s.holdMs;
    }
    const spd = m.speed && m.speed > 0 ? m.speed : 1.0;
    const secs = (totalMs / spd / 1000).toFixed(1);
    return `${count} step${count === 1 ? "" : "s"} · ${secs} s · ${saved ? "Saved ✓" : "Saving…"}`;
  }

  function renderTopBar() {
    const btnRec = $("#btnRecord");
    const lblRec = $("#lblRecord");
    const btnPlay = $("#btnPlay");
    const btnAbort = $("#btnAbort");
    const pill = $("#statusPill");
    const stText = $("#statusText");

    const isRec = appState.state === "recording";
    const isIdle = appState.state === "idle";
    btnRec.classList.toggle("recording", isRec);
    lblRec.textContent = isRec ? "Stop" : "Record";
    btnRec.disabled = !(isIdle || isRec);

    const m = currentMacro();
    btnPlay.disabled = !isIdle || !m || !m.steps || m.steps.length === 0;
    btnAbort.disabled = !(appState.state === "countdown" || appState.state === "playing");

    pill.setAttribute("data-state", appState.state);
    stText.textContent = appState.statusText || "Ready";
  }

  function renderMacroPanel() {
    const listEl = $("#macroList");
    const q = ($("#inpSearchMacros").value || "").trim().toLowerCase();
    listEl.innerHTML = "";

    const schedByMacro = {};
    for (const sc of appState.schedules || []) {
      if (!schedByMacro[sc.macroId] || (sc.enabled && !schedByMacro[sc.macroId].enabled)) {
        schedByMacro[sc.macroId] = sc;
      }
    }

    for (const m of appState.macros || []) {
      if (q && !m.name.toLowerCase().includes(q)) continue;
      const card = document.createElement("div");
      card.className = "macro-card" + (m.id === appState.activeMacroId ? " selected" : "");

      const sc = schedByMacro[m.id];
      let badgeHTML = "";
      if (sc && sc.enabled && sc.nextRun) {
        const dt = new Date(sc.nextRun);
        const timeStr = dt.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
        badgeHTML = `<span class="badge" title="${escapeHtml(sc.summary)}">Next: ${escapeHtml(timeStr)}</span>`;
      } else if (sc) {
        badgeHTML = `<span class="badge">${escapeHtml(sc.summary)}</span>`;
      }

      const stepsCount = (m.steps || []).length;
      let totalMs = 0;
      for (const s of m.steps || []) {
        if (s.delayMs > 0) totalMs += s.delayMs;
        if (s.holdMs > 0) totalMs += s.holdMs;
      }
      const durStr = formatDurationShort(totalMs / (m.speed || 1));

      card.innerHTML = `
        <div class="macro-card-top">
          <div class="macro-card-name">${escapeHtml(m.name)}</div>
          ${badgeHTML}
        </div>
        <div class="macro-card-meta">${stepsCount} step${stepsCount === 1 ? "" : "s"} · ${durStr}</div>
        <div class="macro-card-actions">
          <button class="card-act-btn" data-act="play" title="Play macro">▶ Play</button>
          <button class="card-act-btn" data-act="schedule" title="Schedule this macro">🗓 Schedule</button>
          <button class="card-act-btn" data-act="dup" title="Duplicate macro">⧉</button>
          <button class="card-act-btn danger" data-act="del" title="Delete macro">🗑</button>
        </div>
      `;

      card.addEventListener("click", async (e) => {
        const actBtn = e.target.closest("[data-act]");
        if (actBtn) {
          e.stopPropagation();
          const act = actBtn.getAttribute("data-act");
          if (act === "play") {
            await window.apiStartPlay(m.id, $("#inpCountdown").value, 0);
          } else if (act === "schedule") {
            openSchedulerForMacro(m.id);
          } else if (act === "dup") {
            await window.apiDuplicateMacro(m.id);
            showToast("info", "Duplicated macro");
          } else if (act === "del") {
            const snap = await window.apiDeleteMacro(m.id);
            window.__onClickState(snap);
            showToast("info", "Deleted " + m.name);
          }
          return;
        }
        const snap = await window.apiSelectMacro(m.id);
        window.__onClickState(snap);
      });

      listEl.appendChild(card);
    }
  }

  function renderEditorPane() {
    const m = currentMacro();
    const emptyEl = $("#editorEmpty");
    const contentEl = $("#editorContent");

    if (!m) {
      emptyEl.classList.remove("hidden");
      contentEl.classList.add("hidden");
      return;
    }
    emptyEl.classList.add("hidden");
    contentEl.classList.remove("hidden");

    const nameInput = $("#inpMacroName");
    if (document.activeElement !== nameInput) {
      nameInput.value = m.name;
    }

    const dupCount = (appState.macros || []).filter(
      (x) => x.id !== m.id && x.name.trim().toLowerCase() === m.name.trim().toLowerCase()
    ).length;
    $("#dupNameWarn").classList.toggle("hidden", dupCount === 0);

    $("#editorSubtitle").textContent = formatEditorSubtitle(m, true);
    $("#selMacroSpeed").value = String(m.speed || 1);
    updateUndoRedoButtons();

    const stepList = $("#stepList");
    // Avoid clobbering focused inline input while typing
    if (stepList.contains(document.activeElement) && document.activeElement.tagName === "INPUT") {
      return;
    }
    stepList.innerHTML = "";

    (m.steps || []).forEach((st, idx) => {
      const row = document.createElement("div");
      row.className =
        "step-row" +
        (idx === selectedStepIndex ? " selected" : "") +
        (idx === appState.playingStepIdx ? " playing" : "");
      row.draggable = true;
      row.dataset.index = String(idx);

      const delaySec = ((st.delayMs || 0) / 1000).toFixed(2);

      row.innerHTML = `
        <span class="drag-grip" title="Drag to reorder step (or press Alt+↑/↓)">⋮⋮</span>
        <span class="step-index">${idx + 1}</span>
        <span class="step-type-badge">${stepTypeLabel(st)}</span>
        <div class="step-controls">${buildStepControlsHTML(st, idx)}</div>
        <label class="delay-chip" title="Delay before this step executes (seconds)">
          <span>+</span>
          <input type="number" step="0.05" min="0" max="86400" data-field="delaySec" value="${delaySec}" />
          <span>s</span>
        </label>
        <button class="icon-btn danger" data-step-del="${idx}" title="Delete step">🗑</button>
      `;

      // Row selection
      row.addEventListener("click", (e) => {
        if (e.target.closest("input, button, select")) return;
        selectedStepIndex = idx;
        $$("#stepList .step-row").forEach((r, i) => r.classList.toggle("selected", i === idx));
      });

      // Drag and drop reordering
      row.addEventListener("dragstart", (e) => {
        dragFromIndex = idx;
        row.classList.add("dragging");
        e.dataTransfer.effectAllowed = "move";
      });
      row.addEventListener("dragend", () => {
        dragFromIndex = -1;
        $$("#stepList .step-row").forEach((r) =>
          r.classList.remove("dragging", "drag-over-top", "drag-over-bottom")
        );
      });
      row.addEventListener("dragover", (e) => {
        e.preventDefault();
        if (dragFromIndex < 0 || dragFromIndex === idx) return;
        const rect = row.getBoundingClientRect();
        const topHalf = e.clientY < rect.top + rect.height / 2;
        row.classList.toggle("drag-over-top", topHalf);
        row.classList.toggle("drag-over-bottom", !topHalf);
      });
      row.addEventListener("dragleave", () => {
        row.classList.remove("drag-over-top", "drag-over-bottom");
      });
      row.addEventListener("drop", (e) => {
        e.preventDefault();
        row.classList.remove("drag-over-top", "drag-over-bottom");
        if (dragFromIndex < 0 || dragFromIndex === idx) return;
        const rect = row.getBoundingClientRect();
        const insertAfter = e.clientY >= rect.top + rect.height / 2;
        reorderStep(m, dragFromIndex, idx, insertAfter);
      });

      // Bind inline edits
      bindRowControls(row, m, st, idx);
      stepList.appendChild(row);
    });
  }

  function stepTypeLabel(st) {
    switch (st.type) {
      case "click":
        return "🖱️ Click";
      case "double_click":
        return "🖱️ Double Click";
      case "drag":
        return "📐 Drag";
      case "wheel":
        return "🧭 Scroll";
      case "key":
        return "⌨️ Key";
      case "text":
        return "🔤 Type Text";
      case "wait":
        return "⏱️ Wait";
      default:
        return st.type;
    }
  }

  function buildStepControlsHTML(st, idx) {
    if (st.type === "click" || st.type === "double_click") {
      const btn = st.button || "left";
      return `
        <div class="seg-group" role="group" aria-label="Mouse button">
          <button type="button" class="seg-btn ${btn === "left" ? "active" : ""}" data-btn="left">Left</button>
          <button type="button" class="seg-btn ${btn === "right" ? "active" : ""}" data-btn="right">Right</button>
          <button type="button" class="seg-btn ${btn === "middle" ? "active" : ""}" data-btn="middle">Middle</button>
        </div>
        <label class="coord-chip"><span>X</span><input type="number" data-field="x" value="${st.x || 0}" /></label>
        <label class="coord-chip"><span>Y</span><input type="number" data-field="y" value="${st.y || 0}" /></label>
        <button type="button" class="icon-btn" data-pick="xy" title="Pick coordinate on screen">🎯 Pick</button>
      `;
    }
    if (st.type === "drag") {
      const btn = st.button || "left";
      return `
        <div class="seg-group">
          <button type="button" class="seg-btn ${btn === "left" ? "active" : ""}" data-btn="left">Left</button>
          <button type="button" class="seg-btn ${btn === "right" ? "active" : ""}" data-btn="right">Right</button>
          <button type="button" class="seg-btn ${btn === "middle" ? "active" : ""}" data-btn="middle">Middle</button>
        </div>
        <label class="coord-chip"><span>X1</span><input type="number" data-field="x" value="${st.x || 0}" /></label>
        <label class="coord-chip"><span>Y1</span><input type="number" data-field="y" value="${st.y || 0}" /></label>
        <span>→</span>
        <label class="coord-chip"><span>X2</span><input type="number" data-field="x2" value="${st.x2 || 0}" /></label>
        <label class="coord-chip"><span>Y2</span><input type="number" data-field="y2" value="${st.y2 || 0}" /></label>
      `;
    }
    if (st.type === "wheel") {
      const notches = Math.round((st.wheel || -120) / 120);
      return `
        <label class="coord-chip">
          <span>Notches (+up / -down)</span>
          <input type="number" data-field="notches" value="${notches}" />
        </label>
        <label class="coord-chip"><span>X</span><input type="number" data-field="x" value="${st.x || 0}" /></label>
        <label class="coord-chip"><span>Y</span><input type="number" data-field="y" value="${st.y || 0}" /></label>
      `;
    }
    if (st.type === "key") {
      const label = st.keyName || "Press key…";
      return `
        <button type="button" class="key-capture-btn" data-keycap="${idx}" title="Click then press any key or modifier chord">${escapeHtml(label)}</button>
        <label class="coord-chip" title="Key hold duration (ms)">
          <span>Hold</span>
          <input type="number" min="0" max="60000" data-field="holdMs" value="${st.holdMs || 40}" />
          <span>ms</span>
        </label>
      `;
    }
    if (st.type === "text") {
      return `
        <input type="text" class="step-text-input" data-field="text" value="${escapeAttr(st.text || "")}" placeholder="Text to type…" />
      `;
    }
    if (st.type === "wait") {
      return `<span style="color:var(--text-secondary)">Pure wait step (adjust delay on right)</span>`;
    }
    return "";
  }

  function bindRowControls(row, m, st, idx) {
    // Button switchers (Left / Right / Middle)
    row.querySelectorAll("[data-btn]").forEach((btnEl) => {
      btnEl.addEventListener("click", () => {
        pushUndoSnapshot(m);
        st.button = btnEl.getAttribute("data-btn");
        scheduleSaveMacro(m);
        renderEditorPane();
      });
    });

    // Screen coordinate picker
    const pickBtn = row.querySelector("[data-pick]");
    if (pickBtn) {
      pickBtn.addEventListener("click", async () => {
        showToast("info", "Click anywhere on screen to capture coordinates (Esc to cancel)…");
        const res = await window.apiPickPoint();
        if (res && res.ok) {
          pushUndoSnapshot(m);
          st.x = res.x;
          st.y = res.y;
          scheduleSaveMacro(m);
          renderEditorPane();
        }
      });
    }

    // Key chord capture button
    const keyBtn = row.querySelector("[data-keycap]");
    if (keyBtn) {
      keyBtn.addEventListener("click", () => {
        keyBtn.classList.add("capturing");
        keyBtn.textContent = "Press key chord…";
        const onKeyDown = (ev) => {
          ev.preventDefault();
          ev.stopPropagation();
          if (["Control", "Shift", "Alt", "Meta"].includes(ev.key)) return;
          window.removeEventListener("keydown", onKeyDown, true);
          keyBtn.classList.remove("capturing");

          pushUndoSnapshot(m);
          let mods = 0;
          const parts = [];
          if (ev.ctrlKey) {
            mods |= 1;
            parts.push("Ctrl");
          }
          if (ev.altKey) {
            mods |= 2;
            parts.push("Alt");
          }
          if (ev.shiftKey) {
            mods |= 4;
            parts.push("Shift");
          }
          if (ev.metaKey) {
            mods |= 8;
            parts.push("Win");
          }
          const kName = normalizeBrowserKeyName(ev.key, ev.code);
          parts.push(kName);
          st.mods = mods;
          st.key = ev.keyCode || 0;
          st.keyName = parts.join("+");
          scheduleSaveMacro(m);
          renderEditorPane();
        };
        window.addEventListener("keydown", onKeyDown, true);
      });
    }

    // Numeric & text field changes
    row.querySelectorAll("input[data-field]").forEach((inp) => {
      inp.addEventListener("focus", () => {
        selectedStepIndex = idx;
      });
      inp.addEventListener("change", () => {
        pushUndoSnapshot(m);
        const f = inp.getAttribute("data-field");
        if (f === "delaySec") {
          const sec = Math.max(0, parseFloat(inp.value) || 0);
          st.delayMs = Math.round(sec * 1000);
        } else if (f === "x") {
          st.x = parseInt(inp.value, 10) || 0;
        } else if (f === "y") {
          st.y = parseInt(inp.value, 10) || 0;
        } else if (f === "x2") {
          st.x2 = parseInt(inp.value, 10) || 0;
        } else if (f === "y2") {
          st.y2 = parseInt(inp.value, 10) || 0;
        } else if (f === "notches") {
          const n = parseInt(inp.value, 10) || -1;
          st.wheel = n * 120;
        } else if (f === "holdMs") {
          st.holdMs = Math.max(0, parseInt(inp.value, 10) || 0);
        } else if (f === "text") {
          st.text = inp.value;
        }
        scheduleSaveMacro(m);
      });
    });

    // Delete step button
    const delBtn = row.querySelector("[data-step-del]");
    if (delBtn) {
      delBtn.addEventListener("click", () => {
        pushUndoSnapshot(m);
        m.steps.splice(idx, 1);
        if (selectedStepIndex >= m.steps.length) {
          selectedStepIndex = m.steps.length - 1;
        }
        scheduleSaveMacro(m);
        renderEditorPane();
      });
    }
  }

  function reorderStep(m, fromIdx, toIdx, insertAfter) {
    pushUndoSnapshot(m);
    const [moved] = m.steps.splice(fromIdx, 1);
    let target = toIdx;
    if (fromIdx < toIdx) target--;
    if (insertAfter) target++;
    if (target < 0) target = 0;
    if (target > m.steps.length) target = m.steps.length;
    m.steps.splice(target, 0, moved);
    selectedStepIndex = target;
    scheduleSaveMacro(m);
    renderEditorPane();
  }

  function normalizeBrowserKeyName(key, code) {
    if (key === " ") return "Space";
    if (key === "Escape") return "Esc";
    if (key === "ArrowUp") return "Up";
    if (key === "ArrowDown") return "Down";
    if (key === "ArrowLeft") return "Left";
    if (key === "ArrowRight") return "Right";
    if (key.length === 1) return key.toUpperCase();
    return key;
  }

  // --- Scheduler Tab ---

  function openSchedulerForMacro(macroId) {
    switchTab("scheduler");
    populateNewScheduleForm(macroId);
  }

  function populateNewScheduleForm(prefillMacroId) {
    activeScheduleEditId = "";
    $("#schedFormTitle").textContent = "Create Schedule";
    $("#btnRunSchedNow").classList.add("hidden");
    $("#btnDeleteSched").classList.add("hidden");
    $("#schedId").value = "";

    const macroSel = $("#schedMacro");
    macroSel.innerHTML = "";
    for (const m of appState.macros || []) {
      const opt = document.createElement("option");
      opt.value = m.id;
      opt.textContent = m.name;
      macroSel.appendChild(opt);
    }
    if (prefillMacroId) {
      macroSel.value = prefillMacroId;
    } else if (appState.activeMacroId) {
      macroSel.value = appState.activeMacroId;
    }

    const selMacro = (appState.macros || []).find((m) => m.id === macroSel.value);
    $("#schedName").value = selMacro ? selMacro.name : "";
    $("#schedEnabled").checked = true;

    const in5 = new Date(Date.now() + 5 * 60 * 1000);
    $("#schedStart").value = toLocalDateTimeInput(in5);
    $("#schedRepeat").value = "none";
    $("#schedInterval").value = "15m";
    $$("#fieldWeekdays input[type=checkbox]").forEach((cb) => (cb.checked = false));
    $("#schedEnd").value = "never";
    $("#schedEndRuns").value = "10";
    $("#schedEndDuration").value = "2h";
    $("#schedEndUntil").value = "";
    $("#schedWinStart").value = "";
    $("#schedWinEnd").value = "";
    $("#schedPlaysPerRun").value = "1";
    $("#schedPlayGap").value = "1s";
    $("#schedMissed").value = "skip";
    $("#schedOverlap").value = "skip";

    syncScheduleFormVisibility();
    refreshSchedulePreview();
  }

  function loadScheduleIntoForm(sc) {
    activeScheduleEditId = sc.id;
    $("#schedFormTitle").textContent = "Edit Schedule";
    $("#btnRunSchedNow").classList.remove("hidden");
    $("#btnDeleteSched").classList.remove("hidden");
    $("#schedId").value = sc.id;

    const macroSel = $("#schedMacro");
    macroSel.innerHTML = "";
    for (const m of appState.macros || []) {
      const opt = document.createElement("option");
      opt.value = m.id;
      opt.textContent = m.name;
      macroSel.appendChild(opt);
    }
    macroSel.value = sc.macroId;
    $("#schedName").value = sc.name || "";
    $("#schedEnabled").checked = !!sc.enabled;
    $("#schedStart").value = toLocalDateTimeInput(new Date(sc.start));
    $("#schedRepeat").value = sc.repeat || "none";
    $("#schedInterval").value = formatDurationShort(sc.intervalMs || 900000);

    const mask = sc.weekdays || 0;
    $$("#fieldWeekdays input[type=checkbox]").forEach((cb) => {
      const bit = parseInt(cb.getAttribute("data-day"), 10);
      cb.checked = (mask & (1 << bit)) !== 0;
    });

    $("#schedEnd").value = sc.end || "never";
    $("#schedEndRuns").value = String(sc.endRuns || 10);
    $("#schedEndDuration").value = formatDurationShort(sc.endDurationMs || 7200000);
    $("#schedEndUntil").value = sc.endAt ? toLocalDateTimeInput(new Date(sc.endAt)) : "";

    $("#schedWinStart").value = sc.windowStart ? formatHHMM(sc.windowStart) : "";
    $("#schedWinEnd").value = sc.windowEnd ? formatHHMM(sc.windowEnd) : "";
    $("#schedPlaysPerRun").value = String(sc.playsPerRun || 1);
    $("#schedPlayGap").value = formatDurationShort(sc.playGapMs || 1000);
    $("#schedMissed").value = sc.missed || "skip";
    $("#schedOverlap").value = sc.overlap || "skip";

    syncScheduleFormVisibility();
    refreshSchedulePreview();
  }

  function renderSchedulerTab() {
    const macroSel = $("#schedMacro");
    if (macroSel && macroSel.options.length !== (appState.macros || []).length) {
      const cur = macroSel.value;
      macroSel.innerHTML = "";
      for (const m of appState.macros || []) {
        const opt = document.createElement("option");
        opt.value = m.id;
        opt.textContent = m.name;
        macroSel.appendChild(opt);
      }
      if (cur) macroSel.value = cur;
    }

    const listEl = $("#scheduleList");
    listEl.innerHTML = "";
    for (const sc of appState.schedules || []) {
      const card = document.createElement("div");
      card.className = "macro-card" + (sc.id === activeScheduleEditId ? " selected" : "");
      const nextStr = sc.nextRun
        ? "Next: " + new Date(sc.nextRun).toLocaleString([], { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" })
        : sc.enabled
        ? "Finished"
        : "Paused";

      card.innerHTML = `
        <div class="macro-card-top">
          <div class="macro-card-name">${escapeHtml(sc.name || sc.macroName)}</div>
          <label class="toggle-row" title="Enable or pause schedule">
            <input type="checkbox" data-sched-toggle="${sc.id}" ${sc.enabled ? "checked" : ""} />
          </label>
        </div>
        <div class="macro-card-meta">${escapeHtml(sc.macroName)} · ${escapeHtml(sc.summary)}</div>
        <div class="macro-card-meta"><span class="badge">${escapeHtml(nextStr)}</span></div>
      `;

      card.addEventListener("click", (e) => {
        const toggle = e.target.closest("[data-sched-toggle]");
        if (toggle) {
          e.stopPropagation();
          window.apiToggleSchedule(sc.id, toggle.checked).then((snap) => window.__onClickState(snap));
          return;
        }
        loadScheduleIntoForm(sc);
        renderSchedulerTab();
      });

      listEl.appendChild(card);
    }
  }

  function syncScheduleFormVisibility() {
    const rep = $("#schedRepeat").value;
    $("#fieldInterval").classList.toggle("hidden", rep !== "interval");
    $("#fieldWeekdays").classList.toggle("hidden", rep !== "weekly");

    const end = $("#schedEnd").value;
    $("#fieldEndRuns").classList.toggle("hidden", end !== "after_runs");
    $("#fieldEndDuration").classList.toggle("hidden", end !== "for_duration");
    $("#fieldEndUntil").classList.toggle("hidden", end !== "until_time");
  }

  function buildScheduleFromForm() {
    const macroId = $("#schedMacro").value;
    const selMacro = (appState.macros || []).find((m) => m.id === macroId);
    const name = ($("#schedName").value || "").trim() || (selMacro ? selMacro.name : "Scheduled Macro");
    const startVal = $("#schedStart").value;
    const startDate = startVal ? new Date(startVal) : new Date(Date.now() + 60000);

    let mask = 0;
    $$("#fieldWeekdays input[type=checkbox]").forEach((cb) => {
      if (cb.checked) {
        mask |= 1 << parseInt(cb.getAttribute("data-day"), 10);
      }
    });

    const ws = parseHHMMInput($("#schedWinStart").value);
    const we = parseHHMMInput($("#schedWinEnd").value);
    const endUntilVal = $("#schedEndUntil").value;
    const endAt = endUntilVal ? new Date(endUntilVal).toISOString() : null;

    return {
      id: $("#schedId").value || "",
      macroId: macroId,
      name: name,
      enabled: $("#schedEnabled").checked,
      start: startDate.toISOString(),
      repeat: $("#schedRepeat").value,
      intervalMs: parseDurationToMs($("#schedInterval").value, 900000),
      weekdays: mask,
      windowStart: ws,
      windowEnd: we,
      end: $("#schedEnd").value,
      endRuns: parseInt($("#schedEndRuns").value, 10) || 10,
      endAt: endAt,
      endDurationMs: parseDurationToMs($("#schedEndDuration").value, 7200000),
      playsPerRun: Math.max(1, parseInt($("#schedPlaysPerRun").value, 10) || 1),
      playGapMs: parseDurationToMs($("#schedPlayGap").value, 1000),
      missed: $("#schedMissed").value,
      overlap: $("#schedOverlap").value
    };
  }

  async function refreshSchedulePreview() {
    const ul = $("#schedPreviewList");
    if (!ul || typeof window.apiPreviewSchedule !== "function") return;
    try {
      const sc = buildScheduleFromForm();
      const list = await window.apiPreviewSchedule(sc);
      ul.innerHTML = "";
      if (!list || list.length === 0) {
        ul.innerHTML = `<li>No future runs (check start time or end condition)</li>`;
        return;
      }
      for (const item of list) {
        const li = document.createElement("li");
        li.textContent = item;
        ul.appendChild(li);
      }
    } catch (_) {}
  }

  // --- Activity & Settings Tabs ---

  function renderActivityTab() {
    const wrap = $("#historyList");
    const hist = appState.history || [];
    if (hist.length === 0) {
      wrap.innerHTML = `<div class="empty-state"><p>No macro runs logged yet.</p></div>`;
      return;
    }
    let rows = "";
    for (const h of hist) {
      const dt = new Date(h.time).toLocaleString();
      rows += `
        <tr>
          <td>${escapeHtml(dt)}</td>
          <td><strong>${escapeHtml(h.macroName || "")}</strong></td>
          <td>${escapeHtml(h.trigger || "")}${h.scheduleName ? " (" + escapeHtml(h.scheduleName) + ")" : ""}</td>
          <td><span class="badge">${escapeHtml(h.result || "")}</span></td>
          <td>${escapeHtml(h.detail || "")}</td>
        </tr>
      `;
    }
    wrap.innerHTML = `
      <table class="history-table">
        <thead>
          <tr><th>Time</th><th>Macro</th><th>Trigger</th><th>Result</th><th>Notes</th></tr>
        </thead>
        <tbody>${rows}</tbody>
      </table>
    `;
  }

  function renderSettingsTab() {
    const s = appState.settings || {};
    $("#setRecordMouse").checked = s.recordMouse !== false;
    $("#setRecordKeyboard").checked = s.recordKeyboard !== false;
    $("#setMergeTyping").checked = s.mergeTyping !== false;
    $("#setHideWhilePlaying").checked = s.hideWhilePlaying !== false;
    $("#setWarnSecs").value = String(s.scheduleWarnSecs ?? 5);
    $("#setTheme").value = s.theme || "dark";
    $("#setKeepAwake").checked = !!s.keepAwake;
    $("#setStartWithWin").checked = !!s.startWithWindows;
    $("#lblStorageDir").textContent = appState.storageDir || "";
  }

  // --- Navigation & Event Wiring ---

  function switchTab(tabName) {
    $$(".nav-item").forEach((b) => b.classList.toggle("active", b.getAttribute("data-tab") === tabName));
    $$(".tab-view").forEach((v) => v.classList.toggle("active", v.id === "tab-" + tabName));
    if (tabName === "scheduler" && !activeScheduleEditId) {
      populateNewScheduleForm(appState.activeMacroId);
    }
  }

  function initEvents() {
    $$(".nav-item").forEach((btn) => {
      btn.addEventListener("click", () => switchTab(btn.getAttribute("data-tab")));
    });

    $("#btnRecord").addEventListener("click", () => window.apiToggleRecord(""));
    $("#btnNewRecording").addEventListener("click", () => window.apiToggleRecord(""));
    $("#btnAppendRec").addEventListener("click", () => {
      const m = currentMacro();
      if (m) window.apiToggleRecord(m.id);
    });

    $("#btnPlay").addEventListener("click", () => {
      const m = currentMacro();
      if (m) window.apiStartPlay(m.id, $("#inpCountdown").value, 0);
    });

    $("#btnAbort").addEventListener("click", () => window.apiAbort());

    $("#btnNewBlankMacro").addEventListener("click", async () => {
      await window.apiCreateMacro("");
      showToast("success", "Created new blank macro");
    });

    $("#inpSearchMacros").addEventListener("input", renderMacroPanel);

    $("#inpMacroName").addEventListener("change", () => {
      const m = currentMacro();
      if (!m) return;
      const val = $("#inpMacroName").value.trim();
      if (!val) {
        $("#inpMacroName").value = m.name;
        showToast("error", "Macro name cannot be empty");
        return;
      }
      pushUndoSnapshot(m);
      m.name = val;
      scheduleSaveMacro(m);
    });

    $("#selMacroSpeed").addEventListener("change", () => {
      const m = currentMacro();
      if (!m) return;
      pushUndoSnapshot(m);
      m.speed = parseFloat($("#selMacroSpeed").value) || 1.0;
      scheduleSaveMacro(m);
    });

    $("#btnScheduleThis").addEventListener("click", () => {
      const m = currentMacro();
      if (m) openSchedulerForMacro(m.id);
    });

    $("#btnTestFromHere").addEventListener("click", () => {
      const m = currentMacro();
      if (!m) return;
      const idx = selectedStepIndex >= 0 ? selectedStepIndex : 0;
      window.apiStartPlay(m.id, $("#inpCountdown").value, idx);
    });

    // Insert Step Dropdown
    $("#btnInsertStep").addEventListener("click", (e) => {
      e.stopPropagation();
      $("#menuInsertStep").classList.toggle("hidden");
    });
    document.addEventListener("click", () => $("#menuInsertStep").classList.add("hidden"));

    $$("#menuInsertStep [data-insert]").forEach((btn) => {
      btn.addEventListener("click", () => {
        const m = currentMacro();
        if (!m) return;
        pushUndoSnapshot(m);
        const kind = btn.getAttribute("data-insert");
        const newStep = createDefaultStep(kind);
        const insertAt = selectedStepIndex >= 0 ? selectedStepIndex + 1 : m.steps.length;
        m.steps.splice(insertAt, 0, newStep);
        selectedStepIndex = insertAt;
        scheduleSaveMacro(m);
        renderEditorPane();
      });
    });

    // Undo / Redo
    $("#btnUndo").addEventListener("click", doUndo);
    $("#btnRedo").addEventListener("click", doRedo);
    window.addEventListener("keydown", (e) => {
      if (e.ctrlKey && e.key.toLowerCase() === "z") {
        if (document.activeElement && ["INPUT", "TEXTAREA"].includes(document.activeElement.tagName)) return;
        e.preventDefault();
        doUndo();
      } else if (e.ctrlKey && e.key.toLowerCase() === "y") {
        if (document.activeElement && ["INPUT", "TEXTAREA"].includes(document.activeElement.tagName)) return;
        e.preventDefault();
        doRedo();
      } else if (e.altKey && (e.key === "ArrowUp" || e.key === "ArrowDown")) {
        const m = currentMacro();
        if (!m || selectedStepIndex < 0) return;
        e.preventDefault();
        const dir = e.key === "ArrowUp" ? -1 : 1;
        const target = selectedStepIndex + dir;
        if (target >= 0 && target < m.steps.length) {
          reorderStep(m, selectedStepIndex, target, dir > 0);
        }
      }
    });

    // Export / Import JSON Modal
    $("#btnExportMacro").addEventListener("click", async () => {
      const m = currentMacro();
      if (!m) return;
      const jsonStr = await window.apiExportMacroJSON(m.id);
      $("#jsonModalTitle").textContent = "Export Macro JSON — " + m.name;
      $("#txtModalJSON").value = jsonStr;
      $("#txtModalJSON").readOnly = true;
      $("#btnModalCopy").classList.remove("hidden");
      $("#btnModalImportConfirm").classList.add("hidden");
      $("#jsonModal").classList.remove("hidden");
    });

    $("#btnImportMacro").addEventListener("click", () => {
      $("#jsonModalTitle").textContent = "Import Macro JSON";
      $("#txtModalJSON").value = "";
      $("#txtModalJSON").readOnly = false;
      $("#btnModalCopy").classList.add("hidden");
      $("#btnModalImportConfirm").classList.remove("hidden");
      $("#jsonModal").classList.remove("hidden");
    });

    $("#btnCloseModal").addEventListener("click", () => $("#jsonModal").classList.add("hidden"));
    $("#btnModalCopy").addEventListener("click", () => {
      navigator.clipboard.writeText($("#txtModalJSON").value);
      showToast("success", "Copied JSON to clipboard");
    });
    $("#btnModalImportConfirm").addEventListener("click", async () => {
      try {
        await window.apiImportMacroJSON($("#txtModalJSON").value);
        $("#jsonModal").classList.add("hidden");
        showToast("success", "Imported macro");
      } catch (e) {
        showToast("error", String(e));
      }
    });

    // Scheduler Form Events
    $("#btnNewSchedule").addEventListener("click", () => populateNewScheduleForm(appState.activeMacroId));
    $("#schedRepeat").addEventListener("change", () => {
      syncScheduleFormVisibility();
      refreshSchedulePreview();
    });
    $("#schedEnd").addEventListener("change", () => {
      syncScheduleFormVisibility();
      refreshSchedulePreview();
    });
    $("#formSchedule").addEventListener("input", () => refreshSchedulePreview());

    $$(".quick-chips .chip").forEach((chip) => {
      chip.addEventListener("click", () => {
        if (chip.hasAttribute("data-plus-min")) {
          const mins = parseInt(chip.getAttribute("data-plus-min"), 10);
          $("#schedStart").value = toLocalDateTimeInput(new Date(Date.now() + mins * 60000));
        } else if (chip.hasAttribute("data-tomorrow")) {
          const d = new Date();
          d.setDate(d.getDate() + 1);
          d.setHours(9, 0, 0, 0);
          $("#schedStart").value = toLocalDateTimeInput(d);
        }
        refreshSchedulePreview();
      });
    });

    $("#formSchedule").addEventListener("submit", async (e) => {
      e.preventDefault();
      try {
        const sc = buildScheduleFromForm();
        const saved = await window.apiSaveSchedule(sc);
        activeScheduleEditId = saved.id;
        showToast("success", "Saved schedule: " + saved.name);
      } catch (err) {
        showToast("error", String(err));
      }
    });

    $("#btnRunSchedNow").addEventListener("click", async () => {
      if (!activeScheduleEditId) return;
      await window.apiRunScheduleNow(activeScheduleEditId);
      showToast("info", "Triggered schedule");
    });

    $("#btnDeleteSched").addEventListener("click", async () => {
      if (!activeScheduleEditId) return;
      const snap = await window.apiDeleteSchedule(activeScheduleEditId);
      activeScheduleEditId = "";
      window.__onClickState(snap);
      populateNewScheduleForm(appState.activeMacroId);
      showToast("info", "Deleted schedule");
    });

    $("#btnClearHistory").addEventListener("click", async () => {
      const snap = await window.apiClearHistory();
      window.__onClickState(snap);
    });

    $("#btnSaveSettings").addEventListener("click", async () => {
      try {
        const cfg = Object.assign({}, appState.settings, {
          recordMouse: $("#setRecordMouse").checked,
          recordKeyboard: $("#setRecordKeyboard").checked,
          mergeTyping: $("#setMergeTyping").checked,
          hideWhilePlaying: $("#setHideWhilePlaying").checked,
          scheduleWarnSecs: parseInt($("#setWarnSecs").value, 10) || 0,
          theme: $("#setTheme").value,
          keepAwake: $("#setKeepAwake").checked,
          startWithWindows: $("#setStartWithWin").checked,
          defaultCountdown: $("#inpCountdown").value || "5s"
        });
        const snap = await window.apiSaveSettings(cfg);
        window.__onClickState(snap);
        showToast("success", "Settings saved");
      } catch (err) {
        showToast("error", String(err));
      }
    });
  }

  function doUndo() {
    const m = currentMacro();
    if (!m || undoStack.length === 0) return;
    redoStack.push(JSON.stringify(m));
    const prev = JSON.parse(undoStack.pop());
    Object.assign(m, prev);
    updateUndoRedoButtons();
    scheduleSaveMacro(m);
    renderEditorPane();
  }

  function doRedo() {
    const m = currentMacro();
    if (!m || redoStack.length === 0) return;
    undoStack.push(JSON.stringify(m));
    const next = JSON.parse(redoStack.pop());
    Object.assign(m, next);
    updateUndoRedoButtons();
    scheduleSaveMacro(m);
    renderEditorPane();
  }

  function createDefaultStep(kind) {
    switch (kind) {
      case "click":
      case "double_click":
        return { type: kind, button: "left", x: 500, y: 400, delayMs: 250, holdMs: 40, glide: true };
      case "drag":
        return { type: "drag", button: "left", x: 400, y: 400, x2: 650, y2: 400, delayMs: 250, holdMs: 250, glide: true };
      case "key":
        return { type: "key", key: 13, keyName: "Enter", mods: 0, delayMs: 200, holdMs: 40 };
      case "text":
        return { type: "text", text: "Hello", delayMs: 200 };
      case "wheel":
        return { type: "wheel", wheel: -360, x: 500, y: 400, delayMs: 200, glide: true };
      case "wait":
      default:
        return { type: "wait", delayMs: 1000 };
    }
  }

  function toLocalDateTimeInput(d) {
    const pad = (n) => String(n).padStart(2, "0");
    return (
      d.getFullYear() +
      "-" +
      pad(d.getMonth() + 1) +
      "-" +
      pad(d.getDate()) +
      "T" +
      pad(d.getHours()) +
      ":" +
      pad(d.getMinutes()) +
      ":" +
      pad(d.getSeconds())
    );
  }

  function formatHHMM(tod) {
    return String(tod.hour || 0).padStart(2, "0") + ":" + String(tod.minute || 0).padStart(2, "0");
  }

  function parseHHMMInput(val) {
    if (!val || !val.includes(":")) return null;
    const [h, m] = val.split(":").map((x) => parseInt(x, 10) || 0);
    return { hour: h, minute: m };
  }

  function parseDurationToMs(s, fallbackMs) {
    s = String(s || "").trim().toLowerCase();
    if (!s) return fallbackMs;
    if (/^\d+$/.test(s)) return parseInt(s, 10) * 1000;
    let totalSec = 0;
    const re = /(\d+)\s*([hms])/g;
    let match;
    let saw = false;
    while ((match = re.exec(s)) !== null) {
      saw = true;
      const n = parseInt(match[1], 10);
      const u = match[2];
      if (u === "h") totalSec += n * 3600;
      else if (u === "m") totalSec += n * 60;
      else totalSec += n;
    }
    return saw ? totalSec * 1000 : fallbackMs;
  }

  function formatDurationShort(ms) {
    const sec = Math.max(0, Math.round((ms || 0) / 1000));
    if (sec < 60) return ((ms || 0) / 1000).toFixed(1) + "s";
    const h = Math.floor(sec / 3600);
    const m = Math.floor((sec % 3600) / 60);
    const s = sec % 60;
    let out = [];
    if (h > 0) out.push(h + "h");
    if (m > 0) out.push(m + "m");
    if (s > 0) out.push(s + "s");
    return out.join(" ");
  }

  function escapeHtml(s) {
    return String(s ?? "")
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");
  }

  function escapeAttr(s) {
    return escapeHtml(s).replace(/"/g, "&quot;");
  }

  window.addEventListener("DOMContentLoaded", async () => {
    initEvents();
    if (typeof window.apiGetState === "function") {
      const snap = await window.apiGetState();
      if (snap && snap.settings && snap.settings.defaultCountdown) {
        $("#inpCountdown").value = snap.settings.defaultCountdown;
      }
      window.__onClickState(snap);
    }
  });
})();
