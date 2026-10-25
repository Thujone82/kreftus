(function () {
  var SHARD_DUST = 1e9;
  var RING_MOD_NAMES = { attack: "Attack", health: "Health", defense: "Defense", lifesteal: "Lifesteal", dustMult: "Dust" };
  var EXTRA_DESC = {
    Doppelganger_Signet: "Summon a Doppelganger: 50% damage reduction, 2x Attack, +1 Poison Stack Rate while alive. Explodes after taking your Max Health in damage or if it would kill, redealing fight damage and shredding 50% Enemy Defense.",
    Doppelganger_Diadem: "Can Bleed for 10s. 2x Attack if Signet is equipped, x2 again if Doppelganger is alive. Doppelganger has 50% more health and can resurrect once per fight."
  };

  var ICONS = {
    spire: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2L5 12h4.5l-2 7h9l-2-7H19L12 2zm-4.5 18.5h9V22h-9v-1.5z"/></svg>',
    sword: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="14.5 17.5 3 6 3 3 6 3 17.5 14.5"/><line x1="13" y1="19" x2="19" y2="13"/><line x1="16" y1="16" x2="20" y2="20"/><line x1="19" y1="21" x2="21" y2="19"/></svg>',
    dagger: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 4.5L19.5 9.5 7 22H2v-5L14.5 4.5z"/><path d="M12 7l5 5"/></svg>',
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
    unlock: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 9.9-1"/></svg>'
  };

  function iconForItem(key) {
    if (!key) return ICONS.sword;
    if (key === "Menacing_Mask") return ICONS.mask;
    if (key === "Grounded_Crown" || key === "Doppelganger_Diadem") return ICONS.crown;
    if (key === "Doppelganger_Signet" || key === "Mood_Bracelet") return ICONS.ring;
    if (key === "Bad_Medkit") return ICONS.medkit;
    if (key === "Lifegiving_Gem" || key === "Unlucky_Coin" || key === "Recycler" || key === "Omni_Enhancer" || key === "The_Doomspring" || key === "Basket_of_Souls") return ICONS.gem;
    if (/Sword|Blade/.test(key)) return ICONS.sword;
    if (/Dagger|Shank|Piercer|Nails/.test(key)) return ICONS.dagger;
    if (/Cleaver|Hatchet|Greataxe|Scythe|Harpoon/.test(key)) return ICONS.axe;
    if (/Armor|Suit|Protector|Aegis/.test(key)) return ICONS.shield;
    if (/Goo|Pouch|Chemistry|Mold|Snimp|Globulator|Plague|Slime|Golem|Spores|Myco/.test(key)) return ICONS.flask;
    if (/Battery|Shock|Wired|Eelimp|Goggles|Stormbringer|Fibrillator/.test(key)) return ICONS.bolt;
    if (/Boots/.test(key)) return ICONS.boots;
    if (/Pants|Raincoat|Labcoat|Wraps/.test(key)) return ICONS.coat;
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
    filters: { ring: true, contract: true, limb: true, hideUnused: false },
    stepOrder: {},
    lockedPlans: {},
    keepPopoverOpen: false
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
    var totals = stepsCostDust(steps);
    var bDust = bnk ? (bnk.dust || 0) : 0;
    var bShards = bnk ? (bnk.shards || 0) : 0;
    var remDust = Math.max(0, totals.dust - bDust);
    var remShards = Math.max(0, totals.shards - bShards);
    var remDustEq = Math.max(remDust, remShards * SHARD_DUST);
    var waitDust = remDust <= 0 ? 0 : (rate > 0 ? remDust / rate : Infinity);
    var waitShards = remShards <= 0 ? 0 : (rate > 0 ? (remShards * SHARD_DUST) / rate : Infinity);
    var wait = Math.max(waitDust, waitShards);
    return {
      dust: totals.dust,
      shards: totals.shards,
      combined: totals.combined,
      remDust: remDust,
      remShards: remShards,
      remDustEq: remDustEq,
      waitDust: waitDust,
      waitShards: waitShards,
      wait: wait
    };
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
    } catch (e) {}
  }

  function getLockedBuild(level) {
    var entry = (state.lockedPlans && state.lockedPlans[String(level)]) || null;
    if (!entry) return null;
    return typeof entry === "string" ? entry : (entry.name || null);
  }

  function setLockedBuild(level, name) {
    if (!state.lockedPlans) state.lockedPlans = {};
    var all = buildsFor(level);
    var target = null;
    for (var k = 0; k < all.length; k++) {
      if (all[k].n === name) { target = all[k]; break; }
    }
    var rawSteps = target ? gapSteps(target) : [];
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
      name: name,
      steps: steps
    };
    try { localStorage.setItem("saPlanner.lockedPlans", JSON.stringify(state.lockedPlans)); } catch (e) {}
  }

  function clearLockedBuild(level) {
    if (state.lockedPlans && state.lockedPlans[String(level)]) {
      var entry = state.lockedPlans[String(level)];
      var name = typeof entry === "string" ? entry : (entry && entry.name);
      delete state.lockedPlans[String(level)];
      try { localStorage.setItem("saPlanner.lockedPlans", JSON.stringify(state.lockedPlans)); } catch (e) {}
      if (name) resetStepOrder(level, { n: name });
    }
  }

  function syncLockedPlan(level, target) {
    if (!state.lockedPlans || !target) return null;
    var entry = state.lockedPlans[String(level)];
    if (!entry) return null;
    var name = typeof entry === "string" ? entry : entry.name;
    if (!name || target.n !== name) return null;

    var currentGaps = gapSteps(target);
    var planSteps = (typeof entry === "object" && Array.isArray(entry.steps)) ? entry.steps : null;

    function evalStep(s) {
      var done = false;
      var curFrom = s.from;
      var curCost = s.cost;
      var curShards = !!s.shards;
      var reach = true;

      if (s.kind === "level") {
        var curLv = userLevel(s.key);
        done = curLv >= s.to;
        if (!done) {
          curFrom = curLv;
          curCost = upgradeCostRange(s.key, curLv, s.to);
          curShards = isShards(s.key);
        }
      } else if (s.kind === "limbs") {
        var curL = extraLimbs();
        done = curL >= s.to;
        if (!done) {
          curFrom = curL;
          curCost = extraLimbCost(curL, s.to);
        }
      } else if (s.kind === "ring") {
        var curR = hasRing() ? ringLevel() : 0;
        done = curR >= s.to;
        if (!done) {
          curFrom = Math.max(1, curR);
          curCost = ringCostRange(curFrom, s.to);
          curShards = true;
        }
      } else if (s.kind === "ringUnlock") {
        done = hasRing();
        if (!done) {
          curCost = oneTimerRingCost();
          curShards = true;
        }
      } else if (s.kind === "contract") {
        var cat = itemCat(s.key);
        var zone = (cat && cat.zone) || s.zone;
        reach = (state.radon + 1) >= zone;
        var owned = userOwns(s.key);
        if (s.after && s.after.to > 1) {
          var curLv = userLevel(s.key);
          done = owned && curLv >= s.after.to;
          if (!done && owned) {
            curCost = upgradeCostRange(s.key, curLv, s.after.to);
            curShards = isShards(s.key);
          }
        } else {
          done = owned;
          if (!done) {
            curCost = contractPrice(s.key);
            curShards = isShards(s.key);
          }
        }
      } else if (s.kind === "missing") {
        done = userOwns(s.key);
      }

      return {
        kind: s.kind,
        key: s.key || null,
        from: curFrom,
        to: s.to,
        cost: done ? 0 : curCost,
        shards: curShards,
        zone: s.zone,
        after: s.after || null,
        reach: reach,
        completed: done || !!s.completed
      };
    }

    if (planSteps && planSteps.length) {
      var evaluated = planSteps.map(evalStep);
      var existingKeys = {};
      evaluated.forEach(function (s) { existingKeys[stepKey(s)] = true; });
      currentGaps.forEach(function (g) {
        if (!existingKeys[stepKey(g)]) {
          evaluated.push(evalStep(g));
        }
      });
      if (!hasCustomStepOrder(level, target)) {
        var done = evaluated.filter(function (s) { return s.completed; });
        var pending = evaluated.filter(function (s) { return !s.completed; });
        evaluated = done.concat(sortSmartSteps(target, pending));
      }
      state.lockedPlans[String(level)] = { name: name, steps: evaluated };
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
          if (kind === "level" && target.i && target.i[itemKey]) {
            var need = target.i[itemKey];
            var curLv = userLevel(itemKey);
            initial.push({
              kind: "level",
              key: itemKey,
              from: Math.max(0, curLv - 1),
              to: need,
              cost: upgradeCostAt(itemKey, Math.max(0, curLv - 1)),
              shards: isShards(itemKey),
              completed: true
            });
          } else if (kind === "ring") {
            var rLv = (target.r && target.r.lv) || 1;
            initial.push({
              kind: "ring",
              from: Math.max(1, rLv - 1),
              to: rLv,
              cost: ringLevelCost(Math.max(1, rLv - 1)),
              shards: true,
              completed: true
            });
          } else if (kind === "limbs") {
            var lSlots = (target.l || 4) - 4;
            initial.push({
              kind: "limbs",
              from: Math.max(0, lSlots - 1),
              to: lSlots,
              cost: extraLimbCost(Math.max(0, lSlots - 1), lSlots),
              shards: false,
              completed: true
            });
          } else if (kind === "contract") {
            var cZone = (itemCat(itemKey) && itemCat(itemKey).zone) || target.c;
            initial.push({
              kind: "contract",
              key: itemKey,
              zone: cZone,
              completed: true
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

    var ordered = applyStepOrder(level, target, initial);
    var evaluatedInitial = ordered.map(evalStep);
    state.lockedPlans[String(level)] = { name: name, steps: evaluatedInitial };
    try { localStorage.setItem("saPlanner.lockedPlans", JSON.stringify(state.lockedPlans)); } catch (e) {}
    return evaluatedInitial;
  }

  function toggleLockBuild(name) {
    var cur = getLockedBuild(state.level);
    if (cur === name) {
      clearLockedBuild(state.level);
      toast("Unlocked advisor for SA " + state.level);
    } else {
      setLockedBuild(state.level, name);
      toast("Locked advisor to " + name);
    }
    var rec = bestRunnable(state.level);
    var rate = activeDustPs(rec);
    renderNext(rec, rate);
    renderBetter(rec, rate);
  }
  window.saToggleLockBuild = toggleLockBuild;

  function stepKey(st) {
    return st.kind + ":" + (st.key || "");
  }

  function planOrderKey(level, build) {
    return String(level) + "::" + (build && build.n ? build.n : "");
  }

  function hasCustomStepOrder(level, build) {
    var k = planOrderKey(level, build);
    return !!(state.stepOrder && state.stepOrder[k] && state.stepOrder[k].length);
  }

  function saveStepOrder(level, build, keys) {
    var k = planOrderKey(level, build);
    if (!state.stepOrder) state.stepOrder = {};
    state.stepOrder[k] = keys;
    try { localStorage.setItem("saPlanner.stepOrder", JSON.stringify(state.stepOrder)); } catch (e) {}
  }

  function resetStepOrder(level, build) {
    var k = planOrderKey(level, build);
    if (state.stepOrder && state.stepOrder[k]) {
      delete state.stepOrder[k];
      try { localStorage.setItem("saPlanner.stepOrder", JSON.stringify(state.stepOrder)); } catch (e) {}
    }
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
      var isEq = isStepEquipped(s);
      var needsLimb = false;

      if (limbStep && !isEq && (s.kind === "level" || s.kind === "contract" || s.kind === "missing")) {
        needsLimb = true;
      }

      var effWait = needsLimb ? (limbWait + w) : w;

      return {
        step: s,
        wait: w,
        effWait: effWait,
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

      if (Math.abs(sa.effWait - sb.effWait) > 1e-6) {
        return sa.effWait - sb.effWait;
      }

      if (sa.isEq !== sb.isEq) {
        return sa.isEq ? -1 : 1;
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
    var k = planOrderKey(level, build);
    var saved = state.stepOrder && state.stepOrder[k];
    if (!saved || !saved.length) return sortSmartSteps(build, steps);
    var rankMap = {};
    saved.forEach(function (id, idx) { rankMap[id] = idx; });
    return steps.slice().sort(function (a, b) {
      var ia = rankMap.hasOwnProperty(stepKey(a)) ? rankMap[stepKey(a)] : 999;
      var ib = rankMap.hasOwnProperty(stepKey(b)) ? rankMap[stepKey(b)] : 999;
      return ia - ib;
    });
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

  function stepLevel(delta) {
    var sel = $("levelSelect");
    if (!sel || !sel.options.length) return;
    var nextIdx = Math.max(0, Math.min(sel.options.length - 1, sel.selectedIndex + delta));
    if (nextIdx !== sel.selectedIndex) {
      sel.selectedIndex = nextIdx;
      render();
    }
  }

  function selectLevel(lv) {
    var sel = $("levelSelect");
    if (!sel) return;
    if (SA_DATA.builds[String(lv)]) {
      sel.value = String(lv);
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
      html += '<button type="button" class="btn sm ' + (state.level === fight ? 'primary' : 'ghost') + '" onclick="window.saSelectLevel&&saSelectLevel(' + fight + ')">Fighting (SA ' + fight + ')</button>';
    }
    if (max && max !== fight && SA_DATA.builds[String(max)]) {
      html += '<button type="button" class="btn sm ' + (state.level === max ? 'primary' : 'ghost') + '" onclick="window.saSelectLevel&&saSelectLevel(' + max + ')">Max (SA ' + max + ')</button>';
    }
    el.innerHTML = html;
  }

  function kpi(label, value, note, tone, icon) {
    return '<div class="kpi ' + (tone || 'tone-accent') + '">' +
      '<div class="kpi-label">' + (icon || '') + '<span>' + esc(label) + '</span></div>' +
      '<div class="kpi-value">' + esc(String(value)) + '</div>' +
      (note ? '<div class="kpi-note">' + esc(note) + '</div>' : '') +
      '</div>';
  }

  function renderChips(rec, rate) {
    var ab = state.ab;
    var owned = ownedItemKeys();
    var equipped = owned.filter(function (k) { return userItem(k) && userItem(k).equipped; }).length;
    var mods = ringMods().map(function (m) { return RING_MOD_NAMES[m] || m; }).join(" / ");
    $("chips").innerHTML = [
      kpi("Enemy Level", "SA " + (ab.enemyLevel || "—"), "Max unlocked: SA " + (ab.maxEnemyLevel || "—"), "tone-accent", ICONS.spire),
      kpi("Dust Bank", prettify(ab.dust), rate ? ("Active rate: " + prettify(rate) + "/s") : "No active rate", "tone-buy", ICONS.dust),
      kpi("Shard Bank", prettify(ab.shards), "= " + prettify((ab.shards || 0) * SHARD_DUST) + " dust", "tone-teal", ICONS.gem),
      kpi("Huffy Limbs", maxSlots() + " slots", equipped + " equipped · " + owned.length + "/" + SA_DATA.itemOrder.length + " owned", "tone-warn", ICONS.limb),
      kpi("The Ring", hasRing() ? ("Lv " + ringLevel()) : "Locked", hasRing() ? (mods || "No custom mods") : ("Unlock: " + prettify(oneTimerRingCost()) + " shards"), "tone-info", ICONS.ring),
      kpi("U2 Progress", "Z" + (state.radon + 1), ab.activeContract ? ("Active: " + itemName(ab.activeContract)) : "No active contract", "tone-violet", ICONS.contract)
    ].join("");
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
    var bnk = bank();
    var lockedName = getLockedBuild(state.level);
    if (lockedName) {
      var all = buildsFor(state.level);
      var target = null;
      for (var k = 0; k < all.length; k++) {
        if (all[k].n === lockedName) { target = all[k]; break; }
      }
      if (target) {
        var planSteps = syncLockedPlan(state.level, target);
        if (!planSteps || !planSteps.length) {
          planSteps = applyStepOrder(state.level, target, gapSteps(target));
        }
        var completedCount = planSteps.filter(function (s) { return s.completed; }).length;
        var pendingSteps = planSteps.filter(function (s) { return !s.completed; });
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
            gain: (target.d || 0) - cur
          };
        }
        var firstPending = null;
        var activeIdxInPlan = 0;
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
            gain: (target.d || 0) - cur,
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
      var steps = applyStepOrder(state.level, b, gapSteps(b));
      var wInfo = stepsWaitInfo(steps, 1, bnk);
      var effCost = wInfo.remDustEq > 0 ? wInfo.remDustEq : Math.max(1, wInfo.combined * 1e-6);
      var blocked = steps.some(function (s) { return s.kind === "contract" && !s.reach; });
      return { build: b, steps: steps, cost: wInfo.combined, remaining: steps.length, blocked: blocked, roi: ((b.d || 0) - cur) / effCost };
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
    if (!p || !first) return null;
    return {
      step: first,
      gain: (p.build.d || 0) - cur,
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
    state.level = level;
    var rec = bestRunnable(level);
    var rate = activeDustPs(rec);
    var est = sheetDustPs(rec);
    $("dustEstimate").textContent = rec ? ("Sheet: " + prettify(est) + "/s") : "No runnable build";
    $("controls").hidden = false;
    $("results").hidden = false;
    if ($("sourceStatus")) {
      $("sourceStatus").className = "pill ok";
      $("sourceStatus").textContent = "U2 Z" + (state.radon + 1) + " · SA " + (state.ab.enemyLevel || level) + "/" + (state.ab.maxEnemyLevel || level);
    }
    renderQuickLevels();
    renderChips(rec, rate);
    renderRecommended(rec, rate);
    renderNext(rec, rate);
    renderBoard(rec, rate);
    renderBetter(rec, rate);
  }

  function formatModList(mods) {
    return (mods || []).map(function (m) { return RING_MOD_NAMES[m] || m; }).join(" / ");
  }

  function loadoutDelta(rec) {
    if (!rec) return null;
    var reqKeys = Object.keys(rec.i || {});
    var eqKeys = ownedItemKeys().filter(function (k) {
      return userItem(k) && userItem(k).equipped;
    });
    var toEquip = reqKeys.filter(function (k) {
      return !(userItem(k) && userItem(k).equipped);
    });
    var toUnequip = eqKeys.filter(function (k) {
      return !rec.i[k];
    });
    var reqMods = (rec.r && rec.r.m) ? rec.r.m : [];
    var curMods = hasRing() ? ringMods() : [];
    var missingMods = reqMods.filter(function (m) {
      return curMods.indexOf(m) === -1;
    });
    var ringMismatch = missingMods.length > 0;
    var aligned = toEquip.length === 0 && toUnequip.length === 0 && !ringMismatch;
    return {
      toEquip: toEquip,
      toUnequip: toUnequip,
      reqMods: reqMods,
      curMods: curMods,
      missingMods: missingMods,
      ringMismatch: ringMismatch,
      aligned: aligned
    };
  }

  function renderRecommended(rec, rate) {
    var el = $("recommend");
    if (!rec) {
      el.innerHTML = "<div class='rec'><p class='empty'>No sheet build on SA " + state.level + " is fully runnable with your current items, levels, limbs, and ring. Check the Next Upgrade Advisor and Better Builds below for the fastest unlock path.</p></div>";
      return;
    }
    var delta = loadoutDelta(rec);
    var mods = formatModList(rec.r && rec.r.m || []);
    var curModTxt = formatModList(delta.curMods) || "None";
    var itemCards = Object.keys(rec.i).map(function (k) {
      var sh = isShards(k);
      var myLv = userLevel(k);
      var isEq = !!(userItem(k) && userItem(k).equipped);
      var cls = "loadout-item" + (sh ? " shards" : "") + (!isEq ? " unequipped" : "");
      var tip = isEq ? ("Equipped · Your level: Lv " + myLv) : ("Not equipped — equip into loadout · Your level: Lv " + myLv);
      var lvText = isEq ? ("Lv " + rec.i[k]) : ("Equip · Lv " + rec.i[k]);
      return "<div class='" + cls + "' title='" + esc(tip) + "'>" +
        "<span class='li-name'>" + iconForItem(k) + "<span>" + esc(itemName(k)) + "</span></span>" +
        "<span class='li-lv'>" + esc(lvText) + "</span>" +
        "</div>";
    }).join("");

    var metrics = [];
    metrics.push("<span class='pill sm buy'>" + ICONS.dust + " " + prettify(rec.d) + "/s</span>");
    if (rec.k != null) metrics.push("<span class='pill sm info'>" + ICONS.clock + " " + rec.k + "s kills</span>");
    if (rec.wr != null) metrics.push("<span class='pill sm accent'>" + Math.round((rec.wr <= 1 ? rec.wr : rec.wr / 100) * 100) + "% WR</span>");
    metrics.push("<span class='pill sm'>" + ICONS.limb + " " + (rec.l || "?") + " Limbs</span>");
    if (rec.r) {
      if (delta.ringMismatch) {
        metrics.push("<span class='pill sm warn-pill' title='Your Ring is currently set to " + esc(curModTxt) + " — switch to " + esc(mods) + "'>" + ICONS.warn + " Ring " + rec.r.lv + " (" + esc(curModTxt) + " → " + esc(mods) + ")</span>");
      } else {
        metrics.push("<span class='pill sm teal'>" + ICONS.ring + " Ring " + rec.r.lv + (mods ? " (" + esc(mods) + ")" : "") + "</span>");
      }
    }
    if (rec.c) metrics.push("<span class='pill sm violet'>" + ICONS.contract + " Z" + rec.c + "</span>");

    var statusHtml = delta.aligned
      ? "<span class='pill sm buy'>" + ICONS.check + " Active &amp; Aligned</span>"
      : "<div style='display:flex;gap:6px;flex-wrap:wrap;'><span class='pill sm warn-pill'>" + ICONS.warn + " Adjustments Needed</span><span class='pill sm buy'>" + ICONS.check + " Runnable</span></div>";

    var deltaHtml = "";
    if (!delta.aligned) {
      var dRows = [];
      if (delta.ringMismatch) {
        dRows.push("<div class='delta-row'><span class='delta-lbl'>Ring Mod:</span><span class='pill sm warn-pill'>" + ICONS.ring + " " + esc(curModTxt) + " → " + esc(mods) + "</span></div>");
      }
      if (delta.toUnequip.length) {
        var unPills = delta.toUnequip.map(function (k) {
          return "<span class='pill sm sell'>" + iconForItem(k) + " " + esc(itemName(k)) + "</span>";
        }).join("");
        dRows.push("<div class='delta-row'><span class='delta-lbl'>Unequip:</span>" + unPills + "</div>");
      }
      if (delta.toEquip.length) {
        var eqPills = delta.toEquip.map(function (k) {
          return "<span class='pill sm buy'>" + iconForItem(k) + " " + esc(itemName(k)) + "</span>";
        }).join("");
        dRows.push("<div class='delta-row'><span class='delta-lbl'>Equip:</span>" + eqPills + "</div>");
      }
      deltaHtml = "<div class='delta-box'>" +
        "<div class='delta-title'>" + ICONS.warn + "<span>Loadout Alignment Guide</span></div>" +
        "<div class='delta-rows'>" + dRows.join("") + "</div>" +
        "</div>";
    }

    el.innerHTML =
      "<div class='rec'>" +
      "<div class='rec-top'>" +
      "<div class='rec-head'><span>" + esc(rec.n) + "</span></div>" +
      statusHtml +
      "</div>" +
      "<div class='rec-metrics'>" + metrics.join("") + "</div>" +
      deltaHtml +
      "<div class='loadout-grid'>" + itemCards + "</div>" +
      (rec.note ? "<div class='notes'>" + ICONS.info + "<span>" + esc(rec.note) + "</span></div>" : "") +
      "</div>";
  }

  function stepFullCost(st) {
    var c = st.cost || 0;
    if (st.after && (!st.after.shards === !st.shards)) {
      c += st.after.cost || 0;
    }
    return c;
  }

  function renderStepPopover(pick, rate, bnk) {
    var steps = pick.allSteps && pick.allSteps.length ? pick.allSteps : [pick.step];
    var activeIdx = pick.activeIndex || 0;
    var completedCount = pick.completedCount || 0;
    var totalSteps = pick.totalSteps || steps.length;
    var pendingSteps = steps.filter(function (s) { return !s.completed; });
    var canDrag = pendingSteps.length > 1;
    var isCustom = canDrag && hasCustomStepOrder(state.level, pick.build);
    var availDust = bnk ? (bnk.dust || 0) : 0;
    var availShards = bnk ? (bnk.shards || 0) : 0;
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
      var w;
      if (st.shards) {
        var usedS = Math.min(availShards, totalStepCost);
        availShards -= usedS;
        var needS = Math.max(0, totalStepCost - usedS);
        w = needS <= 0 ? 0 : (rate > 0 ? (needS * SHARD_DUST) / rate : Infinity);
      } else {
        var usedD = Math.min(availDust, totalStepCost);
        availDust -= usedD;
        var needD = Math.max(0, totalStepCost - usedD);
        w = needD <= 0 ? 0 : (rate > 0 ? needD / rate : Infinity);
      }
      var costStr = totalStepCost ? (prettify(totalStepCost) + (st.shards ? " shards" : " dust")) : "—";
      var timeStr = w === 0 ? "Ready now" : (w != null ? fmtTime(w) : "—");
      var subLines = [];
      if (st.after && st.after.to > 1) {
        subLines.push("+ Lv 1 → " + st.after.to + " (" + prettify(st.after.cost || 0) + (st.after.shards ? " shards" : " dust") + ")");
      }
      if (isAct && st.to > (st.from || 0) + 1 && pick.step) {
        subLines.push("Now: Lv " + pick.step.from + " → " + pick.step.to + " (" + prettify(pick.step.cost || 0) + (pick.step.shards ? " shards" : " dust") + ")");
      } else if (isAct) {
        subLines.push("Current recommended step");
      } else {
        subLines.push("Step " + (idx + 1) + " of " + totalSteps);
      }
      return "<li class='sp-item" + (isAct ? " active" : "") + "'" +
        (canDrag ? " draggable='true' data-step-idx='" + idx + "' data-step-key='" + esc(stepKey(st)) + "' title='Drag to reorder step priority'" : "") + ">" +
        (canDrag ? "<span class='sp-grip'>" + ICONS.grip + "</span>" : "") +
        "<span class='sp-num'>" + (idx + 1) + "</span>" +
        "<div class='sp-main'>" +
        "<div class='sp-label " + stepClass(st) + "'>" + iconForStep(st) + "<span>" + esc(stepLabel(st)) + "</span></div>" +
        (subLines.length ? "<div class='sp-sub'>" + esc(subLines.join(" · ")) + "</div>" : "") +
        "</div>" +
        "<div class='sp-right'>" +
        "<span class='sp-cost'>" + esc(costStr) + "</span>" +
        "<span class='sp-time" + (w === 0 ? " ready" : "") + "'>" + esc(timeStr) + "</span>" +
        "</div>" +
        "</li>";
    }).join("");

    var wInfo = stepsWaitInfo(pendingSteps.length ? pendingSteps : steps, rate, bnk);
    var costParts = [];
    if (wInfo.dust > 0) costParts.push(prettify(wInfo.dust) + " dust");
    if (wInfo.shards > 0) costParts.push(prettify(wInfo.shards) + " shards");
    var totalCostLabel = costParts.length ? costParts.join(" + ") : "0 dust";
    var totalWaitStr = wInfo.wait === 0 ? "Ready now" : fmtTime(wInfo.wait);

    var remParts = [];
    if (wInfo.dust > 0) remParts.push(prettify(wInfo.remDust) + " dust");
    if (wInfo.shards > 0) remParts.push(prettify(wInfo.remShards) + " shards");
    var parHtml = "";
    if (wInfo.dust > 0 && wInfo.shards > 0) {
      var wdStr = wInfo.waitDust === 0 ? "0s" : fmtTime(wInfo.waitDust);
      var wsStr = wInfo.waitShards === 0 ? "0s" : fmtTime(wInfo.waitShards);
      parHtml = "<div class='sp-foot-sub'><span>Need: " + esc(remParts.join(" + ")) + "</span><span>Total: " + esc(wdStr) + " dust ∥ " + esc(wsStr) + " shards</span></div>";
    } else if ((wInfo.dust > 0 && (bnk.dust || 0) > 0) || (wInfo.shards > 0 && (bnk.shards || 0) > 0)) {
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
    return "<div class='step-popover' role='tooltip'>" +
      "<div class='sp-head'>" +
      "<span class='sp-title'>" + ICONS.clock + "<span>" + esc(titleTxt) + "</span>" + hintHtml + "</span>" +
      "<div class='sp-head-right'><span class='sp-target'>+" + prettify(pick.gain) + "/s → " + esc(pick.build.n) + "</span></div>" +
      "</div>" +
      "<ol class='sp-list' id='spStepList'>" + itemsHtml + "</ol>" +
      "<div class='sp-foot'>" +
      "<span>Total: <b>" + esc(totalCostLabel) + "</b></span>" +
      "<span>Est. wait: <b>" + esc(totalWaitStr) + "</b></span>" +
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
    }
    if (resetBtn) {
      resetBtn.addEventListener("click", function (ev) {
        ev.preventDefault();
        ev.stopPropagation();
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
        renderNext(rec, rate);
      });
    }
    if (!listEl || !pick.allSteps || pick.allSteps.length <= 1) return;
    var items = Array.prototype.slice.call(listEl.querySelectorAll(".sp-item[draggable='true']"));
    var dragFromIdx = null;

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
      if (toIdx == null || toIdx === dragFromIdx) return;
      var stepsCopy = pick.allSteps.slice();
      var moved = stepsCopy.splice(dragFromIdx, 1)[0];
      stepsCopy.splice(toIdx, 0, moved);
      var keys = stepsCopy.map(stepKey);
      saveStepOrder(state.level, pick.build, keys);
      if (state.lockedPlans && state.lockedPlans[String(state.level)]) {
        var entry = state.lockedPlans[String(state.level)];
        if (typeof entry === "object") {
          entry.steps = stepsCopy;
          try { localStorage.setItem("saPlanner.lockedPlans", JSON.stringify(state.lockedPlans)); } catch (e) {}
        }
      }
      state.keepPopoverOpen = true;
      renderNext(rec, rate);
    });
  }

  function renderNext(rec, rate) {
    var el = $("nextUpgrade");
    var pick = bestNextUpgrade(rec);
    if (!pick) {
      el.innerHTML = rec
        ? "<div class='next'><div class='next-kicker'>" + ICONS.check + " Peak Build Reached</div><p class='empty'>You already match the strongest sheet farm on SA " + state.level + ".</p></div>"
        : "<div class='next'><div class='next-kicker'>" + ICONS.info + " Next Upgrade</div><p class='empty'>No upgrade on this level matches the current filters.</p></div>";
      return;
    }
    if (pick.mode === "completed") {
      var btnUnlock = "<button type='button' class='btn sm accent lock-plan-btn locked' id='lockPlanBtn' title='Advisor is locked to this completed build. Click to unlock and allow advisor to recommend new builds.'>" + ICONS.lock + "<span>Locked (Done)</span></button>";
      var totalDone = pick.totalSteps || (pick.allSteps ? pick.allSteps.length : 1);
      var donePill = "<span class='pill sm buy'>" + ICONS.check + " " + totalDone + " of " + totalDone + " Steps Complete</span>";
      var bnk = bank();
      var badgeHtml = (pick.allSteps && pick.allSteps.length)
        ? "<div class='step-pill-wrap" + (state.keepPopoverOpen ? " open" : "") + "' id='stepPillWrap' tabindex='0'>" + donePill + renderStepPopover(pick, rate, bnk) + "</div>"
        : donePill;
      el.innerHTML =
        "<div class='next'>" +
        "<div class='next-top'>" +
        "<div class='next-kicker' style='color:var(--buy);'>" + ICONS.check + " Plan Completed</div>" +
        "<div class='next-top-actions'>" + btnUnlock + badgeHtml + "</div>" +
        "</div>" +
        "<div class='next-action-box'>" +
        "<div class='next-head' style='color:var(--buy);'>" + ICONS.check + "<span>" + esc(pick.build.n) + " is fully unlocked!</span></div>" +
        "<div class='next-meta'><span class='pill sm buy'>" + ICONS.check + " Runnable now · " + prettify(pick.build.d) + "/s</span></div>" +
        "</div>" +
        "<div class='next-why'>You have reached your locked target build (" + totalDone + " of " + totalDone + " steps completed). Click <b>Locked (Done)</b> to unlock the advisor for new recommendations.</div>" +
        "</div>";
      var btnComp = $("lockPlanBtn");
      if (btnComp) {
        btnComp.addEventListener("click", function () {
          clearLockedBuild(state.level);
          toast("Unlocked advisor for SA " + state.level);
          renderNext(rec, rate);
          renderBetter(rec, rate);
        });
      }
      bindPlanDrag(pick, rec, rate);
      return;
    }

    var s = pick.step;
    var bnk = bank();
    var rawCost = s.cost || 0;
    var wait = timeForCost(rawCost, !!s.shards, rate, s.shards ? 0 : bnk.dust, s.shards ? bnk.shards : 0);
    var costTxt = rawCost ? (prettify(rawCost) + (s.shards ? " shards" : " dust")) : "";
    var totalPlanSteps = pick.totalSteps || (pick.allSteps ? pick.allSteps.length : pick.remaining);
    var completedPlanSteps = pick.completedCount || 0;
    var currentStepNum = completedPlanSteps + 1;
    var isCustom = pick.remaining > 1 && hasCustomStepOrder(state.level, pick.build);
    var why;
    if (pick.isLocked) {
      if (completedPlanSteps > 0) {
        why = "Advisor locked to <b>" + esc(pick.build.n) + "</b> (Step " + currentStepNum + " of " + totalPlanSteps + " · " + pick.remaining + " steps remaining). Mid-plan recommendations will not switch until you click Unlock.";
      } else {
        why = "Advisor locked to <b>" + esc(pick.build.n) + "</b> (" + pick.remaining + " steps remaining). Mid-plan recommendations will not switch until you click Unlock.";
      }
    } else if (pick.mode === "unlocks") {
      why = "Highest dust/s gain per cost among single-step upgrades that immediately unlock a stronger sheet build (click Lock Plan to latch).";
    } else if (isCustom) {
      why = "Custom step order active — hover Step 1 of " + pick.remaining + " to drag and reorder your unlock plan.";
    } else {
      why = "First step on the fewest-gap path to a stronger sheet build (hover Step 1 of " + pick.remaining + " to reorder plan).";
    }

    var lockBtnHtml = pick.isLocked
      ? "<button type='button' class='btn sm accent lock-plan-btn locked' id='lockPlanBtn' title='Advisor is locked to " + esc(pick.build.n) + ". Click to unlock and re-enable automatic recommendations.'>" + ICONS.lock + "<span>Locked</span></button>"
      : "<button type='button' class='btn sm ghost lock-plan-btn' id='lockPlanBtn' title='Lock advisor to " + esc(pick.build.n) + " so recommendations do not switch mid-plan.'>" + ICONS.unlock + "<span>Lock Plan</span></button>";

    var kickerHtml = pick.isLocked
      ? "<div class='next-kicker' style='color:var(--accent);'>" + ICONS.lock + " Locked Plan</div>"
      : "<div class='next-kicker'>Recommended Next Action</div>";

    var pillInner = pick.mode === "unlocks"
      ? "<span class='pill sm buy'>" + ICONS.bolt + " 1-Step Unlock</span>"
      : "<span class='pill sm info'>" + ICONS.clock + " Step " + currentStepNum + " of " + totalPlanSteps + "</span>";
    var badgeHtml = "<div class='step-pill-wrap" + (state.keepPopoverOpen ? " open" : "") + "' id='stepPillWrap' tabindex='0'>" + pillInner + renderStepPopover(pick, rate, bnk) + "</div>";
    var waitBadge = wait === 0
      ? "<span class='pill sm buy'>" + ICONS.check + " Ready now</span>"
      : (wait != null ? "<span class='pill sm'>" + ICONS.clock + " " + fmtTime(wait) + "</span>" : "");

    var metaLockPill = pick.isLocked ? ("<span class='pill sm accent' title='Advisor locked to this plan'>" + ICONS.lock + " Plan Locked</span>") : "";

    el.innerHTML =
      "<div class='next'>" +
      "<div class='next-top'>" +
      kickerHtml +
      "<div class='next-top-actions'>" + lockBtnHtml + badgeHtml + "</div>" +
      "</div>" +
      "<div class='next-action-box'>" +
      "<div class='next-head " + stepClass(s) + "'>" + iconForStep(s) + "<span>" + esc(stepLabel(s)) + "</span></div>" +
      "<div class='next-meta'>" +
      (costTxt ? "<span class='pill sm accent'>" + (s.shards ? ICONS.gem : ICONS.dust) + " " + esc(costTxt) + "</span>" : "") +
      waitBadge +
      metaLockPill +
      "<span class='pill sm buy'>+" + prettify(pick.gain) + "/s → " + esc(pick.build.n) + " (" + prettify(pick.build.d) + "/s)</span>" +
      "</div>" +
      "</div>" +
      "<div class='next-why'>" + why + "</div>" +
      "</div>";

    var lockBtn = $("lockPlanBtn");
    if (lockBtn) {
      lockBtn.addEventListener("click", function () {
        toggleLockBuild(pick.build.n);
      });
    }

    bindPlanDrag(pick, rec, rate);
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
    if (!hide || (rec && rec.r)) html += ringCard(rec && rec.r, rate, bnk);
    if (rec && rec.l && rec.l > maxSlots()) html += limbsCard(rec.l, rate, bnk);
    $("board").innerHTML = html;
    if ($("boardMeta")) {
      $("boardMeta").textContent = "· " + order.length + " items shown" + (rec ? (" (" + readyCount + " build-ready)") : "");
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

    var pct = !owned ? 0 : (need ? Math.min(100, Math.round((lv / need) * 100)) : 100);
    var lines = [];
    var topLv = "";
    var meta = "";

    if (!owned) {
      topLv = "<div class='lv'><span>Locked</span>" + (cat && cat.zone ? "<span class='muted' style='font-size:12px;'>Z" + cat.zone + "</span>" : "") + "</div>";
      if (cat && cat.zone) {
        var active = state.ab.activeContract === key;
        var reach = (state.radon + 1) >= cat.zone;
        var cPrice = contractPrice(key);
        var tUnlock = timeForCost(cPrice, isShards(key), rate, bnk.dust, bnk.shards);
        meta += "<div class='cost'><span>Contract</span><span>" + prettify(cPrice) + " " + currencyName(key) + "</span></div>";
        meta += "<div class='time'><span>Wait</span><span>" + fmtTime(tUnlock) + "</span></div>";
        if (active) meta += "<div class='flag'>" + ICONS.info + " Accepted, not finished</div>";
        if (!reach) meta += "<div class='flag bad'>" + ICONS.warn + " Void map unreachable</div>";
        lines.push("Complete the Z" + cat.zone + " contract to unlock " + itemName(key) + ".");
      }
    } else {
      topLv = "<div class='lv'><span>Lv " + lv + (need && need > lv ? " <span class='gap'>→ " + need + "</span>" : "") + "</span>" +
        (equipped ? "<span class='muted' style='font-size:11px;font-weight:500;'>Equipped</span>" : (need ? "<span class='gap' style='font-size:11px;font-weight:600;'>Not equipped</span>" : "")) + "</div>";
      if (cat && !cat.noUp) {
        var next = upgradeCostAt(key, lv);
        var tNext = timeForCost(next, isShards(key), rate, bnk.dust, bnk.shards);
        meta += "<div class='cost'><span>Next (Lv " + (lv + 1) + ")</span><span>" + prettify(next) + " " + currencyName(key) + "</span></div>";
        meta += "<div class='time'><span>Wait</span><span>" + fmtTime(tNext) + "</span></div>";
        lines.push("Next upgrade: " + (cat.upgrade || "—") + ".");
        lines.push("Cost " + prettify(next) + " " + currencyName(key) + " (" + fmtTime(tNext) + ").");
        if (need > lv) {
          var gap = upgradeCostRange(key, lv, need);
          var tGap = timeForCost(gap, isShards(key), rate, bnk.dust, bnk.shards);
          meta += "<div class='gapcost'><span>To Lv " + need + "</span><span>" + prettify(gap) + " · " + fmtTime(tGap) + "</span></div>";
          lines.push("To reach the build: Lv " + lv + " → Lv " + need + ".");
        }
      } else {
        meta += "<div class='cost'><span>Upgrade</span><span>Unupgradable</span></div>";
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
      "<div class='lv-bar'><span style='width:" + pct + "%'></span></div>" +
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
        "<div class='time'><span>Wait</span><span>" + fmtTime(t) + "</span></div>";
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
      meta += "<div class='time'><span>Wait</span><span>" + fmtTime(tNext) + "</span></div>";
      if (need > lv) {
        var gap = ringCostRange(lv, need);
        meta += "<div class='gapcost'><span>To Lv " + need + "</span><span>" + prettify(gap) + " shards · " + fmtTime(timeForCost(gap, true, rate, bnk.dust, bnk.shards)) + "</span></div>";
        lines.push("To reach the build: Ring Lv " + lv + " → Lv " + need + ".");
      }
      lines.unshift(describeRing(lv, ringMods()));
      lines.push("Next level: all ring stats scale. " + nextRingSlotNote(lv + 1));
    }
    return "<article class='" + cls + "' title='" + esc(["The Ring"].concat(lines).join(" — ")) + "'>" +
      "<div>" +
      "<div class='card-head-row'><h3>" + ICONS.ring + "<span>The Ring</span></h3>" + badge + "</div>" +
      topLv +
      "<div class='lv-bar'><span style='width:" + pct + "%'></span></div>" +
      "</div>" +
      "<div class='card-meta-rows'>" + meta + "</div>" +
      cardTooltip("The Ring", lines) + "</article>";
  }

  function limbsCard(needSlots, rate, bnk) {
    var from = extraLimbs();
    var to = needSlots - 4;
    var cost = extraLimbCost(from, to);
    var t = timeForCost(cost, false, rate, bnk.dust, bnk.shards);
    var pct = Math.min(100, Math.round((maxSlots() / needSlots) * 100));
    return "<article class='card warn rec'>" +
      "<div>" +
      "<div class='card-head-row'><h3>" + ICONS.limb + "<span>Extra Limbs</span></h3><span class='card-badge warn'>Need " + needSlots + "</span></div>" +
      "<div class='lv'><span>" + maxSlots() + " slots <span class='gap'>→ " + needSlots + "</span></span></div>" +
      "<div class='lv-bar'><span style='width:" + pct + "%'></span></div>" +
      "</div>" +
      "<div class='card-meta-rows'>" +
      "<div class='cost'><span>Cost</span><span>" + prettify(cost) + " dust</span></div>" +
      "<div class='time'><span>Wait</span><span>" + fmtTime(t) + "</span></div>" +
      "</div>" +
      cardTooltip("Extra Limbs", ["Huffy can equip 1 additional item per level.", "Need " + from + " → " + to + "."]) +
      "</article>";
  }

  function renderBetter(rec, rate) {
    var cur = rec ? rec.d || 0 : 0;
    var bnk = bank();
    var list = betterBuilds(rec);
    list.sort(function (a, b) {
      var wa = stepsWaitInfo(gapSteps(a), 1, bnk);
      var wb = stepsWaitInfo(gapSteps(b), 1, bnk);
      var ca = wa.remDustEq > 0 ? wa.remDustEq : Math.max(1, wa.combined * 1e-6);
      var cb = wb.remDustEq > 0 ? wb.remDustEq : Math.max(1, wb.combined * 1e-6);
      var ra = ((a.d || 0) - cur) / ca;
      var rb = ((b.d || 0) - cur) / cb;
      return rb - ra;
    });
    list = list.slice(0, 8);
    if (!list.length) {
      $("better").innerHTML = rec
        ? "<p class='empty'>You already match the strongest runnable sheet build on this level, or nothing cheaper is listed with the current filters.</p>"
        : "<p class='empty'>No builds on this tab match the current filters.</p>";
      return;
    }
    var rows = list.map(function (b) {
      var steps = gapSteps(b);
      var cost = stepsCostDust(steps);
      var wInfo = stepsWaitInfo(steps, rate, bnk);
      var pay = wInfo.wait;
      var blocked = steps.some(function (s) { return s.kind === "contract" && !s.reach; });
      return "<tr" + (blocked ? " class='unreachable'" : "") + ">" +
        "<td><strong>" + esc(b.n) + "</strong>" +
        (blocked ? " <span class='pill sm sell' title='Requires a Void Map zone higher than your Highest Zone U2'>" + ICONS.warn + " Z" + (b.c || "?") + "</span>" : "") +
        "<div class='sub mono' style='margin-top:2px;'>" + prettify(b.d) + "/s" + (b.k != null ? " · " + b.k + "s" : "") + "</div></td>" +
        "<td><div class='steps'>" + steps.map(function (s) {
          return "<span class='step-badge " + stepClass(s) + "'>" + iconForStep(s) + "<span>" + esc(stepLabel(s)) + "</span></span>";
        }).join("") + "</div></td>" +
        "<td class='mono'>" + (cost.dust ? prettify(cost.dust) + " dust" : "") +
        (cost.dust && cost.shards ? "<br>" : "") +
        (cost.shards ? prettify(cost.shards) + " shards" : "") + "</td>" +
        "<td class='mono pos'>+" + prettify((b.d || 0) - cur) + "/s</td>" +
        "<td class='mono'>" + fmtTime(pay) + "</td>" +
        "<td>" + (function() {
          var isRowLocked = getLockedBuild(state.level) === b.n;
          return isRowLocked
            ? "<button type='button' class='btn sm accent lock-row-btn locked' data-lock-build='" + esc(b.n) + "' title='Advisor is locked to this build. Click to unlock.'>" + ICONS.lock + "<span>Locked</span></button>"
            : "<button type='button' class='btn sm ghost lock-row-btn' data-lock-build='" + esc(b.n) + "' title='Lock Next Upgrade Advisor to this build'>" + ICONS.unlock + "<span>Lock</span></button>";
        })() + "</td></tr>";
    }).join("");
    var betterEl = $("better");
    betterEl.innerHTML = "<div class='tbl-wrap'><table><thead><tr><th>Build</th><th>Missing Upgrades</th><th>Total Cost</th><th>Δ d/s</th><th>Payback</th><th>Plan</th></tr></thead><tbody>" + rows + "</tbody></table></div>";
    betterEl.onclick = function(ev) {
      var btn = ev.target.closest("button[data-lock-build]");
      if (!btn) return;
      ev.preventDefault();
      toggleLockBuild(btn.getAttribute("data-lock-build"));
    };
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
    var owned = ownedItemKeys();
    var equipped = owned.filter(function (k) { return userItem(k) && userItem(k).equipped; }).length;
    var contracts = owned.filter(function (k) { var c = itemCat(k); return c && c.zone; }).length;
    var contractItems = Object.keys(SA_DATA.items).filter(function (k) { return SA_DATA.items[k].zone; }).length;
    var contract = state.ab.activeContract;
    var mods = ringMods().map(function (m) { return RING_MOD_NAMES[m] || m; }).join(" / ");
    $("saveFacts").innerHTML = [
      fact("Highest Zone U2", state.radon + 1, ICONS.spire),
      fact("Void maps", "to Z" + (state.radon + 1), ICONS.shield),
      fact("Contract", contract ? itemName(contract) : "none", ICONS.contract),
      fact("Items", owned.length + "/" + SA_DATA.itemOrder.length, ICONS.sword),
      fact("Equipped", equipped + "/" + maxSlots(), ICONS.limb),
      fact("Contracts", contracts + "/" + contractItems, ICONS.check),
      fact("Ring mods", hasRing() ? (mods || "none") : "locked", ICONS.ring)
    ].join("");
  }

  function setSaveCollapsed(collapsed) {
    $("saveEditor").hidden = collapsed;
    $("saveSummary").hidden = !collapsed;
    if ($("savePanel")) {
      $("savePanel").classList.toggle("hero-mode", !collapsed && !state.ab);
    }
    if ($("cancelEditBtn")) {
      $("cancelEditBtn").classList.toggle("hidden", !state.ab);
    }
    if ($("heroFeats")) {
      $("heroFeats").classList.toggle("hidden", !!state.ab);
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
    var cur = rec ? rec.d || 0 : 0;
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
    var visible = betterBuilds(rec);
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
      state.filters.contract = !shouldDisableContractsOnLoad(state.level);
      applyFilterBoxes();
      $("workspace").hidden = false;
      renderSaveFacts();
      setSaveCollapsed(true);
      render();
      toast("Save loaded · SA Lv " + state.level + " (U2 Z" + (state.radon + 1) + ")");
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
    if ($("btnHelp") && $("helpDlg")) {
      $("btnHelp").addEventListener("click", function () { $("helpDlg").showModal(); });
      $("btnCloseHelp").addEventListener("click", function () { $("helpDlg").close(); });
      $("helpDlg").addEventListener("click", function (ev) {
        if (ev.target === $("helpDlg")) $("helpDlg").close();
      });
    }
    document.addEventListener("keydown", function (ev) {
      var tag = (ev.target && ev.target.tagName) || "";
      if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" || ev.ctrlKey || ev.metaKey || ev.altKey) return;
      if (ev.key === "t" || ev.key === "T") { ev.preventDefault(); toggleTheme(); }
      else if (ev.key === "u" || ev.key === "U") { ev.preventDefault(); onUpdate(); }
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
    loadPrefs();
    setTheme(state.theme);
    if ($("saveBox") && !$("saveEditor").hidden) $("saveBox").focus();
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
