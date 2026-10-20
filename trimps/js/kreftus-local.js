/* Local overlay - not from upstream; restored by updtrimps.ps1
 * Size Spire Assault to the real window (innerHeight), not vh/dvh.
 * Desktop Trimps does not update 100vh correctly on resize.
 */
(function () {
  function toolbarHeight() {
    var row = document.getElementById('settingsRow');
    return (row && row.offsetParent !== null && row.offsetHeight) ? row.offsetHeight : 0;
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
    if (el.style.height === next && el.style.top === '0px') return;
    el.style.top = '0px';
    el.style.height = next;
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
