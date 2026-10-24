(function () {
  var SHARD_DUST = 1e9;
  var RING_MOD_NAMES = { attack: "Attack", health: "Health", defense: "Defense", lifesteal: "Lifesteal", dustMult: "Dust" };
  var EXTRA_DESC = {
    Doppelganger_Signet: "Summon a Doppelganger: 50% damage reduction, 2x Attack, +1 Poison Stack Rate while alive. Explodes after taking your Max Health in damage or if it would kill, redealing fight damage and shredding 50% Enemy Defense.",
    Doppelganger_Diadem: "Can Bleed for 10s. 2x Attack if Signet is equipped, x2 again if Doppelganger is alive. Doppelganger has 50% more health and can resurrect once per fight."
  };

  var state = {
    save: null,
    ab: null,
    radon: 0,
    level: 0,
    theme: "dark",
    dustOverride: "",
    filters: { ring: true, contract: true, limb: true, hideUnused: false }
  };

  function $(id) { return document.getElementById(id); }

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

  function itemCat(key) { return SA_DATA.items[key] || null; }
  function itemName(key) { var c = itemCat(key); return c ? c.name : key.replace(/__/g, "-").replace(/_/g, " "); }
  function isShards(key) { var c = itemCat(key); return c && c.dust === "shards"; }
  function currencyName(key) { return isShards(key) ? "shards" : "dust"; }

  function upgradeCostAt(key, level) {
    var c = itemCat(key);
    if (!c || c.noUp) return 0;
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

  function ownsContractZone(zone) {
    if (!zone) return true;
    for (var key in SA_DATA.items) {
      if (SA_DATA.items[key].zone === zone && userOwns(key)) return true;
    }
    return false;
  }

  function itemForZone(zone) {
    for (var key in SA_DATA.items) {
      if (SA_DATA.items[key].zone === zone) return key;
    }
    return null;
  }

  function decodeSave(raw) {
    var s = String(raw || "").replace(/\s+/g, "");
    if (!s) throw new Error("Paste a Trimps export string.");
    var json = LZString.decompressFromBase64(s);
    if (!json) throw new Error("Could not decode that string.");
    var data = JSON.parse(json);
    if (!data || !data.global) throw new Error("That does not look like a full Trimps save.");
    var ab = data.global.autoBattleData;
    if (!ab) throw new Error("This save has no Spire Assault data. Unlock SA in U2 first.");
    return { save: data, ab: ab, radon: data.global.highestRadonLevelCleared || 0 };
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
    list.sort(function (a, b) { return (b.d || 0) - (a.d || 0); });
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
    var rank = { contract: 0, missing: 1, ringUnlock: 2, level: 3, limbs: 4, ring: 5 };
    steps.sort(function (a, b) {
      var ra = rank.hasOwnProperty(a.kind) ? rank[a.kind] : 9;
      var rb = rank.hasOwnProperty(b.kind) ? rank[b.kind] : 9;
      return ra - rb;
    });
    return steps;
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
    return { dust: dust, shards: shards, combined: dust + shards * SHARD_DUST };
  }

  function sheetDustPs(build) { return build && build.d ? build.d : 0; }

  function activeDustPs(recommended) {
    var over = parseNum(state.dustOverride);
    if (isFinite(over) && over > 0) return over;
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

  function bank() {
    return {
      dust: (state.ab && state.ab.dust) || 0,
      shards: (state.ab && state.ab.shards) || 0
    };
  }

  function setTheme(theme) {
    state.theme = theme === "light" ? "light" : "dark";
    document.documentElement.setAttribute("data-theme", state.theme);
    $("themeBtn").textContent = state.theme === "dark" ? "Light" : "Dark";
    try { localStorage.setItem("saPlanner.theme", state.theme); } catch (e) {}
  }

  function loadPrefs() {
    try {
      var t = localStorage.getItem("saPlanner.theme");
      if (t) setTheme(t);
      var d = localStorage.getItem("saPlanner.dustPs");
      if (d) {
        state.dustOverride = d;
        $("dustOverride").value = d;
      }
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
    } catch (e) {}
  }

  function fillLevels(selected) {
    var sel = $("levelSelect");
    sel.innerHTML = "";
    sheetLevels().forEach(function (lv) {
      var o = document.createElement("option");
      o.value = String(lv);
      o.textContent = "SA " + lv;
      sel.appendChild(o);
    });
    if (selected && SA_DATA.builds[String(selected)]) sel.value = String(selected);
    else if (sel.options.length) sel.selectedIndex = 0;
    state.level = Number(sel.value);
  }

  function renderChips() {
    var ab = state.ab;
    $("chips").innerHTML = [
      chip("Max SA", ab.maxEnemyLevel),
      chip("Fighting", ab.enemyLevel),
      chip("Dust", prettify(ab.dust)),
      chip("Shards", prettify(ab.shards)),
      chip("Limbs", maxSlots() + " slots"),
      chip("Ring", hasRing() ? "Lv " + ringLevel() : "locked")
    ].join("");
  }

  function chip(label, value) {
    return '<div class="chip"><span>' + esc(label) + '</span><b>' + esc(String(value)) + '</b></div>';
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

  function snapshotFarm() {
    return {
      items: JSON.parse(JSON.stringify(state.ab.items || {})),
      bonuses: JSON.parse(JSON.stringify(state.ab.bonuses || {})),
      oneTimers: JSON.parse(JSON.stringify(state.ab.oneTimers || {})),
      rings: JSON.parse(JSON.stringify(state.ab.rings || {}))
    };
  }

  function restoreFarm(snap) {
    state.ab.items = snap.items;
    state.ab.bonuses = snap.bonuses;
    state.ab.oneTimers = snap.oneTimers;
    state.ab.rings = snap.rings;
  }

  function applyAtomic(s) {
    if (s.kind === "contract" || s.kind === "missing") {
      if (!state.ab.items) state.ab.items = {};
      state.ab.items[s.key] = { owned: true, level: 1, equipped: false };
      return;
    }
    if (s.kind === "level") {
      if (!state.ab.items) state.ab.items = {};
      if (!state.ab.items[s.key]) state.ab.items[s.key] = { owned: true, level: 1 };
      state.ab.items[s.key].owned = true;
      state.ab.items[s.key].level = (state.ab.items[s.key].level || 1) + 1;
      return;
    }
    if (s.kind === "limbs") {
      if (!state.ab.bonuses) state.ab.bonuses = {};
      state.ab.bonuses.Extra_Limbs = extraLimbs() + 1;
      return;
    }
    if (s.kind === "ringUnlock") {
      if (!state.ab.oneTimers) state.ab.oneTimers = {};
      state.ab.oneTimers.The_Ring = true;
      if (!state.ab.rings) state.ab.rings = { level: 1, mods: [] };
      return;
    }
    if (s.kind === "ring") {
      if (!state.ab.rings) state.ab.rings = { level: 1, mods: [] };
      state.ab.rings.level = (state.ab.rings.level || 1) + 1;
    }
  }

  function bestAfter(step) {
    var snap = snapshotFarm();
    applyAtomic(step);
    var next = bestRunnable(state.level);
    restoreFarm(snap);
    return next;
  }

  function betterBuilds(rec) {
    var cur = rec ? rec.d || 0 : 0;
    return buildsFor(state.level).filter(function (b) {
      return (b.d || 0) > cur + 1e-9 && !runnable(b) && allowedUpgrade(b);
    });
  }

  function bestNextUpgrade(rec) {
    var cur = rec ? rec.d || 0 : 0;
    var list = betterBuilds(rec);
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
        var gain = (unlocked ? unlocked.d || 0 : 0) - cur;
        var cost = stepsCostDust([s]).combined;
        if (gain > 0 && unlocked) immediate.push({ step: s, gain: gain, cost: cost, roi: cost > 0 ? gain / cost : gain, build: unlocked, mode: "unlocks" });
      });
    });
    if (immediate.length) {
      immediate.sort(function (a, b) { return b.roi - a.roi; });
      return immediate[0];
    }
    var paths = list.map(function (b) {
      var steps = gapSteps(b);
      var cost = stepsCostDust(steps).combined;
      var blocked = steps.some(function (s) { return s.kind === "contract" && !s.reach; });
      return { build: b, steps: steps, cost: cost, remaining: steps.length, blocked: blocked, roi: cost > 0 ? ((b.d || 0) - cur) / cost : 0 };
    });
    paths.sort(function (a, b) { return (a.blocked - b.blocked) || a.remaining - b.remaining || b.roi - a.roi; });
    var p = null;
    var first = null;
    for (var i = 0; i < paths.length; i++) {
      for (var j = 0; j < paths[i].steps.length; j++) {
        if (paths[i].steps[j].kind === "contract" && !paths[i].steps[j].reach) continue;
        first = atomicStep(paths[i].steps[j]);
        p = paths[i];
        break;
      }
      if (first) break;
    }
    if (!p || !first) return null;
    return {
      step: first,
      gain: (p.build.d || 0) - cur,
      cost: stepsCostDust([first]).combined,
      build: p.build,
      mode: "toward",
      remaining: p.remaining
    };
  }

  function render() {
    if (!state.ab) return;
    var level = Number($("levelSelect").value);
    state.level = level;
    var rec = bestRunnable(level);
    var rate = activeDustPs(rec);
    var est = sheetDustPs(rec);
    $("dustEstimate").textContent = rec ? ("sheet " + prettify(est) + "/s") : "no runnable build";
    $("controls").hidden = false;
    $("results").hidden = false;
    renderChips();
    renderRecommended(rec, rate);
    renderNext(rec, rate);
    renderBoard(rec, rate);
    renderBetter(rec, rate);
  }

  function renderRecommended(rec, rate) {
    var el = $("recommend");
    if (!rec) {
      el.innerHTML = "<p class='empty rec'>No sheet build on this level is fully runnable with your current items, levels, limbs, and ring. See better builds below for the cheapest unlock path.</p>";
      return;
    }
    var mods = (rec.r && rec.r.m || []).map(function (m) { return RING_MOD_NAMES[m] || m; }).join(" / ");
    var items = Object.keys(rec.i).map(function (k) { return esc(itemName(k)) + " " + rec.i[k]; }).join(", ");
    el.innerHTML =
      "<div class='rec'><div class='rec-head'><strong>" + esc(rec.n) + "</strong> · " + prettify(rec.d) + "/s · " +
      (rec.k != null ? rec.k + "s kills" : "") +
      (rec.wr != null ? " · " + Math.round((rec.wr <= 1 ? rec.wr : rec.wr / 100) * 100) + "% WR" : "") +
      "</div>" +
      "<div class='rec-meta'>Limbs " + (rec.l || "?") +
      (rec.r ? " · Ring " + rec.r.lv + (mods ? " (" + esc(mods) + ")" : "") : "") +
      (rec.c ? " · highest contract Z" + rec.c : "") +
      "</div>" +
      "<div class='rec-items'>" + items + "</div>" +
      (rec.note ? "<p class='notes'>" + esc(rec.note) + "</p>" : "") +
      "</div>";
  }

  function renderNext(rec, rate) {
    var el = $("nextUpgrade");
    var pick = bestNextUpgrade(rec);
    if (!pick) {
      el.innerHTML = rec
        ? "<div class='next'><div class='next-kicker'>Next upgrade</div><div class='empty'>You already match the strongest sheet farm on this level.</div></div>"
        : "<div class='next'><div class='next-kicker'>Next upgrade</div><div class='empty'>No single upgrade on this tab beats your current setup with the current filters.</div></div>";
      return;
    }
    var s = pick.step;
    var bnk = bank();
    var wait = timeForCost(pick.cost, !!s.shards, rate, s.shards ? 0 : bnk.dust, s.shards ? bnk.shards : 0);
    var costTxt = pick.cost ? (prettify(s.shards ? pick.cost : pick.cost) + (s.shards ? " shards" : " dust")) : "";
    if (s.shards && pick.cost) costTxt = prettify(pick.cost) + " shards";
    else if (pick.cost) costTxt = prettify(pick.cost) + " dust";
    var why = pick.mode === "unlocks"
      ? "Highest dust/s gain per cost among upgrades that immediately unlock a better sheet farm."
      : "First step on the fewest-gap path to a better sheet farm (no one-step unlock on this tab).";
    el.innerHTML =
      "<div class='next'><div class='next-kicker'>Next upgrade</div>" +
      "<div class='next-head'><span class='" + stepClass(s) + "'>" + esc(stepLabel(s)) + "</span></div>" +
      "<div class='next-meta'>" + (costTxt ? esc(costTxt) + " · " : "") +
      (wait != null ? fmtTime(wait) + " · " : "") +
      (pick.mode === "unlocks"
        ? "unlocks " + esc(pick.build.n) + " (" + prettify(pick.build.d) + "/s, +" + prettify(pick.gain) + "/s)"
        : "toward " + esc(pick.build.n) + " (" + prettify(pick.build.d) + "/s, +" + prettify(pick.gain) + "/s after " + pick.remaining + " steps)") +
      "</div><div class='next-why'>" + why + "</div></div>";
  }

  function cardTooltip(title, lines) {
    return "<div class='tip'><strong>" + esc(title) + "</strong>" + lines.map(function (l) { return "<div>" + esc(l) + "</div>"; }).join("") + "</div>";
  }

  function renderBoard(rec, rate) {
    var recItems = rec ? rec.i : {};
    var hide = !!state.filters.hideUnused && rec;
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

    order.forEach(function (k) {
      html += itemCard(k, recItems[k], rate, bnk);
    });
    if (!hide || (rec && rec.r)) html += ringCard(rec && rec.r, rate, bnk);
    if (rec && rec.l && rec.l > maxSlots()) html += limbsCard(rec.l, rate, bnk);
    $("board").innerHTML = html;
  }

  function itemCard(key, recLv, rate, bnk) {
    var cat = itemCat(key);
    var owned = userOwns(key);
    var lv = owned ? userLevel(key) : 0;
    var need = recLv || 0;
    var cls = "card";
    if (need) cls += " rec";
    if (!owned) cls += " locked";
    else if (need && lv < need) cls += " warn";
    else if (need && lv >= need) cls += " ok";

    var lines = [];
    var body = "";
    if (!owned) {
      if (cat && cat.zone) {
        var active = state.ab.activeContract === key;
        var reach = (state.radon + 1) >= cat.zone;
        body += "<div class='lv'>Locked</div>";
        body += "<div class='cost'>Z" + cat.zone + " contract · " + prettify(contractPrice(key)) + " " + currencyName(key) + "</div>";
        var tUnlock = timeForCost(contractPrice(key), isShards(key), rate, bnk.dust, bnk.shards);
        body += "<div class='time'>" + fmtTime(tUnlock) + "</div>";
        if (active) body += "<div class='flag'>Accepted, not finished</div>";
        if (!reach) body += "<div class='flag bad'>Void map not reachable yet</div>";
        lines.push("Complete the Z" + cat.zone + " contract to unlock " + itemName(key) + ".");
      } else {
        body += "<div class='lv'>Locked</div>";
      }
    } else {
      body += "<div class='lv'>Lv " + lv + (need && need > lv ? " <span class='gap'>→ " + need + "</span>" : "") + "</div>";
      if (cat && !cat.noUp) {
        var next = upgradeCostAt(key, lv);
        var tNext = timeForCost(next, isShards(key), rate, bnk.dust, bnk.shards);
        body += "<div class='cost'>Next: " + prettify(next) + " " + currencyName(key) + "</div>";
        body += "<div class='time'>" + fmtTime(tNext) + "</div>";
        lines.push("Next upgrade: " + (cat.upgrade || "—") + ".");
        lines.push("Cost " + prettify(next) + " " + currencyName(key) + " (" + fmtTime(tNext) + ").");
        if (need > lv) {
          var gap = upgradeCostRange(key, lv, need);
          var tGap = timeForCost(gap, isShards(key), rate, bnk.dust, bnk.shards);
          body += "<div class='gapcost'>To Lv " + need + ": " + prettify(gap) + " · " + fmtTime(tGap) + "</div>";
          lines.push("To reach the build: Lv " + lv + " → Lv " + need + ".");
        }
      } else {
        body += "<div class='cost'>Unupgradable</div>";
      }
      lines.unshift(describeItem(key, lv));
    }

    return "<article class='" + cls + "' title='" + esc([itemName(key)].concat(lines).join(" — ")) + "'>" +
      "<h3>" + esc(itemName(key)) + "</h3>" + body +
      cardTooltip(itemName(key), lines) +
      "</article>";
  }

  function ringCard(recRing, rate, bnk) {
    var cls = "card ring";
    var need = recRing ? recRing.lv : 0;
    var lines = [];
    var body = "";
    if (!hasRing()) {
      cls += " locked";
      var cost = oneTimerRingCost();
      var t = timeForCost(cost, true, rate, bnk.dust, bnk.shards);
      body = "<div class='lv'>Locked</div><div class='cost'>One-timer · " + prettify(cost) + " shards</div><div class='time'>" + fmtTime(t) + "</div>";
      lines.push("Unlock The Ring (one-timer, not a contract).");
      lines.push(nextRingSlotNote(1));
    } else {
      var lv = ringLevel();
      if (need && lv < need) cls += " warn rec";
      else if (need) cls += " ok rec";
      var next = ringLevelCost(lv);
      var tNext = timeForCost(next, true, rate, bnk.dust, bnk.shards);
      var mods = ringMods().map(function (m) { return RING_MOD_NAMES[m] || m; }).join(" / ");
      body = "<div class='lv'>Lv " + lv + (need && need > lv ? " <span class='gap'>→ " + need + "</span>" : "") + "</div>";
      body += "<div class='mods'>" + esc(mods || "no custom mods") + "</div>";
      body += "<div class='cost'>Next: " + prettify(next) + " shards</div><div class='time'>" + fmtTime(tNext) + "</div>";
      if (need > lv) {
        var gap = ringCostRange(lv, need);
        body += "<div class='gapcost'>To Lv " + need + ": " + prettify(gap) + " shards · " + fmtTime(timeForCost(gap, true, rate, bnk.dust, bnk.shards)) + "</div>";
        lines.push("To reach the build: Ring Lv " + lv + " → Lv " + need + ".");
      }
      lines.unshift(describeRing(lv, ringMods()));
      lines.push("Next level: all ring stats scale. " + nextRingSlotNote(lv + 1));
    }
    return "<article class='" + cls + "' title='" + esc(["The Ring"].concat(lines).join(" — ")) + "'><h3>The Ring</h3>" + body + cardTooltip("The Ring", lines) + "</article>";
  }

  function limbsCard(needSlots, rate, bnk) {
    var from = extraLimbs();
    var to = needSlots - 4;
    var cost = extraLimbCost(from, to);
    var t = timeForCost(cost, false, rate, bnk.dust, bnk.shards);
    return "<article class='card warn rec'><h3>Extra Limbs</h3>" +
      "<div class='lv'>" + maxSlots() + " slots <span class='gap'>→ " + needSlots + "</span></div>" +
      "<div class='cost'>" + prettify(cost) + " dust</div><div class='time'>" + fmtTime(t) + "</div>" +
      cardTooltip("Extra Limbs", ["Huffy can equip 1 additional item per level.", "Need " + from + " → " + to + "."]) +
      "</article>";
  }

  function renderBetter(rec, rate) {
    var cur = rec ? rec.d || 0 : 0;
    var list = betterBuilds(rec);
    list.sort(function (a, b) {
      var ca = stepsCostDust(gapSteps(a)).combined;
      var cb = stepsCostDust(gapSteps(b)).combined;
      var ra = ca > 0 ? ((a.d || 0) - cur) / ca : 0;
      var rb = cb > 0 ? ((b.d || 0) - cur) / cb : 0;
      return rb - ra;
    });
    list = list.slice(0, 8);
    if (!list.length) {
      $("better").innerHTML = rec
        ? "<p class='empty'>You already match the strongest runnable sheet build on this level, or nothing cheaper is listed with the current filters.</p>"
        : "<p class='empty'>No builds on this tab match the current filters.</p>";
      return;
    }
    var bnk = bank();
    var rows = list.map(function (b) {
      var steps = gapSteps(b);
      var cost = stepsCostDust(steps);
      var pay = timeForCost(cost.combined, false, rate, 0, 0);
      var blocked = steps.some(function (s) { return s.kind === "contract" && !s.reach; });
      return "<tr" + (blocked ? " class='unreachable'" : "") + "><td><strong>" + esc(b.n) + "</strong><div class='sub'>" + prettify(b.d) + "/s" +
        (b.k != null ? " · " + b.k + "s" : "") + "</td>" +
        "<td class='steps'>" + steps.map(function (s) {
          return "<div class='" + stepClass(s) + "'>" + esc(stepLabel(s)) + "</div>";
        }).join("") + "</td>" +
        "<td>" + (cost.dust ? prettify(cost.dust) + " dust" : "") +
        (cost.dust && cost.shards ? "<br>" : "") +
        (cost.shards ? prettify(cost.shards) + " shards" : "") + "</td>" +
        "<td>" + prettify((b.d || 0) - cur) + "/s</td>" +
        "<td>" + fmtTime(pay) + "</td></tr>";
    }).join("");
    $("better").innerHTML = "<table><thead><tr><th>Build</th><th>Missing</th><th>Cost</th><th>Δ d/s</th><th>Payback</th></tr></thead><tbody>" + rows + "</tbody></table>";
  }

  function fact(label, value) {
    return '<div class="fact"><span>' + esc(label) + '</span>' + esc(String(value)) + '</div>';
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
    var owned = ownedItemKeys();
    var equipped = owned.filter(function (k) { return userItem(k) && userItem(k).equipped; }).length;
    var contracts = owned.filter(function (k) { var c = itemCat(k); return c && c.zone; }).length;
    var contractItems = Object.keys(SA_DATA.items).filter(function (k) { return SA_DATA.items[k].zone; }).length;
    var contract = state.ab.activeContract;
    var mods = ringMods().map(function (m) { return RING_MOD_NAMES[m] || m; }).join(" / ");
    $("saveFacts").innerHTML = [
      fact("Highest Zone U2", state.radon + 1),
      fact("Void maps", "to Z" + (state.radon + 1)),
      fact("Contract", contract ? itemName(contract) : "none"),
      fact("Items", owned.length + "/" + SA_DATA.itemOrder.length),
      fact("Equipped", equipped),
      fact("Contracts", contracts + "/" + contractItems),
      fact("Ring mods", hasRing() ? (mods || "none") : "locked")
    ].join("");
  }

  function setSaveCollapsed(collapsed) {
    $("saveEditor").hidden = collapsed;
    $("saveSummary").hidden = !collapsed;
  }

  function onUpdate() {
    setSaveCollapsed(false);
    $("err").hidden = true;
    var box = $("saveBox");
    box.focus();
    box.select();
  }

  function onLoad() {
    $("err").hidden = true;
    try {
      var decoded = decodeSave($("saveBox").value);
      state.save = decoded.save;
      state.ab = decoded.ab;
      state.radon = decoded.radon;
      var pick = decoded.ab.enemyLevel || decoded.ab.maxEnemyLevel || 4;
      if (pick > (decoded.ab.maxEnemyLevel || pick)) pick = decoded.ab.maxEnemyLevel;
      fillLevels(pick);
      $("workspace").hidden = false;
      renderSaveFacts();
      setSaveCollapsed(true);
      render();
    } catch (e) {
      $("err").hidden = false;
      $("err").textContent = e.message || String(e);
      setSaveCollapsed(false);
      if (!state.ab) $("workspace").hidden = true;
    }
  }

  function toggleTheme() { setTheme(state.theme === "dark" ? "light" : "dark"); }

  function onDustInput() {
    state.dustOverride = $("dustOverride").value;
    try { localStorage.setItem("saPlanner.dustPs", state.dustOverride); } catch (e) {}
    if (state.ab) render();
  }

  function onFilterChange() {
    state.filters = readFilters();
    try { localStorage.setItem("saPlanner.filters", JSON.stringify(state.filters)); } catch (e) {}
    if (state.ab) render();
  }

  var bound = false;
  function bind() {
    if (bound) return;
    bound = true;
    $("themeBtn").addEventListener("click", toggleTheme);
    $("updateBtn").addEventListener("click", onUpdate);
    $("saveBox").addEventListener("paste", function () { setTimeout(onLoad, 0); });
    $("levelSelect").addEventListener("change", render);
    $("dustOverride").addEventListener("input", onDustInput);
    ["filtRing", "filtContract", "filtLimb", "filtHideUnused"].forEach(function (id) {
      $(id).addEventListener("change", onFilterChange);
    });
    loadPrefs();
    setTheme(state.theme);
    if ($("saveBox") && !$("saveEditor").hidden) $("saveBox").focus();
  }

  window.saLoad = onLoad;
  window.saUpdate = onUpdate;
  window.saTheme = toggleTheme;
  window.saRender = render;
  window.saDust = onDustInput;
  window.saFilter = onFilterChange;
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
