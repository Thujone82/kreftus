// Small helpers for top-level UI bits: progress bar, toast, modal open/close,
// stats in the Settings modal. App wiring lives in app.js.

(function (global) {
    'use strict';

    // --- Progress bar -----------------------------------------------------

    let progressTimer = null;

    function showProgress(label) {
        const bar = document.getElementById('progressBar');
        const lbl = document.getElementById('progressBarLabel');
        const fill = document.getElementById('progressBarFill');
        if (!bar) return;
        if (lbl) lbl.textContent = label || 'Working\u2026';
        if (fill) fill.style.width = '0%';
        bar.classList.remove('hidden');
    }

    function updateProgress(done, total, labelPrefix) {
        const lbl = document.getElementById('progressBarLabel');
        const fill = document.getElementById('progressBarFill');
        if (!fill || !lbl) return;
        const pct = (total > 0) ? Math.min(100, Math.round((done / total) * 100)) : 0;
        fill.style.width = pct + '%';
        const prefix = labelPrefix || 'Working';
        lbl.textContent = total > 0
            ? `${prefix}\u2026 ${done} / ${total} (${pct}%)`
            : `${prefix}\u2026`;
    }

    function hideProgress(delayMs) {
        const bar = document.getElementById('progressBar');
        if (!bar) return;
        if (progressTimer) clearTimeout(progressTimer);
        progressTimer = setTimeout(() => bar.classList.add('hidden'), delayMs || 600);
    }

    // --- Toast -------------------------------------------------------------

    let toastTimer = null;
    function toast(message, durationMs) {
        const el = document.getElementById('toast');
        if (!el) return;
        el.textContent = message;
        el.classList.remove('hidden');
        if (toastTimer) clearTimeout(toastTimer);
        toastTimer = setTimeout(() => el.classList.add('hidden'), durationMs || 3200);
    }

    // --- Modal -------------------------------------------------------------

    function openModal(id) {
        const el = document.getElementById(id);
        if (el) el.classList.remove('hidden');
    }
    function closeModal(id) {
        const el = document.getElementById(id);
        if (el) el.classList.add('hidden');
    }

    // --- Stats panel (inside Settings modal) -------------------------------

    async function refreshStats() {
        const trees = await HeritageDB.getAllTrees();
        const total = trees.length;
        const found = trees.reduce((n, t) => n + (t.found ? 1 : 0), 0);
        const removed = trees.reduce((n, t) => n + (t.removed != null ? 1 : 0), 0);
        const lastSync = await HeritageDB.getMeta('lastSyncAt');

        const setText = (id, v) => { const e = document.getElementById(id); if (e) e.textContent = v; };
        setText('statTotal',   String(total));
        setText('statFound',   String(found));
        setText('statRemoved', String(removed));
        setText('statLastSync', lastSync ? formatLocalDate(lastSync) : 'never');
    }

    function formatLocalDate(iso) {
        try {
            return new Date(iso).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' });
        } catch (e) { return iso || ''; }
    }

    // --- Nearby panel open/close ------------------------------------------

    function openNearby() {
        const panel = document.getElementById('nearbyPanel');
        if (!panel) return;
        panel.classList.remove('hidden');
        panel.setAttribute('aria-hidden', 'false');
        HeritageNearby.render();
    }
    function closeNearby() {
        const panel = document.getElementById('nearbyPanel');
        if (!panel) return;
        panel.classList.add('hidden');
        panel.setAttribute('aria-hidden', 'true');
    }

    global.HeritageUI = {
        showProgress,
        updateProgress,
        hideProgress,
        toast,
        openModal,
        closeModal,
        refreshStats,
        openNearby,
        closeNearby,
        formatLocalDate
    };
})(window);
