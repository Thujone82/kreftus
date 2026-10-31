/* ==========================================================================
   Kreftus Local Customizations for Trimps
   1. Settings Drawer Theme Toggle (to the right of #searchSettings, default ON)
      - Turning OFF disables kreftus-local.css completely and clears JS helpers
   2. Dynamically sizes Spire Assault (#autoItemsDiv) to fit above #settingsRow
   3. Injects Mobile Quick-Switch Bar (<= 1100px) for fast 1-tap section filtering
   4. Dynamically auto-scales any overflowing text in resource/misc cards
   ========================================================================== */
(function () {
  'use strict';

  var THEME_STORAGE_KEY = 'kreftus.trimps.themeEnabled';
  var MVIEW_STORAGE_KEY = 'kreftus.trimps.mobileView';
  var MVIEW_TABS = [
    { id: 'all', label: '📱 All' },
    { id: 'combat', label: '⚔️ Combat' },
    { id: 'buy', label: '🏗️ Buy' },
    { id: 'resources', label: '📦 Resources' },
    { id: 'log', label: '📜 Log' }
  ];

  var FIT_SELECTORS = [
    '#food .ownedArea', '#wood .ownedArea', '#metal .ownedArea', '#science .ownedArea',
    '#fragments .ownedArea', '#gems .ownedArea', '#helium .ownedArea', '#trimps > .ownedArea',
    '#fragments > .title', '#gems > .title', '#helium > .title',
    '#foodPs', '#woodPs', '#metalPs', '#sciencePs',
    '#fragmentsPs', '#gemsPs', '#heliumPh'
  ];

  function isThemeEnabled() {
    try {
      return localStorage.getItem(THEME_STORAGE_KEY) !== 'false';
    } catch (e) {
      return true;
    }
  }

  function findThemeLink() {
    var links = document.querySelectorAll('link[rel="stylesheet"]');
    for (var i = 0; i < links.length; i++) {
      var href = links[i].getAttribute('href') || '';
      if (href.indexOf('kreftus-local.css') !== -1) {
        return links[i];
      }
    }
    return null;
  }

  function injectToggleBaseCss() {
    if (document.getElementById('kreftusThemeToggleStyle')) return;
    var style = document.createElement('style');
    style.id = 'kreftusThemeToggleStyle';
    style.textContent =
      '#settingTitleBar #searchSettingsTitle { margin: 0 1vw !important; }' +
      '#settingTitleBar #searchSettings { width: 20% !important; vertical-align: middle; }' +
      '#kreftusThemeToggleBtn {' +
      '  display: inline-block;' +
      '  vertical-align: middle;' +
      '  width: auto !important;' +
      '  min-width: 11vw;' +
      '  padding: 0.25vw 0.8vw;' +
      '  margin: 0 0 0 0.8vw !important;' +
      '  font-size: 1.35vw;' +
      '  line-height: 1.5vw;' +
      '  cursor: pointer;' +
      '  user-select: none;' +
      '}' +
      '@media screen and (max-width: 1100px) {' +
      '  #kreftusThemeToggleBtn {' +
      '    font-size: 14px !important;' +
      '    line-height: 1.3 !important;' +
      '    padding: 6px 12px !important;' +
      '    margin: 4px 0 4px 8px !important;' +
      '  }' +
      '}';
    document.head.appendChild(style);
  }

  function clearJsOverrides() {
    var nodes = document.querySelectorAll(FIT_SELECTORS.join(','));
    for (var i = 0; i < nodes.length; i++) {
      nodes[i].style.transform = '';
      nodes[i].style.transformOrigin = '';
    }
    var tip = document.getElementById('tooltipDiv');
    if (tip) {
      tip.style.maxHeight = '';
    }
    var itemsDiv = document.getElementById('autoItemsDiv');
    if (itemsDiv) {
      itemsDiv.style.maxHeight = '';
    }
  }

  function applyThemeState(enabled) {
    var link = findThemeLink();
    if (link) {
      link.disabled = !enabled;
    }

    var btn = document.getElementById('kreftusThemeToggleBtn');
    if (btn) {
      btn.textContent = enabled ? 'SA Theme: On' : 'SA Theme: Off';
      btn.classList.toggle('settingBtn1', enabled);
      btn.classList.toggle('settingBtntrue', enabled);
      btn.classList.toggle('settingBtn0', !enabled);
      btn.classList.toggle('settingBtnfalse', !enabled);
    }

    var nav = document.getElementById('kreftusMobileNav');
    if (!enabled) {
      if (nav) nav.style.display = 'none';
      MVIEW_TABS.forEach(function (t) {
        document.body.classList.remove('kreftus-mview-' + t.id);
      });
      clearJsOverrides();
    } else {
      if (nav) nav.style.display = '';
      var savedView = 'all';
      try {
        savedView = localStorage.getItem(MVIEW_STORAGE_KEY) || 'all';
      } catch (e) {}
      MVIEW_TABS.forEach(function (t) {
        document.body.classList.toggle('kreftus-mview-' + t.id, t.id === savedView);
      });
      fitResourceBoxText();
      adjustSpireAssaultPopup();
    }
  }

  function initThemeToggleButton() {
    injectToggleBaseCss();
    var searchInput = document.getElementById('searchSettings');
    if (!searchInput || document.getElementById('kreftusThemeToggleBtn')) return;

    var btn = document.createElement('div');
    btn.id = 'kreftusThemeToggleBtn';
    btn.className = 'noselect optionContainer settingsBtn settingTypeBtn';
    btn.title = 'Toggle custom SA Planner dark theme and mobile layout enhancements';
    btn.addEventListener('click', function () {
      var nextState = !isThemeEnabled();
      try {
        localStorage.setItem(THEME_STORAGE_KEY, nextState ? 'true' : 'false');
      } catch (e) {}
      applyThemeState(nextState);
    });

    if (searchInput.nextSibling) {
      searchInput.parentNode.insertBefore(btn, searchInput.nextSibling);
    } else {
      searchInput.parentNode.appendChild(btn);
    }

    applyThemeState(isThemeEnabled());
  }

  function adjustSpireAssaultPopup() {
    if (!isThemeEnabled()) return;

    var tip = document.getElementById('tooltipDiv');
    if (!tip || tip.style.display === 'none') return;

    /* Clamp regular tooltips onto screen on narrow viewports */
    if (window.innerWidth <= 1100 && !tip.classList.contains('tooltipExtraBiggest')) {
      tip.style.left = Math.max(8, Math.min(tip.offsetLeft || 12, window.innerWidth - tip.offsetWidth - 8)) + 'px';
      var maxTop = Math.max(8, window.innerHeight - tip.offsetHeight - 68);
      if (tip.offsetTop > maxTop) {
        tip.style.top = Math.max(8, maxTop) + 'px';
      }
    }

    if (!tip.classList.contains('tooltipExtraBiggest')) return;

    var atkBar = document.getElementById('autoBattleTrimpAttackBar');
    if (atkBar && atkBar.parentElement) atkBar.parentElement.classList.add('autoBattleAttackHolder');
    var eatkBar = document.getElementById('autoBattleEnemyAttackBar');
    if (eatkBar && eatkBar.parentElement) eatkBar.parentElement.classList.add('autoBattleAttackHolder');

    var itemsDiv = document.getElementById('autoItemsDiv');
    if (!itemsDiv) return;

    var settingsRow = document.getElementById('settingsRow');
    var bottomLimit = window.innerHeight - 16;
    if (settingsRow && settingsRow.offsetHeight > 0) {
      var srRect = settingsRow.getBoundingClientRect();
      if (srRect.top > 0) bottomLimit = srRect.top - 10;
    }

    tip.style.top = '12px';
    tip.style.maxHeight = Math.max(320, bottomLimit - 12) + 'px';

    var tipCost = document.getElementById('tipCost');
    var footerReserve = (tipCost ? tipCost.offsetHeight : 36) + 36;
    var itemsRect = itemsDiv.getBoundingClientRect();
    var availableForItems = bottomLimit - itemsRect.top - footerReserve;

    if (availableForItems > 180) {
      itemsDiv.style.maxHeight = Math.floor(availableForItems) + 'px';
    }
  }

  function fitResourceBoxText() {
    if (!isThemeEnabled()) return;

    var nodes = document.querySelectorAll(FIT_SELECTORS.join(','));
    for (var i = 0; i < nodes.length; i++) {
      var el = nodes[i];
      if (!el || el.offsetParent === null) continue;
      var isRateInCollectRow = el.id === 'foodPs' || el.id === 'woodPs' || el.id === 'metalPs' || el.id === 'sciencePs';
      var box = isRateInCollectRow ? el.closest('.col-xs-6') : el.closest('.playerGather');
      if (!box) continue;

      /* Reset transform before measuring natural width */
      el.style.transform = '';
      el.style.transformOrigin = '';

      var maxW = box.clientWidth - (isRateInCollectRow ? 4 : 12);
      var naturalW = el.scrollWidth;
      if (maxW > 20 && naturalW > maxW) {
        var scale = Math.max(0.65, maxW / naturalW);
        el.style.transformOrigin = isRateInCollectRow ? 'right center' : 'center center';
        el.style.transform = 'scaleX(' + scale.toFixed(3) + ')';
      }
    }
  }

  function initMobileQuickNav() {
    var innerWrapper = document.getElementById('innerWrapper');
    if (!innerWrapper || document.getElementById('kreftusMobileNav')) return;

    var nav = document.createElement('div');
    nav.id = 'kreftusMobileNav';
    nav.setAttribute('role', 'navigation');
    nav.setAttribute('aria-label', 'Mobile Section Switcher');

    var savedView = 'all';
    try {
      savedView = localStorage.getItem(MVIEW_STORAGE_KEY) || 'all';
    } catch (e) {}

    function setMobileView(viewId) {
      MVIEW_TABS.forEach(function (t) {
        document.body.classList.remove('kreftus-mview-' + t.id);
      });
      if (isThemeEnabled()) {
        document.body.classList.add('kreftus-mview-' + viewId);
      }
      var btns = nav.querySelectorAll('.kreftus-mnav-btn');
      for (var i = 0; i < btns.length; i++) {
        btns[i].classList.toggle('active', btns[i].getAttribute('data-view') === viewId);
      }
      try {
        localStorage.setItem(MVIEW_STORAGE_KEY, viewId);
      } catch (e) {}
    }

    MVIEW_TABS.forEach(function (t) {
      var btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'kreftus-mnav-btn' + (t.id === savedView ? ' active' : '');
      btn.setAttribute('data-view', t.id);
      btn.textContent = t.label;
      btn.addEventListener('click', function () {
        setMobileView(t.id);
      });
      nav.appendChild(btn);
    });

    innerWrapper.insertBefore(nav, innerWrapper.firstChild);
    setMobileView(savedView);
  }

  /* Apply saved theme state immediately even before DOMContentLoaded if link exists */
  var earlyLink = findThemeLink();
  if (earlyLink) {
    earlyLink.disabled = !isThemeEnabled();
  }

  function initKreftusEnhancements() {
    initMobileQuickNav();
    initThemeToggleButton();
    applyThemeState(isThemeEnabled());

    var tip = document.getElementById('tooltipDiv');
    if (!tip) {
      setTimeout(initKreftusEnhancements, 500);
      return;
    }

    var rafPending = false;
    var observer = new MutationObserver(function () {
      if (rafPending) return;
      rafPending = true;
      requestAnimationFrame(function () {
        rafPending = false;
        adjustSpireAssaultPopup();
      });
    });

    observer.observe(tip, {
      attributes: true,
      attributeFilter: ['class', 'style'],
      childList: true
    });

    window.addEventListener('resize', function () {
      adjustSpireAssaultPopup();
      fitResourceBoxText();
    });

    setInterval(fitResourceBoxText, 600);
    setTimeout(fitResourceBoxText, 150);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initKreftusEnhancements);
  } else {
    initKreftusEnhancements();
  }
})();
