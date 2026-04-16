// Throttled geocoding queue (fallback only).
//
// The bundled snapshot is pre-geocoded by heritage/heritage.ps1, so this
// module almost never has any work to do. It exists so that any tree whose
// address couldn't be resolved by the script (rare) can still be attempted
// from the browser.
//
// Uses the free OpenStreetMap Nominatim API (no API key needed). Nominatim's
// Acceptable Use Policy caps us at 1 request/second for a shared endpoint
// and asks for an identifying User-Agent. Browsers silently drop the
// User-Agent header so the request actually goes out with the browser's
// native UA; that's fine for the tiny volume this fallback produces.

(function (global) {
    'use strict';

    const PORTLAND_OR_SUFFIX = ', Portland, OR, USA';
    const STEP_MS = 1200;
    const BASE = 'https://nominatim.openstreetmap.org/search';

    let running = false;

    function wait(ms) { return new Promise((res) => setTimeout(res, ms)); }

    function sanitizeLocation(loc) {
        if (!loc) return '';
        let t = String(loc);
        t = t.replace(/\([^)]*\)/g, ' ');
        t = t.replace(/\s+/g, ' ').trim();
        return t;
    }

    function buildAddress(tree) {
        const loc = sanitizeLocation(tree.location);
        if (!loc) return null;
        if (/removed from list/i.test(loc)) return null;
        if (/\bPortland\b/i.test(loc)) {
            if (!/\bOR\b|\bOregon\b/i.test(loc)) return loc + ', OR, USA';
            if (!/\bUSA\b|\bUS\b|United States/i.test(loc)) return loc + ', USA';
            return loc;
        }
        return loc + PORTLAND_OR_SUFFIX;
    }

    async function geocodeOnce(address) {
        const url = `${BASE}?q=${encodeURIComponent(address)}&format=json&limit=1&countrycodes=us&addressdetails=1`;
        try {
            const resp = await fetch(url, {
                headers: {
                    'Accept': 'application/json',
                    // Browsers strip the User-Agent header but setting it is harmless.
                    'User-Agent': 'PDXHeritageTrees/1.0'
                }
            });
            if (!resp.ok) {
                if (resp.status === 429) return { ok: false, status: 'RATE_LIMITED' };
                return { ok: false, status: `HTTP_${resp.status}` };
            }
            const data = await resp.json();
            if (Array.isArray(data) && data.length > 0) {
                const first = data[0];
                const lat = parseFloat(first.lat);
                const lng = parseFloat(first.lon);
                if (Number.isFinite(lat) && Number.isFinite(lng)) {
                    return { ok: true, lat, lng, status: 'OK' };
                }
            }
            return { ok: false, status: 'ZERO_RESULTS' };
        } catch (err) {
            return { ok: false, status: 'REQUEST_ERROR', message: err && err.message };
        }
    }

    /**
     * Process every tree that still needs coordinates.
     *
     * @param {{ onProgress?: ({done,total,currentId})=>void,
     *           onTree?: (tree)=>void,
     *           includeFailed?: boolean }} opts
     * @returns {Promise<{attempted:number, succeeded:number, failed:number, skipped:number}>}
     */
    async function runAll(opts) {
        if (running) return { attempted: 0, succeeded: 0, failed: 0, skipped: 0 };
        running = true;
        try {
            const onProgress = (opts && opts.onProgress) || (() => {});
            const onTree     = (opts && opts.onTree)     || (() => {});
            const includeFailed = !!(opts && opts.includeFailed);

            const all = await HeritageDB.getAllTrees();
            const needs = all.filter((t) =>
                ((t.geocodeStatus === 'pending') ||
                 (includeFailed && t.geocodeStatus === 'failed') ||
                 ((t.lat == null || t.lng == null) && t.removed == null))
            );
            const queue = needs.filter((t) => buildAddress(t) !== null);
            const total = queue.length;

            let done = 0, succeeded = 0, failed = 0;
            onProgress({ done, total, currentId: null });

            for (const tree of queue) {
                const address = buildAddress(tree);
                const result = await geocodeOnce(address);
                const now = new Date().toISOString();
                const updated = await HeritageDB.updateTree(tree.id, (t) => {
                    t.geocodeTriedAt = now;
                    if (result.ok) {
                        t.lat = result.lat;
                        t.lng = result.lng;
                        t.geocodeStatus = 'ok';
                    } else {
                        t.geocodeStatus = 'failed';
                        t.geocodeLastError = result.status || 'ERROR';
                    }
                    return t;
                });
                if (updated) onTree(updated);
                done++;
                if (result.ok) succeeded++; else failed++;
                onProgress({ done, total, currentId: tree.id });

                if (result.status === 'RATE_LIMITED') {
                    await wait(10000);
                }
                await wait(STEP_MS);
            }

            return { attempted: total, succeeded, failed, skipped: needs.length - queue.length };
        } finally {
            running = false;
        }
    }

    function isRunning() { return running; }

    global.HeritageGeocode = {
        runAll,
        isRunning,
        sanitizeLocation,
        buildAddress
    };
})(window);
