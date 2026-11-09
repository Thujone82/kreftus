(function () {
  "use strict";

  var THEME_KEY = "saPlanner.theme";
  var SHARED_SAVE_KEY = "saPlanner.saveString";
  var savePanelCollapsed = false;

  function $(id) {
    return document.getElementById(id);
  }

  function formatSci(num) {
    var n = Number(num);
    if (!isFinite(n) || n === 0) return "0";
    if (Math.abs(n) < 10000) return String(Math.round(n));
    return n.toExponential(2).replace("+", "");
  }

  // ---------- Toast Notifications ----------
  function toast(msg, tone) {
    var host = $("surkyToasts");
    if (!host) return;
    var el = document.createElement("div");
    el.className = "toast" + (tone ? " " + tone : "");
    el.textContent = msg;
    host.appendChild(el);
    setTimeout(function () {
      el.classList.add("toast-out");
      setTimeout(function () {
        if (el.parentNode) el.parentNode.removeChild(el);
      }, 360);
    }, 2200);
  }

  // ---------- Clipboard Helpers ----------
  function execCopyFallback(text) {
    return new Promise(function (resolve, reject) {
      try {
        var ta = document.createElement("textarea");
        ta.value = text;
        ta.style.position = "fixed";
        ta.style.left = "-9999px";
        ta.style.opacity = "0";
        document.body.appendChild(ta);
        ta.focus();
        ta.select();
        var ok = document.execCommand("copy");
        document.body.removeChild(ta);
        if (ok) resolve();
        else reject(new Error("execCommand copy returned false"));
      } catch (err) {
        reject(err);
      }
    });
  }

  function copyTextToClipboard(text) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      return navigator.clipboard.writeText(text).catch(function () {
        return execCopyFallback(text);
      });
    }
    return execCopyFallback(text);
  }

  function classifySaveString(raw) {
    var str = (raw || "").trim();
    if (!str) return { kind: "empty" };
    try {
      if (typeof LZString === "undefined") return { kind: "invalid" };
      var decompressed = LZString.decompressFromBase64(str);
      if (!decompressed) return { kind: "invalid" };
      var parsed = JSON.parse(decompressed);
      if (!parsed || typeof parsed !== "object") return { kind: "invalid" };
      if (typeof parsed.portal === "object" && typeof parsed.global === "object") {
        return { kind: "save", data: parsed, raw: str };
      }
      return { kind: "perks", data: parsed, raw: str };
    } catch (e) {
      return { kind: "invalid" };
    }
  }

  // ---------- Theme Management ----------
  function getSavedTheme() {
    try {
      var t = localStorage.getItem(THEME_KEY);
      if (t === "light" || t === "dark") return t;
    } catch (e) {}
    return "dark";
  }

  function syncThemeUI(theme) {
    var sun = $("surkyIcoSun");
    var moon = $("surkyIcoMoon");
    var lbl = $("surkyThemeLbl");
    if (sun && moon && lbl) {
      if (theme === "dark") {
        sun.classList.remove("hidden");
        moon.classList.add("hidden");
        lbl.textContent = "Light";
      } else {
        sun.classList.add("hidden");
        moon.classList.remove("hidden");
        lbl.textContent = "Dark";
      }
    }
  }

  function setTheme(theme, persist) {
    if (theme !== "light" && theme !== "dark") theme = "dark";
    document.documentElement.setAttribute("data-theme", theme);
    if (persist !== false) {
      try {
        localStorage.setItem(THEME_KEY, theme);
      } catch (e) {}
    }
    syncThemeUI(theme);

    var dmCheckbox = (typeof inputs !== "undefined" && inputs["dark-mode"]) || $("input-dark-mode");
    var wantDark = theme === "dark";
    if (dmCheckbox && dmCheckbox.checked !== wantDark) {
      dmCheckbox.checked = wantDark;
      if (typeof switchDarkMode === "function") {
        switchDarkMode();
      }
    }
  }

  function toggleTheme() {
    var cur = document.documentElement.getAttribute("data-theme") || "dark";
    var next = cur === "dark" ? "light" : "dark";
    setTheme(next, true);
    toast("Switched to " + (next === "dark" ? "Dark" : "Light") + " theme");
  }

  // ---------- Save & Availability Helpers ----------
  function hasSurkySavedSave() {
    try {
      var raw = localStorage.getItem("trimpsSave");
      if (!raw) return false;
      var obj = JSON.parse(raw);
      return !!(obj && obj.save);
    } catch (e) {
      return false;
    }
  }

  function getSharedSaveString() {
    try {
      var raw = localStorage.getItem(SHARED_SAVE_KEY);
      if (!raw || !raw.trim()) return null;
      var check = classifySaveString(raw);
      return check.kind === "save" ? check.raw : null;
    } catch (e) {
      return null;
    }
  }

  function isGameLoaded() {
    return typeof game === "object" && game !== null && typeof game.portal === "object";
  }

  // ---------- Save Acquisition Panel Collapse (matching sa.html) ----------
  function setSaveCollapsed(collapsed) {
    savePanelCollapsed = !!collapsed;
    var panel = $("surkySavePanel");
    if (panel) {
      panel.classList.toggle("surky-save-collapsed", savePanelCollapsed);
    }
    syncActionButtons();
  }

  function openSaveEditor() {
    var saveBox = $("save-text-area");
    setSaveCollapsed(false);
    if (saveBox) {
      // Restore the raw save string if a warning message overwrote the textarea
      if (hasSurkySavedSave() && ((typeof warnings === "object" && warnings && warnings.anyActive) || !saveBox.value.trim())) {
        try {
          if (typeof restoreSaveText === "function") {
            restoreSaveText();
          }
        } catch (e) {}
      }
      saveBox.focus();
      saveBox.select();
    }
  }

  function cancelSaveEdit() {
    if (isGameLoaded()) {
      setSaveCollapsed(true);
    }
  }

  function syncActionButtons() {
    var hasExport = $("export-button") && !$("export-button").disabled;
    var topAutobuy = $("btnTopAutobuy");
    var topCopy = $("btnTopCopyPerks");
    var inlineCopy = $("btnInlineCopyPerks");
    if (topAutobuy) topAutobuy.disabled = !hasExport;
    if (topCopy) topCopy.disabled = !hasExport;
    if (inlineCopy) inlineCopy.disabled = !hasExport;

    var hasRestore = hasSurkySavedSave();
    var topRestore = $("btnTopRestore");
    if (topRestore) topRestore.classList.toggle("hidden", !hasRestore);

    var sharedStr = getSharedSaveString();
    var topShared = $("btnTopSharedSave");
    var inlineShared = $("btnInlineSharedSave");
    if (topShared) topShared.classList.toggle("hidden", !sharedStr);
    if (inlineShared) inlineShared.classList.toggle("hidden", !sharedStr);

    var loaded = isGameLoaded();
    var topUpdateSave = $("btnTopUpdateSave");
    if (topUpdateSave) {
      topUpdateSave.classList.toggle("hidden", !(loaded && savePanelCollapsed));
    }
    var inlineCancelSave = $("btnInlineCancelSave");
    if (inlineCancelSave) {
      inlineCancelSave.classList.toggle("hidden", !(loaded && !savePanelCollapsed));
    }

    var origSpoilers = $("input-hide-locked");
    var mirrorSpoilers = $("surkySpoilersToggle");
    if (origSpoilers && mirrorSpoilers) {
      mirrorSpoilers.checked = origSpoilers.checked;
    }
  }

  // ---------- Status Pills, Best Perk Highlight & Warning Banner ----------
  function refreshStatusUI() {
    syncActionButtons();

    var statusPill = $("surkyStatusPill");
    var radonPill = $("surkyRadonPill");
    var bestPill = $("surkyBestPerkPill");

    if (typeof game === "object" && game && game.global) {
      var g = game.global;
      var u = g.universe || 2;
      var z = g.world || 1;
      var hzeU2 = g.highestRadonLevelCleared ? g.highestRadonLevelCleared + 1 : 0;
      var saLv = (g.autoBattleData && g.autoBattleData.maxEnemyLevel) ? (g.autoBattleData.maxEnemyLevel - 1) : 0;
      var parts = ["U" + u + " Z" + z];
      if (hzeU2 > 0) parts.push("HZE " + hzeU2);
      if (saLv > 0) parts.push("SA " + saLv);
      if (statusPill) {
        statusPill.textContent = parts.join(" · ");
        statusPill.classList.add("ok");
      }
      if (radonPill && typeof props === "object" && props.perksRadon > 0) {
        radonPill.textContent = "Rn: " + formatSci(props.perksRadon);
        radonPill.classList.remove("hidden");
      }
    } else {
      if (statusPill) {
        statusPill.textContent = "No save loaded";
        statusPill.classList.remove("ok");
      }
      if (radonPill) radonPill.classList.add("hidden");
    }

    // Highlight best perk row & update pill
    var prevBest = document.querySelectorAll("#perks-col > .row.surky-best-perk");
    for (var i = 0; i < prevBest.length; i++) {
      prevBest[i].classList.remove("surky-best-perk");
    }
    if (typeof game === "object" && game && typeof props === "object" && props.bestPerk) {
      var bestLbl = $("label-" + String(props.bestPerk).toLowerCase());
      var bestRow = bestLbl ? bestLbl.parentElement : null;
      var rowStyle = bestRow ? (bestRow.getAttribute("style") || "") : "";
      var canStillBuy = bestRow && rowStyle.indexOf("rgba(11, 200, 35") !== -1;
      if (canStillBuy) {
        bestRow.classList.add("surky-best-perk");
        if (bestPill) {
          bestPill.textContent = "Best: " + props.bestPerk;
          bestPill.classList.remove("hidden");
        }
      } else if (bestPill) {
        bestPill.classList.add("hidden");
      }
    } else if (bestPill) {
      bestPill.classList.add("hidden");
    }

    // Highlight save textarea & update visible warning banner if an upstream warning is active
    var saveBox = $("save-text-area");
    var warnBanner = $("surkyWarnBanner");
    var hasWarn = typeof warnings === "object" && warnings && !!warnings.anyActive;
    if (saveBox) {
      saveBox.classList.toggle("surky-warn-active", hasWarn);
    }
    if (warnBanner) {
      if (hasWarn) {
        var msgs = [];
        var keys = ["rnGain", "rnLessThanSave", "greedTribs"];
        for (var k = 0; k < keys.length; k++) {
          var wObj = warnings[keys[k]];
          if (wObj && wObj.active && wObj.message) {
            msgs.push(wObj.message);
          }
        }
        warnBanner.textContent = msgs.join("\n");
        warnBanner.classList.remove("hidden");
      } else {
        warnBanner.textContent = "";
        warnBanner.classList.add("hidden");
      }
    }
  }

  // ---------- Save Loading Actions ----------
  function focusSaveTextarea(hintMsg, tone) {
    setSaveCollapsed(false);
    var box = $("save-text-area");
    if (box) {
      box.focus();
      box.select();
    }
    if (hintMsg) toast(hintMsg, tone || "warn");
  }

  function applyVerifiedSave(rawStr, sourceLabel) {
    var check = classifySaveString(rawStr);
    if (check.kind === "perks") {
      toast("That's a perk string, not a Trimps save. Click Export in Trimps first.", "bad");
      return false;
    }
    if (check.kind !== "save") {
      toast("Invalid Trimps save string.", "bad");
      return false;
    }
    var box = $("save-text-area");
    if (!box) return false;
    box.value = check.raw;
    if (typeof loadSave === "function") {
      loadSave();
      setTimeout(function () {
        if (isGameLoaded()) {
          setSaveCollapsed(true);
        }
        refreshStatusUI();
        var g = check.data.global || {};
        var msg = (sourceLabel || "Loaded save") + " (U" + (g.universe || 2) + " Z" + (g.world || 1) + ")";
        toast(msg);
      }, 25);
    }
    return true;
  }

  function pasteSaveFromClipboard() {
    if (navigator.clipboard && navigator.clipboard.readText) {
      navigator.clipboard.readText().then(function (txt) {
        if (!txt || !txt.trim()) {
          focusSaveTextarea("Clipboard is empty — paste your Trimps save in the box.", "warn");
          return;
        }
        var check = classifySaveString(txt);
        if (check.kind === "save") {
          applyVerifiedSave(check.raw, "Pasted save from clipboard");
        } else if (check.kind === "perks") {
          toast("Clipboard contains a perk string, not a full Trimps save.", "bad");
        } else {
          focusSaveTextarea("Clipboard doesn't contain a valid Trimps save.", "bad");
        }
      }).catch(function () {
        focusSaveTextarea("Clipboard read blocked — press Ctrl+V in the save box.", "warn");
      });
    } else {
      focusSaveTextarea("Press Ctrl+V in the save box to paste your Trimps save.", "warn");
    }
  }

  function useSharedSave() {
    var shared = getSharedSaveString();
    if (!shared) {
      toast("No shared Trimps save found in browser storage.", "warn");
      syncActionButtons();
      return;
    }
    applyVerifiedSave(shared, "Loaded saved Trimps save");
  }

  function restorePreviousSave() {
    if (!hasSurkySavedSave()) {
      toast("No previous save stored in Surky yet.", "warn");
      return;
    }
    if (typeof restoreSave === "function") {
      restoreSave();
      setTimeout(function () {
        if (isGameLoaded()) {
          setSaveCollapsed(true);
        }
        refreshStatusUI();
        toast("Restored previous Surky save");
      }, 25);
    }
  }

  function analyzeCurrentTextarea() {
    var box = $("save-text-area");
    if (!box || !box.value.trim()) {
      pasteSaveFromClipboard();
      return;
    }
    applyVerifiedSave(box.value, "Loaded save");
  }

  // ---------- Perk Export & Autobuy Actions ----------
  function copyExportedPerks() {
    if (typeof game !== "object" || !game) {
      toast("Import a Trimps save first before exporting perks.", "bad");
      return;
    }
    if (typeof exportPerkString === "function") {
      exportPerkString();
    }
    var expInput = $("input-export");
    var val = expInput ? expInput.value.trim() : "";
    if (!val || val.indexOf("Import save") === 0) {
      toast("No perk string generated yet — load a save first.", "warn");
      return;
    }
    copyTextToClipboard(val).then(function () {
      toast("Copied perk string to clipboard! Paste in Trimps → Portal → Import Perks.");
    }).catch(function () {
      if (expInput) {
        expInput.focus();
        expInput.select();
      }
      toast("Perk string selected — press Ctrl+C to copy.", "warn");
    });
  }

  // ---------- DOM Enhancements & Function Wrappers ----------
  function injectSaveAndExportToolbars() {
    // Hide the empty .col-sm-1 wrapper around #input-dark-mode
    var dmInput = $("input-dark-mode");
    if (dmInput && dmInput.parentElement && dmInput.parentElement.classList.contains("col-sm-1")) {
      dmInput.parentElement.classList.add("surky-hidden-dm");
    }

    var saveBox = $("save-text-area");
    if (saveBox) {
      // Tag the outer row of the save acquisition panel so it can collapse cleanly
      var formGroup = saveBox.closest(".col-sm-12.form-group");
      var outerRow = formGroup ? formGroup.parentElement : null;
      if (outerRow && outerRow.classList.contains("row")) {
        outerRow.id = "surkySavePanel";
      }

      saveBox.setAttribute("placeholder", "Paste your Trimps export string here (starts with N4Ig...) or click Paste from Clipboard (V)");
      saveBox.setAttribute("spellcheck", "false");

      var toolbar = document.createElement("div");
      toolbar.className = "surky-save-toolbar";
      toolbar.innerHTML =
        '<div class="surky-save-actions">' +
          '<button type="button" class="btn primary" id="btnInlinePaste">' +
            '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/><rect x="8" y="2" width="8" height="4" rx="1" ry="1"/></svg>' +
            '<span>Paste from Clipboard</span>' +
          '</button>' +
          '<button type="button" class="btn" id="btnInlineAnalyze" title="Load and analyze the save string currently in the box">' +
            '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>' +
            '<span>Load Save</span>' +
          '</button>' +
          '<button type="button" class="btn hidden" id="btnInlineSharedSave" title="Load the save string already stored in browser localStorage (L)">' +
            '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>' +
            '<span>Use Saved Save</span>' +
          '</button>' +
          '<button type="button" class="btn ghost hidden" id="btnInlineCancelSave" title="Collapse save panel (Esc)">' +
            '<span>Collapse</span>' +
          '</button>' +
        '</div>' +
        '<span class="surky-save-hint">Shortcuts: <kbd class="kbd">V</kbd> Paste · <kbd class="kbd">U</kbd> Edit Save · <kbd class="kbd">A</kbd> Autobuy · <kbd class="kbd">C</kbd> Copy Perks · <kbd class="kbd">?</kbd> Help</span>';
      saveBox.parentNode.appendChild(toolbar);
    }

    // Inject persistent warning banner above the 2-column grid so warnings stay visible when save panel collapses
    var mainGridRow = document.querySelector("body > .container-fluid > .row.mt-1.mb-1");
    if (mainGridRow && mainGridRow.parentNode) {
      var warnBanner = document.createElement("div");
      warnBanner.id = "surkyWarnBanner";
      warnBanner.className = "surky-warn-banner hidden";
      mainGridRow.parentNode.insertBefore(warnBanner, mainGridRow);
    }

    // Inject synced "Hide spoilers" checkbox inside #aux-col header so it stays accessible when save panel is collapsed
    var auxHeaderCol = document.querySelector("#aux-col > .row:first-child > .col-sm-12");
    var origSpoilers = $("input-hide-locked");
    if (auxHeaderCol && origSpoilers) {
      auxHeaderCol.classList.add("surky-aux-header");
      var spoilersWrap = document.createElement("label");
      spoilersWrap.className = "surky-spoilers-pill";
      spoilersWrap.title = "Hide locked perks and run inputs to prevent spoilers";
      spoilersWrap.innerHTML =
        '<input type="checkbox" class="form-check-input" id="surkySpoilersToggle" />' +
        '<span>Hide spoilers</span>';
      auxHeaderCol.appendChild(spoilersWrap);
      var mirrorToggle = $("surkySpoilersToggle");
      if (mirrorToggle) {
        mirrorToggle.checked = origSpoilers.checked;
        mirrorToggle.addEventListener("change", function () {
          origSpoilers.checked = mirrorToggle.checked;
          if (typeof switchHideUnused === "function") {
            switchHideUnused();
          }
        });
        origSpoilers.addEventListener("change", function () {
          mirrorToggle.checked = origSpoilers.checked;
        });
      }
    }

    // Enhance #input-export with an adjacent Copy button
    var expInput = $("input-export");
    if (expInput && expInput.parentNode) {
      var wrap = document.createElement("div");
      wrap.className = "surky-export-wrap";
      expInput.parentNode.insertBefore(wrap, expInput);
      wrap.appendChild(expInput);

      var copyBtn = document.createElement("button");
      copyBtn.type = "button";
      copyBtn.className = "btn primary";
      copyBtn.id = "btnInlineCopyPerks";
      copyBtn.disabled = true;
      copyBtn.title = "Copy exported perk string to clipboard (C)";
      copyBtn.innerHTML =
        '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>' +
        '<span>Copy</span>';
      copyBtn.addEventListener("click", copyExportedPerks);
      wrap.appendChild(copyBtn);
    }

    // Add page footer with attribution & docs link
    var footer = document.createElement("footer");
    footer.className = "surky-footer";
    footer.innerHTML =
      '<div><b>Surky Perk Optimizer</b> · Universe 2 Radon calculator by <a href="https://github.com/nsheetz/perks" target="_blank" rel="noopener">nsheetz</a> (UI adapted from justme)</div>' +
      '<div>/*__UPSTREAM_META__*/ · <a href="surky-build/README.html" target="_blank" rel="noopener">Documentation</a> · <a href="#top">Back to Top ↑</a></div>';
    document.body.appendChild(footer);
  }

  function wrapUpstreamFunctions() {
    if (typeof window.evaluatePerks === "function") {
      var origEval = window.evaluatePerks;
      window.evaluatePerks = function (debug) {
        var res = origEval.apply(this, arguments);
        refreshStatusUI();
        return res;
      };
    }

    if (typeof window.loadSave === "function") {
      var origLoadSave = window.loadSave;
      window.loadSave = function () {
        var res = origLoadSave.apply(this, arguments);
        setTimeout(function () {
          if (isGameLoaded()) {
            setSaveCollapsed(true);
          } else {
            setSaveCollapsed(false);
          }
          refreshStatusUI();
        }, 15);
        return res;
      };
    }

    var inClearAutobuy = false;

    if (typeof window.clearAndAutobuyPerks === "function") {
      var origClearAutobuy = window.clearAndAutobuyPerks;
      window.clearAndAutobuyPerks = function () {
        inClearAutobuy = true;
        try {
          var res = origClearAutobuy.apply(this, arguments);
          if (typeof exportPerkString === "function" && typeof game === "object" && game) {
            exportPerkString();
          }
          refreshStatusUI();
          if (typeof props === "object" && props.radonSpent > 0) {
            toast("Optimized perks (" + formatSci(props.radonSpent) + " Rn spent) — press C to copy!");
          }
          return res;
        } finally {
          inClearAutobuy = false;
        }
      };
    }

    if (typeof window.autobuyPerks === "function") {
      var origAutobuy = window.autobuyPerks;
      window.autobuyPerks = function () {
        var res = origAutobuy.apply(this, arguments);
        if (!inClearAutobuy) {
          if (typeof exportPerkString === "function" && typeof game === "object" && game) {
            exportPerkString();
          }
          refreshStatusUI();
          if (typeof props === "object" && props.radonSpent > 0) {
            toast("Autobought perks (" + formatSci(props.radonSpent) + " Rn spent) — press C to copy!");
          }
        }
        return res;
      };
    }

    // Upgrade exportPerks to use modern async clipboard + toast feedback
    window.exportPerks = function () {
      copyExportedPerks();
    };
  }

  function bindEvents() {
    var btnTopPaste = $("btnTopPaste");
    if (btnTopPaste) btnTopPaste.addEventListener("click", pasteSaveFromClipboard);

    var btnTopUpdateSave = $("btnTopUpdateSave");
    if (btnTopUpdateSave) btnTopUpdateSave.addEventListener("click", openSaveEditor);

    var btnInlinePaste = $("btnInlinePaste");
    if (btnInlinePaste) btnInlinePaste.addEventListener("click", pasteSaveFromClipboard);

    var btnInlineAnalyze = $("btnInlineAnalyze");
    if (btnInlineAnalyze) btnInlineAnalyze.addEventListener("click", analyzeCurrentTextarea);

    var btnInlineCancelSave = $("btnInlineCancelSave");
    if (btnInlineCancelSave) btnInlineCancelSave.addEventListener("click", cancelSaveEdit);

    var btnTopShared = $("btnTopSharedSave");
    if (btnTopShared) btnTopShared.addEventListener("click", useSharedSave);

    var btnInlineShared = $("btnInlineSharedSave");
    if (btnInlineShared) btnInlineShared.addEventListener("click", useSharedSave);

    var btnTopRestore = $("btnTopRestore");
    if (btnTopRestore) btnTopRestore.addEventListener("click", restorePreviousSave);

    var btnTopAutobuy = $("btnTopAutobuy");
    if (btnTopAutobuy) {
      btnTopAutobuy.addEventListener("click", function () {
        if (typeof window.clearAndAutobuyPerks === "function" && !$("clear-autobuy-button").disabled) {
          window.clearAndAutobuyPerks();
        }
      });
    }

    var btnTopCopy = $("btnTopCopyPerks");
    if (btnTopCopy) btnTopCopy.addEventListener("click", copyExportedPerks);

    var btnTopTheme = $("btnTopTheme");
    if (btnTopTheme) btnTopTheme.addEventListener("click", toggleTheme);

    var helpDlg = $("surkyHelpDlg");
    var btnTopHelp = $("btnTopHelp");
    var btnCloseHelp = $("btnCloseSurkyHelp");
    if (btnTopHelp && helpDlg) {
      btnTopHelp.addEventListener("click", function () {
        if (typeof helpDlg.showModal === "function") helpDlg.showModal();
      });
    }
    if (btnCloseHelp && helpDlg) {
      btnCloseHelp.addEventListener("click", function () {
        helpDlg.close();
      });
    }
    if (helpDlg) {
      helpDlg.addEventListener("click", function (ev) {
        if (ev.target === helpDlg) helpDlg.close();
      });
    }

    // Cross-tab storage listener for shared theme & shared save
    window.addEventListener("storage", function (ev) {
      if (ev.key === THEME_KEY && ev.newValue) {
        setTheme(ev.newValue, false);
      } else if (ev.key === SHARED_SAVE_KEY || ev.key === "trimpsSave") {
        syncActionButtons();
      }
    });

    // Global keyboard shortcuts
    window.addEventListener("keydown", function (ev) {
      if (ev.key === "Escape" && !savePanelCollapsed && isGameLoaded() && (!helpDlg || !helpDlg.open)) {
        cancelSaveEdit();
        return;
      }
      if (ev.ctrlKey || ev.altKey || ev.metaKey) return;
      var t = ev.target;
      if (t && (t.tagName === "INPUT" || t.tagName === "TEXTAREA" || t.tagName === "SELECT" || t.isContentEditable)) {
        return;
      }
      var k = ev.key;
      if (k === "v" || k === "V") {
        ev.preventDefault();
        pasteSaveFromClipboard();
      } else if (k === "u" || k === "U") {
        ev.preventDefault();
        if (savePanelCollapsed || !isGameLoaded()) {
          openSaveEditor();
        } else {
          cancelSaveEdit();
        }
      } else if (k === "c" || k === "C") {
        ev.preventDefault();
        copyExportedPerks();
      } else if (k === "a" || k === "A") {
        var clrBtn = $("clear-autobuy-button");
        if (clrBtn && !clrBtn.disabled && typeof window.clearAndAutobuyPerks === "function") {
          ev.preventDefault();
          window.clearAndAutobuyPerks();
        }
      } else if (k === "b" || k === "B") {
        var abBtn = $("autobuy-button");
        if (abBtn && !abBtn.disabled && typeof window.autobuyPerks === "function") {
          ev.preventDefault();
          window.autobuyPerks();
        }
      } else if (k === "r" || k === "R") {
        if (hasSurkySavedSave()) {
          ev.preventDefault();
          restorePreviousSave();
        }
      } else if (k === "l" || k === "L") {
        if (getSharedSaveString()) {
          ev.preventDefault();
          useSharedSave();
        }
      } else if (k === "t" || k === "T") {
        ev.preventDefault();
        toggleTheme();
      } else if (k === "?") {
        if (helpDlg && typeof helpDlg.showModal === "function") {
          ev.preventDefault();
          if (helpDlg.open) helpDlg.close();
          else helpDlg.showModal();
        }
      }
    });
  }

  // ---------- Initialize Overlay ----------
  injectSaveAndExportToolbars();
  wrapUpstreamFunctions();
  bindEvents();
  setTheme(getSavedTheme(), false);
  refreshStatusUI();
})();
