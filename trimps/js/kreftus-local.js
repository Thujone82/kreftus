/* Local overlay - not from upstream; restored by updtrimps.ps1
 *
 * Cap #autoItemsDiv to the pixels between its top edge and the Save bar.
 * Do not size the tooltip: that stretches short tabs and still does not
 * create a scrollport on the items list.
 */
(function () {
  var raf = 0;

  function sizeSpireAssault() {
    var tip = document.getElementById('tooltipDiv');
    var items = document.getElementById('autoItemsDiv');
    var bar = document.getElementById('settingsRow');
    if (!tip) return;

    var isSa = tip.className.indexOf('tooltipExtraBiggest') !== -1 && tip.style.display === 'block';
    if (!isSa) {
      if (items) items.style.maxHeight = '';
      return;
    }
    if (!items || !bar) return;

    var barTop = bar.getBoundingClientRect().top;
    var itemsRect = items.getBoundingClientRect();
    var tipRect = tip.getBoundingClientRect();
    if (itemsRect.height === 0) return;

    var belowItems = Math.max(0, tipRect.bottom - itemsRect.bottom);
    var itemsMax = Math.floor(barTop - itemsRect.top - belowItems - 4);
    if (itemsMax < 64) itemsMax = 64;

    var scroll = items.scrollTop;
    if (items.scrollHeight > itemsMax + 1) {
      var next = itemsMax + 'px';
      if (items.style.maxHeight !== next) items.style.maxHeight = next;
    } else if (items.style.maxHeight !== 'none') {
      items.style.maxHeight = 'none';
    }
    if (items.scrollTop !== scroll) items.scrollTop = scroll;
  }

  function schedule() {
    if (raf) return;
    raf = window.requestAnimationFrame(function () {
      raf = 0;
      sizeSpireAssault();
    });
  }

  function hookPopup() {
    if (!window.autoBattle || !autoBattle.popup || autoBattle.popup._kreftus) return;
    var orig = autoBattle.popup;
    var wrapped = function () {
      var ret = orig.apply(this, arguments);
      schedule();
      return ret;
    };
    wrapped._kreftus = true;
    autoBattle.popup = wrapped;
  }

  window.addEventListener('resize', schedule);
  if (window.visualViewport) {
    window.visualViewport.addEventListener('resize', schedule);
  }

  function bind() {
    hookPopup();
    schedule();
    var tip = document.getElementById('tooltipDiv');
    if (!tip || !window.MutationObserver) return;
    new MutationObserver(schedule).observe(tip, {
      attributes: true,
      attributeFilter: ['class', 'style']
    });
    new MutationObserver(schedule).observe(tip, {
      childList: true,
      subtree: true
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', bind);
  } else {
    bind();
  }
})();
