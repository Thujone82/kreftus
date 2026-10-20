/* Local overlay - not from upstream; restored by updtrimps.ps1
 * Cap Spire Assault to the real window (innerHeight minus Save bar).
 * Do not force height: short tabs stay compact; long tabs scroll.
 */
(function () {
  function toolbarHeight() {
    var row = document.getElementById('settingsRow');
    if (!row) return 0;
    var r = row.getBoundingClientRect();
    return r.height > 0 ? Math.ceil(r.height) : 0;
  }

  function sizeSpireAssault() {
    var el = document.getElementById('tooltipDiv');
    if (!el) return;
    var isSa = el.className.indexOf('tooltipExtraBiggest') !== -1 && el.style.display !== 'none';
    if (!isSa) {
      if (el.getAttribute('data-sa-sized')) {
        el.style.height = '';
        el.style.maxHeight = '';
        el.removeAttribute('data-sa-sized');
      }
      return;
    }
    var h = window.innerHeight - toolbarHeight();
    if (h < 80) h = 80;
    var next = h + 'px';
    if (el.style.maxHeight === next && !el.style.height) return;
    el.style.top = '0px';
    el.style.height = '';
    el.style.maxHeight = next;
    el.setAttribute('data-sa-sized', '1');
  }

  window.addEventListener('resize', sizeSpireAssault);
  if (window.visualViewport) {
    window.visualViewport.addEventListener('resize', sizeSpireAssault);
  }

  function bind() {
    sizeSpireAssault();
    var el = document.getElementById('tooltipDiv');
    if (!el || !window.MutationObserver) return;
    new MutationObserver(sizeSpireAssault).observe(el, {
      attributes: true,
      attributeFilter: ['class', 'style']
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', bind);
  } else {
    bind();
  }
})();
