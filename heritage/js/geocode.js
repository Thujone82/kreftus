// Throttled geocoding queue.
//
// Runs google.maps.Geocoder against any tree with geocodeStatus === 'pending'
// (or a manually-requested 'failed' retry). Reports progress via a callback
// so the UI can drive the top progress bar.
//
// Throttling: concurrency 1, ~180ms between requests. Google Geocoding API's
// per-second cap is generous, but being polite avoids temporary over-quota
// failures (OVER_QUERY_LIMIT).

(function (global) {
    'use strict';

    const PORTLAND_OR_SUFFIX = ', Portland, OR';
    const STEP_MS = 180;

    let running = false;

    function wait(ms) { return new Promise((res) => setTimeout(res, ms)); }

    function sanitizeLocation(loc) {
        if (!loc) return '';
        let t = String(loc);
        // Strip parenthetical qualifiers like "(private, side yard)".
        t = t.replace(/\([^)]*\)/g, ' ');
        // Collapse whitespace.
        t = t.replace(/\s+/g, ' ').trim();
        return t;
    }

    function buildAddress(tree) {
        const loc = sanitizeLocation(tree.location);
        if (!loc) return null;
        // If the location already says "Removed from list ..." there is no real address to geocode.
        if (/removed from list/i.test(loc)) return null;
        // If the location already mentions Portland/Oregon/OR, don't double it up.
        if (/\bPortland\b/i.test(loc)) return loc;
        return loc + PORTLAND_OR_SUFFIX;
    }

    function geocodeOnce(geocoder, address) {
        return new Promise((resolve) => {
            geocoder.geocode({ address }, (results, status) => {
                if (status === 'OK' && results && results[0] && results[0].geometry) {
                    const loc = results[0].geometry.location;
                    resolve({ ok: true, lat: loc.lat(), lng: loc.lng(), status });
                } else {
                    resolve({ ok: false, status });
                }
            });
        });
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

            if (!global.google || !global.google.maps || !global.google.maps.Geocoder) {
                throw new Error('Google Maps Geocoder is not available. Is the API key correct?');
            }
            const geocoder = new global.google.maps.Geocoder();

            const all = await HeritageDB.getAllTrees();
            const needs = all.filter((t) =>
                (t.geocodeStatus === 'pending') ||
                (includeFailed && t.geocodeStatus === 'failed') ||
                (!t.lat || !t.lng) && (t.removed == null)
            );
            // Skip removed trees if we don't have a real address for them.
            const queue = needs.filter((t) => buildAddress(t) !== null);
            const total = queue.length;

            let done = 0, succeeded = 0, failed = 0;
            onProgress({ done, total, currentId: null });

            for (const tree of queue) {
                const address = buildAddress(tree);
                const result = await geocodeOnce(geocoder, address);
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

                // Back off if Google told us we're going too fast.
                if (result.status === 'OVER_QUERY_LIMIT') {
                    await wait(2000);
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
