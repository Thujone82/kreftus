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

  // ---------- Trimps 5.9.0 - 5.10.1 Compatibility Layer ----------
  function getPurchasedMutators() {
    if (isGameLoaded() && game.global && typeof game.global.u2MutationData === "object" && game.global.u2MutationData) {
      return game.global.u2MutationData;
    }
    return {};
  }

  // Runed mutator (5.9.0 outer ring) increases Runetrinket cap by 50% (1,500 per Obs level instead of 1,000)
  function getTrinketPerLevelCap() {
    var muts = getPurchasedMutators();
    return muts.Runed ? 1500 : 1000;
  }

  function applyTrimps510FixesAfterReadInputs() {
    if (typeof props !== "object" || !props) return;

    // 1. Trimps 5.10.0: Glass Radon bonus caps at Z400 (no extra 10% Radon/zone from pushing VM zone >= 400)
    props.glassRadon = props.glassDone && ((props.vmZone > 174 && props.vmZone < 200) || (props.vmZone > 200 && props.vmZone < 400));

    // 2. Trimps 5.8.0+: Mutated enemy stat scaling at Z201+ (+1% Attack and +2% Health per zone)
    if (props.targetZone >= 201) {
      props.logEnemyScaling += Math.log(1.01) + Math.log(1.02);
    }

    if (!isGameLoaded() || !game.global) return;
    var muts = getPurchasedMutators();

    // 3. Trimps 5.8.0: "Small Trimps" (u2Mutations.tree.Trimps) multiplies U2 max housing by 1.5x (used in Combat Respec)
    if (game.resources && game.resources.trimps) {
      var smallTrimpsMult = muts.Trimps ? 1.5 : 1;
      props.maxTrimps = game.resources.trimps.max * game.resources.trimps.maxMod * props.scaffoldingBonus * smallTrimpsMult;
    }

    // 4. Trimps 5.8.0 - 5.9.0: Randimp1 / Randimp2 and Tauntimp Farming mutators increase expected exotic imp-orts per zone
    var extraImpsPerZone = 0;
    if (muts.Randimp2) {
      extraImpsPerZone += 6.0 / 5; // 3 drops per Randimp instead of 1 (+2 drops across 3 Randimps/zone / 5 types)
    } else if (muts.Randimp1) {
      extraImpsPerZone += 3.0 / 5; // 2 drops per Randimp instead of 1 (+1 drop across 3 Randimps/zone / 5 types)
    }
    if (muts.Tauntimps) {
      extraImpsPerZone += 0.2; // +2 guaranteed Tauntimps every 10 zones
    }

    if (extraImpsPerZone > 0) {
      props.imperzone += extraImpsPerZone;

      // Recompute tauntimp housing correction coefficients with updated imperzone
      var housingTypes = { collector: [50, 1.12] };
      if (props.hubEnabled || props.specialChallenge === "downsize") {
        housingTypes.hut = [1, 1.24];
        housingTypes.house = [1, 1.22];
        housingTypes.mansion = [9, 1.20];
        housingTypes.hotel = [15, 1.18];
        housingTypes.resort = [26, 1.16];
        housingTypes.gateway = [31, 1.14];
      }
      var coefC = 0;
      var termR = 0;
      var typeCount = 0;
      for (var key in housingTypes) {
        if (!Object.prototype.hasOwnProperty.call(housingTypes, key)) continue;
        var startZone = housingTypes[key][0];
        var scaling = housingTypes[key][1];
        var tauntRate = Math.pow(1.003, props.imperzone);
        var scaleZones = props.targetZone - startZone;
        var houseRate = Math.log(1.25) / Math.log(scaling);
        var taunTemp = Math.pow(tauntRate, scaleZones);
        var tauntRm1 = tauntRate - 1;
        var factor = key === "collector" ? props.collectHubs : 1;
        coefC += factor * (taunTemp - 1) / tauntRm1;
        termR += factor * tauntRate * houseRate * (taunTemp * (scaleZones * tauntRm1 - 1) + 1) / (tauntRm1 * tauntRm1);
        typeCount++;
      }
      props.coefC = coefC / typeCount;
      props.termR = termR / typeCount;
    }

    // 5. Trimps 5.8.0: GeneAttack and GeneHealth mutators each divide breed speed (potency) by 50
    props.potency = 0.00085 * Math.pow(1.1, Math.floor(props.targetZone / 5)) * Math.pow(1.003, props.targetZone * props.imperzone);
    if (muts.GeneAttack) props.potency /= 50;
    if (muts.GeneHealth) props.potency /= 50;
  }

  function installTrimps510TrinketFunctions() {
    // Override getTrinketValues to use dynamic Runetrinket cap per level (1,500 with Runed mutator, 1,000 otherwise)
    window.getTrinketValues = function (motivCost, powCost, toughCost) {
      var capPerLevel = getTrinketPerLevelCap();
      var obsLevel = Math.max(1, 1 + perks.Observation.level);
      var shinies = props.trinkets + shinyCollect(perks.Observation.level + 1);
      if (props.trinkets == 0) {
        shinies += 10;
      }
      shinies = Math.min(shinies, obsLevel * capPerLevel);
      var cappedBaseTrinkets = Math.min(props.trinkets, obsLevel * capPerLevel);

      var marginalShinePerRun = (100 + shinies * obsLevel) / (100 + cappedBaseTrinkets * obsLevel);
      var marginalShinePerTrinket = 1 + obsLevel / (100 + shinies * obsLevel);

      var res = [];
      var shines = [marginalShinePerRun, marginalShinePerTrinket];
      for (var s = 0; s < shines.length; s++) {
        var marginalShine = shines[s];
        var motF = perks.Motivation.priceFact;
        var motN = (marginalShine - 1) * (20 + perks.Motivation.level);
        var motShineCost = motivCost / (motF - 1) * (Math.pow(motF, motN) - 1);
        var powF = perks.Power.priceFact;
        var powN = (marginalShine - 1) * (20 + perks.Power.level);
        var powShineCost = powCost / (powF - 1) * (Math.pow(powF, powN) - 1);
        var touF = perks.Toughness.priceFact;
        var touN = (marginalShine - 1) * (20 + perks.Toughness.level);
        var touShineCost = toughCost / (touF - 1) * (Math.pow(touF, touN) - 1);
        res.push(motShineCost + powShineCost + touShineCost);
      }
      res.push(shinies);
      return res;
    };

    // Override getObservationGains to respect Runed mutator cap
    window.getObservationGains = function (levels) {
      var capPerLevel = getTrinketPerLevelCap();
      var obsLevel = 1 + perks.Observation.level;

      var currentCollect = shinyCollect(obsLevel);
      var shinies = props.trinkets + currentCollect;
      shinies = Math.min(shinies, obsLevel * capPerLevel);

      var nextCollect = shinyCollect(obsLevel + levels);
      var shiniesNext = props.trinkets + nextCollect;
      if (props.trinkets == 0) shiniesNext += 10;
      shiniesNext = Math.min(shiniesNext, (obsLevel + levels) * capPerLevel);

      var directGain = (100 + shiniesNext * (obsLevel + levels)) / (100 + shinies * obsLevel);
      var maxDirectGain = Infinity;
      if (shinies >= obsLevel * capPerLevel) {
        maxDirectGain = (100 + (obsLevel + levels) * (obsLevel + levels) * capPerLevel) / (100 + obsLevel * shinies);
      }

      var moreTrinkets;
      if (props.trinkets >= (obsLevel + levels) * capPerLevel) {
        moreTrinkets = 0;
      } else if (props.trinkets >= obsLevel * capPerLevel) {
        moreTrinkets = Math.max(0, shiniesNext - props.trinkets);
      } else {
        moreTrinkets = shiniesNext - shinies;
      }

      return [directGain, moreTrinkets, maxDirectGain];
    };

    // Override iterateValueFeedback to respect Runed mutator cap on lines 1818 & 1831
    window.iterateValueFeedback = function (valueArray) {
      var capPerLevel = getTrinketPerLevelCap();
      var Va = valueArray[0];
      var Vh = valueArray[1];
      var Vm = valueArray[2];
      var Vf = valueArray[3];
      var Vres = valueArray[4];
      var Vrad = valueArray[5];
      var Vp = valueArray[6];
      var Ve = valueArray[7];
      var tribs = valueArray[8];
      var collectors = valueArray[9];
      var hubEnabled = valueArray[10];
      var mets = valueArray[11];
      var trinketRate = valueArray[12];
      var trinkets = valueArray[13];
      var obsLevel = valueArray[14];
      var Vpushed = valueArray[15] !== undefined ? valueArray[15] : 1;
      var VmDone = valueArray[16] !== undefined ? valueArray[16] : 1;
      var VfDone = valueArray[17] !== undefined ? valueArray[17] : 1;
      var VresDone = valueArray[18] !== undefined ? valueArray[18] : 1;
      var VpDone = valueArray[19] !== undefined ? valueArray[19] : 1;

      if (tribs < 1250) {
        var greedRes = getGreedResourceFeedback(Vf * Vres / (VfDone * VresDone), tribs);
        var greedback = greedRes[0];
        tribs = greedRes[1];
        Vres *= greedback;
        Vrad *= greedback;
      }

      var moreHousing = Math.log(Vf * Vres / (VfDone * VresDone)) / Math.log(1.12);
      var baseHousing = collectors;
      if (hubEnabled || props.specialChallenge == "downsize") {
        moreHousing *= props.collectHubs;
        moreHousing += 5 * Math.log(Vres / VresDone) / Math.log(1.2);
        baseHousing *= 6 + props.collectHubs;
      }
      var tauntCorrectedHousingBase = baseHousing * props.coefC - props.termR;
      var tauntCorrectedHousingNext = (baseHousing + moreHousing) * props.coefC - props.termR;
      var housingGain = tauntCorrectedHousingNext / tauntCorrectedHousingBase;
      if (!(housingGain > 1) || tauntCorrectedHousingBase <= 0) housingGain = 1;
      if (props.specialChallenge == "downsize") {
        var trumPop = 2 * props.targetZone * (5 + perks.Trumps.level);
        housingGain = 1 + (housingGain - 1) * baseHousing / (baseHousing + trumPop);
      }
      Vres *= housingGain;
      Vp *= (props.specialChallenge == "trappa" || props.specialChallenge == "combat") ? 1 : housingGain;
      collectors += housingGain;

      var VmAdjusted = Vm * Vres / (VmDone * VresDone);
      if (!(props.specialChallenge == "combat")) {
        Va *= Math.pow(props.equipScaling.attack, Math.log(VmAdjusted));
        Vh *= Math.pow(props.equipScaling.health, Math.log(VmAdjusted));
      }

      var smithyGain = Math.pow(1.25, Math.log(Vres / VresDone) / Math.log(props.smithyScaling));
      if (!(props.specialChallenge == "combat") && !(props.specialChallenge == "smithless")) {
        Va *= smithyGain;
        Vh *= smithyGain;
      }

      var coordAdjust = props.coordLimited;
      if (props.specialChallenge == "downsize") {
        coordAdjust = Math.max(coordAdjust, 1);
      } else if (props.specialChallenge == "trappa" || (props.specialChallenge == "combat" && props.isTrappa)) {
        coordAdjust = 1;
      } else if (props.specialChallenge == "combat") {
        coordAdjust = perks.Carpentry.level < props.carpNeeded ? 1 : 0;
      }
      var coordGain = Math.pow(1.25, Math.log(Vp / VpDone) / Math.log(1.25) * coordAdjust);
      Va *= coordGain;
      Vh *= coordGain;

      var moreMets = Math.log(Vf * Vres / (VfDone * VresDone)) / Math.log(5);
      var metEff = 0.01 + 0.0005 * props.antennae;
      var metRes = 0.5 + 0.25 * (props.antennae >= 20 ? (Math.floor(props.antennae / 5) - 3) : 0);
      var metProd = mets * metEff;
      var metProdNext = (mets + moreMets) * metEff;
      var metRadGain = (1 + metProdNext) / (1 + metProd);
      var metFoodGain = props.antennae >= 5 ? (1 + metProdNext * metRes) / (1 + metProd * metRes) : 1;
      var metHPGain = props.antennae >= 10 ? (1 + metProdNext * metRes) / (1 + metProd * metRes) : 1;
      var metMineGain = props.antennae >= 15 ? (1 + metProdNext * metRes) / (1 + metProd * metRes) : 1;
      Vrad *= metRadGain;
      Vf *= metFoodGain;
      if (!(props.specialChallenge == "combat")) Vh *= metHPGain;
      Vm *= metMineGain;
      mets += moreMets;

      var pushPower = Va * ((props.specialChallenge == "combat" && props.isTrappa) ? 1 : Vh) / Vpushed;
      var pushZones = Math.log(pushPower) / props.logEnemyScaling;
      if (props.specialChallenge == "resplus") {
        Vres *= Math.pow(1.1, pushZones);
      } else {
        Vres *= Math.pow(1.25, pushZones);
      }
      Va *= Math.pow(1.25, pushZones * (1 - Math.min(1, coordAdjust)));
      Vh *= Math.pow(1.25, pushZones * (1 - Math.min(1, coordAdjust)));

      if (props.clearWeight >= 0 && props.glassRadon) {
        Vrad *= Math.pow(1.1, pushZones);
      }

      var s3RnFinal = props.s3Rn || (props.clearWeight == 0 && (perks.Observation.level == 0 || props.trinkets >= ((1 + perks.Observation.level) * capPerLevel)));
      if (props.scruffyLevel >= 3 && s3RnFinal) {
        Vrad *= Math.pow(1.03, pushZones);
      }
      if (s3RnFinal) {
        Vrad *= 1 + vmRadFrac * pushZones / (400 + props.targetZone);
      }

      var moreTrinkets = pushZones * trinketRate;
      var trinketMax = capPerLevel * obsLevel;
      if (moreTrinkets + trinkets > trinketMax) moreTrinkets = trinketMax - trinkets;
      if (moreTrinkets < 0) moreTrinkets = 0;
      var trinketGain = 1 + moreTrinkets * obsLevel / (100 + trinkets * obsLevel);
      Va *= trinketGain;
      Vh *= trinketGain;
      Vres *= trinketGain;
      trinkets += moreTrinkets;

      valueArray[0] = Va;
      valueArray[1] = Vh;
      valueArray[2] = Vm;
      valueArray[3] = Vf;
      valueArray[4] = Vres;
      valueArray[5] = Vrad;
      valueArray[6] = Vp;
      valueArray[7] = Ve;
      valueArray[8] = tribs;
      valueArray[9] = collectors;
      valueArray[11] = mets;
      valueArray[13] = trinkets;
      valueArray[15] = Va * ((props.specialChallenge == "combat" && props.isTrappa) ? 1 : Vh);
      valueArray[16] = Vm;
      valueArray[17] = Vf;
      valueArray[18] = Vres;
      valueArray[19] = Vp;
    };
  }

  // Update trinket/zone-push recommendation callouts after autobuyPerks when Runed raises cap above 1000/level
  function updateRunedStrategyCallouts() {
    var capPerLevel = getTrinketPerLevelCap();
    if (capPerLevel <= 1000 || typeof perks !== "object" || !perks.Observation) return;
    var obsEffLevel = 1 + perks.Observation.level;
    if (perks.Observation.level <= 0 || props.trinkets < obsEffLevel * 1000 || props.trinkets >= obsEffLevel * capPerLevel) {
      return;
    }
    var trinketsPerRun = shinyCollect(obsEffLevel);
    if (typeof trinkElem === "object" && trinkElem) {
      var TText = "Average trinkets/run: " + trinketsPerRun.toPrecision(3);
      TText += "\n\xa0\xa0(equivalent to about " + props.trinketRadonPerRun.toExponential(3) + " radon)";
      trinkElem.innerText = TText;
    }
    if (props.scruffyLevel < 23 && typeof zRecElem === "object" && zRecElem) {
      var trinketsPerZone = obsDropRate(obsEffLevel, props.targetZone) / 100;
      var trinketGainPct = trinketsPerZone / trinketsPerRun * 100;
      var ZText = "Pushing one more zone would give about ";
      var rnGainPerZone = (props.scruffyLevel >= 3 ? 1.03 : 1) * (1 + vmRadFrac / (400 + props.targetZone));
      var rnGainPct = (rnGainPerZone - 1) * 100;
      var totGainPct = rnGainPct;
      ZText += rnGainPct.toPrecision(3) + "% more radon next run";
      if (trinketsPerZone > 0) {
        ZText += " and";
        var effRnGain = props.s3Rn ? rnGainPerZone : 1;
        var totGainPerZone = (effRnGain * props.radonPerRun + props.trinketRadonPerRun + props.radonPerTrinket * trinketsPerZone) / (props.radonPerRun + props.trinketRadonPerRun);
        totGainPct = (totGainPerZone - 1) * 100;
        ZText += " " + trinketGainPct.toPrecision(3) + "% more trinkets this run.";
      } else {
        ZText += ".";
      }
      if (perks.Observation.level == 0 || props.s3Rn) {
        ZText += "\n\nIf the next run will be a radon challenge, pushing one more zone is worth extending the current run by about " + totGainPct.toPrecision(3) + "% of the run time of a typical farming run.";
      } else if (props.radonWeight > 0 && trinketsPerZone > 0) {
        ZText += "\n\nConsidering only the immediate trinket gains, pushing one more zone is worth extending the current run by about " + totGainPct.toPrecision(3) + "% of the current run time.";
      }
      if (trinketsPerZone > 0) {
        var nextCacheZone = Math.ceil((props.targetZone + 2) / 25) * 25 - 1;
        var nextCacheRnGain = 1 + vmRadFrac * (nextCacheZone - props.targetZone) / (400 + props.targetZone);
        nextCacheRnGain *= props.scruffyLevel >= 3 ? Math.pow(1.03, nextCacheZone - props.targetZone) : 1;
        var nextCacheTrinkets = shinyCollectLoop(obsEffLevel, nextCacheZone) - shinyCollectLoop(obsEffLevel, props.targetZone);
        if (!props.s3Rn) nextCacheRnGain = 1;
        var nextCacheTotGain = (nextCacheRnGain * props.radonPerRun + props.trinketRadonPerRun + props.radonPerTrinket * nextCacheTrinkets) / (props.radonPerRun + props.trinketRadonPerRun);
        var cacheGainPct = (nextCacheTotGain - 1) * 100;
        ZText += "\n\nPushing to the next trinket cache by clearing zone " + nextCacheZone + " would give " +
          nextCacheTrinkets.toPrecision(3) +
          " more trinkets (in addition to any Radon gains if you have S3 selected), which would be worth extending the run by about " +
          cacheGainPct.toPrecision(3) + "% of the current run time.";
      }
      zRecElem.innerText = ZText;
    }
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
      var muts = getPurchasedMutators();
      var mutCount = Object.keys(muts).filter(function (k) { return !!muts[k]; }).length;
      var parts = ["U" + u + " Z" + z];
      if (hzeU2 > 0) parts.push("HZE " + hzeU2);
      if (saLv > 0) parts.push("SA " + saLv);
      if (mutCount > 0) parts.push("Mut " + mutCount);
      if (statusPill) {
        statusPill.textContent = parts.join(" · ");
        statusPill.classList.add("ok");
        var active510Notes = [];
        if (muts.Runed) active510Notes.push("Runed (+50% Trinket cap)");
        if (muts.Trimps) active510Notes.push("Small Trimps (1.5x Housing)");
        if (muts.GeneAttack) active510Notes.push("GeneAttack (/50 breed)");
        if (muts.GeneHealth) active510Notes.push("GeneHealth (/50 breed)");
        if (muts.Randimp2) active510Notes.push("Really Randimp (3x)");
        else if (muts.Randimp1) active510Notes.push("Randimp (2x)");
        if (muts.Tauntimps) active510Notes.push("Tauntimp Farming");
        statusPill.title = active510Notes.length
          ? "Active 5.10.1 Mutator adjustments: " + active510Notes.join(", ")
          : "Loaded Trimps save summary";
      }
      if (radonPill && typeof props === "object" && props.perksRadon > 0) {
        radonPill.textContent = "Rn: " + formatSci(props.perksRadon);
        radonPill.classList.remove("hidden");
      }
    } else {
      if (statusPill) {
        statusPill.textContent = "No save loaded";
        statusPill.classList.remove("ok");
        statusPill.title = "";
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

    // Trimps 5.10.1: Remove copy-pasted max="30" attribute from Smithology & Expansion inputs (uncapped in config.js)
    var smithInput = $("input-smithology");
    if (smithInput) smithInput.removeAttribute("max");
    var expPerkInput = $("input-expansion");
    if (expPerkInput) expPerkInput.removeAttribute("max");

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
      '<div><b>Surky Perk Optimizer</b> · Universe 2 Radon calculator by <a href="https://github.com/nsheetz/perks" target="_blank" rel="noopener">nsheetz</a> (updated for Trimps 5.10.1)</div>' +
      '<div>/*__UPSTREAM_META__*/ · <a href="surky-build/README.html" target="_blank" rel="noopener">Documentation</a> · <a href="#top">Back to Top ↑</a></div>';
    document.body.appendChild(footer);
  }

  function wrapUpstreamFunctions() {
    installTrimps510TrinketFunctions();

    if (typeof window.readInputs === "function") {
      var origReadInputs = window.readInputs;
      window.readInputs = function () {
        var res = origReadInputs.apply(this, arguments);
        applyTrimps510FixesAfterReadInputs();
        return res;
      };
    }

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
          updateRunedStrategyCallouts();
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
        updateRunedStrategyCallouts();
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
