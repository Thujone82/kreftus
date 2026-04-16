// Loads heritage/data/trees.json and merges it into the IndexedDB.
//
// Merge rules (hard requirement from the product spec):
//   - NEW id in JSON (not in DB)     -> insert a new record. found=false,
//                                       notes="", no lat/lng. Needs geocoding.
//   - EXISTING id:
//       * Always update canonical fields if they changed:
//           year, name, species, commonName
//       * Update location text if changed. If location changed, clear lat/lng
//         and mark geocodeStatus = "pending" so the geocoder retries it. The
//         user's found/foundDate/notes are STILL preserved.
//       * Update `removed` year (only way to add a "newly removed" marker).
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
        return {
            id: source.id,
            year: Number.isFinite(source.year) ? source.year : null,
            name: source.name || '',
            species,
            commonName: common,
            location: source.location || '',
            removed: (source.removed === null || source.removed === undefined) ? null : Number(source.removed)
        };
    }

    function isSameLocation(a, b) {
        return (a || '').trim() === (b || '').trim();
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
            return {
                ...d,
                lat: null,
                lng: null,
                geocodeStatus: 'pending',
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
            const prev = byId.get(d.id);
            if (!prev) {
                toWrite.push({
                    ...d,
                    lat: null,
                    lng: null,
                    geocodeStatus: 'pending',
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

            if (!locChanged && !nameChanged && !yearChanged && !removedChanged) {
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
            if (locChanged) {
                // Force a fresh geocode, but leave old lat/lng in place until we have a new one,
                // so the marker stays on the map in the meantime.
                merged.geocodeStatus = 'pending';
                merged.geocodeTriedAt = null;
                locationChanged++;
            }
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
