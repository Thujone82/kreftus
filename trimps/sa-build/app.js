(function () {
  var SHARD_DUST = 1e9;
  var RING_MOD_NAMES = { attack: "Attack", health: "Health", defense: "Defense", lifesteal: "Lifesteal", dustMult: "Dust" };

  var FARM_LEVELS_FALLBACK = [4, 5, 6, 7, 8, 10, 11, 13, 14, 15, 22, 25, 27, 28, 29, 35, 38, 40, 42, 43, 45, 50, 54, 59, 65, 70, 71, 74, 75, 76, 80, 83, 84, 85, 98, 104, 109, 113, 116, 117, 119, 121, 122, 123, 125, 127, 130, 131, 133, 134, 139, 142, 146, 147, 148, 153, 156, 159];
  var FARM_LEVELS_SET = {};
  FARM_LEVELS_FALLBACK.forEach(function (lv) { FARM_LEVELS_SET[lv] = true; });

  function isFarmLevel(level) {
    var l = Number(level);
    if (typeof SA_DATA !== "undefined" && Array.isArray(SA_DATA.farmLevels) && SA_DATA.farmLevels.length) {
      return SA_DATA.farmLevels.indexOf(l) !== -1;
    }
    return !!FARM_LEVELS_SET[l];
  }

  function isWorkingToClear(level) {
    if (!state.ab) return false;
    var l = level != null ? Number(level) : (state.ab.enemyLevel || state.level || 1);
    var fightLvl = state.ab.enemyLevel || 1;
    var maxLvl = state.ab.maxEnemyLevel || fightLvl;
    var isMaxCleared = Boolean(state.ab.sealed || state.ab.canSeal || fightLvl > 160 || maxLvl > 160);
    if (isMaxCleared) return false;
    return (fightLvl >= maxLvl) && (l === fightLvl || l === maxLvl);
  }

  function getLevelDefaultMode(level) {
    var l = level != null ? Number(level) : state.level;
    if (!state.ab) return "farm";
    var maxL = state.ab.maxEnemyLevel || 1;
    // Any level that is not yet cleared (l >= maxL) is a PUSH level
    if (l >= maxL) return "push";
    return "farm";
  }

  function getLevelMode(level) {
    var l = level != null ? Number(level) : state.level;
    if (state.levelModeOverrides && state.levelModeOverrides[l]) {
      return state.levelModeOverrides[l];
    }
    return getLevelDefaultMode(l);
  }

  function updateLevelSelectOptionTag(level) {
    var sel = $("levelSelect");
    if (!sel || !sel.options) return;
    for (var i = 0; i < sel.options.length; i++) {
      if (Number(sel.options[i].value) === Number(level)) {
        sel.options[i].textContent = "SA " + level + " · " + levelTag(level);
        break;
      }
    }
    syncCustomSelect("levelSelect");
  }

  var customSelects = {};
  var openCustomSelectState = null;

  function closeCustomSelectMenu(restoreFocus) {
    var st = openCustomSelectState;
    if (!st) return;
    openCustomSelectState = null;
    if (st.menu && st.menu.parentNode) st.menu.parentNode.removeChild(st.menu);
    st.menu = null;
    if (st.trigger) {
      st.trigger.setAttribute("aria-expanded", "false");
      if (restoreFocus) {
        try { st.trigger.focus({ preventScroll: true }); } catch (e) { st.trigger.focus(); }
      }
    }
    if (st.cleanup) {
      st.cleanup();
      st.cleanup = null;
    }
  }

  function syncCustomSelect(selectId) {
    var st = customSelects[selectId];
    var sel = $(selectId);
    if (!st || !sel) return;
    var opt = sel.options && sel.selectedIndex >= 0 ? sel.options[sel.selectedIndex] : null;
    var label = opt ? opt.textContent : "";
    if (st.labelEl && st.labelEl.textContent !== label) st.labelEl.textContent = label;
    if (openCustomSelectState === st && st.rowEls) {
      for (var i = 0; i < st.rowEls.length; i++) {
        st.rowEls[i].classList.toggle("selected", i === sel.selectedIndex);
        st.rowEls[i].setAttribute("aria-selected", i === sel.selectedIndex ? "true" : "false");
      }
    }
  }

  function openCustomSelectMenu(selectId) {
    var st = customSelects[selectId];
    var sel = $(selectId);
    if (!st || !sel || !sel.options || !sel.options.length) return;
    if (openCustomSelectState === st) {
      closeCustomSelectMenu(true);
      return;
    }
    closeCustomSelectMenu(false);

    var trigger = st.trigger;
    var menu = document.createElement("div");
    menu.className = "sa-select-menu";
    menu.setAttribute("role", "listbox");
    menu.tabIndex = -1;

    var selectedIdx = sel.selectedIndex >= 0 ? sel.selectedIndex : 0;
    var activeIdx = selectedIdx;
    var rowEls = [];
    var typeQuery = "";
    var typeTimer = null;

    function setActive(idx, ensureVisible) {
      if (!rowEls.length) return;
      idx = Math.max(0, Math.min(rowEls.length - 1, idx));
      if (rowEls[activeIdx]) rowEls[activeIdx].classList.remove("active");
      activeIdx = idx;
      var el = rowEls[activeIdx];
      el.classList.add("active");
      menu.setAttribute("aria-activedescendant", el.id);
      if (ensureVisible && el.scrollIntoView) {
        el.scrollIntoView({ block: "nearest" });
      }
    }

    function commitIdx(idx) {
      var changed = sel.selectedIndex !== idx;
      sel.selectedIndex = idx;
      syncCustomSelect(selectId);
      closeCustomSelectMenu(true);
      if (changed) {
        sel.dispatchEvent(new Event("change", { bubbles: true }));
      }
    }

    for (var i = 0; i < sel.options.length; i++) {
      (function (idx) {
        var opt = sel.options[idx];
        var item = document.createElement("div");
        item.id = selectId + "_opt_" + idx;
        item.className = "sa-select-opt" + (idx === selectedIdx ? " selected active" : "");
        item.setAttribute("role", "option");
        item.setAttribute("aria-selected", idx === selectedIdx ? "true" : "false");

        var txt = document.createElement("span");
        txt.textContent = opt.textContent;
        item.appendChild(txt);

        if (idx === selectedIdx) {
          var mark = document.createElement("span");
          mark.className = "sa-select-opt-mark";
          mark.textContent = "●";
          item.appendChild(mark);
        }

        item.addEventListener("mouseenter", function () { setActive(idx, false); });
        item.addEventListener("click", function (ev) {
          ev.stopPropagation();
          commitIdx(idx);
        });
        menu.appendChild(item);
        rowEls.push(item);
      })(i);
    }

    var host = (trigger.closest && trigger.closest("dialog")) || document.body;
    menu.style.visibility = "hidden";
    menu.style.left = "0px";
    menu.style.top = "0px";
    host.appendChild(menu);

    var r = trigger.getBoundingClientRect();
    var minW = Math.max(r.width, (st.opts && st.opts.menuMinWidth) || 190);
    menu.style.minWidth = Math.round(minW) + "px";

    var spaceBelow = window.innerHeight - r.bottom - 12;
    var spaceAbove = r.top - 12;
    var openAbove = spaceBelow < 220 && spaceAbove > spaceBelow;
    var maxH = Math.min(460, Math.max(140, openAbove ? spaceAbove : spaceBelow));
    menu.style.maxHeight = Math.round(maxH) + "px";

    var menuW = Math.max(minW, menu.offsetWidth);
    var left = Math.max(8, Math.min(window.innerWidth - menuW - 8, r.left));
    var top = openAbove
      ? Math.max(8, r.top - Math.min(menu.scrollHeight, maxH) - 4)
      : (r.bottom + 4);
    menu.style.left = Math.round(left) + "px";
    menu.style.top = Math.round(top) + "px";

    // Center the active/current level vertically in the dropdown list on open.
    var activeEl = rowEls[activeIdx];
    if (activeEl) {
      var mRect = menu.getBoundingClientRect();
      var aRect = activeEl.getBoundingClientRect();
      var centerOffset = (aRect.top - mRect.top) + menu.scrollTop + (aRect.height / 2);
      var maxScroll = Math.max(0, menu.scrollHeight - menu.clientHeight);
      menu.scrollTop = Math.max(0, Math.min(maxScroll, Math.round(centerOffset - (menu.clientHeight / 2))));
      menu.setAttribute("aria-activedescendant", activeEl.id);
    }
    menu.style.visibility = "visible";

    function onDocPointerDown(ev) {
      if (menu.contains(ev.target) || trigger.contains(ev.target)) return;
      closeCustomSelectMenu(false);
    }
    function onWinScroll(ev) {
      if (ev.target === menu || (ev.target && menu.contains && menu.contains(ev.target))) return;
      closeCustomSelectMenu(false);
    }
    function onWinResize() {
      closeCustomSelectMenu(false);
    }
    menu.addEventListener("keydown", function (ev) {
      if (ev.key === "ArrowDown") {
        ev.preventDefault(); ev.stopPropagation();
        setActive(activeIdx + 1, true);
      } else if (ev.key === "ArrowUp") {
        ev.preventDefault(); ev.stopPropagation();
        setActive(activeIdx - 1, true);
      } else if (ev.key === "PageDown") {
        ev.preventDefault(); ev.stopPropagation();
        setActive(activeIdx + 10, true);
      } else if (ev.key === "PageUp") {
        ev.preventDefault(); ev.stopPropagation();
        setActive(activeIdx - 10, true);
      } else if (ev.key === "Home") {
        ev.preventDefault(); ev.stopPropagation();
        setActive(0, true);
      } else if (ev.key === "End") {
        ev.preventDefault(); ev.stopPropagation();
        setActive(rowEls.length - 1, true);
      } else if (ev.key === "Enter" || ev.key === " ") {
        ev.preventDefault(); ev.stopPropagation();
        commitIdx(activeIdx);
      } else if (ev.key === "Escape") {
        ev.preventDefault(); ev.stopPropagation();
        closeCustomSelectMenu(true);
      } else if (ev.key === "Tab") {
        closeCustomSelectMenu(false);
      } else if (/^[0-9]$/.test(ev.key)) {
        ev.preventDefault(); ev.stopPropagation();
        typeQuery += ev.key;
        if (typeTimer) clearTimeout(typeTimer);
        typeTimer = setTimeout(function () { typeQuery = ""; }, 700);
        var needle = "sa " + typeQuery;
        for (var j = 0; j < sel.options.length; j++) {
          if (sel.options[j].value.indexOf(typeQuery) === 0 ||
              sel.options[j].textContent.toLowerCase().indexOf(needle) === 0) {
            setActive(j, true);
            break;
          }
        }
      }
    });

    document.addEventListener("mousedown", onDocPointerDown, true);
    window.addEventListener("scroll", onWinScroll, true);
    window.addEventListener("resize", onWinResize);

    st.menu = menu;
    st.rowEls = rowEls;
    st.cleanup = function () {
      if (typeTimer) clearTimeout(typeTimer);
      document.removeEventListener("mousedown", onDocPointerDown, true);
      window.removeEventListener("scroll", onWinScroll, true);
      window.removeEventListener("resize", onWinResize);
      st.rowEls = null;
    };
    openCustomSelectState = st;
    trigger.setAttribute("aria-expanded", "true");
    try { menu.focus({ preventScroll: true }); } catch (e) { menu.focus(); }
  }

  function initCenteredSelect(selectId, opts) {
    var sel = $(selectId);
    if (!sel || customSelects[selectId]) return;
    sel.classList.add("sa-select-native");
    sel.tabIndex = -1;
    sel.setAttribute("aria-hidden", "true");

    var trigger = document.createElement("button");
    trigger.type = "button";
    trigger.className = "sa-select-trigger" + (opts && opts.compact ? " compact" : "");
    trigger.setAttribute("aria-haspopup", "listbox");
    trigger.setAttribute("aria-expanded", "false");

    var labelEl = document.createElement("span");
    labelEl.className = "sa-select-value";
    trigger.appendChild(labelEl);

    var caret = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    caret.setAttribute("viewBox", "0 0 24 24");
    caret.setAttribute("fill", "none");
    caret.setAttribute("stroke", "currentColor");
    caret.setAttribute("stroke-width", "2.4");
    caret.setAttribute("stroke-linecap", "round");
    caret.setAttribute("stroke-linejoin", "round");
    caret.setAttribute("class", "sa-select-caret");
    caret.setAttribute("aria-hidden", "true");
    var poly = document.createElementNS("http://www.w3.org/2000/svg", "polyline");
    poly.setAttribute("points", "6 9 12 15 18 9");
    caret.appendChild(poly);
    trigger.appendChild(caret);

    sel.parentNode.insertBefore(trigger, sel.nextSibling);

    trigger.addEventListener("click", function (ev) {
      ev.stopPropagation();
      openCustomSelectMenu(selectId);
    });
    trigger.addEventListener("keydown", function (ev) {
      if (ev.key === "ArrowDown" || ev.key === "ArrowUp" || ev.key === "Enter" || ev.key === " ") {
        ev.preventDefault();
        ev.stopPropagation();
        openCustomSelectMenu(selectId);
      }
    });

    var lbl = document.querySelector('label[for="' + selectId + '"]');
    if (lbl) {
      lbl.addEventListener("click", function (ev) {
        ev.preventDefault();
        trigger.focus();
        openCustomSelectMenu(selectId);
      });
    }

    customSelects[selectId] = {
      id: selectId,
      sel: sel,
      trigger: trigger,
      labelEl: labelEl,
      opts: opts || {},
      menu: null,
      rowEls: null,
      cleanup: null
    };
    syncCustomSelect(selectId);
  }

  function setLevelMode(level, mode) {
    var l = level != null ? Number(level) : state.level;
    if (!state.levelModeOverrides) state.levelModeOverrides = {};
    state.levelModeOverrides[l] = mode;
    updateLevelSelectOptionTag(l);
  }

  function toggleLevelMode(level) {
    var l = level != null ? Number(level) : state.level;
    var cur = getLevelMode(l);
    var next = (cur === "farm") ? "push" : "farm";
    setLevelMode(l, next);
    render();
  }
  window.saToggleLevelMode = toggleLevelMode;
  window.saSetLevelMode = function(mode) {
    setLevelMode(state.level, mode);
    render();
  };
  window.saSetFarmMode = function() {
    setLevelMode(state.level, "farm");
    render();
  };
  window.saSetPushMode = function() {
    setLevelMode(state.level, "push");
    render();
  };

  var buildKillTimeCache = {};
  function buildSecPerKill(b, level) {
    if (!b) return Infinity;
    if (b.k && b.k > 0) return b.k;
    var lvl = Number(level || state.level || (state.ab && state.ab.enemyLevel) || 1);
    var bId = (typeof buildId === "function") ? buildId(b) : ((b.n || "") + "_" + (b.d || 0));
    var cacheKey = bId + "@" + lvl;
    if (buildKillTimeCache[cacheKey] !== undefined) {
      return buildKillTimeCache[cacheKey];
    }
    if (typeof SASimEngine !== "undefined" && state.save) {
      try {
        var res = SASimEngine.runBatch({ save: state.save, build: b, level: lvl }, { kills: 25, maxFights: 40 });
        if (res && res.kills > 0 && res.secPerKill > 0) {
          var wr = (res.winRate && res.winRate > 0) ? (res.winRate / 100) : 1;
          var effK = res.secPerKill / wr;
          buildKillTimeCache[cacheKey] = effK;
          return effK;
        } else if (res && res.winRate === 0) {
          buildKillTimeCache[cacheKey] = Infinity;
          return Infinity;
        }
      } catch (e) {
        console.warn("Simulation fallback for buildSecPerKill failed:", e);
      }
    }
    return 15;
  }

  function levelTag(level) {
    return isFarmLevel(level) ? "Farm" : "Push";
  }

  function isLevelCleared(level) {
    if (!state.ab) return false;
    var l = Number(level);
    var maxL = state.ab.maxEnemyLevel || 1;
    if (state.ab.sealed || state.ab.canSeal || maxL > 160) return true;
    return l < maxL;
  }
  var EXTRA_DESC = {
    Doppelganger_Signet: "Summon a Doppelganger: 50% damage reduction, 2x Attack, +1 Poison Stack Rate while alive. Explodes after taking your Max Health in damage or if it would kill, redealing fight damage and shredding 50% Enemy Defense.",
    Doppelganger_Diadem: "Can Bleed for 10s. 2x Attack if Signet is equipped, x2 again if Doppelganger is alive. Doppelganger has 50% more health and can resurrect once per fight."
  };

  var ICONS = {
    spire: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2L5 12h4.5l-2 7h9l-2-7H19L12 2zm-4.5 18.5h9V22h-9v-1.5z"/></svg>',
    sword: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="14.5 17.5 3 6 3 3 6 3 17.5 14.5"/><line x1="13" y1="19" x2="19" y2="13"/><line x1="16" y1="16" x2="20" y2="20"/><line x1="19" y1="21" x2="21" y2="19"/></svg>',
    dagger: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 4L13 15 9 11 20 4z"/><path d="M7 9l8 8"/><path d="M11 13l-5 5"/><circle cx="4.5" cy="19.5" r="1.5"/></svg>',
    pants: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 3h12l1 18h-5l-2-9-2 9H5L6 3z"/><line x1="6" y1="7" x2="18" y2="7"/></svg>',
    goggles: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="7" cy="14" r="4"/><circle cx="17" cy="14" r="4"/><path d="M11 13h2"/><path d="M1 13h2M21 13h2"/><path d="M3.5 11.5C4.5 7 7.5 5 12 5s7.5 2 8.5 6.5"/></svg>',
    spring: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 3h12"/><path d="M7 3l10 4.5-10 4.5 10 4.5-10 4.5"/><path d="M6 21h12"/></svg>',
    coin: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><path d="M14.8 9a3.5 3.5 0 1 0 0 6"/><line x1="12" y1="6.5" x2="12" y2="17.5"/></svg>',
    gear: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>',
    bag: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 3h6l-1 4h-4L9 3z"/><path d="M8 7h8"/><path d="M8.5 7C5 9 4 13 4 16.5A4.5 4.5 0 0 0 8.5 21h7a4.5 4.5 0 0 0 4.5-4.5C20 13 19 9 15.5 7"/></svg>',
    bottle: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 2h6"/><path d="M10 2v4l-3 3v11a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2V9l-3-3V2"/><path d="M7 14h10"/></svg>',
    axe: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 12l-8.5 8.5a2.12 2.12 0 0 1-3-3L11 9"/><path d="M15 13L9 7l4-4 6 6h3a8 8 0 0 1-7 7z"/></svg>',
    shield: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>',
    flask: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 3h6"/><path d="M10 3v6.5L4 19a2 2 0 0 0 1.7 3h12.6A2 2 0 0 0 20 19l-6-9.5V3"/><path d="M7.5 15h9"/></svg>',
    bolt: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>',
    boots: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 3h6v9l5 2a3 3 0 0 1 2 2.8V20H4v-5l3-2V3z"/><line x1="4" y1="17" x2="20" y2="17"/></svg>',
    coat: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M8 3L3 7l3 4 2-2v12h8V9l2 2 3-4-5-4-2 2h-4L8 3z"/></svg>',
    gem: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="6 3 18 3 22 9 12 22 2 9"/><polyline points="2 9 12 14 22 9"/><line x1="12" y1="3" x2="12" y2="22"/></svg>',
    medkit: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="6" width="18" height="14" rx="2"/><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><line x1="12" y1="10" x2="12" y2="16"/><line x1="9" y1="13" x2="15" y2="13"/></svg>',
    glove: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 11V6a2 2 0 0 0-4 0v4"/><path d="M14 10V4a2 2 0 0 0-4 0v6"/><path d="M10 10.5V6a2 2 0 0 0-4 0v8"/><path d="M6 12a2 2 0 0 0-4 0v3a7 7 0 0 0 14 0v-4"/></svg>',
    mask: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 4h16v7a8 8 0 0 1-16 0V4z"/><circle cx="9" cy="10" r="1.5"/><circle cx="15" cy="10" r="1.5"/><path d="M9 15c1.5 1 4.5 1 6 0"/></svg>',
    crown: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="2 18 5 6 9.5 12 12 4 14.5 12 19 6 22 18"/><line x1="2" y1="21" x2="22" y2="21"/></svg>',
    ring: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="13" r="7"/><polygon points="9 3 15 3 14 6 10 6"/><circle cx="12" cy="13" r="3"/></svg>',
    limb: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><line x1="12" y1="8" x2="12" y2="16"/><line x1="8" y1="12" x2="16" y2="12"/></svg>',
    contract: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="8" y1="13" x2="16" y2="13"/><line x1="8" y1="17" x2="13" y2="17"/></svg>',
    dust: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3v3M12 18v3M4.93 4.93l2.12 2.12M16.95 16.95l2.12 2.12M3 12h3M18 12h3M4.93 19.07l2.12-2.12M16.95 7.05l2.12-2.12"/><circle cx="12" cy="12" r="3"/></svg>',
    clock: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>',
    check: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>',
    warn: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>',
    info: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>',
    grip: '<svg viewBox="0 0 24 24" fill="currentColor"><circle cx="9" cy="6" r="1.7"/><circle cx="15" cy="6" r="1.7"/><circle cx="9" cy="12" r="1.7"/><circle cx="15" cy="12" r="1.7"/><circle cx="9" cy="18" r="1.7"/><circle cx="15" cy="18" r="1.7"/></svg>',
    lock: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>',
    unlock: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 9.9-1"/></svg>',
    target: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="6"/><circle cx="12" cy="12" r="2"/></svg>',
    hammer: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 12-8.373 8.373a1 1 0 1 1-3-3L12 9"/><path d="m18 15 4-4"/><path d="m21.5 11.5-1.914-1.914A2 2 0 0 0 18.172 9H15l-3-3 2-2h3.172a2 2 0 0 1 1.414.586L21.5 7.5a2 2 0 0 1 0 2.828l-.172.172z"/></svg>'
  };

  function iconForItem(key) {
    if (!key) return ICONS.sword;
    if (key === "Menacing_Mask") return ICONS.mask;
    if (key === "Grounded_Crown" || key === "Doppelganger_Diadem") return ICONS.crown;
    if (key === "Doppelganger_Signet" || key === "Mood_Bracelet") return ICONS.ring;
    if (key === "Bad_Medkit") return ICONS.medkit;
    if (key === "Unlucky_Coin") return ICONS.coin;
    if (key === "The_Doomspring") return ICONS.spring;
    if (key === "Nozzled_Goggles") return ICONS.goggles;
    if (key === "Pants") return ICONS.pants;
    if (key === "Recycler") return ICONS.gear;
    if (key === "Bag_of_Nails" || key === "Putrid_Pouch") return ICONS.bag;
    if (key === "Eelimp_in_a_Bottle") return ICONS.bottle;
    if (key === "Lifegiving_Gem" || key === "Omni_Enhancer" || key === "Basket_of_Souls") return ICONS.gem;
    if (/Sword|Blade/.test(key)) return ICONS.sword;
    if (/Dagger|Shank|Piercer/.test(key)) return ICONS.dagger;
    if (/Cleaver|Hatchet|Greataxe|Scythe|Harpoon/.test(key)) return ICONS.axe;
    if (/Armor|Suit|Protector|Aegis/.test(key)) return ICONS.shield;
    if (/Goo|Chemistry|Mold|Snimp|Globulator|Plague|Slime|Golem|Spores|Myco/.test(key)) return ICONS.flask;
    if (/Battery|Shock|Wired|Stormbringer|Fibrillator/.test(key)) return ICONS.bolt;
    if (/Boots/.test(key)) return ICONS.boots;
    if (/Raincoat|Labcoat|Wraps/.test(key)) return ICONS.coat;
    if (/Gloves|Paw|Mitts/.test(key)) return ICONS.glove;
    return ICONS.sword;
  }

  function iconForStep(s) {
    if (!s) return ICONS.bolt;
    if (s.kind === "contract" || s.kind === "missing") return ICONS.contract;
    if (s.kind === "limbs") return ICONS.limb;
    if (s.kind === "ring" || s.kind === "ringUnlock") return ICONS.ring;
    if (s.kind === "level") return iconForItem(s.key);
    return ICONS.bolt;
  }

  var state = {
    save: null,
    ab: null,
    radon: 0,
    level: 0,
    theme: "dark",
    dustOverride: "",
    simulatedDustPs: 0,
    simulatedDustTimePerKill: 0,
    simulatedDustWinRate: 0,
    simulatedLevel: 0,
    customDustRates: {},
    levelModeOverrides: {},
    filters: { ring: true, contract: true, limb: true, hideUnused: false },
    stepOrder: {},
    lockedPlans: {},
    keepPopoverOpen: false,
    saveLoadedAt: null,
    lastAdvisorStep: null,
    lastPlanSteps: null,
    lastSimulatedStepsSig: null,
    planSimInProgress: false,
    swapGuideCollapsed: false
  };

  function $(id) { return document.getElementById(id); }

  function toast(msg) {
    var wrap = $("toasts");
    if (!wrap) return;
    var d = document.createElement("div");
    d.className = "toast";
    d.textContent = msg;
    wrap.appendChild(d);
    setTimeout(function () {
      if (d.parentNode) d.parentNode.removeChild(d);
    }, 3200);
  }

  function prettify(n) {
    if (n == null || !isFinite(n)) return "—";
    var sign = n < 0 ? "-" : "";
    n = Math.abs(n);
    if (n < 1000) return sign + (n < 10 && n % 1 ? n.toFixed(2) : n < 100 && n % 1 ? n.toFixed(1) : String(Math.round(n)));
    var suffixes = ["", "K", "M", "B", "T", "Qa", "Qi", "Sx", "Sp", "Oc", "No", "Dc"];
    var e = Math.floor(Math.log10(n) / 3);
    if (e >= suffixes.length) return sign + n.toExponential(2);
    var v = n / Math.pow(10, e * 3);
    return sign + v.toFixed(v >= 100 ? 0 : v >= 10 ? 1 : 2) + suffixes[e];
  }

  function parseNum(s) {
    if (s == null || s === "") return NaN;
    s = String(s).trim().replace(/,/g, "");
    var m = s.match(/^([0-9]*\.?[0-9]+)\s*([a-zA-Z]+)?$/);
    if (!m) return Number(s);
    var n = Number(m[1]);
    var suf = { k: 1e3, m: 1e6, b: 1e9, t: 1e12, qa: 1e15, qi: 1e18, sx: 1e21, sp: 1e24 };
    if (m[2]) {
      var k = m[2].toLowerCase();
      if (suf[k]) n *= suf[k];
    }
    return n;
  }

  function fmtTime(sec) {
    if (!isFinite(sec) || sec < 0) return "—";
    if (sec === 0) return "ready";
    if (sec < 60) return Math.ceil(sec) + "s";
    if (sec < 3600) return (sec / 60).toFixed(sec < 600 ? 1 : 0) + "m";
    if (sec < 86400) return (sec / 3600).toFixed(sec < 36000 ? 1 : 0) + "h";
    if (sec < 86400 * 30) return (sec / 86400).toFixed(1) + "d";
    if (sec < 86400 * 365 * 100) return (sec / (86400 * 365)).toFixed(1) + "y";
    return "100y+";
  }

  function fmtDuration(sec) {
    if (!isFinite(sec) || sec <= 0) return "--";
    if (sec < 60) return Math.ceil(sec) + "s";
    if (sec < 3600) {
      var m = Math.floor(sec / 60);
      var s = Math.floor(sec % 60);
      return m + "m" + (s > 0 ? " " + s + "s" : "");
    }
    if (sec < 86400) {
      var h = Math.floor(sec / 3600);
      var remM = Math.floor((sec % 3600) / 60);
      return h + "h " + (remM < 10 ? "0" : "") + remM + "m";
    }
    var d = Math.floor(sec / 86400);
    var remH = Math.floor((sec % 86400) / 3600);
    return d + "d " + remH + "h";
  }

  function itemCat(key) { return SA_DATA.items[key] || null; }
  function itemName(key) { var c = itemCat(key); return c ? c.name : key.replace(/__/g, "-").replace(/_/g, " "); }
  function isShards(key) { var c = itemCat(key); return c && c.dust === "shards"; }
  function currencyName(key) { return isShards(key) ? "shards" : "dust"; }

  function upgradeCostAt(key, level) {
    var c = itemCat(key);
    if (!c || c.noUp || (c.max && level >= c.max)) return 0;
    return c.start * Math.pow(c.mod, Math.max(0, level - 1));
  }

  function upgradeCostRange(key, fromLv, toLv) {
    var total = 0;
    for (var lv = fromLv; lv < toLv; lv++) total += upgradeCostAt(key, lv);
    return total;
  }

  function contractPrice(key) {
    var c = itemCat(key);
    if (!c || !c.zone) return 0;
    var total = 100 * Math.pow(1.2023, c.zone - 75);
    if (c.dust === "shards") total /= SHARD_DUST;
    return total;
  }

  function extraLimbCost(fromLv, toLv) {
    var total = 0;
    for (var lv = fromLv; lv < toLv; lv++) total += Math.ceil(100 * Math.pow(100, lv));
    return total;
  }

  function ringLevelCost(level) { return Math.ceil(15 * Math.pow(2, level)); }

  function ringCostRange(fromLv, toLv) {
    var total = 0;
    for (var lv = fromLv; lv < toLv; lv++) total += ringLevelCost(lv);
    return total;
  }

  function ringSlots(level) { return Math.min(3, Math.floor(level / 15) + 1); }

  function itemOrder() {
    return SA_DATA.itemOrder.filter(function (k) { return SA_DATA.items[k]; }).slice().sort(function (a, b) {
      var za = SA_DATA.items[a].zone || 0;
      var zb = SA_DATA.items[b].zone || 0;
      return za - zb;
    });
  }

  function oneTimerRingCost() {
    var order = itemOrder();
    var index = 42;
    var last = order[index];
    if (!last) last = order[order.length - 1];
    return Math.ceil(contractPrice(last) / 2);
  }

  function describeItem(key, level) {
    if (EXTRA_DESC[key]) return EXTRA_DESC[key];
    var c = itemCat(key);
    if (!c) return "";
    if (c.upgrade) return "Level " + level + ". Each level: " + c.upgrade + ".";
    return "Level " + level + ".";
  }

  function describeRing(level, mods) {
    var bleed = level * 25 * Math.pow(1.5, Math.floor(level / 10));
    var poison = level * 15 * Math.pow(5, Math.floor(level / 10));
    var parts = ["+" + prettify(bleed) + "% Bleed/Shock Damage", "+" + prettify(poison) + " Poison Damage"];
    if (level >= 10) {
      var calc = level - 9;
      var chance = calc * 20 * Math.pow(1.25, Math.floor(calc / 10));
      parts.push("+" + prettify(chance) + "% Status Chance");
    }
    var stats = {
      attack: { base: 50, ten: 5, s: " Attack" },
      health: { base: 200, ten: 5, s: " Health" },
      defense: { base: 25, ten: 2.5, s: " Defense" },
      lifesteal: { base: 2.5, ten: 4, s: "% Lifesteal" },
      dustMult: { base: 5, ten: 2, s: "% Dust Mult" }
    };
    (mods || []).forEach(function (m) {
      var st = stats[m];
      if (!st) return;
      var amt = st.base * level * Math.pow(st.ten, Math.floor(level / 10));
      parts.push("+" + prettify(amt) + st.s);
    });
    return parts.join(". ") + ".";
  }

  function nextRingSlotNote(level) {
    if (level < 5) return "First custom slot at Ring 5.";
    if (level < 15) return "Second custom slot at Ring 15.";
    if (level < 30) return "Third custom slot at Ring 30.";
    return "All custom slots unlocked.";
  }

  function userItem(key) {
    if (!state.ab || !state.ab.items) return null;
    return state.ab.items[key] || null;
  }

  function userOwns(key) {
    var it = userItem(key);
    return !!(it && it.owned !== false);
  }

  function userLevel(key) {
    var it = userItem(key);
    return it && it.owned !== false ? (it.level || 1) : 0;
  }

  function extraLimbs() {
    var b = state.ab && state.ab.bonuses;
    if (!b) return 0;
    if (typeof b.Extra_Limbs === "number") return b.Extra_Limbs;
    if (b.Extra_Limbs && typeof b.Extra_Limbs.level === "number") return b.Extra_Limbs.level;
    return 0;
  }

  function maxSlots() { return 4 + extraLimbs(); }

  function hasRing() {
    var ot = state.ab && state.ab.oneTimers;
    return !!(ot && (ot.The_Ring === true || (ot.The_Ring && ot.The_Ring.owned)));
  }

  function ringLevel() { return (state.ab && state.ab.rings && state.ab.rings.level) || 1; }
  function ringMods() { return (state.ab && state.ab.rings && state.ab.rings.mods) || []; }
var ONE_TIMER_MILESTONES = [
    { key: "Gathermate", req: 7, name: "Gathermate", desc: "+5% Food, Wood & Metal in U2 per SA level", icon: "spire" },
    { key: "Smithriffic", req: 11, name: "Smithriffic", desc: "+1 Smithy when completing Melting Point", icon: "shield" },
    { key: "Championism", req: 15, name: "Championism", desc: "Unlock Championism Perk in U2", icon: "target" },
    { key: "Master_of_Arms", req: 19, name: "Master of Arms", desc: "+200 Health, +10 Attack, +2 Poison Damage", icon: "sword" },
    { key: "Artisan", req: 21, name: "Artisan", desc: "U2 Equipment 1% cheaper per SA level", icon: "coin" },
    { key: "Battlescruff", req: 23, name: "Battlescruff", desc: "+2% Scruffy XP per SA level", icon: "crown" },
    { key: "Collectology", req: 28, name: "Collectology", desc: "+2 Hubs each + 1 extra per 30 SA levels", icon: "bag" },
    { key: "Dusty_Tome", req: 32, name: "Dusty Tome", desc: "+5% Dust found on all levels per SA level", icon: "dust" },
    { key: "Whirlwind_of_Arms", req: 34, name: "Whirlwind of Arms", desc: "+1000 Health, +25 Attack, +10 Poison Damage", icon: "axe" },
    { key: "Nullicious", req: 36, name: "Nullicious", desc: "+0.5% Nu on U2 Heirlooms per SA level", icon: "gem" },
    { key: "Suprism", req: 39, name: "Suprism", desc: "+3% Prismatic Shield per SA level", icon: "shield" },
    { key: "The_Ring", req: 42, name: "The Ring", desc: "Unlock The Ring & customizable slots", icon: "ring" },
    { key: "Mass_Hysteria", req: 45, name: "Mass Hysteria", desc: "Permanent Frenzy (always active)", icon: "bolt" },
    { key: "Burstier", req: 48, name: "Burstier", desc: "Gamma Burst triggers 1 attack faster", icon: "bolt" },
    { key: "Expanding_Tauntimp", req: 51, name: "Expanding Tauntimp", desc: "Compounding Tauntimp housing in U2", icon: "spire" },
    { key: "More_Expansion", req: 53, name: "More Expansion", desc: "Unlock Expansion Perk in U2", icon: "crown" }
  ];

  function ringBaseStats(lvl) {
    if (!lvl || lvl <= 0) return { statusDmg: 0, poisonDmg: 0, statusChance: 0, slots: 0, nextSlotLvl: 5, levelsToNext: 5 };
    var statusDmg = lvl * 25 * Math.pow(1.5, Math.floor(lvl / 10));
    var poisonDmg = lvl * 15 * Math.pow(5, Math.floor(lvl / 10));
    var statusChance = 0;
    if (lvl >= 10) {
      var calcLvl = lvl - 9;
      statusChance = calcLvl * 20 * Math.pow(1.25, Math.floor(calcLvl / 10));
    }
    var slots = (lvl < 5) ? 0 : Math.min(3, Math.floor(lvl / 15) + 1);
    var nextSlotLvl = (lvl < 5) ? 5 : (lvl < 15 ? 15 : (lvl < 30 ? 30 : null));
    return {
      statusDmg: statusDmg,
      poisonDmg: poisonDmg,
      statusChance: statusChance,
      slots: slots,
      nextSlotLvl: nextSlotLvl,
      levelsToNext: nextSlotLvl ? (nextSlotLvl - lvl) : 0
    };
  }

  function nextMilestoneInfo(ownedCount) {
    var ot = (state.ab && state.ab.oneTimers) || {};
    var count = (ownedCount != null) ? ownedCount : ownedItemKeys().length;
    var next = null;
    var list = ONE_TIMER_MILESTONES.map(function (m) {
      var isOwned = !!(ot[m.key] && (ot[m.key] === true || ot[m.key].owned));
      var canBuy = count >= m.req;
      var obj = {
        key: m.key,
        name: m.name,
        req: m.req,
        desc: m.desc,
        icon: m.icon,
        owned: isOwned,
        unlocked: canBuy,
        toGo: Math.max(0, m.req - count)
      };
      if (!isOwned && !next) next = obj;
      return obj;
    });
    return { next: next, list: list };
  }

  function saLevelReq(lvl) {
    var l = Number(lvl) || 1;
    if (l < 20) return 10 * l;
    return 190 + (15 * (l - 19));
  }

  function getCurrentEquippedCombatBuild(lvl) {
    var l = Number(lvl) || (state.ab && state.ab.enemyLevel) || state.level || 1;
    if (!state.ab) return null;

    var eqItems = {};
    var count = 0;
    var abItems = state.ab.items || {};
    for (var ik in abItems) {
      var it = abItems[ik];
      if (it && typeof it === "object" && it.equipped) {
        eqItems[ik] = it.level || 1;
        count++;
      } else if (typeof it === "number" && state.ab.equipped && state.ab.equipped.indexOf(ik) !== -1) {
        eqItems[ik] = it;
        count++;
      }
    }
    if (count === 0) return null;

    var rObj = state.ab.rings || state.ab.ring || {};
    var rLv = Number(rObj.level || rObj.lv || 1);
    var rMods = (rObj.mods || rObj.m || []).slice();

    return {
      n: "Current Loadout",
      i: eqItems,
      l: count,
      r: { lv: rLv, m: rMods },
      isCurrentEquipped: true
    };
  }

  var levelSimCache = {};

  function getCurrentCombatStats(lvl) {
    var l = Number(lvl) || (state.ab && (state.ab.enemyLevel || state.ab.maxEnemyLevel)) || state.level || 1;
    // 1. Live simulator session if active on this exact level
    if (simState.session && simState.config && Number(simState.config.level) === l) {
      var snap = simState.session.getState();
      if (snap) {
        var sKillSec = (snap.avgLast100Sec && snap.avgLast100Sec > 0) ? snap.avgLast100Sec : snap.timePerKill;
        var sWr = (snap.winRate && snap.winRate > 0) ? (snap.winRate / 100) : 0;
        var sEffK = (sKillSec && sKillSec > 0 && sWr > 0) ? (sKillSec / sWr) : Infinity;
        if (snap.dustPs != null && isFinite(snap.dustPs)) {
          return {
            level: l,
            dustPs: snap.dustPs,
            secPerKill: (sKillSec && sKillSec > 0) ? sKillSec : Infinity,
            effSecPerKill: sEffK,
            winRate: snap.winRate || 0,
            kills: snap.kills || 0,
            fromLiveSession: true
          };
        }
      }
    }
    var sig = (state.save && state.ab) ? getLoadoutSignature() : "nosave";
    var cacheKey = sig + "@" + l;
    if (levelSimCache[cacheKey]) {
      return levelSimCache[cacheKey];
    }
    // 2. Reuse calibrated live save farm results when requesting the same level
    if (state.simulatedDustPs != null && Number(state.simulatedLevel) === l && state.simulatedDustTimePerKill > 0) {
      var fWr = (state.simulatedDustWinRate && state.simulatedDustWinRate > 0) ? (state.simulatedDustWinRate / 100) : 0;
      var fEffK = fWr > 0 ? (state.simulatedDustTimePerKill / fWr) : Infinity;
      var fightStats = {
        level: l,
        dustPs: state.simulatedDustPs,
        secPerKill: state.simulatedDustTimePerKill,
        effSecPerKill: fEffK,
        winRate: state.simulatedDustWinRate || 0,
        kills: state.simulatedDustWinRate > 0 ? 1 : 0
      };
      levelSimCache[cacheKey] = fightStats;
      return fightStats;
    }
    // 3. Run and cache a combat simulation on target level l using current loadout
    if (typeof SASimEngine !== "undefined" && state.save && getCurrentEquippedCombatBuild(l)) {
      try {
        var res = SASimEngine.runBatch({ save: state.save, level: l }, { kills: 300, maxFights: 600, ticks: 150000 });
        if (res) {
          var wr = (res.winRate && res.winRate > 0) ? (res.winRate / 100) : 0;
          var rawK = (res.kills > 0 && res.secPerKill > 0) ? res.secPerKill : Infinity;
          var effK = (isFinite(rawK) && wr > 0) ? (rawK / wr) : Infinity;
          var stats = {
            level: l,
            dustPs: (res.dustPs && res.dustPs > 0 && isFinite(res.dustPs)) ? res.dustPs : 0,
            secPerKill: rawK,
            effSecPerKill: effK,
            winRate: res.winRate || 0,
            kills: res.kills || 0
          };
          levelSimCache[cacheKey] = stats;
          return stats;
        }
      } catch (e) {
        console.warn("Error simulating current combat stats on SA " + l + ":", e);
      }
    }
    return null;
  }

  function levelCurrentDustPs(lvl, recommended) {
    var over = parseNum(state.dustOverride);
    if (isFinite(over) && over > 0) return over;
    var targetLvl = Number(lvl) || state.level || 1;
    var stats = getCurrentCombatStats(targetLvl);
    if (stats && typeof stats.dustPs === "number" && isFinite(stats.dustPs)) {
      return stats.dustPs;
    }
    var fightLevel = (state.ab && (state.ab.enemyLevel || state.ab.maxEnemyLevel)) || targetLvl;
    if (targetLvl === Number(fightLevel) && state.simulatedDustPs && state.simulatedDustPs > 0) {
      return state.simulatedDustPs;
    }
    if (recommended && recommended.d) return recommended.d;
    return sheetDustPs(recommended) || 0;
  }

  function getCurrentCombatSecPerKill(lvl) {
    var stats = getCurrentCombatStats(lvl);
    return stats ? stats.effSecPerKill : null;
  }

  function getActiveBuildForLevel(lvl) {
    var l = Number(lvl) || (state.ab && state.ab.enemyLevel) || state.level || 1;
    var lockedEntry = state.lockedPlans && state.lockedPlans[String(l)];
    var lockedTarget = lockedEntry ? findLockedTarget(l, lockedEntry) : null;
    if (lockedTarget) return lockedTarget;
    if (simState.config && Number(simState.config.level) === l && simState.config.items) {
      var sItems = {};
      var sLimbs = 0;
      var eqList = simState.config.equipped || [];
      for (var ik in simState.config.items) {
        var sit = simState.config.items[ik];
        var isEq = false;
        var itLv = 1;
        if (typeof sit === "object" && sit) {
          isEq = !!sit.equipped;
          itLv = sit.level || 1;
        } else if (typeof sit === "number") {
          isEq = eqList.indexOf(ik) !== -1;
          itLv = sit;
        }
        if (isEq) {
          sItems[ik] = itLv;
          sLimbs++;
        }
      }
      if (sLimbs > 0) {
        return {
          n: (simState.pendingLockBuild && simState.pendingLockBuild.n) || "Simulator Loadout",
          i: sItems,
          l: sLimbs,
          r: simState.config.ring ? { lv: simState.config.ring.level, m: (simState.config.ring.mods || []).slice() } : undefined
        };
      }
    }
    var rec = bestRunnable(l);
    if (rec) return rec;
    return null;
  }

  var pushSimCache = {};
  function getPushSecPerKill(enemyLvl) {
    var l = Number(enemyLvl) || (state.ab && state.ab.enemyLevel) || state.level || 1;

    // 1. If simulator has live session results for this level, use them directly
    if (simState.session && simState.config && Number(simState.config.level) === l) {
      var snap = simState.session.getState();
      if (snap) {
        var sKillSec = (snap.avgLast100Sec && snap.avgLast100Sec > 0) ? snap.avgLast100Sec : snap.timePerKill;
        if (sKillSec && sKillSec > 0) {
          var sWr = (snap.winRate && snap.winRate > 0) ? (snap.winRate / 100) : 1;
          return sKillSec / sWr;
        }
      }
    }

    // 2. If a plan is explicitly locked for this level, use that locked build's simulated speed
    var lockedEntry = state.lockedPlans && state.lockedPlans[String(l)];
    var lockedTarget = lockedEntry ? findLockedTarget(l, lockedEntry) : null;
    if (lockedTarget) {
      var bK = buildSecPerKill(lockedTarget, l);
      if (bK && bK > 0 && isFinite(bK)) {
        return bK;
      }
    }

    // 3. Use user's active combat loadout speed against this level
    var curCombatK = getCurrentCombatSecPerKill(l);
    if (curCombatK && curCombatK > 0 && isFinite(curCombatK)) {
      return curCombatK;
    }

    // 4. Fall back to best runnable sheet build
    var rec = bestRunnable(l);
    if (rec) {
      var recK = buildSecPerKill(rec, l);
      if (recK && recK > 0 && isFinite(recK)) {
        return recK;
      }
    }

    if (state.simulatedLevel === l && state.simulatedDustTimePerKill > 0) {
      var wr = (state.simulatedDustWinRate && state.simulatedDustWinRate > 0) ? (state.simulatedDustWinRate / 100) : 1;
      return state.simulatedDustTimePerKill / wr;
    }

    return null;
  }

  function combatDiagnostics(rateOverride) {
    var ab = state.ab || {};
    var saveStats = (state.save && state.save.stats) || {};
    var saKills = saveStats.saKills || {};

    var curKills = ab.enemiesKilled || 0;
    var enemyLvl = ab.enemyLevel || state.level || 1;
    var maxLvl = ab.maxEnemyLevel || enemyLvl;
    var needed = saLevelReq(enemyLvl);
    var remaining = Math.max(0, needed - curKills);
    var progressPct = needed > 0 ? Math.min(100, (curKills / needed) * 100) : 0;

    var s = ab.settings || {};
    var isPractice = (s.practice && typeof s.practice.enabled !== 'undefined') ? (s.practice.enabled === 1) : (s.practice === 1);
    var isMaxCleared = Boolean(ab.sealed || ab.canSeal || enemyLvl > 160);
    var autoLevel = ab.autoLevel !== false;

    var mode = "farming";
    var modeLabel = "Farming";
    if (isPractice) {
      mode = "practice";
      modeLabel = "Practicing (Rewards Paused)";
    } else if (isMaxCleared || (enemyLvl >= maxLvl && (ab.canSeal || ab.sealed))) {
      mode = "max";
      modeLabel = "Farming (Max Level Cleared)";
    } else if (enemyLvl >= maxLvl) {
      mode = "pushing";
      modeLabel = "Completing SA " + enemyLvl + " (Pushing)";
    } else {
      mode = "farming";
      modeLabel = "Farming SA " + enemyLvl;
    }

    var totalKillsAllTime = (typeof saKills.valueTotal === 'number' && saKills.valueTotal > 0)
      ? saKills.valueTotal
      : ((typeof saKills.value === 'number' && saKills.value > 0) ? saKills.value : curKills);
    var totalKillsPortal = (typeof saKills.value === 'number') ? saKills.value : 0;

    var sKills = ab.sessionEnemiesKilled || 0;
    var sDeaths = ab.sessionTrimpsKilled || 0;
    var sTotal = sKills + sDeaths;
    var winRate = sTotal > 0 ? (sKills / sTotal * 100) : 100;

    var curRec = bestRunnable(enemyLvl);
    var userRate = (typeof rateOverride === "number" && rateOverride > 0)
      ? rateOverride
      : (typeof activeDustPs === "function" ? activeDustPs(curRec) : 0);

    var secPerKill = null;
    var etaSec = null;
    var lockedEntry = state.lockedPlans && state.lockedPlans[String(enemyLvl)];
    var lockedTarget = lockedEntry ? findLockedTarget(enemyLvl, lockedEntry) : null;
    var activeTargetBuild = lockedTarget || getActiveBuildForLevel(enemyLvl);
    var buildName = lockedTarget ? lockedTarget.n : (mode === "pushing" ? "Current Loadout" : (activeTargetBuild ? (activeTargetBuild.n || "Build") : null));
    var dustPerKill = 0;
    if (curRec && curRec.d && curRec.k) {
      dustPerKill = curRec.d * curRec.k;
    } else {
      var baseReward = (1 + ((enemyLvl - 1) * 5)) * Math.pow(1.19, enemyLvl - 1);
      if (enemyLvl >= 50) baseReward *= Math.pow(1.1, enemyLvl - 49);
      dustPerKill = baseReward * 4.3;
    }

    if (mode === "pushing" && remaining > 0) {
      // 1. Accurately simulated kill time from live save combat
      var simSpk = getPushSecPerKill(enemyLvl);
      if (simSpk && simSpk > 0 && isFinite(simSpk)) {
        secPerKill = simSpk;
      } else if (curRec && curRec.k && curRec.k > 0) {
        secPerKill = curRec.k;
      } else {
        var anyBuilds2 = buildsFor(enemyLvl);
        var foundK = null;
        for (var bIdx = 0; bIdx < anyBuilds2.length; bIdx++) {
          if (anyBuilds2[bIdx] && anyBuilds2[bIdx].k && anyBuilds2[bIdx].k > 0) {
            foundK = anyBuilds2[bIdx].k;
            break;
          }
        }
        secPerKill = foundK || 15;
      }

      if (secPerKill && secPerKill > 0 && isFinite(secPerKill)) {
        etaSec = remaining * secPerKill;
      }
    }

    return {
      mode: mode,
      modeLabel: modeLabel,
      isPushing: mode === "pushing",
      isFarming: mode === "farming",
      isPractice: isPractice,
      isMax: mode === "max",
      autoLevel: autoLevel,
      enemyLevel: enemyLvl,
      maxEnemyLevel: maxLvl,
      curKills: curKills,
      neededKills: needed,
      remainingKills: remaining,
      progressPct: progressPct,
      dustPerKill: dustPerKill,
      secPerKill: secPerKill,
      etaSec: etaSec,
      buildName: buildName,
      totalKillsAllTime: totalKillsAllTime,
      totalKillsPortal: totalKillsPortal,
      sessionKills: sKills,
      sessionDeaths: sDeaths,
      sessionTotal: sTotal,
      winRate: winRate,
      winRateStr: winRate.toFixed(1) + "%"
    };
  }

  function macroBonusInfo() {
    var b = (state.ab && state.ab.bonuses) || {};
    var radonLvl = (b.Radon && typeof b.Radon.level === "number") ? b.Radon.level : (typeof b.Radon === "number" ? b.Radon : 0);
    var statsLvl = (b.Stats && typeof b.Stats.level === "number") ? b.Stats.level : (typeof b.Stats === "number" ? b.Stats : 0);
    var scaffoldLvl = (b.Scaffolding && typeof b.Scaffolding.level === "number") ? b.Scaffolding.level : (typeof b.Scaffolding === "number" ? b.Scaffolding : 0);
    return {
      radonLvl: radonLvl,
      radonPct: radonLvl * 10,
      statsLvl: statsLvl,
      statsPct: statsLvl * 10,
      scaffoldLvl: scaffoldLvl
    };
  }

  function getBonusList(bnk, rate) {
    var mb = macroBonusInfo();
    var rRate = (typeof rate === "number" && rate > 0) ? rate : (activeDustPs() || 0);
    var bnkObj = bnk || bank();
    var bankD = bnkObj ? (bnkObj.dust || 0) : 0;
    var bankS = bnkObj ? (bnkObj.shards || 0) : 0;
    var limbs = extraLimbs();

    var list = [
      {
        id: "Radon",
        name: "Radon Bonus",
        shortName: "Radon",
        icon: ICONS.target,
        level: mb.radonLvl,
        maxLevel: 50,
        isMax: mb.radonLvl >= 50,
        currency: "dust",
        isShards: false,
        effect: "+" + mb.radonPct + "% Radon in U2",
        nextEffect: "+10% Radon",
        cost: mb.radonLvl >= 50 ? 0 : Math.ceil(30000 * Math.pow(3, mb.radonLvl))
      },
      {
        id: "Stats",
        name: "Stats Bonus",
        shortName: "Stats",
        icon: ICONS.sword,
        level: mb.statsLvl,
        maxLevel: 50,
        isMax: mb.statsLvl >= 50,
        currency: "dust",
        isShards: false,
        effect: "+" + mb.statsPct + "% Attack & Health in U2",
        nextEffect: "+10% Attack & Health",
        cost: mb.statsLvl >= 50 ? 0 : Math.ceil(20000 * Math.pow(3, mb.statsLvl))
      },
      {
        id: "Scaffolding",
        name: "Scaffolding",
        shortName: "Scaffolding",
        icon: ICONS.hammer,
        level: mb.scaffoldLvl,
        maxLevel: 18,
        isMax: mb.scaffoldLvl >= 18,
        currency: "shards",
        isShards: true,
        effect: mb.scaffoldLvl > 0 ? (mb.scaffoldLvl + " / 18 Scaffolding") : "None",
        nextEffect: "+100% Housing (Compounding)",
        cost: mb.scaffoldLvl >= 18 ? 0 : Math.ceil(50 * Math.pow(10, mb.scaffoldLvl))
      },
      {
        id: "Extra_Limbs",
        name: "Extra Limbs",
        shortName: "Limbs",
        icon: ICONS.limb,
        level: limbs,
        maxLevel: 14,
        isMax: limbs >= 14,
        currency: "dust",
        isShards: false,
        effect: (4 + limbs) + " item slots (4 base + " + limbs + ")",
        nextEffect: "+1 Item Slot (" + (5 + limbs) + " total)",
        cost: limbs >= 14 ? 0 : Math.ceil(100 * Math.pow(100, limbs))
      }
    ];

    list.forEach(function (item) {
      if (item.isMax) {
        item.isReady = false;
        item.wait = 0;
        item.need = 0;
        item.statusTxt = "Max Level";
        item.statusCls = "muted";
      } else {
        var curBank = item.isShards ? bankS : bankD;
        item.need = Math.max(0, item.cost - curBank);
        if (item.cost <= 0 || curBank >= item.cost) {
          item.isReady = true;
          item.wait = 0;
          item.statusTxt = "Upgrade Ready";
          item.statusCls = "buy";
        } else {
          item.isReady = false;
          if (item.isShards && !canEarnShards()) {
            item.wait = Infinity;
            item.statusTxt = "Locked (SA 51+)";
            item.statusCls = "warn";
          } else {
            item.wait = timeForCost(item.cost, item.isShards, rRate, bankD, bankS);
            item.statusTxt = (isFinite(item.wait) && item.wait > 0) ? fmtTime(item.wait) : "—";
            item.statusCls = "warn";
          }
        }
      }
    });

    return list;
  }

  function renderBonusesHtml(mb, bnk, rate) {
    var bList = getBonusList(bnk, rate);
    var macroVal = "+" + mb.radonPct + "% Rn · +" + mb.statsPct + "% Stats" + (mb.scaffoldLvl > 0 ? (" · " + mb.scaffoldLvl + " Scaff") : "");
    var macroTitle = "Spire Macro Bonuses (Purchased with Dust / Shards):" +
      bList.map(function (b) {
        var statusStr = b.isMax
          ? "Max Level"
          : (b.isReady
            ? ("Upgrade Ready (" + prettify(b.cost) + " " + b.currency + ")")
            : (b.statusTxt + " to save (" + prettify(b.cost) + " " + b.currency + (b.need > 0 ? (", need " + prettify(b.need)) : "") + ")"));
        return "\n• " + b.name + ": Lv " + b.level + (b.maxLevel ? ("/" + b.maxLevel) : "") + " (" + b.effect + ") — " + statusStr;
      }).join("");

    var rowsHtml = bList.map(function (b) {
      var badgeHtml = b.isMax
        ? "<span class='bp-item-status pill sm muted'>Max Level</span>"
        : (b.isReady
          ? ("<span class='bp-item-status pill sm buy'>" + ICONS.check + " Upgrade Ready</span>")
          : ("<span class='bp-item-status pill sm warn'>" + ICONS.clock + " " + esc(b.statusTxt) + "</span>"));
      var costStr = b.isMax
        ? "—"
        : ("Cost: " + prettify(b.cost) + " " + b.currency + (b.need > 0 ? (" (need " + prettify(b.need) + ")") : ""));
      var lvlStr = "Lv " + b.level + (b.maxLevel ? (" / " + b.maxLevel) : "");
      return "<div class='bp-item' data-bonus-id='" + esc(b.id) + "'>" +
        "<div class='bp-item-head'>" +
          "<span class='bp-item-name'>" + b.icon + " " + esc(b.name) + "</span>" +
          "<span class='bp-item-lvl mono'>" + esc(lvlStr) + "</span>" +
          badgeHtml +
        "</div>" +
        "<div class='bp-item-sub'>" +
          "<span class='bp-item-desc'>" + esc(b.effect) + "</span>" +
          "<span class='bp-item-cost mono'>" + esc(costStr) + "</span>" +
        "</div>" +
      "</div>";
    }).join("");

    var rRate = (typeof rate === "number" && rate > 0) ? rate : (activeDustPs() || 0);
    var rateStr = rRate ? (prettify(rRate) + "/s") : "No rate";
    var bnkObj = bnk || bank();
    var bankStr = "Bank: " + prettify(bnkObj.dust) + " dust" + (canEarnShards() ? (" · " + prettify(bnkObj.shards) + " shards") : "");

    return "<div class='fact-wrap fact-spire-buffs-wrap' id='factSpireBuffsWrap' tabindex='0'>" +
      '<div class="fact fact-spire-buffs" id="factSpireBuffs" title="' + esc(macroTitle) + '" style="cursor:pointer;" onclick="window.saOpenPresets&&saOpenPresets()">' +
        ICONS.target + '<span>Spire Buffs</span><b id="factSpireBuffsVal">' + esc(macroVal) + '</b>' +
      '</div>' +
      "<div class='bonuses-popover' id='bonusesPopover' role='tooltip'>" +
        "<div class='bp-head'>" +
          "<div class='bp-title'>" + ICONS.target + "<span>Spire Bonuses</span></div>" +
          "<span class='bp-rate mono' id='bpLiveRate'>" + esc(rateStr) + "</span>" +
        "</div>" +
        "<div class='bp-list' id='bpList'>" + rowsHtml + "</div>" +
        "<div class='bp-footer' id='bpFooter'>" +
          "<span id='bpBankInfo'>" + esc(bankStr) + "</span>" +
          "<span class='bp-link' onclick='window.saOpenPresets&&saOpenPresets()'>Presets &amp; Settings →</span>" +
        "</div>" +
      "</div>" +
    "</div>";
  }

  function updateBonusesPopover(rate, bnk) {
    var pop = $("bonusesPopover");
    var fact = $("factSpireBuffs");
    if (!fact && !pop) return;
    var bList = getBonusList(bnk, rate);

    var macroTitle = "Spire Macro Bonuses (Purchased with Dust / Shards):" +
      bList.map(function (b) {
        var statusStr = b.isMax
          ? "Max Level"
          : (b.isReady
            ? ("Upgrade Ready (" + prettify(b.cost) + " " + b.currency + ")")
            : (b.statusTxt + " to save (" + prettify(b.cost) + " " + b.currency + (b.need > 0 ? (", need " + prettify(b.need)) : "") + ")"));
        return "\n• " + b.name + ": Lv " + b.level + (b.maxLevel ? ("/" + b.maxLevel) : "") + " (" + b.effect + ") — " + statusStr;
      }).join("");
    if (fact && fact.title !== macroTitle) fact.title = macroTitle;

    var popRate = $("bpLiveRate");
    if (popRate) {
      var rRate = (typeof rate === "number" && rate > 0) ? rate : (activeDustPs() || 0);
      popRate.textContent = rRate ? (prettify(rRate) + "/s") : "No rate";
    }
    var popBank = $("bpBankInfo");
    if (popBank && bnk) {
      popBank.textContent = "Bank: " + prettify(bnk.dust) + " dust" + (canEarnShards() ? (" · " + prettify(bnk.shards) + " shards") : "");
    }

    bList.forEach(function (b) {
      var itemEl = document.querySelector("#bpList [data-bonus-id='" + b.id + "']");
      if (!itemEl) return;
      var stEl = itemEl.querySelector(".bp-item-status");
      if (stEl) {
        var newCls = "bp-item-status pill sm " + b.statusCls;
        var newHtml = b.isMax
          ? "Max Level"
          : (b.isReady
            ? (ICONS.check + " Upgrade Ready")
            : (ICONS.clock + " " + esc(b.statusTxt)));
        if (stEl.className !== newCls) stEl.className = newCls;
        if (stEl.innerHTML !== newHtml) stEl.innerHTML = newHtml;
      }
      var costEl = itemEl.querySelector(".bp-item-cost");
      if (costEl) {
        var costStr = b.isMax
          ? "—"
          : ("Cost: " + prettify(b.cost) + " " + b.currency + (b.need > 0 ? (" (need " + prettify(b.need) + ")") : ""));
        if (costEl.textContent !== costStr) costEl.textContent = costStr;
      }
    });
  }

  function saSettingsAudit() {
    var ab = state.ab || {};
    var s = ab.settings || {};
    var practiceVal = (s.practice && typeof s.practice.enabled !== 'undefined') ? s.practice.enabled : s.practice;
    var loadRingVal = (s.loadRing && typeof s.loadRing.enabled !== 'undefined') ? s.loadRing.enabled : (typeof s.loadRing !== 'undefined' ? s.loadRing : 1);
    var loadLevelVal = (s.loadLevel && typeof s.loadLevel.enabled !== 'undefined') ? s.loadLevel.enabled : (typeof s.loadLevel !== 'undefined' ? s.loadLevel : 1);
    var loadHideVal = (s.loadHide && typeof s.loadHide.enabled !== 'undefined') ? s.loadHide.enabled : (typeof s.loadHide !== 'undefined' ? s.loadHide : 0);
    return {
      autoLevel: ab.autoLevel !== false,
      practiceActive: practiceVal === 1,
      loadRingDisabled: loadRingVal === 0,
      loadLevelDisabled: loadLevelVal === 0,
      loadHideDisabled: loadHideVal === 1
    };
  }

  function parsePreset(slot) {
    var ab = state.ab;
    var p = (ab && ab.presets && ab.presets[slot]) || [];
    var names = (ab && ab.presets && ab.presets.names) || [];
    var slotIdx = (slot === "p1") ? 0 : (slot === "p2" ? 1 : 2);
    var name = (names[slotIdx] && String(names[slotIdx]).trim()) || ("Preset " + (slotIdx + 1));
    var items = [];
    var level = null;
    var ring = [];
    for (var i = 0; i < p.length; i++) {
      var entry = p[i];
      if (Array.isArray(entry)) {
        if (entry[0] === "level") level = entry[1];
        else if (entry[0] === "ring") ring = entry.slice(1);
      } else if (typeof entry === "string") {
        items.push(entry);
      }
    }
    return {
      slot: slot,
      slotTag: "P" + (slotIdx + 1),
      slotNum: slotIdx + 1,
      name: name,
      items: items,
      level: level,
      ringMods: ring,
      isEmpty: items.length === 0 && !level && ring.length === 0
    };
  }

  function getAllPresets() {
    return ["p1", "p2", "p3"].map(parsePreset);
  }

  function comparePresetToBuild(preset, build) {
    if (!preset || preset.isEmpty || !build || !build.i) return null;
    var reqKeys = Object.keys(build.i);
    var missing = reqKeys.filter(function (k) { return preset.items.indexOf(k) === -1; });
    var extra = preset.items.filter(function (k) { return reqKeys.indexOf(k) === -1; });
    var ringMatch = true;
    if (build.r && build.r.m && build.r.m.length > 0) {
      ringMatch = build.r.m.every(function (m) { return preset.ringMods.indexOf(m) !== -1; });
    }
    var swaps = missing.length;
    var isExact = missing.length === 0 && extra.length === 0 && ringMatch;
    var isClose = swaps <= 1 && extra.length <= 1 && ringMatch;
    return {
      preset: preset,
      isExact: isExact,
      isClose: isClose,
      swaps: swaps,
      missing: missing,
      extra: extra,
      ringMatch: ringMatch
    };
  }

  function findBestPresetMatch(build) {
    if (!build) return null;
    var presets = getAllPresets();
    var best = null;
    for (var i = 0; i < presets.length; i++) {
      var res = comparePresetToBuild(presets[i], build);
      if (!res) continue;
      if (res.isExact) return res;
      if (res.isClose) {
        if (!best || res.swaps < best.swaps) best = res;
      }
    }
    return best;
  }

  function ownsContractZone(zone) {
    if (!zone) return true;
    for (var key in SA_DATA.items) {
      if (SA_DATA.items[key].zone === zone && userOwns(key)) return true;
    }
    return false;
  }

  function decodeSave(raw) {
    var s = String(raw || "").replace(/\s+/g, "");
    if (!s) throw new Error("Paste a Trimps export string (in Trimps, click Export and copy to clipboard).");
    if (s.length > 2000000) throw new Error("Save string exceeds maximum allowed size.");
    var json = null;
    try {
      json = LZString.decompressFromBase64(s);
    } catch (e) {
      json = null;
    }
    if (!json || json.charAt(0) !== "{") {
      throw new Error("Could not decode that export string. Make sure you copied the full Base64 save string from Trimps → Export.");
    }
    var data;
    try {
      data = JSON.parse(json);
    } catch (e) {
      throw new Error("Decoded string is not valid JSON. Check that the export string was not truncated.");
    }
    if (!data || typeof data !== "object" || !data.global) {
      throw new Error("That does not look like a full Trimps save export.");
    }
    var ab = data.global.autoBattleData;
    var highestRadon = Number(data.global.highestRadonLevelCleared) || 0;
    if (!ab || typeof ab !== "object" || !ab.items) {
      throw new Error("This save has no Spire Assault data yet. Reach Universe 2 (Radon) to unlock Spire Assault.");
    }
    ab.enemyLevel = Math.max(1, Math.min(999, Math.floor(Number(ab.enemyLevel) || 1)));
    ab.maxEnemyLevel = Math.max(1, Math.min(999, Math.floor(Number(ab.maxEnemyLevel) || ab.enemyLevel || 1)));
    ab.dust = Math.max(0, Number(ab.dust) || 0);
    ab.shards = Math.max(0, Number(ab.shards) || 0);
    ab.shardDust = Math.max(0, Number(ab.shardDust) || 0);
    ab.enemiesKilled = Math.max(0, Number(ab.enemiesKilled) || 0);
    ab.sessionEnemiesKilled = Math.max(0, Number(ab.sessionEnemiesKilled) || 0);
    ab.sessionTrimpsKilled = Math.max(0, Number(ab.sessionTrimpsKilled) || 0);
    return { save: data, ab: ab, radon: Math.max(0, Math.floor(highestRadon)) };
  }

  function sheetLevels() {
    return Object.keys(SA_DATA.builds).map(Number).sort(function (a, b) { return a - b; });
  }

  function buildsFor(level) { return SA_DATA.builds[String(level)] || []; }

  function runnable(build) {
    var keys = Object.keys(build.i || {});
    for (var i = 0; i < keys.length; i++) {
      var k = keys[i];
      if (!userOwns(k) || userLevel(k) < build.i[k]) return false;
    }
    if ((build.l || 0) > maxSlots()) return false;
    if (build.r) {
      if (!hasRing() || ringLevel() < (build.r.lv || 1)) return false;
    }
    if (build.c && !ownsContractZone(build.c)) return false;
    return true;
  }

  function bestRunnable(level) {
    var list = buildsFor(level).filter(runnable);
    var mode = getLevelMode(level);
    if (mode === "push") {
      list.sort(function (a, b) {
        var ka = buildSecPerKill(a, level);
        var kb = buildSecPerKill(b, level);
        if (Math.abs(ka - kb) > 0.01) return ka - kb;
        return (b.d || 0) - (a.d || 0);
      });
    } else {
      list.sort(function (a, b) {
        var da = a.d || 0;
        var db = b.d || 0;
        if (Math.abs(da - db) > 0.01) return db - da;
        return buildSecPerKill(a, level) - buildSecPerKill(b, level);
      });
    }
    return list[0] || null;
  }

  function gapSteps(build) {
    var steps = [];
    var keys = Object.keys(build.i || {});
    keys.forEach(function (k) {
      var need = build.i[k];
      var cat = itemCat(k);
      if (!userOwns(k)) {
        if (cat && cat.zone) {
          var active = state.ab.activeContract === k;
          var reach = (state.radon + 1) >= cat.zone;
          steps.push({
            kind: "contract",
            key: k,
            zone: cat.zone,
            active: active,
            reach: reach,
            cost: contractPrice(k),
            shards: isShards(k),
            after: { kind: "level", key: k, from: 1, to: need, cost: upgradeCostRange(k, 1, need), shards: isShards(k) }
          });
        } else {
          steps.push({ kind: "missing", key: k });
        }
      } else if (userLevel(k) < need) {
        steps.push({ kind: "level", key: k, from: userLevel(k), to: need, cost: upgradeCostRange(k, userLevel(k), need), shards: isShards(k) });
      }
    });
    if ((build.l || 0) > maxSlots()) {
      var from = extraLimbs();
      var to = (build.l || 4) - 4;
      if (to > from) steps.push({ kind: "limbs", from: from, to: to, cost: extraLimbCost(from, to), shards: false });
    }
    if (build.r) {
      if (!hasRing()) {
        steps.push({ kind: "ringUnlock", cost: oneTimerRingCost(), shards: true });
        if ((build.r.lv || 1) > 1) steps.push({ kind: "ring", from: 1, to: build.r.lv, cost: ringCostRange(1, build.r.lv), shards: true });
      } else if (ringLevel() < (build.r.lv || 1)) {
        steps.push({ kind: "ring", from: ringLevel(), to: build.r.lv, cost: ringCostRange(ringLevel(), build.r.lv), shards: true });
      }
    }
    return sortSmartSteps(build, steps);
  }

  function readFilters() {
    return {
      ring: $("filtRing") ? $("filtRing").checked : state.filters.ring,
      contract: $("filtContract") ? $("filtContract").checked : state.filters.contract,
      limb: $("filtLimb") ? $("filtLimb").checked : state.filters.limb,
      hideUnused: $("filtHideUnused") ? $("filtHideUnused").checked : state.filters.hideUnused
    };
  }

  function applyFilterBoxes() {
    if ($("filtRing")) $("filtRing").checked = state.filters.ring;
    if ($("filtContract")) $("filtContract").checked = state.filters.contract;
    if ($("filtLimb")) $("filtLimb").checked = state.filters.limb;
    if ($("filtHideUnused")) $("filtHideUnused").checked = !!state.filters.hideUnused;
  }

  function stepsNeed(steps, kind) {
    if (kind === "ring") return steps.some(function (s) { return s.kind === "ring" || s.kind === "ringUnlock"; });
    if (kind === "contract") return steps.some(function (s) { return s.kind === "contract"; });
    if (kind === "limb") return steps.some(function (s) { return s.kind === "limbs"; });
    return false;
  }

  function allowedUpgrade(build) {
    var f = state.filters;
    var steps = gapSteps(build);
    if (!f.ring && stepsNeed(steps, "ring")) return false;
    if (!f.contract && stepsNeed(steps, "contract")) return false;
    if (!f.limb && stepsNeed(steps, "limb")) return false;
    return true;
  }

  function stepsCostDust(steps) {
    var dust = 0, shards = 0;
    steps.forEach(function (s) {
      var c = s.cost || 0;
      if (s.shards) shards += c; else dust += c;
      if (s.after) {
        if (s.after.shards) shards += s.after.cost || 0;
        else dust += s.after.cost || 0;
      }
    });
    return { dust: dust, shards: shards, combined: Math.max(dust, shards * SHARD_DUST) };
  }

  function stepsWaitInfo(steps, rate, bnk) {
    var c = calculateStepWaitTimes(steps, bnk, rate);
    var totals = stepsCostDust(steps);
    var remDustEq = Math.max(c.remDust, c.remShards * SHARD_DUST);
    return {
      dust: totals.dust,
      shards: totals.shards,
      combined: totals.combined,
      remDust: c.remDust,
      remShards: c.remShards,
      remDustEq: remDustEq,
      waitDust: c.waitDust,
      waitShards: c.waitShards,
      wait: c.totalWait
    };
  }

  function calculateStepWaitTimes(steps, bnk, defaultRate) {
    if (!steps || !steps.length) {
      return {
        cumDustSec: 0,
        cumShardSec: 0,
        totalWait: 0,
        waitDust: 0,
        waitShards: 0,
        remDust: 0,
        remShards: 0,
        totalCostDust: 0,
        totalCostShards: 0
      };
    }
    var availD = bnk ? (bnk.dust || 0) : 0;
    var availS = bnk ? (bnk.shards || 0) : 0;
    var cumDustSec = 0;
    var cumShardSec = 0;
    var remDustTotal = 0;
    var remShardsTotal = 0;
    var totalCostDust = 0;
    var totalCostShards = 0;

    for (var i = 0; i < steps.length; i++) {
      var s = steps[i];
      if (s.completed) {
        s.estWait = 0;
        continue;
      }
      var cost = stepFullCost(s);
      var stepR = (s._stepRate && s._stepRate > 0) ? s._stepRate : defaultRate;
      if (s.shards) {
        totalCostShards += cost;
        if (availS >= cost) {
          availS -= cost;
          s.estWait = cumShardSec;
        } else {
          var needS = cost - availS;
          availS = 0;
          remShardsTotal += needS;
          var farmSec = (canEarnShards() && stepR > 0) ? (needS * SHARD_DUST) / stepR : Infinity;
          cumShardSec += farmSec;
          s.estWait = cumShardSec;
        }
      } else {
        totalCostDust += cost;
        if (availD >= cost) {
          availD -= cost;
          s.estWait = cumDustSec;
        } else {
          var needD = cost - availD;
          availD = 0;
          remDustTotal += needD;
          var farmSec = stepR > 0 ? (needD / stepR) : Infinity;
          cumDustSec += farmSec;
          s.estWait = cumDustSec;
        }
      }
    }

    var totalWait = Math.max(cumDustSec, cumShardSec);
    return {
      cumDustSec: cumDustSec,
      cumShardSec: cumShardSec,
      totalWait: totalWait,
      waitDust: cumDustSec,
      waitShards: cumShardSec,
      remDust: remDustTotal,
      remShards: remShardsTotal,
      totalCostDust: totalCostDust,
      totalCostShards: totalCostShards
    };
  }

  function sheetDustPs(build) { return build && build.d ? build.d : 0; }

  function getLoadoutSignature() {
    if (!state.ab) return "";
    var eq = [];
    var allItems = state.ab.items || {};
    for (var k in allItems) {
      if (allItems[k] && allItems[k].equipped) {
        eq.push(k + "=" + (allItems[k].level || 1));
      }
    }
    eq.sort();
    var r = hasRing() ? (ringLevel() + "[" + ringMods().slice().sort().join(",") + "]") : "noRing";
    var fightLvl = state.ab.enemyLevel || 0;
    var slots = maxSlots();
    var contract = state.ab.activeContract || "";
    return "fight:" + fightLvl + "|slots:" + slots + "|contract:" + contract + "|ring:" + r + "|eq:" + eq.join(";");
  }

  function syncCustomDustRate(level) {
    if (!state.ab) return;
    var curSig = getLoadoutSignature();
    var lvlKey = String(level);
    var entry = state.customDustRates && state.customDustRates[lvlKey];

    if (entry && entry.sig === curSig && entry.rate) {
      state.dustOverride = entry.rate;
      if ($("dustOverride") && document.activeElement !== $("dustOverride") && $("dustOverride").value !== entry.rate) {
        $("dustOverride").value = entry.rate;
      }
      if ($("dustOverride")) {
        $("dustOverride").title = "Custom rate locked to SA " + level + " and current loadout. Reverts to estimate if loadout changes.";
      }
    } else {
      state.dustOverride = "";
      if ($("dustOverride") && document.activeElement !== $("dustOverride") && $("dustOverride").value !== "") {
        $("dustOverride").value = "";
      }
      if ($("dustOverride")) {
        $("dustOverride").title = "Enter your in-game dust/s to override the sheet estimate for SA " + level;
      }
      if (entry && entry.sig !== curSig) {
        delete state.customDustRates[lvlKey];
        try {
          localStorage.setItem("saPlanner.customRates", JSON.stringify(state.customDustRates));
        } catch (e) {}
      }
    }
  }

  function canEarnShards() {
    if (!state.ab) return false;
    var fLevel = state.ab.enemyLevel || state.level || state.ab.maxEnemyLevel || 0;
    return fLevel > 50;
  }

  function autoCalibrateSaveRate() {
    if (typeof SASimEngine === "undefined" || !state.save) return;
    var fightLevel = (state.ab && (state.ab.enemyLevel || state.ab.maxEnemyLevel)) || state.level || 1;
    state.simulatedDustPs = 0;
    state.simulatedDustTimePerKill = 0;
    state.simulatedDustWinRate = 0;
    state.simulatedLevel = fightLevel;
    try {
      var simRes = SASimEngine.runBatch({ save: state.save, level: fightLevel }, { kills: 300, maxFights: 600, ticks: 150000 });
      if (simRes && simRes.dustPs > 0) {
        state.simulatedDustPs = simRes.dustPs;
        state.simulatedDustTimePerKill = simRes.timePerKill || simRes.secPerKill;
        state.simulatedDustWinRate = simRes.winRate;
        state.simulatedLevel = fightLevel;
        if ($("dustOverride") && document.activeElement !== $("dustOverride") && !state.dustOverride) {
          $("dustOverride").placeholder = prettify(simRes.dustPs) + "/s (Simulated)";
        }
      }
    } catch (e) {
      console.warn("Auto-calibration sim error:", e);
    }
  }

  function activeDustPs(recommended) {
    var over = parseNum(state.dustOverride);
    if (isFinite(over) && over > 0) return over;
    if (state.simulatedDustPs && state.simulatedDustPs > 0) {
      return state.simulatedDustPs;
    }
    if (recommended && recommended.d) return recommended.d;
    if (state.ab) {
      var fightLevel = (state.ab && (state.ab.enemyLevel || state.ab.maxEnemyLevel)) || state.level;
      if (fightLevel) {
        var fightRec = bestRunnable(fightLevel);
        if (fightRec && fightRec.d) return fightRec.d;
      }
    }
    return sheetDustPs(recommended) || 0;
  }

  function timeForCost(cost, shards, rate, bankDust, bankShards) {
    if (!rate) return Infinity;
    if (shards) {
      var needS = Math.max(0, cost - (bankShards || 0));
      return (needS * SHARD_DUST) / rate;
    }
    return Math.max(0, cost - (bankDust || 0)) / rate;
  }

  function bank(atTime) {
    if (!state.ab) {
      return { dust: 0, shards: 0, rawDust: 0, rawShards: 0, dustEarned: 0, shardsEarned: 0, elapsedSec: 0, rate: 0 };
    }
    var baseDust = Number(state.ab.dust) || 0;
    var baseShards = Number(state.ab.shards) || 0;
    var baseShardDust = Number(state.ab.shardDust) || 0;
    var now = atTime || Date.now();
    var loadedAt = state.saveLoadedAt || now;
    var elapsedSec = Math.max(0, (now - loadedAt) / 1000);
    if (elapsedSec > 7 * 86400) elapsedSec = 7 * 86400;
    var rate = activeDustPs();
    var dustEarned = rate > 0 ? (rate * elapsedSec) : 0;
    var shardsEarned = 0;
    if (canEarnShards() && rate > 0) {
      shardsEarned = Math.floor((baseShardDust + dustEarned) / SHARD_DUST);
    }
    return {
      dust: baseDust + dustEarned,
      shards: baseShards + shardsEarned,
      rawDust: baseDust,
      rawShards: baseShards,
      dustEarned: dustEarned,
      shardsEarned: shardsEarned,
      elapsedSec: elapsedSec,
      rate: rate
    };
  }

  function setTheme(theme) {
    state.theme = theme === "light" ? "light" : "dark";
    document.documentElement.setAttribute("data-theme", state.theme);
    var sun = $("icoSun"), moon = $("icoMoon"), lbl = $("themeLbl");
    if (sun && moon) {
      sun.classList.toggle("hidden", state.theme === "light");
      moon.classList.toggle("hidden", state.theme === "dark");
    }
    if (lbl) lbl.textContent = state.theme === "dark" ? "Light" : "Dark";
    else if ($("themeBtn")) $("themeBtn").textContent = state.theme === "dark" ? "Light" : "Dark";
    try { localStorage.setItem("saPlanner.theme", state.theme); } catch (e) {}
  }

  function loadPrefs() {
    try {
      var t = localStorage.getItem("saPlanner.theme");
      if (t) setTheme(t);
      var cr = localStorage.getItem("saPlanner.customRates");
      if (cr) {
        try {
          var parsedCr = JSON.parse(cr);
          if (parsedCr && typeof parsedCr === "object") state.customDustRates = parsedCr;
        } catch (e) {}
      }
      try { localStorage.removeItem("saPlanner.dustPs"); } catch (e) {}
      var f = localStorage.getItem("saPlanner.filters");
      if (f) {
        var parsed = JSON.parse(f);
        state.filters = {
          ring: parsed.ring !== false,
          contract: parsed.contract !== false,
          limb: parsed.limb !== false,
          hideUnused: !!parsed.hideUnused
        };
        applyFilterBoxes();
      }
      var so = localStorage.getItem("saPlanner.stepOrder");
      if (so) {
        var parsedSo = JSON.parse(so);
        if (parsedSo && typeof parsedSo === "object") state.stepOrder = parsedSo;
      }
      var lp = localStorage.getItem("saPlanner.lockedPlans");
      if (lp) {
        var parsedLp = JSON.parse(lp);
        if (parsedLp && typeof parsedLp === "object") state.lockedPlans = parsedLp;
      }
      var sgc = localStorage.getItem("saPlanner.swapGuideCollapsed");
      if (sgc != null) state.swapGuideCollapsed = (sgc === "1");
      var savedStr = localStorage.getItem("saPlanner.saveString");
      var savedTime = localStorage.getItem("saPlanner.saveLoadedAt");
      if (savedStr) {
        $("saveBox").value = savedStr;
        if (savedTime && !isNaN(Number(savedTime))) {
          state.saveLoadedAt = Number(savedTime);
        }
        onLoad(true);
      }
    } catch (e) {}
  }

  function buildId(b) {
    if (!b) return "";
    var items = Object.keys(b.i || {}).sort().map(function (k) { return k + ":" + b.i[k]; }).join(",");
    var rLv = (b.r && b.r.lv) || 0;
    var rMods = (b.r && b.r.m) ? b.r.m.join("+") : "";
    return (b.n || "") + "|" + (b.d || 0) + "|" + (b.l || 0) + "|" + rLv + ":" + rMods + "|" + items;
  }

  function findLockedTarget(level, entry) {
    if (!entry) return null;
    var all = buildsFor(level);
    var id = typeof entry === "object" ? entry.id : null;
    var name = typeof entry === "string" ? entry : entry.name;
    var d = typeof entry === "object" ? entry.d : null;
    var k;
    if (state.lockedPlans && state.lockedPlans[String(level)] && state.lockedPlans[String(level)].customBuild) {
      var cb = state.lockedPlans[String(level)].customBuild;
      if (id && (buildId(cb) === id || cb.id === id)) return cb;
      if (name && (cb.n === name || buildId(cb) === name)) return cb;
    }
    if (id) {
      for (k = 0; k < all.length; k++) {
        if (buildId(all[k]) === id) return all[k];
      }
    }
    if (name && d != null) {
      for (k = 0; k < all.length; k++) {
        if (all[k].n === name && all[k].d === d) return all[k];
      }
    }
    if (name) {
      for (k = 0; k < all.length; k++) {
        if (buildId(all[k]) === name) return all[k];
      }
      var candidates = all.filter(function (b) { return b.n === name; });
      if (candidates.length === 1) return candidates[0];
      if (candidates.length > 1) {
        var allowed = candidates.filter(function (b) { return !runnable(b) && allowedUpgrade(b); });
        if (allowed.length) {
          allowed.sort(function (a, b) { return gapSteps(a).length - gapSteps(b).length; });
          return allowed[0];
        }
        return candidates[0];
      }
    }
    return null;
  }

  function isSameLockedBuild(level, b) {
    if (!b || !state.lockedPlans) return false;
    var entry = state.lockedPlans[String(level)];
    if (!entry) return false;
    var target = findLockedTarget(level, entry);
    return !!(target && buildId(target) === buildId(b));
  }

  function getLockedBuild(level) {
    var entry = (state.lockedPlans && state.lockedPlans[String(level)]) || null;
    if (!entry) return null;
    var target = findLockedTarget(level, entry);
    if (target) return buildId(target);
    return typeof entry === "string" ? entry : (entry.id || entry.name || null);
  }

  function setLockedBuild(level, buildOrId) {
    if (!state.lockedPlans) state.lockedPlans = {};
    var target = (buildOrId && typeof buildOrId === "object")
      ? buildOrId
      : findLockedTarget(level, { id: String(buildOrId), name: String(buildOrId) });
    if (!target) return;
    var rawSteps = expandSteps(gapSteps(target));
    var ordered = applyStepOrder(level, target, rawSteps);
    var steps = ordered.map(function (s) {
      return {
        kind: s.kind,
        key: s.key || null,
        from: s.from,
        to: s.to,
        cost: s.cost,
        shards: !!s.shards,
        zone: s.zone,
        after: s.after || null,
        completed: false
      };
    });
    state.lockedPlans[String(level)] = {
      id: buildId(target),
      name: target.n,
      d: target.d || 0,
      steps: steps,
      customBuild: target
    };
    try { localStorage.setItem("saPlanner.lockedPlans", JSON.stringify(state.lockedPlans)); } catch (e) {}
  }

  function clearLockedBuild(level) {
    state.lastSimulatedStepsSig = null;
    if (state.lockedPlans && state.lockedPlans[String(level)]) {
      var entry = state.lockedPlans[String(level)];
      var target = findLockedTarget(level, entry);
      delete state.lockedPlans[String(level)];
      try { localStorage.setItem("saPlanner.lockedPlans", JSON.stringify(state.lockedPlans)); } catch (e) {}
      if (target) resetStepOrder(level, target);
      else if (entry) {
        var name = typeof entry === "string" ? entry : entry.name;
        if (name) resetStepOrder(level, { n: name });
      }
    }
  }

  function syncLockedPlan(level, target) {
    if (!state.lockedPlans || !target) return null;
    var entry = state.lockedPlans[String(level)];
    if (!entry) return null;
    var lockedTarget = findLockedTarget(level, entry);
    if (!lockedTarget || buildId(lockedTarget) !== buildId(target)) return null;
    var name = target.n;
    var id = buildId(target);

    var currentGaps = expandSteps(gapSteps(target));
    var planSteps = (typeof entry === "object" && Array.isArray(entry.steps)) ? entry.steps : null;

    function evalStep(s) {
      var done = false;
      var curFrom = s.from != null ? s.from : 0;
      var curTo = s.to != null ? s.to : curFrom + 1;
      var curCost = s.cost;
      var curShards = !!s.shards;
      var reach = true;

      if (s.kind === "level") {
        var curLv = userLevel(s.key);
        done = curLv >= curTo || !!s.completed;
        if (!done) {
          curCost = upgradeCostRange(s.key, curFrom, curTo);
          curShards = isShards(s.key);
        }
      } else if (s.kind === "limbs") {
        var curL = extraLimbs();
        done = curL >= curTo || !!s.completed;
        if (!done) {
          curCost = extraLimbCost(curFrom, curTo);
        }
      } else if (s.kind === "ring") {
        var curR = hasRing() ? ringLevel() : 0;
        done = curR >= curTo || !!s.completed;
        if (!done) {
          curCost = ringCostRange(curFrom, curTo);
          curShards = true;
        }
      } else if (s.kind === "ringUnlock") {
        done = hasRing() || !!s.completed;
        if (!done) {
          curCost = oneTimerRingCost();
          curShards = true;
        }
      } else if (s.kind === "contract") {
        var cat = itemCat(s.key);
        var zone = (cat && cat.zone) || s.zone;
        reach = (state.radon + 1) >= zone;
        done = userOwns(s.key) || !!s.completed;
        if (!done) {
          curCost = contractPrice(s.key);
          curShards = isShards(s.key);
        }
      } else if (s.kind === "missing") {
        done = userOwns(s.key) || !!s.completed;
      }

      return {
        kind: s.kind,
        key: s.key || null,
        from: curFrom,
        to: curTo,
        cost: done ? 0 : curCost,
        shards: curShards,
        zone: s.zone,
        after: s.after || null,
        reach: reach,
        completed: done
      };
    }

    if (planSteps && planSteps.length) {
      var expandedPlan = expandSteps(planSteps);
      var evaluatedPlan = expandedPlan.map(evalStep);

      var currentGapMap = {};
      currentGaps.forEach(function (g) { currentGapMap[stepKey(g)] = g; });

      var seenDone = {};
      var doneSteps = [];
      var seenPending = {};
      var pendingSteps = [];

      evaluatedPlan.forEach(function (s) {
        var k = stepKey(s);
        if (s.completed) {
          if (!seenDone[k]) {
            seenDone[k] = true;
            doneSteps.push(s);
          }
        } else {
          if (currentGapMap[k] && !seenPending[k]) {
            seenPending[k] = true;
            pendingSteps.push(currentGapMap[k]);
          }
        }
      });

      currentGaps.forEach(function (g) {
        var k = stepKey(g);
        if (!seenPending[k]) {
          seenPending[k] = true;
          pendingSteps.push(g);
        }
      });

      var sortedPending;
      if (!hasCustomStepOrder(level, target)) {
        sortedPending = sortSmartSteps(target, pendingSteps);
      } else {
        sortedPending = applyStepOrder(level, target, pendingSteps);
      }

      var evaluated = doneSteps.concat(sortedPending);
      state.lockedPlans[String(level)] = { id: id, name: name, d: target.d || 0, steps: evaluated };
      try { localStorage.setItem("saPlanner.lockedPlans", JSON.stringify(state.lockedPlans)); } catch (e) {}
      return evaluated;
    }

    var gapKeyMap = {};
    currentGaps.forEach(function (g) { gapKeyMap[stepKey(g)] = g; });
    var initial = [];

    var savedKeys = state.stepOrder && state.stepOrder[planOrderKey(level, target)];
    if (savedKeys && savedKeys.length > currentGaps.length) {
      savedKeys.forEach(function (k) {
        if (gapKeyMap[k]) {
          initial.push(gapKeyMap[k]);
        } else {
          var parts = k.split(":");
          var kind = parts[0];
          var itemKey = parts[1] || "";
          var rangeParts = (parts[2] || "").split("->");
          var fromLv = rangeParts[0] ? Number(rangeParts[0]) : null;
          var toLv = rangeParts[1] ? Number(rangeParts[1]) : (fromLv != null ? fromLv + 1 : null);
          if (kind === "level" && target.i && target.i[itemKey]) {
            var curLv = userLevel(itemKey);
            var f = fromLv != null ? fromLv : Math.max(0, curLv - 1);
            var t = toLv != null ? toLv : f + 1;
            initial.push({
              kind: "level",
              key: itemKey,
              from: f,
              to: t,
              cost: upgradeCostRange(itemKey, f, t),
              shards: isShards(itemKey),
              completed: curLv >= t
            });
          } else if (kind === "ring") {
            var rLv = hasRing() ? ringLevel() : 0;
            var rf = fromLv != null ? fromLv : Math.max(1, rLv - 1);
            var rt = toLv != null ? toLv : rf + 1;
            initial.push({
              kind: "ring",
              from: rf,
              to: rt,
              cost: ringCostRange(rf, rt),
              shards: true,
              completed: rLv >= rt
            });
          } else if (kind === "limbs") {
            var curLimbs = extraLimbs();
            var lf = fromLv != null ? fromLv : Math.max(0, curLimbs - 1);
            var lt = toLv != null ? toLv : lf + 1;
            initial.push({
              kind: "limbs",
              from: lf,
              to: lt,
              cost: extraLimbCost(lf, lt),
              shards: false,
              completed: curLimbs >= lt
            });
          } else if (kind === "contract") {
            var cZone = (itemCat(itemKey) && itemCat(itemKey).zone) || target.c;
            initial.push({
              kind: "contract",
              key: itemKey,
              zone: cZone,
              completed: userOwns(itemKey)
            });
          }
        }
      });
    } else {
      var completedCandidate = null;
      if (target.i) {
        var itemKeys = Object.keys(target.i);
        for (var i = 0; i < itemKeys.length; i++) {
          var ik = itemKeys[i];
          if (!gapKeyMap["level:" + ik] && userOwns(ik) && userLevel(ik) === target.i[ik] && target.i[ik] > 1) {
            completedCandidate = {
              kind: "level",
              key: ik,
              from: target.i[ik] - 1,
              to: target.i[ik],
              cost: upgradeCostAt(ik, target.i[ik] - 1),
              shards: isShards(ik),
              completed: true
            };
            break;
          }
        }
      }
      if (completedCandidate) {
        initial.push(completedCandidate);
      }
      currentGaps.forEach(function (g) {
        if (!initial.some(function (it) { return stepKey(it) === stepKey(g); })) {
          initial.push(g);
        }
      });
    }

    var ordered = applyStepOrder(level, target, expandSteps(initial));
    var evaluatedInitial = ordered.map(evalStep);
    var doneInit = evaluatedInitial.filter(function (s) { return s.completed; });
    var pendingInit = evaluatedInitial.filter(function (s) { return !s.completed; });
    evaluatedInitial = doneInit.concat(pendingInit);
    state.lockedPlans[String(level)] = { id: id, name: name, d: target.d || 0, steps: evaluatedInitial, customBuild: target };
    try { localStorage.setItem("saPlanner.lockedPlans", JSON.stringify(state.lockedPlans)); } catch (e) {}
    return evaluatedInitial;
  }

  function toggleLockBuild(buildOrId) {
    state.lastSimulatedStepsSig = null;
    var target = (buildOrId && typeof buildOrId === "object")
      ? buildOrId
      : findLockedTarget(state.level, { id: String(buildOrId), name: String(buildOrId) });
    if (target && isSameLockedBuild(state.level, target)) {
      clearLockedBuild(state.level);
      toast("Unlocked advisor for SA " + state.level);
    } else if (target) {
      setLockedBuild(state.level, target);
      toast("Locked advisor to " + target.n + (target.d ? " (" + prettify(target.d) + "/s)" : ""));
    } else {
      clearLockedBuild(state.level);
      toast("Unlocked advisor for SA " + state.level);
    }
    pushSimCache = {};
    levelSimCache = {};
    state._cachedDiag = null;
    var rec = bestRunnable(state.level);
    var rate = activeDustPs(rec);
    renderChips(rec, rate);
    renderRecommended(rec, rate);
    renderNext(rec, rate);
    renderBetter(rec, rate);
  }
  window.saToggleLockBuild = toggleLockBuild;

  function stepKey(st) {
    if (!st) return "";
    var f = st.from != null ? st.from : 0;
    var t = st.to != null ? st.to : f + 1;
    return st.kind + ":" + (st.key || "") + ":" + f + "->" + t;
  }

  function expandStep(s) {
    if (!s) return [];
    if (s.kind === "level") {
      var from = s.from != null ? s.from : 0;
      var to = s.to != null ? s.to : from + 1;
      if (to <= from + 1) return [s];
      var list = [];
      for (var lv = from; lv < to; lv++) {
        var isDone = userLevel(s.key) >= lv + 1 || (s.completed && lv + 1 <= to);
        list.push({
          kind: "level",
          key: s.key,
          from: lv,
          to: lv + 1,
          cost: upgradeCostAt(s.key, lv),
          shards: isShards(s.key),
          completed: isDone
        });
      }
      return list;
    }
    if (s.kind === "ring") {
      var fromR = s.from != null ? s.from : 1;
      var toR = s.to != null ? s.to : fromR + 1;
      if (toR <= fromR + 1) return [s];
      var listR = [];
      var curR = hasRing() ? ringLevel() : 0;
      for (var r = Math.max(1, fromR); r < toR; r++) {
        var isDoneR = curR >= r + 1 || (s.completed && r + 1 <= toR);
        listR.push({
          kind: "ring",
          from: r,
          to: r + 1,
          cost: ringLevelCost(r),
          shards: true,
          completed: isDoneR
        });
      }
      return listR;
    }
    if (s.kind === "limbs") {
      var fromL = s.from != null ? s.from : 0;
      var toL = s.to != null ? s.to : fromL + 1;
      if (toL <= fromL + 1) return [s];
      var listL = [];
      var curL = extraLimbs();
      for (var l = fromL; l < toL; l++) {
        var isDoneL = curL >= l + 1 || (s.completed && l + 1 <= toL);
        listL.push({
          kind: "limbs",
          from: l,
          to: l + 1,
          cost: extraLimbCost(l, l + 1),
          shards: false,
          completed: isDoneL
        });
      }
      return listL;
    }
    if (s.kind === "contract" && s.after && s.after.to > 1) {
      var listC = [];
      var owned = userOwns(s.key);
      listC.push({
        kind: "contract",
        key: s.key,
        zone: s.zone,
        active: s.active,
        reach: s.reach,
        cost: s.cost,
        shards: !!s.shards,
        completed: owned || !!s.completed
      });
      for (var alv = 1; alv < s.after.to; alv++) {
        var isDoneA = (owned && userLevel(s.key) >= alv + 1) || !!s.completed;
        listC.push({
          kind: "level",
          key: s.key,
          from: alv,
          to: alv + 1,
          cost: upgradeCostAt(s.key, alv),
          shards: isShards(s.key),
          completed: isDoneA
        });
      }
      return listC;
    }
    return [s];
  }

  function expandSteps(steps) {
    if (!steps || !steps.length) return [];
    var res = [];
    steps.forEach(function (s) {
      var expanded = expandStep(s);
      expanded.forEach(function (item) { res.push(item); });
    });
    return res;
  }

  function planOrderKey(level, build) {
    return String(level) + "::" + (build ? (buildId(build) || build.n || "") : "");
  }

  function getSavedStepOrder(level, build) {
    if (!state.stepOrder || !build) return null;
    var k = planOrderKey(level, build);
    if (state.stepOrder[k] && state.stepOrder[k].length) return state.stepOrder[k];
    var legacyK = String(level) + "::" + (build.n || "");
    if (state.stepOrder[legacyK] && state.stepOrder[legacyK].length) return state.stepOrder[legacyK];
    return null;
  }

  function hasCustomStepOrder(level, build) {
    return !!getSavedStepOrder(level, build);
  }

  function saveStepOrder(level, build, keys) {
    var k = planOrderKey(level, build);
    if (!state.stepOrder) state.stepOrder = {};
    state.stepOrder[k] = keys;
    try { localStorage.setItem("saPlanner.stepOrder", JSON.stringify(state.stepOrder)); } catch (e) {}
  }

  function resetStepOrder(level, build) {
    if (!state.stepOrder || !build) return;
    var k = planOrderKey(level, build);
    var legacyK = String(level) + "::" + (build.n || "");
    delete state.stepOrder[k];
    delete state.stepOrder[legacyK];
    try { localStorage.setItem("saPlanner.stepOrder", JSON.stringify(state.stepOrder)); } catch (e) {}
  }

  function sortSmartSteps(build, steps) {
    if (!steps || steps.length <= 1 || !state.ab) return steps || [];

    var bnk = bank();
    var rec = bestRunnable(state.level);
    var rate = activeDustPs(rec);
    var curSlots = maxSlots();
    var limbStep = steps.find(function (s) { return s.kind === "limbs"; });

    function isStepEquipped(s) {
      if (s.kind === "level") {
        var uItem = userItem(s.key);
        return !!(uItem && uItem.equipped);
      }
      if (s.kind === "ring") return hasRing();
      if (s.kind === "limbs") return true;
      return false;
    }

    function isStepInCurrentBuild(s) {
      if (s.kind === "level") {
        var inRec = !!(rec && rec.i && rec.i[s.key] != null);
        var uItem = userItem(s.key);
        var isEq = !!(uItem && uItem.equipped);
        return inRec || isEq;
      }
      if (s.kind === "ring") {
        return hasRing();
      }
      return false;
    }

    function stepWait(s) {
      if (s.completed) return 0;
      var c = stepFullCost(s);
      var isS = !!s.shards;
      var bD = bnk ? (bnk.dust || 0) : 0;
      var bS = bnk ? (bnk.shards || 0) : 0;
      if (isS) {
        var needS = Math.max(0, c - bS);
        return needS <= 0 ? 0 : (rate > 0 ? (needS * SHARD_DUST) / rate : (needS * SHARD_DUST));
      } else {
        var needD = Math.max(0, c - bD);
        return needD <= 0 ? 0 : (rate > 0 ? needD / rate : needD);
      }
    }

    var limbWait = limbStep ? stepWait(limbStep) : 0;

    function stepScore(s) {
      var w = stepWait(s);
      var inCur = isStepInCurrentBuild(s);
      var isEq = isStepEquipped(s);
      var needsLimb = false;

      if (limbStep && !inCur && (s.kind === "level" || s.kind === "contract" || s.kind === "missing")) {
        needsLimb = true;
      }

      var effWait = needsLimb ? (limbWait + w) : w;

      return {
        step: s,
        wait: w,
        effWait: effWait,
        inCurrentBuild: inCur,
        isEq: isEq,
        needsLimb: needsLimb,
        cost: stepFullCost(s)
      };
    }

    var scores = steps.map(stepScore);
    var scoreMap = new Map();
    scores.forEach(function (sc) { scoreMap.set(sc.step, sc); });

    return steps.slice().sort(function (a, b) {
      if (!!a.completed !== !!b.completed) return a.completed ? -1 : 1;

      var sa = scoreMap.get(a) || { effWait: 0, cost: 0, isEq: false, needsLimb: false };
      var sb = scoreMap.get(b) || { effWait: 0, cost: 0, isEq: false, needsLimb: false };

      if (a.kind === "contract" && b.kind === "level" && a.key === b.key) return -1;
      if (b.kind === "contract" && a.kind === "level" && a.key === b.key) return 1;

      if (a.kind === "ringUnlock" && b.kind === "ring") return -1;
      if (b.kind === "ringUnlock" && a.kind === "ring") return 1;

      if (a.kind === "limbs" && sb.needsLimb) return -1;
      if (b.kind === "limbs" && sa.needsLimb) return 1;

      if (a.key === b.key && a.kind === b.kind && (a.from != null && b.from != null)) {
        return a.from - b.from;
      }

      // Prioritize items in current build ahead of items only in the next build,
      // increasing the player's earning rate immediately while progressing
      if (sa.inCurrentBuild !== sb.inCurrentBuild) {
        return sa.inCurrentBuild ? -1 : 1;
      }

      if (Math.abs(sa.effWait - sb.effWait) > 1e-6) {
        return sa.effWait - sb.effWait;
      }

      if (sa.cost !== sb.cost) {
        return sa.cost - sb.cost;
      }

      var rank = { contract: 0, ringUnlock: 1, level: 2, ring: 3, limbs: 4, missing: 5 };
      var ra = rank.hasOwnProperty(a.kind) ? rank[a.kind] : 9;
      var rb = rank.hasOwnProperty(b.kind) ? rank[b.kind] : 9;
      return ra - rb;
    });
  }

  function applyStepOrder(level, build, steps) {
    if (!build || !steps || steps.length <= 1) return steps;
    var saved = getSavedStepOrder(level, build);
    if (!saved || !saved.length) return sortSmartSteps(build, steps);
    var rankMap = {};
    saved.forEach(function (id, idx) { rankMap[id] = idx; });
    return steps.slice().sort(function (a, b) {
      if (!!a.completed !== !!b.completed) return a.completed ? -1 : 1;
      var ka = stepKey(a);
      var kb = stepKey(b);
      var legacyKa = a.kind + ":" + (a.key || "");
      var legacyKb = b.kind + ":" + (b.key || "");
      var ia = rankMap.hasOwnProperty(ka) ? rankMap[ka] : (rankMap.hasOwnProperty(legacyKa) ? rankMap[legacyKa] : 999);
      var ib = rankMap.hasOwnProperty(kb) ? rankMap[kb] : (rankMap.hasOwnProperty(legacyKb) ? rankMap[legacyKb] : 999);
      if (ia !== ib) return ia - ib;
      if (a.key === b.key && a.kind === b.kind && (a.from != null && b.from != null)) {
        return a.from - b.from;
      }
      return 0;
    });
  }

  function fillLevels(selected) {
    var sel = $("levelSelect");
    sel.innerHTML = "";
    sheetLevels().forEach(function (lv) {
      var o = document.createElement("option");
      o.value = String(lv);
      var tag = levelTag(lv);
      o.textContent = "SA " + lv + " · " + tag;
      sel.appendChild(o);
    });
    if (selected && SA_DATA.builds[String(selected)]) sel.value = String(selected);
    else if (sel.options.length) sel.selectedIndex = 0;
    state.level = Number(sel.value);
    syncCustomSelect("levelSelect");
  }

  function stepLevel(delta) {
    var sel = $("levelSelect");
    if (!sel || !sel.options.length) return;
    var nextIdx = Math.max(0, Math.min(sel.options.length - 1, sel.selectedIndex + delta));
    if (nextIdx !== sel.selectedIndex) {
      sel.selectedIndex = nextIdx;
      syncCustomSelect("levelSelect");
      render();
    }
  }

  function selectLevel(lv) {
    state.lastSimulatedStepsSig = null;
    var sel = $("levelSelect");
    if (!sel) return;
    if (SA_DATA.builds[String(lv)]) {
      sel.value = String(lv);
      syncCustomSelect("levelSelect");
      render();
    }
  }

  function renderQuickLevels() {
    var el = $("quickLevels");
    if (!el || !state.ab) return;
    var fight = state.ab.enemyLevel;
    var max = state.ab.maxEnemyLevel;
    var html = "";
    if (fight && SA_DATA.builds[String(fight)]) {
      var fTag = isFarmLevel(fight) ? "Farm" : "Push";
      var isCur = state.level === fight;
      html += '<button type="button" class="btn sm ' + (isCur ? 'primary' : 'ghost') + '" onclick="window.saSelectLevel&&saSelectLevel(' + fight + ')" title="Fighting SA ' + fight + ' (' + fTag + ' Level)">Fighting (SA ' + fight + '<span class="ql-tag"> · ' + fTag + '</span>)</button>';
    }
    if (max && max !== fight && SA_DATA.builds[String(max)]) {
      var mTag = isFarmLevel(max) ? "Farm" : "Push";
      var isMax = state.level === max;
      html += '<button type="button" class="btn sm ' + (isMax ? 'primary' : 'ghost') + '" onclick="window.saSelectLevel&&saSelectLevel(' + max + ')" title="Highest cleared SA ' + max + ' (' + mTag + ' Level)">Max (SA ' + max + '<span class="ql-tag"> · ' + mTag + '</span>)</button>';
    }
    el.innerHTML = html;
  }

  function renderLevelModeToggle() {
    var el = $("levelModeToggle");
    if (!el) return;
    var mode = getLevelMode(state.level);
    var isFarmDef = getLevelDefaultMode(state.level) === "farm";
    var isOverridden = !!(state.levelModeOverrides && state.levelModeOverrides[state.level]);
    var isPushingCur = isWorkingToClear(state.level);
    var tipFarm = mode === "farm"
      ? (isOverridden ? "Farm Mode active (User override · held until refresh)" : "Farm Mode active (Default · Dust Priority)")
      : "Switch to Farm Mode (Prioritizes maximum dust rate for recommendations)";
    var tipPush = mode === "push"
      ? (isOverridden ? "Push Mode active (User override · held until refresh)" : ("Push Mode active (" + (isPushingCur ? "Active push level · Clear Priority" : "Clear Priority") + ")"))
      : "Switch to Push Mode (Prioritizes fastest clear time over dust rate for recommendations)";

    el.innerHTML =
      "<div class='mode-toggle-group' style='display:inline-flex;align-items:center;background:var(--surface-2);border:1px solid var(--border-strong);border-radius:10px;padding:2px;gap:2px;'>" +
      "<button type='button' class='mode-toggle-btn " + (mode === "farm" ? "mode-active-farm" : "") + "' onclick='window.saSetFarmMode&&saSetFarmMode()' title='" + esc(tipFarm) + "'>" +
      "🌾 Farm" +
      "</button>" +
      "<button type='button' class='mode-toggle-btn " + (mode === "push" ? "mode-active-push" : "") + "' onclick='window.saSetPushMode&&saSetPushMode()' title='" + esc(tipPush) + "'>" +
      "⚔️ Push" +
      "</button>" +
      "</div>";
  }

  function kpi(label, value, note, tone, icon, extraAttr) {
    return '<div class="kpi ' + (tone || 'tone-accent') + '"' + (extraAttr ? ' ' + extraAttr : '') + '>' +
      '<div class="kpi-label">' + (icon || '') + '<span>' + esc(label) + '</span></div>' +
      '<div class="kpi-value">' + esc(String(value)) + '</div>' +
      (note ? '<div class="kpi-note">' + esc(note) + '</div>' : '') +
      '</div>';
  }

  function fmtLocalEta(secFromNow) {
    if (!isFinite(secFromNow) || secFromNow == null) return "--";
    if (secFromNow <= 0) return "Ready now";
    var now = new Date();
    var eta = new Date(now.getTime() + secFromNow * 1000);
    var isSameDay = eta.getFullYear() === now.getFullYear() &&
                    eta.getMonth() === now.getMonth() &&
                    eta.getDate() === now.getDate();
    var tomorrow = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1);
    var isTomorrow = eta.getFullYear() === tomorrow.getFullYear() &&
                     eta.getMonth() === tomorrow.getMonth() &&
                     eta.getDate() === tomorrow.getDate();
    var pad = function (n) { return (n < 10 ? "0" : "") + n; };
    var timeStr = pad(eta.getHours()) + ":" + pad(eta.getMinutes());
    if (isSameDay) return timeStr;
    if (isTomorrow) return "Tomorrow " + timeStr;
    if (secFromNow < 6 * 86400) {
      var dayName = eta.toLocaleDateString([], { weekday: "short" });
      return dayName + " " + timeStr;
    }
    return eta.toLocaleDateString([], { month: "numeric", day: "numeric" }) + " " + timeStr;
  }

  function nextActionNote(s) {
    if (!s) return "";
    if (s.kind === "contract") return "Z" + s.zone + " " + itemName(s.key);
    if (s.kind === "level") return itemName(s.key) + " " + s.from + " → " + s.to;
    if (s.kind === "limbs") return "Extra Limbs " + s.from + " → " + s.to;
    if (s.kind === "ringUnlock") return "Unlock The Ring";
    if (s.kind === "ring") return "Ring " + s.from + " → " + s.to;
    return stepLabel(s);
  }

  function fitKpiValue(el) {
    if (!el || el.clientWidth === 0) return;
    el.style.fontSize = "";
    var avail = el.clientWidth - 2;
    if (avail > 0 && el.scrollWidth > avail) {
      var computed = window.getComputedStyle ? parseFloat(window.getComputedStyle(el).fontSize) : 18;
      if (!computed || isNaN(computed)) computed = 18;
      var ratio = avail / el.scrollWidth;
      var target = Math.max(9, Math.floor(computed * ratio * 10) / 10);
      el.style.fontSize = target + "px";
      var guard = 15;
      while (el.scrollWidth > avail && target > 8.5 && guard-- > 0) {
        target -= 0.5;
        el.style.fontSize = target + "px";
      }
    }
  }

  function renderNextActionKpi(pick, rate, bnk) {
    if (pick.mode === "completed") {
      var titleComp = "All steps in active plan (" + esc(pick.build.n) + ") are fully unlocked!";
      return '<div class="kpi tone-buy" id="kpiNextAction" title="' + esc(titleComp) + '">' +
        '<div class="kpi-label">' + ICONS.check + '<span>Next Action</span></div>' +
        '<div class="kpi-value val-sm" id="kpiNextActionVal" style="color:var(--buy);">Plan Completed</div>' +
        '<div class="kpi-note kpi-note-prog">' +
          '<div class="kpi-prog-row">' +
            '<span class="kpi-prog-action">' + esc(pick.build.n) + '</span>' +
            '<span class="kpi-prog-pct" style="color:var(--buy);">100%</span>' +
          '</div>' +
          '<div class="kpi-bar"><span style="width:100%; background:var(--buy);"></span></div>' +
        '</div>' +
        '</div>';
    }
    var s = pick.step;
    if (!s) return "";
    var actNote = nextActionNote(s);
    var rawCost = s.cost || 0;
    var isSh = !!s.shards;
    var curFunds = isSh ? (bnk.shards || 0) : (bnk.dust || 0);
    var wait = timeForCost(rawCost, isSh, rate, bnk.dust, bnk.shards);
    var isReady = (wait === 0) || (rawCost <= 0) || (curFunds >= rawCost);

    var pct = rawCost > 0 ? Math.min(100, Math.max(0, (curFunds / rawCost) * 100)) : 100;
    var pctTxt = (isReady || pct >= 99.95) ? "100%" : (pct.toFixed(pct >= 10 ? 0 : 1) + "%");

    if (isReady) {
      var titleReady = "Next Action Ready!\nAction: " + actNote +
        (rawCost > 0 ? ("\nCost: " + prettify(rawCost) + (isSh ? " shards" : " dust")) : "") +
        "\nAffordable now with your bank!";
      return '<div class="kpi tone-buy" id="kpiNextAction" title="' + esc(titleReady) + '">' +
        '<div class="kpi-label">' + ICONS.check + '<span>Next Action</span></div>' +
        '<div class="kpi-value val-sm" id="kpiNextActionVal" style="color:var(--buy);">Next Action Ready</div>' +
        '<div class="kpi-note kpi-note-prog">' +
          '<div class="kpi-prog-row">' +
            '<span class="kpi-prog-action" title="' + esc(actNote) + '">' + esc(actNote) + '</span>' +
            '<span class="kpi-prog-pct" id="kpiNextActionPct" style="color:var(--buy);">100%</span>' +
          '</div>' +
          '<div class="kpi-bar"><span id="kpiNextActionBar" style="width:100%; background:var(--buy);"></span></div>' +
        '</div>' +
        '</div>';
    }

    var etaStr = fmtLocalEta(wait);
    var tone = isSh ? "tone-teal" : "tone-accent";
    var barBg = isSh ? "var(--teal)" : "var(--accent)";
    var titleWait = "Next Action: " + actNote +
      "\nCost: " + prettify(rawCost) + (isSh ? " shards" : " dust") +
      "\nCurrent bank: " + prettify(curFunds) + (isSh ? " shards" : " dust") + " (" + pctTxt + ")" +
      "\nAvailable at: " + etaStr + " (" + fmtTime(wait) + " at " + prettify(rate) + "/s)";

    return '<div class="kpi ' + tone + '" id="kpiNextAction" title="' + esc(titleWait) + '">' +
      '<div class="kpi-label">' + ICONS.clock + '<span>Next Action</span></div>' +
      '<div class="kpi-value" id="kpiNextActionVal">' + esc(etaStr) + '</div>' +
      '<div class="kpi-note kpi-note-prog">' +
        '<div class="kpi-prog-row">' +
          '<span class="kpi-prog-action" title="' + esc(actNote) + '">' + esc(actNote) + '</span>' +
          '<span class="kpi-prog-pct" id="kpiNextActionPct">' + esc(pctTxt) + '</span>' +
        '</div>' +
        '<div class="kpi-bar"><span id="kpiNextActionBar" style="width:' + pct.toFixed(1) + '%; background:' + barBg + ';"></span></div>' +
      '</div>' +
      '</div>';
  }

  function renderChips(rec, rate) {
    var ab = state.ab;
    var bnk = bank();
    var owned = ownedItemKeys();
    var equipped = owned.filter(function (k) { return userItem(k) && userItem(k).equipped; }).length;
    var mods = ringMods().map(function (m) { return RING_MOD_NAMES[m] || m; }).join(" / ");

    var dustVal = prettify(bnk.dust);
    var dustNote = rate ? ("Active rate: " + prettify(rate) + "/s") : "No active rate";
    if (bnk.dustEarned > 0) {
      dustNote = "Base " + prettify(bnk.rawDust) + " · +" + prettify(bnk.dustEarned) + " earned (" + fmtTime(bnk.elapsedSec) + ")";
    }

    var shardVal = prettify(bnk.shards);
    var shardNote = "";
    if (bnk.shardsEarned > 0) {
      shardNote = "Base " + prettify(bnk.rawShards) + " · +" + prettify(bnk.shardsEarned) + " earned";
    } else if (canEarnShards() && rate > 0) {
      var curShardDust = ((ab.shardDust || 0) + bnk.dustEarned) % SHARD_DUST;
      var pct = Math.min(99.9, (curShardDust / SHARD_DUST) * 100).toFixed(1);
      shardNote = "Next: " + pct + "%";
    }

    var dustTitle = "Live estimated dust bank\nSnapshot from save: " + prettify(bnk.rawDust) +
      (bnk.dustEarned > 0 ? ("\nEarned since load: +" + prettify(bnk.dustEarned) + " (" + fmtTime(bnk.elapsedSec) + " ago)") : "") +
      (rate ? ("\nActive earning rate: " + prettify(rate) + "/s") : "");

    var shardTitle = "Live estimated shard bank\nSnapshot from save: " + prettify(bnk.rawShards) + " shards" +
      (bnk.shardsEarned > 0 ? ("\nEarned since load: +" + bnk.shardsEarned) : "") +
      "\nDust equivalent: " + prettify((bnk.shards || 0) * SHARD_DUST) + " dust";

    // Item 4: Kill & Death Diagnostics + Level Mode
    var diag = combatDiagnostics(rate);
    var audit = saSettingsAudit();
    var autoLevelPill = '';
    if (state.save) {
      autoLevelPill = !audit.autoLevel
        ? '<span class="pill sm warn-pill" style="margin-left:auto; font-size:10px; padding:1px 6px; text-transform:none; letter-spacing:normal;" title="Trimps setting \'AutoLevel\' is OFF. Huffy will NOT automatically advance to the next level when current kill requirement is met.">' + ICONS.warn + 'AutoLevel OFF</span>'
        : '<span class="pill sm buy" style="margin-left:auto; font-size:10px; padding:1px 6px; text-transform:none; letter-spacing:normal;" title="Trimps setting \'AutoLevel\' is ON (Auto-advances when cleared)">' + ICONS.check + 'AutoLevel ON</span>';
    }
    var enemyVal = 'SA ' + (diag.enemyLevel || '—');
    var enemyNote = '';
    var enemyTone = 'tone-accent';
    var enemyKpiHtml = '';

    if (diag.isPushing) {
      var killsEarned = (diag.secPerKill && diag.secPerKill > 0 && bnk.elapsedSec > 0)
        ? Math.min(diag.remainingKills, Math.floor(bnk.elapsedSec / diag.secPerKill))
        : 0;
      var curRemaining = Math.max(0, diag.remainingKills - killsEarned);
      var curKills = diag.curKills + killsEarned;
      var curPct = diag.neededKills > 0 ? Math.min(100, (curKills / diag.neededKills) * 100) : 0;
      var curEtaSec = Math.max(0, (diag.etaSec || 0) - bnk.elapsedSec);

      var etaDuration = curEtaSec > 0 ? ("~" + fmtDuration(curEtaSec)) : "Ready to clear";
      var etaClock = curEtaSec > 0 ? fmtLocalEta(curEtaSec) : "";
      var etaCombined = etaDuration + (etaClock ? " (ETA " + etaClock + ")" : "");

      var enemyLvlTag = diag.enemyLevel ? (isFarmLevel(diag.enemyLevel) ? " (Farm Level)" : " (Push Level)") : "";
      var enemyTitle = "Enemy Level: SA " + diag.enemyLevel + enemyLvlTag +
        "\nStatus: " + diag.modeLabel +
        "\n• Progress: " + prettify(curKills) + " / " + prettify(diag.neededKills) + " kills (" + curPct.toFixed(1) + "%)" +
        (killsEarned > 0 ? ("\n• Estimated Kills Since Load: +" + prettify(killsEarned) + " (" + fmtTime(bnk.elapsedSec) + " ago)") : "") +
        "\n• Remaining: " + prettify(curRemaining) + " kills to clear SA " + diag.enemyLevel +
        (diag.secPerKill && isFinite(diag.secPerKill) ? ("\n• Speed: ~" + diag.secPerKill.toFixed(1) + "s per kill" + (diag.buildName ? " (" + diag.buildName + ")" : " (Simulated)")) : "") +
        "\n• Estimated Completion: " + etaCombined +
        "\n• AutoLevel: " + (diag.autoLevel ? "ON (Auto-advances when cleared)" : "OFF (Will NOT advance when cleared!)") +
        "\n\nCombat Statistics:" +
        (diag.sessionTotal > 0
          ? ("\n• Session Win Rate: " + diag.winRateStr + " (" + prettify(diag.sessionKills) + "W / " + prettify(diag.sessionDeaths) + "L in " + prettify(diag.sessionTotal) + " fights)")
          : "") +
        "\n• Total Enemies Slain: " + prettify(diag.totalKillsAllTime) + " (All-Time)" +
        (diag.totalKillsPortal > 0 ? ("\n• Kills This Portal: " + prettify(diag.totalKillsPortal)) : "") +
        "\n\nClick to inspect Presets, Settings & Diagnostics";

      enemyKpiHtml = '<div class="kpi tone-accent" id="kpiEnemyLevel" style="cursor:pointer;" onclick="window.saOpenPresets&&saOpenPresets()" title="' + esc(enemyTitle) + '">' +
        '<div class="kpi-label">' + ICONS.spire + '<span>Enemy Level</span>' + autoLevelPill + '</div>' +
        '<div class="kpi-value">' + esc(enemyVal) + '</div>' +
        '<div class="kpi-note kpi-note-prog">' +
          '<div class="kpi-prog-row">' +
            '<span class="kpi-prog-txt">Pushing · ' + prettify(curRemaining) + ' to go (' + curPct.toFixed(0) + '%)</span>' +
            '<span class="kpi-prog-eta mono" style="color:var(--accent); font-weight:650;" title="Estimated completion time' + (etaClock ? ' (' + etaClock + ')' : '') + '">' + esc(etaDuration) + '</span>' +
          '</div>' +
          '<div class="kpi-bar"><span style="width:' + Math.max(2, curPct).toFixed(1) + '%; background:var(--accent);"></span></div>' +
        '</div>' +
      '</div>';
    } else {
      if (diag.isFarming) {
        enemyNote = 'Farming · Max SA ' + diag.maxEnemyLevel;
        enemyTone = 'tone-info';
      } else if (diag.isMax) {
        enemyNote = 'Max Cleared · SA ' + diag.maxEnemyLevel;
        enemyTone = 'tone-buy';
      } else if (diag.isPractice) {
        enemyNote = 'Practice Mode Active';
        enemyTone = 'tone-sell';
      } else {
        enemyNote = 'Max unlocked: SA ' + (diag.maxEnemyLevel || '—');
      }

      var enemyTitleNonPush = 'Enemy Level: SA ' + diag.enemyLevel +
        '\nStatus: ' + diag.modeLabel +
        '\n• Highest Unlocked: SA ' + diag.maxEnemyLevel +
        '\n• AutoLevel: ' + (diag.autoLevel ? 'ON (Auto-advances when cleared)' : 'OFF (Will NOT advance when cleared!)') +
        '\n\nCombat Statistics:' +
        (diag.sessionTotal > 0
          ? ('\n• Session Win Rate: ' + diag.winRateStr + ' (' + prettify(diag.sessionKills) + 'W / ' + prettify(diag.sessionDeaths) + 'L in ' + prettify(diag.sessionTotal) + ' fights)')
          : '') +
        '\n• Total Enemies Slain: ' + prettify(diag.totalKillsAllTime) + ' (All-Time)' +
        (diag.totalKillsPortal > 0 ? ('\n• Kills This Portal: ' + prettify(diag.totalKillsPortal)) : '') +
        '\n\nClick to inspect Presets, Settings & Diagnostics';

      enemyKpiHtml = '<div class="kpi ' + enemyTone + '" id="kpiEnemyLevel" style="cursor:pointer;" onclick="window.saOpenPresets&&saOpenPresets()" title="' + esc(enemyTitleNonPush) + '">' +
        '<div class="kpi-label">' + ICONS.spire + '<span>Enemy Level</span>' + autoLevelPill + '</div>' +
        '<div class="kpi-value">' + esc(enemyVal) + '</div>' +
        (enemyNote ? '<div class="kpi-note">' + esc(enemyNote) + '</div>' : '') +
      '</div>';
    }

    // Item 2: One-Timer Milestones
    var msInfo = nextMilestoneInfo(owned.length);
    var limbVal = maxSlots() + " slots";
    var limbNote = equipped + " eq · " + owned.length + "/" + SA_DATA.itemOrder.length + " owned" +
      (msInfo.next ? (" · Next: " + msInfo.next.name) : "");
    var limbTitle = "Huffy Limbs & Milestone Progress" +
      "\nEquipped items: " + equipped + " / " + maxSlots() + " slots" +
      "\nTotal items owned: " + owned.length + " / " + SA_DATA.itemOrder.length +
      (msInfo.next
        ? ("\n\nNext Milestone: " + msInfo.next.name + " (" + msInfo.next.req + " items · " + msInfo.next.toGo + " to go)" +
           "\nBonus: " + msInfo.next.desc)
        : "\n\nAll item milestones unlocked!") +
      "\n\nClick to inspect all Presets & Milestones";

    // Item 3: The Ring Base Stats & Slots
    var rLvl = ringLevel();
    var rUnlocked = hasRing();
    var rStats = ringBaseStats(rLvl);
    var ringVal = rUnlocked ? ("Lv " + rLvl) : "Locked";
    var ringNote = rUnlocked
      ? ((rStats.nextSlotLvl ? ("Next slot Lv " + rStats.nextSlotLvl + " · ") : "") + (mods || (rStats.slots > 0 ? "No mods set" : "0/3 slots")))
      : ("Unlock: " + prettify(oneTimerRingCost()) + " shards");
    var ringTitle = rUnlocked
      ? ("The Ring · Level " + rLvl +
         "\n\nInherent Ring Bonuses:" +
         "\n• +" + prettify(rStats.statusDmg) + "% Bleed & Shock Damage" +
         "\n• +" + prettify(rStats.poisonDmg) + " Poison Damage" +
         (rStats.statusChance > 0 ? ("\n• +" + prettify(rStats.statusChance) + "% Status Chance") : "\n• Status Chance unlocks at Lv 10") +
         "\n\nCustomizable Slots: " + rStats.slots + " / 3 unlocked" +
         (rStats.nextSlotLvl
           ? ("\n• Next slot at Level " + rStats.nextSlotLvl + " (" + rStats.levelsToNext + " levels to go)")
           : "\n• All 3 customizable slots unlocked") +
         (mods ? ("\n\nEquipped Mods:\n• " + mods.split(" / ").join("\n• ")) : "") +
         "\n\nClick to inspect Presets & Diagnostics")
      : ("The Ring (Locked)" +
         "\nUnlock Cost: " + prettify(oneTimerRingCost()) + " shards" +
         "\nRequires 42 items owned to unlock in Spire Assault.");

    var pick = bestNextUpgrade(rec, rate);
    state._cachedRec = rec;
    state._cachedDiag = diag;
    state._cachedPick = pick;
    var lastKpiHtml = (pick && pick.isLocked)
      ? renderNextActionKpi(pick, rate, bnk)
      : kpi("U2 Progress", "Z" + (state.radon + 1), ab.activeContract ? ("Active: " + itemName(ab.activeContract)) : "No active contract", "tone-violet", ICONS.contract, "id='kpiU2Progress'");

    $("chips").innerHTML = [
      enemyKpiHtml,
      kpi("Dust Bank", dustVal, dustNote, "tone-buy", ICONS.dust, "id='kpiDustBank' title='" + esc(dustTitle) + "'"),
      kpi("Shard Bank", shardVal, shardNote, "tone-teal", ICONS.gem, "id='kpiShardBank' title='" + esc(shardTitle) + "'"),
      kpi("Huffy Limbs", limbVal, limbNote, "tone-warn", ICONS.limb, "style='cursor:pointer;' onclick='window.saOpenPresets&&saOpenPresets()' title='" + esc(limbTitle) + "'"),
      kpi("The Ring", ringVal, ringNote, "tone-info", ICONS.ring, "style='cursor:pointer;' onclick='window.saOpenPresets&&saOpenPresets()' title='" + esc(ringTitle) + "'"),
      lastKpiHtml
    ].join("");
    fitKpiValue($("kpiNextActionVal"));
  }

  function esc(s) {
    return String(s).replace(/[&<>"']/g, function (c) {
      return ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c];
    });
  }

  function stepLabel(s) {
    if (s.kind === "contract") {
      var extra = s.active ? " (accepted, not finished)" : (!s.reach ? " (void map not reachable yet)" : "");
      return "Complete Z" + s.zone + " contract to unlock " + itemName(s.key) + extra;
    }
    if (s.kind === "level") return itemName(s.key) + " Lv " + s.from + " → " + s.to;
    if (s.kind === "limbs") return "Extra Limbs " + s.from + " → " + s.to;
    if (s.kind === "ringUnlock") return "Unlock The Ring (one-timer)";
    if (s.kind === "ring") return "Ring Lv " + s.from + " → " + s.to;
    if (s.kind === "missing") return "Missing " + itemName(s.key);
    return "";
  }

  function stepClass(s) {
    if (s.kind === "contract" || s.kind === "missing") return "step-contract";
    if (s.kind === "limbs") return "step-limb";
    if (s.kind === "ring" || s.kind === "ringUnlock") return "step-ring";
    if (s.kind === "level") return "step-item";
    return "";
  }

  function stepBadgeHtml(s) {
    if (s.kind === "level") {
      return "<span class='step-badge " + stepClass(s) + "'>" + iconForStep(s) +
        "<span class='step-text'><span class='step-name'>" + esc(itemName(s.key)) + "</span><span class='step-sub'>Lv " + s.from + " → " + s.to + "</span></span></span>";
    }
    if (s.kind === "ring") {
      return "<span class='step-badge " + stepClass(s) + "'>" + iconForStep(s) +
        "<span class='step-text'><span class='step-name'>The Ring</span><span class='step-sub'>Lv " + s.from + " → " + s.to + "</span></span></span>";
    }
    if (s.kind === "ringUnlock") {
      return "<span class='step-badge " + stepClass(s) + "'>" + iconForStep(s) +
        "<span class='step-text'><span class='step-name'>The Ring</span><span class='step-sub'>Unlock</span></span></span>";
    }
    if (s.kind === "limbs") {
      return "<span class='step-badge " + stepClass(s) + "'>" + iconForStep(s) +
        "<span class='step-text'><span class='step-name'>Extra Limbs</span><span class='step-sub'>" + s.from + " → " + s.to + "</span></span></span>";
    }
    if (s.kind === "contract") {
      var cSub = "Z" + s.zone + (s.active ? " (active)" : (!s.reach ? " (unreached)" : ""));
      return "<span class='step-badge " + stepClass(s) + "'>" + iconForStep(s) +
        "<span class='step-text'><span class='step-name'>" + esc(itemName(s.key)) + "</span><span class='step-sub'>" + esc(cSub) + "</span></span></span>";
    }
    if (s.kind === "missing") {
      return "<span class='step-badge " + stepClass(s) + "'>" + iconForStep(s) +
        "<span class='step-text'><span class='step-name'>" + esc(itemName(s.key)) + "</span><span class='step-sub'>Missing</span></span></span>";
    }
    return "<span class='step-badge " + stepClass(s) + "'>" + iconForStep(s) + "<span>" + esc(stepLabel(s)) + "</span></span>";
  }

  function atomicStep(s) {
    if (s.kind === "level" && s.to > s.from + 1) {
      return { kind: "level", key: s.key, from: s.from, to: s.from + 1, cost: upgradeCostAt(s.key, s.from), shards: s.shards };
    }
    if (s.kind === "limbs" && s.to > s.from + 1) {
      return { kind: "limbs", from: s.from, to: s.from + 1, cost: extraLimbCost(s.from, s.from + 1), shards: false };
    }
    if (s.kind === "ring" && s.to > s.from + 1) {
      return { kind: "ring", from: s.from, to: s.from + 1, cost: ringLevelCost(s.from), shards: true };
    }
    return s;
  }

  function bestAfter(step) {
    if (!state.ab || !step) return bestRunnable(state.level);
    var items = state.ab.items || (state.ab.items = {});
    var bonuses = state.ab.bonuses || (state.ab.bonuses = {});
    var oneTimers = state.ab.oneTimers || (state.ab.oneTimers = {});
    var rings = state.ab.rings || (state.ab.rings = { level: 1, mods: [] });

    var hadItem = false, prevItem = null, prevOwned = false, prevLv = 1;
    var prevExtraLimbs = bonuses.Extra_Limbs;
    var prevTheRing = oneTimers.The_Ring;
    var prevRingLv = rings.level;

    if (step.kind === "contract" || step.kind === "missing" || step.kind === "level") {
      hadItem = Object.prototype.hasOwnProperty.call(items, step.key);
      prevItem = items[step.key];
      if (prevItem && typeof prevItem === "object") {
        prevOwned = prevItem.owned;
        prevLv = prevItem.level;
      }
    }

    try {
      if (step.kind === "contract" || step.kind === "missing") {
        items[step.key] = { owned: true, level: 1, equipped: false };
      } else if (step.kind === "level") {
        if (!items[step.key] || typeof items[step.key] !== "object") {
          items[step.key] = { owned: true, level: 2 };
        } else {
          items[step.key].owned = true;
          items[step.key].level = (items[step.key].level || 1) + 1;
        }
      } else if (step.kind === "limbs") {
        bonuses.Extra_Limbs = extraLimbs() + 1;
      } else if (step.kind === "ringUnlock") {
        oneTimers.The_Ring = true;
        if (!rings.level) rings.level = 1;
      } else if (step.kind === "ring") {
        rings.level = (rings.level || 1) + 1;
      }
      return bestRunnable(state.level);
    } finally {
      if (step.kind === "contract" || step.kind === "missing" || step.kind === "level") {
        if (!hadItem) {
          delete items[step.key];
        } else {
          items[step.key] = prevItem;
          if (prevItem && typeof prevItem === "object") {
            prevItem.owned = prevOwned;
            prevItem.level = prevLv;
          }
        }
      } else if (step.kind === "limbs") {
        bonuses.Extra_Limbs = prevExtraLimbs;
      } else if (step.kind === "ringUnlock") {
        oneTimers.The_Ring = prevTheRing;
        rings.level = prevRingLv;
      } else if (step.kind === "ring") {
        rings.level = prevRingLv;
      }
    }
  }

  function betterBuilds(rec, cmpRateOverride) {
    var mode = getLevelMode(state.level);
    if (mode === "push") {
      var curCombatK = getCurrentCombatSecPerKill(state.level);
      var recK = rec ? buildSecPerKill(rec, state.level) : Infinity;
      var curK = (curCombatK && curCombatK > 0) ? Math.min(curCombatK, recK) : recK;
      return buildsFor(state.level).filter(function (b) {
        var bk = buildSecPerKill(b, state.level);
        return (!rec || bk < curK - 0.01) && !runnable(b) && allowedUpgrade(b);
      });
    }
    var cur = (typeof cmpRateOverride === "number" && cmpRateOverride >= 0)
      ? cmpRateOverride
      : Math.max(levelCurrentDustPs(state.level, rec), rec ? (rec.d || 0) : 0);
    return buildsFor(state.level).filter(function (b) {
      return (b.d || 0) > cur + 1e-9 && !runnable(b) && allowedUpgrade(b);
    });
  }

  function bestNextUpgrade(rec, rateOverride, cmpRateOverride) {
    var earnRate = (typeof rateOverride === "number" && rateOverride > 0)
      ? rateOverride
      : (activeDustPs(rec) || 1);
    var cur = (typeof cmpRateOverride === "number" && cmpRateOverride >= 0)
      ? cmpRateOverride
      : levelCurrentDustPs(state.level, rec);
    var cmpCur = Math.max(cur, rec ? (rec.d || 0) : 0);
    var bnk = bank();
    var lockedEntry = state.lockedPlans && state.lockedPlans[String(state.level)];
    if (lockedEntry) {
      var target = findLockedTarget(state.level, lockedEntry);
      if (target) {
        var planSteps = syncLockedPlan(state.level, target);
        if (!planSteps || !planSteps.length) {
          planSteps = applyStepOrder(state.level, target, expandSteps(gapSteps(target)));
        }
        var doneSteps = planSteps.filter(function (s) { return s.completed; });
        var pendingSteps = planSteps.filter(function (s) { return !s.completed; });
        planSteps = doneSteps.concat(pendingSteps);
        var completedCount = doneSteps.length;
        var totalSteps = planSteps.length;

        if (pendingSteps.length === 0 || runnable(target)) {
          return {
            mode: "completed",
            build: target,
            isLocked: true,
            remaining: 0,
            totalSteps: totalSteps,
            completedCount: completedCount,
            allSteps: planSteps,
            gain: (target.d || 0) - cmpCur
          };
        }
        var firstPending = null;
        var activeIdxInPlan = doneSteps.length;
        for (var tj = 0; tj < planSteps.length; tj++) {
          if (planSteps[tj].completed) continue;
          if (planSteps[tj].kind === "contract" && !planSteps[tj].reach) continue;
          firstPending = planSteps[tj];
          activeIdxInPlan = tj;
          break;
        }
        if (!firstPending) {
          for (var tj = 0; tj < planSteps.length; tj++) {
            if (!planSteps[tj].completed) {
              firstPending = planSteps[tj];
              activeIdxInPlan = tj;
              break;
            }
          }
        }
        if (firstPending) {
          var targetFirst = atomicStep(firstPending);
          return {
            step: targetFirst,
            gain: (target.d || 0) - cmpCur,
            cost: stepsCostDust([targetFirst]).combined,
            build: target,
            mode: "locked",
            remaining: pendingSteps.length,
            totalSteps: totalSteps,
            completedCount: completedCount,
            allSteps: planSteps,
            activeIndex: activeIdxInPlan,
            isLocked: true
          };
        }
      }
    }
    var list = betterBuilds(rec, cmpCur);
    var mode = getLevelMode(state.level);
    var curCombatK = getCurrentCombatSecPerKill(state.level);
    var recK = rec ? buildSecPerKill(rec, state.level) : Infinity;
    var curK = (mode === "push" && curCombatK && curCombatK > 0)
      ? Math.min(curCombatK, recK)
      : recK;
    var seen = {};
    var immediate = [];
    list.forEach(function (b) {
      gapSteps(b).forEach(function (raw) {
        if (raw.kind === "contract" && !raw.reach) return;
        var s = atomicStep(raw);
        var id = s.kind + ":" + (s.key || "") + ":" + (s.from || 0);
        if (seen[id]) return;
        seen[id] = true;
        var unlocked = bestAfter(s);
        var gain = (mode === "push")
          ? Math.max(0, curK - (unlocked ? buildSecPerKill(unlocked, state.level) : Infinity))
          : ((unlocked ? unlocked.d || 0 : 0) - cmpCur);
        var wInfo = stepsWaitInfo([s], 1, bnk);
        var effCost = wInfo.remDustEq > 0 ? wInfo.remDustEq : Math.max(1, wInfo.combined * 1e-6);
        if (gain > 0 && unlocked) immediate.push({ step: s, gain: gain, cost: wInfo.combined, roi: gain / effCost, build: unlocked, mode: "unlocks", remaining: 1, allSteps: [s], activeIndex: 0 });
      });
    });
    if (immediate.length) {
      immediate.sort(function (a, b) { return b.roi - a.roi; });
      return immediate[0];
    }
    var paths = list.map(function (b) {
      var steps = applyStepOrder(state.level, b, expandSteps(gapSteps(b)));
      var wInfo = calculateStepWaitTimes(steps, bnk, earnRate);
      var effCost = Math.max(wInfo.remDust, wInfo.remShards * SHARD_DUST);
      if (effCost <= 0) effCost = Math.max(1, (wInfo.totalCostDust + wInfo.totalCostShards * SHARD_DUST) * 1e-6);
      var blocked = steps.some(function (s) { return s.kind === "contract" && !s.reach; });
      var pathGain = (mode === "push")
        ? Math.max(0, curK - buildSecPerKill(b, state.level))
        : ((b.d || 0) - cmpCur);
      return { build: b, steps: steps, cost: wInfo.combined, remaining: steps.length, blocked: blocked, roi: pathGain / effCost };
    });
    paths.sort(function (a, b) { return (a.blocked - b.blocked) || a.remaining - b.remaining || b.roi - a.roi; });
    var p = null;
    var first = null;
    var activeIdx = 0;
    for (var i = 0; i < paths.length; i++) {
      for (var j = 0; j < paths[i].steps.length; j++) {
        if (paths[i].steps[j].kind === "contract" && !paths[i].steps[j].reach) continue;
        first = atomicStep(paths[i].steps[j]);
        p = paths[i];
        activeIdx = j;
        break;
      }
      if (first) break;
    }

    // If no better build exists in push or farm mode without a locked plan, use simulator Upgrade Value
    if (!lockedEntry) {
      var shouldFallbackROI = false;
      if (mode === "push") {
        var hasFasterPath = (immediate.length > 0) || (paths.some(function(item) { return !item.blocked && item.roi > 0; }));
        var isCurWinning = (curCombatK && isFinite(curCombatK) && curCombatK > 0);
        var isCurOptimal = isCurWinning && (curCombatK <= recK + 0.01);
        if (isCurWinning && (!hasFasterPath || isCurOptimal)) {
          shouldFallbackROI = true;
        } else if (!hasFasterPath && (!p || !first) && isCurWinning) {
          shouldFallbackROI = true;
        }
      } else if (mode === "farm") {
        var isWinningFarm = (cur > 0) || (curCombatK && isFinite(curCombatK) && curCombatK > 0);
        var hasBetterFarmPath = (immediate.length > 0) || (paths.some(function(item) { return !item.blocked && item.roi > 0; }));
        if (isWinningFarm && (!hasBetterFarmPath || !p || !first)) {
          shouldFallbackROI = true;
        }
      }
      if (shouldFallbackROI) {
        if (typeof SASimEngine !== "undefined" && state.save) {
          try {
            var cfgSrc = (simState.config && simState.config.items && Number(simState.config.level) === Number(state.level))
              ? simState.config
              : { save: state.save, level: state.level };
            var roiResults = SASimEngine.evaluateUpgradeROI(cfgSrc, { maxKills: 20 });
            if (roiResults && roiResults.length > 0) {
              var top = roiResults[0];
              var uStep = (top.type === "ring")
                ? { kind: "ring", from: top.currentLevel, to: top.nextLevel, cost: top.cost, shards: top.shards }
                : { kind: "level", key: top.key, from: top.currentLevel, to: top.nextLevel, cost: top.cost, shards: top.shards };
              var curBld = getCurrentEquippedCombatBuild(state.level) || { n: "Current Loadout" };
              var newK = (top.newSecPerKill && top.newSecPerKill > 0) ? top.newSecPerKill : (curCombatK || 15);
              var cloneBld = Object.assign({}, curBld, { k: newK });
              return {
                step: uStep,
                mode: "upgradeValue",
                isLocked: false,
                isOptimal: true,
                remaining: 1,
                totalSteps: 1,
                completedCount: 0,
                allSteps: [uStep],
                activeIndex: 0,
                gain: top.deltaDustPs,
                cost: top.cost,
                shards: top.shards,
                label: top.label,
                paybackSec: top.paybackSec,
                currentDustPs: cur,
                currentCombatSec: (curCombatK && isFinite(curCombatK)) ? curCombatK : top.baselineSecPerKill,
                newCombatSec: newK,
                deltaSecPerKill: top.deltaSecPerKill,
                build: cloneBld
              };
            }
          } catch (e) {
            console.warn("Simulator evaluateUpgradeROI fallback failed:", e);
          }
        }
      }
    }

    if (!p || !first) return null;
    return {
      step: first,
      gain: (p.build.d || 0) - cmpCur,
      cost: stepsCostDust([first]).combined,
      build: p.build,
      mode: "toward",
      remaining: p.remaining,
      allSteps: p.steps,
      activeIndex: activeIdx
    };
  }

  function render() {
    if (!state.ab) return;
    var level = Number($("levelSelect").value);
    syncCustomSelect("levelSelect");
    state.level = level;
    syncCustomDustRate(level);
    // Override custom with sim on SA level change
    if (state.previousRenderLevel != null && state.previousRenderLevel !== level) {
      state.dustOverride = "";
      if ($("dustOverride")) $("dustOverride").value = "";
      if ($("dustOverrideWrap")) $("dustOverrideWrap").style.display = "none";
    }
    state.previousRenderLevel = level;

    var rec = bestRunnable(level);
    var rate = activeDustPs(rec);
    var lvlStats = getCurrentCombatStats(level);
    var est = sheetDustPs(rec);
    if ($("btnOpenSim")) $("btnOpenSim").style.display = "inline-flex";

    var gearIcon = $("dustGearIcon");
    var closeIcon = $("dustCloseIcon");
    var toggleBtn = $("btnDustToggleOverride");

    if (state.dustOverride) {
      var customNum = parseNum(state.dustOverride);
      var customTxt = (isFinite(customNum) && customNum > 0) ? prettify(customNum) : esc(state.dustOverride);
      $("dustEstimate").innerHTML = "<span class='tone-buy'>Custom: <b>" + customTxt + "/s</b></span> <span style='opacity:0.75;font-size:11px;'>(Sheet: " + prettify(est) + "/s)</span>";
      $("dustEstimate").title = "Custom override: " + customTxt + "/s. Click X to revert to simulated rate.";
      if (gearIcon) gearIcon.style.display = "none";
      if (closeIcon) closeIcon.style.display = "block";
      if (toggleBtn) toggleBtn.title = "Clear custom override and revert to simulated rate";
    } else if (lvlStats) {
      var farmLvl = (state.ab && (state.ab.enemyLevel || state.ab.maxEnemyLevel)) || state.simulatedLevel || level;
      var muts = (state.save && state.save.global && state.save.global.u2MutationData) || {};
      var mutNote = muts.Dust2 ? " • Mutators: Dusty & Dustier (+50% Dust)" : (muts.Dust ? " • Mutators: Dusty (+25% Dust)" : "");
      if (lvlStats.dustPs > 0) {
        $("dustEstimate").innerHTML = "<span style='color:var(--buy);font-weight:700;'>Sim: " + prettify(lvlStats.dustPs) + "/s</span> <span style='opacity:0.75;font-size:11px;'>(SA " + level + ")</span>";
      } else {
        var wrLbl = lvlStats.winRate > 0 ? (Math.round(lvlStats.winRate) + "% WR") : "0% WR";
        $("dustEstimate").innerHTML = "<span style='color:var(--warn);font-weight:700;'>Sim: 0/s (" + wrLbl + ")</span> <span style='opacity:0.75;font-size:11px;'>(SA " + level + ")</span>";
      }
      var kTip = isFinite(lvlStats.secPerKill) ? lvlStats.secPerKill.toFixed(2) : "--";
      var wrTip = lvlStats.winRate != null ? (" · " + Math.round(lvlStats.winRate) + "% WR") : "";
      var liveNote = (Number(level) !== Number(farmLvl) && state.simulatedDustPs > 0)
        ? ("\nLive save farm rate (SA " + farmLvl + "): " + prettify(state.simulatedDustPs) + "/s (used for bank & build wait estimates).")
        : "";
      $("dustEstimate").title = "Simulated current loadout on SA " + level + ": " + prettify(lvlStats.dustPs) + "/s (~" + kTip + "s/kill" + wrTip + ")" + mutNote + "." + liveNote + "\nClick Simulator to view live combat.";
      if ($("dustOverride") && document.activeElement !== $("dustOverride") && !state.dustOverride && state.simulatedDustPs > 0) {
        $("dustOverride").placeholder = prettify(state.simulatedDustPs) + "/s";
      }
      if (gearIcon) gearIcon.style.display = "block";
      if (closeIcon) closeIcon.style.display = "none";
      if (toggleBtn) toggleBtn.title = "Customize dust rate override";
    } else if (state.simulatedDustPs && state.simulatedDustPs > 0) {
      var farmLvlFallback = (state.ab && (state.ab.enemyLevel || state.ab.maxEnemyLevel)) || state.simulatedLevel || state.level;
      $("dustEstimate").innerHTML = "<span style='color:var(--buy);font-weight:700;'>Sim: " + prettify(state.simulatedDustPs) + "/s</span> <span style='opacity:0.75;font-size:11px;'>(SA " + farmLvlFallback + ")</span>";
      var muts2 = (state.save && state.save.global && state.save.global.u2MutationData) || {};
      var mutNote2 = muts2.Dust2 ? " • Mutators: Dusty & Dustier (+50% Dust)" : (muts2.Dust ? " • Mutators: Dusty (+25% Dust)" : "");
      $("dustEstimate").title = "Auto-calibrated from live save (" + prettify(state.simulatedDustPs) + "/s farming SA " + farmLvlFallback + " · ~" + (state.simulatedDustTimePerKill ? state.simulatedDustTimePerKill.toFixed(2) : "--") + "s/kill)" + mutNote2 + ".\nClick Simulator to view live combat.";
      if ($("dustOverride") && document.activeElement !== $("dustOverride") && !state.dustOverride) {
        $("dustOverride").placeholder = prettify(state.simulatedDustPs) + "/s";
      }
      if (gearIcon) gearIcon.style.display = "block";
      if (closeIcon) closeIcon.style.display = "none";
      if (toggleBtn) toggleBtn.title = "Customize dust rate override";
    } else if (rec) {
      $("dustEstimate").textContent = "Sheet: " + prettify(est) + "/s";
      $("dustEstimate").title = "Recommended build: " + rec.n;
      if (gearIcon) gearIcon.style.display = "block";
      if (closeIcon) closeIcon.style.display = "none";
      if (toggleBtn) toggleBtn.title = "Customize dust rate override";
    } else {
      var pSum = probeBuildSummary(level, null, rate);
      if (pSum && pSum.target) {
        $("dustEstimate").innerHTML = "<span style='color:var(--warn);font-weight:650;'>No runnable build</span> <span style='color:var(--muted);font-size:11px;margin-left:4px;'>(Build: " + esc(pSum.buildTimeStr) + " · Clear: ~" + esc(pSum.clearTimeStr) + ")</span>";
        $("dustEstimate").title = "SA " + level + " requires upgrades.\nTarget: " + pSum.target.n + "\nBuild Time: " + pSum.buildTimeStr + "\nClear Time: ~" + pSum.clearTimeStr + (pSum.totalTimeStr ? ("\nTotal: ~" + pSum.totalTimeStr) : "");
      } else {
        $("dustEstimate").textContent = "No runnable build";
        $("dustEstimate").title = "No sheet builds recorded for SA " + level;
      }
    }
    $("controls").hidden = false;
    $("results").hidden = false;
    if ($("sourceStatus")) {
      $("sourceStatus").className = "pill ok";
      var hDiag = combatDiagnostics();
      var saTag = "SA " + hDiag.enemyLevel + (hDiag.isPushing ? (" (Pushing " + hDiag.progressPct.toFixed(0) + "%)") : (hDiag.isFarming ? (" (Farming / Max " + hDiag.maxEnemyLevel + ")") : ""));
      $("sourceStatus").innerHTML = ICONS.check + " <span>Save Loaded · U2 Z" + (state.radon + 1) + " · " + saTag + "</span>";
      $("sourceStatus").title = "Save loaded successfully\nHighest U2 Zone: Z" + (state.radon + 1) + "\nSA Status: " + hDiag.modeLabel + (hDiag.isPushing ? ("\nLevel Progress: " + prettify(hDiag.curKills) + " / " + prettify(hDiag.neededKills) + " (" + hDiag.remainingKills + " remaining)") : "") + "\nAutoLevel: " + (hDiag.autoLevel ? "ON" : "OFF");
    }
    renderQuickLevels();
    renderLevelModeToggle();
    renderChips(rec, rate);
    renderSaveFacts();
    renderRecommended(rec, rate);
    renderNext(rec, rate);
    renderBoard(rec, rate);
    renderBetter(rec, rate);
    var dlg = $("presetsDlg");
    if (dlg && dlg.open) renderPresetsModal();
  }

  function formatModList(mods) {
    return (mods || []).map(function (m) { return RING_MOD_NAMES[m] || m; }).join(" / ");
  }

  var optimalExtraLimbCache = {};
  function findOptimalExtraLimbItems(targetBuild, level, mode, extraCount) {
    if (!targetBuild || !state.save || extraCount <= 0) {
      return { bestItems: [], isCurOptimal: true };
    }
    var lvl = Number(level || state.level || 1);
    var m = mode || getLevelMode(lvl);
    var bId = (typeof buildId === "function") ? buildId(targetBuild) : ((targetBuild.n || "") + "_" + (targetBuild.d || 0));

    var ownedKeys = ownedItemKeys().filter(function (k) {
      return userOwns(k) && userLevel(k) > 0;
    });
    var candidateKeys = ownedKeys.filter(function (k) {
      return !(targetBuild.i && targetBuild.i[k]);
    });
    if (candidateKeys.length === 0) {
      return { bestItems: [], isCurOptimal: true };
    }

    var curExtraEquipped = ownedKeys.filter(function (k) {
      return userItem(k) && userItem(k).equipped && !(targetBuild.i && targetBuild.i[k]);
    });

    var cacheKey = bId + "@" + lvl + "@" + m + "@" + extraCount + "@" + curExtraEquipped.sort().join(",") + "@" + candidateKeys.map(function (k) { return k + ":" + userLevel(k); }).sort().join(",");
    if (optimalExtraLimbCache[cacheKey]) {
      return optimalExtraLimbCache[cacheKey];
    }

    var baseItems = Object.assign({}, targetBuild.i || {});
    var chosen = [];
    var available = candidateKeys.slice();

    for (var slot = 0; slot < extraCount && available.length > 0; slot++) {
      var bestCand = null;
      var bestScore = -Infinity;
      var curEquippedInSlot = curExtraEquipped[slot] || null;
      var curCandScore = -Infinity;

      for (var ci = 0; ci < available.length; ci++) {
        var cand = available[ci];
        var testBuild = {
          n: targetBuild.n || "Sim",
          i: Object.assign({}, baseItems),
          r: targetBuild.r
        };
        for (var chIdx = 0; chIdx < chosen.length; chIdx++) {
          testBuild.i[chosen[chIdx]] = Math.max(1, userLevel(chosen[chIdx]) || 1);
        }
        testBuild.i[cand] = Math.max(1, userLevel(cand) || 1);

        var score = -Infinity;
        try {
          var simRes = SASimEngine.runBatch({ save: state.save, build: testBuild, level: lvl }, { kills: 15, maxFights: 25, ticks: 10000 });
          if (simRes) {
            if (m === "push") {
              if (simRes.winRate > 0) {
                var effK = simRes.secPerKill / (simRes.winRate / 100);
                score = 1e6 - effK;
              } else {
                score = -1e6 + (simRes.ticks || 0);
              }
            } else {
              score = simRes.dustPs || 0;
            }
          }
        } catch (e) {
          score = -Infinity;
        }

        if (cand === curEquippedInSlot) {
          curCandScore = score;
        }

        if (score > bestScore) {
          bestScore = score;
          bestCand = cand;
        }
      }

      var selectedItem = bestCand;
      if (curEquippedInSlot && curCandScore > -Infinity && bestCand && bestCand !== curEquippedInSlot) {
        var isSignificantlyBetter = false;
        if (m === "push") {
          if (curCandScore < 0 && bestScore > 0) {
            isSignificantlyBetter = true;
          } else if (bestScore > 0 && curCandScore > 0) {
            var bestEffK = 1e6 - bestScore;
            var curEffK = 1e6 - curCandScore;
            if (bestEffK < curEffK * 0.95) {
              isSignificantlyBetter = true;
            }
          }
        } else {
          if (bestScore > curCandScore * 1.02) {
            isSignificantlyBetter = true;
          }
        }
        if (!isSignificantlyBetter) {
          selectedItem = curEquippedInSlot;
        }
      }

      if (selectedItem) {
        chosen.push(selectedItem);
        baseItems[selectedItem] = Math.max(1, userLevel(selectedItem) || 1);
        var remIdx = available.indexOf(selectedItem);
        if (remIdx !== -1) available.splice(remIdx, 1);
      }
    }

    var result = {
      bestItems: chosen,
      isCurOptimal: chosen.length > 0 && chosen.every(function (k) { return curExtraEquipped.indexOf(k) !== -1; })
    };
    optimalExtraLimbCache[cacheKey] = result;
    return result;
  }

  function loadoutDelta(rec, level, mode) {
    if (!rec) return null;
    var lvl = (level != null) ? Number(level) : Number(state.level || 1);
    var m = mode || getLevelMode(lvl);

    var reqKeys = Object.keys(rec.i || {});
    var eqKeys = ownedItemKeys().filter(function (k) {
      return userItem(k) && userItem(k).equipped;
    });
    var planToEquip = reqKeys.filter(function (k) {
      return !(userItem(k) && userItem(k).equipped);
    });
    var curExtraEquipped = eqKeys.filter(function (k) {
      return !rec.i[k];
    });

    var userMaxSlots = maxSlots();
    var surplusSlots = Math.max(0, userMaxSlots - reqKeys.length);

    var toEquip = planToEquip.slice();
    var toUnequip = [];
    var toSwap = [];
    var extraLimbOptimal = [];

    if (surplusSlots > 0 && typeof SASimEngine !== "undefined" && state.save) {
      var optExtra = findOptimalExtraLimbItems(rec, lvl, m, surplusSlots);
      if (optExtra && optExtra.bestItems && optExtra.bestItems.length > 0) {
        var bestKeys = optExtra.bestItems.slice();
        var unassignedCur = curExtraEquipped.slice();

        // Pass 1: if an equipped extra item is already in bestKeys, it's optimal!
        for (var i = unassignedCur.length - 1; i >= 0; i--) {
          var curK = unassignedCur[i];
          var bIdx = bestKeys.indexOf(curK);
          if (bIdx !== -1) {
            extraLimbOptimal.push({ item: curK, isCurEquipped: true });
            bestKeys.splice(bIdx, 1);
            unassignedCur.splice(i, 1);
          }
        }

        // Pass 2: for remaining equipped extra items within surplus slots,
        // if there is a better available item, recommend SWAP (change that limb, never unequip!)
        while (unassignedCur.length > 0 && bestKeys.length > 0) {
          var fromK = unassignedCur.shift();
          var toK = bestKeys.shift();
          toSwap.push({
            from: fromK,
            to: toK,
            isExtraLimb: true,
            reason: (m === "push" ? "Faster clear with extra limb" : "Higher dust with extra limb")
          });
        }

        // Pass 3: if user has open slots and unassigned best items, equip them
        while (bestKeys.length > 0) {
          var eqK = bestKeys.shift();
          toEquip.push(eqK);
          extraLimbOptimal.push({ item: eqK, isCurEquipped: false, isExtraLimb: true });
        }

        // Pass 4: only if user has more items equipped than maxSlots, unequip the excess
        while (unassignedCur.length > 0) {
          toUnequip.push(unassignedCur.shift());
        }
      } else {
        // Fallback: keep up to surplusSlots items equipped without unequipping them
        if (curExtraEquipped.length > surplusSlots) {
          toUnequip = curExtraEquipped.slice(surplusSlots);
        }
        for (var ce = 0; ce < Math.min(curExtraEquipped.length, surplusSlots); ce++) {
          extraLimbOptimal.push({ item: curExtraEquipped[ce], isCurEquipped: true });
        }
      }
    } else {
      toUnequip = curExtraEquipped;
    }

    var reqMods = (rec.r && rec.r.m) ? rec.r.m : [];
    var curMods = hasRing() ? ringMods() : [];
    var missingMods = reqMods.filter(function (mod) {
      return curMods.indexOf(mod) === -1;
    });
    var extraMods = curMods.filter(function (mod) {
      return reqMods.indexOf(mod) === -1;
    });
    var ringMismatch = !!(rec.r && missingMods.length > 0);
    var aligned = toEquip.length === 0 && toUnequip.length === 0 && toSwap.length === 0 && !ringMismatch;

    return {
      toEquip: toEquip,
      toUnequip: toUnequip,
      toSwap: toSwap,
      extraLimbOptimal: extraLimbOptimal,
      reqMods: reqMods,
      curMods: curMods,
      missingMods: missingMods,
      ringMismatch: ringMismatch,
      aligned: aligned
    };
  }

  function toggleSwapGuide() {
    state.swapGuideCollapsed = !state.swapGuideCollapsed;
    try {
      localStorage.setItem("saPlanner.swapGuideCollapsed", state.swapGuideCollapsed ? "1" : "0");
    } catch (e) {}
    var box = $("swapGuideBox");
    if (box) {
      if (state.swapGuideCollapsed) box.classList.add("collapsed");
      else box.classList.remove("collapsed");
    }
  }

  function renderAlignmentGuide(b, delta, titleOverride, isModal) {
    if (!b || !delta) return "";
    var title = titleOverride || "Steps to Initiate Plan · Loadout & Ring Changes";
    var reqMods = formatModList(b.r && b.r.m || []) || "None";
    var curMods = formatModList(delta.curMods) || "None";
    var curRingLv = hasRing() ? ringLevel() : 0;
    var reqRingLv = (b.r && b.r.lv) || 0;
    var ringLvGap = b.r && curRingLv < reqRingLv;

    var missingMods = delta.missingMods || [];
    var hasRingModSwap = b.r && (!hasRing() || missingMods.length > 0);
    var hasItemSwaps = delta && (delta.toEquip.length > 0 || delta.toUnequip.length > 0 || (delta.toSwap && delta.toSwap.length > 0));
    var hasAnySwap = hasRingModSwap || hasItemSwaps || ringLvGap;

    if (!hasAnySwap) {
      var alignedExtraHtml = "";
      if (delta && delta.extraLimbOptimal && delta.extraLimbOptimal.length) {
        var alignedEqOpt = delta.extraLimbOptimal.filter(function (o) { return o.isCurEquipped; });
        if (alignedEqOpt.length > 0) {
          var eqNames = alignedEqOpt.map(function (o) { return itemName(o.item); }).join(", ");
          alignedExtraHtml = "<div style='color:var(--muted);font-size:12px;margin-top:4px;'>Extra Limb: <b>" + esc(eqNames) + "</b> is verified optimal.</div>";
        }
      }
      return "<div class='delta-box aligned' style='margin-top:10px;'>" +
        "<div class='delta-title' style='color:var(--buy);'>" +
          "<div class='delta-title-left'>" + ICONS.check + "<span>Active Loadout &amp; Ring Mods Aligned</span></div>" +
        "</div>" +
        alignedExtraHtml +
      "</div>";
    }

    var isCollapsed = !isModal && !!state.swapGuideCollapsed;
    var dRows = [];
    if (b.r) {
      if (hasRingModSwap || ringLvGap) {
        var rFrom = (hasRing() ? ("Lv " + curRingLv) : "Locked") + " (" + curMods + ")";
        var rTo = "Lv " + reqRingLv + " (" + reqMods + ")";
        dRows.push(
          "<div class='delta-row'><span class='delta-lbl'>Ring:</span>" +
          "<span class='pill sm " + (hasRingModSwap ? "warn-pill" : "teal") + "'>" + ICONS.ring + " " + esc(rFrom + " → " + rTo) + "</span></div>"
        );
      }
    }
    if (delta.toSwap && delta.toSwap.length) {
      var swPills = delta.toSwap.map(function (sw) {
        var fromName = itemName(sw.from);
        var toName = itemName(sw.to);
        var toLv = userLevel(sw.to) || 1;
        return "<span class='pill sm accent' title='" + esc(sw.reason || "Change extra limb to optimal choice") + "'>" +
          iconForItem(sw.from) + " " + esc(fromName) + " → " + iconForItem(sw.to) + " " + esc(toName + " (Lv " + toLv + ")") +
          "</span>";
      }).join("");
      dRows.push("<div class='delta-row'><span class='delta-lbl'>Extra Limb Swap:</span>" + swPills + "</div>");
    }
    if (delta.extraLimbOptimal && delta.extraLimbOptimal.length) {
      var guideEqOpt = delta.extraLimbOptimal.filter(function (o) { return o.isCurEquipped; });
      if (guideEqOpt.length > 0) {
        var optPills = guideEqOpt.map(function (o) {
          return "<span class='pill sm buy' title='Equipped item is verified optimal for your extra limb slot'>" +
            iconForItem(o.item) + " " + esc(itemName(o.item)) + " (Optimal)" +
            "</span>";
        }).join("");
        dRows.push("<div class='delta-row'><span class='delta-lbl'>Extra Limb:</span>" + optPills + "</div>");
      }
    }
    if (delta.toUnequip && delta.toUnequip.length) {
      var unPills = delta.toUnequip.map(function (k) {
        return "<span class='pill sm sell'>" + iconForItem(k) + " " + esc(itemName(k)) + "</span>";
      }).join("");
      dRows.push("<div class='delta-row'><span class='delta-lbl'>Unequip:</span>" + unPills + "</div>");
    }
    if (delta.toEquip && delta.toEquip.length) {
      var eqPills = delta.toEquip.map(function (k) {
        var owned = userOwns(k);
        var myLv = owned ? userLevel(k) : 0;
        var needLv = (b.i && b.i[k]) ? b.i[k] : myLv;
        var isExtra = !(b.i && b.i[k]);
        var lvSuffix = !owned ? " (Locked)" : (isExtra ? (" (Lv " + myLv + " · Extra)") : (myLv < needLv ? (" (Lv " + myLv + "→" + needLv + ")") : (" (Lv " + needLv + ")")));
        return "<span class='pill sm buy'>" + iconForItem(k) + " " + esc(itemName(k) + lvSuffix) + "</span>";
      }).join("");
      dRows.push("<div class='delta-row'><span class='delta-lbl'>Equip:</span>" + eqPills + "</div>");
    }

    var chevronSvg = "<svg class='delta-chevron' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2.5' stroke-linecap='round' stroke-linejoin='round'><polyline points='6 9 12 15 18 9'></polyline></svg>";
    var clickAttr = isModal ? "" : " onclick=\"window.saToggleSwapGuide&&saToggleSwapGuide()\" role=\"button\" tabindex=\"0\" onkeydown=\"if(event.key==='Enter'||event.key===' '){event.preventDefault();window.saToggleSwapGuide&&saToggleSwapGuide();}\"";
    var boxClass = "delta-box" + (isModal ? "" : " collapsible") + (isCollapsed ? " collapsed" : "");
    var boxId = isModal ? "" : " id='swapGuideBox'";

    return "<div" + boxId + " class='" + boxClass + "' style='margin-top:10px;'" + clickAttr + ">" +
      "<div class='delta-title'>" +
        "<div class='delta-title-left'>" + ICONS.warn + "<span>" + esc(title) + "</span></div>" +
        (isModal ? "" : chevronSvg) +
      "</div>" +
      "<div class='delta-rows'>" + dRows.join("") + "</div>" +
      "</div>";
  }

  function probeBuildSummary(level, rec, rateOverride, stepsOverride) {
    var bnk = bank();
    var r = (typeof rateOverride === "number" && rateOverride > 0)
      ? rateOverride
      : (activeDustPs(rec) || (state.ab ? activeDustPs(bestRunnable(level)) : 0));
    var pick = (rec && runnable(rec)) ? null : bestNextUpgrade(rec, r, levelCurrentDustPs(level, rec));
    var target = (rec && rec.n) ? rec : (pick && pick.build ? pick.build : null);
    var allSteps = stepsOverride || null;

    if (!target) {
      var allB = buildsFor(level);
      if (allB && allB.length) {
        var bestScore = Infinity;
        var probeMode = getLevelMode(level);
        for (var i = 0; i < allB.length; i++) {
          var b = allB[i];
          var st = applyStepOrder(level, b, expandSteps(gapSteps(b)));
          var wi = calculateStepWaitTimes(st, bnk, r);
          var kSec = buildSecPerKill(b, level);
          var score = (probeMode === "push")
            ? ((st.length * 1e12) + (kSec * 1e6) + (isFinite(wi.totalWait) ? wi.totalWait : 1e11))
            : ((st.length * 1e12) + (isFinite(wi.totalWait) ? wi.totalWait : 1e11));
          if (score < bestScore) {
            bestScore = score;
            target = b;
            allSteps = st;
          }
        }
      }
    }

    if (!target) return null;

    if (!allSteps && !runnable(target)) {
      if (pick && pick.build && (pick.build.n === target.n || buildId(pick.build) === buildId(target)) && pick.allSteps) {
        allSteps = pick.allSteps;
      } else if (state.lastPlanSteps && state.lastAdvisorStep && state.lastAdvisorStep.build && buildId(state.lastAdvisorStep.build) === buildId(target)) {
        allSteps = state.lastPlanSteps;
      } else {
        allSteps = applyStepOrder(level, target, expandSteps(gapSteps(target)));
      }
    } else if (!allSteps) {
      allSteps = [];
    }

    var wInfo = allSteps.length ? calculateStepWaitTimes(allSteps, bnk, r) : { totalWait: 0, wait: 0, remDust: 0, remShards: 0, remDustEq: 0 };
    wInfo.wait = wInfo.totalWait;
    var hasPendingSteps = allSteps.some(function (st) { return !st.completed; });
    var buildSec = (!hasPendingSteps && runnable(target)) ? 0 : wInfo.wait;
    var neededKills = saLevelReq(level);
    var isCurrentLevel = state.ab && (state.ab.enemyLevel || state.ab.maxEnemyLevel) === level;
    var liveKillsEarned = 0;
    if (isCurrentLevel && bnk && bnk.elapsedSec > 0) {
      var liveSpk = getPushSecPerKill(level) || 15;
      liveKillsEarned = Math.floor(bnk.elapsedSec / liveSpk);
    }
    var totalKillsDone = (state.ab && state.ab.enemiesKilled) ? (state.ab.enemiesKilled + liveKillsEarned) : liveKillsEarned;
    var remainingKills = isLevelCleared(level) ? 0 : (isCurrentLevel ? Math.max(0, neededKills - totalKillsDone) : neededKills);

    var secPerKill = (target.k && target.k > 0) ? target.k : buildSecPerKill(target, level);
    if (!secPerKill || !isFinite(secPerKill) || secPerKill <= 0) secPerKill = 15;

    var clearSec = remainingKills * secPerKill;
    var totalSec = (isFinite(buildSec) && isFinite(clearSec)) ? (buildSec + clearSec) : null;

    var buildTimeStr = (buildSec === 0) ? "Ready now" : fmtDuration(buildSec);
    var clearTimeStr = fmtDuration(clearSec);
    var totalTimeStr = totalSec != null ? fmtDuration(totalSec) : "--";

    var summaryObj = {
      target: target,
      steps: allSteps,
      stepCount: allSteps.length,
      buildSec: buildSec,
      clearSec: clearSec,
      totalSec: totalSec,
      buildTimeStr: buildTimeStr,
      clearTimeStr: clearTimeStr,
      totalTimeStr: totalTimeStr,
      remainingKills: remainingKills,
      secPerKill: secPerKill,
      wInfo: wInfo,
      pick: pick
    };
    state.lastPlanSummary = summaryObj;
    return summaryObj;
  }

  function renderRecommended(rec, rate) {
    var el = $("recommend");
    var lockedEntry = state.lockedPlans && state.lockedPlans[String(state.level)];
    var lockedTarget = lockedEntry ? findLockedTarget(state.level, lockedEntry) : null;

    var activeBuild = lockedTarget || rec;
    var isPlanLocked = !!lockedTarget;
    var mode = getLevelMode(state.level);
    var isOptimalClear = false;
    var isOptimalFarm = false;
    var currentCombatSec = null;

    if (mode === "push" && !isPlanLocked) {
      currentCombatSec = getCurrentCombatSecPerKill(state.level);
      var isCurWinning = isFinite(currentCombatSec) && currentCombatSec > 0;
      var recK = rec ? buildSecPerKill(rec, state.level) : Infinity;
      if (isCurWinning && (recK >= currentCombatSec - 0.01 || !rec)) {
        // Current loadout can actually beat the level and is as fast or faster than any runnable sheet build
        var curCombatBuild = getCurrentEquippedCombatBuild(state.level);
        if (curCombatBuild) {
          activeBuild = curCombatBuild;
          activeBuild.k = currentCombatSec;
          isOptimalClear = true;
        }
      } else if (!rec || !isCurWinning) {
        // Current build cannot beat the level (or no runnable sheet build exists):
        // Recommend the fastest ROI push plan instead!
        var pushPick = bestNextUpgrade(rec, rate);
        if (pushPick && pushPick.build) {
          activeBuild = pushPick.build;
        }
      }
    } else if (mode === "farm" && !isPlanLocked) {
      var curStats = getCurrentCombatStats(state.level);
      var curDust = levelCurrentDustPs(state.level, rec);
      var isWinningOnLevel = curDust > 0 || !!(curStats && curStats.kills > 0);
      var recDust = rec ? (rec.d || 0) : 0;
      if (isWinningOnLevel && curDust > 0 && curDust > recDust) {
        // Current loadout outperforms the runnable sheet build on this level.
        // Check if there is a better income sheet available via the advisor:
        var farmPick = bestNextUpgrade(rec, rate, curDust);
        if (farmPick && farmPick.build && (farmPick.build.d || 0) > curDust) {
          // Target the better income sheet build (e.g. anon1)
          activeBuild = farmPick.build;
        } else {
          // No better income sheet build available: set Current Loadout as optimal farm
          var curFarmBuild = getCurrentEquippedCombatBuild(state.level);
          if (curFarmBuild) {
            activeBuild = curFarmBuild;
            activeBuild.d = curDust;
            isOptimalFarm = true;
          }
        }
      }
    }

    if (!activeBuild) {
      var summary = probeBuildSummary(state.level, null, rate);
      if (!summary || !summary.target) {
        el.innerHTML = "<div class='rec'><p class='empty'>No sheet build on SA " + state.level + " is recorded in the database.</p></div>";
        return;
      }
      activeBuild = summary.target;
    }

    var isTargetOnly = !runnable(activeBuild);
    var pickForRec = bestNextUpgrade(rec, rate);
    var pStepsForRec = (pickForRec && pickForRec.build && buildId(pickForRec.build) === buildId(activeBuild)) ? pickForRec.allSteps : null;
    var summaryInfo = probeBuildSummary(state.level, activeBuild, rate, pStepsForRec);
    state.lastActiveBuild = activeBuild;
    var targetDelta = loadoutDelta(activeBuild, state.level, mode);
    var targetMods = formatModList(activeBuild.r && activeBuild.r.m || []);

    var itemCards = Object.keys(activeBuild.i).map(function (k) {
      var sh = isShards(k);
      var myLv = userLevel(k);
      var reqLv = activeBuild.i[k];
      var isEq = !!(userItem(k) && userItem(k).equipped);
      var under = myLv < reqLv;
      var cls = "loadout-item" + (sh ? " shards" : "") + (!isEq ? " unequipped" : "") + (under ? " underleveled" : "");
      var tip = (under ? ("Upgrade needed: Lv " + myLv + " → Lv " + reqLv) : ("Requirement met: Lv " + myLv + " >= " + reqLv)) +
        (isEq ? " · Equipped" : " · Unequipped");
      var lvText = under ? ("Lv " + myLv + " → " + reqLv) : ("Lv " + reqLv);
      return "<div class='" + cls + "' title='" + esc(tip) + "'>" +
        "<span class='li-name'>" + iconForItem(k) + "<span>" + esc(itemName(k)) + "</span></span>" +
        "<span class='li-lv'>" + esc(lvText) + "</span>" +
        "</div>";
    }).join("");

    if (targetDelta) {
      var extraItemsToShow = [];
      if (targetDelta.extraLimbOptimal) {
        targetDelta.extraLimbOptimal.forEach(function (opt) {
          if (!activeBuild.i || !activeBuild.i[opt.item]) {
            extraItemsToShow.push({ key: opt.item, isEq: opt.isCurEquipped, isSwap: false });
          }
        });
      }
      if (targetDelta.toSwap) {
        targetDelta.toSwap.forEach(function (sw) {
          extraItemsToShow.push({ key: sw.to, isEq: false, isSwap: true, from: sw.from });
        });
      }
      var extraCardsHtml = extraItemsToShow.map(function (entry) {
        var k = entry.key;
        var myLv = userLevel(k) || 1;
        var sh = isShards(k);
        var cls = "loadout-item" + (sh ? " shards" : "") + (!entry.isEq ? " unequipped" : "");
        var tip = "Extra Limb Slot: " + (entry.isEq ? "Equipped (Optimal choice)" : (entry.isSwap ? ("Recommended swap from " + itemName(entry.from)) : "Recommended choice"));
        var lvText = "Lv " + myLv + " · Extra";
        return "<div class='" + cls + "' title='" + esc(tip) + "' style='border-color:color-mix(in srgb,var(--buy) 45%,var(--border));'>" +
          "<span class='li-name'>" + iconForItem(k) + "<span>" + esc(itemName(k)) + "</span></span>" +
          "<span class='li-lv' style='color:var(--buy);font-weight:600;'>" + esc(lvText) + "</span>" +
          "</div>";
      }).join("");
      itemCards += extraCardsHtml;
    }

    var recSub = $("recCardSub");
    if (recSub) {
      if (isPlanLocked) {
        recSub.textContent = "Custom locked plan for SA " + state.level;
      } else if (isOptimalClear) {
        recSub.textContent = "Current loadout is optimal for push (Fastest Clear Active)";
      } else if (isOptimalFarm) {
        recSub.textContent = "Current loadout is optimal for farm (Highest Dust Active)";
      } else if (isTargetOnly) {
        recSub.textContent = (mode === "push")
          ? "Target clear setup (Upgrades needed)"
          : "Target farm setup (Upgrades needed)";
      } else {
        recSub.textContent = (mode === "push")
          ? "Fastest runnable clear setup (Clear Priority)"
          : "Best runnable farm setup right now (Dust Priority)";
      }
    }
    var isFarm = mode === "farm";
    var lvlTagPill = "<span class='pill sm " + (isFarm ? "buy" : "accent") + "' title='Recommendation Mode: " + (isFarm ? "Farm Mode (Dust rate prioritized)" : "Push Mode (Clear time prioritized)") + "'>" +
      (isFarm ? ICONS.dust + " Farm Mode" : ICONS.spire + " Push Mode") + "</span>";

    var metrics = [];
    metrics.push(lvlTagPill);
    var dispD = activeBuild.d || ((isOptimalClear || isOptimalFarm) ? (isOptimalFarm ? levelCurrentDustPs(state.level, rec) : rate) : 0);
    if (dispD > 0) metrics.push("<span class='pill sm buy' title='Expected Dust Earning Rate: " + prettify(dispD) + "/s'>" + ICONS.dust + " " + prettify(dispD) + "/s</span>");
    if (activeBuild.k != null && isFinite(activeBuild.k)) metrics.push("<span class='pill sm info'>" + ICONS.clock + " ~" + Number(activeBuild.k).toFixed(1) + "s kills" + (isOptimalClear ? " (Fastest)" : "") + "</span>");
    if (activeBuild.wr != null) metrics.push("<span class='pill sm accent'>" + Math.round((activeBuild.wr <= 1 ? activeBuild.wr : activeBuild.wr / 100) * 100) + "% WR</span>");
    if (summaryInfo && summaryInfo.clearSec > 0) {
      metrics.push("<span class='pill sm info' title='Anticipated time to clear SA " + state.level + " (" + prettify(summaryInfo.remainingKills) + " kills @ " + summaryInfo.secPerKill.toFixed(1) + "s/kill)'>" + ICONS.clock + " Clear: ~" + esc(summaryInfo.clearTimeStr) + "</span>");
    }
    metrics.push("<span class='pill sm'>" + ICONS.limb + " " + (activeBuild.l || "?") + " Limbs</span>");
    if (activeBuild.r) {
      if (targetDelta.ringMismatch) {
        var curModTxt = formatModList(targetDelta.curMods) || "None";
        metrics.push("<span class='pill sm warn-pill' title='Your Ring is currently set to " + esc(curModTxt) + " — switch to " + esc(targetMods) + "'>" + ICONS.warn + " Ring " + activeBuild.r.lv + " (" + esc(curModTxt) + " → " + esc(targetMods) + ")</span>");
      } else {
        metrics.push("<span class='pill sm teal'>" + ICONS.ring + " Ring " + activeBuild.r.lv + (targetMods ? " (" + esc(targetMods) + ")" : "") + "</span>");
      }
    }
    if (activeBuild.c) metrics.push("<span class='pill sm violet'>" + ICONS.contract + " Z" + activeBuild.c + "</span>");

    var remSteps = summaryInfo ? summaryInfo.steps.length : 0;
    var simBtn = "<button type='button' class='btn sm ghost sim-plan-btn' id='recSimPlanBtn' title='Load " + esc(activeBuild.n) + " into Combat Simulator'>⚔️ Sim</button>";
    var lockBtn = (isPlanLocked || (summaryInfo && summaryInfo.pick && summaryInfo.pick.isLocked))
      ? "<button type='button' class='btn sm accent lock-plan-btn locked' id='recLockPlanBtn' title='Advisor is locked to " + esc(activeBuild.n) + ". Click to unlock.'>" + ICONS.lock + "<span>Locked</span></button>"
      : "<button type='button' class='btn sm ghost lock-plan-btn' id='recLockPlanBtn' title='Lock advisor to " + esc(activeBuild.n) + " so recommendations follow this build.'>" + ICONS.unlock + "<span>Lock Plan</span></button>";

    var statusPills = [];
    if (isPlanLocked) {
      statusPills.push("<span class='pill sm accent'>" + ICONS.lock + " Active Plan</span>");
    }
    if (isOptimalClear) {
      statusPills.push("<span class='pill sm buy'>" + ICONS.check + " Fastest Clear Active</span>");
      statusPills.push("<span class='pill sm buy'>" + ICONS.check + " Optimal &amp; In Play</span>");
    } else if (isOptimalFarm) {
      statusPills.push("<span class='pill sm buy'>" + ICONS.check + " Highest Dust Active</span>");
      statusPills.push("<span class='pill sm buy'>" + ICONS.check + " Optimal &amp; In Play</span>");
    } else if (isTargetOnly) {
      statusPills.push("<span class='pill sm warn-pill'>" + ICONS.bolt + " " + remSteps + " Upgrade" + (remSteps === 1 ? "" : "s") + " Needed</span>");
    } else if (targetDelta.aligned) {
      statusPills.push("<span class='pill sm buy'>" + ICONS.check + " Active &amp; Aligned</span>");
    } else {
      statusPills.push("<span class='pill sm warn-pill'>" + ICONS.warn + " Adjustments Needed</span><span class='pill sm buy'>" + ICONS.check + " Runnable</span>");
    }

    var presetMatch = findBestPresetMatch(activeBuild);
    if (presetMatch) {
      if (presetMatch.isExact) {
        statusPills.push("<span class='pill sm buy' style='cursor:pointer;' onclick='window.saOpenPresets&&saOpenPresets()' title='Matches in-game " + esc(presetMatch.preset.name) + " (" + presetMatch.preset.slotTag + "). Click to open presets modal.'>" +
          ICONS.check + " Matches " + esc(presetMatch.preset.slotTag) + ' ("' + esc(presetMatch.preset.name) + '")</span>');
      } else if (presetMatch.isClose) {
        var swapItemNames = presetMatch.missing.map(itemName).join(", ") || "1 mod/item";
        statusPills.push("<span class='pill sm accent' style='cursor:pointer;' onclick='window.saOpenPresets&&saOpenPresets()' title='1 swap from in-game " + esc(presetMatch.preset.name) + " (" + presetMatch.preset.slotTag + "): " + esc(swapItemNames) + ". Click to open presets modal.'>" +
          ICONS.target + " ~ " + esc(presetMatch.preset.slotTag) + ' (1 swap)</span>');
      }
    }

    statusPills.push(simBtn);
    statusPills.push(lockBtn);

    var timingBox = "";
    if (summaryInfo && (summaryInfo.buildSec > 0 || isTargetOnly || isPlanLocked)) {
      timingBox =
        "<div class='rec-timing-banner'>" +
          "<div class='time-stat' title='Time to save required dust/shards for all " + remSteps + " upgrade steps (" + (rate ? (prettify(rate) + "/s dust rate") : "at current rate") + ")'>" +
            "<span class='time-lbl'>" + ICONS.hammer + " Build Time:</span>" +
            "<span class='mono' id='recBuildTimeVal' style='font-size:13.5px;font-weight:700;color:" + (summaryInfo.buildSec === 0 ? "var(--buy)" : "var(--text)") + ";'>" + esc(summaryInfo.buildTimeStr) + "</span>" +
          "</div>" +
          "<div class='time-sep'></div>" +
          "<div class='time-stat' title='Anticipated time to clear SA " + state.level + " once " + esc(activeBuild.n) + " is active (" + prettify(summaryInfo.remainingKills) + " kills @ ~" + summaryInfo.secPerKill.toFixed(1) + "s per kill)'>" +
            "<span class='time-lbl'>" + ICONS.clock + " Clear Time:</span>" +
            "<span class='mono' id='recClearTimeVal' style='font-size:13.5px;font-weight:700;color:var(--accent);'>~" + esc(summaryInfo.clearTimeStr) + "</span>" +
          "</div>" +
          (summaryInfo.totalTimeStr ? (
            "<div class='time-sep'></div>" +
            "<div class='time-stat' title='Total duration: Build Time + Clear Time'>" +
              "<span class='time-lbl'>" + ICONS.spire + " Total:</span>" +
              "<span class='mono' id='recTotalTimeVal' style='font-size:13.5px;font-weight:700;color:var(--buy);'>~" + esc(summaryInfo.totalTimeStr) + "</span>" +
            "</div>"
          ) : "") +
        "</div>";
    }

    var headPrefix = isPlanLocked ? "Target Plan:" : ((isOptimalClear || isOptimalFarm) ? ("Current Loadout (" + (isOptimalFarm ? "Optimal Farm" : "Optimal Clear") + "):") : (isTargetOnly ? "Target Build:" : "Recommended Build:"));
    var swapTitle = isPlanLocked ? "Steps to Initiate Plan · Loadout & Ring Changes" : ((isOptimalClear || isOptimalFarm) ? "Current Combat Loadout · Fully Aligned" : (isTargetOnly ? "Steps to Initiate Plan · Loadout & Ring Changes" : "Steps to Initiate Build · Loadout & Ring Changes"));
    var swapGuideHtml = (isOptimalClear || isOptimalFarm)
      ? ("<div class='delta-box' style='margin-top:4px;border-color:rgba(45,206,137,0.35);background:color-mix(in srgb,var(--buy) 6%,var(--surface-solid));'>" +
          "<div class='delta-title' style='color:var(--buy);'>" + ICONS.check + "<span>Current Loadout Is Optimal · No Slower Swaps Recommended</span></div>" +
          "<div style='color:var(--muted);font-size:12px;'>" +
          (isOptimalFarm
            ? ("Your current equipped items earn " + prettify(dispD) + "/s on SA " + state.level + ", which is higher than all recorded runnable sheet builds. Keep this loadout active while farming.")
            : ("Your current equipped items clear SA " + state.level + " in ~" + (isFinite(currentCombatSec) ? currentCombatSec.toFixed(1) : "15.0") + "s per kill, which is faster than all recorded runnable sheet builds. Keep this loadout active while pushing.")) +
          "</div>" +
          "</div>")
      : renderAlignmentGuide(activeBuild, targetDelta, swapTitle);

    var explText = "";
    if (isPlanLocked && rec && rec.n !== activeBuild.n) {
      explText = "<div class='notes' style='margin-top:8px;'>" + ICONS.info + "<span>Advisor is locked to your plan <b>" + esc(activeBuild.n) + "</b>. Currently running: <b>" + esc(rec.n) + "</b> (" + prettify(rec.d) + "/s). Follow the swap guide above and the Next Upgrade Advisor on the right to transition.</span></div>";
    } else if (isOptimalClear || isOptimalFarm) {
      explText = "<div class='notes' style='margin-top:8px;'>" + ICONS.info + "<span>" +
        (isOptimalFarm
          ? ("Your current equipped loadout earns " + prettify(dispD) + "/s on SA " + state.level + ". No higher-income runnable sheet build exists on this level. Follow the Upgrade Value advisor on the right for the highest-impact stat upgrades.")
          : ("Your current equipped loadout clears SA " + state.level + " in ~" + (isFinite(currentCombatSec) ? currentCombatSec.toFixed(1) : "15.0") + "s per kill. No faster runnable clear build exists on this level. Follow the Upgrade Value advisor on the right for the highest-impact stat upgrades.")) +
        "</span></div>";
    } else if (isTargetOnly) {
      explText = "<div class='notes' style='margin-top:8px;'>" + ICONS.info + "<span>No sheet build on SA " + state.level + " is currently runnable with your items and levels. Targeting <b>" + esc(activeBuild.n) + "</b> (" + remSteps + " upgrade step" + (remSteps === 1 ? "" : "s") + " to unlock). Follow the swap guide above and the Next Upgrade Advisor on the right to progress toward this build.</span></div>";
    } else if (activeBuild.note) {
      explText = "<div class='notes' style='margin-top:8px;'>" + ICONS.info + "<span>" + esc(activeBuild.note) + "</span></div>";
    }

    el.innerHTML =
      "<div class='rec'>" +
      "<div class='rec-top'>" +
      "<div class='rec-head'><span style='color:var(--muted);font-size:12px;font-weight:700;text-transform:uppercase;letter-spacing:0.5px;'>" + headPrefix + "</span> <span>" + esc(activeBuild.n) + "</span></div>" +
      "<div style='display:flex;gap:6px;flex-wrap:wrap;align-items:center;'>" + statusPills.join("") + "</div>" +
      "</div>" +
      timingBox +
      "<div class='rec-metrics'>" + metrics.join("") + "</div>" +
      swapGuideHtml +
      explText +
      "<div class='loadout-grid'>" + itemCards + "</div>" +
      "</div>";

    var rLockBtn = $("recLockPlanBtn");
    if (rLockBtn) {
      rLockBtn.addEventListener("click", function () {
        toggleLockBuild(activeBuild);
      });
    }
    var rSimBtn = $("recSimPlanBtn");
    if (rSimBtn) {
      rSimBtn.addEventListener("click", function () {
        openSimModal(activeBuild);
      });
    }
  }

  var planSimToken = 0;

  function updatePlanSimIndicator(inProgress, currentStep, totalSteps) {
    state.planSimInProgress = !!inProgress;
    var el = $("spSimStatus");
    if (!el) return;
    if (inProgress) {
      el.style.display = "inline-flex";
      el.className = "sp-sim-status simulating";
      el.textContent = "⚡ Simulating (" + currentStep + "/" + totalSteps + ")...";
    } else {
      el.className = "sp-sim-status";
      el.style.display = "inline-flex";
      el.textContent = "⚡ 9001x Calibrated";
      setTimeout(function () {
        if (el && !state.planSimInProgress) {
          el.textContent = "⚡ 9001x";
        }
      }, 3500);
    }
  }

  function buildPillTitle(buildName, buildSec, buildTimeStr, clearSec, clearTimeStr, totalSec, totalTimeStr, cleared) {
    var bName = buildName || "Build";
    var buildEta = (buildSec > 0 && isFinite(buildSec)) ? (" (Completes: " + fmtLocalEta(buildSec) + ")") : "";
    if (cleared) {
      return "Farming time to unlock " + bName + ": " + buildTimeStr + buildEta + " (SA " + state.level + " already cleared)";
    }
    var clearEta = (totalSec != null && totalSec > 0 && isFinite(totalSec)) ? (" · ETA: " + fmtLocalEta(totalSec)) : "";
    return "Farming time to unlock " + bName + ": " + buildTimeStr + buildEta +
      " · Clear time with " + bName + ": ~" + clearTimeStr +
      (totalTimeStr ? (" (Total: ~" + totalTimeStr + clearEta + ")") : "");
  }

  function refreshPlanPopoverWaitTimes() {
    if (!state.lastPlanSteps || !state.lastPlanSteps.length) return;
    var bnk = bank();
    var defaultRate = activeDustPs(bestRunnable(state.level));
    var calc = calculateStepWaitTimes(state.lastPlanSteps, bnk, defaultRate);

    var listEl = $("spStepList");
    if (listEl) {
      var timeEls = listEl.querySelectorAll(".sp-time");
      var subEls = listEl.querySelectorAll(".sp-sub");
      var pSteps = state.lastPlanSteps;
      for (var pi = 0; pi < pSteps.length; pi++) {
        var pst = pSteps[pi];
        var tEl = timeEls[pi];
        if (!tEl || pst.completed) continue;
        var pw = pst.estWait != null ? pst.estWait : 0;
        var pStr = pw === 0 ? "Ready now" : (pw != null && isFinite(pw) ? fmtTime(pw) : "—");
        if (tEl.textContent !== pStr) tEl.textContent = pStr;
        if (pw === 0) {
          if (!tEl.classList.contains("ready")) tEl.classList.add("ready");
        } else {
          if (tEl.classList.contains("ready")) tEl.classList.remove("ready");
        }
        if (pst._stepRate && isFinite(pst._stepRate) && Math.abs(pst._stepRate - defaultRate) > 1e6) {
          tEl.title = "Wait: " + pStr + " @ " + prettify(pst._stepRate) + "/s simulated earn rate (9001 fights)";
          if (subEls && subEls[pi]) {
            var subText = subEls[pi].textContent || "";
            if (!subText.includes("Sim:")) {
              subEls[pi].textContent = subText + " · Sim: " + prettify(pst._stepRate) + "/s";
            }
          }
        }
      }
    }

    var totalWaitEl = $("spTotalWait");
    if (totalWaitEl) {
      var twStr = calc.totalWait === 0 ? "Ready now" : fmtTime(calc.totalWait);
      if (totalWaitEl.textContent !== twStr) totalWaitEl.textContent = twStr;
    }

    var durStr = calc.totalWait === 0 ? "Ready now" : fmtDuration(calc.totalWait);
    var recBVal = $("recBuildTimeVal");
    if (recBVal && recBVal.textContent !== durStr) {
      recBVal.textContent = durStr;
    }
    var recTVal = $("recTotalTimeVal");
    if (recTVal) {
      var clearSec = (state.lastPlanSummary && state.lastPlanSummary.clearSec) || 0;
      var totalSec = calc.totalWait + clearSec;
      var tStr = "~" + fmtDuration(totalSec);
      if (recTVal.textContent !== tStr) recTVal.textContent = tStr;
    }
    var advBPill = $("advPlanBuildTime");
    if (advBPill) {
      var cleared = isLevelCleared(state.level);
      var advClearSec = (state.lastPlanSummary && state.lastPlanSummary.clearSec) || 0;
      var advClearStr = fmtDuration(advClearSec);
      var newHtml = cleared
        ? (ICONS.hammer + " Build: " + durStr)
        : (ICONS.clock + " Build: " + durStr + " · Clear: ~" + advClearStr);
      if (advBPill.innerHTML !== newHtml) advBPill.innerHTML = newHtml;
      var advBuildName = advBPill.getAttribute("data-build-name") || (state.lastPlanSummary && state.lastPlanSummary.target && state.lastPlanSummary.target.n) || "Build";
      var advTotalSec = (isFinite(calc.totalWait) && isFinite(advClearSec)) ? (calc.totalWait + advClearSec) : null;
      var advTotalStr = advTotalSec != null ? fmtDuration(advTotalSec) : "";
      var newTitle = buildPillTitle(advBuildName, calc.totalWait, durStr, advClearSec, advClearStr, advTotalSec, advTotalStr, cleared);
      if (advBPill.title !== newTitle) advBPill.title = newTitle;
    }

    var advWaitBadge = $("advisorWaitBadge");
    if (advWaitBadge && state.lastPlanSteps) {
      var firstPending = state.lastPlanSteps.find(function (s) { return !s.completed; });
      if (firstPending) {
        var fw = firstPending.estWait != null ? firstPending.estWait : 0;
        if (fw === 0) {
          advWaitBadge.className = "pill sm buy";
          advWaitBadge.innerHTML = ICONS.check + " Ready now";
        } else {
          advWaitBadge.className = "pill sm";
          var advPrefix = advWaitBadge.getAttribute("data-step-prefix") || "";
          advWaitBadge.innerHTML = ICONS.clock + " " + advPrefix + fmtTime(fw);
        }
      }
    }
  }

  var betterSimToken = 0;
  var stepSimCache = {};
  var planRatesCache = {};

  function getStepCfgKey(fightLevel, cfg) {
    if (!cfg) return "";
    var eqStr = (cfg.equipped || []).map(function (k) {
      return k + ":" + (cfg.items && cfg.items[k] ? cfg.items[k].level : 1);
    }).join(",");
    var ringLv = (cfg.ring && cfg.ring.level) || 0;
    var ringOT = (cfg.oneTimers && cfg.oneTimers.The_Ring) ? 1 : 0;
    return fightLevel + "|" + ringLv + "|" + ringOT + "|" + eqStr;
  }

  function triggerPlanStepSimulation(build, steps, defaultRate) {
    if (typeof SASimEngine === "undefined" || !state.save || !steps || steps.length <= 1) return;

    var pending = steps.filter(function (s) { return !s.completed; });
    if (!pending.length) return;

    var fightLevel = (state.ab && (state.ab.enemyLevel || state.ab.maxEnemyLevel)) || state.level || 1;
    var stepsSig = pending.map(function (s) { return stepKey(s) + "@" + (s.to || 1); }).join(";");
    var loadoutSig = getLoadoutSignature();
    var fullSig = "sig:" + fightLevel + "|" + loadoutSig + "|" + stepsSig;
    var bId = build ? buildId(build) : "active";
    var planCacheKey = fightLevel + "|" + bId + "|" + stepsSig;

    // Fast-path 1: Already marked as fully simulated for this exact sequence
    if (state.lastSimulatedStepsSig === fullSig) {
      return;
    }

    // Fast-path 2: Plan rates already computed for this exact save and step sequence!
    // No combat simulation needed; simply apply the rates in 0ms!
    if (planRatesCache[planCacheKey]) {
      var cached = planRatesCache[planCacheKey];
      for (var ci = 0; ci < pending.length; ci++) {
        if (cached[ci]) {
          pending[ci]._stepRate = cached[ci].stepRate;
          pending[ci]._postRate = cached[ci].postRate;
          pending[ci]._postRateLevel = cached[ci].postRateLevel;
        }
      }
      state.lastSimulatedStepsSig = fullSig;
      state.activePlanSimSig = null;
      updatePlanSimIndicator(false, pending.length, pending.length);
      refreshPlanPopoverWaitTimes();
      return;
    }

    // Fast-path 3: Simulation is already running for this exact signature, do not restart
    if (state.activePlanSimSig === fullSig) {
      return;
    }
    state.activePlanSimSig = fullSig;

    var currentToken = ++planSimToken;
    updatePlanSimIndicator(true, 0, pending.length);

    var baseCfg = SASimEngine.normalizeConfig({ save: state.save, level: fightLevel });
    var currentCfg = JSON.parse(JSON.stringify(baseCfg));
    var currentRate = (state.simulatedDustPs && state.simulatedDustPs > 0) ? state.simulatedDustPs : defaultRate;

    var stepIdx = 0;

    function runNextStep() {
      if (currentToken !== planSimToken) return;

      if (stepIdx >= pending.length) {
        state.lastSimulatedStepsSig = fullSig;
        state.activePlanSimSig = null;
        planRatesCache[planCacheKey] = pending.map(function (s) {
          return { stepRate: s._stepRate, postRate: s._postRate, postRateLevel: s._postRateLevel };
        });
        updatePlanSimIndicator(false, pending.length, pending.length);
        refreshPlanPopoverWaitTimes();
        if (state.lastBetterBuilds && state.lastBetterBuilds.length) {
          for (var bi = 0; bi < state.lastBetterBuilds.length; bi++) {
            var bCand = state.lastBetterBuilds[bi];
            if (isSameLockedBuild(state.level, bCand) || (build && buildId(bCand) === buildId(build))) {
              updateBetterBuildRow(bi, bCand, defaultRate);
            }
          }
        }
        return;
      }

      var st = pending[stepIdx];
      st._stepRate = currentRate;

      var affectsCombat = false;
      if (st.kind === "level" || st.kind === "contract") {
        if (!currentCfg.items[st.key]) {
          currentCfg.items[st.key] = { level: st.to || 1, equipped: false };
        } else {
          currentCfg.items[st.key].level = st.to || (st.from != null ? st.from + 1 : 1);
        }
        if (st.after && st.after.to) {
          currentCfg.items[st.key].level = Math.max(currentCfg.items[st.key].level, st.after.to);
        }
        if (currentCfg.equipped.indexOf(st.key) !== -1 || (currentCfg.items[st.key] && currentCfg.items[st.key].equipped)) {
          affectsCombat = true;
        }
      } else if (st.kind === "ring") {
        var rNext = st.to || (st.from != null ? st.from + 1 : (currentCfg.ring.level + 1));
        currentCfg.ring.level = rNext;
        currentCfg.rings.level = rNext;
        affectsCombat = true;
      } else if (st.kind === "ringUnlock") {
        currentCfg.oneTimers.The_Ring = true;
        affectsCombat = true;
      }

      if (affectsCombat) {
        var cfgKey = getStepCfgKey(fightLevel, currentCfg);
        if (stepSimCache[cfgKey] != null && stepSimCache[cfgKey] > 0) {
          currentRate = stepSimCache[cfgKey];
        } else {
          try {
            var simRes = SASimEngine.runBatch(currentCfg, { kills: 9001, maxKills: 9001, maxFights: 9001, fights: 9001, ticks: 1000000 });
            if (simRes && simRes.dustPs > 0) {
              currentRate = simRes.dustPs;
              stepSimCache[cfgKey] = currentRate;
            }
          } catch (err) {
            console.warn("Step sim error:", err);
          }
        }
      }

      st._postRate = currentRate;
      st._postRateLevel = fightLevel;
      stepIdx++;

      updatePlanSimIndicator(true, stepIdx, pending.length);
      refreshPlanPopoverWaitTimes();

      setTimeout(runNextStep, 0);
    }

    setTimeout(runNextStep, 0);
  }

  function calculateBuildReturnTime(b, bSteps, bWInfo, curRate) {
    var cur = (typeof curRate === "number" && curRate >= 0) ? curRate : (state.lastBetterCur != null ? state.lastBetterCur : 0);
    var simRate = b._simDustPs;
    if (!simRate && bSteps && bSteps.length) {
      for (var si = bSteps.length - 1; si >= 0; si--) {
        var rLvl = bSteps[si]._postRateLevel;
        if (bSteps[si]._postRate && bSteps[si]._postRate > 0 && (rLvl == null || Number(rLvl) === Number(state.level))) {
          simRate = bSteps[si]._postRate;
          break;
        }
      }
    }
    var effectiveDust = (simRate && simRate > 0) ? simRate : (b.d || 0);
    var gain = effectiveDust - cur;
    var totalCost = (bWInfo.totalCostDust || 0) + ((bWInfo.totalCostShards || 0) * SHARD_DUST);
    var effRate = (gain > 0) ? gain : (effectiveDust > 0 ? effectiveDust : 0);
    var retSec = (totalCost === 0)
      ? 0
      : (effRate > 0 ? (totalCost / effRate) : Infinity);

    var retStr = (totalCost === 0 || retSec === 0)
      ? "0s"
      : (isFinite(retSec) ? fmtTime(retSec) : "Never");

    return {
      totalCost: totalCost,
      effectiveDust: effectiveDust,
      gain: gain,
      retSec: retSec,
      retStr: retStr,
      isSimmed: !!(b._isSimulated || (bSteps && bSteps.some(function (s) { return s._stepRate > 0; })))
    };
  }

  function updateBetterBuildRow(idx, b, defaultRate) {
    if (!b) return;
    var cell = document.querySelector("#better .bb-earn-return[data-build-idx='" + idx + "']");
    if (!cell) return;
    var bnk = bank();
    var bSteps = b._planSteps || applyStepOrder(state.level, b, expandSteps(gapSteps(b)));
    var bWInfo = calculateStepWaitTimes(bSteps, bnk, defaultRate);
    var bEarn = bWInfo.totalWait;
    var curCmp = (state.lastBetterCur != null) ? state.lastBetterCur : defaultRate;
    var retInfo = calculateBuildReturnTime(b, bSteps, bWInfo, curCmp);
    var bBlockedContracts = bSteps.filter(function (s) { return s.kind === "contract" && !s.reach; });
    var bBlocked = bBlockedContracts.length > 0;
    var bBlockedZone = 0;
    if (bBlocked) {
      bBlockedContracts.forEach(function (s) { if (s.zone && s.zone > bBlockedZone) bBlockedZone = s.zone; });
      if (!bBlockedZone && b.c) bBlockedZone = b.c;
    }
    var bEarnStr = (bBlocked && bEarn === 0) ? ("Z" + (bBlockedZone || "?")) : (bEarn === 0 ? "ready" : fmtTime(bEarn));
    var bRetStr = retInfo.retStr;
    var bEarnReturnVal = bEarnStr + "/" + bRetStr;
    var isSimmed = retInfo.isSimmed;
    var bEarnDurStr = fmtDuration(bEarn);
    var bEarnReturnTip = bBlocked
      ? ("Requires unreached Zone " + (bBlockedZone || "?") + " contract (highest zone: " + (state.radon + 1) + ") · Dust wait: " + bEarnStr + " (" + bEarnDurStr + ") · Return: " + bRetStr)
      : ("Earn: " + bEarnStr + " (" + bEarnDurStr + " remaining wait" + (isSimmed ? " · Sim 9001" : "") + ") · Return: " + bRetStr + " (payback: cost / " + (retInfo.gain > 0 ? "Δ d/s" : "plan d/s") + ")");
    if (cell.textContent !== bEarnReturnVal) cell.textContent = bEarnReturnVal;
    if (cell.title !== bEarnReturnTip) cell.title = bEarnReturnTip;
  }

  function triggerBetterBuildsSimulation(list, defaultRate) {
    if (typeof SASimEngine === "undefined" || !state.save || !list || !list.length) return;
    if (state.activePlanSimSig) return; // Prioritize active plan simulation, avoid competing tasks

    var fightLevel = (state.ab && (state.ab.enemyLevel || state.ab.maxEnemyLevel)) || state.level || 1;
    var startRate = (state.simulatedDustPs && state.simulatedDustPs > 0) ? state.simulatedDustPs : defaultRate;

    var buildQueue = [];
    for (var bi = 0; bi < list.length; bi++) {
      var b = list[bi];
      var steps = b._planSteps || applyStepOrder(state.level, b, expandSteps(gapSteps(b)));
      b._planSteps = steps;
      var pending = steps.filter(function (s) { return !s.completed; });
      if (pending.length > 0) {
        var stepsSig = pending.map(function (s) { return stepKey(s) + "@" + (s.to || 1); }).join(";");
        var bId = buildId(b);
        var planCacheKey = fightLevel + "|" + bId + "|" + stepsSig;
        if (planRatesCache[planCacheKey]) {
          var cached = planRatesCache[planCacheKey];
          for (var ci = 0; ci < pending.length; ci++) {
            if (cached[ci]) {
              pending[ci]._stepRate = cached[ci].stepRate;
              pending[ci]._postRate = cached[ci].postRate;
              pending[ci]._postRateLevel = cached[ci].postRateLevel;
            }
          }
          b._lastSimSig = planCacheKey;
          b._isSimulated = true;
          updateBetterBuildRow(bi, b, defaultRate);
        } else if (b._lastSimSig !== planCacheKey) {
          buildQueue.push({ build: b, idx: bi, steps: steps, pending: pending, cacheKey: planCacheKey });
        }
      }
    }

    if (!buildQueue.length) return;
    if (state.isBetterSimRunning) return; // Keep running current background queue; do not spawn duplicates

    state.isBetterSimRunning = true;
    var baseCfg = SASimEngine.normalizeConfig({ save: state.save, level: fightLevel });
    var qIdx = 0;
    var pIdx = 0;
    var currentCfg = JSON.parse(JSON.stringify(baseCfg));
    var currentRate = startRate;

    function runQueueStep() {
      if (state.activePlanSimSig) {
        state.isBetterSimRunning = false;
        return;
      }
      if (qIdx >= buildQueue.length) {
        state.isBetterSimRunning = false;
        return;
      }

      var curItem = buildQueue[qIdx];
      if (pIdx === 0) {
        currentCfg = JSON.parse(JSON.stringify(baseCfg));
        currentRate = startRate;
      }

      if (pIdx >= curItem.pending.length) {
        curItem.build._lastSimSig = curItem.cacheKey;
        curItem.build._isSimulated = true;
        planRatesCache[curItem.cacheKey] = curItem.pending.map(function (s) {
          return { stepRate: s._stepRate, postRate: s._postRate, postRateLevel: s._postRateLevel };
        });
        updateBetterBuildRow(curItem.idx, curItem.build, defaultRate);
        qIdx++;
        pIdx = 0;
        setTimeout(runQueueStep, 0);
        return;
      }

      var st = curItem.pending[pIdx];
      st._stepRate = currentRate;

      var affectsCombat = false;
      if (st.kind === "level" || st.kind === "contract") {
        if (!currentCfg.items[st.key]) {
          currentCfg.items[st.key] = { level: st.to || 1, equipped: false };
        } else {
          currentCfg.items[st.key].level = st.to || (st.from != null ? st.from + 1 : 1);
        }
        if (st.after && st.after.to) {
          currentCfg.items[st.key].level = Math.max(currentCfg.items[st.key].level, st.after.to);
        }
        if (currentCfg.equipped.indexOf(st.key) !== -1 || (currentCfg.items[st.key] && currentCfg.items[st.key].equipped)) {
          affectsCombat = true;
        }
      } else if (st.kind === "ring") {
        var rNext = st.to || (st.from != null ? st.from + 1 : (currentCfg.ring.level + 1));
        currentCfg.ring.level = rNext;
        currentCfg.rings.level = rNext;
        affectsCombat = true;
      } else if (st.kind === "ringUnlock") {
        currentCfg.oneTimers.The_Ring = true;
        affectsCombat = true;
      }

      if (affectsCombat) {
        var cfgKey = getStepCfgKey(fightLevel, currentCfg);
        if (stepSimCache[cfgKey] != null && stepSimCache[cfgKey] > 0) {
          currentRate = stepSimCache[cfgKey];
        } else {
          try {
            var simRes = SASimEngine.runBatch(currentCfg, { kills: 9001, maxKills: 9001, maxFights: 9001, fights: 9001, ticks: 1000000 });
            if (simRes && simRes.dustPs > 0) {
              currentRate = simRes.dustPs;
              stepSimCache[cfgKey] = currentRate;
            }
          } catch (err) {
            console.warn("Build list step sim error:", err);
          }
        }
      }

      st._postRate = currentRate;
      st._postRateLevel = fightLevel;
      pIdx++;

      updateBetterBuildRow(curItem.idx, curItem.build, defaultRate);
      setTimeout(runQueueStep, 0);
    }

    setTimeout(runQueueStep, 0);
  }

  function stepFullCost(st) {
    var c = st.cost || 0;
    if (st.after && (!st.after.shards === !st.shards)) {
      c += st.after.cost || 0;
    }
    return c;
  }

  function renderStepPopover(pick, rate, bnk) {
    var rawSteps = pick.allSteps && pick.allSteps.length ? pick.allSteps : [pick.step];
    var doneList = rawSteps.filter(function (s) { return s && s.completed; });
    var pendingList = rawSteps.filter(function (s) { return !s || !s.completed; });
    var steps = doneList.concat(pendingList);
    var completedCount = doneList.length;
    var totalSteps = pick.totalSteps || steps.length;
    var pendingSteps = pendingList;
    var activeIdx = pick.activeIndex != null ? pick.activeIndex : completedCount;
    var canDrag = pendingSteps.length > 1;
    var isCustom = canDrag && hasCustomStepOrder(state.level, pick.build);

    var calc = calculateStepWaitTimes(steps, bnk, rate);

    var itemsHtml = steps.map(function (st, idx) {
      if (st.completed) {
        var doneSub = "Step " + (idx + 1) + " of " + totalSteps + " · Completed";
        return "<li class='sp-item completed' data-step-idx='" + idx + "' data-step-key='" + esc(stepKey(st)) + "'>" +
          "<span class='sp-num done' title='Completed'>" + ICONS.check + "</span>" +
          "<div class='sp-main'>" +
          "<div class='sp-label " + stepClass(st) + "'>" + iconForStep(st) + "<span>" + esc(stepLabel(st)) + "</span></div>" +
          "<div class='sp-sub'>" + esc(doneSub) + "</div>" +
          "</div>" +
          "<div class='sp-right'>" +
          "<span class='sp-cost' style='color:var(--buy);'>Done</span>" +
          "<span class='sp-time ready'>Completed</span>" +
          "</div>" +
          "</li>";
      }
      var isAct = idx === activeIdx;
      var totalStepCost = stepFullCost(st);
      var w = st.estWait != null ? st.estWait : 0;
      var costStr = totalStepCost ? (prettify(totalStepCost) + (st.shards ? " shards" : " dust")) : "—";
      var timeStr = w === 0 ? "Ready now" : (w != null && isFinite(w) ? fmtTime(w) : "—");
      var subLines = [];
      if (st.after && st.after.to > 1) {
        subLines.push("+ Lv 1 → " + st.after.to + " (" + prettify(st.after.cost || 0) + (st.after.shards ? " shards" : " dust") + ")");
      }
      if (isAct) {
        subLines.push("Current recommended step");
      } else {
        subLines.push("Step " + (idx + 1) + " of " + totalSteps);
      }
      var timeTip = "";
      if (st._stepRate && isFinite(st._stepRate) && Math.abs(st._stepRate - rate) > 1e6) {
        subLines.push("Sim: " + prettify(st._stepRate) + "/s");
        timeTip = " title='Estimated wait: " + esc(timeStr) + " @ " + prettify(st._stepRate) + "/s simulated earn rate (9001 fights)'";
      }
      return "<li class='sp-item" + (isAct ? " active" : "") + "'" +
        (canDrag ? " draggable='true' data-step-idx='" + idx + "' data-step-key='" + esc(stepKey(st)) + "' title='Drag or touch to reorder step priority'" : "") + ">" +
        (canDrag ? "<span class='sp-grip'>" + ICONS.grip + "</span>" : "") +
        "<span class='sp-num'>" + (idx + 1) + "</span>" +
        "<div class='sp-main'>" +
        "<div class='sp-label " + stepClass(st) + "'>" + iconForStep(st) + "<span>" + esc(stepLabel(st)) + "</span></div>" +
        (subLines.length ? "<div class='sp-sub'>" + esc(subLines.join(" · ")) + "</div>" : "") +
        "</div>" +
        "<div class='sp-right'>" +
        "<span class='sp-cost'>" + esc(costStr) + "</span>" +
        "<span class='sp-time" + (w === 0 ? " ready" : "") + "'" + timeTip + ">" + esc(timeStr) + "</span>" +
        "</div>" +
        "</li>";
    }).join("");

    var costParts = [];
    if (calc.totalCostDust > 0) costParts.push(prettify(calc.totalCostDust) + " dust");
    if (calc.totalCostShards > 0) costParts.push(prettify(calc.totalCostShards) + " shards");
    var totalCostLabel = costParts.length ? costParts.join(" + ") : "0 dust";
    var totalWaitStr = calc.totalWait === 0 ? "Ready now" : fmtTime(calc.totalWait);

    var remParts = [];
    if (calc.remDust > 0) remParts.push(prettify(calc.remDust) + " dust");
    if (calc.remShards > 0) remParts.push(prettify(calc.remShards) + " shards");
    var parHtml = "";
    if (calc.totalCostDust > 0 && calc.totalCostShards > 0) {
      var wdStr = calc.cumDustSec === 0 ? "0s" : fmtTime(calc.cumDustSec);
      var wsStr = calc.cumShardSec === 0 ? "0s" : fmtTime(calc.cumShardSec);
      parHtml = "<div class='sp-foot-sub'><span>Need: " + esc(remParts.join(" + ")) + "</span><span>Total: " + esc(wdStr) + " dust ∥ " + esc(wsStr) + " shards</span></div>";
    } else if ((calc.totalCostDust > 0 && (bnk.dust || 0) > 0) || (calc.totalCostShards > 0 && (bnk.shards || 0) > 0)) {
      parHtml = "<div class='sp-foot-sub'><span>Need after bank: " + esc(remParts.join(" + ")) + "</span><span></span></div>";
    }

    var titleTxt;
    if (steps.length === 1) {
      titleTxt = "Single-Step Unlock";
    } else if (completedCount > 0 && completedCount === totalSteps) {
      titleTxt = "Unlock Roadmap · All " + totalSteps + " Steps Completed!";
    } else if (completedCount > 0) {
      titleTxt = "Unlock Roadmap · Step " + (completedCount + 1) + " of " + totalSteps + " (" + completedCount + " completed)";
    } else {
      titleTxt = "Unlock Roadmap · " + totalSteps + " Steps";
    }
    var hintHtml = canDrag ? (isCustom ? "<button type='button' class='sp-reset-btn' id='spResetOrderBtn' title='Restore default step order'>Reset order</button>" : "<span class='sp-hint'>(drag to reorder)</span>") : "";
    var simBadgeHtml = "<span class='sp-sim-status" + (state.planSimInProgress ? " simulating" : "") + "' id='spSimStatus'" +
      (!state.lastSimulatedStepsSig && !state.planSimInProgress ? " style='display:none;'" : "") + ">" +
      (state.planSimInProgress ? "⚡ Simulating 9001x..." : (state.lastSimulatedStepsSig ? "⚡ 9001x" : "")) +
      "</span>";

    return "<div class='step-popover' role='tooltip'>" +
      "<div class='sp-head'>" +
      "<span class='sp-title'>" + ICONS.clock + "<span>" + esc(titleTxt) + "</span>" + hintHtml + "</span>" +
      "<div class='sp-head-right'>" + simBadgeHtml + "<span class='sp-target'>+" + prettify(pick.gain) + "/s → " + esc(pick.build.n) + "</span></div>" +
      "</div>" +
      "<ol class='sp-list' id='spStepList'>" + itemsHtml + "</ol>" +
      "<div class='sp-foot'>" +
      "<span>Total: <b>" + esc(totalCostLabel) + "</b></span>" +
      "<span>Est. wait: <b id='spTotalWait'>" + esc(totalWaitStr) + "</b></span>" +
      parHtml +
      "</div>" +
      "</div>";
  }

  function bindPlanDrag(pick, rec, rate) {
    var wrap = $("stepPillWrap");
    var listEl = $("spStepList");
    var resetBtn = $("spResetOrderBtn");

    if (wrap) {
      wrap.addEventListener("mouseleave", function () {
        if (state.keepPopoverOpen) {
          state.keepPopoverOpen = false;
          wrap.classList.remove("open");
        }
      });
      // Toggle popover on mobile click/tap on pill
      var pill = wrap.querySelector ? wrap.querySelector(".pill") : null;
      if (pill) {
        pill.addEventListener("click", function (ev) {
          ev.preventDefault();
          ev.stopPropagation();
          var isOpen = wrap.classList.contains("open");
          if (isOpen) {
            wrap.classList.remove("open");
            state.keepPopoverOpen = false;
          } else {
            wrap.classList.add("open");
            state.keepPopoverOpen = true;
          }
        });
      }
      if (!wrap._docClickBound) {
        wrap._docClickBound = true;
        document.addEventListener("click", function (ev) {
          var w = $("stepPillWrap");
          if (w && w.classList.contains("open") && !w.contains(ev.target)) {
            w.classList.remove("open");
            state.keepPopoverOpen = false;
          }
        });
      }
    }

    if (resetBtn) {
      resetBtn.addEventListener("click", function (ev) {
        ev.preventDefault();
        ev.stopPropagation();
        state.lastSimulatedStepsSig = null;
        resetStepOrder(state.level, pick.build);
        if (state.lockedPlans && state.lockedPlans[String(state.level)]) {
          var entry = state.lockedPlans[String(state.level)];
          if (typeof entry === "object" && entry.steps) {
            var done = entry.steps.filter(function (s) { return s.completed; });
            var pending = entry.steps.filter(function (s) { return !s.completed; });
            var sortedPending = sortSmartSteps(pick.build, pending);
            entry.steps = done.concat(sortedPending);
            try { localStorage.setItem("saPlanner.lockedPlans", JSON.stringify(state.lockedPlans)); } catch (e) {}
          }
        }
        state.keepPopoverOpen = true;
        renderChips(rec, rate);
        renderNext(rec, rate);
      });
    }

    if (!listEl || !pick.allSteps || pick.allSteps.length <= 1) return;
    var items = Array.prototype.slice.call(listEl.querySelectorAll(".sp-item[draggable='true']"));
    var dragFromIdx = null;

    function commitReorder(fromIdx, toIdx) {
      if (fromIdx == null || toIdx == null || fromIdx === toIdx) return;
      var stepsCopy = (pick.allSteps || steps).slice();
      var moved = stepsCopy.splice(fromIdx, 1)[0];
      stepsCopy.splice(toIdx, 0, moved);
      var done = stepsCopy.filter(function (s) { return s.completed; });
      var pending = stepsCopy.filter(function (s) { return !s.completed; });
      stepsCopy = done.concat(pending);
      var keys = stepsCopy.map(stepKey);
      state.lastSimulatedStepsSig = null;
      saveStepOrder(state.level, pick.build, keys);
      if (state.lockedPlans && state.lockedPlans[String(state.level)]) {
        var entry = state.lockedPlans[String(state.level)];
        if (typeof entry === "object") {
          entry.steps = stepsCopy;
          try { localStorage.setItem("saPlanner.lockedPlans", JSON.stringify(state.lockedPlans)); } catch (e) {}
        }
      }
      state.keepPopoverOpen = true;
      renderChips(rec, rate);
      renderNext(rec, rate);
    }

    function closestItemIndex(clientY) {
      var bestIdx = null;
      var bestDist = Infinity;
      items.forEach(function (el) {
        var r = el.getBoundingClientRect();
        var mid = r.top + r.height / 2;
        var d = Math.abs(clientY - mid);
        if (d < bestDist) {
          bestDist = d;
          bestIdx = Number(el.getAttribute("data-step-idx"));
        }
      });
      return bestIdx;
    }

    // --- Desktop HTML5 Drag & Drop ---
    items.forEach(function (el) {
      el.addEventListener("dragstart", function (ev) {
        dragFromIdx = Number(el.getAttribute("data-step-idx"));
        el.classList.add("dragging");
        if (wrap) wrap.classList.add("open");
        if (ev.dataTransfer) {
          ev.dataTransfer.effectAllowed = "move";
          try { ev.dataTransfer.setData("text/plain", String(dragFromIdx)); } catch (e) {}
        }
      });
      el.addEventListener("dragend", function () {
        el.classList.remove("dragging");
        items.forEach(function (it) { it.classList.remove("drag-over"); });
      });
    });

    listEl.addEventListener("dragover", function (ev) {
      if (dragFromIdx == null) return;
      ev.preventDefault();
      if (ev.dataTransfer) ev.dataTransfer.dropEffect = "move";
      var overIdx = closestItemIndex(ev.clientY);
      items.forEach(function (it) {
        var idx = Number(it.getAttribute("data-step-idx"));
        it.classList.toggle("drag-over", idx === overIdx && idx !== dragFromIdx);
      });
    });

    listEl.addEventListener("dragleave", function (ev) {
      if (!listEl.contains(ev.relatedTarget)) {
        items.forEach(function (it) { it.classList.remove("drag-over"); });
      }
    });

    listEl.addEventListener("drop", function (ev) {
      if (dragFromIdx == null) return;
      ev.preventDefault();
      var toIdx = closestItemIndex(ev.clientY);
      items.forEach(function (it) { it.classList.remove("drag-over"); });
      commitReorder(dragFromIdx, toIdx);
    });

    // --- Touch Drag for Mobile Devices ---
    var touchDragEl = null;
    var touchFromIdx = null;
    var touchOverIdx = null;
    var touchStartY = 0;
    var isTouchDragging = false;
    var itemSlots = [];

    function closestSlotIndex(clientY) {
      if (!itemSlots.length) return closestItemIndex(clientY);
      var bestIdx = null;
      var bestDist = Infinity;
      itemSlots.forEach(function (s) {
        var d = Math.abs(clientY - s.mid);
        if (d < bestDist) {
          bestDist = d;
          bestIdx = s.idx;
        }
      });
      return bestIdx;
    }

    function cleanupTouch() {
      document.removeEventListener("touchmove", onTouchMove);
      document.removeEventListener("touchend", onTouchEnd);
      document.removeEventListener("touchcancel", onTouchCancel);
      if (touchDragEl) {
        touchDragEl.classList.remove("dragging");
        touchDragEl.classList.remove("touch-active");
        touchDragEl.style.transform = "";
        touchDragEl.style.zIndex = "";
        touchDragEl.style.boxShadow = "";
      }
      items.forEach(function (it) { it.classList.remove("drag-over"); });
      touchDragEl = null;
      touchFromIdx = null;
      touchOverIdx = null;
      isTouchDragging = false;
      itemSlots = [];
    }

    function onTouchMove(e) {
      if (!touchDragEl || touchFromIdx == null) return;
      var touch = e.touches[0];
      var dy = touch.clientY - touchStartY;

      if (!isTouchDragging) {
        if (Math.abs(dy) > 3) {
          isTouchDragging = true;
          touchDragEl.classList.add("dragging");
          touchDragEl.classList.add("touch-active");
          if (wrap) wrap.classList.add("open");
        }
      }

      if (isTouchDragging) {
        if (e.cancelable) e.preventDefault();
        touchDragEl.style.transform = "translateY(" + dy + "px)";
        touchDragEl.style.zIndex = "40";
        touchDragEl.style.boxShadow = "0 8px 24px rgba(0, 0, 0, 0.55)";

        var overIdx = closestSlotIndex(touch.clientY);
        touchOverIdx = overIdx;
        items.forEach(function (it) {
          var idx = Number(it.getAttribute("data-step-idx"));
          it.classList.toggle("drag-over", idx === overIdx && idx !== touchFromIdx);
        });
      }
    }

    function onTouchEnd(e) {
      if (!touchDragEl) return;
      var from = touchFromIdx;
      var to = touchOverIdx;
      var wasDragging = isTouchDragging;
      cleanupTouch();
      if (wasDragging && to != null && from != null && to !== from) {
        if (e.cancelable) e.preventDefault();
        commitReorder(from, to);
      }
    }

    function onTouchCancel() {
      cleanupTouch();
    }

    function onTouchStart(e) {
      var itemEl = e.target.closest(".sp-item[draggable='true']");
      if (!itemEl) return;
      if (e.target.closest("button, a, input, select")) return;

      var touch = e.touches[0];
      touchStartY = touch.clientY;
      touchFromIdx = Number(itemEl.getAttribute("data-step-idx"));
      touchOverIdx = touchFromIdx;
      touchDragEl = itemEl;
      isTouchDragging = false;

      // Capture resting midpoints of each draggable slot before any translation
      itemSlots = items.map(function (it) {
        var r = it.getBoundingClientRect();
        return { idx: Number(it.getAttribute("data-step-idx")), mid: r.top + r.height / 2 };
      });

      document.addEventListener("touchmove", onTouchMove, { passive: false });
      document.addEventListener("touchend", onTouchEnd, { passive: false });
      document.addEventListener("touchcancel", onTouchCancel, { passive: false });
    }

    listEl.addEventListener("touchstart", onTouchStart, { passive: true });
  }

  function renderNext(rec, rate) {
    var el = $("nextUpgrade");
    var pick = bestNextUpgrade(rec, rate);
    state._cachedRec = rec;
    state._cachedPick = pick;
    if (!pick) {
      el.innerHTML = rec
        ? "<div class='next'><div class='next-kicker'>" + ICONS.check + " Peak Build Reached</div><p class='empty'>You already match the strongest sheet farm on SA " + state.level + ".</p></div>"
        : "<div class='next'><div class='next-kicker'>" + ICONS.info + " Next Upgrade</div><p class='empty'>No upgrade on this level matches the current filters.</p></div>";
      return;
    }
    if (pick.mode === "completed") {
      var btnUnlock = "<button type='button' class='btn sm accent lock-plan-btn locked' id='lockPlanBtn' title='Advisor is locked to this completed build. Click to unlock and allow advisor to recommend new builds.'>" + ICONS.lock + "<span>Locked (Done)</span></button>";
      var totalDone = pick.totalSteps || (pick.allSteps ? pick.allSteps.length : 1);
      var stepWord = totalDone === 1 ? "Step" : "Steps";
      var donePill = "<span class='pill sm buy'>" + ICONS.check + " " + totalDone + " of " + totalDone + " " + stepWord + " Complete</span>";
      var bnk = bank();
      var badgeHtml = (pick.allSteps && pick.allSteps.length)
        ? "<div class='step-pill-wrap" + (state.keepPopoverOpen ? " open" : "") + "' id='stepPillWrap' tabindex='0'>" + donePill + renderStepPopover(pick, rate, bnk) + "</div>"
        : donePill;

      var cDelta = loadoutDelta(pick.build, state.level, getLevelMode(state.level));
      var cReqMods = formatModList(pick.build.r && pick.build.r.m || []) || "None";
      var cCurMods = formatModList(cDelta ? cDelta.curMods : []) || "None";
      var cAligned = !cDelta || cDelta.aligned;

      var cPresetMatch = findBestPresetMatch(pick.build);
      var cPresetBadge = "";
      if (cPresetMatch) {
        if (cPresetMatch.isExact) {
          cPresetBadge = "<span class='pill sm buy' style='cursor:pointer;' onclick='window.saOpenPresets&&saOpenPresets()' title='Matches in-game " + esc(cPresetMatch.preset.name) + " (" + cPresetMatch.preset.slotTag + "). Click to open presets modal.'>" +
            ICONS.check + " Matches " + esc(cPresetMatch.preset.slotTag) + ' ("' + esc(cPresetMatch.preset.name) + '")</span>';
        } else if (cPresetMatch.isClose) {
          cPresetBadge = "<span class='pill sm accent' style='cursor:pointer;' onclick='window.saOpenPresets&&saOpenPresets()' title='1 swap from in-game " + esc(cPresetMatch.preset.name) + " (" + cPresetMatch.preset.slotTag + "). Click to open presets modal.'>" +
            ICONS.target + " ~ " + esc(cPresetMatch.preset.slotTag) + ' (1 swap)</span>';
        }
      }
      var alignStatusPill = cAligned
        ? ("<span class='pill sm buy'>" + ICONS.check + " Active &amp; Aligned</span>" + (cPresetBadge ? " " + cPresetBadge : ""))
        : ("<span class='pill sm warn-pill'>" + ICONS.warn + " Alignment Needed to Activate</span>" + (cPresetBadge ? " " + cPresetBadge : ""));

      var alignGuideHtml = "";
      if (!cAligned) {
        var cRows = [];
        if (cDelta.ringMismatch) {
          cRows.push("<div class='delta-row'><span class='delta-lbl'>Ring Mod:</span><span class='pill sm warn-pill'>" + ICONS.ring + " " + esc(cCurMods) + " → " + esc(cReqMods) + "</span></div>");
        }
        if (cDelta.toSwap && cDelta.toSwap.length) {
          var cSwPills = cDelta.toSwap.map(function (sw) {
            var toLv = userLevel(sw.to) || 1;
            return "<span class='pill sm accent' title='" + esc(sw.reason || "Change extra limb to optimal choice") + "'>" +
              iconForItem(sw.from) + " " + esc(itemName(sw.from)) + " → " + iconForItem(sw.to) + " " + esc(itemName(sw.to) + " (Lv " + toLv + ")") + "</span>";
          }).join("");
          cRows.push("<div class='delta-row'><span class='delta-lbl'>Extra Limb Swap:</span>" + cSwPills + "</div>");
        }
        if (cDelta.extraLimbOptimal && cDelta.extraLimbOptimal.length) {
          var cOptEq = cDelta.extraLimbOptimal.filter(function (o) { return o.isCurEquipped; });
          if (cOptEq.length) {
            var cOptPills = cOptEq.map(function (o) {
              return "<span class='pill sm buy' title='Equipped item is verified optimal for your extra limb slot'>" +
                iconForItem(o.item) + " " + esc(itemName(o.item)) + " (Optimal)</span>";
            }).join("");
            cRows.push("<div class='delta-row'><span class='delta-lbl'>Extra Limb:</span>" + cOptPills + "</div>");
          }
        }
        if (cDelta.toUnequip.length) {
          var cUnPills = cDelta.toUnequip.map(function (k) {
            return "<span class='pill sm sell'>" + iconForItem(k) + " " + esc(itemName(k)) + "</span>";
          }).join("");
          cRows.push("<div class='delta-row'><span class='delta-lbl'>Unequip:</span>" + cUnPills + "</div>");
        }
        if (cDelta.toEquip.length) {
          var cEqPills = cDelta.toEquip.map(function (k) {
            var lv = pick.build.i && pick.build.i[k] ? (" (Lv " + pick.build.i[k] + ")") : "";
            return "<span class='pill sm buy'>" + iconForItem(k) + " " + esc(itemName(k) + lv) + "</span>";
          }).join("");
          cRows.push("<div class='delta-row'><span class='delta-lbl'>Equip:</span>" + cEqPills + "</div>");
        }
        alignGuideHtml = "<div class='delta-box' style='margin-top:4px;'>" +
          "<div class='delta-title'>" + ICONS.warn + "<span>Put " + esc(pick.build.n) + " in Play · Alignment Steps</span></div>" +
          "<div class='delta-rows'>" + cRows.join("") + "</div>" +
          "</div>";
      } else {
        alignGuideHtml = "<div class='delta-box' style='margin-top:4px;border-color:rgba(45,206,137,0.35);background:color-mix(in srgb,var(--buy) 6%,var(--surface-solid));'>" +
          "<div class='delta-title' style='color:var(--buy);'>" + ICONS.check + "<span>Build In Play · Loadout &amp; Ring Aligned</span></div>" +
          "<div style='color:var(--muted);font-size:12px;'>All required items and Ring mods (" + esc(cReqMods) + ") are equipped and active.</div>" +
          "</div>";
      }

      var whyDone = cAligned
        ? "Your locked target build is unlocked and fully aligned in play (" + totalDone + " of " + totalDone + " " + stepWord.toLowerCase() + " completed). Click <b>Locked (Done)</b> to unlock the advisor for new recommendations."
        : "All upgrades for <b>" + esc(pick.build.n) + "</b> are complete (" + totalDone + " of " + totalDone + " " + stepWord.toLowerCase() + "). Apply the loadout and Ring changes above in Trimps to put this build into play, then click <b>Locked (Done)</b> for new recommendations.";

      el.innerHTML =
        "<div class='next'>" +
        "<div class='next-top'>" +
        "<div class='next-kicker' style='color:var(--buy);'>" + ICONS.check + " Plan Completed</div>" +
        "<div class='next-top-actions'>" + btnUnlock + badgeHtml + "</div>" +
        "</div>" +
        "<div class='next-action-box'>" +
        "<div class='next-head' style='color:var(--buy);'>" + ICONS.check + "<span>" + esc(pick.build.n) + " is fully unlocked!</span></div>" +
        "<div class='next-meta'><span class='pill sm buy'>" + ICONS.check + " Runnable now · " + prettify(pick.build.d) + "/s</span>" + alignStatusPill + "</div>" +
        alignGuideHtml +
        "</div>" +
        "<div class='next-why'>" + whyDone + "</div>" +
        "</div>";
      var btnComp = $("lockPlanBtn");
      if (btnComp) {
        btnComp.addEventListener("click", function () {
          clearLockedBuild(state.level);
          toast("Unlocked advisor for SA " + state.level);
          renderChips(rec, rate);
          renderNext(rec, rate);
          renderBetter(rec, rate);
        });
      }
      bindPlanDrag(pick, rec, rate);
      return;
    }

    var s = pick.step;
    state.lastAdvisorStep = s;
    state.lastPlanSteps = pick.allSteps || [s];
    if (pick.allSteps && pick.allSteps.length > 1) {
      triggerPlanStepSimulation(pick.build, pick.allSteps, rate);
    }
    var bnk = bank();
    var rawCost = s.cost || 0;
    var wait = timeForCost(rawCost, !!s.shards, rate, s.shards ? 0 : bnk.dust, s.shards ? bnk.shards : 0);
    var costTxt = rawCost ? (prettify(rawCost) + (s.shards ? " shards" : " dust")) : "";
    var totalPlanSteps = pick.totalSteps || (pick.allSteps ? pick.allSteps.length : pick.remaining);
    var completedPlanSteps = pick.completedCount || 0;
    var currentStepNum = completedPlanSteps + 1;
    var isCustom = pick.remaining > 1 && hasCustomStepOrder(state.level, pick.build);
    var mode = getLevelMode(state.level);
    var nextSub = $("nextCardSub");
    if (nextSub) {
      nextSub.textContent = (mode === "push")
        ? "Fastest progression step to clear SA " + state.level + " (Clear Priority)"
        : "Highest ROI progression step (Dust Priority)";
    }
    var why;
    if (pick.isLocked) {
      var remWord = pick.remaining === 1 ? "step" : "steps";
      if (completedPlanSteps > 0) {
        why = "Advisor locked to <b>" + esc(pick.build.n) + "</b> (Step " + currentStepNum + " of " + totalPlanSteps + " · " + pick.remaining + " " + remWord + " remaining). Mid-plan recommendations will not switch until you click Unlock.";
      } else {
        why = "Advisor locked to <b>" + esc(pick.build.n) + "</b> (" + pick.remaining + " " + remWord + " remaining). Mid-plan recommendations will not switch until you click Unlock.";
      }
    } else if (pick.mode === "upgradeValue") {
      var kStr = (pick.currentCombatSec && isFinite(pick.currentCombatSec)) ? pick.currentCombatSec.toFixed(1) : "—";
      if (mode === "farm") {
        var curD = (pick.currentDustPs && pick.currentDustPs > 0) ? (prettify(pick.currentDustPs) + "/s") : (prettify(levelCurrentDustPs(state.level, rec)) + "/s");
        var gainNote = (pick.gain && pick.gain > 0) ? (" (+" + prettify(pick.gain) + "/s)") : "";
        why = "Your current loadout earns " + curD + " on SA " + state.level + " (~" + kStr + "s/kill), higher than all recorded runnable sheet builds. The simulator's <b>Upgrade Value</b> analysis recommends <b>" + esc(pick.label || (pick.step && pick.step.key) || "this upgrade") + "</b>" + gainNote + " as the highest-value upgrade.";
      } else {
        var saveNote = (pick.deltaSecPerKill && pick.deltaSecPerKill > 0)
          ? (" (saving ~" + pick.deltaSecPerKill.toFixed(1) + "s per kill)")
          : "";
        why = "Your current loadout clears SA " + state.level + " in ~" + kStr + "s per kill (faster than all recorded runnable sheet builds). The simulator's <b>Upgrade Value</b> analysis recommends <b>" + esc(pick.label || (pick.step && pick.step.key) || "this upgrade") + "</b>" + saveNote + " as the highest-value upgrade to improve combat performance.";
      }
    } else if (pick.mode === "unlocks") {
      why = (mode === "push")
        ? "Largest clear time improvement among single-step upgrades that immediately unlock a faster sheet build (click Lock Plan to latch)."
        : "Highest dust/s gain per cost among single-step upgrades that immediately unlock a stronger sheet build (click Lock Plan to latch).";
    } else if (isCustom) {
      why = "Custom step order active — hover Step 1 of " + pick.remaining + " to drag and reorder your unlock plan.";
    } else {
      why = (mode === "push")
        ? "First step on the fewest-gap path to a faster clear build for SA " + state.level + " (hover Step 1 of " + pick.remaining + " to reorder plan)."
        : "First step on the fewest-gap path to a stronger sheet build (hover Step 1 of " + pick.remaining + " to reorder plan).";
    }

    var nextSimBtnHtml = "<button type='button' class='btn sm ghost sim-plan-btn' id='nextSimPlanBtn' title='Load " + esc(pick.build.n) + " into Combat Simulator'>⚔️ Sim</button>";
    var lockBtnHtml = pick.isLocked
      ? "<button type='button' class='btn sm accent lock-plan-btn locked' id='lockPlanBtn' title='Advisor is locked to " + esc(pick.build.n) + ". Click to unlock and re-enable automatic recommendations.'>" + ICONS.lock + "<span>Locked</span></button>"
      : "<button type='button' class='btn sm ghost lock-plan-btn' id='lockPlanBtn' title='Lock advisor to " + esc(pick.build.n) + " so recommendations do not switch mid-plan.'>" + ICONS.unlock + "<span>Lock Plan</span></button>";

    var kickerHtml = pick.isLocked
      ? "<div class='next-kicker' style='color:var(--accent);'>" + ICONS.target + " Active Plan</div>"
      : (pick.mode === "upgradeValue"
        ? "<div class='next-kicker' style='color:var(--buy);'>" + ICONS.bolt + " " + (mode === "push" ? "Optimal Clear Active" : "Optimal Farm Active") + " · Upgrade Value Recommendation</div>"
        : "<div class='next-kicker'>Recommended Next Action</div>");

    var pillInner = (pick.mode === "upgradeValue")
      ? "<span class='pill sm buy'>" + ICONS.bolt + " #1 Upgrade Value</span>"
      : ((pick.mode === "unlocks" || totalPlanSteps === 1)
        ? "<span class='pill sm buy'>" + ICONS.bolt + " 1-Step Unlock</span>"
        : "<span class='pill sm info'>" + ICONS.clock + " Step " + currentStepNum + " of " + totalPlanSteps + "</span>");
    var badgeHtml = "<div class='step-pill-wrap" + (state.keepPopoverOpen ? " open" : "") + "' id='stepPillWrap' tabindex='0'>" + pillInner + renderStepPopover(pick, rate, bnk) + "</div>";
    var pSum = probeBuildSummary(state.level, pick.build, rate, pick.allSteps);
    var hasBuildPill = !!(pSum && pSum.target);
    var waitMatchesBuild = hasBuildPill && (
      pick.remaining <= 1 ||
      (wait === 0 ? pSum.buildSec === 0 : (wait === pSum.buildSec || (isFinite(wait) && isFinite(pSum.buildSec) && Math.abs(wait - pSum.buildSec) < 60)))
    );

    var waitBadge = "";
    if (!waitMatchesBuild) {
      if (wait === 0) {
        waitBadge = "<span class='pill sm buy' id='advisorWaitBadge' title='Current step is affordable now'>" + ICONS.check + " Ready now</span>";
      } else if (wait != null) {
        var stepPrefix = (hasBuildPill && pick.remaining > 1) ? "Step: " : "";
        waitBadge = "<span class='pill sm' id='advisorWaitBadge' data-step-prefix='" + esc(stepPrefix) + "' title='Wait time for this step: " + fmtDuration(wait) + "'>" + ICONS.clock + " " + stepPrefix + fmtTime(wait) + "</span>";
      }
    }

    var metaLockPill = pick.isLocked ? ("<span class='pill sm accent' title='Advisor locked to this plan'>" + ICONS.lock + " Plan Locked</span>") : "";

    el.innerHTML =
      "<div class='next'>" +
      "<div class='next-top'>" +
      kickerHtml +
      "<div class='next-top-actions'>" + nextSimBtnHtml + lockBtnHtml + badgeHtml + "</div>" +
      "</div>" +
      "<div class='next-action-box'>" +
      "<div class='next-head " + stepClass(s) + "'>" + iconForStep(s) + "<span>" + esc(stepLabel(s)) + "</span></div>" +
      "<div class='next-meta'>" +
      (costTxt ? "<span class='pill sm accent'>" + (s.shards ? ICONS.gem : ICONS.dust) + " " + esc(costTxt) + "</span>" : "") +
      waitBadge +
      metaLockPill +
      (function () {
        if (!pSum || !pSum.target) return "";
        var cleared = isLevelCleared(state.level);
        var advBuildLabel = (pick.mode === "upgradeValue" && pick.label) ? pick.label : pick.build.n;
        var pillTitle = buildPillTitle(advBuildLabel, pSum.buildSec, pSum.buildTimeStr, pSum.clearSec, pSum.clearTimeStr, pSum.totalSec, pSum.totalTimeStr, cleared);
        if (cleared) {
          return "<span class='pill sm' id='advPlanBuildTime' data-build-name='" + esc(advBuildLabel) + "' title='" + esc(pillTitle) + "'>" +
            ICONS.hammer + " Build: " + pSum.buildTimeStr + "</span>";
        }
        var mainPill = "<span class='pill sm' id='advPlanBuildTime' data-build-name='" + esc(advBuildLabel) + "' title='" + esc(pillTitle) + "'>" +
          ICONS.clock + " Build: " + pSum.buildTimeStr + " · Clear: ~" + pSum.clearTimeStr + "</span>";

        if (mode === "push") {
          var curK = getCurrentCombatSecPerKill(state.level);
          if (!curK || !isFinite(curK) || curK <= 0) {
            curK = (state.lastActiveBuild && state.lastActiveBuild.k && isFinite(state.lastActiveBuild.k) && state.lastActiveBuild.k > 0)
              ? state.lastActiveBuild.k
              : (pick.currentCombatSec && isFinite(pick.currentCombatSec) && pick.currentCombatSec > 0)
                ? pick.currentCombatSec
                : (rec && isFinite(buildSecPerKill(rec, state.level)) ? buildSecPerKill(rec, state.level) : 15);
          }
          var targetK = pSum.secPerKill;
          var curClearSec = (state.lastPlanSummary && isFinite(state.lastPlanSummary.clearSec) && state.lastPlanSummary.clearSec > 0)
            ? state.lastPlanSummary.clearSec
            : (pSum.remainingKills * curK);
          var newClearSec = pSum.clearSec;
          var savedSec = (isFinite(curClearSec) && isFinite(newClearSec) && curClearSec > newClearSec)
            ? (curClearSec - newClearSec)
            : 0;
          if (pick.mode === "upgradeValue" && pick.deltaSecPerKill && pick.deltaSecPerKill > 0 && pSum.remainingKills > 0) {
            var simSaved = pSum.remainingKills * pick.deltaSecPerKill;
            if (simSaved > savedSec) {
              savedSec = simSaved;
            }
          }
          if (savedSec > 0 && isFinite(savedSec)) {
            var savedTitle = "Time saved to clear SA " + state.level + " if this upgrade is performed\n" +
              "• Current estimated clear time: ~" + fmtDuration(curClearSec) + " (~" + (isFinite(curK) ? curK.toFixed(1) : "15.0") + "s/kill)\n" +
              "• Estimated clear time after upgrade: ~" + fmtDuration(newClearSec) + " (~" + targetK.toFixed(1) + "s/kill)\n" +
              "• Time saved (Δ): ~" + fmtDuration(savedSec) + " across " + prettify(pSum.remainingKills) + " remaining kills";
            mainPill += "<span class='pill sm buy' id='advPlanSavedTime' title='" + esc(savedTitle) + "'>" +
              ICONS.bolt + " Saved: ~" + fmtDuration(savedSec) + "</span>";
          }
        }
        return mainPill;
      })() +
      (function () {
        var pMatch = findBestPresetMatch(pick.build);
        if (!pMatch) return "";
        if (pMatch.isExact) {
          return "<span class='pill sm buy' style='cursor:pointer;' onclick='window.saOpenPresets&&saOpenPresets()' title='Target build " + esc(pick.build.n) + " matches in-game " + esc(pMatch.preset.name) + " (" + pMatch.preset.slotTag + "). Click to open presets modal.'>" +
            ICONS.check + " Matches " + esc(pMatch.preset.slotTag) + ' ("' + esc(pMatch.preset.name) + '")</span>';
        } else if (pMatch.isClose) {
          return "<span class='pill sm accent' style='cursor:pointer;' onclick='window.saOpenPresets&&saOpenPresets()' title='Target build " + esc(pick.build.n) + " is 1 swap from in-game " + esc(pMatch.preset.name) + " (" + pMatch.preset.slotTag + "). Click to open presets modal.'>" +
            ICONS.target + " ~ " + esc(pMatch.preset.slotTag) + ' (1 swap)</span>';
        }
        return "";
      })() +
      (function () {
        if (pick.mode === "upgradeValue") {
          var gainTxt = (pick.gain && pick.gain > 0) ? ("+" + prettify(pick.gain) + "/s Δ Dust") : "";
          var payTxt = (pick.paybackSec && isFinite(pick.paybackSec) && pick.paybackSec > 0) ? ("Payback ~" + fmtTime(pick.paybackSec)) : "";
          return (gainTxt ? ("<span class='pill sm buy'>" + ICONS.dust + " " + esc(gainTxt) + "</span>") : "") +
            (payTxt ? ("<span class='pill sm info'>" + ICONS.clock + " " + esc(payTxt) + "</span>") : "") +
            "<span class='pill sm accent' style='cursor:pointer;' onclick=\"window.saOpenSimTab&&saOpenSimTab('sensitivity')\" title='Open Simulator Upgrade Value analysis'>⚔️ View in Simulator</span>";
        }
        if (mode === "push") {
          var targetK = buildSecPerKill(pick.build, state.level);
          var recK = rec ? buildSecPerKill(rec, state.level) : null;
          var kDiff = (recK && recK > targetK) ? (" (-" + (recK - targetK).toFixed(1) + "s)") : "";
          return "<span class='pill sm buy'>" + ICONS.clock + " ~" + targetK.toFixed(1) + "s kills" + kDiff + " → " + esc(pick.build.n) + "</span>";
        }
        return "<span class='pill sm buy'>+" + prettify(pick.gain) + "/s → " + esc(pick.build.n) + " (" + prettify(pick.build.d) + "/s)</span>";
      })() +
      "</div>" +
      "</div>" +
      "<div class='next-why'>" + why + "</div>" +
      "</div>";

    var lockBtn = $("lockPlanBtn");
    if (lockBtn) {
      lockBtn.addEventListener("click", function () {
        toggleLockBuild(pick.build);
      });
    }
    var nSimBtn = $("nextSimPlanBtn");
    if (nSimBtn) {
      nSimBtn.addEventListener("click", function () {
        openSimModal(pick.build);
      });
    }

    bindPlanDrag(pick, rec, rate);
  }

  function cardTooltip(title, lines) {
    return "<div class='tip'><strong>" + esc(title) + "</strong>" + lines.map(function (l) { return "<div>" + esc(l) + "</div>"; }).join("") + "</div>";
  }

  function activeTargetBuild(rec, rate) {
    if (state.lastActiveBuild && state.lastActiveBuild.i) return state.lastActiveBuild;
    var lockedEntry = state.lockedPlans && state.lockedPlans[String(state.level)];
    var lockedTarget = lockedEntry ? findLockedTarget(state.level, lockedEntry) : null;
    if (lockedTarget && lockedTarget.i) return lockedTarget;
    if (rec && rec.i) return rec;
    var summary = probeBuildSummary(state.level, null, rate);
    return (summary && summary.target) ? summary.target : null;
  }

  function renderBoard(rec, rate) {
    var build = activeTargetBuild(rec, rate);
    var recItems = build ? (build.i || {}) : {};
    var hide = !!state.filters.hideUnused && !!build;
    var keys = {};
    if (hide) {
      Object.keys(recItems).forEach(function (k) { keys[k] = true; });
    } else {
      SA_DATA.itemOrder.forEach(function (k) { if (userOwns(k) || recItems[k]) keys[k] = true; });
      Object.keys(recItems).forEach(function (k) { keys[k] = true; });
    }
    var order = SA_DATA.itemOrder.filter(function (k) { return keys[k]; });
    var bnk = bank();
    var html = "";
    var readyCount = 0, warnCount = 0, lockCount = 0;

    order.forEach(function (k) {
      var owned = userOwns(k);
      var need = recItems[k] || 0;
      var lv = owned ? userLevel(k) : 0;
      if (!owned) lockCount++;
      else if (need && lv < need) warnCount++;
      else if (need && lv >= need) readyCount++;
      html += itemCard(k, recItems[k], rate, bnk);
    });
    if (!hide || (build && build.r)) html += ringCard(build && build.r, rate, bnk);
    if (build && build.l && build.l > maxSlots()) html += limbsCard(build.l, rate, bnk);
    $("board").innerHTML = html;
    if ($("boardMeta")) {
      $("boardMeta").textContent = "· " + order.length + " items shown" + (build ? (" (" + readyCount + " build-ready)") : "");
    }
  }

  function itemCard(key, recLv, rate, bnk) {
    var cat = itemCat(key);
    var owned = userOwns(key);
    var uItem = userItem(key);
    var equipped = !!(uItem && uItem.equipped);
    var lv = owned ? userLevel(key) : 0;
    var need = recLv || 0;
    var cls = "card";
    if (need) cls += " rec";
    if (!owned) cls += " locked";
    else if (need && lv < need) cls += " warn";
    else if (need && lv >= need && !equipped) cls += " warn";
    else if (need && lv >= need) cls += " ok";

    var badge = "";
    if (!owned) badge = "<span class='card-badge lock'>Locked</span>";
    else if (need && lv < need) badge = "<span class='card-badge warn'>Need Lv " + need + "</span>";
    else if (need && lv >= need && !equipped) badge = "<span class='card-badge warn'>Equip Item</span>";
    else if (need && lv >= need) badge = "<span class='card-badge ok'>Equipped</span>";
    else if (equipped) badge = "<span class='card-badge sell'>Unequip</span>";

    var lines = [];
    var topLv = "";
    var meta = "";
    var isMaxLevel = !cat || cat.noUp || (cat.max && lv >= cat.max);
    var isSh = isShards(key);
    var curFunds = isSh ? (bnk.shards || 0) : (bnk.dust || 0);
    var nextCost = 0;
    var pct = 0;
    var isReady = false;
    var barTip = "";

    if (!owned) {
      pct = 0;
      barTip = "Locked contract";
      topLv = "<div class='lv'><span>Locked</span>" + (cat && cat.zone ? "<span class='muted' style='font-size:12px;'>Z" + cat.zone + "</span>" : "") + "</div>";
      if (cat && cat.zone) {
        var active = state.ab.activeContract === key;
        var reach = (state.radon + 1) >= cat.zone;
        var cPrice = contractPrice(key);
        var tUnlock = timeForCost(cPrice, isSh, rate, bnk.dust, bnk.shards);
        var waitStr = !reach ? ("Z" + cat.zone) : fmtTime(tUnlock);
        meta += "<div class='cost'><span>Contract</span><span>" + prettify(cPrice) + " " + currencyName(key) + "</span></div>";
        meta += "<div class='time'><span>Wait</span><span class='board-wait'" + (!reach ? " data-unreached-zone='" + cat.zone + "'" : "") + " data-item-cost='" + cPrice + "' data-item-shards='" + (isSh ? "1" : "0") + "'>" + waitStr + "</span></div>";
        if (active) meta += "<div class='flag'>" + ICONS.info + " Accepted, not finished</div>";
        if (!reach) meta += "<div class='flag bad'>" + ICONS.warn + " Void map unreachable</div>";
        lines.push("Complete the Z" + cat.zone + " contract to unlock " + itemName(key) + ".");
      }
    } else {
      topLv = "<div class='lv'><span>Lv " + lv + (need && need > lv ? " <span class='gap'>→ " + need + "</span>" : "") + "</span>" +
        (equipped ? "<span class='muted' style='font-size:11px;font-weight:500;'>Equipped</span>" : (need ? "<span class='gap' style='font-size:11px;font-weight:600;'>Not equipped</span>" : "")) + "</div>";
      if (!isMaxLevel) {
        var next = upgradeCostAt(key, lv);
        nextCost = next;
        var tNext = timeForCost(next, isSh, rate, bnk.dust, bnk.shards);
        isReady = (next <= 0) || (curFunds >= next);
        pct = next > 0 ? Math.min(100, Math.max(0, (curFunds / next) * 100)) : 100;
        if (isReady) {
          barTip = "Ready to upgrade to Lv " + (lv + 1) + "! (" + prettify(next) + " " + currencyName(key) + ")";
        } else {
          barTip = "Progress to Lv " + (lv + 1) + ": " + pct.toFixed(1) + "% (" + prettify(curFunds) + " / " + prettify(next) + " " + currencyName(key) + ")";
        }
        meta += "<div class='cost'><span>Next (Lv " + (lv + 1) + ")</span><span>" + prettify(next) + " " + currencyName(key) + "</span></div>";
        meta += "<div class='time'><span>Wait</span><span class='board-wait' data-item-cost='" + next + "' data-item-shards='" + (isSh ? "1" : "0") + "'>" + fmtTime(tNext) + "</span></div>";
        lines.push("Next upgrade: " + (cat.upgrade || "—") + ".");
        lines.push("Cost " + prettify(next) + " " + currencyName(key) + " (" + fmtTime(tNext) + ").");
        if (need > lv) {
          var gap = upgradeCostRange(key, lv, need);
          var tGap = timeForCost(gap, isSh, rate, bnk.dust, bnk.shards);
          meta += "<div class='gapcost'><span>To Lv " + need + "</span><span>" + prettify(gap) + " · <span class='board-wait' data-item-cost='" + gap + "' data-item-shards='" + (isSh ? "1" : "0") + "'>" + fmtTime(tGap) + "</span></span></div>";
          lines.push("To reach the build: Lv " + lv + " → Lv " + need + ".");
        }
      } else {
        pct = 0;
        barTip = "Max Level — cannot be upgraded";
        meta += "<div class='cost max-level' style='justify-content:center;'><span style='color:var(--muted);font-weight:600;'>Max Level</span></div>";
        lines.push("Max Level (Cannot be upgraded).");
      }
      if (need && !equipped) {
        lines.push("Recommended build uses this item — equip it into your loadout.");
      } else if (!need && equipped) {
        lines.push("Currently equipped, but not used by the recommended build.");
      }
      lines.unshift(describeItem(key, lv));
    }

    return "<article class='" + cls + "' title='" + esc([itemName(key)].concat(lines).join(" — ")) + "'>" +
      "<div>" +
      "<div class='card-head-row'><h3>" + iconForItem(key) + "<span>" + esc(itemName(key)) + "</span></h3>" + badge + "</div>" +
      topLv +
      "<div class='lv-bar' title='" + esc(barTip) + "'><span class='board-bar" + (isReady ? " ready" : "") + "' data-item-cost='" + nextCost + "' data-item-shards='" + (isSh ? "1" : "0") + "' data-next-lv='" + (lv + 1) + "' data-currency='" + currencyName(key) + "' data-max-level='" + (isMaxLevel ? "1" : "0") + "' style='width:" + (isReady ? 100 : pct.toFixed(1)) + "%'></span></div>" +
      "</div>" +
      "<div class='card-meta-rows'>" + meta + "</div>" +
      cardTooltip(itemName(key), lines) +
      "</article>";
  }

  function ringCard(recRing, rate, bnk) {
    var cls = "card ring";
    var need = recRing ? recRing.lv : 0;
    var reqMods = (recRing && recRing.m) ? recRing.m : [];
    var curMods = hasRing() ? ringMods() : [];
    var missingMods = reqMods.filter(function (m) { return curMods.indexOf(m) === -1; });
    var modMismatch = missingMods.length > 0;
    var lines = [];
    var badge = "";
    var topLv = "";
    var meta = "";
    var pct = 100;

    if (!hasRing()) {
      cls += " locked";
      pct = 0;
      badge = "<span class='card-badge lock'>Locked</span>";
      var cost = oneTimerRingCost();
      var t = timeForCost(cost, true, rate, bnk.dust, bnk.shards);
      topLv = "<div class='lv'><span>Locked</span><span class='muted' style='font-size:11px;'>One-timer</span></div>";
      meta = "<div class='cost'><span>Unlock</span><span>" + prettify(cost) + " shards</span></div>" +
        "<div class='time'><span>Wait</span><span class='board-wait' data-item-cost='" + cost + "' data-item-shards='1'>" + fmtTime(t) + "</span></div>";
      lines.push("Unlock The Ring (one-timer, not a contract).");
      lines.push(nextRingSlotNote(1));
    } else {
      var lv = ringLevel();
      if (need && lv < need) {
        cls += " warn rec";
        badge = "<span class='card-badge warn'>Need Lv " + need + "</span>";
        pct = Math.min(100, Math.round((lv / need) * 100));
      } else if (need && modMismatch) {
        cls += " warn rec";
        badge = "<span class='card-badge warn'>Swap Mod</span>";
      } else if (need) {
        cls += " ok rec";
        badge = "<span class='card-badge ok'>Build Ready</span>";
      } else {
        badge = "<span class='card-badge eq'>" + ringSlots(lv) + " Slots</span>";
      }
      var next = ringLevelCost(lv);
      var tNext = timeForCost(next, true, rate, bnk.dust, bnk.shards);
      var curShards = bnk.shards || 0;
      var ringReady = (next <= 0) || (curShards >= next);
      pct = next > 0 ? Math.min(100, Math.max(0, (curShards / next) * 100)) : 100;
      var ringBarTip = ringReady ? ("Ready to upgrade Ring to Lv " + (lv + 1) + "! (" + prettify(next) + " shards)") : ("Progress to Ring Lv " + (lv + 1) + ": " + pct.toFixed(1) + "% (" + prettify(curShards) + " / " + prettify(next) + " shards)");
      var mods = formatModList(curMods);
      var reqModTxt = formatModList(reqMods);
      topLv = "<div class='lv'><span>Lv " + lv + (need && need > lv ? " <span class='gap'>→ " + need + "</span>" : "") + "</span></div>";
      if (modMismatch) {
        meta += "<div class='mods warn'>" + ICONS.warn + "<span>" + esc(mods || "none") + " → " + esc(reqModTxt) + "</span></div>";
        lines.push("Ring mod out of alignment: switch " + (mods || "none") + " → " + reqModTxt + ".");
      } else {
        meta += "<div class='mods'>" + esc(mods || "no custom mods") + "</div>";
      }
      meta += "<div class='cost'><span>Next (Lv " + (lv + 1) + ")</span><span>" + prettify(next) + " shards</span></div>";
      meta += "<div class='time'><span>Wait</span><span class='board-wait' data-item-cost='" + next + "' data-item-shards='1'>" + fmtTime(tNext) + "</span></div>";
      if (need > lv) {
        var gap = ringCostRange(lv, need);
        meta += "<div class='gapcost'><span>To Lv " + need + "</span><span>" + prettify(gap) + " shards · <span class='board-wait' data-item-cost='" + gap + "' data-item-shards='1'>" + fmtTime(timeForCost(gap, true, rate, bnk.dust, bnk.shards)) + "</span></span></div>";
        lines.push("To reach the build: Ring Lv " + lv + " → Lv " + need + ".");
      }
      lines.unshift(describeRing(lv, ringMods()));
      lines.push("Next level: all ring stats scale. " + nextRingSlotNote(lv + 1));
    }
    return "<article class='" + cls + "' title='" + esc(["The Ring"].concat(lines).join(" — ")) + "'>" +
      "<div>" +
      "<div class='card-head-row'><h3>" + ICONS.ring + "<span>The Ring</span></h3>" + badge + "</div>" +
      topLv +
      "<div class='lv-bar' title='" + esc(hasRing() ? ringBarTip : "Unlock The Ring") + "'><span class='board-bar" + (hasRing() && ringReady ? " ready" : "") + "' data-item-cost='" + (hasRing() ? next : cost) + "' data-item-shards='1' data-next-lv='" + (hasRing() ? (lv + 1) : "1") + "' data-currency='shards' data-max-level='0' style='width:" + (hasRing() ? (ringReady ? 100 : pct.toFixed(1)) : 0) + "%'></span></div>" +
      "</div>" +
      "<div class='card-meta-rows'>" + meta + "</div>" +
      cardTooltip("The Ring", lines) + "</article>";
  }

  function limbsCard(needSlots, rate, bnk) {
    var from = extraLimbs();
    var to = needSlots - 4;
    var cost = extraLimbCost(from, to);
    var t = timeForCost(cost, false, rate, bnk.dust, bnk.shards);
    var curDust = bnk.dust || 0;
    var isReady = (cost <= 0) || (curDust >= cost);
    var pct = cost > 0 ? Math.min(100, Math.max(0, (curDust / cost) * 100)) : 100;
    var limbBarTip = isReady ? ("Ready to buy Extra Limbs! (" + prettify(cost) + " dust)") : ("Progress to Extra Limbs: " + pct.toFixed(1) + "% (" + prettify(curDust) + " / " + prettify(cost) + " dust)");
    return "<article class='card warn rec'>" +
      "<div>" +
      "<div class='card-head-row'><h3>" + ICONS.limb + "<span>Extra Limbs</span></h3><span class='card-badge warn'>Need " + needSlots + "</span></div>" +
      "<div class='lv'><span>" + maxSlots() + " slots <span class='gap'>→ " + needSlots + "</span></span></div>" +
      "<div class='lv-bar' title='" + esc(limbBarTip) + "'><span class='board-bar" + (isReady ? " ready" : "") + "' data-item-cost='" + cost + "' data-item-shards='0' data-next-lv='" + needSlots + " slots' data-currency='dust' data-max-level='0' style='width:" + (isReady ? 100 : pct.toFixed(1)) + "%'></span></div>" +
      "</div>" +
      "<div class='card-meta-rows'>" +
      "<div class='cost'><span>Cost</span><span>" + prettify(cost) + " dust</span></div>" +
      "<div class='time'><span>Wait</span><span class='board-wait' data-item-cost='" + cost + "' data-item-shards='0'>" + fmtTime(t) + "</span></div>" +
      "</div>" +
      cardTooltip("Extra Limbs", ["Huffy can equip 1 additional item per level.", "Need " + from + " → " + to + "."]) +
      "</article>";
  }

  function renderBetterPopover(b, cur) {
    var delta = loadoutDelta(b, state.level, getLevelMode(state.level));
    var reqMods = (b.r && b.r.m) ? b.r.m : [];
    var curMods = hasRing() ? ringMods() : [];
    var reqModTxt = formatModList(reqMods) || "None";
    var curModTxt = hasRing() ? (formatModList(curMods) || "None") : "Locked";
    var curRingLv = hasRing() ? ringLevel() : 0;
    var reqRingLv = (b.r && b.r.lv) || 0;
    var ringLvGap = reqRingLv > curRingLv;

    var missingMods = reqMods.filter(function (m) { return curMods.indexOf(m) === -1; });
    var hasRingModSwap = b.r && (!hasRing() || missingMods.length > 0);
    var hasItemSwaps = delta && (delta.toEquip.length > 0 || delta.toUnequip.length > 0 || (delta.toSwap && delta.toSwap.length > 0));
    var hasAnySwap = hasRingModSwap || hasItemSwaps || ringLvGap;

    var dRows = [];
    if (b.r) {
      if (hasRingModSwap || ringLvGap) {
        var rFrom = (hasRing() ? ("Lv " + curRingLv) : "Locked") + " (" + curModTxt + ")";
        var rTo = "Lv " + reqRingLv + " (" + reqModTxt + ")";
        dRows.push(
          "<div class='delta-row'><span class='delta-lbl'>Ring:</span>" +
          "<span class='pill sm " + (hasRingModSwap ? "warn-pill" : "teal") + "'>" + ICONS.ring + " " + esc(rFrom + " → " + rTo) + "</span></div>"
        );
      } else {
        dRows.push(
          "<div class='delta-row'><span class='delta-lbl'>Ring:</span>" +
          "<span class='pill sm buy'>" + ICONS.check + " Lv " + reqRingLv + " (" + esc(reqModTxt) + ") — Aligned</span></div>"
        );
      }
    }
    if (delta && delta.toSwap && delta.toSwap.length) {
      var popSwPills = delta.toSwap.map(function (sw) {
        var toLv = userLevel(sw.to) || 1;
        return "<span class='pill sm accent'>" + iconForItem(sw.from) + " " + esc(itemName(sw.from)) + " → " + iconForItem(sw.to) + " " + esc(itemName(sw.to) + " (Lv " + toLv + ")") + "</span>";
      }).join("");
      dRows.push("<div class='delta-row'><span class='delta-lbl'>Extra Limb Swap:</span>" + popSwPills + "</div>");
    }
    if (delta && delta.extraLimbOptimal && delta.extraLimbOptimal.length) {
      var popOptEq = delta.extraLimbOptimal.filter(function (o) { return o.isCurEquipped; });
      if (popOptEq.length) {
        var popOptPills = popOptEq.map(function (o) {
          return "<span class='pill sm buy'>" + iconForItem(o.item) + " " + esc(itemName(o.item)) + " (Optimal)</span>";
        }).join("");
        dRows.push("<div class='delta-row'><span class='delta-lbl'>Extra Limb:</span>" + popOptPills + "</div>");
      }
    }
    if (delta && delta.toUnequip.length) {
      var unPills = delta.toUnequip.map(function (k) {
        return "<span class='pill sm sell'>" + iconForItem(k) + " " + esc(itemName(k)) + "</span>";
      }).join("");
      dRows.push("<div class='delta-row'><span class='delta-lbl'>Unequip:</span>" + unPills + "</div>");
    }
    if (delta && delta.toEquip.length) {
      var eqPills = delta.toEquip.map(function (k) {
        var owned = userOwns(k);
        var myLv = owned ? userLevel(k) : 0;
        var needLv = b.i[k] || 1;
        var lvSuffix = !owned ? " (Contract)" : (myLv < needLv ? (" (Lv " + myLv + "→" + needLv + ")") : (" (Lv " + needLv + ")"));
        return "<span class='pill sm buy'>" + iconForItem(k) + " " + esc(itemName(k) + lvSuffix) + "</span>";
      }).join("");
      dRows.push("<div class='delta-row'><span class='delta-lbl'>Equip:</span>" + eqPills + "</div>");
    }

    var swapSectionHtml;
    if (!hasItemSwaps && !hasRingModSwap) {
      swapSectionHtml = "<div class='delta-box' style='border-color:rgba(45,206,137,0.35);background:color-mix(in srgb,var(--buy) 6%,var(--surface-solid));'>" +
        "<div class='delta-title' style='color:var(--buy);'>" + ICONS.check + "<span>Active Loadout &amp; Ring Mods Aligned</span></div>" +
        (dRows.length ? "<div class='delta-rows'>" + dRows.join("") + "</div>" : "<div style='color:var(--muted);font-size:11.5px;'>You already have the exact items and Ring mods equipped for this build.</div>") +
        "</div>";
    } else {
      swapSectionHtml = "<div class='delta-box'>" +
        "<div class='delta-title'>" + ICONS.warn + "<span>Loadout &amp; Ring Changes Needed</span></div>" +
        "<div class='delta-rows'>" + dRows.join("") + "</div>" +
        "</div>";
    }

    var itemKeys = Object.keys(b.i || {});
    var loadoutItemsHtml = itemKeys.map(function (k) {
      var needLv = b.i[k];
      var owned = userOwns(k);
      var myLv = owned ? userLevel(k) : 0;
      var isEq = !!(userItem(k) && userItem(k).equipped);
      var sh = isShards(k);
      var cls = "bb-li" + (sh ? " shards" : "") + (!isEq ? " swap-in" : "") + (!owned ? " locked-item" : (myLv < needLv ? " need-up" : ""));
      var lvBadge = !owned ? ("Z" + ((itemCat(k) && itemCat(k).zone) || "?") + " → Lv " + needLv) : (myLv < needLv ? ("Lv " + myLv + "→" + needLv) : ("Lv " + needLv));
      return "<div class='" + cls + "'>" +
        "<span class='bb-li-name'>" + iconForItem(k) + "<span>" + esc(itemName(k)) + "</span></span>" +
        "<span class='bb-li-lv'>" + esc(lvBadge) + "</span>" +
        "</div>";
    }).join("");

    var metaPills = [];
    metaPills.push("<span class='pill sm buy'>" + ICONS.dust + " " + prettify(b.d) + "/s</span>");
    if (b.k != null) metaPills.push("<span class='pill sm info'>" + ICONS.clock + " " + b.k + "s</span>");
    if (b.wr != null) metaPills.push("<span class='pill sm accent'>" + Math.round((b.wr <= 1 ? b.wr : b.wr / 100) * 100) + "% WR</span>");
    metaPills.push("<span class='pill sm'>" + ICONS.limb + " " + (b.l || itemKeys.length) + " Limbs</span>");
    if (b.r) metaPills.push("<span class='pill sm teal'>" + ICONS.ring + " R" + b.r.lv + (reqMods.length ? " (" + esc(reqModTxt) + ")" : "") + "</span>");

    var itemSwapCount = delta ? (Math.max(delta.toEquip.length, delta.toUnequip.length) + (delta.toSwap ? delta.toSwap.length : 0)) : 0;
    var swapCount = itemSwapCount + (hasRingModSwap ? 1 : 0);
    var pillCls = "bb-info-pill " + (swapCount === 0 ? "aligned" : "has-swaps");
    var pillTxt = swapCount === 0 ? "Aligned" : (swapCount + (swapCount === 1 ? " swap" : " swaps"));

    var gain = (b.d || 0) - cur;
    var popHtml = "<div class='bb-popover' role='tooltip'>" +
      "<div class='sp-head'>" +
      "<span class='sp-title'>" + ICONS.info + "<span>" + esc(b.n) + " · Build &amp; Swap Details</span></span>" +
      "<div class='sp-head-right'><span class='sp-target'>+" + prettify(gain) + "/s</span></div>" +
      "</div>" +
      "<div class='rec-metrics'>" + metaPills.join("") + "</div>" +
      swapSectionHtml +
      "<div class='bb-pop-sec'>" +
      "<div class='bb-pop-kicker'><span>Target Loadout (" + itemKeys.length + " items)</span><span>Amber border = not currently equipped</span></div>" +
      "<div class='bb-loadout-grid'>" + loadoutItemsHtml + "</div>" +
      "</div>" +
      (b.note ? "<div class='notes' style='padding:7px 10px;font-size:11.5px;'>" + ICONS.info + "<span>" + esc(b.note) + "</span></div>" : "") +
      "</div>";

    return "<div class='bb-info-wrap' tabindex='0' data-open-build='" + esc(buildId(b)) + "'>" +
      "<button type='button' class='" + pillCls + "' data-open-build='" + esc(buildId(b)) + "' title='Click to view item swaps, Ring settings, and build guidance'>" + ICONS.info + "<span>" + esc(pillTxt) + "</span></button>" +
      popHtml +
      "</div>";
  }

  function openBuildDetailsModal(build) {
    if (!build) return;
    var dlg = $("buildDetailsDlg");
    if (!dlg) return;

    var cur = (state.lastBetterCur != null) ? state.lastBetterCur : levelCurrentDustPs(state.level);
    var delta = loadoutDelta(build);
    var gain = (build.d || 0) - cur;

    var bTitle = $("bddTitle");
    if (bTitle) bTitle.textContent = (build.n || "Build") + " · Swap & Loadout Guidance";

    var bBody = $("bddBody");
    if (bBody) {
      var reqMods = formatModList(build.r && build.r.m || []) || "None";
      var metaPills = [];
      metaPills.push("<span class='pill sm buy'>" + ICONS.dust + " " + prettify(build.d) + "/s (" + (gain >= 0 ? "+" : "") + prettify(gain) + "/s)</span>");
      if (build.k != null) metaPills.push("<span class='pill sm info'>" + ICONS.clock + " " + build.k + "s kills</span>");
      if (build.wr != null) metaPills.push("<span class='pill sm accent'>" + Math.round((build.wr <= 1 ? build.wr : build.wr / 100) * 100) + "% WR</span>");
      metaPills.push("<span class='pill sm'>" + ICONS.limb + " " + (build.l || "?") + " Limbs</span>");
      if (build.r) metaPills.push("<span class='pill sm teal'>" + ICONS.ring + " Ring Lv " + build.r.lv + " (" + esc(reqMods) + ")</span>");

      var swapGuideHtml = renderAlignmentGuide(build, delta, "Loadout & Ring Changes Needed (Swaps)", true);

      var itemKeys = Object.keys(build.i || {});
      var itemCards = itemKeys.map(function (k) {
        var owned = userOwns(k);
        var myLv = owned ? userLevel(k) : 0;
        var needLv = build.i[k] || 1;
        var isEq = !!(userItem(k) && userItem(k).equipped);
        var sh = isShards(k);
        var cls = "loadout-item" + (sh ? " shards" : "") + (!isEq ? " unequipped" : "") + (myLv < needLv ? " underleveled" : "");
        var lvBadge = !owned ? ("Locked · Lv " + needLv) : (myLv < needLv ? ("Lv " + myLv + " → " + needLv) : ("Lv " + needLv));
        var tip = (isEq ? "Equipped" : "Unequipped") + " · Target: Lv " + needLv + (owned ? (" (Current: Lv " + myLv + ")") : " (Not yet unlocked)");
        return "<div class='" + cls + "' title='" + esc(tip) + "'>" +
          "<span class='li-name'>" + iconForItem(k) + "<span>" + esc(itemName(k)) + "</span></span>" +
          "<span class='li-lv'>" + esc(lvBadge) + "</span>" +
          "</div>";
      }).join("");

      var isLocked = isSameLockedBuild(state.level, build);
      var bId = buildId(build);
      var lockBtnHtml = isLocked
        ? "<button type='button' class='btn sm accent' id='bddLockBtn'>" + ICONS.lock + "<span>Plan Locked</span></button>"
        : "<button type='button' class='btn sm ghost' id='bddLockBtn'>" + ICONS.unlock + "<span>Lock Plan</span></button>";
      var simBtnHtml = "<button type='button' class='btn sm ghost' id='bddSimBtn'>⚔️ Sim Build</button>";

      bBody.innerHTML =
        "<div class='rec-metrics' style='gap:6px;flex-wrap:wrap;'>" + metaPills.join("") + "</div>" +
        swapGuideHtml +
        "<div style='display:flex;flex-direction:column;gap:6px;margin-top:6px;'>" +
        "<div class='bb-pop-kicker'><span>Target Loadout (" + itemKeys.length + " items)</span><span style='color:var(--warn);'>Amber border = not currently equipped</span></div>" +
        "<div class='loadout-grid' style='grid-template-columns:repeat(auto-fill, minmax(170px, 1fr));gap:6px;'>" + itemCards + "</div>" +
        "</div>" +
        "<div style='display:flex;justify-content:flex-end;align-items:center;gap:8px;margin-top:10px;padding-top:10px;border-top:1px solid var(--border);'>" +
        simBtnHtml +
        lockBtnHtml +
        "<button type='button' class='btn sm ghost' id='bddCloseBtn'>Close</button>" +
        "</div>";

      var lBtn = $("bddLockBtn");
      if (lBtn) {
        lBtn.onclick = function() {
          toggleLockBuild(build);
          closeBuildDetailsModal();
        };
      }
      var sBtn = $("bddSimBtn");
      if (sBtn) {
        sBtn.onclick = function() {
          closeBuildDetailsModal();
          openSimModal(build);
        };
      }
      var cBtn = $("bddCloseBtn");
      if (cBtn) cBtn.onclick = closeBuildDetailsModal;
    }

    if (typeof dlg.showModal === "function") dlg.showModal();
    else dlg.setAttribute("open", "");
  }

  function closeBuildDetailsModal() {
    var dlg = $("buildDetailsDlg");
    if (dlg) {
      if (typeof dlg.close === "function") dlg.close();
      else dlg.removeAttribute("open");
    }
  }

  function renderBetter(rec, rate, cmpRateOverride) {
    var cur = (typeof cmpRateOverride === "number" && cmpRateOverride >= 0)
      ? cmpRateOverride
      : Math.max(levelCurrentDustPs(state.level, rec), rec ? (rec.d || 0) : 0);
    var bnk = bank();
    var list = betterBuilds(rec, cur);
    var mode = getLevelMode(state.level);
    var curK = rec ? buildSecPerKill(rec, state.level) : Infinity;
    list.sort(function (a, b) {
      var stepsA = a._planSteps || applyStepOrder(state.level, a, expandSteps(gapSteps(a)));
      a._planSteps = stepsA;
      var stepsB = b._planSteps || applyStepOrder(state.level, b, expandSteps(gapSteps(b)));
      b._planSteps = stepsB;
      var wa = calculateStepWaitTimes(stepsA, bnk, rate);
      var wb = calculateStepWaitTimes(stepsB, bnk, rate);
      var ca = Math.max(wa.remDust, wa.remShards * SHARD_DUST);
      if (ca <= 0) ca = Math.max(1, (wa.totalCostDust + wa.totalCostShards * SHARD_DUST) * 1e-6);
      var cb = Math.max(wb.remDust, wb.remShards * SHARD_DUST);
      if (cb <= 0) cb = Math.max(1, (wb.totalCostDust + wb.totalCostShards * SHARD_DUST) * 1e-6);
      if (mode === "push") {
        var ka = buildSecPerKill(a, state.level);
        var kb = buildSecPerKill(b, state.level);
        var ga = Math.max(0, curK - ka);
        var gb = Math.max(0, curK - kb);
        var ra = ga / ca;
        var rb = gb / cb;
        if (Math.abs(ra - rb) > 1e-12) return rb - ra;
        return ka - kb;
      }
      var ra = ((a.d || 0) - cur) / ca;
      var rb = ((b.d || 0) - cur) / cb;
      return rb - ra;
    });
    list = list.slice(0, 8);
    state.lastBetterBuilds = list;
    state.lastBetterCur = cur;
    if (!list.length) {
      var curTxt = (cur && cur > 0) ? (" (" + prettify(cur) + "/s)") : "";
      $("better").innerHTML = rec
        ? "<p class='empty'>Your current dust rate" + curTxt + " matches or exceeds all listed sheet builds on this level, or nothing cheaper is listed with the current filters.</p>"
        : "<p class='empty'>No builds on this tab match the current filters.</p>";
      return;
    }
    var rows = list.map(function (b, idx) {
      var steps = b._planSteps || applyStepOrder(state.level, b, expandSteps(gapSteps(b)));
      b._planSteps = steps;
      var cost = stepsCostDust(steps);
      var wInfo = calculateStepWaitTimes(steps, bnk, rate);
      var earn = wInfo.totalWait;
      var retInfo = calculateBuildReturnTime(b, steps, wInfo, cur);
      var gain = retInfo.gain;
      var blockedContracts = steps.filter(function (s) { return s.kind === "contract" && !s.reach; });
      var blocked = blockedContracts.length > 0;
      var blockedZone = 0;
      if (blocked) {
        blockedContracts.forEach(function (s) { if (s.zone && s.zone > blockedZone) blockedZone = s.zone; });
        if (!blockedZone && b.c) blockedZone = b.c;
      }
      var earnStr = (blocked && earn === 0) ? ("Z" + (blockedZone || "?")) : (earn === 0 ? "ready" : fmtTime(earn));
      var retStr = retInfo.retStr;
      var earnReturnVal = earnStr + "/" + retStr;
      var isSimmed = retInfo.isSimmed;
      var earnDurStr = fmtDuration(earn);
      var earnReturnTip = blocked
        ? ("Requires unreached Zone " + (blockedZone || "?") + " contract (highest zone: " + (state.radon + 1) + ") · Dust wait: " + earnStr + " (" + earnDurStr + ") · Return: " + retStr)
        : ("Earn: " + earnStr + " (" + earnDurStr + " remaining wait" + (isSimmed ? " · Sim 9001" : "") + ") · Return: " + retStr + " (payback: cost / " + (retInfo.gain > 0 ? "Δ d/s" : "plan d/s") + ")");
      return "<tr" + (blocked ? " class='unreachable'" : "") + ">" +
        "<td><div class='bb-build-cell'><strong>" + esc(b.n) + "</strong>" +
        renderBetterPopover(b, cur) +
        (blocked ? "<span class='pill sm sell' title='Requires a Void Map zone higher than your Highest Zone U2'>" + ICONS.warn + " Z" + esc(String(b.c || "?")) + "</span>" : "") +
        "</div><div class='sub mono' style='margin-top:3px;'>" + prettify(b.d) + "/s" + (b.k != null ? " · " + esc(String(b.k)) + "s" : "") + "</div></td>" +
        "<td><div class='steps'>" + steps.map(function (s) {
          return stepBadgeHtml(s);
        }).join("") + "</div></td>" +
        "<td class='mono'>" + (cost.dust ? prettify(cost.dust) + " dust" : "") +
        (cost.dust && cost.shards ? "<br>" : "") +
        (cost.shards ? prettify(cost.shards) + " shards" : "") + "</td>" +
        (mode === "push"
          ? (function() {
              var bk = buildSecPerKill(b, state.level);
              var diff = isFinite(curK) && curK > bk ? (" (-" + (curK - bk).toFixed(1) + "s)") : "";
              return "<td class='mono pos'>" + bk.toFixed(1) + "s kills" + (diff ? "<span style='font-size:11px;opacity:0.8;'> " + diff + "</span>" : "") + "</td>";
            })()
          : ("<td class='mono pos'>+" + prettify(gain) + "/s</td>")) +
        "<td class='mono bb-earn-return" + (blocked && earn === 0 ? " blocked" : "") + "' data-build-idx='" + idx + "' title='" + esc(earnReturnTip) + "'>" + esc(earnReturnVal) + "</td>" +
        "<td class='bb-actions-td'><div class='bb-actions-cell'>" +
        "<button type='button' class='btn sm ghost sim-row-btn' data-sim-build='" + esc(buildId(b)) + "' title='Load into Combat Simulator'>⚔️ Sim</button>" +
        (function() {
          var isRowLocked = isSameLockedBuild(state.level, b);
          var bId = buildId(b);
          return isRowLocked
            ? "<button type='button' class='btn sm accent lock-row-btn locked' data-lock-build='" + esc(bId) + "' title='Advisor is locked to this build. Click to unlock.'>" + ICONS.lock + "<span>Locked</span></button>"
            : "<button type='button' class='btn sm ghost lock-row-btn' data-lock-build='" + esc(bId) + "' title='Lock Next Upgrade Advisor to this build'>" + ICONS.unlock + "<span>Lock</span></button>";
        })() + "</div></td></tr>";
    }).join("");
    var betterEl = $("better");
    var colGainLbl = (mode === "push") ? "Kill Speed (Δ)" : "Δ d/s";
    betterEl.innerHTML = "<div class='tbl-wrap'><div class='tbl-swipe-hint'>↔ Swipe table horizontally to view all columns &amp; actions</div><table><thead><tr><th>Build</th><th class='th-missing'>Missing Upgrades</th><th class='th-cost'>Total Cost</th><th class='th-gain'>" + colGainLbl + "</th><th class='th-earn'>Earn / Return</th><th class='th-actions'>Sim / Plan</th></tr></thead><tbody>" + rows + "</tbody></table></div>";
    betterEl.onclick = function(ev) {
      var sBtn = ev.target.closest("button[data-sim-build]");
      if (sBtn) {
        ev.preventDefault();
        var sId = sBtn.getAttribute("data-sim-build");
        var targetB = (state.lastBetterBuilds || []).find(function(b) { return buildId(b) === sId; }) ||
                      findLockedTarget(state.level, { id: sId });
        if (targetB) openSimModal(targetB);
        return;
      }
      var bBtn = ev.target.closest("[data-open-build]");
      if (bBtn) {
        ev.preventDefault();
        var bId = bBtn.getAttribute("data-open-build");
        var targetBuild = (state.lastBetterBuilds || []).find(function(b) { return buildId(b) === bId; }) ||
                          findLockedTarget(state.level, { id: bId });
        if (targetBuild) openBuildDetailsModal(targetBuild);
        return;
      }
      var btn = ev.target.closest("button[data-lock-build]");
      if (!btn) return;
      ev.preventDefault();
      toggleLockBuild(btn.getAttribute("data-lock-build"));
    };
    triggerBetterBuildsSimulation(list, rate);
  }

  function fact(label, value, icon) {
    return '<div class="fact">' + (icon || '') + '<span>' + esc(label) + '</span>' + esc(String(value)) + '</div>';
  }

  function ownedItemKeys() {
    var keys = [];
    var items = (state.ab && state.ab.items) || {};
    for (var k in items) {
      if (items[k] && items[k].owned !== false) keys.push(k);
    }
    return keys;
  }

  function renderSaveFacts() {
    var bnk = bank();
    var rate = activeDustPs();
    var ago = bnk.elapsedSec < 5 ? "just now" : (fmtTime(bnk.elapsedSec) + " ago");
    var loadedTimeStr = state.saveLoadedAt ? (new Date(state.saveLoadedAt).toLocaleTimeString()) : "";
    var exportTimeStr = (state.save && state.save.global && state.save.global.lastOnline)
      ? (new Date(state.save.global.lastOnline).toLocaleTimeString())
      : "";
    var factTitle = "Loaded in planner: " + (loadedTimeStr || "—");
    if (exportTimeStr) factTitle += "\nExported from Trimps: " + exportTimeStr;
    factTitle += "\nClick to reset live accumulation timer to now";

    // 1. Update loaded time pill in top header
    var loadedEl = $("factLoadedTime");
    if (loadedEl) loadedEl.textContent = ago;
    var loadedWrap = $("factLoaded");
    if (loadedWrap) loadedWrap.title = factTitle;

    // 2. Update full save facts tooltip on sourceStatus pill in top header
    var owned = ownedItemKeys();
    var equipped = owned.filter(function (k) { return userItem(k) && userItem(k).equipped; }).length;
    var contracts = owned.filter(function (k) { var c = itemCat(k); return c && c.zone; }).length;
    var contractItems = Object.keys(SA_DATA.items).filter(function (k) { return SA_DATA.items[k].zone; }).length;
    var contract = state.ab.activeContract;
    var mods = ringMods().map(function (m) { return RING_MOD_NAMES[m] || m; }).join(" / ");
    var mb = macroBonusInfo();
    var muts = (state.save && state.save.global && state.save.global.u2MutationData) || {};
    var mutStr = muts.Dust2 ? "Dusty & Dustier (+50% Dust)" : (muts.Dust ? "Dusty (+25% Dust)" : "");

    var saveTooltip = "Trimps Save Snapshot:" +
      "\n• Highest Zone U2: Z" + (state.radon + 1) + " (Void maps to Z" + (state.radon + 1) + ")" +
      "\n• Active Contract: " + (contract ? itemName(contract) : "None") +
      "\n• Items: " + owned.length + " / " + SA_DATA.itemOrder.length + " (" + equipped + " equipped)" +
      "\n• Contracts Completed: " + contracts + " / " + contractItems +
      "\n• Ring Mods: " + (hasRing() ? (mods || "None") : "Locked") +
      "\n• Spire Buffs: +" + mb.radonPct + "% Radon · +" + mb.statsPct + "% Stats · " + mb.scaffoldLvl + " Scaffolding" +
      (mutStr ? ("\n• Mutators: " + mutStr) : "") +
      "\n• Loaded in planner: " + (loadedTimeStr || "—") + (exportTimeStr ? (" (Exported: " + exportTimeStr + ")") : "");

    var statusEl = $("sourceStatus");
    if (statusEl) statusEl.title = saveTooltip;

    // 3. Spire Macro Buffs capsule & Bonuses Popover
    var bonusesHtml = renderBonusesHtml(mb, bnk, rate);

    // 4. In-Game Settings Safety Alerts
    var audit = saSettingsAudit();
    var warningFacts = [];
    if (audit.practiceActive) {
      warningFacts.push(
        '<div class="fact fact-warn" style="border-color:rgba(244,63,94,0.6); background:var(--sell-soft); color:var(--sell);" title="Trimps Practice Mode is ON! Game runs at 10x speed with 0 dust/shards and no level progression.">' +
        ICONS.warn + '<b>Practice Mode (0 Dust)</b></div>'
      );
    }

    if (audit.loadRingDisabled) {
      warningFacts.push(
        '<div class="fact fact-warn" style="border-color:rgba(251,191,36,0.5); background:var(--warn-soft); color:var(--warn);" title="Trimps setting \'Load Ring\' is OFF. Loading a preset in-game will NOT change your Ring mods.">' +
        ICONS.warn + '<span>Presets:</span> <b>Ring load OFF</b></div>'
      );
    }

    // In controls-bar center: Spire Buffs (with interactive Bonuses popover) and warning alerts
    var factsCenter = $("saveFacts");
    if (factsCenter) {
      factsCenter.innerHTML = [
        bonusesHtml,
        warningFacts.join("")
      ].join("");
    }
  }

  function setSaveCollapsed(collapsed) {
    $("saveEditor").hidden = collapsed;
    if ($("savePanel")) {
      $("savePanel").hidden = collapsed;
      $("savePanel").classList.toggle("hero-mode", !collapsed && !state.ab);
    }
    if ($("cancelEditBtn")) {
      $("cancelEditBtn").classList.toggle("hidden", !state.ab);
    }
    if ($("heroFeats")) {
      $("heroFeats").classList.toggle("hidden", !!state.ab);
    }
    if ($("btnPresets")) {
      $("btnPresets").style.display = (state.ab && collapsed) ? "inline-flex" : "none";
    }
    if ($("btnShareBuild")) {
      $("btnShareBuild").style.display = (state.ab && collapsed) ? "inline-flex" : "none";
    }
    if ($("updateBtn")) {
      $("updateBtn").style.display = (state.ab && collapsed) ? "inline-flex" : "none";
    }
    if ($("factLoaded")) {
      $("factLoaded").style.display = (state.ab && collapsed) ? "inline-flex" : "none";
    }
  }

  function onUpdate() {
    setSaveCollapsed(false);
    $("err").hidden = true;
    var box = $("saveBox");
    box.focus();
    box.select();
  }

  function onCancelEdit() {
    if (state.ab) {
      $("err").hidden = true;
      setSaveCollapsed(true);
    }
  }

  function onPasteClipboard() {
    if (navigator.clipboard && navigator.clipboard.readText) {
      navigator.clipboard.readText().then(function (txt) {
        if (txt && txt.trim()) {
          $("saveBox").value = txt.trim();
          onLoad();
        } else {
          onUpdate();
        }
      }).catch(function () {
        onUpdate();
      });
    } else {
      onUpdate();
    }
  }

  function isBlockedByContract(build) {
    return gapSteps(build).some(function (s) { return s.kind === "contract" && !s.reach; });
  }

  function shouldDisableContractsOnLoad(level) {
    var rec = bestRunnable(level);
    var cur = Math.max(levelCurrentDustPs(level, rec), rec ? (rec.d || 0) : 0);
    var allBetter = buildsFor(level).filter(function (b) {
      return (b.d || 0) > cur + 1e-9 && !runnable(b);
    });
    if (!allBetter.length) return false;
    var contractBetter = allBetter.filter(function (b) {
      return stepsNeed(gapSteps(b), "contract");
    });
    if (!contractBetter.length) return false;
    if (contractBetter.every(isBlockedByContract)) return true;

    var savedContract = state.filters.contract;
    state.filters.contract = true;
    var visible = betterBuilds(rec, cur);
    state.filters.contract = savedContract;
    visible.sort(function (a, b) {
      var ca = stepsCostDust(gapSteps(a)).combined;
      var cb = stepsCostDust(gapSteps(b)).combined;
      var ra = ca > 0 ? ((a.d || 0) - cur) / ca : 0;
      var rb = cb > 0 ? ((b.d || 0) - cur) / cb : 0;
      return rb - ra;
    });
    var top8 = visible.slice(0, 8);
    return top8.length > 0 && top8.every(isBlockedByContract);
  }

  function updateLiveBank() {
    if (document.hidden) return;
    if (!state.ab || !state.saveLoadedAt) return;
    if ($("saveEditor") && !$("saveEditor").hidden) return;
    if (document.activeElement === $("dustOverride")) return;

    var rec = state._cachedRec !== undefined ? state._cachedRec : bestRunnable(state.level);
    var rate = activeDustPs(rec);
    var bnk = bank();

    var eCard = $("kpiEnemyLevel");
    if (eCard) {
      var eDiag = (simState.session && simState.active) ? combatDiagnostics(rate) : (state._cachedDiag || combatDiagnostics(rate));
      if (eDiag.isPushing && eDiag.etaSec > 0) {
        var liveRemSec = Math.max(0, eDiag.etaSec - bnk.elapsedSec);
        var liveEtaDur = liveRemSec > 0 ? ("~" + fmtDuration(liveRemSec)) : "Ready to clear";
        var liveKillsEarned = (eDiag.secPerKill && eDiag.secPerKill > 0)
          ? Math.min(eDiag.remainingKills, Math.floor(bnk.elapsedSec / eDiag.secPerKill))
          : 0;
        var liveRemaining = Math.max(0, eDiag.remainingKills - liveKillsEarned);
        var liveKillsTotal = eDiag.curKills + liveKillsEarned;
        var livePct = Math.min(100, (liveKillsTotal / eDiag.neededKills) * 100);

        var txtEl = eCard.querySelector(".kpi-prog-txt");
        var etaEl = eCard.querySelector(".kpi-prog-eta");
        var barEl = eCard.querySelector(".kpi-bar > span");
        if (txtEl) txtEl.textContent = "Pushing · " + prettify(liveRemaining) + " to go (" + livePct.toFixed(0) + "%)";
        if (etaEl) etaEl.textContent = liveEtaDur;
        if (barEl) barEl.style.width = Math.max(2, livePct).toFixed(1) + "%";
      }
    }

    var dCard = $("kpiDustBank");
    if (dCard) {
      var dVal = dCard.querySelector(".kpi-value");
      var dNote = dCard.querySelector(".kpi-note");
      if (dVal) dVal.textContent = prettify(bnk.dust);
      if (dNote) {
        if (bnk.dustEarned > 0) {
          dNote.textContent = "Base " + prettify(bnk.rawDust) + " · +" + prettify(bnk.dustEarned) + " earned (" + fmtTime(bnk.elapsedSec) + ")";
        } else {
          dNote.textContent = rate ? ("Active rate: " + prettify(rate) + "/s") : "No active rate";
        }
      }
      dCard.title = "Live estimated dust bank\nSnapshot from save: " + prettify(bnk.rawDust) +
        (bnk.dustEarned > 0 ? ("\nEarned since load: +" + prettify(bnk.dustEarned) + " (" + fmtTime(bnk.elapsedSec) + " ago)") : "") +
        (rate ? ("\nActive earning rate: " + prettify(rate) + "/s") : "");
    }

    var sCard = $("kpiShardBank");
    if (sCard) {
      var sVal = sCard.querySelector(".kpi-value");
      var sNote = sCard.querySelector(".kpi-note");
      if (sVal) sVal.textContent = prettify(bnk.shards);
      if (sNote) {
        if (bnk.shardsEarned > 0) {
          sNote.textContent = "Base " + prettify(bnk.rawShards) + " · +" + prettify(bnk.shardsEarned) + " earned";
        } else if (canEarnShards() && rate > 0) {
          var curShardDust = ((state.ab.shardDust || 0) + bnk.dustEarned) % SHARD_DUST;
          var pct = Math.min(99.9, (curShardDust / SHARD_DUST) * 100).toFixed(1);
          sNote.textContent = "Next: " + pct + "%";
        } else {
          sNote.textContent = "";
        }
      }
      sCard.title = "Live estimated shard bank\nSnapshot from save: " + prettify(bnk.rawShards) + " shards" +
        (bnk.shardsEarned > 0 ? ("\nEarned since load: +" + bnk.shardsEarned) : "") +
        "\nDust equivalent: " + prettify((bnk.shards || 0) * SHARD_DUST) + " dust";
    }

    var fLoaded = $("factLoadedTime");
    if (fLoaded) {
      var ago = bnk.elapsedSec < 5 ? "just now" : (fmtTime(bnk.elapsedSec) + " ago");
      fLoaded.textContent = ago;
    }

    updateBonusesPopover(rate, bnk);

    var dlg = $("presetsDlg");
    if (dlg && dlg.open) {
      var mBar = $("modalCombatBar");
      var mTxt = $("modalCombatKillsTxt");
      var mEta = $("modalCombatEta");
      if (mBar && mTxt) {
        var mDiag = combatDiagnostics(rate);
        if (mDiag.isPushing) {
          var mEarned = (mDiag.secPerKill && mDiag.secPerKill > 0 && bnk.elapsedSec > 0)
            ? Math.min(mDiag.remainingKills, Math.floor(bnk.elapsedSec / mDiag.secPerKill))
            : 0;
          var mRemaining = Math.max(0, mDiag.remainingKills - mEarned);
          var mKillsTotal = mDiag.curKills + mEarned;
          var mPct = mDiag.neededKills > 0 ? Math.min(100, (mKillsTotal / mDiag.neededKills) * 100) : 0;
          var mRemSec = Math.max(0, (mDiag.etaSec || 0) - bnk.elapsedSec);
          var mDuration = mRemSec > 0 ? ("~" + fmtDuration(mRemSec)) : (mDiag.etaSec > 0 ? "Ready to clear" : "");
          var mClock = mRemSec > 0 ? fmtLocalEta(mRemSec) : "";
          var mEtaStr = mDuration ? (mDuration + (mClock ? " (ETA " + mClock + ")" : "")) : "";

          mTxt.innerHTML = "<b>" + prettify(mKillsTotal) + " / " + prettify(mDiag.neededKills) + "</b> kills (" + mPct.toFixed(1) + "%) · " +
            "<b style='color:var(--accent);'>" + prettify(mRemaining) + " kills to clear</b>";
          if (mEta) mEta.textContent = mEtaStr;
          mBar.style.width = Math.max(2, mPct).toFixed(1) + "%";
        }
      }
    }

    var badge = $("advisorWaitBadge");
    if (badge && state.lastAdvisorStep) {
      var st = state.lastAdvisorStep;
      var rawCost = st.cost || 0;
      var wait = timeForCost(rawCost, !!st.shards, rate, bnk.dust, bnk.shards);
      if (wait === 0) {
        if (!badge.classList.contains("buy")) {
          badge.className = "pill sm buy";
          badge.innerHTML = ICONS.check + " Ready now";
          renderNext(rec, rate);
        }
      } else if (wait != null && isFinite(wait)) {
        badge.className = "pill sm";
        var stepPrefix = badge.getAttribute("data-step-prefix") || "";
        badge.innerHTML = ICONS.clock + " " + stepPrefix + fmtTime(wait);
      }
    }

    var naCard = $("kpiNextAction");
    if (naCard) {
      var pick = state._cachedPick !== undefined ? state._cachedPick : bestNextUpgrade(rec, rate);
      if (pick && pick.isLocked) {
        if (pick.mode === "completed") {
          var valEl = $("kpiNextActionVal");
          if (!valEl || valEl.textContent !== "Plan Completed") {
            renderChips(rec, rate);
            return;
          }
        } else if (pick.step) {
          var s = pick.step;
          var actNote = nextActionNote(s);
          var rawCost = s.cost || 0;
          var isSh = !!s.shards;
          var curFunds = isSh ? (bnk.shards || 0) : (bnk.dust || 0);
          var wait = timeForCost(rawCost, isSh, rate, bnk.dust, bnk.shards);
          var isReady = (wait === 0) || (rawCost <= 0) || (curFunds >= rawCost);
          var pct = rawCost > 0 ? Math.min(100, Math.max(0, (curFunds / rawCost) * 100)) : 100;
          var pctTxt = (isReady || pct >= 99.95) ? "100%" : (pct.toFixed(pct >= 10 ? 0 : 1) + "%");

          var valEl = $("kpiNextActionVal");
          var actEl = naCard.querySelector(".kpi-prog-action");
          var currentActNote = actEl ? actEl.textContent : "";
          var curIsReady = valEl && valEl.textContent === "Next Action Ready";

          if (currentActNote !== actNote || curIsReady !== isReady) {
            renderChips(rec, rate);
            return;
          }

          var pctEl = $("kpiNextActionPct");
          var barEl = $("kpiNextActionBar");
          if (valEl) {
            var newEta = fmtLocalEta(wait);
            if (valEl.textContent !== newEta) {
              valEl.textContent = newEta;
              fitKpiValue(valEl);
            }
          }
          if (pctEl) pctEl.textContent = pctTxt;
          if (barEl) barEl.style.width = pct.toFixed(1) + "%";
          var titleWait = "Next Action: " + actNote +
            "\nCost: " + prettify(rawCost) + (isSh ? " shards" : " dust") +
            "\nCurrent bank: " + prettify(curFunds) + (isSh ? " shards" : " dust") + " (" + pctTxt + ")" +
            "\nAvailable at: " + fmtLocalEta(wait) + " (" + fmtTime(wait) + " at " + prettify(rate) + "/s)";
          naCard.title = titleWait;
        }
      } else {
        renderChips(rec, rate);
      }
    }

    if (state.lastPlanSteps && state.lastPlanSteps.length) {
      refreshPlanPopoverWaitTimes();
    }

    if (state.lastBetterBuilds && state.lastBetterBuilds.length) {
      var cells = document.querySelectorAll("#better .bb-earn-return");
      for (var ci = 0; ci < cells.length; ci++) {
        var cell = cells[ci];
        var idx = parseInt(cell.getAttribute("data-build-idx"), 10);
        var b = state.lastBetterBuilds[idx];
        if (b) {
          var bSteps = b._planSteps || applyStepOrder(state.level, b, expandSteps(gapSteps(b)));
          b._planSteps = bSteps;
          var bWInfo = calculateStepWaitTimes(bSteps, bnk, rate);
          var bEarn = bWInfo.totalWait;
          var retInfo = calculateBuildReturnTime(b, bSteps, bWInfo, state.lastBetterCur);
          var bBlockedContracts = bSteps.filter(function (s) { return s.kind === "contract" && !s.reach; });
          var bBlocked = bBlockedContracts.length > 0;
          var bBlockedZone = 0;
          if (bBlocked) {
            bBlockedContracts.forEach(function (s) { if (s.zone && s.zone > bBlockedZone) bBlockedZone = s.zone; });
            if (!bBlockedZone && b.c) bBlockedZone = b.c;
          }
          var bEarnStr = (bBlocked && bEarn === 0) ? ("Z" + (bBlockedZone || "?")) : (bEarn === 0 ? "ready" : fmtTime(bEarn));
          var bRetStr = retInfo.retStr;
          var bEarnReturnVal = bEarnStr + "/" + bRetStr;
          var isSimmed = retInfo.isSimmed;
          var bEarnDurStr = fmtDuration(bEarn);
          var bEarnReturnTip = bBlocked
            ? ("Requires unreached Zone " + (bBlockedZone || "?") + " contract (highest zone: " + (state.radon + 1) + ") · Dust wait: " + fmtTime(bEarn) + " · Return: " + bRetStr)
            : ("Earn: " + bEarnStr + " (remaining wait with bank" + (isSimmed ? " · Sim 9001" : "") + ") · Return: " + bRetStr + " (payback: cost / " + (retInfo.gain > 0 ? "Δ d/s" : "plan d/s") + ")");
          if (cell.textContent !== bEarnReturnVal) cell.textContent = bEarnReturnVal;
          if (cell.title !== bEarnReturnTip) cell.title = bEarnReturnTip;
        }
      }
    }

    var bWaitEls = document.querySelectorAll("#board .board-wait");
    for (var bi = 0; bi < bWaitEls.length; bi++) {
      var el = bWaitEls[bi];
      var unreach = el.getAttribute("data-unreached-zone");
      if (unreach) {
        if (el.textContent !== ("Z" + unreach)) el.textContent = "Z" + unreach;
        continue;
      }
      var c = parseFloat(el.getAttribute("data-item-cost")) || 0;
      var sh = el.getAttribute("data-item-shards") === "1";
      var w = timeForCost(c, sh, rate, bnk.dust, bnk.shards);
      var txt = fmtTime(w);
      if (el.textContent !== txt) el.textContent = txt;
    }

    var bBarEls = document.querySelectorAll("#board .board-bar");
    for (var bbi = 0; bbi < bBarEls.length; bbi++) {
      var bar = bBarEls[bbi];
      if (bar.getAttribute("data-max-level") === "1") continue;
      var bCost = parseFloat(bar.getAttribute("data-item-cost")) || 0;
      if (bCost <= 0) continue;
      var bSh = bar.getAttribute("data-item-shards") === "1";
      var bFunds = bSh ? (bnk.shards || 0) : (bnk.dust || 0);
      var bReady = bFunds >= bCost;
      var bPct = bReady ? 100 : Math.min(100, Math.max(0, (bFunds / bCost) * 100));
      var bPctStr = (bReady ? 100 : bPct.toFixed(1)) + "%";
      if (bar.style.width !== bPctStr) bar.style.width = bPctStr;
      if (bReady && !bar.classList.contains("ready")) bar.classList.add("ready");
      else if (!bReady && bar.classList.contains("ready")) bar.classList.remove("ready");
      var pLv = bar.getAttribute("data-next-lv");
      var pCurr = bar.getAttribute("data-currency") || (bSh ? "shards" : "dust");
      if (bar.parentElement) {
        var nTip = bReady ? ("Ready to upgrade" + (pLv ? (" to Lv " + pLv) : "") + "! (" + prettify(bCost) + " " + pCurr + ")") :
          ("Progress" + (pLv ? (" to Lv " + pLv) : "") + ": " + bPct.toFixed(1) + "% (" + prettify(bFunds) + " / " + prettify(bCost) + " " + pCurr + ")");
        if (bar.parentElement.title !== nTip) bar.parentElement.title = nTip;
      }
    }
  }

  var liveBankTimer = null;
  function startLiveBankTimer() {
    if (liveBankTimer) clearInterval(liveBankTimer);
    liveBankTimer = setInterval(updateLiveBank, 1000);
  }

  function resetBankTimer() {
    state.saveLoadedAt = Date.now();
    try { localStorage.setItem("saPlanner.saveLoadedAt", String(state.saveLoadedAt)); } catch (e) {}
    toast("Live accumulation timer reset to now");
    render();
  }

  function onLoad(isRestore) {
    $("err").hidden = true;
    try {
      var rawVal = $("saveBox").value;
      var decoded = decodeSave(rawVal);
      state.save = decoded.save;
      state.lastSimulatedStepsSig = null;
      state.activePlanSimSig = null;
      state.isBetterSimRunning = false;
      stepSimCache = {};
      planRatesCache = {};
      pushSimCache = {};
      levelSimCache = {};
      buildKillTimeCache = {};
      betterSimToken++;
      if (state.lastBetterBuilds) {
        state.lastBetterBuilds.forEach(function (b) {
          delete b._lastSimSig;
          delete b._planSteps;
          delete b._isSimulated;
        });
      }
      state.ab = decoded.ab;
      state.radon = decoded.radon;
      if (!isRestore || !state.saveLoadedAt) {
        state.saveLoadedAt = Date.now();
      }
      try {
        localStorage.setItem("saPlanner.saveString", rawVal);
        localStorage.setItem("saPlanner.saveLoadedAt", String(state.saveLoadedAt));
      } catch (e) {}
      var pick = decoded.ab.enemyLevel || decoded.ab.maxEnemyLevel || 4;
      if (pick > (decoded.ab.maxEnemyLevel || pick)) pick = decoded.ab.maxEnemyLevel;
      fillLevels(pick);
      if (!isRestore) {
        simState.config = null;
        simState.lastMcScan = null;
        simState.lastMcSig = null;
        state.levelModeOverrides = {};
        state.dustOverride = "";
        if ($("dustOverride")) $("dustOverride").value = "";
        if ($("dustOverrideWrap")) $("dustOverrideWrap").style.display = "none";
        if (state.customDustRates && state.customDustRates[String(pick)]) {
          delete state.customDustRates[String(pick)];
        }
      }
      autoCalibrateSaveRate();
      if (!isRestore && state.simulatedDustPs > 0) {
        toast("Dust rate auto-calibrated: " + prettify(state.simulatedDustPs) + "/s");
      }
      var curSig = getLoadoutSignature();
      if (state.customDustRates && typeof state.customDustRates === "object") {
        var changed = false;
        for (var k in state.customDustRates) {
          if (state.customDustRates[k] && state.customDustRates[k].sig !== curSig) {
            delete state.customDustRates[k];
            changed = true;
          }
        }
        if (changed) {
          try {
            localStorage.setItem("saPlanner.customRates", JSON.stringify(state.customDustRates));
          } catch (e) {}
        }
      }
      state.filters.contract = !shouldDisableContractsOnLoad(state.level);
      applyFilterBoxes();
      $("workspace").hidden = false;
      renderSaveFacts();
      setSaveCollapsed(true);
      render();
      startLiveBankTimer();
      if (!isRestore) {
        toast("Save loaded · SA Lv " + state.level + " (U2 Z" + (state.radon + 1) + ")");
      }
    } catch (e) {
      $("err").hidden = false;
      $("err").textContent = e.message || String(e);
      setSaveCollapsed(false);
      if (!state.ab) $("workspace").hidden = true;
    }
  }

  function toggleTheme() { setTheme(state.theme === "dark" ? "light" : "dark"); }

  function onDustInput() {
    var raw = $("dustOverride") ? $("dustOverride").value : "";
    var trimmed = raw.trim();
    state.dustOverride = trimmed;
    var curSig = getLoadoutSignature();
    var lvlKey = String(state.level);

    if (!state.customDustRates) state.customDustRates = {};
    if (trimmed) {
      state.customDustRates[lvlKey] = {
        rate: trimmed,
        sig: curSig
      };
    } else {
      delete state.customDustRates[lvlKey];
    }
    try {
      localStorage.setItem("saPlanner.customRates", JSON.stringify(state.customDustRates));
    } catch (e) {}

    if (state.ab) {
      render();
      var dlg = $("presetsDlg");
      if (dlg && dlg.open) renderPresetsModal();
    }
  }

  function onFilterChange() {
    state.filters = readFilters();
    try { localStorage.setItem("saPlanner.filters", JSON.stringify(state.filters)); } catch (e) {}
    if (state.ab) render();
  }

function renderPresetsModal() {
    var body = $("presetsDlgBody");
    if (!body) return;
    if (!state.ab) {
      body.innerHTML = "<p class='empty' style='padding:20px 0;'>No save loaded. Please load a Trimps save to inspect presets, milestones, and diagnostics.</p>";
      return;
    }

    var bnk = bank();
    var presets = getAllPresets();
    var curRec = bestRunnable(state.level);
    var rate = activeDustPs(curRec);
    var diag = combatDiagnostics(rate);
    var audit = saSettingsAudit();
    var msInfo = nextMilestoneInfo();
    var mb = macroBonusInfo();
    var rStats = ringBaseStats(ringLevel());

    // 1. Presets Section
    var presetCardsHtml = presets.map(function (p) {
      var matchInfo = curRec ? comparePresetToBuild(p, curRec) : null;
      var matchBadge = "";
      if (matchInfo) {
        if (matchInfo.isExact) {
          matchBadge = "<div class='preset-status-tag buy'>" + ICONS.check + " Exact match for SA " + state.level + " build</div>";
        } else if (matchInfo.isClose) {
          matchBadge = "<div class='preset-status-tag accent'>" + ICONS.target + " 1 swap from SA " + state.level + " (" + esc(matchInfo.missing.map(itemName).join(", ")) + ")</div>";
        }
      }

      var itemsList = p.items.length
        ? p.items.map(function (k) {
            var isMissingFromRec = curRec && curRec.i && !curRec.i[k];
            var cls = "preset-item-pill" + (isMissingFromRec ? " diff" : "");
            return "<span class='" + cls + "' title='" + esc(itemName(k)) + "'>" + iconForItem(k) + "<span>" + esc(itemName(k)) + "</span></span>";
          }).join("")
        : "<span class='empty' style='font-size:11.5px;'>No items saved</span>";

      var ringModsHtml = p.ringMods.length
        ? "<div class='preset-ring-mods'>" + ICONS.ring + " <span>" + esc(p.ringMods.map(function (m) { return RING_MOD_NAMES[m] || m; }).join(" / ")) + "</span></div>"
        : (hasRing() ? "<div class='preset-ring-mods muted'>" + ICONS.ring + " <span>No ring mods saved</span></div>" : "");

      var lvlBadge = p.level != null ? ("<span class='pill sm'>SA " + p.level + "</span>") : "";
      var simBtn = !p.isEmpty
        ? ("<button type='button' class='btn sm ghost preset-sim-btn' onclick=\"window.saOpenPresetInSim&&saOpenPresetInSim('" + esc(p.slot) + "')\" title='Load " + esc(p.name) + " (" + esc(p.slotTag) + ") into Combat Simulator'>⚔️ Sim</button>")
        : "";

      return "<div class='preset-card" + (p.isEmpty ? " empty-preset" : "") + "'>" +
        "<div class='preset-card-head'>" +
          "<div class='preset-name-wrap'>" +
            "<span class='preset-slot-badge'>" + esc(p.slotTag) + "</span>" +
            "<b class='preset-name' title='" + esc(p.name) + "'>" + esc(p.name) + "</b>" +
          "</div>" +
          "<div class='preset-actions-wrap'>" +
            lvlBadge +
            simBtn +
          "</div>" +
        "</div>" +
        matchBadge +
        "<div class='preset-card-body'>" +
          "<div class='preset-items-wrap'>" + itemsList + "</div>" +
          ringModsHtml +
        "</div>" +
      "</div>";
    }).join("");

    // 2. Settings & Safety Audit Banner
    var settingsBadges = [];
    if (audit.practiceActive) {
      settingsBadges.push("<span class='pill sm sell'>" + ICONS.warn + " Practice Mode ACTIVE (0 Dust/Shards)</span>");
    } else {
      settingsBadges.push("<span class='pill sm buy'>" + ICONS.check + " Practice Mode Off (Normal)</span>");
    }
    if (audit.autoLevel) {
      settingsBadges.push("<span class='pill sm buy'>" + ICONS.check + " AutoLevel ON</span>");
    } else {
      settingsBadges.push("<span class='pill sm warn-pill' title='Huffy will not automatically advance to the next level when current level kill requirement is met'>" + ICONS.warn + " AutoLevel OFF</span>");
    }
    if (audit.loadRingDisabled) {
      settingsBadges.push("<span class='pill sm warn-pill'>" + ICONS.warn + " Load Ring OFF (Won't change mods)</span>");
    } else {
      settingsBadges.push("<span class='pill sm buy'>" + ICONS.check + " Load Ring ON</span>");
    }
    if (audit.loadLevelDisabled) {
      settingsBadges.push("<span class='pill sm muted'>" + ICONS.info + " Load Enemy Level OFF</span>");
    } else {
      settingsBadges.push("<span class='pill sm buy'>" + ICONS.check + " Load Enemy Level ON</span>");
    }

    // 3. Milestones List
    var milestonesHtml = msInfo.list.map(function (m) {
      var statusBadge = m.owned
        ? "<span class='pill sm buy'>" + ICONS.check + " Unlocked</span>"
        : (m.unlocked
          ? "<span class='pill sm accent'>" + ICONS.target + " Ready to Buy</span>"
          : "<span class='pill sm muted'>" + m.toGo + " items away</span>");
      return "<div class='milestone-row" + (m.owned ? " owned" : "") + "'>" +
        "<div class='milestone-req'><span class='mono'>" + m.req + " items</span></div>" +
        "<div class='milestone-info'>" +
          "<b>" + esc(m.name) + "</b>" +
          "<small>" + esc(m.desc) + "</small>" +
        "</div>" +
        "<div class='milestone-status'>" + statusBadge + "</div>" +
      "</div>";
    }).join("");

    body.innerHTML =
      "<div class='dlg-section'>" +
        "<div class='dlg-section-head'>" +
          "<h4>In-Game Presets (ab.presets)</h4>" +
          "<span class='text-muted' style='font-size:12px;'>Comparing against SA " + state.level + " recommendations</span>" +
        "</div>" +
        "<div class='preset-grid'>" + presetCardsHtml + "</div>" +
      "</div>" +

      "<div class='dlg-section'>" +
        "<div class='dlg-section-head'>" +
          "<h4>Game Settings &amp; Combat Diagnostics</h4>" +
        "</div>" +
        "<div class='audit-card'>" +
          "<div class='audit-row'><span class='audit-lbl'>SA In-Game Settings:</span><div class='audit-pills'>" + settingsBadges.join("") + "</div></div>" +
          "<div class='audit-row'><span class='audit-lbl'>Level Combat Status:</span><div class='audit-val' style='display:grid;gap:6px;width:100%;'>" +
            (diag.isPushing
              ? (function () {
                  var liveKillsEarned = (diag.secPerKill && diag.secPerKill > 0 && bnk.elapsedSec > 0)
                    ? Math.min(diag.remainingKills, Math.floor(bnk.elapsedSec / diag.secPerKill))
                    : 0;
                  var liveRemaining = Math.max(0, diag.remainingKills - liveKillsEarned);
                  var liveKillsTotal = diag.curKills + liveKillsEarned;
                  var livePct = diag.neededKills > 0 ? Math.min(100, (liveKillsTotal / diag.neededKills) * 100) : 0;
                  var liveRemSec = Math.max(0, (diag.etaSec || 0) - bnk.elapsedSec);

                  var etaDuration = liveRemSec > 0 ? ("~" + fmtDuration(liveRemSec)) : (diag.etaSec > 0 ? "Ready to clear" : "");
                  var etaClock = liveRemSec > 0 ? fmtLocalEta(liveRemSec) : "";
                  var etaStr = etaDuration ? (etaDuration + (etaClock ? " (ETA " + etaClock + ")" : "")) : "";
                  return "<div id='modalCombatStatus' style='display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:6px;width:100%;'>" +
                    "<div>" +
                      "<span class='pill sm accent'>" + ICONS.target + " Completing SA " + diag.enemyLevel + " (Pushing)</span> " +
                      "<span id='modalCombatKillsTxt'><b>" + prettify(liveKillsTotal) + " / " + prettify(diag.neededKills) + "</b> kills (" + livePct.toFixed(1) + "%) · " +
                      "<b style='color:var(--accent);'>" + prettify(liveRemaining) + " kills to clear</b></span>" +
                    "</div>" +
                    "<div id='modalCombatEta' class='mono' style='color:var(--accent);font-weight:650;font-size:12px;'>" + esc(etaStr) + "</div>" +
                  "</div>" +
                  "<div class='kpi-bar' style='height:6px;width:100%;margin-top:2px;'><span id='modalCombatBar' style='width:" + Math.max(2, livePct).toFixed(1) + "%;background:var(--accent);'></span></div>";
                })()
              : (diag.isFarming
                ? ("<span class='pill sm muted'>" + ICONS.spire + " Farming SA " + diag.enemyLevel + "</span> " +
                   "<span>Farming lower level for dust &amp; shards (Highest unlocked: SA " + diag.maxEnemyLevel + ")</span>")
                : (diag.isMax
                  ? ("<span class='pill sm buy'>" + ICONS.check + " SA Conquered</span> <span>Highest SA level completed (SA " + diag.maxEnemyLevel + ")</span>")
                  : ("<span class='pill sm sell'>" + ICONS.warn + " Practice Mode</span> <span>Combat rewards and level progression paused</span>")))) +
          "</div></div>" +
          "<div class='audit-row'><span class='audit-lbl'>Combat Statistics:</span><span class='audit-val'>" +
            (diag.sessionTotal > 0
              ? (diag.winRateStr + " Win Rate · " + prettify(diag.sessionKills) + " Wins / " + prettify(diag.sessionDeaths) + " Losses · ")
              : "") +
            "Total Enemies Slain: <b>" + prettify(diag.totalKillsAllTime) + "</b> (All-Time)" +
            (diag.totalKillsPortal > 0 ? (" · <b>" + prettify(diag.totalKillsPortal) + "</b> (This Portal)") : "") +
          "</span></div>" +
          "<div class='audit-row'><span class='audit-lbl'>Macro Buffs (U2):</span><span class='audit-val'>" +
            (function () {
              var bList = getBonusList(bnk, rate);
              return bList.map(function (b) {
                var stStr = b.isMax ? "Max" : (b.isReady ? "Upgrade Ready" : (b.statusTxt + " to save"));
                return b.shortName + " Lv " + b.level + (b.maxLevel ? ("/" + b.maxLevel) : "") + " (" + stStr + ")";
              }).join(" · ");
            })() +
          "</span></div>" +
          "<div class='audit-row'><span class='audit-lbl'>The Ring Inherent:</span><span class='audit-val'>+" +
            prettify(rStats.statusDmg) + "% Bleed/Shock · +" +
            prettify(rStats.poisonDmg) + " Poison · +" +
            prettify(rStats.statusChance) + "% Status Chance · " +
            rStats.slots + "/3 slots unlocked" +
            (rStats.nextSlotLvl ? (" (Next at Lv " + rStats.nextSlotLvl + ")") : " (Max slots)") +
          "</span></div>" +
        "</div>" +
      "</div>" +

      "<div class='dlg-section'>" +
        "<div class='dlg-section-head'>" +
          "<h4>One-Timer Milestone Track (" + ownedItemKeys().length + " / " + SA_DATA.itemOrder.length + " Items Owned)</h4>" +
          (msInfo.next ? ("<span class='pill sm accent'>" + ICONS.target + " Next: " + esc(msInfo.next.name) + " in " + msInfo.next.toGo + " items</span>") : "") +
        "</div>" +
        "<div class='milestone-list'>" + milestonesHtml + "</div>" +
      "</div>";
  }

  function openPresetInSim(slot) {
    if (!state.ab || !state.save) {
      toast("Load a Trimps save first to use the Simulator.");
      return;
    }
    var p = parsePreset(slot);
    if (!p || p.isEmpty) {
      toast("Preset " + (p ? p.slotTag : String(slot).toUpperCase()) + " is empty.");
      return;
    }
    var bItems = {};
    p.items.forEach(function (k) {
      bItems[k] = userLevel(k) || 1;
    });
    var simLvl = Number(p.level || state.level || (state.ab && (state.ab.enemyLevel || state.ab.maxEnemyLevel)) || 1);
    var targetBuild = {
      n: p.name + " (" + p.slotTag + ")",
      level: simLvl,
      saLevel: simLvl,
      i: bItems,
      r: {
        lv: ringLevel(),
        m: p.ringMods || []
      },
      l: p.items.length
    };
    var pDlg = $("presetsDlg");
    if (pDlg && typeof pDlg.close === "function") {
      try { pDlg.close(); } catch (e) {}
    }
    openSimModal(targetBuild, { forceReset: true });
    toast("Loaded " + p.name + " (" + p.slotTag + ") into Combat Simulator");
  }
  window.saOpenPresetInSim = openPresetInSim;

  function openPresetsModal() {
    renderPresetsModal();
    var dlg = $("presetsDlg");
    if (dlg) {
      if (dlg.open) return;
      if (typeof dlg.showModal === "function") dlg.showModal();
      else dlg.setAttribute("open", "");
    }
  }

  function closePresetsModal() {
    var dlg = $("presetsDlg");
    if (dlg) {
      if (typeof dlg.close === "function") dlg.close();
      else dlg.removeAttribute("open");
    }
  }


  /* ---------- Combat Simulator & Sandbox Controller ---------- */
  var simState = {
    active: false,
    session: null,
    config: null,
    speed: 10,
    paused: false,
    animId: null,
    pendingTimeout: null,
    lastFrameTs: 0,
    accumulatedMs: 0,
    activeTab: "arena",
    pendingLockBuild: null,
    lastMcScan: null,
    lastMcSig: null,
    addItemMenuOpen: false,
    addItemFilter: "all",
    addItemSearch: ""
  };

  function syncSimEquipped() {
    if (!simState.config || !simState.config.items) return;
    var eq = [];
    var order = (typeof SA_DATA !== "undefined" && SA_DATA.itemOrder) ? SA_DATA.itemOrder : Object.keys(simState.config.items);
    order.forEach(function(k) {
      var it = simState.config.items[k];
      if (it && typeof it === "object" && it.equipped) eq.push(k);
    });
    for (var k in simState.config.items) {
      if (eq.indexOf(k) === -1 && simState.config.items[k] && typeof simState.config.items[k] === "object" && simState.config.items[k].equipped) {
        eq.push(k);
      }
    }
    simState.config.equipped = eq;
  }

  function fillSimLevelSelect(selectedLevel) {
    var sel = $("simLevelSelect");
    if (!sel) return;
    sel.innerHTML = "";
    var lvls = sheetLevels();
    if (lvls.indexOf(selectedLevel) === -1) lvls.push(selectedLevel);
    lvls.sort(function(a, b) { return a - b; });
    lvls.forEach(function(l) {
      var opt = document.createElement("option");
      opt.value = l;
      opt.textContent = "SA " + l + " (" + levelTag(l) + ")";
      if (l === selectedLevel) opt.selected = true;
      sel.appendChild(opt);
    });
    syncCustomSelect("simLevelSelect");
  }

  function openSimModal(targetBuild, options) {
    if (!state.save) {
      toast("Load a Trimps save first to use the Simulator.");
      return;
    }
    var dlg = $("simDlg");
    if (!dlg) return;

    try {
      simState.active = true;
      simState.paused = false;
      closeSimAddItemMenu();

      var shouldResetLoadout = !!targetBuild || !!(options && options.forceReset) || !simState.config;
      if (shouldResetLoadout) {
        var simLvl = (targetBuild && (targetBuild.saLevel || targetBuild.level)) || state.level || (state.ab && (state.ab.enemyLevel || state.ab.maxEnemyLevel)) || 1;
        fillSimLevelSelect(simLvl);

        var cfgSource = targetBuild
          ? { save: state.save, build: targetBuild, level: simLvl }
          : { save: state.save, level: simLvl };
        simState.config = SASimEngine.normalizeConfig(cfgSource);
        if (targetBuild) {
          var maxLimbs = (simState.config.extraLimbs || 0) + 4;
          var curEqCount = simState.config.equipped ? simState.config.equipped.length : 0;
          var simSurplus = Math.max(0, maxLimbs - curEqCount);
          if (simSurplus > 0) {
            var simOpt = findOptimalExtraLimbItems(targetBuild, simLvl, getLevelMode(simLvl), simSurplus);
            if (simOpt && simOpt.bestItems && simOpt.bestItems.length > 0) {
              simOpt.bestItems.forEach(function (bestK) {
                if (simState.config.equipped.indexOf(bestK) === -1) {
                  simState.config.equipped.push(bestK);
                  if (!simState.config.items[bestK]) {
                    simState.config.items[bestK] = { level: userLevel(bestK) || 1, equipped: true };
                  } else {
                    simState.config.items[bestK].equipped = true;
                  }
                }
              });
            }
          }
        }
        syncSimEquipped();
        simState.lastMcScan = null;
        simState.lastMcSig = null;
      } else {
        var persistedLvl = Number(simState.config.level || simState.config.enemyLevel || state.level || 1);
        fillSimLevelSelect(persistedLvl);
        syncSimEquipped();
      }

      var playBtn = $("simBtnPlay");
      if (playBtn) playBtn.textContent = "Pause";

      renderSimSandboxUI();
      startSimSession();
      var targetTab = (options && options.tab) ? options.tab : "arena";
      switchSimTab(targetTab);

      if (typeof dlg.showModal === "function") dlg.showModal();
      else dlg.setAttribute("open", "");

      if (simState.speed === "batch") {
        runSimBatch1K();
      }
    } catch (err) {
      simState.active = false;
      console.error("openSimModal error:", err);
      toast("Simulator error: " + (err.message || String(err)));
    }
  }

  function closeSimModal() {
    simState.active = false;
    simState.paused = true;
    closeCustomSelectMenu(false);
    closeSimAddItemMenu();
    if (simState.animId) {
      cancelAnimationFrame(simState.animId);
      simState.animId = null;
    }
    if (simState.pendingTimeout) {
      clearTimeout(simState.pendingTimeout);
      simState.pendingTimeout = null;
    }
    if (simState.session) {
      simState.session = null;
    }
    var dlg = $("simDlg");
    if (dlg) {
      if (dlg.open && typeof dlg.close === "function") dlg.close();
      else dlg.removeAttribute("open");
    }
  }

  function toggleSimAddItemMenu(forceOpen) {
    var menu = $("simAddItemMenu");
    if (!menu) return;
    var nextState = typeof forceOpen === "boolean" ? forceOpen : !simState.addItemMenuOpen;
    simState.addItemMenuOpen = nextState;
    menu.hidden = !nextState;
    var btn = $("simBtnAddItem");
    if (btn) btn.classList.toggle("accent", nextState);
    if (nextState) {
      renderSimAddItemMenu();
      var searchInp = $("simAddItemSearch");
      if (searchInp) searchInp.focus();
    }
  }

  function closeSimAddItemMenu() {
    simState.addItemMenuOpen = false;
    var menu = $("simAddItemMenu");
    if (menu) menu.hidden = true;
    var btn = $("simBtnAddItem");
    if (btn) btn.classList.remove("accent");
  }

  function renderSimAddItemMenu() {
    var listEl = $("simAddItemList");
    if (!listEl || !simState.config) return;

    var filterMode = simState.addItemFilter || "all";
    var query = (simState.addItemSearch || "").trim().toLowerCase();
    var order = (typeof SA_DATA !== "undefined" && SA_DATA.itemOrder) ? SA_DATA.itemOrder : Object.keys(SA_DATA.items || {});

    var filterBtns = document.querySelectorAll("#simAddItemFilters [data-item-filter]");
    filterBtns.forEach(function(b) {
      b.classList.toggle("active", b.getAttribute("data-item-filter") === filterMode);
    });

    var filtered = order.filter(function(k) {
      var owned = userOwns(k);
      if (filterMode === "owned" && !owned) return false;
      if (filterMode === "unowned" && owned) return false;
      if (query) {
        var nameStr = itemName(k).toLowerCase();
        var keyStr = k.toLowerCase();
        if (nameStr.indexOf(query) === -1 && keyStr.indexOf(query) === -1) return false;
      }
      return true;
    });

    if (!filtered.length) {
      listEl.innerHTML = "<div style='grid-column:1/-1;padding:12px;text-align:center;color:var(--muted);font-size:12px;'>No matching items found.</div>";
      return;
    }

    listEl.innerHTML = filtered.map(function(k) {
      var owned = userOwns(k);
      var saveLv = userLevel(k);
      var cat = itemCat(k);
      var cfgIt = simState.config.items[k];
      var inSim = !!(cfgIt && ((typeof cfgIt === "object" && cfgIt.equipped) || (typeof cfgIt === "number" && Array.isArray(simState.config.equipped) && simState.config.equipped.indexOf(k) !== -1)));
      var simLv = cfgIt ? (typeof cfgIt === "number" ? cfgIt : (cfgIt.level || 1)) : (owned ? Math.max(1, saveLv) : 1);

      var cls = "sim-add-item-option" + (inSim ? " in-sim" : "") + (!owned ? " unowned" : "");
      var badgeHtml = "";
      if (!owned) {
        var zoneTxt = (cat && cat.zone) ? (" · Z" + cat.zone) : "";
        badgeHtml = "<span class='sim-unowned-pill' title='Not yet owned in your loaded save" + (cat && cat.zone ? " (Contract Z" + cat.zone + ")" : "") + "'>Unowned" + esc(zoneTxt) + "</span>";
      } else {
        badgeHtml = "<span class='sim-add-item-meta'>Save Lv " + saveLv + "</span>";
      }

      var statusIndicator = inSim
        ? "<span style='color:var(--buy);font-weight:700;font-size:11px;' title='Equipped in Simulator (Lv " + simLv + ")'>✓ Lv " + simLv + "</span>"
        : "<span style='color:var(--muted);font-size:12px;font-weight:700;'>+</span>";

      return "<button type='button' class='" + cls + "' data-toggle-sim-item='" + esc(k) + "' title='" + esc(itemName(k) + (owned ? " (Owned Lv " + saveLv + ")" : " (Not yet owned in save)")) + "'>" +
        "<span class='sim-item-info'>" +
        iconForItem(k) +
        "<span class='sim-item-name'>" + esc(itemName(k)) + "</span>" +
        "</span>" +
        "<span style='display:inline-flex;align-items:center;gap:6px;flex:none;'>" +
        badgeHtml +
        statusIndicator +
        "</span>" +
        "</button>";
    }).join("");

    listEl.onclick = function(ev) {
      var btn = ev.target.closest("[data-toggle-sim-item]");
      if (!btn || !simState.config) return;
      var k = btn.getAttribute("data-toggle-sim-item");
      var cat = itemCat(k);
      var isUnupgradable = !cat || !!cat.noUp || (cat.max && cat.max <= 1);
      var curIt = simState.config.items[k];
      var currentlyEquipped = !!(curIt && ((typeof curIt === "object" && curIt.equipped) || (typeof curIt === "number" && Array.isArray(simState.config.equipped) && simState.config.equipped.indexOf(k) !== -1)));

      if (currentlyEquipped) {
        if (typeof curIt === "object") {
          curIt.equipped = false;
        } else {
          simState.config.items[k] = { level: curIt || 1, equipped: false };
        }
      } else {
        var defaultLv = 1;
        if (!isUnupgradable) {
          var existingLv = (curIt && (typeof curIt === "number" ? curIt : curIt.level)) || 0;
          defaultLv = existingLv > 0 ? existingLv : (userOwns(k) ? Math.max(1, userLevel(k)) : 1);
        }
        simState.config.items[k] = { level: defaultLv, equipped: true };
      }

      syncSimEquipped();
      renderSimSandboxUI();
      startSimSession();
      if (simState.activeTab === "sensitivity") runUpgradeSensitivity();
    };
  }

  function renderSimSandboxUI() {
    if (!simState.config) return;
    syncSimEquipped();

    // 1. Limbs Pill
    var equippedCount = simState.config.equipped ? simState.config.equipped.length : 0;
    var maxLimbs = (simState.config.extraLimbs || 0) + 4;
    var huffySub = $("simHuffySub");
    if (huffySub) huffySub.textContent = maxLimbs + " Limbs";
    var limbsPill = $("simLimbsPill");
    if (limbsPill) {
      limbsPill.textContent = "Equipped: " + equippedCount + " / " + maxLimbs + " Limbs";
      limbsPill.className = "pill sm " + (equippedCount > maxLimbs ? "warn-pill" : (equippedCount === maxLimbs ? "buy" : ""));
    }

    // 2. Ring Box
    var ringVal = $("simRingLevelVal");
    if (ringVal) ringVal.textContent = "Lv " + simState.config.ring.level;
    var allowedSlots = SASimEngine.unlockedSlotsForLevel(simState.config.ring.level);
    var curMods = (simState.config.ring.mods || []).slice();
    var slotsPill = $("simRingSlotsPill");
    if (slotsPill) {
      slotsPill.textContent = curMods.length + " / " + allowedSlots + " slots used";
      slotsPill.className = "pill sm " + (curMods.length > allowedSlots ? "sell" : "teal");
    }

    var ringModsList = $("simRingModsList");
    if (ringModsList) {
      var allRingMods = ["Attack", "Dust", "Defense", "Health", "Lifesteal"];
      ringModsList.innerHTML = allRingMods.map(function(m) {
        var active = curMods.includes(m);
        return "<span class='sim-ring-mod-chk " + (active ? "active" : "") + "' data-ring-mod='" + esc(m) + "'>" +
          (active ? ICONS.check : "+") + " " + esc(m) + "</span>";
      }).join("");

      ringModsList.onclick = function(ev) {
        var chk = ev.target.closest("span[data-ring-mod]");
        if (!chk) return;
        var modName = chk.getAttribute("data-ring-mod");
        var active = simState.config.ring.mods.includes(modName);
        if (active) {
          simState.config.ring.mods = simState.config.ring.mods.filter(function(m) { return m !== modName; });
        } else {
          var allowed = SASimEngine.unlockedSlotsForLevel(simState.config.ring.level);
          if (simState.config.ring.mods.length >= allowed) {
            simState.config.ring.mods.shift(); // Remove oldest
          }
          simState.config.ring.mods.push(modName);
        }
        renderSimSandboxUI();
        startSimSession();
      };
    }

    // 3. Items Grid (filtered to active items being simulated)
    var grid = $("simItemsGrid");
    if (grid) {
      var order = SA_DATA.itemOrder || Object.keys(simState.config.items);
      var activeKeys = order.filter(function(k) {
        var it = simState.config.items[k];
        if (!it) return false;
        if (typeof it === "object") return !!it.equipped;
        if (typeof it === "number") return Array.isArray(simState.config.equipped) && simState.config.equipped.indexOf(k) !== -1;
        return false;
      });
      grid.innerHTML = activeKeys.map(function(k) {
        var item = simState.config.items[k];
        var lv = typeof item === "number" ? item : (item.level || 0);
        var sh = isShards(k);
        var cat = itemCat(k);
        var owned = userOwns(k);
        var isUnupgradable = !cat || !!cat.noUp || (cat.max && cat.max <= 1);
        if (isUnupgradable) {
          lv = 1;
          if (typeof item === "number") simState.config.items[k] = 1;
          else if (item && typeof item === "object") item.level = 1;
        }

        var stepperHtml = isUnupgradable
          ? "<span class='sim-step-val " + (sh ? "tone-teal" : "") + "' style='min-width:32px;padding:0 4px;'>Lv 1</span>"
          : "<button type='button' class='sim-step-btn' data-item-minus='" + esc(k) + "' title='Decrease level'>−</button>" +
            "<span class='sim-step-val " + (sh ? "tone-teal" : "") + "'>Lv " + lv + "</span>" +
            "<button type='button' class='sim-step-btn' data-item-plus='" + esc(k) + "' title='Increase level'>+</button>";

        var unownedBadge = !owned
          ? "<span class='sim-unowned-pill' title='Not yet owned in your loaded save" + (cat && cat.zone ? " (Contract Z" + cat.zone + ")" : "") + "'>Unowned</span>"
          : "";
        var rowCls = "sim-item-row equipped" + (!owned ? " unowned" : "");
        var infoTitle = itemName(k) + (owned ? " (Owned Lv " + userLevel(k) + " in save)" : " — NOT OWNED in save" + (cat && cat.zone ? " (Contract Z" + cat.zone + ")" : ""));

        return "<div class='" + rowCls + "' data-item-key='" + esc(k) + "'>" +
          "<div class='sim-item-info' title='" + esc(infoTitle) + "'>" +
          iconForItem(k) +
          "<span class='sim-item-name'>" + esc(itemName(k)) + "</span>" +
          unownedBadge +
          "</div>" +
          "<div class='sim-stepper'>" +
          stepperHtml +
          "<button type='button' class='sim-item-remove' data-item-remove='" + esc(k) + "' title='Remove item from sandbox loadout'>×</button>" +
          "</div>" +
          "</div>";
      }).join("");

      grid.onclick = function(ev) {
        var removeTarget = ev.target.closest("[data-item-remove]");
        if (removeTarget) {
          var rk = removeTarget.getAttribute("data-item-remove");
          if (simState.config.items[rk]) {
            if (typeof simState.config.items[rk] === "object") {
              simState.config.items[rk].equipped = false;
            } else {
              simState.config.items[rk] = { level: simState.config.items[rk] || 1, equipped: false };
            }
            syncSimEquipped();
            renderSimSandboxUI();
            startSimSession();
            if (simState.activeTab === "sensitivity") runUpgradeSensitivity();
          }
          return;
        }

        var minusTarget = ev.target.closest("[data-item-minus]");
        if (minusTarget) {
          var k = minusTarget.getAttribute("data-item-minus");
          var cat = itemCat(k);
          if (cat && (cat.noUp || (cat.max && cat.max <= 1))) return;
          if (simState.config.items[k]) {
            if (typeof simState.config.items[k] === "number" && simState.config.items[k] > 1) {
              simState.config.items[k]--;
            } else if (simState.config.items[k].level > 1) {
              simState.config.items[k].level--;
            }
            renderSimSandboxUI();
            startSimSession();
            if (simState.activeTab === "sensitivity") runUpgradeSensitivity();
          }
          return;
        }

        var plusTarget = ev.target.closest("[data-item-plus]");
        if (plusTarget) {
          var k = plusTarget.getAttribute("data-item-plus");
          var cat = itemCat(k);
          if (cat && (cat.noUp || (cat.max && cat.max <= 1))) return;
          var curVal = (typeof simState.config.items[k] === "number") ? simState.config.items[k] : ((simState.config.items[k] && simState.config.items[k].level) || 0);
          if (cat && cat.max && curVal >= cat.max) return;
          if (!simState.config.items[k]) simState.config.items[k] = { level: 0, equipped: true };
          if (typeof simState.config.items[k] === "number") {
            simState.config.items[k]++;
          } else {
            simState.config.items[k].level = (simState.config.items[k].level || 0) + 1;
            simState.config.items[k].equipped = true;
          }
          syncSimEquipped();
          renderSimSandboxUI();
          startSimSession();
          if (simState.activeTab === "sensitivity") runUpgradeSensitivity();
          return;
        }
      };
    }

    if (simState.addItemMenuOpen) {
      renderSimAddItemMenu();
    }
  }

  function startSimSession() {
    if (!simState.config) return;
    simState.session = SASimEngine.createSession(simState.config);
    simState.lastFrameTs = 0;
    simState.accumulatedMs = 0;
    var snap = simState.session.getState();
    updateSimUI(snap);
    clearSimLog();
    addSimLogMsg("0.0s", "Combat session initialized for SA " + simState.config.level, "init");
    if (!simState.animId && simState.active) {
      simState.animId = requestAnimationFrame(simAnimTick);
    }
  }

  function simAnimTick(ts) {
    var dlg = $("simDlg");
    if (!simState.active || !dlg || !dlg.open) {
      simState.active = false;
      if (simState.animId) {
        cancelAnimationFrame(simState.animId);
        simState.animId = null;
      }
      return;
    }
    if (!simState.lastFrameTs) simState.lastFrameTs = ts;
    var deltaRealMs = Math.min(100, ts - simState.lastFrameTs);
    simState.lastFrameTs = ts;

    if (!simState.paused && simState.session && simState.speed !== "batch") {
      var speedMult = Number(simState.speed) || 10;
      var simulatedCombatMs = deltaRealMs * speedMult;
      simState.accumulatedMs += simulatedCombatMs;

      var ticksToRun = Math.floor(simState.accumulatedMs / 300);
      if (ticksToRun > 0) {
        if (ticksToRun > 60) ticksToRun = 60;
        simState.accumulatedMs -= (ticksToRun * 300);

        for (var t = 0; t < ticksToRun; t++) {
          simState.session.step();
        }
        var snap = simState.session.getState();
        updateSimUI(snap);
      }
    }

    if (simState.active && dlg && dlg.open) {
      simState.animId = requestAnimationFrame(simAnimTick);
    } else {
      simState.animId = null;
    }
  }

  function updateSimUI(snap) {
    if (!snap) return;

    // Top Summary
    var sDust = $("simStatDust");
    if (sDust) sDust.textContent = prettify(snap.dustPs || 0) + "/s";
    var sShards = $("simStatShards");
    if (sShards) sShards.textContent = prettify(snap.shardsPs || 0) + "/s";
    var sWr = $("simStatWinRate");
    if (sWr) sWr.textContent = (snap.winRate != null ? snap.winRate.toFixed(1) + "%" : "--");
    var sTk = $("simStatKillTime");
    if (sTk) sTk.textContent = (snap.timePerKill != null && snap.timePerKill > 0 ? snap.timePerKill.toFixed(2) + "s" : "--");
    var sFg = $("simStatFights");
    if (sFg) sFg.textContent = (snap.wins || 0) + " / " + (snap.losses || 0);

    // Huffy
    var hHpBar = $("simHuffyHpBar");
    if (hHpBar) hHpBar.style.width = Math.max(0, Math.min(100, (snap.huffyHp / Math.max(1, snap.huffyMaxHp)) * 100)) + "%";
    var hHpTxt = $("simHuffyHpTxt");
    if (hHpTxt) hHpTxt.textContent = prettify(snap.huffyHp) + " / " + prettify(snap.huffyMaxHp);

    var hAtkPct = snap.huffyMaxAtkTimer > 0 ? Math.max(0, Math.min(100, ((snap.huffyMaxAtkTimer - snap.huffyAtkTimer) / snap.huffyMaxAtkTimer) * 100)) : 0;
    var hAtkBar = $("simHuffyAtkBar");
    if (hAtkBar) hAtkBar.style.width = hAtkPct + "%";
    var hAtkTxt = $("simHuffyAtkTxt");
    if (hAtkTxt) hAtkTxt.textContent = (snap.huffyAtkTimer / 1000).toFixed(2) + "s / " + (snap.huffyMaxAtkTimer / 1000).toFixed(2) + "s";

    var hAtk = $("simHuffyStatAtk");
    if (hAtk) hAtk.textContent = prettify(snap.huffyAtk);
    var hDef = $("simHuffyStatDef");
    if (hDef) hDef.textContent = prettify(snap.huffyDef);
    var hLs = $("simHuffyStatLs");
    if (hLs) hLs.textContent = (snap.huffyLs || 0) + "%";
    var hDm = $("simHuffyStatDustMult");
    if (hDm) hDm.textContent = (snap.huffyDustMult || 0) + "%";

    var hDebuffs = $("simHuffyDebuffs");
    if (hDebuffs) {
      var dHtml = [];
      if (snap.huffyBuffs) {
        if (snap.huffyBuffs.bleed) { var bAmp = snap.huffyBuffs.bleed.amp != null ? snap.huffyBuffs.bleed.amp : Math.round((snap.huffyBuffs.bleed.dmg || 0) * 100); dHtml.push("<span class='sim-debuff-pill bleed'>Bleed: +" + bAmp + "%</span>"); }
        if (snap.huffyBuffs.poison) { var pAmp = snap.huffyBuffs.poison.amp != null ? snap.huffyBuffs.poison.amp : Math.round((snap.huffyBuffs.poison.stacks || 0) * 100); dHtml.push("<span class='sim-debuff-pill poison'>Poison: +" + pAmp + "%</span>"); }
        if (snap.huffyBuffs.shock) dHtml.push("<span class='sim-debuff-pill shock'>Shock: +" + (snap.huffyBuffs.shock.amp || 0) + "%</span>");
      }
      hDebuffs.innerHTML = dHtml.join("");
    }

    // Center live timer & average
    var fTime = $("simFightTime");
    if (fTime) {
      fTime.textContent = (snap.currentFightSec || 0).toFixed(1) + "s";
      fTime.title = "Current fight elapsed time";
    }
    var aTime = $("simAvgTime");
    if (aTime) {
      if (snap.avgLast100Sec != null && snap.avgLast100Sec > 0) {
        aTime.textContent = snap.avgLast100Sec.toFixed(2) + "s avg";
      } else {
        aTime.textContent = "-- avg";
      }
      aTime.title = "Average kill duration over completed fights";
    }

    // Enemy
    var eName = $("simEnemyName");
    if (eName) eName.textContent = snap.enemyName || "Enemy";
    var eSub = $("simEnemySub");
    if (eSub) eSub.textContent = "SA " + (snap.enemyLevel || simState.config.level);

    var eHpBar = $("simEnemyHpBar");
    if (eHpBar) eHpBar.style.width = Math.max(0, Math.min(100, (snap.enemyHp / Math.max(1, snap.enemyMaxHp)) * 100)) + "%";
    var eHpTxt = $("simEnemyHpTxt");
    if (eHpTxt) eHpTxt.textContent = prettify(snap.enemyHp) + " / " + prettify(snap.enemyMaxHp);

    var eAtkPct = snap.enemyMaxAtkTimer > 0 ? Math.max(0, Math.min(100, ((snap.enemyMaxAtkTimer - snap.enemyAtkTimer) / snap.enemyMaxAtkTimer) * 100)) : 0;
    var eAtkBar = $("simEnemyAtkBar");
    if (eAtkBar) eAtkBar.style.width = eAtkPct + "%";
    var eAtkTxt = $("simEnemyAtkTxt");
    if (eAtkTxt) eAtkTxt.textContent = (snap.enemyAtkTimer / 1000).toFixed(2) + "s / " + (snap.enemyMaxAtkTimer / 1000).toFixed(2) + "s";

    var eAtk = $("simEnemyStatAtk");
    if (eAtk) eAtk.textContent = prettify(snap.enemyAtk);
    var eDef = $("simEnemyStatDef");
    if (eDef) eDef.textContent = prettify(snap.enemyDef);
    var eLs = $("simEnemyStatLs");
    if (eLs) eLs.textContent = (snap.enemyLs || 0) + "%";
    var eEnrage = $("simEnemyStatEnrage");
    if (eEnrage) eEnrage.textContent = snap.enemyEnrageTimer > 0 ? (snap.enemyEnrageTimer / 1000).toFixed(0) + "s" : "ENRAGED";

    var eDebuffs = $("simEnemyDebuffs");
    if (eDebuffs) {
      var edHtml = [];
      if (snap.enemyBuffs) {
        if (snap.enemyBuffs.bleed) { var ebAmp = snap.enemyBuffs.bleed.amp != null ? snap.enemyBuffs.bleed.amp : Math.round((snap.enemyBuffs.bleed.dmg || 0) * 100); edHtml.push("<span class='sim-debuff-pill bleed'>Bleed: +" + ebAmp + "%</span>"); }
        if (snap.enemyBuffs.poison) { var epAmp = snap.enemyBuffs.poison.amp != null ? snap.enemyBuffs.poison.amp : Math.round((snap.enemyBuffs.poison.stacks || 0) * 100); edHtml.push("<span class='sim-debuff-pill poison'>Poison: +" + epAmp + "%</span>"); }
        if (snap.enemyBuffs.shock) edHtml.push("<span class='sim-debuff-pill shock'>Shock: +" + (snap.enemyBuffs.shock.amp || 0) + "%</span>");
      }
      eDebuffs.innerHTML = edHtml.join("");
    }

    // Recent Log Events
    if (snap.recentEvents && snap.recentEvents.length) {
      snap.recentEvents.forEach(function(ev) {
        addSimLogMsg((snap.time || 0).toFixed(1) + "s", ev.msg, ev.type);
      });
      snap.recentEvents.length = 0;
    }
  }

  function clearSimLog() {
    var logWrap = $("simCombatLog");
    if (logWrap) logWrap.innerHTML = "";
  }

  function addSimLogMsg(timeStr, msg, type) {
    var logWrap = $("simCombatLog");
    if (!logWrap) return;
    var line = document.createElement("div");
    line.className = "sim-log-line";
    var safeType = /^(init|hit|crit|bleed|poison|shock|kill|death|info|warn)$/.test(type) ? type : "";
    line.innerHTML = "<span class='sim-log-time'>" + esc(timeStr) + "</span><span class='sim-log-msg " + safeType + "'>" + esc(msg) + "</span>";
    if (typeof logWrap.insertBefore === "function") {
      logWrap.insertBefore(line, logWrap.firstChild);
    } else if (typeof logWrap.appendChild === "function") {
      logWrap.appendChild(line);
    }
    while (logWrap.children && logWrap.children.length > 50) {
      if (typeof logWrap.removeChild === "function" && logWrap.lastChild) {
        logWrap.removeChild(logWrap.lastChild);
      } else {
        break;
      }
    }
  }

  function setSimSpeed(speed) {
    simState.speed = speed;
    var btns = document.querySelectorAll("#simSpeedGroup button");
    btns.forEach(function (b) {
      b.classList.toggle("active", b.getAttribute("data-speed") === String(speed));
    });
    if (speed === "batch") {
      runSimBatch1K();
    }
  }

  function toggleSimPlay() {
    simState.paused = !simState.paused;
    var btn = $("simBtnPlay");
    if (btn) btn.textContent = simState.paused ? "Resume" : "Pause";
  }

  function stepSimFrame() {
    if (!simState.session) return;
    simState.paused = true;
    var btn = $("simBtnPlay");
    if (btn) btn.textContent = "Resume";
    simState.session.step();
    updateSimUI(simState.session.getState());
  }

  function resetSimBattle() {
    if (!simState.session) return;
    simState.session.reset();
    simState.lastFrameTs = 0;
    simState.accumulatedMs = 0;
    updateSimUI(simState.session.getState());
    clearSimLog();
    addSimLogMsg("0.0s", "Battle reset to 0s", "init");
  }

  function resetSimToSave() {
    if (!state.save) return;
    var lvl = (simState.config && simState.config.level) || state.level;
    simState.config = SASimEngine.normalizeConfig({ save: state.save, level: lvl });
    syncSimEquipped();
    simState.lastMcScan = null;
    simState.lastMcSig = null;
    renderSimSandboxUI();
    startSimSession();
    toast("Sandbox reset to your live save loadout");
  }

  function runSimBatch1K() {
    if (!simState.session) return;
    simState.session.reset(true);
    var targetFights = 1000;
    var maxTicks = 200000;
    var ticks = 0;
    while ((simState.session.ab.sessionEnemiesKilled + simState.session.ab.sessionTrimpsKilled) < targetFights && ticks < maxTicks) {
      simState.session.step();
      ticks++;
    }
    var snap = simState.session.getState();
    updateSimUI(snap);
    var tStr = (snap.time != null ? snap.time.toFixed(1) : "0.0") + "s";
    addSimLogMsg(tStr, "Batch complete: 1,000 fights simulated (" + ticks + " ticks).", "kill");
  }

  function switchSimTab(tab) {
    simState.activeTab = tab;
    var tabMap = {
      arena: $("simPanelArena"),
      montecarlo: $("simPanelMonteCarlo") || $("simPanelMontecarlo"),
      ringopt: $("simPanelRingOpt") || $("simPanelRingopt"),
      sensitivity: $("simPanelSensitivity")
    };
    for (var t in tabMap) {
      var btn = document.querySelector("#simTabs button[data-tab='" + t + "']");
      if (btn) btn.classList.toggle("active", t === tab);
      var panel = tabMap[t];
      if (panel) panel.style.display = t === tab ? (t === "arena" ? "block" : "flex") : "none";
    }

    if (tab === "montecarlo") {
      runMonteCarloPush(false);
    } else if (tab === "ringopt") {
      runRingOptimizer();
    } else if (tab === "sensitivity") {
      runUpgradeSensitivity();
    }
  }

  function onSimLevelChange() {
    var sel = $("simLevelSelect");
    if (!sel || !simState.config) return;
    var newLvl = Number(sel.value) || 1;
    simState.config.level = newLvl;
    simState.config.enemyLevel = newLvl;
    renderSimSandboxUI();
    startSimSession();
    if (simState.activeTab === "sensitivity") {
      runUpgradeSensitivity();
    } else if (simState.activeTab === "montecarlo") {
      runMonteCarloPush(false);
    } else if (simState.activeTab === "ringopt") {
      runRingOptimizer();
    }
  }

  function onSimLockToPlan() {
    if (!simState.config) return;
    var lvl = simState.config.level || state.level;
    var allSheetBuilds = buildsFor(lvl);

    syncSimEquipped();
    var equippedKeys = simState.config.equipped || [];
    var matchedBuild = null;
    for (var i = 0; i < allSheetBuilds.length; i++) {
      var sb = allSheetBuilds[i];
      var itemKeys = Object.keys(sb.i || {});
      if (itemKeys.length !== equippedKeys.length) continue;
      var match = true;
      for (var j = 0; j < itemKeys.length; j++) {
        var k = itemKeys[j];
        var cfgItem = simState.config.items[k];
        if (!cfgItem || !cfgItem.equipped || cfgItem.level < sb.i[k]) {
          match = false;
          break;
        }
      }
      if (match) {
        matchedBuild = sb;
        break;
      }
    }

    if (matchedBuild) {
      toggleLockBuild(matchedBuild);
      closeSimModal();
      return;
    }

    var defaultName = "Custom SA " + lvl + " Build";
    simState.pendingLockBuild = {
      n: defaultName,
      d: (simState.session && Math.round(simState.session.getState().dustPs)) || 0,
      l: Object.keys(simState.config.items).filter(function(k) { return simState.config.items[k] && simState.config.items[k].equipped; }).length,
      r: { lv: simState.config.ring.level, m: (simState.config.ring.mods || []).slice() },
      i: (function() {
        var map = {};
        for (var k in simState.config.items) {
          if (simState.config.items[k] && simState.config.items[k].equipped) {
            map[k] = simState.config.items[k].level || 1;
          }
        }
        return map;
      })(),
      saLevel: lvl
    };

    var promptDlg = $("simLockPromptDlg");
    if (promptDlg) {
      $("simCustomPlanName").value = defaultName;
      if (typeof promptDlg.showModal === "function") promptDlg.showModal();
      else promptDlg.setAttribute("open", "");
    }
  }

  function confirmSimCustomLock() {
    var promptDlg = $("simLockPromptDlg");
    var nameInput = $("simCustomPlanName");
    var name = (nameInput && nameInput.value.trim()) || (simState.pendingLockBuild && simState.pendingLockBuild.n) || "Custom SA Build";
    if (simState.pendingLockBuild) {
      simState.pendingLockBuild.n = name;
      toggleLockBuild(simState.pendingLockBuild);
      simState.pendingLockBuild = null;
    }
    if (promptDlg) {
      if (typeof promptDlg.close === "function") promptDlg.close();
      else promptDlg.removeAttribute("open");
    }
    closeSimModal();
  }

  function closeSimLockPrompt() {
    simState.pendingLockBuild = null;
    var promptDlg = $("simLockPromptDlg");
    if (promptDlg) {
      if (typeof promptDlg.close === "function") promptDlg.close();
      else promptDlg.removeAttribute("open");
    }
  }

  function getSimBuildSignature(cfg) {
    if (!cfg) return "";
    var items = [];
    if (cfg.items) {
      for (var k in cfg.items) {
        var it = cfg.items[k];
        if (typeof it === "object" && it.equipped) {
          items.push(k + ":" + (it.level || 0));
        } else if (typeof it === "number" && it > 0) {
          items.push(k + ":" + it);
        }
      }
    }
    items.sort();
    var ring = cfg.ring ? (cfg.ring.level + "|" + (cfg.ring.mods || []).slice().sort().join(",")) : "";
    return items.join(";") + "||" + ring;
  }

  function renderMonteCarloHeatmap(scan, resDiv) {
    if (!scan || !scan.levels || !scan.levels.length) {
      resDiv.innerHTML = "<p class='empty'>No push simulation results generated.</p>";
      return;
    }

    var maxWin = scan.highestWinLevel;
    var max50 = scan.maxWin50;

    // Top summary action banner
    var headerHtml = "<div style='display:flex;align-items:center;justify-content:space-between;flex-wrap:wrap;gap:12px;padding:12px 16px;background:var(--card-bg,#1e232d);border:1px solid var(--border,#2d3748);border-radius:8px;margin-bottom:14px;'>" +
      "<div style='display:flex;align-items:center;gap:16px;flex-wrap:wrap;'>" +
      "<div>" +
      "<div style='font-size:11px;color:var(--muted);text-transform:uppercase;letter-spacing:0.5px;font-weight:600;'>Highest Winning Level</div>" +
      "<div style='font-size:16px;font-weight:700;color:var(--text);'>SA " + maxWin + " <span style='font-size:11.5px;font-weight:normal;color:var(--muted);'>(≥1 Win)</span></div>" +
      "</div>" +
      "<div style='border-left:1px solid var(--border,#2d3748);padding-left:16px;'>" +
      "<div style='font-size:11px;color:var(--muted);text-transform:uppercase;letter-spacing:0.5px;font-weight:600;'>Max Stable Level (≥50% WR)</div>" +
      "<div style='font-size:16px;font-weight:700;color:var(--buy,#10b981);'>SA " + max50 + "</div>" +
      "</div>" +
      "</div>" +
      "<button type='button' class='btn sm accent' id='simBtnSetToMaxWin' data-set-level='" + max50 + "' style='font-weight:600;padding:6px 14px;box-shadow:0 0 10px rgba(247,147,26,0.3);'>" +
      "🎯 Set Simulator to SA " + max50 + " (Max ≥50% Level)" +
      "</button>" +
      "</div>";

    // Heatmap Table
    var rows = scan.levels.map(function(r) {
      var wr = r.winRate;
      var tone = wr >= 99 ? "tone-buy" : (wr >= 50 ? "tone-warn" : (wr > 0 ? "tone-warn" : "tone-bad"));
      var barColor = wr >= 99 ? "var(--buy,#10b981)" : (wr >= 80 ? "#14b8a6" : (wr >= 50 ? "var(--warn,#f59e0b)" : (wr > 0 ? "#f97316" : "var(--bad,#ef4444)")));
      var isCur = simState.config && simState.config.level === r.targetLevel;

      var clearTimeHtml = "Unclearable";
      var enemyDef = SASimEngine.getEnemyDef(r.targetLevel);
      var neededKills = enemyDef ? enemyDef.neededKills : 1000;
      var curKills = (state.ab && state.ab.enemyLevel === r.targetLevel) ? (state.ab.enemyKills || 0) : 0;
      var remKills = Math.max(0, neededKills - curKills);
      if (wr > 0 && r.timePerKill > 0) {
        var effectiveSecPerKill = r.timePerKill / (wr / 100);
        clearTimeHtml = "~" + fmtTime(remKills * effectiveSecPerKill);
      }

      return "<tr" + (isCur ? " style='background:rgba(247,147,26,0.1);font-weight:600;'" : "") + ">" +
        "<td><b>SA " + r.targetLevel + "</b>" + (isCur ? " <span class='pill sm' style='margin-left:4px;'>Current</span>" : "") + (r.targetLevel === max50 ? " <span class='pill sm buy' style='margin-left:4px;'>Max Stable</span>" : "") + "</td>" +
        "<td style='min-width:140px;'>" +
        "<div style='display:flex;align-items:center;gap:8px;'>" +
        "<div style='flex:1;background:rgba(255,255,255,0.08);height:8px;border-radius:4px;overflow:hidden;'>" +
        "<div style='width:" + wr.toFixed(1) + "%;background:" + barColor + ";height:100%;border-radius:4px;'></div>" +
        "</div>" +
        "<span class='mono " + tone + "' style='font-weight:700;width:52px;text-align:right;'>" + wr.toFixed(1) + "%</span>" +
        "</div>" +
        "</td>" +
        "<td class='mono'>" + r.wins + " / " + r.fights + "</td>" +
        "<td class='mono'>" + r.avgFightDuration.toFixed(1) + "s</td>" +
        "<td class='mono'>" + (wr > 0 ? "<span class='tone-buy'>" + clearTimeHtml + "</span>" : "<span class='tone-bad'>Unclearable</span>") + "</td>" +
        "<td><button type='button' class='btn sm ghost' data-set-level='" + r.targetLevel + "'>Test SA " + r.targetLevel + "</button></td>" +
        "</tr>";
    }).join("");

    var tableHtml = "<div class='tbl-wrap'><table class='sim-ext-table'><thead><tr>" +
      "<th>Level</th><th>Win Rate (Heatmap)</th><th>Wins / 500</th><th>Avg Fight Time</th><th>Est. Clear Time</th><th>Action</th>" +
      "</tr></thead><tbody>" + rows + "</tbody></table></div>";

    resDiv.innerHTML = headerHtml + tableHtml;

    resDiv.onclick = function(ev) {
      var btn = ev.target.closest("[data-set-level]");
      if (!btn) return;
      var targetLvl = Number(btn.getAttribute("data-set-level"));
      if (targetLvl && simState.config) {
        simState.config.level = targetLvl;
        simState.config.enemyLevel = targetLvl;
        if ($("simLevelSelect")) {
          $("simLevelSelect").value = String(targetLvl);
          syncCustomSelect("simLevelSelect");
        }
        renderSimSandboxUI();
        startSimSession();
        switchSimTab("arena");
        toast("Simulator set to SA " + targetLvl);
      }
    };
  }

  function runMonteCarloPush(force) {
    if (!simState.config || !simState.active) return;
    var resDiv = $("simMcResults");
    if (!resDiv) return;

    var curSig = getSimBuildSignature(simState.config);
    if (!force && simState.lastMcScan && simState.lastMcSig === curSig) {
      renderMonteCarloHeatmap(simState.lastMcScan, resDiv);
      return;
    }

    resDiv.style.display = "block";
    resDiv.innerHTML = "<div style='padding:24px;text-align:center;color:var(--muted);'><div class='spin' style='font-size:24px;margin-bottom:8px;'>🎲</div>Scanning all SA levels for push viability heatmap (500 battles / level)...</div>";

    if (simState.pendingTimeout) clearTimeout(simState.pendingTimeout);
    simState.pendingTimeout = setTimeout(function () {
      simState.pendingTimeout = null;
      if (!simState.active) return;
      try {
        var activeLvl = (simState.config && (simState.config.level || simState.config.enemyLevel)) ||
          (state.ab && (state.ab.enemyLevel || state.ab.maxEnemyLevel)) ||
          state.level || 1;
        var scan = SASimEngine.scanPushHeatmap(simState.config, { fights: 500, activeLevel: activeLvl });
        if (!scan || !scan.levels || !scan.levels.length) {
          resDiv.innerHTML = "<p class='empty'>No push simulation results generated.</p>";
          return;
        }

        simState.lastMcScan = scan;
        simState.lastMcSig = curSig;
        renderMonteCarloHeatmap(scan, resDiv);
      } catch (e) {
        resDiv.innerHTML = "<div style='color:var(--bad);padding:10px;'>Push heatmap error: " + esc(e.message || String(e)) + "</div>";
      }
    }, 10);
  }

  function runRingOptimizer() {
    if (!simState.config || !simState.active) return;
    var resDiv = $("simRingOptResults");
    if (resDiv) {
      resDiv.innerHTML = "<div style='padding:20px;text-align:center;color:var(--muted);'>Simulating all valid Ring mod combinations...</div>";
    }

    if (simState.pendingTimeout) clearTimeout(simState.pendingTimeout);
    simState.pendingTimeout = setTimeout(function () {
      simState.pendingTimeout = null;
      if (!simState.active) return;
      try {
        var results = SASimEngine.optimizeRing(simState.config, { maxKills: 20 });
        if (!results || !results.length) {
          resDiv.innerHTML = "<p class='empty'>No valid ring permutations found for your Ring level.</p>";
          return;
        }

        var curModsStr = (simState.config.ring.mods || []).slice().sort().join(" / ");
        var curEntry = null;
        for (var ri = 0; ri < results.length; ri++) {
          if (results[ri].mods.slice().sort().join(" / ") === curModsStr) {
            curEntry = results[ri];
            break;
          }
        }
        var curRate = curEntry ? curEntry.dustPs : ((simState.session && simState.session.getState().dustPs) || 0);
        var hasBetter = results.some(function (x) { return x.dustPs > curRate + 0.5; });
        var currentPillCls = hasBetter ? "sell" : "buy";

        var rows = results.map(function (r, idx) {
          var modsStr = r.mods.join(" / ");
          var isCurrent = modsStr === curModsStr;
          var diff = r.dustPs - curRate;
          var diffStr = "=";
          var diffCls = "muted";
          if (!isCurrent && Math.abs(diff) >= 1) {
            diffStr = diff > 0 ? ("+" + prettify(diff) + "/s") : ("-" + prettify(Math.abs(diff)) + "/s");
            diffCls = diff > 0 ? "pos" : "neg";
          }

          return "<tr" + (isCurrent ? " style='background:rgba(247,147,26,0.1);font-weight:600;'" : "") + ">" +
            "<td>#" + (idx + 1) + "</td>" +
            "<td><b>" + esc(modsStr) + "</b>" + (isCurrent ? " <span class='pill sm " + currentPillCls + "' style='margin-left:4px;'>Current</span>" : "") + "</td>" +
            "<td class='mono'><b>" + prettify(r.dustPs) + "/s</b></td>" +
            "<td class='mono " + diffCls + "'>" + esc(diffStr) + "</td>" +
            "<td>" + r.winRate.toFixed(0) + "%</td>" +
            "<td>" + (r.timePerKill ? r.timePerKill.toFixed(2) + "s" : "--") + "</td>" +
            "<td><button type='button' class='btn sm ghost' data-apply-mods='" + esc(r.mods.join(",")) + "'>Apply to Sandbox</button></td>" +
            "</tr>";
        }).join("");

        resDiv.innerHTML = "<div class='tbl-wrap'><table class='sim-ext-table'><thead><tr><th>Rank</th><th>Ring Mods</th><th>Simulated Dust/s</th><th>Δ vs Current</th><th>Win Rate</th><th>TTK</th><th>Action</th></tr></thead><tbody>" + rows + "</tbody></table></div>";

        resDiv.onclick = function (ev) {
          var btn = ev.target.closest("button[data-apply-mods]");
          if (!btn) return;
          var mods = btn.getAttribute("data-apply-mods").split(",");
          simState.config.ring.mods = mods;
          renderSimSandboxUI();
          startSimSession();
          runUpgradeSensitivity();
          toast("Applied Ring mods: " + mods.join(" / "));
        };
      } catch (e) {
        resDiv.innerHTML = "<div style='color:var(--bad);padding:10px;'>Ring optimizer error: " + esc(e.message || String(e)) + "</div>";
      }
    }, 10);
  }

  function runUpgradeSensitivity() {
    if (!simState.config || !simState.active) return;
    var resDiv = $("simSensitivityResults");
    if (resDiv) {
      resDiv.innerHTML = "<div style='padding:20px;text-align:center;color:var(--muted);'>Simulating upgrade sensitivity on equipped items and ring...</div>";
    }

    if (simState.pendingTimeout) clearTimeout(simState.pendingTimeout);
    simState.pendingTimeout = setTimeout(function () {
      simState.pendingTimeout = null;
      if (!simState.active) return;
      try {
        var results = SASimEngine.evaluateUpgradeROI(simState.config, { maxKills: 20 });
        if (!results || !results.length) {
          resDiv.innerHTML = "<p class='empty'>No upgrade candidates available to test.</p>";
          return;
        }

        var bnk = bank();
        var earnRate = activeDustPs() || (state.simulatedDustPs || 0);
        if (!earnRate || earnRate <= 0) {
          var rec = bestRunnable(state.level);
          earnRate = (rec && rec.d) ? rec.d : 0;
        }

        var rows = results.map(function (r, idx) {
          var cost = Number(r.cost) || 0;
          var isSh = !!r.shards;
          var resName = isSh ? "shards" : "dust";
          var curFunds = isSh ? (bnk.shards || 0) : (bnk.dust || 0);
          var isReady = (cost <= 0) || (curFunds >= cost);
          var costTxt = cost ? (prettify(cost) + " " + resName) : "Free";

          var costSubTxt = "";
          var costSubCls = "";
          var popoverHtml = "";
          var titleText = "";

          if (isReady) {
            var pctOfTotal = curFunds > 0 ? ((cost / curFunds) * 100) : 0;
            var pctStr = (pctOfTotal < 0.1 && pctOfTotal > 0) ? "<0.1%" : (pctOfTotal.toFixed(1) + "%");
            var remAfter = Math.max(0, curFunds - cost);
            costSubTxt = cost > 0 ? (pctStr + " of bank") : "Free";
            costSubCls = "ready";

            titleText = "Cost: " + costTxt + "\n" +
              "• Status: Ready to buy now!\n" +
              "• Takes: " + pctStr + " of your current " + resName + " (" + prettify(cost) + " / " + prettify(curFunds) + ")\n" +
              "• Balance remaining after purchase: " + prettify(remAfter) + " " + resName;

            popoverHtml = "<div class='sim-cost-popover " + (idx >= results.length / 2 ? "pop-up" : "pop-down") + "'>" +
              "<div class='scp-head'><span class='scp-badge ready'>" + ICONS.check + " Ready to Buy</span><span class='scp-pct'>" + pctStr + " of total</span></div>" +
              "<div class='scp-body'>" +
                "<div class='scp-row'><span class='scp-lbl'>Cost:</span><span class='mono scp-val'>" + prettify(cost) + " " + resName + "</span></div>" +
                "<div class='scp-row'><span class='scp-lbl'>Your Bank:</span><span class='mono scp-val'>" + prettify(curFunds) + " " + resName + "</span></div>" +
                "<div class='scp-row highlight'><span class='scp-lbl'>Will take:</span><span class='mono scp-val buy'>" + pctStr + " of total " + resName + "</span></div>" +
                "<div class='scp-row'><span class='scp-lbl'>Remaining after:</span><span class='mono scp-val'>" + prettify(remAfter) + " " + resName + "</span></div>" +
              "</div>" +
            "</div>";
          } else {
            var need = Math.max(0, cost - curFunds);
            var waitSec = timeForCost(cost, isSh, earnRate, bnk.dust, bnk.shards);
            var waitStr = (isFinite(waitSec) && waitSec > 0) ? fmtTime(waitSec) : (waitSec === 0 ? "Ready now" : "Unattainable");
            var progressPct = cost > 0 ? Math.min(99.9, (curFunds / cost) * 100) : 0;
            var shardRate = (canEarnShards() && earnRate > 0) ? (earnRate / SHARD_DUST) : 0;
            var earnStr = isSh ? (prettify(shardRate) + "/s shards (~" + prettify(earnRate) + "/s dust)") : (prettify(earnRate) + "/s dust");
            costSubTxt = "~" + waitStr;
            costSubCls = "wait";

            titleText = "Cost: " + costTxt + "\n" +
              "• Status: Saving required (~" + waitStr + ")\n" +
              "• Need: " + prettify(need) + " more " + resName + " (" + progressPct.toFixed(1) + "% funded: " + prettify(curFunds) + " / " + prettify(cost) + ")\n" +
              "• Time to finish saving: ~" + waitStr + "\n" +
              "• Earning rate from main build: " + earnStr;

            popoverHtml = "<div class='sim-cost-popover " + (idx >= results.length / 2 ? "pop-up" : "pop-down") + "'>" +
              "<div class='scp-head'><span class='scp-badge wait'>" + ICONS.clock + " Saving Required</span><span class='scp-pct'>~" + waitStr + "</span></div>" +
              "<div class='scp-body'>" +
                "<div class='scp-row'><span class='scp-lbl'>Cost:</span><span class='mono scp-val'>" + prettify(cost) + " " + resName + "</span></div>" +
                "<div class='scp-row'><span class='scp-lbl'>Your Bank:</span><span class='mono scp-val'>" + prettify(curFunds) + " " + resName + " (" + progressPct.toFixed(1) + "% funded)</span></div>" +
                "<div class='scp-row highlight'><span class='scp-lbl'>Time to finish:</span><span class='mono scp-val accent'>~" + waitStr + "</span></div>" +
                "<div class='scp-row'><span class='scp-lbl'>Still needed:</span><span class='mono scp-val warn'>" + prettify(need) + " " + resName + "</span></div>" +
                "<div class='scp-footer'>Based on main build earn rate: <b class='mono'>" + earnStr + "</b></div>" +
              "</div>" +
            "</div>";
          }

          var diffStr = r.deltaDustPs > 0 ? ("+" + prettify(r.deltaDustPs) + "/s") : ("-" + prettify(Math.abs(r.deltaDustPs)) + "/s");
          var diffCls = r.deltaDustPs > 0 ? "pos" : (r.deltaDustPs < 0 ? "neg" : "muted");
          var paybackStr = (r.paybackSec != null && isFinite(r.paybackSec) && r.paybackSec > 0) ? fmtTime(r.paybackSec) : (r.deltaDustPs > 0 ? "Instant" : "Never");

          return "<tr>" +
            "<td>#" + (idx + 1) + "</td>" +
            "<td><b>" + esc(r.label) + "</b></td>" +
            "<td class='mono sim-cost-cell' title='" + esc(titleText) + "'>" +
              "<div class='sim-cost-wrap' tabindex='0'>" +
                "<div class='sim-cost-main'>" +
                  "<span>" + esc(costTxt) + "</span>" +
                  "<span class='sim-cost-sub " + costSubCls + "'>" + esc(costSubTxt) + "</span>" +
                "</div>" +
                popoverHtml +
              "</div>" +
            "</td>" +
            "<td class='mono " + diffCls + "'><b>" + esc(diffStr) + "</b></td>" +
            "<td class='mono'>" + esc(paybackStr) + "</td>" +
            "<td><button type='button' class='btn sm ghost' data-apply-upgrade='" + esc(r.type + ":" + (r.key || "")) + "'>Apply to Sandbox</button></td>" +
            "</tr>";
        }).join("");

        resDiv.innerHTML = "<div class='tbl-wrap'><table class='sim-ext-table'><thead><tr><th>Rank</th><th>Upgrade Step</th><th>Cost</th><th>Simulated Δ Dust/s</th><th>Payback</th><th>Action</th></tr></thead><tbody>" + rows + "</tbody></table></div>";

        resDiv.onclick = function (ev) {
          var btn = ev.target.closest("button[data-apply-upgrade]");
          if (!btn) return;
          var parts = btn.getAttribute("data-apply-upgrade").split(":");
          var type = parts[0];
          var key = parts[1];
          if (type === "ring") {
            simState.config.ring.level++;
          } else if (type === "item" && key && simState.config.items[key]) {
            var cat = itemCat(key);
            if (cat && (cat.noUp || (cat.max && cat.max <= 1))) return;
            if (cat && cat.max && simState.config.items[key].level >= cat.max) return;
            simState.config.items[key].level++;
          }
          renderSimSandboxUI();
          startSimSession();
          switchSimTab("arena");
          toast("Applied upgrade: " + (type === "ring" ? "Ring" : itemName(key)) + " +1");
        };
      } catch (e) {
        resDiv.innerHTML = "<div style='color:var(--bad);padding:10px;'>Upgrade sensitivity error: " + esc(e.message || String(e)) + "</div>";
      }
    }, 10);
  }

  var bound = false;
  function bind() {
    if (bound) return;
    bound = true;
    $("themeBtn").addEventListener("click", toggleTheme);
    $("updateBtn").addEventListener("click", onUpdate);
    $("saveBox").addEventListener("paste", function () { setTimeout(onLoad, 0); });
    $("levelSelect").addEventListener("change", function () {
      syncCustomSelect("levelSelect");
      render();
    });
    initCenteredSelect("levelSelect", { menuMinWidth: 190 });
    $("dustOverride").addEventListener("input", onDustInput);
    $("dustOverride").addEventListener("keydown", function(ev) {
      if (ev.key === "Enter") {
        ev.preventDefault();
        $("dustOverride").blur();
      }
    });
    var btnDustToggle = $("btnDustToggleOverride");
    if (btnDustToggle) {
      btnDustToggle.addEventListener("click", function() {
        if (state.dustOverride) {
          state.dustOverride = "";
          if ($("dustOverride")) $("dustOverride").value = "";
          if ($("dustOverrideWrap")) $("dustOverrideWrap").style.display = "none";
          render();
          toast("Custom rate cleared; reverted to " + (state.simulatedDustPs ? "simulated" : "sheet") + " rate");
        } else {
          var wrap = $("dustOverrideWrap");
          if (wrap) {
            var isShown = wrap.style.display !== "none";
            wrap.style.display = isShown ? "none" : "inline-flex";
            if (!isShown && $("dustOverride")) $("dustOverride").focus();
          }
        }
      });
    }
    ["filtRing", "filtContract", "filtLimb", "filtHideUnused"].forEach(function (id) {
      $(id).addEventListener("change", onFilterChange);
    });
    if ($("btnHelp") && $("helpDlg")) {
      $("btnHelp").addEventListener("click", function () { $("helpDlg").showModal(); });
      $("btnCloseHelp").addEventListener("click", function () { $("helpDlg").close(); });
      $("helpDlg").addEventListener("click", function (ev) {
        if (ev.target === $("helpDlg")) $("helpDlg").close();
      });
    }
    if ($("presetsDlg")) {
      var btnP = $("btnPresets");
      if (btnP) btnP.addEventListener("click", openPresetsModal);
      var btnCloseP = $("btnClosePresets");
      if (btnCloseP) btnCloseP.addEventListener("click", closePresetsModal);
      $("presetsDlg").addEventListener("click", function (ev) {
        if (ev.target === $("presetsDlg")) closePresetsModal();
      });
    }
    if ($("buildDetailsDlg")) {
      var btnCloseBD = $("btnCloseBuildDetails");
      if (btnCloseBD) btnCloseBD.addEventListener("click", closeBuildDetailsModal);
      var bdDlg = $("buildDetailsDlg");
      if (bdDlg) {
        bdDlg.addEventListener("click", function (ev) {
          if (ev.target === bdDlg) closeBuildDetailsModal();
        });
        bdDlg.addEventListener("close", closeBuildDetailsModal);
        bdDlg.addEventListener("cancel", closeBuildDetailsModal);
      }
    }
    if ($("simDlg")) {
      var btnS = $("btnOpenSim");
      if (btnS) btnS.addEventListener("click", function() { openSimModal(); });
      var btnCloseS = $("btnCloseSim");
      if (btnCloseS) btnCloseS.addEventListener("click", closeSimModal);
      var sDlg = $("simDlg");
      if (sDlg) {
        sDlg.addEventListener("click", function (ev) {
          if (ev.target === sDlg) closeSimModal();
        });
        sDlg.addEventListener("close", closeSimModal);
        sDlg.addEventListener("cancel", closeSimModal);
      }

      var sLevel = $("simLevelSelect");
      if (sLevel) {
        sLevel.addEventListener("change", function () {
          syncCustomSelect("simLevelSelect");
          onSimLevelChange();
        });
        initCenteredSelect("simLevelSelect", { compact: true, menuMinWidth: 190 });
      }
      var sPlay = $("simBtnPlay");
      if (sPlay) sPlay.addEventListener("click", toggleSimPlay);
      var sStep = $("simBtnStep");
      if (sStep) sStep.addEventListener("click", stepSimFrame);
      var sReset = $("simBtnReset");
      if (sReset) sReset.addEventListener("click", resetSimBattle);
      var sAddItem = $("simBtnAddItem");
      if (sAddItem) sAddItem.addEventListener("click", function() { toggleSimAddItemMenu(); });
      var sCloseAddItem = $("simBtnCloseAddItem");
      if (sCloseAddItem) sCloseAddItem.addEventListener("click", closeSimAddItemMenu);
      var sAddSearch = $("simAddItemSearch");
      if (sAddSearch) {
        sAddSearch.addEventListener("input", function() {
          simState.addItemSearch = sAddSearch.value || "";
          renderSimAddItemMenu();
        });
      }
      var sAddFilters = $("simAddItemFilters");
      if (sAddFilters) {
        sAddFilters.addEventListener("click", function(ev) {
          var fBtn = ev.target.closest("[data-item-filter]");
          if (!fBtn) return;
          simState.addItemFilter = fBtn.getAttribute("data-item-filter") || "all";
          renderSimAddItemMenu();
        });
      }
      var sResetSave = $("simBtnResetToSave");
      if (sResetSave) sResetSave.addEventListener("click", resetSimToSave);
      var sLockPlan = $("simBtnLockToPlan");
      if (sLockPlan) sLockPlan.addEventListener("click", onSimLockToPlan);

      var speedGroup = $("simSpeedGroup");
      if (speedGroup) {
        speedGroup.onclick = function(ev) {
          var sBtn = ev.target.closest("button[data-speed]");
          if (sBtn) setSimSpeed(sBtn.getAttribute("data-speed"));
        };
      }

      var tabsGroup = $("simTabs");
      if (tabsGroup) {
        tabsGroup.onclick = function(ev) {
          var tBtn = ev.target.closest("button[data-tab]");
          if (tBtn) switchSimTab(tBtn.getAttribute("data-tab"));
        };
      }

      var rRingMinus = $("simRingMinus");
      if (rRingMinus) rRingMinus.addEventListener("click", function() {
        if (simState.config && simState.config.ring.level > 1) {
          simState.config.ring.level--;
          renderSimSandboxUI();
          startSimSession();
        }
      });
      var rRingPlus = $("simRingPlus");
      if (rRingPlus) rRingPlus.addEventListener("click", function() {
        if (simState.config) {
          simState.config.ring.level++;
          renderSimSandboxUI();
          startSimSession();
        }
      });

      var btnPush = $("simBtnRunPush");
      if (btnPush) btnPush.addEventListener("click", function () { runMonteCarloPush(true); });
    }

    if ($("simLockPromptDlg")) {
      var btnConfirm = $("btnConfirmLockPrompt");
      if (btnConfirm) btnConfirm.addEventListener("click", confirmSimCustomLock);
      var btnCloseLp = $("btnCloseLockPrompt");
      if (btnCloseLp) btnCloseLp.addEventListener("click", closeSimLockPrompt);
      var btnCancelLp = $("btnCancelLockPrompt");
      if (btnCancelLp) btnCancelLp.addEventListener("click", closeSimLockPrompt);
      $("simLockPromptDlg").addEventListener("click", function(ev) {
        if (ev.target === $("simLockPromptDlg")) closeSimLockPrompt();
      });
    }

    document.addEventListener("keydown", function (ev) {
      if (openCustomSelectState || (ev.target && ev.target.closest && ev.target.closest(".sa-select-menu"))) return;
      var tag = (ev.target && ev.target.tagName) || "";
      if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" || ev.ctrlKey || ev.metaKey || ev.altKey) return;
      if (ev.key === "t" || ev.key === "T") { ev.preventDefault(); toggleTheme(); }
      else if (ev.key === "u" || ev.key === "U") { ev.preventDefault(); onUpdate(); }
      else if (ev.key === "s" || ev.key === "S") {
        if (state.ab) {
          ev.preventDefault();
          var sDlg = $("simDlg");
          if (sDlg && sDlg.open) closeSimModal();
          else openSimModal();
        }
      }
      else if (ev.key === "p" || ev.key === "P") {
        if (state.ab) {
          ev.preventDefault();
          var pDlg = $("presetsDlg");
          if (pDlg && pDlg.open) closePresetsModal();
          else openPresetsModal();
        }
      }
      else if (ev.key === "v" || ev.key === "V") { ev.preventDefault(); onPasteClipboard(); }
      else if (ev.key === "h" || ev.key === "H") {
        if ($("filtHideUnused") && state.ab) {
          ev.preventDefault();
          $("filtHideUnused").checked = !$("filtHideUnused").checked;
          onFilterChange();
        }
      }
      else if (ev.key === "[") { ev.preventDefault(); stepLevel(-1); }
      else if (ev.key === "]") { ev.preventDefault(); stepLevel(1); }
      else if (ev.key === "?") {
        if ($("helpDlg")) { ev.preventDefault(); $("helpDlg").showModal(); }
      }
    });
    window.addEventListener("resize", function () {
      fitKpiValue($("kpiNextActionVal"));
    });
    if (typeof ResizeObserver !== "undefined") {
      var chipsEl = $("chips");
      if (chipsEl) {
        var ro = new ResizeObserver(function () {
          fitKpiValue($("kpiNextActionVal"));
        });
        ro.observe(chipsEl);
      }
    }
    if (document.fonts && document.fonts.ready) {
      document.fonts.ready.then(function () {
        fitKpiValue($("kpiNextActionVal"));
      });
    }
    document.addEventListener("visibilitychange", function () {
      if (!document.hidden) updateLiveBank();
    });
    window.saOpenSimTab = function (tabName) {
      openSimModal(null, { tab: tabName || "sensitivity" });
    };
    window.addEventListener("hashchange", function () {
      loadSharedBuildFromHash(true);
    });
    updateFooterDbMeta();
    loadPrefs();
    loadSharedBuildFromHash(false);
    setTheme(state.theme);
    if ($("saveBox") && !$("saveEditor").hidden) $("saveBox").focus();
  }

  function shareBuildUrl() {
    if (!state.ab) {
      toast("Load a save or build first to share a link.");
      return;
    }
    var lvl = state.level || (state.ab && state.ab.enemyLevel) || 1;
    var activeB = null;
    if (simState.active && simState.config) {
      lvl = simState.config.level || lvl;
      var sItems = [];
      for (var sk in simState.config.items) {
        var sit = simState.config.items[sk];
        if (sit && (sit.equipped || (typeof sit === "number" && simState.config.equipped && simState.config.equipped.indexOf(sk) !== -1))) {
          var slv = typeof sit === "number" ? sit : (sit.level || 1);
          sItems.push(sk + ":" + slv);
        }
      }
      var sRingLv = (simState.config.ring && simState.config.ring.level) || 0;
      var sRingMods = (simState.config.ring && simState.config.ring.mods) ? simState.config.ring.mods.join(",") : "";
      var sLimbs = (simState.config.extraLimbs || 0) + 4;
      var sHash = "#lv=" + encodeURIComponent(lvl) +
        "&limbs=" + encodeURIComponent(sLimbs) +
        "&ring=" + encodeURIComponent(sRingLv + (sRingMods ? ":" + sRingMods : "")) +
        "&items=" + encodeURIComponent(sItems.join(","));
      copyShareHash(sHash, "SA " + lvl + " sandbox build");
      return;
    }
    var lockedEntry = state.lockedPlans && state.lockedPlans[String(lvl)];
    var lockedTarget = lockedEntry ? findLockedTarget(lvl, lockedEntry) : null;
    var rec = bestRunnable(lvl);
    activeB = lockedTarget || rec;

    var itemPairs = [];
    var limbsCount = maxSlots();
    var rLv = hasRing() ? ringLevel() : 0;
    var rModsStr = hasRing() ? ringMods().join(",") : "";

    if (activeB && activeB.i) {
      Object.keys(activeB.i).forEach(function (k) {
        var lv = Math.max(userLevel(k) || 0, activeB.i[k] || 1);
        itemPairs.push(k + ":" + lv);
      });
      limbsCount = activeB.l || Object.keys(activeB.i).length || limbsCount;
      if (activeB.r) {
        rLv = Math.max(rLv, activeB.r.lv || 1);
        if (activeB.r.m && activeB.r.m.length) rModsStr = activeB.r.m.join(",");
      }
    } else {
      ownedItemKeys().forEach(function (k) {
        var uIt = userItem(k);
        if (uIt && uIt.equipped) {
          itemPairs.push(k + ":" + (uIt.level || 1));
        }
      });
    }

    var hash = "#lv=" + encodeURIComponent(lvl) +
      "&limbs=" + encodeURIComponent(limbsCount) +
      "&ring=" + encodeURIComponent(rLv + (rModsStr ? ":" + rModsStr : "")) +
      "&items=" + encodeURIComponent(itemPairs.join(","));
    copyShareHash(hash, (activeB && activeB.n) ? (activeB.n + " (SA " + lvl + ")") : ("SA " + lvl + " loadout"));
  }

  function copyShareHash(hash, label) {
    try {
      if (window.history && window.history.replaceState) {
        window.history.replaceState(null, "", hash);
      } else {
        window.location.hash = hash;
      }
    } catch (e) {}
    var fullUrl = window.location.origin + window.location.pathname + hash;
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(fullUrl).then(function () {
        toast("Copied share link for " + label + "!");
      }).catch(function () {
        toast("Share link updated in address bar (" + label + ")");
      });
    } else {
      toast("Share link updated in address bar (" + label + ")");
    }
  }

  function loadSharedBuildFromHash(isHashChange) {
    var rawHash = (window.location.hash || "").replace(/^#/, "").trim();
    if (!rawHash || rawHash.indexOf("lv=") === -1) return;
    var params = {};
    rawHash.split("&").forEach(function (pair) {
      var eq = pair.indexOf("=");
      if (eq > 0) {
        var k = decodeURIComponent(pair.slice(0, eq));
        var v = decodeURIComponent(pair.slice(eq + 1));
        params[k] = v;
      }
    });
    var lv = Math.max(1, Math.min(200, Math.floor(Number(params.lv) || 0)));
    if (!lv) return;

    var limbs = Math.max(4, Math.min(16, Math.floor(Number(params.limbs) || 4)));
    var rLv = 0;
    var rMods = [];
    if (params.ring) {
      var rParts = String(params.ring).split(":");
      rLv = Math.max(0, Math.min(999, Math.floor(Number(rParts[0]) || 0)));
      if (rParts[1]) {
        rParts[1].split(",").forEach(function (m) {
          var cleanM = m.trim();
          if (RING_MOD_NAMES[cleanM] || /^(Attack|Health|Defense|Lifesteal|Dust)$/.test(cleanM)) {
            rMods.push(cleanM);
          }
        });
      }
    }

    var itemMap = {};
    if (params.items) {
      String(params.items).split(",").forEach(function (entry) {
        var p = entry.split(":");
        var ik = (p[0] || "").trim();
        var ilv = Math.max(1, Math.min(999, Math.floor(Number(p[1]) || 1)));
        if (ik && SA_DATA && SA_DATA.items && SA_DATA.items[ik]) {
          itemMap[ik] = ilv;
        }
      });
    }
    var itemKeys = Object.keys(itemMap);
    if (!itemKeys.length) return;

    var sharedBuild = {
      n: "Shared SA " + lv + " Build",
      d: 0,
      l: Math.max(limbs, itemKeys.length),
      r: rLv > 0 ? { lv: rLv, m: rMods } : null,
      i: itemMap,
      saLevel: lv
    };

    if (!state.ab) {
      var synthItems = {};
      Object.keys(SA_DATA.items).forEach(function (k) {
        if (itemMap[k]) {
          synthItems[k] = { owned: true, equipped: true, level: itemMap[k] };
        } else if (!SA_DATA.items[k].zone) {
          synthItems[k] = { owned: true, equipped: false, level: 1 };
        }
      });
      var synthAb = {
        enemyLevel: lv,
        maxEnemyLevel: Math.max(lv, 60),
        dust: 0,
        shards: 0,
        shardDust: 0,
        enemiesKilled: 0,
        sessionEnemiesKilled: 0,
        sessionTrimpsKilled: 0,
        autoLevel: true,
        items: synthItems,
        bonuses: { Extra_Limbs: Math.max(0, sharedBuild.l - 4), Radon: 0, Stats: 0, Scaffolding: 0 },
        oneTimers: { The_Ring: rLv > 0, Dusty_Tome: true, Master_of_Arms: true, Whirlwind_of_Arms: true },
        rings: { level: Math.max(1, rLv), mods: rMods },
        presets: { names: ["Shared Build", "Preset 2", "Preset 3"], p1: itemKeys.slice(), p2: [], p3: [] },
        settings: { practice: { enabled: 0 }, loadRing: { enabled: 1 }, loadLevel: { enabled: 1 } }
      };
      state.ab = synthAb;
      state.radon = 250;
      state.save = { global: { autoBattleData: synthAb, highestRadonLevelCleared: 250, u2MutationData: {} }, stats: { saKills: { value: 0, valueTotal: 0 } } };
      state.saveLoadedAt = Date.now();
      fillLevels(lv);
      autoCalibrateSaveRate();
      if (state.simulatedDustPs > 0) sharedBuild.d = Math.round(state.simulatedDustPs);
      $("workspace").hidden = false;
      renderSaveFacts();
      setSaveCollapsed(true);
      render();
      startLiveBankTimer();
      toast("Loaded shared SA " + lv + " build from URL link!");
    } else if (isHashChange) {
      selectLevel(lv);
      openSimModal(sharedBuild);
      toast("Loaded shared SA " + lv + " build into Combat Simulator");
    }
  }

  function updateFooterDbMeta() {
    var el = $("footerDbMeta");
    if (!el || typeof SA_DATA === "undefined") return;
    var count = SA_DATA.totalBuilds;
    if (!count && SA_DATA.builds) {
      count = 0;
      for (var k in SA_DATA.builds) count += (SA_DATA.builds[k] || []).length;
    }
    var lvls = SA_DATA.levelCount || (SA_DATA.builds ? Object.keys(SA_DATA.builds).length : 0);
    var syncDate = SA_DATA.lastSynced || "";
    var countTxt = count ? (count.toLocaleString ? count.toLocaleString() : count) : "3,582";
    var lvlTxt = lvls ? (" across " + lvls + " levels") : "";
    var syncTxt = syncDate ? (" · Last synced " + syncDate) : "";
    el.textContent = " · " + countTxt + " builds" + lvlTxt + syncTxt;
  }

  window.saLoad = onLoad;
  window.saUpdate = onUpdate;
  window.saCancelEdit = onCancelEdit;
  window.saPaste = onPasteClipboard;
  window.saStepLevel = stepLevel;
  window.saSelectLevel = selectLevel;
  window.saTheme = toggleTheme;
  window.saRender = render;
  window.saDust = onDustInput;
  window.saFilter = onFilterChange;
  window.saResetBankTimer = resetBankTimer;
  window.saOpenPresets = openPresetsModal;
  window.saClosePresets = closePresetsModal;
  window.saOpenBuildDetails = openBuildDetailsModal;
  window.saCloseBuildDetails = closeBuildDetailsModal;
  window.saOpenSim = openSimModal;
  window.saCloseSim = closeSimModal;
  window.saCloseLockPrompt = closeSimLockPrompt;
  window.saToggleSwapGuide = toggleSwapGuide;
  window.saShareBuildUrl = shareBuildUrl;
  try {
    if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", bind);
    else bind();
  } catch (e) {
    if ($("err")) {
      $("err").hidden = false;
      $("err").textContent = e.message || String(e);
    }
  }
})();
