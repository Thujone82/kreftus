// Loads heritage/data/trees.json and merges it into the IndexedDB.
//
// The bundled snapshot is produced by heritage/heritage.ps1 which pre-geocodes
// every tree. Coordinates therefore arrive in the JSON and are written
// straight into the DB. The browser-side geocoder only runs for the rare cases
// where the scraper couldn't resolve an address.
//
// Merge rules (hard requirement from the product spec):
//   - NEW id in JSON (not in DB)     -> insert a new record with the JSON's
//                                       lat/lng (if any). found=false, notes="".
//   - EXISTING id:
//       * Always update canonical fields if they changed:
//           year, name, species, commonName, removed.
//       * Update location text if changed. If the JSON has fresh coords, use
//         them; otherwise clear lat/lng and mark geocodeStatus = "pending" so
//         the fallback geocoder can retry. User found/foundDate/notes are
//         preserved in every case.
//       * When the snapshot provides authoritative coords (scraper succeeded)
//         they overwrite any stale lat/lng in the DB.
//       * NEVER overwrite found, foundDate, notes.
//
// Anything extra in the JSON that we don't know about is ignored. Anything in
// the DB that isn't in the JSON anymore is left alone (we do not delete; the
// City occasionally re-uses IDs or restores trees).

(function (global) {
    'use strict';

    const DATA_URL = 'data/trees.json';

    async function fetchSnapshot() {
        const url = DATA_URL + '?t=' + Date.now(); // cache-bust for sync calls
        const resp = await fetch(url, { cache: 'no-cache' });
        if (!resp.ok) throw new Error(`Failed to load ${DATA_URL}: HTTP ${resp.status}`);
        const snap = await resp.json();
        if (!snap || !Array.isArray(snap.trees)) {
            throw new Error(`${DATA_URL} is not in the expected format.`);
        }
        return snap;
    }

    function decorate(source) {
        // `source` is one element from the scraper's trees array.
        const { species, common } = HeritageWiki.splitSpeciesAndCommon(source.name || '');
        const lat = Number(source.lat);
        const lng = Number(source.lng);
        const hasCoords = Number.isFinite(lat) && Number.isFinite(lng);
        return {
            id: source.id,
            year: Number.isFinite(source.year) ? source.year : null,
            name: source.name || '',
            species,
            commonName: common,
            location: source.location || '',
            removed: (source.removed === null || source.removed === undefined) ? null : Number(source.removed),
            lat: hasCoords ? lat : null,
            lng: hasCoords ? lng : null,
            geocodeStatus: source.geocodeStatus || (hasCoords ? 'ok' : null)
        };
    }

    function isSameLocation(a, b) {
        return (a || '').trim() === (b || '').trim();
    }

    /** Normalize DB / JSON lat-lng for comparison (IndexedDB may round types). */
    function finiteCoord(v) {
        if (v === null || v === undefined) return null;
        if (typeof v === 'string' && v.trim() === '') return null;
        const n = Number(v);
        return Number.isFinite(n) ? n : null;
    }

    /**
     * Initial population: fills the DB from the snapshot if empty.
     * Returns { inserted, total }.
     */
    async function initialLoad(snapshot) {
        const existing = await HeritageDB.countTrees();
        if (existing > 0) return { inserted: 0, total: existing };
        const now = new Date().toISOString();
        const records = snapshot.trees.map((src) => {
            const d = decorate(src);
            const hasCoords = (d.lat != null && d.lng != null);
            // Removed trees don't need geocoding even if coords are missing.
            const defaultStatus = hasCoords ? 'ok'
                : (d.removed != null ? 'skipped-removed' : 'pending');
            return {
                id: d.id,
                year: d.year,
                name: d.name,
                species: d.species,
                commonName: d.commonName,
                location: d.location,
                removed: d.removed,
                lat: d.lat,
                lng: d.lng,
                geocodeStatus: d.geocodeStatus || defaultStatus,
                geocodeTriedAt: null,
                found: false,
                foundDate: null,
                notes: '',
                firstSeenAt: now,
                lastUpdatedAt: now
            };
        });
        await HeritageDB.putManyTrees(records);
        await HeritageDB.setMeta('lastSyncAt',    now);
        await HeritageDB.setMeta('sourceUrl',     snapshot.sourceUrl || '');
        await HeritageDB.setMeta('sourceVersion', snapshot.scrapedAt || '');
        return { inserted: records.length, total: records.length };
    }

    /**
     * Diff-merge an already-populated DB against a fresh snapshot.
     * Preserves user fields. Returns a summary object.
     */
    async function mergeUpdate(snapshot) {
        const existing = await HeritageDB.getAllTrees();
        const byId = new Map(existing.map((t) => [t.id, t]));
        const now = new Date().toISOString();

        const toWrite = [];
        let added = 0, updated = 0, newlyRemoved = 0, locationChanged = 0;

        for (const src of snapshot.trees) {
            const d = decorate(src);
            const hasCoords = (d.lat != null && d.lng != null);
            const prev = byId.get(d.id);
            if (!prev) {
                const defaultStatus = hasCoords ? 'ok'
                    : (d.removed != null ? 'skipped-removed' : 'pending');
                toWrite.push({
                    id: d.id,
                    year: d.year,
                    name: d.name,
                    species: d.species,
                    commonName: d.commonName,
                    location: d.location,
                    removed: d.removed,
                    lat: d.lat,
                    lng: d.lng,
                    geocodeStatus: d.geocodeStatus || defaultStatus,
                    geocodeTriedAt: null,
                    found: false,
                    foundDate: null,
                    notes: '',
                    firstSeenAt: now,
                    lastUpdatedAt: now
                });
                added++;
                continue;
            }

            const locChanged = !isSameLocation(prev.location, d.location);
            const removedAppeared = (prev.removed == null) && (d.removed != null);
            const nameChanged = prev.name !== d.name;
            const yearChanged = prev.year !== d.year;
            const removedChanged = (prev.removed || null) !== (d.removed || null);
            const pl = finiteCoord(prev.lat);
            const pg = finiteCoord(prev.lng);
            const coordsChanged = hasCoords && (
                pl === null || pg === null ||
                Math.abs(pl - d.lat) > 1e-8 || Math.abs(pg - d.lng) > 1e-8
            );
            const dbMissingCoords = (prev.lat == null || prev.lng == null);

            if (!locChanged && !nameChanged && !yearChanged && !removedChanged
                && !coordsChanged && !(dbMissingCoords && hasCoords)) {
                // Nothing to persist.
                continue;
            }

            const merged = {
                ...prev,
                year: d.year,
                name: d.name,
                species: d.species,
                commonName: d.commonName,
                location: d.location,
                removed: d.removed,
                lastUpdatedAt: now
                // found / foundDate / notes intentionally untouched.
            };
            if (hasCoords) {
                // Scraper result is authoritative.
                merged.lat = d.lat;
                merged.lng = d.lng;
                merged.geocodeStatus = 'ok';
                merged.geocodeTriedAt = null;
            } else if (locChanged) {
                // Snapshot doesn't have coords for this location - queue fallback geocode.
                merged.lat = null;
                merged.lng = null;
                merged.geocodeStatus = (d.removed != null) ? 'skipped-removed' : 'pending';
                merged.geocodeTriedAt = null;
            } else if (d.removed != null && prev.geocodeStatus === 'pending') {
                merged.geocodeStatus = 'skipped-removed';
            }
            if (locChanged) locationChanged++;
            toWrite.push(merged);
            updated++;
            if (removedAppeared) newlyRemoved++;
        }

        if (toWrite.length > 0) await HeritageDB.putManyTrees(toWrite);
        await HeritageDB.setMeta('lastSyncAt',    now);
        await HeritageDB.setMeta('sourceUrl',     snapshot.sourceUrl || '');
        await HeritageDB.setMeta('sourceVersion', snapshot.scrapedAt || '');

        return { added, updated, newlyRemoved, locationChanged, total: existing.length + added };
    }

    global.HeritageSync = {
        fetchSnapshot,
        initialLoad,
        mergeUpdate
    };
})(window);
